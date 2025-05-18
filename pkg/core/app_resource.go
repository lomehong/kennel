package core

import (
	"os"
	"time"

	"github.com/lomehong/kennel/pkg/resource"
)

// 添加资源管理和限制到App结构体
func (app *App) initResourceManager() {
	// 获取配置
	historyLimit := app.configManager.GetInt("resource.history_limit")
	if historyLimit <= 0 {
		historyLimit = 100
	}

	updateIntervalStr := app.configManager.GetString("resource.update_interval")
	updateInterval := 5 * time.Second
	if updateIntervalStr != "" {
		if duration, err := time.ParseDuration(updateIntervalStr); err == nil {
			updateInterval = duration
		}
	}

	diskPathsStr := app.configManager.GetString("resource.disk_paths")
	diskPaths := []string{"/"}
	if diskPathsStr != "" {
		diskPaths = []string{diskPathsStr}
	}

	networkIfacesStr := app.configManager.GetString("resource.network_interfaces")
	networkIfaces := []string{}
	if networkIfacesStr != "" {
		networkIfaces = []string{networkIfacesStr}
	}

	processID := int32(os.Getpid())

	// 创建资源管理器
	app.resourceManager = resource.NewResourceManager(
		resource.WithLogger(app.logger.Named("resource-manager")),
		resource.WithHistoryLimit(historyLimit),
		resource.WithUpdateInterval(updateInterval),
		resource.WithDiskPaths(diskPaths),
		resource.WithNetworkInterfaces(networkIfaces),
		resource.WithProcessID(processID),
		resource.WithContext(app.ctx),
	)

	// 注册告警处理器
	app.resourceManager.RegisterAlertHandler(func(resourceType resource.ResourceType, message string) {
		app.logger.Warn("资源告警",
			"resource_type", resourceType,
			"message", message,
		)
	})

	// 从配置中加载资源限制
	app.loadResourceLimitsFromConfig()

	// 启动资源管理器
	app.resourceManager.Start()

	app.logger.Info("资源管理器已初始化",
		"history_limit", historyLimit,
		"update_interval", updateInterval,
		"disk_paths", diskPaths,
		"network_interfaces", networkIfaces,
	)
}

// loadResourceLimitsFromConfig 从配置中加载资源限制
func (app *App) loadResourceLimitsFromConfig() {
	// 获取CPU限制
	cpuLimitStr := app.configManager.GetString("resource.limits.cpu")
	if cpuLimitStr != "" {
		cpuLimit := app.configManager.GetFloat64("resource.limits.cpu")
		if cpuLimit <= 0 {
			cpuLimit = 80 // 默认80%
		}
		cpuAction := app.configManager.GetString("resource.limits.cpu_action")
		if cpuAction == "" {
			cpuAction = "log"
		}
		app.resourceManager.LimitCPU(cpuLimit, resource.ResourceLimitAction(cpuAction))
		app.logger.Info("已设置CPU限制",
			"limit", cpuLimit,
			"action", cpuAction,
		)
	}

	// 获取内存限制
	memoryLimitStr := app.configManager.GetString("resource.limits.memory")
	if memoryLimitStr != "" {
		memoryLimit := uint64(app.configManager.GetInt("resource.limits.memory"))
		if memoryLimit <= 0 {
			memoryLimit = 1024 * 1024 * 1024 // 默认1GB
		}
		memoryAction := app.configManager.GetString("resource.limits.memory_action")
		if memoryAction == "" {
			memoryAction = "alert"
		}
		app.resourceManager.LimitMemory(memoryLimit, resource.ResourceLimitAction(memoryAction))
		app.logger.Info("已设置内存限制",
			"limit", memoryLimit,
			"action", memoryAction,
		)
	}

	// 获取磁盘限制
	diskLimitStr := app.configManager.GetString("resource.limits.disk")
	if diskLimitStr != "" {
		diskLimit := uint64(app.configManager.GetInt("resource.limits.disk"))
		if diskLimit <= 0 {
			diskLimit = 10 * 1024 * 1024 * 1024 // 默认10GB
		}
		diskAction := app.configManager.GetString("resource.limits.disk_action")
		if diskAction == "" {
			diskAction = "alert"
		}
		app.resourceManager.LimitDisk(diskLimit, resource.ResourceLimitAction(diskAction))
		app.logger.Info("已设置磁盘限制",
			"limit", diskLimit,
			"action", diskAction,
		)
	}

	// 获取网络限制
	networkLimitStr := app.configManager.GetString("resource.limits.network")
	if networkLimitStr != "" {
		networkLimit := uint64(app.configManager.GetInt("resource.limits.network"))
		if networkLimit <= 0 {
			networkLimit = 10 * 1024 * 1024 // 默认10MB/s
		}
		networkAction := app.configManager.GetString("resource.limits.network_action")
		if networkAction == "" {
			networkAction = "throttle"
		}
		app.resourceManager.LimitNetwork(networkLimit, resource.ResourceLimitAction(networkAction))
		app.logger.Info("已设置网络限制",
			"limit", networkLimit,
			"action", networkAction,
		)
	}

	// 设置GOMAXPROCS
	gomaxprocsStr := app.configManager.GetString("resource.gomaxprocs")
	if gomaxprocsStr != "" {
		gomaxprocs := app.configManager.GetInt("resource.gomaxprocs")
		if gomaxprocs > 0 {
			app.resourceManager.SetGOMAXPROCS(gomaxprocs)
			app.logger.Info("已设置GOMAXPROCS", "value", gomaxprocs)
		}
	}

	// 设置进程优先级
	priorityStr := app.configManager.GetString("resource.process_priority")
	if priorityStr != "" {
		priority := app.configManager.GetInt("resource.process_priority")
		if err := app.resourceManager.SetProcessPriority(priority); err != nil {
			app.logger.Error("设置进程优先级失败", "error", err)
		} else {
			app.logger.Info("已设置进程优先级", "value", priority)
		}
	}
}

