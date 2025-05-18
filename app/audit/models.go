package main

import (
	"sync"
	"time"
)

// AuditLog 表示一条审计日志
type AuditLog struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	User      string                 `json:"user"`
	Details   map[string]interface{} `json:"details"`
}

// AuditLogStore 管理审计日志存储
type AuditLogStore struct {
	mu   sync.RWMutex
	logs []AuditLog
}

// NewAuditLogStore 创建一个新的审计日志存储
func NewAuditLogStore() *AuditLogStore {
	return &AuditLogStore{
		logs: make([]AuditLog, 0, 100), // 预分配容量，减少动态扩容
	}
}

// AddLog 添加一条审计日志
func (s *AuditLogStore) AddLog(log AuditLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, log)
}

// GetLogs 获取所有审计日志
func (s *AuditLogStore) GetLogs() []AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 返回一个副本，避免外部修改
	logsCopy := make([]AuditLog, len(s.logs))
	copy(logsCopy, s.logs)
	return logsCopy
}

// FilterLogs 根据条件过滤审计日志
func (s *AuditLogStore) FilterLogs(eventType, user string, startTime, endTime time.Time) []AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 过滤日志
	filteredLogs := make([]AuditLog, 0)
	for _, log := range s.logs {
		// 检查事件类型
		if eventType != "" && log.EventType != eventType {
			continue
		}
		
		// 检查用户
		if user != "" && log.User != user {
			continue
		}
		
		// 检查时间范围
		if !startTime.IsZero() && log.Timestamp.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && log.Timestamp.After(endTime) {
			continue
		}
		
		filteredLogs = append(filteredLogs, log)
	}
	return filteredLogs
}

// ClearLogs 清除所有审计日志，但保留清除事件
func (s *AuditLogStore) ClearLogs(clearEvent AuditLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = []AuditLog{clearEvent}
}

// CleanupOldLogs 清理超过保留期的旧日志
func (s *AuditLogStore) CleanupOldLogs(retentionDays int) int {
	if retentionDays <= 0 {
		return 0
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	newLogs := make([]AuditLog, 0, len(s.logs))
	removedCount := 0
	
	for _, log := range s.logs {
		if log.Timestamp.After(cutoffTime) {
			newLogs = append(newLogs, log)
		} else {
			removedCount++
		}
	}
	
	s.logs = newLogs
	return removedCount
}

// LogsToMap 将审计日志转换为map切片
func LogsToMap(logs []AuditLog) []map[string]interface{} {
	// 转换日志为map切片
	logsMap := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		logMap := make(map[string]interface{})
		logMap["timestamp"] = log.Timestamp.Format(time.RFC3339)
		logMap["event_type"] = log.EventType
		logMap["user"] = log.User
		logMap["details"] = log.Details
		logsMap[i] = logMap
	}
	
	return logsMap
}
