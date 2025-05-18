package profiler

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// ProfileType 性能分析类型
type ProfileType string

// 预定义性能分析类型
const (
	ProfileTypeCPU     ProfileType = "cpu"     // CPU分析
	ProfileTypeHeap    ProfileType = "heap"    // 堆分析
	ProfileTypeBlock   ProfileType = "block"   // 阻塞分析
	ProfileTypeGoroutine ProfileType = "goroutine" // 协程分析
	ProfileTypeThreadcreate ProfileType = "threadcreate" // 线程创建分析
	ProfileTypeMutex   ProfileType = "mutex"   // 互斥锁分析
	ProfileTypeTrace   ProfileType = "trace"   // 执行追踪
	ProfileTypeAllocs  ProfileType = "allocs"  // 内存分配分析
)

// ProfileFormat 性能分析数据格式
type ProfileFormat string

// 预定义性能分析数据格式
const (
	ProfileFormatPprof ProfileFormat = "pprof"  // pprof格式
	ProfileFormatJSON  ProfileFormat = "json"   // JSON格式
	ProfileFormatText  ProfileFormat = "text"   // 文本格式
	ProfileFormatSVG   ProfileFormat = "svg"    // SVG格式
	ProfileFormatPDF   ProfileFormat = "pdf"    // PDF格式
	ProfileFormatHTML  ProfileFormat = "html"   // HTML格式
)

// ProfileOptions 性能分析选项
type ProfileOptions struct {
	Duration  time.Duration  // 分析持续时间
	Rate      int            // 采样率
	Debug     int            // 调试级别
	Format    ProfileFormat  // 输出格式
	OutputDir string         // 输出目录
	FileName  string         // 文件名
	Labels    map[string]string // 标签
}

// DefaultProfileOptions 默认性能分析选项
func DefaultProfileOptions() ProfileOptions {
	return ProfileOptions{
		Duration:  30 * time.Second,
		Rate:      100,
		Debug:     0,
		Format:    ProfileFormatPprof,
		OutputDir: "profiles",
		FileName:  "",
		Labels:    make(map[string]string),
	}
}

// ProfileResult 性能分析结果
type ProfileResult struct {
	Type      ProfileType    // 分析类型
	StartTime time.Time      // 开始时间
	EndTime   time.Time      // 结束时间
	Duration  time.Duration  // 持续时间
	FilePath  string         // 文件路径
	Size      int64          // 文件大小
	Format    ProfileFormat  // 文件格式
	Labels    map[string]string // 标签
	Error     error          // 错误信息
}

// Profiler 性能分析器接口
type Profiler interface {
	// Start 开始性能分析
	Start(ctx context.Context, profileType ProfileType, options ProfileOptions) error

	// Stop 停止性能分析
	Stop(profileType ProfileType) (*ProfileResult, error)

	// IsRunning 检查指定类型的性能分析是否正在运行
	IsRunning(profileType ProfileType) bool

	// GetRunningProfiles 获取正在运行的性能分析
	GetRunningProfiles() map[ProfileType]ProfileOptions

	// GetResults 获取性能分析结果
	GetResults() []*ProfileResult

	// GetResult 获取指定类型的最新性能分析结果
	GetResult(profileType ProfileType) *ProfileResult

	// WriteProfile 将性能分析数据写入指定的写入器
	WriteProfile(profileType ProfileType, format ProfileFormat, w io.Writer) error

	// ConvertProfile 转换性能分析数据格式
	ConvertProfile(inputPath string, outputPath string, format ProfileFormat) error

	// AnalyzeProfile 分析性能分析数据
	AnalyzeProfile(profileType ProfileType, filePath string) (map[string]interface{}, error)

	// Cleanup 清理性能分析数据
	Cleanup(olderThan time.Duration) error
}

// StandardProfiler 标准性能分析器
type StandardProfiler struct {
	options        map[ProfileType]ProfileOptions // 性能分析选项
	results        []*ProfileResult               // 性能分析结果
	cpuProfile     *os.File                       // CPU分析文件
	running        map[ProfileType]bool           // 正在运行的性能分析
	outputDir      string                         // 输出目录
	maxResults     int                            // 最大结果数量
	logger         hclog.Logger                   // 日志记录器
	mu             sync.RWMutex                   // 互斥锁
}

// NewStandardProfiler 创建标准性能分析器
func NewStandardProfiler(outputDir string, maxResults int, logger hclog.Logger) *StandardProfiler {
	// 如果未指定输出目录，使用默认目录
	if outputDir == "" {
		outputDir = "profiles"
	}

	// 如果未指定最大结果数量，使用默认值
	if maxResults <= 0 {
		maxResults = 100
	}

	// 如果未指定日志记录器，使用空日志记录器
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Error("创建性能分析输出目录失败", "dir", outputDir, "error", err)
	}

	return &StandardProfiler{
		options:    make(map[ProfileType]ProfileOptions),
		results:    make([]*ProfileResult, 0, maxResults),
		running:    make(map[ProfileType]bool),
		outputDir:  outputDir,
		maxResults: maxResults,
		logger:     logger.Named("profiler"),
	}
}

