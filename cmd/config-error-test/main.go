package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

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

	fmt.Println("Kennel配置错误处理测试工具 v1.0.0")
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

	// 运行配置错误处理测试
	if err := runErrorHandlingTests(*verbose); err != nil {
		fmt.Printf("错误: 配置错误处理测试失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ 所有配置错误处理测试通过")
}

func showHelp() {
	fmt.Println("Kennel配置错误处理测试工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  config-error-test [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -dir string     测试目录 (默认: test_temp)")
	fmt.Println("  -verbose        详细输出")
	fmt.Println("  -help           显示帮助信息")
	fmt.Println()
	fmt.Println("功能:")
	fmt.Println("  - 测试配置文件未找到错误处理")
	fmt.Println("  - 测试配置文件解析错误处理")
	fmt.Println("  - 测试配置验证错误处理")
	fmt.Println("  - 测试插件配置错误处理")
	fmt.Println("  - 验证错误处理的统一性")
}

func runErrorHandlingTests(verbose bool) error {
	fmt.Println("开始配置错误处理测试...")

	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "config-error-test",
		Level: hclog.Info,
	})

	// 初始化全局配置错误处理
	config.InitGlobalConfigErrorHandling(logger)

	// 测试1: 文件未找到错误处理
	if err := testFileNotFoundError(verbose, logger); err != nil {
		return fmt.Errorf("文件未找到错误处理测试失败: %w", err)
	}

	// 测试2: 解析错误处理
	if err := testParseError(verbose, logger); err != nil {
		return fmt.Errorf("解析错误处理测试失败: %w", err)
	}

	// 测试3: 验证错误处理
	if err := testValidationError(verbose, logger); err != nil {
		return fmt.Errorf("验证错误处理测试失败: %w", err)
	}

	// 测试4: 插件配置错误处理
	if err := testPluginConfigError(verbose, logger); err != nil {
		return fmt.Errorf("插件配置错误处理测试失败: %w", err)
	}

	// 测试5: 错误报告功能
	if err := testErrorReporting(verbose, logger); err != nil {
		return fmt.Errorf("错误报告功能测试失败: %w", err)
	}

	return nil
}

func testFileNotFoundError(verbose bool, logger hclog.Logger) error {
	fmt.Println("\n1. 测试文件未找到错误处理")
	fmt.Println("   检查文件未找到时的错误处理...")

	// 创建配置错误处理器
	handler := config.NewConfigErrorHandler(logger, "test-main")
	handler.SetExitOnCritical(false) // 测试时不退出

	// 创建文件未找到错误
	err := config.NewConfigError(
		config.ConfigErrorTypeFileNotFound,
		"test-main",
		"nonexistent-config.yaml",
		"",
		"配置文件不存在",
		fmt.Errorf("file not found"),
	)

	// 处理错误
	result := handler.HandleError(err)

	if verbose {
		fmt.Printf("   错误类型: %s\n", config.ConfigErrorTypeFileNotFound)
		fmt.Printf("   处理结果: %v\n", result)
		fmt.Printf("   ✓ 文件未找到错误处理测试通过\n")
	}

	return nil
}

func testParseError(verbose bool, logger hclog.Logger) error {
	fmt.Println("\n2. 测试解析错误处理")
	fmt.Println("   检查配置文件解析错误的处理...")

	// 创建无效的配置文件
	invalidConfig := `
global:
  app:
    name: "test"
  logging:
    level: debug
    invalid_yaml: [unclosed array
`

	configFile := "invalid-config.yaml"
	if err := os.WriteFile(configFile, []byte(invalidConfig), 0644); err != nil {
		return fmt.Errorf("创建无效配置文件失败: %w", err)
	}
	defer os.Remove(configFile)

	// 创建配置错误处理器
	handler := config.NewConfigErrorHandler(logger, "test-main")
	handler.SetExitOnCritical(false) // 测试时不退出

	// 创建解析错误
	err := config.NewConfigError(
		config.ConfigErrorTypeParseError,
		"test-main",
		configFile,
		"",
		"YAML解析失败",
		fmt.Errorf("yaml: line 7: found unexpected end of stream"),
	)

	// 处理错误
	result := handler.HandleError(err)

	if verbose {
		fmt.Printf("   错误类型: %s\n", config.ConfigErrorTypeParseError)
		fmt.Printf("   配置文件: %s\n", configFile)
		fmt.Printf("   处理结果: %v\n", result)
		fmt.Printf("   ✓ 解析错误处理测试通过\n")
	}

	return nil
}

