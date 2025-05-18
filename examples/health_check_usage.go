package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lomehong/kennel/pkg/core"
	"github.com/lomehong/kennel/pkg/health"
)

// 本示例展示如何在AppFramework中使用健康检查和自我修复
func main() {
	// 创建应用程序实例
	app := core.NewApp("config.yaml")

	// 初始化应用程序
	if err := app.Init(); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 健康检查和自我修复使用示例 ===")

	// 示例1: 基本健康检查
	fmt.Println("\n=== 示例1: 基本健康检查 ===")
	basicHealthCheck(app)

	// 示例2: 自定义健康检查器
	fmt.Println("\n=== 示例2: 自定义健康检查器 ===")
	customHealthChecker(app)

	// 示例3: 自我修复
	fmt.Println("\n=== 示例3: 自我修复 ===")
	selfHealing(app)

	// 示例4: 健康监控
	fmt.Println("\n=== 示例4: 健康监控 ===")
	healthMonitoring(app)

	// 停止应用程序
	app.Stop()
	fmt.Println("\n应用程序已停止")
}

// 基本健康检查
func basicHealthCheck(app *core.App) {
	// 获取健康检查注册表
	registry := app.GetHealthRegistry()

	// 创建上下文
	ctx := context.Background()

	// 运行所有健康检查
	fmt.Println("运行所有健康检查")
	results := registry.RunChecks(ctx)

	// 输出结果
	for name, result := range results {
		fmt.Printf("- %s: %s (%s)\n", name, result.Status, result.Message)
	}

	// 获取系统状态
	systemStatus := registry.GetSystemStatus(ctx)
	fmt.Printf("\n系统状态: %s (%s)\n", systemStatus.Status, systemStatus.Message)

	// 运行单个健康检查
	fmt.Println("\n运行内存健康检查")
	memoryResult, ok := registry.RunCheck(ctx, "memory")
	if ok {
		fmt.Printf("- 内存: %s (%s)\n", memoryResult.Status, memoryResult.Message)
		if memoryResult.Details["used_percent"] != nil {
			fmt.Printf("  使用率: %.2f%%\n", memoryResult.Details["used_percent"])
		}
	} else {
		fmt.Println("- 内存检查器未找到")
	}

	// 检查应用程序健康状态
	fmt.Println("\n检查应用程序健康状态")
	appResult, ok := registry.RunCheck(ctx, "app")
	if ok {
		fmt.Printf("- 应用程序: %s (%s)\n", appResult.Status, appResult.Message)
		if appResult.Details["uptime"] != nil {
			fmt.Printf("  运行时间: %s\n", appResult.Details["uptime"])
		}
		if appResult.Details["version"] != nil {
			fmt.Printf("  版本: %s\n", appResult.Details["version"])
		}
	} else {
		fmt.Println("- 应用程序检查器未找到")
	}
}

// 自定义健康检查器
func customHealthChecker(app *core.App) {
	// 创建自定义健康检查器
	customChecker := health.NewSimpleChecker(
		"custom_checker",
		"自定义检查器",
		"custom",
		func(ctx context.Context) health.CheckResult {
			// 模拟一些检查逻辑
			time.Sleep(100 * time.Millisecond)

			// 返回健康状态
			return health.CheckResult{
				Status:  health.StatusHealthy,
				Message: "自定义检查通过",
				Details: map[string]interface{}{
					"custom_value": 42,
					"check_time":   time.Now().Format(time.RFC3339),
				},
			}
		},
	)

	// 注册自定义检查器
	app.RegisterHealthChecker(customChecker)

	// 创建上下文
	ctx := context.Background()

	// 运行自定义健康检查
	fmt.Println("运行自定义健康检查")
	result, ok := app.GetHealthRegistry().RunCheck(ctx, "custom_checker")
	if ok {
		fmt.Printf("- 自定义检查器: %s (%s)\n", result.Status, result.Message)
		if result.Details["custom_value"] != nil {
			fmt.Printf("  自定义值: %v\n", result.Details["custom_value"])
		}
		if result.Details["check_time"] != nil {
			fmt.Printf("  检查时间: %s\n", result.Details["check_time"])
		}
	} else {
		fmt.Println("- 自定义检查器未找到")
	}

	// 注销自定义检查器
	app.UnregisterHealthChecker("custom_checker")
	fmt.Println("\n已注销自定义检查器")
}

