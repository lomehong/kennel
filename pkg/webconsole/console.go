package webconsole

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/interfaces"
	"github.com/lomehong/kennel/pkg/logger"
)

// Console 定义Web控制台
type Console struct {
	// 配置
	config Config

	// 应用实例
	app interfaces.AppInterface

	// HTTP服务器
	server *http.Server

	// Gin引擎
	engine *gin.Engine

	// 日志
	logger hclog.Logger

	// 互斥锁
	mu sync.RWMutex

	// 是否已初始化
	initialized bool

	// 是否已启动
	started bool
}

// NewConsole 创建一个新的Web控制台
func NewConsole(config Config, app interfaces.AppInterface) (*Console, error) {
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("无效的Web控制台配置: %w", err)
	}

	// 设置Gin模式
	if config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin引擎
	engine := gin.New()

	// 创建日志
	log := logger.NewLogger("web-console", hclog.LevelFromString(config.LogLevel))

	// 创建Web控制台
	console := &Console{
		config:      config,
		app:         app,
		engine:      engine,
		logger:      log.GetHCLogger(),
		initialized: false,
		started:     false,
	}

	return console, nil
}

// Init 初始化Web控制台
func (c *Console) Init() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return fmt.Errorf("Web控制台已初始化")
	}

	c.logger.Info("初始化Web控制台")

	// 设置中间件
	c.setupMiddleware()

	// 设置路由
	c.setupRoutes()

	// 创建HTTP服务器
	c.server = &http.Server{
		Addr:    c.config.GetAddress(),
		Handler: c.engine,
	}

	c.initialized = true
	c.logger.Info("Web控制台初始化完成")

	return nil
}

// Start 启动Web控制台
func (c *Console) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.initialized {
		return fmt.Errorf("Web控制台未初始化")
	}

	if c.started {
		return fmt.Errorf("Web控制台已启动")
	}

	c.logger.Info("启动Web控制台", "address", c.config.GetAddress())

	// 启动HTTP服务器
	go func() {
		var err error
		if c.config.EnableHTTPS {
			err = c.server.ListenAndServeTLS(c.config.CertFile, c.config.KeyFile)
		} else {
			err = c.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			c.logger.Error("Web控制台启动失败", "error", err)
		}
	}()

	c.started = true
	c.logger.Info("Web控制台已启动", "address", c.config.GetAddress(), "auth", c.config.EnableAuth)

	return nil
}

// Stop 停止Web控制台
func (c *Console) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	c.logger.Info("停止Web控制台")

	// 设置关闭超时
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := c.server.Shutdown(shutdownCtx); err != nil {
		c.logger.Error("Web控制台关闭失败", "error", err)
		return fmt.Errorf("Web控制台关闭失败: %w", err)
	}

	c.started = false
	c.logger.Info("Web控制台已停止")

	return nil
}

// setupMiddleware 设置中间件
func (c *Console) setupMiddleware() {
	// 使用日志中间件
	c.engine.Use(gin.Logger())

	// 使用恢复中间件
	c.engine.Use(gin.Recovery())

	// 使用CORS中间件
	c.engine.Use(c.corsMiddleware())

	// 使用认证中间件
	if c.config.EnableAuth {
		c.engine.Use(c.authMiddleware())
	}

	// 使用CSRF中间件
	if c.config.EnableCSRF {
		c.engine.Use(c.csrfMiddleware())
	}

	// 使用请求限制中间件
	c.engine.Use(c.rateLimitMiddleware())
}

