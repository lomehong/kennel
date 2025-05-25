//go:build windows

package interceptor

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/lomehong/kennel/pkg/logging"
)

// WinDivert API 常量
const (
	WINDIVERT_LAYER_NETWORK         = 0
	WINDIVERT_LAYER_NETWORK_FORWARD = 1
	WINDIVERT_FLAG_SNIFF            = 1
	WINDIVERT_FLAG_DROP             = 2
	WINDIVERT_FLAG_RECV_ONLY        = 4
	WINDIVERT_FLAG_SEND_ONLY        = 8
	WINDIVERT_FLAG_NO_INSTALL       = 16
	WINDIVERT_FLAG_FRAGMENTS        = 32
)

// Windows错误代码常量
const (
	ERROR_INVALID_HANDLE = 6
	ERROR_ACCESS_DENIED  = 5
	ERROR_NO_MORE_ITEMS  = 259
)

// WinDivert 地址结构
type WinDivertAddress struct {
	Timestamp   int64
	Layer       uint8
	Event       uint8
	Sniffed     uint8
	Outbound    uint8
	Loopback    uint8
	Impostor    uint8
	IPv6        uint8
	IPChecksum  uint8
	TCPChecksum uint8
	UDPChecksum uint8
	Reserved1   uint8
	Reserved2   uint32
	IfIdx       uint32
	SubIfIdx    uint32
	Reserved3   uint64
}

// IP 头部结构
type IPHeader struct {
	VersionIHL          uint8
	TOS                 uint8
	Length              uint16
	ID                  uint16
	FlagsFragmentOffset uint16
	TTL                 uint8
	Protocol            uint8
	Checksum            uint16
	SrcAddr             uint32
	DstAddr             uint32
}

// TCP 头部结构
type TCPHeader struct {
	SrcPort    uint16
	DstPort    uint16
	SeqNum     uint32
	AckNum     uint32
	DataOffset uint8
	Flags      uint8
	Window     uint16
	Checksum   uint16
	UrgentPtr  uint16
}

// WinDivertInterceptorImpl Windows平台真实流量拦截器
type WinDivertInterceptorImpl struct {
	config         InterceptorConfig
	packetCh       chan *PacketInfo
	reinjectCh     chan *PacketInfo // 重新注入通道
	stopCh         chan struct{}
	stats          InterceptorStats
	logger         logging.Logger
	processCache   ProcessCache
	processTracker *ProcessTracker
	running        int32
	mu             sync.RWMutex

	// 性能优化组件
	rateLimiter *AdaptiveLimiter

	// WinDivert 相关
	handle        syscall.Handle
	windivertDLL  *syscall.LazyDLL
	driverManager *WinDivertDriverManager
	installer     *WinDivertInstaller

	// WinDivert API 函数
	winDivertOpen              *syscall.LazyProc
	winDivertRecv              *syscall.LazyProc
	winDivertSend              *syscall.LazyProc
	winDivertClose             *syscall.LazyProc
	winDivertHelperParsePacket *syscall.LazyProc
}

// NewWinDivertInterceptor 创建Windows流量拦截器
func NewWinDivertInterceptor(logger logging.Logger) TrafficInterceptor {
	interceptor := &WinDivertInterceptorImpl{
		logger:         logger,
		stopCh:         make(chan struct{}),
		handle:         syscall.InvalidHandle,
		processTracker: NewProcessTracker(logger),
		driverManager:  NewWinDivertDriverManager(logger),
		installer:      NewWinDivertInstaller(logger),
	}

	// 加载WinDivert DLL
	interceptor.loadWinDivertDLL()

	return interceptor
}

// loadWinDivertDLL 加载WinDivert动态链接库
func (w *WinDivertInterceptorImpl) loadWinDivertDLL() {
	// 首先尝试从当前目录加载
	w.windivertDLL = syscall.NewLazyDLL("./WinDivert.dll")

	w.winDivertOpen = w.windivertDLL.NewProc("WinDivertOpen")
	w.winDivertRecv = w.windivertDLL.NewProc("WinDivertRecv")
	w.winDivertSend = w.windivertDLL.NewProc("WinDivertSend")
	w.winDivertClose = w.windivertDLL.NewProc("WinDivertClose")
	w.winDivertHelperParsePacket = w.windivertDLL.NewProc("WinDivertHelperParsePacket")
}

// Initialize 初始化拦截器
func (w *WinDivertInterceptorImpl) Initialize(config InterceptorConfig) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.config = config
	w.packetCh = make(chan *PacketInfo, config.ChannelSize)
	w.reinjectCh = make(chan *PacketInfo, config.ChannelSize) // 重新注入通道
	w.processCache = NewProcessCache(config.CacheSize)
	w.stats.StartTime = time.Now()

	// 初始化自适应流量限制器
	w.rateLimiter = NewAdaptiveLimiter(
		1000,        // 每秒最大1000个数据包
		10485760,    // 每秒最大10MB
		100,         // 突发大小100
		80.0,        // CPU阈值80%
		80.0,        // 内存阈值80%
		time.Minute, // 检查间隔1分钟
		w.logger,
	)

	w.logger.Info("初始化WinDivert拦截器",
		"filter", config.Filter,
		"buffer_size", config.BufferSize,
		"workers", config.WorkerCount,
		"mode", config.Mode,
		"auto_reinject", config.AutoReinject,
		"rate_limiter", "enabled")

	return nil
}

