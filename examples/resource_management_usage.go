package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lomehong/kennel/pkg/core"
	"github.com/lomehong/kennel/pkg/resource"
)

// 本示例展示如何在AppFramework中使用资源管理和限制
func main() {
	// 创建应用程序实例
	app := core.NewApp("config.yaml")

	// 初始化应用程序
	if err := app.Init(); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 资源管理和限制使用示例 ===")

	// 示例1: 获取资源使用情况
	fmt.Println("\n=== 示例1: 获取资源使用情况 ===")
	getResourceUsage(app)

	// 示例2: 设置资源限制
	fmt.Println("\n=== 示例2: 设置资源限制 ===")
	setResourceLimits(app)

	// 示例3: 资源告警处理
	fmt.Println("\n=== 示例3: 资源告警处理 ===")
	handleResourceAlerts(app)

	// 示例4: 资源使用统计
	fmt.Println("\n=== 示例4: 资源使用统计 ===")
	resourceUsageStatistics(app)

	// 示例5: 资源优化
	fmt.Println("\n=== 示例5: 资源优化 ===")
	optimizeResources(app)

	// 示例6: 自定义资源限制动作
	fmt.Println("\n=== 示例6: 自定义资源限制动作 ===")
	customResourceLimitActions(app)

	// 停止应用程序
	app.Stop()
	fmt.Println("\n应用程序已停止")
}

// 获取资源使用情况
func getResourceUsage(app *core.App) {
	// 获取资源管理器
	resourceManager := app.GetResourceManager()

	// 获取当前资源使用情况
	usage := resourceManager.GetResourceUsage()
	if usage == nil {
		fmt.Println("无法获取资源使用情况")
		return
	}

	// 打印CPU使用情况
	fmt.Println("CPU使用情况:")
	fmt.Printf("  - CPU使用率: %.2f%%\n", usage.CPUUsage)
	fmt.Printf("  - CPU核心数: %d\n", usage.CPUCount)
	if len(usage.CPUPercent) > 0 {
		fmt.Println("  - 各核心使用率:")
		for i, percent := range usage.CPUPercent {
			fmt.Printf("    - 核心 %d: %.2f%%\n", i, percent)
		}
	}

	// 打印内存使用情况
	fmt.Println("\n内存使用情况:")
	fmt.Printf("  - 内存使用率: %.2f%%\n", usage.MemoryPercent)
	fmt.Printf("  - 总内存: %s\n", formatBytes(usage.MemoryTotal))
	fmt.Printf("  - 已用内存: %s\n", formatBytes(usage.MemoryUsage))
	fmt.Printf("  - 空闲内存: %s\n", formatBytes(usage.MemoryFree))
	fmt.Printf("  - 可用内存: %s\n", formatBytes(usage.MemoryAvailable))

	// 打印磁盘使用情况
	fmt.Println("\n磁盘使用情况:")
	fmt.Printf("  - 磁盘使用率: %.2f%%\n", usage.DiskPercent)
	fmt.Printf("  - 总磁盘空间: %s\n", formatBytes(usage.DiskTotal))
	fmt.Printf("  - 已用磁盘空间: %s\n", formatBytes(usage.DiskUsage))
	fmt.Printf("  - 空闲磁盘空间: %s\n", formatBytes(usage.DiskFree))
	fmt.Printf("  - 磁盘读取: %s\n", formatBytes(usage.DiskReadBytes))
	fmt.Printf("  - 磁盘写入: %s\n", formatBytes(usage.DiskWriteBytes))

	// 打印网络使用情况
	fmt.Println("\n网络使用情况:")
	fmt.Printf("  - 发送字节数: %s\n", formatBytes(usage.NetworkSentBytes))
	fmt.Printf("  - 接收字节数: %s\n", formatBytes(usage.NetworkReceivedBytes))
	fmt.Printf("  - 发送数据包数: %d\n", usage.NetworkSentPackets)
	fmt.Printf("  - 接收数据包数: %d\n", usage.NetworkRecvPackets)

	// 打印进程信息
	fmt.Println("\n进程信息:")
	fmt.Printf("  - 进程ID: %d\n", usage.ProcessID)
	fmt.Printf("  - 进程名称: %s\n", usage.ProcessName)
	fmt.Printf("  - 进程状态: %s\n", usage.ProcessStatus)
	fmt.Printf("  - 线程数: %d\n", usage.ProcessThreads)
	fmt.Printf("  - 文件描述符数: %d\n", usage.ProcessFDs)

	// 获取进程详细信息
	procInfo, err := resourceManager.GetProcessInfo()
	if err != nil {
		fmt.Printf("获取进程详细信息失败: %v\n", err)
	} else {
		fmt.Println("\n进程详细信息:")
		for key, value := range procInfo {
			fmt.Printf("  - %s: %v\n", key, value)
		}
	}

	// 获取系统信息
	sysInfo := resourceManager.GetSystemInfo()
	fmt.Println("\n系统信息:")
	for key, value := range sysInfo {
		fmt.Printf("  - %s: %v\n", key, value)
	}
}

