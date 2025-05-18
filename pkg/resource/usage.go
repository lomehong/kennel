package resource

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// ResourceType 资源类型
type ResourceType string

// 预定义资源类型
const (
	ResourceTypeCPU     ResourceType = "cpu"     // CPU
	ResourceTypeMemory  ResourceType = "memory"  // 内存
	ResourceTypeDisk    ResourceType = "disk"    // 磁盘
	ResourceTypeNetwork ResourceType = "network" // 网络
)

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	// CPU使用情况
	CPUUsage      float64   // CPU使用率（百分比）
	CPUTime       float64   // CPU时间（秒）
	CPUUserTime   float64   // CPU用户时间（秒）
	CPUSystemTime float64   // CPU系统时间（秒）
	CPUCount      int       // CPU核心数
	CPUPercent    []float64 // 每个CPU核心的使用率

	// 内存使用情况
	MemoryUsage     uint64  // 内存使用量（字节）
	MemoryPercent   float64 // 内存使用率（百分比）
	MemoryRSS       uint64  // 常驻内存集（字节）
	MemoryVMS       uint64  // 虚拟内存大小（字节）
	MemorySwap      uint64  // 交换内存使用量（字节）
	MemoryTotal     uint64  // 总内存（字节）
	MemoryFree      uint64  // 空闲内存（字节）
	MemoryAvailable uint64  // 可用内存（字节）

	// 磁盘使用情况
	DiskUsage      uint64  // 磁盘使用量（字节）
	DiskPercent    float64 // 磁盘使用率（百分比）
	DiskTotal      uint64  // 总磁盘空间（字节）
	DiskFree       uint64  // 空闲磁盘空间（字节）
	DiskReadBytes  uint64  // 磁盘读取字节数
	DiskWriteBytes uint64  // 磁盘写入字节数
	DiskReadCount  uint64  // 磁盘读取次数
	DiskWriteCount uint64  // 磁盘写入次数
	DiskReadTime   uint64  // 磁盘读取时间（毫秒）
	DiskWriteTime  uint64  // 磁盘写入时间（毫秒）

	// 网络使用情况
	NetworkSentBytes      uint64 // 发送字节数
	NetworkReceivedBytes  uint64 // 接收字节数
	NetworkSentPackets    uint64 // 发送数据包数
	NetworkRecvPackets    uint64 // 接收数据包数
	NetworkDroppedPackets uint64 // 丢弃数据包数
	NetworkErrorPackets   uint64 // 错误数据包数

	// 进程信息
	ProcessID         int32   // 进程ID
	ProcessName       string  // 进程名称
	ProcessStatus     string  // 进程状态
	ProcessCreateTime int64   // 进程创建时间（Unix时间戳）
	ProcessThreads    int32   // 进程线程数
	ProcessFDs        int32   // 进程文件描述符数
	ProcessChildren   []int32 // 子进程ID列表

	// 时间信息
	Timestamp time.Time // 时间戳
}

// ResourceUsageSnapshot 资源使用快照
type ResourceUsageSnapshot struct {
	Current  *ResourceUsage            // 当前资源使用情况
	Previous *ResourceUsage            // 上一次资源使用情况
	Delta    *ResourceUsage            // 资源使用变化
	History  []*ResourceUsage          // 历史资源使用情况
	Stats    map[string]interface{}    // 统计信息
	Limits   map[ResourceType]uint64   // 资源限制
	Alerts   map[ResourceType][]string // 资源告警
}

// NewResourceUsage 创建资源使用情况
func NewResourceUsage() *ResourceUsage {
	return &ResourceUsage{
		Timestamp: time.Now(),
	}
}

// NewResourceUsageSnapshot 创建资源使用快照
func NewResourceUsageSnapshot() *ResourceUsageSnapshot {
	return &ResourceUsageSnapshot{
		Current:  NewResourceUsage(),
		Previous: NewResourceUsage(),
		Delta:    NewResourceUsage(),
		History:  make([]*ResourceUsage, 0),
		Stats:    make(map[string]interface{}),
		Limits:   make(map[ResourceType]uint64),
		Alerts:   make(map[ResourceType][]string),
	}
}

