package concurrency

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-hclog"
)

// TaskFunc 表示一个任务函数
type TaskFunc func() error

// TaskWithContextFunc 表示一个带上下文的任务函数
type TaskWithContextFunc func(ctx context.Context) error

// TaskResult 表示任务执行结果
type TaskResult struct {
	ID        string
	Error     error
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

// Task 表示一个任务
type Task struct {
	ID          string
	Description string
	Func        TaskFunc
	CtxFunc     TaskWithContextFunc
	Priority    int
	Timeout     time.Duration
	Context     context.Context
	Cancel      context.CancelFunc
	Result      chan TaskResult
	CreatedAt   time.Time
}

// WorkerPool 表示一个工作池
type WorkerPool struct {
	name           string
	workers        int
	queue          chan *Task
	priorityQueue  chan *Task
	wg             sync.WaitGroup
	logger         hclog.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	stopped        atomic.Bool
	taskCount      atomic.Int64
	successCount   atomic.Int64
	failureCount   atomic.Int64
	processingTime atomic.Int64
}

// WorkerPoolOption 工作池配置选项
type WorkerPoolOption func(*WorkerPool)

// WithLogger 设置日志记录器
func WithLogger(logger hclog.Logger) WorkerPoolOption {
	return func(wp *WorkerPool) {
		wp.logger = logger
	}
}

// WithContext 设置上下文
func WithContext(ctx context.Context) WorkerPoolOption {
	return func(wp *WorkerPool) {
		if wp.cancel != nil {
			wp.cancel()
		}
		wp.ctx, wp.cancel = context.WithCancel(ctx)
	}
}

// WithQueueSize 设置队列大小
func WithQueueSize(size int) WorkerPoolOption {
	return func(wp *WorkerPool) {
		if size <= 0 {
			size = 100
		}
		wp.queue = make(chan *Task, size)
		wp.priorityQueue = make(chan *Task, size/10)
	}
}

// NewWorkerPool 创建一个新的工作池
func NewWorkerPool(name string, workers int, options ...WorkerPoolOption) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	ctx, cancel := context.WithCancel(context.Background())

	wp := &WorkerPool{
		name:          name,
		workers:       workers,
		queue:         make(chan *Task, 100),
		priorityQueue: make(chan *Task, 10),
		logger:        hclog.NewNullLogger(),
		ctx:           ctx,
		cancel:        cancel,
	}

	// 应用选项
	for _, option := range options {
		option(wp)
	}

	return wp
}

// Start 启动工作池
func (wp *WorkerPool) Start() {
	wp.logger.Info("启动工作池", "name", wp.name, "workers", wp.workers)

	// 重置停止标志
	wp.stopped.Store(false)

	// 启动工作协程
	wp.wg.Add(wp.workers)
	for i := 0; i < wp.workers; i++ {
		workerID := i
		go wp.worker(workerID)
	}
}

// worker 工作协程
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	wp.logger.Debug("工作协程启动", "worker_id", id)

	for {
		// 检查上下文是否已取消
		if wp.ctx.Err() != nil {
			wp.logger.Debug("工作协程退出", "worker_id", id, "reason", "context canceled")
			return
		}

		// 检查是否已停止
		if wp.stopped.Load() {
			wp.logger.Debug("工作协程退出", "worker_id", id, "reason", "pool stopped")
			return
		}

		// 优先从优先级队列获取任务
		var task *Task
		var ok bool

		select {
		case task, ok = <-wp.priorityQueue:
			if !ok {
				wp.logger.Debug("优先级队列已关闭", "worker_id", id)
				return
			}
		default:
			// 优先级队列为空，从普通队列获取任务
			select {
			case task, ok = <-wp.priorityQueue:
				if !ok {
					wp.logger.Debug("优先级队列已关闭", "worker_id", id)
					return
				}
			case task, ok = <-wp.queue:
				if !ok {
					wp.logger.Debug("任务队列已关闭", "worker_id", id)
					return
				}
			case <-wp.ctx.Done():
				wp.logger.Debug("工作协程退出", "worker_id", id, "reason", "context canceled")
				return
			}
		}

		// 执行任务
		wp.executeTask(id, task)
	}
}

