package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigMigration 配置迁移器
type ConfigMigration struct {
	sourceFile string
	targetFile string
}

// NewConfigMigration 创建配置迁移器
func NewConfigMigration(sourceFile, targetFile string) *ConfigMigration {
	return &ConfigMigration{
		sourceFile: sourceFile,
		targetFile: targetFile,
	}
}

// LegacyConfig 旧版配置结构
type LegacyConfig struct {
	PluginDir    string                 `yaml:"plugin_dir"`
	LogLevel     string                 `yaml:"log_level"`
	LogFile      string                 `yaml:"log_file"`
	WebConsole   map[string]interface{} `yaml:"web_console"`
	EnableAssets bool                   `yaml:"enable_assets"`
	EnableDevice bool                   `yaml:"enable_device"`
	EnableDLP    bool                   `yaml:"enable_dlp"`
	EnableControl bool                  `yaml:"enable_control"`
	EnableAudit  bool                   `yaml:"enable_audit"`
	EnableComm   bool                   `yaml:"enable_comm"`
	Comm         map[string]interface{} `yaml:"comm"`
	Assets       map[string]interface{} `yaml:"assets"`
	Device       map[string]interface{} `yaml:"device"`
	DLP          map[string]interface{} `yaml:"dlp"`
	Control      map[string]interface{} `yaml:"control"`
	Audit        map[string]interface{} `yaml:"audit"`
}

// NewConfig 新版配置结构
type NewConfig struct {
	Global        GlobalConfig        `yaml:"global"`
	PluginManager PluginManagerConfig `yaml:"plugin_manager"`
	Comm          CommConfig          `yaml:"comm"`
	WebConsole    WebConsoleConfig    `yaml:"web_console"`
	Plugins       PluginsConfig       `yaml:"plugins"`
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	App     AppConfig     `yaml:"app"`
	Logging LoggingConfig `yaml:"logging"`
	System  SystemConfig  `yaml:"system"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
	Author      string `yaml:"author"`
	License     string `yaml:"license"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"max_size"`
	MaxAge     int    `yaml:"max_age"`
	MaxBackups int    `yaml:"max_backups"`
	Compress   bool   `yaml:"compress"`
}

// SystemConfig 系统配置
type SystemConfig struct {
	MaxConcurrency int    `yaml:"max_concurrency"`
	WorkerPoolSize int    `yaml:"worker_pool_size"`
	QueueSize      int    `yaml:"queue_size"`
	Timeout        string `yaml:"timeout"`
}

// PluginManagerConfig 插件管理配置
type PluginManagerConfig struct {
	PluginDir string                 `yaml:"plugin_dir"`
	Discovery map[string]interface{} `yaml:"discovery"`
	Isolation map[string]interface{} `yaml:"isolation"`
	Lifecycle map[string]interface{} `yaml:"lifecycle"`
}

// CommConfig 通讯配置
type CommConfig struct {
	Enabled       bool   `yaml:"enabled"`
	ServerAddress string `yaml:"server_address"`
	ServerPort    int    `yaml:"server_port"`
	Protocol      string `yaml:"protocol"`
	Timeout       int    `yaml:"timeout"`
	RetryInterval int    `yaml:"retry_interval"`
	MaxRetries    int    `yaml:"max_retries"`
	KeepAlive     bool   `yaml:"keep_alive"`
}

// WebConsoleConfig Web控制台配置
type WebConsoleConfig struct {
	Enabled        bool     `yaml:"enabled"`
	Host           string   `yaml:"host"`
	Port           int      `yaml:"port"`
	EnableHTTPS    bool     `yaml:"enable_https"`
	CertFile       string   `yaml:"cert_file"`
	KeyFile        string   `yaml:"key_file"`
	EnableAuth     bool     `yaml:"enable_auth"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	StaticDir      string   `yaml:"static_dir"`
	LogLevel       string   `yaml:"log_level"`
	RateLimit      int      `yaml:"rate_limit"`
	EnableCSRF     bool     `yaml:"enable_csrf"`
	APIPrefix      string   `yaml:"api_prefix"`
	SessionTimeout string   `yaml:"session_timeout"`
	AllowOrigins   []string `yaml:"allow_origins"`
}

