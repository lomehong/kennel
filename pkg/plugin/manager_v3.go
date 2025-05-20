package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"github.com/lomehong/kennel/pkg/plugin/communication"
	"github.com/lomehong/kennel/pkg/plugin/dependency"
	"github.com/lomehong/kennel/pkg/plugin/discovery"
	"github.com/lomehong/kennel/pkg/plugin/docs"
	"github.com/lomehong/kennel/pkg/plugin/isolation"
	"github.com/lomehong/kennel/pkg/plugin/lifecycle"
	"github.com/lomehong/kennel/pkg/plugin/registry"
	"github.com/lomehong/kennel/pkg/plugin/sdk"
)

// PluginManagerV3 新版插件管理器
// 负责插件的加载、卸载、启动和停止
type PluginManagerV3 struct {
	// 插件注册表
	registry registry.PluginRegistry

	// 插件发现器
	discoverer discovery.PluginDiscoverer

	// 生命周期管理器映射
	lifecycleManagers map[string]*lifecycle.PluginLifecycleManager

	// 插件实例映射
	plugins map[string]api.Plugin

	// 日志记录器
	logger hclog.Logger

	// 互斥锁
	mu sync.RWMutex

	// 上下文
	ctx context.Context

	// 取消函数
	cancel context.CancelFunc

	// 配置
	config ManagerConfigV3

	// 依赖管理器
	dependencyManager *dependency.DependencyManager

	// 依赖注入器
	dependencyInjector *dependency.DependencyInjector

	// 依赖解析器
	dependencyResolver *dependency.DependencyResolver

	// 插件隔离器
	isolator isolation.PluginIsolator

	// 沙箱映射
	sandboxes map[string]isolation.PluginSandbox

	// 通信工厂
	communicationFactory *communication.DefaultCommunicationFactory

	// 文档生成器
	docGenerator *docs.DocGenerator
}

// ManagerConfigV3 新版管理器配置
type ManagerConfigV3 struct {
	// 插件目录
	PluginsDir string

	// 自动发现
	AutoDiscover bool

	// 扫描间隔
	ScanInterval time.Duration

	// 健康检查间隔
	HealthCheckInterval time.Duration

	// 自动启动
	AutoStart bool

	// 隔离级别
	IsolationLevel isolation.IsolationLevel

	// 资源限制
	ResourceLimits map[string]int64

	// 操作超时
	OperationTimeout time.Duration
}

// DefaultManagerConfigV3 返回默认管理器配置
func DefaultManagerConfigV3() ManagerConfigV3 {
	return ManagerConfigV3{
		PluginsDir:          "plugins",
		AutoDiscover:        true,
		ScanInterval:        60 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		AutoStart:           true,
		IsolationLevel:      isolation.IsolationLevelBasic,
		ResourceLimits: map[string]int64{
			"memory": 256 * 1024 * 1024, // 256MB
			"cpu":    50,                // 50%
		},
		OperationTimeout: 30 * time.Second,
	}
}

// NewPluginManagerV3 创建一个新的插件管理器
func NewPluginManagerV3(logger hclog.Logger, config ManagerConfigV3) *PluginManagerV3 {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建插件注册表
	reg := registry.NewPluginRegistry(logger.Named("registry"))

	// 创建插件发现器
	disc := discovery.NewFileSystemDiscoverer(
		[]string{config.PluginsDir},
		discovery.WithLogger(logger.Named("discoverer")),
		discovery.WithRecursive(true),
	)

	// 创建依赖管理器
	depManager := dependency.NewDependencyManager(logger.Named("dependency-manager"))

	// 创建依赖注入器
	depInjector := dependency.NewDependencyInjector(logger.Named("dependency-injector"))

	// 创建插件隔离器
	isolator := isolation.NewPluginIsolator(logger.Named("isolator"))

	// 创建通信工厂
	commFactory := communication.NewCommunicationFactory(logger.Named("communication-factory"))

	// 创建文档生成器
	docGen := docs.NewDocGenerator(reg, logger.Named("doc-generator"))

	manager := &PluginManagerV3{
		registry:             reg,
		discoverer:           disc,
		lifecycleManagers:    make(map[string]*lifecycle.PluginLifecycleManager),
		plugins:              make(map[string]api.Plugin),
		logger:               logger.Named("plugin-manager-v3"),
		ctx:                  ctx,
		cancel:               cancel,
		config:               config,
		dependencyManager:    depManager,
		dependencyInjector:   depInjector,
		isolator:             isolator,
		sandboxes:            make(map[string]isolation.PluginSandbox),
		communicationFactory: commFactory,
		docGenerator:         docGen,
	}

	// 创建依赖解析器
	manager.dependencyResolver = dependency.NewDependencyResolver(
		depManager,
		depInjector,
		manager,
		logger.Named("dependency-resolver"),
	)

	return manager
}

