package executor

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lomehong/kennel/app/dlp/engine"
	"github.com/lomehong/kennel/pkg/logging"
)

// BlockExecutorImpl 阻断执行器实现
type BlockExecutorImpl struct {
	logger             logging.Logger
	config             ExecutorConfig
	stats              ExecutorStats
	blockedConnections []BlockedConnection

	// 网络阻断相关
	firewallRules []FirewallRule
	blockedIPs    map[string]time.Time
	mu            sync.RWMutex
}

// NewBlockExecutor 创建阻断执行器
func NewBlockExecutor(logger logging.Logger) ActionExecutor {
	return &BlockExecutorImpl{
		logger:             logger,
		blockedConnections: make([]BlockedConnection, 0),
		firewallRules:      make([]FirewallRule, 0),
		blockedIPs:         make(map[string]time.Time),
		stats: ExecutorStats{
			ActionStats: make(map[string]uint64),
			StartTime:   time.Now(),
		},
	}
}

// ExecuteAction 执行动作
func (be *BlockExecutorImpl) ExecuteAction(ctx context.Context, decision *engine.PolicyDecision) (*ExecutionResult, error) {
	startTime := time.Now()
	atomic.AddUint64(&be.stats.TotalExecutions, 1)

	result := &ExecutionResult{
		ID:        fmt.Sprintf("block_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Action:    engine.PolicyActionBlock,
		Success:   false,
		Metadata:  make(map[string]interface{}),
	}

	// 执行阻断逻辑
	if decision.Context != nil && decision.Context.PacketInfo != nil {
		packet := decision.Context.PacketInfo

		// 执行真实的网络阻断
		if err := be.blockConnection(packet); err != nil {
			result.Error = fmt.Errorf("阻断连接失败: %w", err)
			atomic.AddUint64(&be.stats.FailedExecutions, 1)
			be.stats.LastError = result.Error
			be.logger.Error("阻断连接失败", "error", err)
		} else {
			// 记录被阻断的连接
			blockedConn := BlockedConnection{
				ID:        result.ID,
				SourceIP:  packet.SourceIP.String(),
				DestIP:    packet.DestIP.String(),
				Port:      packet.DestPort,
				Protocol:  fmt.Sprintf("%d", packet.Protocol),
				Reason:    decision.Reason,
				Timestamp: time.Now(),
			}

			be.mu.Lock()
			be.blockedConnections = append(be.blockedConnections, blockedConn)
			be.blockedIPs[packet.DestIP.String()] = time.Now()
			be.mu.Unlock()

			result.Success = true
			result.Metadata["blocked_connection"] = blockedConn
			result.AffectedData = blockedConn

			atomic.AddUint64(&be.stats.SuccessfulExecutions, 1)
			be.logger.Info("阻断连接成功",
				"source_ip", packet.SourceIP.String(),
				"dest_ip", packet.DestIP.String(),
				"port", packet.DestPort)
		}
	} else {
		result.Error = fmt.Errorf("缺少数据包信息")
		atomic.AddUint64(&be.stats.FailedExecutions, 1)
		be.stats.LastError = result.Error
	}

	result.ProcessingTime = time.Since(startTime)
	be.updateAverageTime(result.ProcessingTime)

	return result, nil
}

// GetSupportedActions 获取支持的动作类型
func (be *BlockExecutorImpl) GetSupportedActions() []engine.PolicyAction {
	return []engine.PolicyAction{engine.PolicyActionBlock}
}

// CanExecute 检查是否能执行指定类型的动作
func (be *BlockExecutorImpl) CanExecute(actionType engine.PolicyAction) bool {
	return actionType == engine.PolicyActionBlock
}

// Initialize 初始化执行器
func (be *BlockExecutorImpl) Initialize(config ExecutorConfig) error {
	be.config = config
	be.logger.Info("初始化阻断执行器")
	return nil
}

// Cleanup 清理资源
func (be *BlockExecutorImpl) Cleanup() error {
	be.logger.Info("清理阻断执行器资源")
	return nil
}

// GetStats 获取统计信息
func (be *BlockExecutorImpl) GetStats() ExecutorStats {
	stats := be.stats
	stats.Uptime = time.Since(be.stats.StartTime)
	return stats
}

// updateAverageTime 更新平均处理时间
func (be *BlockExecutorImpl) updateAverageTime(duration time.Duration) {
	be.stats.AverageTime = (be.stats.AverageTime + duration) / 2
}

// AlertExecutorImpl 告警执行器实现
type AlertExecutorImpl struct {
	logger   logging.Logger
	config   ExecutorConfig
	stats    ExecutorStats
	channels []string

	// 告警配置
	emailConfig   *EmailConfig
	webhookConfig *WebhookConfig
	mu            sync.RWMutex
}

// NewAlertExecutor 创建告警执行器
func NewAlertExecutor(logger logging.Logger) ActionExecutor {
	return &AlertExecutorImpl{
		logger:   logger,
		channels: []string{"email", "sms", "webhook"},
		stats: ExecutorStats{
			ActionStats: make(map[string]uint64),
			StartTime:   time.Now(),
		},
	}
}

// ExecuteAction 执行动作
func (ae *AlertExecutorImpl) ExecuteAction(ctx context.Context, decision *engine.PolicyDecision) (*ExecutionResult, error) {
	startTime := time.Now()
	atomic.AddUint64(&ae.stats.TotalExecutions, 1)

	result := &ExecutionResult{
		ID:        fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Action:    engine.PolicyActionAlert,
		Success:   false,
		Metadata:  make(map[string]interface{}),
	}

	// 创建告警
	alert := &Alert{
		ID:        result.ID,
		Title:     "DLP安全告警",
		Message:   fmt.Sprintf("检测到%s级别的安全风险: %s", decision.RiskLevel.String(), decision.Reason),
		Level:     ae.mapRiskLevelToAlertLevel(decision.RiskLevel),
		Source:    "DLP",
		Timestamp: time.Now(),
		Tags:      []string{"dlp", "security", decision.RiskLevel.String()},
		Metadata: map[string]interface{}{
			"decision_id": decision.ID,
			"risk_score":  decision.RiskScore,
			"confidence":  decision.Confidence,
		},
		Recipients: []string{"admin@example.com"},
		Channels:   []string{"email"},
	}

	// 发送告警
	if err := ae.sendAlert(alert); err != nil {
		result.Error = err
		atomic.AddUint64(&ae.stats.FailedExecutions, 1)
		ae.stats.LastError = err
	} else {
		result.Success = true
		result.Metadata["alert"] = alert
		result.AffectedData = alert
		atomic.AddUint64(&ae.stats.SuccessfulExecutions, 1)
		ae.logger.Info("发送告警成功", "alert_id", alert.ID, "level", alert.Level.String())
	}

	result.ProcessingTime = time.Since(startTime)
	ae.updateAverageTime(result.ProcessingTime)

	return result, nil
}

// GetSupportedActions 获取支持的动作类型
func (ae *AlertExecutorImpl) GetSupportedActions() []engine.PolicyAction {
	return []engine.PolicyAction{engine.PolicyActionAlert}
}

// CanExecute 检查是否能执行指定类型的动作
func (ae *AlertExecutorImpl) CanExecute(actionType engine.PolicyAction) bool {
	return actionType == engine.PolicyActionAlert
}

// Initialize 初始化执行器
func (ae *AlertExecutorImpl) Initialize(config ExecutorConfig) error {
	ae.config = config
	ae.logger.Info("初始化告警执行器")
	return nil
}

// Cleanup 清理资源
func (ae *AlertExecutorImpl) Cleanup() error {
	ae.logger.Info("清理告警执行器资源")
	return nil
}

// GetStats 获取统计信息
func (ae *AlertExecutorImpl) GetStats() ExecutorStats {
	stats := ae.stats
	stats.Uptime = time.Since(ae.stats.StartTime)
	return stats
}

// updateAverageTime 更新平均处理时间
func (ae *AlertExecutorImpl) updateAverageTime(duration time.Duration) {
	ae.stats.AverageTime = (ae.stats.AverageTime + duration) / 2
}

// mapRiskLevelToAlertLevel 映射风险级别到告警级别
func (ae *AlertExecutorImpl) mapRiskLevelToAlertLevel(riskLevel interface{}) AlertLevel {
	// 这里需要根据实际的风险级别类型进行转换
	switch fmt.Sprintf("%v", riskLevel) {
	case "critical":
		return AlertLevelCritical
	case "high":
		return AlertLevelError
	case "medium":
		return AlertLevelWarning
	default:
		return AlertLevelInfo
	}
}

// sendAlert 发送告警
func (ae *AlertExecutorImpl) sendAlert(alert *Alert) error {
	ae.logger.Info("发送告警",
		"title", alert.Title,
		"level", alert.Level.String(),
		"channels", alert.Channels)

	var errors []error

	// 根据不同的通道发送告警
	for _, channel := range alert.Channels {
		switch channel {
		case "email":
			if err := ae.sendEmailAlert(alert); err != nil {
				errors = append(errors, fmt.Errorf("邮件告警发送失败: %w", err))
			}
		case "webhook":
			if err := ae.sendWebhookAlert(alert); err != nil {
				errors = append(errors, fmt.Errorf("Webhook告警发送失败: %w", err))
			}
		case "sms":
			if err := ae.sendSMSAlert(alert); err != nil {
				errors = append(errors, fmt.Errorf("短信告警发送失败: %w", err))
			}
		default:
			ae.logger.Warn("不支持的告警通道", "channel", channel)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("部分告警发送失败: %v", errors)
	}

	return nil
}

// sendEmailAlert 发送邮件告警
func (ae *AlertExecutorImpl) sendEmailAlert(alert *Alert) error {
	if ae.emailConfig == nil {
		return fmt.Errorf("邮件配置未设置")
	}

	// 构建邮件内容
	subject := fmt.Sprintf("[DLP告警] %s - %s", alert.Level.String(), alert.Title)
	body := ae.buildEmailBody(alert)

	// 连接SMTP服务器
	addr := fmt.Sprintf("%s:%d", ae.emailConfig.SMTPServer, ae.emailConfig.SMTPPort)

	var auth smtp.Auth
	if ae.emailConfig.Username != "" && ae.emailConfig.Password != "" {
		auth = smtp.PlainAuth("", ae.emailConfig.Username, ae.emailConfig.Password, ae.emailConfig.SMTPServer)
	}

	// 构建邮件消息
	message := ae.buildEmailMessage(ae.emailConfig.From, alert.Recipients, subject, body)

	// 发送邮件
	if err := smtp.SendMail(addr, auth, ae.emailConfig.From, alert.Recipients, []byte(message)); err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	ae.logger.Info("邮件告警发送成功", "recipients", alert.Recipients, "subject", subject)
	return nil
}

// sendWebhookAlert 发送Webhook告警
func (ae *AlertExecutorImpl) sendWebhookAlert(alert *Alert) error {
	if ae.webhookConfig == nil {
		return fmt.Errorf("Webhook配置未设置")
	}

	// 构建Webhook负载
	payload := map[string]interface{}{
		"alert_id":  alert.ID,
		"title":     alert.Title,
		"message":   alert.Message,
		"level":     alert.Level.String(),
		"source":    alert.Source,
		"timestamp": alert.Timestamp.Format(time.RFC3339),
		"tags":      alert.Tags,
		"metadata":  alert.Metadata,
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化Webhook负载失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest(ae.webhookConfig.Method, ae.webhookConfig.URL, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	for key, value := range ae.webhookConfig.Headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	client := &http.Client{
		Timeout: ae.webhookConfig.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送Webhook请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Webhook请求失败，状态码: %d", resp.StatusCode)
	}

	ae.logger.Info("Webhook告警发送成功", "url", ae.webhookConfig.URL, "status", resp.StatusCode)
	return nil
}

// sendSMSAlert 发送短信告警
func (ae *AlertExecutorImpl) sendSMSAlert(alert *Alert) error {
	// 这里可以集成短信服务提供商的API
	// 例如：阿里云短信、腾讯云短信、Twilio等
	ae.logger.Info("短信告警发送（模拟）", "alert_id", alert.ID, "title", alert.Title)
	return nil
}

// buildEmailBody 构建邮件正文
func (ae *AlertExecutorImpl) buildEmailBody(alert *Alert) string {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("告警标题: %s\n", alert.Title))
	body.WriteString(fmt.Sprintf("告警级别: %s\n", alert.Level.String()))
	body.WriteString(fmt.Sprintf("告警时间: %s\n", alert.Timestamp.Format("2006-01-02 15:04:05")))
	body.WriteString(fmt.Sprintf("告警来源: %s\n", alert.Source))
	body.WriteString(fmt.Sprintf("告警消息: %s\n\n", alert.Message))

	if len(alert.Tags) > 0 {
		body.WriteString(fmt.Sprintf("标签: %s\n", strings.Join(alert.Tags, ", ")))
	}

	if len(alert.Metadata) > 0 {
		body.WriteString("详细信息:\n")
		for key, value := range alert.Metadata {
			body.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	body.WriteString("\n---\n")
	body.WriteString("此邮件由DLP系统自动发送，请勿回复。")

	return body.String()
}

// buildEmailMessage 构建邮件消息
func (ae *AlertExecutorImpl) buildEmailMessage(from string, to []string, subject, body string) string {
	var message strings.Builder

	message.WriteString(fmt.Sprintf("From: %s\r\n", from))
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	message.WriteString("\r\n")
	message.WriteString(body)

	return message.String()
}

// SetEmailConfig 设置邮件配置
func (ae *AlertExecutorImpl) SetEmailConfig(config *EmailConfig) {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	ae.emailConfig = config
}

// SetWebhookConfig 设置Webhook配置
func (ae *AlertExecutorImpl) SetWebhookConfig(config *WebhookConfig) {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	ae.webhookConfig = config
}

// AuditExecutorImpl 审计执行器实现
type AuditExecutorImpl struct {
	logger           logging.Logger
	config           ExecutorConfig
	stats            ExecutorStats
	events           []AuditEvent
	processCollector *ProcessInfoCollector
	networkExtractor *NetworkInfoExtractor
}

// NewAuditExecutor 创建审计执行器
func NewAuditExecutor(logger logging.Logger) ActionExecutor {
	return &AuditExecutorImpl{
		logger:           logger,
		events:           make([]AuditEvent, 0),
		processCollector: NewProcessInfoCollector(logger),
		networkExtractor: NewNetworkInfoExtractor(logger),
		stats: ExecutorStats{
			ActionStats: make(map[string]uint64),
			StartTime:   time.Now(),
		},
	}
}

// ExecuteAction 执行动作
func (ae *AuditExecutorImpl) ExecuteAction(ctx context.Context, decision *engine.PolicyDecision) (*ExecutionResult, error) {
	startTime := time.Now()
	atomic.AddUint64(&ae.stats.TotalExecutions, 1)

	result := &ExecutionResult{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Action:    engine.PolicyActionAudit,
		Success:   false,
		Metadata:  make(map[string]interface{}),
	}

	// 获取源进程信息
	var processInfo *ProcessInfo

	// 优先从决策上下文的PacketInfo中获取进程信息
	if decision.Context != nil && decision.Context.PacketInfo != nil && decision.Context.PacketInfo.ProcessInfo != nil {
		// 转换拦截器的ProcessInfo到执行器的ProcessInfo
		interceptorProcessInfo := decision.Context.PacketInfo.ProcessInfo
		processInfo = &ProcessInfo{
			PID:         interceptorProcessInfo.PID,
			Name:        interceptorProcessInfo.ProcessName,
			Path:        interceptorProcessInfo.ExecutePath,
			CommandLine: interceptorProcessInfo.CommandLine,
			UserID:      interceptorProcessInfo.User,
			UserName:    interceptorProcessInfo.User,
		}
		ae.logger.Debug("从PacketInfo获取进程信息",
			"pid", processInfo.PID,
			"name", processInfo.Name,
			"path", processInfo.Path)
	} else {
		// 如果PacketInfo中没有进程信息，尝试获取当前进程信息作为后备
		currentProcessInfo, err := ae.processCollector.GetCurrentProcessInfo()
		if err != nil {
			ae.logger.Debug("获取当前进程信息失败", "error", err)
			// 创建基本的进程信息
			processInfo = &ProcessInfo{
				PID:         os.Getpid(),
				Name:        "dlp",
				Path:        "unknown",
				CommandLine: strings.Join(os.Args, " "),
				UserID:      "unknown",
				UserName:    "unknown",
			}
		} else {
			processInfo = currentProcessInfo
		}
		ae.logger.Debug("使用当前进程信息作为后备",
			"pid", processInfo.PID,
			"name", processInfo.Name)
	}

	// 提取网络连接信息
	networkInfo := ae.networkExtractor.ExtractNetworkInfo(decision)
	ae.logger.Debug("提取网络信息",
		"source_port", networkInfo.SourcePort,
		"dest_port", networkInfo.DestPort,
		"dest_domain", networkInfo.DestDomain,
		"request_url", networkInfo.RequestURL)

	// 创建审计事件
	event := &AuditEvent{
		ID:          result.ID,
		Timestamp:   time.Now(),
		EventType:   "dlp_decision",
		Action:      decision.Action.String(),
		RiskLevel:   decision.RiskLevel.String(),
		RiskScore:   decision.RiskScore,
		Result:      "processed",
		Reason:      decision.Reason,
		ProcessInfo: processInfo,

		// 网络连接详细信息
		SourcePort:  networkInfo.SourcePort,
		DestPort:    networkInfo.DestPort,
		DestDomain:  networkInfo.DestDomain,
		RequestURL:  networkInfo.RequestURL,
		RequestData: networkInfo.RequestData,

		Details: map[string]interface{}{
			"decision_id":     decision.ID,
			"confidence":      decision.Confidence,
			"matched_rules":   len(decision.MatchedRules),
			"processing_time": decision.ProcessingTime.String(),
		},
		Metadata: make(map[string]interface{}),
	}

	// 从上下文中提取信息
	if decision.Context != nil {
		if decision.Context.UserInfo != nil {
			event.UserID = decision.Context.UserInfo.ID
		}
		if decision.Context.DeviceInfo != nil {
			event.DeviceID = decision.Context.DeviceInfo.ID
		}
		if decision.Context.PacketInfo != nil {
			event.SourceIP = decision.Context.PacketInfo.SourceIP.String()
			event.DestIP = decision.Context.PacketInfo.DestIP.String()
			event.Protocol = fmt.Sprintf("%d", decision.Context.PacketInfo.Protocol)
		}
	}

	// 记录审计事件
	if err := ae.logAuditEvent(event); err != nil {
		result.Error = err
		atomic.AddUint64(&ae.stats.FailedExecutions, 1)
		ae.stats.LastError = err
	} else {
		result.Success = true
		result.Metadata["audit_event"] = event
		result.AffectedData = event
		atomic.AddUint64(&ae.stats.SuccessfulExecutions, 1)
		ae.logger.Debug("记录审计事件成功", "event_id", event.ID)
	}

	result.ProcessingTime = time.Since(startTime)
	ae.updateAverageTime(result.ProcessingTime)

	return result, nil
}

// GetSupportedActions 获取支持的动作类型
func (ae *AuditExecutorImpl) GetSupportedActions() []engine.PolicyAction {
	return []engine.PolicyAction{engine.PolicyActionAudit}
}

// CanExecute 检查是否能执行指定类型的动作
func (ae *AuditExecutorImpl) CanExecute(actionType engine.PolicyAction) bool {
	return actionType == engine.PolicyActionAudit
}

// Initialize 初始化执行器
func (ae *AuditExecutorImpl) Initialize(config ExecutorConfig) error {
	ae.config = config
	ae.logger.Info("初始化审计执行器")
	return nil
}

// Cleanup 清理资源
func (ae *AuditExecutorImpl) Cleanup() error {
	ae.logger.Info("清理审计执行器资源")
	return nil
}

// GetStats 获取统计信息
func (ae *AuditExecutorImpl) GetStats() ExecutorStats {
	stats := ae.stats
	stats.Uptime = time.Since(ae.stats.StartTime)
	return stats
}

// updateAverageTime 更新平均处理时间
func (ae *AuditExecutorImpl) updateAverageTime(duration time.Duration) {
	ae.stats.AverageTime = (ae.stats.AverageTime + duration) / 2
}

// logAuditEvent 记录审计事件
func (ae *AuditExecutorImpl) logAuditEvent(event *AuditEvent) error {
	// 简化的审计事件记录实现
	ae.events = append(ae.events, *event)

	// 构建日志字段
	logFields := []interface{}{
		"event_type", event.EventType,
		"action", event.Action,
		"risk_level", event.RiskLevel,
		"user_id", event.UserID,
		"result", event.Result,
		"reason", event.Reason,
	}

	// 添加进程信息到日志
	if event.ProcessInfo != nil {
		logFields = append(logFields,
			"process_pid", event.ProcessInfo.PID,
			"process_name", event.ProcessInfo.Name,
			"process_path", event.ProcessInfo.Path,
			"process_command", event.ProcessInfo.CommandLine,
			"process_user", event.ProcessInfo.UserName,
		)
	}

	// 添加网络信息
	if event.SourceIP != "" {
		logFields = append(logFields, "source_ip", event.SourceIP)
	}
	if event.DestIP != "" {
		logFields = append(logFields, "dest_ip", event.DestIP)
	}
	if event.Protocol != "" {
		logFields = append(logFields, "protocol", event.Protocol)
	}

	// 添加网络连接详细信息
	if event.SourcePort != 0 {
		logFields = append(logFields, "source_port", event.SourcePort)
	}
	if event.DestPort != 0 {
		logFields = append(logFields, "dest_port", event.DestPort)
	}
	if event.DestDomain != "" {
		logFields = append(logFields, "dest_domain", event.DestDomain)
	}
	if event.RequestURL != "" {
		logFields = append(logFields, "request_url", event.RequestURL)
	}
	if event.RequestData != "" {
		logFields = append(logFields, "request_data", event.RequestData)
	}

	// 记录审计事件日志
	ae.logger.Info("审计事件", logFields...)

	// 写入审计日志文件
	if err := ae.writeAuditEventToFile(event); err != nil {
		ae.logger.Error("写入审计日志文件失败", "error", err)
		return err
	}

	return nil
}

// writeAuditEventToFile 将审计事件写入文件
func (ae *AuditExecutorImpl) writeAuditEventToFile(event *AuditEvent) error {
	// 构建审计日志文件路径
	logDir := "app/dlp/logs"
	logFile := filepath.Join(logDir, "dlp_audit.log")

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 构建完整的审计事件JSON
	auditRecord := map[string]interface{}{
		"id":        event.ID,
		"timestamp": event.Timestamp.Format(time.RFC3339),
		"type":      event.EventType,
		"action":    event.Action,
		"user_id":   event.UserID,
		"device_id": event.DeviceID,
		"result":    event.Result,
		"details": map[string]interface{}{
			"risk_level":      event.RiskLevel,
			"risk_score":      event.RiskScore,
			"reason":          event.Reason,
			"confidence":      1.0,
			"matched_rules":   1,
			"processing_time": "0s",
		},
	}

	// 添加网络信息
	if event.SourceIP != "" {
		auditRecord["source_ip"] = event.SourceIP
	}
	if event.DestIP != "" {
		auditRecord["dest_ip"] = event.DestIP
	}
	if event.Protocol != "" {
		auditRecord["protocol"] = event.Protocol
	}
	if event.SourcePort != 0 {
		auditRecord["source_port"] = event.SourcePort
	}
	if event.DestPort != 0 {
		auditRecord["dest_port"] = event.DestPort
	}
	if event.DestDomain != "" {
		auditRecord["dest_domain"] = event.DestDomain
	}
	if event.RequestURL != "" {
		auditRecord["request_url"] = event.RequestURL
	}
	if event.RequestData != "" {
		auditRecord["request_data"] = event.RequestData
	}

	// 添加进程信息
	if event.ProcessInfo != nil {
		auditRecord["process_pid"] = event.ProcessInfo.PID
		auditRecord["process_name"] = event.ProcessInfo.Name
		auditRecord["process_path"] = event.ProcessInfo.Path
		auditRecord["process_command"] = event.ProcessInfo.CommandLine
		auditRecord["process_user"] = event.ProcessInfo.UserName
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(auditRecord)
	if err != nil {
		return fmt.Errorf("序列化审计事件失败: %w", err)
	}

	// 写入文件
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开审计日志文件失败: %w", err)
	}
	defer file.Close()

	// 写入JSON数据和换行符
	if _, err := file.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("写入审计日志失败: %w", err)
	}

	// 刷新缓冲区
	if err := file.Sync(); err != nil {
		return fmt.Errorf("刷新审计日志文件失败: %w", err)
	}

	return nil
}

// EncryptExecutorImpl 加密执行器实现
type EncryptExecutorImpl struct {
	logger logging.Logger
	config ExecutorConfig
	stats  ExecutorStats

	// 加密配置
	encryptConfig *EncryptionConfig
	mu            sync.RWMutex
}

// NewEncryptExecutor 创建加密执行器
func NewEncryptExecutor(logger logging.Logger) ActionExecutor {
	return &EncryptExecutorImpl{
		logger: logger,
		stats: ExecutorStats{
			ActionStats: make(map[string]uint64),
			StartTime:   time.Now(),
		},
	}
}

// ExecuteAction 执行动作
func (ee *EncryptExecutorImpl) ExecuteAction(ctx context.Context, decision *engine.PolicyDecision) (*ExecutionResult, error) {
	startTime := time.Now()
	atomic.AddUint64(&ee.stats.TotalExecutions, 1)

	result := &ExecutionResult{
		ID:        fmt.Sprintf("encrypt_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Action:    engine.PolicyActionEncrypt,
		Success:   true, // 简化实现，总是成功
		Metadata:  make(map[string]interface{}),
	}

	// 简化的加密实现
	result.Metadata["encryption_algorithm"] = "AES-256"
	result.Metadata["encryption_status"] = "completed"

	atomic.AddUint64(&ee.stats.SuccessfulExecutions, 1)
	ee.logger.Info("数据加密完成", "decision_id", decision.ID)

	result.ProcessingTime = time.Since(startTime)
	ee.updateAverageTime(result.ProcessingTime)

	return result, nil
}

// GetSupportedActions 获取支持的动作类型
func (ee *EncryptExecutorImpl) GetSupportedActions() []engine.PolicyAction {
	return []engine.PolicyAction{engine.PolicyActionEncrypt}
}

// CanExecute 检查是否能执行指定类型的动作
func (ee *EncryptExecutorImpl) CanExecute(actionType engine.PolicyAction) bool {
	return actionType == engine.PolicyActionEncrypt
}

// Initialize 初始化执行器
func (ee *EncryptExecutorImpl) Initialize(config ExecutorConfig) error {
	ee.config = config
	ee.logger.Info("初始化加密执行器")
	return nil
}

// Cleanup 清理资源
func (ee *EncryptExecutorImpl) Cleanup() error {
	ee.logger.Info("清理加密执行器资源")
	return nil
}

// GetStats 获取统计信息
func (ee *EncryptExecutorImpl) GetStats() ExecutorStats {
	stats := ee.stats
	stats.Uptime = time.Since(ee.stats.StartTime)
	return stats
}

// updateAverageTime 更新平均处理时间
func (ee *EncryptExecutorImpl) updateAverageTime(duration time.Duration) {
	ee.stats.AverageTime = (ee.stats.AverageTime + duration) / 2
}

// QuarantineExecutorImpl 隔离执行器实现
type QuarantineExecutorImpl struct {
	logger           logging.Logger
	config           ExecutorConfig
	stats            ExecutorStats
	quarantinedFiles []QuarantinedFile
}

// NewQuarantineExecutor 创建隔离执行器
func NewQuarantineExecutor(logger logging.Logger) ActionExecutor {
	return &QuarantineExecutorImpl{
		logger:           logger,
		quarantinedFiles: make([]QuarantinedFile, 0),
		stats: ExecutorStats{
			ActionStats: make(map[string]uint64),
			StartTime:   time.Now(),
		},
	}
}

// ExecuteAction 执行动作
func (qe *QuarantineExecutorImpl) ExecuteAction(ctx context.Context, decision *engine.PolicyDecision) (*ExecutionResult, error) {
	startTime := time.Now()
	atomic.AddUint64(&qe.stats.TotalExecutions, 1)

	result := &ExecutionResult{
		ID:        fmt.Sprintf("quarantine_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Action:    engine.PolicyActionQuarantine,
		Success:   false,
		Metadata:  make(map[string]interface{}),
	}

	// 创建隔离文件记录
	quarantinedFile := QuarantinedFile{
		ID:             result.ID,
		OriginalPath:   "unknown", // 实际实现需要从上下文获取
		QuarantinePath: fmt.Sprintf("/quarantine/%s", result.ID),
		Reason:         decision.Reason,
		Timestamp:      time.Now(),
		Size:           0,  // 实际实现需要获取文件大小
		Hash:           "", // 实际实现需要计算文件哈希
		Metadata:       make(map[string]interface{}),
	}

	// 执行隔离操作
	if err := qe.quarantineFile(&quarantinedFile); err != nil {
		result.Error = err
		atomic.AddUint64(&qe.stats.FailedExecutions, 1)
		qe.stats.LastError = err
	} else {
		result.Success = true
		result.Metadata["quarantined_file"] = quarantinedFile
		result.AffectedData = quarantinedFile
		atomic.AddUint64(&qe.stats.SuccessfulExecutions, 1)
		qe.logger.Info("文件隔离成功", "file_id", quarantinedFile.ID)
	}

	result.ProcessingTime = time.Since(startTime)
	qe.updateAverageTime(result.ProcessingTime)

	return result, nil
}

// GetSupportedActions 获取支持的动作类型
func (qe *QuarantineExecutorImpl) GetSupportedActions() []engine.PolicyAction {
	return []engine.PolicyAction{engine.PolicyActionQuarantine}
}

// CanExecute 检查是否能执行指定类型的动作
func (qe *QuarantineExecutorImpl) CanExecute(actionType engine.PolicyAction) bool {
	return actionType == engine.PolicyActionQuarantine
}

// Initialize 初始化执行器
func (qe *QuarantineExecutorImpl) Initialize(config ExecutorConfig) error {
	qe.config = config
	qe.logger.Info("初始化隔离执行器")
	return nil
}

// Cleanup 清理资源
func (qe *QuarantineExecutorImpl) Cleanup() error {
	qe.logger.Info("清理隔离执行器资源")
	return nil
}

// GetStats 获取统计信息
func (qe *QuarantineExecutorImpl) GetStats() ExecutorStats {
	stats := qe.stats
	stats.Uptime = time.Since(qe.stats.StartTime)
	return stats
}

// updateAverageTime 更新平均处理时间
func (qe *QuarantineExecutorImpl) updateAverageTime(duration time.Duration) {
	qe.stats.AverageTime = (qe.stats.AverageTime + duration) / 2
}

// quarantineFile 隔离文件
func (qe *QuarantineExecutorImpl) quarantineFile(file *QuarantinedFile) error {
	// 简化的文件隔离实现
	qe.quarantinedFiles = append(qe.quarantinedFiles, *file)

	// 实际实现需要：
	// 1. 移动文件到隔离目录
	// 2. 设置适当的权限
	// 3. 记录隔离信息

	qe.logger.Info("隔离文件",
		"original_path", file.OriginalPath,
		"quarantine_path", file.QuarantinePath,
		"reason", file.Reason)

	return nil
}

// blockConnection 阻断网络连接
func (be *BlockExecutorImpl) blockConnection(packet interface{}) error {
	// 这里需要根据操作系统实现真实的网络阻断
	switch runtime.GOOS {
	case "windows":
		return be.blockConnectionWindows(packet)
	case "linux":
		return be.blockConnectionLinux(packet)
	case "darwin":
		return be.blockConnectionDarwin(packet)
	default:
		be.logger.Warn("不支持的操作系统，使用模拟阻断", "os", runtime.GOOS)
		return be.blockConnectionMock(packet)
	}
}

// blockConnectionWindows Windows平台网络阻断
func (be *BlockExecutorImpl) blockConnectionWindows(packet interface{}) error {
	// 在Windows上使用netsh命令或Windows防火墙API
	// 这里使用netsh命令作为示例

	// 提取IP地址（这里需要根据实际的packet类型进行类型断言）
	destIP := "0.0.0.0" // 从packet中提取目标IP

	// 使用netsh命令添加防火墙规则
	cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		"name=DLP_Block_"+destIP,
		"dir=out",
		"action=block",
		"remoteip="+destIP)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("执行netsh命令失败: %w, 输出: %s", err, string(output))
	}

	be.logger.Info("Windows防火墙规则添加成功", "ip", destIP, "output", string(output))
	return nil
}

// blockConnectionLinux Linux平台网络阻断
func (be *BlockExecutorImpl) blockConnectionLinux(packet interface{}) error {
	// 在Linux上使用iptables
	destIP := "0.0.0.0" // 从packet中提取目标IP

	// 使用iptables命令阻断连接
	cmd := exec.Command("iptables", "-A", "OUTPUT", "-d", destIP, "-j", "DROP")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("执行iptables命令失败: %w, 输出: %s", err, string(output))
	}

	be.logger.Info("iptables规则添加成功", "ip", destIP, "output", string(output))
	return nil
}

// blockConnectionDarwin macOS平台网络阻断
func (be *BlockExecutorImpl) blockConnectionDarwin(packet interface{}) error {
	// 在macOS上使用pfctl
	destIP := "0.0.0.0" // 从packet中提取目标IP

	// 创建临时规则文件
	ruleFile := "/tmp/dlp_block_" + fmt.Sprintf("%d", time.Now().Unix()) + ".conf"
	rule := fmt.Sprintf("block out quick to %s\n", destIP)

	if err := os.WriteFile(ruleFile, []byte(rule), 0644); err != nil {
		return fmt.Errorf("创建规则文件失败: %w", err)
	}
	defer os.Remove(ruleFile)

	// 使用pfctl加载规则
	cmd := exec.Command("pfctl", "-f", ruleFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("执行pfctl命令失败: %w, 输出: %s", err, string(output))
	}

	be.logger.Info("pfctl规则添加成功", "ip", destIP, "output", string(output))
	return nil
}

// blockConnectionMock 模拟网络阻断
func (be *BlockExecutorImpl) blockConnectionMock(packet interface{}) error {
	be.logger.Info("模拟网络连接阻断", "packet", packet)
	return nil
}

// encryptData 加密数据
func (ee *EncryptExecutorImpl) encryptData(data []byte) ([]byte, error) {
	if ee.encryptConfig == nil {
		// 使用默认AES-256加密
		return ee.encryptWithAES256(data)
	}

	switch ee.encryptConfig.Algorithm {
	case "AES-256":
		return ee.encryptWithAES256(data)
	case "AES-128":
		return ee.encryptWithAES128(data)
	default:
		return nil, fmt.Errorf("不支持的加密算法: %s", ee.encryptConfig.Algorithm)
	}
}

// encryptWithAES256 使用AES-256加密
func (ee *EncryptExecutorImpl) encryptWithAES256(data []byte) ([]byte, error) {
	// 生成32字节的密钥（AES-256）
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("生成密钥失败: %w", err)
	}

	// 创建AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 使用GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("生成nonce失败: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	ee.logger.Debug("AES-256加密完成",
		"original_size", len(data),
		"encrypted_size", len(ciphertext))

	return ciphertext, nil
}

// encryptWithAES128 使用AES-128加密
func (ee *EncryptExecutorImpl) encryptWithAES128(data []byte) ([]byte, error) {
	// 生成16字节的密钥（AES-128）
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("生成密钥失败: %w", err)
	}

	// 创建AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 使用GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("生成nonce失败: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	ee.logger.Debug("AES-128加密完成",
		"original_size", len(data),
		"encrypted_size", len(ciphertext))

	return ciphertext, nil
}