// Start 启动流量拦截
func (w *WinDivertInterceptorImpl) Start() error {
	if !atomic.CompareAndSwapInt32(&w.running, 0, 1) {
		return fmt.Errorf("拦截器已在运行")
	}

	w.logger.Info("启动生产级WinDivert流量拦截")

	// 1. 检查管理员权限
	if !w.isRunningAsAdmin() {
		atomic.StoreInt32(&w.running, 0)
		return fmt.Errorf("WinDivert需要管理员权限，请以管理员身份运行程序")
	}
	w.logger.Info("✓ 管理员权限检查通过")

	// 2. 诊断并修复驱动问题
	if err := w.driverManager.DiagnoseDriverIssues(); err != nil {
		w.logger.Error("WinDivert驱动诊断失败", "error", err)

		// 尝试自动修复
		w.logger.Info("尝试自动修复WinDivert驱动问题")
		if err := w.repairWinDivertDriver(); err != nil {
			atomic.StoreInt32(&w.running, 0)
			return fmt.Errorf("WinDivert驱动修复失败: %w", err)
		}
	}

	// 3. 确保WinDivert文件已安装
	if err := w.installer.AutoInstallIfNeeded(); err != nil {
		w.logger.Error("WinDivert安装检查失败", "error", err)
		atomic.StoreInt32(&w.running, 0)
		return fmt.Errorf("WinDivert未正确安装: %w", err)
	}
	w.logger.Info("✓ WinDivert文件安装检查通过")

	// 4. 安装并注册驱动
	if err := w.driverManager.InstallAndRegisterDriver(); err != nil {
		w.logger.Error("WinDivert驱动安装失败", "error", err)
		atomic.StoreInt32(&w.running, 0)
		return fmt.Errorf("WinDivert驱动安装失败: %w", err)
	}
	w.logger.Info("✓ WinDivert驱动安装和注册完成")

	// 检查WinDivert DLL是否可用
	if err := w.windivertDLL.Load(); err != nil {
		w.logger.Debug("从当前目录加载WinDivert.dll失败，尝试其他路径", "error", err)

		// 尝试从系统PATH加载
		w.windivertDLL = syscall.NewLazyDLL("WinDivert.dll")
		if err := w.windivertDLL.Load(); err != nil {
			w.logger.Debug("从系统PATH加载WinDivert.dll失败，尝试安装目录", "error", err)

			// 尝试从安装目录加载
			installPath := "C:\\Program Files\\WinDivert\\WinDivert.dll"
			w.windivertDLL = syscall.NewLazyDLL(installPath)
			if err := w.windivertDLL.Load(); err != nil {
				w.logger.Error("无法从任何路径加载WinDivert.dll",
					"current_dir", "./WinDivert.dll",
					"system_path", "WinDivert.dll",
					"install_path", installPath,
					"error", err)

				// 显示安装信息
				info := w.installer.GetInstallationInfo()
				w.logger.Info("WinDivert安装信息", "info", info)

				return fmt.Errorf("加载WinDivert.dll失败: %w", err)
			}
		}

		// 重新加载API函数
		w.winDivertOpen = w.windivertDLL.NewProc("WinDivertOpen")
		w.winDivertRecv = w.windivertDLL.NewProc("WinDivertRecv")
		w.winDivertSend = w.windivertDLL.NewProc("WinDivertSend")
		w.winDivertClose = w.windivertDLL.NewProc("WinDivertClose")
		w.winDivertHelperParsePacket = w.windivertDLL.NewProc("WinDivertHelperParsePacket")
	}

	w.logger.Info("WinDivert.dll加载成功")

	// 检查WinDivert驱动状态
	if err := w.checkWinDivertDriver(); err != nil {
		w.logger.Warn("WinDivert驱动检查警告", "error", err)
		// 不直接返回错误，继续尝试
	}

	// 打开WinDivert句柄 - 使用优化过滤器排除私有网络流量
	filter := w.buildOptimizedFilter()

	w.logger.Info("使用优化WinDivert过滤器（排除私有网络）",
		"filter", filter,
		"config_filter", w.config.Filter)

	// 使用重试机制打开WinDivert句柄
	handle, err := w.openWinDivertHandleWithRetry(filter)
	if err != nil {
		atomic.StoreInt32(&w.running, 0) // 重置运行状态
		return err
	}

	w.handle = syscall.Handle(handle)

	// 启动进程跟踪器
	w.processTracker.StartPeriodicUpdate(5 * time.Second)

	// 初始化连接表
	if err := w.processTracker.UpdateConnectionTables(); err != nil {
		w.logger.Warn("初始化连接表失败", "error", err)
	}

	// 启动数据包接收协程
	for i := 0; i < w.config.WorkerCount; i++ {
		go w.packetReceiver(i)
	}

	// 启动重新注入协程（如果启用自动重新注入）
	if w.config.AutoReinject {
		go w.reinjectWorker()
	}

	w.logger.Info("WinDivert流量拦截已启动",
		"handle", w.handle,
		"mode", w.config.Mode,
		"auto_reinject", w.config.AutoReinject)
	return nil
}

// Stop 停止流量拦截
func (w *WinDivertInterceptorImpl) Stop() error {
	if !atomic.CompareAndSwapInt32(&w.running, 1, 0) {
		return fmt.Errorf("拦截器未在运行")
	}

	w.logger.Info("停止WinDivert流量拦截")

	// 发送停止信号
	close(w.stopCh)

	// 关闭WinDivert句柄
	if w.handle != syscall.InvalidHandle {
		w.winDivertClose.Call(uintptr(w.handle))
		w.handle = syscall.InvalidHandle
	}

	// 关闭数据包通道
	if w.packetCh != nil {
		close(w.packetCh)
		w.packetCh = nil
	}

	return nil
}

