package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/lomehong/kennel/pkg/config"
	"gopkg.in/yaml.v3"
)

// 添加配置热更新和动态配置到App结构体
func (app *App) initConfigSystem() {
	// 获取配置文件路径
	configFile := app.configManager.GetConfigPath()
	if configFile == "" {
		app.logger.Warn("未找到配置文件路径，跳过配置热更新初始化")
		return
	}

	// 创建配置监视器
	watcher, err := config.NewConfigWatcher(app.logger.Named("config-watcher"))
	if err != nil {
		app.logger.Error("创建配置监视器失败", "error", err)
		return
	}
	app.configWatcher = watcher

	// 创建配置验证器
	validator := func(cfg map[string]interface{}) error {
		// 验证服务器配置
		if server, ok := cfg["server"].(map[string]interface{}); ok {
			if port, ok := server["port"].(int); ok {
				if port < 1024 || port > 65535 {
					return fmt.Errorf("服务器端口应该在 1024-65535 范围内")
				}
			}
		}

		// 验证日志配置
		if logging, ok := cfg["logging"].(map[string]interface{}); ok {
			if level, ok := logging["level"].(string); ok {
				validLevels := map[string]bool{
					"trace": true,
					"debug": true,
					"info":  true,
					"warn":  true,
					"error": true,
				}
				if !validLevels[level] {
					return fmt.Errorf("无效的日志级别: %s", level)
				}
			}
		}

		return nil
	}

	// 创建配置变更监听器
	listener := func(oldConfig, newConfig map[string]interface{}) error {
		app.logger.Info("配置已变更")

		// 处理日志配置变更
		if err := app.handleLoggingConfigChange(oldConfig, newConfig); err != nil {
			app.logger.Error("处理日志配置变更失败", "error", err)
		}

		// 处理服务器配置变更
		if err := app.handleServerConfigChange(oldConfig, newConfig); err != nil {
			app.logger.Error("处理服务器配置变更失败", "error", err)
		}

		// 处理插件配置变更
		if err := app.handlePluginConfigChange(oldConfig, newConfig); err != nil {
			app.logger.Error("处理插件配置变更失败", "error", err)
		}

		// 处理健康检查配置变更
		if err := app.handleHealthConfigChange(oldConfig, newConfig); err != nil {
			app.logger.Error("处理健康检查配置变更失败", "error", err)
		}

		// 通知配置变更
		app.notifyConfigChange(oldConfig, newConfig)

		return nil
	}

	// 创建动态配置
	app.dynamicConfig = config.NewDynamicConfig(
		configFile,
		app.logger.Named("dynamic-config"),
		config.WithWatcher(watcher),
		config.WithValidator(validator),
		config.WithListener(listener),
		config.WithMaxHistorySize(app.configManager.GetIntOrDefault("config.max_history_size", 10)),
	)

	// 启动配置监视
	app.configWatcher.Start()
	app.logger.Info("配置热更新系统已初始化", "config_file", configFile)
}

// handleLoggingConfigChange 处理日志配置变更
func (app *App) handleLoggingConfigChange(oldConfig, newConfig map[string]interface{}) error {
	// 获取旧的日志配置
	var oldLevel string
	if oldLogging, ok := oldConfig["logging"].(map[string]interface{}); ok {
		if level, ok := oldLogging["level"].(string); ok {
			oldLevel = level
		}
	}

	// 获取新的日志配置
	var newLevel string
	if newLogging, ok := newConfig["logging"].(map[string]interface{}); ok {
		if level, ok := newLogging["level"].(string); ok {
			newLevel = level
		}
	}

	// 检查日志级别是否变更
	if oldLevel != newLevel && newLevel != "" {
		app.logger.Info("日志级别已变更", "old", oldLevel, "new", newLevel)
		// 更新日志级别
		app.configManager.Set("log_level", newLevel)
		// 在实际应用中，这里可能需要重新配置日志记录器
	}

	return nil
}

