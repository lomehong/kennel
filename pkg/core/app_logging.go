package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// 添加日志增强和结构化日志到App结构体
func (app *App) initLoggingSystem() {
	// 获取日志配置
	logLevel := app.configManager.GetString("log_level")
	if logLevel == "" {
		logLevel = "info"
	}

	logFormat := app.configManager.GetString("log_format")
	if logFormat == "" {
		logFormat = "json"
	}

	logOutput := app.configManager.GetString("log_output")
	if logOutput == "" {
		logOutput = "stdout"
	}

	logFile := app.configManager.GetString("log_file")
	if logFile == "" {
		logFile = "logs/app.log"
	}

	// 创建日志配置
	config := &logging.LogConfig{
		Level:            logging.LogLevel(logLevel),
		Format:           logging.LogFormat(logFormat),
		Output:           logging.LogOutput(logOutput),
		FilePath:         logFile,
		RotationInterval: app.configManager.GetDurationOrDefault("log_rotation_interval", 24*time.Hour),
		MaxSize:          app.configManager.GetInt64OrDefault("log_max_size", 100*1024*1024), // 100MB
		MaxAge:           app.configManager.GetDurationOrDefault("log_max_age", 7*24*time.Hour),
		MaxBackups:       app.configManager.GetIntOrDefault("log_max_backups", 10),
		IncludeLocation:  app.configManager.GetBoolOrDefault("log_include_location", true),
		IncludeTimestamp: app.configManager.GetBoolOrDefault("log_include_timestamp", true),
		TimeFormat:       app.configManager.GetStringOrDefault("log_time_format", time.RFC3339),
		DefaultContext: map[string]string{
			"app_name":    app.configManager.GetStringOrDefault("app_name", "appframework"),
			"app_version": app.version,
		},
	}

	// 确保日志目录存在
	if config.Output == logging.LogOutputFile {
		dir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			app.logger.Error("创建日志目录失败", "error", err)
		}
	}

	// 创建增强日志记录器
	enhancedLogger, err := logging.NewEnhancedLogger(config)
	if err != nil {
		app.logger.Error("创建增强日志记录器失败", "error", err)
		return
	}

	// 保存原始日志记录器
	app.originalLogger = app.logger

	// 设置增强日志记录器
	app.enhancedLogger = enhancedLogger
	app.logger = enhancedLogger.GetHCLogger()

	app.logger.Info("日志增强系统已初始化",
		"level", config.Level,
		"format", config.Format,
		"output", config.Output,
	)
}

// GetEnhancedLogger 获取增强日志记录器
func (app *App) GetEnhancedLogger() logging.Logger {
	return app.enhancedLogger
}

// GetLoggerWithContext 获取带上下文的日志记录器
func (app *App) GetLoggerWithContext(ctx context.Context) logging.Logger {
	if app.enhancedLogger != nil {
		return app.enhancedLogger.WithContext(ctx)
	}
	return nil
}

// GetLoggerWithField 获取带字段的日志记录器
func (app *App) GetLoggerWithField(key string, value interface{}) logging.Logger {
	if app.enhancedLogger != nil {
		return app.enhancedLogger.WithField(key, value)
	}
	return nil
}

// GetLoggerWithFields 获取带多个字段的日志记录器
func (app *App) GetLoggerWithFields(fields map[string]interface{}) logging.Logger {
	if app.enhancedLogger != nil {
		return app.enhancedLogger.WithFields(fields)
	}
	return nil
}

// GetNamedLogger 获取命名的日志记录器
func (app *App) GetNamedLogger(name string) logging.Logger {
	if app.enhancedLogger != nil {
		return app.enhancedLogger.Named(name)
	}
	return nil
}

// SetLogLevel 设置日志级别
func (app *App) SetLogLevel(level string) error {
	if app.enhancedLogger != nil {
		logLevel := logging.LogLevel(level)
		app.enhancedLogger.SetLevel(logLevel)
		app.configManager.Set("log_level", level)
		app.logger.Info("日志级别已更改", "level", level)
		return nil
	}
	return fmt.Errorf("增强日志记录器未初始化")
}

// GetLogLevel 获取日志级别
func (app *App) GetLogLevel() string {
	if app.enhancedLogger != nil {
		return string(app.enhancedLogger.GetLevel())
	}
	return ""
}

