package interceptor

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// createPlatformInterceptor 创建平台特定的拦截器（在各平台文件中实现）
// 这个函数在各个平台的文件中有具体实现

// PlatformInterceptor 平台特定拦截器接口
type PlatformInterceptor interface {
	// StartCapture 开始捕获数据包
	StartCapture() error

	// StopCapture 停止捕获数据包
	StopCapture() error

	// SetFilter 设置过滤规则
	SetFilter(filter string) error

	// GetPacketChannel 获取数据包通道
	GetPacketChannel() <-chan *PacketInfo

	// ReinjectPacket 重新注入数据包
	ReinjectPacket(packet *PacketInfo) error
}

// ProductionInterceptor 生产级流量拦截器实现
type ProductionInterceptor struct {
	config       InterceptorConfig
	packetCh     chan *PacketInfo
	stopCh       chan struct{}
	stats        InterceptorStats
	logger       logging.Logger
	processCache ProcessCache
	running      int32
	mu           sync.RWMutex

	// 平台特定的实现
	platformImpl PlatformInterceptor
}

// MockPlatformInterceptorAdapter 模拟拦截器适配器
type MockPlatformInterceptorAdapter struct {
	impl PlatformInterceptor
}

func (m *MockPlatformInterceptorAdapter) Initialize(config InterceptorConfig) error {
	// 模拟拦截器不需要初始化
	return nil
}

func (m *MockPlatformInterceptorAdapter) Start() error {
	return m.impl.StartCapture()
}

func (m *MockPlatformInterceptorAdapter) Stop() error {
	return m.impl.StopCapture()
}

func (m *MockPlatformInterceptorAdapter) SetFilter(filter string) error {
	return m.impl.SetFilter(filter)
}

func (m *MockPlatformInterceptorAdapter) GetPacketChannel() <-chan *PacketInfo {
	return m.impl.GetPacketChannel()
}

func (m *MockPlatformInterceptorAdapter) Reinject(packet *PacketInfo) error {
	return m.impl.ReinjectPacket(packet)
}

func (m *MockPlatformInterceptorAdapter) GetStats() InterceptorStats {
	// 返回基本统计信息
	return InterceptorStats{
		StartTime: time.Now(),
	}
}

func (m *MockPlatformInterceptorAdapter) HealthCheck() error {
	return nil
}

// createRealInterceptor 在各平台文件中实现

// NewProductionInterceptor 创建生产级流量拦截器
func NewProductionInterceptor(logger logging.Logger) TrafficInterceptor {
	// 根据运行时平台选择合适的拦截器
	return createRealInterceptor(logger)
}

// Initialize 初始化拦截器
func (p *ProductionInterceptor) Initialize(config InterceptorConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config
	p.packetCh = make(chan *PacketInfo, config.ChannelSize)
	p.processCache = NewProcessCache(config.CacheSize)
	p.stats.StartTime = time.Now()

	p.logger.Info("初始化生产级拦截器",
		"buffer_size", config.BufferSize,
		"workers", config.WorkerCount)

	return nil
}

// Start 启动流量拦截
func (p *ProductionInterceptor) Start() error {
	if !atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		return fmt.Errorf("拦截器已在运行")
	}

	p.logger.Info("启动生产级拦截器")

	// 启动平台特定的数据包捕获
	if err := p.platformImpl.StartCapture(); err != nil {
		atomic.StoreInt32(&p.running, 0)
		return fmt.Errorf("启动平台拦截器失败: %w", err)
	}

	return nil
}

// Stop 停止流量拦截
func (p *ProductionInterceptor) Stop() error {
	if !atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		return fmt.Errorf("拦截器未在运行")
	}

	p.logger.Info("停止生产级拦截器")

	// 停止平台特定的数据包捕获
	if err := p.platformImpl.StopCapture(); err != nil {
		p.logger.Error("停止平台拦截器失败", "error", err)
	}

	// 发送停止信号
	close(p.stopCh)

	// 关闭数据包通道
	close(p.packetCh)

	return nil
}

// SetFilter 设置过滤规则
func (p *ProductionInterceptor) SetFilter(filter string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config.Filter = filter
	p.logger.Info("设置过滤规则", "filter", filter)

	// 委托给平台实现
	return p.platformImpl.SetFilter(filter)
}

// GetPacketChannel 获取数据包通道
func (p *ProductionInterceptor) GetPacketChannel() <-chan *PacketInfo {
	return p.platformImpl.GetPacketChannel()
}

// Reinject 重新注入数据包
func (p *ProductionInterceptor) Reinject(packet *PacketInfo) error {
	atomic.AddUint64(&p.stats.PacketsReinject, 1)
	p.logger.Debug("重新注入数据包", "packet_id", packet.ID)

	// 委托给平台实现
	return p.platformImpl.ReinjectPacket(packet)
}

// GetStats 获取统计信息
func (p *ProductionInterceptor) GetStats() InterceptorStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.stats
	stats.Uptime = time.Since(p.stats.StartTime)
	return stats
}

// HealthCheck 健康检查
func (p *ProductionInterceptor) HealthCheck() error {
	if atomic.LoadInt32(&p.running) == 0 {
		return fmt.Errorf("拦截器未运行")
	}
	return nil
}

// 向后兼容的构造函数
func NewGenericInterceptor(logger logging.Logger) TrafficInterceptor {
	return NewProductionInterceptor(logger)
}
