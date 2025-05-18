package concurrency

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/go-hclog"
)

// 以下是并发控制器和工作池的使用示例

// ExampleWorkerPoolBasic 展示工作池的基本用法
func ExampleWorkerPoolBasic() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "worker-pool",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建工作池
	pool := NewWorkerPool("example-pool", 4, WithLogger(logger))

	// 启动工作池
	pool.Start()

	// 提交任务
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("task-%d", i)
		description := fmt.Sprintf("Task %d", i)

		// 创建任务函数
		taskFunc := func(taskID string) TaskFunc {
			return func() error {
				logger.Info("执行任务", "id", taskID)
				taskNum, _ := strconv.Atoi(taskID[5:7])
				time.Sleep(time.Duration(100*taskNum) * time.Millisecond)
				logger.Info("任务完成", "id", taskID)
				return nil
			}
		}

		// 提交任务
		resultChan, err := pool.SubmitFunc(id, description, taskFunc(id))
		if err != nil {
			logger.Error("提交任务失败", "id", id, "error", err)
			continue
		}

		// 处理结果（异步）
		go func(taskID string) {
			result := <-resultChan
			if result.Error != nil {
				logger.Error("任务执行失败", "id", taskID, "error", result.Error)
			} else {
				logger.Info("任务执行成功", "id", taskID, "duration", result.Duration)
			}
		}(id)
	}

	// 等待所有任务完成
	time.Sleep(2 * time.Second)

	// 获取统计信息
	stats := pool.Stats()
	logger.Info("工作池统计信息", "stats", stats)

	// 停止工作池
	pool.Stop()
}

// ExampleWorkerPoolWithContext 展示带上下文的工作池用法
func ExampleWorkerPoolWithContext() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "worker-pool-context",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建工作池
	pool := NewWorkerPool("example-pool", 4, WithLogger(logger), WithContext(ctx))

	// 启动工作池
	pool.Start()

	// 提交带上下文的任务
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("ctx-task-%d", i)
		description := fmt.Sprintf("Context Task %d", i)

		// 创建任务函数
		taskFunc := func(taskID string, index int) TaskWithContextFunc {
			return func(ctx context.Context) error {
				logger.Info("执行带上下文的任务", "id", taskID)

				select {
				case <-time.After(time.Duration(index*500) * time.Millisecond):
					logger.Info("任务完成", "id", taskID)
					return nil
				case <-ctx.Done():
					logger.Warn("任务被取消", "id", taskID, "reason", ctx.Err())
					return ctx.Err()
				}
			}
		}

		// 提交任务
		resultChan, err := pool.SubmitWithContext(id, description, ctx, taskFunc(id, i))
		if err != nil {
			logger.Error("提交任务失败", "id", id, "error", err)
			continue
		}

		// 处理结果（异步）
		go func(taskID string) {
			result := <-resultChan
			if result.Error != nil {
				logger.Error("任务执行失败", "id", taskID, "error", result.Error)
			} else {
				logger.Info("任务执行成功", "id", taskID, "duration", result.Duration)
			}
		}(id)
	}

	// 等待一段时间后取消上下文
	time.Sleep(1500 * time.Millisecond)
	logger.Info("取消上下文")
	cancel()

	// 等待所有任务完成
	time.Sleep(500 * time.Millisecond)

	// 获取统计信息
	stats := pool.Stats()
	logger.Info("工作池统计信息", "stats", stats)

	// 停止工作池
	pool.Stop()
}

// ExampleWorkerPoolWithTimeout 展示带超时的工作池用法
func ExampleWorkerPoolWithTimeout() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "worker-pool-timeout",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建工作池
	pool := NewWorkerPool("example-pool", 4, WithLogger(logger))

	// 启动工作池
	pool.Start()

	// 提交带超时的任务
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("timeout-task-%d", i)
		description := fmt.Sprintf("Timeout Task %d", i)
		timeout := time.Duration(i*300) * time.Millisecond

		// 创建任务函数
		taskFunc := func(taskID string, index int) TaskWithContextFunc {
			return func(ctx context.Context) error {
				logger.Info("执行带超时的任务", "id", taskID, "timeout", timeout)

				select {
				case <-time.After(time.Second): // 所有任务都尝试运行1秒
					logger.Info("任务完成", "id", taskID)
					return nil
				case <-ctx.Done():
					logger.Warn("任务超时或被取消", "id", taskID, "reason", ctx.Err())
					return ctx.Err()
				}
			}
		}

		// 提交任务
		resultChan, err := pool.SubmitWithTimeout(id, description, timeout, taskFunc(id, i))
		if err != nil {
			logger.Error("提交任务失败", "id", id, "error", err)
			continue
		}

		// 处理结果（异步）
		go func(taskID string) {
			result := <-resultChan
			if result.Error != nil {
				logger.Error("任务执行失败", "id", taskID, "error", result.Error)
			} else {
				logger.Info("任务执行成功", "id", taskID, "duration", result.Duration)
			}
		}(id)
	}

	// 等待所有任务完成
	time.Sleep(2 * time.Second)

	// 获取统计信息
	stats := pool.Stats()
	logger.Info("工作池统计信息", "stats", stats)

	// 停止工作池
	pool.Stop()
}

