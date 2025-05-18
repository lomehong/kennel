package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/go-hclog"
	"gopkg.in/yaml.v3"
)

// ConfigFormat 配置格式
type ConfigFormat string

// 预定义配置格式
const (
	ConfigFormatYAML ConfigFormat = "yaml"
	ConfigFormatJSON ConfigFormat = "json"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	// 全局配置
	globalConfig map[string]interface{}

	// 插件管理配置
	pluginManagerConfig map[string]interface{}

	// 插件配置
	pluginConfigs map[string]map[string]interface{}

	// 配置文件路径
	configPath string

	// 配置格式
	format ConfigFormat

	// 配置监视器
	watcher *fsnotify.Watcher

	// 配置变更监听器
	listeners []ConfigChangeListener

	// 配置验证器
	validators []ConfigValidator

	// 日志记录器
	logger hclog.Logger

	// 互斥锁
	mu sync.RWMutex
}

// ConfigChangeListener 配置变更监听器
type ConfigChangeListener func(configType string, oldConfig, newConfig map[string]interface{}) error

// ConfigValidator 配置验证器
type ConfigValidator interface {
	// 验证配置
	Validate(config map[string]interface{}) error

	// 获取默认配置
	GetDefaults() map[string]interface{}

	// 获取配置架构
	GetSchema() map[string]interface{}
}

// ConfigManagerOption 配置管理器选项
type ConfigManagerOption func(*ConfigManager)

// WithConfigPath 设置配置文件路径
func WithConfigPath(path string) ConfigManagerOption {
	return func(cm *ConfigManager) {
		cm.configPath = path
	}
}

// WithConfigFormat 设置配置格式
func WithConfigFormat(format ConfigFormat) ConfigManagerOption {
	return func(cm *ConfigManager) {
		cm.format = format
	}
}

// WithConfigLogger 设置日志记录器
func WithConfigLogger(logger hclog.Logger) ConfigManagerOption {
	return func(cm *ConfigManager) {
		cm.logger = logger
	}
}

// WithConfigValidator 添加配置验证器
func WithConfigValidator(validator ConfigValidator) ConfigManagerOption {
	return func(cm *ConfigManager) {
		cm.validators = append(cm.validators, validator)
	}
}

// WithConfigChangeListener 添加配置变更监听器
func WithConfigChangeListener(listener ConfigChangeListener) ConfigManagerOption {
	return func(cm *ConfigManager) {
		cm.listeners = append(cm.listeners, listener)
	}
}

// NewConfigManager 创建配置管理器
func NewConfigManager(options ...ConfigManagerOption) (*ConfigManager, error) {
	// 创建配置监视器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("创建配置监视器失败: %w", err)
	}

	cm := &ConfigManager{
		globalConfig:        make(map[string]interface{}),
		pluginManagerConfig: make(map[string]interface{}),
		pluginConfigs:       make(map[string]map[string]interface{}),
		configPath:          "config.yaml",
		format:              ConfigFormatYAML,
		watcher:             watcher,
		listeners:           make([]ConfigChangeListener, 0),
		validators:          make([]ConfigValidator, 0),
		logger:              hclog.NewNullLogger(),
	}

	// 应用选项
	for _, option := range options {
		option(cm)
	}

	// 确定配置格式
	if cm.format == "" {
		ext := filepath.Ext(cm.configPath)
		if ext == ".json" {
			cm.format = ConfigFormatJSON
		} else {
			cm.format = ConfigFormatYAML
		}
	}

	// 加载配置
	if err := cm.Load(); err != nil {
		return nil, err
	}

	// 启动配置监视
	go cm.watchConfig()

	return cm, nil
}

