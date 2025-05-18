package errors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestLogRecoveryHandler(t *testing.T) {
	logger := hclog.NewNullLogger()
	handler := NewLogRecoveryHandler(logger)

	// 测试处理panic
	err := handler.HandlePanic("test panic")
	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeCritical, err.(*AppError).Type)
	assert.Equal(t, "PANIC", err.(*AppError).Code)
	assert.Contains(t, err.(*AppError).Message, "test panic")
	assert.NotNil(t, err.(*AppError).Context["stack"])
}

func TestRecoveryHandlerChain(t *testing.T) {
	logger := hclog.NewNullLogger()
	handler1 := NewLogRecoveryHandler(logger)
	handler2 := NewLogRecoveryHandler(logger)
	chain := NewRecoveryHandlerChain(handler1, handler2)

	// 测试处理panic
	err := chain.HandlePanic("test panic")
	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeCritical, err.(*AppError).Type)
	assert.Equal(t, "PANIC", err.(*AppError).Code)
	assert.Contains(t, err.(*AppError).Message, "test panic")

	// 测试添加处理器
	handler3 := NewLogRecoveryHandler(logger)
	chain.AddHandler(handler3)
	assert.Len(t, chain.handlers, 3)
}

func TestRecoveryManager(t *testing.T) {
	logger := hclog.NewNullLogger()
	handler := NewLogRecoveryHandler(logger)
	manager := NewRecoveryManager(logger, handler)

	// 测试处理panic
	err := manager.HandlePanic("test panic")
	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeCritical, err.(*AppError).Type)
	assert.Equal(t, "PANIC", err.(*AppError).Code)
	assert.Contains(t, err.(*AppError).Message, "test panic")

	// 测试统计信息
	stats := manager.GetStats()
	assert.Equal(t, int64(1), stats.TotalPanics)
	assert.Equal(t, int64(1), stats.RecoveredPanics)
	assert.WithinDuration(t, time.Now(), stats.LastPanicTime, time.Second)
}

func TestRecoveryManager_SafeGo(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewRecoveryManager(logger, nil)

	// 测试正常执行
	done := make(chan bool)
	manager.SafeGo(func() {
		done <- true
	})
	assert.True(t, <-done)

	// 测试panic恢复
	manager.SafeGo(func() {
		panic("test panic")
	})

	// 等待panic处理
	time.Sleep(100 * time.Millisecond)

	// 验证统计信息
	stats := manager.GetStats()
	assert.Equal(t, int64(1), stats.TotalPanics)
	assert.Equal(t, int64(1), stats.RecoveredPanics)
}

func TestRecoveryManager_SafeGoWithContext(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewRecoveryManager(logger, nil)

	// 创建上下文
	ctx := context.Background()

	// 测试正常执行
	done := make(chan bool)
	manager.SafeGoWithContext(ctx, func(ctx context.Context) {
		done <- true
	})
	assert.True(t, <-done)

	// 测试panic恢复
	manager.SafeGoWithContext(ctx, func(ctx context.Context) {
		panic("test panic")
	})

	// 等待panic处理
	time.Sleep(100 * time.Millisecond)

	// 验证统计信息
	stats := manager.GetStats()
	assert.Equal(t, int64(1), stats.TotalPanics)
	assert.Equal(t, int64(1), stats.RecoveredPanics)
}

func TestRecoveryManager_SafeExec(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewRecoveryManager(logger, nil)

	// 测试正常执行
	result := "success"
	err := manager.SafeExec(func() error {
		return nil
	})
	assert.Nil(t, err)

	// 测试返回错误
	expectedErr := fmt.Errorf("test error")
	err = manager.SafeExec(func() error {
		return expectedErr
	})
	assert.Equal(t, expectedErr, err)

	// 测试panic恢复
	err = manager.SafeExec(func() error {
		panic("test panic")
		return nil
	})
	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeCritical, err.(*AppError).Type)
	assert.Equal(t, "PANIC", err.(*AppError).Code)
	assert.Contains(t, err.(*AppError).Message, "test panic")

	// 验证统计信息
	stats := manager.GetStats()
	assert.Equal(t, int64(1), stats.TotalPanics)
	assert.Equal(t, int64(1), stats.RecoveredPanics)
}

