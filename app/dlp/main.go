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

	// 尝试从配置文件加载配置
	configPath := "config.yaml"
	fileConfig, err := loadConfigFromFile(configPath)
	if err != nil {
		logger.Warn("加载配置文件失败，使用默认配置", "error", err, "config_path", configPath)
		fileConfig = make(map[string]interface{})
	} else {
		logger.Info("已加载配置文件", "config_path", configPath)
	}

	// 创建配置，合并文件配置和默认配置
	config := &plugin.ModuleConfig{
		Settings: mergeConfigs(fileConfig, map[string]interface{}{
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
		}),
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
