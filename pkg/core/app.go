package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/concurrency"
	"github.com/lomehong/kennel/pkg/config"
	"github.com/lomehong/kennel/pkg/errors"
	"github.com/lomehong/kennel/pkg/events"
	"github.com/lomehong/kennel/pkg/health"
	"github.com/lomehong/kennel/pkg/interfaces"
	"github.com/lomehong/kennel/pkg/logger"
	"github.com/lomehong/kennel/pkg/logging"
	"github.com/lomehong/kennel/pkg/plugin"
	"github.com/lomehong/kennel/pkg/resource"
	"github.com/lomehong/kennel/pkg/system"
	"github.com/lomehong/kennel/pkg/timeout"
	"github.com/lomehong/kennel/pkg/webconsole"
)

// App 是应用程序的核心
type App struct {
	// 配置管理器
	configManager *ConfigManager

	// 配置监视器
	configWatcher *config.ConfigWatcher

	// 动态配置
	dynamicConfig *config.DynamicConfig

	// 插件管理器
	pluginManager *plugin.PluginManager

	// 插件注册表
	pluginRegistry *PluginRegistry

	// 通讯管理器
	commManager *CommManager

	// Web控制台
	webConsole interfaces.WebConsoleInterface

	// Web控制台工厂
	webConsoleFactory interfaces.WebConsoleFactory

	// 日志
	logger hclog.Logger

	// 原始日志记录器
	originalLogger hclog.Logger

	// 增强日志记录器
	enhancedLogger logging.Logger

	// 是否正在运行
	running bool

	// 启动时间
	startTime time.Time

	// 版本
	version string

	// 资源追踪器
	resourceTracker *resource.ResourceTracker

	// 资源管理器
	resourceManager *resource.ResourceManager

	// 超时控制器
	timeoutController *timeout.TimeoutController

	// 并发控制器
	concurrencyController *concurrency.ConcurrencyController

	// 错误处理器注册表
	errorRegistry *errors.ErrorHandlerRegistry

	// 恢复管理器
	recoveryManager *errors.RecoveryManager

	// 健康检查注册表
	healthRegistry *health.CheckerRegistry

	// 自我修复器
	selfHealer *health.RepairSelfHealer

	// 健康监控器
	healthMonitor *health.HealthMonitor

	// 系统监控器
	systemMonitor *system.Monitor

	// 系统指标收集器
	metricsCollector *system.MetricsCollector

	// 日志管理器
	logManager *logging.LogManager

	// 事件管理器
	eventManager *events.EventManager

	// 上下文和取消函数
	ctx    context.Context
	cancel context.CancelFunc
}

// NewApp 创建一个新的应用程序实例
func NewApp(configFile string) *App {
	// 创建日志
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "app",
		Output: os.Stdout,
		Level:  hclog.Debug,
	})

	// 创建配置管理器
	configManager := NewConfigManager(configFile)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建资源追踪器
	resourceTracker := resource.NewResourceTracker(
		resource.WithTrackerLogger(logger.Named("resource-tracker")),
		resource.WithCleanupInterval(5*time.Minute),
	)

	// 创建应用程序实例
	app := &App{
		configManager:     configManager,
		logger:            logger,
		startTime:         time.Now(),
		version:           "1.0.0", // 设置版本号
		webConsoleFactory: webconsole.NewFactory(),
		resourceTracker:   resourceTracker,
		ctx:               ctx,
		cancel:            cancel,
	}

	// 创建插件注册表
	pluginDir := configManager.GetString("plugin_dir")
	if pluginDir == "" {
		pluginDir = "app"
	}
	app.pluginRegistry = NewPluginRegistry(logger, pluginDir)

	// 初始化超时控制器
	app.initTimeoutController()

	// 初始化并发控制器
	app.initConcurrencyController()

	// 初始化错误处理和Panic恢复
	app.initErrorHandling()

	// 初始化插件管理器
	app.initPluginManager()

	// 初始化健康检查和自我修复系统
	app.initHealthSystem()

	// 初始化资源管理器
	app.initResourceManager()

	// 初始化配置热更新和动态配置系统
	app.initConfigSystem()

	// 初始化日志增强和结构化日志系统
	app.initLoggingSystem()

	// 初始化系统监控器
	app.initSystemMonitor()

	// 初始化日志管理器
	app.initLogManager()

	// 初始化事件管理器
	app.initEventManager()

	return app
}

