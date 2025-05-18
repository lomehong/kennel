package timeout

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

// IOOperation 执行IO操作，带有超时控制
func IOOperation(ctx context.Context, description string, fn func() error) error {
	// 创建完成通道
	done := make(chan error, 1)

	// 在后台执行IO操作
	go func() {
		done <- fn()
	}()

	// 等待操作完成或上下文取消
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("IO操作超时或取消: %s: %w", description, ctx.Err())
	}
}

// ReadWithTimeout 带超时的读取操作
func ReadWithTimeout(ctx context.Context, reader io.Reader, buffer []byte) (int, error) {
	// 创建完成通道
	done := make(chan struct {
		n   int
		err error
	}, 1)

	// 在后台执行读取操作
	go func() {
		n, err := reader.Read(buffer)
		done <- struct {
			n   int
			err error
		}{n, err}
	}()

	// 等待操作完成或上下文取消
	select {
	case result := <-done:
		return result.n, result.err
	case <-ctx.Done():
		return 0, fmt.Errorf("读取操作超时或取消: %w", ctx.Err())
	}
}

// WriteWithTimeout 带超时的写入操作
func WriteWithTimeout(ctx context.Context, writer io.Writer, data []byte) (int, error) {
	// 创建完成通道
	done := make(chan struct {
		n   int
		err error
	}, 1)

	// 在后台执行写入操作
	go func() {
		n, err := writer.Write(data)
		done <- struct {
			n   int
			err error
		}{n, err}
	}()

	// 等待操作完成或上下文取消
	select {
	case result := <-done:
		return result.n, result.err
	case <-ctx.Done():
		return 0, fmt.Errorf("写入操作超时或取消: %w", ctx.Err())
	}
}

// CloseWithTimeout 带超时的关闭操作
func CloseWithTimeout(ctx context.Context, closer io.Closer) error {
	// 创建完成通道
	done := make(chan error, 1)

	// 在后台执行关闭操作
	go func() {
		done <- closer.Close()
	}()

	// 等待操作完成或上下文取消
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("关闭操作超时或取消: %w", ctx.Err())
	}
}

// DialWithTimeout 带超时的网络连接操作
func DialWithTimeout(ctx context.Context, network, address string) (net.Conn, error) {
	// 创建完成通道
	done := make(chan struct {
		conn net.Conn
		err  error
	}, 1)

	// 在后台执行连接操作
	go func() {
		// 使用net.Dialer，它本身支持上下文
		dialer := &net.Dialer{}
		conn, err := dialer.DialContext(ctx, network, address)
		done <- struct {
			conn net.Conn
			err  error
		}{conn, err}
	}()

	// 等待操作完成或上下文取消
	select {
	case result := <-done:
		return result.conn, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("连接操作超时或取消: %w", ctx.Err())
	}
}

// RetryWithTimeout 带超时和重试的操作
func RetryWithTimeout(ctx context.Context, description string, attempts int, delay time.Duration, fn func() error) error {
	var err error

	// 执行重试
	for i := 0; i < attempts; i++ {
		// 检查上下文是否已取消
		if ctx.Err() != nil {
			return fmt.Errorf("操作已取消: %s: %w", description, ctx.Err())
		}

		// 执行操作
		err = fn()
		if err == nil {
			return nil
		}

		// 如果不是最后一次尝试，等待后重试
		if i < attempts-1 {
			// 使用带上下文的延迟
			select {
			case <-time.After(delay):
				// 继续下一次尝试
			case <-ctx.Done():
				return fmt.Errorf("操作已取消: %s: %w", description, ctx.Err())
			}
		}
	}

	return fmt.Errorf("操作失败，已重试%d次: %s: %w", attempts, description, err)
}

// TimeoutMiddleware 超时中间件，用于包装处理函数
type TimeoutMiddleware struct {
	Timeout time.Duration
}

// NewTimeoutMiddleware 创建一个新的超时中间件
func NewTimeoutMiddleware(timeout time.Duration) *TimeoutMiddleware {
	return &TimeoutMiddleware{
		Timeout: timeout,
	}
}

// Wrap 包装处理函数，添加超时控制
func (tm *TimeoutMiddleware) Wrap(handler func(context.Context) error) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), tm.Timeout)
		defer cancel()

		// 创建完成通道
		done := make(chan error, 1)

		// 在后台执行处理函数
		go func() {
			done <- handler(ctx)
		}()

		// 等待处理完成或上下文取消
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// WrapHandler 包装处理函数，添加超时控制
func WrapHandler(timeout time.Duration, handler func(context.Context) error) func() error {
	return NewTimeoutMiddleware(timeout).Wrap(handler)
}

// RunWithTimeout 使用超时运行函数
func RunWithTimeout(timeout time.Duration, fn func() error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 创建完成通道
	done := make(chan error, 1)

	// 在后台执行函数
	go func() {
		done <- fn()
	}()

	// 等待函数完成或上下文取消
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