// ResourceUsageTracker 资源使用跟踪器
type ResourceUsageTracker struct {
	snapshot      *ResourceUsageSnapshot // 资源使用快照
	historyLimit  int                    // 历史记录限制
	processID     int32                  // 进程ID
	diskPaths     []string               // 磁盘路径
	networkIfaces []string               // 网络接口
	mu            sync.RWMutex           // 互斥锁
}

// NewResourceUsageTracker 创建资源使用跟踪器
func NewResourceUsageTracker(processID int32, historyLimit int) *ResourceUsageTracker {
	if historyLimit <= 0 {
		historyLimit = 100
	}

	return &ResourceUsageTracker{
		snapshot:      NewResourceUsageSnapshot(),
		historyLimit:  historyLimit,
		processID:     processID,
		diskPaths:     []string{"/"},
		networkIfaces: []string{},
	}
}

// SetDiskPaths 设置磁盘路径
func (t *ResourceUsageTracker) SetDiskPaths(paths []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.diskPaths = paths
}

// SetNetworkInterfaces 设置网络接口
func (t *ResourceUsageTracker) SetNetworkInterfaces(ifaces []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.networkIfaces = ifaces
}

// SetResourceLimit 设置资源限制
func (t *ResourceUsageTracker) SetResourceLimit(resourceType ResourceType, limit uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.snapshot.Limits[resourceType] = limit
}

// GetResourceLimit 获取资源限制
func (t *ResourceUsageTracker) GetResourceLimit(resourceType ResourceType) (uint64, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	limit, exists := t.snapshot.Limits[resourceType]
	return limit, exists
}

// RemoveResourceLimit 移除资源限制
func (t *ResourceUsageTracker) RemoveResourceLimit(resourceType ResourceType) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.snapshot.Limits, resourceType)
}

// GetSnapshot 获取资源使用快照
func (t *ResourceUsageTracker) GetSnapshot() *ResourceUsageSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// 复制快照
	snapshot := &ResourceUsageSnapshot{
		Current:  t.snapshot.Current,
		Previous: t.snapshot.Previous,
		Delta:    t.snapshot.Delta,
		History:  make([]*ResourceUsage, len(t.snapshot.History)),
		Stats:    make(map[string]interface{}),
		Limits:   make(map[ResourceType]uint64),
		Alerts:   make(map[ResourceType][]string),
	}

	// 复制历史记录
	copy(snapshot.History, t.snapshot.History)

	// 复制统计信息
	for k, v := range t.snapshot.Stats {
		snapshot.Stats[k] = v
	}

	// 复制资源限制
	for k, v := range t.snapshot.Limits {
		snapshot.Limits[k] = v
	}

	// 复制资源告警
	for k, v := range t.snapshot.Alerts {
		alerts := make([]string, len(v))
		copy(alerts, v)
		snapshot.Alerts[k] = alerts
	}

	return snapshot
}

// Update 更新资源使用情况
func (t *ResourceUsageTracker) Update() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 保存上一次资源使用情况
	t.snapshot.Previous = t.snapshot.Current
	t.snapshot.Current = NewResourceUsage()

	// 更新CPU使用情况
	if err := t.updateCPUUsage(); err != nil {
		return fmt.Errorf("更新CPU使用情况失败: %w", err)
	}

	// 更新内存使用情况
	if err := t.updateMemoryUsage(); err != nil {
		return fmt.Errorf("更新内存使用情况失败: %w", err)
	}

	// 更新磁盘使用情况
	if err := t.updateDiskUsage(); err != nil {
		return fmt.Errorf("更新磁盘使用情况失败: %w", err)
	}

	// 更新网络使用情况
	if err := t.updateNetworkUsage(); err != nil {
		return fmt.Errorf("更新网络使用情况失败: %w", err)
	}

	// 更新进程信息
	if err := t.updateProcessInfo(); err != nil {
		return fmt.Errorf("更新进程信息失败: %w", err)
	}

	// 计算资源使用变化
	t.calculateDelta()

	// 添加到历史记录
	t.addToHistory(t.snapshot.Current)

	// 计算统计信息
	t.calculateStats()

	// 检查资源限制
	t.checkResourceLimits()

	return nil
}

