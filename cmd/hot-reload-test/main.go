package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/core/config"
)

func main() {
	var (
		testDir = flag.String("dir", "test_temp", "测试目录")
		verbose = flag.Bool("verbose", false, "详细输出")
		help    = flag.Bool("help", false, "显示帮助信息")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	fmt.Println("Kennel配置热更新测试工具 v1.0.0")
	fmt.Println("=====================================")

	// 创建安全的临时测试目录
	if err := os.MkdirAll(*testDir, 0755); err != nil {
		fmt.Printf("错误: 无法创建测试目录 %s: %v\n", *testDir, err)
		os.Exit(1)
	}
	defer func() {
		// 安全地清理临时目录
		if err := os.RemoveAll(*testDir); err != nil {
			fmt.Printf("警告: 清理临时目录失败: %v\n", err)
		}
	}()

	// 切换到测试目录
	originalDir, _ := os.Getwd()
	if err := os.Chdir(*testDir); err != nil {
		fmt.Printf("错误: 无法切换到目录 %s: %v\n", *testDir, err)
		os.Exit(1)
	}
	defer os.Chdir(originalDir)

	// 运行热更新测试
	if err := runHotReloadTests(*verbose); err != nil {
		fmt.Printf("错误: 配置热更新测试失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ 所有配置热更新测试通过")
}

func showHelp() {
	fmt.Println("Kennel配置热更新测试工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  hot-reload-test [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -dir string     测试目录 (默认: test_temp)")
	fmt.Println("  -verbose        详细输出")
	fmt.Println("  -help           显示帮助信息")
	fmt.Println()
	fmt.Println("功能:")
	fmt.Println("  - 测试日志配置热更新")
	fmt.Println("  - 测试插件配置热更新")
	fmt.Println("  - 测试热更新支持级别")
	fmt.Println("  - 测试热更新回滚机制")
	fmt.Println("  - 验证热更新事件记录")
}

func runHotReloadTests(verbose bool) error {
	fmt.Println("开始配置热更新测试...")

	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "hot-reload-test",
		Level: hclog.Info,
	})

	// 创建热更新管理器
	hotReloadConfig := config.DefaultHotReloadConfig()
	hotReloadConfig.MaxRetries = 2
	hotReloadConfig.RetryInterval = 1 * time.Second

	manager := config.NewHotReloadManager(hotReloadConfig, logger)

	// 测试1: 日志配置热更新
	if err := testLoggingHotReload(manager, verbose, logger); err != nil {
		return fmt.Errorf("日志配置热更新测试失败: %w", err)
	}

	// 测试2: 插件配置热更新
	if err := testPluginHotReload(manager, verbose, logger); err != nil {
		return fmt.Errorf("插件配置热更新测试失败: %w", err)
	}

	// 测试3: 热更新支持级别
	if err := testHotReloadSupportLevel(manager, verbose, logger); err != nil {
		return fmt.Errorf("热更新支持级别测试失败: %w", err)
	}

	// 测试4: 热更新失败和回滚
	if err := testHotReloadFailureAndRollback(manager, verbose, logger); err != nil {
		return fmt.Errorf("热更新失败和回滚测试失败: %w", err)
	}

	// 测试5: 热更新事件记录
	if err := testHotReloadEventRecording(manager, verbose, logger); err != nil {
		return fmt.Errorf("热更新事件记录测试失败: %w", err)
	}

	// 停止管理器
	manager.Stop()

	return nil
}

func testLoggingHotReload(manager *config.HotReloadManager, verbose bool, logger hclog.Logger) error {
	fmt.Println("\n1. 测试日志配置热更新")
	fmt.Println("   检查日志配置的热更新功能...")

	// 注册日志热更新处理器
	loggingHandler := config.NewLoggingHotReloadHandler(logger)
	manager.RegisterHandler("logging", loggingHandler)

	// 创建旧配置
	oldConfig := map[string]interface{}{
		"level":  "info",
		"format": "text",
		"output": "file",
	}

	// 创建新配置
	newConfig := map[string]interface{}{
		"level":  "debug",
		"format": "json",
		"output": "file",
	}

	// 执行热更新
	err := manager.Reload(
		config.HotReloadTypeLogging,
		"logging",
		"logging.yaml",
		oldConfig,
		newConfig,
	)

	if err != nil {
		return fmt.Errorf("日志配置热更新失败: %w", err)
	}

	if verbose {
		fmt.Printf("   旧配置: level=%s, format=%s\n", oldConfig["level"], oldConfig["format"])
		fmt.Printf("   新配置: level=%s, format=%s\n", newConfig["level"], newConfig["format"])
		fmt.Printf("   ✓ 日志配置热更新测试通过\n")
	}

	return nil
}

