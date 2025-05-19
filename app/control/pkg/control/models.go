package control

import (
	"sync"
	"time"
)

// ProcessInfo 包含进程信息
type ProcessInfo struct {
	PID       int     `json:"pid"`
	Name      string  `json:"name"`
	CPU       float64 `json:"cpu"`
	Memory    float64 `json:"memory"`
	StartTime string  `json:"start_time"`
	User      string  `json:"user"`
}

// ProcessCache 管理进程信息缓存
type ProcessCache struct {
	cachedProcesses []ProcessInfo // 缓存的进程信息
	cacheTime       time.Time     // 缓存时间
	cacheMutex      sync.RWMutex  // 用于保护缓存的互斥锁
}

// NewProcessCache 创建一个新的进程缓存
func NewProcessCache() *ProcessCache {
	return &ProcessCache{
		cachedProcesses: nil,
		cacheTime:       time.Time{}, // 零值，表示缓存无效
	}
}

// GetCachedProcesses 获取缓存的进程信息
func (c *ProcessCache) GetCachedProcesses(cacheExpiration time.Duration) ([]ProcessInfo, bool) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	cacheValid := c.cachedProcesses != nil && time.Since(c.cacheTime) < cacheExpiration
	if !cacheValid {
		return nil, false
	}

	// 返回一个副本，避免外部修改
	processesCopy := make([]ProcessInfo, len(c.cachedProcesses))
	copy(processesCopy, c.cachedProcesses)

	return processesCopy, true
}

// SetCachedProcesses 设置缓存的进程信息
func (c *ProcessCache) SetCachedProcesses(processes []ProcessInfo) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.cachedProcesses = processes
	c.cacheTime = time.Now()
}

// ProcessesToMap 将进程信息转换为map切片
func ProcessesToMap(processes []ProcessInfo) []map[string]interface{} {
	// 转换进程为map切片
	processesMap := make([]map[string]interface{}, len(processes))
	for i, process := range processes {
		processMap := make(map[string]interface{})
		processMap["pid"] = process.PID
		processMap["name"] = process.Name
		processMap["cpu"] = process.CPU
		processMap["memory"] = process.Memory
		processMap["start_time"] = process.StartTime
		processMap["user"] = process.User
		processesMap[i] = processMap
	}

	return processesMap
}

// CommandResult 命令执行结果
type CommandResult struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error"`
	Duration int64  `json:"duration_ms"` // 执行时间（毫秒）
}

// CommandResultToMap 将命令执行结果转换为map
func CommandResultToMap(result *CommandResult) map[string]interface{} {
	return map[string]interface{}{
		"command":     result.Command,
		"exit_code":   result.ExitCode,
		"output":      result.Output,
		"error":       result.Error,
		"duration_ms": result.Duration,
	}
}

// SoftwareInstallResult 软件安装结果
type SoftwareInstallResult struct {
	Package  string `json:"package"`
	Success  bool   `json:"success"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error"`
	Duration int64  `json:"duration_ms"` // 安装时间（毫秒）
}

// SoftwareInstallResultToMap 将软件安装结果转换为map
func SoftwareInstallResultToMap(result *SoftwareInstallResult) map[string]interface{} {
	return map[string]interface{}{
		"package":     result.Package,
		"success":     result.Success,
		"exit_code":   result.ExitCode,
		"output":      result.Output,
		"error":       result.Error,
		"duration_ms": result.Duration,
	}
}