// updateCPUUsage 更新CPU使用情况
func (t *ResourceUsageTracker) updateCPUUsage() error {
	// 获取CPU使用率
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return err
	}
	if len(percent) > 0 {
		t.snapshot.Current.CPUUsage = percent[0]
	}

	// 获取每个CPU核心的使用率
	perCPUPercent, err := cpu.Percent(0, true)
	if err != nil {
		return err
	}
	t.snapshot.Current.CPUPercent = perCPUPercent
	t.snapshot.Current.CPUCount = runtime.NumCPU()

	// 如果指定了进程ID，获取进程的CPU使用情况
	if t.processID > 0 {
		proc, err := process.NewProcess(t.processID)
		if err != nil {
			return err
		}

		// 获取CPU时间
		times, err := proc.Times()
		if err != nil {
			return err
		}
		t.snapshot.Current.CPUTime = times.Total()
		t.snapshot.Current.CPUUserTime = times.User
		t.snapshot.Current.CPUSystemTime = times.System

		// 获取进程CPU使用率
		procPercent, err := proc.CPUPercent()
		if err != nil {
			return err
		}
		t.snapshot.Current.CPUUsage = procPercent
	}

	return nil
}

// updateMemoryUsage 更新内存使用情况
func (t *ResourceUsageTracker) updateMemoryUsage() error {
	// 获取系统内存使用情况
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	t.snapshot.Current.MemoryTotal = memInfo.Total
	t.snapshot.Current.MemoryFree = memInfo.Free
	t.snapshot.Current.MemoryAvailable = memInfo.Available
	t.snapshot.Current.MemoryPercent = memInfo.UsedPercent

	// 如果指定了进程ID，获取进程的内存使用情况
	if t.processID > 0 {
		proc, err := process.NewProcess(t.processID)
		if err != nil {
			return err
		}

		// 获取内存信息
		memInfo, err := proc.MemoryInfo()
		if err != nil {
			return err
		}
		t.snapshot.Current.MemoryRSS = memInfo.RSS
		t.snapshot.Current.MemoryVMS = memInfo.VMS
		t.snapshot.Current.MemoryUsage = memInfo.RSS

		// 获取内存使用率
		memPercent, err := proc.MemoryPercent()
		if err != nil {
			return err
		}
		t.snapshot.Current.MemoryPercent = float64(memPercent)
	}

	return nil
}

// updateDiskUsage 更新磁盘使用情况
func (t *ResourceUsageTracker) updateDiskUsage() error {
	// 获取磁盘使用情况
	for _, path := range t.diskPaths {
		usage, err := disk.Usage(path)
		if err != nil {
			return err
		}
		t.snapshot.Current.DiskUsage += usage.Used
		t.snapshot.Current.DiskTotal += usage.Total
		t.snapshot.Current.DiskFree += usage.Free
		// 使用加权平均计算总体磁盘使用率
		t.snapshot.Current.DiskPercent = float64(t.snapshot.Current.DiskUsage) / float64(t.snapshot.Current.DiskTotal) * 100
	}

	// 获取磁盘IO统计信息
	ioStats, err := disk.IOCounters()
	if err != nil {
		return err
	}
	for _, stat := range ioStats {
		t.snapshot.Current.DiskReadBytes += stat.ReadBytes
		t.snapshot.Current.DiskWriteBytes += stat.WriteBytes
		t.snapshot.Current.DiskReadCount += stat.ReadCount
		t.snapshot.Current.DiskWriteCount += stat.WriteCount
		t.snapshot.Current.DiskReadTime += stat.ReadTime
		t.snapshot.Current.DiskWriteTime += stat.WriteTime
	}

	return nil
}

