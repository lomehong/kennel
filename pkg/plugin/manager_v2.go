package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"github.com/lomehong/kennel/pkg/plugin/discovery"
	"github.com/lomehong/kennel/pkg/plugin/lifecycle"
	"github.com/lomehong/kennel/pkg/plugin/registry"
)

// PluginManagerV2 新版插件管理器
// 负责插件的加载、卸载、启动和停止
type PluginManagerV2 struct {
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
	config ManagerConfigV2
}

// ManagerConfigV2 新版管理器配置
type ManagerConfigV2 struct {
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
}

// DefaultManagerConfigV2 返回默认管理器配置
func DefaultManagerConfigV2() ManagerConfigV2 {
	return ManagerConfigV2{
		PluginsDir:          "plugins",
		AutoDiscover:        true,
		ScanInterval:        60 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		AutoStart:           true,
	}
}

// NewPluginManagerV2 创建一个新的插件管理器
func NewPluginManagerV2(logger hclog.Logger, config ManagerConfigV2) *PluginManagerV2 {
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

	return &PluginManagerV2{
		registry:          reg,
		discoverer:        disc,
		lifecycleManagers: make(map[string]*lifecycle.PluginLifecycleManager),
		plugins:           make(map[string]api.Plugin),
		logger:            logger.Named("plugin-manager-v2"),
		ctx:               ctx,
		cancel:            cancel,
		config:            config,
	}
}

// Start 启动插件管理器
func (m *PluginManagerV2) Start() error {
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
func (m *PluginManagerV2) Stop() error {
	m.logger.Info("停止插件管理器")

	// 取消上下文
	m.cancel()

	// 停止所有插件
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, lcm := range m.lifecycleManagers {
		m.logger.Info("停止插件", "id", id)
		if err := lcm.Stop(context.Background()); err != nil {
			m.logger.Error("停止插件失败", "id", id, "error", err)
		}
		lcm.Close()
	}

	return nil
}

// startDiscovery 启动插件发现
func (m *PluginManagerV2) startDiscovery() {
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
func (m *PluginManagerV2) discoverPlugins() {
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
func (m *PluginManagerV2) startHealthCheck() {
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
func (m *PluginManagerV2) checkPluginsHealth() {
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
func (m *PluginManagerV2) LoadPlugin(id string) (api.Plugin, error) {
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
	plugin := createMockPluginV2(metadata)

	// 创建生命周期管理器
	lcm := lifecycle.NewPluginLifecycleManager(plugin, api.PluginConfig{
		ID:      id,
		Enabled: true,
	}, m.logger.Named(fmt.Sprintf("lifecycle-%s", id)))

	// 存储插件和生命周期管理器
	m.plugins[id] = plugin
	m.lifecycleManagers[id] = lcm

	// 初始化插件
	if err := lcm.Init(m.ctx); err != nil {
		m.logger.Error("初始化插件失败", "id", id, "error", err)
		delete(m.plugins, id)
		delete(m.lifecycleManagers, id)
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

// UnloadPlugin 卸载插件
func (m *PluginManagerV2) UnloadPlugin(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查插件是否已加载
	lcm, exists := m.lifecycleManagers[id]
	if !exists {
		return fmt.Errorf("插件 %s 未加载", id)
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

	m.logger.Info("插件已卸载", "id", id)
	return nil
}

// StartPlugin 启动插件
func (m *PluginManagerV2) StartPlugin(id string) error {
	m.mu.RLock()
	lcm, exists := m.lifecycleManagers[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("插件 %s 未加载", id)
	}

	m.logger.Info("启动插件", "id", id)
	return lcm.Start(m.ctx)
}

// StopPlugin 停止插件
func (m *PluginManagerV2) StopPlugin(id string) error {
	m.mu.RLock()
	lcm, exists := m.lifecycleManagers[id]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("插件 %s 未加载", id)
	}

	m.logger.Info("停止插件", "id", id)
	return lcm.Stop(m.ctx)
}

// GetPlugin 获取插件
func (m *PluginManagerV2) GetPlugin(id string) (api.Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	plugin, exists := m.plugins[id]
	return plugin, exists
}

// ListPlugins 列出所有插件
func (m *PluginManagerV2) ListPlugins() []api.PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]api.PluginInfo, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin.GetInfo())
	}

	return plugins
}

// GetPluginStatus 获取插件状态
func (m *PluginManagerV2) GetPluginStatus(id string) (api.PluginStatus, error) {
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

// createMockPluginV2 创建模拟插件
// 仅用于演示，实际实现应该加载真实的插件
func createMockPluginV2(metadata api.PluginMetadata) api.Plugin {
	return &mockPluginV2{
		info: api.PluginInfo{
			ID:          metadata.ID,
			Name:        metadata.Name,
			Version:     metadata.Version,
			Description: metadata.Description,
			Author:      metadata.Author,
			License:     metadata.License,
			Tags:        metadata.Tags,
			Capabilities: metadata.Capabilities,
			Dependencies: metadata.Dependencies,
		},
	}
}

// mockPluginV2 模拟插件
type mockPluginV2 struct {
	info api.PluginInfo
}

// GetInfo 获取插件信息
func (p *mockPluginV2) GetInfo() api.PluginInfo {
	return p.info
}

// Init 初始化插件
func (p *mockPluginV2) Init(ctx context.Context, config api.PluginConfig) error {
	return nil
}

// Start 启动插件
func (p *mockPluginV2) Start(ctx context.Context) error {
	return nil
}

// Stop 停止插件
func (p *mockPluginV2) Stop(ctx context.Context) error {
	return nil
}

// HealthCheck 执行健康检查
func (p *mockPluginV2) HealthCheck(ctx context.Context) (api.HealthStatus, error) {
	return api.HealthStatus{
		Status:      "healthy",
		Details:     make(map[string]interface{}),
		LastChecked: time.Now(),
	}, nil
}
