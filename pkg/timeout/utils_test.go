package timeout

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIOOperation(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 测试正常完成的操作
	err := IOOperation(ctx, "successful operation", func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 测试返回错误的操作
	expectedErr := errors.New("operation failed")
	err = IOOperation(ctx, "failing operation", func() error {
		return expectedErr
	})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// 测试超时的操作
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = IOOperation(ctx, "timeout operation", func() error {
		time.Sleep(500 * time.Millisecond)
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IO操作超时或取消")
}

func TestReadWithTimeout(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 创建一个读取器
	data := []byte("test data")
	reader := bytes.NewReader(data)

	// 测试正常读取
	buffer := make([]byte, len(data))
	n, err := ReadWithTimeout(ctx, reader, buffer)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, buffer)

	// 测试超时读取
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 创建一个会阻塞的读取器
	blockingReader := &blockingReader{}
	buffer = make([]byte, 10)
	_, err = ReadWithTimeout(ctx, blockingReader, buffer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "读取操作超时或取消")
}

func TestWriteWithTimeout(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 创建一个写入器
	buffer := &bytes.Buffer{}

	// 测试正常写入
	data := []byte("test data")
	n, err := WriteWithTimeout(ctx, buffer, data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, buffer.Bytes())

	// 测试超时写入
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 创建一个会阻塞的写入器
	blockingWriter := &blockingWriter{}
	_, err = WriteWithTimeout(ctx, blockingWriter, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "写入操作超时或取消")
}

func TestCloseWithTimeout(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// 创建一个可关闭的对象
	closer := &testCloser{
		closeFunc: func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}

	// 测试正常关闭
	err := CloseWithTimeout(ctx, closer)
	assert.NoError(t, err)

	// 测试返回错误的关闭
	closer = &testCloser{
		closeFunc: func() error {
			return errors.New("close failed")
		},
	}
	err = CloseWithTimeout(ctx, closer)
	assert.Error(t, err)
	assert.Equal(t, "close failed", err.Error())

	// 测试超时关闭
	ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	closer = &testCloser{
		closeFunc: func() error {
			time.Sleep(500 * time.Millisecond)
			return nil
		},
	}
	err = CloseWithTimeout(ctx, closer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "关闭操作超时或取消")
}

func TestRetryWithTimeout(t *testing.T) {
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试成功的重试
	attempts := 0
	err := RetryWithTimeout(ctx, "successful retry", 3, 100*time.Millisecond, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)

	// 测试失败的重试
	attempts = 0
	err = RetryWithTimeout(ctx, "failing retry", 3, 100*time.Millisecond, func() error {
		attempts++
		return errors.New("persistent error")
	})
	assert.Error(t, err)
	assert.Equal(t, 3, attempts)
	assert.Contains(t, err.Error(), "操作失败，已重试3次")

	// 测试超时的重试
	ctx, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	attempts = 0
	err = RetryWithTimeout(ctx, "timeout retry", 5, 100*time.Millisecond, func() error {
		attempts++
		time.Sleep(150 * time.Millisecond)
		return errors.New("temporary error")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "操作已取消")
	assert.Less(t, attempts, 5) // 应该在完成所有尝试前超时
}

func TestTimeoutMiddleware(t *testing.T) {
	// 创建超时中间件
	middleware := NewTimeoutMiddleware(500 * time.Millisecond)

	// 测试正常完成的处理函数
	handler := middleware.Wrap(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	err := handler()
	assert.NoError(t, err)

	// 测试超时的处理函数
	handler = middleware.Wrap(func(ctx context.Context) error {
		time.Sleep(1 * time.Second)
		return nil
	})
	err = handler()
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)

	// 测试返回错误的处理函数
	expectedErr := errors.New("handler failed")
	handler = middleware.Wrap(func(ctx context.Context) error {
		return expectedErr
	})
	err = handler()
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// 测试检查上下文取消的处理函数
	handler = middleware.Wrap(func(ctx context.Context) error {
		select {
		case <-time.After(1 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	err = handler()
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestRunWithTimeout(t *testing.T) {
	// 测试正常完成的函数
	err := RunWithTimeout(500*time.Millisecond, func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 测试超时的函数
	err = RunWithTimeout(100*time.Millisecond, func() error {
		time.Sleep(500 * time.Millisecond)
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)

	// 测试返回错误的函数
	expectedErr := errors.New("function failed")
	err = RunWithTimeout(500*time.Millisecond, func() error {
		return expectedErr
	})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// 测试辅助类型

// blockingReader 是一个会永远阻塞的读取器
type blockingReader struct{}

func (r *blockingReader) Read(p []byte) (int, error) {
	// 永远阻塞
	select {}
}

// blockingWriter 是一个会永远阻塞的写入器
type blockingWriter struct{}

func (w *blockingWriter) Write(p []byte) (int, error) {
	// 永远阻塞
	select {}
}

// testCloser 是一个可配置的关闭器
type testCloser struct {
	closeFunc func() error
}

func (c *testCloser) Close() error {
	return c.closeFunc()
}
