package sdk

import (
	"io"
	"os"

	"github.com/hashicorp/go-hclog"
)

// 定义日志级别常量
const (
	Off = 6 // 与 hclog.Off 相同的值
)

// LogLevel 日志级别
type LogLevel string

// 预定义日志级别
const (
	LogLevelTrace LogLevel = "trace"
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelNone  LogLevel = "none"
)

// Logger 日志接口
type Logger interface {
	// Trace 记录跟踪级别日志
	Trace(msg string, args ...interface{})

	// Debug 记录调试级别日志
	Debug(msg string, args ...interface{})

	// Info 记录信息级别日志
	Info(msg string, args ...interface{})

	// Warn 记录警告级别日志
	Warn(msg string, args ...interface{})

	// Error 记录错误级别日志
	Error(msg string, args ...interface{})

	// With 返回带有附加字段的新日志记录器
	With(args ...interface{}) Logger

	// Named 返回带有名称的新日志记录器
	Named(name string) Logger

	// GetLevel 获取日志级别
	GetLevel() LogLevel

	// SetLevel 设置日志级别
	SetLevel(level LogLevel)

	// SetOutput 设置日志输出
	SetOutput(output io.Writer)
}

// defaultLogger 默认日志记录器
type defaultLogger struct {
	logger hclog.Logger
}

// NewLogger 创建新的日志记录器
func NewLogger(name string, level LogLevel) Logger {
	hclogLevel := hclog.Info
	switch level {
	case LogLevelTrace:
		hclogLevel = hclog.Trace
	case LogLevelDebug:
		hclogLevel = hclog.Debug
	case LogLevelInfo:
		hclogLevel = hclog.Info
	case LogLevelWarn:
		hclogLevel = hclog.Warn
	case LogLevelError:
		hclogLevel = hclog.Error
	case LogLevelNone:
		hclogLevel = Off
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Name:   name,
		Level:  hclogLevel,
		Output: os.Stderr,
	})

	return &defaultLogger{
		logger: logger,
	}
}

// Trace 记录跟踪级别日志
func (l *defaultLogger) Trace(msg string, args ...interface{}) {
	l.logger.Trace(msg, args...)
}

// Debug 记录调试级别日志
func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

// Info 记录信息级别日志
func (l *defaultLogger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

// Warn 记录警告级别日志
func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

// Error 记录错误级别日志
func (l *defaultLogger) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

// With 返回带有附加字段的新日志记录器
func (l *defaultLogger) With(args ...interface{}) Logger {
	return &defaultLogger{
		logger: l.logger.With(args...),
	}
}

// Named 返回带有名称的新日志记录器
func (l *defaultLogger) Named(name string) Logger {
	return &defaultLogger{
		logger: l.logger.Named(name),
	}
}

// GetLevel 获取日志级别
func (l *defaultLogger) GetLevel() LogLevel {
	// hclog 没有提供 GetLevel 方法，这里返回默认值
	return LogLevelInfo
}

// SetLevel 设置日志级别
func (l *defaultLogger) SetLevel(level LogLevel) {
	// hclog不支持直接设置级别，这里是一个空实现
}

// SetOutput 设置日志输出
func (l *defaultLogger) SetOutput(output io.Writer) {
	// hclog不支持直接设置输出，这里是一个空实现
}
