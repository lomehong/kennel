package core

import (
	"context"
	"io"
	"time"

	"github.com/lomehong/kennel/pkg/timeout"
)

// 添加超时控制器到App结构体
func (app *App) initTimeoutController() {
	// 创建超时控制器
	app.timeoutController = timeout.NewTimeoutController(
		timeout.WithLogger(app.logger.Named("timeout-controller")),
		timeout.WithDefaultTimeout(timeout.OperationTypeIO, 30*time.Second),
		timeout.WithDefaultTimeout(timeout.OperationTypeNetwork, 60*time.Second),
		timeout.WithDefaultTimeout(timeout.OperationTypeDatabase, 30*time.Second),
		timeout.WithDefaultTimeout(timeout.OperationTypePlugin, 120*time.Second),
		timeout.WithMonitorInterval(5*time.Second),
	)

	app.logger.Info("超时控制器已初始化")
}

// GetTimeoutController 获取超时控制器
func (app *App) GetTimeoutController() *timeout.TimeoutController {
	return app.timeoutController
}

// WithTimeout 使用超时执行函数
func (app *App) WithTimeout(opType timeout.OperationType, id string, description string, timeout time.Duration, fn func(context.Context) error) error {
	if app.timeoutController == nil {
		// 如果超时控制器未初始化，使用简单的上下文超时
		ctx, cancel := context.WithTimeout(app.ctx, timeout)
		defer cancel()
		return fn(ctx)
	}

	return app.timeoutController.WithTimeout(opType, id, description, timeout, fn)
}

// ExecuteWithTimeout 使用超时执行函数（自动生成ID）
func (app *App) ExecuteWithTimeout(opType timeout.OperationType, description string, timeout time.Duration, fn func(context.Context) error) error {
	if app.timeoutController == nil {
		// 如果超时控制器未初始化，使用简单的上下文超时
		ctx, cancel := context.WithTimeout(app.ctx, timeout)
		defer cancel()
		return fn(ctx)
	}

	return app.timeoutController.ExecuteWithTimeout(opType, description, timeout, fn)
}

// IOOperation 执行IO操作，带有超时控制
func (app *App) IOOperation(description string, timeoutDuration time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(app.ctx, timeoutDuration)
	defer cancel()
	return timeout.IOOperation(ctx, description, fn)
}

// ReadWithTimeout 带超时的读取操作
func (app *App) ReadWithTimeout(reader interface{}, buffer []byte, timeoutDuration time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(app.ctx, timeoutDuration)
	defer cancel()
	return timeout.ReadWithTimeout(ctx, reader.(io.Reader), buffer)
}

// WriteWithTimeout 带超时的写入操作
func (app *App) WriteWithTimeout(writer interface{}, data []byte, timeoutDuration time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(app.ctx, timeoutDuration)
	defer cancel()
	return timeout.WriteWithTimeout(ctx, writer.(io.Writer), data)
}

// CloseWithTimeout 带超时的关闭操作
func (app *App) CloseWithTimeout(closer interface{}, timeoutDuration time.Duration) error {
	ctx, cancel := context.WithTimeout(app.ctx, timeoutDuration)
	defer cancel()
	return timeout.CloseWithTimeout(ctx, closer.(io.Closer))
}

// RetryWithTimeout 带超时和重试的操作
func (app *App) RetryWithTimeout(description string, attempts int, delay time.Duration, timeoutDuration time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(app.ctx, timeoutDuration)
	defer cancel()
	return timeout.RetryWithTimeout(ctx, description, attempts, delay, fn)
}

// 定义接口类型，避免导入循环依赖
type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}

type Closer interface {
	Close() error
}
