package health

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCPUUsageCheck(t *testing.T) {
	// 创建CPU使用率检查
	check := NewCPUUsageCheck(100.0, 30*time.Second)

	// 验证基本属性
	assert.Equal(t, CPUUsageCheckName, check.Name())
	assert.Equal(t, SystemCheckType, check.Type())
	assert.Equal(t, "system", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.False(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Contains(t, []HealthStatus{HealthStatusHealthy, HealthStatusUnhealthy}, result.Status)
	assert.Contains(t, result.Message, "CPU使用率")
	assert.Contains(t, result.Details, "cpu_usage")
	assert.Contains(t, result.Details, "threshold")
	assert.Contains(t, result.Details, "cpu_count")
}

func TestMemoryUsageCheck(t *testing.T) {
	// 创建内存使用率检查
	check := NewMemoryUsageCheck(100.0, 30*time.Second)

	// 验证基本属性
	assert.Equal(t, MemoryUsageCheckName, check.Name())
	assert.Equal(t, SystemCheckType, check.Type())
	assert.Equal(t, "system", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.False(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Contains(t, []HealthStatus{HealthStatusHealthy, HealthStatusUnhealthy}, result.Status)
	assert.Contains(t, result.Message, "内存使用率")
	assert.Contains(t, result.Details, "memory_usage")
	assert.Contains(t, result.Details, "threshold")
	assert.Contains(t, result.Details, "total")
	assert.Contains(t, result.Details, "used")
	assert.Contains(t, result.Details, "free")
}

func TestDiskUsageCheck(t *testing.T) {
	// 创建磁盘使用率检查
	check := NewDiskUsageCheck(".", 100.0, 30*time.Second)

	// 验证基本属性
	assert.Equal(t, DiskUsageCheckName, check.Name())
	assert.Equal(t, SystemCheckType, check.Type())
	assert.Equal(t, "system", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.False(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Contains(t, []HealthStatus{HealthStatusHealthy, HealthStatusUnhealthy}, result.Status)
	assert.Contains(t, result.Message, "磁盘使用率")
	assert.Contains(t, result.Details, "disk_usage")
	assert.Contains(t, result.Details, "threshold")
	assert.Contains(t, result.Details, "path")
	assert.Contains(t, result.Details, "total")
	assert.Contains(t, result.Details, "used")
	assert.Contains(t, result.Details, "free")
}

func TestDiskSpaceCheck(t *testing.T) {
	// 创建磁盘空间检查
	check := NewDiskSpaceCheck(".", 1024, 30*time.Second)

	// 验证基本属性
	assert.Equal(t, DiskSpaceCheckName, check.Name())
	assert.Equal(t, SystemCheckType, check.Type())
	assert.Equal(t, "system", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.False(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Contains(t, []HealthStatus{HealthStatusHealthy, HealthStatusUnhealthy}, result.Status)
	assert.Contains(t, result.Message, "磁盘空闲空间")
	assert.Contains(t, result.Details, "free_space")
	assert.Contains(t, result.Details, "min_free_space")
	assert.Contains(t, result.Details, "path")
	assert.Contains(t, result.Details, "total")
	assert.Contains(t, result.Details, "used")
}

func TestGoroutineCountCheck(t *testing.T) {
	// 创建协程数量检查
	check := NewGoroutineCountCheck(1000, 30*time.Second)

	// 验证基本属性
	assert.Equal(t, GoroutineCountCheckName, check.Name())
	assert.Equal(t, SystemCheckType, check.Type())
	assert.Equal(t, "system", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.False(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Contains(t, []HealthStatus{HealthStatusHealthy, HealthStatusUnhealthy}, result.Status)
	assert.Contains(t, result.Message, "协程数量")
	assert.Contains(t, result.Details, "goroutine_count")
	assert.Contains(t, result.Details, "threshold")
}

func TestProcessMemoryCheck(t *testing.T) {
	// 创建进程内存检查
	check := NewProcessMemoryCheck(100.0, 30*time.Second)

	// 验证基本属性
	assert.Equal(t, ProcessMemoryCheckName, check.Name())
	assert.Equal(t, SystemCheckType, check.Type())
	assert.Equal(t, "system", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.False(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Contains(t, []HealthStatus{HealthStatusHealthy, HealthStatusUnhealthy}, result.Status)
	assert.Contains(t, result.Message, "进程内存使用率")
	assert.Contains(t, result.Details, "memory_percent")
	assert.Contains(t, result.Details, "threshold")
	assert.Contains(t, result.Details, "rss")
	assert.Contains(t, result.Details, "vms")
	assert.Contains(t, result.Details, "system_total")
}

func TestTempDirCleanupCheck(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "health-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建临时文件
	oldFile := filepath.Join(tempDir, "old.txt")
	err = os.WriteFile(oldFile, []byte("old"), 0644)
	assert.NoError(t, err)

	// 修改文件时间
	oldTime := time.Now().Add(-2 * time.Hour)
	err = os.Chtimes(oldFile, oldTime, oldTime)
	assert.NoError(t, err)

	// 创建新文件
	newFile := filepath.Join(tempDir, "new.txt")
	err = os.WriteFile(newFile, []byte("new"), 0644)
	assert.NoError(t, err)

	// 创建临时目录清理检查
	check := NewTempDirCleanupCheck(tempDir, 1*time.Hour, 30*time.Second)

	// 验证基本属性
	assert.Equal(t, "temp_dir_cleanup", check.Name())
	assert.Equal(t, SystemCheckType, check.Type())
	assert.Equal(t, "system", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.True(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "过期文件")
	assert.Contains(t, result.Details, "temp_dir")
	assert.Contains(t, result.Details, "expired_count")
	assert.Contains(t, result.Details, "max_age")
	assert.Contains(t, result.Details, "expired_files")

	// 执行恢复
	err = check.Recover(context.Background())
	assert.NoError(t, err)

	// 验证旧文件已删除
	_, err = os.Stat(oldFile)
	assert.True(t, os.IsNotExist(err))

	// 验证新文件仍存在
	_, err = os.Stat(newFile)
	assert.NoError(t, err)

	// 再次执行检查
	result = check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusHealthy, result.Status)
	assert.Contains(t, result.Message, "没有过期文件")
}
