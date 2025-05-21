package control

import (
	"io"

	"github.com/hashicorp/go-hclog"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// LoggerAdapter 将 hclog.Logger 适配为 sdk.Logger
type LoggerAdapter struct {
	logger hclog.Logger
}

// NewLoggerAdapter 创建一个新的日志适配器
func NewLoggerAdapter(logger hclog.Logger) sdk.Logger {
	return &LoggerAdapter{
		logger: logger,
	}
}

// Trace 记录跟踪级别日志
func (l *LoggerAdapter) Trace(msg string, args ...interface{}) {
	l.logger.Trace(msg, args...)
}

// Debug 记录调试级别日志
func (l *LoggerAdapter) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, args...)
}

// Info 记录信息级别日志
func (l *LoggerAdapter) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, args...)
}

// Warn 记录警告级别日志
func (l *LoggerAdapter) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, args...)
}

// Error 记录错误级别日志
func (l *LoggerAdapter) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, args...)
}

// With 返回带有附加字段的新日志记录器
func (l *LoggerAdapter) With(args ...interface{}) sdk.Logger {
	return &LoggerAdapter{
		logger: l.logger.With(args...),
	}
}

// Named 返回带有名称的新日志记录器
func (l *LoggerAdapter) Named(name string) sdk.Logger {
	return &LoggerAdapter{
		logger: l.logger.Named(name),
	}
}

// GetLevel 获取日志级别
func (l *LoggerAdapter) GetLevel() sdk.LogLevel {
	// hclog.Logger 接口没有 GetLevel 方法，我们返回一个默认值
	// 实际应用中可以根据需要调整默认级别
	return sdk.LogLevelInfo
}

// SetLevel 设置日志级别
func (l *LoggerAdapter) SetLevel(level sdk.LogLevel) {
	// hclog 不支持直接设置级别，这是一个空实现
}

// SetOutput 设置日志输出
func (l *LoggerAdapter) SetOutput(output io.Writer) {
	// hclog 不支持直接设置输出，这是一个空实现
}
