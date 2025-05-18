package logger

import (
	"github.com/hashicorp/go-hclog"
)

// Logger 定义了日志接口
type Logger interface {
	// 基本日志方法
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
}

// AppLogger 是Logger接口的实现
type AppLogger struct {
	logger hclog.Logger
}

// NewLogger 创建一个新的日志器
func NewLogger(name string, level hclog.Level) Logger {
	return &AppLogger{
		logger: hclog.New(&hclog.LoggerOptions{
			Name:   name,
			Level:  level,
			Output: hclog.DefaultOutput,
		}),
	}
}

// Trace 记录跟踪级别的日志
func (l *AppLogger) Trace(msg string, args ...interface{}) {
	l.logger.Trace(msg, args...)
}

// Debug 记录调试级别的日志
func (l *AppLogger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

// Info 记录信息级别的日志
func (l *AppLogger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

// Warn 记录警告级别的日志
func (l *AppLogger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

// Error 记录错误级别的日志
func (l *AppLogger) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

// With 创建带有上下文的新日志器
func (l *AppLogger) With(args ...interface{}) Logger {
	return &AppLogger{
		logger: l.logger.With(args...),
	}
}

// SetLevel 设置日志级别
func (l *AppLogger) SetLevel(level hclog.Level) {
	l.logger.SetLevel(level)
}

// GetHCLogger 获取原始的hclog.Logger
func (l *AppLogger) GetHCLogger() hclog.Logger {
	return l.logger
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
