package main

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/lomehong/kennel/pkg/sdk/go"
)

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
	logger sdk.Logger
	rules  map[string]*DLPRule
	mu     sync.RWMutex
}

// NewRuleManager 创建一个新的规则管理器
func NewRuleManager(logger sdk.Logger) *RuleManager {
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
	
	// 获取规则配置
	rulesConfig := sdk.GetConfigSlice(config, "rules")
	if len(rulesConfig) == 0 {
		m.logger.Warn("未找到规则配置")
		return nil
	}
	
	// 加载规则
	for _, ruleConfig := range rulesConfig {
		ruleMap, ok := ruleConfig.(map[string]interface{})
		if !ok {
			m.logger.Warn("规则配置格式错误", "rule", ruleConfig)
			continue
		}
		
		// 创建规则
		rule := &DLPRule{
			ID:          sdk.GetConfigString(ruleMap, "id", ""),
			Name:        sdk.GetConfigString(ruleMap, "name", ""),
			Description: sdk.GetConfigString(ruleMap, "description", ""),
			Pattern:     sdk.GetConfigString(ruleMap, "pattern", ""),
			Action:      sdk.GetConfigString(ruleMap, "action", "alert"),
			Enabled:     sdk.GetConfigBool(ruleMap, "enabled", true),
		}
		
		// 检查必要字段
		if rule.ID == "" || rule.Pattern == "" {
			m.logger.Warn("规则缺少必要字段", "rule", rule)
			continue
		}
		
		// 编译正则表达式
		regex, err := regexp.Compile(rule.Pattern)
		if err != nil {
			m.logger.Error("编译正则表达式失败", "pattern", rule.Pattern, "error", err)
			continue
		}
		rule.regex = regex
		
		// 添加规则
		m.rules[rule.ID] = rule
		m.logger.Debug("加载规则", "id", rule.ID, "name", rule.Name)
	}
	
	m.logger.Info("加载规则完成", "count", len(m.rules))
	return nil
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
