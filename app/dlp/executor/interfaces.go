package executor

import (
	"context"
	"time"

	"github.com/lomehong/kennel/app/dlp/engine"
	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

// ExecutionResult 执行结果
type ExecutionResult struct {
	ID             string                 `json:"id"`
	Timestamp      time.Time              `json:"timestamp"`
	Action         engine.PolicyAction    `json:"action"`
	Success        bool                   `json:"success"`
	Error          error                  `json:"error,omitempty"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Metadata       map[string]interface{} `json:"metadata"`
	AffectedData   interface{}            `json:"affected_data,omitempty"`
}

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	Timeout         time.Duration  `yaml:"timeout" json:"timeout"`
	MaxRetries      int            `yaml:"max_retries" json:"max_retries"`
	RetryInterval   time.Duration  `yaml:"retry_interval" json:"retry_interval"`
	EnableAudit     bool           `yaml:"enable_audit" json:"enable_audit"`
	AuditLevel      string         `yaml:"audit_level" json:"audit_level"`
	MaxConcurrency  int            `yaml:"max_concurrency" json:"max_concurrency"`
	BufferSize      int            `yaml:"buffer_size" json:"buffer_size"`
	EnableMetrics   bool           `yaml:"enable_metrics" json:"enable_metrics"`
	MetricsInterval time.Duration  `yaml:"metrics_interval" json:"metrics_interval"`
	Logger          logging.Logger `yaml:"-" json:"-"`
}

// DefaultExecutorConfig 返回默认执行器配置
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryInterval:   1 * time.Second,
		EnableAudit:     true,
		AuditLevel:      "info",
		MaxConcurrency:  100,
		BufferSize:      1000,
		EnableMetrics:   true,
		MetricsInterval: 1 * time.Minute,
	}
}

// ActionExecutor 动作执行器接口
type ActionExecutor interface {
	// ExecuteAction 执行动作
	ExecuteAction(ctx context.Context, decision *engine.PolicyDecision) (*ExecutionResult, error)

	// GetSupportedActions 获取支持的动作类型
	GetSupportedActions() []engine.PolicyAction

	// CanExecute 检查是否能执行指定类型的动作
	CanExecute(actionType engine.PolicyAction) bool

	// Initialize 初始化执行器
	Initialize(config ExecutorConfig) error

	// Cleanup 清理资源
	Cleanup() error

	// GetStats 获取统计信息
	GetStats() ExecutorStats
}

// ExecutorStats 执行器统计信息
type ExecutorStats struct {
	TotalExecutions      uint64            `json:"total_executions"`
	SuccessfulExecutions uint64            `json:"successful_executions"`
	FailedExecutions     uint64            `json:"failed_executions"`
	AverageTime          time.Duration     `json:"average_time"`
	ActionStats          map[string]uint64 `json:"action_stats"`
	LastError            error             `json:"last_error,omitempty"`
	StartTime            time.Time         `json:"start_time"`
	Uptime               time.Duration     `json:"uptime"`
}

// ExecutionManager 执行管理器接口
type ExecutionManager interface {
	// RegisterExecutor 注册执行器
	RegisterExecutor(actionType engine.PolicyAction, executor ActionExecutor) error

	// GetExecutor 获取执行器
	GetExecutor(actionType engine.PolicyAction) (ActionExecutor, bool)

	// ExecuteDecision 执行决策
	ExecuteDecision(ctx context.Context, decision *engine.PolicyDecision) (*ExecutionResult, error)

	// GetSupportedActions 获取支持的动作类型
	GetSupportedActions() []engine.PolicyAction

	// GetStats 获取统计信息
	GetStats() ManagerStats

	// Start 启动管理器
	Start() error

	// Stop 停止管理器
	Stop() error

	// HealthCheck 健康检查
	HealthCheck() error
}

// ManagerStats 管理器统计信息
type ManagerStats struct {
	TotalRequests      uint64                   `json:"total_requests"`
	ProcessedRequests  uint64                   `json:"processed_requests"`
	FailedRequests     uint64                   `json:"failed_requests"`
	AverageTime        time.Duration            `json:"average_time"`
	ExecutorStats      map[string]ExecutorStats `json:"executor_stats"`
	ActionDistribution map[string]uint64        `json:"action_distribution"`
	LastError          error                    `json:"last_error,omitempty"`
	StartTime          time.Time                `json:"start_time"`
	Uptime             time.Duration            `json:"uptime"`
}

// BlockExecutor 阻断执行器接口
type BlockExecutor interface {
	ActionExecutor

	// BlockPacket 阻断数据包
	BlockPacket(packet *interceptor.PacketInfo) error

	// BlockConnection 阻断连接
	BlockConnection(sourceIP, destIP string, port uint16) error

	// GetBlockedConnections 获取被阻断的连接
	GetBlockedConnections() []BlockedConnection
}

// BlockedConnection 被阻断的连接
type BlockedConnection struct {
	ID        string        `json:"id"`
	SourceIP  string        `json:"source_ip"`
	DestIP    string        `json:"dest_ip"`
	Port      uint16        `json:"port"`
	Protocol  string        `json:"protocol"`
	Reason    string        `json:"reason"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
}

// AlertExecutor 告警执行器接口
type AlertExecutor interface {
	ActionExecutor

	// SendAlert 发送告警
	SendAlert(alert *Alert) error

	// GetAlertChannels 获取告警通道
	GetAlertChannels() []string

	// ConfigureChannel 配置告警通道
	ConfigureChannel(channel string, config map[string]interface{}) error
}

// Alert 告警信息
type Alert struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Level      AlertLevel             `json:"level"`
	Source     string                 `json:"source"`
	Timestamp  time.Time              `json:"timestamp"`
	Tags       []string               `json:"tags"`
	Metadata   map[string]interface{} `json:"metadata"`
	Recipients []string               `json:"recipients"`
	Channels   []string               `json:"channels"`
}

