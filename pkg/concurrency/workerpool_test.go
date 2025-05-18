package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestWorkerPool_Start(t *testing.T) {
	logger := hclog.NewNullLogger()
	pool := NewWorkerPool("test-pool", 4, WithLogger(logger))

	// 启动工作池
	pool.Start()

	// 验证工作池是否已启动
	assert.False(t, pool.stopped.Load())

	// 清理
	pool.Stop()
}

func TestWorkerPool_Submit(t *testing.T) {
	logger := hclog.NewNullLogger()
	pool := NewWorkerPool("test-pool", 4, WithLogger(logger))

	// 启动工作池
	pool.Start()

	// 创建任务
	resultChan := make(chan TaskResult, 1)
	task := &Task{
		ID:          "test-task",
		Description: "Test Task",
		Func: func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
		Result:    resultChan,
		CreatedAt: time.Now(),
	}

	// 提交任务
	err := pool.Submit(task)
	assert.NoError(t, err)

	// 等待任务完成
	result := <-resultChan
	assert.Equal(t, "test-task", result.ID)
	assert.NoError(t, result.Error)
	assert.True(t, result.Duration >= 100*time.Millisecond)

	// 清理
	pool.Stop()
}

func TestWorkerPool_SubmitFunc(t *testing.T) {
	logger := hclog.NewNullLogger()
	pool := NewWorkerPool("test-pool", 4, WithLogger(logger))

	// 启动工作池
	pool.Start()

	// 提交任务函数
	resultChan, err := pool.SubmitFunc("test-task", "Test Task", func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 等待任务完成
	result := <-resultChan
	assert.Equal(t, "test-task", result.ID)
	assert.NoError(t, result.Error)
	assert.True(t, result.Duration >= 100*time.Millisecond)

	// 清理
	pool.Stop()
}

func TestWorkerPool_SubmitWithContext(t *testing.T) {
	logger := hclog.NewNullLogger()
	pool := NewWorkerPool("test-pool", 4, WithLogger(logger))

	// 启动工作池
	pool.Start()

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 提交带上下文的任务
	resultChan, err := pool.SubmitWithContext("test-task", "Test Task", ctx, func(ctx context.Context) error {
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	assert.NoError(t, err)

	// 等待任务完成
	result := <-resultChan
	assert.Equal(t, "test-task", result.ID)
	assert.NoError(t, result.Error)
	assert.True(t, result.Duration >= 100*time.Millisecond)

	// 清理
	pool.Stop()
}

func TestWorkerPool_SubmitWithTimeout(t *testing.T) {
	logger := hclog.NewNullLogger()
	pool := NewWorkerPool("test-pool", 4, WithLogger(logger))

	// 启动工作池
	pool.Start()

	// 提交带超时的任务（正常完成）
	resultChan, err := pool.SubmitWithTimeout("success-task", "Success Task", 5*time.Second, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 等待任务完成
	result := <-resultChan
	assert.Equal(t, "success-task", result.ID)
	assert.NoError(t, result.Error)

	// 提交带超时的任务（超时）
	resultChan, err = pool.SubmitWithTimeout("timeout-task", "Timeout Task", 100*time.Millisecond, func(ctx context.Context) error {
		select {
		case <-time.After(500 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	assert.NoError(t, err)

	// 等待任务完成
	result = <-resultChan
	assert.Equal(t, "timeout-task", result.ID)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "context deadline exceeded")

	// 清理
	pool.Stop()
}

func TestWorkerPool_Stop(t *testing.T) {
	logger := hclog.NewNullLogger()
	pool := NewWorkerPool("test-pool", 4, WithLogger(logger))

	// 启动工作池
	pool.Start()

	// 停止工作池
	pool.Stop()

	// 验证工作池是否已停止
	assert.True(t, pool.stopped.Load())

	// 尝试提交任务，应该返回错误
	err := pool.Submit(&Task{
		ID:          "test-task",
		Description: "Test Task",
		Func: func() error {
			return nil
		},
	})
	assert.Error(t, err)
}

func TestWorkerPool_Stats(t *testing.T) {
	logger := hclog.NewNullLogger()
	pool := NewWorkerPool("test-pool", 4, WithLogger(logger))

	// 启动工作池
	pool.Start()

	// 提交成功任务
	resultChan, err := pool.SubmitFunc("success-task", "Success Task", func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})
	assert.NoError(t, err)

	// 等待任务完成
	<-resultChan

	// 提交失败任务
	resultChan, err = pool.SubmitFunc("failure-task", "Failure Task", func() error {
		time.Sleep(100 * time.Millisecond)
		return errors.New("task failed")
	})
	assert.NoError(t, err)

	// 等待任务完成
	<-resultChan

	// 获取统计信息
	stats := pool.Stats()
	assert.Equal(t, "test-pool", stats["name"])
	assert.Equal(t, 4, stats["workers"])
	assert.Equal(t, int64(2), stats["task_count"])
	assert.Equal(t, int64(1), stats["success_count"])
	assert.Equal(t, int64(1), stats["failure_count"])

	// 清理
	pool.Stop()
}

func TestWorkerPool_PriorityQueue(t *testing.T) {
	logger := hclog.NewNullLogger()
	pool := NewWorkerPool("test-pool", 1, WithLogger(logger))

	// 启动工作池
	pool.Start()

	// 创建通道记录执行顺序
	executionOrder := make(chan string, 3)

	// 提交普通任务
	task1 := &Task{
		ID:          "normal-task",
		Description: "Normal Task",
		Func: func() error {
			time.Sleep(200 * time.Millisecond)
			executionOrder <- "normal"
			return nil
		},
		Priority: 0,
	}
	err := pool.Submit(task1)
	assert.NoError(t, err)

	// 提交高优先级任务
	task2 := &Task{
		ID:          "high-priority-task",
		Description: "High Priority Task",
		Func: func() error {
			executionOrder <- "high"
			return nil
		},
		Priority: 10,
	}
	err = pool.Submit(task2)
	assert.NoError(t, err)

	// 提交另一个普通任务
	task3 := &Task{
		ID:          "another-normal-task",
		Description: "Another Normal Task",
		Func: func() error {
			executionOrder <- "another"
			return nil
		},
		Priority: 0,
	}
	err = pool.Submit(task3)
	assert.NoError(t, err)

	// 等待所有任务完成
	first := <-executionOrder
	second := <-executionOrder
	third := <-executionOrder

	// 验证高优先级任务是否先执行
	assert.Equal(t, "high", first)
	assert.Equal(t, "normal", second)
	assert.Equal(t, "another", third)

	// 清理
	pool.Stop()
}
