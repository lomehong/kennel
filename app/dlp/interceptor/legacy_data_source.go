//go:build windows

package interceptor

import (
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// LegacyDataSource 基于现有ProcessTracker的数据源实现
type LegacyDataSource struct {
	logger         logging.Logger
	processTracker *ProcessTracker
	mu             sync.RWMutex

	// 统计信息
	stats struct {
		queriesHandled    int64
		successfulLookups int64
		failedLookups     int64
		lastQueryTime     time.Time
		mu                sync.RWMutex
	}
}

// NewLegacyDataSource 创建基于现有ProcessTracker的数据源
func NewLegacyDataSource(logger logging.Logger) ProcessDataSource {
	ds := &LegacyDataSource{
		logger:         logger,
		processTracker: NewProcessTracker(logger),
	}

	// 启动定期更新
	ds.processTracker.StartPeriodicUpdate(10 * time.Second)

	return ds
}

// GetProcessInfo 获取进程信息
func (ds *LegacyDataSource) GetProcessInfo(packet *PacketInfo) *ProcessInfo {
	ds.stats.mu.Lock()
	ds.stats.queriesHandled++
	ds.stats.lastQueryTime = time.Now()
	ds.stats.mu.Unlock()

	if packet == nil {
		return nil
	}

	startTime := time.Now()

	// 使用现有的ProcessTracker查找进程
	var pid uint32

	// 策略1：尝试四元组匹配（TCP连接）
	if packet.Protocol == ProtocolTCP {
		pid = ds.processTracker.GetProcessByConnectionEx(
			packet.Protocol,
			packet.SourceIP,
			packet.SourcePort,
			packet.DestIP,
			packet.DestPort,
		)
	}

	// 策略2：如果四元组匹配失败，尝试本地连接匹配
	if pid == 0 {
		pid = ds.processTracker.GetProcessByConnection(
			packet.Protocol,
			packet.SourceIP,
			packet.SourcePort,
		)
	}

	// 策略3：尝试反向查找（交换源和目标）
	if pid == 0 {
		pid = ds.processTracker.GetProcessByConnection(
			packet.Protocol,
			packet.DestIP,
			packet.DestPort,
		)
	}

	queryDuration := time.Since(startTime)

	if pid == 0 {
		ds.stats.mu.Lock()
		ds.stats.failedLookups++
		ds.stats.mu.Unlock()

		ds.logger.Debug("Legacy数据源未找到进程",
			"protocol", packet.Protocol,
			"source_ip", packet.SourceIP.String(),
			"source_port", packet.SourcePort,
			"dest_ip", packet.DestIP.String(),
			"dest_port", packet.DestPort,
			"query_duration", queryDuration,
		)
		return nil
	}

	// 获取进程详细信息
	processInfo := ds.processTracker.GetProcessInfo(pid)
	if processInfo == nil {
		ds.stats.mu.Lock()
		ds.stats.failedLookups++
		ds.stats.mu.Unlock()

		ds.logger.Debug("Legacy数据源找到PID但无法获取进程信息",
			"pid", pid,
			"query_duration", queryDuration,
		)
		return nil
	}

	ds.stats.mu.Lock()
	ds.stats.successfulLookups++
	ds.stats.mu.Unlock()

	ds.logger.Debug("Legacy数据源找到进程信息",
		"pid", processInfo.PID,
		"process_name", processInfo.ProcessName,
		"execute_path", processInfo.ExecutePath,
		"user", processInfo.User,
		"protocol", packet.Protocol,
		"source_ip", packet.SourceIP.String(),
		"source_port", packet.SourcePort,
		"dest_ip", packet.DestIP.String(),
		"dest_port", packet.DestPort,
		"query_duration", queryDuration,
	)

	return processInfo
}

// Priority 返回数据源优先级
func (ds *LegacyDataSource) Priority() int {
	return 50 // 中等优先级，作为ETW的备选方案
}

// Name 返回数据源名称
func (ds *LegacyDataSource) Name() string {
	return "Legacy"
}

// Stop 停止数据源
func (ds *LegacyDataSource) Stop() error {
	ds.logger.Info("停止Legacy数据源")

	if ds.processTracker != nil {
		ds.processTracker.StopPeriodicUpdate()
	}

	ds.logger.Info("Legacy数据源已停止")
	return nil
}

// GetStats 获取统计信息
func (ds *LegacyDataSource) GetStats() map[string]interface{} {
	ds.stats.mu.RLock()
	defer ds.stats.mu.RUnlock()

	successRate := float64(0)
	if ds.stats.queriesHandled > 0 {
		successRate = float64(ds.stats.successfulLookups) / float64(ds.stats.queriesHandled) * 100
	}

	stats := map[string]interface{}{
		"name":               ds.Name(),
		"priority":           ds.Priority(),
		"queries_handled":    ds.stats.queriesHandled,
		"successful_lookups": ds.stats.successfulLookups,
		"failed_lookups":     ds.stats.failedLookups,
		"success_rate":       successRate,
		"last_query_time":    ds.stats.lastQueryTime,
	}

	// 添加ProcessTracker的统计信息
	if ds.processTracker != nil {
		trackerStats := ds.processTracker.GetMonitoringStats()
		stats["process_tracker"] = trackerStats
	}

	return stats
}

// GetProcessTracker 获取底层的ProcessTracker（用于调试和监控）
func (ds *LegacyDataSource) GetProcessTracker() *ProcessTracker {
	return ds.processTracker
}

// UpdateConnectionTables 手动更新连接表
func (ds *LegacyDataSource) UpdateConnectionTables() error {
	if ds.processTracker == nil {
		return nil
	}
	return ds.processTracker.UpdateConnectionTables()
}

// ClearCache 清理缓存
func (ds *LegacyDataSource) ClearCache() {
	if ds.processTracker != nil {
		ds.processTracker.ClearCache()
	}
	ds.logger.Info("Legacy数据源缓存已清理")
}

// IsMonitoringActive 检查监控是否活跃
func (ds *LegacyDataSource) IsMonitoringActive() bool {
	if ds.processTracker == nil {
		return false
	}

	stats := ds.processTracker.GetMonitoringStats()
	if active, ok := stats["monitoring_active"].(bool); ok {
		return active
	}

	return false
}

// SetPeriodicUpdateInterval 设置定期更新间隔
func (ds *LegacyDataSource) SetPeriodicUpdateInterval(interval time.Duration) {
	if ds.processTracker == nil {
		return
	}

	// 停止当前监控
	ds.processTracker.StopPeriodicUpdate()

	// 启动新的监控
	ds.processTracker.StartPeriodicUpdate(interval)

	ds.logger.Info("Legacy数据源更新间隔已设置", "interval", interval)
}
