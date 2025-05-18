package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lomehong/kennel/pkg/core"
	"github.com/lomehong/kennel/pkg/plugin"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	app     *core.App
	rootCmd = &cobra.Command{
		Use:   "agent",
		Short: "跨平台终端代理框架",
		Long: `一个基于Go的跨平台（Windows + macOS）终端代理框架，
具有模块化、可插拔的特性，包括终端资产管理、设备管理、
数据防泄漏（DLP）、终端管控等功能，统一在一个CLI应用程序下。`,
	}
)

// 初始化函数，设置cobra命令
func init() {
	// 全局标志
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "配置文件路径")

	// 添加子命令
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(pluginCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
}

// version命令
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("AppFramework v0.1.0")
	},
}

// plugin命令及子命令
var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "插件管理",
}

// 初始化plugin子命令
func init() {
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginLoadCmd)
}

// plugin list命令
var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有已加载的插件",
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化应用程序
		if app == nil {
			app = core.NewApp(cfgFile)
		}

		if err := app.Init(); err != nil {
			fmt.Printf("初始化应用程序失败: %v\n", err)
			os.Exit(1)
		}

		// 获取插件列表
		plugins := app.GetPluginManager().ListPlugins()

		if len(plugins) == 0 {
			fmt.Println("没有已加载的插件")
			return
		}

		fmt.Println("已加载的插件:")
		for _, p := range plugins {
			fmt.Printf("- %s (v%s)\n", p.Name, p.Version)
		}
	},
}

// plugin load命令
var pluginLoadCmd = &cobra.Command{
	Use:   "load [plugin_path]",
	Short: "加载插件",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化应用程序
		if app == nil {
			app = core.NewApp(cfgFile)
		}

		if err := app.Init(); err != nil {
			fmt.Printf("初始化应用程序失败: %v\n", err)
			os.Exit(1)
		}

		// 创建插件配置
		config := &plugin.PluginConfig{
			ID:      filepath.Base(args[0]),
			Name:    filepath.Base(args[0]),
			Version: "1.0.0",
			Path:    args[0],
			Enabled: true,
		}

		// 加载插件
		_, err := app.GetPluginManager().LoadPlugin(config)
		if err != nil {
			fmt.Printf("加载插件失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("插件 %s 加载成功\n", args[0])
	},
}

// start命令
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动代理",
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化应用程序
		if app == nil {
			app = core.NewApp(cfgFile)
		}

		if err := app.Init(); err != nil {
			fmt.Printf("初始化应用程序失败: %v\n", err)
			os.Exit(1)
		}

		// 设置信号处理
		setupSignalHandlers()

		// 启动应用程序
		if err := app.Start(); err != nil {
			fmt.Printf("启动应用程序失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("应用程序已启动，按 Ctrl+C 优雅终止")

		// 等待应用程序停止
		for app.IsRunning() {
			time.Sleep(500 * time.Millisecond)
		}

		fmt.Println("应用程序已停止")
	},
}

// setupSignalHandlers 设置信号处理器
func setupSignalHandlers() {
	// 创建一个通道，用于接收信号
	sigCh := make(chan os.Signal, 1)

	// 注册要处理的信号
	// SIGINT: Ctrl+C
	// SIGTERM: 终止信号，通常由系统发送
	// SIGHUP: 终端关闭时发送
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// 在后台处理信号
	go func() {
		sig := <-sigCh
		fmt.Printf("\n收到信号 %v，开始优雅终止...\n", sig)

		// 停止应用程序
		app.Stop()
	}()
}

// stop命令
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止代理",
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化应用程序
		if app == nil {
			app = core.NewApp(cfgFile)
		}

		if err := app.Init(); err != nil {
			fmt.Printf("初始化应用程序失败: %v\n", err)
			os.Exit(1)
		}

		// 停止应用程序
		app.Stop()
		fmt.Println("代理已停止")
	},
}

func main() {
	// 创建应用程序实例
	app = core.NewApp(cfgFile)

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