// SetFilter 设置过滤规则
func (w *WinDivertInterceptorImpl) SetFilter(filter string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.config.Filter = filter
	w.logger.Info("设置WinDivert过滤规则", "filter", filter)

	// 如果正在运行，需要重启以应用新过滤器
	if atomic.LoadInt32(&w.running) == 1 {
		w.logger.Warn("过滤器更改需要重启拦截器")
	}

	return nil
}

// GetPacketChannel 获取数据包通道
func (w *WinDivertInterceptorImpl) GetPacketChannel() <-chan *PacketInfo {
	return w.packetCh
}

// Reinject 重新注入数据包
func (w *WinDivertInterceptorImpl) Reinject(packet *PacketInfo) error {
	if w.handle == syscall.InvalidHandle {
		return fmt.Errorf("WinDivert句柄无效")
	}

	// 使用原始地址信息重新注入
	var addr *WinDivertAddress
	if addrData, exists := packet.Metadata["windivert_address"]; exists {
		if originalAddr, ok := addrData.(*WinDivertAddress); ok {
			addr = originalAddr
		}
	}

	// 如果没有原始地址信息，构造默认地址
	if addr == nil {
		addr = &WinDivertAddress{
			Outbound: 1,
			IfIdx:    0,
			SubIfIdx: 0,
		}
		if packet.Direction == PacketDirectionInbound {
			addr.Outbound = 0
		}
		// 从元数据中获取接口信息
		if ifIdx, exists := packet.Metadata["interface_index"]; exists {
			if idx, ok := ifIdx.(uint32); ok {
				addr.IfIdx = idx
			}
		}
		if subIfIdx, exists := packet.Metadata["sub_interface_index"]; exists {
			if idx, ok := subIfIdx.(uint32); ok {
				addr.SubIfIdx = idx
			}
		}
	}

	var written uint32
	ret, _, errno := w.winDivertSend.Call(
		uintptr(w.handle),
		uintptr(unsafe.Pointer(&packet.Payload[0])),
		uintptr(len(packet.Payload)),
		uintptr(unsafe.Pointer(&written)),
		uintptr(unsafe.Pointer(addr)),
	)

	if ret == 0 {
		return fmt.Errorf("重新注入数据包失败: %v", errno)
	}

	atomic.AddUint64(&w.stats.PacketsReinject, 1)
	w.logger.Debug("重新注入数据包", "packet_id", packet.ID, "size", written, "direction", packet.Direction)
	return nil
}

// GetStats 获取统计信息
func (w *WinDivertInterceptorImpl) GetStats() InterceptorStats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	stats := w.stats
	stats.Uptime = time.Since(w.stats.StartTime)
	return stats
}

// HealthCheck 健康检查
func (w *WinDivertInterceptorImpl) HealthCheck() error {
	if atomic.LoadInt32(&w.running) == 0 {
		return fmt.Errorf("拦截器未运行")
	}

	if w.handle == syscall.InvalidHandle {
		return fmt.Errorf("WinDivert句柄无效")
	}

	return nil
}

// packetReceiver 数据包接收协程（性能优化版本）
func (w *WinDivertInterceptorImpl) packetReceiver(workerID int) {
	w.logger.Debug("启动数据包接收协程", "worker_id", workerID)
	defer w.logger.Debug("数据包接收协程退出", "worker_id", workerID)

	buffer := make([]byte, w.config.BufferSize)
	errorCount := 0
	maxErrors := 10 // 最大连续错误次数

	// 性能优化：批量处理和自适应延迟
	batchSize := 5
	packets := make([]*PacketInfo, 0, batchSize)
	lastProcessTime := time.Now()
	adaptiveDelay := time.Microsecond * 100 // 初始延迟100微秒

	// 性能监控
	processedCount := uint64(0)
	lastStatsTime := time.Now()

	for {
		select {
		case <-w.stopCh:
			// 处理剩余的数据包
			if len(packets) > 0 {
				w.processBatch(packets, workerID)
			}
			return
		default:
			// 接收数据包
			packet, err := w.receivePacket(buffer)
			if err != nil {
				if atomic.LoadInt32(&w.running) == 1 {
					errorCount++

					// 减少错误日志频率，避免日志洪水
					if errorCount%10 == 1 {
						w.logger.Error("接收数据包失败",
							"worker_id", workerID,
							"error", err,
							"error_count", errorCount)
					}

					// 如果连续错误过多，可能是句柄失效，退出协程
					if errorCount >= maxErrors {
						w.logger.Error("连续错误过多，退出数据包接收协程",
							"worker_id", workerID,
							"max_errors", maxErrors)
						atomic.StoreInt32(&w.running, 0)
						return
					}

					// 自适应延迟：错误越多，延迟越长
					adaptiveDelay = time.Duration(errorCount) * time.Millisecond * 50
					if adaptiveDelay > time.Second {
						adaptiveDelay = time.Second
					}
					time.Sleep(adaptiveDelay)
				}
				continue
			}

			// 重置错误计数和延迟
			errorCount = 0
			adaptiveDelay = time.Microsecond * 100

			if packet != nil {
				// 应用流量限制
				if w.rateLimiter != nil && !w.rateLimiter.AllowPacket(int64(packet.Size)) {
					// 数据包被流量限制器丢弃
					atomic.AddUint64(&w.stats.PacketsDropped, 1)
					continue
				}

				atomic.AddUint64(&w.stats.PacketsProcessed, 1)
				atomic.AddUint64(&w.stats.BytesProcessed, uint64(packet.Size))
				processedCount++

				// 添加到批处理队列
				packets = append(packets, packet)

				// 当批次满了或者距离上次处理时间超过阈值时，处理批次
				if len(packets) >= batchSize || time.Since(lastProcessTime) > time.Millisecond*5 {
					w.processBatch(packets, workerID)
					packets = packets[:0] // 重置切片但保留容量
					lastProcessTime = time.Now()
				}

				// 定期输出性能统计
				if time.Since(lastStatsTime) > time.Minute*5 {
					w.logger.Info("数据包处理性能统计",
						"worker_id", workerID,
						"processed_last_5min", processedCount,
						"avg_per_second", processedCount/300)
					processedCount = 0
					lastStatsTime = time.Now()
				}
			}
		}
	}
}