// updateNetworkUsage 更新网络使用情况
func (t *ResourceUsageTracker) updateNetworkUsage() error {
	// 获取网络IO统计信息
	netStats, err := net.IOCounters(true)
	if err != nil {
		return err
	}

	// 如果没有指定网络接口，使用所有接口
	if len(t.networkIfaces) == 0 {
		for _, stat := range netStats {
			t.snapshot.Current.NetworkSentBytes += stat.BytesSent
			t.snapshot.Current.NetworkReceivedBytes += stat.BytesRecv
			t.snapshot.Current.NetworkSentPackets += stat.PacketsSent
			t.snapshot.Current.NetworkRecvPackets += stat.PacketsRecv
			t.snapshot.Current.NetworkDroppedPackets += stat.Dropout
			t.snapshot.Current.NetworkErrorPackets += stat.Errin + stat.Errout
		}
	} else {
		// 使用指定的网络接口
		for _, iface := range t.networkIfaces {
			for _, stat := range netStats {
				if stat.Name == iface {
					t.snapshot.Current.NetworkSentBytes += stat.BytesSent
					t.snapshot.Current.NetworkReceivedBytes += stat.BytesRecv
					t.snapshot.Current.NetworkSentPackets += stat.PacketsSent
					t.snapshot.Current.NetworkRecvPackets += stat.PacketsRecv
					t.snapshot.Current.NetworkDroppedPackets += stat.Dropout
					t.snapshot.Current.NetworkErrorPackets += stat.Errin + stat.Errout
					break
				}
			}
		}
	}

	return nil
}

// updateProcessInfo 更新进程信息
func (t *ResourceUsageTracker) updateProcessInfo() error {
	// 如果指定了进程ID，获取进程信息
	if t.processID > 0 {
		proc, err := process.NewProcess(t.processID)
		if err != nil {
			return err
		}

		// 根据操作系统选择不同的实现
		if runtime.GOOS == "windows" {
			// Windows平台使用特定的实现
			if err := UpdateProcessInfo(proc, t.snapshot.Current); err != nil {
				return fmt.Errorf("更新Windows进程信息失败: %w", err)
			}
		} else {
			// 非Windows平台使用通用实现
			// 获取进程名称
			name, err := proc.Name()
			if err != nil {
				return err
			}
			t.snapshot.Current.ProcessName = name

			// 获取进程状态
			status, err := proc.Status()
			if err != nil {
				return err
			}
			if len(status) > 0 {
				t.snapshot.Current.ProcessStatus = status[0]
			} else {
				t.snapshot.Current.ProcessStatus = "unknown"
			}

			// 获取进程创建时间
			createTime, err := proc.CreateTime()
			if err != nil {
				return err
			}
			t.snapshot.Current.ProcessCreateTime = createTime

			// 获取进程线程数
			numThreads, err := proc.NumThreads()
			if err != nil {
				return err
			}
			t.snapshot.Current.ProcessThreads = numThreads

			// 获取进程文件描述符数
			numFDs, err := proc.NumFDs()
			if err != nil {
				return err
			}
			t.snapshot.Current.ProcessFDs = numFDs

			// 获取子进程
			children, err := proc.Children()
			if err != nil {
				// 忽略错误，可能没有子进程
				t.snapshot.Current.ProcessChildren = []int32{}
			} else {
				childrenIDs := make([]int32, len(children))
				for i, child := range children {
					childrenIDs[i] = child.Pid
				}
				t.snapshot.Current.ProcessChildren = childrenIDs
			}
		}

		// 设置进程ID
		t.snapshot.Current.ProcessID = t.processID
	} else {
		// 如果没有指定进程ID，使用当前进程
		pid := int32(os.Getpid())
		t.processID = pid
		return t.updateProcessInfo()
	}

	return nil
}

