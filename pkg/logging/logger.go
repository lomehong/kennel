package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/rs/zerolog"
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
	LogLevelFatal LogLevel = "fatal"
)

// LogFormat 日志格式
type LogFormat string

// 预定义日志格式
const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// LogOutput 日志输出
type LogOutput string

// 预定义日志输出
const (
	LogOutputStdout LogOutput = "stdout"
	LogOutputStderr LogOutput = "stderr"
	LogOutputFile   LogOutput = "file"
)

// LogContextKey 日志上下文键
type LogContextKey string

// 预定义日志上下文键
const (
	LogContextKeyRequestID LogContextKey = "request_id"
	LogContextKeyUserID    LogContextKey = "user_id"
	LogContextKeySessionID LogContextKey = "session_id"
	LogContextKeyTraceID   LogContextKey = "trace_id"
	LogContextKeySpanID    LogContextKey = "span_id"
)

// LogConfig 日志配置
type LogConfig struct {
	Level            LogLevel          // 日志级别
	Format           LogFormat         // 日志格式
	Output           LogOutput         // 日志输出
	FilePath         string            // 日志文件路径
	RotationInterval time.Duration     // 日志轮转间隔
	MaxSize          int64             // 日志文件最大大小（字节）
	MaxAge           time.Duration     // 日志文件最大保留时间
	MaxBackups       int               // 日志文件最大备份数量
	IncludeLocation  bool              // 是否包含代码位置
	IncludeTimestamp bool              // 是否包含时间戳
	TimeFormat       string            // 时间格式
	DefaultContext   map[string]string // 默认上下文
}

// DefaultLogConfig 默认日志配置
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:            LogLevelInfo,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		FilePath:         "logs/app.log",
		RotationInterval: 24 * time.Hour,
		MaxSize:          100 * 1024 * 1024,  // 100MB
		MaxAge:           7 * 24 * time.Hour, // 7天
		MaxBackups:       10,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
		DefaultContext:   make(map[string]string),
	}
}

// Logger 日志记录器接口
type Logger interface {
	// 基本日志方法
	Trace(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})

	// 带上下文的日志方法
	WithContext(ctx context.Context) Logger
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger

	// 获取子日志记录器
	Named(name string) Logger

	// 设置日志级别
	SetLevel(level LogLevel)

	// 获取日志级别
	GetLevel() LogLevel

	// 获取原始日志记录器
	GetHCLogger() hclog.Logger
	GetZeroLogger() *zerolog.Logger

	// 关闭日志记录器
	Close() error
}

// EnhancedLogger 增强日志记录器
type EnhancedLogger struct {
	hcLogger   hclog.Logger
	zeroLogger *zerolog.Logger
	config     *LogConfig
	writer     io.Writer
	rotator    *LogRotator
	fields     map[string]interface{}
	mu         sync.RWMutex
}

// NewEnhancedLogger 创建一个新的增强日志记录器
func NewEnhancedLogger(config *LogConfig) (*EnhancedLogger, error) {
	if config == nil {
		config = DefaultLogConfig()
	}

	// 创建日志输出
	writer, rotator, err := createLogWriter(config)
	if err != nil {
		return nil, fmt.Errorf("创建日志输出失败: %w", err)
	}

	// 创建zerolog日志记录器
	zeroLogger := zerolog.New(writer)
	if config.IncludeTimestamp {
		zeroLogger = zeroLogger.With().Timestamp().Logger()
	}

	// 设置日志级别
	zeroLevel := getZeroLogLevel(config.Level)
	zeroLogger = zeroLogger.Level(zeroLevel)

	// 设置时间格式
	zerolog.TimeFieldFormat = config.TimeFormat

	// 创建hclog日志记录器
	hcLevel := getHCLogLevel(config.Level)
	hcOptions := &hclog.LoggerOptions{
		Name:            "app",
		Level:           hcLevel,
		Output:          writer,
		IncludeLocation: config.IncludeLocation,
		TimeFormat:      config.TimeFormat,
	}

	if config.Format == LogFormatJSON {
		hcOptions.JSONFormat = true
	}

	hcLogger := hclog.New(hcOptions)

	// 创建增强日志记录器
	logger := &EnhancedLogger{
		hcLogger:   hcLogger,
		zeroLogger: &zeroLogger,
		config:     config,
		writer:     writer,
		rotator:    rotator,
		fields:     make(map[string]interface{}),
	}

	// 添加默认上下文
	for k, v := range config.DefaultContext {
		logger.fields[k] = v
	}

	return logger, nil
}

