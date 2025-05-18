package plugin

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
)

// 以下是插件隔离和沙箱的使用示例

// ExamplePluginIsolation 展示插件隔离的基本用法
func ExamplePluginIsolation() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin-isolation",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建插件隔离配置
	config := DefaultPluginIsolationConfig()
	config.Level = IsolationLevelBasic
	config.TimeoutDuration = 5 * time.Second

	// 创建插件隔离器
	isolator := NewPluginIsolator(config, WithLogger(logger))

	// 执行正常函数
	logger.Info("执行正常函数")
	err := isolator.ExecuteFunc("example-plugin", func() error {
		logger.Info("函数正在执行")
		time.Sleep(1 * time.Second)
		logger.Info("函数执行完成")
		return nil
	})

	if err != nil {
		logger.Error("函数执行失败", "error", err)
	} else {
		logger.Info("函数执行成功")
	}

	// 执行返回错误的函数
	logger.Info("执行返回错误的函数")
	err = isolator.ExecuteFunc("example-plugin", func() error {
		logger.Info("函数正在执行")
		time.Sleep(1 * time.Second)
		logger.Info("函数返回错误")
		return fmt.Errorf("示例错误")
	})

	if err != nil {
		logger.Error("函数执行失败", "error", err)
	} else {
		logger.Info("函数执行成功")
	}

	// 执行超时函数
	logger.Info("执行超时函数")
	err = isolator.ExecuteFunc("example-plugin", func() error {
		logger.Info("函数正在执行")
		time.Sleep(10 * time.Second) // 超过超时时间
		logger.Info("函数执行完成")
		return nil
	})

	if err != nil {
		logger.Error("函数执行失败", "error", err)
	} else {
		logger.Info("函数执行成功")
	}

	// 获取统计信息
	stats := isolator.GetStats()
	logger.Info("隔离器统计信息",
		"total_calls", stats.TotalCalls,
		"successful_calls", stats.SuccessfulCalls,
		"failed_calls", stats.FailedCalls,
		"timeouts", stats.Timeouts,
		"panics", stats.Panics,
		"avg_execution_time", stats.AvgExecutionTime,
	)
}

// ExamplePluginSandbox 展示插件沙箱的用法
func ExamplePluginSandbox() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin-sandbox",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建插件隔离配置
	config := DefaultPluginIsolationConfig()
	config.Level = IsolationLevelBasic
	config.TimeoutDuration = 5 * time.Second

	// 创建插件隔离器
	isolator := NewPluginIsolator(config, WithLogger(logger))

	// 创建插件沙箱
	sandbox := NewPluginSandbox("example-plugin", isolator,
		WithSandboxLogger(logger.Named("sandbox")),
	)

	// 执行函数
	logger.Info("执行函数")
	err := sandbox.Execute(func() error {
		logger.Info("函数正在执行")
		time.Sleep(1 * time.Second)
		logger.Info("函数执行完成")
		return nil
	})

	if err != nil {
		logger.Error("函数执行失败", "error", err)
	} else {
		logger.Info("函数执行成功")
	}

	// 执行带上下文的函数
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	logger.Info("执行带上下文的函数")
	err = sandbox.ExecuteWithContext(ctx, func(ctx context.Context) error {
		logger.Info("函数正在执行")

		select {
		case <-time.After(1 * time.Second):
			logger.Info("函数执行完成")
			return nil
		case <-ctx.Done():
			logger.Warn("函数被取消", "error", ctx.Err())
			return ctx.Err()
		}
	})

	if err != nil {
		logger.Error("函数执行失败", "error", err)
	} else {
		logger.Info("函数执行成功")
	}

	// 获取沙箱状态
	state := sandbox.GetState()
	logger.Info("沙箱状态", "state", state.String())

	// 获取统计信息
	stats := sandbox.GetStats()
	logger.Info("沙箱统计信息", "stats", stats)

	// 暂停沙箱
	logger.Info("暂停沙箱")
	sandbox.Pause()

	// 尝试在暂停状态下执行函数
	logger.Info("在暂停状态下执行函数")
	err = sandbox.Execute(func() error {
		return nil
	})

	if err != nil {
		logger.Error("函数执行失败", "error", err)
	} else {
		logger.Info("函数执行成功")
	}

	// 恢复沙箱
	logger.Info("恢复沙箱")
	sandbox.Resume()

	// 停止沙箱
	logger.Info("停止沙箱")
	sandbox.Stop()
}

