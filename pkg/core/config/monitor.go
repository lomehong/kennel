package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// MonitorType 监控类型
type MonitorType string

const (
	MonitorTypeConfigChange   MonitorType = "config_change"
	MonitorTypeConfigHealth   MonitorType = "config_health"
	MonitorTypeConfigUsage    MonitorType = "config_usage"
	MonitorTypeConfigSecurity MonitorType = "config_security"
)

// MonitorLevel 监控级别
type MonitorLevel string

const (
	MonitorLevelInfo     MonitorLevel = "info"
	MonitorLevelWarning  MonitorLevel = "warning"
	MonitorLevelError    MonitorLevel = "error"
	MonitorLevelCritical MonitorLevel = "critical"
)

// MonitorEvent 监控事件
type MonitorEvent struct {
	ID         string                 `json:"id"`
	Type       MonitorType            `json:"type"`
	Level      MonitorLevel           `json:"level"`
	Component  string                 `json:"component"`
	ConfigPath string                 `json:"config_path"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details"`
	Timestamp  time.Time              `json:"timestamp"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
	Tags       []string               `json:"tags"`
}

// MonitorRule 监控规则
type MonitorRule struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	Type        MonitorType            `yaml:"type"`
	Level       MonitorLevel           `yaml:"level"`
	Component   string                 `yaml:"component"`
	Condition   string                 `yaml:"condition"`
	Threshold   map[string]interface{} `yaml:"threshold"`
	Enabled     bool                   `yaml:"enabled"`
	Description string                 `yaml:"description"`
	Tags        []string               `yaml:"tags"`
}

// MonitorMetrics 监控指标
type MonitorMetrics struct {
	ConfigChanges     int64     `json:"config_changes"`
	ConfigErrors      int64     `json:"config_errors"`
	ConfigValidations int64     `json:"config_validations"`
	HotReloads        int64     `json:"hot_reloads"`
	HotReloadFailures int64     `json:"hot_reload_failures"`
	LastConfigChange  time.Time `json:"last_config_change"`
	LastConfigError   time.Time `json:"last_config_error"`
	ConfigHealthScore float64   `json:"config_health_score"`
	ActiveAlerts      int64     `json:"active_alerts"`
	ResolvedAlerts    int64     `json:"resolved_alerts"`
}

// AlertChannel 告警通道
type AlertChannel interface {
	Send(event MonitorEvent) error
	GetType() string
	IsEnabled() bool
}

// ConfigMonitor 配置监控器
type ConfigMonitor struct {
	rules         []MonitorRule
	events        []MonitorEvent
	metrics       MonitorMetrics
	alertChannels []AlertChannel
	logger        hclog.Logger
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup

	// 监控配置
	checkInterval  time.Duration
	eventRetention time.Duration
	maxEvents      int
	enabledTypes   map[MonitorType]bool
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Enabled        bool                 `yaml:"enabled"`
	CheckInterval  time.Duration        `yaml:"check_interval"`
	EventRetention time.Duration        `yaml:"event_retention"`
	MaxEvents      int                  `yaml:"max_events"`
	EnabledTypes   map[MonitorType]bool `yaml:"enabled_types"`
	Rules          []MonitorRule        `yaml:"rules"`
	AlertChannels  []AlertChannelConfig `yaml:"alert_channels"`
}

// AlertChannelConfig 告警通道配置
type AlertChannelConfig struct {
	Type     string                 `yaml:"type"`
	Enabled  bool                   `yaml:"enabled"`
	Settings map[string]interface{} `yaml:"settings"`
}

// DefaultMonitorConfig 默认监控配置
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		Enabled:        true,
		CheckInterval:  30 * time.Second,
		EventRetention: 24 * time.Hour,
		MaxEvents:      10000,
		EnabledTypes: map[MonitorType]bool{
			MonitorTypeConfigChange:   true,
			MonitorTypeConfigHealth:   true,
			MonitorTypeConfigUsage:    true,
			MonitorTypeConfigSecurity: true,
		},
		Rules: []MonitorRule{
			{
				ID:          "config_error_rate",
				Name:        "配置错误率监控",
				Type:        MonitorTypeConfigHealth,
				Level:       MonitorLevelWarning,
				Component:   "*",
				Condition:   "error_rate > 0.1",
				Threshold:   map[string]interface{}{"error_rate": 0.1},
				Enabled:     true,
				Description: "监控配置错误率，超过10%时告警",
				Tags:        []string{"health", "error"},
			},
			{
				ID:          "hot_reload_failure",
				Name:        "热更新失败监控",
				Type:        MonitorTypeConfigChange,
				Level:       MonitorLevelError,
				Component:   "*",
				Condition:   "hot_reload_failure_rate > 0.2",
				Threshold:   map[string]interface{}{"failure_rate": 0.2},
				Enabled:     true,
				Description: "监控热更新失败率，超过20%时告警",
				Tags:        []string{"hot_reload", "failure"},
			},
		},
	}
}

// NewConfigMonitor 创建配置监控器
func NewConfigMonitor(config *MonitorConfig, logger hclog.Logger) *ConfigMonitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	monitor := &ConfigMonitor{
		rules:          config.Rules,
		events:         make([]MonitorEvent, 0),
		alertChannels:  make([]AlertChannel, 0),
		logger:         logger.Named("config-monitor"),
		ctx:            ctx,
		cancel:         cancel,
		checkInterval:  config.CheckInterval,
		eventRetention: config.EventRetention,
		maxEvents:      config.MaxEvents,
		enabledTypes:   config.EnabledTypes,
	}

	// 初始化指标
	monitor.metrics = MonitorMetrics{
		ConfigHealthScore: 100.0,
	}

	return monitor
}

// Start 启动监控
func (cm *ConfigMonitor) Start() {
	cm.logger.Info("启动配置监控")

	// 启动监控检查
	cm.wg.Add(1)
	go cm.runMonitorChecks()

	// 启动事件清理
	cm.wg.Add(1)
	go cm.runEventCleanup()
}

// Stop 停止监控
func (cm *ConfigMonitor) Stop() {
	cm.logger.Info("停止配置监控")
	cm.cancel()
	cm.wg.Wait()
}

// AddRule 添加监控规则
func (cm *ConfigMonitor) AddRule(rule MonitorRule) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.rules = append(cm.rules, rule)
	cm.logger.Info("添加监控规则", "rule_id", rule.ID, "rule_name", rule.Name)
}

// RemoveRule 移除监控规则
func (cm *ConfigMonitor) RemoveRule(ruleID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, rule := range cm.rules {
		if rule.ID == ruleID {
			cm.rules = append(cm.rules[:i], cm.rules[i+1:]...)
			cm.logger.Info("移除监控规则", "rule_id", ruleID)
			break
		}
	}
}

// AddAlertChannel 添加告警通道
func (cm *ConfigMonitor) AddAlertChannel(channel AlertChannel) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.alertChannels = append(cm.alertChannels, channel)
	cm.logger.Info("添加告警通道", "type", channel.GetType())
}

// RecordEvent 记录监控事件
func (cm *ConfigMonitor) RecordEvent(eventType MonitorType, level MonitorLevel, component, configPath, message string, details map[string]interface{}) {
	// 检查是否启用该类型的监控
	if enabled, ok := cm.enabledTypes[eventType]; !ok || !enabled {
		return
	}

	event := MonitorEvent{
		ID:         generateEventID(),
		Type:       eventType,
		Level:      level,
		Component:  component,
		ConfigPath: configPath,
		Message:    message,
		Details:    details,
		Timestamp:  time.Now(),
		Resolved:   false,
		Tags:       []string{},
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 添加事件
	cm.events = append(cm.events, event)

	// 限制事件数量
	if len(cm.events) > cm.maxEvents {
		cm.events = cm.events[len(cm.events)-cm.maxEvents:]
	}

	// 更新指标
	cm.updateMetrics(event)

	// 检查是否需要发送告警
	if level == MonitorLevelError || level == MonitorLevelCritical {
		cm.sendAlert(event)
	}

	cm.logger.Info("记录监控事件",
		"type", eventType,
		"level", level,
		"component", component,
		"message", message,
	)
}

// GetEvents 获取监控事件
func (cm *ConfigMonitor) GetEvents() []MonitorEvent {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 返回副本
	events := make([]MonitorEvent, len(cm.events))
	copy(events, cm.events)
	return events
}

// GetEventsByType 按类型获取事件
func (cm *ConfigMonitor) GetEventsByType(eventType MonitorType) []MonitorEvent {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var events []MonitorEvent
	for _, event := range cm.events {
		if event.Type == eventType {
			events = append(events, event)
		}
	}
	return events
}

// GetEventsByComponent 按组件获取事件
func (cm *ConfigMonitor) GetEventsByComponent(component string) []MonitorEvent {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var events []MonitorEvent
	for _, event := range cm.events {
		if event.Component == component {
			events = append(events, event)
		}
	}
	return events
}

// GetMetrics 获取监控指标
func (cm *ConfigMonitor) GetMetrics() MonitorMetrics {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.metrics
}

// ResolveEvent 解决事件
func (cm *ConfigMonitor) ResolveEvent(eventID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, event := range cm.events {
		if event.ID == eventID {
			now := time.Now()
			cm.events[i].Resolved = true
			cm.events[i].ResolvedAt = &now
			cm.metrics.ResolvedAlerts++
			cm.metrics.ActiveAlerts--
			cm.logger.Info("解决监控事件", "event_id", eventID)
			return nil
		}
	}

	return fmt.Errorf("未找到事件: %s", eventID)
}

// runMonitorChecks 运行监控检查
func (cm *ConfigMonitor) runMonitorChecks() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.performHealthChecks()
		}
	}
}

// runEventCleanup 运行事件清理
func (cm *ConfigMonitor) runEventCleanup() {
	defer cm.wg.Done()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.cleanupOldEvents()
		}
	}
}

// performHealthChecks 执行健康检查
func (cm *ConfigMonitor) performHealthChecks() {
	cm.mu.RLock()
	rules := cm.rules
	cm.mu.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if cm.checkRule(rule) {
			cm.RecordEvent(
				rule.Type,
				rule.Level,
				rule.Component,
				"",
				fmt.Sprintf("监控规则触发: %s", rule.Name),
				map[string]interface{}{
					"rule_id":   rule.ID,
					"condition": rule.Condition,
					"threshold": rule.Threshold,
				},
			)
		}
	}
}

// checkRule 检查监控规则
func (cm *ConfigMonitor) checkRule(rule MonitorRule) bool {
	// 这里应该实现具体的规则检查逻辑
	// 简化实现，实际应该根据规则条件和阈值进行检查
	return false
}

// updateMetrics 更新指标
func (cm *ConfigMonitor) updateMetrics(event MonitorEvent) {
	switch event.Type {
	case MonitorTypeConfigChange:
		cm.metrics.ConfigChanges++
		cm.metrics.LastConfigChange = event.Timestamp
	case MonitorTypeConfigHealth:
		if event.Level == MonitorLevelError || event.Level == MonitorLevelCritical {
			cm.metrics.ConfigErrors++
			cm.metrics.LastConfigError = event.Timestamp
		}
	}

	if event.Level == MonitorLevelError || event.Level == MonitorLevelCritical {
		cm.metrics.ActiveAlerts++
	}

	// 计算健康分数
	cm.calculateHealthScore()
}

// calculateHealthScore 计算健康分数
func (cm *ConfigMonitor) calculateHealthScore() {
	// 简化的健康分数计算
	totalEvents := len(cm.events)
	if totalEvents == 0 {
		cm.metrics.ConfigHealthScore = 100.0
		return
	}

	errorEvents := 0
	for _, event := range cm.events {
		if event.Level == MonitorLevelError || event.Level == MonitorLevelCritical {
			errorEvents++
		}
	}

	errorRate := float64(errorEvents) / float64(totalEvents)
	cm.metrics.ConfigHealthScore = (1.0 - errorRate) * 100.0
}

// sendAlert 发送告警
func (cm *ConfigMonitor) sendAlert(event MonitorEvent) {
	for _, channel := range cm.alertChannels {
		if channel.IsEnabled() {
			if err := channel.Send(event); err != nil {
				cm.logger.Error("发送告警失败",
					"channel", channel.GetType(),
					"event_id", event.ID,
					"error", err,
				)
			}
		}
	}
}

// cleanupOldEvents 清理旧事件
func (cm *ConfigMonitor) cleanupOldEvents() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cutoff := time.Now().Add(-cm.eventRetention)
	var newEvents []MonitorEvent

	for _, event := range cm.events {
		if event.Timestamp.After(cutoff) {
			newEvents = append(newEvents, event)
		}
	}

	removed := len(cm.events) - len(newEvents)
	cm.events = newEvents

	if removed > 0 {
		cm.logger.Info("清理旧监控事件", "removed", removed)
	}
}

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

// LogAlertChannel 日志告警通道
type LogAlertChannel struct {
	logger  hclog.Logger
	enabled bool
}

// NewLogAlertChannel 创建日志告警通道
func NewLogAlertChannel(logger hclog.Logger, enabled bool) *LogAlertChannel {
	return &LogAlertChannel{
		logger:  logger.Named("log-alert-channel"),
		enabled: enabled,
	}
}

// Send 发送告警
func (lac *LogAlertChannel) Send(event MonitorEvent) error {
	lac.logger.Error("配置监控告警",
		"event_id", event.ID,
		"type", event.Type,
		"level", event.Level,
		"component", event.Component,
		"message", event.Message,
		"timestamp", event.Timestamp,
	)
	return nil
}

// GetType 获取通道类型
func (lac *LogAlertChannel) GetType() string {
	return "log"
}

// IsEnabled 是否启用
func (lac *LogAlertChannel) IsEnabled() bool {
	return lac.enabled
}

// WebhookAlertChannel Webhook告警通道
type WebhookAlertChannel struct {
	url     string
	timeout time.Duration
	enabled bool
	logger  hclog.Logger
}

// NewWebhookAlertChannel 创建Webhook告警通道
func NewWebhookAlertChannel(url string, timeout time.Duration, enabled bool, logger hclog.Logger) *WebhookAlertChannel {
	return &WebhookAlertChannel{
		url:     url,
		timeout: timeout,
		enabled: enabled,
		logger:  logger.Named("webhook-alert-channel"),
	}
}

// Send 发送告警
func (wac *WebhookAlertChannel) Send(event MonitorEvent) error {
	// 这里应该实现实际的HTTP请求发送
	wac.logger.Info("发送Webhook告警",
		"url", wac.url,
		"event_id", event.ID,
		"level", event.Level,
	)
	return nil
}

// GetType 获取通道类型
func (wac *WebhookAlertChannel) GetType() string {
	return "webhook"
}

// IsEnabled 是否启用
func (wac *WebhookAlertChannel) IsEnabled() bool {
	return wac.enabled
}

// EmailAlertChannel 邮件告警通道
type EmailAlertChannel struct {
	smtpServer string
	recipients []string
	enabled    bool
	logger     hclog.Logger
}

// NewEmailAlertChannel 创建邮件告警通道
func NewEmailAlertChannel(smtpServer string, recipients []string, enabled bool, logger hclog.Logger) *EmailAlertChannel {
	return &EmailAlertChannel{
		smtpServer: smtpServer,
		recipients: recipients,
		enabled:    enabled,
		logger:     logger.Named("email-alert-channel"),
	}
}

// Send 发送告警
func (eac *EmailAlertChannel) Send(event MonitorEvent) error {
	// 这里应该实现实际的邮件发送
	eac.logger.Info("发送邮件告警",
		"smtp_server", eac.smtpServer,
		"recipients", eac.recipients,
		"event_id", event.ID,
		"level", event.Level,
	)
	return nil
}

// GetType 获取通道类型
func (eac *EmailAlertChannel) GetType() string {
	return "email"
}

// IsEnabled 是否启用
func (eac *EmailAlertChannel) IsEnabled() bool {
	return eac.enabled
}
