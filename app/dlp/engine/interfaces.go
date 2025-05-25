package engine

import (
	"context"
	"time"

	"github.com/lomehong/kennel/app/dlp/analyzer"
	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/app/dlp/parser"
	"github.com/lomehong/kennel/pkg/logging"
)

// PolicyDecision 策略决策结果
type PolicyDecision struct {
	ID             string                 `json:"id"`
	Timestamp      time.Time              `json:"timestamp"`
	Action         PolicyAction           `json:"action"`
	RiskLevel      analyzer.RiskLevel     `json:"risk_level"`
	RiskScore      float64                `json:"risk_score"`
	Confidence     float64                `json:"confidence"`
	Reason         string                 `json:"reason"`
	MatchedRules   []*MatchedRule         `json:"matched_rules"`
	Metadata       map[string]interface{} `json:"metadata"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Context        *DecisionContext       `json:"context"`
}

// PolicyAction 策略动作
type PolicyAction int

const (
	PolicyActionAllow PolicyAction = iota
	PolicyActionBlock
	PolicyActionAlert
	PolicyActionAudit
	PolicyActionEncrypt
	PolicyActionQuarantine
	PolicyActionRedirect
)

// String 返回策略动作的字符串表示
func (pa PolicyAction) String() string {
	switch pa {
	case PolicyActionAllow:
		return "allow"
	case PolicyActionBlock:
		return "block"
	case PolicyActionAlert:
		return "alert"
	case PolicyActionAudit:
		return "audit"
	case PolicyActionEncrypt:
		return "encrypt"
	case PolicyActionQuarantine:
		return "quarantine"
	case PolicyActionRedirect:
		return "redirect"
	default:
		return "unknown"
	}
}

// MatchedRule 匹配的规则
type MatchedRule struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	RuleType    string                 `json:"rule_type"`
	Priority    int                    `json:"priority"`
	Action      PolicyAction           `json:"action"`
	Confidence  float64                `json:"confidence"`
	MatchedData interface{}            `json:"matched_data"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// DecisionContext 决策上下文
type DecisionContext struct {
	PacketInfo     *interceptor.PacketInfo `json:"packet_info"`
	ParsedData     *parser.ParsedData      `json:"parsed_data"`
	AnalysisResult *analyzer.AnalysisResult `json:"analysis_result"`
	UserInfo       *UserInfo               `json:"user_info"`
	DeviceInfo     *DeviceInfo             `json:"device_info"`
	SessionInfo    *SessionInfo            `json:"session_info"`
	Environment    *Environment            `json:"environment"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Department  string   `json:"department"`
	Role        string   `json:"role"`
	Groups      []string `json:"groups"`
	Permissions []string `json:"permissions"`
	RiskLevel   string   `json:"risk_level"`
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	OS           string `json:"os"`
	Version      string `json:"version"`
	Location     string `json:"location"`
	NetworkInfo  string `json:"network_info"`
	TrustLevel   string `json:"trust_level"`
	Compliance   bool   `json:"compliance"`
}

// SessionInfo 会话信息
type SessionInfo struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"start_time"`
	Duration  time.Duration `json:"duration"`
	Activity  string    `json:"activity"`
	RiskScore float64   `json:"risk_score"`
}

// Environment 环境信息
type Environment struct {
	Location    string    `json:"location"`
	Network     string    `json:"network"`
	TimeZone    string    `json:"time_zone"`
	WorkingHours bool     `json:"working_hours"`
	Holiday     bool      `json:"holiday"`
	Timestamp   time.Time `json:"timestamp"`
}

// PolicyRule 策略规则
type PolicyRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	Enabled     bool                   `json:"enabled"`
	Conditions  []*RuleCondition       `json:"conditions"`
	Actions     []*RuleAction          `json:"actions"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Version     string                 `json:"version"`
}

// RuleCondition 规则条件
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type"`
}

