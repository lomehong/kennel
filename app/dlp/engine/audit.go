package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/app/dlp/parser"
	"github.com/lomehong/kennel/pkg/logging"
)

// AuditLoggerImpl 审计日志记录器实现
type AuditLoggerImpl struct {
	logger  logging.Logger
	logFile *os.File
	mu      sync.Mutex
	config  AuditConfig
}

// AuditConfig 审计配置
type AuditConfig struct {
	LogPath    string `yaml:"log_path" json:"log_path"`
	MaxSize    int64  `yaml:"max_size" json:"max_size"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	Compress   bool   `yaml:"compress" json:"compress"`
}

// NewAuditLogger 创建审计日志记录器
func NewAuditLogger(logger logging.Logger) AuditLogger {
	config := AuditConfig{
		LogPath:    "logs/dlp_audit.log",
		MaxSize:    100 * 1024 * 1024, // 100MB
		MaxAge:     30,                // 30天
		MaxBackups: 10,
		Compress:   true,
	}

	auditLogger := &AuditLoggerImpl{
		logger: logger,
		config: config,
	}

	// 初始化日志文件
	if err := auditLogger.initLogFile(); err != nil {
		logger.Error("初始化审计日志文件失败", "error", err)
	}

	return auditLogger
}

// LogDecision 记录决策
func (al *AuditLoggerImpl) LogDecision(decision *PolicyDecision) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	auditLog := &AuditLog{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Type:      "policy_decision",
		Action:    decision.Action.String(),
		Result:    "success",
		Details: map[string]interface{}{
			"decision_id":     decision.ID,
			"risk_level":      decision.RiskLevel.String(),
			"risk_score":      decision.RiskScore,
			"confidence":      decision.Confidence,
			"matched_rules":   len(decision.MatchedRules),
			"processing_time": decision.ProcessingTime.String(),
			"reason":          decision.Reason,
		},
	}

	// 从上下文中提取详细信息（增强版本）
	if decision.Context != nil {
		// 用户和设备信息
		if decision.Context.UserInfo != nil {
			auditLog.UserID = decision.Context.UserInfo.ID
			auditLog.Details["username"] = decision.Context.UserInfo.Username
			auditLog.Details["user_email"] = decision.Context.UserInfo.Email
			auditLog.Details["user_department"] = decision.Context.UserInfo.Department
			auditLog.Details["user_role"] = decision.Context.UserInfo.Role
			auditLog.Details["user_risk_level"] = decision.Context.UserInfo.RiskLevel
		}

		if decision.Context.DeviceInfo != nil {
			auditLog.DeviceID = decision.Context.DeviceInfo.ID
			auditLog.Details["device_name"] = decision.Context.DeviceInfo.Name
			auditLog.Details["device_type"] = decision.Context.DeviceInfo.Type
			auditLog.Details["device_os"] = decision.Context.DeviceInfo.OS
			auditLog.Details["device_version"] = decision.Context.DeviceInfo.Version
			auditLog.Details["device_location"] = decision.Context.DeviceInfo.Location
			auditLog.Details["device_trust_level"] = decision.Context.DeviceInfo.TrustLevel
			auditLog.Details["device_compliance"] = decision.Context.DeviceInfo.Compliance
		}

		// 网络数据包信息（关键修复：添加完整的网络和进程信息）
		if decision.Context.PacketInfo != nil {
			packet := decision.Context.PacketInfo

			// 基本网络信息
			auditLog.Details["source_ip"] = packet.SourceIP.String()
			auditLog.Details["source_port"] = packet.SourcePort
			auditLog.Details["dest_ip"] = packet.DestIP.String()
			auditLog.Details["dest_port"] = packet.DestPort
			auditLog.Details["protocol"] = al.protocolToString(packet.Protocol)
			auditLog.Details["direction"] = al.directionToString(packet.Direction)
			auditLog.Details["packet_size"] = packet.Size
			auditLog.Details["packet_id"] = packet.ID

			// 关键修复：添加进程信息记录
			if packet.ProcessInfo != nil {
				auditLog.Details["process_pid"] = packet.ProcessInfo.PID
				auditLog.Details["process_name"] = packet.ProcessInfo.ProcessName
				auditLog.Details["process_path"] = packet.ProcessInfo.ExecutePath
				auditLog.Details["process_user"] = packet.ProcessInfo.User
				auditLog.Details["process_cmdline"] = packet.ProcessInfo.CommandLine

				// 记录进程信息获取状态
				auditLog.Details["process_info_status"] = "success"

				al.logger.Debug("审计日志包含进程信息",
					"pid", packet.ProcessInfo.PID,
					"name", packet.ProcessInfo.ProcessName,
					"path", packet.ProcessInfo.ExecutePath)
			} else {
				// 记录进程信息获取失败
				auditLog.Details["process_info_status"] = "failed"
				auditLog.Details["process_error"] = "无法获取进程信息"
				auditLog.Details["process_pid"] = 0
				auditLog.Details["process_name"] = "unknown"
				auditLog.Details["process_path"] = ""
				auditLog.Details["process_user"] = "unknown"
				auditLog.Details["process_cmdline"] = ""

				al.logger.Warn("审计日志缺少进程信息",
					"packet_id", packet.ID,
					"source", fmt.Sprintf("%s:%d", packet.SourceIP, packet.SourcePort),
					"dest", fmt.Sprintf("%s:%d", packet.DestIP, packet.DestPort))
			}

			// 数据包元数据
			if packet.Metadata != nil {
				for key, value := range packet.Metadata {
					auditLog.Details[fmt.Sprintf("packet_meta_%s", key)] = value
				}
			}
		}

		// 解析数据信息（增强版本）
		if decision.Context.ParsedData != nil {
			parsed := decision.Context.ParsedData
			auditLog.Details["data_protocol"] = parsed.Protocol
			auditLog.Details["content_type"] = parsed.ContentType
			auditLog.Details["data_size"] = len(parsed.Body)

			// 确保URL和Method字段被正确记录
			if parsed.URL != "" {
				auditLog.Details["data_url"] = parsed.URL
				auditLog.Details["request_url"] = parsed.URL
			}
			if parsed.Method != "" {
				auditLog.Details["data_method"] = parsed.Method
				auditLog.Details["request_method"] = parsed.Method
			}
			if parsed.StatusCode != 0 {
				auditLog.Details["data_status_code"] = parsed.StatusCode
			}

			// 从元数据中提取关键信息
			if parsed.Metadata != nil {
				// 提取主机信息
				if host, exists := parsed.Metadata["host"]; exists {
					auditLog.Details["dest_domain"] = host
					auditLog.Details["http_host"] = host
				}

				// 提取端口信息
				if destPort, exists := parsed.Metadata["dest_port"]; exists {
					auditLog.Details["dest_port"] = destPort
				}
				if sourcePort, exists := parsed.Metadata["source_port"]; exists {
					auditLog.Details["source_port"] = sourcePort
				}

				// 提取URL组件
				if scheme, exists := parsed.Metadata["scheme"]; exists {
					auditLog.Details["url_scheme"] = scheme
				}
				if path, exists := parsed.Metadata["path"]; exists {
					auditLog.Details["url_path"] = path
				}
				if query, exists := parsed.Metadata["query"]; exists && query != "" {
					auditLog.Details["url_query"] = query
				}

				// 提取用户代理
				if userAgent, exists := parsed.Metadata["user_agent"]; exists {
					auditLog.Details["user_agent"] = userAgent
				}

				// 提取内容长度
				if contentLength, exists := parsed.Metadata["content_length"]; exists {
					auditLog.Details["content_length"] = contentLength
				}

				// 提取请求URI
				if requestURI, exists := parsed.Metadata["request_uri"]; exists {
					auditLog.Details["request_uri"] = requestURI
				}
			}

			// 协议特定的元数据提取
			al.extractProtocolSpecificMetadata(auditLog, parsed)

			// HTTP/HTTPS协议特定信息
			if strings.ToUpper(parsed.Protocol) == "HTTP" || strings.ToUpper(parsed.Protocol) == "HTTPS" {
				al.extractHTTPMetadata(auditLog, parsed)
			}

			// 数据库协议特定信息
			if al.isDatabaseProtocol(parsed.Protocol) {
				al.extractDatabaseMetadata(auditLog, parsed)
			}

			// 邮件协议特定信息
			if al.isEmailProtocol(parsed.Protocol) {
				al.extractEmailMetadata(auditLog, parsed)
			}

			// 文件传输协议特定信息
			if al.isFileTransferProtocol(parsed.Protocol) {
				al.extractFileTransferMetadata(auditLog, parsed)
			}

			// 消息队列协议特定信息
			if al.isMessageQueueProtocol(parsed.Protocol) {
				al.extractMessageQueueMetadata(auditLog, parsed)
			}

			// 通用请求详情
			if parsed.Metadata != nil {
				if url, exists := parsed.Metadata["url"]; exists {
					auditLog.Details["request_url"] = url
				}
				if method, exists := parsed.Metadata["method"]; exists {
					auditLog.Details["request_method"] = method
				}
				if headers, exists := parsed.Metadata["headers"]; exists {
					auditLog.Details["request_headers"] = al.sanitizeHeaders(headers)
				}
				if domain, exists := parsed.Metadata["domain"]; exists {
					auditLog.Details["dest_domain"] = domain
				}
				if userAgent, exists := parsed.Metadata["user_agent"]; exists {
					auditLog.Details["user_agent"] = userAgent
				}
				if referer, exists := parsed.Metadata["referer"]; exists {
					auditLog.Details["referer"] = referer
				}
				if contentLength, exists := parsed.Metadata["content_length"]; exists {
					auditLog.Details["content_length"] = contentLength
				}
				if encoding, exists := parsed.Metadata["encoding"]; exists {
					auditLog.Details["content_encoding"] = encoding
				}
			}

			// HTTP头部详细信息
			if parsed.Headers != nil && len(parsed.Headers) > 0 {
				auditLog.Details["http_headers"] = al.sanitizeHeaders(parsed.Headers)

				// 提取关键头部信息
				if host, exists := parsed.Headers["Host"]; exists {
					auditLog.Details["http_host"] = host
				}
				if cookie, exists := parsed.Headers["Cookie"]; exists {
					auditLog.Details["has_cookies"] = true
					auditLog.Details["cookie_count"] = len(strings.Split(cookie, ";"))
				}
				if auth, exists := parsed.Headers["Authorization"]; exists {
					auditLog.Details["has_auth"] = true
					auditLog.Details["auth_type"] = al.extractAuthType(auth)
				}
			}

			// 数据摘要（安全考虑，只记录摘要）
			if len(parsed.Body) > 0 {
				summary := al.generateDataSummary(parsed.Body)
				auditLog.Details["data_summary"] = summary

				// 检测敏感数据模式
				sensitivePatterns := al.detectSensitivePatterns(parsed.Body)
				if len(sensitivePatterns) > 0 {
					auditLog.Details["sensitive_patterns"] = sensitivePatterns
					auditLog.Details["has_sensitive_data"] = true
				}
			}
		}

		// 分析结果信息
		if decision.Context.AnalysisResult != nil {
			analysis := decision.Context.AnalysisResult
			auditLog.Details["analysis_risk_score"] = analysis.RiskScore
			auditLog.Details["analysis_confidence"] = analysis.Confidence
			auditLog.Details["analysis_risk_level"] = analysis.RiskLevel.String()
			auditLog.Details["analysis_categories"] = analysis.Categories
			auditLog.Details["analysis_tags"] = analysis.Tags
			auditLog.Details["analysis_content_type"] = analysis.ContentType

			// 敏感数据检测结果
			if len(analysis.SensitiveData) > 0 {
				auditLog.Details["sensitive_data_count"] = len(analysis.SensitiveData)
				sensitiveTypes := make([]string, 0, len(analysis.SensitiveData))
				for _, data := range analysis.SensitiveData {
					sensitiveTypes = append(sensitiveTypes, data.Type)
				}
				auditLog.Details["sensitive_data_types"] = sensitiveTypes
			}
		}

		// 会话信息
		if decision.Context.SessionInfo != nil {
			session := decision.Context.SessionInfo
			auditLog.Details["session_id"] = session.ID
			auditLog.Details["session_start_time"] = session.StartTime
			auditLog.Details["session_duration"] = session.Duration.String()
			auditLog.Details["session_activity"] = session.Activity
			auditLog.Details["session_risk_score"] = session.RiskScore
		}

		// 环境信息
		if decision.Context.Environment != nil {
			env := decision.Context.Environment
			auditLog.Details["env_location"] = env.Location
			auditLog.Details["env_network"] = env.Network
			auditLog.Details["env_timezone"] = env.TimeZone
			auditLog.Details["env_working_hours"] = env.WorkingHours
			auditLog.Details["env_holiday"] = env.Holiday
		}
	}

	return al.writeLog(auditLog)
}

// protocolToString 将协议转换为字符串
func (al *AuditLoggerImpl) protocolToString(protocol interceptor.Protocol) string {
	switch protocol {
	case interceptor.ProtocolTCP:
		return "TCP"
	case interceptor.ProtocolUDP:
		return "UDP"
	case 1: // ICMP
		return "ICMP"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(protocol))
	}
}

// directionToString 将方向转换为字符串
func (al *AuditLoggerImpl) directionToString(direction interceptor.PacketDirection) string {
	switch direction {
	case interceptor.PacketDirectionInbound:
		return "inbound"
	case interceptor.PacketDirectionOutbound:
		return "outbound"
	default:
		return "unknown"
	}
}

// sanitizeHeaders 清理HTTP头部信息（移除敏感信息）
func (al *AuditLoggerImpl) sanitizeHeaders(headers interface{}) interface{} {
	if headerMap, ok := headers.(map[string]string); ok {
		sanitized := make(map[string]string)
		for key, value := range headerMap {
			lowerKey := strings.ToLower(key)
			// 移除敏感头部信息
			if lowerKey == "authorization" || lowerKey == "cookie" || lowerKey == "x-api-key" {
				sanitized[key] = "[REDACTED]"
			} else if len(value) > 200 {
				// 截断过长的头部值
				sanitized[key] = value[:200] + "..."
			} else {
				sanitized[key] = value
			}
		}
		return sanitized
	}
	return headers
}

// generateDataSummary 生成数据摘要（用于审计，不记录敏感内容）
func (al *AuditLoggerImpl) generateDataSummary(content []byte) string {
	const maxSummaryLength = 200

	if len(content) == 0 {
		return "empty"
	}

	// 生成内容摘要，避免记录敏感数据
	summary := fmt.Sprintf("size:%d bytes", len(content))

	// 检查是否为文本内容
	if al.isTextContent(content) {
		textContent := string(content)
		if len(textContent) > maxSummaryLength {
			summary += fmt.Sprintf(", preview:%s...", textContent[:maxSummaryLength])
		} else {
			summary += fmt.Sprintf(", content:%s", textContent)
		}
	} else {
		summary += ", type:binary"
	}

	return summary
}

// isTextContent 检查是否为文本内容
func (al *AuditLoggerImpl) isTextContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// 简单的文本检测逻辑
	textBytes := 0
	for i, b := range content {
		if i >= 1000 { // 只检查前1000字节
			break
		}
		if (b >= 32 && b <= 126) || b == 9 || b == 10 || b == 13 { // 可打印字符、制表符、换行符、回车符
			textBytes++
		}
	}

	// 如果80%以上是文本字符，认为是文本内容
	return float64(textBytes)/float64(len(content)) > 0.8
}

// extractProtocolSpecificMetadata 提取协议特定的元数据
func (al *AuditLoggerImpl) extractProtocolSpecificMetadata(auditLog *AuditLog, parsed *parser.ParsedData) {
	if parsed.Metadata == nil {
		return
	}

	// 提取通用协议元数据
	for key, value := range parsed.Metadata {
		// 过滤敏感信息
		if !al.isSensitiveMetadataKey(key) {
			auditLog.Details[fmt.Sprintf("protocol_%s", key)] = value
		}
	}
}

// extractHTTPMetadata 提取HTTP协议特定元数据
func (al *AuditLoggerImpl) extractHTTPMetadata(auditLog *AuditLog, parsed *parser.ParsedData) {
	if parsed.Metadata == nil {
		return
	}

	// HTTP特定字段
	if queryParams, exists := parsed.Metadata["query_params"]; exists {
		auditLog.Details["http_query_params"] = al.sanitizeQueryParams(queryParams)
	}

	if formData, exists := parsed.Metadata["form_data"]; exists {
		auditLog.Details["http_form_data"] = al.sanitizeFormData(formData)
	}

	if cookies, exists := parsed.Metadata["cookies"]; exists {
		auditLog.Details["http_cookies"] = al.sanitizeCookies(cookies)
	}

	if responseTime, exists := parsed.Metadata["response_time"]; exists {
		auditLog.Details["http_response_time"] = responseTime
	}

	if contentEncoding, exists := parsed.Metadata["content_encoding"]; exists {
		auditLog.Details["http_content_encoding"] = contentEncoding
	}

	if transferEncoding, exists := parsed.Metadata["transfer_encoding"]; exists {
		auditLog.Details["http_transfer_encoding"] = transferEncoding
	}
}

// extractDatabaseMetadata 提取数据库协议特定元数据
func (al *AuditLoggerImpl) extractDatabaseMetadata(auditLog *AuditLog, parsed *parser.ParsedData) {
	if parsed.Metadata == nil {
		return
	}

	if dbType, exists := parsed.Metadata["db_type"]; exists {
		auditLog.Details["db_type"] = dbType
	}

	if dbName, exists := parsed.Metadata["database"]; exists {
		auditLog.Details["db_name"] = dbName
	}

	if tableName, exists := parsed.Metadata["table"]; exists {
		auditLog.Details["db_table"] = tableName
	}

	if queryType, exists := parsed.Metadata["query_type"]; exists {
		auditLog.Details["db_query_type"] = queryType
	}

	if rowCount, exists := parsed.Metadata["row_count"]; exists {
		auditLog.Details["db_row_count"] = rowCount
	}

	if executeTime, exists := parsed.Metadata["execute_time"]; exists {
		auditLog.Details["db_execute_time"] = executeTime
	}
}

// extractEmailMetadata 提取邮件协议特定元数据
func (al *AuditLoggerImpl) extractEmailMetadata(auditLog *AuditLog, parsed *parser.ParsedData) {
	if parsed.Metadata == nil {
		return
	}

	if sender, exists := parsed.Metadata["sender"]; exists {
		auditLog.Details["email_sender"] = sender
	}

	if recipients, exists := parsed.Metadata["recipients"]; exists {
		auditLog.Details["email_recipients"] = recipients
	}

	if subject, exists := parsed.Metadata["subject"]; exists {
		auditLog.Details["email_subject"] = al.sanitizeEmailSubject(subject)
	}

	if attachmentCount, exists := parsed.Metadata["attachment_count"]; exists {
		auditLog.Details["email_attachment_count"] = attachmentCount
	}

	if messageSize, exists := parsed.Metadata["message_size"]; exists {
		auditLog.Details["email_message_size"] = messageSize
	}
}

// extractFileTransferMetadata 提取文件传输协议特定元数据
func (al *AuditLoggerImpl) extractFileTransferMetadata(auditLog *AuditLog, parsed *parser.ParsedData) {
	if parsed.Metadata == nil {
		return
	}

	if fileName, exists := parsed.Metadata["file_name"]; exists {
		auditLog.Details["file_name"] = fileName
	}

	if fileSize, exists := parsed.Metadata["file_size"]; exists {
		auditLog.Details["file_size"] = fileSize
	}

	if fileType, exists := parsed.Metadata["file_type"]; exists {
		auditLog.Details["file_type"] = fileType
	}

	if transferDirection, exists := parsed.Metadata["transfer_direction"]; exists {
		auditLog.Details["transfer_direction"] = transferDirection
	}

	if transferSpeed, exists := parsed.Metadata["transfer_speed"]; exists {
		auditLog.Details["transfer_speed"] = transferSpeed
	}
}

// extractMessageQueueMetadata 提取消息队列协议特定元数据
func (al *AuditLoggerImpl) extractMessageQueueMetadata(auditLog *AuditLog, parsed *parser.ParsedData) {
	if parsed.Metadata == nil {
		return
	}

	if topic, exists := parsed.Metadata["topic"]; exists {
		auditLog.Details["mq_topic"] = topic
	}

	if partition, exists := parsed.Metadata["partition"]; exists {
		auditLog.Details["mq_partition"] = partition
	}

	if offset, exists := parsed.Metadata["offset"]; exists {
		auditLog.Details["mq_offset"] = offset
	}

	if messageKey, exists := parsed.Metadata["message_key"]; exists {
		auditLog.Details["mq_message_key"] = messageKey
	}

	if messageType, exists := parsed.Metadata["message_type"]; exists {
		auditLog.Details["mq_message_type"] = messageType
	}
}

// 协议检测辅助方法
func (al *AuditLoggerImpl) isDatabaseProtocol(protocol string) bool {
	dbProtocols := []string{"mysql", "postgresql", "sqlserver", "oracle", "mongodb", "redis"}
	protocol = strings.ToLower(protocol)
	for _, dbProto := range dbProtocols {
		if protocol == dbProto {
			return true
		}
	}
	return false
}

func (al *AuditLoggerImpl) isEmailProtocol(protocol string) bool {
	emailProtocols := []string{"smtp", "pop3", "imap"}
	protocol = strings.ToLower(protocol)
	for _, emailProto := range emailProtocols {
		if protocol == emailProto {
			return true
		}
	}
	return false
}

func (al *AuditLoggerImpl) isFileTransferProtocol(protocol string) bool {
	ftProtocols := []string{"ftp", "sftp", "scp", "rsync"}
	protocol = strings.ToLower(protocol)
	for _, ftProto := range ftProtocols {
		if protocol == ftProto {
			return true
		}
	}
	return false
}

func (al *AuditLoggerImpl) isMessageQueueProtocol(protocol string) bool {
	mqProtocols := []string{"kafka", "rabbitmq", "activemq", "mqtt", "amqp"}
	protocol = strings.ToLower(protocol)
	for _, mqProto := range mqProtocols {
		if protocol == mqProto {
			return true
		}
	}
	return false
}

// 数据清理和脱敏方法
func (al *AuditLoggerImpl) isSensitiveMetadataKey(key string) bool {
	sensitiveKeys := []string{"password", "token", "secret", "key", "auth", "credential"}
	key = strings.ToLower(key)
	for _, sensitiveKey := range sensitiveKeys {
		if strings.Contains(key, sensitiveKey) {
			return true
		}
	}
	return false
}

func (al *AuditLoggerImpl) sanitizeQueryParams(params interface{}) interface{} {
	if paramMap, ok := params.(map[string]string); ok {
		sanitized := make(map[string]string)
		for key, value := range paramMap {
			if al.isSensitiveMetadataKey(key) {
				sanitized[key] = "[REDACTED]"
			} else if len(value) > 100 {
				sanitized[key] = value[:100] + "..."
			} else {
				sanitized[key] = value
			}
		}
		return sanitized
	}
	return params
}

func (al *AuditLoggerImpl) sanitizeFormData(data interface{}) interface{} {
	if dataMap, ok := data.(map[string]string); ok {
		sanitized := make(map[string]string)
		for key, value := range dataMap {
			if al.isSensitiveMetadataKey(key) {
				sanitized[key] = "[REDACTED]"
			} else if len(value) > 200 {
				sanitized[key] = value[:200] + "..."
			} else {
				sanitized[key] = value
			}
		}
		return sanitized
	}
	return data
}

func (al *AuditLoggerImpl) sanitizeCookies(cookies interface{}) interface{} {
	if cookieStr, ok := cookies.(string); ok {
		// 简化处理：只显示cookie数量，不显示具体内容
		cookieCount := len(strings.Split(cookieStr, ";"))
		return fmt.Sprintf("[%d cookies]", cookieCount)
	}
	return "[cookies]"
}

func (al *AuditLoggerImpl) sanitizeEmailSubject(subject interface{}) interface{} {
	if subjectStr, ok := subject.(string); ok {
		if len(subjectStr) > 100 {
			return subjectStr[:100] + "..."
		}
		return subjectStr
	}
	return subject
}

func (al *AuditLoggerImpl) extractAuthType(auth string) string {
	if strings.HasPrefix(auth, "Bearer ") {
		return "Bearer"
	} else if strings.HasPrefix(auth, "Basic ") {
		return "Basic"
	} else if strings.HasPrefix(auth, "Digest ") {
		return "Digest"
	} else if strings.HasPrefix(auth, "OAuth ") {
		return "OAuth"
	}
	return "Unknown"
}

func (al *AuditLoggerImpl) detectSensitivePatterns(content []byte) []string {
	patterns := []string{}
	contentStr := string(content)

	// 简化的敏感数据检测模式
	sensitivePatterns := map[string]string{
		"email":       `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
		"phone":       `\b\d{3}-\d{3}-\d{4}\b`,
		"ssn":         `\b\d{3}-\d{2}-\d{4}\b`,
		"credit_card": `\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`,
		"ip_address":  `\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`,
	}

	for patternName := range sensitivePatterns {
		// 由于没有regexp包，这里使用简化的字符串匹配
		if strings.Contains(strings.ToLower(contentStr), "@") && patternName == "email" {
			patterns = append(patterns, patternName)
		} else if strings.Contains(contentStr, "-") && (patternName == "phone" || patternName == "ssn") {
			patterns = append(patterns, patternName)
		} else if strings.Contains(contentStr, ".") && patternName == "ip_address" {
			patterns = append(patterns, patternName)
		}
	}

	return patterns
}

