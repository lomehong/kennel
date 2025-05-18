package core

import (
	"context"
	"os"
	"time"

	"github.com/lomehong/kennel/pkg/health"
)

// 添加健康检查和自我修复到App结构体
func (app *App) initHealthSystem() {
	// 创建健康检查注册表
	app.healthRegistry = health.NewCheckerRegistry(app.logger.Named("health-registry"))

	// 创建自我修复器
	app.selfHealer = health.NewRepairSelfHealer(app.healthRegistry, app.logger.Named("self-healer"))

	// 创建监控配置
	monitorConfig := &health.MonitorConfig{
		CheckInterval:    app.configManager.GetDurationOrDefault("health.check_interval", 30*time.Second),
		InitialDelay:     app.configManager.GetDurationOrDefault("health.initial_delay", 5*time.Second),
		AutoRepair:       app.configManager.GetBool("health.auto_repair"),
		FailureThreshold: app.configManager.GetIntOrDefault("health.failure_threshold", 3),
		SuccessThreshold: app.configManager.GetIntOrDefault("health.success_threshold", 1),
	}

	// 创建健康监控器
	app.healthMonitor = health.NewHealthMonitor(app.healthRegistry, app.selfHealer, monitorConfig, app.logger.Named("health-monitor"))

	// 注册默认健康检查器
	app.registerDefaultHealthCheckers()

	// 注册默认修复策略
	app.registerDefaultRepairStrategies()

	app.logger.Info("健康检查和自我修复系统已初始化")
}

// registerDefaultHealthCheckers 注册默认健康检查器
func (app *App) registerDefaultHealthCheckers() {
	// 注册内存检查器
	memoryThreshold := app.configManager.GetFloat64OrDefault("health.memory_threshold", 80.0)
	app.healthRegistry.RegisterChecker(health.NewMemoryChecker(memoryThreshold))

	// 注册CPU检查器
	cpuThreshold := app.configManager.GetFloat64OrDefault("health.cpu_threshold", 70.0)
	app.healthRegistry.RegisterChecker(health.NewCPUChecker(cpuThreshold))

	// 注册磁盘检查器
	diskPath := app.configManager.GetString("health.disk_path")
	if diskPath == "" {
		// 获取当前工作目录
		wd, err := os.Getwd()
		if err == nil {
			diskPath = wd
		} else {
			diskPath = "/"
		}
	}
	diskThreshold := app.configManager.GetFloat64OrDefault("health.disk_threshold", 90.0)
	app.healthRegistry.RegisterChecker(health.NewDiskChecker(diskPath, diskThreshold))

	// 注册Goroutine检查器
	goroutineThreshold := app.configManager.GetIntOrDefault("health.goroutine_threshold", 1000)
	app.healthRegistry.RegisterChecker(health.NewGoroutineChecker(goroutineThreshold))

	// 注册进程检查器
	app.healthRegistry.RegisterChecker(health.NewProcessChecker(os.Getpid()))

	// 注册应用程序自定义检查器
	app.registerAppHealthChecker()
}

// registerAppHealthChecker 注册应用程序自定义健康检查器
func (app *App) registerAppHealthChecker() {
	// 创建应用程序健康检查器
	appChecker := health.NewSimpleChecker(
		"app",
		"应用程序健康检查",
		"app",
		func(ctx context.Context) health.CheckResult {
			// 检查应用程序是否正在运行
			if !app.running {
				return health.CheckResult{
					Status:  health.StatusStopped,
					Message: "应用程序已停止",
					Details: map[string]interface{}{
						"running": false,
					},
				}
			}

			// 检查应用程序组件
			details := map[string]interface{}{
				"running":     true,
				"start_time":  app.startTime,
				"uptime":      time.Since(app.startTime).String(),
				"version":     app.version,
				"config_file": app.configManager.GetConfigPath(),
			}

			// 返回健康状态
			return health.CheckResult{
				Status:  health.StatusHealthy,
				Message: "应用程序运行正常",
				Details: details,
			}
		},
	)

	// 注册检查器
	app.healthRegistry.RegisterChecker(appChecker)
}

