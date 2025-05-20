package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"gopkg.in/yaml.v3"
)

// LoadConfig 从文件加载配置
// path: 配置文件路径
// config: 配置对象指针
// 返回: 错误
func LoadConfig(path string, config interface{}) error {
	// 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 根据文件扩展名选择解析方式
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		// 解析JSON
		if err := json.Unmarshal(data, config); err != nil {
			return fmt.Errorf("解析JSON配置失败: %w", err)
		}
	case ".yaml", ".yml":
		// 解析YAML
		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("解析YAML配置失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的配置文件格式: %s", ext)
	}

	return nil
}

// SaveConfig 保存配置到文件
// path: 配置文件路径
// config: 配置对象
// 返回: 错误
func SaveConfig(path string, config interface{}) error {
	// 创建目录
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 根据文件扩展名选择序列化方式
	ext := strings.ToLower(filepath.Ext(path))
	var data []byte
	var err error

	switch ext {
	case ".json":
		// 序列化为JSON
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("序列化JSON配置失败: %w", err)
		}
	case ".yaml", ".yml":
		// 序列化为YAML
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("序列化YAML配置失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的配置文件格式: %s", ext)
	}

	// 写入文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// MergeConfig 合并配置
// target: 目标配置对象指针
// source: 源配置对象
// 返回: 错误
func MergeConfig(target, source interface{}) error {
	// 获取目标和源的反射值
	targetValue := reflect.ValueOf(target)
	sourceValue := reflect.ValueOf(source)

	// 检查目标是否为指针
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("目标必须为指针")
	}

	// 获取目标的元素值
	targetElem := targetValue.Elem()

	// 检查目标和源是否为结构体
	if targetElem.Kind() != reflect.Struct || sourceValue.Kind() != reflect.Struct {
		return fmt.Errorf("目标和源必须为结构体")
	}

	// 获取源的类型
	sourceType := sourceValue.Type()

	// 遍历源的字段
	for i := 0; i < sourceValue.NumField(); i++ {
		sourceField := sourceValue.Field(i)
		sourceFieldType := sourceType.Field(i)

		// 查找目标中对应的字段
		targetField := targetElem.FieldByName(sourceFieldType.Name)
		if !targetField.IsValid() {
			continue
		}

		// 检查字段是否可设置
		if !targetField.CanSet() {
			continue
		}

		// 检查类型是否兼容
		if !sourceField.Type().AssignableTo(targetField.Type()) {
			continue
		}

		// 设置字段值
		targetField.Set(sourceField)
	}

	return nil
}

// CreateLogger 创建日志记录器
// name: 日志记录器名称
// level: 日志级别
// output: 日志输出
// 返回: 日志记录器
func CreateLogger(name, level string, output string) (hclog.Logger, error) {
	// 设置日志级别
	logLevel := hclog.LevelFromString(level)
	if logLevel == hclog.NoLevel {
		logLevel = hclog.Info
	}

	// 设置日志输出
	var logOutput *os.File
	var err error

	if output == "" || output == "stdout" {
		logOutput = os.Stdout
	} else if output == "stderr" {
		logOutput = os.Stderr
	} else {
		// 创建目录
		dir := filepath.Dir(output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败: %w", err)
		}

		// 打开日志文件
		logOutput, err = os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("打开日志文件失败: %w", err)
		}
	}

	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   name,
		Level:  logLevel,
		Output: logOutput,
	})

	return logger, nil
}

// WaitForSignal 等待信号
// ctx: 上下文
// signals: 信号通道
// 返回: 收到的信号
func WaitForSignal(ctx context.Context, signals <-chan os.Signal) os.Signal {
	select {
	case <-ctx.Done():
		return nil
	case sig := <-signals:
		return sig
	}
}

// RetryWithBackoff 带退避的重试
// ctx: 上下文
// fn: 要重试的函数
// maxRetries: 最大重试次数
// initialBackoff: 初始退避时间
// maxBackoff: 最大退避时间
// 返回: 错误
func RetryWithBackoff(ctx context.Context, fn func() error, maxRetries int, initialBackoff, maxBackoff time.Duration) error {
	var err error
	backoff := initialBackoff

	for i := 0; i < maxRetries; i++ {
		// 执行函数
		err = fn()
		if err == nil {
			return nil
		}

		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 继续重试
		}

		// 等待退避时间
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// 继续重试
		}

		// 增加退避时间
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	return fmt.Errorf("达到最大重试次数: %w", err)
}

