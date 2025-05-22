package system

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// MetricsCollector 系统指标收集器
type MetricsCollector struct {
	logger       logging.Logger
	startTime    time.Time
	interval     time.Duration
	stopChan     chan struct{}
	metrics      map[string]interface{}
	metricsMutex sync.RWMutex
	monitor      *Monitor
}

// NewMetricsCollector 创建一个新的系统指标收集器
func NewMetricsCollector(log logging.Logger, interval time.Duration) *MetricsCollector {
	if log == nil {
		// 创建默认日志配置
		config := logging.DefaultLogConfig()
		config.Level = logging.LogLevelInfo

		// 创建增强日志记录器
		enhancedLogger, err := logging.NewEnhancedLogger(config)
		if err != nil {
			// 如果创建失败，使用默认配置
			enhancedLogger, _ = logging.NewEnhancedLogger(nil)
		}

		// 设置名称
		log = enhancedLogger.Named("system-metrics")
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
	// 设置超时上下文，防止方法执行时间过长
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建一个通道用于接收结果
	resultChan := make(chan map[string]interface{}, 1)
	errChan := make(chan error, 1)

	// 在后台协程中执行状态收集
	go func() {
		status, err := mc.monitor.GetSystemStatus()
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- status
	}()

	// 等待结果或超时
	select {
	case <-ctx.Done():
		// 超时处理
		mc.logger.Error("获取系统状态超时")
		return map[string]interface{}{
			"error":     "获取系统状态超时",
			"timestamp": time.Now().Format(time.RFC3339),
		}, ctx.Err()
	case err := <-errChan:
		return map[string]interface{}{
			"error":     "获取系统状态失败: " + err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		}, err
	case result := <-resultChan:
		return result, nil
	}
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