func TestRecoveryManager_SafeExecWithContext(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewRecoveryManager(logger, nil)

	// 创建上下文
	ctx := context.Background()

	// 测试正常执行
	err := manager.SafeExecWithContext(ctx, func(ctx context.Context) error {
		return nil
	})
	assert.Nil(t, err)

	// 测试返回错误
	expectedErr := fmt.Errorf("test error")
	err = manager.SafeExecWithContext(ctx, func(ctx context.Context) error {
		return expectedErr
	})
	assert.Equal(t, expectedErr, err)

	// 测试panic恢复
	err = manager.SafeExecWithContext(ctx, func(ctx context.Context) error {
		panic("test panic")
		return nil
	})
	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeCritical, err.(*AppError).Type)
	assert.Equal(t, "PANIC", err.(*AppError).Code)
	assert.Contains(t, err.(*AppError).Message, "test panic")

	// 验证统计信息
	stats := manager.GetStats()
	assert.Equal(t, int64(1), stats.TotalPanics)
	assert.Equal(t, int64(1), stats.RecoveredPanics)
}

func TestDefaultRecoveryHandler(t *testing.T) {
	logger := hclog.NewNullLogger()
	handler := DefaultRecoveryHandler(logger)

	// 验证处理器类型
	assert.IsType(t, &LogRecoveryHandler{}, handler)

	// 测试处理panic
	err := handler.HandlePanic("test panic")
	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeCritical, err.(*AppError).Type)
	assert.Equal(t, "PANIC", err.(*AppError).Code)
	assert.Contains(t, err.(*AppError).Message, "test panic")
}

func TestDefaultRecoveryManager(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := DefaultRecoveryManager(logger)

	// 验证管理器配置
	assert.NotNil(t, manager.handler)
	assert.IsType(t, &LogRecoveryHandler{}, manager.handler)

	// 测试处理panic
	err := manager.HandlePanic("test panic")
	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeCritical, err.(*AppError).Type)
	assert.Equal(t, "PANIC", err.(*AppError).Code)
	assert.Contains(t, err.(*AppError).Message, "test panic")
}

func TestSafeGo(t *testing.T) {
	// 测试正常执行
	done := make(chan bool)
	SafeGo(func() {
		done <- true
	})
	assert.True(t, <-done)

	// 测试panic恢复
	SafeGo(func() {
		panic("test panic")
	})

	// 等待panic处理
	time.Sleep(100 * time.Millisecond)
}

func TestSafeGoWithContext(t *testing.T) {
	// 创建上下文
	ctx := context.Background()

	// 测试正常执行
	done := make(chan bool)
	SafeGoWithContext(ctx, func(ctx context.Context) {
		done <- true
	})
	assert.True(t, <-done)

	// 测试panic恢复
	SafeGoWithContext(ctx, func(ctx context.Context) {
		panic("test panic")
	})

	// 等待panic处理
	time.Sleep(100 * time.Millisecond)
}

func TestSafeExec(t *testing.T) {
	// 测试正常执行
	err := SafeExec(func() error {
		return nil
	})
	assert.Nil(t, err)

	// 测试返回错误
	expectedErr := fmt.Errorf("test error")
	err = SafeExec(func() error {
		return expectedErr
	})
	assert.Equal(t, expectedErr, err)

	// 测试panic恢复
	err = SafeExec(func() error {
		panic("test panic")
		return nil
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "panic: test panic")
}

func TestSafeExecWithContext(t *testing.T) {
	// 创建上下文
	ctx := context.Background()

	// 测试正常执行
	err := SafeExecWithContext(ctx, func(ctx context.Context) error {
		return nil
	})
	assert.Nil(t, err)

	// 测试返回错误
	expectedErr := fmt.Errorf("test error")
	err = SafeExecWithContext(ctx, func(ctx context.Context) error {
		return expectedErr
	})
	assert.Equal(t, expectedErr, err)

	// 测试panic恢复
	err = SafeExecWithContext(ctx, func(ctx context.Context) error {
		panic("test panic")
		return nil
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "panic: test panic")
}

func TestGetGoroutineID(t *testing.T) {
	id := GetGoroutineID()
	assert.NotZero(t, id)
}
