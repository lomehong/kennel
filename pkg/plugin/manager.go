package plugin

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/lomehong/kennel/pkg/concurrency"
	"github.com/lomehong/kennel/pkg/errors"
	"github.com/lomehong/kennel/pkg/resource"
)

// PluginManager 插件管理器
type PluginManager struct {
	plugins             map[string]*ManagedPlugin
	sandboxes           map[string]*PluginSandbox
	isolator            *PluginIsolator
	logger              hclog.Logger
	pluginsDir          string
	resourceTracker     *resource.ResourceTracker
	workerpool          *concurrency.WorkerPool
	errorRegistry       *errors.ErrorHandlerRegistry
	recoveryManager     *errors.RecoveryManager
	ctx                 context.Context
	cancel              context.CancelFunc
	mu                  sync.RWMutex
	healthCheckInterval time.Duration
	idleTimeout         time.Duration
}

// ManagedPlugin 受管理的插件
type ManagedPlugin struct {
	ID        string
	Name      string
	Version   string
	Path      string
	Client    *plugin.Client
	Interface interface{}
	Sandbox   *PluginSandbox
	Config    *PluginConfig
	State     PluginState
	LastError error
	StartTime time.Time
	StopTime  time.Time
}

// PluginConfig 插件配置
type PluginConfig struct {
	ID             string
	Name           string
	Version        string
	Path           string
	IsolationLevel IsolationLevel
	AutoStart      bool
	AutoRestart    bool
	Enabled        bool
	Dependencies   []string
	ResourceLimits map[string]int
	Environment    map[string]string
	Args           []string
	Timeout        time.Duration
}

// PluginManagerOption 插件管理器配置选项
type PluginManagerOption func(*PluginManager)

// WithPluginManagerLogger 设置日志记录器
func WithPluginManagerLogger(logger hclog.Logger) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.logger = logger
	}
}

// WithPluginsDir 设置插件目录
func WithPluginsDir(dir string) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.pluginsDir = dir
	}
}

// WithPluginManagerResourceTracker 设置资源追踪器
func WithPluginManagerResourceTracker(tracker *resource.ResourceTracker) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.resourceTracker = tracker
	}
}

// WithPluginManagerWorkerPool 设置工作池
func WithPluginManagerWorkerPool(pool *concurrency.WorkerPool) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.workerpool = pool
	}
}

// WithPluginManagerErrorRegistry 设置错误处理器注册表
func WithPluginManagerErrorRegistry(registry *errors.ErrorHandlerRegistry) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.errorRegistry = registry
	}
}

// WithPluginManagerRecoveryManager 设置恢复管理器
func WithPluginManagerRecoveryManager(manager *errors.RecoveryManager) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.recoveryManager = manager
	}
}

// WithPluginManagerContext 设置上下文
func WithPluginManagerContext(ctx context.Context) PluginManagerOption {
	return func(pm *PluginManager) {
		if pm.cancel != nil {
			pm.cancel()
		}
		pm.ctx, pm.cancel = context.WithCancel(ctx)
	}
}

// WithHealthCheckInterval 设置健康检查间隔
func WithHealthCheckInterval(interval time.Duration) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.healthCheckInterval = interval
	}
}

// WithIdleTimeout 设置空闲超时
func WithIdleTimeout(timeout time.Duration) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.idleTimeout = timeout
	}
}

// NewPluginManager 创建一个新的插件管理器
func NewPluginManager(options ...PluginManagerOption) *PluginManager {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &PluginManager{
		plugins:             make(map[string]*ManagedPlugin),
		sandboxes:           make(map[string]*PluginSandbox),
		logger:              hclog.NewNullLogger(),
		pluginsDir:          "plugins",
		ctx:                 ctx,
		cancel:              cancel,
		healthCheckInterval: 30 * time.Second,
		idleTimeout:         10 * time.Minute,
	}

	// 应用选项
	for _, option := range options {
		option(pm)
	}

	// 创建插件隔离器
	isolationConfig := DefaultPluginIsolationConfig()
	pm.isolator = NewPluginIsolator(isolationConfig,
		WithLogger(pm.logger.Named("isolator")),
		WithResourceTracker(pm.resourceTracker),
		WithWorkerPool(pm.workerpool),
		WithErrorRegistry(pm.errorRegistry),
		WithRecoveryManager(pm.recoveryManager),
		WithContext(pm.ctx),
	)

	return pm
}

// LoadPlugin 加载插件
func (pm *PluginManager) LoadPlugin(config *PluginConfig) (*ManagedPlugin, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 检查插件是否已加载
	if _, exists := pm.plugins[config.ID]; exists {
		return nil, fmt.Errorf("插件 %s 已加载", config.ID)
	}

	// 创建插件沙箱
	sandbox := NewPluginSandbox(config.ID, pm.isolator,
		WithSandboxLogger(pm.logger.Named(fmt.Sprintf("sandbox-%s", config.ID))),
		WithSandboxContext(pm.ctx),
	)

	// 创建受管理的插件
	managedPlugin := &ManagedPlugin{
		ID:        config.ID,
		Name:      config.Name,
		Version:   config.Version,
		Path:      filepath.Join(pm.pluginsDir, config.Path),
		Sandbox:   sandbox,
		Config:    config,
		State:     PluginStateInitializing,
		StartTime: time.Now(),
	}

	// 存储插件
	pm.plugins[config.ID] = managedPlugin
	pm.sandboxes[config.ID] = sandbox

	pm.logger.Info("插件已加载", "id", config.ID, "name", config.Name, "version", config.Version)

	// 如果配置为自动启动，则启动插件
	if config.AutoStart {
		go func() {
			if err := pm.StartPlugin(config.ID); err != nil {
				pm.logger.Error("自动启动插件失败", "id", config.ID, "error", err)
			}
		}()
	}

	return managedPlugin, nil
}

