package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/app/control/pkg/control"
	"github.com/lomehong/kennel/pkg/plugin/sdk"
)

func main() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "control-plugin",
		Level:  hclog.LevelFromString("info"),
		Output: os.Stdout,
	})

	logger.Info("启动终端管控插件V2")

	// 创建终端管控模块
	module := control.NewControlModuleV2(logger)

	// 创建插件运行器配置
	config := sdk.DefaultRunnerConfig()
	config.PluginID = module.GetInfo().ID
	config.LogLevel = "info"
	config.LogFile = "logs/control-plugin.log"
	config.ShutdownTimeout = 30 * time.Second
	config.HealthCheckInterval = 30 * time.Second

	// 运行插件
	logger.Info("运行插件", "id", module.GetInfo().ID)
	if err := sdk.Run(module, config); err != nil {
		fmt.Fprintf(os.Stderr, "运行插件失败: %v\n", err)
		os.Exit(1)
	}
}

// 使用构建器模式创建插件的示例
func createPluginWithBuilder() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "control-plugin",
		Level:  hclog.LevelFromString("info"),
		Output: os.Stdout,
	})

	// 使用构建器创建插件
	plugin := sdk.NewPluginBuilder("control").
		WithName("终端管控插件").
		WithVersion("2.0.0").
		WithDescription("终端管控模块，用于远程执行命令、管理进程和安装软件").
		WithAuthor("Kennel Team").
		WithLicense("MIT").
		WithTag("control").
		WithTag("terminal").
		WithTag("process").
		WithCapability("process_management").
		WithCapability("command_execution").
		WithCapability("ai_assistant").
		WithLogger(logger).
		WithInitFunc(func(ctx context.Context, config sdk.PluginConfig) error {
			logger.Info("初始化插件")
			return nil
		}).
		WithStartFunc(func(ctx context.Context) error {
			logger.Info("启动插件")
			return nil
		}).
		WithStopFunc(func(ctx context.Context) error {
			logger.Info("停止插件")
			return nil
		}).
		WithHealthCheckFunc(func(ctx context.Context) (sdk.HealthStatus, error) {
			return sdk.HealthStatus{
				Status:      "healthy",
				Details:     make(map[string]interface{}),
				LastChecked: time.Now(),
			}, nil
		}).
		Build()

	// 创建插件运行器配置
	config := sdk.DefaultRunnerConfig()
	config.PluginID = "control"
	config.LogLevel = "info"
	config.LogFile = "logs/control-plugin.log"
	config.ShutdownTimeout = 30 * time.Second
	config.HealthCheckInterval = 30 * time.Second

	// 运行插件
	if err := sdk.Run(plugin, config); err != nil {
		fmt.Fprintf(os.Stderr, "运行插件失败: %v\n", err)
		os.Exit(1)
	}
}