// executeTask 执行任务
func (wp *WorkerPool) executeTask(workerID int, task *Task) {
	wp.logger.Debug("开始执行任务", "worker_id", workerID, "task_id", task.ID, "description", task.Description)

	// 记录开始时间
	startTime := time.Now()

	// 创建任务结果
	result := TaskResult{
		ID:        task.ID,
		StartTime: startTime,
	}

	// 执行任务
	var err error
	if task.CtxFunc != nil {
		// 如果有上下文函数，使用上下文函数
		if task.Context != nil {
			err = task.CtxFunc(task.Context)
		} else {
			err = task.CtxFunc(wp.ctx)
		}
	} else if task.Func != nil {
		// 否则使用普通函数
		err = task.Func()
	} else {
		err = fmt.Errorf("任务没有可执行的函数")
	}

	// 记录结束时间
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// 更新结果
	result.Error = err
	result.EndTime = endTime
	result.Duration = duration

	// 更新统计信息
	wp.taskCount.Add(1)
	wp.processingTime.Add(int64(duration))
	if err != nil {
		wp.failureCount.Add(1)
		wp.logger.Error("任务执行失败", "worker_id", workerID, "task_id", task.ID, "error", err, "duration", duration)
	} else {
		wp.successCount.Add(1)
		wp.logger.Debug("任务执行成功", "worker_id", workerID, "task_id", task.ID, "duration", duration)
	}

	// 发送结果
	if task.Result != nil {
		select {
		case task.Result <- result:
			// 结果已发送
		default:
			wp.logger.Warn("无法发送任务结果，通道已满或已关闭", "task_id", task.ID)
		}
	}
}

// Submit 提交任务
func (wp *WorkerPool) Submit(task *Task) error {
	if wp.stopped.Load() {
		return fmt.Errorf("工作池已停止")
	}

	// 如果任务有超时设置，创建带超时的上下文
	if task.Timeout > 0 && task.CtxFunc != nil {
		var ctx context.Context
		var cancel context.CancelFunc
		if task.Context != nil {
			ctx, cancel = context.WithTimeout(task.Context, task.Timeout)
		} else {
			ctx, cancel = context.WithTimeout(wp.ctx, task.Timeout)
		}
		task.Context = ctx
		task.Cancel = cancel
	}

	// 根据优先级提交任务
	if task.Priority > 0 {
		select {
		case wp.priorityQueue <- task:
			return nil
		case <-wp.ctx.Done():
			return fmt.Errorf("工作池已取消")
		default:
			return fmt.Errorf("优先级队列已满")
		}
	} else {
		select {
		case wp.queue <- task:
			return nil
		case <-wp.ctx.Done():
			return fmt.Errorf("工作池已取消")
		default:
			return fmt.Errorf("任务队列已满")
		}
	}
}

// SubmitFunc 提交任务函数
func (wp *WorkerPool) SubmitFunc(id string, description string, fn TaskFunc) (chan TaskResult, error) {
	resultChan := make(chan TaskResult, 1)
	task := &Task{
		ID:          id,
		Description: description,
		Func:        fn,
		Result:      resultChan,
		CreatedAt:   time.Now(),
	}
	err := wp.Submit(task)
	if err != nil {
		close(resultChan)
		return nil, err
	}
	return resultChan, nil
}

// SubmitWithContext 提交带上下文的任务
func (wp *WorkerPool) SubmitWithContext(id string, description string, ctx context.Context, fn TaskWithContextFunc) (chan TaskResult, error) {
	resultChan := make(chan TaskResult, 1)
	task := &Task{
		ID:          id,
		Description: description,
		CtxFunc:     fn,
		Context:     ctx,
		Result:      resultChan,
		CreatedAt:   time.Now(),
	}
	err := wp.Submit(task)
	if err != nil {
		close(resultChan)
		return nil, err
	}
	return resultChan, nil
}

// SubmitWithTimeout 提交带超时的任务
func (wp *WorkerPool) SubmitWithTimeout(id string, description string, timeout time.Duration, fn TaskWithContextFunc) (chan TaskResult, error) {
	resultChan := make(chan TaskResult, 1)
	task := &Task{
		ID:          id,
		Description: description,
		CtxFunc:     fn,
		Timeout:     timeout,
		Result:      resultChan,
		CreatedAt:   time.Now(),
	}
	err := wp.Submit(task)
	if err != nil {
		close(resultChan)
		return nil, err
	}
	return resultChan, nil
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	if wp.stopped.Load() {
		return
	}

	wp.logger.Info("停止工作池", "name", wp.name)
	wp.stopped.Store(true)
	wp.cancel()
	wp.wg.Wait()
}

// Wait 等待所有任务完成
func (wp *WorkerPool) Wait() {
	wp.logger.Info("等待所有任务完成", "name", wp.name)
	wp.wg.Wait()
}

// Stats 获取工作池统计信息
func (wp *WorkerPool) Stats() map[string]interface{} {
	taskCount := wp.taskCount.Load()
	processingTime := wp.processingTime.Load()
	
	var avgProcessingTime int64
	if taskCount > 0 {
		avgProcessingTime = processingTime / taskCount
	}
	
	return map[string]interface{}{
		"name":               wp.name,
		"workers":            wp.workers,
		"queue_size":         len(wp.queue),
		"priority_queue_size": len(wp.priorityQueue),
		"task_count":         taskCount,
		"success_count":      wp.successCount.Load(),
		"failure_count":      wp.failureCount.Load(),
		"avg_processing_time": time.Duration(avgProcessingTime).String(),
		"total_processing_time": time.Duration(processingTime).String(),
	}
}