// Start 开始性能分析
func (p *StandardProfiler) Start(ctx context.Context, profileType ProfileType, options ProfileOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查是否已经在运行
	if p.running[profileType] {
		return fmt.Errorf("性能分析已在运行: %s", profileType)
	}

	// 设置选项
	p.options[profileType] = options

	// 创建输出目录
	if err := os.MkdirAll(options.OutputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 生成文件名
	if options.FileName == "" {
		options.FileName = fmt.Sprintf("%s-%s.%s", profileType, time.Now().Format("20060102-150405"), options.Format)
	}

	// 构建文件路径
	filePath := filepath.Join(options.OutputDir, options.FileName)

	// 根据分析类型启动相应的分析
	var err error
	switch profileType {
	case ProfileTypeCPU:
		err = p.startCPUProfile(filePath)
	case ProfileTypeHeap, ProfileTypeBlock, ProfileTypeGoroutine, ProfileTypeThreadcreate, ProfileTypeMutex, ProfileTypeAllocs:
		// 这些类型在Stop时收集，不需要在Start时做特殊处理
		runtime.SetBlockProfileRate(options.Rate)
		runtime.SetMutexProfileFraction(options.Rate)
	case ProfileTypeTrace:
		err = p.startTraceProfile(ctx, filePath, options.Duration)
	default:
		return fmt.Errorf("不支持的性能分析类型: %s", profileType)
	}

	if err != nil {
		return fmt.Errorf("启动性能分析失败: %v", err)
	}

	// 标记为正在运行
	p.running[profileType] = true

	// 如果设置了持续时间，启动定时器自动停止
	if options.Duration > 0 && profileType != ProfileTypeTrace {
		go func() {
			select {
			case <-time.After(options.Duration):
				p.Stop(profileType)
			case <-ctx.Done():
				p.Stop(profileType)
			}
		}()
	}

	p.logger.Info("开始性能分析", "type", profileType, "duration", options.Duration, "file", filePath)
	return nil
}

// startCPUProfile 启动CPU分析
func (p *StandardProfiler) startCPUProfile(filePath string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建CPU分析文件失败: %v", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("启动CPU分析失败: %v", err)
	}

	p.cpuProfile = f
	return nil
}

// startTraceProfile 启动执行追踪
func (p *StandardProfiler) startTraceProfile(ctx context.Context, filePath string, duration time.Duration) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建执行追踪文件失败: %v", err)
	}

	if err := trace.Start(f); err != nil {
		f.Close()
		return fmt.Errorf("启动执行追踪失败: %v", err)
	}

	// 在指定时间后自动停止追踪
	go func() {
		select {
		case <-time.After(duration):
			trace.Stop()
			f.Close()
		case <-ctx.Done():
			trace.Stop()
			f.Close()
		}
	}()

	return nil
}

// Stop 停止性能分析
func (p *StandardProfiler) Stop(profileType ProfileType) (*ProfileResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查是否正在运行
	if !p.running[profileType] {
		return nil, fmt.Errorf("性能分析未运行: %s", profileType)
	}

	// 获取选项
	options, ok := p.options[profileType]
	if !ok {
		return nil, fmt.Errorf("未找到性能分析选项: %s", profileType)
	}

	// 构建文件路径
	filePath := filepath.Join(options.OutputDir, options.FileName)

	// 根据分析类型停止相应的分析
	var err error
	switch profileType {
	case ProfileTypeCPU:
		pprof.StopCPUProfile()
		if p.cpuProfile != nil {
			p.cpuProfile.Close()
			p.cpuProfile = nil
		}
	case ProfileTypeHeap, ProfileTypeBlock, ProfileTypeGoroutine, ProfileTypeThreadcreate, ProfileTypeMutex, ProfileTypeAllocs:
		err = p.writeProfile(profileType, filePath)
	case ProfileTypeTrace:
		// 追踪在startTraceProfile中已经设置了自动停止
	default:
		return nil, fmt.Errorf("不支持的性能分析类型: %s", profileType)
	}

	if err != nil {
		return nil, fmt.Errorf("停止性能分析失败: %v", err)
	}

	// 标记为未运行
	p.running[profileType] = false

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	var fileSize int64
	if err == nil {
		fileSize = fileInfo.Size()
	}

	// 创建结果
	result := &ProfileResult{
		Type:      profileType,
		StartTime: time.Now().Add(-options.Duration),
		EndTime:   time.Now(),
		Duration:  options.Duration,
		FilePath:  filePath,
		Size:      fileSize,
		Format:    options.Format,
		Labels:    options.Labels,
	}

	// 添加到结果列表
	p.results = append(p.results, result)

	// 如果结果数量超过最大值，移除最旧的结果
	if len(p.results) > p.maxResults {
		p.results = p.results[1:]
	}

	p.logger.Info("停止性能分析", "type", profileType, "duration", result.Duration, "file", filePath, "size", fileSize)
	return result, nil
}

// writeProfile 写入性能分析数据
func (p *StandardProfiler) writeProfile(profileType ProfileType, filePath string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建性能分析文件失败: %v", err)
	}
	defer f.Close()

	switch profileType {
	case ProfileTypeHeap:
		return pprof.WriteHeapProfile(f)
	case ProfileTypeAllocs:
		return pprof.Lookup("allocs").WriteTo(f, 0)
	case ProfileTypeBlock:
		return pprof.Lookup("block").WriteTo(f, 0)
	case ProfileTypeGoroutine:
		return pprof.Lookup("goroutine").WriteTo(f, 0)
	case ProfileTypeThreadcreate:
		return pprof.Lookup("threadcreate").WriteTo(f, 0)
	case ProfileTypeMutex:
		return pprof.Lookup("mutex").WriteTo(f, 0)
	default:
		return fmt.Errorf("不支持的性能分析类型: %s", profileType)
	}
}