// RuleAction 规则动作
type RuleAction struct {
	Type       PolicyAction           `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// PolicyEngineConfig 策略引擎配置
type PolicyEngineConfig struct {
	MaxRules        int           `yaml:"max_rules" json:"max_rules"`
	Timeout         time.Duration `yaml:"timeout" json:"timeout"`
	EnableCache     bool          `yaml:"enable_cache" json:"enable_cache"`
	CacheSize       int           `yaml:"cache_size" json:"cache_size"`
	CacheTTL        time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
	EnableAudit     bool          `yaml:"enable_audit" json:"enable_audit"`
	AuditLevel      string        `yaml:"audit_level" json:"audit_level"`
	DefaultAction   PolicyAction  `yaml:"default_action" json:"default_action"`
	RulesPath       string        `yaml:"rules_path" json:"rules_path"`
	EnableMLEngine  bool          `yaml:"enable_ml_engine" json:"enable_ml_engine"`
	MLModelPath     string        `yaml:"ml_model_path" json:"ml_model_path"`
	MaxConcurrency  int           `yaml:"max_concurrency" json:"max_concurrency"`
	Logger          logging.Logger `yaml:"-" json:"-"`
}

// DefaultPolicyEngineConfig 返回默认策略引擎配置
func DefaultPolicyEngineConfig() PolicyEngineConfig {
	return PolicyEngineConfig{
		MaxRules:       10000,
		Timeout:        30 * time.Second,
		EnableCache:    true,
		CacheSize:      10000,
		CacheTTL:       1 * time.Hour,
		EnableAudit:    true,
		AuditLevel:     "info",
		DefaultAction:  PolicyActionAudit,
		EnableMLEngine: false,
		MaxConcurrency: 100,
	}
}

// PolicyEngine 策略引擎接口
type PolicyEngine interface {
	// EvaluatePolicy 评估策略
	EvaluatePolicy(ctx context.Context, context *DecisionContext) (*PolicyDecision, error)

	// LoadRules 加载规则
	LoadRules(rules []*PolicyRule) error

	// AddRule 添加规则
	AddRule(rule *PolicyRule) error

	// RemoveRule 删除规则
	RemoveRule(ruleID string) error

	// UpdateRule 更新规则
	UpdateRule(rule *PolicyRule) error

	// GetRule 获取规则
	GetRule(ruleID string) (*PolicyRule, bool)

	// GetRules 获取所有规则
	GetRules() []*PolicyRule

	// GetStats 获取统计信息
	GetStats() EngineStats

	// Start 启动引擎
	Start() error

	// Stop 停止引擎
	Stop() error

	// HealthCheck 健康检查
	HealthCheck() error
}

// EngineStats 引擎统计信息
type EngineStats struct {
	TotalDecisions    uint64            `json:"total_decisions"`
	AllowedDecisions  uint64            `json:"allowed_decisions"`
	BlockedDecisions  uint64            `json:"blocked_decisions"`
	AlertDecisions    uint64            `json:"alert_decisions"`
	AuditDecisions    uint64            `json:"audit_decisions"`
	FailedDecisions   uint64            `json:"failed_decisions"`
	AverageTime       time.Duration     `json:"average_time"`
	RuleStats         map[string]uint64 `json:"rule_stats"`
	LastError         error             `json:"last_error,omitempty"`
	StartTime         time.Time         `json:"start_time"`
	Uptime            time.Duration     `json:"uptime"`
}

// RuleEvaluator 规则评估器接口
type RuleEvaluator interface {
	// EvaluateRule 评估规则
	EvaluateRule(rule *PolicyRule, context *DecisionContext) (*RuleEvaluationResult, error)

	// GetSupportedTypes 获取支持的规则类型
	GetSupportedTypes() []string

	// CanEvaluate 检查是否能评估指定类型的规则
	CanEvaluate(ruleType string) bool
}

// RuleEvaluationResult 规则评估结果
type RuleEvaluationResult struct {
	RuleID     string                 `json:"rule_id"`
	Matched    bool                   `json:"matched"`
	Confidence float64                `json:"confidence"`
	Actions    []*RuleAction          `json:"actions"`
	Metadata   map[string]interface{} `json:"metadata"`
	Reason     string                 `json:"reason"`
}

// ConditionEvaluator 条件评估器接口
type ConditionEvaluator interface {
	// EvaluateCondition 评估条件
	EvaluateCondition(condition *RuleCondition, context *DecisionContext) (bool, error)

	// GetSupportedOperators 获取支持的操作符
	GetSupportedOperators() []string

	// GetSupportedFields 获取支持的字段
	GetSupportedFields() []string
}

// ActionExecutor 动作执行器接口
type ActionExecutor interface {
	// ExecuteAction 执行动作
	ExecuteAction(action *RuleAction, context *DecisionContext) error

	// GetSupportedActions 获取支持的动作类型
	GetSupportedActions() []PolicyAction

	// CanExecute 检查是否能执行指定类型的动作
	CanExecute(actionType PolicyAction) bool
}

// RuleManager 规则管理器接口
type RuleManager interface {
	// LoadRules 加载规则
	LoadRules(source string) ([]*PolicyRule, error)

	// SaveRules 保存规则
	SaveRules(rules []*PolicyRule, destination string) error

	// ValidateRule 验证规则
	ValidateRule(rule *PolicyRule) error

	// GetRulesByType 根据类型获取规则
	GetRulesByType(ruleType string) []*PolicyRule

	// GetRulesByPriority 根据优先级获取规则
	GetRulesByPriority() []*PolicyRule

	// ImportRules 导入规则
	ImportRules(data []byte, format string) ([]*PolicyRule, error)

	// ExportRules 导出规则
	ExportRules(rules []*PolicyRule, format string) ([]byte, error)
}

// AuditLogger 审计日志记录器接口
type AuditLogger interface {
	// LogDecision 记录决策
	LogDecision(decision *PolicyDecision) error

	// LogRuleChange 记录规则变更
	LogRuleChange(action string, rule *PolicyRule) error

	// LogEngineEvent 记录引擎事件
	LogEngineEvent(event string, metadata map[string]interface{}) error

	// GetAuditLogs 获取审计日志
	GetAuditLogs(filter *AuditFilter) ([]*AuditLog, error)
}

// AuditLog 审计日志
type AuditLog struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Action    string                 `json:"action"`
	UserID    string                 `json:"user_id"`
	DeviceID  string                 `json:"device_id"`
	Details   map[string]interface{} `json:"details"`
	Result    string                 `json:"result"`
}

// AuditFilter 审计过滤器
type AuditFilter struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Type      string    `json:"type"`
	Action    string    `json:"action"`
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

// MLEngine 机器学习引擎接口
type MLEngine interface {
	// PredictRisk 预测风险
	PredictRisk(context *DecisionContext) (float64, error)

	// TrainModel 训练模型
	TrainModel(data []TrainingData) error

	// LoadModel 加载模型
	LoadModel(modelPath string) error

	// SaveModel 保存模型
	SaveModel(modelPath string) error

	// GetModelInfo 获取模型信息
	GetModelInfo() *ModelInfo

	// IsReady 检查模型是否就绪
	IsReady() bool
}

// TrainingData 训练数据
type TrainingData struct {
	Features []float64 `json:"features"`
	Label    float64   `json:"label"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Type        string    `json:"type"`
	Accuracy    float64   `json:"accuracy"`
	TrainedAt   time.Time `json:"trained_at"`
	FeatureCount int      `json:"feature_count"`
	SampleCount  int      `json:"sample_count"`
}