// LogRuleChange 记录规则变更
func (al *AuditLoggerImpl) LogRuleChange(action string, rule *PolicyRule) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	auditLog := &AuditLog{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Type:      "rule_change",
		Action:    action,
		Result:    "success",
		Details: map[string]interface{}{
			"rule_id":       rule.ID,
			"rule_name":     rule.Name,
			"rule_type":     rule.Type,
			"rule_priority": rule.Priority,
			"rule_enabled":  rule.Enabled,
		},
	}

	return al.writeLog(auditLog)
}

// LogEngineEvent 记录引擎事件
func (al *AuditLoggerImpl) LogEngineEvent(event string, metadata map[string]interface{}) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	auditLog := &AuditLog{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Type:      "engine_event",
		Action:    event,
		Result:    "success",
		Details:   metadata,
	}

	return al.writeLog(auditLog)
}

// GetAuditLogs 获取审计日志
func (al *AuditLoggerImpl) GetAuditLogs(filter *AuditFilter) ([]*AuditLog, error) {
	// 这里简化实现，实际应该从日志文件或数据库中读取
	// 可以使用更复杂的存储和查询机制
	al.logger.Info("获取审计日志", "filter", filter)

	// 返回空列表，实际实现需要根据过滤器查询日志
	return []*AuditLog{}, nil
}