// processBatch 批量处理数据包（性能优化）
func (w *WinDivertInterceptorImpl) processBatch(packets []*PacketInfo, workerID int) {
	for _, packet := range packets {
		// 根据模式决定处理方式
		switch w.config.Mode {
		case ModeMonitorOnly:
			// 监控模式：立即重新注入，然后发送到分析通道（优化性能）
			if w.config.AutoReinject {
				// 优先重新注入，避免网络延迟
				if err := w.Reinject(packet); err != nil {
					w.logger.Debug("重新注入失败", "error", err, "packet_id", packet.ID)
					atomic.AddUint64(&w.stats.PacketsDropped, 1)
					continue // 跳过这个数据包
				}
			}

			// 非阻塞发送到分析通道
			select {
			case w.packetCh <- packet:
				// 成功发送到分析通道
			default:
				// 分析通道满了，不影响重新注入
				atomic.AddUint64(&w.stats.PacketsDropped, 1)
			}

		case ModeInterceptAndAllow:
			// 拦截并允许模式：发送到分析通道，分析后自动重新注入
			select {
			case w.packetCh <- packet:
				// 成功发送到分析通道，等待分析完成后重新注入
			case <-w.stopCh:
				return
			default:
				atomic.AddUint64(&w.stats.PacketsDropped, 1)
				// 如果分析通道满了，直接重新注入以避免阻断流量
				if w.config.AutoReinject {
					if err := w.Reinject(packet); err != nil {
						w.logger.Debug("直接重新注入失败", "error", err, "packet_id", packet.ID)
					}
				}
			}

		case ModeInterceptAndBlock:
			// 拦截并阻断模式：发送到分析通道，根据策略决定是否重新注入
			select {
			case w.packetCh <- packet:
				// 成功发送到分析通道，等待策略决策
			case <-w.stopCh:
				return
			default:
				atomic.AddUint64(&w.stats.PacketsDropped, 1)
				// 减少丢包日志频率
				if w.stats.PacketsDropped%100 == 1 {
					w.logger.Warn("数据包通道已满，丢弃数据包",
						"worker_id", workerID,
						"dropped_total", w.stats.PacketsDropped,
						"batch_size", len(packets))
				}
			}
		}
	}
}

// reinjectWorker 重新注入工作协程
func (w *WinDivertInterceptorImpl) reinjectWorker() {
	w.logger.Debug("启动重新注入工作协程")
	defer w.logger.Debug("重新注入工作协程退出")

	for {
		select {
		case packet := <-w.reinjectCh:
			if err := w.Reinject(packet); err != nil {
				w.logger.Debug("重新注入数据包失败", "error", err, "packet_id", packet.ID)
			}
		case <-w.stopCh:
			return
		}
	}
}

// receivePacket 接收单个数据包
func (w *WinDivertInterceptorImpl) receivePacket(buffer []byte) (*PacketInfo, error) {
	// 检查句柄是否有效
	if w.handle == syscall.InvalidHandle {
		return nil, fmt.Errorf("WinDivert句柄无效")
	}

	// 检查拦截器是否仍在运行
	if atomic.LoadInt32(&w.running) == 0 {
		return nil, fmt.Errorf("拦截器已停止")
	}

	var received uint32
	addr := &WinDivertAddress{}

	ret, _, errno := w.winDivertRecv.Call(
		uintptr(w.handle),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
		uintptr(unsafe.Pointer(&received)),
		uintptr(unsafe.Pointer(addr)),
	)

	if ret == 0 {
		// 检查具体的错误类型
		if sysErr, ok := errno.(syscall.Errno); ok {
			switch sysErr {
			case ERROR_INVALID_HANDLE:
				w.logger.Error("WinDivert句柄已失效，需要重新初始化")
				atomic.StoreInt32(&w.running, 0) // 标记为停止状态
				return nil, fmt.Errorf("WinDivert句柄失效")
			case ERROR_ACCESS_DENIED:
				w.logger.Error("WinDivert访问被拒绝，请检查管理员权限")
				return nil, fmt.Errorf("访问被拒绝，需要管理员权限")
			default:
				// 对于其他错误，记录详细信息
				w.logger.Debug("WinDivert接收数据包失败", "errno", sysErr, "handle", w.handle)
				return nil, fmt.Errorf("接收数据包失败: %v (错误代码: %d)", sysErr, sysErr)
			}
		} else {
			w.logger.Debug("WinDivert接收数据包失败", "errno", errno, "handle", w.handle)
			return nil, fmt.Errorf("接收数据包失败: %v", errno)
		}
	}

	if received == 0 {
		return nil, nil
	}

	// 解析数据包
	packet, err := w.parsePacket(buffer[:received], addr)
	if err != nil {
		w.logger.Debug("解析数据包失败", "error", err, "size", received)
		return nil, err
	}

	// 应用层额外过滤验证：确保不处理私有网络流量
	if packet != nil && w.shouldFilterPacket(packet) {
		w.logger.Debug("数据包被应用层过滤器排除",
			"dest_ip", packet.DestIP.String(),
			"dest_port", packet.DestPort,
			"direction", packet.Direction)
		return nil, nil // 返回nil表示数据包被过滤
	}

	return packet, nil
}

