package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lomehong/kennel/pkg/core/plugin"
	"github.com/lomehong/kennel/pkg/logging"
)

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

	// 创建默认配置
	config := &plugin.ModuleConfig{
		Settings: map[string]interface{}{
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
		},
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