// Init 初始化应用程序
func (app *App) Init() error {
	app.logger.Info("初始化应用程序")

	// 初始化配置
	if err := app.configManager.InitConfig(); err != nil {
		app.logger.Error("初始化配置失败", "error", err)
		return fmt.Errorf("初始化配置失败: %w", err)
	}

	// 获取插件目录
	pluginDir := app.configManager.GetString("plugin_dir")
	if !filepath.IsAbs(pluginDir) {
		// 如果是相对路径，转换为绝对路径
		execDir, err := os.Executable()
		if err != nil {
			app.logger.Error("无法获取可执行文件路径", "error", err)
			return fmt.Errorf("无法获取可执行文件路径: %w", err)
		}

		pluginDir = filepath.Join(filepath.Dir(execDir), pluginDir)
	}

	// 创建插件管理器
	app.pluginManager = plugin.NewPluginManager(
		plugin.WithPluginManagerLogger(app.logger.Named("plugin-manager")),
		plugin.WithPluginsDir(pluginDir),
		plugin.WithPluginManagerResourceTracker(app.resourceTracker),
		plugin.WithPluginManagerWorkerPool(app.GetDefaultWorkerPool()),
		plugin.WithPluginManagerErrorRegistry(app.errorRegistry),
		plugin.WithPluginManagerRecoveryManager(app.recoveryManager),
		plugin.WithPluginManagerContext(app.ctx),
	)

	// 加载插件
	app.logger.Info("在初始化阶段加载插件")
	if err := app.LoadPluginsFromConfig(); err != nil {
		app.logger.Error("加载插件失败", "error", err)
		// 不返回错误，继续初始化过程
		app.logger.Warn("插件加载失败，但应用程序将继续初始化")
	} else {
		// 检查是否加载了任何插件
		loadedPlugins := app.pluginManager.ListPlugins()
		if len(loadedPlugins) == 0 {
			app.logger.Info("没有从配置中加载任何插件，尝试加载自动发现的插件")
			if err := app.loadDiscoveredPlugins(); err != nil {
				app.logger.Error("加载自动发现的插件失败", "error", err)
			} else {
				app.logger.Info("成功加载自动发现的插件")
			}
		} else {
			app.logger.Info("成功加载插件", "count", len(loadedPlugins))
		}
	}

	// 创建通讯管理器
	app.commManager = NewCommManager(app.configManager)

	// 插件管理器和通讯管理器的集成在实际应用中可能需要实现

	// 初始化通讯管理器
	if err := app.commManager.Init(); err != nil {
		app.logger.Error("初始化通讯管理器失败", "error", err)
		return fmt.Errorf("初始化通讯管理器失败: %w", err)
	}

	// 初始化Web控制台
	if app.configManager.GetBool("web_console.enabled") {
		app.logger.Info("初始化Web控制台")

		// 创建Web控制台配置适配器
		configAdapter := &webconsole.ConfigAdapter{
			Enabled:      app.configManager.GetBool("web_console.enabled"),
			Host:         app.configManager.GetString("web_console.host"),
			Port:         app.configManager.GetInt("web_console.port"),
			EnableHTTPS:  app.configManager.GetBool("web_console.enable_https"),
			CertFile:     app.configManager.GetString("web_console.cert_file"),
			KeyFile:      app.configManager.GetString("web_console.key_file"),
			EnableAuth:   app.configManager.GetBool("web_console.enable_auth"),
			Username:     app.configManager.GetString("web_console.username"),
			Password:     app.configManager.GetString("web_console.password"),
			StaticDir:    app.configManager.GetString("web_console.static_dir"),
			LogLevel:     app.configManager.GetString("web_console.log_level"),
			RateLimit:    app.configManager.GetInt("web_console.rate_limit"),
			EnableCSRF:   app.configManager.GetBool("web_console.enable_csrf"),
			APIPrefix:    app.configManager.GetString("web_console.api_prefix"),
			AllowOrigins: []string{"*"}, // 默认允许所有来源
		}

		// 如果RateLimit未设置，使用默认值100
		if configAdapter.RateLimit <= 0 {
			configAdapter.RateLimit = 100
		}

		// 如果APIPrefix未设置，使用默认值"/api"
		if configAdapter.APIPrefix == "" {
			configAdapter.APIPrefix = "/api"
		}

		// 确保静态文件目录是绝对路径
		if configAdapter.StaticDir != "" && !filepath.IsAbs(configAdapter.StaticDir) {
			// 获取当前工作目录
			workDir, err := os.Getwd()
			if err == nil {
				// 转换为绝对路径
				configAdapter.StaticDir = filepath.Join(workDir, configAdapter.StaticDir)
				app.logger.Debug("静态文件目录转换为绝对路径", "path", configAdapter.StaticDir)
			}
		}

		// 解析会话超时时间
		sessionTimeoutStr := app.configManager.GetString("web_console.session_timeout")
		if sessionTimeoutStr != "" {
			if timeout, err := time.ParseDuration(sessionTimeoutStr); err == nil {
				configAdapter.SessionTimeout = timeout
			}
		}

		// 创建App接口适配器
		appAdapter := NewAppInterfaceAdapter(app)

		// 使用工厂创建Web控制台
		console, err := app.webConsoleFactory.CreateWebConsole(configAdapter, appAdapter)
		if err != nil {
			app.logger.Error("创建Web控制台失败", "error", err)
			return fmt.Errorf("创建Web控制台失败: %w", err)
		}

		// 初始化Web控制台
		if err := console.Init(); err != nil {
			app.logger.Error("初始化Web控制台失败", "error", err)
			return fmt.Errorf("初始化Web控制台失败: %w", err)
		}

		app.webConsole = console
	}

	// 记录系统初始化完成事件
	if app.eventManager != nil {
		app.eventManager.PublishEvent(events.Event{
			Type:    "system.initialized",
			Message: "系统初始化完成",
			Source:  "app",
			Data: map[string]interface{}{
				"version": app.version,
				"modules": app.pluginManager.ListPlugins(),
			},
		})
	}

	app.logger.Info("应用程序初始化完成")
	return nil
}

