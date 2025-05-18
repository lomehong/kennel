package resource

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shirou/gopsutil/v3/process"
)

// ResourceManagerOption 资源管理器选项
type ResourceManagerOption func(*ResourceManager)

// WithLogger 设置日志记录器
func WithLogger(logger hclog.Logger) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.logger = logger
	}
}

// WithHistoryLimit 设置历史记录限制
func WithHistoryLimit(limit int) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.historyLimit = limit
	}
}

// WithUpdateInterval 设置更新间隔
func WithUpdateInterval(interval time.Duration) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.updateInterval = interval
	}
}

// WithDiskPaths 设置磁盘路径
func WithDiskPaths(paths []string) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.diskPaths = paths
	}
}

// WithNetworkInterfaces 设置网络接口
func WithNetworkInterfaces(ifaces []string) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.networkIfaces = ifaces
	}
}

// WithProcessID 设置进程ID
func WithProcessID(pid int32) ResourceManagerOption {
	return func(rm *ResourceManager) {
		rm.processID = pid
	}
}

// WithContext 设置上下文
func WithContext(ctx context.Context) ResourceManagerOption {
	return func(rm *ResourceManager) {
		if rm.cancel != nil {
			rm.cancel()
		}
		rm.ctx, rm.cancel = context.WithCancel(ctx)
	}
}

// ResourceManager 资源管理器
type ResourceManager struct {
	tracker        *ResourceUsageTracker // 资源使用跟踪器
	limiter        *ResourceLimiter      // 资源限制器
	logger         hclog.Logger          // 日志记录器
	historyLimit   int                   // 历史记录限制
	updateInterval time.Duration         // 更新间隔
	diskPaths      []string              // 磁盘路径
	networkIfaces  []string              // 网络接口
	processID      int32                 // 进程ID
	ctx            context.Context       // 上下文
	cancel         context.CancelFunc    // 取消函数
	mu             sync.RWMutex          // 互斥锁
}

// NewResourceManager 创建资源管理器
func NewResourceManager(options ...ResourceManagerOption) *ResourceManager {
	ctx, cancel := context.WithCancel(context.Background())

	rm := &ResourceManager{
		logger:         hclog.NewNullLogger(),
		historyLimit:   100,
		updateInterval: 5 * time.Second,
		diskPaths:      []string{"/"},
		networkIfaces:  []string{},
		processID:      int32(os.Getpid()),
		ctx:            ctx,
		cancel:         cancel,
	}

	// 应用选项
	for _, option := range options {
		option(rm)
	}

	// 创建资源使用跟踪器
	rm.tracker = NewResourceUsageTracker(rm.processID, rm.historyLimit)
	rm.tracker.SetDiskPaths(rm.diskPaths)
	rm.tracker.SetNetworkInterfaces(rm.networkIfaces)

	// 创建资源限制器
	rm.limiter = NewResourceLimiter(rm.tracker, rm.logger)

	return rm
}

// Start 启动资源管理器
func (rm *ResourceManager) Start() {
	// 启动资源限制器
	rm.limiter.Start()

	// 启动资源使用跟踪
	go func() {
		ticker := time.NewTicker(rm.updateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 更新资源使用情况
				if err := rm.tracker.Update(); err != nil {
					rm.logger.Error("更新资源使用情况失败", "error", err)
				}
			case <-rm.ctx.Done():
				return
			}
		}
	}()

	rm.logger.Info("资源管理器已启动",
		"update_interval", rm.updateInterval,
		"history_limit", rm.historyLimit,
		"process_id", rm.processID,
	)
}

// Stop 停止资源管理器
func (rm *ResourceManager) Stop() {
	// 停止资源限制器
	rm.limiter.Stop()

	// 取消上下文
	rm.cancel()

	rm.logger.Info("资源管理器已停止")
}

// GetResourceUsage 获取资源使用情况
func (rm *ResourceManager) GetResourceUsage() *ResourceUsage {
	return rm.limiter.GetResourceUsage()
}

// GetResourceUsageHistory 获取资源使用历史
func (rm *ResourceManager) GetResourceUsageHistory() []*ResourceUsage {
	return rm.limiter.GetResourceUsageHistory()
}

// GetResourceStats 获取资源统计信息
func (rm *ResourceManager) GetResourceStats() map[string]interface{} {
	return rm.limiter.GetResourceStats()
}

// GetResourceAlerts 获取资源告警
func (rm *ResourceManager) GetResourceAlerts() map[ResourceType][]string {
	return rm.limiter.GetResourceAlerts()
}

// GetResourceLimits 获取资源限制
func (rm *ResourceManager) GetResourceLimits() map[ResourceType][]ResourceLimit {
	return rm.limiter.GetLimits()
}

// LimitCPU 限制CPU使用
func (rm *ResourceManager) LimitCPU(percent float64, action ResourceLimitAction) {
	rm.limiter.LimitCPU(percent, action)
}

// LimitMemory 限制内存使用
func (rm *ResourceManager) LimitMemory(bytes uint64, action ResourceLimitAction) {
	rm.limiter.LimitMemory(bytes, action)
}

// LimitDisk 限制磁盘使用
func (rm *ResourceManager) LimitDisk(bytes uint64, action ResourceLimitAction) {
	rm.limiter.LimitDisk(bytes, action)
}

// LimitNetwork 限制网络使用
func (rm *ResourceManager) LimitNetwork(bytesPerSecond uint64, action ResourceLimitAction) {
	rm.limiter.LimitNetwork(bytesPerSecond, action)
}

