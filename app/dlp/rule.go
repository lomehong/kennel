package main

import (
	"fmt"
	"regexp"
	"time"
)

// Rule 表示一个DLP规则
type Rule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Patterns    []string `json:"patterns"`
	Action      string   `json:"action"` // block, alert, log
	Enabled     bool     `json:"enabled"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// RuleManager 管理DLP规则
type RuleManager struct {
	rules         []Rule
	compiledRules map[string][]*regexp.Regexp
}

// NewRuleManager 创建一个新的规则管理器
func NewRuleManager() *RuleManager {
	manager := &RuleManager{
		rules:         make([]Rule, 0),
		compiledRules: make(map[string][]*regexp.Regexp),
	}

	// 添加默认规则
	manager.addDefaultRules()

	return manager
}

// LoadRules 从配置中加载规则
func (m *RuleManager) LoadRules(config map[string]interface{}) {
	// 如果配置中有规则，则加载
	if rulesConfig, ok := config["rules"].([]interface{}); ok {
		for _, ruleConfig := range rulesConfig {
			if ruleMap, ok := ruleConfig.(map[string]interface{}); ok {
				rule := Rule{}

				if id, ok := ruleMap["id"].(string); ok {
					rule.ID = id
				}

				if name, ok := ruleMap["name"].(string); ok {
					rule.Name = name
				}

				if desc, ok := ruleMap["description"].(string); ok {
					rule.Description = desc
				}

				if patterns, ok := ruleMap["patterns"].([]interface{}); ok {
					rule.Patterns = make([]string, 0, len(patterns))
					for _, p := range patterns {
						if pattern, ok := p.(string); ok {
							rule.Patterns = append(rule.Patterns, pattern)
						}
					}
				}

				if action, ok := ruleMap["action"].(string); ok {
					rule.Action = action
				}

				if enabled, ok := ruleMap["enabled"].(bool); ok {
					rule.Enabled = enabled
				}

				if createdAt, ok := ruleMap["created_at"].(string); ok {
					rule.CreatedAt = createdAt
				} else {
					rule.CreatedAt = time.Now().Format(time.RFC3339)
				}

				if updatedAt, ok := ruleMap["updated_at"].(string); ok {
					rule.UpdatedAt = updatedAt
				} else {
					rule.UpdatedAt = time.Now().Format(time.RFC3339)
				}

				m.rules = append(m.rules, rule)
				m.compileRule(rule)
			}
		}
	}
}

// GetRules 获取所有规则
func (m *RuleManager) GetRules() []Rule {
	return m.rules
}

// GetEnabledRules 获取所有启用的规则
func (m *RuleManager) GetEnabledRules() []Rule {
	enabledRules := make([]Rule, 0)
	for _, rule := range m.rules {
		if rule.Enabled {
			enabledRules = append(enabledRules, rule)
		}
	}
	return enabledRules
}

// GetCompiledRules 获取编译后的规则
func (m *RuleManager) GetCompiledRules() map[string][]*regexp.Regexp {
	return m.compiledRules
}

// ListRules 列出所有规则
func (m *RuleManager) ListRules() (map[string]interface{}, error) {
	return map[string]interface{}{
		"rules": m.rules,
		"count": len(m.rules),
	}, nil
}

// AddRule 添加规则
func (m *RuleManager) AddRule(params map[string]interface{}) (map[string]interface{}, error) {
	// 提取参数
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("规则名称不能为空")
	}

	description, _ := params["description"].(string)

	var patterns []string
	if patternsParam, ok := params["patterns"].([]interface{}); ok {
		patterns = make([]string, 0, len(patternsParam))
		for _, p := range patternsParam {
			if pattern, ok := p.(string); ok && pattern != "" {
				patterns = append(patterns, pattern)
			}
		}
	}

	if len(patterns) == 0 {
		return nil, fmt.Errorf("规则模式不能为空")
	}

	action, ok := params["action"].(string)
	if !ok || action == "" {
		action = "log" // 默认动作
	}

	// 验证动作是否有效
	if action != "block" && action != "alert" && action != "log" {
		return nil, fmt.Errorf("无效的动作: %s", action)
	}

	// 创建规则
	now := time.Now().Format(time.RFC3339)
	rule := Rule{
		ID:          fmt.Sprintf("rule%03d", len(m.rules)+1),
		Name:        name,
		Description: description,
		Patterns:    patterns,
		Action:      action,
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 编译规则
	err := m.compileRule(rule)
	if err != nil {
		return nil, fmt.Errorf("编译规则失败: %v", err)
	}

	// 添加规则
	m.rules = append(m.rules, rule)

	return map[string]interface{}{
		"rule": rule,
	}, nil
}

// DeleteRule 删除规则
func (m *RuleManager) DeleteRule(id string) (map[string]interface{}, error) {
	// 查找规则
	index := -1
	for i, rule := range m.rules {
		if rule.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return nil, fmt.Errorf("规则不存在: %s", id)
	}

	// 删除规则
	deletedRule := m.rules[index]
	m.rules = append(m.rules[:index], m.rules[index+1:]...)

	// 删除编译后的规则
	delete(m.compiledRules, id)

	return map[string]interface{}{
		"success": true,
		"rule":    deletedRule,
	}, nil
}

// 编译规则
func (m *RuleManager) compileRule(rule Rule) error {
	compiledPatterns := make([]*regexp.Regexp, 0, len(rule.Patterns))

	for _, pattern := range rule.Patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("编译正则表达式失败: %s, %v", pattern, err)
		}
		compiledPatterns = append(compiledPatterns, re)
	}

	m.compiledRules[rule.ID] = compiledPatterns
	return nil
}

// 添加默认规则
func (m *RuleManager) addDefaultRules() {
	now := time.Now().Format(time.RFC3339)

	// 身份证号码规则
	idCardRule := Rule{
		ID:          "rule001",
		Name:        "身份证号码",
		Description: "检测中国大陆居民身份证号码",
		Patterns:    []string{`\d{17}[\dXx]`, `\d{15}`},
		Action:      "block",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.rules = append(m.rules, idCardRule)
	m.compileRule(idCardRule)

	// 信用卡号码规则
	creditCardRule := Rule{
		ID:          "rule002",
		Name:        "信用卡号码",
		Description: "检测信用卡号码",
		Patterns:    []string{`\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}`},
		Action:      "alert",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.rules = append(m.rules, creditCardRule)
	m.compileRule(creditCardRule)

	// 手机号码规则
	phoneRule := Rule{
		ID:          "rule003",
		Name:        "手机号码",
		Description: "检测中国大陆手机号码",
		Patterns:    []string{`1[3-9]\d{9}`},
		Action:      "log",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.rules = append(m.rules, phoneRule)
	m.compileRule(phoneRule)
}
