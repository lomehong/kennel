package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

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

// ConfigVersion 配置版本
type ConfigVersion struct {
	Version   int       // 版本号
	Timestamp time.Time // 时间戳
	Comment   string    // 注释
}

// ConfigValidator 配置验证器
type ConfigValidator func(config map[string]interface{}) error

// ConfigChangeListener 配置变更监听器
type ConfigChangeListener func(oldConfig, newConfig map[string]interface{}) error

// DynamicConfig 动态配置
type DynamicConfig struct {
	path           string                   // 配置文件路径
	format         ConfigFormat             // 配置格式
	config         map[string]interface{}   // 当前配置
	history        []map[string]interface{} // 配置历史
	versions       []ConfigVersion          // 版本历史
	maxHistorySize int                      // 最大历史记录大小
	validators     []ConfigValidator        // 验证器
	listeners      []ConfigChangeListener   // 监听器
	watcher        *ConfigWatcher           // 配置监视器
	logger         hclog.Logger             // 日志记录器
	mu             sync.RWMutex             // 互斥锁
}

// DynamicConfigOption 动态配置选项
type DynamicConfigOption func(*DynamicConfig)

// WithFormat 设置配置格式
func WithFormat(format ConfigFormat) DynamicConfigOption {
	return func(dc *DynamicConfig) {
		dc.format = format
	}
}

// WithMaxHistorySize 设置最大历史记录大小
func WithMaxHistorySize(size int) DynamicConfigOption {
	return func(dc *DynamicConfig) {
		dc.maxHistorySize = size
	}
}

// WithValidator 添加验证器
func WithValidator(validator ConfigValidator) DynamicConfigOption {
	return func(dc *DynamicConfig) {
		dc.validators = append(dc.validators, validator)
	}
}

// WithListener 添加监听器
func WithListener(listener ConfigChangeListener) DynamicConfigOption {
	return func(dc *DynamicConfig) {
		dc.listeners = append(dc.listeners, listener)
	}
}

// WithWatcher 设置配置监视器
func WithWatcher(watcher *ConfigWatcher) DynamicConfigOption {
	return func(dc *DynamicConfig) {
		dc.watcher = watcher
	}
}

// NewDynamicConfig 创建一个新的动态配置
func NewDynamicConfig(path string, logger hclog.Logger, options ...DynamicConfigOption) *DynamicConfig {
	// 确定配置格式
	format := ConfigFormatYAML
	ext := filepath.Ext(path)
	if ext == ".json" {
		format = ConfigFormatJSON
	}

	dc := &DynamicConfig{
		path:           path,
		format:         format,
		config:         make(map[string]interface{}),
		history:        make([]map[string]interface{}, 0),
		versions:       make([]ConfigVersion, 0),
		maxHistorySize: 10,
		validators:     make([]ConfigValidator, 0),
		listeners:      make([]ConfigChangeListener, 0),
		logger:         logger,
	}

	// 应用选项
	for _, option := range options {
		option(dc)
	}

	// 加载配置
	if err := dc.Load(); err != nil {
		logger.Error("加载配置失败", "error", err)
	}

	// 如果有监视器，添加处理器
	if dc.watcher != nil {
		dc.watcher.AddPath(path)
		dc.watcher.AddHandler(path, func(event ChangeEvent) error {
			if event.Type == ChangeTypeUpdate || event.Type == ChangeTypeCreate {
				return dc.Reload()
			}
			return nil
		})
	}

	return dc
}

// Load 加载配置
func (dc *DynamicConfig) Load() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(dc.path); os.IsNotExist(err) {
		dc.logger.Warn("配置文件不存在", "path", dc.path)
		return nil
	}

	// 读取文件
	data, err := ioutil.ReadFile(dc.path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	var config map[string]interface{}
	switch dc.format {
	case ConfigFormatYAML:
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("解析YAML配置失败: %w", err)
		}
	case ConfigFormatJSON:
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("解析JSON配置失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的配置格式: %s", dc.format)
	}

	// 验证配置
	for _, validator := range dc.validators {
		if err := validator(config); err != nil {
			return fmt.Errorf("配置验证失败: %w", err)
		}
	}

	// 保存旧配置
	oldConfig := dc.config

	// 更新配置
	dc.config = config

	// 添加到历史记录
	dc.addToHistory(oldConfig)

	dc.logger.Info("加载配置成功", "path", dc.path)
	return nil
}