// Start 启动应用程序
func (app *App) Start() error {
	app.logger.Info("启动应用程序")

	// 设置运行状态
	app.running = true

	// 记录系统启动事件
	if app.eventManager != nil {
		app.eventManager.PublishEvent(events.Event{
			Type:    "system.startup",
			Message: "系统正在启动",
			Source:  "app",
			Data: map[string]interface{}{
				"version": app.version,
			},
		})
	}

	// 记录系统启动日志
	if app.logManager != nil {
		app.logManager.Log("info", "系统正在启动", "app", map[string]interface{}{
			"version": app.version,
			"uptime":  "0s",
		})
	}

	// 处理信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		app.logger.Info("收到信号", "signal", sig)
		app.Stop()
	}()

	// 并发初始化模块
	var wg sync.WaitGroup

	// 插件已在初始化阶段加载，这里不再重复加载
	app.logger.Info("插件已在初始化阶段加载，启动阶段不再重复加载")

	// 等待所有模块初始化完成
	wg.Wait()

	// 连接到服务器（如果启用了通讯功能）
	if app.configManager.GetBool("enable_comm") && app.configManager.GetBool("comm.enabled") {
		app.logger.Info("启用通讯功能，连接到服务器")
		if err := app.commManager.Connect(); err != nil {
			app.logger.Error("连接服务器失败", "error", err)
			// 不返回错误，继续运行应用程序
		}
	} else {
		if app.configManager.GetBool("enable_comm") {
			app.logger.Info("通讯功能已在配置中禁用 (comm.enabled=false)")
		} else {
			app.logger.Info("通讯功能已全局禁用 (enable_comm=false)")
		}
	}

	// 启动Web控制台（如果已初始化）
	if app.webConsole != nil {
		app.logger.Info("启动Web控制台")
		fmt.Println("正在启动Web控制台...")
		if err := app.webConsole.Start(); err != nil {
			app.logger.Error("启动Web控制台失败", "error", err)
			fmt.Println("启动Web控制台失败:", err)
			// 不返回错误，继续运行应用程序
		} else {
			fmt.Println("Web控制台已启动，可以通过浏览器访问")
		}
	}

	app.logger.Info("应用程序已启动")
	return nil
}

// 用于防止Stop方法被并发调用
var stopMutex sync.Mutex
var stopInProgress bool

