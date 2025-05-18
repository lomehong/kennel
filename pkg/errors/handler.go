package errors

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	// Handle 处理错误
	Handle(err error) error
	// Name 返回处理器名称
	Name() string
}

// LogErrorHandler 日志错误处理器
type LogErrorHandler struct {
	logger hclog.Logger
	name   string
}

// NewLogErrorHandler 创建一个新的日志错误处理器
func NewLogErrorHandler(logger hclog.Logger) *LogErrorHandler {
	return &LogErrorHandler{
		logger: logger,
		name:   "log",
	}
}

// Handle 处理错误
func (h *LogErrorHandler) Handle(err error) error {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if As(err, &appErr) {
		// 记录应用程序错误
		h.logger.Error("应用程序错误",
			"type", appErr.Type.String(),
			"code", appErr.Code,
			"message", appErr.Message,
			"cause", appErr.Cause,
			"context", appErr.Context,
			"stack", appErr.Stack,
			"time", appErr.Time,
		)
	} else {
		// 记录普通错误
		h.logger.Error("错误", "error", err)
	}

	// 标记为已处理
	return MarkHandled(err, h.Name())
}

// Name 返回处理器名称
func (h *LogErrorHandler) Name() string {
	return h.name
}

// RetryErrorHandler 重试错误处理器
type RetryErrorHandler struct {
	maxRetries int
	backoff    time.Duration
	name       string
}

// NewRetryErrorHandler 创建一个新的重试错误处理器
func NewRetryErrorHandler(maxRetries int, backoff time.Duration) *RetryErrorHandler {
	return &RetryErrorHandler{
		maxRetries: maxRetries,
		backoff:    backoff,
		name:       "retry",
	}
}

// Handle 处理错误
func (h *RetryErrorHandler) Handle(err error) error {
	if err == nil {
		return nil
	}

	// 检查错误是否可重试
	if !IsRetriable(err) {
		return err
	}

	// 获取重试信息
	retriable, maxRetries, retryDelay := GetRetryInfo(err)
	if !retriable {
		return err
	}

	// 使用错误中的重试信息或默认值
	if maxRetries <= 0 {
		maxRetries = h.maxRetries
	}
	if retryDelay <= 0 {
		retryDelay = h.backoff
	}

	// 标记为已处理
	return MarkHandled(err, h.Name())
}

// Name 返回处理器名称
func (h *RetryErrorHandler) Name() string {
	return h.name
}

// PanicErrorHandler Panic错误处理器
type PanicErrorHandler struct {
	logger hclog.Logger
	name   string
}

// NewPanicErrorHandler 创建一个新的Panic错误处理器
func NewPanicErrorHandler(logger hclog.Logger) *PanicErrorHandler {
	return &PanicErrorHandler{
		logger: logger,
		name:   "panic",
	}
}

// Handle 处理错误
func (h *PanicErrorHandler) Handle(err error) error {
	if err == nil {
		return nil
	}

	// 检查错误是否为严重错误
	if IsType(err, ErrorTypeCritical) {
		h.logger.Error("严重错误，触发panic", "error", err)
		panic(err)
	}

	return err
}

// Name 返回处理器名称
func (h *PanicErrorHandler) Name() string {
	return h.name
}

// ErrorHandlerChain 错误处理器链
type ErrorHandlerChain struct {
	handlers []ErrorHandler
	name     string
}

// NewErrorHandlerChain 创建一个新的错误处理器链
func NewErrorHandlerChain(handlers ...ErrorHandler) *ErrorHandlerChain {
	return &ErrorHandlerChain{
		handlers: handlers,
		name:     "chain",
	}
}

// Handle 处理错误
func (c *ErrorHandlerChain) Handle(err error) error {
	if err == nil {
		return nil
	}

	// 依次调用所有处理器
	for _, handler := range c.handlers {
		err = handler.Handle(err)
	}

	return err
}

// Name 返回处理器名称
func (c *ErrorHandlerChain) Name() string {
	return c.name
}

// AddHandler 添加处理器
func (c *ErrorHandlerChain) AddHandler(handler ErrorHandler) {
	c.handlers = append(c.handlers, handler)
}

// ErrorHandlerRegistry 错误处理器注册表
type ErrorHandlerRegistry struct {
	handlers map[ErrorType]ErrorHandler
	mu       sync.RWMutex
	logger   hclog.Logger
}

// NewErrorHandlerRegistry 创建一个新的错误处理器注册表
func NewErrorHandlerRegistry(logger hclog.Logger) *ErrorHandlerRegistry {
	return &ErrorHandlerRegistry{
		handlers: make(map[ErrorType]ErrorHandler),
		logger:   logger,
	}
}

// RegisterHandler 注册处理器
func (r *ErrorHandlerRegistry) RegisterHandler(errorType ErrorType, handler ErrorHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[errorType] = handler
	r.logger.Debug("注册错误处理器", "type", errorType.String(), "handler", handler.Name())
}

// GetHandler 获取处理器
func (r *ErrorHandlerRegistry) GetHandler(errorType ErrorType) (ErrorHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.handlers[errorType]
	return handler, ok
}

// Handle 处理错误
func (r *ErrorHandlerRegistry) Handle(err error) error {
	if err == nil {
		return nil
	}

	// 获取错误类型
	var errorType ErrorType
	var appErr *AppError
	if As(err, &appErr) {
		errorType = appErr.Type
	} else {
		errorType = ErrorTypeInternal
	}

	// 获取处理器
	r.mu.RLock()
	handler, ok := r.handlers[errorType]
	r.mu.RUnlock()

	if !ok {
		// 使用默认处理器
		r.logger.Warn("未找到错误处理器", "type", errorType.String())
		return err
	}

	// 处理错误
	return handler.Handle(err)
}

// HandleWithContext 使用上下文处理错误
func (r *ErrorHandlerRegistry) HandleWithContext(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// 检查上下文是否已取消
	if ctx.Err() != nil {
		r.logger.Debug("上下文已取消，跳过错误处理", "error", err, "context_error", ctx.Err())
		return err
	}

	return r.Handle(err)
}

// DefaultErrorHandler 默认错误处理器
func DefaultErrorHandler(logger hclog.Logger) ErrorHandler {
	// 创建处理器链
	chain := NewErrorHandlerChain(
		NewLogErrorHandler(logger),
		NewRetryErrorHandler(3, 1*time.Second),
	)

	return chain
}

// DefaultErrorHandlerRegistry 默认错误处理器注册表
func DefaultErrorHandlerRegistry(logger hclog.Logger) *ErrorHandlerRegistry {
	registry := NewErrorHandlerRegistry(logger)

	// 注册默认处理器
	defaultHandler := DefaultErrorHandler(logger)
	registry.RegisterHandler(ErrorTypeTemporary, defaultHandler)
	registry.RegisterHandler(ErrorTypePermanent, defaultHandler)
	registry.RegisterHandler(ErrorTypeValidation, defaultHandler)
	registry.RegisterHandler(ErrorTypeNotFound, defaultHandler)
	registry.RegisterHandler(ErrorTypePermission, defaultHandler)
	registry.RegisterHandler(ErrorTypeInternal, defaultHandler)
	registry.RegisterHandler(ErrorTypeExternal, defaultHandler)

	// 注册严重错误处理器
	registry.RegisterHandler(ErrorTypeCritical, NewPanicErrorHandler(logger))

	return registry
}
