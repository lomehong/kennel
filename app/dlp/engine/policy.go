package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lomehong/kennel/app/dlp/analyzer"
	"github.com/lomehong/kennel/pkg/logging"
)

// PolicyEngineImpl 策略引擎实现
type PolicyEngineImpl struct {
	config        PolicyEngineConfig
	logger        logging.Logger
	rules         map[string]*PolicyRule
	ruleEvaluator RuleEvaluator
	auditLogger   AuditLogger
	mlEngine      MLEngine
	stats         EngineStats
	running       int32
	mu            sync.RWMutex
}

// NewPolicyEngine 创建策略引擎
func NewPolicyEngine(logger logging.Logger, config PolicyEngineConfig) PolicyEngine {
	return &PolicyEngineImpl{
		config:        config,
		logger:        logger,
		rules:         make(map[string]*PolicyRule),
		ruleEvaluator: NewRuleEvaluator(logger),
		auditLogger:   NewAuditLogger(logger),
		stats: EngineStats{
			RuleStats: make(map[string]uint64),
			StartTime: time.Now(),
		},
	}
}

// EvaluatePolicy 评估策略
func (pe *PolicyEngineImpl) EvaluatePolicy(ctx context.Context, context *DecisionContext) (*PolicyDecision, error) {
	startTime := time.Now()
	atomic.AddUint64(&pe.stats.TotalDecisions, 1)

	// 创建决策结果
	decision := &PolicyDecision{
		ID:           fmt.Sprintf("decision_%d", time.Now().UnixNano()),
		Timestamp:    time.Now(),
		Action:       pe.config.DefaultAction,
		RiskLevel:    context.AnalysisResult.RiskLevel,
		RiskScore:    context.AnalysisResult.RiskScore,
		Confidence:   0.0,
		MatchedRules: make([]*MatchedRule, 0),
		Metadata:     make(map[string]interface{}),
		Context:      context,
	}

	// 获取排序后的规则列表
	rules := pe.getSortedRules()

	// 评估规则
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// 检查超时
		select {
		case <-ctx.Done():
			atomic.AddUint64(&pe.stats.FailedDecisions, 1)
			return nil, fmt.Errorf("策略评估超时")
		default:
		}

		// 评估规则
		result, err := pe.ruleEvaluator.EvaluateRule(rule, context)
		if err != nil {
			pe.logger.Warn("规则评估失败", "rule_id", rule.ID, "error", err)
			continue
		}

		if result.Matched {
			// 记录匹配的规则
			matchedRule := &MatchedRule{
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				RuleType:    rule.Type,
				Priority:    rule.Priority,
				Confidence:  result.Confidence,
				MatchedData: result.Metadata,
				Metadata:    result.Metadata,
			}

			// 确定动作
			if len(result.Actions) > 0 {
				matchedRule.Action = result.Actions[0].Type
				decision.Action = result.Actions[0].Type
			}

			decision.MatchedRules = append(decision.MatchedRules, matchedRule)

			// 更新置信度
			if result.Confidence > decision.Confidence {
				decision.Confidence = result.Confidence
			}

			// 更新统计信息
			pe.mu.Lock()
			pe.stats.RuleStats[rule.ID]++
			pe.mu.Unlock()

			// 如果是高优先级规则且置信度高，可以提前结束评估
			if rule.Priority >= 90 && result.Confidence >= 0.9 {
				decision.Reason = fmt.Sprintf("高优先级规则匹配: %s", rule.Name)
				break
			}
		}
	}

	// 使用机器学习引擎进行风险预测
	if pe.config.EnableMLEngine && pe.mlEngine != nil && pe.mlEngine.IsReady() {
		mlRisk, err := pe.mlEngine.PredictRisk(context)
		if err == nil {
			decision.Metadata["ml_risk_score"] = mlRisk
			// 如果ML预测的风险更高，更新风险评分
			if mlRisk > decision.RiskScore {
				decision.RiskScore = mlRisk
			}
		}
	}

	// 最终决策逻辑
	pe.finalizeDecision(decision)

	// 更新统计信息
	processingTime := time.Since(startTime)
	decision.ProcessingTime = processingTime
	pe.updateStats(decision)

	// 记录审计日志
	if pe.config.EnableAudit && pe.auditLogger != nil {
		if err := pe.auditLogger.LogDecision(decision); err != nil {
			pe.logger.Error("记录审计日志失败", "error", err)
		}
	}

	pe.logger.Debug("策略评估完成",
		"decision_id", decision.ID,
		"action", decision.Action.String(),
		"risk_score", decision.RiskScore,
		"matched_rules", len(decision.MatchedRules),
		"processing_time", processingTime)

	return decision, nil
}

