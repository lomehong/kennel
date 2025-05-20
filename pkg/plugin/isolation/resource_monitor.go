package isolation

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shirou/gopsutil/v3/process"
)

// ResourceMonitor 资源监控器
// 监控插件沙箱的资源使用情况
type ResourceMonitor struct {
	// 沙箱映射
	sandboxes map[string]PluginSandbox
	
	// 进程映射
	processes map[string]*process.Process
	
	// 资源使用情况
	usage map[string]ResourceUsage
	
	// 互斥锁
	mu sync.RWMutex
	
	// 日志记录器
	logger hclog.Logger
	
	// 上下文
	ctx context.Context
	
	// 取消函数
	cancel context.CancelFunc
	
	// 监控间隔
	monitorInterval time.Duration
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	// CPU使用率
	CPUPercent float64
	
	// 内存使用量
	MemoryUsage uint64
	
	// 内存使用率
	MemoryPercent float32
	
	// 磁盘读取字节数
	DiskReadBytes uint64
	
	// 磁盘写入字节数
	DiskWriteBytes uint64
	
	// 网络接收字节数
	NetRecvBytes uint64
	
	// 网络发送字节数
	NetSentBytes uint64
	
	// 线程数
	NumThreads int32
	
	// 文件描述符数
	NumFDs int32
	
	// 上次更新时间
	LastUpdate time.Time
}

// NewResourceMonitor 创建一个新的资源监控器
func NewResourceMonitor(logger hclog.Logger) *ResourceMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	monitor := &ResourceMonitor{
		sandboxes:       make(map[string]PluginSandbox),
		processes:       make(map[string]*process.Process),
		usage:           make(map[string]ResourceUsage),
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		monitorInterval: 5 * time.Second,
	}
	
	// 启动监控
	go monitor.monitorResources()
	
	return monitor
}

// RegisterSandbox 注册沙箱
func (m *ResourceMonitor) RegisterSandbox(sandbox PluginSandbox) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	id := sandbox.GetID()
	m.sandboxes[id] = sandbox
	
	// 尝试查找关联的进程
	m.findProcess(id)
	
	m.logger.Debug("沙箱已注册", "id", id)
}

// UnregisterSandbox 注销沙箱
func (m *ResourceMonitor) UnregisterSandbox(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.sandboxes, id)
	delete(m.processes, id)
	delete(m.usage, id)
	
	m.logger.Debug("沙箱已注销", "id", id)
}

// GetResourceUsage 获取资源使用情况
func (m *ResourceMonitor) GetResourceUsage(id string) (ResourceUsage, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	usage, exists := m.usage[id]
	return usage, exists
}

// Stop 停止监控
func (m *ResourceMonitor) Stop() {
	m.cancel()
}

// monitorResources 监控资源
func (m *ResourceMonitor) monitorResources() {
	ticker := time.NewTicker(m.monitorInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateResourceUsage()
		}
	}
}

// updateResourceUsage 更新资源使用情况
func (m *ResourceMonitor) updateResourceUsage() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 更新所有进程的资源使用情况
	for id, proc := range m.processes {
		// 检查进程是否存在
		if !m.processExists(proc) {
			m.logger.Debug("进程不存在，尝试重新查找", "id", id)
			if !m.findProcess(id) {
				continue
			}
			proc = m.processes[id]
		}
		
		// 获取CPU使用率
		cpuPercent, err := proc.CPUPercent()
		if err != nil {
			m.logger.Debug("获取CPU使用率失败", "id", id, "error", err)
			cpuPercent = 0
		}
		
		// 获取内存使用情况
		memInfo, err := proc.MemoryInfo()
		if err != nil {
			m.logger.Debug("获取内存使用情况失败", "id", id, "error", err)
			memInfo = nil
		}
		
		// 获取内存使用率
		memPercent, err := proc.MemoryPercent()
		if err != nil {
			m.logger.Debug("获取内存使用率失败", "id", id, "error", err)
			memPercent = 0
		}
		
		// 获取IO计数器
		ioCounters, err := proc.IOCounters()
		if err != nil {
			m.logger.Debug("获取IO计数器失败", "id", id, "error", err)
			ioCounters = nil
		}
		
		// 获取线程数
		numThreads, err := proc.NumThreads()
		if err != nil {
			m.logger.Debug("获取线程数失败", "id", id, "error", err)
			numThreads = 0
		}
		
		// 获取文件描述符数
		numFDs, err := proc.NumFDs()
		if err != nil {
			m.logger.Debug("获取文件描述符数失败", "id", id, "error", err)
			numFDs = 0
		}
		
		// 更新资源使用情况
		usage := ResourceUsage{
			CPUPercent:    cpuPercent,
			NumThreads:    numThreads,
			NumFDs:        numFDs,
			LastUpdate:    time.Now(),
		}
		
		if memInfo != nil {
			usage.MemoryUsage = memInfo.RSS
		}
		
		usage.MemoryPercent = memPercent
		
		if ioCounters != nil {
			usage.DiskReadBytes = ioCounters.ReadBytes
			usage.DiskWriteBytes = ioCounters.WriteBytes
		}
		
		// 存储资源使用情况
		m.usage[id] = usage
		
		// 检查资源限制
		m.checkResourceLimits(id, usage)
	}
}

