package profiler

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
)

// CommandHandler 性能分析命令行处理器
type CommandHandler struct {
	profiler Profiler
	logger   hclog.Logger
}

// NewCommandHandler 创建性能分析命令行处理器
func NewCommandHandler(profiler Profiler, logger hclog.Logger) *CommandHandler {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}
	return &CommandHandler{
		profiler: profiler,
		logger:   logger.Named("profiler-cmd"),
	}
}

// RegisterCommands 注册命令
func (h *CommandHandler) RegisterCommands(rootCmd *cobra.Command) {
	// 性能分析命令
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "性能分析工具",
		Long:  "用于收集和分析应用程序性能数据的工具",
	}
	rootCmd.AddCommand(profileCmd)

	// 启动性能分析命令
	startCmd := &cobra.Command{
		Use:   "start [type]",
		Short: "启动性能分析",
		Long:  "启动指定类型的性能分析",
		Args:  cobra.ExactArgs(1),
		Run:   h.startCommandHandler,
	}
	startCmd.Flags().DurationP("duration", "d", 30*time.Second, "性能分析持续时间")
	startCmd.Flags().IntP("rate", "r", 100, "采样率")
	startCmd.Flags().StringP("format", "f", "pprof", "输出格式 (pprof, json, text, svg, pdf, html)")
	startCmd.Flags().StringP("output-dir", "o", "profiles", "输出目录")
	startCmd.Flags().StringP("file-name", "n", "", "输出文件名")
	profileCmd.AddCommand(startCmd)

	// 停止性能分析命令
	stopCmd := &cobra.Command{
		Use:   "stop [type]",
		Short: "停止性能分析",
		Long:  "停止指定类型的性能分析",
		Args:  cobra.ExactArgs(1),
		Run:   h.stopCommandHandler,
	}
	profileCmd.AddCommand(stopCmd)

	// 列出性能分析结果命令
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "列出性能分析结果",
		Long:  "列出所有性能分析结果",
		Run:   h.listCommandHandler,
	}
	profileCmd.AddCommand(listCmd)

	// 列出正在运行的性能分析命令
	runningCmd := &cobra.Command{
		Use:   "running",
		Short: "列出正在运行的性能分析",
		Long:  "列出所有正在运行的性能分析",
		Run:   h.runningCommandHandler,
	}
	profileCmd.AddCommand(runningCmd)

	// 分析性能分析数据命令
	analyzeCmd := &cobra.Command{
		Use:   "analyze [type]",
		Short: "分析性能分析数据",
		Long:  "分析指定类型的性能分析数据",
		Args:  cobra.ExactArgs(1),
		Run:   h.analyzeCommandHandler,
	}
	analyzeCmd.Flags().StringP("file", "f", "", "性能分析文件路径")
	profileCmd.AddCommand(analyzeCmd)

	// 清理性能分析数据命令
	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "清理性能分析数据",
		Long:  "清理指定时间前的性能分析数据",
		Run:   h.cleanupCommandHandler,
	}
	cleanupCmd.Flags().DurationP("older-than", "o", 24*time.Hour, "清理指定时间前的数据")
	profileCmd.AddCommand(cleanupCmd)

	// 为每种性能分析类型添加快捷命令
	profileCmd.AddCommand(&cobra.Command{
		Use:   "cpu",
		Short: "收集CPU性能分析",
		Long:  "收集CPU使用情况的性能分析",
		Run: func(cmd *cobra.Command, args []string) {
			h.profileCommandHandler(cmd, append([]string{string(ProfileTypeCPU)}, args...))
		},
	})
	profileCmd.AddCommand(&cobra.Command{
		Use:   "heap",
		Short: "收集堆内存性能分析",
		Long:  "收集堆内存使用情况的性能分析",
		Run: func(cmd *cobra.Command, args []string) {
			h.profileCommandHandler(cmd, append([]string{string(ProfileTypeHeap)}, args...))
		},
	})
	profileCmd.AddCommand(&cobra.Command{
		Use:   "block",
		Short: "收集阻塞性能分析",
		Long:  "收集goroutine阻塞情况的性能分析",
		Run: func(cmd *cobra.Command, args []string) {
			h.profileCommandHandler(cmd, append([]string{string(ProfileTypeBlock)}, args...))
		},
	})
	profileCmd.AddCommand(&cobra.Command{
		Use:   "goroutine",
		Short: "收集协程性能分析",
		Long:  "收集goroutine信息的性能分析",
		Run: func(cmd *cobra.Command, args []string) {
			h.profileCommandHandler(cmd, append([]string{string(ProfileTypeGoroutine)}, args...))
		},
	})
	profileCmd.AddCommand(&cobra.Command{
		Use:   "threadcreate",
		Short: "收集线程创建性能分析",
		Long:  "收集线程创建情况的性能分析",
		Run: func(cmd *cobra.Command, args []string) {
			h.profileCommandHandler(cmd, append([]string{string(ProfileTypeThreadcreate)}, args...))
		},
	})
	profileCmd.AddCommand(&cobra.Command{
		Use:   "mutex",
		Short: "收集互斥锁性能分析",
		Long:  "收集互斥锁争用情况的性能分析",
		Run: func(cmd *cobra.Command, args []string) {
			h.profileCommandHandler(cmd, append([]string{string(ProfileTypeMutex)}, args...))
		},
	})
	profileCmd.AddCommand(&cobra.Command{
		Use:   "trace",
		Short: "收集执行追踪",
		Long:  "收集程序执行追踪",
		Run: func(cmd *cobra.Command, args []string) {
			h.profileCommandHandler(cmd, append([]string{string(ProfileTypeTrace)}, args...))
		},
	})
	profileCmd.AddCommand(&cobra.Command{
		Use:   "allocs",
		Short: "收集内存分配性能分析",
		Long:  "收集内存分配情况的性能分析",
		Run: func(cmd *cobra.Command, args []string) {
			h.profileCommandHandler(cmd, append([]string{string(ProfileTypeAllocs)}, args...))
		},
	})
}

