package core

import (
	"context"
	"fmt"
	"time"

	"github.com/lomehong/kennel/pkg/concurrency"
)

// 添加并发控制器到App结构体
func (app *App) initConcurrencyController() {
	// 创建并发控制器
	app.concurrencyController = concurrency.NewConcurrencyController(
		concurrency.WithControllerLogger(app.logger.Named("concurrency-controller")),
		concurrency.WithControllerContext(app.ctx),
		concurrency.WithResourceLimit(concurrency.ResourceTypeCPU, app.configManager.GetIntOrDefault("concurrency.cpu_limit", 8)),
		concurrency.WithResourceLimit(concurrency.ResourceTypeMemory, app.configManager.GetIntOrDefault("concurrency.memory_limit", 100)),
		concurrency.WithResourceLimit(concurrency.ResourceTypeIO, app.configManager.GetIntOrDefault("concurrency.io_limit", 50)),
		concurrency.WithResourceLimit(concurrency.ResourceTypeNetwork, app.configManager.GetIntOrDefault("concurrency.network_limit", 30)),
		concurrency.WithDefaultLimit(app.configManager.GetIntOrDefault("concurrency.default_limit", 10)),
	)

	// 创建默认工作池
	defaultPoolSize := app.configManager.GetIntOrDefault("concurrency.default_pool_size", 10)
	defaultPool, err := app.concurrencyController.CreatePool("default", defaultPoolSize)
	if err != nil {
		app.logger.Error("创建默认工作池失败", "error", err)
	} else {
		app.logger.Info("创建默认工作池成功", "size", defaultPoolSize)
		defaultPool.Start()
	}

	// 创建IO工作池
	ioPoolSize := app.configManager.GetIntOrDefault("concurrency.io_pool_size", 20)
	ioPool, err := app.concurrencyController.CreatePool("io", ioPoolSize)
	if err != nil {
		app.logger.Error("创建IO工作池失败", "error", err)
	} else {
		app.logger.Info("创建IO工作池成功", "size", ioPoolSize)
		ioPool.Start()
	}

	// 创建网络工作池
	networkPoolSize := app.configManager.GetIntOrDefault("concurrency.network_pool_size", 30)
	networkPool, err := app.concurrencyController.CreatePool("network", networkPoolSize)
	if err != nil {
		app.logger.Error("创建网络工作池失败", "error", err)
	} else {
		app.logger.Info("创建网络工作池成功", "size", networkPoolSize)
		networkPool.Start()
	}

	app.logger.Info("并发控制器已初始化")
}

// GetConcurrencyController 获取并发控制器
func (app *App) GetConcurrencyController() *concurrency.ConcurrencyController {
	return app.concurrencyController
}

// GetWorkerPool 获取工作池
func (app *App) GetWorkerPool(name string) (*concurrency.WorkerPool, bool) {
	if app.concurrencyController == nil {
		return nil, false
	}
	return app.concurrencyController.GetPool(name)
}

// GetDefaultWorkerPool 获取默认工作池
func (app *App) GetDefaultWorkerPool() *concurrency.WorkerPool {
	pool, _ := app.GetWorkerPool("default")
	return pool
}

// SubmitTask 提交任务到默认工作池
func (app *App) SubmitTask(id string, description string, fn func() error) (chan concurrency.TaskResult, error) {
	pool := app.GetDefaultWorkerPool()
	if pool == nil {
		return nil, fmt.Errorf("默认工作池不存在")
	}
	return pool.SubmitFunc(id, description, fn)
}

// SubmitTaskWithContext 提交带上下文的任务到默认工作池
func (app *App) SubmitTaskWithContext(id string, description string, ctx context.Context, fn func(context.Context) error) (chan concurrency.TaskResult, error) {
	pool := app.GetDefaultWorkerPool()
	if pool == nil {
		return nil, fmt.Errorf("默认工作池不存在")
	}
	return pool.SubmitWithContext(id, description, ctx, fn)
}

// SubmitTaskWithTimeout 提交带超时的任务到默认工作池
func (app *App) SubmitTaskWithTimeout(id string, description string, timeout time.Duration, fn func(context.Context) error) (chan concurrency.TaskResult, error) {
	pool := app.GetDefaultWorkerPool()
	if pool == nil {
		return nil, fmt.Errorf("默认工作池不存在")
	}
	return pool.SubmitWithTimeout(id, description, timeout, fn)
}

// WithResource 使用资源执行函数
func (app *App) WithResource(resourceType concurrency.ResourceType, count int, fn func() error) error {
	if app.concurrencyController == nil {
		return fn()
	}
	return app.concurrencyController.WithResource(resourceType, count, fn)
}

// WithResourceContext 使用资源执行带上下文的函数
func (app *App) WithResourceContext(ctx context.Context, resourceType concurrency.ResourceType, count int, fn func(context.Context) error) error {
	if app.concurrencyController == nil {
		return fn(ctx)
	}
	return app.concurrencyController.WithResourceContext(ctx, resourceType, count, fn)
}

// GetResourceUsage 获取资源使用情况
func (app *App) GetResourceUsage() map[string]map[string]int {
	if app.concurrencyController == nil {
		return nil
	}
	return app.concurrencyController.GetResourceUsage()
}

// GetPoolStats 获取工作池统计信息
func (app *App) GetPoolStats(name string) (map[string]interface{}, error) {
	if app.concurrencyController == nil {
		return nil, fmt.Errorf("并发控制器未初始化")
	}
	return app.concurrencyController.GetPoolStats(name)
}

// GetAllPoolStats 获取所有工作池统计信息
func (app *App) GetAllPoolStats() map[string]map[string]interface{} {
	if app.concurrencyController == nil {
		return nil
	}
	return app.concurrencyController.GetAllPoolStats()
}