// Stop 停止应用程序，支持优雅终止
func (app *App) Stop() {
	// 使用互斥锁确保Stop方法不会被并发调用
	stopMutex.Lock()
	defer stopMutex.Unlock()

	// 检查是否已经在停止过程中
	if stopInProgress {
		app.logger.Warn("应用程序已经在停止过程中，跳过重复调用")
		return
	}

	// 检查应用程序是否正在运行
	if !app.running {
		app.logger.Warn("应用程序未在运行，无需停止")
		return
	}

	// 标记为正在停止
	stopInProgress = true

	app.logger.Info("开始优雅终止应用程序")

	// 取消应用程序上下文
	if app.cancel != nil {
		app.logger.Debug("取消应用程序上下文")
		app.cancel()
	}

	// 创建一个通道，用于等待所有清理工作完成
	done := make(chan struct{})

	// 在后台执行清理工作
	go func() {
		// 获取终止超时时间（默认为30秒）
		shutdownTimeout := 30 * time.Second
		timeoutStr := app.configManager.GetString("shutdown_timeout")
		if timeoutStr != "" {
			if t, err := time.ParseDuration(timeoutStr); err == nil {
				shutdownTimeout = t
			}
		} else if timeoutInt := app.configManager.GetInt("shutdown_timeout"); timeoutInt > 0 {
			shutdownTimeout = time.Duration(timeoutInt) * time.Second
		}

		app.logger.Info("优雅终止超时设置", "timeout", shutdownTimeout)

		// 创建一个上下文，用于控制终止超时
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		// 停止Web控制台
		if app.webConsole != nil {
			app.logger.Info("正在停止Web控制台...")
			if err := app.webConsole.Stop(ctx); err != nil {
				app.logger.Error("停止Web控制台失败", "error", err)
			} else {
				app.logger.Info("Web控制台已停止")
			}
		}

		// 创建一个通道，用于等待插件关闭完成
		pluginsDone := make(chan struct{})

		// 在后台关闭所有插件
		go func() {
			app.logger.Info("开始关闭所有插件")
			// 停止所有插件
			plugins := app.pluginManager.ListPlugins()
			for _, p := range plugins {
				app.logger.Info("停止插件", "id", p.ID)
				app.pluginManager.StopPlugin(p.ID)
			}
			close(pluginsDone)
		}()

		// 等待插件关闭完成或超时
		select {
		case <-ctx.Done():
			app.logger.Warn("插件关闭超时，强制终止")
		case <-pluginsDone:
			app.logger.Info("所有插件已正常关闭")
		}

		// 执行其他清理工作
		app.logger.Info("执行其他清理工作")

		// 断开与服务器的连接
		if app.commManager != nil && app.commManager.IsConnected() {
			app.logger.Info("正在断开与服务器的连接...")

			// 获取超时时间
			timeout := 5 * time.Second
			if timeoutSec := app.configManager.GetInt("comm_shutdown_timeout"); timeoutSec > 0 {
				timeout = time.Duration(timeoutSec) * time.Second
			}

			// 设置超时上下文
			disconnectCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			app.logger.Info("等待通讯模块关闭", "timeout", timeout)

			// 创建完成通道
			done := make(chan struct{})

			// 在新的goroutine中断开连接
			go func() {
				app.commManager.Disconnect()
				close(done)
			}()

			// 等待断开连接完成或超时
			select {
			case <-done:
				app.logger.Info("已断开与服务器的连接")
			case <-disconnectCtx.Done():
				app.logger.Warn("断开连接超时")
			}
		}

		// 停止资源追踪器并释放所有资源
		if app.resourceTracker != nil {
			app.logger.Info("停止资源追踪器")
			errors := app.resourceTracker.ReleaseAll()
			if len(errors) > 0 {
				for _, err := range errors {
					app.logger.Error("释放资源失败", "error", err)
				}
			}
			app.resourceTracker.Stop()
		}

		// 停止系统指标收集器
		if app.metricsCollector != nil {
			app.logger.Info("停止系统指标收集器")
			app.metricsCollector.Stop()
		}

		// 记录关闭事件
		if app.eventManager != nil {
			app.logger.Info("记录系统关闭事件")
			app.eventManager.PublishEvent(events.Event{
				Type:    "system.shutdown",
				Message: "系统正在关闭",
				Source:  "app",
				Data: map[string]interface{}{
					"uptime": time.Since(app.startTime).String(),
				},
			})
		}

		// 记录系统关闭日志
		if app.logManager != nil {
			app.logger.Info("记录系统关闭日志")
			app.logManager.Log("info", "系统正在关闭", "app", map[string]interface{}{
				"uptime": time.Since(app.startTime).String(),
			})
		}

		// 设置运行状态
		app.running = false

		// 通知主线程清理工作已完成
		close(done)
	}()

	// 等待清理工作完成
	<-done
	app.logger.Info("应用程序已优雅终止")

	// 重置停止标志，允许再次调用Stop（虽然通常不需要）
	stopInProgress = false
}

// IsRunning 检查应用程序是否正在运行
func (app *App) IsRunning() bool {
	return app.running
}

// GetPluginManager 获取插件管理器
func (app *App) GetPluginManager() *plugin.PluginManager {
	return app.pluginManager
}

// GetConfigManager 获取配置管理器
func (app *App) GetConfigManager() *ConfigManager {
	return app.configManager
}

