package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lomehong/kennel/pkg/core"
	"github.com/lomehong/kennel/pkg/plugin"
)

// 本示例展示如何在AppFramework中使用插件隔离
func main() {
	// 创建应用程序实例
	app := core.NewApp("config.yaml")

	// 初始化应用程序
	if err := app.Init(); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 插件隔离使用示例 ===")

	// 示例1: 加载和管理插件
	fmt.Println("\n=== 示例1: 加载和管理插件 ===")
	loadAndManagePlugins(app)

	// 示例2: 插件沙箱
	fmt.Println("\n=== 示例2: 插件沙箱 ===")
	pluginSandbox(app)

	// 示例3: 插件隔离级别
	fmt.Println("\n=== 示例3: 插件隔离级别 ===")
	pluginIsolationLevels(app)

	// 示例4: 插件错误处理和恢复
	fmt.Println("\n=== 示例4: 插件错误处理和恢复 ===")
	pluginErrorHandling(app)

	// 停止应用程序
	app.Stop()
	fmt.Println("\n应用程序已停止")
}

// 加载和管理插件
func loadAndManagePlugins(app *core.App) {
	// 创建插件配置
	config1 := &plugin.PluginConfig{
		ID:             "example-plugin-1",
		Name:           "Example Plugin 1",
		Version:        "1.0.0",
		Path:           "example-plugin-1",
		IsolationLevel: plugin.IsolationLevelBasic,
		AutoStart:      true,
		AutoRestart:    true,
		Enabled:        true,
	}

	config2 := &plugin.PluginConfig{
		ID:             "example-plugin-2",
		Name:           "Example Plugin 2",
		Version:        "1.0.0",
		Path:           "example-plugin-2",
		IsolationLevel: plugin.IsolationLevelStrict,
		AutoStart:      false,
		AutoRestart:    false,
		Enabled:        true,
	}

	// 加载插件
	fmt.Println("加载插件1")
	plugin1, err := app.LoadPlugin(config1)
	if err != nil {
		fmt.Printf("加载插件1失败: %v\n", err)
	} else {
		fmt.Printf("加载插件1成功: ID=%s, 名称=%s, 版本=%s\n", plugin1.ID, plugin1.Name, plugin1.Version)
	}

	fmt.Println("加载插件2")
	plugin2, err := app.LoadPlugin(config2)
	if err != nil {
		fmt.Printf("加载插件2失败: %v\n", err)
	} else {
		fmt.Printf("加载插件2成功: ID=%s, 名称=%s, 版本=%s\n", plugin2.ID, plugin2.Name, plugin2.Version)
	}

	// 等待插件1自动启动
	time.Sleep(100 * time.Millisecond)

	// 手动启动插件2
	fmt.Println("启动插件2")
	err = app.StartPlugin("example-plugin-2")
	if err != nil {
		fmt.Printf("启动插件2失败: %v\n", err)
	} else {
		fmt.Println("启动插件2成功")
	}

	// 列出所有插件
	plugins := app.ListPlugins()
	fmt.Printf("插件列表 (共%d个):\n", len(plugins))
	for _, p := range plugins {
		fmt.Printf("- ID=%s, 名称=%s, 状态=%s\n", p.ID, p.Name, p.State)
	}

	// 重启插件
	fmt.Println("重启插件1")
	err = app.RestartPlugin("example-plugin-1")
	if err != nil {
		fmt.Printf("重启插件1失败: %v\n", err)
	} else {
		fmt.Println("重启插件1成功")
	}

	// 停止插件
	fmt.Println("停止插件2")
	err = app.StopPlugin("example-plugin-2")
	if err != nil {
		fmt.Printf("停止插件2失败: %v\n", err)
	} else {
		fmt.Println("停止插件2成功")
	}

	// 卸载插件
	fmt.Println("卸载插件2")
	err = app.UnloadPlugin("example-plugin-2")
	if err != nil {
		fmt.Printf("卸载插件2失败: %v\n", err)
	} else {
		fmt.Println("卸载插件2成功")
	}
}

