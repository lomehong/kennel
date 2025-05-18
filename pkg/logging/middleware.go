package logging

import (
	"context"
	"net/http"
	"time"
)

// HTTPMiddleware HTTP中间件
type HTTPMiddleware struct {
	logger Logger
}

// NewHTTPMiddleware 创建一个新的HTTP中间件
func NewHTTPMiddleware(logger Logger) *HTTPMiddleware {
	return &HTTPMiddleware{
		logger: logger,
	}
}

// Middleware 创建中间件处理函数
func (m *HTTPMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 开始时间
		startTime := time.Now()

		// 创建日志上下文
		logCtx := NewLoggingContext()

		// 从请求头中获取跟踪ID
		if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
			logCtx.TraceID = traceID
		}

		// 从请求头中获取请求ID
		if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
			logCtx.RequestID = requestID
		}

		// 从请求头中获取会话ID
		if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
			logCtx.SessionID = sessionID
		}

		// 添加请求信息
		logCtx.WithField("method", r.Method).
			WithField("path", r.URL.Path).
			WithField("remote_addr", r.RemoteAddr).
			WithField("user_agent", r.UserAgent())

		// 创建上下文
		ctx := logCtx.ToContext(r.Context())

		// 创建响应记录器
		recorder := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// 添加响应头
		recorder.Header().Set("X-Request-ID", logCtx.RequestID)
		recorder.Header().Set("X-Trace-ID", logCtx.TraceID)

		// 创建日志记录器
		logger := logCtx.ToLogger(m.logger)

		// 记录请求开始
		logger.Info("请求开始",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		// 将日志记录器添加到上下文
		ctx = ContextWithLogger(ctx, logger)

		// 处理请求
		next.ServeHTTP(recorder, r.WithContext(ctx))

		// 计算耗时
		duration := time.Since(startTime)

		// 记录请求结束
		logger.Info("请求结束",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.statusCode,
			"duration", duration.String(),
			"bytes", recorder.bytesWritten,
		)
	})
}

// responseRecorder 响应记录器
type responseRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

// WriteHeader 实现http.ResponseWriter接口
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write 实现http.ResponseWriter接口
func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytesWritten += n
	return n, err
}

// Flush 实现http.Flusher接口
func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// ContextMiddleware 上下文中间件
type ContextMiddleware struct {
	logger Logger
}

// NewContextMiddleware 创建一个新的上下文中间件
func NewContextMiddleware(logger Logger) *ContextMiddleware {
	return &ContextMiddleware{
		logger: logger,
	}
}

// WithContext 创建带上下文的处理函数
func (m *ContextMiddleware) WithContext(handler func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// 开始时间
		startTime := time.Now()

		// 创建日志上下文
		logCtx := NewLoggingContext()

		// 从上下文中获取信息
		if traceID := GetTraceIDFromContext(ctx); traceID != "" {
			logCtx.TraceID = traceID
		}
		if requestID := GetRequestIDFromContext(ctx); requestID != "" {
			logCtx.RequestID = requestID
		}
		if sessionID := GetSessionIDFromContext(ctx); sessionID != "" {
			logCtx.SessionID = sessionID
		}
		if userID := GetUserIDFromContext(ctx); userID != "" {
			logCtx.UserID = userID
		}

		// 创建上下文
		ctx = logCtx.ToContext(ctx)

		// 创建日志记录器
		logger := logCtx.ToLogger(m.logger)

		// 记录操作开始
		logger.Info("操作开始")

		// 将日志记录器添加到上下文
		ctx = ContextWithLogger(ctx, logger)

		// 处理操作
		err := handler(ctx)

		// 计算耗时
		duration := time.Since(startTime)

		// 记录操作结束
		if err != nil {
			logger.Error("操作失败",
				"error", err.Error(),
				"duration", duration.String(),
			)
		} else {
			logger.Info("操作成功",
				"duration", duration.String(),
			)
		}

		return err
	}
}

// LoggingHandler 日志处理器
type LoggingHandler struct {
	logger Logger
	next   http.Handler
}

// NewLoggingHandler 创建一个新的日志处理器
func NewLoggingHandler(logger Logger, next http.Handler) *LoggingHandler {
	return &LoggingHandler{
		logger: logger,
		next:   next,
	}
}

// ServeHTTP 实现http.Handler接口
func (h *LoggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 开始时间
	startTime := time.Now()

	// 创建日志上下文
	logCtx := NewLoggingContext()

	// 从请求头中获取跟踪ID
	if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
		logCtx.TraceID = traceID
	}

	// 从请求头中获取请求ID
	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		logCtx.RequestID = requestID
	}

	// 从请求头中获取会话ID
	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		logCtx.SessionID = sessionID
	}

	// 添加请求信息
	logCtx.WithField("method", r.Method).
		WithField("path", r.URL.Path).
		WithField("remote_addr", r.RemoteAddr).
		WithField("user_agent", r.UserAgent())

	// 创建上下文
	ctx := logCtx.ToContext(r.Context())

	// 创建响应记录器
	recorder := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// 添加响应头
	recorder.Header().Set("X-Request-ID", logCtx.RequestID)
	recorder.Header().Set("X-Trace-ID", logCtx.TraceID)

	// 创建日志记录器
	logger := logCtx.ToLogger(h.logger)

	// 记录请求开始
	logger.Info("请求开始",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	// 将日志记录器添加到上下文
	ctx = ContextWithLogger(ctx, logger)

	// 处理请求
	h.next.ServeHTTP(recorder, r.WithContext(ctx))

	// 计算耗时
	duration := time.Since(startTime)

	// 记录请求结束
	logger.Info("请求结束",
		"method", r.Method,
		"path", r.URL.Path,
		"status", recorder.statusCode,
		"duration", duration.String(),
		"bytes", recorder.bytesWritten,
	)
}