// 设置资源限制
func setResourceLimits(app *core.App) {
	// 获取资源管理器
	resourceManager := app.GetResourceManager()

	// 设置CPU使用限制
	fmt.Println("设置CPU使用限制: 80%")
	resourceManager.LimitCPU(80, resource.ResourceLimitActionLog)

	// 设置内存使用限制
	memoryLimit := uint64(1024 * 1024 * 1024) // 1GB
	fmt.Printf("设置内存使用限制: %s\n", formatBytes(memoryLimit))
	resourceManager.LimitMemory(memoryLimit, resource.ResourceLimitActionAlert)

	// 设置磁盘使用限制
	diskLimit := uint64(10 * 1024 * 1024 * 1024) // 10GB
	fmt.Printf("设置磁盘使用限制: %s\n", formatBytes(diskLimit))
	resourceManager.LimitDisk(diskLimit, resource.ResourceLimitActionThrottle)

	// 设置网络使用限制
	networkLimit := uint64(10 * 1024 * 1024) // 10MB/s
	fmt.Printf("设置网络使用限制: %s/s\n", formatBytes(networkLimit))
	resourceManager.LimitNetwork(networkLimit, resource.ResourceLimitActionReject)

	// 获取资源限制
	limits := resourceManager.GetResourceLimits()
	fmt.Println("\n当前资源限制:")
	for resourceType, resourceLimits := range limits {
		for _, limit := range resourceLimits {
			fmt.Printf("  - %s: %s限制 %s, 动作: %s\n",
				resourceType, limit.LimitType, formatBytes(limit.Value), limit.Action)
		}
	}

	// 移除CPU使用限制
	fmt.Println("\n移除CPU使用限制")
	resourceManager.RemoveLimit(resource.ResourceTypeCPU, resource.ResourceLimitTypeSoft)

	// 获取资源限制
	limits = resourceManager.GetResourceLimits()
	fmt.Println("\n当前资源限制:")
	for resourceType, resourceLimits := range limits {
		for _, limit := range resourceLimits {
			fmt.Printf("  - %s: %s限制 %s, 动作: %s\n",
				resourceType, limit.LimitType, formatBytes(limit.Value), limit.Action)
		}
	}
}

// 资源告警处理
func handleResourceAlerts(app *core.App) {
	// 获取资源管理器
	resourceManager := app.GetResourceManager()

	// 注册告警处理器
	resourceManager.RegisterAlertHandler(func(resourceType resource.ResourceType, message string) {
		fmt.Printf("收到资源告警: %s - %s\n", resourceType, message)
	})

	// 设置一个很低的内存限制，确保会触发告警
	memoryLimit := uint64(1 * 1024 * 1024) // 1MB
	fmt.Printf("设置很低的内存限制: %s\n", formatBytes(memoryLimit))
	resourceManager.LimitMemory(memoryLimit, resource.ResourceLimitActionAlert)

	// 等待一段时间，让告警触发
	fmt.Println("等待告警触发...")
	time.Sleep(2 * time.Second)

	// 获取资源告警
	alerts := resourceManager.GetResourceAlerts()
	fmt.Println("\n当前资源告警:")
	for resourceType, messages := range alerts {
		for _, message := range messages {
			fmt.Printf("  - %s: %s\n", resourceType, message)
		}
	}

	// 移除内存限制
	resourceManager.RemoveLimit(resource.ResourceTypeMemory, resource.ResourceLimitTypeSoft)
}

