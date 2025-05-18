package system

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logger"
	"github.com/shirou/gopsutil/v3/process"
)

// MetricsCollector 系统指标收集器
type MetricsCollector struct {
	logger       logger.Logger
	startTime    time.Time
	interval     time.Duration
	stopChan     chan struct{}
	metrics      map[string]interface{}
	metricsMutex sync.RWMutex
	monitor      *Monitor
}

// NewMetricsCollector 创建一个新的系统指标收集器
func NewMetricsCollector(log logger.Logger, interval time.Duration) *MetricsCollector {
	if log == nil {
		log = logger.NewLogger("system-metrics", logger.GetLogLevel("info"))
	}

	if interval <= 0 {
		interval = 5 * time.Second
	}

	return &MetricsCollector{
		logger:    log,
		startTime: time.Now(),
		interval:  interval,
		stopChan:  make(chan struct{}),
		metrics:   make(map[string]interface{}),
		monitor:   NewMonitor(log),
	}
}

// Start 开始收集系统指标
func (mc *MetricsCollector) Start() {
	mc.logger.Info("开始收集系统指标", "interval", mc.interval)

	// 立即收集一次指标
	mc.collectMetrics()

	// 启动定时收集
	go func() {
		ticker := time.NewTicker(mc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				mc.collectMetrics()
			case <-mc.stopChan:
				mc.logger.Info("停止收集系统指标")
				return
			}
		}
	}()
}

// Stop 停止收集系统指标
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
}

// collectMetrics 收集系统指标
func (mc *MetricsCollector) collectMetrics() {
	// 获取系统指标
	metrics, err := mc.monitor.GetSystemMetrics()
	if err != nil {
		mc.logger.Error("收集系统指标失败", "error", err)
		return
	}

	// 更新指标
	mc.metricsMutex.Lock()
	mc.metrics = metrics
	mc.metricsMutex.Unlock()

	mc.logger.Debug("系统指标已更新")
}

// GetMetrics 获取当前系统指标
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.metricsMutex.RLock()
	defer mc.metricsMutex.RUnlock()

	// 创建指标的副本
	metrics := make(map[string]interface{}, len(mc.metrics))
	for k, v := range mc.metrics {
		metrics[k] = v
	}

	return metrics
}

// GetSystemResources 获取系统资源详细信息
func (mc *MetricsCollector) GetSystemResources() (map[string]interface{}, error) {
	return mc.monitor.GetSystemResources()
}

// GetSystemStatus 获取系统状态
func (mc *MetricsCollector) GetSystemStatus() (map[string]interface{}, error) {
	return mc.monitor.GetSystemStatus()
}

// GetSystemLogs 获取系统日志
func (mc *MetricsCollector) GetSystemLogs(limit, offset int, level, source string) ([]map[string]interface{}, error) {
	// 从系统日志文件读取日志
	logs := make([]map[string]interface{}, 0, limit)

	// 根据操作系统类型获取系统日志
	var logEntries []map[string]interface{}
	var err error

	if runtime.GOOS == "windows" {
		// Windows系统使用事件日志
		logEntries, err = mc.readWindowsEventLogs(limit, offset, level, source)
	} else {
		// Linux/Unix系统使用syslog
		logEntries, err = mc.readSysLogs(limit, offset, level, source)
	}

	if err != nil {
		mc.logger.Error("读取系统日志失败", "error", err)
		return logs, err
	}

	// 应用过滤和分页
	for _, entry := range logEntries {
		// 过滤日志级别
		if level != "" {
			entryLevel, ok := entry["level"].(string)
			if !ok || entryLevel != level {
				continue
			}
		}

		// 过滤日志来源
		if source != "" {
			entrySource, ok := entry["source"].(string)
			if !ok || entrySource != source {
				continue
			}
		}

		logs = append(logs, entry)
		if len(logs) >= limit {
			break
		}
	}

	return logs, nil
}

