package errors

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeTemporary, err.Type)
	assert.Equal(t, "TEST_CODE", err.Code)
	assert.Equal(t, "Test message", err.Message)
	assert.NotEmpty(t, err.Stack)
	assert.WithinDuration(t, time.Now(), err.Time, time.Second)
}

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	err := Wrap(originalErr, ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.NotNil(t, err)
	assert.Equal(t, ErrorTypeTemporary, err.Type)
	assert.Equal(t, "TEST_CODE", err.Code)
	assert.Equal(t, "Test message", err.Message)
	assert.Equal(t, originalErr, err.Cause)
	assert.NotEmpty(t, err.Stack)
	assert.WithinDuration(t, time.Now(), err.Time, time.Second)

	// 测试包装AppError
	appErr := New(ErrorTypeCritical, "CRITICAL", "Critical error")
	wrappedErr := Wrap(appErr, ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.NotNil(t, wrappedErr)
	assert.Equal(t, ErrorTypeCritical, wrappedErr.Type) // 保留原始错误类型
	assert.Equal(t, "CRITICAL", wrappedErr.Code)        // 保留原始错误代码
	assert.Equal(t, "Test message", wrappedErr.Message)
	assert.Equal(t, appErr.Cause, wrappedErr.Cause)
	assert.NotEmpty(t, wrappedErr.Stack)
	assert.WithinDuration(t, time.Now(), wrappedErr.Time, time.Second)
}

func TestWrapIfErr(t *testing.T) {
	// 测试nil错误
	err := WrapIfErr(nil, ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.Nil(t, err)

	// 测试非nil错误
	originalErr := fmt.Errorf("original error")
	err = WrapIfErr(originalErr, ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.NotNil(t, err)
	assert.Equal(t, "[TEST_CODE] Test message: original error", err.Error())
}

func TestAppError_Error(t *testing.T) {
	// 测试无原因错误
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.Equal(t, "[TEST_CODE] Test message", err.Error())

	// 测试有原因错误
	originalErr := fmt.Errorf("original error")
	err = Wrap(originalErr, ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.Equal(t, "[TEST_CODE] Test message: original error", err.Error())
}

func TestAppError_Unwrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	err := Wrap(originalErr, ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.Equal(t, originalErr, err.Unwrap())
}

func TestAppError_WithContext(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	err = err.WithContext("key", "value")
	assert.Equal(t, "value", err.Context["key"])
}

func TestAppError_WithRetry(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	err = err.WithRetry(3, 1*time.Second)
	assert.True(t, err.Retriable)
	assert.Equal(t, 3, err.MaxRetries)
	assert.Equal(t, 1*time.Second, err.RetryDelay)
}

func TestAppError_WithHandled(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	err = err.WithHandled("test_handler")
	assert.True(t, err.Handled)
	assert.Equal(t, "test_handler", err.HandlerName)
}

func TestAppError_IsTemporary(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.True(t, err.IsTemporary())
	assert.False(t, err.IsPermanent())
	assert.False(t, err.IsCritical())
}

func TestAppError_IsPermanent(t *testing.T) {
	err := New(ErrorTypePermanent, "TEST_CODE", "Test message")
	assert.False(t, err.IsTemporary())
	assert.True(t, err.IsPermanent())
	assert.False(t, err.IsCritical())
}

func TestAppError_IsCritical(t *testing.T) {
	err := New(ErrorTypeCritical, "TEST_CODE", "Test message")
	assert.False(t, err.IsTemporary())
	assert.False(t, err.IsPermanent())
	assert.True(t, err.IsCritical())
}

func TestAppError_IsRetriable(t *testing.T) {
	// 临时错误默认可重试
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.True(t, err.IsRetriable())

	// 永久错误默认不可重试
	err = New(ErrorTypePermanent, "TEST_CODE", "Test message")
	assert.False(t, err.IsRetriable())

	// 显式设置可重试
	err = New(ErrorTypePermanent, "TEST_CODE", "Test message").WithRetry(3, 1*time.Second)
	assert.True(t, err.IsRetriable())
}

func TestIs(t *testing.T) {
	err1 := fmt.Errorf("error 1")
	err2 := fmt.Errorf("error 2: %w", err1)
	assert.True(t, Is(err2, err1))
	assert.False(t, Is(err1, err2))
}

func TestAs(t *testing.T) {
	var appErr *AppError
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.True(t, As(err, &appErr))
	assert.Equal(t, ErrorTypeTemporary, appErr.Type)
	assert.Equal(t, "TEST_CODE", appErr.Code)
	assert.Equal(t, "Test message", appErr.Message)
}

func TestIsType(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.True(t, IsType(err, ErrorTypeTemporary))
	assert.False(t, IsType(err, ErrorTypePermanent))

	// 测试非AppError
	err2 := fmt.Errorf("regular error")
	assert.False(t, IsType(err2, ErrorTypeTemporary))
}

func TestGetContext(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message").WithContext("key", "value")
	context := GetContext(err)
	assert.NotNil(t, context)
	assert.Equal(t, "value", context["key"])

	// 测试非AppError
	err2 := fmt.Errorf("regular error")
	context = GetContext(err2)
	assert.Nil(t, context)
}

func TestGetStack(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	stack := GetStack(err)
	assert.NotEmpty(t, stack)

	// 测试非AppError
	err2 := fmt.Errorf("regular error")
	stack = GetStack(err2)
	assert.Empty(t, stack)
}

func TestIsRetriable(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.True(t, IsRetriable(err))

	err = New(ErrorTypePermanent, "TEST_CODE", "Test message")
	assert.False(t, IsRetriable(err))

	err = New(ErrorTypePermanent, "TEST_CODE", "Test message").WithRetry(3, 1*time.Second)
	assert.True(t, IsRetriable(err))

	// 测试非AppError
	err2 := fmt.Errorf("regular error")
	assert.False(t, IsRetriable(err2))
}

func TestGetRetryInfo(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message").WithRetry(3, 1*time.Second)
	retriable, maxRetries, retryDelay := GetRetryInfo(err)
	assert.True(t, retriable)
	assert.Equal(t, 3, maxRetries)
	assert.Equal(t, 1*time.Second, retryDelay)

	// 测试非AppError
	err2 := fmt.Errorf("regular error")
	retriable, maxRetries, retryDelay = GetRetryInfo(err2)
	assert.False(t, retriable)
	assert.Equal(t, 0, maxRetries)
	assert.Equal(t, time.Duration(0), retryDelay)
}

func TestIsHandled(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	assert.False(t, IsHandled(err))

	err = err.WithHandled("test_handler")
	assert.True(t, IsHandled(err))

	// 测试非AppError
	err2 := fmt.Errorf("regular error")
	assert.False(t, IsHandled(err2))
}

func TestMarkHandled(t *testing.T) {
	err := New(ErrorTypeTemporary, "TEST_CODE", "Test message")
	err = MarkHandled(err, "test_handler").(error).(*AppError)
	assert.True(t, err.Handled)
	assert.Equal(t, "test_handler", err.HandlerName)

	// 测试非AppError
	err2 := fmt.Errorf("regular error")
	err2 = MarkHandled(err2, "test_handler")
	assert.Equal(t, "regular error", err2.Error())
}

func TestPredefinedErrors(t *testing.T) {
	assert.Equal(t, ErrorTypeNotFound, ErrNotFound.Type)
	assert.Equal(t, "NOT_FOUND", ErrNotFound.Code)

	assert.Equal(t, ErrorTypeValidation, ErrInvalidInput.Type)
	assert.Equal(t, "INVALID_INPUT", ErrInvalidInput.Code)

	assert.Equal(t, ErrorTypePermission, ErrUnauthorized.Type)
	assert.Equal(t, "UNAUTHORIZED", ErrUnauthorized.Code)

	assert.Equal(t, ErrorTypePermission, ErrForbidden.Type)
	assert.Equal(t, "FORBIDDEN", ErrForbidden.Code)

	assert.Equal(t, ErrorTypeInternal, ErrInternal.Type)
	assert.Equal(t, "INTERNAL", ErrInternal.Code)

	assert.Equal(t, ErrorTypeTemporary, ErrTimeout.Type)
	assert.Equal(t, "TIMEOUT", ErrTimeout.Code)

	assert.Equal(t, ErrorTypeTemporary, ErrUnavailable.Type)
	assert.Equal(t, "UNAVAILABLE", ErrUnavailable.Code)

	assert.Equal(t, ErrorTypePermanent, ErrAlreadyExists.Type)
	assert.Equal(t, "ALREADY_EXISTS", ErrAlreadyExists.Code)
}