// Start 启动插件管理器
func (m *PluginManagerV3) Start() error {
	m.logger.Info("启动插件管理器")

	// 如果启用自动发现，启动插件发现
	if m.config.AutoDiscover {
		m.logger.Info("启动插件自动发现", "interval", m.config.ScanInterval)
		go m.startDiscovery()
	}

	// 如果启用健康检查，启动健康检查
	if m.config.HealthCheckInterval > 0 {
		m.logger.Info("启动插件健康检查", "interval", m.config.HealthCheckInterval)
		go m.startHealthCheck()
	}

	return nil
}

// Stop 停止插件管理器
func (m *PluginManagerV3) Stop() error {
	m.logger.Info("停止插件管理器")

	// 取消上下文
	m.cancel()

	// 停止所有插件
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取依赖顺序
	order, err := m.dependencyManager.GetDependencyOrder()
	if err != nil {
		m.logger.Error("获取依赖顺序失败", "error", err)
		// 继续停止过程
	} else {
		// 按依赖顺序的反向停止插件
		for i := len(order) - 1; i >= 0; i-- {
			id := order[i]
			lcm, exists := m.lifecycleManagers[id]
			if !exists {
				continue
			}

			m.logger.Info("停止插件", "id", id)
			if err := lcm.Stop(m.ctx); err != nil {
				m.logger.Error("停止插件失败", "id", id, "error", err)
			}
			lcm.Close()
		}
	}

	// 销毁所有沙箱
	for id, sandbox := range m.sandboxes {
		m.logger.Info("销毁沙箱", "id", id)
		sandbox.Stop()
		m.isolator.DestroySandbox(id)
	}

	// 关闭隔离器
	m.isolator.Close()

	return nil
}

// startDiscovery 启动插件发现
func (m *PluginManagerV3) startDiscovery() {
	ticker := time.NewTicker(m.config.ScanInterval)
	defer ticker.Stop()

	// 立即执行一次发现
	m.discoverPlugins()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.discoverPlugins()
		}
	}
}

// discoverPlugins 发现插件
func (m *PluginManagerV3) discoverPlugins() {
	m.logger.Debug("开始发现插件")

	// 发现插件
	plugins, err := m.discoverer.DiscoverPlugins(m.ctx)
	if err != nil {
		m.logger.Error("发现插件失败", "error", err)
		return
	}

	m.logger.Debug("发现插件", "count", len(plugins))

	// 注册插件
	for _, metadata := range plugins {
		// 检查插件是否已注册
		if _, exists := m.registry.GetPluginMetadata(metadata.ID); exists {
			continue
		}

		// 注册插件
		m.logger.Info("注册插件", "id", metadata.ID, "name", metadata.Name)
		if err := m.registry.RegisterPlugin(metadata); err != nil {
			m.logger.Error("注册插件失败", "id", metadata.ID, "error", err)
			continue
		}

		// 注册插件依赖
		if err := m.dependencyManager.RegisterPlugin(metadata.ID, metadata.Version, metadata.Dependencies); err != nil {
			m.logger.Error("注册插件依赖失败", "id", metadata.ID, "error", err)
			// 继续注册过程
		}

		// 如果启用自动启动，加载并启动插件
		if m.config.AutoStart {
			m.logger.Info("自动加载插件", "id", metadata.ID)
			if _, err := m.LoadPlugin(metadata.ID); err != nil {
				m.logger.Error("加载插件失败", "id", metadata.ID, "error", err)
				continue
			}
		}
	}
}

