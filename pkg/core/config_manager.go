package core

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/viper"
)

// ConfigManager 管理应用程序配置
type ConfigManager struct {
	// 配置文件路径
	configFile string

	// 默认配置
	defaults map[string]interface{}
}

// NewConfigManager 创建一个新的配置管理器
func NewConfigManager(configFile string) *ConfigManager {
	return &ConfigManager{
		configFile: configFile,
		defaults: map[string]interface{}{
			"plugin_dir":             "plugins",
			"log_level":              "info",
			"log_file":               "agent.log",
			"enable_assets":          true,
			"enable_device":          true,
			"enable_dlp":             true,
			"enable_control":         true,
			"enable_comm":            true,
			"server_url":             "ws://localhost:8080/ws",
			"heartbeat_interval":     "30s",
			"reconnect_interval":     "5s",
			"max_reconnect_attempts": 10,
			"comm_shutdown_timeout":  5,
			"comm_security": map[string]interface{}{
				"enable_tls":            false,
				"verify_server_cert":    true,
				"client_cert_file":      "",
				"client_key_file":       "",
				"ca_cert_file":          "",
				"enable_encryption":     false,
				"encryption_key":        "",
				"enable_auth":           false,
				"auth_token":            "",
				"auth_type":             "token",
				"username":              "",
				"password":              "",
				"enable_compression":    false,
				"compression_level":     6,
				"compression_threshold": 1024,
			},
			"web_console": map[string]interface{}{
				"enabled":         true,
				"host":            "0.0.0.0",
				"port":            8088,
				"enable_https":    false,
				"cert_file":       "",
				"key_file":        "",
				"enable_auth":     true,
				"username":        "admin",
				"password":        "admin",
				"session_timeout": "24h",
				"static_dir":      "./web/dist",
				"log_level":       "info",
				"rate_limit":      100,
				"enable_csrf":     true,
				"api_prefix":      "/api",
			},
			"os":   runtime.GOOS,
			"arch": runtime.GOARCH,
		},
	}
}

// InitConfig 初始化配置
func (cm *ConfigManager) InitConfig() error {
	// 设置默认值
	for k, v := range cm.defaults {
		viper.SetDefault(k, v)
	}

	// 如果指定了配置文件
	if cm.configFile != "" {
		viper.SetConfigFile(cm.configFile)
	} else {
		// 在当前目录和用户主目录中查找配置文件
		viper.AddConfigPath(".")
		viper.AddConfigPath(filepath.Join(os.Getenv("HOME"), ".appframework"))
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// 读取环境变量
	viper.AutomaticEnv()
	viper.SetEnvPrefix("APPFW")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，创建一个默认配置文件
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return cm.CreateDefaultConfig()
		}
		return fmt.Errorf("无法读取配置文件: %w", err)
	}

	return nil
}

// CreateDefaultConfig 创建默认配置文件
func (cm *ConfigManager) CreateDefaultConfig() error {
	// 确定配置文件路径
	configPath := cm.configFile
	if configPath == "" {
		// 使用默认路径
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("无法获取用户主目录: %w", err)
		}

		configDir := filepath.Join(homeDir, ".appframework")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("无法创建配置目录: %w", err)
		}

		configPath = filepath.Join(configDir, "config.yaml")
	}

	// 设置默认值
	for k, v := range cm.defaults {
		viper.Set(k, v)
	}

	// 写入配置文件
	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("无法写入配置文件: %w", err)
	}

	fmt.Printf("已创建默认配置文件: %s\n", configPath)
	return nil
}

// GetString 获取字符串配置
func (cm *ConfigManager) GetString(key string) string {
	return viper.GetString(key)
}

// GetBool 获取布尔配置
func (cm *ConfigManager) GetBool(key string) bool {
	return viper.GetBool(key)
}

// GetInt 获取整数配置
func (cm *ConfigManager) GetInt(key string) int {
	return viper.GetInt(key)
}

// GetIntOrDefault 获取整数配置，如果不存在则返回默认值
func (cm *ConfigManager) GetIntOrDefault(key string, defaultValue int) int {
	if !viper.IsSet(key) {
		return defaultValue
	}
	return viper.GetInt(key)
}

// GetInt64 获取64位整数配置
func (cm *ConfigManager) GetInt64(key string) int64 {
	return viper.GetInt64(key)
}

// GetInt64OrDefault 获取64位整数配置，如果不存在则返回默认值
func (cm *ConfigManager) GetInt64OrDefault(key string, defaultValue int64) int64 {
	if !viper.IsSet(key) {
		return defaultValue
	}
	return viper.GetInt64(key)
}

// GetFloat64 获取浮点数配置
func (cm *ConfigManager) GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}

// GetFloat64OrDefault 获取浮点数配置，如果不存在则返回默认值
func (cm *ConfigManager) GetFloat64OrDefault(key string, defaultValue float64) float64 {
	if !viper.IsSet(key) {
		return defaultValue
	}
	return viper.GetFloat64(key)
}

// GetDuration 获取时间间隔配置
func (cm *ConfigManager) GetDuration(key string) time.Duration {
	return viper.GetDuration(key)
}

// GetDurationOrDefault 获取时间间隔配置，如果不存在则返回默认值
func (cm *ConfigManager) GetDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if !viper.IsSet(key) {
		return defaultValue
	}
	return viper.GetDuration(key)
}

// GetBoolOrDefault 获取布尔配置，如果不存在则返回默认值
func (cm *ConfigManager) GetBoolOrDefault(key string, defaultValue bool) bool {
	if !viper.IsSet(key) {
		return defaultValue
	}
	return viper.GetBool(key)
}

// GetStringOrDefault 获取字符串配置，如果不存在则返回默认值
func (cm *ConfigManager) GetStringOrDefault(key string, defaultValue string) string {
	if !viper.IsSet(key) {
		return defaultValue
	}
	return viper.GetString(key)
}

// GetStringMap 获取字符串映射配置
func (cm *ConfigManager) GetStringMap(key string) map[string]interface{} {
	return viper.GetStringMap(key)
}

// Set 设置配置
func (cm *ConfigManager) Set(key string, value interface{}) {
	viper.Set(key, value)
}

// SaveConfig 保存配置
func (cm *ConfigManager) SaveConfig() error {
	return viper.WriteConfig()
}

// GetAllConfig 获取所有配置
func (cm *ConfigManager) GetAllConfig() map[string]interface{} {
	return viper.AllSettings()
}

// UpdateConfig 更新配置
func (cm *ConfigManager) UpdateConfig(config map[string]interface{}) error {
	// 更新配置
	for k, v := range config {
		viper.Set(k, v)
	}
	return nil
}

// ResetConfig 重置配置
func (cm *ConfigManager) ResetConfig() error {
	// 重置为默认配置
	for k, v := range cm.defaults {
		viper.Set(k, v)
	}
	return viper.WriteConfig()
}

// GetConfigPath 获取配置文件路径
func (cm *ConfigManager) GetConfigPath() string {
	return viper.ConfigFileUsed()
}

// SetConfigPath 设置配置文件路径
func (cm *ConfigManager) SetConfigPath(path string) {
	cm.configFile = path
	viper.SetConfigFile(path)
}

// Get 获取配置值
func (cm *ConfigManager) Get(key string) interface{} {
	return viper.Get(key)
}