// shouldFilterPacket 检查数据包是否应该被过滤（排除私有网络流量）
func (w *WinDivertInterceptorImpl) shouldFilterPacket(packet *PacketInfo) bool {
	if packet == nil {
		return true
	}

	// 目标IP地址已经是net.IP类型
	destIP := packet.DestIP
	if destIP == nil {
		w.logger.Debug("目标IP地址为空", "packet_id", packet.ID)
		return false // IP为空时不过滤，让后续处理决定
	}

	// 检查是否为IPv4地址
	destIPv4 := destIP.To4()
	if destIPv4 == nil {
		// IPv6地址暂时不处理，可以根据需要扩展
		w.logger.Debug("IPv6地址暂不过滤", "dest_ip", destIP.String())
		return false
	}

	// 转换为32位整数便于比较
	destAddr := uint32(destIPv4[0])<<24 | uint32(destIPv4[1])<<16 | uint32(destIPv4[2])<<8 | uint32(destIPv4[3])

	// 检查是否为私有网络或本地地址
	isPrivateOrLocal :=
		// 本地回环 127.0.0.0/8
		(destAddr >= 0x7F000000 && destAddr <= 0x7FFFFFFF) ||
			// 私有网络A类 10.0.0.0/8
			(destAddr >= 0x0A000000 && destAddr <= 0x0AFFFFFF) ||
			// 私有网络B类 172.16.0.0/12
			(destAddr >= 0xAC100000 && destAddr <= 0xAC1FFFFF) ||
			// 私有网络C类 192.168.0.0/16
			(destAddr >= 0xC0A80000 && destAddr <= 0xC0A8FFFF) ||
			// 链路本地地址 169.254.0.0/16
			(destAddr >= 0xA9FE0000 && destAddr <= 0xA9FEFFFF) ||
			// 组播地址 224.0.0.0/4
			(destAddr >= 0xE0000000 && destAddr <= 0xEFFFFFFF) ||
			// 广播地址
			(destAddr == 0xFFFFFFFF)

	if isPrivateOrLocal {
		w.logger.Debug("检测到私有/本地网络流量，将被过滤",
			"dest_ip", destIP.String(),
			"dest_addr_hex", fmt.Sprintf("0x%08X", destAddr),
			"dest_port", packet.DestPort,
			"protocol", packet.Protocol)
		return true
	}

	// 公网地址，不过滤
	w.logger.Debug("检测到公网流量，允许处理",
		"dest_ip", destIP.String(),
		"dest_port", packet.DestPort,
		"protocol", packet.Protocol)
	return false
}

// parsePacket 解析数据包
func (w *WinDivertInterceptorImpl) parsePacket(data []byte, addr *WinDivertAddress) (*PacketInfo, error) {
	if len(data) < 20 { // 最小IP头部长度
		return nil, fmt.Errorf("数据包太小")
	}

	// 解析IP头部
	ipHeader := (*IPHeader)(unsafe.Pointer(&data[0]))

	// 检查IP版本
	version := ipHeader.VersionIHL >> 4
	if version != 4 {
		return nil, fmt.Errorf("不支持的IP版本: %d", version)
	}

	// 计算IP头部长度
	ipHeaderLen := int(ipHeader.VersionIHL&0x0F) * 4
	if len(data) < ipHeaderLen {
		return nil, fmt.Errorf("IP头部长度不足")
	}

	// 创建数据包信息
	packet := &PacketInfo{
		ID:        fmt.Sprintf("windivert_%d_%d", time.Now().UnixNano(), addr.IfIdx),
		Timestamp: time.Now(),
		Protocol:  Protocol(ipHeader.Protocol),
		SourceIP:  intToIP(ipHeader.SrcAddr),
		DestIP:    intToIP(ipHeader.DstAddr),
		Payload:   make([]byte, len(data)),
		Size:      len(data),
		Metadata:  make(map[string]interface{}),
	}

	// 设置方向
	if addr.Outbound == 1 {
		packet.Direction = PacketDirectionOutbound
	} else {
		packet.Direction = PacketDirectionInbound
	}

	// 复制数据包内容
	copy(packet.Payload, data)

	// 解析传输层协议
	if ipHeader.Protocol == 6 { // TCP
		if err := w.parseTCPHeader(packet, data[ipHeaderLen:]); err != nil {
			w.logger.Debug("解析TCP头部失败", "error", err)
		}
	} else if ipHeader.Protocol == 17 { // UDP
		if err := w.parseUDPHeader(packet, data[ipHeaderLen:]); err != nil {
			w.logger.Debug("解析UDP头部失败", "error", err)
		}
	}

	// 获取进程信息
	if processInfo := w.getProcessInfo(packet); processInfo != nil {
		packet.ProcessInfo = processInfo
	}

	// 添加元数据
	packet.Metadata["interface_index"] = addr.IfIdx
	packet.Metadata["sub_interface_index"] = addr.SubIfIdx
	packet.Metadata["timestamp"] = addr.Timestamp
	packet.Metadata["sniffed"] = addr.Sniffed == 1

	// 保存原始WinDivert地址信息，用于重新注入
	packet.Metadata["windivert_address"] = addr

	return packet, nil
}

