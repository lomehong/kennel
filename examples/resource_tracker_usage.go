package main

import (
	"fmt"
	"os"

	"github.com/lomehong/kennel/pkg/core"
	"github.com/lomehong/kennel/pkg/resource"
)

// 本示例展示如何在AppFramework中使用资源追踪器
func main() {
	// 创建应用程序实例
	app := core.NewApp("config.yaml")

	// 初始化应用程序
	if err := app.Init(); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}

	// 获取资源追踪器
	tracker := app.GetResourceTracker()
	if tracker == nil {
		fmt.Println("资源追踪器未初始化")
		os.Exit(1)
	}

	fmt.Println("=== 资源追踪器使用示例 ===")

	// 示例1: 直接使用资源追踪器
	fmt.Println("\n=== 示例1: 直接使用资源追踪器 ===")
	useTrackerDirectly(tracker)

	// 示例2: 使用应用程序的资源追踪方法
	fmt.Println("\n=== 示例2: 使用应用程序的资源追踪方法 ===")
	useAppTrackingMethods(app)

	// 示例3: 使用上下文关联的资源追踪器
	fmt.Println("\n=== 示例3: 使用上下文关联的资源追踪器 ===")
	useContextTracker(app)

	// 获取资源统计信息
	stats := tracker.GetStats()
	fmt.Printf("\n资源统计:\n")
	fmt.Printf("- 总创建资源数: %d\n", stats.TotalCreated)
	fmt.Printf("- 总关闭资源数: %d\n", stats.TotalClosed)
	fmt.Printf("- 当前活动资源数: %d\n", stats.CurrentActive)
	fmt.Printf("- 关闭错误数: %d\n", stats.ClosureErrors)

	// 停止应用程序
	app.Stop()
	fmt.Println("\n应用程序已停止")
}

// 直接使用资源追踪器
func useTrackerDirectly(tracker *resource.ResourceTracker) {
	// 创建临时文件
	file, err := os.CreateTemp("", "example-*.txt")
	if err != nil {
		fmt.Printf("创建临时文件失败: %v\n", err)
		return
	}
	fmt.Printf("创建临时文件: %s\n", file.Name())

	// 写入一些数据
	_, err = file.WriteString("Hello, Resource Tracker!")
	if err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
		file.Close()
		return
	}

	// 创建文件资源
	fileResource := resource.NewFileResource(file)

	// 追踪资源
	tracker.Track(fileResource)
	fmt.Printf("追踪文件资源: %s (ID: %s)\n", fileResource.Path(), fileResource.ID())

	// 使用资源
	_, err = fileResource.File().Seek(0, 0)
	if err != nil {
		fmt.Printf("文件定位失败: %v\n", err)
	}

	data := make([]byte, 100)
	n, err := fileResource.File().Read(data)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
	} else {
		fmt.Printf("读取文件内容: %s\n", data[:n])
	}

	// 释放资源
	err = tracker.Release(fileResource.ID())
	if err != nil {
		fmt.Printf("释放资源失败: %v\n", err)
	} else {
		fmt.Println("资源已释放")
	}

	// 尝试再次使用文件（应该失败，因为已关闭）
	_, err = file.Write([]byte("This should fail"))
	if err != nil {
		fmt.Printf("预期的错误: %v\n", err)
	}
}

// 使用应用程序的资源追踪方法
func useAppTrackingMethods(app *core.App) {
	// 创建临时文件
	file, err := os.CreateTemp("", "app-example-*.txt")
	if err != nil {
		fmt.Printf("创建临时文件失败: %v\n", err)
		return
	}
	fmt.Printf("创建临时文件: %s\n", file.Name())

	// 写入一些数据
	_, err = file.WriteString("Using App's tracking methods!")
	if err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
		file.Close()
		return
	}

	// 创建文件资源
	fileResource := resource.NewFileResource(file)

	// 使用应用程序追踪资源
	app.TrackResource(fileResource)
	fmt.Printf("通过App追踪文件资源: %s (ID: %s)\n", fileResource.Path(), fileResource.ID())

	// 使用资源
	_, err = fileResource.File().Seek(0, 0)
	if err != nil {
		fmt.Printf("文件定位失败: %v\n", err)
	}

	data := make([]byte, 100)
	n, err := fileResource.File().Read(data)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
	} else {
		fmt.Printf("读取文件内容: %s\n", data[:n])
	}

	// 释放资源
	err = app.ReleaseResource(fileResource.ID())
	if err != nil {
		fmt.Printf("释放资源失败: %v\n", err)
	} else {
		fmt.Println("资源已释放")
	}
}

// 使用上下文关联的资源追踪器
func useContextTracker(app *core.App) {
	// 获取上下文关联的资源追踪器
	ctxTracker := app.GetContextResourceTracker()
	if ctxTracker == nil {
		fmt.Println("上下文资源追踪器未初始化")
		return
	}

	// 创建临时文件
	file, err := os.CreateTemp("", "ctx-example-*.txt")
	if err != nil {
		fmt.Printf("创建临时文件失败: %v\n", err)
		return
	}
	fmt.Printf("创建临时文件: %s\n", file.Name())

	// 写入一些数据
	_, err = file.WriteString("Using context tracker!")
	if err != nil {
		fmt.Printf("写入文件失败: %v\n", err)
		file.Close()
		return
	}

	// 使用上下文追踪器追踪文件资源
	fileResource := ctxTracker.TrackFile(file)
	fmt.Printf("通过上下文追踪器追踪文件资源: %s (ID: %s)\n", fileResource.Path(), fileResource.ID())

	// 使用资源
	_, err = fileResource.File().Seek(0, 0)
	if err != nil {
		fmt.Printf("文件定位失败: %v\n", err)
	}

	data := make([]byte, 100)
	n, err := fileResource.File().Read(data)
	if err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
	} else {
		fmt.Printf("读取文件内容: %s\n", data[:n])
	}

	fmt.Println("资源将在应用程序停止时自动释放")

	// 列出当前资源
	resources := app.GetResourceTracker().ListResources()
	fmt.Printf("当前资源数量: %d\n", len(resources))
}
