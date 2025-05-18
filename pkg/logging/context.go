package logging

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// 初始化随机数生成器
func init() {
	rand.Seed(time.Now().UnixNano())
}

// ContextWithRequestID 创建带请求ID的上下文
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		requestID = GenerateRequestID()
	}
	return context.WithValue(ctx, LogContextKeyRequestID, requestID)
}

// GetRequestIDFromContext 从上下文中获取请求ID
func GetRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value(LogContextKeyRequestID).(string); ok {
		return requestID
	}
	return ""
}

// ContextWithUserID 创建带用户ID的上下文
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, LogContextKeyUserID, userID)
}

// GetUserIDFromContext 从上下文中获取用户ID
func GetUserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if userID, ok := ctx.Value(LogContextKeyUserID).(string); ok {
		return userID
	}
	return ""
}

// ContextWithSessionID 创建带会话ID的上下文
func ContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, LogContextKeySessionID, sessionID)
}

// GetSessionIDFromContext 从上下文中获取会话ID
func GetSessionIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if sessionID, ok := ctx.Value(LogContextKeySessionID).(string); ok {
		return sessionID
	}
	return ""
}

// ContextWithTraceID 创建带跟踪ID的上下文
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		traceID = GenerateTraceID()
	}
	return context.WithValue(ctx, LogContextKeyTraceID, traceID)
}

// GetTraceIDFromContext 从上下文中获取跟踪ID
func GetTraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(LogContextKeyTraceID).(string); ok {
		return traceID
	}
	return ""
}

// ContextWithSpanID 创建带跨度ID的上下文
func ContextWithSpanID(ctx context.Context, spanID string) context.Context {
	if spanID == "" {
		spanID = GenerateSpanID()
	}
	return context.WithValue(ctx, LogContextKeySpanID, spanID)
}

// GetSpanIDFromContext 从上下文中获取跨度ID
func GetSpanIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if spanID, ok := ctx.Value(LogContextKeySpanID).(string); ok {
		return spanID
	}
	return ""
}

// ContextWithLogFields 创建带日志字段的上下文
func ContextWithLogFields(ctx context.Context, fields map[string]interface{}) context.Context {
	for k, v := range fields {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}

// GetLogFieldsFromContext 从上下文中获取日志字段
func GetLogFieldsFromContext(ctx context.Context) map[string]interface{} {
	if ctx == nil {
		return nil
	}

	fields := make(map[string]interface{})
	for _, key := range []LogContextKey{
		LogContextKeyRequestID,
		LogContextKeyUserID,
		LogContextKeySessionID,
		LogContextKeyTraceID,
		LogContextKeySpanID,
	} {
		if value := ctx.Value(key); value != nil {
			fields[string(key)] = value
		}
	}

	return fields
}

// GenerateRequestID 生成请求ID
func GenerateRequestID() string {
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), rand.Intn(1000000))
}

// GenerateTraceID 生成跟踪ID
func GenerateTraceID() string {
	return fmt.Sprintf("trace-%d-%d", time.Now().UnixNano(), rand.Intn(1000000))
}

// GenerateSpanID 生成跨度ID
func GenerateSpanID() string {
	return fmt.Sprintf("span-%d-%d", time.Now().UnixNano(), rand.Intn(1000000))
}

// LoggerFromContext 从上下文中获取日志记录器
func LoggerFromContext(ctx context.Context, defaultLogger Logger) Logger {
	if ctx == nil {
		return defaultLogger
	}

	// 从上下文中获取日志记录器
	if logger, ok := ctx.Value("logger").(Logger); ok {
		return logger
	}

	// 创建带上下文的日志记录器
	return defaultLogger.WithContext(ctx)
}

// ContextWithLogger 创建带日志记录器的上下文
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, "logger", logger)
}

// LoggingContext 日志上下文
type LoggingContext struct {
	RequestID string
	UserID    string
	SessionID string
	TraceID   string
	SpanID    string
	Fields    map[string]interface{}
}

// NewLoggingContext 创建一个新的日志上下文
func NewLoggingContext() *LoggingContext {
	return &LoggingContext{
		RequestID: GenerateRequestID(),
		TraceID:   GenerateTraceID(),
		SpanID:    GenerateSpanID(),
		Fields:    make(map[string]interface{}),
	}
}

// WithUserID 设置用户ID
func (c *LoggingContext) WithUserID(userID string) *LoggingContext {
	c.UserID = userID
	return c
}

// WithSessionID 设置会话ID
func (c *LoggingContext) WithSessionID(sessionID string) *LoggingContext {
	c.SessionID = sessionID
	return c
}

// WithField 添加字段
func (c *LoggingContext) WithField(key string, value interface{}) *LoggingContext {
	c.Fields[key] = value
	return c
}

// WithFields 添加多个字段
func (c *LoggingContext) WithFields(fields map[string]interface{}) *LoggingContext {
	for k, v := range fields {
		c.Fields[k] = v
	}
	return c
}

// ToContext 转换为上下文
func (c *LoggingContext) ToContext(ctx context.Context) context.Context {
	ctx = ContextWithRequestID(ctx, c.RequestID)
	ctx = ContextWithUserID(ctx, c.UserID)
	ctx = ContextWithSessionID(ctx, c.SessionID)
	ctx = ContextWithTraceID(ctx, c.TraceID)
	ctx = ContextWithSpanID(ctx, c.SpanID)
	ctx = ContextWithLogFields(ctx, c.Fields)
	return ctx
}

// FromContext 从上下文中创建
func (c *LoggingContext) FromContext(ctx context.Context) *LoggingContext {
	c.RequestID = GetRequestIDFromContext(ctx)
	c.UserID = GetUserIDFromContext(ctx)
	c.SessionID = GetSessionIDFromContext(ctx)
	c.TraceID = GetTraceIDFromContext(ctx)
	c.SpanID = GetSpanIDFromContext(ctx)
	c.Fields = GetLogFieldsFromContext(ctx)
	return c
}

// ToLogger 转换为日志记录器
func (c *LoggingContext) ToLogger(logger Logger) Logger {
	logger = logger.WithField("request_id", c.RequestID)
	if c.UserID != "" {
		logger = logger.WithField("user_id", c.UserID)
	}
	if c.SessionID != "" {
		logger = logger.WithField("session_id", c.SessionID)
	}
	logger = logger.WithField("trace_id", c.TraceID)
	logger = logger.WithField("span_id", c.SpanID)
	logger = logger.WithFields(c.Fields)
	return logger
}
