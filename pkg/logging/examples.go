package logging

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"
)

// 以下是日志增强和结构化日志的使用示例

// ExampleBasicLogging 展示基本日志记录
func ExampleBasicLogging() {
	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelDebug,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 记录不同级别的日志
	logger.Trace("这是一条跟踪日志", "key", "value")
	logger.Debug("这是一条调试日志", "count", 42)
	logger.Info("这是一条信息日志", "enabled", true)
	logger.Warn("这是一条警告日志", "status", "warning")
	logger.Error("这是一条错误日志", "error", "test error")
	// logger.Fatal("这是一条致命日志") // 会导致程序退出
}

// ExampleStructuredLogging 展示结构化日志记录
func ExampleStructuredLogging() {
	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelInfo,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		return
	}
	defer logger.Close()

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
}

// ExampleContextLogging 展示带上下文的日志记录
func ExampleContextLogging() {
	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelInfo,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 创建上下文
	ctx := context.Background()
	ctx = ContextWithRequestID(ctx, "req-123")
	ctx = ContextWithUserID(ctx, "user-456")
	ctx = ContextWithSessionID(ctx, "session-789")
	ctx = ContextWithTraceID(ctx, "trace-abc")
	ctx = ContextWithSpanID(ctx, "span-def")

	// 创建带上下文的日志记录器
	ctxLogger := logger.WithContext(ctx)

	// 记录日志
	ctxLogger.Info("处理请求", "path", "/api/users", "method", "GET")

	// 模拟处理过程
	ctxLogger.Debug("查询数据库", "table", "users", "query", "id = 456")
	ctxLogger.Info("查询结果", "found", true, "user_name", "张三")

	// 模拟响应
	ctxLogger.Info("请求完成", "status", 200, "duration", "10ms")
}

// ExampleNamedLogger 展示命名日志记录器
func ExampleNamedLogger() {
	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelInfo,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 创建命名日志记录器
	userLogger := logger.Named("user-service")
	orderLogger := logger.Named("order-service")
	paymentLogger := logger.Named("payment-service")

	// 记录日志
	userLogger.Info("用户服务启动", "port", 8080)
	orderLogger.Info("订单服务启动", "port", 8081)
	paymentLogger.Info("支付服务启动", "port", 8082)

	// 模拟处理过程
	userLogger.Info("用户登录", "user_id", "user-123")
	orderLogger.Info("创建订单", "order_id", "order-456", "user_id", "user-123")
	paymentLogger.Info("处理支付", "payment_id", "payment-789", "order_id", "order-456")
}

// ExampleFileLogging 展示文件日志记录
func ExampleFileLogging() {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "log-example")
	if err != nil {
		fmt.Printf("创建临时目录失败: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelInfo,
		Format:           LogFormatJSON,
		Output:           LogOutputFile,
		FilePath:         fmt.Sprintf("%s/app.log", tempDir),
		RotationInterval: 24 * time.Hour,
		MaxSize:          10 * 1024 * 1024, // 10MB
		MaxAge:           7 * 24 * time.Hour,
		MaxBackups:       5,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 记录日志
	logger.Info("应用程序启动", "version", "1.0.0")

	// 模拟大量日志
	for i := 0; i < 100; i++ {
		logger.Info("处理请求", "request_id", fmt.Sprintf("req-%d", i))
	}

	logger.Info("应用程序关闭")

	// 显示日志文件
	fmt.Printf("日志文件路径: %s\n", config.FilePath)
}

// ExampleHTTPMiddleware 展示HTTP中间件
func ExampleHTTPMiddleware() {
	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelInfo,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 创建中间件
	middleware := NewHTTPMiddleware(logger)

	// 创建处理函数
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取日志记录器
		ctxLogger := LoggerFromContext(r.Context(), logger)

		// 记录日志
		ctxLogger.Info("处理请求", "path", r.URL.Path, "method", r.Method)

		// 获取请求ID
		requestID := GetRequestIDFromContext(r.Context())
		ctxLogger.Debug("请求ID", "request_id", requestID)

		// 写入响应
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建中间件处理函数
	handlerWithMiddleware := middleware.Middleware(handler)

	// 创建HTTP服务器
	http.Handle("/", handlerWithMiddleware)
	fmt.Println("HTTP服务器启动在 :8080")
	// http.ListenAndServe(":8080", nil) // 实际运行时取消注释
}

// ExampleContextMiddleware 展示上下文中间件
func ExampleContextMiddleware() {
	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelInfo,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 创建中间件
	middleware := NewContextMiddleware(logger)

	// 创建处理函数
	handler := func(ctx context.Context) error {
		// 获取日志记录器
		ctxLogger := LoggerFromContext(ctx, logger)

		// 记录日志
		ctxLogger.Info("处理操作")

		// 获取请求ID
		requestID := GetRequestIDFromContext(ctx)
		ctxLogger.Debug("请求ID", "request_id", requestID)

		// 模拟处理过程
		ctxLogger.Info("操作完成")

		return nil
	}

	// 创建中间件处理函数
	handlerWithMiddleware := middleware.WithContext(handler)

	// 创建上下文
	ctx := context.Background()
	ctx = ContextWithRequestID(ctx, "req-123")
	ctx = ContextWithUserID(ctx, "user-456")

	// 处理操作
	err = handlerWithMiddleware(ctx)
	if err != nil {
		fmt.Printf("处理操作失败: %v\n", err)
	}
}

// ExampleLoggingContext 展示日志上下文
func ExampleLoggingContext() {
	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelInfo,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 创建日志上下文
	logCtx := NewLoggingContext().
		WithUserID("user-456").
		WithSessionID("session-789").
		WithField("component", "example").
		WithField("version", "1.0.0")

	// 创建上下文
	ctx := context.Background()
	ctx = logCtx.ToContext(ctx)

	// 创建日志记录器
	ctxLogger := logCtx.ToLogger(logger)

	// 记录日志
	ctxLogger.Info("使用日志上下文")

	// 从上下文中获取日志上下文
	newLogCtx := &LoggingContext{}
	newLogCtx.FromContext(ctx)

	// 创建新的日志记录器
	newLogger := newLogCtx.ToLogger(logger)

	// 记录日志
	newLogger.Info("从上下文中恢复日志上下文")
}