// Load 加载配置
func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		cm.logger.Warn("配置文件不存在", "path", cm.configPath)
		return nil
	}

	// 读取文件
	data, err := ioutil.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	var config map[string]interface{}
	switch cm.format {
	case ConfigFormatYAML:
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("解析YAML配置失败: %w", err)
		}
	case ConfigFormatJSON:
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("解析JSON配置失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的配置格式: %s", cm.format)
	}

	// 验证配置
	for _, validator := range cm.validators {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("配置验证失败: %w", err)
		}
	}

	// 提取全局配置
	if global, ok := config["global"].(map[string]interface{}); ok {
		cm.globalConfig = global
	} else {
		cm.globalConfig = make(map[string]interface{})
	}

	// 提取插件管理配置
	if pluginManager, ok := config["plugin_manager"].(map[string]interface{}); ok {
		cm.pluginManagerConfig = pluginManager
	} else {
		cm.pluginManagerConfig = make(map[string]interface{})
	}

	// 提取插件配置
	if plugins, ok := config["plugins"].(map[string]interface{}); ok {
		for name, pluginConfig := range plugins {
			if pc, ok := pluginConfig.(map[string]interface{}); ok {
				cm.pluginConfigs[name] = pc
			}
		}
	}

	// 监视配置文件
	cm.watcher.Add(cm.configPath)

	cm.logger.Info("加载配置成功", "path", cm.configPath)
	return nil
}

// Save 保存配置
func (cm *ConfigManager) Save() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 构建完整配置
	config := make(map[string]interface{})
	config["global"] = cm.globalConfig
	config["plugin_manager"] = cm.pluginManagerConfig
	config["plugins"] = cm.pluginConfigs

	// 验证配置
	for _, validator := range cm.validators {
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("配置验证失败: %w", err)
		}
	}

	// 序列化配置
	var data []byte
	var err error
	switch cm.format {
	case ConfigFormatYAML:
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("序列化YAML配置失败: %w", err)
		}
	case ConfigFormatJSON:
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("序列化JSON配置失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的配置格式: %s", cm.format)
	}

	// 确保目录存在
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	if err := ioutil.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	cm.logger.Info("保存配置成功", "path", cm.configPath)
	return nil
}

// watchConfig 监视配置文件变化
func (cm *ConfigManager) watchConfig() {
	for {
		select {
		case event, ok := <-cm.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				cm.logger.Info("配置文件已修改", "path", event.Name)
				if err := cm.Reload(); err != nil {
					cm.logger.Error("重新加载配置失败", "error", err)
				}
			}
		case err, ok := <-cm.watcher.Errors:
			if !ok {
				return
			}
			cm.logger.Error("配置监视器错误", "error", err)
		}
	}
}

// Reload 重新加载配置
func (cm *ConfigManager) Reload() error {
	// 保存旧配置
	cm.mu.RLock()
	oldGlobalConfig := copyMap(cm.globalConfig)
	oldPluginManagerConfig := copyMap(cm.pluginManagerConfig)
	oldPluginConfigs := make(map[string]map[string]interface{})
	for name, config := range cm.pluginConfigs {
		oldPluginConfigs[name] = copyMap(config)
	}
	cm.mu.RUnlock()

	// 加载新配置
	if err := cm.Load(); err != nil {
		return err
	}

	// 通知监听器
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 检查全局配置变化
	if !mapsEqual(oldGlobalConfig, cm.globalConfig) {
		for _, listener := range cm.listeners {
			if err := listener("global", oldGlobalConfig, cm.globalConfig); err != nil {
				cm.logger.Error("配置变更监听器失败", "type", "global", "error", err)
			}
		}
	}

	// 检查插件管理配置变化
	if !mapsEqual(oldPluginManagerConfig, cm.pluginManagerConfig) {
		for _, listener := range cm.listeners {
			if err := listener("plugin_manager", oldPluginManagerConfig, cm.pluginManagerConfig); err != nil {
				cm.logger.Error("配置变更监听器失败", "type", "plugin_manager", "error", err)
			}
		}
	}

	// 检查插件配置变化
	for name, oldConfig := range oldPluginConfigs {
		newConfig, exists := cm.pluginConfigs[name]
		if !exists {
			// 插件配置已删除
			for _, listener := range cm.listeners {
				if err := listener("plugin:"+name, oldConfig, nil); err != nil {
					cm.logger.Error("配置变更监听器失败", "type", "plugin:"+name, "error", err)
				}
			}
		} else if !mapsEqual(oldConfig, newConfig) {
			// 插件配置已更改
			for _, listener := range cm.listeners {
				if err := listener("plugin:"+name, oldConfig, newConfig); err != nil {
					cm.logger.Error("配置变更监听器失败", "type", "plugin:"+name, "error", err)
				}
			}
		}
	}

	// 检查新增的插件配置
	for name, newConfig := range cm.pluginConfigs {
		if _, exists := oldPluginConfigs[name]; !exists {
			// 新增插件配置
			for _, listener := range cm.listeners {
				if err := listener("plugin:"+name, nil, newConfig); err != nil {
					cm.logger.Error("配置变更监听器失败", "type", "plugin:"+name, "error", err)
				}
			}
		}
	}

	return nil
}