// startCommandHandler 处理启动性能分析命令
func (h *CommandHandler) startCommandHandler(cmd *cobra.Command, args []string) {
	// 获取性能分析类型
	profileType := ProfileType(args[0])

	// 获取选项
	duration, _ := cmd.Flags().GetDuration("duration")
	rate, _ := cmd.Flags().GetInt("rate")
	format, _ := cmd.Flags().GetString("format")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	fileName, _ := cmd.Flags().GetString("file-name")

	// 创建选项
	options := ProfileOptions{
		Duration:  duration,
		Rate:      rate,
		Format:    ProfileFormat(format),
		OutputDir: outputDir,
		FileName:  fileName,
		Labels:    make(map[string]string),
	}

	// 启动性能分析
	if err := h.profiler.Start(context.Background(), profileType, options); err != nil {
		fmt.Fprintf(os.Stderr, "启动性能分析失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("已启动%s性能分析，持续时间: %v\n", profileType, duration)
}

// stopCommandHandler 处理停止性能分析命令
func (h *CommandHandler) stopCommandHandler(cmd *cobra.Command, args []string) {
	// 获取性能分析类型
	profileType := ProfileType(args[0])

	// 停止性能分析
	result, err := h.profiler.Stop(profileType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "停止性能分析失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("已停止%s性能分析\n", profileType)
	fmt.Printf("文件: %s\n", result.FilePath)
	fmt.Printf("大小: %d字节\n", result.Size)
	fmt.Printf("持续时间: %v\n", result.Duration)
}

// listCommandHandler 处理列出性能分析结果命令
func (h *CommandHandler) listCommandHandler(cmd *cobra.Command, args []string) {
	// 获取性能分析结果
	results := h.profiler.GetResults()

	if len(results) == 0 {
		fmt.Println("没有性能分析结果")
		return
	}

	fmt.Println("性能分析结果:")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("%-12s %-20s %-20s %-15s %-10s\n", "类型", "开始时间", "结束时间", "持续时间", "大小(字节)")
	fmt.Println("------------------------------------------------------------")

	for _, result := range results {
		fmt.Printf("%-12s %-20s %-20s %-15v %-10d\n",
			result.Type,
			result.StartTime.Format("2006-01-02 15:04:05"),
			result.EndTime.Format("2006-01-02 15:04:05"),
			result.Duration,
			result.Size)
	}
	fmt.Println("------------------------------------------------------------")
}

// runningCommandHandler 处理列出正在运行的性能分析命令
func (h *CommandHandler) runningCommandHandler(cmd *cobra.Command, args []string) {
	// 获取正在运行的性能分析
	profiles := h.profiler.GetRunningProfiles()

	if len(profiles) == 0 {
		fmt.Println("没有正在运行的性能分析")
		return
	}

	fmt.Println("正在运行的性能分析:")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("%-12s %-15s %-10s %-15s\n", "类型", "持续时间", "采样率", "输出格式")
	fmt.Println("------------------------------------------------------------")

	for profileType, options := range profiles {
		fmt.Printf("%-12s %-15v %-10d %-15s\n",
			profileType,
			options.Duration,
			options.Rate,
			options.Format)
	}
	fmt.Println("------------------------------------------------------------")
}

// analyzeCommandHandler 处理分析性能分析数据命令
func (h *CommandHandler) analyzeCommandHandler(cmd *cobra.Command, args []string) {
	// 获取性能分析类型
	profileType := ProfileType(args[0])

	// 获取文件路径
	filePath, _ := cmd.Flags().GetString("file")

	// 如果未指定文件路径，使用最新的性能分析结果
	if filePath == "" {
		result := h.profiler.GetResult(profileType)
		if result == nil {
			fmt.Fprintf(os.Stderr, "未找到%s性能分析结果\n", profileType)
			os.Exit(1)
		}
		filePath = result.FilePath
	}

	// 分析性能分析数据
	analysis, err := h.profiler.AnalyzeProfile(profileType, filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "分析性能分析数据失败: %v\n", err)
		os.Exit(1)
	}

	// 打印分析结果
	fmt.Printf("分析结果 (%s):\n", profileType)
	fmt.Println("------------------------------------------------------------")

	// 打印函数列表
	if functions, ok := analysis["functions"].([]interface{}); ok && len(functions) > 0 {
		fmt.Println("热点函数:")
		fmt.Printf("%-5s %-10s %-10s %-10s %-40s\n", "排名", "累计", "自身", "调用次数", "函数名")
		fmt.Println("------------------------------------------------------------")

		for i, f := range functions {
			if i >= 10 {
				break
			}

			function, ok := f.(map[string]interface{})
			if !ok {
				continue
			}

			name := function["name"].(string)
			if len(name) > 40 {
				name = name[:37] + "..."
			}

			cumulative := function["cumulative"].(float64)
			flat := function["flat"].(float64)
			calls := int64(0)
			if c, ok := function["calls"].(float64); ok {
				calls = int64(c)
			}

			fmt.Printf("%-5d %-10.2f %-10.2f %-10d %-40s\n",
				i+1, cumulative, flat, calls, name)
		}
		fmt.Println("------------------------------------------------------------")
	}

	// 打印内存分配信息
	if profileType == ProfileTypeHeap || profileType == ProfileTypeAllocs {
		if memStats, ok := analysis["memStats"].(map[string]interface{}); ok {
			fmt.Println("内存统计:")
			fmt.Printf("总分配: %v\n", formatBytes(int64(memStats["alloc"].(float64))))
			fmt.Printf("系统内存: %v\n", formatBytes(int64(memStats["sys"].(float64))))
			fmt.Printf("堆对象: %d\n", int64(memStats["objects"].(float64)))
			fmt.Println("------------------------------------------------------------")
		}
	}
}

// cleanupCommandHandler 处理清理性能分析数据命令
func (h *CommandHandler) cleanupCommandHandler(cmd *cobra.Command, args []string) {
	// 获取清理时间
	olderThan, _ := cmd.Flags().GetDuration("older-than")

	// 清理性能分析数据
	if err := h.profiler.Cleanup(olderThan); err != nil {
		fmt.Fprintf(os.Stderr, "清理性能分析数据失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("已清理%v前的性能分析数据\n", olderThan)
}

// profileCommandHandler 处理性能分析命令
func (h *CommandHandler) profileCommandHandler(cmd *cobra.Command, args []string) {
	// 获取性能分析类型
	profileType := ProfileType(args[0])

	// 解析参数
	duration := 30 * time.Second
	if len(args) > 1 {
		if d, err := strconv.Atoi(args[1]); err == nil {
			duration = time.Duration(d) * time.Second
		}
	}

	// 创建选项
	options := DefaultProfileOptions()
	options.Duration = duration

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), duration+5*time.Second)
	defer cancel()

	// 启动性能分析
	fmt.Printf("正在收集%s性能分析，持续时间: %v\n", profileType, duration)
	if err := h.profiler.Start(ctx, profileType, options); err != nil {
		fmt.Fprintf(os.Stderr, "启动性能分析失败: %v\n", err)
		os.Exit(1)
	}

	// 等待性能分析完成
	time.Sleep(duration)

	// 停止性能分析
	result, err := h.profiler.Stop(profileType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "停止性能分析失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("性能分析完成\n")
	fmt.Printf("文件: %s\n", result.FilePath)
	fmt.Printf("大小: %d字节\n", result.Size)
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
