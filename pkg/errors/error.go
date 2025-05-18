package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorType 表示错误类型
type ErrorType int

// 预定义错误类型
const (
	ErrorTypeTemporary ErrorType = iota // 临时错误，可以重试
	ErrorTypePermanent                  // 永久错误，不应重试
	ErrorTypeCritical                   // 严重错误，需要立即处理
	ErrorTypeValidation                 // 验证错误，输入数据无效
	ErrorTypeNotFound                   // 未找到错误，请求的资源不存在
	ErrorTypePermission                 // 权限错误，没有足够的权限
	ErrorTypeInternal                   // 内部错误，系统内部错误
	ErrorTypeExternal                   // 外部错误，外部系统错误
)

// String 返回错误类型的字符串表示
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeTemporary:
		return "Temporary"
	case ErrorTypePermanent:
		return "Permanent"
	case ErrorTypeCritical:
		return "Critical"
	case ErrorTypeValidation:
		return "Validation"
	case ErrorTypeNotFound:
		return "NotFound"
	case ErrorTypePermission:
		return "Permission"
	case ErrorTypeInternal:
		return "Internal"
	case ErrorTypeExternal:
		return "External"
	default:
		return "Unknown"
	}
}

// AppError 表示应用程序错误
type AppError struct {
	Type        ErrorType           // 错误类型
	Code        string              // 错误代码
	Message     string              // 错误消息
	Cause       error               // 原始错误
	Context     map[string]interface{} // 错误上下文
	Stack       string              // 堆栈跟踪
	Time        time.Time           // 错误发生时间
	Retriable   bool                // 是否可重试
	MaxRetries  int                 // 最大重试次数
	RetryDelay  time.Duration       // 重试延迟
	Handled     bool                // 是否已处理
	HandlerName string              // 处理器名称
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现errors.Unwrap接口
func (e *AppError) Unwrap() error {
	return e.Cause
}

// WithContext 添加上下文信息
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithRetry 设置重试信息
func (e *AppError) WithRetry(maxRetries int, retryDelay time.Duration) *AppError {
	e.Retriable = true
	e.MaxRetries = maxRetries
	e.RetryDelay = retryDelay
	return e
}

// WithHandled 标记为已处理
func (e *AppError) WithHandled(handlerName string) *AppError {
	e.Handled = true
	e.HandlerName = handlerName
	return e
}

// IsTemporary 检查错误是否为临时错误
func (e *AppError) IsTemporary() bool {
	return e.Type == ErrorTypeTemporary
}

// IsPermanent 检查错误是否为永久错误
func (e *AppError) IsPermanent() bool {
	return e.Type == ErrorTypePermanent
}

// IsCritical 检查错误是否为严重错误
func (e *AppError) IsCritical() bool {
	return e.Type == ErrorTypeCritical
}

// IsRetriable 检查错误是否可重试
func (e *AppError) IsRetriable() bool {
	return e.Retriable || e.Type == ErrorTypeTemporary
}

// New 创建一个新的应用程序错误
func New(errorType ErrorType, code string, message string) *AppError {
	return &AppError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Time:    time.Now(),
		Stack:   getStackTrace(2),
	}
}

// Wrap 包装一个错误
func Wrap(err error, errorType ErrorType, code string, message string) *AppError {
	if err == nil {
		return nil
	}

	// 如果已经是AppError，保留原始错误类型和代码
	var appErr *AppError
	if errors.As(err, &appErr) {
		return &AppError{
			Type:      appErr.Type,
			Code:      appErr.Code,
			Message:   message,
			Cause:     appErr.Cause,
			Context:   appErr.Context,
			Stack:     getStackTrace(2),
			Time:      time.Now(),
			Retriable: appErr.Retriable,
		}
	}

	return &AppError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Cause:   err,
		Stack:   getStackTrace(2),
		Time:    time.Now(),
	}
}

// WrapIfErr 如果err不为nil，则包装错误
func WrapIfErr(err error, errorType ErrorType, code string, message string) error {
	if err == nil {
		return nil
	}
	return Wrap(err, errorType, code, message)
}

// Is 检查错误是否为指定类型
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As 将错误转换为指定类型
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// IsType 检查错误是否为指定类型
func IsType(err error, errorType ErrorType) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == errorType
	}
	return false
}

// GetContext 获取错误上下文
func GetContext(err error) map[string]interface{} {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Context
	}
	return nil
}

// GetStack 获取错误堆栈
func GetStack(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Stack
	}
	return ""
}

// IsRetriable 检查错误是否可重试
func IsRetriable(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.IsRetriable()
	}
	return false
}

// GetRetryInfo 获取重试信息
func GetRetryInfo(err error) (bool, int, time.Duration) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Retriable, appErr.MaxRetries, appErr.RetryDelay
	}
	return false, 0, 0
}

// IsHandled 检查错误是否已处理
func IsHandled(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Handled
	}
	return false
}

// MarkHandled 标记错误为已处理
func MarkHandled(err error, handlerName string) error {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.WithHandled(handlerName)
	}
	return err
}

// getStackTrace 获取堆栈跟踪
func getStackTrace(skip int) string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var builder strings.Builder
	for {
		frame, more := frames.Next()
		if !more {
			break
		}

		// 跳过标准库和测试文件
		if strings.Contains(frame.File, "runtime/") || strings.Contains(frame.File, "testing/") {
			continue
		}

		builder.WriteString(fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}

	return builder.String()
}

// 预定义错误
var (
	ErrNotFound      = New(ErrorTypeNotFound, "NOT_FOUND", "Resource not found")
	ErrInvalidInput  = New(ErrorTypeValidation, "INVALID_INPUT", "Invalid input")
	ErrUnauthorized  = New(ErrorTypePermission, "UNAUTHORIZED", "Unauthorized access")
	ErrForbidden     = New(ErrorTypePermission, "FORBIDDEN", "Forbidden access")
	ErrInternal      = New(ErrorTypeInternal, "INTERNAL", "Internal server error")
	ErrTimeout       = New(ErrorTypeTemporary, "TIMEOUT", "Operation timed out")
	ErrUnavailable   = New(ErrorTypeTemporary, "UNAVAILABLE", "Service unavailable")
	ErrAlreadyExists = New(ErrorTypePermanent, "ALREADY_EXISTS", "Resource already exists")
)