// startHealthCheck 启动健康检查
func (m *PluginManagerV3) startHealthCheck() {
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkPluginsHealth()
		}
	}
}

// checkPluginsHealth 检查插件健康状态
func (m *PluginManagerV3) checkPluginsHealth() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, lcm := range m.lifecycleManagers {
		// 检查插件是否处于活动状态
		if !lcm.IsActive() {
			continue
		}

		// 检查插件是否健康
		if !lcm.IsHealthy() {
			m.logger.Warn("插件不健康", "id", id)
			// 在实际实现中，可以添加自动重启逻辑
		}
	}
}

// LoadPlugin 加载插件
func (m *PluginManagerV3) LoadPlugin(id string) (api.Plugin, error) {
	// 获取插件元数据
	metadata, exists := m.registry.GetPluginMetadata(id)
	if !exists {
		return nil, fmt.Errorf("插件 %s 未注册", id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查插件是否已加载
	if plugin, exists := m.plugins[id]; exists {
		return plugin, nil
	}

	// 加载插件
	m.logger.Info("加载插件", "id", id, "path", metadata.Location.Path)

	// 在实际实现中，这里应该根据插件类型和位置加载插件
	// 这里简化处理，创建一个模拟插件
	plugin := createMockPluginV3(metadata)

	// 创建沙箱
	sandbox, err := m.createPluginSandbox(id, metadata)
	if err != nil {
		m.logger.Error("创建插件沙箱失败", "id", id, "error", err)
		// 继续加载过程
	} else {
		m.sandboxes[id] = sandbox
	}

	// 创建生命周期管理器
	lcm := lifecycle.NewPluginLifecycleManager(plugin, api.PluginConfig{
		ID:      id,
		Enabled: true,
	}, m.logger.Named(fmt.Sprintf("lifecycle-%s", id)))

	// 存储插件和生命周期管理器
	m.plugins[id] = plugin
	m.lifecycleManagers[id] = lcm

	// 解析插件依赖
	if err := m.dependencyResolver.ResolvePluginDependencies(id); err != nil {
		m.logger.Error("解析插件依赖失败", "id", id, "error", err)
		// 继续加载过程
	}

	// 初始化插件
	if err := lcm.Init(m.ctx); err != nil {
		m.logger.Error("初始化插件失败", "id", id, "error", err)
		delete(m.plugins, id)
		delete(m.lifecycleManagers, id)

		// 销毁沙箱
		if sandbox, exists := m.sandboxes[id]; exists {
			sandbox.Stop()
			m.isolator.DestroySandbox(id)
			delete(m.sandboxes, id)
		}

		return nil, fmt.Errorf("初始化插件失败: %w", err)
	}

	// 如果启用自动启动，启动插件
	if m.config.AutoStart {
		m.logger.Info("自动启动插件", "id", id)
		if err := lcm.Start(m.ctx); err != nil {
			m.logger.Error("启动插件失败", "id", id, "error", err)
			// 不返回错误，保留已初始化的插件
		}
	}

	return plugin, nil
}

// createPluginSandbox 创建插件沙箱
func (m *PluginManagerV3) createPluginSandbox(id string, metadata api.PluginMetadata) (isolation.PluginSandbox, error) {
	// 创建隔离配置
	config := api.IsolationConfig{
		Level:      string(m.config.IsolationLevel),
		Resources:  m.config.ResourceLimits,
		Timeout:    m.config.OperationTimeout,
		WorkingDir: metadata.Location.Path,
		Environment: map[string]string{
			"PLUGIN_ID":      id,
			"PLUGIN_VERSION": metadata.Version,
		},
	}

	// 创建沙箱
	return m.isolator.CreateSandbox(id, config)
}

// UnloadPlugin 卸载插件
func (m *PluginManagerV3) UnloadPlugin(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查插件是否已加载
	lcm, exists := m.lifecycleManagers[id]
	if !exists {
		return fmt.Errorf("插件 %s 未加载", id)
	}

	// 检查是否有其他插件依赖于此插件
	dependents := m.dependencyManager.GetDependents(id)
	if len(dependents) > 0 {
		return fmt.Errorf("插件 %s 被以下插件依赖: %v", id, dependents)
	}

	// 如果插件正在运行，先停止它
	if lcm.IsActive() {
		m.logger.Info("停止插件", "id", id)
		if err := lcm.Stop(m.ctx); err != nil {
			m.logger.Error("停止插件失败", "id", id, "error", err)
			// 继续卸载过程
		}
	}

	// 关闭生命周期管理器
	lcm.Close()

	// 删除插件和生命周期管理器
	delete(m.plugins, id)
	delete(m.lifecycleManagers, id)

	// 销毁沙箱
	if sandbox, exists := m.sandboxes[id]; exists {
		sandbox.Stop()
		m.isolator.DestroySandbox(id)
		delete(m.sandboxes, id)
	}

	m.logger.Info("插件已卸载", "id", id)
	return nil
}

// StartPlugin 启动插件
func (m *PluginManagerV3) StartPlugin(id string) error {
	m.mu.RLock()
	lcm, exists := m.lifecycleManagers[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("插件 %s 未加载", id)
	}

	// 检查依赖
	missingDeps, err := m.dependencyManager.CheckDependencies(id)
	if err != nil {
		return fmt.Errorf("检查插件依赖失败: %w", err)
	}

	if len(missingDeps) > 0 {
		return fmt.Errorf("插件 %s 缺少依赖: %v", id, missingDeps)
	}

	m.logger.Info("启动插件", "id", id)
	return lcm.Start(m.ctx)
}

// StopPlugin 停止插件
func (m *PluginManagerV3) StopPlugin(id string) error {
	m.mu.RLock()
	lcm, exists := m.lifecycleManagers[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("插件 %s 未加载", id)
	}

	// 检查是否有其他插件依赖于此插件
	dependents := m.dependencyManager.GetDependents(id)
	if len(dependents) > 0 {
		return fmt.Errorf("插件 %s 被以下插件依赖: %v", id, dependents)
	}

	m.logger.Info("停止插件", "id", id)
	return lcm.Stop(m.ctx)
}

// GetPlugin 获取插件
func (m *PluginManagerV3) GetPlugin(id string) (api.Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	plugin, exists := m.plugins[id]
	return plugin, exists
}

// ListPlugins 列出所有插件
func (m *PluginManagerV3) ListPlugins() []api.PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]api.PluginInfo, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin.GetInfo())
	}

	return plugins
}

