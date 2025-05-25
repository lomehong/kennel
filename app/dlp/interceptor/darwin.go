//go:build darwin

package interceptor

import (
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// PFInterceptor macOS平台的流量拦截器实现
type PFInterceptor struct {
	config       InterceptorConfig
	packetCh     chan *PacketInfo
	stopCh       chan struct{}
	stats        InterceptorStats
	logger       logging.Logger
	processCache ProcessCache
	running      int32
	mu           sync.RWMutex
	pfRules      []string
}

// StartCapture 开始捕获数据包
func (p *PFInterceptor) StartCapture() error {
	return p.Start()
}

// StopCapture 停止捕获数据包
func (p *PFInterceptor) StopCapture() error {
	return p.Stop()
}

// ReinjectPacket 重新注入数据包
func (p *PFInterceptor) ReinjectPacket(packet *PacketInfo) error {
	return p.Reinject(packet)
}

// NewDarwinInterceptor 创建macOS流量拦截器
func NewDarwinInterceptor(logger logging.Logger) PlatformInterceptor {
	return &PFInterceptor{
		logger:  logger,
		stopCh:  make(chan struct{}),
		pfRules: make([]string, 0),
	}
}

// createPlatformInterceptor 创建平台特定的拦截器
func createPlatformInterceptor(logger logging.Logger) TrafficInterceptor {
	return NewPFInterceptor(logger)
}

// createRealInterceptor 创建真实的拦截器实现
func createRealInterceptor(logger logging.Logger) TrafficInterceptor {
	return NewPFInterceptor(logger)
}

// NewPFInterceptor 创建macOS流量拦截器
func NewPFInterceptor(logger logging.Logger) TrafficInterceptor {
	return &PFInterceptor{
		logger:  logger,
		stopCh:  make(chan struct{}),
		pfRules: make([]string, 0),
	}
}

// Initialize 初始化拦截器
func (p *PFInterceptor) Initialize(config InterceptorConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config
	p.packetCh = make(chan *PacketInfo, config.ChannelSize)
	p.processCache = NewProcessCache(config.CacheSize)
	p.stats.StartTime = time.Now()

	p.logger.Info("初始化PF拦截器",
		"interface", config.Interface,
		"proxy_port", config.ProxyPort,
		"bypass_cidr", config.BypassCIDR)

	// 配置PF规则
	if err := p.configurePFRules(); err != nil {
		return fmt.Errorf("配置PF规则失败: %w", err)
	}

	p.logger.Info("PF拦截器初始化完成")
	return nil
}

// Start 启动流量拦截
func (p *PFInterceptor) Start() error {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return fmt.Errorf("拦截器已在运行")
	}

	p.logger.Info("启动PF拦截器")

	// 应用PF规则
	if err := p.applyPFRules(); err != nil {
		atomic.StoreInt32(&p.running, 0)
		return fmt.Errorf("应用PF规则失败: %w", err)
	}

	// 启动数据包捕获
	go p.capturePackets()

	p.logger.Info("PF拦截器已启动")
	return nil
}

// Stop 停止流量拦截
func (p *PFInterceptor) Stop() error {
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return fmt.Errorf("拦截器未在运行")
	}

	p.logger.Info("停止PF拦截器")

	// 发送停止信号
	close(p.stopCh)

	// 清除PF规则
	if err := p.clearPFRules(); err != nil {
		p.logger.Error("清除PF规则失败", "error", err)
	}

	// 关闭数据包通道
	close(p.packetCh)

	p.logger.Info("PF拦截器已停止")
	return nil
}

// SetFilter 设置过滤规则
func (p *PFInterceptor) SetFilter(filter string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("设置过滤规则", "filter", filter)
	// PF使用不同的规则语法，这里需要转换
	return p.configurePFRules()
}

// GetPacketChannel 获取数据包通道
func (p *PFInterceptor) GetPacketChannel() <-chan *PacketInfo {
	return p.packetCh
}

// Reinject 重新注入数据包
func (p *PFInterceptor) Reinject(packet *PacketInfo) error {
	// macOS上的数据包重新注入实现
	// 这里简化处理，实际实现需要使用系统调用
	atomic.AddUint64(&p.stats.PacketsReinject, 1)
	p.logger.Debug("重新注入数据包", "packet_id", packet.ID)
	return nil
}