// parseTCPHeader 解析TCP头部
func (w *WinDivertInterceptorImpl) parseTCPHeader(packet *PacketInfo, data []byte) error {
	if len(data) < 20 { // 最小TCP头部长度
		return fmt.Errorf("TCP头部长度不足")
	}

	tcpHeader := (*TCPHeader)(unsafe.Pointer(&data[0]))
	packet.SourcePort = ntohs(tcpHeader.SrcPort)
	packet.DestPort = ntohs(tcpHeader.DstPort)
	packet.Protocol = ProtocolTCP

	// 添加TCP特定元数据
	packet.Metadata["tcp_seq"] = ntohl(tcpHeader.SeqNum)
	packet.Metadata["tcp_ack"] = ntohl(tcpHeader.AckNum)
	packet.Metadata["tcp_flags"] = tcpHeader.Flags
	packet.Metadata["tcp_window"] = ntohs(tcpHeader.Window)

	return nil
}

// parseUDPHeader 解析UDP头部
func (w *WinDivertInterceptorImpl) parseUDPHeader(packet *PacketInfo, data []byte) error {
	if len(data) < 8 { // UDP头部长度
		return fmt.Errorf("UDP头部长度不足")
	}

	srcPort := ntohs(*(*uint16)(unsafe.Pointer(&data[0])))
	dstPort := ntohs(*(*uint16)(unsafe.Pointer(&data[2])))

	packet.SourcePort = srcPort
	packet.DestPort = dstPort
	packet.Protocol = ProtocolUDP

	return nil
}

// getProcessInfo 获取进程信息（增强版本 - 多策略查找）
func (w *WinDivertInterceptorImpl) getProcessInfo(packet *PacketInfo) *ProcessInfo {
	// 策略1：四元组精确匹配（TCP连接）
	var pid uint32

	if packet.Protocol == ProtocolTCP {
		if packet.Direction == PacketDirectionOutbound {
			// 出站：本地是源，远程是目标
			pid = w.processTracker.GetProcessByConnectionEx(
				packet.Protocol,
				packet.SourceIP, packet.SourcePort,
				packet.DestIP, packet.DestPort,
			)
		} else {
			// 入站：本地是目标，远程是源
			pid = w.processTracker.GetProcessByConnectionEx(
				packet.Protocol,
				packet.DestIP, packet.DestPort,
				packet.SourceIP, packet.SourcePort,
			)
		}
	}

	// 策略2：如果四元组匹配失败，使用本地连接匹配
	if pid == 0 {
		if packet.Direction == PacketDirectionOutbound {
			pid = w.processTracker.GetProcessByConnection(packet.Protocol, packet.SourceIP, packet.SourcePort)
		} else {
			pid = w.processTracker.GetProcessByConnection(packet.Protocol, packet.DestIP, packet.DestPort)
		}
	}

	// 策略3：如果仍然失败，尝试强制更新连接表后再次查找
	if pid == 0 {
		w.logger.Debug("首次查找失败，强制更新连接表后重试")
		if err := w.processTracker.UpdateConnectionTables(); err != nil {
			w.logger.Error("强制更新连接表失败", "error", err)
		} else {
			// 重新尝试查找
			if packet.Direction == PacketDirectionOutbound {
				pid = w.processTracker.GetProcessByConnection(packet.Protocol, packet.SourceIP, packet.SourcePort)
			} else {
				pid = w.processTracker.GetProcessByConnection(packet.Protocol, packet.DestIP, packet.DestPort)
			}
		}
	}

	if pid == 0 {
		w.logger.Debug("所有策略都未找到对应的进程",
			"direction", packet.Direction,
			"protocol", packet.Protocol,
			"source", fmt.Sprintf("%s:%d", packet.SourceIP, packet.SourcePort),
			"dest", fmt.Sprintf("%s:%d", packet.DestIP, packet.DestPort))

		// 返回一个包含基本信息的ProcessInfo，而不是nil
		return &ProcessInfo{
			PID:         0,
			ProcessName: "unknown_process",
			ExecutePath: "",
			User:        "unknown",
			CommandLine: "",
		}
	}

	// 从进程跟踪器获取详细进程信息
	processInfo := w.processTracker.GetProcessInfo(pid)
	if processInfo == nil {
		w.logger.Debug("获取进程详细信息失败", "pid", pid)
		// 返回基本的进程信息
		return &ProcessInfo{
			PID:         int(pid),
			ProcessName: "process_info_unavailable",
			ExecutePath: "",
			User:        "unknown",
			CommandLine: "",
		}
	}

	w.logger.Debug("成功获取进程信息",
		"pid", processInfo.PID,
		"name", processInfo.ProcessName,
		"path", processInfo.ExecutePath,
		"user", processInfo.User,
		"direction", packet.Direction,
		"protocol", packet.Protocol)

	return processInfo
}

func ntohs(n uint16) uint16 {
	return (n<<8)&0xff00 | n>>8
}

func ntohl(n uint32) uint32 {
	return (n<<24)&0xff000000 | (n<<8)&0xff0000 | (n>>8)&0xff00 | n>>24
}

// buildOptimizedFilter 构建优化的WinDivert过滤器，排除本地和私有网络流量
func (w *WinDivertInterceptorImpl) buildOptimizedFilter() string {
	// 基础过滤器：只拦截出站TCP流量的特定端口
	baseFilter := "outbound and tcp and (tcp.DstPort == 80 or tcp.DstPort == 443 or tcp.DstPort == 21 or tcp.DstPort == 25 or tcp.DstPort == 3306)"

	// 排除本地和私有网络的条件
	excludeConditions := []string{
		// 排除本地回环地址 127.0.0.0/8
		"not (ip.DstAddr >= 127.0.0.0 and ip.DstAddr <= 127.255.255.255)",
		// 排除私有网络A类 10.0.0.0/8
		"not (ip.DstAddr >= 10.0.0.0 and ip.DstAddr <= 10.255.255.255)",
		// 排除私有网络B类 172.16.0.0/12
		"not (ip.DstAddr >= 172.16.0.0 and ip.DstAddr <= 172.31.255.255)",
		// 排除私有网络C类 192.168.0.0/16
		"not (ip.DstAddr >= 192.168.0.0 and ip.DstAddr <= 192.168.255.255)",
		// 排除链路本地地址 169.254.0.0/16
		"not (ip.DstAddr >= 169.254.0.0 and ip.DstAddr <= 169.254.255.255)",
		// 排除组播地址 224.0.0.0/4
		"not (ip.DstAddr >= 224.0.0.0 and ip.DstAddr <= 239.255.255.255)",
		// 排除广播地址
		"not (ip.DstAddr == 255.255.255.255)",
	}

	// 组合过滤器
	filter := baseFilter
	for _, condition := range excludeConditions {
		filter += " and " + condition
	}

	w.logger.Info("构建优化过滤器", "filter", filter)
	return filter
}

