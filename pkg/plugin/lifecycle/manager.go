package lifecycle

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// PluginLifecycleManager 插件生命周期管理器
// 管理插件的生命周期状态
type PluginLifecycleManager struct {
	// 插件实例
	plugin api.Plugin

	// 插件配置
	config api.PluginConfig

	// 当前状态
	state api.PluginState

	// 日志记录器
	logger hclog.Logger

	// 状态变更监听器
	listeners []StateChangeListener

	// 互斥锁
	mu sync.RWMutex

	// 上下文
	ctx context.Context

	// 取消函数
	cancel context.CancelFunc

	// 生命周期钩子
	hooks PluginLifecycleHooks
}

// StateChangeListener 状态变更监听器
type StateChangeListener func(oldState, newState api.PluginState, metadata map[string]interface{})

// PluginLifecycleHooks 插件生命周期钩子
type PluginLifecycleHooks interface {
	// BeforeInit 初始化前
	BeforeInit(ctx context.Context, config api.PluginConfig) error

	// AfterInit 初始化后
	AfterInit(ctx context.Context) error

	// BeforeStart 启动前
	BeforeStart(ctx context.Context) error

	// AfterStart 启动后
	AfterStart(ctx context.Context) error

	// BeforeStop 停止前
	BeforeStop(ctx context.Context) error

	// AfterStop 停止后
	AfterStop(ctx context.Context) error
}

// DefaultPluginLifecycleHooks 默认插件生命周期钩子
type DefaultPluginLifecycleHooks struct {
	// 日志记录器
	logger hclog.Logger
}

// BeforeInit 初始化前
func (h *DefaultPluginLifecycleHooks) BeforeInit(ctx context.Context, config api.PluginConfig) error {
	h.logger.Debug("插件初始化前")
	return nil
}

// AfterInit 初始化后
func (h *DefaultPluginLifecycleHooks) AfterInit(ctx context.Context) error {
	h.logger.Debug("插件初始化后")
	return nil
}

// BeforeStart 启动前
func (h *DefaultPluginLifecycleHooks) BeforeStart(ctx context.Context) error {
	h.logger.Debug("插件启动前")
	return nil
}

// AfterStart 启动后
func (h *DefaultPluginLifecycleHooks) AfterStart(ctx context.Context) error {
	h.logger.Debug("插件启动后")
	return nil
}

// BeforeStop 停止前
func (h *DefaultPluginLifecycleHooks) BeforeStop(ctx context.Context) error {
	h.logger.Debug("插件停止前")
	return nil
}

// AfterStop 停止后
func (h *DefaultPluginLifecycleHooks) AfterStop(ctx context.Context) error {
	h.logger.Debug("插件停止后")
	return nil
}

// NewPluginLifecycleManager 创建一个新的插件生命周期管理器
func NewPluginLifecycleManager(plugin api.Plugin, config api.PluginConfig, logger hclog.Logger) *PluginLifecycleManager {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PluginLifecycleManager{
		plugin: plugin,
		config: config,
		state:  api.PluginStateUnknown,
		logger: logger.Named(fmt.Sprintf("lifecycle-%s", plugin.GetInfo().ID)),
		ctx:    ctx,
		cancel: cancel,
		hooks:  &DefaultPluginLifecycleHooks{logger: logger},
	}
}

// SetHooks 设置生命周期钩子
func (m *PluginLifecycleManager) SetHooks(hooks PluginLifecycleHooks) {
	if hooks == nil {
		return
	}
	m.hooks = hooks
}

// AddStateChangeListener 添加状态变更监听器
func (m *PluginLifecycleManager) AddStateChangeListener(listener StateChangeListener) {
	if listener == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners = append(m.listeners, listener)
}

// GetCurrentState 获取当前状态
func (m *PluginLifecycleManager) GetCurrentState() api.PluginState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// IsActive 是否处于活动状态
func (m *PluginLifecycleManager) IsActive() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state == api.PluginStateRunning
}

// IsHealthy 是否处于健康状态
func (m *PluginLifecycleManager) IsHealthy() bool {
	// 执行健康检查
	status, err := m.plugin.HealthCheck(m.ctx)
	if err != nil {
		m.logger.Warn("健康检查失败", "error", err)
		return false
	}

	return status.Status == "healthy"
}

// TransitionTo 转换到指定状态
func (m *PluginLifecycleManager) TransitionTo(state api.PluginState, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查状态转换是否有效
	if !m.isValidTransition(m.state, state) {
		return fmt.Errorf("无效的状态转换: %s -> %s", m.state, state)
	}

	// 记录旧状态
	oldState := m.state

	// 更新状态
	m.state = state
	m.logger.Info("插件状态变更", "id", m.plugin.GetInfo().ID, "old_state", oldState, "new_state", state)

	// 通知监听器
	m.notifyListeners(oldState, state, metadata)

	return nil
}