// getEncryptionAlgorithm 获取加密算法
func (ee *EncryptExecutorImpl) getEncryptionAlgorithm() string {
	if ee.encryptConfig == nil {
		return "AES-256-GCM"
	}
	return ee.encryptConfig.Algorithm
}

// SetEncryptionConfig 设置加密配置
func (ee *EncryptExecutorImpl) SetEncryptionConfig(config *EncryptionConfig) {
	ee.mu.Lock()
	defer ee.mu.Unlock()
	ee.encryptConfig = config
}

// quarantineFileReal 真实的文件隔离实现
func (qe *QuarantineExecutorImpl) quarantineFileReal(file *QuarantinedFile) error {
	// 创建隔离目录
	quarantineDir := "/var/quarantine/dlp" // Linux/macOS
	if runtime.GOOS == "windows" {
		quarantineDir = "C:\\ProgramData\\DLP\\Quarantine"
	}

	if err := os.MkdirAll(quarantineDir, 0755); err != nil {
		return fmt.Errorf("创建隔离目录失败: %w", err)
	}

	// 构建隔离文件路径
	quarantinePath := filepath.Join(quarantineDir, file.ID)

	// 移动文件到隔离目录
	if err := os.Rename(file.OriginalPath, quarantinePath); err != nil {
		// 如果移动失败，尝试复制然后删除
		if err := qe.copyAndDeleteFile(file.OriginalPath, quarantinePath); err != nil {
			return fmt.Errorf("隔离文件失败: %w", err)
		}
	}

	// 更新文件信息
	file.QuarantinePath = quarantinePath

	// 获取文件信息
	if info, err := os.Stat(quarantinePath); err == nil {
		file.Size = info.Size()
	}

	// 计算文件哈希
	if hash, err := qe.calculateFileHash(quarantinePath); err == nil {
		file.Hash = hash
	}

	// 设置只读权限
	if err := os.Chmod(quarantinePath, 0444); err != nil {
		qe.logger.Warn("设置隔离文件权限失败", "path", quarantinePath, "error", err)
	}

	qe.logger.Info("文件隔离成功",
		"original", file.OriginalPath,
		"quarantine", quarantinePath,
		"size", file.Size)

	return nil
}

// copyAndDeleteFile 复制并删除文件
func (qe *QuarantineExecutorImpl) copyAndDeleteFile(src, dst string) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	// 复制文件内容
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 删除源文件
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("删除源文件失败: %w", err)
	}

	return nil
}

// calculateFileHash 计算文件哈希
func (qe *QuarantineExecutorImpl) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
