package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextWithRequestID(t *testing.T) {
	// 创建上下文
	ctx := context.Background()
	requestID := "req-123"
	ctx = ContextWithRequestID(ctx, requestID)

	// 获取请求ID
	result := GetRequestIDFromContext(ctx)
	assert.Equal(t, requestID, result)
}

func TestContextWithUserID(t *testing.T) {
	// 创建上下文
	ctx := context.Background()
	userID := "user-456"
	ctx = ContextWithUserID(ctx, userID)

	// 获取用户ID
	result := GetUserIDFromContext(ctx)
	assert.Equal(t, userID, result)
}

func TestContextWithSessionID(t *testing.T) {
	// 创建上下文
	ctx := context.Background()
	sessionID := "session-789"
	ctx = ContextWithSessionID(ctx, sessionID)

	// 获取会话ID
	result := GetSessionIDFromContext(ctx)
	assert.Equal(t, sessionID, result)
}

func TestContextWithTraceID(t *testing.T) {
	// 创建上下文
	ctx := context.Background()
	traceID := "trace-abc"
	ctx = ContextWithTraceID(ctx, traceID)

	// 获取跟踪ID
	result := GetTraceIDFromContext(ctx)
	assert.Equal(t, traceID, result)
}

func TestContextWithSpanID(t *testing.T) {
	// 创建上下文
	ctx := context.Background()
	spanID := "span-def"
	ctx = ContextWithSpanID(ctx, spanID)

	// 获取跨度ID
	result := GetSpanIDFromContext(ctx)
	assert.Equal(t, spanID, result)
}

func TestContextWithLogFields(t *testing.T) {
	// 创建上下文
	ctx := context.Background()
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	ctx = ContextWithLogFields(ctx, fields)

	// 获取字段
	assert.Equal(t, "value1", ctx.Value("key1"))
	assert.Equal(t, "value2", ctx.Value("key2"))
}

func TestGetLogFieldsFromContext(t *testing.T) {
	// 创建上下文
	ctx := context.Background()
	ctx = ContextWithRequestID(ctx, "req-123")
	ctx = ContextWithUserID(ctx, "user-456")
	ctx = ContextWithSessionID(ctx, "session-789")
	ctx = ContextWithTraceID(ctx, "trace-abc")
	ctx = ContextWithSpanID(ctx, "span-def")

	// 获取字段
	fields := GetLogFieldsFromContext(ctx)
	assert.Equal(t, "req-123", fields["request_id"])
	assert.Equal(t, "user-456", fields["user_id"])
	assert.Equal(t, "session-789", fields["session_id"])
	assert.Equal(t, "trace-abc", fields["trace_id"])
	assert.Equal(t, "span-def", fields["span_id"])
}

func TestGenerateRequestID(t *testing.T) {
	// 生成请求ID
	requestID := GenerateRequestID()
	assert.NotEmpty(t, requestID)
	assert.Contains(t, requestID, "req-")
}

func TestGenerateTraceID(t *testing.T) {
	// 生成跟踪ID
	traceID := GenerateTraceID()
	assert.NotEmpty(t, traceID)
	assert.Contains(t, traceID, "trace-")
}

func TestGenerateSpanID(t *testing.T) {
	// 生成跨度ID
	spanID := GenerateSpanID()
	assert.NotEmpty(t, spanID)
	assert.Contains(t, spanID, "span-")
}

func TestLoggerFromContext(t *testing.T) {
	// 创建日志记录器
	config := DefaultLogConfig()
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 创建上下文
	ctx := context.Background()
	ctx = ContextWithRequestID(ctx, "req-123")
	ctx = ContextWithUserID(ctx, "user-456")

	// 从上下文中获取日志记录器
	ctxLogger := LoggerFromContext(ctx, logger)
	assert.NotNil(t, ctxLogger)

	// 验证字段
	enhancedLogger, ok := ctxLogger.(*EnhancedLogger)
	assert.True(t, ok)
	assert.Equal(t, "req-123", enhancedLogger.fields["request_id"])
	assert.Equal(t, "user-456", enhancedLogger.fields["user_id"])
}

func TestContextWithLogger(t *testing.T) {
	// 创建日志记录器
	config := DefaultLogConfig()
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 创建上下文
	ctx := context.Background()
	ctx = ContextWithLogger(ctx, logger)

	// 从上下文中获取日志记录器
	result := ctx.Value("logger")
	assert.Equal(t, logger, result)
}

func TestLoggingContext(t *testing.T) {
	// 创建日志上下文
	logCtx := NewLoggingContext()
	assert.NotEmpty(t, logCtx.RequestID)
	assert.NotEmpty(t, logCtx.TraceID)
	assert.NotEmpty(t, logCtx.SpanID)

	// 设置字段
	logCtx.WithUserID("user-456").
		WithSessionID("session-789").
		WithField("key1", "value1").
		WithFields(map[string]interface{}{
			"key2": "value2",
			"key3": "value3",
		})

	// 验证字段
	assert.Equal(t, "user-456", logCtx.UserID)
	assert.Equal(t, "session-789", logCtx.SessionID)
	assert.Equal(t, "value1", logCtx.Fields["key1"])
	assert.Equal(t, "value2", logCtx.Fields["key2"])
	assert.Equal(t, "value3", logCtx.Fields["key3"])

	// 转换为上下文
	ctx := context.Background()
	ctx = logCtx.ToContext(ctx)

	// 验证上下文
	assert.Equal(t, logCtx.RequestID, GetRequestIDFromContext(ctx))
	assert.Equal(t, logCtx.UserID, GetUserIDFromContext(ctx))
	assert.Equal(t, logCtx.SessionID, GetSessionIDFromContext(ctx))
	assert.Equal(t, logCtx.TraceID, GetTraceIDFromContext(ctx))
	assert.Equal(t, logCtx.SpanID, GetSpanIDFromContext(ctx))

	// 创建新的日志上下文
	newLogCtx := &LoggingContext{}
	newLogCtx.FromContext(ctx)

	// 验证新的日志上下文
	assert.Equal(t, logCtx.RequestID, newLogCtx.RequestID)
	assert.Equal(t, logCtx.UserID, newLogCtx.UserID)
	assert.Equal(t, logCtx.SessionID, newLogCtx.SessionID)
	assert.Equal(t, logCtx.TraceID, newLogCtx.TraceID)
	assert.Equal(t, logCtx.SpanID, newLogCtx.SpanID)
}

func TestLoggingContextToLogger(t *testing.T) {
	// 创建日志记录器
	config := DefaultLogConfig()
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 创建日志上下文
	logCtx := NewLoggingContext().
		WithUserID("user-456").
		WithSessionID("session-789").
		WithField("key1", "value1")

	// 转换为日志记录器
	ctxLogger := logCtx.ToLogger(logger)
	assert.NotNil(t, ctxLogger)

	// 验证字段
	enhancedLogger, ok := ctxLogger.(*EnhancedLogger)
	assert.True(t, ok)
	assert.Equal(t, logCtx.RequestID, enhancedLogger.fields["request_id"])
	assert.Equal(t, logCtx.UserID, enhancedLogger.fields["user_id"])
	assert.Equal(t, logCtx.SessionID, enhancedLogger.fields["session_id"])
	assert.Equal(t, logCtx.TraceID, enhancedLogger.fields["trace_id"])
	assert.Equal(t, logCtx.SpanID, enhancedLogger.fields["span_id"])
	assert.Equal(t, "value1", enhancedLogger.fields["key1"])
}