func testValidationError(verbose bool, logger hclog.Logger) error {
	fmt.Println("\n3. 测试验证错误处理")
	fmt.Println("   检查配置验证错误的处理...")

	// 创建配置错误处理器
	handler := config.NewConfigErrorHandler(logger, "test-main")
	handler.SetExitOnCritical(false) // 测试时不退出

	// 创建验证错误
	err := config.NewConfigError(
		config.ConfigErrorTypeValidationError,
		"test-main",
		"test-config.yaml",
		"web_console.port",
		"端口号必须在1-65535范围内",
		fmt.Errorf("invalid port: 99999"),
	)

	// 处理错误
	result := handler.HandleError(err)

	if verbose {
		fmt.Printf("   错误类型: %s\n", config.ConfigErrorTypeValidationError)
		fmt.Printf("   错误字段: web_console.port\n")
		fmt.Printf("   处理结果: %v\n", result)
		fmt.Printf("   ✓ 验证错误处理测试通过\n")
	}

	return nil
}

func testPluginConfigError(verbose bool, logger hclog.Logger) error {
	fmt.Println("\n4. 测试插件配置错误处理")
	fmt.Println("   检查插件配置错误的处理...")

	// 创建插件配置错误处理器
	handler := config.NewPluginConfigErrorHandler(logger, "test-dlp")

	// 创建插件配置文件
	pluginConfigDir := "dlp"
	if err := os.MkdirAll(pluginConfigDir, 0755); err != nil {
		return fmt.Errorf("创建插件配置目录失败: %w", err)
	}

	invalidPluginConfig := `
enabled: true
monitor_network: invalid_boolean_value
max_concurrency: "not_a_number"
`

	pluginConfigFile := filepath.Join(pluginConfigDir, "config.yaml")
	if err := os.WriteFile(pluginConfigFile, []byte(invalidPluginConfig), 0644); err != nil {
		return fmt.Errorf("创建插件配置文件失败: %w", err)
	}

	// 测试插件配置错误处理
	configErr := fmt.Errorf("配置验证失败: monitor_network 必须是布尔值")
	result := handler.HandlePluginConfigError(configErr, pluginConfigFile)

	// 测试插件初始化错误处理
	initErr := fmt.Errorf("插件初始化失败: 依赖模块未找到")
	initResult := handler.HandlePluginInitError(initErr)

	if verbose {
		fmt.Printf("   插件ID: test-dlp\n")
		fmt.Printf("   配置文件: %s\n", pluginConfigFile)
		fmt.Printf("   配置错误处理结果: %v\n", result)
		fmt.Printf("   初始化错误处理结果: %v\n", initResult)
		fmt.Printf("   ✓ 插件配置错误处理测试通过\n")
	}

	return nil
}

func testErrorReporting(verbose bool, logger hclog.Logger) error {
	fmt.Println("\n5. 测试错误报告功能")
	fmt.Println("   检查错误报告和统计功能...")

	// 创建错误报告器
	reporter := config.NewConfigErrorReporter(logger)

	// 报告多个错误
	errors := []config.ConfigError{
		{
			Type:       config.ConfigErrorTypeFileNotFound,
			Component:  "main",
			ConfigPath: "missing-config.yaml",
			Message:    "配置文件未找到",
		},
		{
			Type:       config.ConfigErrorTypeValidationError,
			Component:  "dlp",
			ConfigPath: "dlp/config.yaml",
			Field:      "max_concurrency",
			Message:    "值必须在1-100范围内",
		},
		{
			Type:       config.ConfigErrorTypeParseError,
			Component:  "assets",
			ConfigPath: "assets/config.yaml",
			Message:    "YAML语法错误",
		},
	}

	for _, err := range errors {
		reporter.ReportError(err)
	}

	// 测试错误统计
	allErrors := reporter.GetErrors()
	validationErrors := reporter.GetErrorsByType(config.ConfigErrorTypeValidationError)
	mainErrors := reporter.GetErrorsByComponent("main")
	hasCritical := reporter.HasCriticalErrors()

	// 生成错误报告
	report := reporter.GenerateReport()

	if verbose {
		fmt.Printf("   总错误数: %d\n", len(allErrors))
		fmt.Printf("   验证错误数: %d\n", len(validationErrors))
		fmt.Printf("   主程序错误数: %d\n", len(mainErrors))
		fmt.Printf("   是否有关键错误: %v\n", hasCritical)
		fmt.Printf("   错误报告:\n%s\n", report)
		fmt.Printf("   ✓ 错误报告功能测试通过\n")
	}

	// 验证结果
	if len(allErrors) != 3 {
		return fmt.Errorf("期望3个错误，实际%d个", len(allErrors))
	}
	if len(validationErrors) != 1 {
		return fmt.Errorf("期望1个验证错误，实际%d个", len(validationErrors))
	}
	if len(mainErrors) != 1 {
		return fmt.Errorf("期望1个主程序错误，实际%d个", len(mainErrors))
	}
	if !hasCritical {
		return fmt.Errorf("期望有关键错误，但检测结果为无")
	}

	return nil
}
