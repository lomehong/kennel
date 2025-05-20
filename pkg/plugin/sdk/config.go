package sdk

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

// ConfigManager 配置管理器
// 负责管理插件配置
type ConfigManager struct {
	// 插件ID
	pluginID string

	// 配置目录
	configDir string

	// 配置文件名
	configFile string

	// 日志记录器
	logger hclog.Logger

	// 配置数据
	data map[string]interface{}

	// 环境变量前缀
	envPrefix string
}

// ConfigOption 配置选项
type ConfigOption func(*ConfigManager)

// WithConfigDir 设置配置目录
func WithConfigDir(dir string) ConfigOption {
	return func(cm *ConfigManager) {
		cm.configDir = dir
	}
}

// WithConfigFile 设置配置文件名
func WithConfigFile(file string) ConfigOption {
	return func(cm *ConfigManager) {
		cm.configFile = file
	}
}

// WithEnvPrefix 设置环境变量前缀
func WithEnvPrefix(prefix string) ConfigOption {
	return func(cm *ConfigManager) {
		cm.envPrefix = prefix
	}
}

// NewConfigManager 创建一个新的配置管理器
func NewConfigManager(pluginID string, logger hclog.Logger, options ...ConfigOption) (*ConfigManager, error) {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	// 获取配置目录
	configDir, err := GetPluginConfigDir(pluginID)
	if err != nil {
		return nil, fmt.Errorf("获取配置目录失败: %w", err)
	}

	cm := &ConfigManager{
		pluginID:   pluginID,
		configDir:  configDir,
		configFile: "config.yaml",
		logger:     logger.Named("config-manager"),
		data:       make(map[string]interface{}),
		envPrefix:  strings.ToUpper(pluginID),
	}

	// 应用选项
	for _, option := range options {
		option(cm)
	}

	return cm, nil
}

// Load 加载配置
func (cm *ConfigManager) Load() error {
	// 构建配置文件路径
	configPath := filepath.Join(cm.configDir, cm.configFile)

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cm.logger.Warn("配置文件不存在", "path", configPath)
		return nil
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置文件
	if err := yaml.Unmarshal(data, &cm.data); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	cm.logger.Debug("加载配置", "path", configPath)
	return nil
}

// Save 保存配置
func (cm *ConfigManager) Save() error {
	// 构建配置文件路径
	configPath := filepath.Join(cm.configDir, cm.configFile)

	// 序列化配置
	data, err := yaml.Marshal(cm.data)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入配置文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	cm.logger.Debug("保存配置", "path", configPath)
	return nil
}

// Get 获取配置值
func (cm *ConfigManager) Get(key string) interface{} {
	// 检查环境变量
	envKey := cm.envPrefix + "_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if value, exists := os.LookupEnv(envKey); exists {
		return value
	}

	// 检查配置数据
	keys := strings.Split(key, ".")
	value := interface{}(cm.data)

	for _, k := range keys {
		// 检查值是否为映射
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil
		}

		// 获取下一级值
		value, ok = m[k]
		if !ok {
			return nil
		}
	}

	return value
}

// Set 设置配置值
func (cm *ConfigManager) Set(key string, value interface{}) {
	// 分割键
	keys := strings.Split(key, ".")
	lastKey := keys[len(keys)-1]
	parentKeys := keys[:len(keys)-1]

	// 获取父级映射
	parent := interface{}(cm.data)

	for _, k := range parentKeys {
		// 检查值是否为映射
		m, ok := parent.(map[string]interface{})
		if !ok {
			// 创建新映射
			m = make(map[string]interface{})
			parent = m
		}

		// 获取或创建下一级映射
		next, ok := m[k]
		if !ok {
			next = make(map[string]interface{})
			m[k] = next
		}

		parent = next
	}

	// 设置值
	if m, ok := parent.(map[string]interface{}); ok {
		m[lastKey] = value
	}
}