// createLogWriter 创建日志输出
func createLogWriter(config *LogConfig) (io.Writer, *LogRotator, error) {
	var writer io.Writer
	var rotator *LogRotator

	switch config.Output {
	case LogOutputStdout:
		writer = os.Stdout
	case LogOutputStderr:
		writer = os.Stderr
	case LogOutputFile:
		// 确保目录存在
		dir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, nil, fmt.Errorf("创建日志目录失败: %w", err)
		}

		// 创建日志轮转器
		rotator = NewLogRotator(config.FilePath, config.MaxSize, config.MaxBackups, config.MaxAge)
		writer = rotator
	default:
		return nil, nil, fmt.Errorf("不支持的日志输出: %s", config.Output)
	}

	// 根据格式创建输出
	if config.Format == LogFormatText {
		writer = zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: config.TimeFormat,
		}
	}

	return writer, rotator, nil
}

// getZeroLogLevel 获取zerolog日志级别
func getZeroLogLevel(level LogLevel) zerolog.Level {
	switch level {
	case LogLevelTrace:
		return zerolog.TraceLevel
	case LogLevelDebug:
		return zerolog.DebugLevel
	case LogLevelInfo:
		return zerolog.InfoLevel
	case LogLevelWarn:
		return zerolog.WarnLevel
	case LogLevelError:
		return zerolog.ErrorLevel
	case LogLevelFatal:
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// getHCLogLevel 获取hclog日志级别
func getHCLogLevel(level LogLevel) hclog.Level {
	switch level {
	case LogLevelTrace:
		return hclog.Trace
	case LogLevelDebug:
		return hclog.Debug
	case LogLevelInfo:
		return hclog.Info
	case LogLevelWarn:
		return hclog.Warn
	case LogLevelError:
		return hclog.Error
	default:
		return hclog.Info
	}
}

// Trace 记录跟踪级别日志
func (l *EnhancedLogger) Trace(msg string, args ...interface{}) {
	l.log(LogLevelTrace, msg, args...)
}

// Debug 记录调试级别日志
func (l *EnhancedLogger) Debug(msg string, args ...interface{}) {
	l.log(LogLevelDebug, msg, args...)
}

// Info 记录信息级别日志
func (l *EnhancedLogger) Info(msg string, args ...interface{}) {
	l.log(LogLevelInfo, msg, args...)
}

// Warn 记录警告级别日志
func (l *EnhancedLogger) Warn(msg string, args ...interface{}) {
	l.log(LogLevelWarn, msg, args...)
}

// Error 记录错误级别日志
func (l *EnhancedLogger) Error(msg string, args ...interface{}) {
	l.log(LogLevelError, msg, args...)
}

// Fatal 记录致命级别日志
func (l *EnhancedLogger) Fatal(msg string, args ...interface{}) {
	l.log(LogLevelFatal, msg, args...)
	os.Exit(1)
}

// log 记录日志
func (l *EnhancedLogger) log(level LogLevel, msg string, args ...interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 使用hclog记录日志
	switch level {
	case LogLevelTrace:
		l.hcLogger.Trace(msg, args...)
	case LogLevelDebug:
		l.hcLogger.Debug(msg, args...)
	case LogLevelInfo:
		l.hcLogger.Info(msg, args...)
	case LogLevelWarn:
		l.hcLogger.Warn(msg, args...)
	case LogLevelError:
		l.hcLogger.Error(msg, args...)
	case LogLevelFatal:
		l.hcLogger.Error(msg, args...)
	}

	// 使用zerolog记录日志
	event := l.getZeroLogEvent(level)
	if event == nil {
		return
	}

	// 添加字段
	for k, v := range l.fields {
		event = event.Interface(k, v)
	}

	// 添加调用位置
	if l.config.IncludeLocation {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			event = event.Str("file", file).Int("line", line)
		}
	}

	// 添加参数
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, ok := args[i].(string)
			if ok {
				event = event.Interface(key, args[i+1])
			}
		}
	}

	event.Msg(msg)
}