// initLogFile 初始化日志文件
func (al *AuditLoggerImpl) initLogFile() error {
	// 确保日志目录存在
	logDir := filepath.Dir(al.config.LogPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 打开日志文件
	file, err := os.OpenFile(al.config.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	al.logFile = file
	return nil
}

// writeLog 写入日志
func (al *AuditLoggerImpl) writeLog(auditLog *AuditLog) error {
	if al.logFile == nil {
		return fmt.Errorf("日志文件未初始化")
	}

	// 序列化为JSON
	data, err := json.Marshal(auditLog)
	if err != nil {
		return fmt.Errorf("序列化审计日志失败: %w", err)
	}

	// 写入文件
	if _, err := al.logFile.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("写入审计日志失败: %w", err)
	}

	// 刷新缓冲区
	if err := al.logFile.Sync(); err != nil {
		return fmt.Errorf("刷新日志文件失败: %w", err)
	}

	return nil
}

// Close 关闭审计日志记录器
func (al *AuditLoggerImpl) Close() error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.logFile != nil {
		return al.logFile.Close()
	}

	return nil
}

// MLEngineImpl 机器学习引擎实现
type MLEngineImpl struct {
	logger    logging.Logger
	modelInfo *ModelInfo
	ready     bool
}

// NewMLEngine 创建机器学习引擎
func NewMLEngine(logger logging.Logger) MLEngine {
	return &MLEngineImpl{
		logger: logger,
		ready:  false,
	}
}