// GetString 获取字符串配置值
func (cm *ConfigManager) GetString(key string) string {
	value := cm.Get(key)
	if value == nil {
		return ""
	}

	// 转换为字符串
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetInt 获取整数配置值
func (cm *ConfigManager) GetInt(key string) int {
	value := cm.Get(key)
	if value == nil {
		return 0
	}

	// 转换为整数
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		var i int
		fmt.Sscanf(v, "%d", &i)
		return i
	default:
		return 0
	}
}

// GetBool 获取布尔配置值
func (cm *ConfigManager) GetBool(key string) bool {
	value := cm.Get(key)
	if value == nil {
		return false
	}

	// 转换为布尔值
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true" || v == "1"
	case int:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

// GetFloat 获取浮点数配置值
func (cm *ConfigManager) GetFloat(key string) float64 {
	value := cm.Get(key)
	if value == nil {
		return 0
	}

	// 转换为浮点数
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f
	default:
		return 0
	}
}

// GetStringSlice 获取字符串切片配置值
func (cm *ConfigManager) GetStringSlice(key string) []string {
	value := cm.Get(key)
	if value == nil {
		return nil
	}

	// 转换为字符串切片
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	case string:
		return strings.Split(v, ",")
	default:
		return nil
	}
}

// GetStringMap 获取字符串映射配置值
func (cm *ConfigManager) GetStringMap(key string) map[string]string {
	value := cm.Get(key)
	if value == nil {
		return nil
	}

	// 转换为字符串映射
	switch v := value.(type) {
	case map[string]string:
		return v
	case map[string]interface{}:
		result := make(map[string]string)
		for k, item := range v {
			result[k] = fmt.Sprintf("%v", item)
		}
		return result
	default:
		return nil
	}
}

// GetStruct 获取结构体配置值
func (cm *ConfigManager) GetStruct(key string, result interface{}) error {
	value := cm.Get(key)
	if value == nil {
		return fmt.Errorf("配置 %s 不存在", key)
	}

	// 检查结果是否为指针
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr {
		return fmt.Errorf("结果必须为指针")
	}

	// 解码配置
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           result,
		WeaklyTypedInput: true,
		TagName:          "yaml",
	})
	if err != nil {
		return fmt.Errorf("创建解码器失败: %w", err)
	}

	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("解码配置失败: %w", err)
	}

	return nil
}

// GetPluginConfig 获取插件配置
func (cm *ConfigManager) GetPluginConfig() (api.PluginConfig, error) {
	config := api.PluginConfig{
		ID:       cm.pluginID,
		Enabled:  true,
		LogLevel: "info",
		Settings: make(map[string]interface{}),
	}

	// 获取启用状态
	config.Enabled = cm.GetBool("enabled")

	// 获取日志级别
	if logLevel := cm.GetString("log_level"); logLevel != "" {
		config.LogLevel = logLevel
	}

	// 获取设置
	if settings := cm.Get("settings"); settings != nil {
		if settingsMap, ok := settings.(map[string]interface{}); ok {
			for k, v := range settingsMap {
				config.Settings[k] = v
			}
		}
	}

	return config, nil
}

// SetPluginConfig 设置插件配置
func (cm *ConfigManager) SetPluginConfig(config api.PluginConfig) {
	// 设置启用状态
	cm.Set("enabled", config.Enabled)

	// 设置日志级别
	cm.Set("log_level", config.LogLevel)

	// 设置设置
	for k, v := range config.Settings {
		cm.Set("settings."+k, v)
	}
}

// GetData 获取配置数据
func (cm *ConfigManager) GetData() map[string]interface{} {
	return cm.data
}

// SetData 设置配置数据
func (cm *ConfigManager) SetData(data map[string]interface{}) {
	cm.data = data
}

// GetConfigDir 获取配置目录
func (cm *ConfigManager) GetConfigDir() string {
	return cm.configDir
}

// GetConfigFile 获取配置文件
func (cm *ConfigManager) GetConfigFile() string {
	return cm.configFile
}

// GetConfigPath 获取配置文件路径
func (cm *ConfigManager) GetConfigPath() string {
	return filepath.Join(cm.configDir, cm.configFile)
}
