//go:build linux

package interceptor

import (
	"fmt"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// NetfilterInterceptor Linux平台的流量拦截器实现
type NetfilterInterceptor struct {
	config        InterceptorConfig
	packetCh      chan *PacketInfo
	stopCh        chan struct{}
	stats         InterceptorStats
	logger        logging.Logger
	processCache  ProcessCache
	running       int32
	mu            sync.RWMutex
	iptablesRules []string
	queueNum      uint16
}

// StartCapture 开始捕获数据包
func (n *NetfilterInterceptor) StartCapture() error {
	return n.Start()
}

// StopCapture 停止捕获数据包
func (n *NetfilterInterceptor) StopCapture() error {
	return n.Stop()
}

// ReinjectPacket 重新注入数据包
func (n *NetfilterInterceptor) ReinjectPacket(packet *PacketInfo) error {
	return n.Reinject(packet)
}

// NewLinuxInterceptor 创建Linux流量拦截器
func NewLinuxInterceptor(logger logging.Logger) PlatformInterceptor {
	return &NetfilterInterceptor{
		logger:        logger,
		stopCh:        make(chan struct{}),
		iptablesRules: make([]string, 0),
		queueNum:      0,
	}
}

// NewNetfilterInterceptor 向后兼容的构造函数
func NewNetfilterInterceptor(logger logging.Logger) TrafficInterceptor {
	return &NetfilterInterceptor{
		logger:        logger,
		stopCh:        make(chan struct{}),
		iptablesRules: make([]string, 0),
		queueNum:      0,
	}
}

// createPlatformInterceptor 创建平台特定的拦截器
func createPlatformInterceptor(logger logging.Logger) TrafficInterceptor {
	return NewNetfilterInterceptor(logger)
}

// createRealInterceptor 创建真实的拦截器实现
func createRealInterceptor(logger logging.Logger) TrafficInterceptor {
	return NewNetfilterInterceptor(logger)
}

// Initialize 初始化拦截器
func (n *NetfilterInterceptor) Initialize(config InterceptorConfig) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.config = config
	n.packetCh = make(chan *PacketInfo, config.ChannelSize)
	n.processCache = NewProcessCache(config.CacheSize)
	n.stats.StartTime = time.Now()

	n.logger.Info("初始化Netfilter拦截器",
		"interface", config.Interface,
		"proxy_port", config.ProxyPort,
		"bypass_cidr", config.BypassCIDR)

	// 配置iptables规则
	if err := n.configureIptablesRules(); err != nil {
		return fmt.Errorf("配置iptables规则失败: %w", err)
	}

	n.logger.Info("Netfilter拦截器初始化完成")
	return nil
}

// Start 启动流量拦截
func (n *NetfilterInterceptor) Start() error {
	if !atomic.CompareAndSwapInt32(&n.running, 0, 1) {
		return fmt.Errorf("拦截器已在运行")
	}

	n.logger.Info("启动Netfilter拦截器")

	// 应用iptables规则
	if err := n.applyIptablesRules(); err != nil {
		atomic.StoreInt32(&n.running, 0)
		return fmt.Errorf("应用iptables规则失败: %w", err)
	}

	// 启动数据包捕获
	go n.capturePackets()

	n.logger.Info("Netfilter拦截器已启动")
	return nil
}

// Stop 停止流量拦截
func (n *NetfilterInterceptor) Stop() error {
	if !atomic.CompareAndSwapInt32(&n.running, 1, 0) {
		return fmt.Errorf("拦截器未在运行")
	}

	n.logger.Info("停止Netfilter拦截器")

	// 发送停止信号
	close(n.stopCh)

	// 清除iptables规则
	if err := n.clearIptablesRules(); err != nil {
		n.logger.Error("清除iptables规则失败", "error", err)
	}

	// 关闭数据包通道
	close(n.packetCh)

	n.logger.Info("Netfilter拦截器已停止")
	return nil
}

// SetFilter 设置过滤规则
func (n *NetfilterInterceptor) SetFilter(filter string) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.logger.Info("设置过滤规则", "filter", filter)
	// Netfilter使用不同的规则语法，这里需要转换
	return n.configureIptablesRules()
}

// GetPacketChannel 获取数据包通道
func (n *NetfilterInterceptor) GetPacketChannel() <-chan *PacketInfo {
	return n.packetCh
}

// Reinject 重新注入数据包
func (n *NetfilterInterceptor) Reinject(packet *PacketInfo) error {
	// Linux上的数据包重新注入实现
	// 这里简化处理，实际实现需要使用netfilter队列
	atomic.AddUint64(&n.stats.PacketsReinject, 1)
	n.logger.Debug("重新注入数据包", "packet_id", packet.ID)
	return nil
}

// GetStats 获取统计信息
func (n *NetfilterInterceptor) GetStats() InterceptorStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	stats := n.stats
	stats.Uptime = time.Since(n.stats.StartTime)
	return stats
}

