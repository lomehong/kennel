package timeout

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestTimeoutController_CreateOperation(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewTimeoutController(WithLogger(logger))

	// 创建操作
	op, err := controller.CreateOperation(OperationTypeIO, "test-op", "Test Operation", 5*time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, op)
	assert.Equal(t, "test-op", op.ID)
	assert.Equal(t, OperationTypeIO, op.Type)
	assert.Equal(t, "Test Operation", op.Description)
	assert.Equal(t, 5*time.Second, op.Timeout)

	// 验证操作是否被正确存储
	storedOp, exists := controller.GetOperation("test-op")
	assert.True(t, exists)
	assert.Equal(t, op, storedOp)

	// 尝试创建重复ID的操作，应该返回错误
	_, err = controller.CreateOperation(OperationTypeIO, "test-op", "Duplicate Operation", 5*time.Second)
	assert.Error(t, err)

	// 清理
	controller.Stop()
}

func TestTimeoutController_CancelOperation(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewTimeoutController(WithLogger(logger))

	// 创建操作
	op, err := controller.CreateOperation(OperationTypeIO, "test-op", "Test Operation", 5*time.Second)
	assert.NoError(t, err)

	// 取消操作
	err = controller.CancelOperation("test-op")
	assert.NoError(t, err)

	// 验证操作是否被正确取消
	_, exists := controller.GetOperation("test-op")
	assert.False(t, exists)

	// 验证上下文是否被取消
	assert.Error(t, op.Context.Err())

	// 尝试取消不存在的操作，应该返回错误
	err = controller.CancelOperation("non-existent")
	assert.Error(t, err)

	// 清理
	controller.Stop()
}

func TestTimeoutController_CompleteOperation(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewTimeoutController(WithLogger(logger))

	// 创建操作
	op, err := controller.CreateOperation(OperationTypeIO, "test-op", "Test Operation", 5*time.Second)
	assert.NoError(t, err)

	// 完成操作
	err = controller.CompleteOperation("test-op")
	assert.NoError(t, err)

	// 验证操作是否被正确完成
	_, exists := controller.GetOperation("test-op")
	assert.False(t, exists)

	// 验证上下文是否被取消
	assert.Error(t, op.Context.Err())

	// 尝试完成不存在的操作，应该返回错误
	err = controller.CompleteOperation("non-existent")
	assert.Error(t, err)

	// 清理
	controller.Stop()
}

func TestTimeoutController_ListOperations(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewTimeoutController(WithLogger(logger))

	// 创建多个操作
	_, err := controller.CreateOperation(OperationTypeIO, "op1", "Operation 1", 5*time.Second)
	assert.NoError(t, err)
	_, err = controller.CreateOperation(OperationTypeNetwork, "op2", "Operation 2", 10*time.Second)
	assert.NoError(t, err)

	// 列出所有操作
	operations := controller.ListOperations()
	assert.Len(t, operations, 2)

	// 验证操作列表
	ids := make(map[string]bool)
	for _, op := range operations {
		ids[op.ID] = true
	}
	assert.True(t, ids["op1"])
	assert.True(t, ids["op2"])

	// 清理
	controller.Stop()
}

func TestTimeoutController_WithTimeout(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewTimeoutController(WithLogger(logger))

	// 测试正常完成的操作
	err := controller.WithTimeout(OperationTypeIO, "success-op", "Successful Operation", 5*time.Second, func(ctx context.Context) error {
		// 模拟一些工作
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 验证操作是否被正确完成
	_, exists := controller.GetOperation("success-op")
	assert.False(t, exists)

	// 测试超时的操作
	err = controller.WithTimeout(OperationTypeIO, "timeout-op", "Timeout Operation", 100*time.Millisecond, func(ctx context.Context) error {
		// 模拟长时间运行的操作
		select {
		case <-time.After(500 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))

	// 验证操作是否被正确完成（即使超时）
	_, exists = controller.GetOperation("timeout-op")
	assert.False(t, exists)

	// 测试返回错误的操作
	expectedErr := errors.New("operation failed")
	err = controller.WithTimeout(OperationTypeIO, "error-op", "Error Operation", 5*time.Second, func(ctx context.Context) error {
		return expectedErr
	})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// 验证操作是否被正确完成（即使出错）
	_, exists = controller.GetOperation("error-op")
	assert.False(t, exists)

	// 清理
	controller.Stop()
}

func TestTimeoutController_ExecuteWithTimeout(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewTimeoutController(WithLogger(logger))

	// 测试正常完成的操作
	err := controller.ExecuteWithTimeout(OperationTypeIO, "Successful Operation", 5*time.Second, func(ctx context.Context) error {
		// 模拟一些工作
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 测试超时的操作
	err = controller.ExecuteWithTimeout(OperationTypeIO, "Timeout Operation", 100*time.Millisecond, func(ctx context.Context) error {
		// 模拟长时间运行的操作
		select {
		case <-time.After(500 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))

	// 测试返回错误的操作
	expectedErr := errors.New("operation failed")
	err = controller.ExecuteWithTimeout(OperationTypeIO, "Error Operation", 5*time.Second, func(ctx context.Context) error {
		return expectedErr
	})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// 清理
	controller.Stop()
}

func TestTimeoutController_CleanupTimedOutOperations(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewTimeoutController(
		WithLogger(logger),
		WithMonitorInterval(100*time.Millisecond),
	)

	// 创建一个会超时的操作
	_, err := controller.CreateOperation(OperationTypeIO, "timeout-op", "Timeout Operation", 200*time.Millisecond)
	assert.NoError(t, err)

	// 验证操作是否被正确存储
	_, exists := controller.GetOperation("timeout-op")
	assert.True(t, exists)

	// 等待操作超时和清理
	time.Sleep(500 * time.Millisecond)

	// 验证操作是否被自动清理
	_, exists = controller.GetOperation("timeout-op")
	assert.False(t, exists)

	// 清理
	controller.Stop()
}
