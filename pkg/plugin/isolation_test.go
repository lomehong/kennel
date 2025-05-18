package plugin

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestPluginIsolator_ExecuteFunc(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()
	isolator := NewPluginIsolator(config, WithLogger(logger))

	// 测试正常执行
	err := isolator.ExecuteFunc("test-plugin", func() error {
		return nil
	})
	assert.NoError(t, err)

	// 测试返回错误
	expectedErr := fmt.Errorf("test error")
	err = isolator.ExecuteFunc("test-plugin", func() error {
		return expectedErr
	})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// 测试超时
	config.TimeoutDuration = 100 * time.Millisecond
	isolator = NewPluginIsolator(config, WithLogger(logger))
	err = isolator.ExecuteFunc("test-plugin", func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "超时")

	// 测试panic
	err = isolator.ExecuteFunc("test-plugin", func() error {
		panic("test panic")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic")

	// 获取统计信息
	stats := isolator.GetStats()
	assert.Equal(t, int64(4), stats.TotalCalls)
	assert.Equal(t, int64(1), stats.SuccessfulCalls)
	assert.Equal(t, int64(3), stats.FailedCalls)
	assert.Equal(t, int64(1), stats.Timeouts)
	assert.Equal(t, int64(1), stats.Panics)
}

func TestPluginIsolator_ExecuteWithBasicIsolation(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()
	config.Level = IsolationLevelBasic
	isolator := NewPluginIsolator(config, WithLogger(logger))

	// 测试正常执行
	err := isolator.executeWithBasicIsolation("test-plugin", func() error {
		return nil
	})
	assert.NoError(t, err)

	// 测试返回错误
	expectedErr := fmt.Errorf("test error")
	err = isolator.executeWithBasicIsolation("test-plugin", func() error {
		return expectedErr
	})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// 测试超时
	config.TimeoutDuration = 100 * time.Millisecond
	isolator = NewPluginIsolator(config, WithLogger(logger))
	err = isolator.executeWithBasicIsolation("test-plugin", func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "超时")

	// 测试panic
	err = isolator.executeWithBasicIsolation("test-plugin", func() error {
		panic("test panic")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
}

func TestPluginIsolator_ExecuteWithStrictIsolation(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()
	config.Level = IsolationLevelStrict
	isolator := NewPluginIsolator(config, WithLogger(logger))

	// 没有工作池，应该回退到基本隔离
	err := isolator.executeWithStrictIsolation("test-plugin", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestPluginIsolator_ExecuteWithCompleteIsolation(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()
	config.Level = IsolationLevelComplete
	isolator := NewPluginIsolator(config, WithLogger(logger))

	// 完全隔离尚未实现，应该回退到基本隔离
	err := isolator.executeWithCompleteIsolation("test-plugin", func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestPluginIsolator_WithErrorHandler(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()

	// 创建错误处理器
	errorHandler := &testErrorHandler{}
	config.ErrorHandler = errorHandler

	isolator := NewPluginIsolator(config, WithLogger(logger))

	// 测试错误处理
	expectedErr := fmt.Errorf("test error")
	err := isolator.ExecuteFunc("test-plugin", func() error {
		return expectedErr
	})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.True(t, errorHandler.called)
	assert.Equal(t, expectedErr, errorHandler.lastError)
}

func TestPluginIsolator_WithRecoveryHandler(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()

	// 创建恢复处理器
	recoveryHandler := &testRecoveryHandler{}
	config.RecoveryHandler = recoveryHandler

	isolator := NewPluginIsolator(config, WithLogger(logger))

	// 测试panic恢复
	err := isolator.ExecuteFunc("test-plugin", func() error {
		panic("test panic")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test panic")
	assert.True(t, recoveryHandler.called)
	assert.Equal(t, "test panic", recoveryHandler.lastPanic)
}

// 测试辅助类型

// testErrorHandler 测试用的错误处理器
type testErrorHandler struct {
	called    bool
	lastError error
}

func (h *testErrorHandler) Handle(err error) error {
	h.called = true
	h.lastError = err
	return err
}

func (h *testErrorHandler) Name() string {
	return "test_error_handler"
}

// testRecoveryHandler 测试用的恢复处理器
type testRecoveryHandler struct {
	called    bool
	lastPanic interface{}
}

func (h *testRecoveryHandler) HandlePanic(p interface{}) error {
	h.called = true
	h.lastPanic = p
	return errors.New(errors.ErrorTypeCritical, "PANIC", fmt.Sprintf("Panic: %v", p))
}

func (h *testRecoveryHandler) Name() string {
	return "test_recovery_handler"
}