// 插件沙箱
func pluginSandbox(app *core.App) {
	// 创建插件配置
	config := &plugin.PluginConfig{
		ID:             "sandbox-plugin",
		Name:           "Sandbox Plugin",
		Version:        "1.0.0",
		Path:           "sandbox-plugin",
		IsolationLevel: plugin.IsolationLevelBasic,
		AutoStart:      true,
		Enabled:        true,
	}

	// 加载插件
	fmt.Println("加载沙箱插件")
	_, err := app.LoadPlugin(config)
	if err != nil {
		fmt.Printf("加载沙箱插件失败: %v\n", err)
		return
	}

	// 等待插件启动
	time.Sleep(100 * time.Millisecond)

	// 在插件沙箱中执行函数
	fmt.Println("在沙箱中执行函数")
	err = app.ExecutePluginFunc("sandbox-plugin", func() error {
		fmt.Println("函数正在沙箱中执行")
		time.Sleep(500 * time.Millisecond)
		fmt.Println("函数在沙箱中执行完成")
		return nil
	})

	if err != nil {
		fmt.Printf("函数执行失败: %v\n", err)
	} else {
		fmt.Println("函数执行成功")
	}

	// 在插件沙箱中执行带上下文的函数
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fmt.Println("在沙箱中执行带上下文的函数")
	err = app.ExecutePluginFuncWithContext("sandbox-plugin", ctx, func(ctx context.Context) error {
		fmt.Println("带上下文的函数正在沙箱中执行")

		select {
		case <-time.After(500 * time.Millisecond):
			fmt.Println("带上下文的函数在沙箱中执行完成")
			return nil
		case <-ctx.Done():
			fmt.Printf("带上下文的函数被取消: %v\n", ctx.Err())
			return ctx.Err()
		}
	})

	if err != nil {
		fmt.Printf("带上下文的函数执行失败: %v\n", err)
	} else {
		fmt.Println("带上下文的函数执行成功")
	}
}

// 插件隔离级别
func pluginIsolationLevels(app *core.App) {
	// 创建不同隔离级别的插件配置
	configNone := &plugin.PluginConfig{
		ID:             "isolation-none",
		Name:           "Isolation None",
		Version:        "1.0.0",
		Path:           "isolation-none",
		IsolationLevel: plugin.IsolationLevelNone,
		AutoStart:      true,
		Enabled:        true,
	}

	configBasic := &plugin.PluginConfig{
		ID:             "isolation-basic",
		Name:           "Isolation Basic",
		Version:        "1.0.0",
		Path:           "isolation-basic",
		IsolationLevel: plugin.IsolationLevelBasic,
		AutoStart:      true,
		Enabled:        true,
	}

	configStrict := &plugin.PluginConfig{
		ID:             "isolation-strict",
		Name:           "Isolation Strict",
		Version:        "1.0.0",
		Path:           "isolation-strict",
		IsolationLevel: plugin.IsolationLevelStrict,
		AutoStart:      true,
		Enabled:        true,
	}

	// 加载插件
	fmt.Println("加载不同隔离级别的插件")
	_, err1 := app.LoadPlugin(configNone)
	_, err2 := app.LoadPlugin(configBasic)
	_, err3 := app.LoadPlugin(configStrict)

	if err1 != nil || err2 != nil || err3 != nil {
		fmt.Println("加载插件失败")
		return
	}

	// 等待插件启动
	time.Sleep(100 * time.Millisecond)

	// 在不同隔离级别的插件中执行函数
	fmt.Println("在不同隔离级别的插件中执行函数")

	// 无隔离
	fmt.Println("无隔离:")
	app.ExecutePluginFunc("isolation-none", func() error {
		fmt.Println("函数在无隔离环境中执行")
		return nil
	})

	// 基本隔离
	fmt.Println("基本隔离:")
	app.ExecutePluginFunc("isolation-basic", func() error {
		fmt.Println("函数在基本隔离环境中执行")
		return nil
	})

	// 严格隔离
	fmt.Println("严格隔离:")
	app.ExecutePluginFunc("isolation-strict", func() error {
		fmt.Println("函数在严格隔离环境中执行")
		return nil
	})
}

// 插件错误处理和恢复
func pluginErrorHandling(app *core.App) {
	// 创建插件配置
	config := &plugin.PluginConfig{
		ID:             "error-plugin",
		Name:           "Error Plugin",
		Version:        "1.0.0",
		Path:           "error-plugin",
		IsolationLevel: plugin.IsolationLevelBasic,
		AutoStart:      true,
		Enabled:        true,
	}

	// 加载插件
	fmt.Println("加载错误处理插件")
	_, err := app.LoadPlugin(config)
	if err != nil {
		fmt.Printf("加载错误处理插件失败: %v\n", err)
		return
	}

	// 等待插件启动
	time.Sleep(100 * time.Millisecond)

	// 执行返回错误的函数
	fmt.Println("执行返回错误的函数")
	err = app.ExecutePluginFunc("error-plugin", func() error {
		fmt.Println("函数正在执行")
		return fmt.Errorf("示例错误")
	})

	if err != nil {
		fmt.Printf("函数返回错误: %v\n", err)
	}

	// 执行会panic的函数
	fmt.Println("执行会panic的函数")
	err = app.ExecutePluginFunc("error-plugin", func() error {
		fmt.Println("函数正在执行")
		panic("示例panic")
		return nil
	})

	if err != nil {
		fmt.Printf("函数panic被恢复: %v\n", err)
	}

	// 执行超时函数
	fmt.Println("执行超时函数")
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = app.ExecutePluginFuncWithContext("error-plugin", ctx, func(ctx context.Context) error {
		fmt.Println("函数正在执行")
		time.Sleep(1 * time.Second) // 超过超时时间
		fmt.Println("函数执行完成")
		return nil
	})

	if err != nil {
		fmt.Printf("函数超时: %v\n", err)
	}
}
