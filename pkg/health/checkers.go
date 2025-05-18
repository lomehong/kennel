package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// 常用健康检查器类型
const (
	CheckerTypeSystem   = "system"
	CheckerTypeResource = "resource"
	CheckerTypeNetwork  = "network"
	CheckerTypeService  = "service"
	CheckerTypeCustom   = "custom"
)

// NewMemoryChecker 创建内存健康检查器
func NewMemoryChecker(thresholdPercent float64) Checker {
	return NewSimpleChecker(
		"memory",
		"检查系统内存使用情况",
		CheckerTypeResource,
		func(ctx context.Context) CheckResult {
			v, err := mem.VirtualMemory()
			if err != nil {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: "无法获取内存信息",
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}

			usedPercent := v.UsedPercent
			details := map[string]interface{}{
				"total":        v.Total,
				"used":         v.Used,
				"free":         v.Free,
				"used_percent": usedPercent,
				"threshold":    thresholdPercent,
			}

			if usedPercent >= thresholdPercent {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("内存使用率 %.2f%% 超过阈值 %.2f%%", usedPercent, thresholdPercent),
					Details: details,
				}
			}

			if usedPercent >= thresholdPercent*0.8 {
				return CheckResult{
					Status:  StatusDegraded,
					Message: fmt.Sprintf("内存使用率 %.2f%% 接近阈值 %.2f%%", usedPercent, thresholdPercent),
					Details: details,
				}
			}

			return CheckResult{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("内存使用率 %.2f%% 正常", usedPercent),
				Details: details,
			}
		},
	)
}

// NewCPUChecker 创建CPU健康检查器
func NewCPUChecker(thresholdPercent float64) Checker {
	return NewSimpleChecker(
		"cpu",
		"检查系统CPU使用情况",
		CheckerTypeResource,
		func(ctx context.Context) CheckResult {
			percent, err := cpu.Percent(time.Second, false)
			if err != nil {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: "无法获取CPU信息",
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
					},
				}
			}

			if len(percent) == 0 {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: "无法获取CPU使用率",
					Details: map[string]interface{}{},
				}
			}

			cpuPercent := percent[0]
			details := map[string]interface{}{
				"used_percent": cpuPercent,
				"threshold":    thresholdPercent,
			}

			if cpuPercent >= thresholdPercent {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("CPU使用率 %.2f%% 超过阈值 %.2f%%", cpuPercent, thresholdPercent),
					Details: details,
				}
			}

			if cpuPercent >= thresholdPercent*0.8 {
				return CheckResult{
					Status:  StatusDegraded,
					Message: fmt.Sprintf("CPU使用率 %.2f%% 接近阈值 %.2f%%", cpuPercent, thresholdPercent),
					Details: details,
				}
			}

			return CheckResult{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("CPU使用率 %.2f%% 正常", cpuPercent),
				Details: details,
			}
		},
	)
}

// NewDiskChecker 创建磁盘健康检查器
func NewDiskChecker(path string, thresholdPercent float64) Checker {
	return NewSimpleChecker(
		fmt.Sprintf("disk_%s", path),
		fmt.Sprintf("检查磁盘 %s 使用情况", path),
		CheckerTypeResource,
		func(ctx context.Context) CheckResult {
			usage, err := disk.Usage(path)
			if err != nil {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("无法获取磁盘 %s 信息", path),
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
						"path":  path,
					},
				}
			}

			usedPercent := usage.UsedPercent
			details := map[string]interface{}{
				"path":         path,
				"total":        usage.Total,
				"used":         usage.Used,
				"free":         usage.Free,
				"used_percent": usedPercent,
				"threshold":    thresholdPercent,
			}

			if usedPercent >= thresholdPercent {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("磁盘 %s 使用率 %.2f%% 超过阈值 %.2f%%", path, usedPercent, thresholdPercent),
					Details: details,
				}
			}

			if usedPercent >= thresholdPercent*0.8 {
				return CheckResult{
					Status:  StatusDegraded,
					Message: fmt.Sprintf("磁盘 %s 使用率 %.2f%% 接近阈值 %.2f%%", path, usedPercent, thresholdPercent),
					Details: details,
				}
			}

			return CheckResult{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("磁盘 %s 使用率 %.2f%% 正常", path, usedPercent),
				Details: details,
			}
		},
	)
}

