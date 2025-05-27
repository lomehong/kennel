package plugin

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/lomehong/kennel/pkg/core/config"
)

// ConfigIntegration 配置集成
type ConfigIntegration struct {
	// 配置管理器
	configManager *config.ConfigManager

	// 插件管理器
	pluginManager *PluginManager

	// 插件配置验证器
	validators map[string]*config.PluginConfigValidator
}

// NewConfigIntegration 创建配置集成
func NewConfigIntegration(configManager *config.ConfigManager, pluginManager *PluginManager) *ConfigIntegration {
	return &ConfigIntegration{
		configManager: configManager,
		pluginManager: pluginManager,
		validators:    make(map[string]*config.PluginConfigValidator),
	}
}

// RegisterPluginValidator 注册插件配置验证器
func (ci *ConfigIntegration) RegisterPluginValidator(pluginID string, validator *config.PluginConfigValidator) {
	ci.validators[pluginID] = validator
}

// Initialize 初始化配置集成
func (ci *ConfigIntegration) Initialize() error {
	// 注册配置变更监听器
	ci.configManager.AddConfigChangeListener(ci.handleConfigChange)

	// 获取插件管理配置
	pluginManagerConfig := ci.configManager.GetPluginManagerConfig()

	// 设置插件目录
	if pluginDir, ok := pluginManagerConfig["plugin_dir"].(string); ok {
		ci.pluginManager.SetPluginsDir(pluginDir)
	}

	// 扫描插件目录
	metadataList, err := ci.pluginManager.ScanPluginsDir()
	if err != nil {
		return fmt.Errorf("扫描插件目录失败: %w", err)
	}

	// 注册插件配置验证器
	allValidators := config.GetAllPluginValidators()
	for _, metadata := range metadataList {
		// 获取预定义的验证器
		if validator, exists := allValidators[metadata.ID]; exists {
			ci.RegisterPluginValidator(metadata.ID, validator)
		} else {
			// 创建基本验证器作为后备
			validator := config.NewPluginConfigValidator(metadata.ID)
			validator.AddFieldType("enabled", reflect.Bool)
			validator.AddDefault("enabled", true)
			ci.RegisterPluginValidator(metadata.ID, validator)
		}
	}

	// 验证配置
	allConfig := ci.configManager.GetAllPluginConfigs()
	for pluginID, pluginConfig := range allConfig {
		if validator, exists := ci.validators[pluginID]; exists {
			// 构建完整配置
			fullConfig := map[string]interface{}{
				"plugins": map[string]interface{}{
					pluginID: pluginConfig,
				},
			}

			// 验证配置
			if err := validator.Validate(fullConfig); err != nil {
				return fmt.Errorf("验证插件 %s 配置失败: %w", pluginID, err)
			}
		}
	}

	// 自动加载插件
	if discovery, ok := pluginManagerConfig["discovery"].(map[string]interface{}); ok {
		if autoLoad, ok := discovery["auto_load"].(bool); ok && autoLoad {
			for _, metadata := range metadataList {
				// 获取插件配置
				pluginConfig := ci.configManager.GetPluginConfig(metadata.ID)

				// 检查是否启用
				if enabled, ok := pluginConfig["enabled"].(bool); ok && enabled {
					// 加载插件
					if _, err := ci.pluginManager.LoadPlugin(metadata); err != nil {
						return fmt.Errorf("加载插件 %s 失败: %w", metadata.ID, err)
					}

					// 启动插件
					if err := ci.pluginManager.StartPlugin(metadata.ID); err != nil {
						return fmt.Errorf("启动插件 %s 失败: %w", metadata.ID, err)
					}
				}
			}
		}
	}

	return nil
}

// handleConfigChange 处理配置变更
func (ci *ConfigIntegration) handleConfigChange(configType string, oldConfig, newConfig map[string]interface{}) error {
	// 处理插件管理配置变更
	if configType == "plugin_manager" {
		// 设置插件目录
		if pluginDir, ok := newConfig["plugin_dir"].(string); ok {
			ci.pluginManager.SetPluginsDir(pluginDir)
		}
	}

	// 处理插件配置变更
	if strings.HasPrefix(configType, "plugin:") {
		pluginID := strings.TrimPrefix(configType, "plugin:")

		// 获取插件实例
		plugin, exists := ci.pluginManager.GetPlugin(pluginID)
		if !exists {
			return nil
		}

		// 检查是否启用状态变更
		oldEnabled := getConfigBool(oldConfig, "enabled", true)
		newEnabled := getConfigBool(newConfig, "enabled", true)

		if oldEnabled && !newEnabled {
			// 停止插件
			if err := ci.pluginManager.StopPlugin(pluginID); err != nil {
				return fmt.Errorf("停止插件 %s 失败: %w", pluginID, err)
			}
		} else if !oldEnabled && newEnabled {
			// 启动插件
			if err := ci.pluginManager.StartPlugin(pluginID); err != nil {
				return fmt.Errorf("启动插件 %s 失败: %w", pluginID, err)
			}
		} else if oldEnabled && newEnabled && plugin.State == PluginStateRunning {
			// 重新加载插件配置
			if err := ci.reloadPluginConfig(pluginID, newConfig); err != nil {
				return fmt.Errorf("重新加载插件 %s 配置失败: %w", pluginID, err)
			}
		}
	}

	return nil
}

// reloadPluginConfig 重新加载插件配置
func (ci *ConfigIntegration) reloadPluginConfig(pluginID string, config map[string]interface{}) error {
	// 获取插件实例
	plugin, exists := ci.pluginManager.GetPlugin(pluginID)
	if !exists {
		return fmt.Errorf("插件 %s 不存在", pluginID)
	}

	// 创建模块配置
	moduleConfig := &ModuleConfig{
		ID:           pluginID,
		Name:         plugin.Metadata.Name,
		Version:      plugin.Metadata.Version,
		Settings:     config,
		Dependencies: plugin.Metadata.Dependencies,
	}

	// 重新初始化插件
	ctx := context.Background()
	if err := plugin.Instance.Init(ctx, moduleConfig); err != nil {
		return fmt.Errorf("初始化插件失败: %w", err)
	}

	return nil
}

// LoadPluginFromConfig 从配置加载插件
func (ci *ConfigIntegration) LoadPluginFromConfig(pluginID string) (*PluginInstance, error) {
	// 获取插件配置
	pluginConfig := ci.configManager.GetPluginConfig(pluginID)

	// 检查是否启用
	if enabled, ok := pluginConfig["enabled"].(bool); ok && !enabled {
		return nil, fmt.Errorf("插件 %s 未启用", pluginID)
	}

	// 扫描插件目录
	metadataList, err := ci.pluginManager.ScanPluginsDir()
	if err != nil {
		return nil, fmt.Errorf("扫描插件目录失败: %w", err)
	}

	// 查找插件元数据
	var metadata PluginMetadata
	found := false
	for _, m := range metadataList {
		if m.ID == pluginID {
			metadata = m
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("未找到插件 %s", pluginID)
	}

	// 加载插件
	instance, err := ci.pluginManager.LoadPlugin(metadata)
	if err != nil {
		return nil, fmt.Errorf("加载插件 %s 失败: %w", pluginID, err)
	}

	// 启动插件
	if err := ci.pluginManager.StartPlugin(pluginID); err != nil {
		return nil, fmt.Errorf("启动插件 %s 失败: %w", pluginID, err)
	}

	return instance, nil
}

// getConfigBool 获取配置布尔值
func getConfigBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := config[key].(bool); ok {
		return value
	}
	return defaultValue
}