// GetPluginStatus 获取插件状态
func (m *PluginManagerV3) GetPluginStatus(id string) (api.PluginStatus, error) {
	m.mu.RLock()
	lcm, exists := m.lifecycleManagers[id]
	plugin, pluginExists := m.plugins[id]
	m.mu.RUnlock()

	if !exists || !pluginExists {
		return api.PluginStatus{}, fmt.Errorf("插件 %s 未加载", id)
	}

	// 执行健康检查
	health, err := plugin.HealthCheck(m.ctx)
	if err != nil {
		health = api.HealthStatus{
			Status:      "unknown",
			Details:     map[string]interface{}{"error": err.Error()},
			LastChecked: time.Now(),
		}
	}

	return api.PluginStatus{
		ID:     id,
		State:  lcm.GetCurrentState(),
		Health: health,
	}, nil
}

// GetPluginDependencies 获取插件依赖
func (m *PluginManagerV3) GetPluginDependencies(id string) ([]api.PluginDependency, error) {
	return m.dependencyManager.GetPluginDependencies(id)
}

// GetPluginDependents 获取依赖于指定插件的插件
func (m *PluginManagerV3) GetPluginDependents(id string) []string {
	return m.dependencyManager.GetDependents(id)
}

// GetDependencyOrder 获取依赖顺序
func (m *PluginManagerV3) GetDependencyOrder() ([]string, error) {
	return m.dependencyManager.GetDependencyOrder()
}

// ExecuteInSandbox 在沙箱中执行函数
func (m *PluginManagerV3) ExecuteInSandbox(id string, f func() error) error {
	m.mu.RLock()
	sandbox, exists := m.sandboxes[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("插件 %s 沙箱不存在", id)
	}

	return sandbox.Execute(f)
}