// LoadRules 加载规则
func (pe *PolicyEngineImpl) LoadRules(rules []*PolicyRule) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if len(rules) > pe.config.MaxRules {
		return fmt.Errorf("规则数量超过限制: %d > %d", len(rules), pe.config.MaxRules)
	}

	// 验证规则
	for _, rule := range rules {
		if err := pe.validateRule(rule); err != nil {
			return fmt.Errorf("规则验证失败 [%s]: %w", rule.ID, err)
		}
	}

	// 清空现有规则
	pe.rules = make(map[string]*PolicyRule)

	// 加载新规则
	for _, rule := range rules {
		pe.rules[rule.ID] = rule
		pe.stats.RuleStats[rule.ID] = 0
	}

	pe.logger.Info("加载策略规则", "count", len(rules))
	return nil
}

// AddRule 添加规则
func (pe *PolicyEngineImpl) AddRule(rule *PolicyRule) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if len(pe.rules) >= pe.config.MaxRules {
		return fmt.Errorf("规则数量已达到限制: %d", pe.config.MaxRules)
	}

	if err := pe.validateRule(rule); err != nil {
		return fmt.Errorf("规则验证失败: %w", err)
	}

	if _, exists := pe.rules[rule.ID]; exists {
		return fmt.Errorf("规则已存在: %s", rule.ID)
	}

	pe.rules[rule.ID] = rule
	pe.stats.RuleStats[rule.ID] = 0

	pe.logger.Info("添加策略规则", "rule_id", rule.ID, "rule_name", rule.Name)
	return nil
}

// RemoveRule 删除规则
func (pe *PolicyEngineImpl) RemoveRule(ruleID string) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if _, exists := pe.rules[ruleID]; !exists {
		return fmt.Errorf("规则不存在: %s", ruleID)
	}

	delete(pe.rules, ruleID)
	delete(pe.stats.RuleStats, ruleID)

	pe.logger.Info("删除策略规则", "rule_id", ruleID)
	return nil
}

// UpdateRule 更新规则
func (pe *PolicyEngineImpl) UpdateRule(rule *PolicyRule) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if err := pe.validateRule(rule); err != nil {
		return fmt.Errorf("规则验证失败: %w", err)
	}

	if _, exists := pe.rules[rule.ID]; !exists {
		return fmt.Errorf("规则不存在: %s", rule.ID)
	}

	rule.UpdatedAt = time.Now()
	pe.rules[rule.ID] = rule

	pe.logger.Info("更新策略规则", "rule_id", rule.ID, "rule_name", rule.Name)
	return nil
}

// GetRule 获取规则
func (pe *PolicyEngineImpl) GetRule(ruleID string) (*PolicyRule, bool) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	rule, exists := pe.rules[ruleID]
	return rule, exists
}

// GetRules 获取所有规则
func (pe *PolicyEngineImpl) GetRules() []*PolicyRule {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	rules := make([]*PolicyRule, 0, len(pe.rules))
	for _, rule := range pe.rules {
		rules = append(rules, rule)
	}

	return rules
}

// GetStats 获取统计信息
func (pe *PolicyEngineImpl) GetStats() EngineStats {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	stats := pe.stats
	stats.Uptime = time.Since(pe.stats.StartTime)
	return stats
}

// Start 启动引擎
func (pe *PolicyEngineImpl) Start() error {
	if !atomic.CompareAndSwapInt32(&pe.running, 0, 1) {
		return fmt.Errorf("策略引擎已在运行")
	}

	pe.logger.Info("启动策略引擎")

	// 初始化机器学习引擎
	if pe.config.EnableMLEngine && pe.config.MLModelPath != "" {
		pe.mlEngine = NewMLEngine(pe.logger)
		if err := pe.mlEngine.LoadModel(pe.config.MLModelPath); err != nil {
			pe.logger.Warn("加载ML模型失败", "error", err)
		}
	}

	// 加载默认规则
	if err := pe.loadDefaultRules(); err != nil {
		return fmt.Errorf("加载默认规则失败: %w", err)
	}

	pe.logger.Info("策略引擎已启动")
	return nil
}

// Stop 停止引擎
func (pe *PolicyEngineImpl) Stop() error {
	if !atomic.CompareAndSwapInt32(&pe.running, 1, 0) {
		return fmt.Errorf("策略引擎未在运行")
	}

	pe.logger.Info("停止策略引擎")

	// 清理资源
	pe.mu.Lock()
	pe.rules = make(map[string]*PolicyRule)
	pe.mu.Unlock()

	pe.logger.Info("策略引擎已停止")
	return nil
}

// HealthCheck 健康检查
func (pe *PolicyEngineImpl) HealthCheck() error {
	if atomic.LoadInt32(&pe.running) == 0 {
		return fmt.Errorf("策略引擎未运行")
	}

	pe.mu.RLock()
	ruleCount := len(pe.rules)
	pe.mu.RUnlock()

	if ruleCount == 0 {
		return fmt.Errorf("没有加载任何规则")
	}

	return nil
}

