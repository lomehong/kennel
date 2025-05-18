package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lomehong/kennel/pkg/concurrency"
	"github.com/lomehong/kennel/pkg/core"
)

// 本示例展示如何在AppFramework中使用并发控制器
func main() {
	// 创建应用程序实例
	app := core.NewApp("config.yaml")

	// 初始化应用程序
	if err := app.Init(); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}

	// 获取并发控制器
	controller := app.GetConcurrencyController()
	if controller == nil {
		fmt.Println("并发控制器未初始化")
		os.Exit(1)
	}

	fmt.Println("=== 并发控制器使用示例 ===")

	// 示例1: 使用工作池
	fmt.Println("\n=== 示例1: 使用工作池 ===")
	useWorkerPool(app)

	// 示例2: 使用资源限制
	fmt.Println("\n=== 示例2: 使用资源限制 ===")
	useResourceLimits(app)

	// 示例3: 使用并发模式
	fmt.Println("\n=== 示例3: 使用并发模式 ===")
	useConcurrencyPatterns()

	// 示例4: 使用优先级队列
	fmt.Println("\n=== 示例4: 使用优先级队列 ===")
	usePriorityQueue(app)

	// 获取资源使用情况
	usage := app.GetResourceUsage()
	fmt.Println("\n资源使用情况:")
	for resourceType, info := range usage {
		fmt.Printf("- %s: 使用 %d/%d\n", resourceType, info["used"], info["limit"])
	}

	// 获取工作池统计信息
	stats := app.GetAllPoolStats()
	fmt.Println("\n工作池统计信息:")
	for name, poolStats := range stats {
		fmt.Printf("- %s: 任务数 %d, 成功 %d, 失败 %d\n",
			name, poolStats["task_count"], poolStats["success_count"], poolStats["failure_count"])
	}

	// 停止应用程序
	app.Stop()
	fmt.Println("\n应用程序已停止")
}

// 使用工作池
func useWorkerPool(app *core.App) {
	// 获取默认工作池
	pool, exists := app.GetDefaultWorkerPool()
	if !exists {
		fmt.Println("默认工作池不存在")
		return
	}

	// 提交任务
	for i := 1; i <= 5; i++ {
		i := i // 捕获变量
		id := fmt.Sprintf("task-%d", i)
		description := fmt.Sprintf("任务 %d", i)

		// 提交任务
		resultChan, err := pool.SubmitFunc(id, description, func() error {
			fmt.Printf("执行任务 %d\n", i)
			time.Sleep(time.Duration(i*100) * time.Millisecond)
			fmt.Printf("任务 %d 完成\n", i)
			return nil
		})

		if err != nil {
			fmt.Printf("提交任务失败: %v\n", err)
			continue
		}

		// 处理结果（异步）
		go func(taskID string, taskIndex int) {
			result := <-resultChan
			if result.Error != nil {
				fmt.Printf("任务 %s 执行失败: %v\n", taskID, result.Error)
			} else {
				fmt.Printf("任务 %s 执行成功，耗时: %v\n", taskID, result.Duration)
			}
		}(id, i)
	}

	// 等待所有任务完成
	time.Sleep(1 * time.Second)
}

// 使用资源限制
func useResourceLimits(app *core.App) {
	// 使用CPU资源
	err := app.WithResource(concurrency.ResourceTypeCPU, 2, func() error {
		fmt.Println("使用CPU资源（2个单位）")
		time.Sleep(500 * time.Millisecond)
		fmt.Println("CPU资源使用完成")
		return nil
	})

	if err != nil {
		fmt.Printf("使用CPU资源失败: %v\n", err)
	}

	// 使用IO资源
	err = app.WithResource(concurrency.ResourceTypeIO, 1, func() error {
		fmt.Println("使用IO资源（1个单位）")
		time.Sleep(300 * time.Millisecond)
		fmt.Println("IO资源使用完成")
		return nil
	})

	if err != nil {
		fmt.Printf("使用IO资源失败: %v\n", err)
	}

	// 使用带上下文的资源
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = app.WithResourceContext(ctx, concurrency.ResourceTypeNetwork, 1, func(ctx context.Context) error {
		fmt.Println("使用网络资源（1个单位）")

		select {
		case <-time.After(500 * time.Millisecond):
			fmt.Println("网络资源使用完成")
			return nil
		case <-ctx.Done():
			fmt.Printf("网络资源使用被取消: %v\n", ctx.Err())
			return ctx.Err()
		}
	})

	if err != nil {
		fmt.Printf("使用网络资源失败: %v\n", err)
	}
}

// 使用并发模式
func useConcurrencyPatterns() {
	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建输入通道
	in := make(chan interface{})

	// 发送数据
	go func() {
		for i := 1; i <= 5; i++ {
			in <- i
		}
		close(in)
	}()

	// 使用Map模式
	mapped := concurrency.Map(in, func(v interface{}) interface{} {
		val := v.(int)
		fmt.Printf("Map: 处理 %d\n", val)
		return val * 2
	})

	// 使用Filter模式
	filtered := concurrency.Filter(mapped, func(v interface{}) bool {
		val := v.(int)
		fmt.Printf("Filter: 检查 %d\n", val)
		return val > 5
	})

	// 使用OrDone模式
	orDone := concurrency.OrDone(ctx, filtered)

	// 收集结果
	var results []int
	for v := range orDone {
		val := v.(int)
		fmt.Printf("收到结果: %d\n", val)
		results = append(results, val)
	}

	fmt.Printf("最终结果: %v\n", results)
}

// 使用优先级队列
func usePriorityQueue(app *core.App) {
	// 获取默认工作池
	pool, exists := app.GetDefaultWorkerPool()
	if !exists {
		fmt.Println("默认工作池不存在")
		return
	}

	// 创建高优先级任务
	highPriorityTask := &concurrency.Task{
		ID:          "high-priority",
		Description: "高优先级任务",
		Func: func() error {
			fmt.Println("执行高优先级任务")
			time.Sleep(200 * time.Millisecond)
			fmt.Println("高优先级任务完成")
			return nil
		},
		Priority:  10,
		Result:    make(chan concurrency.TaskResult, 1),
		CreatedAt: time.Now(),
	}

	// 创建普通优先级任务
	normalPriorityTask := &concurrency.Task{
		ID:          "normal-priority",
		Description: "普通优先级任务",
		Func: func() error {
			fmt.Println("执行普通优先级任务")
			time.Sleep(200 * time.Millisecond)
			fmt.Println("普通优先级任务完成")
			return nil
		},
		Priority:  0,
		Result:    make(chan concurrency.TaskResult, 1),
		CreatedAt: time.Now(),
	}

	// 先提交普通优先级任务
	err := pool.Submit(normalPriorityTask)
	if err != nil {
		fmt.Printf("提交普通优先级任务失败: %v\n", err)
	}

	// 再提交高优先级任务
	err = pool.Submit(highPriorityTask)
	if err != nil {
		fmt.Printf("提交高优先级任务失败: %v\n", err)
	}

	// 等待任务完成
	highResult := <-highPriorityTask.Result
	normalResult := <-normalPriorityTask.Result

	fmt.Printf("高优先级任务结果: %v, 耗时: %v\n", highResult.Error, highResult.Duration)
	fmt.Printf("普通优先级任务结果: %v, 耗时: %v\n", normalResult.Error, normalResult.Duration)
}
