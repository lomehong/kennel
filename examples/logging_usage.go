package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/lomehong/kennel/pkg/core"
)

// 本示例展示如何在AppFramework中使用日志增强和结构化日志
func main() {
	// 创建应用程序实例
	app := core.NewApp("config.yaml")

	// 初始化应用程序
	if err := app.Init(); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 日志增强和结构化日志使用示例 ===")

	// 示例1: 基本日志记录
	fmt.Println("\n=== 示例1: 基本日志记录 ===")
	basicLogging(app)

	// 示例2: 结构化日志记录
	fmt.Println("\n=== 示例2: 结构化日志记录 ===")
	structuredLogging(app)

	// 示例3: 带上下文的日志记录
	fmt.Println("\n=== 示例3: 带上下文的日志记录 ===")
	contextLogging(app)

	// 示例4: 命名日志记录器
	fmt.Println("\n=== 示例4: 命名日志记录器 ===")
	namedLogging(app)

	// 示例5: HTTP中间件
	fmt.Println("\n=== 示例5: HTTP中间件 ===")
	httpMiddleware(app)

	// 示例6: 上下文中间件
	fmt.Println("\n=== 示例6: 上下文中间件 ===")
	contextMiddleware(app)

	// 示例7: 日志级别控制
	fmt.Println("\n=== 示例7: 日志级别控制 ===")
	logLevelControl(app)

	// 停止应用程序
	app.Stop()
	fmt.Println("\n应用程序已停止")
}

// 基本日志记录
func basicLogging(app *core.App) {
	// 获取增强日志记录器
	logger := app.GetEnhancedLogger()
	if logger == nil {
		fmt.Println("增强日志记录器未初始化")
		return
	}

	// 记录不同级别的日志
	logger.Trace("这是一条跟踪日志", "key", "value")
	logger.Debug("这是一条调试日志", "count", 42)
	logger.Info("这是一条信息日志", "enabled", true)
	logger.Warn("这是一条警告日志", "status", "warning")
	logger.Error("这是一条错误日志", "error", "test error")
	// logger.Fatal("这是一条致命日志") // 会导致程序退出

	fmt.Println("已记录各级别日志")
}

// 结构化日志记录
func structuredLogging(app *core.App) {
	// 获取增强日志记录器
	logger := app.GetEnhancedLogger()
	if logger == nil {
		fmt.Println("增强日志记录器未初始化")
		return
	}

	// 记录结构化日志
	logger.Info("用户登录",
		"user_id", "user-123",
		"username", "张三",
		"login_time", time.Now().Format(time.RFC3339),
		"ip_address", "192.168.1.1",
		"success", true,
	)

	// 记录带嵌套结构的日志
	logger.Info("订单创建",
		"order_id", "order-456",
		"user_id", "user-123",
		"amount", 99.99,
		"items_count", 3,
		"payment_method", "credit_card",
		"shipping_address", map[string]interface{}{
			"country":  "中国",
			"province": "广东",
			"city":     "深圳",
			"street":   "科技园路",
		},
	)

	fmt.Println("已记录结构化日志")
}

// 带上下文的日志记录
func contextLogging(app *core.App) {
	// 获取增强日志记录器
	logger := app.GetEnhancedLogger()
	if logger == nil {
		fmt.Println("增强日志记录器未初始化")
		return
	}

	// 创建上下文
	ctx := context.Background()
	ctx = app.ContextWithRequestID(ctx, "req-123")
	ctx = app.ContextWithUserID(ctx, "user-456")
	ctx = app.ContextWithSessionID(ctx, "session-789")
	ctx = app.ContextWithTraceID(ctx, "trace-abc")
	ctx = app.ContextWithSpanID(ctx, "span-def")

	// 创建带上下文的日志记录器
	ctxLogger := app.GetLoggerWithContext(ctx)
	if ctxLogger == nil {
		fmt.Println("带上下文的日志记录器创建失败")
		return
	}

	// 记录日志
	ctxLogger.Info("处理请求", "path", "/api/users", "method", "GET")

	// 模拟处理过程
	ctxLogger.Debug("查询数据库", "table", "users", "query", "id = 456")
	ctxLogger.Info("查询结果", "found", true, "user_name", "张三")

	// 模拟响应
	ctxLogger.Info("请求完成", "status", 200, "duration", "10ms")

	// 从上下文中获取日志记录器
	loggerFromCtx := app.LoggerFromContext(ctx)
	if loggerFromCtx != nil {
		loggerFromCtx.Info("从上下文中获取日志记录器")
	}

	// 获取上下文中的字段
	requestID := app.GetRequestIDFromContext(ctx)
	userID := app.GetUserIDFromContext(ctx)
	sessionID := app.GetSessionIDFromContext(ctx)
	traceID := app.GetTraceIDFromContext(ctx)
	spanID := app.GetSpanIDFromContext(ctx)

	fmt.Printf("上下文字段: requestID=%s, userID=%s, sessionID=%s, traceID=%s, spanID=%s\n",
		requestID, userID, sessionID, traceID, spanID)
}