// calculateDelta 计算资源使用变化
func (t *ResourceUsageTracker) calculateDelta() {
	// 如果没有上一次记录，无法计算变化
	if t.snapshot.Previous == nil {
		t.snapshot.Delta = NewResourceUsage()
		return
	}

	delta := NewResourceUsage()

	// 计算CPU使用变化
	delta.CPUUsage = t.snapshot.Current.CPUUsage - t.snapshot.Previous.CPUUsage
	delta.CPUTime = t.snapshot.Current.CPUTime - t.snapshot.Previous.CPUTime
	delta.CPUUserTime = t.snapshot.Current.CPUUserTime - t.snapshot.Previous.CPUUserTime
	delta.CPUSystemTime = t.snapshot.Current.CPUSystemTime - t.snapshot.Previous.CPUSystemTime

	// 计算内存使用变化
	delta.MemoryUsage = t.snapshot.Current.MemoryUsage - t.snapshot.Previous.MemoryUsage
	delta.MemoryPercent = t.snapshot.Current.MemoryPercent - t.snapshot.Previous.MemoryPercent
	delta.MemoryRSS = t.snapshot.Current.MemoryRSS - t.snapshot.Previous.MemoryRSS
	delta.MemoryVMS = t.snapshot.Current.MemoryVMS - t.snapshot.Previous.MemoryVMS

	// 计算磁盘使用变化
	delta.DiskUsage = t.snapshot.Current.DiskUsage - t.snapshot.Previous.DiskUsage
	delta.DiskPercent = t.snapshot.Current.DiskPercent - t.snapshot.Previous.DiskPercent
	delta.DiskReadBytes = t.snapshot.Current.DiskReadBytes - t.snapshot.Previous.DiskReadBytes
	delta.DiskWriteBytes = t.snapshot.Current.DiskWriteBytes - t.snapshot.Previous.DiskWriteBytes
	delta.DiskReadCount = t.snapshot.Current.DiskReadCount - t.snapshot.Previous.DiskReadCount
	delta.DiskWriteCount = t.snapshot.Current.DiskWriteCount - t.snapshot.Previous.DiskWriteCount
	delta.DiskReadTime = t.snapshot.Current.DiskReadTime - t.snapshot.Previous.DiskReadTime
	delta.DiskWriteTime = t.snapshot.Current.DiskWriteTime - t.snapshot.Previous.DiskWriteTime

	// 计算网络使用变化
	delta.NetworkSentBytes = t.snapshot.Current.NetworkSentBytes - t.snapshot.Previous.NetworkSentBytes
	delta.NetworkReceivedBytes = t.snapshot.Current.NetworkReceivedBytes - t.snapshot.Previous.NetworkReceivedBytes
	delta.NetworkSentPackets = t.snapshot.Current.NetworkSentPackets - t.snapshot.Previous.NetworkSentPackets
	delta.NetworkRecvPackets = t.snapshot.Current.NetworkRecvPackets - t.snapshot.Previous.NetworkRecvPackets
	delta.NetworkDroppedPackets = t.snapshot.Current.NetworkDroppedPackets - t.snapshot.Previous.NetworkDroppedPackets
	delta.NetworkErrorPackets = t.snapshot.Current.NetworkErrorPackets - t.snapshot.Previous.NetworkErrorPackets

	// 设置时间戳
	delta.Timestamp = t.snapshot.Current.Timestamp

	t.snapshot.Delta = delta
}

// addToHistory 添加到历史记录
func (t *ResourceUsageTracker) addToHistory(usage *ResourceUsage) {
	// 添加到历史记录
	t.snapshot.History = append(t.snapshot.History, usage)

	// 如果超过历史记录限制，移除最旧的记录
	if len(t.snapshot.History) > t.historyLimit {
		t.snapshot.History = t.snapshot.History[1:]
	}
}