// handleServerConfigChange 处理服务器配置变更
func (app *App) handleServerConfigChange(oldConfig, newConfig map[string]interface{}) error {
	// 获取旧的服务器配置
	var oldHost string
	var oldPort int
	if oldServer, ok := oldConfig["server"].(map[string]interface{}); ok {
		if host, ok := oldServer["host"].(string); ok {
			oldHost = host
		}
		if port, ok := oldServer["port"].(int); ok {
			oldPort = port
		}
	}

	// 获取新的服务器配置
	var newHost string
	var newPort int
	if newServer, ok := newConfig["server"].(map[string]interface{}); ok {
		if host, ok := newServer["host"].(string); ok {
			newHost = host
		}
		if port, ok := newServer["port"].(int); ok {
			newPort = port
		}
	}

	// 检查服务器配置是否变更
	if oldHost != newHost || oldPort != newPort {
		app.logger.Info("服务器配置已变更",
			"old_host", oldHost, "new_host", newHost,
			"old_port", oldPort, "new_port", newPort,
		)
		// 更新服务器配置
		if newHost != "" {
			app.configManager.Set("server_host", newHost)
		}
		if newPort != 0 {
			app.configManager.Set("server_port", newPort)
		}
		// 在实际应用中，这里可能需要重启服务器或更新监听端口
	}

	return nil
}

// handlePluginConfigChange 处理插件配置变更
func (app *App) handlePluginConfigChange(oldConfig, newConfig map[string]interface{}) error {
	// 获取旧的插件配置
	var oldPlugins map[string]interface{}
	if oldPluginsConfig, ok := oldConfig["plugins"].(map[string]interface{}); ok {
		if list, ok := oldPluginsConfig["list"].(map[string]interface{}); ok {
			oldPlugins = list
		}
	}

	// 获取新的插件配置
	var newPlugins map[string]interface{}
	if newPluginsConfig, ok := newConfig["plugins"].(map[string]interface{}); ok {
		if list, ok := newPluginsConfig["list"].(map[string]interface{}); ok {
			newPlugins = list
		}
	}

	// 检查插件配置是否变更
	if oldPlugins == nil || newPlugins == nil {
		return nil
	}

	// 检查新增或修改的插件
	for id, newPluginConfig := range newPlugins {
		newPlugin, ok := newPluginConfig.(map[string]interface{})
		if !ok {
			continue
		}

		oldPlugin, exists := oldPlugins[id]
		if !exists {
			// 新增插件
			app.logger.Info("发现新插件", "id", id)
			// 在实际应用中，这里可能需要加载新插件
		} else {
			// 检查插件配置是否变更
			oldPluginMap, ok := oldPlugin.(map[string]interface{})
			if !ok {
				continue
			}

			// 检查启用状态是否变更
			oldEnabled, _ := oldPluginMap["enabled"].(bool)
			newEnabled, _ := newPlugin["enabled"].(bool)
			if oldEnabled != newEnabled {
				app.logger.Info("插件启用状态已变更", "id", id, "enabled", newEnabled)
				// 在实际应用中，这里可能需要启用或禁用插件
			}
		}
	}

	// 检查删除的插件
	for id := range oldPlugins {
		if _, exists := newPlugins[id]; !exists {
			// 删除插件
			app.logger.Info("插件已删除", "id", id)
			// 在实际应用中，这里可能需要卸载插件
		}
	}

	return nil
}

// handleHealthConfigChange 处理健康检查配置变更
func (app *App) handleHealthConfigChange(oldConfig, newConfig map[string]interface{}) error {
	// 获取旧的健康检查配置
	var oldCheckInterval time.Duration
	var oldAutoRepair bool
	if oldHealth, ok := oldConfig["health"].(map[string]interface{}); ok {
		if interval, ok := oldHealth["check_interval"].(string); ok {
			oldCheckInterval, _ = time.ParseDuration(interval)
		}
		if autoRepair, ok := oldHealth["auto_repair"].(bool); ok {
			oldAutoRepair = autoRepair
		}
	}

	// 获取新的健康检查配置
	var newCheckInterval time.Duration
	var newAutoRepair bool
	if newHealth, ok := newConfig["health"].(map[string]interface{}); ok {
		if interval, ok := newHealth["check_interval"].(string); ok {
			newCheckInterval, _ = time.ParseDuration(interval)
		}
		if autoRepair, ok := newHealth["auto_repair"].(bool); ok {
			newAutoRepair = autoRepair
		}
	}

	// 检查健康检查配置是否变更
	configChanged := false
	if oldCheckInterval != newCheckInterval && newCheckInterval > 0 {
		app.logger.Info("健康检查间隔已变更", "old", oldCheckInterval, "new", newCheckInterval)
		app.configManager.Set("health.check_interval", newCheckInterval)
		configChanged = true
	}

	if oldAutoRepair != newAutoRepair {
		app.logger.Info("自动修复设置已变更", "old", oldAutoRepair, "new", newAutoRepair)
		app.configManager.Set("health.auto_repair", newAutoRepair)
		configChanged = true
	}

	// 如果配置已变更，重启健康监控
	if configChanged && app.healthMonitor != nil {
		app.StopHealthMonitor()
		app.StartHealthMonitor()
	}

	return nil
}