// Reload 重新加载配置
func (dc *DynamicConfig) Reload() error {
	dc.mu.Lock()
	oldConfig := dc.config
	dc.mu.Unlock()

	// 加载配置
	if err := dc.Load(); err != nil {
		return err
	}

	// 通知监听器
	dc.mu.RLock()
	newConfig := dc.config
	listeners := dc.listeners
	dc.mu.RUnlock()

	for _, listener := range listeners {
		if err := listener(oldConfig, newConfig); err != nil {
			dc.logger.Error("配置变更监听器失败", "error", err)
		}
	}

	return nil
}

// Save 保存配置
func (dc *DynamicConfig) Save() error {
	dc.mu.RLock()
	config := dc.config
	dc.mu.RUnlock()

	// 验证配置
	for _, validator := range dc.validators {
		if err := validator(config); err != nil {
			return fmt.Errorf("配置验证失败: %w", err)
		}
	}

	// 序列化配置
	var data []byte
	var err error
	switch dc.format {
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
		return fmt.Errorf("不支持的配置格式: %s", dc.format)
	}

	// 确保目录存在
	dir := filepath.Dir(dc.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	if err := ioutil.WriteFile(dc.path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	dc.logger.Info("保存配置成功", "path", dc.path)
	return nil
}

// Get 获取配置值
func (dc *DynamicConfig) Get(key string) interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.config[key]
}

// GetAll 获取所有配置
func (dc *DynamicConfig) GetAll() map[string]interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	config := make(map[string]interface{})
	for k, v := range dc.config {
		config[k] = v
	}
	return config
}

// Set 设置配置值
func (dc *DynamicConfig) Set(key string, value interface{}) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.config[key] = value
}

// SetAll 设置所有配置
func (dc *DynamicConfig) SetAll(config map[string]interface{}) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.config = config
}

// Update 更新配置
func (dc *DynamicConfig) Update(updates map[string]interface{}) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	for k, v := range updates {
		dc.config[k] = v
	}
}

// Delete 删除配置
func (dc *DynamicConfig) Delete(key string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	delete(dc.config, key)
}

// GetVersion 获取当前版本
func (dc *DynamicConfig) GetVersion() ConfigVersion {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	if len(dc.versions) == 0 {
		return ConfigVersion{
			Version:   0,
			Timestamp: time.Now(),
			Comment:   "初始版本",
		}
	}
	return dc.versions[len(dc.versions)-1]
}

// GetVersions 获取所有版本
func (dc *DynamicConfig) GetVersions() []ConfigVersion {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	versions := make([]ConfigVersion, len(dc.versions))
	copy(versions, dc.versions)
	return versions
}

// Rollback 回滚到指定版本
func (dc *DynamicConfig) Rollback(version int) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// 查找版本
	index := -1
	for i, v := range dc.versions {
		if v.Version == version {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("版本不存在: %d", version)
	}

	// 回滚配置
	oldConfig := dc.config
	dc.config = dc.history[index]

	// 添加到历史记录
	dc.addToHistory(oldConfig)

	dc.logger.Info("回滚配置成功", "version", version)
	return nil
}

// AddValidator 添加验证器
func (dc *DynamicConfig) AddValidator(validator ConfigValidator) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.validators = append(dc.validators, validator)
}

// AddListener 添加监听器
func (dc *DynamicConfig) AddListener(listener ConfigChangeListener) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.listeners = append(dc.listeners, listener)
}

// addToHistory 添加到历史记录
func (dc *DynamicConfig) addToHistory(config map[string]interface{}) {
	// 复制配置
	configCopy := make(map[string]interface{})
	for k, v := range config {
		configCopy[k] = v
	}

	// 添加到历史记录
	dc.history = append(dc.history, configCopy)
	if len(dc.history) > dc.maxHistorySize {
		dc.history = dc.history[1:]
	}

	// 添加版本信息
	version := 1
	if len(dc.versions) > 0 {
		version = dc.versions[len(dc.versions)-1].Version + 1
	}
	dc.versions = append(dc.versions, ConfigVersion{
		Version:   version,
		Timestamp: time.Now(),
		Comment:   fmt.Sprintf("版本 %d", version),
	})
	if len(dc.versions) > dc.maxHistorySize {
		dc.versions = dc.versions[1:]
	}
}
