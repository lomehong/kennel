package concurrency

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestConcurrencyController_CreatePool(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewConcurrencyController(WithControllerLogger(logger))

	// 创建工作池
	pool, err := controller.CreatePool("test-pool", 4)
	assert.NoError(t, err)
	assert.NotNil(t, pool)
	assert.Equal(t, "test-pool", pool.name)
	assert.Equal(t, 4, pool.workers)

	// 验证工作池是否被正确存储
	storedPool, exists := controller.GetPool("test-pool")
	assert.True(t, exists)
	assert.Equal(t, pool, storedPool)

	// 尝试创建重复名称的工作池，应该返回错误
	_, err = controller.CreatePool("test-pool", 2)
	assert.Error(t, err)

	// 清理
	controller.Stop()
}

func TestConcurrencyController_StartStopPool(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewConcurrencyController(WithControllerLogger(logger))

	// 创建工作池
	pool, err := controller.CreatePool("test-pool", 4)
	assert.NoError(t, err)

	// 启动工作池
	err = controller.StartPool("test-pool")
	assert.NoError(t, err)

	// 提交任务
	resultChan, err := pool.SubmitFunc("test-task", "Test Task", func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 等待任务完成
	result := <-resultChan
	assert.NoError(t, result.Error)

	// 停止工作池
	err = controller.StopPool("test-pool")
	assert.NoError(t, err)

	// 尝试启动不存在的工作池，应该返回错误
	err = controller.StartPool("non-existent")
	assert.Error(t, err)

	// 尝试停止不存在的工作池，应该返回错误
	err = controller.StopPool("non-existent")
	assert.Error(t, err)

	// 清理
	controller.Stop()
}

func TestConcurrencyController_RemovePool(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewConcurrencyController(WithControllerLogger(logger))

	// 创建工作池
	_, err := controller.CreatePool("test-pool", 4)
	assert.NoError(t, err)

	// 移除工作池
	err = controller.RemovePool("test-pool")
	assert.NoError(t, err)

	// 验证工作池是否被正确移除
	_, exists := controller.GetPool("test-pool")
	assert.False(t, exists)

	// 尝试移除不存在的工作池，应该返回错误
	err = controller.RemovePool("non-existent")
	assert.Error(t, err)

	// 清理
	controller.Stop()
}

func TestConcurrencyController_ListPools(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewConcurrencyController(WithControllerLogger(logger))

	// 创建多个工作池
	_, err := controller.CreatePool("pool1", 2)
	assert.NoError(t, err)
	_, err = controller.CreatePool("pool2", 3)
	assert.NoError(t, err)

	// 列出所有工作池
	pools := controller.ListPools()
	assert.Len(t, pools, 2)
	assert.Contains(t, pools, "pool1")
	assert.Contains(t, pools, "pool2")

	// 清理
	controller.Stop()
}

func TestConcurrencyController_GetPoolStats(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewConcurrencyController(WithControllerLogger(logger))

	// 创建工作池
	pool, err := controller.CreatePool("test-pool", 4)
	assert.NoError(t, err)

	// 启动工作池
	pool.Start()

	// 提交任务
	resultChan, err := pool.SubmitFunc("test-task", "Test Task", func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 等待任务完成
	<-resultChan

	// 获取工作池统计信息
	stats, err := controller.GetPoolStats("test-pool")
	assert.NoError(t, err)
	assert.Equal(t, "test-pool", stats["name"])
	assert.Equal(t, 4, stats["workers"])
	assert.Equal(t, int64(1), stats["task_count"])
	assert.Equal(t, int64(1), stats["success_count"])
	assert.Equal(t, int64(0), stats["failure_count"])

	// 获取所有工作池统计信息
	allStats := controller.GetAllPoolStats()
	assert.Len(t, allStats, 1)
	assert.Contains(t, allStats, "test-pool")

	// 尝试获取不存在的工作池统计信息，应该返回错误
	_, err = controller.GetPoolStats("non-existent")
	assert.Error(t, err)

	// 清理
	controller.Stop()
}

func TestConcurrencyController_AcquireReleaseResource(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewConcurrencyController(
		WithControllerLogger(logger),
		WithResourceLimit(ResourceTypeCPU, 10),
	)

	// 获取资源
	success := controller.AcquireResource(ResourceTypeCPU, 5)
	assert.True(t, success)

	// 验证资源使用情况
	usage := controller.GetResourceUsage()
	assert.Equal(t, 10, usage["cpu"]["limit"])
	assert.Equal(t, 5, usage["cpu"]["used"])

	// 尝试获取超过限制的资源，应该失败
	success = controller.AcquireResource(ResourceTypeCPU, 6)
	assert.False(t, success)

	// 释放资源
	controller.ReleaseResource(ResourceTypeCPU, 3)

	// 验证资源使用情况
	usage = controller.GetResourceUsage()
	assert.Equal(t, 2, usage["cpu"]["used"])

	// 清理
	controller.Stop()
}

func TestConcurrencyController_WithResource(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewConcurrencyController(
		WithControllerLogger(logger),
		WithResourceLimit(ResourceTypeCPU, 10),
	)

	// 使用资源执行函数
	executed := false
	err := controller.WithResource(ResourceTypeCPU, 5, func() error {
		executed = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, executed)

	// 验证资源使用情况（应该已释放）
	usage := controller.GetResourceUsage()
	assert.Equal(t, 0, usage["cpu"]["used"])

	// 尝试使用超过限制的资源，应该返回错误
	err = controller.WithResource(ResourceTypeCPU, 11, func() error {
		return nil
	})
	assert.Error(t, err)

	// 清理
	controller.Stop()
}

func TestConcurrencyController_WithResourceContext(t *testing.T) {
	logger := hclog.NewNullLogger()
	controller := NewConcurrencyController(
		WithControllerLogger(logger),
		WithResourceLimit(ResourceTypeCPU, 10),
	)

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 使用资源执行带上下文的函数
	executed := false
	err := controller.WithResourceContext(ctx, ResourceTypeCPU, 5, func(ctx context.Context) error {
		executed = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, executed)

	// 验证资源使用情况（应该已释放）
	usage := controller.GetResourceUsage()
	assert.Equal(t, 0, usage["cpu"]["used"])

	// 清理
	controller.Stop()
}