// NewGoroutineChecker 创建Goroutine健康检查器
func NewGoroutineChecker(thresholdCount int) Checker {
	return NewSimpleChecker(
		"goroutine",
		"检查Goroutine数量",
		CheckerTypeSystem,
		func(ctx context.Context) CheckResult {
			count := runtime.NumGoroutine()
			details := map[string]interface{}{
				"count":     count,
				"threshold": thresholdCount,
			}

			if count >= thresholdCount {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("Goroutine数量 %d 超过阈值 %d", count, thresholdCount),
					Details: details,
				}
			}

			if count >= thresholdCount*8/10 {
				return CheckResult{
					Status:  StatusDegraded,
					Message: fmt.Sprintf("Goroutine数量 %d 接近阈值 %d", count, thresholdCount),
					Details: details,
				}
			}

			return CheckResult{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("Goroutine数量 %d 正常", count),
				Details: details,
			}
		},
	)
}

// NewProcessChecker 创建进程健康检查器
func NewProcessChecker(pid int) Checker {
	if pid <= 0 {
		pid = os.Getpid()
	}

	return NewSimpleChecker(
		fmt.Sprintf("process_%d", pid),
		fmt.Sprintf("检查进程 %d 状态", pid),
		CheckerTypeSystem,
		func(ctx context.Context) CheckResult {
			p, err := process.NewProcess(int32(pid))
			if err != nil {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("无法获取进程 %d 信息", pid),
					Error:   err,
					Details: map[string]interface{}{
						"error": err.Error(),
						"pid":   pid,
					},
				}
			}

			running, err := p.IsRunning()
			if err != nil || !running {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("进程 %d 未运行", pid),
					Error:   err,
					Details: map[string]interface{}{
						"pid":     pid,
						"running": running,
					},
				}
			}

			cpuPercent, err := p.CPUPercent()
			memInfo, memErr := p.MemoryInfo()
			memPercent, memPercentErr := p.MemoryPercent()

			details := map[string]interface{}{
				"pid":     pid,
				"running": running,
			}

			if err == nil {
				details["cpu_percent"] = cpuPercent
			}
			if memErr == nil && memInfo != nil {
				details["memory_rss"] = memInfo.RSS
				details["memory_vms"] = memInfo.VMS
			}
			if memPercentErr == nil {
				details["memory_percent"] = memPercent
			}

			return CheckResult{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("进程 %d 运行正常", pid),
				Details: details,
			}
		},
	)
}

// NewTCPChecker 创建TCP连接健康检查器
func NewTCPChecker(address string, timeout time.Duration) Checker {
	return NewSimpleChecker(
		fmt.Sprintf("tcp_%s", address),
		fmt.Sprintf("检查TCP连接 %s", address),
		CheckerTypeNetwork,
		func(ctx context.Context) CheckResult {
			conn, err := net.DialTimeout("tcp", address, timeout)
			if err != nil {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("无法连接到 %s", address),
					Error:   err,
					Details: map[string]interface{}{
						"error":   err.Error(),
						"address": address,
						"timeout": timeout.String(),
					},
				}
			}
			defer conn.Close()

			return CheckResult{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("成功连接到 %s", address),
				Details: map[string]interface{}{
					"address": address,
					"timeout": timeout.String(),
				},
			}
		},
	)
}

// NewHTTPChecker 创建HTTP健康检查器
func NewHTTPChecker(url string, timeout time.Duration, expectedStatus int) Checker {
	return NewSimpleChecker(
		fmt.Sprintf("http_%s", url),
		fmt.Sprintf("检查HTTP服务 %s", url),
		CheckerTypeNetwork,
		func(ctx context.Context) CheckResult {
			client := &http.Client{
				Timeout: timeout,
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("创建HTTP请求失败: %s", url),
					Error:   err,
					Details: map[string]interface{}{
						"error":   err.Error(),
						"url":     url,
						"timeout": timeout.String(),
					},
				}
			}

			resp, err := client.Do(req)
			if err != nil {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("HTTP请求失败: %s", url),
					Error:   err,
					Details: map[string]interface{}{
						"error":   err.Error(),
						"url":     url,
						"timeout": timeout.String(),
					},
				}
			}
			defer resp.Body.Close()

			details := map[string]interface{}{
				"url":            url,
				"timeout":        timeout.String(),
				"status":         resp.StatusCode,
				"expected_status": expectedStatus,
			}

			if expectedStatus > 0 && resp.StatusCode != expectedStatus {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("HTTP状态码不匹配: %d (期望 %d)", resp.StatusCode, expectedStatus),
					Details: details,
				}
			}

			if resp.StatusCode >= 500 {
				return CheckResult{
					Status:  StatusUnhealthy,
					Message: fmt.Sprintf("HTTP服务器错误: %d", resp.StatusCode),
					Details: details,
				}
			}

			if resp.StatusCode >= 400 {
				return CheckResult{
					Status:  StatusDegraded,
					Message: fmt.Sprintf("HTTP客户端错误: %d", resp.StatusCode),
					Details: details,
				}
			}

			return CheckResult{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("HTTP服务正常: %d", resp.StatusCode),
				Details: details,
			}
		},
	)
}