// getZeroLogEvent 获取zerolog事件
func (l *EnhancedLogger) getZeroLogEvent(level LogLevel) *zerolog.Event {
	switch level {
	case LogLevelTrace:
		return l.zeroLogger.Trace()
	case LogLevelDebug:
		return l.zeroLogger.Debug()
	case LogLevelInfo:
		return l.zeroLogger.Info()
	case LogLevelWarn:
		return l.zeroLogger.Warn()
	case LogLevelError:
		return l.zeroLogger.Error()
	case LogLevelFatal:
		return l.zeroLogger.Fatal()
	default:
		return nil
	}
}

// WithContext 创建带上下文的日志记录器
func (l *EnhancedLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	// 复制日志记录器
	newLogger := l.clone()

	// 从上下文中提取字段
	for _, key := range []LogContextKey{
		LogContextKeyRequestID,
		LogContextKeyUserID,
		LogContextKeySessionID,
		LogContextKeyTraceID,
		LogContextKeySpanID,
	} {
		if value := ctx.Value(key); value != nil {
			newLogger.fields[string(key)] = value
		}
	}

	return newLogger
}

// WithField 创建带字段的日志记录器
func (l *EnhancedLogger) WithField(key string, value interface{}) Logger {
	// 复制日志记录器
	newLogger := l.clone()

	// 添加字段
	newLogger.fields[key] = value

	return newLogger
}

// WithFields 创建带多个字段的日志记录器
func (l *EnhancedLogger) WithFields(fields map[string]interface{}) Logger {
	// 复制日志记录器
	newLogger := l.clone()

	// 添加字段
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// Named 创建命名的日志记录器
func (l *EnhancedLogger) Named(name string) Logger {
	// 复制日志记录器
	newLogger := l.clone()

	// 设置名称
	newLogger.hcLogger = l.hcLogger.Named(name)
	newZeroLogger := l.zeroLogger.With().Str("name", name).Logger()
	newLogger.zeroLogger = &newZeroLogger

	return newLogger
}

// SetLevel 设置日志级别
func (l *EnhancedLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 设置配置
	l.config.Level = level

	// 设置hclog级别
	hcLevel := getHCLogLevel(level)
	if hcLogger, ok := l.hcLogger.(interface{ SetLevel(level hclog.Level) }); ok {
		hcLogger.SetLevel(hcLevel)
	}

	// 设置zerolog级别
	zeroLevel := getZeroLogLevel(level)
	newZeroLogger := l.zeroLogger.Level(zeroLevel)
	l.zeroLogger = &newZeroLogger
}

// GetLevel 获取日志级别
func (l *EnhancedLogger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config.Level
}

// GetHCLogger 获取hclog日志记录器
func (l *EnhancedLogger) GetHCLogger() hclog.Logger {
	return l.hcLogger
}

// GetZeroLogger 获取zerolog日志记录器
func (l *EnhancedLogger) GetZeroLogger() *zerolog.Logger {
	return l.zeroLogger
}

// Close 关闭日志记录器
func (l *EnhancedLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 关闭轮转器
	if l.rotator != nil {
		return l.rotator.Close()
	}

	return nil
}

// clone 复制日志记录器
func (l *EnhancedLogger) clone() *EnhancedLogger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 复制字段
	fields := make(map[string]interface{})
	for k, v := range l.fields {
		fields[k] = v
	}

	// 创建新的日志记录器
	return &EnhancedLogger{
		hcLogger:   l.hcLogger,
		zeroLogger: l.zeroLogger,
		config:     l.config,
		writer:     l.writer,
		rotator:    l.rotator,
		fields:     fields,
	}
}
