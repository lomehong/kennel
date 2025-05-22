package logger

import (
	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/logging"
)

// Logger 接口定义了日志记录器的基本方法
type Logger interface {
	// 日志级别方法
	Trace(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})

	// 创建带有上下文的新日志器
	With(args ...interface{}) Logger

	// 设置日志级别
	SetLevel(level hclog.Level)

	// 获取原始的hclog.Logger
	GetHCLogger() hclog.Logger

	// 关闭日志记录器
	Close() error
}

// LoggingAdapter 是一个适配器，用于将 logging.Logger 适配为 logger.Logger
type LoggingAdapter struct {
	newLogger logging.Logger
}

// NewLogger 创建一个新的日志器
// 这个函数现在使用 pkg/logging 包实现
func NewLogger(name string, level hclog.Level) Logger {
	// 创建默认日志配置
	logConfig := logging.DefaultLogConfig()

	// 将旧版日志级别转换为新版日志级别
	switch level {
	case hclog.Trace:
		logConfig.Level = logging.LogLevelTrace
	case hclog.Debug:
		logConfig.Level = logging.LogLevelDebug
	case hclog.Info:
		logConfig.Level = logging.LogLevelInfo
	case hclog.Warn:
		logConfig.Level = logging.LogLevelWarn
	case hclog.Error:
		logConfig.Level = logging.LogLevelError
	default:
		logConfig.Level = logging.LogLevelInfo
	}

	// 创建增强日志记录器
	enhancedLogger, err := logging.NewEnhancedLogger(logConfig)
	if err != nil {
		// 如果创建失败，使用默认配置
		enhancedLogger, _ = logging.NewEnhancedLogger(nil)
	}

	// 设置名称
	namedLogger := enhancedLogger.Named(name)

	// 创建适配器
	return &LoggingAdapter{
		newLogger: namedLogger,
	}
}

// Trace 记录跟踪级别日志
func (l *LoggingAdapter) Trace(msg string, args ...interface{}) {
	l.newLogger.Trace(msg, args...)
}

// Debug 记录调试级别日志
func (l *LoggingAdapter) Debug(msg string, args ...interface{}) {
	l.newLogger.Debug(msg, args...)
}

// Info 记录信息级别日志
func (l *LoggingAdapter) Info(msg string, args ...interface{}) {
	l.newLogger.Info(msg, args...)
}

// Warn 记录警告级别日志
func (l *LoggingAdapter) Warn(msg string, args ...interface{}) {
	l.newLogger.Warn(msg, args...)
}

// Error 记录错误级别日志
func (l *LoggingAdapter) Error(msg string, args ...interface{}) {
	l.newLogger.Error(msg, args...)
}

// With 创建带有上下文的新日志器
func (l *LoggingAdapter) With(args ...interface{}) Logger {
	// 创建字段映射
	fields := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, ok := args[i].(string)
			if ok {
				fields[key] = args[i+1]
			}
		}
	}

	// 使用 WithFields 方法
	return &LoggingAdapter{
		newLogger: l.newLogger.WithFields(fields),
	}
}

// SetLevel 设置日志级别
func (l *LoggingAdapter) SetLevel(level hclog.Level) {
	// 将旧版日志级别转换为新版日志级别
	var logLevel logging.LogLevel
	switch level {
	case hclog.Trace:
		logLevel = logging.LogLevelTrace
	case hclog.Debug:
		logLevel = logging.LogLevelDebug
	case hclog.Info:
		logLevel = logging.LogLevelInfo
	case hclog.Warn:
		logLevel = logging.LogLevelWarn
	case hclog.Error:
		logLevel = logging.LogLevelError
	default:
		logLevel = logging.LogLevelInfo
	}
	l.newLogger.SetLevel(logLevel)
}

// GetHCLogger 获取原始的hclog.Logger
func (l *LoggingAdapter) GetHCLogger() hclog.Logger {
	// 新版日志系统没有 GetHCLogger 方法，返回 nil
	return nil
}

// Close 关闭日志记录器
func (l *LoggingAdapter) Close() error {
	// 调用新版日志系统的 Close 方法
	return l.newLogger.Close()
}

// GetLogLevel 根据字符串获取日志级别
func GetLogLevel(level string) hclog.Level {
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