// registerDefaultRepairStrategies 注册默认修复策略
func (app *App) registerDefaultRepairStrategies() {
	// 注册内存修复策略
	memoryRepairStrategy := health.NewSimpleRepairStrategy(
		"memory_repair_strategy",
		func(result health.CheckResult) bool {
			return result.Status == health.StatusUnhealthy
		},
		func(result health.CheckResult) health.RepairAction {
			return health.NewFreeMemoryAction()
		},
	)
	app.selfHealer.RegisterStrategy("memory", memoryRepairStrategy)

	// 注册应用程序修复策略
	appRepairStrategy := health.NewSimpleRepairStrategy(
		"app_repair_strategy",
		func(result health.CheckResult) bool {
			return result.Status == health.StatusUnhealthy || result.Status == health.StatusStopped
		},
		func(result health.CheckResult) health.RepairAction {
			return health.NewRestartComponentAction("app",
				func(ctx context.Context) error {
					app.logger.Info("停止应用程序组件")
					return nil
				},
				func(ctx context.Context) error {
					app.logger.Info("启动应用程序组件")
					return nil
				},
			)
		},
	)
	app.selfHealer.RegisterStrategy("app", appRepairStrategy)
}

// StartHealthMonitor 启动健康监控
func (app *App) StartHealthMonitor() {
	app.logger.Info("启动健康监控")
	app.healthMonitor.Start()
}

// StopHealthMonitor 停止健康监控
func (app *App) StopHealthMonitor() {
	app.logger.Info("停止健康监控")
	app.healthMonitor.Stop()
}

// GetHealthRegistry 获取健康检查注册表
func (app *App) GetHealthRegistry() *health.CheckerRegistry {
	return app.healthRegistry
}

// GetSelfHealer 获取自我修复器
func (app *App) GetSelfHealer() *health.RepairSelfHealer {
	return app.selfHealer
}

// GetHealthMonitor 获取健康监控器
func (app *App) GetHealthMonitor() *health.HealthMonitor {
	return app.healthMonitor
}

// RegisterHealthChecker 注册健康检查器
func (app *App) RegisterHealthChecker(checker health.Checker) {
	app.healthRegistry.RegisterChecker(checker)
}

// UnregisterHealthChecker 注销健康检查器
func (app *App) UnregisterHealthChecker(name string) {
	app.healthRegistry.UnregisterChecker(name)
}

// RegisterRepairStrategy 注册修复策略
func (app *App) RegisterRepairStrategy(checkerName string, strategy health.RepairStrategy) {
	app.selfHealer.RegisterStrategy(checkerName, strategy)
}

// UnregisterRepairStrategy 注销修复策略
func (app *App) UnregisterRepairStrategy(checkerName string) {
	app.selfHealer.UnregisterStrategy(checkerName)
}

// CheckHealth 检查健康状态
func (app *App) CheckHealth(ctx context.Context) health.CheckResult {
	return app.healthRegistry.GetSystemStatus(ctx)
}

// CheckAndRepair 检查并修复
func (app *App) CheckAndRepair(ctx context.Context, checkerName string) (health.CheckResult, *health.RepairResult, error) {
	return app.selfHealer.CheckAndRepair(ctx, checkerName)
}

// CheckAndRepairAll 检查并修复所有
func (app *App) CheckAndRepairAll(ctx context.Context) (map[string]health.CheckResult, map[string]*health.RepairResult, error) {
	return app.selfHealer.CheckAndRepairAll(ctx)
}

// GetHealthStatus 获取健康状态
func (app *App) GetHealthStatus() map[string]*health.CheckerStatus {
	return app.healthMonitor.GetAllStatus()
}

// GetSystemHealth 获取系统健康状态
func (app *App) GetSystemHealth() health.Status {
	return app.healthMonitor.GetSystemHealth()
}

// GetRepairHistory 获取修复历史
func (app *App) GetRepairHistory() []health.RepairResult {
	return app.selfHealer.GetRepairHistory()
}
