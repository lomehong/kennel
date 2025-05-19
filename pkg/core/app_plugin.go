package core

import (
	"context"
	"fmt"
	"time"

	"github.com/lomehong/kennel/pkg/plugin"
)

// 初始化插件管理器
func (app *App) initPluginManager() {
	// 创建插件管理器
	app.pluginManager = plugin.NewPluginManager(
		plugin.WithPluginManagerLogger(app.logger.Named("plugin-manager")),
		plugin.WithPluginsDir(app.configManager.GetString("plugins.dir")),
		plugin.WithPluginManagerResourceTracker(app.resourceTracker),
		plugin.WithPluginManagerWorkerPool(app.GetDefaultWorkerPool()),
		plugin.WithPluginManagerErrorRegistry(app.errorRegistry),
		plugin.WithPluginManagerRecoveryManager(app.recoveryManager),
		plugin.WithPluginManagerContext(app.ctx),
		plugin.WithHealthCheckInterval(app.configManager.GetDurationOrDefault("plugins.health_check_interval", 30*time.Second)),
		plugin.WithIdleTimeout(app.configManager.GetDurationOrDefault("plugins.idle_timeout", 10*time.Minute)),
	)

	// 启动健康检查
	app.pluginManager.StartHealthCheck()

	app.logger.Info("插件管理器已初始化")
}

// LoadPlugin 加载插件
func (app *App) LoadPlugin(config *plugin.PluginConfig) (*plugin.ManagedPlugin, error) {
	return app.pluginManager.LoadPlugin(config)
}

// StartPlugin 启动插件
func (app *App) StartPlugin(id string) error {
	return app.pluginManager.StartPlugin(id)
}

// StopPlugin 停止插件
func (app *App) StopPlugin(id string) error {
	return app.pluginManager.StopPlugin(id)
}

// RestartPlugin 重启插件
func (app *App) RestartPlugin(id string) error {
	return app.pluginManager.RestartPlugin(id)
}

// UnloadPlugin 卸载插件
func (app *App) UnloadPlugin(id string) error {
	return app.pluginManager.UnloadPlugin(id)
}

// GetPlugin 获取插件
func (app *App) GetPlugin(id string) (*plugin.ManagedPlugin, bool) {
	return app.pluginManager.GetPlugin(id)
}

// ListPlugins 列出所有插件
func (app *App) ListPlugins() []*plugin.ManagedPlugin {
	return app.pluginManager.ListPlugins()
}

// ExecutePluginFunc 在插件沙箱中执行函数
func (app *App) ExecutePluginFunc(id string, f func() error) error {
	return app.pluginManager.ExecutePluginFunc(id, f)
}

// ExecutePluginFuncWithContext 在插件沙箱中执行带上下文的函数
func (app *App) ExecutePluginFuncWithContext(id string, ctx context.Context, f func(context.Context) error) error {
	return app.pluginManager.ExecutePluginFuncWithContext(id, ctx, f)
}

// GetPluginRegistry 获取插件注册表
func (app *App) GetPluginRegistry() *PluginRegistry {
	return app.pluginRegistry
}

// LoadPluginsFromConfig 从配置加载插件
func (app *App) LoadPluginsFromConfig() error {
	// 首先发现可用的插件
	if err := app.pluginRegistry.DiscoverPlugins(); err != nil {
		app.logger.Error("发现插件失败", "error", err)
		return fmt.Errorf("发现插件失败: %w", err)
	}

	// 获取插件配置
	pluginsConfig := app.configManager.GetStringMap("plugins")
	if pluginsConfig == nil {
		// 尝试从旧版配置格式中获取
		return app.LoadPluginsFromLegacyConfig()
	}

	// 加载每个插件
	for id, configData := range pluginsConfig {
		// 转换为map
		_, ok := configData.(map[string]interface{})
		if !ok {
			app.logger.Warn("插件配置格式错误", "id", id)
			continue
		}

		// 检查是否启用
		enabled := app.configManager.GetBoolOrDefault(fmt.Sprintf("plugins.%s.enabled", id), true)
		if !enabled {
			app.logger.Info("插件未启用，跳过加载", "id", id)
			continue
		}

		// 创建插件配置
		config := &plugin.PluginConfig{
			ID:          id,
			Name:        app.configManager.GetStringOrDefault(fmt.Sprintf("plugins.%s.name", id), id),
			Version:     app.configManager.GetStringOrDefault(fmt.Sprintf("plugins.%s.version", id), "1.0.0"),
			Path:        app.configManager.GetStringOrDefault(fmt.Sprintf("plugins.%s.path", id), id),
			AutoStart:   app.configManager.GetBoolOrDefault(fmt.Sprintf("plugins.%s.auto_start", id), false),
			AutoRestart: app.configManager.GetBoolOrDefault(fmt.Sprintf("plugins.%s.auto_restart", id), false),
			Enabled:     true,
		}

		// 注册插件
		if err := app.pluginRegistry.RegisterPlugin(config); err != nil {
			app.logger.Warn("注册插件失败", "id", id, "error", err)
			continue
		}

		// 加载插件
		if err := app.loadPlugin(config); err != nil {
			app.logger.Error("加载插件失败", "id", id, "error", err)
		} else {
			app.logger.Info("加载插件成功", "id", id)
		}
	}

	// 如果没有加载任何插件，尝试加载自动发现的插件
	if len(pluginsConfig) == 0 {
		app.logger.Info("没有找到插件配置，尝试加载自动发现的插件")
		return app.loadDiscoveredPlugins()
	}

	return nil
}