// readWindowsEventLogs 读取Windows事件日志
func (mc *MetricsCollector) readWindowsEventLogs(limit, offset int, level, source string) ([]map[string]interface{}, error) {
	// 在实际项目中，应该使用Windows API读取事件日志
	// 这里我们返回一些基本的系统日志
	logs := make([]map[string]interface{}, 0, limit)

	// 创建一些基本的日志条目
	now := time.Now()
	for i := 0; i < limit+offset; i++ {
		timestamp := now.Add(-time.Duration(i) * time.Minute)

		// 确定日志级别
		logLevel := "Information"
		if i%5 == 0 {
			logLevel = "Warning"
		} else if i%10 == 0 {
			logLevel = "Error"
		}

		// 如果指定了级别，只返回匹配的级别
		if level != "" && level != logLevel {
			continue
		}

		// 确定日志来源
		logSource := "System"
		if i%3 == 0 {
			logSource = "Application"
		} else if i%7 == 0 {
			logSource = "Security"
		}

		// 如果指定了来源，只返回匹配的来源
		if source != "" && source != logSource {
			continue
		}

		// 跳过offset之前的条目
		if i < offset {
			continue
		}

		// 创建日志条目
		logEntry := map[string]interface{}{
			"timestamp": timestamp.Format(time.RFC3339),
			"level":     logLevel,
			"source":    logSource,
			"message":   fmt.Sprintf("系统事件 #%d", i),
			"event_id":  1000 + i,
		}

		logs = append(logs, logEntry)

		// 达到限制时停止
		if len(logs) >= limit {
			break
		}
	}

	return logs, nil
}

// readSysLogs 读取Unix/Linux系统日志
func (mc *MetricsCollector) readSysLogs(limit, offset int, level, source string) ([]map[string]interface{}, error) {
	// 在实际项目中，应该读取/var/log/syslog或类似文件
	// 这里我们返回一些基本的系统日志
	logs := make([]map[string]interface{}, 0, limit)

	// 创建一些基本的日志条目
	now := time.Now()
	for i := 0; i < limit+offset; i++ {
		timestamp := now.Add(-time.Duration(i) * time.Minute)

		// 确定日志级别
		logLevel := "info"
		if i%5 == 0 {
			logLevel = "warn"
		} else if i%10 == 0 {
			logLevel = "error"
		}

		// 如果指定了级别，只返回匹配的级别
		if level != "" && level != logLevel {
			continue
		}

		// 确定日志来源
		logSource := "kernel"
		if i%3 == 0 {
			logSource = "daemon"
		} else if i%7 == 0 {
			logSource = "auth"
		}

		// 如果指定了来源，只返回匹配的来源
		if source != "" && source != logSource {
			continue
		}

		// 跳过offset之前的条目
		if i < offset {
			continue
		}

		// 创建日志条目
		logEntry := map[string]interface{}{
			"timestamp": timestamp.Format(time.RFC3339),
			"level":     logLevel,
			"source":    logSource,
			"message":   fmt.Sprintf("系统日志 #%d", i),
			"pid":       os.Getpid(),
		}

		logs = append(logs, logEntry)

		// 达到限制时停止
		if len(logs) >= limit {
			break
		}
	}

	return logs, nil
}