// getSortedRules 获取按优先级排序的规则
func (pe *PolicyEngineImpl) getSortedRules() []*PolicyRule {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	rules := make([]*PolicyRule, 0, len(pe.rules))
	for _, rule := range pe.rules {
		rules = append(rules, rule)
	}

	// 按优先级降序排序
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})

	return rules
}

// finalizeDecision 最终决策
func (pe *PolicyEngineImpl) finalizeDecision(decision *PolicyDecision) {
	// 如果没有匹配的规则，使用默认动作
	if len(decision.MatchedRules) == 0 {
		decision.Action = pe.config.DefaultAction
		decision.Reason = "无匹配规则，使用默认动作"
		return
	}

	// 根据风险级别调整动作
	switch decision.RiskLevel {
	case analyzer.RiskLevelCritical:
		if decision.Action == PolicyActionAllow {
			decision.Action = PolicyActionBlock
			decision.Reason = "关键风险级别，强制阻断"
		}
	case analyzer.RiskLevelHigh:
		if decision.Action == PolicyActionAllow {
			decision.Action = PolicyActionAlert
			decision.Reason = "高风险级别，发出告警"
		}
	}

	// 设置默认原因
	if decision.Reason == "" {
		decision.Reason = fmt.Sprintf("匹配 %d 个规则", len(decision.MatchedRules))
	}
}

// updateStats 更新统计信息
func (pe *PolicyEngineImpl) updateStats(decision *PolicyDecision) {
	switch decision.Action {
	case PolicyActionAllow:
		atomic.AddUint64(&pe.stats.AllowedDecisions, 1)
	case PolicyActionBlock:
		atomic.AddUint64(&pe.stats.BlockedDecisions, 1)
	case PolicyActionAlert:
		atomic.AddUint64(&pe.stats.AlertDecisions, 1)
	case PolicyActionAudit:
		atomic.AddUint64(&pe.stats.AuditDecisions, 1)
	}

	// 更新平均处理时间
	pe.stats.AverageTime = (pe.stats.AverageTime + decision.ProcessingTime) / 2
}

// validateRule 验证规则
func (pe *PolicyEngineImpl) validateRule(rule *PolicyRule) error {
	if rule.ID == "" {
		return fmt.Errorf("规则ID不能为空")
	}

	if rule.Name == "" {
		return fmt.Errorf("规则名称不能为空")
	}

	if rule.Priority < 0 || rule.Priority > 100 {
		return fmt.Errorf("规则优先级必须在0-100之间")
	}

	if len(rule.Conditions) == 0 {
		return fmt.Errorf("规则必须包含至少一个条件")
	}

	if len(rule.Actions) == 0 {
		return fmt.Errorf("规则必须包含至少一个动作")
	}

	return nil
}

// loadDefaultRules 加载默认规则
func (pe *PolicyEngineImpl) loadDefaultRules() error {
	defaultRules := []*PolicyRule{
		{
			ID:          "block_high_risk",
			Name:        "阻断高风险内容",
			Description: "阻断包含高风险敏感信息的内容",
			Type:        "security",
			Priority:    90,
			Enabled:     true,
			Conditions: []*RuleCondition{
				{
					Field:    "analysis_result.risk_level",
					Operator: "equals",
					Value:    "critical",
					Type:     "string",
				},
			},
			Actions: []*RuleAction{
				{
					Type:       PolicyActionBlock,
					Parameters: map[string]interface{}{"reason": "高风险内容"},
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   "1.0",
		},
		{
			ID:          "alert_medium_risk",
			Name:        "告警中等风险内容",
			Description: "对包含中等风险敏感信息的内容发出告警",
			Type:        "security",
			Priority:    70,
			Enabled:     true,
			Conditions: []*RuleCondition{
				{
					Field:    "analysis_result.risk_level",
					Operator: "equals",
					Value:    "high",
					Type:     "string",
				},
			},
			Actions: []*RuleAction{
				{
					Type:       PolicyActionAlert,
					Parameters: map[string]interface{}{"reason": "中等风险内容"},
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   "1.0",
		},
		{
			ID:          "audit_all",
			Name:        "审计所有内容",
			Description: "对所有内容进行审计记录",
			Type:        "audit",
			Priority:    10,
			Enabled:     true,
			Conditions: []*RuleCondition{
				{
					Field:    "packet_info.protocol",
					Operator: "not_equals",
					Value:    "",
					Type:     "string",
				},
			},
			Actions: []*RuleAction{
				{
					Type:       PolicyActionAudit,
					Parameters: map[string]interface{}{"level": "info"},
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Version:   "1.0",
		},
	}

	return pe.LoadRules(defaultRules)
}
