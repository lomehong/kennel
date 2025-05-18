package logging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPMiddleware(t *testing.T) {
	// 创建日志记录器
	config := DefaultLogConfig()
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 创建中间件
	middleware := NewHTTPMiddleware(logger)

	// 创建处理函数
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取日志记录器
		ctxLogger := LoggerFromContext(r.Context(), logger)
		assert.NotNil(t, ctxLogger)

		// 记录日志
		ctxLogger.Info("处理请求")

		// 获取请求ID
		requestID := GetRequestIDFromContext(r.Context())
		assert.NotEmpty(t, requestID)

		// 获取跟踪ID
		traceID := GetTraceIDFromContext(r.Context())
		assert.NotEmpty(t, traceID)

		// 写入响应
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建中间件处理函数
	handlerWithMiddleware := middleware.Middleware(handler)

	// 创建请求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "req-123")
	req.Header.Set("X-Trace-ID", "trace-abc")
	req.Header.Set("X-Session-ID", "session-789")
	req.Header.Set("User-Agent", "test-agent")

	// 创建响应记录器
	recorder := httptest.NewRecorder()

	// 处理请求
	handlerWithMiddleware.ServeHTTP(recorder, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "OK", recorder.Body.String())

	// 验证响应头
	assert.Equal(t, "req-123", recorder.Header().Get("X-Request-ID"))
	assert.Equal(t, "trace-abc", recorder.Header().Get("X-Trace-ID"))
}

func TestContextMiddleware(t *testing.T) {
	// 创建日志记录器
	config := DefaultLogConfig()
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 创建中间件
	middleware := NewContextMiddleware(logger)

	// 创建处理函数
	handler := func(ctx context.Context) error {
		// 获取日志记录器
		ctxLogger := LoggerFromContext(ctx, logger)
		assert.NotNil(t, ctxLogger)

		// 记录日志
		ctxLogger.Info("处理操作")

		// 获取请求ID
		requestID := GetRequestIDFromContext(ctx)
		assert.NotEmpty(t, requestID)

		// 获取跟踪ID
		traceID := GetTraceIDFromContext(ctx)
		assert.NotEmpty(t, traceID)

		return nil
	}

	// 创建中间件处理函数
	handlerWithMiddleware := middleware.WithContext(handler)

	// 创建上下文
	ctx := context.Background()
	ctx = ContextWithRequestID(ctx, "req-123")
	ctx = ContextWithTraceID(ctx, "trace-abc")
	ctx = ContextWithUserID(ctx, "user-456")

	// 处理操作
	err = handlerWithMiddleware(ctx)
	assert.NoError(t, err)
}

func TestLoggingHandler(t *testing.T) {
	// 创建日志记录器
	config := DefaultLogConfig()
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 创建处理函数
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取日志记录器
		ctxLogger := LoggerFromContext(r.Context(), logger)
		assert.NotNil(t, ctxLogger)

		// 记录日志
		ctxLogger.Info("处理请求")

		// 获取请求ID
		requestID := GetRequestIDFromContext(r.Context())
		assert.NotEmpty(t, requestID)

		// 获取跟踪ID
		traceID := GetTraceIDFromContext(r.Context())
		assert.NotEmpty(t, traceID)

		// 写入响应
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建日志处理器
	loggingHandler := NewLoggingHandler(logger, handler)

	// 创建请求
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "req-123")
	req.Header.Set("X-Trace-ID", "trace-abc")
	req.Header.Set("X-Session-ID", "session-789")
	req.Header.Set("User-Agent", "test-agent")

	// 创建响应记录器
	recorder := httptest.NewRecorder()

	// 处理请求
	loggingHandler.ServeHTTP(recorder, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "OK", recorder.Body.String())

	// 验证响应头
	assert.Equal(t, "req-123", recorder.Header().Get("X-Request-ID"))
	assert.Equal(t, "trace-abc", recorder.Header().Get("X-Trace-ID"))
}

func TestResponseRecorder(t *testing.T) {
	// 创建响应记录器
	recorder := httptest.NewRecorder()
	responseRecorder := &responseRecorder{
		ResponseWriter: recorder,
		statusCode:     http.StatusOK,
	}

	// 写入响应
	responseRecorder.WriteHeader(http.StatusCreated)
	n, err := responseRecorder.Write([]byte("OK"))
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	// 验证状态码
	assert.Equal(t, http.StatusCreated, responseRecorder.statusCode)

	// 验证字节数
	assert.Equal(t, 2, responseRecorder.bytesWritten)

	// 验证响应
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "OK", recorder.Body.String())
}