// ExamplePluginManager 展示插件管理器的用法
func ExamplePluginManager() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin-manager",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建插件管理器
	manager := NewPluginManager(
		WithPluginManagerLogger(logger),
		WithPluginsDir("./plugins"),
		WithHealthCheckInterval(30*time.Second),
		WithIdleTimeout(5*time.Minute),
	)

	// 创建插件配置
	config1 := &PluginConfig{
		ID:             "example-plugin-1",
		Name:           "Example Plugin 1",
		Version:        "1.0.0",
		Path:           "example-plugin-1",
		IsolationLevel: IsolationLevelBasic,
		AutoStart:      true,
		AutoRestart:    true,
	}

	config2 := &PluginConfig{
		ID:             "example-plugin-2",
		Name:           "Example Plugin 2",
		Version:        "1.0.0",
		Path:           "example-plugin-2",
		IsolationLevel: IsolationLevelStrict,
		AutoStart:      false,
		AutoRestart:    false,
	}

	// 加载插件
	logger.Info("加载插件1")
	plugin1, err := manager.LoadPlugin(config1)
	if err != nil {
		logger.Error("加载插件1失败", "error", err)
	} else {
		logger.Info("加载插件1成功", "id", plugin1.ID, "name", plugin1.Name)
	}

	logger.Info("加载插件2")
	plugin2, err := manager.LoadPlugin(config2)
	if err != nil {
		logger.Error("加载插件2失败", "error", err)
	} else {
		logger.Info("加载插件2成功", "id", plugin2.ID, "name", plugin2.Name)
	}

	// 等待插件1自动启动
	time.Sleep(100 * time.Millisecond)

	// 手动启动插件2
	logger.Info("启动插件2")
	err = manager.StartPlugin("example-plugin-2")
	if err != nil {
		logger.Error("启动插件2失败", "error", err)
	} else {
		logger.Info("启动插件2成功")
	}

	// 列出所有插件
	plugins := manager.ListPlugins()
	logger.Info("插件列表", "count", len(plugins))
	for _, p := range plugins {
		logger.Info("插件信息", "id", p.ID, "name", p.Name, "state", p.State)
	}

	// 在插件中执行函数
	logger.Info("在插件1中执行函数")
	err = manager.ExecutePluginFunc("example-plugin-1", func() error {
		logger.Info("函数正在执行")
		time.Sleep(1 * time.Second)
		logger.Info("函数执行完成")
		return nil
	})

	if err != nil {
		logger.Error("函数执行失败", "error", err)
	} else {
		logger.Info("函数执行成功")
	}

	// 在插件中执行带上下文的函数
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	logger.Info("在插件2中执行带上下文的函数")
	err = manager.ExecutePluginFuncWithContext("example-plugin-2", ctx, func(ctx context.Context) error {
		logger.Info("函数正在执行")

		select {
		case <-time.After(1 * time.Second):
			logger.Info("函数执行完成")
			return nil
		case <-ctx.Done():
			logger.Warn("函数被取消", "error", ctx.Err())
			return ctx.Err()
		}
	})

	if err != nil {
		logger.Error("函数执行失败", "error", err)
	} else {
		logger.Info("函数执行成功")
	}

	// 重启插件
	logger.Info("重启插件1")
	err = manager.RestartPlugin("example-plugin-1")
	if err != nil {
		logger.Error("重启插件1失败", "error", err)
	} else {
		logger.Info("重启插件1成功")
	}

	// 停止插件
	logger.Info("停止插件2")
	err = manager.StopPlugin("example-plugin-2")
	if err != nil {
		logger.Error("停止插件2失败", "error", err)
	} else {
		logger.Info("停止插件2成功")
	}

	// 卸载插件
	logger.Info("卸载插件2")
	err = manager.UnloadPlugin("example-plugin-2")
	if err != nil {
		logger.Error("卸载插件2失败", "error", err)
	} else {
		logger.Info("卸载插件2成功")
	}

	// 启动健康检查
	logger.Info("启动健康检查")
	manager.StartHealthCheck()

	// 停止插件管理器
	logger.Info("停止插件管理器")
	manager.Stop()
}
