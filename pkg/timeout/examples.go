package timeout

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
)

// 以下是超时控制器的使用示例

// ExampleBasicUsage 展示超时控制器的基本用法
func ExampleBasicUsage() {
	// 创建超时控制器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "timeout-controller",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	controller := NewTimeoutController(
		WithLogger(logger),
		WithDefaultTimeout(OperationTypeIO, 30*time.Second),
		WithDefaultTimeout(OperationTypeNetwork, 60*time.Second),
		WithMonitorInterval(5*time.Second),
	)

	// 创建一个操作
	op, err := controller.CreateOperation(OperationTypeIO, "file-read", "读取配置文件", 10*time.Second)
	if err != nil {
		logger.Error("创建操作失败", "error", err)
		return
	}

	// 使用操作的上下文执行任务
	go func() {
		// 模拟读取文件
		select {
		case <-time.After(5 * time.Second):
			logger.Info("文件读取完成")
			controller.CompleteOperation("file-read")
		case <-op.Context.Done():
			logger.Warn("文件读取被取消或超时", "error", op.Context.Err())
		}
	}()

	// 等待一段时间
	time.Sleep(6 * time.Second)

	// 列出当前操作
	operations := controller.ListOperations()
	logger.Info("当前操作数量", "count", len(operations))

	// 停止超时控制器
	controller.Stop()
}

// ExampleWithTimeoutExecution 展示使用WithTimeout执行函数
func ExampleWithTimeoutExecution() {
	// 创建超时控制器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "timeout-controller",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	controller := NewTimeoutController(WithLogger(logger))

	// 使用WithTimeout执行函数
	err := controller.WithTimeout(OperationTypeIO, "file-operation", "处理大文件", 5*time.Second, func(ctx context.Context) error {
		logger.Info("开始处理文件")

		// 模拟文件处理
		select {
		case <-time.After(3 * time.Second):
			logger.Info("文件处理完成")
			return nil
		case <-ctx.Done():
			logger.Warn("文件处理被取消或超时", "error", ctx.Err())
			return ctx.Err()
		}
	})

	if err != nil {
		logger.Error("文件处理失败", "error", err)
	}

	// 使用ExecuteWithTimeout执行函数（自动生成ID）
	err = controller.ExecuteWithTimeout(OperationTypeNetwork, "网络请求", 2*time.Second, func(ctx context.Context) error {
		logger.Info("开始网络请求")

		// 模拟网络请求
		select {
		case <-time.After(3 * time.Second):
			logger.Info("网络请求完成")
			return nil
		case <-ctx.Done():
			logger.Warn("网络请求被取消或超时", "error", ctx.Err())
			return ctx.Err()
		}
	})

	if err != nil {
		logger.Error("网络请求失败", "error", err)
	}

	// 停止超时控制器
	controller.Stop()
}

// ExampleIOOperations 展示IO操作的超时控制
func ExampleIOOperations() {
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "io-timeout",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建临时文件
	file, err := os.CreateTemp("", "timeout-example-*.txt")
	if err != nil {
		logger.Error("创建临时文件失败", "error", err)
		return
	}
	defer file.Close()

	// 写入数据
	data := []byte("Hello, Timeout Control!")
	n, err := WriteWithTimeout(ctx, file, data)
	if err != nil {
		logger.Error("写入文件失败", "error", err)
		return
	}
	logger.Info("写入文件成功", "bytes", n)

	// 重置文件指针
	_, err = file.Seek(0, 0)
	if err != nil {
		logger.Error("重置文件指针失败", "error", err)
		return
	}

	// 读取数据
	buffer := make([]byte, 100)
	n, err = ReadWithTimeout(ctx, file, buffer)
	if err != nil {
		logger.Error("读取文件失败", "error", err)
		return
	}
	logger.Info("读取文件成功", "bytes", n, "content", string(buffer[:n]))

	// 关闭文件
	err = CloseWithTimeout(ctx, file)
	if err != nil {
		logger.Error("关闭文件失败", "error", err)
		return
	}
	logger.Info("关闭文件成功")
}

// ExampleNetworkOperations 展示网络操作的超时控制
func ExampleNetworkOperations() {
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "network-timeout",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 连接到服务器
	logger.Info("连接到服务器")
	conn, err := DialWithTimeout(ctx, "tcp", "example.com:80")
	if err != nil {
		logger.Error("连接服务器失败", "error", err)
		return
	}
	defer conn.Close()
	logger.Info("连接服务器成功", "remote", conn.RemoteAddr())

	// 发送HTTP请求
	request := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	n, err := WriteWithTimeout(ctx, conn, request)
	if err != nil {
		logger.Error("发送请求失败", "error", err)
		return
	}
	logger.Info("发送请求成功", "bytes", n)

	// 读取响应
	buffer := make([]byte, 1024)
	n, err = ReadWithTimeout(ctx, conn, buffer)
	if err != nil && err != io.EOF {
		logger.Error("读取响应失败", "error", err)
		return
	}
	logger.Info("读取响应成功", "bytes", n)
	logger.Info("响应内容", "content", string(buffer[:n]))

	// 关闭连接
	err = CloseWithTimeout(ctx, conn)
	if err != nil {
		logger.Error("关闭连接失败", "error", err)
		return
	}
	logger.Info("关闭连接成功")
}

// ExampleRetryWithTimeout 展示带重试的超时操作
func ExampleRetryWithTimeout() {
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "retry-timeout",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 模拟不稳定的网络操作
	attempt := 0
	err := RetryWithTimeout(ctx, "不稳定的网络操作", 5, 2*time.Second, func() error {
		attempt++
		logger.Info("尝试执行操作", "attempt", attempt)

		// 模拟前几次失败，最后一次成功
		if attempt < 3 {
			logger.Warn("操作失败，将重试", "attempt", attempt)
			return fmt.Errorf("网络暂时不可用")
		}

		logger.Info("操作成功", "attempt", attempt)
		return nil
	})

	if err != nil {
		logger.Error("操作最终失败", "error", err)
	} else {
		logger.Info("操作最终成功", "attempts", attempt)
	}
}

// ExampleTimeoutMiddleware 展示超时中间件的使用
func ExampleTimeoutMiddleware() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "timeout-middleware",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建超时中间件
	middleware := NewTimeoutMiddleware(5 * time.Second)

	// 包装处理函数
	handler := middleware.Wrap(func(ctx context.Context) error {
		logger.Info("开始处理请求")

		// 模拟处理请求
		select {
		case <-time.After(3 * time.Second):
			logger.Info("请求处理完成")
			return nil
		case <-ctx.Done():
			logger.Warn("请求处理被取消或超时", "error", ctx.Err())
			return ctx.Err()
		}
	})

	// 执行处理函数
	err := handler()
	if err != nil {
		logger.Error("请求处理失败", "error", err)
	}

	// 使用WrapHandler函数
	handler = WrapHandler(2*time.Second, func(ctx context.Context) error {
		logger.Info("开始处理另一个请求")

		// 模拟处理请求
		select {
		case <-time.After(3 * time.Second):
			logger.Info("请求处理完成")
			return nil
		case <-ctx.Done():
			logger.Warn("请求处理被取消或超时", "error", ctx.Err())
			return ctx.Err()
		}
	})

	// 执行处理函数
	err = handler()
	if err != nil {
		logger.Error("请求处理失败", "error", err)
	}
}
