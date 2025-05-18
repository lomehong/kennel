package health

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
)

// 以下是健康检查和自我修复的使用示例

// ExampleHealthCheck 展示健康检查的基本用法
func ExampleHealthCheck() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "health-check",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建健康检查注册表
	registry := NewCheckerRegistry(logger)

	// 创建内存检查器
	memoryChecker := NewMemoryChecker(80.0)

	// 创建CPU检查器
	cpuChecker := NewCPUChecker(70.0)

	// 创建磁盘检查器
	diskChecker := NewDiskChecker("/", 90.0)

	// 创建Goroutine检查器
	goroutineChecker := NewGoroutineChecker(1000)

	// 创建HTTP检查器
	httpChecker := NewHTTPChecker("https://www.example.com", 5*time.Second, 200)

	// 注册检查器
	registry.RegisterChecker(memoryChecker)
	registry.RegisterChecker(cpuChecker)
	registry.RegisterChecker(diskChecker)
	registry.RegisterChecker(goroutineChecker)
	registry.RegisterChecker(httpChecker)

	// 创建上下文
	ctx := context.Background()

	// 运行所有健康检查
	logger.Info("运行所有健康检查")
	results := registry.RunChecks(ctx)

	// 输出结果
	for name, result := range results {
		logger.Info("检查结果",
			"name", name,
			"status", result.Status,
			"message", result.Message,
		)
	}

	// 获取系统状态
	systemStatus := registry.GetSystemStatus(ctx)
	logger.Info("系统状态",
		"status", systemStatus.Status,
		"message", systemStatus.Message,
	)

	// 运行单个健康检查
	logger.Info("运行内存健康检查")
	memoryResult, _ := registry.RunCheck(ctx, "memory")
	logger.Info("内存检查结果",
		"status", memoryResult.Status,
		"message", memoryResult.Message,
		"used_percent", memoryResult.Details["used_percent"],
	)
}

// ExampleSelfHealing 展示自我修复的用法
func ExampleSelfHealing() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "self-healing",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建健康检查注册表
	registry := NewCheckerRegistry(logger)

	// 创建自我修复器
	healer := NewRepairSelfHealer(registry, logger)

	// 创建一个总是失败的检查器
	alwaysFailChecker := NewSimpleChecker(
		"always_fail",
		"总是失败的检查器",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "总是失败",
				Details: map[string]interface{}{
					"reason": "示例",
				},
			}
		},
	)

	// 注册检查器
	registry.RegisterChecker(alwaysFailChecker)

	// 创建修复动作
	repairAction := NewSimpleRepairAction(
		"fix_always_fail",
		"修复总是失败的检查器",
		func(ctx context.Context) error {
			logger.Info("执行修复动作")
			return nil
		},
	)

	// 创建修复策略
	repairStrategy := NewSimpleRepairStrategy(
		"always_fail_strategy",
		func(result CheckResult) bool {
			return result.Status == StatusUnhealthy
		},
		func(result CheckResult) RepairAction {
			return repairAction
		},
	)

	// 注册修复策略
	healer.RegisterStrategy("always_fail", repairStrategy)

	// 创建上下文
	ctx := context.Background()

	// 检查并修复
	logger.Info("检查并修复")
	checkResult, repairResult, err := healer.CheckAndRepair(ctx, "always_fail")

	if err != nil {
		logger.Error("修复失败", "error", err)
	} else {
		logger.Info("检查结果",
			"status", checkResult.Status,
			"message", checkResult.Message,
		)

		if repairResult != nil {
			logger.Info("修复结果",
				"success", repairResult.Success,
				"action", repairResult.ActionName,
				"message", repairResult.Message,
				"duration", repairResult.Duration,
			)
		}
	}

	// 获取修复历史
	history := healer.GetRepairHistory()
	logger.Info("修复历史", "count", len(history))
	for i, result := range history {
		logger.Info(fmt.Sprintf("历史记录 %d", i+1),
			"checker", result.CheckerName,
			"action", result.ActionName,
			"success", result.Success,
			"time", result.StartTime,
		)
	}
}

// ExampleHealthMonitor 展示健康监控的用法
func ExampleHealthMonitor() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "health-monitor",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建健康检查注册表
	registry := NewCheckerRegistry(logger)

	// 创建自我修复器
	healer := NewRepairSelfHealer(registry, logger)

	// 创建监控配置
	config := &MonitorConfig{
		CheckInterval:    5 * time.Second,
		InitialDelay:     1 * time.Second,
		AutoRepair:       true,
		FailureThreshold: 2,
		SuccessThreshold: 1,
	}

	// 创建健康监控器
	monitor := NewHealthMonitor(registry, healer, config, logger)

	// 创建内存检查器
	memoryChecker := NewMemoryChecker(80.0)

	// 创建CPU检查器
	cpuChecker := NewCPUChecker(70.0)

	// 创建磁盘检查器
	diskChecker := NewDiskChecker("/", 90.0)

	// 注册检查器
	registry.RegisterChecker(memoryChecker)
	registry.RegisterChecker(cpuChecker)
	registry.RegisterChecker(diskChecker)

	// 创建修复动作
	memoryRepairAction := NewFreeMemoryAction()

	// 创建修复策略
	memoryRepairStrategy := NewSimpleRepairStrategy(
		"memory_repair_strategy",
		func(result CheckResult) bool {
			return result.Status == StatusUnhealthy
		},
		func(result CheckResult) RepairAction {
			return memoryRepairAction
		},
	)

	// 注册修复策略
	healer.RegisterStrategy("memory", memoryRepairStrategy)

	// 启动监控
	logger.Info("启动健康监控")
	monitor.Start()

	// 等待一段时间
	logger.Info("监控运行中，按Ctrl+C停止...")
	time.Sleep(30 * time.Second)

	// 获取所有状态
	allStatus := monitor.GetAllStatus()
	logger.Info("所有检查器状态", "count", len(allStatus))
	for name, status := range allStatus {
		logger.Info(name,
			"status", status.Status,
			"message", status.Message,
			"total_checks", status.TotalChecks,
			"total_successes", status.TotalSuccesses,
			"total_failures", status.TotalFailures,
		)
	}

	// 获取系统健康状态
	systemHealth := monitor.GetSystemHealth()
	logger.Info("系统健康状态", "status", systemHealth)

	// 停止监控
	logger.Info("停止健康监控")
	monitor.Stop()
}

// ExampleCompositeChecker 展示组合检查器的用法
func ExampleCompositeChecker() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "composite-checker",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建健康检查注册表
	registry := NewCheckerRegistry(logger)

	// 创建内存检查器
	memoryChecker := NewMemoryChecker(80.0)

	// 创建CPU检查器
	cpuChecker := NewCPUChecker(70.0)

	// 创建组合检查器
	resourceChecker := NewCompositeChecker(
		"resource",
		"资源检查器",
		CheckerTypeResource,
		[]Checker{memoryChecker, cpuChecker},
		nil,
	)

	// 注册检查器
	registry.RegisterChecker(resourceChecker)

	// 创建上下文
	ctx := context.Background()

	// 运行组合检查
	logger.Info("运行组合检查")
	result, _ := registry.RunCheck(ctx, "resource")

	logger.Info("组合检查结果",
		"status", result.Status,
		"message", result.Message,
	)
}
