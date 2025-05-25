package interceptor

import (
	"net"
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// ConnectionKey 连接键，用于映射表索引
type ConnectionKey struct {
	Protocol   Protocol
	LocalAddr  string
	RemoteAddr string
}

// ProcessMapping 进程映射信息
type ProcessMapping struct {
	ProcessInfo *ProcessInfo
	Timestamp   time.Time
	AccessCount int64
	mu          sync.RWMutex
}

// ConnectionMapperImpl 进程-连接映射管理器实现
type ConnectionMapperImpl struct {
	logger   logging.Logger
	mappings map[ConnectionKey]*ProcessMapping
	mu       sync.RWMutex

	// 配置参数
	maxEntries    int
	expireTime    time.Duration
	cleanupTicker *time.Ticker
	stopChan      chan struct{}

	// 统计信息
	stats struct {
		totalMappings   int64
		activeMappings  int64
		expiredMappings int64
		lookupHits      int64
		lookupMisses    int64
		mu              sync.RWMutex
	}
}

// NewConnectionMapper 创建新的连接映射管理器
func NewConnectionMapper(logger logging.Logger) ConnectionMapper {
	cm := &ConnectionMapperImpl{
		logger:     logger,
		mappings:   make(map[ConnectionKey]*ProcessMapping),
		maxEntries: MAX_MAPPING_ENTRIES,
		expireTime: MAPPING_EXPIRE_TIME,
		stopChan:   make(chan struct{}),
	}

	// 启动清理协程
	cm.startCleanupRoutine()

	return cm
}

// AddMapping 添加进程-连接映射
func (cm *ConnectionMapperImpl) AddMapping(conn *ConnectionInfo, proc *ProcessInfo) {
	if conn == nil || proc == nil {
		cm.logger.Warn("尝试添加空的连接或进程信息")
		return
	}

	key := cm.createConnectionKey(conn)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查是否超过最大条目数
	if len(cm.mappings) >= cm.maxEntries {
		cm.logger.Warn("连接映射表已满，清理过期条目", "current_size", len(cm.mappings), "max_entries", cm.maxEntries)
		cm.cleanupExpiredMappingsLocked()

		// 如果清理后仍然满了，删除最旧的条目
		if len(cm.mappings) >= cm.maxEntries {
			cm.removeOldestMappingLocked()
		}
	}

	// 添加新映射
	mapping := &ProcessMapping{
		ProcessInfo: proc,
		Timestamp:   time.Now(),
		AccessCount: 0,
	}

	cm.mappings[key] = mapping

	// 更新统计信息
	cm.stats.mu.Lock()
	cm.stats.totalMappings++
	cm.stats.activeMappings++
	cm.stats.mu.Unlock()

	cm.logger.Debug("添加进程连接映射",
		"key", key,
		"pid", proc.PID,
		"process_name", proc.ProcessName,
		"local_addr", conn.LocalAddr.String(),
		"remote_addr", conn.RemoteAddr.String(),
	)
}

// GetProcessByConnection 根据连接信息获取进程信息
func (cm *ConnectionMapperImpl) GetProcessByConnection(conn *ConnectionInfo) *ProcessInfo {
	if conn == nil {
		return nil
	}

	key := cm.createConnectionKey(conn)
	return cm.getProcessByKey(key)
}

// GetProcessByAddresses 根据地址信息获取进程信息
func (cm *ConnectionMapperImpl) GetProcessByAddresses(protocol Protocol, localAddr, remoteAddr net.Addr) *ProcessInfo {
	key := ConnectionKey{
		Protocol:   protocol,
		LocalAddr:  localAddr.String(),
		RemoteAddr: remoteAddr.String(),
	}

	return cm.getProcessByKey(key)
}

// getProcessByKey 根据键获取进程信息
func (cm *ConnectionMapperImpl) getProcessByKey(key ConnectionKey) *ProcessInfo {
	cm.mu.RLock()
	mapping, exists := cm.mappings[key]
	cm.mu.RUnlock()

	if !exists {
		// 尝试反向查找（交换本地和远程地址）
		reverseKey := ConnectionKey{
			Protocol:   key.Protocol,
			LocalAddr:  key.RemoteAddr,
			RemoteAddr: key.LocalAddr,
		}

		cm.mu.RLock()
		mapping, exists = cm.mappings[reverseKey]
		cm.mu.RUnlock()

		if !exists {
			cm.stats.mu.Lock()
			cm.stats.lookupMisses++
			cm.stats.mu.Unlock()
			return nil
		}
	}

	// 检查映射是否过期
	if time.Since(mapping.Timestamp) > cm.expireTime {
		cm.logger.Debug("连接映射已过期", "key", key, "age", time.Since(mapping.Timestamp))

		// 删除过期映射
		cm.mu.Lock()
		delete(cm.mappings, key)
		cm.stats.activeMappings--
		cm.stats.expiredMappings++
		cm.mu.Unlock()

		cm.stats.mu.Lock()
		cm.stats.lookupMisses++
		cm.stats.mu.Unlock()
		return nil
	}

	// 更新访问计数和时间戳
	mapping.mu.Lock()
	mapping.AccessCount++
	mapping.Timestamp = time.Now() // 更新最后访问时间
	processInfo := mapping.ProcessInfo
	mapping.mu.Unlock()

	cm.stats.mu.Lock()
	cm.stats.lookupHits++
	cm.stats.mu.Unlock()

	cm.logger.Debug("找到进程连接映射",
		"key", key,
		"pid", processInfo.PID,
		"process_name", processInfo.ProcessName,
		"access_count", mapping.AccessCount,
	)

	return processInfo
}

// CleanExpiredMappings 清理过期映射
func (cm *ConnectionMapperImpl) CleanExpiredMappings() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cleanupExpiredMappingsLocked()
}

