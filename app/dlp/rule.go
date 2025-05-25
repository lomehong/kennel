package main

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/lomehong/kennel/pkg/logging"
)

// 辅助函数，用于从配置中获取字符串值
func getConfigString(config map[string]interface{}, key, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// 辅助函数，用于从配置中获取布尔值
func getConfigBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := config[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// 辅助函数，用于从配置中获取切片
func getConfigSlice(config map[string]interface{}, key string) []interface{} {
	if val, ok := config[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			return slice
		}
	}
	return nil
}

// DLPRule 表示一个数据防泄漏规则
type DLPRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
	Action      string `json:"action"`
	Enabled     bool   `json:"enabled"`

	// 编译后的正则表达式
	regex *regexp.Regexp
}

// RuleManager 规则管理器
type RuleManager struct {
	logger logging.Logger
	rules  map[string]*DLPRule
	mu     sync.RWMutex
}

// NewRuleManager 创建一个新的规则管理器
func NewRuleManager(logger logging.Logger) *RuleManager {
	return &RuleManager{
		logger: logger,
		rules:  make(map[string]*DLPRule),
	}
}

// LoadRules 从配置加载规则
func (m *RuleManager) LoadRules(config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 清除现有规则
	m.rules = make(map[string]*DLPRule)

	// 首先尝试从配置中加载规则
	rulesConfig := getConfigSlice(config, "rules")
	if len(rulesConfig) > 0 {
		m.logger.Info("从配置中加载规则")
		return m.loadRulesFromConfig(rulesConfig)
	}

	// 如果配置中没有规则，尝试从文件加载
	m.logger.Info("配置中未找到规则，尝试从文件加载")
	return m.loadRulesFromFile()
}

// loadRulesFromConfig 从配置中加载规则
func (m *RuleManager) loadRulesFromConfig(rulesConfig []interface{}) error {
	// 加载规则
	for _, ruleConfig := range rulesConfig {
		ruleMap, ok := ruleConfig.(map[string]interface{})
		if !ok {
			m.logger.Warn("规则配置格式错误", "rule", ruleConfig)
			continue
		}

		// 创建规则
		rule := &DLPRule{
			ID:          getConfigString(ruleMap, "id", ""),
			Name:        getConfigString(ruleMap, "name", ""),
			Description: getConfigString(ruleMap, "description", ""),
			Pattern:     getConfigString(ruleMap, "pattern", ""),
			Action:      getConfigString(ruleMap, "action", "alert"),
			Enabled:     getConfigBool(ruleMap, "enabled", true),
		}

		if err := m.addRuleInternal(rule); err != nil {
			m.logger.Error("添加规则失败", "rule", rule.ID, "error", err)
		}
	}

	m.logger.Info("从配置加载规则完成", "count", len(m.rules))
	return nil
}

// loadRulesFromFile 从文件加载规则
func (m *RuleManager) loadRulesFromFile() error {
	// 尝试加载默认规则
	defaultRules := m.getDefaultRules()
	for _, rule := range defaultRules {
		if err := m.addRuleInternal(rule); err != nil {
			m.logger.Error("添加默认规则失败", "rule", rule.ID, "error", err)
		}
	}

	m.logger.Info("加载默认规则完成", "count", len(m.rules))
	return nil
}

// addRuleInternal 内部添加规则方法（不加锁）
func (m *RuleManager) addRuleInternal(rule *DLPRule) error {
	// 检查必要字段
	if rule.ID == "" || rule.Pattern == "" {
		return fmt.Errorf("规则缺少必要字段: ID=%s, Pattern=%s", rule.ID, rule.Pattern)
	}

	// 编译正则表达式
	regex, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("编译正则表达式失败: %w", err)
	}
	rule.regex = regex

	// 添加规则
	m.rules[rule.ID] = rule
	m.logger.Debug("加载规则", "id", rule.ID, "name", rule.Name)

	return nil
}