// LoadPluginsFromLegacyConfig 从旧版配置加载插件
func (app *App) LoadPluginsFromLegacyConfig() error {
	// 获取插件配置
	pluginsConfig := app.configManager.GetStringMap("plugins.list")
	if pluginsConfig == nil {
		app.logger.Info("没有找到旧版插件配置，尝试加载自动发现的插件")
		return app.loadDiscoveredPlugins()
	}

	// 加载每个插件
	pluginsLoaded := 0
	for id, configData := range pluginsConfig {
		// 转换为map
		configMap, ok := configData.(map[string]interface{})
		if !ok {
			app.logger.Warn("插件配置格式错误", "id", id)
			continue
		}

		// 创建插件配置
		config := &plugin.PluginConfig{
			ID:             id,
			Name:           getStringOrDefault(configMap, "name", id),
			Version:        getStringOrDefault(configMap, "version", "1.0.0"),
			Path:           getStringOrDefault(configMap, "path", id),
			IsolationLevel: getIsolationLevel(configMap, "isolation_level"),
			AutoStart:      getBoolOrDefault(configMap, "auto_start", false),
			AutoRestart:    getBoolOrDefault(configMap, "auto_restart", false),
			Enabled:        getBoolOrDefault(configMap, "enabled", true),
		}

		// 如果插件未启用，跳过
		if !config.Enabled {
			app.logger.Info("插件未启用，跳过加载", "id", id)
			continue
		}

		// 注册插件
		if err := app.pluginRegistry.RegisterPlugin(config); err != nil {
			app.logger.Warn("注册插件失败", "id", id, "error", err)
			continue
		}

		// 加载插件
		if err := app.loadPlugin(config); err != nil {
			app.logger.Error("加载插件失败", "id", id, "error", err)
		} else {
			app.logger.Info("加载插件成功", "id", id)
			pluginsLoaded++
		}
	}

	// 如果没有加载任何插件，尝试加载自动发现的插件
	if pluginsLoaded == 0 {
		app.logger.Info("从旧版配置中没有加载任何插件，尝试加载自动发现的插件")
		return app.loadDiscoveredPlugins()
	}

	return nil
}

// loadDiscoveredPlugins 加载自动发现的插件
func (app *App) loadDiscoveredPlugins() error {
	// 获取所有已发现的插件
	plugins := app.pluginRegistry.ListPlugins()
	app.logger.Info("开始加载已发现的插件", "count", len(plugins))

	pluginsLoaded := 0
	for _, config := range plugins {
		// 检查是否在配置中启用
		enableKey := fmt.Sprintf("enable_%s", config.ID)
		enabled := app.configManager.GetBoolOrDefault(enableKey, true)
		app.logger.Debug("检查插件是否启用", "id", config.ID, "enableKey", enableKey, "enabled", enabled)

		if !enabled {
			app.logger.Info("插件未在配置中启用，跳过加载", "id", config.ID)
			continue
		}

		// 设置插件自动启动
		config.AutoStart = true

		// 加载插件
		app.logger.Info("开始加载插件", "id", config.ID, "path", config.Path)
		if err := app.loadPlugin(config); err != nil {
			app.logger.Error("加载插件失败", "id", config.ID, "error", err)
		} else {
			app.logger.Info("加载插件成功", "id", config.ID)
			pluginsLoaded++
		}
	}

	app.logger.Info("自动发现的插件加载完成", "loaded", pluginsLoaded, "total", len(plugins))
	return nil
}

// loadPlugin 加载单个插件
func (app *App) loadPlugin(config *plugin.PluginConfig) error {
	app.logger.Info("加载插件", "id", config.ID, "name", config.Name)

	// 使用插件管理器的LoadPlugin方法加载插件
	_, err := app.pluginManager.LoadPlugin(config)
	if err != nil {
		app.logger.Error("加载插件失败", "id", config.ID, "error", err)
		return fmt.Errorf("加载插件失败: %w", err)
	}

	// 如果配置为自动启动，则启动插件
	if config.AutoStart {
		if err := app.pluginManager.StartPlugin(config.ID); err != nil {
			app.logger.Error("启动插件失败", "id", config.ID, "error", err)
			return fmt.Errorf("启动插件失败: %w", err)
		}
	}

	return nil
}

// loadModule 加载模块（兼容旧版本）
func (app *App) loadModule(name string) error {
	app.logger.Info("加载模块", "name", name)

	// 检查是否已注册
	if config, exists := app.pluginRegistry.GetPlugin(name); exists {
		return app.loadPlugin(config)
	}

	// 创建一个新的插件配置
	pluginConfig := &plugin.PluginConfig{
		ID:        name,
		Name:      name,
		Version:   "1.0.0",
		Path:      name,
		AutoStart: true,
		Enabled:   true,
	}

	// 注册插件
	if err := app.pluginRegistry.RegisterPlugin(pluginConfig); err != nil {
		app.logger.Warn("注册插件失败", "name", name, "error", err)
	}

	// 加载插件
	return app.loadPlugin(pluginConfig)
}

// 辅助函数

// getStringOrDefault 获取字符串或默认值
func getStringOrDefault(m map[string]interface{}, key string, defaultValue string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return defaultValue
}

// getBoolOrDefault 获取布尔值或默认值
func getBoolOrDefault(m map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := m[key].(bool); ok {
		return value
	}
	return defaultValue
}

// getIsolationLevel 获取隔离级别
func getIsolationLevel(m map[string]interface{}, key string) plugin.IsolationLevel {
	value, ok := m[key].(string)
	if !ok {
		return plugin.IsolationLevelBasic
	}

	switch value {
	case "none":
		return plugin.IsolationLevelNone
	case "basic":
		return plugin.IsolationLevelBasic
	case "strict":
		return plugin.IsolationLevelStrict
	case "complete":
		return plugin.IsolationLevelComplete
	default:
		return plugin.IsolationLevelBasic
	}
}