// GetCommManager 获取通讯管理器
func (app *App) GetCommManager() *CommManager {
	return app.commManager
}

// GetWebConsole 获取Web控制台
func (app *App) GetWebConsole() interfaces.WebConsoleInterface {
	return app.webConsole
}

// GetStartTime 获取应用程序启动时间
func (app *App) GetStartTime() time.Time {
	return app.startTime
}

// GetVersion 获取应用程序版本
func (app *App) GetVersion() string {
	return app.version
}

// SetVersion 设置应用程序版本
func (app *App) SetVersion(version string) {
	app.version = version
}

// GetLogManager 获取日志管理器
func (app *App) GetLogManager() interfaces.LogManagerInterface {
	return app.logManager
}

// GetEventManager 获取事件管理器
func (app *App) GetEventManager() interfaces.EventManagerInterface {
	return app.eventManager
}

// GetResourceTracker 获取资源追踪器
func (app *App) GetResourceTracker() *resource.ResourceTracker {
	return app.resourceTracker
}

// GetContext 获取应用程序上下文
func (app *App) GetContext() context.Context {
	return app.ctx
}

// TrackResource 追踪资源
func (app *App) TrackResource(res resource.Resource) {
	if app.resourceTracker != nil {
		app.resourceTracker.Track(res)
	}
}

// ReleaseResource 释放资源
func (app *App) ReleaseResource(id string) error {
	if app.resourceTracker != nil {
		return app.resourceTracker.Release(id)
	}
	return fmt.Errorf("资源追踪器未初始化")
}

// GetContextResourceTracker 获取与上下文关联的资源追踪器
func (app *App) GetContextResourceTracker() *resource.ContextResourceTracker {
	if app.resourceTracker != nil {
		return resource.WithTrackerContext(app.ctx, app.resourceTracker)
	}
	return nil
}

// 已移动到 app_plugins.go

// 已删除模拟模块实现，使用真实的插件模块

// RegisterModule 注册模块
func (app *App) RegisterModule(name string, module interface{}) {
	app.logger.Info("注册模块", "name", name)

	// 检查模块是否实现了 plugin.Module 接口
	if moduleImpl, ok := module.(plugin.Module); ok {
		// 创建插件配置
		config := &plugin.PluginConfig{
			ID:      name,
			Name:    moduleImpl.GetInfo().Name,
			Version: moduleImpl.GetInfo().Version,
			Enabled: true,
		}

		// 注册到插件管理器
		_, err := app.pluginManager.LoadPlugin(config)
		if err != nil {
			app.logger.Error("注册模块失败", "name", name, "error", err)
		} else {
			app.logger.Info("模块注册成功", "name", name)
		}
	} else {
		app.logger.Warn("模块未实现 plugin.Module 接口", "name", name)
	}
}

// initSystemMonitor 初始化系统监控器
func (app *App) initSystemMonitor() {
	// 创建系统监控器
	systemLogger := logger.NewLogger("system-monitor", logger.GetLogLevel("info"))
	app.systemMonitor = system.NewMonitor(systemLogger)

	// 创建系统指标收集器
	metricsLogger := logger.NewLogger("system-metrics", logger.GetLogLevel("info"))
	app.metricsCollector = system.NewMetricsCollector(metricsLogger, 5*time.Second)

	// 启动系统指标收集
	app.metricsCollector.Start()
}

// initLogManager 初始化日志管理器
func (app *App) initLogManager() {
	// 创建日志管理器
	logManagerLogger := logger.NewLogger("log-manager", logger.GetLogLevel("info"))

	// 获取日志目录
	logDir := "logs"
	if app.configManager != nil {
		configLogDir := app.configManager.GetString("logging.directory")
		if configLogDir != "" {
			logDir = configLogDir
		}
	}

	app.logManager = logging.NewLogManager(logManagerLogger,
		logging.WithLogDir(logDir),
		logging.WithMaxLogSize(10*1024*1024), // 10MB
		logging.WithMaxLogFiles(10),
		logging.WithMaxEntries(10000),
	)
}

// initEventManager 初始化事件管理器
func (app *App) initEventManager() {
	// 创建事件管理器
	eventManagerLogger := logger.NewLogger("event-manager", logger.GetLogLevel("info"))
	app.eventManager = events.NewEventManager(eventManagerLogger,
		events.WithMaxEvents(10000),
	)
}

// GetSystemMonitor 获取系统监控器
func (app *App) GetSystemMonitor() interfaces.SystemMonitorInterface {
	return app.systemMonitor
}