// 自我修复
func selfHealing(app *core.App) {
	// 创建一个总是失败的检查器
	alwaysFailChecker := health.NewSimpleChecker(
		"always_fail",
		"总是失败的检查器",
		"test",
		func(ctx context.Context) health.CheckResult {
			return health.CheckResult{
				Status:  health.StatusUnhealthy,
				Message: "总是失败",
				Details: map[string]interface{}{
					"reason": "示例",
				},
			}
		},
	)

	// 注册检查器
	app.RegisterHealthChecker(alwaysFailChecker)

	// 创建修复动作
	repaired := false
	repairAction := health.NewSimpleRepairAction(
		"fix_always_fail",
		"修复总是失败的检查器",
		func(ctx context.Context) error {
			fmt.Println("执行修复动作")
			repaired = true
			return nil
		},
	)

	// 创建修复策略
	repairStrategy := health.NewSimpleRepairStrategy(
		"always_fail_strategy",
		func(result health.CheckResult) bool {
			return result.Status == health.StatusUnhealthy
		},
		func(result health.CheckResult) health.RepairAction {
			return repairAction
		},
	)

	// 注册修复策略
	app.RegisterRepairStrategy("always_fail", repairStrategy)

	// 创建上下文
	ctx := context.Background()

	// 检查并修复
	fmt.Println("检查并修复")
	checkResult, repairResult, err := app.CheckAndRepair(ctx, "always_fail")

	if err != nil {
		fmt.Printf("修复失败: %v\n", err)
	} else {
		fmt.Printf("- 检查结果: %s (%s)\n", checkResult.Status, checkResult.Message)

		if repairResult != nil {
			fmt.Printf("- 修复结果: %v (%s)\n", repairResult.Success, repairResult.Message)
			fmt.Printf("  动作: %s\n", repairResult.ActionName)
			fmt.Printf("  耗时: %v\n", repairResult.Duration)
		}
	}

	// 验证修复是否执行
	fmt.Printf("\n修复动作是否执行: %v\n", repaired)

	// 获取修复历史
	history := app.GetRepairHistory()
	fmt.Printf("\n修复历史 (共%d条):\n", len(history))
	for i, result := range history {
		fmt.Printf("- 历史记录 %d:\n", i+1)
		fmt.Printf("  检查器: %s\n", result.CheckerName)
		fmt.Printf("  动作: %s\n", result.ActionName)
		fmt.Printf("  成功: %v\n", result.Success)
		fmt.Printf("  时间: %s\n", result.StartTime.Format(time.RFC3339))
	}

	// 注销检查器和策略
	app.UnregisterHealthChecker("always_fail")
	app.UnregisterRepairStrategy("always_fail")
	fmt.Println("\n已注销检查器和策略")
}

// 健康监控
func healthMonitoring(app *core.App) {
	// 启动健康监控
	fmt.Println("启动健康监控")
	app.StartHealthMonitor()

	// 等待一段时间
	fmt.Println("监控运行中...")
	time.Sleep(2 * time.Second)

	// 获取所有状态
	allStatus := app.GetHealthStatus()
	fmt.Printf("\n所有检查器状态 (共%d个):\n", len(allStatus))
	for name, status := range allStatus {
		fmt.Printf("- %s: %s (%s)\n", name, status.Status, status.Message)
		fmt.Printf("  总检查次数: %d\n", status.TotalChecks)
		fmt.Printf("  成功次数: %d\n", status.TotalSuccesses)
		fmt.Printf("  失败次数: %d\n", status.TotalFailures)
		if !status.LastChecked.IsZero() {
			fmt.Printf("  最后检查时间: %s\n", status.LastChecked.Format(time.RFC3339))
		}
	}

	// 获取系统健康状态
	systemHealth := app.GetSystemHealth()
	fmt.Printf("\n系统健康状态: %s\n", systemHealth)

	// 停止健康监控
	fmt.Println("\n停止健康监控")
	app.StopHealthMonitor()
}
