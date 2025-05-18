package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lomehong/kennel/pkg/core"
	"github.com/lomehong/kennel/pkg/errors"
)

// 本示例展示如何在AppFramework中使用错误处理和Panic恢复
func main() {
	// 创建应用程序实例
	app := core.NewApp("config.yaml")

	// 初始化应用程序
	if err := app.Init(); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 错误处理和Panic恢复使用示例 ===")

	// 示例1: 基本错误处理
	fmt.Println("\n=== 示例1: 基本错误处理 ===")
	basicErrorHandling(app)

	// 示例2: 错误类型和上下文
	fmt.Println("\n=== 示例2: 错误类型和上下文 ===")
	errorTypesAndContext(app)

	// 示例3: 错误重试
	fmt.Println("\n=== 示例3: 错误重试 ===")
	errorRetry(app)

	// 示例4: Panic恢复
	fmt.Println("\n=== 示例4: Panic恢复 ===")
	panicRecovery(app)

	// 示例5: 安全执行
	fmt.Println("\n=== 示例5: 安全执行 ===")
	safeExecution(app)

	// 获取恢复统计信息
	stats := app.GetRecoveryStats()
	fmt.Println("\n恢复统计信息:")
	fmt.Printf("- 总Panic数: %d\n", stats.TotalPanics)
	fmt.Printf("- 已恢复Panic数: %d\n", stats.RecoveredPanics)
	fmt.Printf("- 最后Panic时间: %v\n", stats.LastPanicTime)

	// 停止应用程序
	app.Stop()
	fmt.Println("\n应用程序已停止")
}

// 基本错误处理
func basicErrorHandling(app *core.App) {
	// 创建一个错误
	err := app.NewError(errors.ErrorTypeTemporary, "TEMP_ERROR", "临时错误示例")
	fmt.Printf("创建错误: %v\n", err)

	// 处理错误
	result := app.HandleError(err)
	fmt.Printf("错误处理结果: %v\n", result)
	fmt.Printf("错误是否已处理: %v\n", errors.IsHandled(result))

	// 包装错误
	wrappedErr := app.WrapError(err, errors.ErrorTypePermanent, "WRAPPED", "包装的错误")
	fmt.Printf("包装错误: %v\n", wrappedErr)

	// 处理包装的错误
	result = app.HandleError(wrappedErr)
	fmt.Printf("包装错误处理结果: %v\n", result)
	fmt.Printf("包装错误是否已处理: %v\n", errors.IsHandled(result))
}

// 错误类型和上下文
func errorTypesAndContext(app *core.App) {
	// 创建不同类型的错误
	tempErr := app.NewError(errors.ErrorTypeTemporary, "TEMP_ERROR", "临时错误示例")
	permErr := app.NewError(errors.ErrorTypePermanent, "PERM_ERROR", "永久错误示例")
	validErr := app.NewError(errors.ErrorTypeValidation, "VALID_ERROR", "验证错误示例")
	notFoundErr := app.NewError(errors.ErrorTypeNotFound, "NOT_FOUND", "未找到错误示例")

	// 添加上下文信息
	tempErr.WithContext("user_id", "12345")
	tempErr.WithContext("request_id", "req-67890")

	// 检查错误类型
	fmt.Printf("tempErr是临时错误: %v\n", app.IsErrorType(tempErr, errors.ErrorTypeTemporary))
	fmt.Printf("permErr是永久错误: %v\n", app.IsErrorType(permErr, errors.ErrorTypePermanent))
	fmt.Printf("validErr是验证错误: %v\n", app.IsErrorType(validErr, errors.ErrorTypeValidation))
	fmt.Printf("notFoundErr是未找到错误: %v\n", app.IsErrorType(notFoundErr, errors.ErrorTypeNotFound))

	// 获取错误上下文
	context := app.GetErrorContext(tempErr)
	fmt.Printf("错误上下文: %v\n", context)

	// 获取错误堆栈
	stack := app.GetErrorStack(tempErr)
	fmt.Printf("错误堆栈(前100个字符): %s...\n", stack[:100])
}

