package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// HealResult 修复结果
type HealResult struct {
	Success   bool                   // 是否成功
	Message   string                 // 消息
	Details   map[string]interface{} // 详细信息
	Timestamp time.Time              // 时间戳
	Duration  time.Duration          // 修复耗时
	CheckName string                 // 检查名称
	CheckType string                 // 检查类型
	Component string                 // 组件名称
	Error     error                  // 错误信息
}

// HealHistory 修复历史
type HealHistory struct {
	Check        HealthCheck   // 健康检查
	Results      []*HealResult // 修复结果
	LastResult   *HealResult   // 最后一次结果
	TotalHeals   int           // 总修复次数
	SuccessHeals int           // 成功修复次数
	FailedHeals  int           // 失败修复次数
	LastHealTime time.Time     // 最后一次修复时间
}

// SelfHealer 自我修复器
type SelfHealer struct {
	registry      *HealthCheckRegistry    // 健康检查注册表
	logger        hclog.Logger            // 日志记录器
	history       map[string]*HealHistory // 修复历史
	historyLimit  int                     // 历史限制
	mu            sync.RWMutex            // 互斥锁
	healingChecks map[string]bool         // 正在修复的检查
	healingMu     sync.RWMutex            // 修复互斥锁
}

// NewSelfHealer 创建自我修复器
func NewSelfHealer(registry *HealthCheckRegistry, logger hclog.Logger) *SelfHealer {
	return &SelfHealer{
		registry:      registry,
		logger:        logger.Named("self-healer"),
		history:       make(map[string]*HealHistory),
		historyLimit:  10,
		healingChecks: make(map[string]bool),
	}
}

// Heal 修复健康检查
func (h *SelfHealer) Heal(check HealthCheck) (*HealResult, error) {
	name := check.Name()

	// 检查是否可恢复
	if !check.IsRecoverable() {
		return nil, fmt.Errorf("健康检查 %s 不可恢复", name)
	}

	// 检查是否正在修复
	h.healingMu.Lock()
	if h.healingChecks[name] {
		h.healingMu.Unlock()
		return nil, fmt.Errorf("健康检查 %s 正在修复中", name)
	}
	h.healingChecks[name] = true
	h.healingMu.Unlock()

	// 确保在函数返回时清除修复状态
	defer func() {
		h.healingMu.Lock()
		delete(h.healingChecks, name)
		h.healingMu.Unlock()
	}()

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), check.Timeout())
	defer cancel()

	// 记录开始时间
	startTime := time.Now()

	// 执行修复
	h.logger.Info("开始修复", "name", name, "type", check.Type(), "component", check.Component())
	err := check.Recover(ctx)

	// 创建修复结果
	result := &HealResult{
		Success:   err == nil,
		Message:   "修复成功",
		Details:   make(map[string]interface{}),
		Timestamp: startTime,
		Duration:  time.Since(startTime),
		CheckName: name,
		CheckType: check.Type(),
		Component: check.Component(),
	}

	// 设置错误信息
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("修复失败: %s", err.Error())
		result.Error = err
	}

	// 更新修复历史
	h.updateHealHistory(check, result)

	// 记录修复结果
	if result.Success {
		h.logger.Info("修复成功",
			"name", name,
			"duration", result.Duration,
		)
	} else {
		h.logger.Error("修复失败",
			"name", name,
			"error", err,
			"duration", result.Duration,
		)
	}

	return result, err
}

// HealByName 根据名称修复健康检查
func (h *SelfHealer) HealByName(name string) (*HealResult, error) {
	// 获取健康检查
	check, exists := h.registry.Get(name)
	if !exists {
		return nil, fmt.Errorf("健康检查 %s 不存在", name)
	}

	// 执行修复
	return h.Heal(check)
}

// GetHealHistory 获取修复历史
func (h *SelfHealer) GetHealHistory(name string) (*HealHistory, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	history, exists := h.history[name]
	return history, exists
}

// GetAllHealHistory 获取所有修复历史
func (h *SelfHealer) GetAllHealHistory() map[string]*HealHistory {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 复制历史
	history := make(map[string]*HealHistory, len(h.history))
	for checkName, hist := range h.history {
		history[checkName] = hist
	}

	return history
}

// IsHealing 检查是否正在修复
func (h *SelfHealer) IsHealing(name string) bool {
	h.healingMu.RLock()
	defer h.healingMu.RUnlock()
	return h.healingChecks[name]
}

// SetHistoryLimit 设置历史限制
func (h *SelfHealer) SetHistoryLimit(limit int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.historyLimit = limit

	// 裁剪历史记录
	for _, history := range h.history {
		if len(history.Results) > h.historyLimit {
			history.Results = history.Results[len(history.Results)-h.historyLimit:]
		}
	}
}

// updateHealHistory 更新修复历史
func (h *SelfHealer) updateHealHistory(check HealthCheck, result *HealResult) {
	h.mu.Lock()
	defer h.mu.Unlock()

	name := check.Name()
	history, exists := h.history[name]
	if !exists {
		history = &HealHistory{
			Check:   check,
			Results: make([]*HealResult, 0, h.historyLimit),
		}
		h.history[name] = history
	}

	// 更新统计信息
	history.LastResult = result
	history.LastHealTime = time.Now()
	history.TotalHeals++
	if result.Success {
		history.SuccessHeals++
	} else {
		history.FailedHeals++
	}

	// 添加到历史记录
	if len(history.Results) >= h.historyLimit {
		// 移除最旧的记录
		history.Results = history.Results[1:]
	}
	history.Results = append(history.Results, result)
}

// ClearHealHistory 清空修复历史
func (h *SelfHealer) ClearHealHistory(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.history, name)
}

// ClearAllHealHistory 清空所有修复历史
func (h *SelfHealer) ClearAllHealHistory() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.history = make(map[string]*HealHistory)
}

// GetHealStats 获取修复统计信息
func (h *SelfHealer) GetHealStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	totalHeals := 0
	successHeals := 0
	failedHeals := 0
	healsByComponent := make(map[string]int)
	healsByType := make(map[string]int)

	for _, history := range h.history {
		totalHeals += history.TotalHeals
		successHeals += history.SuccessHeals
		failedHeals += history.FailedHeals

		component := history.Check.Component()
		healsByComponent[component] = healsByComponent[component] + history.TotalHeals

		checkType := history.Check.Type()
		healsByType[checkType] = healsByType[checkType] + history.TotalHeals
	}

	return map[string]interface{}{
		"total_heals":        totalHeals,
		"success_heals":      successHeals,
		"failed_heals":       failedHeals,
		"success_rate":       float64(successHeals) / float64(totalHeals) * 100,
		"heals_by_component": healsByComponent,
		"heals_by_type":      healsByType,
	}
}
