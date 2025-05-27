package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/core/selfprotect"
)

// Application 应用程序结构
type Application struct {
	logger              hclog.Logger
	protectionIntegrator *selfprotect.ProtectionIntegrator
	protectionService   *selfprotect.ProtectionService
	healthChecker       *selfprotect.ProtectionHealthChecker
	reporter            *selfprotect.ProtectionReporter
	ctx                 context.Context
	cancel              context.CancelFunc
}

// NewApplication 创建应用程序
func NewApplication() (*Application, error) {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "kennel-agent",
		Level: hclog.Info,
		Output: os.Stdout,
	})

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	app := &Application{
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// 初始化自我防护
	if err := app.initializeProtection(); err != nil {
		return nil, fmt.Errorf("初始化自我防护失败: %w", err)
	}

	return app, nil
}

// initializeProtection 初始化自我防护
func (app *Application) initializeProtection() error {
	app.logger.Info("初始化自我防护机制")

	// 创建防护集成器
	integrator, err := selfprotect.NewProtectionIntegrator("config.yaml", app.logger)
	if err != nil {
		return fmt.Errorf("创建防护集成器失败: %w", err)
	}

	app.protectionIntegrator = integrator
	app.protectionService = integrator.GetService()

	// 创建健康检查器
	app.healthChecker = selfprotect.NewProtectionHealthChecker(app.protectionService, app.logger)

	// 创建报告器
	app.reporter = selfprotect.NewProtectionReporter(app.protectionService, app.logger)

	return nil
}

// Start 启动应用程序
func (app *Application) Start() error {
	app.logger.Info("启动Kennel Agent")

	// 启动自我防护
	if err := app.protectionIntegrator.Initialize(); err != nil {
		app.logger.Error("启动自我防护失败", "error", err)
		// 注意：自我防护失败不应该阻止程序启动
		// 但应该记录错误并可能降级运行
	}

	// 启动健康检查
	go app.runHealthCheck()

	// 启动状态报告
	go app.runStatusReport()

	// 模拟主程序业务逻辑
	go app.runBusinessLogic()

	app.logger.Info("Kennel Agent已启动")
	return nil
}

// Stop 停止应用程序
func (app *Application) Stop() {
	app.logger.Info("停止Kennel Agent")

	// 取消上下文
	app.cancel()

	// 关闭自我防护
	if app.protectionIntegrator != nil {
		app.protectionIntegrator.Shutdown()
	}

	app.logger.Info("Kennel Agent已停止")
}

// runHealthCheck 运行健康检查
func (app *Application) runHealthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			return
		case <-ticker.C:
			result := app.healthChecker.CheckHealth()
			
			switch result.Status {
			case "healthy":
				app.logger.Debug("自我防护健康检查通过", "message", result.Message)
			case "disabled":
				app.logger.Info("自我防护已禁用", "message", result.Message)
			case "unhealthy":
				app.logger.Warn("自我防护健康检查失败", "message", result.Message, "details", result.Details)
			}
		}
	}
}

// runStatusReport 运行状态报告
func (app *Application) runStatusReport() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			return
		case <-ticker.C:
			app.generateAndLogReport()
		}
	}
}

// generateAndLogReport 生成并记录报告
func (app *Application) generateAndLogReport() {
	report := app.reporter.GenerateReport()
	
	app.logger.Info("自我防护状态报告",
		"enabled", report.Status.Enabled,
		"level", report.Status.Level,
		"total_events", report.TotalEvents,
		"recent_events", report.RecentEvents,
		"health_score", report.Status.Stats.ConfigHealthScore,
		"recommendations", len(report.Recommendations),
	)

	// 如果有建议，记录详细信息
	if len(report.Recommendations) > 0 {
		app.logger.Info("自我防护建议", "recommendations", report.Recommendations)
	}

	// 如果有最近的事件，记录事件信息
	if len(report.Events) > 0 {
		app.logger.Info("最近的防护事件", "count", len(report.Events))
		for _, event := range report.Events {
			app.logger.Debug("防护事件",
				"type", event.Type,
				"action", event.Action,
				"target", event.Target,
				"blocked", event.Blocked,
				"timestamp", event.Timestamp,
			)
		}
	}
}

// runBusinessLogic 运行业务逻辑
func (app *Application) runBusinessLogic() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			return
		case <-ticker.C:
			// 模拟业务逻辑
			app.logger.Debug("执行业务逻辑")
			
			// 检查自我防护状态
			if app.protectionService.IsEnabled() {
				status := app.protectionService.GetStatus()
				app.logger.Debug("自我防护状态",
					"enabled", status.Enabled,
					"level", status.Level,
					"uptime", time.Since(status.StartTime).String(),
				)
			}
		}
	}
}

// handleSignals 处理系统信号
func (app *Application) handleSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		app.logger.Info("收到系统信号", "signal", sig)
		
		// 生成最终报告
		app.logger.Info("生成最终防护报告")
		app.generateAndLogReport()
		
		// 停止应用程序
		app.Stop()
		os.Exit(0)
	}()
}

// demonstrateProtectionFeatures 演示防护功能
func (app *Application) demonstrateProtectionFeatures() {
	app.logger.Info("演示自我防护功能")

	// 1. 显示当前配置
	config := app.protectionService.GetConfig()
	app.logger.Info("当前防护配置",
		"enabled", config.Enabled,
		"level", config.Level,
		"process_protection", config.ProcessProtection.Enabled,
		"file_protection", config.FileProtection.Enabled,
		"registry_protection", config.RegistryProtection.Enabled,
		"service_protection", config.ServiceProtection.Enabled,
	)

	// 2. 显示防护状态
	status := app.protectionService.GetStatus()
	app.logger.Info("当前防护状态",
		"enabled", status.Enabled,
		"level", status.Level,
		"start_time", status.StartTime,
		"total_events", status.Stats.TotalEvents,
		"health_score", status.Stats.ConfigHealthScore,
	)

	// 3. 显示防护事件
	events := app.protectionService.GetEvents()
	app.logger.Info("防护事件统计", "total", len(events))

	// 4. 执行健康检查
	healthResult := app.healthChecker.CheckHealth()
	app.logger.Info("健康检查结果",
		"status", healthResult.Status,
		"message", healthResult.Message,
	)

	// 5. 生成防护报告
	report := app.reporter.GenerateReport()
	app.logger.Info("防护报告摘要",
		"total_events", report.TotalEvents,
		"recent_events", report.RecentEvents,
		"recommendations", len(report.Recommendations),
	)
}

func main() {
	// 创建应用程序
	app, err := NewApplication()
	if err != nil {
		log.Fatalf("创建应用程序失败: %v", err)
	}

	// 设置信号处理
	app.handleSignals()

	// 启动应用程序
	if err := app.Start(); err != nil {
		log.Fatalf("启动应用程序失败: %v", err)
	}

	// 演示防护功能
	time.Sleep(2 * time.Second)
	app.demonstrateProtectionFeatures()

	// 保持程序运行
	app.logger.Info("应用程序正在运行，按Ctrl+C退出")
	select {
	case <-app.ctx.Done():
		return
	}
}
