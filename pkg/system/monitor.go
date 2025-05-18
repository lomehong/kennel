package system

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logger"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Monitor 系统监控器，负责收集系统指标
type Monitor struct {
	logger       logger.Logger
	startTime    time.Time
	lastCPUStat  []cpu.InfoStat
	lastNetStat  []net.IOCountersStat
	lastDiskStat []disk.IOCountersStat
	mutex        sync.RWMutex
}

// NewMonitor 创建一个新的系统监控器
func NewMonitor(log logger.Logger) *Monitor {
	if log == nil {
		log = logger.NewLogger("system-monitor", logger.GetLogLevel("info"))
	}

	return &Monitor{
		logger:    log,
		startTime: time.Now(),
	}
}

// GetSystemMetrics 获取系统指标
func (m *Monitor) GetSystemMetrics() (map[string]interface{}, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	metrics := make(map[string]interface{})

	// 获取CPU使用率
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		m.logger.Error("获取CPU使用率失败", "error", err)
	} else if len(cpuPercent) > 0 {
		metrics["cpu_usage"] = cpuPercent[0]
	}

	// 获取内存使用率
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		m.logger.Error("获取内存信息失败", "error", err)
	} else {
		metrics["memory_usage"] = memInfo.UsedPercent
	}

	// 获取磁盘使用率
	diskInfo, err := disk.Usage("/")
	if err != nil {
		m.logger.Error("获取磁盘信息失败", "error", err)
	} else {
		metrics["disk_usage"] = diskInfo.UsedPercent
	}

	// 获取系统负载
	loadInfo, err := load.Avg()
	if err != nil {
		m.logger.Error("获取系统负载失败", "error", err)
	} else {
		metrics["load_avg"] = map[string]float64{
			"1m":  loadInfo.Load1,
			"5m":  loadInfo.Load5,
			"15m": loadInfo.Load15,
		}
	}

	// 获取网络IO
	netIO, err := net.IOCounters(false)
	if err != nil {
		m.logger.Error("获取网络IO失败", "error", err)
	} else if len(netIO) > 0 {
		metrics["network"] = map[string]interface{}{
			"rx_bytes":   netIO[0].BytesRecv,
			"tx_bytes":   netIO[0].BytesSent,
			"rx_packets": netIO[0].PacketsRecv,
			"tx_packets": netIO[0].PacketsSent,
			"rx_errors":  netIO[0].Errin,
			"tx_errors":  netIO[0].Errout,
		}
	}

	// 获取系统运行时间
	hostInfo, err := host.Info()
	if err != nil {
		m.logger.Error("获取主机信息失败", "error", err)
	} else {
		metrics["uptime"] = hostInfo.Uptime
	}

	// 添加时间戳
	metrics["timestamp"] = time.Now().Format(time.RFC3339)

	return metrics, nil
}

// GetSystemResources 获取系统资源详细信息
func (m *Monitor) GetSystemResources() (map[string]interface{}, error) {
	resources := make(map[string]interface{})

	// 获取CPU信息
	cpuInfo, err := cpu.Info()
	if err != nil {
		m.logger.Error("获取CPU信息失败", "error", err)
	} else {
		cpuPercent, _ := cpu.Percent(0, false)
		cpuTemp, _ := host.SensorsTemperatures()

		var temperature float64
		for _, temp := range cpuTemp {
			if temp.SensorKey == "coretemp_packageid0_input" {
				temperature = temp.Temperature
				break
			}
		}

		resources["cpu"] = map[string]interface{}{
			"cores":       runtime.NumCPU(),
			"usage_pct":   cpuPercent[0],
			"temperature": temperature,
			"frequency":   cpuInfo[0].Mhz,
		}
	}

	// 获取内存信息
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		m.logger.Error("获取内存信息失败", "error", err)
	} else {
		resources["memory"] = map[string]interface{}{
			"total":    memInfo.Total,
			"used":     memInfo.Used,
			"free":     memInfo.Free,
			"used_pct": memInfo.UsedPercent,
			"cached":   memInfo.Cached,
			"buffers":  memInfo.Buffers,
		}
	}

	// 获取磁盘信息
	diskInfo, err := disk.Usage("/")
	if err != nil {
		m.logger.Error("获取磁盘信息失败", "error", err)
	} else {
		diskIO, _ := disk.IOCounters()
		var readRate, writeRate uint64
		for _, io := range diskIO {
			readRate += io.ReadBytes
			writeRate += io.WriteBytes
		}

		resources["disk"] = map[string]interface{}{
			"total":      diskInfo.Total,
			"used":       diskInfo.Used,
			"free":       diskInfo.Free,
			"used_pct":   diskInfo.UsedPercent,
			"read_rate":  readRate,
			"write_rate": writeRate,
		}
	}

	// 获取进程信息
	processes, err := process.Processes()
	if err != nil {
		m.logger.Error("获取进程信息失败", "error", err)
	} else {
		var threadCount int32
		for _, p := range processes {
			numThreads, err := p.NumThreads()
			if err == nil {
				threadCount += numThreads
			}
		}

		resources["process"] = map[string]interface{}{
			"count":      len(processes),
			"threads":    threadCount,
			"goroutines": runtime.NumGoroutine(),
		}
	}

	// 获取运行时信息
	resources["runtime"] = map[string]interface{}{
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"cpu_cores":  runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
	}

	// 添加时间戳
	resources["timestamp"] = time.Now().Format(time.RFC3339)

	return resources, nil
}

// GetSystemStatus 获取系统状态
func (m *Monitor) GetSystemStatus() (map[string]interface{}, error) {
	status := make(map[string]interface{})

	// 获取主机信息
	hostInfo, err := host.Info()
	if err != nil {
		m.logger.Error("获取主机信息失败", "error", err)
	} else {
		status["host"] = map[string]interface{}{
			"hostname":         hostInfo.Hostname,
			"platform":         hostInfo.Platform,
			"platform_version": hostInfo.PlatformVersion,
			"uptime":           fmt.Sprintf("%d小时%d分钟", hostInfo.Uptime/3600, (hostInfo.Uptime%3600)/60),
		}
	}

	// 获取框架信息
	appUptime := int(time.Since(m.startTime).Seconds())
	status["framework"] = map[string]interface{}{
		"version":    "1.0.0", // 从配置或常量中获取
		"start_time": m.startTime.Format(time.RFC3339),
		"uptime":     fmt.Sprintf("%d小时%d分钟", appUptime/3600, (appUptime%3600)/60),
	}

	// 获取运行时信息
	status["runtime"] = map[string]interface{}{
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"cpu_cores":  runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
	}

	// 添加时间戳
	status["timestamp"] = time.Now().Format(time.RFC3339)

	return status, nil
}

// GetHostname 获取主机名
func (m *Monitor) GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		m.logger.Error("获取主机名失败", "error", err)
		return "unknown"
	}
	return hostname
}