// CloseLogger 关闭日志记录器
func (app *App) CloseLogger() error {
	if app.enhancedLogger != nil {
		err := app.enhancedLogger.Close()
		if err != nil {
			return fmt.Errorf("关闭日志记录器失败: %w", err)
		}
		// 恢复原始日志记录器
		if app.originalLogger != nil {
			app.logger = app.originalLogger
		}
		app.enhancedLogger = nil
		return nil
	}
	return nil
}

// CreateRequestContext 创建请求上下文
func (app *App) CreateRequestContext() *logging.LoggingContext {
	return logging.NewLoggingContext()
}

// GetHTTPMiddleware 获取HTTP中间件
func (app *App) GetHTTPMiddleware() *logging.HTTPMiddleware {
	if app.enhancedLogger != nil {
		return logging.NewHTTPMiddleware(app.enhancedLogger)
	}
	return nil
}

// GetContextMiddleware 获取上下文中间件
func (app *App) GetContextMiddleware() *logging.ContextMiddleware {
	if app.enhancedLogger != nil {
		return logging.NewContextMiddleware(app.enhancedLogger)
	}
	return nil
}

// LogWithContext 使用上下文记录日志
func (app *App) LogWithContext(ctx context.Context, level string, msg string, args ...interface{}) {
	if app.enhancedLogger == nil {
		return
	}

	logger := app.enhancedLogger.WithContext(ctx)
	switch logging.LogLevel(level) {
	case logging.LogLevelTrace:
		logger.Trace(msg, args...)
	case logging.LogLevelDebug:
		logger.Debug(msg, args...)
	case logging.LogLevelInfo:
		logger.Info(msg, args...)
	case logging.LogLevelWarn:
		logger.Warn(msg, args...)
	case logging.LogLevelError:
		logger.Error(msg, args...)
	case logging.LogLevelFatal:
		logger.Fatal(msg, args...)
	default:
		logger.Info(msg, args...)
	}
}

// ContextWithRequestID 创建带请求ID的上下文
func (app *App) ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return logging.ContextWithRequestID(ctx, requestID)
}

// ContextWithUserID 创建带用户ID的上下文
func (app *App) ContextWithUserID(ctx context.Context, userID string) context.Context {
	return logging.ContextWithUserID(ctx, userID)
}

// ContextWithSessionID 创建带会话ID的上下文
func (app *App) ContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return logging.ContextWithSessionID(ctx, sessionID)
}

// ContextWithTraceID 创建带跟踪ID的上下文
func (app *App) ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return logging.ContextWithTraceID(ctx, traceID)
}

// ContextWithSpanID 创建带跨度ID的上下文
func (app *App) ContextWithSpanID(ctx context.Context, spanID string) context.Context {
	return logging.ContextWithSpanID(ctx, spanID)
}

// ContextWithLogger 创建带日志记录器的上下文
func (app *App) ContextWithLogger(ctx context.Context, logger logging.Logger) context.Context {
	return logging.ContextWithLogger(ctx, logger)
}

// LoggerFromContext 从上下文中获取日志记录器
func (app *App) LoggerFromContext(ctx context.Context) logging.Logger {
	if app.enhancedLogger != nil {
		return logging.LoggerFromContext(ctx, app.enhancedLogger)
	}
	return nil
}

// GetRequestIDFromContext 从上下文中获取请求ID
func (app *App) GetRequestIDFromContext(ctx context.Context) string {
	return logging.GetRequestIDFromContext(ctx)
}

// GetUserIDFromContext 从上下文中获取用户ID
func (app *App) GetUserIDFromContext(ctx context.Context) string {
	return logging.GetUserIDFromContext(ctx)
}

// GetSessionIDFromContext 从上下文中获取会话ID
func (app *App) GetSessionIDFromContext(ctx context.Context) string {
	return logging.GetSessionIDFromContext(ctx)
}

// GetTraceIDFromContext 从上下文中获取跟踪ID
func (app *App) GetTraceIDFromContext(ctx context.Context) string {
	return logging.GetTraceIDFromContext(ctx)
}

// GetSpanIDFromContext 从上下文中获取跨度ID
func (app *App) GetSpanIDFromContext(ctx context.Context) string {
	return logging.GetSpanIDFromContext(ctx)
}

// GetLogFieldsFromContext 从上下文中获取日志字段
func (app *App) GetLogFieldsFromContext(ctx context.Context) map[string]interface{} {
	return logging.GetLogFieldsFromContext(ctx)
}