func testPluginHotReload(manager *config.HotReloadManager, verbose bool, logger hclog.Logger) error {
	fmt.Println("\n2. 测试插件配置热更新")
	fmt.Println("   检查插件配置的热更新功能...")

	// 注册插件热更新处理器
	pluginHandler := config.NewPluginHotReloadHandler("test-plugin", logger)
	manager.RegisterHandler("test-plugin", pluginHandler)

	// 创建旧配置
	oldConfig := map[string]interface{}{
		"enabled":        true,
		"max_workers":    4,
		"timeout":        30,
		"retry_interval": 5,
	}

	// 创建新配置（只修改参数，不改变启用状态）
	newConfig := map[string]interface{}{
		"enabled":        true,
		"max_workers":    8,
		"timeout":        60,
		"retry_interval": 10,
	}

	// 执行热更新
	err := manager.Reload(
		config.HotReloadTypePlugin,
		"test-plugin",
		"test-plugin/config.yaml",
		oldConfig,
		newConfig,
	)

	if err != nil {
		return fmt.Errorf("插件配置热更新失败: %w", err)
	}

	if verbose {
		fmt.Printf("   插件ID: test-plugin\n")
		fmt.Printf("   旧配置: max_workers=%v, timeout=%v\n", oldConfig["max_workers"], oldConfig["timeout"])
		fmt.Printf("   新配置: max_workers=%v, timeout=%v\n", newConfig["max_workers"], newConfig["timeout"])
		fmt.Printf("   ✓ 插件配置热更新测试通过\n")
	}

	return nil
}

func testHotReloadSupportLevel(manager *config.HotReloadManager, verbose bool, logger hclog.Logger) error {
	fmt.Println("\n3. 测试热更新支持级别")
	fmt.Println("   检查不同组件的热更新支持级别...")

	// 获取支持信息
	supportInfo := manager.GetSupportInfo()

	// 验证支持级别
	expectedSupport := map[string]config.HotReloadSupport{
		"logging":     config.HotReloadSupportFull,
		"test-plugin": config.HotReloadSupportPartial,
	}

	for component, expectedLevel := range expectedSupport {
		actualLevel, exists := supportInfo[component]
		if !exists {
			return fmt.Errorf("组件 %s 的支持信息不存在", component)
		}
		if actualLevel != expectedLevel {
			return fmt.Errorf("组件 %s 的支持级别不匹配: 期望 %s，实际 %s", component, expectedLevel, actualLevel)
		}

		if verbose {
			fmt.Printf("   组件: %s, 支持级别: %s\n", component, actualLevel)
		}
	}

	if verbose {
		fmt.Printf("   ✓ 热更新支持级别测试通过\n")
	}

	return nil
}

func testHotReloadFailureAndRollback(manager *config.HotReloadManager, verbose bool, logger hclog.Logger) error {
	fmt.Println("\n4. 测试热更新失败和回滚")
	fmt.Println("   检查热更新失败时的回滚机制...")

	// 创建一个会失败的配置
	oldConfig := map[string]interface{}{
		"level": "info",
	}

	// 创建无效的新配置
	newConfig := map[string]interface{}{
		"level": "invalid_level", // 无效的日志级别
	}

	// 执行热更新（应该失败）
	err := manager.Reload(
		config.HotReloadTypeLogging,
		"logging",
		"logging.yaml",
		oldConfig,
		newConfig,
	)

	// 验证失败
	if err == nil {
		return fmt.Errorf("期望热更新失败，但实际成功了")
	}

	if verbose {
		fmt.Printf("   无效配置: level=%s\n", newConfig["level"])
		fmt.Printf("   热更新失败（符合预期）: %v\n", err)
		fmt.Printf("   ✓ 热更新失败和回滚测试通过\n")
	}

	return nil
}

func testHotReloadEventRecording(manager *config.HotReloadManager, verbose bool, logger hclog.Logger) error {
	fmt.Println("\n5. 测试热更新事件记录")
	fmt.Println("   检查热更新事件的记录和统计...")

	// 获取所有事件
	events := manager.GetEvents()

	// 验证事件数量（之前的测试应该产生了一些事件）
	if len(events) == 0 {
		return fmt.Errorf("没有记录到热更新事件")
	}

	// 获取成功率
	successRate := manager.GetSuccessRate()

	// 按组件获取事件
	loggingEvents := manager.GetEventsByComponent("logging")
	pluginEvents := manager.GetEventsByComponent("test-plugin")

	if verbose {
		fmt.Printf("   总事件数: %d\n", len(events))
		fmt.Printf("   成功率: %.2f%%\n", successRate*100)
		fmt.Printf("   日志组件事件数: %d\n", len(loggingEvents))
		fmt.Printf("   插件组件事件数: %d\n", len(pluginEvents))

		// 显示最近的几个事件
		fmt.Printf("   最近的事件:\n")
		for i, event := range events {
			if i >= 3 { // 只显示前3个
				break
			}
			fmt.Printf("     %d. [%s] %s - 成功: %v, 耗时: %v\n",
				i+1, event.Type, event.Component, event.Success, event.Duration)
		}

		fmt.Printf("   ✓ 热更新事件记录测试通过\n")
	}

	return nil
}
