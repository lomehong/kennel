package main

import (
	"sync"
	"time"
)

// DLPAlert 表示一个数据防泄漏警报
type DLPAlert struct {
	RuleID      string    `json:"rule_id"`
	RuleName    string    `json:"rule_name"`
	Content     string    `json:"content"`
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	Action      string    `json:"action"`
	Timestamp   time.Time `json:"timestamp"`
}

// AlertManager 警报管理器
type AlertManager struct {
	alerts []DLPAlert
	mu     sync.RWMutex
}

// NewAlertManager 创建一个新的警报管理器
func NewAlertManager() *AlertManager {
	return &AlertManager{
		alerts: make([]DLPAlert, 0),
	}
}

// AddAlert 添加警报
func (m *AlertManager) AddAlert(alert DLPAlert) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 设置时间戳
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}
	
	m.alerts = append(m.alerts, alert)
}

// GetAlerts 获取所有警报
func (m *AlertManager) GetAlerts() []DLPAlert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 返回一个副本，避免外部修改
	alertsCopy := make([]DLPAlert, len(m.alerts))
	copy(alertsCopy, m.alerts)
	
	return alertsCopy
}

// ClearAlerts 清除所有警报
func (m *AlertManager) ClearAlerts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.alerts = make([]DLPAlert, 0)
}

// AlertToMap 将警报转换为map
func AlertToMap(alert DLPAlert) map[string]interface{} {
	return map[string]interface{}{
		"rule_id":      alert.RuleID,
		"rule_name":    alert.RuleName,
		"content":      alert.Content,
		"source":       alert.Source,
		"destination":  alert.Destination,
		"action":       alert.Action,
		"timestamp":    alert.Timestamp.Format(time.RFC3339),
	}
}

// AlertsToMap 将警报列表转换为map列表
func AlertsToMap(alerts []DLPAlert) []map[string]interface{} {
	result := make([]map[string]interface{}, len(alerts))
	for i, alert := range alerts {
		result[i] = AlertToMap(alert)
	}
	return result
}