// cleanupExpiredMappingsLocked 清理过期映射（需要持有锁）
func (cm *ConnectionMapperImpl) cleanupExpiredMappingsLocked() {
	now := time.Now()
	expiredKeys := make([]ConnectionKey, 0)

	for key, mapping := range cm.mappings {
		if now.Sub(mapping.Timestamp) > cm.expireTime {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(cm.mappings, key)
		cm.stats.activeMappings--
		cm.stats.expiredMappings++
	}

	if len(expiredKeys) > 0 {
		cm.logger.Debug("清理过期连接映射", "expired_count", len(expiredKeys), "remaining_count", len(cm.mappings))
	}
}

// removeOldestMappingLocked 删除最旧的映射（需要持有锁）
func (cm *ConnectionMapperImpl) removeOldestMappingLocked() {
	var oldestKey ConnectionKey
	var oldestTime time.Time
	first := true

	for key, mapping := range cm.mappings {
		if first || mapping.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = mapping.Timestamp
			first = false
		}
	}

	if !first {
		delete(cm.mappings, oldestKey)
		cm.stats.activeMappings--
		cm.logger.Debug("删除最旧的连接映射", "key", oldestKey, "age", time.Since(oldestTime))
	}
}

// GetMappingCount 获取当前映射数量
func (cm *ConnectionMapperImpl) GetMappingCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.mappings)
}

// createConnectionKey 创建连接键
func (cm *ConnectionMapperImpl) createConnectionKey(conn *ConnectionInfo) ConnectionKey {
	return ConnectionKey{
		Protocol:   conn.Protocol,
		LocalAddr:  conn.LocalAddr.String(),
		RemoteAddr: conn.RemoteAddr.String(),
	}
}

// startCleanupRoutine 启动清理协程
func (cm *ConnectionMapperImpl) startCleanupRoutine() {
	cm.cleanupTicker = time.NewTicker(MAPPING_CLEANUP_INTERVAL)

	go func() {
		for {
			select {
			case <-cm.cleanupTicker.C:
				cm.CleanExpiredMappings()

			case <-cm.stopChan:
				cm.cleanupTicker.Stop()
				return
			}
		}
	}()

	cm.logger.Debug("连接映射清理协程已启动", "cleanup_interval", MAPPING_CLEANUP_INTERVAL)
}

// Stop 停止连接映射管理器
func (cm *ConnectionMapperImpl) Stop() {
	close(cm.stopChan)
	cm.logger.Debug("连接映射管理器已停止")
}

// GetStats 获取统计信息
func (cm *ConnectionMapperImpl) GetStats() map[string]interface{} {
	cm.stats.mu.RLock()
	defer cm.stats.mu.RUnlock()

	cm.mu.RLock()
	currentMappings := len(cm.mappings)
	cm.mu.RUnlock()

	hitRate := float64(0)
	totalLookups := cm.stats.lookupHits + cm.stats.lookupMisses
	if totalLookups > 0 {
		hitRate = float64(cm.stats.lookupHits) / float64(totalLookups) * 100
	}

	return map[string]interface{}{
		"total_mappings":   cm.stats.totalMappings,
		"active_mappings":  currentMappings,
		"expired_mappings": cm.stats.expiredMappings,
		"lookup_hits":      cm.stats.lookupHits,
		"lookup_misses":    cm.stats.lookupMisses,
		"hit_rate":         hitRate,
		"max_entries":      cm.maxEntries,
		"expire_time":      cm.expireTime.String(),
	}
}