// ExampleConcurrencyController 展示并发控制器的用法
func ExampleConcurrencyController() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "concurrency-controller",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建并发控制器
	controller := NewConcurrencyController(
		WithControllerLogger(logger),
		WithResourceLimit(ResourceTypeCPU, 8),
		WithResourceLimit(ResourceTypeIO, 4),
	)

	// 创建工作池
	cpuPool, err := controller.CreatePool("cpu-pool", 8)
	if err != nil {
		logger.Error("创建CPU工作池失败", "error", err)
		return
	}

	ioPool, err := controller.CreatePool("io-pool", 4)
	if err != nil {
		logger.Error("创建IO工作池失败", "error", err)
		return
	}

	// 启动工作池
	controller.StartPool("cpu-pool")
	controller.StartPool("io-pool")

	// 使用CPU资源
	for i := 1; i <= 5; i++ {
		i := i // 捕获变量
		err := controller.WithResource(ResourceTypeCPU, 2, func() error {
			logger.Info("使用CPU资源", "index", i, "count", 2)

			// 提交CPU密集型任务
			id := fmt.Sprintf("cpu-task-%d", i)
			resultChan, err := cpuPool.SubmitFunc(id, "CPU Task", func() error {
				logger.Info("执行CPU密集型任务", "id", id)
				time.Sleep(500 * time.Millisecond)
				logger.Info("CPU密集型任务完成", "id", id)
				return nil
			})

			if err != nil {
				return err
			}

			// 等待任务完成
			result := <-resultChan
			return result.Error
		})

		if err != nil {
			logger.Error("使用CPU资源失败", "index", i, "error", err)
		}
	}

	// 使用IO资源
	for i := 1; i <= 3; i++ {
		i := i // 捕获变量
		err := controller.WithResource(ResourceTypeIO, 1, func() error {
			logger.Info("使用IO资源", "index", i, "count", 1)

			// 提交IO密集型任务
			id := fmt.Sprintf("io-task-%d", i)
			resultChan, err := ioPool.SubmitFunc(id, "IO Task", func() error {
				logger.Info("执行IO密集型任务", "id", id)
				time.Sleep(800 * time.Millisecond)
				logger.Info("IO密集型任务完成", "id", id)
				return nil
			})

			if err != nil {
				return err
			}

			// 等待任务完成
			result := <-resultChan
			return result.Error
		})

		if err != nil {
			logger.Error("使用IO资源失败", "index", i, "error", err)
		}
	}

	// 获取资源使用情况
	usage := controller.GetResourceUsage()
	logger.Info("资源使用情况", "usage", usage)

	// 获取工作池统计信息
	stats := controller.GetAllPoolStats()
	logger.Info("工作池统计信息", "stats", stats)

	// 停止并发控制器
	controller.Stop()
}

// ExampleConcurrencyPatterns 展示并发模式的用法
func ExampleConcurrencyPatterns() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "concurrency-patterns",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建输入通道
	in := make(chan interface{})

	// 发送数据
	go func() {
		for i := 1; i <= 10; i++ {
			in <- i
			time.Sleep(100 * time.Millisecond)
		}
		close(in)
	}()

	// 创建处理阶段
	stage1 := func(in <-chan interface{}) <-chan interface{} {
		out := make(chan interface{})
		go func() {
			defer close(out)
			for v := range in {
				logger.Info("阶段1处理", "value", v)
				out <- v.(int) * 2
			}
		}()
		return out
	}

	stage2 := func(in <-chan interface{}) <-chan interface{} {
		out := make(chan interface{})
		go func() {
			defer close(out)
			for v := range in {
				logger.Info("阶段2处理", "value", v)
				out <- v.(int) + 1
			}
		}()
		return out
	}

	// 创建管道
	pipeline := Pipeline(in, stage1, stage2)

	// 使用扇出模式并行处理
	fanOut := FanOut(pipeline, 3, func(v interface{}) interface{} {
		val := v.(int)
		logger.Info("扇出处理", "value", val)
		time.Sleep(200 * time.Millisecond)
		return strconv.Itoa(val)
	})

	// 使用OrDone模式处理取消
	orDone := OrDone(ctx, fanOut)

	// 收集结果
	var results []string
	for v := range orDone {
		logger.Info("收到结果", "value", v)
		results = append(results, v.(string))
	}

	logger.Info("处理完成", "results", results)
}