// GetResourceManager 获取资源管理器
func (app *App) GetResourceManager() *resource.ResourceManager {
	return app.resourceManager
}

// GetResourceManagerUsage 获取资源管理器的资源使用情况
func (app *App) GetResourceManagerUsage() *resource.ResourceUsage {
	if app.resourceManager == nil {
		return nil
	}
	return app.resourceManager.GetResourceUsage()
}

// GetResourceUsageHistory 获取资源使用历史
func (app *App) GetResourceUsageHistory() []*resource.ResourceUsage {
	if app.resourceManager == nil {
		return nil
	}
	return app.resourceManager.GetResourceUsageHistory()
}

// GetResourceStats 获取资源统计信息
func (app *App) GetResourceStats() map[string]interface{} {
	if app.resourceManager == nil {
		return nil
	}
	return app.resourceManager.GetResourceStats()
}

// GetResourceAlerts 获取资源告警
func (app *App) GetResourceAlerts() map[resource.ResourceType][]string {
	if app.resourceManager == nil {
		return nil
	}
	return app.resourceManager.GetResourceAlerts()
}

// GetResourceLimits 获取资源限制
func (app *App) GetResourceLimits() map[resource.ResourceType][]resource.ResourceLimit {
	if app.resourceManager == nil {
		return nil
	}
	return app.resourceManager.GetResourceLimits()
}

// LimitCPU 限制CPU使用
func (app *App) LimitCPU(percent float64, action resource.ResourceLimitAction) {
	if app.resourceManager == nil {
		return
	}
	app.resourceManager.LimitCPU(percent, action)
}

// LimitMemory 限制内存使用
func (app *App) LimitMemory(bytes uint64, action resource.ResourceLimitAction) {
	if app.resourceManager == nil {
		return
	}
	app.resourceManager.LimitMemory(bytes, action)
}

// LimitDisk 限制磁盘使用
func (app *App) LimitDisk(bytes uint64, action resource.ResourceLimitAction) {
	if app.resourceManager == nil {
		return
	}
	app.resourceManager.LimitDisk(bytes, action)
}

// LimitNetwork 限制网络使用
func (app *App) LimitNetwork(bytesPerSecond uint64, action resource.ResourceLimitAction) {
	if app.resourceManager == nil {
		return
	}
	app.resourceManager.LimitNetwork(bytesPerSecond, action)
}

// RemoveResourceLimit 移除资源限制
func (app *App) RemoveResourceLimit(resourceType resource.ResourceType, limitType resource.ResourceLimitType) {
	if app.resourceManager == nil {
		return
	}
	app.resourceManager.RemoveLimit(resourceType, limitType)
}

// SetProcessPriority 设置进程优先级
func (app *App) SetProcessPriority(priority int) error {
	if app.resourceManager == nil {
		return nil
	}
	return app.resourceManager.SetProcessPriority(priority)
}

// SetGOMAXPROCS 设置GOMAXPROCS
func (app *App) SetGOMAXPROCS(n int) {
	if app.resourceManager == nil {
		return
	}
	app.resourceManager.SetGOMAXPROCS(n)
}

// OptimizeResourceUsage 优化资源使用
func (app *App) OptimizeResourceUsage() {
	if app.resourceManager == nil {
		return
	}
	app.resourceManager.OptimizeResourceUsage()
}

// RegisterResourceAlertHandler 注册资源告警处理器
func (app *App) RegisterResourceAlertHandler(handler resource.ResourceAlertHandler) {
	if app.resourceManager == nil {
		return
	}
	app.resourceManager.RegisterAlertHandler(handler)
}

// RegisterResourceActionHandler 注册资源动作处理器
func (app *App) RegisterResourceActionHandler(action resource.ResourceLimitAction, handler resource.ResourceActionHandler) {
	if app.resourceManager == nil {
		return
	}
	app.resourceManager.RegisterActionHandler(action, handler)
}

// GetProcessInfo 获取进程信息
func (app *App) GetProcessInfo() (map[string]interface{}, error) {
	if app.resourceManager == nil {
		return nil, nil
	}
	return app.resourceManager.GetProcessInfo()
}

// GetSystemInfo 获取系统信息
func (app *App) GetSystemInfo() map[string]interface{} {
	if app.resourceManager == nil {
		return nil
	}
	return app.resourceManager.GetSystemInfo()
}