// isValidTransition 检查状态转换是否有效
func (m *PluginLifecycleManager) isValidTransition(from, to api.PluginState) bool {
	// 定义有效的状态转换
	validTransitions := map[api.PluginState][]api.PluginState{
		api.PluginStateUnknown:     {api.PluginStateDiscovered, api.PluginStateRegistered},
		api.PluginStateDiscovered:  {api.PluginStateRegistered, api.PluginStateUnloaded},
		api.PluginStateRegistered:  {api.PluginStateLoaded, api.PluginStateUnloaded},
		api.PluginStateLoaded:      {api.PluginStateInitialized, api.PluginStateUnloading, api.PluginStateFailed},
		api.PluginStateInitialized: {api.PluginStateStarting, api.PluginStateUnloading, api.PluginStateFailed},
		api.PluginStateStarting:    {api.PluginStateRunning, api.PluginStateFailed},
		api.PluginStateRunning:     {api.PluginStateStopping, api.PluginStateFailed},
		api.PluginStateStopping:    {api.PluginStateStopped, api.PluginStateFailed},
		api.PluginStateStopped:     {api.PluginStateStarting, api.PluginStateUnloading},
		api.PluginStateFailed:      {api.PluginStateUnloading, api.PluginStateStarting},
		api.PluginStateUnloading:   {api.PluginStateUnloaded},
		api.PluginStateUnloaded:    {api.PluginStateRegistered},
	}

	// 检查是否有效
	for _, validTo := range validTransitions[from] {
		if validTo == to {
			return true
		}
	}

	return false
}

// notifyListeners 通知监听器
func (m *PluginLifecycleManager) notifyListeners(oldState, newState api.PluginState, metadata map[string]interface{}) {
	// 复制监听器列表
	listeners := make([]StateChangeListener, len(m.listeners))
	copy(listeners, m.listeners)

	// 通知监听器
	for _, listener := range listeners {
		go listener(oldState, newState, metadata)
	}
}

// Init 初始化插件
func (m *PluginLifecycleManager) Init(ctx context.Context) error {
	// 检查当前状态
	if m.GetCurrentState() != api.PluginStateLoaded {
		return fmt.Errorf("插件状态错误: %s，无法初始化", m.GetCurrentState())
	}

	// 转换到初始化中状态
	if err := m.TransitionTo(api.PluginStateInitializing, nil); err != nil {
		return err
	}

	// 调用初始化前钩子
	if err := m.hooks.BeforeInit(ctx, m.config); err != nil {
		m.TransitionTo(api.PluginStateFailed, map[string]interface{}{
			"error": err.Error(),
			"phase": "before_init",
		})
		return fmt.Errorf("初始化前钩子失败: %w", err)
	}

	// 初始化插件
	if err := m.plugin.Init(ctx, m.config); err != nil {
		m.TransitionTo(api.PluginStateFailed, map[string]interface{}{
			"error": err.Error(),
			"phase": "init",
		})
		return fmt.Errorf("插件初始化失败: %w", err)
	}

	// 调用初始化后钩子
	if err := m.hooks.AfterInit(ctx); err != nil {
		m.TransitionTo(api.PluginStateFailed, map[string]interface{}{
			"error": err.Error(),
			"phase": "after_init",
		})
		return fmt.Errorf("初始化后钩子失败: %w", err)
	}

	// 转换到已初始化状态
	return m.TransitionTo(api.PluginStateInitialized, nil)
}

// Start 启动插件
func (m *PluginLifecycleManager) Start(ctx context.Context) error {
	// 检查当前状态
	currentState := m.GetCurrentState()
	if currentState != api.PluginStateInitialized && currentState != api.PluginStateStopped {
		return fmt.Errorf("插件状态错误: %s，无法启动", currentState)
	}

	// 转换到启动中状态
	if err := m.TransitionTo(api.PluginStateStarting, nil); err != nil {
		return err
	}

	// 调用启动前钩子
	if err := m.hooks.BeforeStart(ctx); err != nil {
		m.TransitionTo(api.PluginStateFailed, map[string]interface{}{
			"error": err.Error(),
			"phase": "before_start",
		})
		return fmt.Errorf("启动前钩子失败: %w", err)
	}

	// 启动插件
	if err := m.plugin.Start(ctx); err != nil {
		m.TransitionTo(api.PluginStateFailed, map[string]interface{}{
			"error": err.Error(),
			"phase": "start",
		})
		return fmt.Errorf("插件启动失败: %w", err)
	}

	// 调用启动后钩子
	if err := m.hooks.AfterStart(ctx); err != nil {
		m.TransitionTo(api.PluginStateFailed, map[string]interface{}{
			"error": err.Error(),
			"phase": "after_start",
		})
		return fmt.Errorf("启动后钩子失败: %w", err)
	}

	// 转换到运行中状态
	return m.TransitionTo(api.PluginStateRunning, nil)
}

// Stop 停止插件
func (m *PluginLifecycleManager) Stop(ctx context.Context) error {
	// 检查当前状态
	if m.GetCurrentState() != api.PluginStateRunning {
		return fmt.Errorf("插件状态错误: %s，无法停止", m.GetCurrentState())
	}

	// 转换到停止中状态
	if err := m.TransitionTo(api.PluginStateStopping, nil); err != nil {
		return err
	}

	// 调用停止前钩子
	if err := m.hooks.BeforeStop(ctx); err != nil {
		m.logger.Warn("停止前钩子失败", "error", err)
		// 继续停止过程
	}

	// 停止插件
	if err := m.plugin.Stop(ctx); err != nil {
		m.TransitionTo(api.PluginStateFailed, map[string]interface{}{
			"error": err.Error(),
			"phase": "stop",
		})
		return fmt.Errorf("插件停止失败: %w", err)
	}

	// 调用停止后钩子
	if err := m.hooks.AfterStop(ctx); err != nil {
		m.logger.Warn("停止后钩子失败", "error", err)
		// 继续停止过程
	}

	// 转换到已停止状态
	return m.TransitionTo(api.PluginStateStopped, nil)
}

// Close 关闭生命周期管理器
func (m *PluginLifecycleManager) Close() error {
	m.cancel()
	return nil
}