// 资源使用统计
func resourceUsageStatistics(app *core.App) {
	// 获取资源管理器
	resourceManager := app.GetResourceManager()

	// 等待一段时间，收集更多数据
	fmt.Println("收集资源使用数据...")
	time.Sleep(3 * time.Second)

	// 获取资源使用历史
	history := resourceManager.GetResourceUsageHistory()
	fmt.Printf("收集了 %d 条资源使用记录\n", len(history))

	// 获取资源统计信息
	stats := resourceManager.GetResourceStats()
	fmt.Println("\n资源使用统计信息:")
	for key, value := range stats {
		fmt.Printf("  - %s: %v\n", key, value)
	}

	// 打印CPU使用率历史
	if len(history) > 0 {
		fmt.Println("\nCPU使用率历史:")
		for i, usage := range history {
			fmt.Printf("  - 记录 %d: %.2f%% (时间: %s)\n",
				i+1, usage.CPUUsage, usage.Timestamp.Format(time.RFC3339))
		}
	}

	// 打印内存使用率历史
	if len(history) > 0 {
		fmt.Println("\n内存使用率历史:")
		for i, usage := range history {
			fmt.Printf("  - 记录 %d: %.2f%% (时间: %s)\n",
				i+1, usage.MemoryPercent, usage.Timestamp.Format(time.RFC3339))
		}
	}
}

// 资源优化
func optimizeResources(app *core.App) {
	// 获取资源管理器
	resourceManager := app.GetResourceManager()

	// 优化内存使用
	fmt.Println("优化内存使用...")
	resourceManager.OptimizeMemoryUsage()

	// 优化CPU使用
	fmt.Println("优化CPU使用...")
	resourceManager.OptimizeCPUUsage()

	// 设置进程优先级
	fmt.Println("设置进程优先级...")
	err := resourceManager.SetProcessPriority(0)
	if err != nil {
		fmt.Printf("设置进程优先级失败: %v\n", err)
	} else {
		fmt.Println("进程优先级已设置为正常")
	}

	// 设置GOMAXPROCS
	cpuCount := app.GetResourceManager().GetResourceUsage().CPUCount
	fmt.Printf("设置GOMAXPROCS为CPU核心数: %d\n", cpuCount)
	resourceManager.SetGOMAXPROCS(cpuCount)

	// 优化所有资源使用
	fmt.Println("优化所有资源使用...")
	resourceManager.OptimizeResourceUsage()

	// 获取系统信息
	sysInfo := resourceManager.GetSystemInfo()
	fmt.Println("\n优化后的系统信息:")
	fmt.Printf("  - GOMAXPROCS: %v\n", sysInfo["gomaxprocs"])
	fmt.Printf("  - 协程数: %v\n", sysInfo["goroutines"])
	fmt.Printf("  - 内存分配: %s\n", formatBytes(sysInfo["memory_alloc"].(uint64)))
	fmt.Printf("  - 系统内存: %s\n", formatBytes(sysInfo["memory_sys"].(uint64)))
	fmt.Printf("  - 堆内存: %s\n", formatBytes(sysInfo["memory_heap_alloc"].(uint64)))
	fmt.Printf("  - GC次数: %v\n", sysInfo["memory_gc_count"])
}

// 自定义资源限制动作
func customResourceLimitActions(app *core.App) {
	// 获取资源管理器
	resourceManager := app.GetResourceManager()

	// 注册自定义资源限制动作
	customAction := resource.ResourceLimitAction("custom_action")
	resourceManager.RegisterActionHandler(customAction, func(resourceType resource.ResourceType, limit resource.ResourceLimit, usage *resource.ResourceUsage) error {
		fmt.Printf("执行自定义资源限制动作: %s - %s限制 %s\n",
			resourceType, limit.LimitType, formatBytes(limit.Value))
		return nil
	})

	// 设置使用自定义动作的资源限制
	cpuLimit := float64(80)
	fmt.Printf("设置CPU使用限制: %.2f%%, 动作: %s\n", cpuLimit, customAction)
	resourceManager.LimitCPU(cpuLimit, customAction)

	// 等待一段时间，让动作触发
	fmt.Println("等待动作触发...")
	time.Sleep(2 * time.Second)

	// 移除CPU限制
	resourceManager.RemoveLimit(resource.ResourceTypeCPU, resource.ResourceLimitTypeSoft)
}

// formatBytes 格式化字节数
func formatBytes(bytes uint64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
