package profiler_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/profiler"
	"github.com/spf13/cobra"
)

// 示例：创建性能分析器
func ExampleNewStandardProfiler() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "app",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建性能分析器
	p := profiler.NewStandardProfiler("profiles", 100, logger)

	// 使用性能分析器...
	fmt.Println("性能分析器已创建")
	// Output: 性能分析器已创建
}

// 示例：启动和停止CPU性能分析
func ExampleStandardProfiler_Start() {
	// 创建性能分析器
	p := profiler.NewStandardProfiler("profiles", 100, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := profiler.DefaultProfileOptions()
	options.Duration = 5 * time.Second

	// 启动CPU性能分析
	err := p.Start(ctx, profiler.ProfileTypeCPU, options)
	if err != nil {
		fmt.Printf("启动CPU性能分析失败: %v\n", err)
		return
	}

	// 执行一些CPU密集型操作
	for i := 0; i < 1000000; i++ {
		_ = fmt.Sprintf("计算第%d次", i)
	}

	// 停止CPU性能分析
	result, err := p.Stop(profiler.ProfileTypeCPU)
	if err != nil {
		fmt.Printf("停止CPU性能分析失败: %v\n", err)
		return
	}

	fmt.Printf("CPU性能分析已完成，文件: %s\n", result.FilePath)
	// Output: CPU性能分析已完成，文件: profiles/cpu-20060102-150405.pprof
}

// 示例：使用HTTP处理器
func ExampleNewHTTPHandler() {
	// 创建性能分析器
	p := profiler.NewStandardProfiler("profiles", 100, nil)

	// 创建HTTP处理器
	handler := profiler.NewHTTPHandler(p, "/debug/pprof")

	// 创建HTTP服务器
	mux := http.NewServeMux()

	// 注册处理器
	handler.RegisterHandlers(mux)

	// 启动HTTP服务器
	fmt.Println("HTTP处理器已注册")
	// Output: HTTP处理器已注册
}

// 示例：使用命令行处理器
func ExampleNewCommandHandler() {
	// 创建性能分析器
	p := profiler.NewStandardProfiler("profiles", 100, nil)

	// 创建命令行处理器
	handler := profiler.NewCommandHandler(p, nil)

	// 创建根命令
	rootCmd := &cobra.Command{
		Use:   "app",
		Short: "应用程序",
	}

	// 注册命令
	handler.RegisterCommands(rootCmd)

	fmt.Println("命令行处理器已注册")
	// Output: 命令行处理器已注册
}

// 示例：收集堆内存性能分析
func ExampleStandardProfiler_heapProfile() {
	// 创建性能分析器
	p := profiler.NewStandardProfiler("profiles", 100, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := profiler.DefaultProfileOptions()

	// 启动堆内存性能分析
	err := p.Start(ctx, profiler.ProfileTypeHeap, options)
	if err != nil {
		fmt.Printf("启动堆内存性能分析失败: %v\n", err)
		return
	}

	// 执行一些内存分配操作
	var data []string
	for i := 0; i < 10000; i++ {
		data = append(data, fmt.Sprintf("数据项%d", i))
	}

	// 停止堆内存性能分析
	result, err := p.Stop(profiler.ProfileTypeHeap)
	if err != nil {
		fmt.Printf("停止堆内存性能分析失败: %v\n", err)
		return
	}

	fmt.Printf("堆内存性能分析已完成，文件: %s\n", result.FilePath)
	// Output: 堆内存性能分析已完成，文件: profiles/heap-20060102-150405.pprof
}

// 示例：收集阻塞分析
func ExampleStandardProfiler_blockProfile() {
	// 创建性能分析器
	p := profiler.NewStandardProfiler("profiles", 100, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := profiler.DefaultProfileOptions()
	options.Rate = 1 // 设置采样率为1，记录所有阻塞事件

	// 启动阻塞分析
	err := p.Start(ctx, profiler.ProfileTypeBlock, options)
	if err != nil {
		fmt.Printf("启动阻塞分析失败: %v\n", err)
		return
	}

	// 执行一些可能导致阻塞的操作
	ch := make(chan int)
	go func() {
		time.Sleep(100 * time.Millisecond)
		ch <- 1
	}()
	<-ch

	// 停止阻塞分析
	result, err := p.Stop(profiler.ProfileTypeBlock)
	if err != nil {
		fmt.Printf("停止阻塞分析失败: %v\n", err)
		return
	}

	fmt.Printf("阻塞分析已完成，文件: %s\n", result.FilePath)
	// Output: 阻塞分析已完成，文件: profiles/block-20060102-150405.pprof
}

// 示例：收集执行追踪
func ExampleStandardProfiler_traceProfile() {
	// 创建性能分析器
	p := profiler.NewStandardProfiler("profiles", 100, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := profiler.DefaultProfileOptions()
	options.Duration = 1 * time.Second

	// 启动执行追踪
	err := p.Start(ctx, profiler.ProfileTypeTrace, options)
	if err != nil {
		fmt.Printf("启动执行追踪失败: %v\n", err)
		return
	}

	// 执行一些操作
	for i := 0; i < 100; i++ {
		go func(n int) {
			time.Sleep(time.Duration(n) * time.Millisecond)
		}(i % 10)
	}

	// 等待追踪完成
	time.Sleep(2 * time.Second)

	// 获取结果
	result := p.GetResult(profiler.ProfileTypeTrace)
	if result == nil {
		fmt.Println("未找到执行追踪结果")
		return
	}

	fmt.Printf("执行追踪已完成，文件: %s\n", result.FilePath)
	// Output: 执行追踪已完成，文件: profiles/trace-20060102-150405.pprof
}

// 示例：分析性能分析数据
func ExampleStandardProfiler_AnalyzeProfile() {
	// 创建性能分析器
	p := profiler.NewStandardProfiler("profiles", 100, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := profiler.DefaultProfileOptions()
	options.Duration = 1 * time.Second

	// 启动CPU性能分析
	err := p.Start(ctx, profiler.ProfileTypeCPU, options)
	if err != nil {
		fmt.Printf("启动CPU性能分析失败: %v\n", err)
		return
	}

	// 执行一些CPU密集型操作
	for i := 0; i < 1000000; i++ {
		_ = fmt.Sprintf("计算第%d次", i)
	}

	// 停止CPU性能分析
	result, err := p.Stop(profiler.ProfileTypeCPU)
	if err != nil {
		fmt.Printf("停止CPU性能分析失败: %v\n", err)
		return
	}

	// 分析性能分析数据
	analysis, err := p.AnalyzeProfile(profiler.ProfileTypeCPU, result.FilePath)
	if err != nil {
		fmt.Printf("分析性能分析数据失败: %v\n", err)
		return
	}

	// 打印分析结果
	if functions, ok := analysis["functions"].([]interface{}); ok && len(functions) > 0 {
		fmt.Println("热点函数:")
		for i, f := range functions {
			if i >= 3 {
				break
			}
			function := f.(map[string]interface{})
			fmt.Printf("  %s: %.2f%%\n", function["name"], function["flat"])
		}
	}

	fmt.Println("性能分析数据分析完成")
	// Output: 性能分析数据分析完成
}

// 示例：清理性能分析数据
func ExampleStandardProfiler_Cleanup() {
	// 创建性能分析器
	p := profiler.NewStandardProfiler("profiles", 100, nil)

	// 清理7天前的性能分析数据
	err := p.Cleanup(7 * 24 * time.Hour)
	if err != nil {
		fmt.Printf("清理性能分析数据失败: %v\n", err)
		return
	}

	fmt.Println("性能分析数据清理完成")
	// Output: 性能分析数据清理完成
}
