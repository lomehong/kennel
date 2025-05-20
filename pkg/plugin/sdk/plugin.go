package sdk

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// BasePlugin 提供了Plugin接口的基础实现
// 可以被具体的插件实现嵌入以减少重复代码
type BasePlugin struct {
	// 插件信息
	info api.PluginInfo

	// 插件配置
	config api.PluginConfig

	// 插件状态
	state api.PluginState

	// 日志记录器
	logger hclog.Logger

	// 健康状态
	health api.HealthStatus

	// 启动时间
	startTime time.Time

	// 停止时间
	stopTime time.Time

	// 错误信息
	lastError error

	// 统计信息
	stats map[string]interface{}

	// 互斥锁
	mu sync.RWMutex
}

// NewBasePlugin 创建一个新的基础插件
func NewBasePlugin(info api.PluginInfo, logger hclog.Logger) *BasePlugin {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	return &BasePlugin{
		info:   info,
		state:  api.PluginStateUnknown,
		logger: logger.Named(info.ID),
		health: api.HealthStatus{
			Status:      "unknown",
			Details:     make(map[string]interface{}),
			LastChecked: time.Now(),
		},
		stats: make(map[string]interface{}),
	}
}

// GetInfo 返回插件信息
func (p *BasePlugin) GetInfo() api.PluginInfo {
	return p.info
}

// Init 初始化插件
func (p *BasePlugin) Init(ctx context.Context, config api.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("初始化插件", "id", p.info.ID)
	p.config = config
	p.state = api.PluginStateInitialized
	return nil
}

// Start 启动插件
func (p *BasePlugin) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("启动插件", "id", p.info.ID)
	p.state = api.PluginStateRunning
	p.startTime = time.Now()
	return nil
}

// Stop 停止插件
func (p *BasePlugin) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("停止插件", "id", p.info.ID)
	p.state = api.PluginStateStopped
	p.stopTime = time.Now()
	return nil
}

// HealthCheck 执行健康检查
func (p *BasePlugin) HealthCheck(ctx context.Context) (api.HealthStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 更新最后检查时间
	p.health.LastChecked = time.Now()

	// 根据状态设置健康状态
	if p.state == api.PluginStateRunning {
		p.health.Status = "healthy"
	} else if p.state == api.PluginStateFailed {
		p.health.Status = "unhealthy"
	} else {
		p.health.Status = "unknown"
	}

	return p.health, nil
}

// GetState 获取插件状态
func (p *BasePlugin) GetState() api.PluginState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// SetState 设置插件状态
func (p *BasePlugin) SetState(state api.PluginState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
}

// GetConfig 获取插件配置
func (p *BasePlugin) GetConfig() api.PluginConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// SetConfig 设置插件配置
func (p *BasePlugin) SetConfig(config api.PluginConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
}

// GetLogger 获取日志记录器
func (p *BasePlugin) GetLogger() hclog.Logger {
	return p.logger
}

// SetLogger 设置日志记录器
func (p *BasePlugin) SetLogger(logger hclog.Logger) {
	if logger == nil {
		return
	}
	p.logger = logger.Named(p.info.ID)
}

// GetStartTime 获取启动时间
func (p *BasePlugin) GetStartTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.startTime
}

// GetStopTime 获取停止时间
func (p *BasePlugin) GetStopTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stopTime
}

// GetLastError 获取最后一个错误
func (p *BasePlugin) GetLastError() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastError
}

// SetLastError 设置最后一个错误
func (p *BasePlugin) SetLastError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastError = err
}

// GetStats 获取统计信息
func (p *BasePlugin) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 复制统计信息
	stats := make(map[string]interface{}, len(p.stats))
	for k, v := range p.stats {
		stats[k] = v
	}

	// 添加基本信息
	stats["state"] = p.state
	stats["start_time"] = p.startTime
	stats["uptime"] = time.Since(p.startTime).String()
	stats["health"] = p.health.Status

	return stats
}

// SetStat 设置统计信息
func (p *BasePlugin) SetStat(key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats[key] = value
}

// IncrementStat 增加统计计数
func (p *BasePlugin) IncrementStat(key string, delta int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 获取当前值
	current, ok := p.stats[key]
	if !ok {
		// 如果不存在，初始化为delta
		p.stats[key] = delta
		return
	}

	// 根据类型增加值
	switch v := current.(type) {
	case int:
		p.stats[key] = v + int(delta)
	case int64:
		p.stats[key] = v + delta
	case float64:
		p.stats[key] = v + float64(delta)
	default:
		// 如果类型不匹配，重置为delta
		p.stats[key] = delta
	}
}