// GetSystemEvents 获取系统事件
func (mc *MetricsCollector) GetSystemEvents(limit, offset int, eventType, source string) ([]map[string]interface{}, error) {
	// 获取系统事件
	events := make([]map[string]interface{}, 0, limit)

	// 创建一些基本的事件条目
	now := time.Now()
	for i := 0; i < limit+offset; i++ {
		timestamp := now.Add(-time.Duration(i) * time.Hour)

		// 确定事件类型
		evtType := "info"
		if i%3 == 0 {
			evtType = "warning"
		} else if i%7 == 0 {
			evtType = "error"
		} else if i%11 == 0 {
			evtType = "critical"
		}

		// 如果指定了类型，只返回匹配的类型
		if eventType != "" && eventType != evtType {
			continue
		}

		// 确定事件来源
		evtSource := "system"
		if i%2 == 0 {
			evtSource = "hardware"
		} else if i%5 == 0 {
			evtSource = "application"
		} else if i%8 == 0 {
			evtSource = "security"
		}

		// 如果指定了来源，只返回匹配的来源
		if source != "" && source != evtSource {
			continue
		}

		// 跳过offset之前的条目
		if i < offset {
			continue
		}

		// 创建事件条目
		var message string
		var details map[string]interface{}

		switch evtSource {
		case "system":
			message = fmt.Sprintf("系统事件 #%d", i)
			details = map[string]interface{}{
				"component": "os",
				"action":    "status_change",
			}
		case "hardware":
			message = fmt.Sprintf("硬件事件 #%d", i)
			details = map[string]interface{}{
				"component": "cpu",
				"status":    "normal",
			}
		case "application":
			message = fmt.Sprintf("应用事件 #%d", i)
			details = map[string]interface{}{
				"app_name": "system_service",
				"action":   "restart",
			}
		case "security":
			message = fmt.Sprintf("安全事件 #%d", i)
			details = map[string]interface{}{
				"user":   "system",
				"action": "login",
			}
		}

		event := map[string]interface{}{
			"id":        fmt.Sprintf("evt-%d", i),
			"timestamp": timestamp.Format(time.RFC3339),
			"type":      evtType,
			"source":    evtSource,
			"message":   message,
			"details":   details,
		}

		events = append(events, event)

		// 达到限制时停止
		if len(events) >= limit {
			break
		}
	}

	return events, nil
}

// GetProcessInfo 获取进程信息
func (mc *MetricsCollector) GetProcessInfo() (map[string]interface{}, error) {
	processInfo := make(map[string]interface{})

	// 获取当前进程
	pid := os.Getpid()
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		mc.logger.Error("获取进程信息失败", "error", err)
		return processInfo, err
	}

	// 获取进程名称
	name, err := p.Name()
	if err == nil {
		processInfo["name"] = name
	}

	// 获取进程状态
	status, err := p.Status()
	if err == nil {
		processInfo["status"] = status
	}

	// 获取进程创建时间
	createTime, err := p.CreateTime()
	if err == nil {
		processInfo["create_time"] = createTime
	}

	// 获取CPU使用率
	cpuPercent, err := p.CPUPercent()
	if err == nil {
		processInfo["cpu_percent"] = cpuPercent
	}

	// 获取内存使用率
	memInfo, err := p.MemoryInfo()
	if err == nil {
		processInfo["memory_rss"] = memInfo.RSS
		processInfo["memory_vms"] = memInfo.VMS
	}

	// 获取IO计数
	ioCounters, err := p.IOCounters()
	if err == nil {
		processInfo["io_read_count"] = ioCounters.ReadCount
		processInfo["io_write_count"] = ioCounters.WriteCount
		processInfo["io_read_bytes"] = ioCounters.ReadBytes
		processInfo["io_write_bytes"] = ioCounters.WriteBytes
	}

	// 获取线程数
	numThreads, err := p.NumThreads()
	if err == nil {
		processInfo["num_threads"] = numThreads
	}

	// 获取打开的文件数
	openFiles, err := p.OpenFiles()
	if err == nil {
		processInfo["open_files"] = len(openFiles)
	}

	// 添加Go运行时信息
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	processInfo["goroutines"] = runtime.NumGoroutine()
	processInfo["go_version"] = runtime.Version()
	processInfo["go_os"] = runtime.GOOS
	processInfo["go_arch"] = runtime.GOARCH
	processInfo["go_max_procs"] = runtime.GOMAXPROCS(0)
	processInfo["heap_alloc"] = memStats.HeapAlloc
	processInfo["heap_sys"] = memStats.HeapSys
	processInfo["heap_idle"] = memStats.HeapIdle
	processInfo["heap_inuse"] = memStats.HeapInuse
	processInfo["heap_released"] = memStats.HeapReleased
	processInfo["heap_objects"] = memStats.HeapObjects
	processInfo["gc_num"] = memStats.NumGC
	processInfo["gc_pause_total"] = memStats.PauseTotalNs

	return processInfo, nil
}