// findProcess 查找进程
func (m *ResourceMonitor) findProcess(id string) bool {
	// 获取所有进程
	processes, err := process.Processes()
	if err != nil {
		m.logger.Error("获取进程列表失败", "error", err)
		return false
	}
	
	// 查找匹配的进程
	for _, proc := range processes {
		name, err := proc.Name()
		if err != nil {
			continue
		}
		
		// 检查进程名称是否匹配
		if name == id || name == id+".exe" {
			m.processes[id] = proc
			m.logger.Debug("找到进程", "id", id, "pid", proc.Pid)
			return true
		}
		
		// 检查命令行是否包含ID
		cmdline, err := proc.Cmdline()
		if err != nil {
			continue
		}
		
		if cmdline != "" && (cmdline == id || cmdline == id+".exe" || cmdline == "./"+id || cmdline == "./"+id+".exe") {
			m.processes[id] = proc
			m.logger.Debug("找到进程", "id", id, "pid", proc.Pid, "cmdline", cmdline)
			return true
		}
	}
	
	return false
}

// processExists 检查进程是否存在
func (m *ResourceMonitor) processExists(proc *process.Process) bool {
	// 检查进程是否存在
	exists, err := proc.IsRunning()
	if err != nil {
		return false
	}
	return exists
}

// checkResourceLimits 检查资源限制
func (m *ResourceMonitor) checkResourceLimits(id string, usage ResourceUsage) {
	// 获取沙箱
	sandbox, exists := m.sandboxes[id]
	if !exists {
		return
	}
	
	// 获取隔离配置
	config := sandbox.GetConfig()
	
	// 检查CPU限制
	if cpuLimit, ok := config.Resources["cpu"]; ok && usage.CPUPercent > float64(cpuLimit) {
		m.logger.Warn("CPU使用率超过限制", "id", id, "usage", usage.CPUPercent, "limit", cpuLimit)
		// 在实际实现中，可以采取限制措施
	}
	
	// 检查内存限制
	if memLimit, ok := config.Resources["memory"]; ok && usage.MemoryUsage > uint64(memLimit) {
		m.logger.Warn("内存使用量超过限制", "id", id, "usage", usage.MemoryUsage, "limit", memLimit)
		// 在实际实现中，可以采取限制措施
	}
}

// GetSystemResourceUsage 获取系统资源使用情况
func (m *ResourceMonitor) GetSystemResourceUsage() map[string]interface{} {
	// 获取系统内存信息
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// 获取CPU数量
	numCPU := runtime.NumCPU()
	
	// 获取goroutine数量
	numGoroutine := runtime.NumGoroutine()
	
	return map[string]interface{}{
		"num_cpu":        numCPU,
		"num_goroutine":  numGoroutine,
		"alloc":          memStats.Alloc,
		"total_alloc":    memStats.TotalAlloc,
		"sys":            memStats.Sys,
		"heap_alloc":     memStats.HeapAlloc,
		"heap_sys":       memStats.HeapSys,
		"heap_idle":      memStats.HeapIdle,
		"heap_inuse":     memStats.HeapInuse,
		"heap_released":  memStats.HeapReleased,
		"heap_objects":   memStats.HeapObjects,
		"stack_inuse":    memStats.StackInuse,
		"stack_sys":      memStats.StackSys,
		"mspan_inuse":    memStats.MSpanInuse,
		"mspan_sys":      memStats.MSpanSys,
		"mcache_inuse":   memStats.MCacheInuse,
		"mcache_sys":     memStats.MCacheSys,
		"gc_next":        memStats.NextGC,
		"gc_last":        memStats.LastGC,
		"gc_num":         memStats.NumGC,
		"gc_cpu_fraction": memStats.GCCPUFraction,
	}
}