// calculateStats 计算统计信息
func (t *ResourceUsageTracker) calculateStats() {
	// 如果历史记录为空，无法计算统计信息
	if len(t.snapshot.History) == 0 {
		return
	}

	// 计算CPU使用率统计信息
	var cpuUsageSum float64
	var cpuUsageMax float64
	var cpuUsageMin float64 = 100.0
	for _, usage := range t.snapshot.History {
		cpuUsageSum += usage.CPUUsage
		if usage.CPUUsage > cpuUsageMax {
			cpuUsageMax = usage.CPUUsage
		}
		if usage.CPUUsage < cpuUsageMin {
			cpuUsageMin = usage.CPUUsage
		}
	}
	t.snapshot.Stats["cpu_usage_avg"] = cpuUsageSum / float64(len(t.snapshot.History))
	t.snapshot.Stats["cpu_usage_max"] = cpuUsageMax
	t.snapshot.Stats["cpu_usage_min"] = cpuUsageMin

	// 计算内存使用率统计信息
	var memoryPercentSum float64
	var memoryPercentMax float64
	var memoryPercentMin float64 = 100.0
	for _, usage := range t.snapshot.History {
		memoryPercentSum += usage.MemoryPercent
		if usage.MemoryPercent > memoryPercentMax {
			memoryPercentMax = usage.MemoryPercent
		}
		if usage.MemoryPercent < memoryPercentMin {
			memoryPercentMin = usage.MemoryPercent
		}
	}
	t.snapshot.Stats["memory_percent_avg"] = memoryPercentSum / float64(len(t.snapshot.History))
	t.snapshot.Stats["memory_percent_max"] = memoryPercentMax
	t.snapshot.Stats["memory_percent_min"] = memoryPercentMin

	// 计算磁盘使用率统计信息
	var diskPercentSum float64
	var diskPercentMax float64
	var diskPercentMin float64 = 100.0
	for _, usage := range t.snapshot.History {
		diskPercentSum += usage.DiskPercent
		if usage.DiskPercent > diskPercentMax {
			diskPercentMax = usage.DiskPercent
		}
		if usage.DiskPercent < diskPercentMin {
			diskPercentMin = usage.DiskPercent
		}
	}
	t.snapshot.Stats["disk_percent_avg"] = diskPercentSum / float64(len(t.snapshot.History))
	t.snapshot.Stats["disk_percent_max"] = diskPercentMax
	t.snapshot.Stats["disk_percent_min"] = diskPercentMin

	// 计算网络使用统计信息
	var networkSentBytesSum uint64
	var networkReceivedBytesSum uint64
	for _, usage := range t.snapshot.History {
		networkSentBytesSum += usage.NetworkSentBytes
		networkReceivedBytesSum += usage.NetworkReceivedBytes
	}
	t.snapshot.Stats["network_sent_bytes_avg"] = networkSentBytesSum / uint64(len(t.snapshot.History))
	t.snapshot.Stats["network_received_bytes_avg"] = networkReceivedBytesSum / uint64(len(t.snapshot.History))
}

// checkResourceLimits 检查资源限制
func (t *ResourceUsageTracker) checkResourceLimits() {
	// 清空告警
	t.snapshot.Alerts = make(map[ResourceType][]string)

	// 检查CPU使用率限制
	if limit, exists := t.snapshot.Limits[ResourceTypeCPU]; exists {
		if uint64(t.snapshot.Current.CPUUsage) > limit {
			t.snapshot.Alerts[ResourceTypeCPU] = append(
				t.snapshot.Alerts[ResourceTypeCPU],
				fmt.Sprintf("CPU使用率超过限制: %.2f%% > %d%%", t.snapshot.Current.CPUUsage, limit),
			)
		}
	}

	// 检查内存使用限制
	if limit, exists := t.snapshot.Limits[ResourceTypeMemory]; exists {
		if t.snapshot.Current.MemoryUsage > limit {
			t.snapshot.Alerts[ResourceTypeMemory] = append(
				t.snapshot.Alerts[ResourceTypeMemory],
				fmt.Sprintf("内存使用超过限制: %d > %d", t.snapshot.Current.MemoryUsage, limit),
			)
		}
	}

	// 检查磁盘使用限制
	if limit, exists := t.snapshot.Limits[ResourceTypeDisk]; exists {
		if t.snapshot.Current.DiskUsage > limit {
			t.snapshot.Alerts[ResourceTypeDisk] = append(
				t.snapshot.Alerts[ResourceTypeDisk],
				fmt.Sprintf("磁盘使用超过限制: %d > %d", t.snapshot.Current.DiskUsage, limit),
			)
		}
	}

	// 检查网络使用限制
	if limit, exists := t.snapshot.Limits[ResourceTypeNetwork]; exists {
		if t.snapshot.Current.NetworkSentBytes+t.snapshot.Current.NetworkReceivedBytes > limit {
			t.snapshot.Alerts[ResourceTypeNetwork] = append(
				t.snapshot.Alerts[ResourceTypeNetwork],
				fmt.Sprintf("网络使用超过限制: %d > %d", t.snapshot.Current.NetworkSentBytes+t.snapshot.Current.NetworkReceivedBytes, limit),
			)
		}
	}
}
