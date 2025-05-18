package errors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestLogErrorHandler(t *testing.T) {
	logger := hclog.NewNullLogger()
	handler := NewLogErrorHandler(logger)

	// 测试处理AppError
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	result := handler.Handle(err)
	assert.True(t, IsHandled(result))
	assert.Equal(t, "log", result.(*AppError).HandlerName)

	// 测试处理普通错误
	err2 := fmt.Errorf("regular error")
	result = handler.Handle(err2)
	assert.Equal(t, "regular error", result.Error())

	// 测试处理nil错误
	result = handler.Handle(nil)
	assert.Nil(t, result)
}

func TestRetryErrorHandler(t *testing.T) {
	handler := NewRetryErrorHandler(3, 1*time.Second)

	// 测试处理可重试错误
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	result := handler.Handle(err)
	assert.True(t, IsHandled(result))
	assert.Equal(t, "retry", result.(*AppError).HandlerName)

	// 测试处理不可重试错误
	err = New(ErrorTypePermanent, "TEST_CODE", "Test message")
	result = handler.Handle(err)
	assert.Equal(t, err, result)

	// 测试处理显式设置为可重试的错误
	err = New(ErrorTypePermanent, "TEST_CODE", "Test message").WithRetry(5, 2*time.Second)
	result = handler.Handle(err)
	assert.True(t, IsHandled(result))
	assert.Equal(t, "retry", result.(*AppError).HandlerName)

	// 测试处理nil错误
	result = handler.Handle(nil)
	assert.Nil(t, result)
}

func TestPanicErrorHandler(t *testing.T) {
	logger := hclog.NewNullLogger()
	handler := NewPanicErrorHandler(logger)

	// 测试处理非严重错误
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	result := handler.Handle(err)
	assert.Equal(t, err, result)

	// 测试处理严重错误（会触发panic）
	err = New(ErrorTypeCritical, "CRITICAL", "Critical error")
	assert.Panics(t, func() {
		handler.Handle(err)
	})

	// 测试处理nil错误
	result = handler.Handle(nil)
	assert.Nil(t, result)
}

func TestErrorHandlerChain(t *testing.T) {
	logger := hclog.NewNullLogger()
	logHandler := NewLogErrorHandler(logger)
	retryHandler := NewRetryErrorHandler(3, 1*time.Second)
	chain := NewErrorHandlerChain(logHandler, retryHandler)

	// 测试处理可重试错误
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	result := chain.Handle(err)
	assert.True(t, IsHandled(result))
	assert.Equal(t, "retry", result.(*AppError).HandlerName)

	// 测试添加处理器
	panicHandler := NewPanicErrorHandler(logger)
	chain.AddHandler(panicHandler)
	assert.Len(t, chain.handlers, 3)

	// 测试处理nil错误
	result = chain.Handle(nil)
	assert.Nil(t, result)
}

func TestErrorHandlerRegistry(t *testing.T) {
	logger := hclog.NewNullLogger()
	registry := NewErrorHandlerRegistry(logger)

	// 注册处理器
	logHandler := NewLogErrorHandler(logger)
	retryHandler := NewRetryErrorHandler(3, 1*time.Second)
	panicHandler := NewPanicErrorHandler(logger)

	registry.RegisterHandler(ErrorTypeTemporary, retryHandler)
	registry.RegisterHandler(ErrorTypePermanent, logHandler)
	registry.RegisterHandler(ErrorTypeCritical, panicHandler)

	// 测试获取处理器
	handler, ok := registry.GetHandler(ErrorTypeTemporary)
	assert.True(t, ok)
	assert.Equal(t, "retry", handler.Name())

	handler, ok = registry.GetHandler(ErrorTypePermanent)
	assert.True(t, ok)
	assert.Equal(t, "log", handler.Name())

	handler, ok = registry.GetHandler(ErrorTypeCritical)
	assert.True(t, ok)
	assert.Equal(t, "panic", handler.Name())

	// 测试获取不存在的处理器
	handler, ok = registry.GetHandler(ErrorTypeValidation)
	assert.False(t, ok)
	assert.Nil(t, handler)

	// 测试处理错误
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	result := registry.Handle(err)
	assert.True(t, IsHandled(result))
	assert.Equal(t, "retry", result.(*AppError).HandlerName)

	// 测试处理不存在处理器的错误类型
	err = New(ErrorTypeValidation, "TEST_CODE", "Test message")
	result = registry.Handle(err)
	assert.Equal(t, err, result)

	// 测试处理普通错误
	err2 := fmt.Errorf("regular error")
	result = registry.Handle(err2)
	assert.Equal(t, "regular error", result.Error())

	// 测试处理nil错误
	result = registry.Handle(nil)
	assert.Nil(t, result)
}

func TestErrorHandlerRegistry_HandleWithContext(t *testing.T) {
	logger := hclog.NewNullLogger()
	registry := NewErrorHandlerRegistry(logger)

	// 注册处理器
	logHandler := NewLogErrorHandler(logger)
	registry.RegisterHandler(ErrorTypeTemporary, logHandler)

	// 创建上下文
	ctx := context.Background()

	// 测试处理错误
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	result := registry.HandleWithContext(ctx, err)
	assert.True(t, IsHandled(result))
	assert.Equal(t, "log", result.(*AppError).HandlerName)

	// 测试处理已取消上下文
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	result = registry.HandleWithContext(cancelCtx, err)
	assert.Equal(t, err, result)

	// 测试处理nil错误
	result = registry.HandleWithContext(ctx, nil)
	assert.Nil(t, result)
}

func TestDefaultErrorHandler(t *testing.T) {
	logger := hclog.NewNullLogger()
	handler := DefaultErrorHandler(logger)

	// 验证处理器类型
	assert.IsType(t, &ErrorHandlerChain{}, handler)
	chain := handler.(*ErrorHandlerChain)
	assert.Len(t, chain.handlers, 2)
	assert.IsType(t, &LogErrorHandler{}, chain.handlers[0])
	assert.IsType(t, &RetryErrorHandler{}, chain.handlers[1])

	// 测试处理错误
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	result := handler.Handle(err)
	assert.True(t, IsHandled(result))
}

func TestDefaultErrorHandlerRegistry(t *testing.T) {
	logger := hclog.NewNullLogger()
	registry := DefaultErrorHandlerRegistry(logger)

	// 验证注册的处理器
	for _, errorType := range []ErrorType{
		ErrorTypeTemporary,
		ErrorTypePermanent,
		ErrorTypeValidation,
		ErrorTypeNotFound,
		ErrorTypePermission,
		ErrorTypeInternal,
		ErrorTypeExternal,
	} {
		handler, ok := registry.GetHandler(errorType)
		assert.True(t, ok)
		assert.IsType(t, &ErrorHandlerChain{}, handler)
	}

	// 验证严重错误处理器
	handler, ok := registry.GetHandler(ErrorTypeCritical)
	assert.True(t, ok)
	assert.IsType(t, &PanicErrorHandler{}, handler)

	// 测试处理错误
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	result := registry.Handle(err)
	assert.True(t, IsHandled(result))
}