// ParsePluginConfig 解析插件配置
// configData: 配置数据
// 返回: 插件配置和错误
func ParsePluginConfig(configData map[string]interface{}) (api.PluginConfig, error) {
	config := api.PluginConfig{
		Settings: make(map[string]interface{}),
	}

	// 解析基本字段
	if id, ok := configData["id"].(string); ok {
		config.ID = id
	} else {
		return config, fmt.Errorf("缺少插件ID")
	}

	if enabled, ok := configData["enabled"].(bool); ok {
		config.Enabled = enabled
	} else {
		config.Enabled = true
	}

	if logLevel, ok := configData["log_level"].(string); ok {
		config.LogLevel = logLevel
	} else {
		config.LogLevel = "info"
	}

	// 解析设置
	if settings, ok := configData["settings"].(map[string]interface{}); ok {
		for k, v := range settings {
			config.Settings[k] = v
		}
	}

	return config, nil
}

// FormatPluginError 格式化插件错误
// pluginID: 插件ID
// message: 错误消息
// err: 原始错误
// 返回: 格式化的错误
func FormatPluginError(pluginID, message string, err error) error {
	if err == nil {
		return fmt.Errorf("插件 %s: %s", pluginID, message)
	}
	return fmt.Errorf("插件 %s: %s: %w", pluginID, message, err)
}

// ValidatePluginInfo 验证插件信息
// info: 插件信息
// 返回: 错误
func ValidatePluginInfo(info api.PluginInfo) error {
	if info.ID == "" {
		return fmt.Errorf("插件ID不能为空")
	}

	if info.Name == "" {
		return fmt.Errorf("插件名称不能为空")
	}

	if info.Version == "" {
		return fmt.Errorf("插件版本不能为空")
	}

	return nil
}

// CreatePluginInfoFromConfig 从配置创建插件信息
// config: 配置数据
// 返回: 插件信息和错误
func CreatePluginInfoFromConfig(config map[string]interface{}) (api.PluginInfo, error) {
	info := api.PluginInfo{
		Capabilities: make(map[string]bool),
	}

	// 解析基本字段
	if id, ok := config["id"].(string); ok {
		info.ID = id
	} else {
		return info, fmt.Errorf("缺少插件ID")
	}

	if name, ok := config["name"].(string); ok {
		info.Name = name
	} else {
		info.Name = info.ID
	}

	if version, ok := config["version"].(string); ok {
		info.Version = version
	} else {
		return info, fmt.Errorf("缺少插件版本")
	}

	if description, ok := config["description"].(string); ok {
		info.Description = description
	}

	if author, ok := config["author"].(string); ok {
		info.Author = author
	}

	if license, ok := config["license"].(string); ok {
		info.License = license
	}

	// 解析标签
	if tags, ok := config["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				info.Tags = append(info.Tags, tagStr)
			}
		}
	}

	// 解析能力
	if capabilities, ok := config["capabilities"].(map[string]interface{}); ok {
		for k, v := range capabilities {
			if enabled, ok := v.(bool); ok {
				info.Capabilities[k] = enabled
			}
		}
	}

	// 解析依赖
	if dependencies, ok := config["dependencies"].([]interface{}); ok {
		for _, dep := range dependencies {
			if depMap, ok := dep.(map[string]interface{}); ok {
				dependency := api.PluginDependency{}

				if id, ok := depMap["id"].(string); ok {
					dependency.ID = id
				} else {
					continue
				}

				if version, ok := depMap["version"].(string); ok {
					dependency.Version = version
				}

				if optional, ok := depMap["optional"].(bool); ok {
					dependency.Optional = optional
				}

				info.Dependencies = append(info.Dependencies, dependency)
			}
		}
	}

	return info, nil
}

// GetPluginDataDir 获取插件数据目录
// pluginID: 插件ID
// 返回: 数据目录路径和错误
func GetPluginDataDir(pluginID string) (string, error) {
	// 获取用户数据目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	// 创建插件数据目录
	dataDir := filepath.Join(homeDir, ".kennel", "plugins", pluginID, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("创建数据目录失败: %w", err)
	}

	return dataDir, nil
}

// GetPluginConfigDir 获取插件配置目录
// pluginID: 插件ID
// 返回: 配置目录路径和错误
func GetPluginConfigDir(pluginID string) (string, error) {
	// 获取用户配置目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	// 创建插件配置目录
	configDir := filepath.Join(homeDir, ".kennel", "plugins", pluginID, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("创建配置目录失败: %w", err)
	}

	return configDir, nil
}

// GetPluginLogDir 获取插件日志目录
// pluginID: 插件ID
// 返回: 日志目录路径和错误
func GetPluginLogDir(pluginID string) (string, error) {
	// 获取用户日志目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}

	// 创建插件日志目录
	logDir := filepath.Join(homeDir, ".kennel", "plugins", pluginID, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("创建日志目录失败: %w", err)
	}

	return logDir, nil
}
