package profiler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStandardProfiler(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "profiler-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建性能分析器
	p := NewStandardProfiler(tempDir, 10, logger)

	// 验证性能分析器
	assert.Equal(t, tempDir, p.outputDir)
	assert.Equal(t, 10, p.maxResults)
	assert.NotNil(t, p.logger)
	assert.NotNil(t, p.options)
	assert.NotNil(t, p.results)
	assert.NotNil(t, p.running)
}

func TestStandardProfiler_Start_Stop(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "profiler-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建性能分析器
	p := NewStandardProfiler(tempDir, 10, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := DefaultProfileOptions()
	options.Duration = 1 * time.Second
	options.OutputDir = tempDir

	// 测试 CPU 分析
	t.Run("CPU", func(t *testing.T) {
		// 启动 CPU 分析
		err := p.Start(ctx, ProfileTypeCPU, options)
		require.NoError(t, err)

		// 验证正在运行
		assert.True(t, p.IsRunning(ProfileTypeCPU))

		// 执行一些 CPU 密集型操作
		for i := 0; i < 1000000; i++ {
			_ = i * i
		}

		// 停止 CPU 分析
		result, err := p.Stop(ProfileTypeCPU)
		require.NoError(t, err)

		// 验证结果
		assert.Equal(t, ProfileTypeCPU, result.Type)
		assert.Equal(t, options.Duration, result.Duration)
		assert.Equal(t, options.Format, result.Format)
		assert.True(t, result.Size > 0)
		assert.FileExists(t, result.FilePath)

		// 验证未运行
		assert.False(t, p.IsRunning(ProfileTypeCPU))
	})

	// 测试堆内存分析
	t.Run("Heap", func(t *testing.T) {
		// 启动堆内存分析
		err := p.Start(ctx, ProfileTypeHeap, options)
		require.NoError(t, err)

		// 验证正在运行
		assert.True(t, p.IsRunning(ProfileTypeHeap))

		// 执行一些内存分配操作
		var data []int
		for i := 0; i < 1000000; i++ {
			data = append(data, i)
		}
		_ = data

		// 停止堆内存分析
		result, err := p.Stop(ProfileTypeHeap)
		require.NoError(t, err)

		// 验证结果
		assert.Equal(t, ProfileTypeHeap, result.Type)
		assert.Equal(t, options.Format, result.Format)
		assert.True(t, result.Size > 0)
		assert.FileExists(t, result.FilePath)

		// 验证未运行
		assert.False(t, p.IsRunning(ProfileTypeHeap))
	})

	// 测试阻塞分析
	t.Run("Block", func(t *testing.T) {
		// 启动阻塞分析
		err := p.Start(ctx, ProfileTypeBlock, options)
		require.NoError(t, err)

		// 验证正在运行
		assert.True(t, p.IsRunning(ProfileTypeBlock))

		// 执行一些可能导致阻塞的操作
		ch := make(chan int)
		go func() {
			time.Sleep(100 * time.Millisecond)
			ch <- 1
		}()
		<-ch

		// 停止阻塞分析
		result, err := p.Stop(ProfileTypeBlock)
		require.NoError(t, err)

		// 验证结果
		assert.Equal(t, ProfileTypeBlock, result.Type)
		assert.Equal(t, options.Format, result.Format)
		assert.FileExists(t, result.FilePath)

		// 验证未运行
		assert.False(t, p.IsRunning(ProfileTypeBlock))
	})
}

func TestStandardProfiler_GetResults(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "profiler-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建性能分析器
	p := NewStandardProfiler(tempDir, 10, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := DefaultProfileOptions()
	options.Duration = 1 * time.Second
	options.OutputDir = tempDir

	// 启动和停止多个性能分析
	profileTypes := []ProfileType{ProfileTypeCPU, ProfileTypeHeap, ProfileTypeBlock}
	for _, profileType := range profileTypes {
		err := p.Start(ctx, profileType, options)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		_, err = p.Stop(profileType)
		require.NoError(t, err)
	}

	// 获取结果
	results := p.GetResults()

	// 验证结果
	assert.Len(t, results, len(profileTypes))
	for i, result := range results {
		assert.Equal(t, profileTypes[i], result.Type)
		assert.Equal(t, options.Format, result.Format)
		assert.FileExists(t, result.FilePath)
	}

	// 获取指定类型的结果
	result := p.GetResult(ProfileTypeCPU)
	assert.NotNil(t, result)
	assert.Equal(t, ProfileTypeCPU, result.Type)
}

func TestStandardProfiler_GetRunningProfiles(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "profiler-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建性能分析器
	p := NewStandardProfiler(tempDir, 10, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := DefaultProfileOptions()
	options.Duration = 10 * time.Second
	options.OutputDir = tempDir

	// 启动多个性能分析
	profileTypes := []ProfileType{ProfileTypeCPU, ProfileTypeHeap}
	for _, profileType := range profileTypes {
		err := p.Start(ctx, profileType, options)
		require.NoError(t, err)
	}

	// 获取正在运行的性能分析
	profiles := p.GetRunningProfiles()

	// 验证结果
	assert.Len(t, profiles, len(profileTypes))
	for _, profileType := range profileTypes {
		assert.Contains(t, profiles, profileType)
		assert.Equal(t, options, profiles[profileType])
	}

	// 停止性能分析
	for _, profileType := range profileTypes {
		_, err := p.Stop(profileType)
		require.NoError(t, err)
	}

	// 再次获取正在运行的性能分析
	profiles = p.GetRunningProfiles()
	assert.Empty(t, profiles)
}

func TestStandardProfiler_Cleanup(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "profiler-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建性能分析器
	p := NewStandardProfiler(tempDir, 10, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := DefaultProfileOptions()
	options.Duration = 1 * time.Second
	options.OutputDir = tempDir

	// 启动和停止多个性能分析
	profileTypes := []ProfileType{ProfileTypeCPU, ProfileTypeHeap, ProfileTypeBlock}
	for _, profileType := range profileTypes {
		err := p.Start(ctx, profileType, options)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		_, err = p.Stop(profileType)
		require.NoError(t, err)
	}

	// 获取结果
	results := p.GetResults()
	assert.Len(t, results, len(profileTypes))

	// 修改结果的结束时间，使其看起来很旧
	for _, result := range results {
		result.EndTime = time.Now().Add(-2 * time.Hour)
	}

	// 清理 1 小时前的性能分析数据
	err = p.Cleanup(1 * time.Hour)
	require.NoError(t, err)

	// 验证结果已被清理
	results = p.GetResults()
	assert.Empty(t, results)

	// 验证文件已被删除
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestStandardProfiler_WriteProfile(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "profiler-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建性能分析器
	p := NewStandardProfiler(tempDir, 10, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := DefaultProfileOptions()
	options.Duration = 1 * time.Second
	options.OutputDir = tempDir

	// 启动和停止 CPU 分析
	err = p.Start(ctx, ProfileTypeCPU, options)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	_, err = p.Stop(ProfileTypeCPU)
	require.NoError(t, err)

	// 创建输出文件
	outputFile := filepath.Join(tempDir, "output.pprof")
	f, err := os.Create(outputFile)
	require.NoError(t, err)
	defer f.Close()

	// 写入性能分析数据
	err = p.WriteProfile(ProfileTypeCPU, ProfileFormatPprof, f)
	require.NoError(t, err)

	// 验证输出文件
	assert.FileExists(t, outputFile)
	fileInfo, err := os.Stat(outputFile)
	require.NoError(t, err)
	assert.True(t, fileInfo.Size() > 0)
}

func TestStandardProfiler_ConvertProfile(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "profiler-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建性能分析器
	p := NewStandardProfiler(tempDir, 10, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := DefaultProfileOptions()
	options.Duration = 1 * time.Second
	options.OutputDir = tempDir

	// 启动和停止 CPU 分析
	err = p.Start(ctx, ProfileTypeCPU, options)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	result, err := p.Stop(ProfileTypeCPU)
	require.NoError(t, err)

	// 转换为文本格式
	textFile := filepath.Join(tempDir, "cpu.txt")
	err = p.ConvertProfile(result.FilePath, textFile, ProfileFormatText)
	require.NoError(t, err)

	// 验证输出文件
	assert.FileExists(t, textFile)
	fileInfo, err := os.Stat(textFile)
	require.NoError(t, err)
	assert.True(t, fileInfo.Size() > 0)
}

func TestStandardProfiler_AnalyzeProfile(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "profiler-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建性能分析器
	p := NewStandardProfiler(tempDir, 10, nil)

	// 创建上下文
	ctx := context.Background()

	// 创建性能分析选项
	options := DefaultProfileOptions()
	options.Duration = 1 * time.Second
	options.OutputDir = tempDir

	// 启动和停止 CPU 分析
	err = p.Start(ctx, ProfileTypeCPU, options)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	result, err := p.Stop(ProfileTypeCPU)
	require.NoError(t, err)

	// 分析性能分析数据
	analysis, err := p.AnalyzeProfile(ProfileTypeCPU, result.FilePath)
	require.NoError(t, err)

	// 验证分析结果
	assert.NotNil(t, analysis)
	assert.Contains(t, analysis, "functions")
}
