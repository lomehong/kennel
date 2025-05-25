package interceptor

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/lomehong/kennel/pkg/logging"
)

// DefaultInterceptorFactory 默认拦截器工厂
type DefaultInterceptorFactory struct {
	logger logging.Logger
	mu     sync.RWMutex
}

// NewInterceptorFactory 创建拦截器工厂
func NewInterceptorFactory(logger logging.Logger) InterceptorFactory {
	return &DefaultInterceptorFactory{
		logger: logger,
	}
}

// CreateInterceptor 创建拦截器
func (f *DefaultInterceptorFactory) CreateInterceptor(platform string) (TrafficInterceptor, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// 如果没有指定平台，使用当前运行时平台
	if platform == "" {
		platform = runtime.GOOS
	}

	f.logger.Info("创建流量拦截器", "platform", platform)

	// 使用生产级拦截器
	return NewProductionInterceptor(f.logger), nil
}

// GetSupportedPlatforms 获取支持的平台
func (f *DefaultInterceptorFactory) GetSupportedPlatforms() []string {
	return []string{"windows", "darwin", "linux"}
}

// NewTrafficInterceptor 创建流量拦截器的便捷函数
func NewTrafficInterceptor(logger logging.Logger) (TrafficInterceptor, error) {
	factory := NewInterceptorFactory(logger)
	return factory.CreateInterceptor(runtime.GOOS)
}

// ProcessCacheImpl 进程缓存实现
type ProcessCacheImpl struct {
	cache map[uint32]*ProcessInfo
	mu    sync.RWMutex
	size  int
}

// NewProcessCache 创建进程缓存
func NewProcessCache(size int) ProcessCache {
	return &ProcessCacheImpl{
		cache: make(map[uint32]*ProcessInfo),
		size:  size,
	}
}

// Get 获取进程信息
func (c *ProcessCacheImpl) Get(pid uint32) *ProcessInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if info, exists := c.cache[pid]; exists {
		return info
	}
	return nil
}

// Set 设置进程信息
func (c *ProcessCacheImpl) Set(pid uint32, info *ProcessInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果缓存已满，删除一个旧条目
	if len(c.cache) >= c.size {
		// 简单的FIFO策略，删除第一个找到的条目
		for k := range c.cache {
			delete(c.cache, k)
			break
		}
	}

	c.cache[pid] = info
}

// Delete 删除进程信息
func (c *ProcessCacheImpl) Delete(pid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, pid)
}

// Clear 清空缓存
func (c *ProcessCacheImpl) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[uint32]*ProcessInfo)
}

// Size 获取缓存大小
func (c *ProcessCacheImpl) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// PacketFilterImpl 数据包过滤器实现
type PacketFilterImpl struct {
	rules  []string
	logger logging.Logger
	mu     sync.RWMutex
}

// NewPacketFilter 创建数据包过滤器
func NewPacketFilter(logger logging.Logger) PacketFilter {
	return &PacketFilterImpl{
		rules:  make([]string, 0),
		logger: logger,
	}
}

// Match 检查数据包是否匹配过滤条件
func (f *PacketFilterImpl) Match(packet *PacketInfo) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// 如果没有规则，默认匹配所有
	if len(f.rules) == 0 {
		return true
	}

	// 这里可以实现更复杂的匹配逻辑
	// 目前简化处理，只检查基本条件
	for _, rule := range f.rules {
		if f.matchRule(packet, rule) {
			return true
		}
	}

	return false
}

// matchRule 匹配单个规则
func (f *PacketFilterImpl) matchRule(packet *PacketInfo, rule string) bool {
	// 简化的规则匹配实现
	// 实际实现中应该支持更复杂的规则语法
	switch rule {
	case "tcp":
		return packet.Protocol == ProtocolTCP
	case "udp":
		return packet.Protocol == ProtocolUDP
	case "outbound":
		return packet.Direction == PacketDirectionOutbound
	case "inbound":
		return packet.Direction == PacketDirectionInbound
	default:
		return true
	}
}

// SetRules 设置过滤规则
func (f *PacketFilterImpl) SetRules(rules []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.rules = make([]string, len(rules))
	copy(f.rules, rules)

	f.logger.Info("设置数据包过滤规则", "rules", rules)
	return nil
}

// GetRules 获取过滤规则
func (f *PacketFilterImpl) GetRules() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	rules := make([]string, len(f.rules))
	copy(rules, f.rules)
	return rules
}

// InterceptorManagerImpl 拦截器管理器实现
type InterceptorManagerImpl struct {
	interceptors map[string]TrafficInterceptor
	logger       logging.Logger
	mu           sync.RWMutex
}

// NewInterceptorManager 创建拦截器管理器
func NewInterceptorManager(logger logging.Logger) InterceptorManager {
	return &InterceptorManagerImpl{
		interceptors: make(map[string]TrafficInterceptor),
		logger:       logger,
	}
}

// RegisterInterceptor 注册拦截器
func (m *InterceptorManagerImpl) RegisterInterceptor(name string, interceptor TrafficInterceptor) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.interceptors[name]; exists {
		return fmt.Errorf("拦截器已存在: %s", name)
	}

	m.interceptors[name] = interceptor
	m.logger.Info("注册拦截器", "name", name)
	return nil
}

// GetInterceptor 获取拦截器
func (m *InterceptorManagerImpl) GetInterceptor(name string) (TrafficInterceptor, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	interceptor, exists := m.interceptors[name]
	return interceptor, exists
}

// StartAll 启动所有拦截器
func (m *InterceptorManagerImpl) StartAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, interceptor := range m.interceptors {
		if err := interceptor.Start(); err != nil {
			m.logger.Error("启动拦截器失败", "name", name, "error", err)
			return fmt.Errorf("启动拦截器 %s 失败: %w", name, err)
		}
		m.logger.Info("拦截器已启动", "name", name)
	}

	return nil
}

// StopAll 停止所有拦截器
func (m *InterceptorManagerImpl) StopAll() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var lastErr error
	for name, interceptor := range m.interceptors {
		if err := interceptor.Stop(); err != nil {
			m.logger.Error("停止拦截器失败", "name", name, "error", err)
			lastErr = err
		} else {
			m.logger.Info("拦截器已停止", "name", name)
		}
	}

	return lastErr
}

// GetStats 获取所有拦截器统计信息
func (m *InterceptorManagerImpl) GetStats() map[string]InterceptorStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]InterceptorStats)
	for name, interceptor := range m.interceptors {
		stats[name] = interceptor.GetStats()
	}

	return stats
}