// RemoveLimit 移除资源限制
func (rm *ResourceManager) RemoveLimit(resourceType ResourceType, limitType ResourceLimitType) {
	rm.limiter.RemoveLimit(resourceType, limitType)
}

// SetProcessPriority 设置进程优先级
func (rm *ResourceManager) SetProcessPriority(priority int) error {
	return rm.limiter.SetProcessPriority(priority)
}

// SetGOMAXPROCS 设置GOMAXPROCS
func (rm *ResourceManager) SetGOMAXPROCS(n int) {
	rm.limiter.SetGOMAXPROCS(n)
}

// RegisterAlertHandler 注册告警处理器
func (rm *ResourceManager) RegisterAlertHandler(handler ResourceAlertHandler) {
	rm.limiter.RegisterAlertHandler(handler)
}

// RegisterActionHandler 注册动作处理器
func (rm *ResourceManager) RegisterActionHandler(action ResourceLimitAction, handler ResourceActionHandler) {
	rm.limiter.RegisterActionHandler(action, handler)
}

// GetProcessInfo 获取进程信息
func (rm *ResourceManager) GetProcessInfo() (map[string]interface{}, error) {
	// 获取进程ID
	pid := rm.processID
	if pid <= 0 {
		pid = int32(os.Getpid())
	}

	// 获取进程
	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("获取进程失败: %w", err)
	}

	// 获取进程信息
	info := make(map[string]interface{})

	// 获取进程名称
	name, err := proc.Name()
	if err == nil {
		info["name"] = name
	}

	// 获取进程状态
	status, err := proc.Status()
	if err == nil {
		info["status"] = status
	}

	// 获取进程创建时间
	createTime, err := proc.CreateTime()
	if err == nil {
		info["create_time"] = createTime
	}

	// 获取进程CPU时间
	cpuTimes, err := proc.Times()
	if err == nil {
		info["cpu_user"] = cpuTimes.User
		info["cpu_system"] = cpuTimes.System
		info["cpu_total"] = cpuTimes.Total()
	}

	// 获取进程内存信息
	memInfo, err := proc.MemoryInfo()
	if err == nil {
		info["memory_rss"] = memInfo.RSS
		info["memory_vms"] = memInfo.VMS
	}

	// 获取进程线程数
	numThreads, err := proc.NumThreads()
	if err == nil {
		info["num_threads"] = numThreads
	}

	// 获取进程文件描述符数
	numFDs, err := proc.NumFDs()
	if err == nil {
		info["num_fds"] = numFDs
	}

	// 获取进程IO计数器
	ioCounters, err := proc.IOCounters()
	if err == nil {
		info["io_read_count"] = ioCounters.ReadCount
		info["io_write_count"] = ioCounters.WriteCount
		info["io_read_bytes"] = ioCounters.ReadBytes
		info["io_write_bytes"] = ioCounters.WriteBytes
	}

	// 获取进程连接数
	connections, err := proc.Connections()
	if err == nil {
		info["num_connections"] = len(connections)
	}

	// 获取进程命令行
	cmdline, err := proc.Cmdline()
	if err == nil {
		info["cmdline"] = cmdline
	}

	// 获取进程环境变量
	environ, err := proc.Environ()
	if err == nil {
		info["environ"] = environ
	}

	// 获取进程父进程ID
	ppid, err := proc.Ppid()
	if err == nil {
		info["ppid"] = ppid
	}

	// 获取进程子进程
	children, err := proc.Children()
	if err == nil {
		childrenIDs := make([]int32, len(children))
		for i, child := range children {
			childrenIDs[i] = child.Pid
		}
		info["children"] = childrenIDs
	}

	return info, nil
}

// GetSystemInfo 获取系统信息
func (rm *ResourceManager) GetSystemInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// 获取CPU信息
	info["cpu_count"] = runtime.NumCPU()
	info["goroutines"] = runtime.NumGoroutine()
	info["gomaxprocs"] = runtime.GOMAXPROCS(0)

	// 获取内存信息
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	info["memory_alloc"] = memStats.Alloc
	info["memory_total_alloc"] = memStats.TotalAlloc
	info["memory_sys"] = memStats.Sys
	info["memory_heap_alloc"] = memStats.HeapAlloc
	info["memory_heap_sys"] = memStats.HeapSys
	info["memory_heap_idle"] = memStats.HeapIdle
	info["memory_heap_inuse"] = memStats.HeapInuse
	info["memory_stack_inuse"] = memStats.StackInuse
	info["memory_stack_sys"] = memStats.StackSys
	info["memory_gc_count"] = memStats.NumGC
	info["memory_gc_pause_total"] = memStats.PauseTotalNs

	// 获取系统信息
	info["os"] = runtime.GOOS
	info["arch"] = runtime.GOARCH
	info["go_version"] = runtime.Version()

	return info
}

// OptimizeMemoryUsage 优化内存使用
func (rm *ResourceManager) OptimizeMemoryUsage() {
	// 强制进行垃圾回收
	runtime.GC()
	// 释放未使用的内存
	debug.FreeOSMemory()
	rm.logger.Info("已优化内存使用")
}

// OptimizeCPUUsage 优化CPU使用
func (rm *ResourceManager) OptimizeCPUUsage() {
	// 设置GOMAXPROCS为CPU核心数
	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)
	rm.logger.Info("已优化CPU使用", "gomaxprocs", n)
}

// OptimizeResourceUsage 优化资源使用
func (rm *ResourceManager) OptimizeResourceUsage() {
	// 优化内存使用
	rm.OptimizeMemoryUsage()
	// 优化CPU使用
	rm.OptimizeCPUUsage()
	rm.logger.Info("已优化资源使用")
}