// PluginsConfig 插件配置
type PluginsConfig struct {
	Assets  PluginConfig `yaml:"assets"`
	Device  PluginConfig `yaml:"device"`
	DLP     PluginConfig `yaml:"dlp"`
	Control PluginConfig `yaml:"control"`
	Audit   PluginConfig `yaml:"audit"`
}

// PluginConfig 插件配置
type PluginConfig struct {
	Enabled        bool                   `yaml:"enabled"`
	Name           string                 `yaml:"name"`
	Version        string                 `yaml:"version"`
	Path           string                 `yaml:"path"`
	AutoStart      bool                   `yaml:"auto_start"`
	AutoRestart    bool                   `yaml:"auto_restart"`
	IsolationLevel string                 `yaml:"isolation_level"`
	Settings       map[string]interface{} `yaml:"settings"`
}

// Migrate 执行配置迁移
func (cm *ConfigMigration) Migrate() error {
	// 读取旧版配置
	legacyConfig, err := cm.readLegacyConfig()
	if err != nil {
		return fmt.Errorf("读取旧版配置失败: %w", err)
	}

	// 转换为新版配置
	newConfig := cm.convertToNewConfig(legacyConfig)

	// 写入新版配置
	if err := cm.writeNewConfig(newConfig); err != nil {
		return fmt.Errorf("写入新版配置失败: %w", err)
	}

	return nil
}