// AlertLevel 告警级别
type AlertLevel int

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelError
	AlertLevelCritical
)

// String 返回告警级别的字符串表示
func (al AlertLevel) String() string {
	switch al {
	case AlertLevelInfo:
		return "info"
	case AlertLevelWarning:
		return "warning"
	case AlertLevelError:
		return "error"
	case AlertLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// AuditExecutor 审计执行器接口
type AuditExecutor interface {
	ActionExecutor

	// LogAuditEvent 记录审计事件
	LogAuditEvent(event *AuditEvent) error

	// GetAuditLogs 获取审计日志
	GetAuditLogs(filter *AuditFilter) ([]*AuditEvent, error)

	// ExportAuditLogs 导出审计日志
	ExportAuditLogs(filter *AuditFilter, format string) ([]byte, error)
}

// AuditEvent 审计事件
type AuditEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	Action    string    `json:"action"`
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	SourceIP  string    `json:"source_ip"`
	DestIP    string    `json:"dest_ip"`
	Protocol  string    `json:"protocol"`
	RiskLevel string    `json:"risk_level"`
	RiskScore float64   `json:"risk_score"`
	Result    string    `json:"result"`
	Reason    string    `json:"reason"`

	// 网络连接详细信息
	SourcePort  uint16 `json:"source_port"`            // 源进程使用的本地端口号
	DestPort    uint16 `json:"dest_port"`              // 目标服务器端口号
	DestDomain  string `json:"dest_domain,omitempty"`  // 目标域名（如果可解析）
	RequestURL  string `json:"request_url,omitempty"`  // 完整HTTP/HTTPS请求URL
	RequestData string `json:"request_data,omitempty"` // 发送的内容摘要或关键信息

	// 进程信息
	ProcessInfo *ProcessInfo `json:"process_info,omitempty"`

	Details  map[string]interface{} `json:"details"`
	Metadata map[string]interface{} `json:"metadata"`
}

// AuditFilter 审计过滤器
type AuditFilter struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	EventType    string    `json:"event_type"`
	Action       string    `json:"action"`
	UserID       string    `json:"user_id"`
	DeviceID     string    `json:"device_id"`
	RiskLevel    string    `json:"risk_level"`
	MinRiskScore float64   `json:"min_risk_score"`
	Limit        int       `json:"limit"`
	Offset       int       `json:"offset"`
}

// EncryptExecutor 加密执行器接口
type EncryptExecutor interface {
	ActionExecutor

	// EncryptData 加密数据
	EncryptData(data []byte, algorithm string) ([]byte, error)

	// DecryptData 解密数据
	DecryptData(encryptedData []byte, algorithm string) ([]byte, error)

	// GetSupportedAlgorithms 获取支持的加密算法
	GetSupportedAlgorithms() []string

	// GenerateKey 生成密钥
	GenerateKey(algorithm string, keySize int) ([]byte, error)
}

// QuarantineExecutor 隔离执行器接口
type QuarantineExecutor interface {
	ActionExecutor

	// QuarantineFile 隔离文件
	QuarantineFile(filePath string, reason string) error

	// RestoreFile 恢复文件
	RestoreFile(quarantineID string) error

	// GetQuarantinedFiles 获取隔离的文件
	GetQuarantinedFiles() []QuarantinedFile

	// DeleteQuarantinedFile 删除隔离的文件
	DeleteQuarantinedFile(quarantineID string) error
}

