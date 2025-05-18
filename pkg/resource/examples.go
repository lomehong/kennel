package resource

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
)

// 以下是资源追踪器的使用示例

// ExampleBasicUsage 展示资源追踪器的基本用法
func ExampleBasicUsage() {
	// 创建资源追踪器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "resource-tracker",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	tracker := NewResourceTracker(
		WithTrackerLogger(logger),
		WithCleanupInterval(5*time.Minute),
	)

	// 创建并追踪文件资源
	file, err := os.Open("example.txt")
	if err != nil {
		logger.Error("打开文件失败", "error", err)
		return
	}

	fileResource := tracker.TrackFile(file)
	// 资源已经通过TrackFile方法追踪

	// 使用文件资源
	data := make([]byte, 100)
	n, err := fileResource.File().Read(data)
	if err != nil && err != io.EOF {
		logger.Error("读取文件失败", "error", err)
	} else {
		logger.Info("读取文件成功", "bytes", n)
	}

	// 释放资源
	if err := tracker.Release(fileResource.ID()); err != nil {
		logger.Error("释放资源失败", "error", err)
	}

	// 获取资源统计信息
	stats := tracker.GetStats()
	logger.Info("资源统计",
		"total_created", stats.TotalCreated,
		"total_closed", stats.TotalClosed,
		"current_active", stats.CurrentActive,
		"closure_errors", stats.ClosureErrors,
	)

	// 停止资源追踪器
	tracker.Stop()
}

// ExampleWithContext 展示与上下文关联的资源追踪器用法
func ExampleWithContext() {
	// 创建资源追踪器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "resource-tracker",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	tracker := NewResourceTracker(WithTrackerLogger(logger))

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建与上下文关联的资源追踪器
	ctxTracker := WithTrackerContext(ctx, tracker)

	// 创建并追踪文件资源
	file, err := os.Open("example.txt")
	if err != nil {
		logger.Error("打开文件失败", "error", err)
		return
	}

	fileResource := ctxTracker.TrackFile(file)

	// 使用文件资源
	data := make([]byte, 100)
	n, err := fileResource.File().Read(data)
	if err != nil && err != io.EOF {
		logger.Error("读取文件失败", "error", err)
	} else {
		logger.Info("读取文件成功", "bytes", n)
	}

	// 上下文取消时，资源会自动释放
	logger.Info("等待上下文取消...")
	<-ctx.Done()

	// 验证资源是否已释放
	_, exists := tracker.Get(fileResource.ID())
	logger.Info("资源是否存在", "exists", exists)

	// 停止资源追踪器
	tracker.Stop()
}

// ExampleMultipleResources 展示管理多种资源的用法
func ExampleMultipleResources() {
	// 创建资源追踪器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "resource-tracker",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	tracker := NewResourceTracker(WithTrackerLogger(logger))

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建与上下文关联的资源追踪器
	ctxTracker := WithTrackerContext(ctx, tracker)

	// 创建并追踪文件资源
	file, err := os.Open("example.txt")
	if err == nil {
		fileResource := ctxTracker.TrackFile(file)
		logger.Info("文件资源已追踪", "id", fileResource.ID())
	}

	// 创建并追踪网络连接资源
	conn, err := net.Dial("tcp", "example.com:80")
	if err == nil {
		netResource := ctxTracker.TrackNetwork(conn)
		logger.Info("网络资源已追踪", "id", netResource.ID(), "remote", netResource.RemoteAddr())
	}

	// 创建并追踪数据库连接资源
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
	if err == nil {
		dbResource := ctxTracker.TrackDatabase(db, "mysql://localhost:3306/dbname")
		logger.Info("数据库资源已追踪", "id", dbResource.ID(), "conn", dbResource.ConnInfo())
	}

	// 创建并追踪自定义资源
	customResource := ctxTracker.TrackGeneric("custom:1", "custom-resource", func() error {
		logger.Info("关闭自定义资源")
		return nil
	})
	logger.Info("自定义资源已追踪", "id", customResource.ID())

	// 列出所有资源
	resources := tracker.ListResources()
	logger.Info("当前资源数量", "count", len(resources))
	for _, res := range resources {
		logger.Info("资源", "id", res.ID(), "type", res.Type(), "created", res.CreatedAt())
	}

	// 取消上下文，释放所有资源
	logger.Info("取消上下文，释放所有资源")
	cancel()

	// 等待资源释放
	time.Sleep(100 * time.Millisecond)

	// 验证资源是否已释放
	resources = tracker.ListResources()
	logger.Info("释放后资源数量", "count", len(resources))

	// 获取资源统计信息
	stats := tracker.GetStats()
	logger.Info("资源统计",
		"total_created", stats.TotalCreated,
		"total_closed", stats.TotalClosed,
		"current_active", stats.CurrentActive,
		"closure_errors", stats.ClosureErrors,
	)

	// 停止资源追踪器
	tracker.Stop()
}

// ExampleResourceLeakDetection 展示资源泄漏检测的用法
func ExampleResourceLeakDetection() {
	// 创建资源追踪器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "resource-tracker",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 设置较短的清理间隔，用于演示
	tracker := NewResourceTracker(
		WithTrackerLogger(logger),
		WithCleanupInterval(1*time.Second),
	)

	// 创建一些资源但不显式释放它们
	for i := 0; i < 5; i++ {
		// 创建一个通用资源
		resource := NewGenericResource(
			fmt.Sprintf("leak:%d", i),
			fmt.Sprintf("leaky-resource-%d", i),
			func() error {
				logger.Info("关闭泄漏资源")
				return nil
			},
		)
		tracker.Track(resource)

		// 只更新部分资源的最后使用时间
		if i%2 == 0 {
			resource.UpdateLastUsed()
		} else {
			// 将最后使用时间设置为过去，模拟泄漏
			resource.lastUsedAt = time.Now().Add(-10 * time.Minute)
		}
	}

	// 列出初始资源
	resources := tracker.ListResources()
	logger.Info("初始资源数量", "count", len(resources))

	// 等待自动清理
	logger.Info("等待自动清理...")
	time.Sleep(2 * time.Second)

	// 列出清理后的资源
	resources = tracker.ListResources()
	logger.Info("清理后资源数量", "count", len(resources))
	for _, res := range resources {
		logger.Info("剩余资源", "id", res.ID(), "type", res.Type(), "last_used", res.LastUsedAt())
	}

	// 获取资源统计信息
	stats := tracker.GetStats()
	logger.Info("资源统计",
		"total_created", stats.TotalCreated,
		"total_closed", stats.TotalClosed,
		"current_active", stats.CurrentActive,
		"closure_errors", stats.ClosureErrors,
	)

	// 停止资源追踪器
	tracker.Stop()
}
