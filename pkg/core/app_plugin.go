package core

import (
	"context"
	"time"

	"github.com/lomehong/kennel/pkg/plugin"
)

// 添加插件隔离到App结构体
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

// 这个方法已经在app.go中定义，这里不需要重复定义

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

// LoadPluginsFromConfig 从配置加载插件
func (app *App) LoadPluginsFromConfig() error {
	// 获取插件配置
	pluginsConfig := app.configManager.GetStringMap("plugins.list")
	if pluginsConfig == nil {
		app.logger.Info("没有找到插件配置")
		return nil
	}

	// 加载每个插件
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

		// 加载插件
		_, err := app.LoadPlugin(config)
		if err != nil {
			app.logger.Error("加载插件失败", "id", id, "error", err)
		} else {
			app.logger.Info("加载插件成功", "id", id)
		}
	}

	return nil
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
