package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEnhancedLogger(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "logger-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelDebug,
		Format:           LogFormatJSON,
		Output:           LogOutputFile,
		FilePath:         filepath.Join(tempDir, "test.log"),
		RotationInterval: 24 * time.Hour,
		MaxSize:          1024 * 1024, // 1MB
		MaxAge:           7 * 24 * time.Hour,
		MaxBackups:       3,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 记录日志
	logger.Debug("这是一条调试日志", "key", "value")
	logger.Info("这是一条信息日志", "count", 42, "enabled", true)
	logger.Warn("这是一条警告日志", "status", "warning")
	logger.Error("这是一条错误日志", "error", "test error")

	// 读取日志文件
	data, err := ioutil.ReadFile(config.FilePath)
	assert.NoError(t, err)

	// 验证日志内容
	lines := strings.Split(string(data), "\n")
	assert.GreaterOrEqual(t, len(lines), 4)

	// 解析JSON日志
	var logEntry map[string]interface{}
	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	assert.NoError(t, err)

	// 验证日志字段
	assert.Equal(t, "这是一条调试日志", logEntry["message"])
	assert.Equal(t, "debug", logEntry["level"])
	assert.Equal(t, "value", logEntry["key"])
	assert.Contains(t, logEntry, "time")
}

func TestEnhancedLoggerWithContext(t *testing.T) {
	// 创建缓冲区
	var buf bytes.Buffer

	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelDebug,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 替换输出
	logger.writer = &buf

	// 创建上下文
	ctx := context.Background()
	ctx = ContextWithRequestID(ctx, "req-123")
	ctx = ContextWithUserID(ctx, "user-456")
	ctx = ContextWithSessionID(ctx, "session-789")
	ctx = ContextWithTraceID(ctx, "trace-abc")
	ctx = ContextWithSpanID(ctx, "span-def")

	// 创建带上下文的日志记录器
	ctxLogger := logger.WithContext(ctx)

	// 记录日志
	ctxLogger.Info("带上下文的日志", "key", "value")

	// 解析JSON日志
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	// 验证日志字段
	assert.Equal(t, "带上下文的日志", logEntry["message"])
	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "value", logEntry["key"])
	assert.Equal(t, "req-123", logEntry["request_id"])
	assert.Equal(t, "user-456", logEntry["user_id"])
	assert.Equal(t, "session-789", logEntry["session_id"])
	assert.Equal(t, "trace-abc", logEntry["trace_id"])
	assert.Equal(t, "span-def", logEntry["span_id"])
}

func TestEnhancedLoggerWithFields(t *testing.T) {
	// 创建缓冲区
	var buf bytes.Buffer

	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelDebug,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 替换输出
	logger.writer = &buf

	// 创建带字段的日志记录器
	fieldLogger := logger.WithField("component", "test").
		WithField("version", "1.0.0")

	// 记录日志
	fieldLogger.Info("带字段的日志")

	// 解析JSON日志
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	// 验证日志字段
	assert.Equal(t, "带字段的日志", logEntry["message"])
	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "test", logEntry["component"])
	assert.Equal(t, "1.0.0", logEntry["version"])
}

func TestEnhancedLoggerWithMultipleFields(t *testing.T) {
	// 创建缓冲区
	var buf bytes.Buffer

	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelDebug,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 替换输出
	logger.writer = &buf

	// 创建带多个字段的日志记录器
	fields := map[string]interface{}{
		"component": "test",
		"version":   "1.0.0",
		"enabled":   true,
		"count":     42,
	}
	fieldLogger := logger.WithFields(fields)

	// 记录日志
	fieldLogger.Info("带多个字段的日志")

	// 解析JSON日志
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	// 验证日志字段
	assert.Equal(t, "带多个字段的日志", logEntry["message"])
	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "test", logEntry["component"])
	assert.Equal(t, "1.0.0", logEntry["version"])
	assert.Equal(t, true, logEntry["enabled"])
	assert.Equal(t, float64(42), logEntry["count"])
}

func TestEnhancedLoggerNamed(t *testing.T) {
	// 创建缓冲区
	var buf bytes.Buffer

	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelDebug,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 替换输出
	logger.writer = &buf

	// 创建命名的日志记录器
	namedLogger := logger.Named("test")

	// 记录日志
	namedLogger.Info("命名的日志")

	// 解析JSON日志
	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	assert.NoError(t, err)

	// 验证日志字段
	assert.Equal(t, "命名的日志", logEntry["message"])
	assert.Equal(t, "info", logEntry["level"])
	assert.Contains(t, logEntry["name"], "test")
}

func TestEnhancedLoggerSetLevel(t *testing.T) {
	// 创建缓冲区
	var buf bytes.Buffer

	// 创建日志配置
	config := &LogConfig{
		Level:            LogLevelInfo,
		Format:           LogFormatJSON,
		Output:           LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       time.RFC3339,
	}

	// 创建日志记录器
	logger, err := NewEnhancedLogger(config)
	assert.NoError(t, err)
	defer logger.Close()

	// 替换输出
	logger.writer = &buf

	// 记录调试日志（不应该输出）
	logger.Debug("这是一条调试日志")
	assert.Empty(t, buf.String())

	// 设置日志级别
	logger.SetLevel(LogLevelDebug)

	// 记录调试日志（应该输出）
	logger.Debug("这是一条调试日志")
	assert.NotEmpty(t, buf.String())

	// 验证日志级别
	assert.Equal(t, LogLevelDebug, logger.GetLevel())
}