// QuarantinedFile 隔离的文件
type QuarantinedFile struct {
	ID             string                 `json:"id"`
	OriginalPath   string                 `json:"original_path"`
	QuarantinePath string                 `json:"quarantine_path"`
	Reason         string                 `json:"reason"`
	Timestamp      time.Time              `json:"timestamp"`
	Size           int64                  `json:"size"`
	Hash           string                 `json:"hash"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// RedirectExecutor 重定向执行器接口
type RedirectExecutor interface {
	ActionExecutor

	// RedirectConnection 重定向连接
	RedirectConnection(originalDest, newDest string) error

	// GetRedirectRules 获取重定向规则
	GetRedirectRules() []RedirectRule

	// AddRedirectRule 添加重定向规则
	AddRedirectRule(rule *RedirectRule) error

	// RemoveRedirectRule 删除重定向规则
	RemoveRedirectRule(ruleID string) error
}

// RedirectRule 重定向规则
type RedirectRule struct {
	ID           string     `json:"id"`
	OriginalDest string     `json:"original_dest"`
	NewDest      string     `json:"new_dest"`
	Protocol     string     `json:"protocol"`
	Reason       string     `json:"reason"`
	Enabled      bool       `json:"enabled"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// NotificationService 通知服务接口
type NotificationService interface {
	// SendNotification 发送通知
	SendNotification(notification *Notification) error

	// GetSupportedChannels 获取支持的通知渠道
	GetSupportedChannels() []string

	// ConfigureChannel 配置通知渠道
	ConfigureChannel(channel string, config map[string]interface{}) error

	// TestChannel 测试通知渠道
	TestChannel(channel string) error
}

// Notification 通知
type Notification struct {
	ID         string                 `json:"id"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Level      AlertLevel             `json:"level"`
	Channel    string                 `json:"channel"`
	Recipients []string               `json:"recipients"`
	Metadata   map[string]interface{} `json:"metadata"`
	Timestamp  time.Time              `json:"timestamp"`
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// RecordExecution 记录执行指标
	RecordExecution(action engine.PolicyAction, duration time.Duration, success bool)

	// RecordError 记录错误指标
	RecordError(action engine.PolicyAction, error string)

	// GetMetrics 获取指标
	GetMetrics() map[string]interface{}

	// ResetMetrics 重置指标
	ResetMetrics()
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// DefaultRetryPolicy 返回默认重试策略
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []string{
			"timeout",
			"connection_error",
			"temporary_failure",
		},
	}
}

// FirewallRule 防火墙规则
type FirewallRule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Action     string                 `json:"action"`   // block, allow, drop
	Protocol   string                 `json:"protocol"` // tcp, udp, icmp, all
	SourceIP   string                 `json:"source_ip"`
	DestIP     string                 `json:"dest_ip"`
	SourcePort string                 `json:"source_port"`
	DestPort   string                 `json:"dest_port"`
	Direction  string                 `json:"direction"` // inbound, outbound, both
	Enabled    bool                   `json:"enabled"`
	CreatedAt  time.Time              `json:"created_at"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
	Reason     string                 `json:"reason"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	SMTPServer string   `json:"smtp_server"`
	SMTPPort   int      `json:"smtp_port"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	From       string   `json:"from"`
	Recipients []string `json:"recipients"`
	UseTLS     bool     `json:"use_tls"`
	UseSSL     bool     `json:"use_ssl"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	Timeout    time.Duration     `json:"timeout"`
	RetryCount int               `json:"retry_count"`
	RetryDelay time.Duration     `json:"retry_delay"`
}

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	Algorithm  string `json:"algorithm"` // AES-256, RSA, etc.
	KeySize    int    `json:"key_size"`
	Mode       string `json:"mode"` // CBC, GCM, etc.
	KeyPath    string `json:"key_path"`
	CertPath   string `json:"cert_path"`
	Passphrase string `json:"passphrase"`
}

// QuarantineConfig 隔离配置
type QuarantineConfig struct {
	QuarantineDir    string        `json:"quarantine_dir"`
	MaxFileSize      int64         `json:"max_file_size"`
	RetentionPeriod  time.Duration `json:"retention_period"`
	CompressionLevel int           `json:"compression_level"`
	EncryptFiles     bool          `json:"encrypt_files"`
}
