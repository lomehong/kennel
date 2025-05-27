//go:build selfprotect
// +build selfprotect

package selfprotect

// WhitelistChecker 白名单检查器接口
type WhitelistChecker interface {
	// IsProcessWhitelisted 检查进程是否在白名单中
	IsProcessWhitelisted(processName string, processPath string) bool

	// IsUserWhitelisted 检查用户是否在白名单中
	IsUserWhitelisted(username string) bool

	// IsSignatureWhitelisted 检查签名是否在白名单中
	IsSignatureWhitelisted(signature string) bool

	// AddToWhitelist 添加到白名单
	AddToWhitelist(item string, itemType string) error

	// RemoveFromWhitelist 从白名单移除
	RemoveFromWhitelist(item string, itemType string) error
}

// ProtectionValidator 防护验证器接口
type ProtectionValidator interface {
	// ValidateConfig 验证配置
	ValidateConfig(config *ProtectionConfig) error

	// ValidatePermissions 验证权限
	ValidatePermissions() error

	// ValidateEnvironment 验证环境
	ValidateEnvironment() error

	// ValidateIntegrity 验证完整性
	ValidateIntegrity() error
}

// ProtectionHealth 防护健康状态
type ProtectionHealth struct {
	Overall    string                     `json:"overall"`
	Components map[string]ComponentHealth `json:"components"`
	Issues     []HealthIssue              `json:"issues"`
	LastCheck  string                     `json:"last_check"`
}

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Status     string `json:"status"`
	LastCheck  string `json:"last_check"`
	ErrorCount int    `json:"error_count"`
	LastError  string `json:"last_error,omitempty"`
}

// HealthIssue 健康问题
type HealthIssue struct {
	Component   string `json:"component"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// ProtectionMetrics 防护指标接口
type ProtectionMetrics interface {
	// RecordEvent 记录事件
	RecordEvent(eventType string, action string, success bool)

	// RecordLatency 记录延迟
	RecordLatency(operation string, duration float64)

	// IncrementCounter 增加计数器
	IncrementCounter(name string, labels map[string]string)

	// SetGauge 设置仪表
	SetGauge(name string, value float64, labels map[string]string)

	// GetMetrics 获取指标
	GetMetrics() map[string]interface{}
}

// ProtectionAuditor 防护审计器接口
type ProtectionAuditor interface {
	// LogSecurityEvent 记录安全事件
	LogSecurityEvent(event SecurityEvent) error

	// LogAccessAttempt 记录访问尝试
	LogAccessAttempt(attempt AccessAttempt) error

	// LogConfigChange 记录配置变更
	LogConfigChange(change ConfigChange) error

	// GetAuditLogs 获取审计日志
	GetAuditLogs(filter AuditFilter) ([]AuditLog, error)
}

// SecurityEvent 安全事件
type SecurityEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Severity  string                 `json:"severity"`
	Source    string                 `json:"source"`
	Target    string                 `json:"target"`
	Action    string                 `json:"action"`
	Result    string                 `json:"result"`
	Timestamp string                 `json:"timestamp"`
	Details   map[string]interface{} `json:"details"`
}

// AccessAttempt 访问尝试
type AccessAttempt struct {
	ID        string `json:"id"`
	User      string `json:"user"`
	Process   string `json:"process"`
	Target    string `json:"target"`
	Action    string `json:"action"`
	Result    string `json:"result"`
	Timestamp string `json:"timestamp"`
	IPAddress string `json:"ip_address,omitempty"`
}

// ConfigChange 配置变更
type ConfigChange struct {
	ID        string      `json:"id"`
	User      string      `json:"user"`
	Component string      `json:"component"`
	Field     string      `json:"field"`
	OldValue  interface{} `json:"old_value"`
	NewValue  interface{} `json:"new_value"`
	Timestamp string      `json:"timestamp"`
	Reason    string      `json:"reason,omitempty"`
}

// AuditLog 审计日志
type AuditLog struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	User      string                 `json:"user"`
	Action    string                 `json:"action"`
	Target    string                 `json:"target"`
	Result    string                 `json:"result"`
	Details   map[string]interface{} `json:"details"`
}

// AuditFilter 审计过滤器
type AuditFilter struct {
	StartTime string   `json:"start_time,omitempty"`
	EndTime   string   `json:"end_time,omitempty"`
	Types     []string `json:"types,omitempty"`
	Users     []string `json:"users,omitempty"`
	Actions   []string `json:"actions,omitempty"`
	Results   []string `json:"results,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Offset    int      `json:"offset,omitempty"`
}