// setupRoutes 设置路由
func (c *Console) setupRoutes() {
	// 记录API前缀
	c.logger.Info("设置API路由", "prefix", c.config.APIPrefix)

	// 注册模拟API路由
	c.registerMockAPIRoutes(c.engine)

	// API路由组
	api := c.engine.Group(c.config.APIPrefix)
	{
		// 添加调试路由
		api.GET("/ping", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{
				"message": "pong",
				"time":    time.Now().Format(time.RFC3339),
			})
		})

		// 插件管理API
		plugins := api.Group("/plugins")
		{
			plugins.GET("", c.getPlugins)
			plugins.GET("/:id", c.getPlugin)
			plugins.PUT("/:id/status", c.updatePluginStatus)
			plugins.PUT("/:id/config", c.updatePluginConfig)
			plugins.GET("/:id/logs", c.getPluginLogs)
		}

		// 指标监控API
		metrics := api.Group("/metrics")
		{
			metrics.GET("", c.getMetrics)
			metrics.GET("/comm", c.getCommMetrics)
			metrics.GET("/system", c.getSystemMetrics)
		}

		// 系统监控API
		system := api.Group("/system")
		{
			system.GET("/status", c.getSystemStatus)
			system.GET("/resources", c.getSystemResources)
			system.GET("/logs", c.getSystemLogs)
			system.GET("/events", c.getSystemEvents)
		}

		// 配置管理API
		config := api.Group("/config")
		{
			config.GET("", c.getConfig)
			config.PUT("", c.updateConfig)
			config.POST("/reset", c.resetConfig)
		}

		// 通讯管理API
		comm := api.Group("/comm")
		{
			comm.GET("/status", c.getCommStatus)
			comm.POST("/connect", c.connectComm)
			comm.POST("/disconnect", c.disconnectComm)
			comm.GET("/config", c.getCommConfig)
			comm.GET("/stats", c.getCommStats)
			comm.GET("/logs", c.getCommLogs)

			// 通讯测试API
			commTest := comm.Group("/test")
			{
				commTest.POST("/connection", c.testCommConnection)
				commTest.POST("/send-receive", c.testCommSendReceive)
				commTest.POST("/encryption", c.testCommEncryption)
				commTest.POST("/compression", c.testCommCompression)
				commTest.POST("/performance", c.testCommPerformance)
				commTest.GET("/history", c.getCommTestHistory)
			}
		}
	}

	// 记录已注册的路由
	routes := c.engine.Routes()
	c.logger.Debug("已注册的路由", "count", len(routes))
	for _, route := range routes {
		c.logger.Debug("路由", "method", route.Method, "path", route.Path)
	}

	// 静态文件
	// 注意：必须在设置API路由之后设置静态文件路由
	c.logger.Debug("设置静态文件路由", "staticDir", c.config.StaticDir)

	// 检查静态文件目录是否存在
	if _, err := os.Stat(c.config.StaticDir); os.IsNotExist(err) {
		c.logger.Warn("静态文件目录不存在", "path", c.config.StaticDir)
	} else {
		c.logger.Debug("静态文件目录存在", "path", c.config.StaticDir)

		// 检查index.html是否存在
		indexPath := filepath.Join(c.config.StaticDir, "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			c.logger.Warn("index.html文件不存在", "path", indexPath)
		} else {
			c.logger.Debug("index.html文件存在", "path", indexPath)
		}

		// 检查assets目录是否存在
		assetsPath := filepath.Join(c.config.StaticDir, "assets")
		if _, err := os.Stat(assetsPath); os.IsNotExist(err) {
			c.logger.Warn("assets目录不存在", "path", assetsPath)
		} else {
			c.logger.Debug("assets目录存在", "path", assetsPath)
		}
	}

	// 设置静态资源路由 - 只使用一种路径格式，避免冲突
	// 静态资源目录
	c.engine.Static("/assets", filepath.Join(c.config.StaticDir, "assets"))

	// 图标文件
	c.engine.StaticFile("/favicon.ico", filepath.Join(c.config.StaticDir, "favicon.ico"))
	c.engine.StaticFile("/favicon.svg", filepath.Join(c.config.StaticDir, "favicon.svg"))

	// API代理脚本
	c.engine.StaticFile("/api-proxy.js", filepath.Join(c.config.StaticDir, "api-proxy.js"))

	// 首页
	c.engine.StaticFile("/", filepath.Join(c.config.StaticDir, "index.html"))
	c.engine.StaticFile("/index.html", filepath.Join(c.config.StaticDir, "index.html"))

	// 为SPA路由添加特定路径处理
	spaRoutes := []string{
		"/plugins",
		"/plugins/:id",
		"/metrics",
		"/system",
		"/comm",
		"/config",
	}

	for _, route := range spaRoutes {
		c.engine.GET(route, func(ctx *gin.Context) {
			c.logger.Debug("SPA路由", "path", ctx.Request.URL.Path)
			ctx.File(filepath.Join(c.config.StaticDir, "index.html"))
		})
	}

	// 404处理
	c.engine.NoRoute(func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		c.logger.Debug("收到请求", "path", path, "method", ctx.Request.Method)

		// 对于API请求，返回404 JSON
		if len(path) >= len(c.config.APIPrefix) && path[:len(c.config.APIPrefix)] == c.config.APIPrefix {
			c.logger.Debug("API请求未找到", "path", path)
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "API not found",
				"path":  path,
			})
			return
		}

		// 对于静态资源请求，如果文件不存在，返回404
		if strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "assets/") {
			c.logger.Warn("静态资源不存在", "path", path)
			ctx.Status(http.StatusNotFound)
			return
		}

		// 对于其他请求，返回index.html（用于SPA路由）
		c.logger.Debug("处理SPA路由请求", "path", path)
		indexPath := filepath.Join(c.config.StaticDir, "index.html")

		// 检查index.html是否存在
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			c.logger.Error("index.html文件不存在", "path", indexPath)
			ctx.String(http.StatusInternalServerError, "Web控制台前端文件未找到")
			return
		}

		ctx.File(indexPath)
	})
}
