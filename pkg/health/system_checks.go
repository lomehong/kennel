package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// SystemCheckType 系统检查类型
const SystemCheckType = "system"

// 系统健康检查名称
const (
	CPUUsageCheckName     = "cpu_usage"
	MemoryUsageCheckName  = "memory_usage"
	DiskUsageCheckName    = "disk_usage"
	DiskSpaceCheckName    = "disk_space"
	GoroutineCountCheckName = "goroutine_count"
	ProcessMemoryCheckName = "process_memory"
)

// NewCPUUsageCheck 创建CPU使用率检查
func NewCPUUsageCheck(threshold float64, interval time.Duration) HealthCheck {
	return &BaseHealthCheck{
		name:             CPUUsageCheckName,
		checkType:        SystemCheckType,
		component:        "system",
		timeout:          5 * time.Second,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 获取CPU使用率
			percentages, err := cpu.Percent(time.Second, false)
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("获取CPU使用率失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}

			// 计算平均使用率
			var totalPercent float64
			for _, percent := range percentages {
				totalPercent += percent
			}
			avgPercent := totalPercent / float64(len(percentages))

			// 检查是否超过阈值
			if avgPercent > threshold {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("CPU使用率过高: %.2f%% > %.2f%%", avgPercent, threshold),
					Details: map[string]interface{}{
						"cpu_usage":  avgPercent,
						"threshold":  threshold,
						"cpu_count":  runtime.NumCPU(),
						"percentages": percentages,
					},
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("CPU使用率正常: %.2f%% <= %.2f%%", avgPercent, threshold),
				Details: map[string]interface{}{
					"cpu_usage":  avgPercent,
					"threshold":  threshold,
					"cpu_count":  runtime.NumCPU(),
					"percentages": percentages,
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewMemoryUsageCheck 创建内存使用率检查
func NewMemoryUsageCheck(threshold float64, interval time.Duration) HealthCheck {
	return &BaseHealthCheck{
		name:             MemoryUsageCheckName,
		checkType:        SystemCheckType,
		component:        "system",
		timeout:          5 * time.Second,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 获取内存使用情况
			memInfo, err := mem.VirtualMemory()
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("获取内存使用情况失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}

			// 检查是否超过阈值
			if memInfo.UsedPercent > threshold {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("内存使用率过高: %.2f%% > %.2f%%", memInfo.UsedPercent, threshold),
					Details: map[string]interface{}{
						"memory_usage": memInfo.UsedPercent,
						"threshold":    threshold,
						"total":        memInfo.Total,
						"used":         memInfo.Used,
						"free":         memInfo.Free,
					},
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("内存使用率正常: %.2f%% <= %.2f%%", memInfo.UsedPercent, threshold),
				Details: map[string]interface{}{
					"memory_usage": memInfo.UsedPercent,
					"threshold":    threshold,
					"total":        memInfo.Total,
					"used":         memInfo.Used,
					"free":         memInfo.Free,
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewDiskUsageCheck 创建磁盘使用率检查
func NewDiskUsageCheck(path string, threshold float64, interval time.Duration) HealthCheck {
	return &BaseHealthCheck{
		name:             DiskUsageCheckName,
		checkType:        SystemCheckType,
		component:        "system",
		timeout:          5 * time.Second,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 获取磁盘使用情况
			diskInfo, err := disk.Usage(path)
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("获取磁盘使用情况失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
						"path":  path,
					},
				}
			}

			// 检查是否超过阈值
			if diskInfo.UsedPercent > threshold {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("磁盘使用率过高: %.2f%% > %.2f%%", diskInfo.UsedPercent, threshold),
					Details: map[string]interface{}{
						"disk_usage": diskInfo.UsedPercent,
						"threshold":  threshold,
						"path":       path,
						"total":      diskInfo.Total,
						"used":       diskInfo.Used,
						"free":       diskInfo.Free,
					},
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("磁盘使用率正常: %.2f%% <= %.2f%%", diskInfo.UsedPercent, threshold),
				Details: map[string]interface{}{
					"disk_usage": diskInfo.UsedPercent,
					"threshold":  threshold,
					"path":       path,
					"total":      diskInfo.Total,
					"used":       diskInfo.Used,
					"free":       diskInfo.Free,
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewDiskSpaceCheck 创建磁盘空间检查
func NewDiskSpaceCheck(path string, minFreeSpace int64, interval time.Duration) HealthCheck {
	return &BaseHealthCheck{
		name:             DiskSpaceCheckName,
		checkType:        SystemCheckType,
		component:        "system",
		timeout:          5 * time.Second,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 获取磁盘使用情况
			diskInfo, err := disk.Usage(path)
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("获取磁盘使用情况失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
						"path":  path,
					},
				}
			}

			// 检查是否低于最小空闲空间
			if diskInfo.Free < uint64(minFreeSpace) {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("磁盘空闲空间不足: %d < %d", diskInfo.Free, minFreeSpace),
					Details: map[string]interface{}{
						"free_space":    diskInfo.Free,
						"min_free_space": minFreeSpace,
						"path":          path,
						"total":         diskInfo.Total,
						"used":          diskInfo.Used,
					},
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("磁盘空闲空间充足: %d >= %d", diskInfo.Free, minFreeSpace),
				Details: map[string]interface{}{
					"free_space":    diskInfo.Free,
					"min_free_space": minFreeSpace,
					"path":          path,
					"total":         diskInfo.Total,
					"used":          diskInfo.Used,
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewGoroutineCountCheck 创建协程数量检查
func NewGoroutineCountCheck(threshold int, interval time.Duration) HealthCheck {
	return &BaseHealthCheck{
		name:             GoroutineCountCheckName,
		checkType:        SystemCheckType,
		component:        "system",
		timeout:          5 * time.Second,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 获取协程数量
			count := runtime.NumGoroutine()

			// 检查是否超过阈值
			if count > threshold {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("协程数量过多: %d > %d", count, threshold),
					Details: map[string]interface{}{
						"goroutine_count": count,
						"threshold":       threshold,
					},
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("协程数量正常: %d <= %d", count, threshold),
				Details: map[string]interface{}{
					"goroutine_count": count,
					"threshold":       threshold,
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewProcessMemoryCheck 创建进程内存检查
func NewProcessMemoryCheck(threshold float64, interval time.Duration) HealthCheck {
	return &BaseHealthCheck{
		name:             ProcessMemoryCheckName,
		checkType:        SystemCheckType,
		component:        "system",
		timeout:          5 * time.Second,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 获取当前进程
			proc, err := process.NewProcess(int32(os.Getpid()))
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("获取进程信息失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}

			// 获取内存使用情况
			memInfo, err := proc.MemoryInfo()
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("获取进程内存信息失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}

			// 获取系统内存
			sysMemInfo, err := mem.VirtualMemory()
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("获取系统内存信息失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}

			// 计算进程内存使用率
			memPercent := float64(memInfo.RSS) / float64(sysMemInfo.Total) * 100

			// 检查是否超过阈值
			if memPercent > threshold {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("进程内存使用率过高: %.2f%% > %.2f%%", memPercent, threshold),
					Details: map[string]interface{}{
						"memory_percent": memPercent,
						"threshold":      threshold,
						"rss":            memInfo.RSS,
						"vms":            memInfo.VMS,
						"system_total":   sysMemInfo.Total,
					},
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("进程内存使用率正常: %.2f%% <= %.2f%%", memPercent, threshold),
				Details: map[string]interface{}{
					"memory_percent": memPercent,
					"threshold":      threshold,
					"rss":            memInfo.RSS,
					"vms":            memInfo.VMS,
					"system_total":   sysMemInfo.Total,
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewTempDirCleanupCheck 创建临时目录清理检查
func NewTempDirCleanupCheck(tempDir string, maxAge time.Duration, interval time.Duration) HealthCheck {
	return &BaseHealthCheck{
		name:             "temp_dir_cleanup",
		checkType:        SystemCheckType,
		component:        "system",
		timeout:          30 * time.Second,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      true,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 检查目录是否存在
			_, err := os.Stat(tempDir)
			if os.IsNotExist(err) {
				return &HealthCheckResult{
					Status:  HealthStatusHealthy,
					Message: fmt.Sprintf("临时目录不存在: %s", tempDir),
					Details: map[string]interface{}{
						"temp_dir": tempDir,
					},
				}
			}

			// 获取目录中的文件
			files, err := os.ReadDir(tempDir)
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("读取临时目录失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error":    err.Error(),
						"temp_dir": tempDir,
					},
				}
			}

			// 检查是否有过期文件
			now := time.Now()
			var expiredFiles []string
			for _, file := range files {
				if file.IsDir() {
					continue
				}

				info, err := file.Info()
				if err != nil {
					continue
				}

				if now.Sub(info.ModTime()) > maxAge {
					expiredFiles = append(expiredFiles, file.Name())
				}
			}

			if len(expiredFiles) > 0 {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("临时目录中有 %d 个过期文件", len(expiredFiles)),
					Details: map[string]interface{}{
						"temp_dir":      tempDir,
						"expired_count": len(expiredFiles),
						"max_age":       maxAge.String(),
						"expired_files": expiredFiles,
					},
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "临时目录中没有过期文件",
				Details: map[string]interface{}{
					"temp_dir": tempDir,
					"max_age":  maxAge.String(),
					"file_count": len(files),
				},
			}
		},
		recoverFunc: func(ctx context.Context) error {
			// 检查目录是否存在
			_, err := os.Stat(tempDir)
			if os.IsNotExist(err) {
				return nil
			}

			// 获取目录中的文件
			files, err := os.ReadDir(tempDir)
			if err != nil {
				return fmt.Errorf("读取临时目录失败: %w", err)
			}

			// 删除过期文件
			now := time.Now()
			deletedCount := 0
			for _, file := range files {
				if file.IsDir() {
					continue
				}

				info, err := file.Info()
				if err != nil {
					continue
				}

				if now.Sub(info.ModTime()) > maxAge {
					filePath := filepath.Join(tempDir, file.Name())
					if err := os.Remove(filePath); err != nil {
						return fmt.Errorf("删除过期文件失败: %w", err)
					}
					deletedCount++
				}
			}

			return nil
		},
	}
}