// 命名日志记录器
func namedLogging(app *core.App) {
	// 获取增强日志记录器
	logger := app.GetEnhancedLogger()
	if logger == nil {
		fmt.Println("增强日志记录器未初始化")
		return
	}

	// 创建命名日志记录器
	userLogger := app.GetNamedLogger("user-service")
	orderLogger := app.GetNamedLogger("order-service")
	paymentLogger := app.GetNamedLogger("payment-service")

	// 记录日志
	userLogger.Info("用户服务启动", "port", 8080)
	orderLogger.Info("订单服务启动", "port", 8081)
	paymentLogger.Info("支付服务启动", "port", 8082)

	// 模拟处理过程
	userLogger.Info("用户登录", "user_id", "user-123")
	orderLogger.Info("创建订单", "order_id", "order-456", "user_id", "user-123")
	paymentLogger.Info("处理支付", "payment_id", "payment-789", "order_id", "order-456")

	fmt.Println("已使用命名日志记录器记录日志")
}

// HTTP中间件
func httpMiddleware(app *core.App) {
	// 获取HTTP中间件
	middleware := app.GetHTTPMiddleware()
	if middleware == nil {
		fmt.Println("HTTP中间件未初始化")
		return
	}

	// 创建处理函数
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取日志记录器
		logger := app.LoggerFromContext(r.Context())
		if logger == nil {
			fmt.Println("从上下文中获取日志记录器失败")
			return
		}

		// 记录日志
		logger.Info("处理请求", "path", r.URL.Path, "method", r.Method)

		// 获取请求ID
		requestID := app.GetRequestIDFromContext(r.Context())
		logger.Debug("请求ID", "request_id", requestID)

		// 写入响应
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建中间件处理函数
	handlerWithMiddleware := middleware.Middleware(handler)

	// 创建HTTP服务器
	http.Handle("/", handlerWithMiddleware)
	fmt.Println("HTTP服务器已配置中间件")
	// http.ListenAndServe(":8080", nil) // 实际运行时取消注释
}

// 上下文中间件
func contextMiddleware(app *core.App) {
	// 获取上下文中间件
	middleware := app.GetContextMiddleware()
	if middleware == nil {
		fmt.Println("上下文中间件未初始化")
		return
	}

	// 创建处理函数
	handler := func(ctx context.Context) error {
		// 获取日志记录器
		logger := app.LoggerFromContext(ctx)
		if logger == nil {
			fmt.Println("从上下文中获取日志记录器失败")
			return fmt.Errorf("日志记录器未找到")
		}

		// 记录日志
		logger.Info("处理操作")

		// 获取请求ID
		requestID := app.GetRequestIDFromContext(ctx)
		logger.Debug("请求ID", "request_id", requestID)

		// 模拟处理过程
		logger.Info("操作完成")

		return nil
	}

	// 创建中间件处理函数
	handlerWithMiddleware := middleware.WithContext(handler)

	// 创建上下文
	ctx := context.Background()
	ctx = app.ContextWithRequestID(ctx, "req-123")
	ctx = app.ContextWithUserID(ctx, "user-456")

	// 处理操作
	err := handlerWithMiddleware(ctx)
	if err != nil {
		fmt.Printf("处理操作失败: %v\n", err)
	} else {
		fmt.Println("操作已处理")
	}
}

// 日志级别控制
func logLevelControl(app *core.App) {
	// 获取增强日志记录器
	logger := app.GetEnhancedLogger()
	if logger == nil {
		fmt.Println("增强日志记录器未初始化")
		return
	}

	// 获取当前日志级别
	currentLevel := app.GetLogLevel()
	fmt.Printf("当前日志级别: %s\n", currentLevel)

	// 记录调试日志（根据当前级别可能不会输出）
	logger.Debug("这是一条调试日志")

	// 设置日志级别为调试
	err := app.SetLogLevel("debug")
	if err != nil {
		fmt.Printf("设置日志级别失败: %v\n", err)
		return
	}

	// 获取新的日志级别
	newLevel := app.GetLogLevel()
	fmt.Printf("新的日志级别: %s\n", newLevel)

	// 记录调试日志（现在应该会输出）
	logger.Debug("这是一条调试日志")

	// 恢复日志级别
	err = app.SetLogLevel(currentLevel)
	if err != nil {
		fmt.Printf("恢复日志级别失败: %v\n", err)
	}
}
