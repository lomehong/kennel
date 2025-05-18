package core

import (
	"context"
	"fmt"

	"github.com/lomehong/kennel/pkg/errors"
)

// 添加错误处理和Panic恢复到App结构体
func (app *App) initErrorHandling() {
	// 创建错误处理器注册表
	app.errorRegistry = errors.DefaultErrorHandlerRegistry(app.logger.Named("error-handler"))

	// 创建恢复管理器
	app.recoveryManager = errors.DefaultRecoveryManager(app.logger.Named("recovery-manager"))

	app.logger.Info("错误处理和Panic恢复已初始化")
}

// GetErrorRegistry 获取错误处理器注册表
func (app *App) GetErrorRegistry() *errors.ErrorHandlerRegistry {
	return app.errorRegistry
}

// GetRecoveryManager 获取恢复管理器
func (app *App) GetRecoveryManager() *errors.RecoveryManager {
	return app.recoveryManager
}

// HandleError 处理错误
func (app *App) HandleError(err error) error {
	if err == nil {
		return nil
	}

	return app.errorRegistry.Handle(err)
}

// HandleErrorWithContext 使用上下文处理错误
func (app *App) HandleErrorWithContext(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	return app.errorRegistry.HandleWithContext(ctx, err)
}

// SafeGo 安全地启动goroutine
func (app *App) SafeGo(f func()) {
	app.recoveryManager.SafeGo(f)
}

// SafeGoWithContext 安全地启动带上下文的goroutine
func (app *App) SafeGoWithContext(ctx context.Context, f func(context.Context)) {
	app.recoveryManager.SafeGoWithContext(ctx, f)
}

// SafeExec 安全地执行函数
func (app *App) SafeExec(f func() error) error {
	return app.recoveryManager.SafeExec(f)
}

// SafeExecWithContext 安全地执行带上下文的函数
func (app *App) SafeExecWithContext(ctx context.Context, f func(context.Context) error) error {
	return app.recoveryManager.SafeExecWithContext(ctx, f)
}

// NewError 创建一个新的应用程序错误
func (app *App) NewError(errorType errors.ErrorType, code string, message string) *errors.AppError {
	return errors.New(errorType, code, message)
}

// WrapError 包装一个错误
func (app *App) WrapError(err error, errorType errors.ErrorType, code string, message string) error {
	return errors.Wrap(err, errorType, code, message)
}

// IsErrorType 检查错误是否为指定类型
func (app *App) IsErrorType(err error, errorType errors.ErrorType) bool {
	return errors.IsType(err, errorType)
}

// IsErrorRetriable 检查错误是否可重试
func (app *App) IsErrorRetriable(err error) bool {
	return errors.IsRetriable(err)
}

// GetErrorContext 获取错误上下文
func (app *App) GetErrorContext(err error) map[string]interface{} {
	return errors.GetContext(err)
}

// GetErrorStack 获取错误堆栈
func (app *App) GetErrorStack(err error) string {
	return errors.GetStack(err)
}

// RegisterErrorHandler 注册错误处理器
func (app *App) RegisterErrorHandler(errorType errors.ErrorType, handler errors.ErrorHandler) {
	app.errorRegistry.RegisterHandler(errorType, handler)
}

// GetErrorHandler 获取错误处理器
func (app *App) GetErrorHandler(errorType errors.ErrorType) (errors.ErrorHandler, bool) {
	return app.errorRegistry.GetHandler(errorType)
}

// GetRecoveryStats 获取恢复统计信息
func (app *App) GetRecoveryStats() errors.RecoveryStats {
	return app.recoveryManager.GetStats()
}

// HandlePanic 处理panic
func (app *App) HandlePanic(p interface{}) error {
	return app.recoveryManager.HandlePanic(p)
}

// RunSafely 安全地运行函数，处理错误和panic
func (app *App) RunSafely(f func() error) error {
	err := app.SafeExec(f)
	if err != nil {
		return app.HandleError(err)
	}
	return nil
}

// RunSafelyWithContext 安全地运行带上下文的函数，处理错误和panic
func (app *App) RunSafelyWithContext(ctx context.Context, f func(context.Context) error) error {
	err := app.SafeExecWithContext(ctx, f)
	if err != nil {
		return app.HandleErrorWithContext(ctx, err)
	}
	return nil
}

// MustRun 运行函数，如果出错则panic
func (app *App) MustRun(f func() error) {
	if err := app.RunSafely(f); err != nil {
		panic(fmt.Sprintf("MustRun failed: %v", err))
	}
}

// MustRunWithContext 运行带上下文的函数，如果出错则panic
func (app *App) MustRunWithContext(ctx context.Context, f func(context.Context) error) {
	if err := app.RunSafelyWithContext(ctx, f); err != nil {
		panic(fmt.Sprintf("MustRunWithContext failed: %v", err))
	}
}