// StartPlugin 启动插件
func (pm *PluginManager) StartPlugin(id string) error {
	pm.mu.Lock()
	plugin, exists := pm.plugins[id]
	if !exists {
		pm.mu.Unlock()
		return fmt.Errorf("插件 %s 不存在", id)
	}

	// 检查插件状态
	if plugin.State == PluginStateRunning {
		pm.mu.Unlock()
		return fmt.Errorf("插件 %s 已在运行", id)
	}

	// 更新插件状态
	plugin.State = PluginStateInitializing
	plugin.StartTime = time.Now()
	pm.mu.Unlock()

	// 启动插件（这里简化为设置状态）
	// 在实际实现中，这里应该启动插件进程
	plugin.Sandbox.SetState(PluginStateRunning)
	plugin.State = PluginStateRunning

	pm.logger.Info("插件已启动", "id", id)
	return nil
}

// StopPlugin 停止插件
func (pm *PluginManager) StopPlugin(id string) error {
	pm.mu.Lock()
	plugin, exists := pm.plugins[id]
	if !exists {
		pm.mu.Unlock()
		return fmt.Errorf("插件 %s 不存在", id)
	}

	// 检查插件状态
	if plugin.State != PluginStateRunning {
		pm.mu.Unlock()
		return fmt.Errorf("插件 %s 未在运行", id)
	}

	// 更新插件状态
	plugin.State = PluginStateStopped
	plugin.StopTime = time.Now()
	pm.mu.Unlock()

	// 停止插件沙箱
	plugin.Sandbox.Stop()

	pm.logger.Info("插件已停止", "id", id)
	return nil
}

// RestartPlugin 重启插件
func (pm *PluginManager) RestartPlugin(id string) error {
	// 停止插件
	if err := pm.StopPlugin(id); err != nil {
		return fmt.Errorf("停止插件失败: %w", err)
	}

	// 启动插件
	if err := pm.StartPlugin(id); err != nil {
		return fmt.Errorf("启动插件失败: %w", err)
	}

	pm.logger.Info("插件已重启", "id", id)
	return nil
}

// UnloadPlugin 卸载插件
func (pm *PluginManager) UnloadPlugin(id string) error {
	pm.mu.Lock()
	plugin, exists := pm.plugins[id]
	if !exists {
		pm.mu.Unlock()
		return fmt.Errorf("插件 %s 不存在", id)
	}

	// 如果插件正在运行，先停止它
	if plugin.State == PluginStateRunning {
		pm.mu.Unlock()
		if err := pm.StopPlugin(id); err != nil {
			return fmt.Errorf("停止插件失败: %w", err)
		}
		pm.mu.Lock()
	}

	// 删除插件
	delete(pm.plugins, id)
	delete(pm.sandboxes, id)
	pm.mu.Unlock()

	pm.logger.Info("插件已卸载", "id", id)
	return nil
}

// GetPlugin 获取插件
func (pm *PluginManager) GetPlugin(id string) (*ManagedPlugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	plugin, exists := pm.plugins[id]
	return plugin, exists
}

// ListPlugins 列出所有插件
func (pm *PluginManager) ListPlugins() []*ManagedPlugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	plugins := make([]*ManagedPlugin, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// ExecutePluginFunc 在插件沙箱中执行函数
func (pm *PluginManager) ExecutePluginFunc(id string, f func() error) error {
	pm.mu.RLock()
	sandbox, exists := pm.sandboxes[id]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("插件 %s 不存在", id)
	}

	return sandbox.Execute(f)
}

// ExecutePluginFuncWithContext 在插件沙箱中执行带上下文的函数
func (pm *PluginManager) ExecutePluginFuncWithContext(id string, ctx context.Context, f func(context.Context) error) error {
	pm.mu.RLock()
	sandbox, exists := pm.sandboxes[id]
	pm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("插件 %s 不存在", id)
	}

	return sandbox.ExecuteWithContext(ctx, f)
}

// StartHealthCheck 启动健康检查
func (pm *PluginManager) StartHealthCheck() {
	go func() {
		ticker := time.NewTicker(pm.healthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pm.checkPluginsHealth()
			case <-pm.ctx.Done():
				return
			}
		}
	}()
}

// checkPluginsHealth 检查插件健康状态
func (pm *PluginManager) checkPluginsHealth() {
	pm.mu.RLock()
	plugins := make([]*ManagedPlugin, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}
	pm.mu.RUnlock()

	for _, plugin := range plugins {
		// 检查插件是否健康
		if plugin.State == PluginStateRunning && !plugin.Sandbox.IsHealthy() {
			pm.logger.Warn("插件不健康，尝试重启", "id", plugin.ID)
			if plugin.Config.AutoRestart {
				go func(id string) {
					if err := pm.RestartPlugin(id); err != nil {
						pm.logger.Error("重启插件失败", "id", id, "error", err)
					}
				}(plugin.ID)
			}
		}

		// 检查插件是否空闲
		if plugin.State == PluginStateRunning && plugin.Sandbox.IsIdle(pm.idleTimeout) {
			pm.logger.Info("插件空闲，暂停", "id", plugin.ID)
			plugin.Sandbox.Pause()
			plugin.State = PluginStatePaused
		}
	}
}

// Stop 停止插件管理器
func (pm *PluginManager) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 停止所有插件
	for id, plugin := range pm.plugins {
		if plugin.State == PluginStateRunning {
			pm.logger.Info("停止插件", "id", id)
			plugin.Sandbox.Stop()
			plugin.State = PluginStateStopped
			plugin.StopTime = time.Now()
		}
	}

	// 取消上下文
	pm.cancel()

	pm.logger.Info("插件管理器已停止")
}