// getDefaultRules 获取默认规则
func (m *RuleManager) getDefaultRules() []*DLPRule {
	return []*DLPRule{
		{
			ID:          "credit_card",
			Name:        "信用卡号检测",
			Description: "检测信用卡号码",
			Pattern:     `\b(?:\d{4}[-\s]?){3}\d{4}\b`,
			Action:      "block",
			Enabled:     true,
		},
		{
			ID:          "ssn",
			Name:        "社会保障号检测",
			Description: "检测社会保障号码",
			Pattern:     `\b\d{3}-\d{2}-\d{4}\b`,
			Action:      "alert",
			Enabled:     true,
		},
		{
			ID:          "email",
			Name:        "邮箱地址检测",
			Description: "检测邮箱地址",
			Pattern:     `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
			Action:      "audit",
			Enabled:     true,
		},
		{
			ID:          "phone",
			Name:        "电话号码检测",
			Description: "检测电话号码",
			Pattern:     `\b(?:\+86[-\s]?)?(?:1[3-9]\d{9}|\d{3,4}[-\s]?\d{7,8})\b`,
			Action:      "alert",
			Enabled:     true,
		},
		{
			ID:          "id_card",
			Name:        "身份证号检测",
			Description: "检测身份证号码",
			Pattern:     `\b[1-9]\d{5}(?:18|19|20)\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3}[\dXx]\b`,
			Action:      "block",
			Enabled:     true,
		},
		{
			ID:          "password",
			Name:        "密码检测",
			Description: "检测可能的密码字段",
			Pattern:     `(?i)(?:password|pwd|pass|secret|key)[:=]\s*[^\s]+`,
			Action:      "audit",
			Enabled:     true,
		},
	}
}

// GetRule 获取规则
func (m *RuleManager) GetRule(id string) (*DLPRule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rule, ok := m.rules[id]
	return rule, ok
}

// GetRules 获取所有规则
func (m *RuleManager) GetRules() []*DLPRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回一个副本，避免外部修改
	rules := make([]*DLPRule, 0, len(m.rules))
	for _, rule := range m.rules {
		rules = append(rules, rule)
	}

	return rules
}

// AddRule 添加规则
func (m *RuleManager) AddRule(rule *DLPRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查规则ID是否已存在
	if _, exists := m.rules[rule.ID]; exists {
		return fmt.Errorf("规则ID已存在: %s", rule.ID)
	}

	// 编译正则表达式
	regex, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("编译正则表达式失败: %w", err)
	}
	rule.regex = regex

	// 添加规则
	m.rules[rule.ID] = rule
	m.logger.Info("添加规则", "id", rule.ID, "name", rule.Name)

	return nil
}

// UpdateRule 更新规则
func (m *RuleManager) UpdateRule(rule *DLPRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查规则ID是否存在
	if _, exists := m.rules[rule.ID]; !exists {
		return fmt.Errorf("规则ID不存在: %s", rule.ID)
	}

	// 编译正则表达式
	regex, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("编译正则表达式失败: %w", err)
	}
	rule.regex = regex

	// 更新规则
	m.rules[rule.ID] = rule
	m.logger.Info("更新规则", "id", rule.ID, "name", rule.Name)

	return nil
}

// DeleteRule 删除规则
func (m *RuleManager) DeleteRule(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查规则ID是否存在
	if _, exists := m.rules[id]; !exists {
		return fmt.Errorf("规则ID不存在: %s", id)
	}

	// 删除规则
	delete(m.rules, id)
	m.logger.Info("删除规则", "id", id)

	return nil
}

// EnableRule 启用规则
func (m *RuleManager) EnableRule(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查规则ID是否存在
	rule, exists := m.rules[id]
	if !exists {
		return fmt.Errorf("规则ID不存在: %s", id)
	}

	// 启用规则
	rule.Enabled = true
	m.logger.Info("启用规则", "id", id, "name", rule.Name)

	return nil
}

// DisableRule 禁用规则
func (m *RuleManager) DisableRule(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查规则ID是否存在
	rule, exists := m.rules[id]
	if !exists {
		return fmt.Errorf("规则ID不存在: %s", id)
	}

	// 禁用规则
	rule.Enabled = false
	m.logger.Info("禁用规则", "id", id, "name", rule.Name)

	return nil
}

// RuleToMap 将规则转换为map
func RuleToMap(rule *DLPRule) map[string]interface{} {
	return map[string]interface{}{
		"id":          rule.ID,
		"name":        rule.Name,
		"description": rule.Description,
		"pattern":     rule.Pattern,
		"action":      rule.Action,
		"enabled":     rule.Enabled,
	}
}

// RulesToMap 将规则列表转换为map列表
func RulesToMap(rules []*DLPRule) []map[string]interface{} {
	result := make([]map[string]interface{}, len(rules))
	for i, rule := range rules {
		result[i] = RuleToMap(rule)
	}
	return result
}
