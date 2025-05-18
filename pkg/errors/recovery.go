package errors

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// RecoveryHandler 恢复处理器接口
type RecoveryHandler interface {
	// HandlePanic 处理panic
	HandlePanic(p interface{}) error
	// Name 返回处理器名称
	Name() string
}

// LogRecoveryHandler 日志恢复处理器
type LogRecoveryHandler struct {
	logger hclog.Logger
	name   string
}

// NewLogRecoveryHandler 创建一个新的日志恢复处理器
func NewLogRecoveryHandler(logger hclog.Logger) *LogRecoveryHandler {
	return &LogRecoveryHandler{
		logger: logger,
		name:   "log_recovery",
	}
}

// HandlePanic 处理panic
func (h *LogRecoveryHandler) HandlePanic(p interface{}) error {
	stack := debug.Stack()
	h.logger.Error("恢复panic",
		"panic", p,
		"stack", string(stack),
	)

	// 将panic转换为错误
	return New(ErrorTypeCritical, "PANIC", fmt.Sprintf("Panic: %v", p)).
		WithContext("stack", string(stack))
}

// Name 返回处理器名称
func (h *LogRecoveryHandler) Name() string {
	return h.name
}

// RecoveryHandlerChain 恢复处理器链
type RecoveryHandlerChain struct {
	handlers []RecoveryHandler
	name     string
}

// NewRecoveryHandlerChain 创建一个新的恢复处理器链
func NewRecoveryHandlerChain(handlers ...RecoveryHandler) *RecoveryHandlerChain {
	return &RecoveryHandlerChain{
		handlers: handlers,
		name:     "recovery_chain",
	}
}

// HandlePanic 处理panic
func (c *RecoveryHandlerChain) HandlePanic(p interface{}) error {
	var err error

	// 依次调用所有处理器
	for _, handler := range c.handlers {
		err = handler.HandlePanic(p)
	}

	return err
}

// Name 返回处理器名称
func (c *RecoveryHandlerChain) Name() string {
	return c.name
}

// AddHandler 添加处理器
func (c *RecoveryHandlerChain) AddHandler(handler RecoveryHandler) {
	c.handlers = append(c.handlers, handler)
}

// RecoveryManager 恢复管理器
type RecoveryManager struct {
	handler RecoveryHandler
	logger  hclog.Logger
	stats   RecoveryStats
	mu      sync.RWMutex
}

// RecoveryStats 恢复统计信息
type RecoveryStats struct {
	TotalPanics     int64
	RecoveredPanics int64
	LastPanicTime   time.Time
}

// NewRecoveryManager 创建一个新的恢复管理器
func NewRecoveryManager(logger hclog.Logger, handler RecoveryHandler) *RecoveryManager {
	if handler == nil {
		handler = NewLogRecoveryHandler(logger)
	}

	return &RecoveryManager{
		handler: handler,
		logger:  logger,
	}
}

// HandlePanic 处理panic
func (m *RecoveryManager) HandlePanic(p interface{}) error {
	m.mu.Lock()
	m.stats.TotalPanics++
	m.stats.LastPanicTime = time.Now()
	m.mu.Unlock()

	err := m.handler.HandlePanic(p)

	m.mu.Lock()
	m.stats.RecoveredPanics++
	m.mu.Unlock()

	return err
}

// GetStats 获取统计信息
func (m *RecoveryManager) GetStats() RecoveryStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// SafeGo 安全地启动goroutine
func (m *RecoveryManager) SafeGo(f func()) {
	go func() {
		defer func() {
			if p := recover(); p != nil {
				m.HandlePanic(p)
			}
		}()
		f()
	}()
}

// SafeGoWithContext 安全地启动带上下文的goroutine
func (m *RecoveryManager) SafeGoWithContext(ctx context.Context, f func(context.Context)) {
	go func() {
		defer func() {
			if p := recover(); p != nil {
				m.HandlePanic(p)
			}
		}()
		f(ctx)
	}()
}

// SafeExec 安全地执行函数
func (m *RecoveryManager) SafeExec(f func() error) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = m.HandlePanic(p)
		}
	}()
	return f()
}

// SafeExecWithContext 安全地执行带上下文的函数
func (m *RecoveryManager) SafeExecWithContext(ctx context.Context, f func(context.Context) error) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = m.HandlePanic(p)
		}
	}()
	return f(ctx)
}

// DefaultRecoveryHandler 默认恢复处理器
func DefaultRecoveryHandler(logger hclog.Logger) RecoveryHandler {
	return NewLogRecoveryHandler(logger)
}

// DefaultRecoveryManager 默认恢复管理器
func DefaultRecoveryManager(logger hclog.Logger) *RecoveryManager {
	return NewRecoveryManager(logger, DefaultRecoveryHandler(logger))
}

// SafeGo 安全地启动goroutine
func SafeGo(f func()) {
	go func() {
		defer func() {
			if p := recover(); p != nil {
				stack := debug.Stack()
				fmt.Printf("Recovered from panic: %v\n%s\n", p, stack)
			}
		}()
		f()
	}()
}

// SafeGoWithContext 安全地启动带上下文的goroutine
func SafeGoWithContext(ctx context.Context, f func(context.Context)) {
	go func() {
		defer func() {
			if p := recover(); p != nil {
				stack := debug.Stack()
				fmt.Printf("Recovered from panic: %v\n%s\n", p, stack)
			}
		}()
		f(ctx)
	}()
}

// SafeExec 安全地执行函数
func SafeExec(f func() error) (err error) {
	defer func() {
		if p := recover(); p != nil {
			stack := debug.Stack()
			fmt.Printf("Recovered from panic: %v\n%s\n", p, stack)
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return f()
}

// SafeExecWithContext 安全地执行带上下文的函数
func SafeExecWithContext(ctx context.Context, f func(context.Context) error) (err error) {
	defer func() {
		if p := recover(); p != nil {
			stack := debug.Stack()
			fmt.Printf("Recovered from panic: %v\n%s\n", p, stack)
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return f(ctx)
}

// GetGoroutineID 获取当前goroutine的ID
func GetGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	var id uint64
	fmt.Sscanf(string(b), "goroutine %d ", &id)
	return id
}
