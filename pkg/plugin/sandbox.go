package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/errors"
)

// PluginSandbox 插件沙箱
type PluginSandbox struct {
	pluginID         string
	isolator         *PluginIsolator
	logger           hclog.Logger
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.RWMutex
	state            PluginState
	startTime        time.Time
	lastActivityTime time.Time
	callCount        int64
	errorCount       int64
	panicCount       int64
	totalExecTime    time.Duration
}

// PluginState 插件状态
type PluginState int

// 预定义插件状态
const (
	PluginStateUnknown PluginState = iota
	PluginStateInitializing
	PluginStateRunning
	PluginStatePaused
	PluginStateStopped
	PluginStateError
)

// String 返回插件状态的字符串表示
func (ps PluginState) String() string {
	switch ps {
	case PluginStateUnknown:
		return "Unknown"
	case PluginStateInitializing:
		return "Initializing"
	case PluginStateRunning:
		return "Running"
	case PluginStatePaused:
		return "Paused"
	case PluginStateStopped:
		return "Stopped"
	case PluginStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// PluginSandboxOption 插件沙箱配置选项
type PluginSandboxOption func(*PluginSandbox)

// WithSandboxLogger 设置日志记录器
func WithSandboxLogger(logger hclog.Logger) PluginSandboxOption {
	return func(ps *PluginSandbox) {
		ps.logger = logger
	}
}

// WithSandboxContext 设置上下文
func WithSandboxContext(ctx context.Context) PluginSandboxOption {
	return func(ps *PluginSandbox) {
		if ps.cancel != nil {
			ps.cancel()
		}
		ps.ctx, ps.cancel = context.WithCancel(ctx)
	}
}

// NewPluginSandbox 创建一个新的插件沙箱
func NewPluginSandbox(pluginID string, isolator *PluginIsolator, options ...PluginSandboxOption) *PluginSandbox {
	ctx, cancel := context.WithCancel(context.Background())

	ps := &PluginSandbox{
		pluginID:         pluginID,
		isolator:         isolator,
		logger:           hclog.NewNullLogger(),
		ctx:              ctx,
		cancel:           cancel,
		state:            PluginStateInitializing,
		startTime:        time.Now(),
		lastActivityTime: time.Now(),
	}

	// 应用选项
	for _, option := range options {
		option(ps)
	}

	return ps
}

// Execute 在沙箱中执行函数
func (ps *PluginSandbox) Execute(f func() error) error {
	ps.mu.Lock()
	if ps.state != PluginStateRunning && ps.state != PluginStateInitializing {
		ps.mu.Unlock()
		return fmt.Errorf("插件 %s 不在运行状态，当前状态: %s", ps.pluginID, ps.state)
	}
	ps.state = PluginStateRunning
	ps.callCount++
	ps.lastActivityTime = time.Now()
	ps.mu.Unlock()

	startTime := time.Now()

	// 在隔离环境中执行函数
	err := ps.isolator.ExecuteFunc(ps.pluginID, f)

	// 更新统计信息
	execTime := time.Since(startTime)
	ps.mu.Lock()
	ps.totalExecTime += execTime
	if err != nil {
		ps.errorCount++
		if errors.IsType(err, errors.ErrorTypeCritical) {
			ps.state = PluginStateError
		}
	}
	ps.mu.Unlock()

	return err
}

// ExecuteWithContext 在沙箱中执行带上下文的函数
func (ps *PluginSandbox) ExecuteWithContext(ctx context.Context, f func(context.Context) error) error {
	return ps.Execute(func() error {
		return f(ctx)
	})
}

// GetState 获取插件状态
func (ps *PluginSandbox) GetState() PluginState {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.state
}

// SetState 设置插件状态
func (ps *PluginSandbox) SetState(state PluginState) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.state = state
	ps.lastActivityTime = time.Now()
}

// GetStats 获取统计信息
func (ps *PluginSandbox) GetStats() map[string]interface{} {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var avgExecTime time.Duration
	if ps.callCount > 0 {
		avgExecTime = ps.totalExecTime / time.Duration(ps.callCount)
	}

	return map[string]interface{}{
		"plugin_id":          ps.pluginID,
		"state":              ps.state.String(),
		"start_time":         ps.startTime,
		"uptime":             time.Since(ps.startTime).String(),
		"last_activity_time": ps.lastActivityTime,
		"call_count":         ps.callCount,
		"error_count":        ps.errorCount,
		"panic_count":        ps.panicCount,
		"total_exec_time":    ps.totalExecTime.String(),
		"avg_exec_time":      avgExecTime.String(),
	}
}

// Stop 停止插件沙箱
func (ps *PluginSandbox) Stop() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.state = PluginStateStopped
	ps.cancel()
	ps.lastActivityTime = time.Now()
}

// Pause 暂停插件沙箱
func (ps *PluginSandbox) Pause() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.state = PluginStatePaused
	ps.lastActivityTime = time.Now()
}

// Resume 恢复插件沙箱
func (ps *PluginSandbox) Resume() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.state = PluginStateRunning
	ps.lastActivityTime = time.Now()
}

// Reset 重置插件沙箱
func (ps *PluginSandbox) Reset() {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// 取消旧的上下文
	ps.cancel()

	// 创建新的上下文
	ps.ctx, ps.cancel = context.WithCancel(context.Background())

	// 重置状态
	ps.state = PluginStateInitializing
	ps.lastActivityTime = time.Now()
	ps.callCount = 0
	ps.errorCount = 0
	ps.panicCount = 0
	ps.totalExecTime = 0
}

// IsHealthy 检查插件是否健康
func (ps *PluginSandbox) IsHealthy() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.state == PluginStateRunning || ps.state == PluginStateInitializing
}

// GetID 获取插件ID
func (ps *PluginSandbox) GetID() string {
	return ps.pluginID
}

// GetUptime 获取插件运行时间
func (ps *PluginSandbox) GetUptime() time.Duration {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return time.Since(ps.startTime)
}

// GetLastActivityTime 获取最后活动时间
func (ps *PluginSandbox) GetLastActivityTime() time.Time {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.lastActivityTime
}

// IsIdle 检查插件是否空闲
func (ps *PluginSandbox) IsIdle(duration time.Duration) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return time.Since(ps.lastActivityTime) > duration
}
