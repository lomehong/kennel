package errors

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
)

// 以下是错误处理和Panic恢复的使用示例

// ExampleErrorHandling 展示错误处理的基本用法
func ExampleErrorHandling() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "error-handling",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建错误处理器
	handler := DefaultErrorHandler(logger)

	// 创建一个临时错误
	err := New(ErrorTypeTemporary, "TEMP_ERROR", "临时错误示例")
	logger.Info("创建临时错误", "error", err)

	// 添加上下文信息
	err = err.WithContext("user_id", "12345")
	err = err.WithContext("request_id", "req-67890")

	// 设置重试信息
	err = err.WithRetry(3, 1*time.Second)

	// 处理错误
	result := handler.Handle(err)
	logger.Info("错误处理结果", "handled", IsHandled(result))

	// 包装错误
	wrappedErr := Wrap(err, ErrorTypePermanent, "WRAPPED", "包装的错误")
	logger.Info("包装错误", "error", wrappedErr)

	// 检查错误类型
	logger.Info("错误类型检查",
		"is_temporary", IsType(wrappedErr, ErrorTypeTemporary),
		"is_permanent", IsType(wrappedErr, ErrorTypePermanent),
		"is_critical", IsType(wrappedErr, ErrorTypeCritical),
	)

	// 获取错误上下文
	context := GetContext(wrappedErr)
	logger.Info("错误上下文", "context", context)

	// 检查错误是否可重试
	retriable := IsRetriable(wrappedErr)
	logger.Info("错误是否可重试", "retriable", retriable)

	// 获取重试信息
	canRetry, maxRetries, retryDelay := GetRetryInfo(wrappedErr)
	logger.Info("重试信息",
		"can_retry", canRetry,
		"max_retries", maxRetries,
		"retry_delay", retryDelay,
	)
}

// ExampleErrorHandlerRegistry 展示错误处理器注册表的用法
func ExampleErrorHandlerRegistry() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "error-registry",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建错误处理器注册表
	registry := DefaultErrorHandlerRegistry(logger)

	// 创建不同类型的错误
	tempErr := New(ErrorTypeTemporary, "TEMP_ERROR", "临时错误示例")
	permErr := New(ErrorTypePermanent, "PERM_ERROR", "永久错误示例")
	// 创建但不使用，仅作为示例
	_ = New(ErrorTypeCritical, "CRIT_ERROR", "严重错误示例")
	validErr := New(ErrorTypeValidation, "VALID_ERROR", "验证错误示例")

	// 处理临时错误
	logger.Info("处理临时错误")
	result := registry.Handle(tempErr)
	logger.Info("处理结果", "handled", IsHandled(result))

	// 处理永久错误
	logger.Info("处理永久错误")
	result = registry.Handle(permErr)
	logger.Info("处理结果", "handled", IsHandled(result))

	// 处理验证错误
	logger.Info("处理验证错误")
	result = registry.Handle(validErr)
	logger.Info("处理结果", "handled", IsHandled(result))

	// 处理严重错误（注意：这会触发panic，所以在实际代码中需要捕获）
	logger.Info("处理严重错误（将触发panic）")
	// 在实际代码中，应该这样使用：
	// defer func() {
	//     if p := recover(); p != nil {
	//         logger.Error("从panic中恢复", "panic", p)
	//     }
	// }()
	// registry.Handle(critErr)
}

// ExamplePanicRecovery 展示Panic恢复的用法
func ExamplePanicRecovery() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "panic-recovery",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建恢复管理器
	manager := DefaultRecoveryManager(logger)

	// 使用SafeGo启动goroutine
	logger.Info("启动安全的goroutine")
	manager.SafeGo(func() {
		logger.Info("goroutine正在执行")
		// 模拟一些工作
		time.Sleep(100 * time.Millisecond)
		logger.Info("goroutine执行完成")
	})

	// 使用SafeGo启动会panic的goroutine
	logger.Info("启动会panic的goroutine")
	manager.SafeGo(func() {
		logger.Info("goroutine正在执行")
		// 模拟一些工作
		time.Sleep(100 * time.Millisecond)
		// 触发panic
		panic("故意触发的panic")
	})

	// 等待goroutine执行
	time.Sleep(300 * time.Millisecond)

	// 获取恢复统计信息
	stats := manager.GetStats()
	logger.Info("恢复统计信息",
		"total_panics", stats.TotalPanics,
		"recovered_panics", stats.RecoveredPanics,
		"last_panic_time", stats.LastPanicTime,
	)

	// 使用SafeExec执行函数
	logger.Info("安全执行函数")
	err := manager.SafeExec(func() error {
		logger.Info("函数正在执行")
		// 返回一个错误
		return fmt.Errorf("函数返回的错误")
	})
	logger.Info("函数执行结果", "error", err)

	// 使用SafeExec执行会panic的函数
	logger.Info("安全执行会panic的函数")
	err = manager.SafeExec(func() error {
		logger.Info("函数正在执行")
		// 触发panic
		panic("函数中的panic")
		return nil
	})
	logger.Info("函数执行结果", "error", err)
}

// ExampleErrorRetry 展示错误重试的用法
func ExampleErrorRetry() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "error-retry",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建一个可重试的操作
	operation := func(attempt int) error {
		logger.Info("执行操作", "attempt", attempt)

		// 模拟操作（前两次失败，第三次成功）
		if attempt < 3 {
			return New(ErrorTypeTemporary, "TEMP_ERROR", fmt.Sprintf("临时错误，尝试次数：%d", attempt)).
				WithRetry(5, 500*time.Millisecond)
		}

		logger.Info("操作成功")
		return nil
	}

	// 执行带重试的操作
	var err error
	maxRetries := 5
	// 默认重试延迟，在此示例中由错误对象指定
	_ = 500 * time.Millisecond // 仅作为示例

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = operation(attempt)

		if err == nil {
			// 操作成功
			break
		}

		// 检查错误是否可重试
		if !IsRetriable(err) {
			logger.Error("不可重试的错误", "error", err)
			break
		}

		// 获取重试信息
		retriable, maxAttempts, delay := GetRetryInfo(err)
		if !retriable || attempt >= maxAttempts {
			logger.Error("达到最大重试次数", "error", err)
			break
		}

		// 等待后重试
		logger.Info("等待后重试", "delay", delay)
		time.Sleep(delay)
	}

	if err != nil {
		logger.Error("操作最终失败", "error", err)
	} else {
		logger.Info("操作最终成功")
	}
}

// ExampleContextErrorHandling 展示带上下文的错误处理
func ExampleContextErrorHandling() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "context-error",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建错误处理器注册表
	registry := DefaultErrorHandlerRegistry(logger)

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建一个操作
	operation := func(ctx context.Context) error {
		logger.Info("执行操作")

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return Wrap(ctx.Err(), ErrorTypeTemporary, "CONTEXT_ERROR", "上下文已取消")
		default:
			// 继续执行
		}

		// 模拟一些工作
		time.Sleep(100 * time.Millisecond)

		// 返回一个错误
		return New(ErrorTypeTemporary, "TEMP_ERROR", "临时错误示例")
	}

	// 执行操作
	err := operation(ctx)

	// 使用上下文处理错误
	result := registry.HandleWithContext(ctx, err)
	logger.Info("处理结果", "handled", IsHandled(result))

	// 取消上下文
	logger.Info("取消上下文")
	cancel()

	// 再次执行操作
	err = operation(ctx)

	// 使用上下文处理错误
	result = registry.HandleWithContext(ctx, err)
	logger.Info("处理结果", "error", result)
}