// createMockPluginV3 创建模拟插件
// 仅用于演示，实际实现应该加载真实的插件
func createMockPluginV3(metadata api.PluginMetadata) api.Plugin {
	return &mockPluginV3{
		info: api.PluginInfo{
			ID:           metadata.ID,
			Name:         metadata.Name,
			Version:      metadata.Version,
			Description:  metadata.Description,
			Author:       metadata.Author,
			License:      metadata.License,
			Tags:         metadata.Tags,
			Capabilities: metadata.Capabilities,
			Dependencies: metadata.Dependencies,
		},
	}
}

// mockPluginV3 模拟插件
type mockPluginV3 struct {
	info api.PluginInfo
}

// GetInfo 获取插件信息
func (p *mockPluginV3) GetInfo() api.PluginInfo {
	return p.info
}

// Init 初始化插件
func (p *mockPluginV3) Init(ctx context.Context, config api.PluginConfig) error {
	return nil
}

// Start 启动插件
func (p *mockPluginV3) Start(ctx context.Context) error {
	return nil
}

// Stop 停止插件
func (p *mockPluginV3) Stop(ctx context.Context) error {
	return nil
}

// HealthCheck 执行健康检查
func (p *mockPluginV3) HealthCheck(ctx context.Context) (api.HealthStatus, error) {
	return api.HealthStatus{
		Status:      "healthy",
		Details:     make(map[string]interface{}),
		LastChecked: time.Now(),
	}, nil
}

// GetRegistry 获取插件注册表
func (m *PluginManagerV3) GetRegistry() registry.PluginRegistry {
	return m.registry
}

// InjectPlugin 注入插件
// 用于测试
func (m *PluginManagerV3) InjectPlugin(id string, plugin api.Plugin) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.plugins[id] = plugin
	m.lifecycleManagers[id] = lifecycle.NewPluginLifecycleManager(plugin, api.PluginConfig{
		ID:      id,
		Enabled: true,
	}, m.GetLogger().Named(fmt.Sprintf("lifecycle-%s", id)))
}

// GetLogger 获取日志记录器
func (m *PluginManagerV3) GetLogger() hclog.Logger {
	return m.logger
}

// CreateCommunication 创建通信
func (m *PluginManagerV3) CreateCommunication(protocol sdk.CommunicationProtocol, options map[string]interface{}) (sdk.Communication, error) {
	return m.communicationFactory.CreateCommunication(protocol, options)
}

// RegisterCommunicationHandler 注册通信处理器
func (m *PluginManagerV3) RegisterCommunicationHandler(protocol sdk.CommunicationProtocol, handler sdk.CommunicationHandler) {
	m.communicationFactory.RegisterHandler(protocol, handler)
}

// GeneratePluginDocs 生成插件文档
func (m *PluginManagerV3) GeneratePluginDocs(outputDir string) error {
	return m.docGenerator.GenerateAllPluginDocs(outputDir)
}

// GeneratePluginDiagram 生成插件依赖关系图
func (m *PluginManagerV3) GeneratePluginDiagram(outputPath string) error {
	return m.docGenerator.GeneratePluginDiagram(outputPath)
}

// GetPluginDocGenerator 获取插件文档生成器
func (m *PluginManagerV3) GetPluginDocGenerator() *docs.DocGenerator {
	return m.docGenerator
}

// GetCommunicationFactory 获取通信工厂
func (m *PluginManagerV3) GetCommunicationFactory() *communication.DefaultCommunicationFactory {
	return m.communicationFactory
}

// GetDependencyManager 获取依赖管理器
func (m *PluginManagerV3) GetDependencyManager() *dependency.DependencyManager {
	return m.dependencyManager
}

// GetDependencyInjector 获取依赖注入器
func (m *PluginManagerV3) GetDependencyInjector() *dependency.DependencyInjector {
	return m.dependencyInjector
}

// GetDependencyResolver 获取依赖解析器
func (m *PluginManagerV3) GetDependencyResolver() *dependency.DependencyResolver {
	return m.dependencyResolver
}

// GetIsolator 获取隔离器
func (m *PluginManagerV3) GetIsolator() isolation.PluginIsolator {
	return m.isolator
}