// GetGlobalConfig 获取全局配置
func (cm *ConfigManager) GetGlobalConfig() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return copyMap(cm.globalConfig)
}

// GetPluginManagerConfig 获取插件管理配置
func (cm *ConfigManager) GetPluginManagerConfig() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return copyMap(cm.pluginManagerConfig)
}

// GetPluginConfig 获取插件配置
func (cm *ConfigManager) GetPluginConfig(name string) map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	if config, exists := cm.pluginConfigs[name]; exists {
		return copyMap(config)
	}
	return make(map[string]interface{})
}

// GetAllPluginConfigs 获取所有插件配置
func (cm *ConfigManager) GetAllPluginConfigs() map[string]map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	configs := make(map[string]map[string]interface{})
	for name, config := range cm.pluginConfigs {
		configs[name] = copyMap(config)
	}
	return configs
}

// SetGlobalConfig 设置全局配置
func (cm *ConfigManager) SetGlobalConfig(config map[string]interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.globalConfig = copyMap(config)
}

// SetPluginManagerConfig 设置插件管理配置
func (cm *ConfigManager) SetPluginManagerConfig(config map[string]interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.pluginManagerConfig = copyMap(config)
}

// SetPluginConfig 设置插件配置
func (cm *ConfigManager) SetPluginConfig(name string, config map[string]interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.pluginConfigs[name] = copyMap(config)
}

// AddConfigChangeListener 添加配置变更监听器
func (cm *ConfigManager) AddConfigChangeListener(listener ConfigChangeListener) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.listeners = append(cm.listeners, listener)
}

// Close 关闭配置管理器
func (cm *ConfigManager) Close() error {
	return cm.watcher.Close()
}

// copyMap 复制映射
func copyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = copyMap(val)
		case []interface{}:
			result[k] = copySlice(val)
		default:
			result[k] = v
		}
	}
	return result
}

// copySlice 复制切片
func copySlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]interface{}:
			result[i] = copyMap(val)
		case []interface{}:
			result[i] = copySlice(val)
		default:
			result[i] = v
		}
	}
	return result
}

// mapsEqual 比较两个映射是否相等
func mapsEqual(m1, m2 map[string]interface{}) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k, v1 := range m1 {
		v2, ok := m2[k]
		if !ok {
			return false
		}
		switch val1 := v1.(type) {
		case map[string]interface{}:
			val2, ok := v2.(map[string]interface{})
			if !ok || !mapsEqual(val1, val2) {
				return false
			}
		case []interface{}:
			val2, ok := v2.([]interface{})
			if !ok || !slicesEqual(val1, val2) {
				return false
			}
		default:
			if v1 != v2 {
				return false
			}
		}
	}
	return true
}

// slicesEqual 比较两个切片是否相等
func slicesEqual(s1, s2 []interface{}) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i, v1 := range s1 {
		v2 := s2[i]
		switch val1 := v1.(type) {
		case map[string]interface{}:
			val2, ok := v2.(map[string]interface{})
			if !ok || !mapsEqual(val1, val2) {
				return false
			}
		case []interface{}:
			val2, ok := v2.([]interface{})
			if !ok || !slicesEqual(val1, val2) {
				return false
			}
		default:
			if v1 != v2 {
				return false
			}
		}
	}
	return true
}
