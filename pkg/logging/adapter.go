package logging

import (
	"github.com/hashicorp/go-hclog"
)

// LegacyLogger 定义了与旧版 pkg/logger.Logger 兼容的接口
// 这个接口用于平滑迁移，使旧代码能够继续工作
type LegacyLogger interface {
	// 基本日志方法
	Trace(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})

	// 创建带有上下文的新日志器
	With(args ...interface{}) LegacyLogger

	// 设置日志级别
	SetLevel(level hclog.Level)

	// 获取原始的hclog.Logger
	GetHCLogger() hclog.Logger
}

// LegacyLoggerAdapter 将 Logger 适配为 LegacyLogger
type LegacyLoggerAdapter struct {
	logger Logger
}

// NewLegacyLoggerAdapter 创建一个新的旧版日志适配器
func NewLegacyLoggerAdapter(logger Logger) LegacyLogger {
	return &LegacyLoggerAdapter{
		logger: logger,
	}
}

// Trace 记录跟踪级别日志
func (l *LegacyLoggerAdapter) Trace(msg string, args ...interface{}) {
	l.logger.Trace(msg, args...)
}

// Debug 记录调试级别日志
func (l *LegacyLoggerAdapter) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

// Info 记录信息级别日志
func (l *LegacyLoggerAdapter) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

// Warn 记录警告级别日志
func (l *LegacyLoggerAdapter) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

// Error 记录错误级别日志
func (l *LegacyLoggerAdapter) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

// With 创建带有上下文的新日志器
func (l *LegacyLoggerAdapter) With(args ...interface{}) LegacyLogger {
	// 将参数转换为字段
	fields := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, ok := args[i].(string)
			if ok {
				fields[key] = args[i+1]
			}
		}
	}

	// 创建新的日志记录器
	newLogger := l.logger.WithFields(fields)
	return &LegacyLoggerAdapter{
		logger: newLogger,
	}
}

// SetLevel 设置日志级别
func (l *LegacyLoggerAdapter) SetLevel(level hclog.Level) {
	// 将 hclog.Level 转换为 LogLevel
	var logLevel LogLevel
	switch level {
	case hclog.Trace:
		logLevel = LogLevelTrace
	case hclog.Debug:
		logLevel = LogLevelDebug
	case hclog.Info:
		logLevel = LogLevelInfo
	case hclog.Warn:
		logLevel = LogLevelWarn
	case hclog.Error:
		logLevel = LogLevelError
	default:
		logLevel = LogLevelInfo
	}

	l.logger.SetLevel(logLevel)
}

// GetHCLogger 获取原始的hclog.Logger
func (l *LegacyLoggerAdapter) GetHCLogger() hclog.Logger {
	return l.logger.GetHCLogger()
}

// NewLegacyLogger 创建一个新的旧版日志记录器
// 这个函数用于替代 pkg/logger.NewLogger
func NewLegacyLogger(name string, level hclog.Level) LegacyLogger {
	// 创建日志配置
	config := DefaultLogConfig()

	// 设置日志级别
	switch level {
	case hclog.Trace:
		config.Level = LogLevelTrace
	case hclog.Debug:
		config.Level = LogLevelDebug
	case hclog.Info:
		config.Level = LogLevelInfo
	case hclog.Warn:
		config.Level = LogLevelWarn
	case hclog.Error:
		config.Level = LogLevelError
	default:
		config.Level = LogLevelInfo
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	if err != nil {
		// 如果创建失败，使用默认配置
		logger, _ = NewEnhancedLogger(nil)
	}

	// 设置名称
	namedLogger := logger.Named(name)

	// 创建适配器
	return &LegacyLoggerAdapter{
		logger: namedLogger,
	}
}

// GetLegacyLogLevel 根据字符串获取日志级别
// 这个函数用于替代 pkg/logger.GetLogLevel
func GetLegacyLogLevel(level string) hclog.Level {
	switch level {
	case "trace":
		return hclog.Trace
	case "debug":
		return hclog.Debug
	case "info":
		return hclog.Info
	case "warn":
		return hclog.Warn
	case "error":
		return hclog.Error
	default:
		return hclog.Info
	}
}
