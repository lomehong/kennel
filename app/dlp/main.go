package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lomehong/kennel/pkg/core/plugin"
	"github.com/lomehong/kennel/pkg/logging"
	"gopkg.in/yaml.v2"
)

// loadConfigFromFile 从配置文件加载配置
func loadConfigFromFile(configPath string) (map[string]interface{}, error) {
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML配置
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return config, nil
}

// mergeConfigs 合并配置，fileConfig优先级高于defaultConfig
func mergeConfigs(fileConfig, defaultConfig map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 先复制默认配置
	for k, v := range defaultConfig {
		result[k] = v
	}

	// 再复制文件配置，覆盖默认配置
	for k, v := range fileConfig {
		result[k] = v
	}

	return result
}

func main() {
	// 设置环境变量，确保插件使用正确的 Magic Cookie
	os.Setenv("PLUGIN_MAGIC_COOKIE", "kennel")

	// 创建日志记录器
	logConfig := logging.DefaultLogConfig()
	logConfig.Level = logging.LogLevelInfo
	logger, err := logging.NewEnhancedLogger(logConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建日志记录器失败: %v\n", err)
		os.Exit(1)
	}

	// 创建数据防泄漏模块
	module := NewDLPModule(logger.Named("dlp"))

	// 尝试从配置文件加载配置 - 优先使用当前目录的配置文件
	configPaths := []string{
		"config.yaml",       // 当前目录
		"../config.yaml",    // 上级目录
		"../../config.yaml", // 根目录
	}

	var fileConfig map[string]interface{}
	var configPath string
	var configErr error

	for _, path := range configPaths {
		logger.Info("尝试加载配置文件", "path", path)
		fileConfig, configErr = loadConfigFromFile(path)
		if configErr == nil {
			configPath = path
			logger.Info("已加载配置文件", "config_path", configPath, "keys_count", len(fileConfig))

			// 调试：打印配置键
			keys := make([]string, 0, len(fileConfig))
			for k := range fileConfig {
				keys = append(keys, k)
			}
			logger.Info("配置文件包含的键", "keys", keys)

			// 检查OCR配置
			if ocrConfig, ok := fileConfig["ocr"]; ok {
				logger.Info("找到OCR配置", "ocr_config", ocrConfig)
			} else {
				logger.Warn("配置文件中未找到OCR配置")
			}
			break
		} else {
			logger.Warn("加载配置文件失败", "path", path, "error", configErr)
		}
	}

	if configErr != nil {
		logger.Warn("所有配置文件加载失败，使用默认配置", "error", configErr, "tried_paths", configPaths)
		fileConfig = make(map[string]interface{})
	}

	// 提取DLP配置段
	var dlpConfig map[string]interface{}
	if dlpValue, exists := fileConfig["dlp"]; exists {
		logger.Info("找到DLP配置值", "type", fmt.Sprintf("%T", dlpValue))

		// 处理 map[interface{}]interface{} 类型
		if dlpMap, ok := dlpValue.(map[interface{}]interface{}); ok {
			dlpConfig = make(map[string]interface{})
			for k, v := range dlpMap {
				if keyStr, ok := k.(string); ok {
					dlpConfig[keyStr] = v
				}
			}
			// 获取配置键列表
			keys := make([]string, 0, len(dlpConfig))
			for k := range dlpConfig {
				keys = append(keys, k)
			}
			logger.Info("找到DLP配置段", "keys", keys)
		} else if dlpSection, ok := dlpValue.(map[string]interface{}); ok {
			dlpConfig = dlpSection
			// 获取配置键列表
			keys := make([]string, 0, len(dlpConfig))
			for k := range dlpConfig {
				keys = append(keys, k)
			}
			logger.Info("找到DLP配置段", "keys", keys)
		} else {
			logger.Warn("DLP配置段类型转换失败", "type", fmt.Sprintf("%T", dlpValue))
			dlpConfig = make(map[string]interface{})
		}
	} else {
		logger.Warn("配置文件中未找到DLP配置段，使用默认配置")
		dlpConfig = make(map[string]interface{})
	}

	// 合并DLP配置和默认配置
	defaultDLPConfig := map[string]interface{}{
		"log_level":         "info",
		"monitor_clipboard": true,
		"monitor_files":     true,
		"monitor_network":   true,
		"monitored_directories": []string{
			"data/dlp/monitored",
		},
		"monitored_file_types": []string{
			"*.txt", "*.doc", "*.docx", "*.xls", "*.xlsx", "*.pdf",
		},
		"network_protocols": []string{
			"http", "https", "ftp", "smtp",
		},
	}

	// 创建配置，使用DLP配置段
	config := &plugin.ModuleConfig{
		Settings: mergeConfigs(dlpConfig, defaultDLPConfig),
	}

	// 初始化模块
	if err := module.Init(context.Background(), config); err != nil {
		fmt.Fprintf(os.Stderr, "初始化模块失败: %v\n", err)
		os.Exit(1)
	}

	// 启动模块
	if err := module.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动模块失败: %v\n", err)
		os.Exit(1)
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("DLP插件已启动，按Ctrl+C退出")

	// 等待信号
	sig := <-sigChan
	logger.Info("收到退出信号，正在关闭", "signal", sig.String())

	// 停止模块
	if err := module.Stop(); err != nil {
		logger.Error("停止模块失败", "error", err)
	}

	logger.Info("DLP插件已退出")
	os.Exit(0)
}