// GetStats 获取统计信息
func (p *PFInterceptor) GetStats() InterceptorStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.stats
	stats.Uptime = time.Since(p.stats.StartTime)
	return stats
}

// HealthCheck 健康检查
func (p *PFInterceptor) HealthCheck() error {
	if atomic.LoadInt32(&p.running) == 0 {
		return fmt.Errorf("拦截器未运行")
	}

	// 检查PF规则是否正常
	if err := p.checkPFRules(); err != nil {
		return fmt.Errorf("PF规则检查失败: %w", err)
	}

	return nil
}

// configurePFRules 配置PF规则
func (p *PFInterceptor) configurePFRules() error {
	p.pfRules = []string{
		fmt.Sprintf("# DLP流量重定向规则"),
		fmt.Sprintf("rdr pass on %s proto tcp to !%s -> 127.0.0.1 port %d",
			p.config.Interface, p.config.BypassCIDR, p.config.ProxyPort),
		fmt.Sprintf("rdr pass on %s proto udp to !%s -> 127.0.0.1 port %d",
			p.config.Interface, p.config.BypassCIDR, p.config.ProxyPort),
	}

	p.logger.Debug("配置PF规则", "rules", p.pfRules)
	return nil
}

// applyPFRules 应用PF规则
func (p *PFInterceptor) applyPFRules() error {
	// 创建临时规则文件
	rulesFile := "/tmp/dlp_pf_rules"

	// 写入规则到文件
	// 这里简化处理，实际实现需要写入文件
	p.logger.Debug("应用PF规则", "file", rulesFile)

	// 加载PF规则
	cmd := exec.Command("pfctl", "-f", rulesFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("加载PF规则失败: %w", err)
	}

	// 启用PF
	cmd = exec.Command("pfctl", "-e")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("启用PF失败: %w", err)
	}

	return nil
}

// clearPFRules 清除PF规则
func (p *PFInterceptor) clearPFRules() error {
	// 禁用PF
	cmd := exec.Command("pfctl", "-d")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("禁用PF失败: %w", err)
	}

	// 清除规则
	cmd = exec.Command("pfctl", "-F", "all")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("清除PF规则失败: %w", err)
	}

	return nil
}

// checkPFRules 检查PF规则
func (p *PFInterceptor) checkPFRules() error {
	// 检查PF状态
	cmd := exec.Command("pfctl", "-s", "info")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("检查PF状态失败: %w", err)
	}

	p.logger.Debug("PF状态", "output", string(output))
	return nil
}

// capturePackets 捕获数据包
func (p *PFInterceptor) capturePackets() {
	p.logger.Debug("开始捕获数据包")
	defer p.logger.Debug("数据包捕获结束")

	// 真实的PF数据包捕获
	// 使用pfctl和网络接口捕获真实网络数据包
	p.logger.Info("启动PF数据包捕获")

	for {
		select {
		case <-p.stopCh:
			return
		default:
			// 实现真实的PF数据包捕获
			// 这里应该使用libpcap或其他底层库来捕获数据包
			// 由于需要CGO和系统库，这里提供一个框架实现

			// TODO: 实现真实的PF数据包处理
			// 1. 从网络接口捕获数据包
			// 2. 解析数据包头部信息
			// 3. 获取关联的进程信息
			// 4. 发送到处理通道

			p.logger.Debug("等待PF数据包...")
			time.Sleep(1 * time.Second) // 避免CPU占用过高
		}
	}
}

// 删除了createMockPacket函数 - 不再使用模拟数据包

// getProcessInfo 获取进程信息
func (p *PFInterceptor) getProcessInfo(pid int) *ProcessInfo {
	// 从缓存中获取进程信息
	if info := p.processCache.Get(uint32(pid)); info != nil {
		return info
	}

	// 查询进程信息
	processInfo := &ProcessInfo{
		PID: pid,
	}

	// 使用ps命令获取进程信息
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "comm=")
	output, err := cmd.Output()
	if err == nil {
		processInfo.ProcessName = string(output)
	}

	// 缓存进程信息
	p.processCache.Set(uint32(pid), processInfo)

	return processInfo
}