// notifyConfigChange 通知配置变更
func (app *App) notifyConfigChange(oldConfig, newConfig map[string]interface{}) {
	// 在实际应用中，这里可能需要通知其他组件配置已变更
}

// SaveConfig 保存配置
func (app *App) SaveConfig() error {
	if app.dynamicConfig != nil {
		return app.dynamicConfig.Save()
	}
	return app.configManager.SaveConfig()
}

// ReloadConfig 重新加载配置
func (app *App) ReloadConfig() error {
	if app.dynamicConfig != nil {
		return app.dynamicConfig.Reload()
	}
	return app.configManager.InitConfig()
}

// GetConfigValue 获取配置值
func (app *App) GetConfigValue(key string) interface{} {
	if app.dynamicConfig != nil {
		return app.dynamicConfig.Get(key)
	}
	return app.configManager.Get(key)
}

// SetConfigValue 设置配置值
func (app *App) SetConfigValue(key string, value interface{}) {
	if app.dynamicConfig != nil {
		app.dynamicConfig.Set(key, value)
	} else {
		app.configManager.Set(key, value)
	}
}

// GetConfigVersion 获取配置版本
func (app *App) GetConfigVersion() config.ConfigVersion {
	if app.dynamicConfig != nil {
		return app.dynamicConfig.GetVersion()
	}
	return config.ConfigVersion{
		Version:   0,
		Timestamp: time.Now(),
		Comment:   "未使用动态配置",
	}
}

// RollbackConfig 回滚配置
func (app *App) RollbackConfig(version int) error {
	if app.dynamicConfig != nil {
		return app.dynamicConfig.Rollback(version)
	}
	return fmt.Errorf("未使用动态配置，无法回滚")
}

// ExportConfig 导出配置
func (app *App) ExportConfig(path string) error {
	// 获取所有配置
	var config map[string]interface{}
	if app.dynamicConfig != nil {
		config = app.dynamicConfig.GetAll()
	} else {
		config = app.configManager.GetAllConfig()
	}

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 导出配置
	if app.dynamicConfig != nil {
		// 直接写入文件
		data, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("序列化配置失败: %w", err)
		}

		// 写入文件
		if err := ioutil.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("写入配置文件失败: %w", err)
		}
	} else {
		// 使用配置管理器保存
		oldPath := app.configManager.GetConfigPath()
		app.configManager.SetConfigPath(path)
		if err := app.configManager.SaveConfig(); err != nil {
			app.configManager.SetConfigPath(oldPath)
			return fmt.Errorf("保存配置失败: %w", err)
		}
		app.configManager.SetConfigPath(oldPath)
	}

	return nil
}

// ImportConfig 导入配置
func (app *App) ImportConfig(path string) error {
	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s", path)
	}

	// 读取文件
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	var importedConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &importedConfig); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 设置配置
	if app.dynamicConfig != nil {
		app.dynamicConfig.SetAll(importedConfig)
		if err := app.dynamicConfig.Save(); err != nil {
			return fmt.Errorf("保存配置失败: %w", err)
		}
	} else {
		for k, v := range importedConfig {
			app.configManager.Set(k, v)
		}
		if err := app.configManager.SaveConfig(); err != nil {
			return fmt.Errorf("保存配置失败: %w", err)
		}
	}

	// 重新加载配置
	return app.ReloadConfig()
}