// buildFallbackFilters 构建备用过滤器列表
func (w *WinDivertInterceptorImpl) buildFallbackFilters() []struct {
	filter string
	flag   uintptr
	desc   string
} {
	// 主过滤器：排除私有网络的完整过滤器
	optimizedFilter := w.buildOptimizedFilter()

	return []struct {
		filter string
		flag   uintptr
		desc   string
	}{
		// 首选：优化过滤器 + 嗅探模式
		{optimizedFilter, WINDIVERT_FLAG_SNIFF, "优化过滤器，嗅探模式，排除私有网络"},
		// 备选1：优化过滤器 + 默认模式
		{optimizedFilter, 0, "优化过滤器，默认模式，排除私有网络"},
		// 备选2：简化过滤器 + 嗅探模式（只排除本地回环）
		{"outbound and tcp and (tcp.DstPort == 80 or tcp.DstPort == 443) and not (ip.DstAddr >= 127.0.0.0 and ip.DstAddr <= 127.255.255.255)", WINDIVERT_FLAG_SNIFF, "简化过滤器，排除本地回环"},
		// 备选3：基础TCP过滤器 + 嗅探模式
		{"outbound and tcp and (tcp.DstPort == 80 or tcp.DstPort == 443)", WINDIVERT_FLAG_SNIFF, "基础TCP过滤器，嗅探模式"},
		// 备选4：最简单的TCP过滤器
		{"tcp", WINDIVERT_FLAG_SNIFF, "最简单TCP过滤器，嗅探模式"},
		// 备选5：所有流量（最后的选择）
		{"true", WINDIVERT_FLAG_SNIFF, "所有流量，嗅探模式"},
	}
}

