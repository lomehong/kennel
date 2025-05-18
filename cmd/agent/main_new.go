package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/core/config"
	"github.com/lomehong/kennel/pkg/core/plugin"
	"github.com/spf13/cobra"
)

var (
	// 配置文件路径
	configPath string

	// 日志级别
	logLevel string

	// 插件目录
	pluginDir string

	// 版本信息
	version = "1.0.0"
)

func main() {
	// 创建根命令
	rootCmd := &cobra.Command{
		Use:   "agent",
		Short: "Kennel Agent",
		Long:  "Kennel Agent - 跨平台终端代理框架",
		Run: func(cmd *cobra.Command, args []string) {
			// 显示帮助信息
			cmd.Help()
		},
	}

	// 添加全局标志
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yaml", "配置文件路径")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "日志级别")
	rootCmd.PersistentFlags().StringVarP(&pluginDir, "plugin-dir", "p", "plugins", "插件目录")

	// 添加版本命令
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Kennel Agent v%s\n", version)
		},
	}
	rootCmd.AddCommand(versionCmd)

	// 添加启动命令
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "启动代理",
		Run: func(cmd *cobra.Command, args []string) {
			startAgent()
		},
	}
	rootCmd.AddCommand(startCmd)

	// 添加停止命令
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "停止代理",
		Run: func(cmd *cobra.Command, args []string) {
			stopAgent()
		},
	}
	rootCmd.AddCommand(stopCmd)

	// 添加插件命令
	pluginCmd := &cobra.Command{
		Use:   "plugin",
		Short: "插件管理",
	}
	rootCmd.AddCommand(pluginCmd)

	// 添加插件列表命令
	pluginListCmd := &cobra.Command{
		Use:   "list",
		Short: "列出插件",
		Run: func(cmd *cobra.Command, args []string) {
			listPlugins()
		},
	}
	pluginCmd.AddCommand(pluginListCmd)

	// 添加加载插件命令
	pluginLoadCmd := &cobra.Command{
		Use:   "load [plugin-id]",
		Short: "加载插件",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			loadPlugin(args[0])
		},
	}
	pluginCmd.AddCommand(pluginLoadCmd)

	// 添加卸载插件命令
	pluginUnloadCmd := &cobra.Command{
		Use:   "unload [plugin-id]",
		Short: "卸载插件",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			unloadPlugin(args[0])
		},
	}
	pluginCmd.AddCommand(pluginUnloadCmd)

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// startAgent 启动代理
func startAgent() {
	// 创建日志记录器
	logger := createLogger()
	logger.Info("启动代理")

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建配置管理器
	configManager, err := createConfigManager(logger)
	if err != nil {
		logger.Error("创建配置管理器失败", "error", err)
		os.Exit(1)
	}

	// 创建插件管理器
	pluginManager := createPluginManager(ctx, logger)

	// 创建配置集成
	configIntegration := plugin.NewConfigIntegration(configManager, pluginManager)

	// 初始化配置集成
	if err := configIntegration.Initialize(); err != nil {
		logger.Error("初始化配置集成失败", "error", err)
		os.Exit(1)
	}

	// 启动健康检查
	pluginManager.StartHealthCheck()

	// 设置信号处理
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// 等待信号
	<-signalCh

	// 停止插件管理器
	logger.Info("停止代理")
	pluginManager.Stop()

	// 关闭配置管理器
	configManager.Close()
}

// stopAgent 停止代理
func stopAgent() {
	fmt.Println("停止代理")
	// 在实际应用中，这里应该发送信号给正在运行的代理进程
}

// listPlugins 列出插件
func listPlugins() {
	// 创建日志记录器
	logger := createLogger()

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建插件管理器
	pluginManager := createPluginManager(ctx, logger)

	// 扫描插件目录
	metadataList, err := pluginManager.ScanPluginsDir()
	if err != nil {
		logger.Error("扫描插件目录失败", "error", err)
		os.Exit(1)
	}

	// 显示插件列表
	fmt.Println("插件列表:")
	for _, metadata := range metadataList {
		fmt.Printf("- %s (v%s): %s\n", metadata.Name, metadata.Version, metadata.Description)
		fmt.Printf("  ID: %s\n", metadata.ID)
		fmt.Printf("  语言: %s\n", metadata.Language)
		fmt.Printf("  路径: %s\n", metadata.Path)
		fmt.Println()
	}
}

// loadPlugin 加载插件
func loadPlugin(pluginID string) {
	// 创建日志记录器
	logger := createLogger()

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建配置管理器
	configManager, err := createConfigManager(logger)
	if err != nil {
		logger.Error("创建配置管理器失败", "error", err)
		os.Exit(1)
	}

	// 创建插件管理器
	pluginManager := createPluginManager(ctx, logger)

	// 创建配置集成
	configIntegration := plugin.NewConfigIntegration(configManager, pluginManager)

	// 加载插件
	instance, err := configIntegration.LoadPluginFromConfig(pluginID)
	if err != nil {
		logger.Error("加载插件失败", "error", err)
		os.Exit(1)
	}

	fmt.Printf("插件 %s (v%s) 已加载\n", instance.Metadata.Name, instance.Metadata.Version)
}

// unloadPlugin 卸载插件
func unloadPlugin(pluginID string) {
	// 创建日志记录器
	logger := createLogger()

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建插件管理器
	pluginManager := createPluginManager(ctx, logger)

	// 卸载插件
	if err := pluginManager.UnloadPlugin(pluginID); err != nil {
		logger.Error("卸载插件失败", "error", err)
		os.Exit(1)
	}

	fmt.Printf("插件 %s 已卸载\n", pluginID)
}

// createLogger 创建日志记录器
func createLogger() hclog.Logger {
	level := hclog.LevelFromString(logLevel)
	if level == hclog.NoLevel {
		level = hclog.Info
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:   "agent",
		Level:  level,
		Output: os.Stderr,
	})
}

// createConfigManager 创建配置管理器
func createConfigManager(logger hclog.Logger) (*config.ConfigManager, error) {
	// 确保配置文件路径是绝对路径
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("获取配置文件绝对路径失败: %w", err)
	}

	// 创建配置管理器
	configManager, err := config.NewConfigManager(
		config.WithConfigPath(absPath),
		config.WithConfigLogger(logger.Named("config")),
	)
	if err != nil {
		return nil, fmt.Errorf("创建配置管理器失败: %w", err)
	}

	return configManager, nil
}

// createPluginManager 创建插件管理器
func createPluginManager(ctx context.Context, logger hclog.Logger) *plugin.PluginManager {
	// 创建插件管理器
	pluginManager := plugin.NewPluginManager(
		plugin.WithPluginLogger(logger.Named("plugin")),
		plugin.WithPluginsDir(pluginDir),
		plugin.WithPluginContext(ctx),
		plugin.WithHealthCheckInterval(30*time.Second),
	)

	return pluginManager
}