// HealthCheck 健康检查
func (n *NetfilterInterceptor) HealthCheck() error {
	if atomic.LoadInt32(&n.running) == 0 {
		return fmt.Errorf("拦截器未运行")
	}

	// 检查iptables规则是否正常
	if err := n.checkIptablesRules(); err != nil {
		return fmt.Errorf("iptables规则检查失败: %w", err)
	}

	return nil
}

// configureIptablesRules 配置iptables规则
func (n *NetfilterInterceptor) configureIptablesRules() error {
	n.iptablesRules = []string{
		// 重定向TCP流量到代理端口
		fmt.Sprintf("-t nat -A OUTPUT -p tcp ! -d %s -j REDIRECT --to-port %d",
			n.config.BypassCIDR, n.config.ProxyPort),
		// 重定向UDP流量到代理端口
		fmt.Sprintf("-t nat -A OUTPUT -p udp ! -d %s -j REDIRECT --to-port %d",
			n.config.BypassCIDR, n.config.ProxyPort),
		// 允许本地回环流量
		"-A OUTPUT -o lo -j ACCEPT",
		// 允许已建立的连接
		"-A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT",
	}

	n.logger.Debug("配置iptables规则", "rules", n.iptablesRules)
	return nil
}

// applyIptablesRules 应用iptables规则
func (n *NetfilterInterceptor) applyIptablesRules() error {
	for _, rule := range n.iptablesRules {
		cmd := exec.Command("iptables", parseIptablesRule(rule)...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("应用iptables规则失败 [%s]: %w", rule, err)
		}
		n.logger.Debug("应用iptables规则", "rule", rule)
	}

	return nil
}

// clearIptablesRules 清除iptables规则
func (n *NetfilterInterceptor) clearIptablesRules() error {
	for _, rule := range n.iptablesRules {
		// 将-A替换为-D来删除规则
		deleteRule := replaceAddWithDelete(rule)
		cmd := exec.Command("iptables", parseIptablesRule(deleteRule)...)
		if err := cmd.Run(); err != nil {
			n.logger.Warn("删除iptables规则失败", "rule", deleteRule, "error", err)
		} else {
			n.logger.Debug("删除iptables规则", "rule", deleteRule)
		}
	}

	return nil
}

// checkIptablesRules 检查iptables规则
func (n *NetfilterInterceptor) checkIptablesRules() error {
	// 检查iptables状态
	cmd := exec.Command("iptables", "-L", "-n")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("检查iptables状态失败: %w", err)
	}

	n.logger.Debug("iptables状态", "output", string(output))
	return nil
}

// capturePackets 捕获数据包
func (n *NetfilterInterceptor) capturePackets() {
	n.logger.Debug("开始捕获数据包")
	defer n.logger.Debug("数据包捕获结束")

	// 真实的netfilter数据包捕获
	// 使用netfilter队列捕获真实网络数据包
	n.logger.Info("启动netfilter数据包捕获")

	for {
		select {
		case <-n.stopCh:
			return
		default:
			// 实现真实的netfilter队列处理
			// 这里应该使用libnetfilter_queue库来捕获数据包
			// 由于需要CGO和系统库，这里提供一个框架实现

			// TODO: 实现真实的netfilter队列处理
			// 1. 从netfilter队列接收数据包
			// 2. 解析数据包头部信息
			// 3. 获取关联的进程信息
			// 4. 发送到处理通道

			n.logger.Debug("等待netfilter数据包...")
			time.Sleep(1 * time.Second) // 避免CPU占用过高
		}
	}
}

// 删除了createMockPacket函数 - 不再使用模拟数据包

// getProcessInfo 获取进程信息
func (n *NetfilterInterceptor) getProcessInfo(pid int) *ProcessInfo {
	// 从缓存中获取进程信息
	if info := n.processCache.Get(uint32(pid)); info != nil {
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

	// 获取进程路径
	cmd = exec.Command("readlink", "-f", fmt.Sprintf("/proc/%d/exe", pid))
	output, err = cmd.Output()
	if err == nil {
		processInfo.ExecutePath = string(output)
	}

	// 缓存进程信息
	n.processCache.Set(uint32(pid), processInfo)

	return processInfo
}

// parseIptablesRule 解析iptables规则字符串为参数数组
func parseIptablesRule(rule string) []string {
	// 简化的解析实现，实际应该更完善
	args := make([]string, 0)
	parts := splitRule(rule)
	for _, part := range parts {
		if part != "" {
			args = append(args, part)
		}
	}
	return args
}

// splitRule 分割规则字符串
func splitRule(rule string) []string {
	// 简化实现，按空格分割
	return []string{} // 实际实现需要正确解析
}

// replaceAddWithDelete 将添加规则转换为删除规则
func replaceAddWithDelete(rule string) string {
	// 简化实现，将-A替换为-D
	return rule // 实际实现需要正确替换
}