// readLegacyConfig 读取旧版配置
func (cm *ConfigMigration) readLegacyConfig() (*LegacyConfig, error) {
	data, err := os.ReadFile(cm.sourceFile)
	if err != nil {
		return nil, err
	}

	var config LegacyConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// convertToNewConfig 转换为新版配置
func (cm *ConfigMigration) convertToNewConfig(legacy *LegacyConfig) *NewConfig {
	newConfig := &NewConfig{
		Global: GlobalConfig{
			App: AppConfig{
				Name:        "Kennel Agent",
				Version:     "2.0.0",
				Description: "企业级终端安全管理系统",
				Author:      "Kennel Team",
				License:     "MIT",
			},
			Logging: LoggingConfig{
				Level:      legacy.LogLevel,
				Format:     "json",
				Output:     "file",
				File:       legacy.LogFile,
				MaxSize:    100,
				MaxAge:     30,
				MaxBackups: 10,
				Compress:   true,
			},
			System: SystemConfig{
				MaxConcurrency: 10,
				WorkerPoolSize: 5,
				QueueSize:      1000,
				Timeout:        "30s",
			},
		},
		PluginManager: PluginManagerConfig{
			PluginDir: legacy.PluginDir,
			Discovery: map[string]interface{}{
				"scan_interval":    60,
				"auto_load":        true,
				"follow_symlinks":  false,
				"watch_changes":    true,
				"parallel_loading": true,
			},
			Isolation: map[string]interface{}{
				"default_level": "basic",
				"enable_sandbox": true,
				"resource_limits": map[string]interface{}{
					"memory": 268435456, // 256MB
					"cpu":    50,        // 50%
				},
			},
			Lifecycle: map[string]interface{}{
				"startup_timeout":  "30s",
				"shutdown_timeout": "10s",
				"health_check_interval": "30s",
				"restart_policy": "on-failure",
			},
		},
		Comm: CommConfig{
			Enabled:       getCommBool(legacy.Comm, "enabled", false),
			ServerAddress: getCommString(legacy.Comm, "server_address", "localhost"),
			ServerPort:    getCommInt(legacy.Comm, "server_port", 9000),
			Protocol:      getCommString(legacy.Comm, "protocol", "tcp"),
			Timeout:       getCommInt(legacy.Comm, "timeout", 30),
			RetryInterval: getCommInt(legacy.Comm, "retry_interval", 5),
			MaxRetries:    getCommInt(legacy.Comm, "max_retries", 3),
			KeepAlive:     getCommBool(legacy.Comm, "keep_alive", true),
		},
		WebConsole: cm.convertWebConsole(legacy.WebConsole),
		Plugins: PluginsConfig{
			Assets:  cm.convertPluginConfig("assets", legacy.EnableAssets, legacy.Assets),
			Device:  cm.convertPluginConfig("device", legacy.EnableDevice, legacy.Device),
			DLP:     cm.convertPluginConfig("dlp", legacy.EnableDLP, legacy.DLP),
			Control: cm.convertPluginConfig("control", legacy.EnableControl, legacy.Control),
			Audit:   cm.convertPluginConfig("audit", legacy.EnableAudit, legacy.Audit),
		},
	}

	return newConfig
}

// convertWebConsole 转换Web控制台配置
func (cm *ConfigMigration) convertWebConsole(webConsole map[string]interface{}) WebConsoleConfig {
	if webConsole == nil {
		webConsole = make(map[string]interface{})
	}

	return WebConsoleConfig{
		Enabled:        getBool(webConsole, "enabled", true),
		Host:           getString(webConsole, "host", "0.0.0.0"),
		Port:           getInt(webConsole, "port", 8088),
		EnableHTTPS:    getBool(webConsole, "enable_https", false),
		CertFile:       getString(webConsole, "cert_file", ""),
		KeyFile:        getString(webConsole, "key_file", ""),
		EnableAuth:     getBool(webConsole, "enable_auth", false),
		Username:       getString(webConsole, "username", "admin"),
		Password:       getString(webConsole, "password", "admin"),
		StaticDir:      getString(webConsole, "static_dir", "web/dist"),
		LogLevel:       getString(webConsole, "log_level", "info"),
		RateLimit:      getInt(webConsole, "rate_limit", 100),
		EnableCSRF:     getBool(webConsole, "enable_csrf", false),
		APIPrefix:      getString(webConsole, "api_prefix", "/api"),
		SessionTimeout: getString(webConsole, "session_timeout", "24h"),
		AllowOrigins:   getStringSlice(webConsole, "allow_origins", []string{"*"}),
	}
}

// convertPluginConfig 转换插件配置
func (cm *ConfigMigration) convertPluginConfig(name string, enabled bool, settings map[string]interface{}) PluginConfig {
	if settings == nil {
		settings = make(map[string]interface{})
	}

	return PluginConfig{
		Enabled:        enabled,
		Name:           getPluginName(name),
		Version:        "1.0.0",
		Path:           name,
		AutoStart:      true,
		AutoRestart:    true,
		IsolationLevel: "basic",
		Settings:       settings,
	}
}

// writeNewConfig 写入新版配置
func (cm *ConfigMigration) writeNewConfig(config *NewConfig) error {
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(cm.targetFile), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(cm.targetFile, data, 0644)
}

// 辅助函数
func getBool(m map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return defaultValue
}

func getString(m map[string]interface{}, key string, defaultValue string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return defaultValue
}

func getStringSlice(m map[string]interface{}, key string, defaultValue []string) []string {
	if v, ok := m[key].([]interface{}); ok {
		result := make([]string, len(v))
		for i, item := range v {
			if str, ok := item.(string); ok {
				result[i] = str
			}
		}
		return result
	}
	return defaultValue
}

func getCommBool(m map[string]interface{}, key string, defaultValue bool) bool {
	if m == nil {
		return defaultValue
	}
	return getBool(m, key, defaultValue)
}

func getCommString(m map[string]interface{}, key string, defaultValue string) string {
	if m == nil {
		return defaultValue
	}
	return getString(m, key, defaultValue)
}

func getCommInt(m map[string]interface{}, key string, defaultValue int) int {
	if m == nil {
		return defaultValue
	}
	return getInt(m, key, defaultValue)
}

func getPluginName(id string) string {
	names := map[string]string{
		"assets":  "资产管理插件",
		"device":  "设备管理插件",
		"dlp":     "数据防泄漏插件",
		"control": "终端管控插件",
		"audit":   "安全审计插件",
	}
	if name, ok := names[id]; ok {
		return name
	}
	return id
}