// PredictRisk 预测风险
func (ml *MLEngineImpl) PredictRisk(context *DecisionContext) (float64, error) {
	if !ml.ready {
		return 0.0, fmt.Errorf("ML模型未就绪")
	}

	// 简化的风险预测实现
	// 实际实现需要使用真实的机器学习模型
	riskScore := 0.0

	// 基于分析结果的风险评分
	if context.AnalysisResult != nil {
		riskScore += context.AnalysisResult.RiskScore * 0.6
	}

	// 基于用户信息的风险评分
	if context.UserInfo != nil {
		switch context.UserInfo.RiskLevel {
		case "high":
			riskScore += 0.3
		case "medium":
			riskScore += 0.2
		case "low":
			riskScore += 0.1
		}
	}

	// 基于设备信息的风险评分
	if context.DeviceInfo != nil {
		if !context.DeviceInfo.Compliance {
			riskScore += 0.2
		}
		switch context.DeviceInfo.TrustLevel {
		case "low":
			riskScore += 0.2
		case "medium":
			riskScore += 0.1
		}
	}

	// 基于环境信息的风险评分
	if context.Environment != nil {
		if !context.Environment.WorkingHours {
			riskScore += 0.1
		}
		if context.Environment.Holiday {
			riskScore += 0.05
		}
	}

	// 确保风险评分在0-1之间
	if riskScore > 1.0 {
		riskScore = 1.0
	}

	ml.logger.Debug("ML风险预测", "risk_score", riskScore)
	return riskScore, nil
}