// 错误重试
func errorRetry(app *core.App) {
	// 创建一个可重试的操作
	operation := func(attempt int) error {
		fmt.Printf("执行操作，尝试次数: %d\n", attempt)

		// 模拟操作（前两次失败，第三次成功）
		if attempt < 3 {
			return app.NewError(errors.ErrorTypeTemporary, "TEMP_ERROR", fmt.Sprintf("临时错误，尝试次数：%d", attempt)).
				WithRetry(5, 500*time.Millisecond)
		}

		fmt.Println("操作成功")
		return nil
	}

	// 执行带重试的操作
	var err error
	maxRetries := 5

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = operation(attempt)

		if err == nil {
			// 操作成功
			break
		}

		// 检查错误是否可重试
		if !app.IsErrorRetriable(err) {
			fmt.Printf("不可重试的错误: %v\n", err)
			break
		}

		// 获取重试信息
		retriable, maxAttempts, delay := errors.GetRetryInfo(err)
		if !retriable || attempt >= maxAttempts {
			fmt.Printf("达到最大重试次数: %v\n", err)
			break
		}

		// 等待后重试
		fmt.Printf("等待后重试，延迟: %v\n", delay)
		time.Sleep(delay)
	}

	if err != nil {
		fmt.Printf("操作最终失败: %v\n", err)
	} else {
		fmt.Println("操作最终成功")
	}
}

// Panic恢复
func panicRecovery(app *core.App) {
	// 使用SafeGo启动goroutine
	fmt.Println("启动安全的goroutine")
	app.SafeGo(func() {
		fmt.Println("goroutine正在执行")
		// 模拟一些工作
		time.Sleep(100 * time.Millisecond)
		fmt.Println("goroutine执行完成")
	})

	// 使用SafeGo启动会panic的goroutine
	fmt.Println("启动会panic的goroutine")
	app.SafeGo(func() {
		fmt.Println("goroutine正在执行")
		// 模拟一些工作
		time.Sleep(100 * time.Millisecond)
		// 触发panic
		panic("故意触发的panic")
	})

	// 等待goroutine执行
	time.Sleep(300 * time.Millisecond)
}

// 安全执行
func safeExecution(app *core.App) {
	// 使用SafeExec执行函数
	fmt.Println("安全执行函数")
	err := app.SafeExec(func() error {
		fmt.Println("函数正在执行")
		// 返回一个错误
		return fmt.Errorf("函数返回的错误")
	})
	fmt.Printf("函数执行结果: %v\n", err)

	// 使用SafeExec执行会panic的函数
	fmt.Println("安全执行会panic的函数")
	err = app.SafeExec(func() error {
		fmt.Println("函数正在执行")
		// 触发panic
		panic("函数中的panic")
		return nil
	})
	fmt.Printf("函数执行结果: %v\n", err)

	// 使用RunSafely执行函数（自动处理错误）
	fmt.Println("使用RunSafely执行函数")
	err = app.RunSafely(func() error {
		fmt.Println("函数正在执行")
		// 返回一个错误
		return app.NewError(errors.ErrorTypeTemporary, "TEMP_ERROR", "临时错误示例")
	})
	fmt.Printf("函数执行结果: %v\n", err)
	fmt.Printf("错误是否已处理: %v\n", errors.IsHandled(err))

	// 使用上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 使用RunSafelyWithContext执行函数
	fmt.Println("使用RunSafelyWithContext执行函数")
	err = app.RunSafelyWithContext(ctx, func(ctx context.Context) error {
		fmt.Println("函数正在执行")
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 继续执行
		}
		// 返回一个错误
		return app.NewError(errors.ErrorTypeTemporary, "TEMP_ERROR", "临时错误示例")
	})
	fmt.Printf("函数执行结果: %v\n", err)
	fmt.Printf("错误是否已处理: %v\n", errors.IsHandled(err))
}