// openWinDivertHandleWithRetry 使用重试机制打开WinDivert句柄
func (w *WinDivertInterceptorImpl) openWinDivertHandleWithRetry(originalFilter string) (uintptr, error) {
	// 使用优化的过滤器配置，排除私有网络流量
	testConfigs := w.buildFallbackFilters()

	var lastError syscall.Errno
	maxRetries := 2
	retryDelay := time.Second

	for retry := 0; retry < maxRetries; retry++ {
		w.logger.Debug("尝试打开WinDivert句柄", "retry", retry+1, "max_retries", maxRetries)

		for i, config := range testConfigs {
			w.logger.Debug("尝试配置",
				"config_index", i+1,
				"filter", config.filter,
				"flag", config.flag,
				"desc", config.desc)

			// 转换过滤器字符串为ANSI字符串
			testFilterPtr, err := syscall.BytePtrFromString(config.filter)
			if err != nil {
				w.logger.Debug("转换过滤器字符串失败", "filter", config.filter, "error", err)
				continue
			}

			ret, _, errno := w.winDivertOpen.Call(
				uintptr(unsafe.Pointer(testFilterPtr)),
				uintptr(WINDIVERT_LAYER_NETWORK),
				uintptr(0),           // priority (INT16)
				uintptr(config.flag), // flags (UINT64)
			)

			if ret != uintptr(syscall.InvalidHandle) {
				w.logger.Info("WinDivert句柄打开成功",
					"filter", config.filter,
					"flag", config.flag,
					"handle", ret,
					"retry", retry+1,
					"desc", config.desc)
				return ret, nil
			}

			if errno != nil {
				if sysErr, ok := errno.(syscall.Errno); ok {
					lastError = sysErr
				}
			}
			w.logger.Debug("WinDivert句柄打开失败",
				"filter", config.filter,
				"flag", config.flag,
				"error", errno,
				"error_code", lastError)
		}

		// 如果不是最后一次重试，等待一段时间再重试
		if retry < maxRetries-1 {
			w.logger.Info("等待重试", "delay", retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // 指数退避
		}
	}

	// 所有重试都失败了
	if lastError == ERROR_ACCESS_DENIED {
		return 0, fmt.Errorf("打开WinDivert句柄失败: 访问被拒绝，请以管理员身份运行程序")
	}

	// 提供更详细的错误信息
	var errorMsg string
	switch lastError {
	case 87: // ERROR_INVALID_PARAMETER
		errorMsg = "参数无效，可能是过滤器语法错误或WinDivert版本不兼容"
	case 2: // ERROR_FILE_NOT_FOUND
		errorMsg = "找不到WinDivert驱动文件"
	case 1275: // ERROR_DRIVER_FAILED_SLEEP
		errorMsg = "驱动程序加载失败"
	default:
		errorMsg = fmt.Sprintf("未知错误 (错误代码: %d)", lastError)
	}

	return 0, fmt.Errorf("打开WinDivert句柄失败: %s，已重试%d次", errorMsg, maxRetries)
}

// checkWinDivertDriver 检查WinDivert驱动状态
func (w *WinDivertInterceptorImpl) checkWinDivertDriver() error {
	w.logger.Debug("检查WinDivert驱动状态")

	// 检查驱动文件是否存在
	driverPaths := []string{
		"C:\\Program Files\\WinDivert\\WinDivert64.sys",
		"C:\\Program Files\\WinDivert\\WinDivert32.sys",
		"C:\\Program Files\\WinDivert\\WinDivert.sys",
		"./WinDivert64.sys",
		"./WinDivert32.sys",
		"./WinDivert.sys",
		// 添加当前工作目录的路径
		"WinDivert64.sys",
		"WinDivert32.sys",
		"WinDivert.sys",
	}

	var foundDriver bool
	for _, path := range driverPaths {
		if _, err := os.Stat(path); err == nil {
			w.logger.Debug("找到WinDivert驱动文件", "path", path)
			foundDriver = true
			break
		}
	}

	if !foundDriver {
		return fmt.Errorf("未找到WinDivert驱动文件")
	}

	// 检查API函数是否已加载
	if w.winDivertOpen == nil {
		return fmt.Errorf("WinDivert API函数未加载")
	}

	// 使用defer和recover来安全地调用API
	defer func() {
		if r := recover(); r != nil {
			w.logger.Warn("WinDivert API调用发生panic", "error", r)
		}
	}()

	// 尝试简单的API调用来验证驱动是否可用
	// 使用一个无效的过滤器来测试API是否响应
	testFilterPtr, err := syscall.BytePtrFromString("invalid_test_filter")
	if err != nil {
		return fmt.Errorf("创建测试过滤器失败: %w", err)
	}

	// 这个调用应该失败，但如果驱动正常，应该返回特定的错误代码
	ret, _, errno := w.winDivertOpen.Call(
		uintptr(unsafe.Pointer(testFilterPtr)),
		uintptr(WINDIVERT_LAYER_NETWORK),
		uintptr(0), // priority (INT16)
		uintptr(0), // flags (UINT64)
	)

	if ret != uintptr(syscall.InvalidHandle) {
		// 如果意外成功，关闭句柄
		if w.winDivertClose != nil {
			w.winDivertClose.Call(ret)
		}
		w.logger.Debug("WinDivert驱动测试意外成功")
		return nil
	}

	// 检查错误代码
	if errno != nil {
		if sysErr, ok := errno.(syscall.Errno); ok {
			switch sysErr {
			case 87: // ERROR_INVALID_PARAMETER - 这是预期的，说明驱动正常响应
				w.logger.Debug("WinDivert驱动响应正常")
				return nil
			case 2: // ERROR_FILE_NOT_FOUND
				return fmt.Errorf("WinDivert驱动文件未找到")
			case 1275: // ERROR_DRIVER_FAILED_SLEEP
				return fmt.Errorf("WinDivert驱动加载失败")
			case 5: // ERROR_ACCESS_DENIED
				return fmt.Errorf("WinDivert驱动访问被拒绝，需要管理员权限")
			default:
				w.logger.Debug("WinDivert驱动测试返回错误", "error_code", sysErr)
				// 其他错误可能是正常的，继续
				return nil
			}
		}
	}

	return nil
}

// isRunningAsAdmin 检查是否以管理员身份运行
func (w *WinDivertInterceptorImpl) isRunningAsAdmin() bool {
	// 尝试打开一个需要管理员权限的资源
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		w.logger.Debug("管理员权限检查失败", "error", err)
		return false
	}
	return true
}

// repairWinDivertDriver 修复WinDivert驱动问题
func (w *WinDivertInterceptorImpl) repairWinDivertDriver() error {
	w.logger.Info("开始修复WinDivert驱动")

	// 1. 重新安装WinDivert文件
	if err := w.installer.InstallWinDivert(); err != nil {
		w.logger.Error("重新安装WinDivert失败", "error", err)
		return err
	}

	// 2. 重新安装和注册驱动
	if err := w.driverManager.InstallAndRegisterDriver(); err != nil {
		w.logger.Error("重新安装驱动失败", "error", err)
		return err
	}

	// 3. 重新加载DLL
	w.loadWinDivertDLL()

	w.logger.Info("WinDivert驱动修复完成")
	return nil
}

// RestartWithDriverRepair 重启并修复驱动
func (w *WinDivertInterceptorImpl) RestartWithDriverRepair() error {
	w.logger.Info("重启WinDivert拦截器并修复驱动")

	// 停止当前拦截器
	if atomic.LoadInt32(&w.running) == 1 {
		if err := w.Stop(); err != nil {
			w.logger.Warn("停止拦截器失败", "error", err)
		}
	}

	// 重启驱动服务
	if err := w.driverManager.RestartDriverService(); err != nil {
		w.logger.Error("重启驱动服务失败", "error", err)
		return err
	}

	// 重新启动拦截器
	return w.Start()
}

// PerformHealthCheck 执行健康检查并自动修复
func (w *WinDivertInterceptorImpl) PerformHealthCheck() error {
	// 基本健康检查
	if err := w.HealthCheck(); err != nil {
		w.logger.Warn("基本健康检查失败", "error", err)

		// 尝试自动修复
		if repairErr := w.RestartWithDriverRepair(); repairErr != nil {
			return fmt.Errorf("健康检查失败且自动修复失败: %w", repairErr)
		}

		w.logger.Info("自动修复成功")
	}

	// 驱动状态检查
	if err := w.driverManager.DiagnoseDriverIssues(); err != nil {
		w.logger.Warn("驱动状态检查失败", "error", err)
		return err
	}

	w.logger.Info("WinDivert健康检查通过")
	return nil
}