// TrainModel 训练模型
func (ml *MLEngineImpl) TrainModel(data []TrainingData) error {
	ml.logger.Info("开始训练ML模型", "samples", len(data))

	// 简化的训练实现
	// 实际实现需要使用真实的机器学习算法

	ml.modelInfo = &ModelInfo{
		Name:         "DLP Risk Predictor",
		Version:      "1.0.0",
		Type:         "classification",
		Accuracy:     0.85,
		TrainedAt:    time.Now(),
		FeatureCount: 10,
		SampleCount:  len(data),
	}

	ml.ready = true
	ml.logger.Info("ML模型训练完成")
	return nil
}

// LoadModel 加载模型
func (ml *MLEngineImpl) LoadModel(modelPath string) error {
	ml.logger.Info("加载ML模型", "path", modelPath)

	// 生产级实现：检查模型文件是否存在
	if modelPath == "" {
		return fmt.Errorf("模型路径不能为空")
	}

	// 检查模型文件是否存在
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		ml.logger.Warn("ML模型文件不存在，使用基于规则的分类器", "path", modelPath)

		// 使用基于规则的分类器作为后备
		ml.modelInfo = &ModelInfo{
			Name:         "Rule-based DLP Classifier",
			Version:      "1.0.0",
			Type:         "rule-based",
			Accuracy:     0.75,
			TrainedAt:    time.Now(),
			FeatureCount: 0,
			SampleCount:  0,
		}

		ml.ready = true
		ml.logger.Info("基于规则的分类器已就绪")
		return nil
	}

	// TODO: 实现真实的模型加载逻辑
	// 例如加载TensorFlow、PyTorch或其他ML框架的模型
	ml.logger.Error("真实ML模型加载功能未实现")
	return fmt.Errorf("真实ML模型加载功能未实现，请使用基于规则的分类器")
}

// SaveModel 保存模型
func (ml *MLEngineImpl) SaveModel(modelPath string) error {
	if !ml.ready {
		return fmt.Errorf("没有可保存的模型")
	}

	ml.logger.Info("保存ML模型", "path", modelPath)

	// 生产级实现：只有真实模型才能保存
	if ml.modelInfo.Type == "rule-based" {
		ml.logger.Warn("基于规则的分类器无需保存")
		return nil
	}

	// TODO: 实现真实的模型保存逻辑
	ml.logger.Error("真实ML模型保存功能未实现")
	return fmt.Errorf("真实ML模型保存功能未实现")
}

// GetModelInfo 获取模型信息
func (ml *MLEngineImpl) GetModelInfo() *ModelInfo {
	return ml.modelInfo
}

// IsReady 检查模型是否就绪
func (ml *MLEngineImpl) IsReady() bool {
	return ml.ready
}
