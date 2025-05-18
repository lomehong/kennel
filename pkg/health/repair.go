package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// RepairAction 修复动作接口
type RepairAction interface {
	// Execute 执行修复动作
	Execute(ctx context.Context) error
	// Name 返回修复动作名称
	Name() string
	// Description 返回修复动作描述
	Description() string
}

// RepairActionFunc 修复动作函数类型
type RepairActionFunc func(ctx context.Context) error

// SimpleRepairAction 简单修复动作
type SimpleRepairAction struct {
	name        string
	description string
	actionFunc  RepairActionFunc
}

// NewSimpleRepairAction 创建一个新的简单修复动作
func NewSimpleRepairAction(name, description string, actionFunc RepairActionFunc) *SimpleRepairAction {
	return &SimpleRepairAction{
		name:        name,
		description: description,
		actionFunc:  actionFunc,
	}
}

// Execute 执行修复动作
func (a *SimpleRepairAction) Execute(ctx context.Context) error {
	return a.actionFunc(ctx)
}

// Name 返回修复动作名称
func (a *SimpleRepairAction) Name() string {
	return a.name
}

// Description 返回修复动作描述
func (a *SimpleRepairAction) Description() string {
	return a.description
}

// RepairStrategy 修复策略接口
type RepairStrategy interface {
	// ShouldRepair 判断是否应该修复
	ShouldRepair(result CheckResult) bool
	// GetRepairAction 获取修复动作
	GetRepairAction(result CheckResult) RepairAction
	// Name 返回修复策略名称
	Name() string
}

// SimpleRepairStrategy 简单修复策略
type SimpleRepairStrategy struct {
	name         string
	shouldRepair func(result CheckResult) bool
	getAction    func(result CheckResult) RepairAction
}

// NewSimpleRepairStrategy 创建一个新的简单修复策略
func NewSimpleRepairStrategy(name string, shouldRepair func(result CheckResult) bool, getAction func(result CheckResult) RepairAction) *SimpleRepairStrategy {
	return &SimpleRepairStrategy{
		name:         name,
		shouldRepair: shouldRepair,
		getAction:    getAction,
	}
}

// ShouldRepair 判断是否应该修复
func (s *SimpleRepairStrategy) ShouldRepair(result CheckResult) bool {
	return s.shouldRepair(result)
}

// GetRepairAction 获取修复动作
func (s *SimpleRepairStrategy) GetRepairAction(result CheckResult) RepairAction {
	return s.getAction(result)
}

// Name 返回修复策略名称
func (s *SimpleRepairStrategy) Name() string {
	return s.name
}

// RepairResult 修复结果
type RepairResult struct {
	CheckerName string                 // 检查器名称
	ActionName  string                 // 修复动作名称
	Success     bool                   // 是否成功
	Message     string                 // 消息
	Details     map[string]interface{} // 详细信息
	Error       error                  // 错误信息
	StartTime   time.Time              // 开始时间
	EndTime     time.Time              // 结束时间
	Duration    time.Duration          // 耗时
}

// RepairSelfHealer 自我修复器
type RepairSelfHealer struct {
	registry       *CheckerRegistry
	strategies     map[string]RepairStrategy
	repairHistory  []RepairResult
	maxHistorySize int
	mu             sync.RWMutex
	logger         hclog.Logger
}

// NewRepairSelfHealer 创建一个新的自我修复器
func NewRepairSelfHealer(registry *CheckerRegistry, logger hclog.Logger) *RepairSelfHealer {
	return &RepairSelfHealer{
		registry:       registry,
		strategies:     make(map[string]RepairStrategy),
		repairHistory:  make([]RepairResult, 0),
		maxHistorySize: 100,
		logger:         logger,
	}
}

// RegisterStrategy 注册修复策略
func (h *RepairSelfHealer) RegisterStrategy(checkerName string, strategy RepairStrategy) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.strategies[checkerName] = strategy
	h.logger.Debug("注册修复策略", "checker", checkerName, "strategy", strategy.Name())
}

// UnregisterStrategy 注销修复策略
func (h *RepairSelfHealer) UnregisterStrategy(checkerName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.strategies, checkerName)
	h.logger.Debug("注销修复策略", "checker", checkerName)
}

// GetStrategy 获取修复策略
func (h *RepairSelfHealer) GetStrategy(checkerName string) (RepairStrategy, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	strategy, ok := h.strategies[checkerName]
	return strategy, ok
}

// CheckAndRepair 检查并修复
func (h *RepairSelfHealer) CheckAndRepair(ctx context.Context, checkerName string) (CheckResult, *RepairResult, error) {
	// 运行健康检查
	checkResult, ok := h.registry.RunCheck(ctx, checkerName)
	if !ok {
		return checkResult, nil, fmt.Errorf("health checker %s not found", checkerName)
	}

	// 如果健康，不需要修复
	if checkResult.Status == StatusHealthy {
		return checkResult, nil, nil
	}

	// 获取修复策略
	h.mu.RLock()
	strategy, ok := h.strategies[checkerName]
	h.mu.RUnlock()
	if !ok {
		return checkResult, nil, fmt.Errorf("no repair strategy for %s", checkerName)
	}

	// 判断是否应该修复
	if !strategy.ShouldRepair(checkResult) {
		return checkResult, nil, nil
	}

	// 获取修复动作
	action := strategy.GetRepairAction(checkResult)
	if action == nil {
		return checkResult, nil, fmt.Errorf("no repair action for %s", checkerName)
	}

	// 执行修复动作
	h.logger.Info("开始修复", "checker", checkerName, "action", action.Name())
	startTime := time.Now()
	err := action.Execute(ctx)
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// 记录修复结果
	repairResult := &RepairResult{
		CheckerName: checkerName,
		ActionName:  action.Name(),
		StartTime:   startTime,
		EndTime:     endTime,
		Duration:    duration,
	}

	if err != nil {
		h.logger.Error("修复失败", "checker", checkerName, "action", action.Name(), "error", err)
		repairResult.Success = false
		repairResult.Message = fmt.Sprintf("修复失败: %v", err)
		repairResult.Error = err
	} else {
		h.logger.Info("修复成功", "checker", checkerName, "action", action.Name(), "duration", duration)
		repairResult.Success = true
		repairResult.Message = "修复成功"
	}

	// 添加到历史记录
	h.mu.Lock()
	h.repairHistory = append(h.repairHistory, *repairResult)
	if len(h.repairHistory) > h.maxHistorySize {
		h.repairHistory = h.repairHistory[1:]
	}
	h.mu.Unlock()

	// 再次检查健康状态
	newCheckResult, _ := h.registry.RunCheck(ctx, checkerName)

	return newCheckResult, repairResult, err
}

// CheckAndRepairAll 检查并修复所有
func (h *RepairSelfHealer) CheckAndRepairAll(ctx context.Context) (map[string]CheckResult, map[string]*RepairResult, error) {
	// 运行所有健康检查
	checkResults := h.registry.RunChecks(ctx)
	repairResults := make(map[string]*RepairResult)
	var lastErr error

	// 遍历检查结果
	for checkerName, checkResult := range checkResults {
		// 如果健康，不需要修复
		if checkResult.Status == StatusHealthy {
			continue
		}

		// 获取修复策略
		h.mu.RLock()
		strategy, ok := h.strategies[checkerName]
		h.mu.RUnlock()
		if !ok {
			h.logger.Debug("没有修复策略", "checker", checkerName)
			continue
		}

		// 判断是否应该修复
		if !strategy.ShouldRepair(checkResult) {
			h.logger.Debug("不需要修复", "checker", checkerName)
			continue
		}

		// 获取修复动作
		action := strategy.GetRepairAction(checkResult)
		if action == nil {
			h.logger.Debug("没有修复动作", "checker", checkerName)
			continue
		}

		// 执行修复动作
		h.logger.Info("开始修复", "checker", checkerName, "action", action.Name())
		startTime := time.Now()
		err := action.Execute(ctx)
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// 记录修复结果
		repairResult := &RepairResult{
			CheckerName: checkerName,
			ActionName:  action.Name(),
			StartTime:   startTime,
			EndTime:     endTime,
			Duration:    duration,
		}

		if err != nil {
			h.logger.Error("修复失败", "checker", checkerName, "action", action.Name(), "error", err)
			repairResult.Success = false
			repairResult.Message = fmt.Sprintf("修复失败: %v", err)
			repairResult.Error = err
			lastErr = err
		} else {
			h.logger.Info("修复成功", "checker", checkerName, "action", action.Name(), "duration", duration)
			repairResult.Success = true
			repairResult.Message = "修复成功"
		}

		repairResults[checkerName] = repairResult

		// 添加到历史记录
		h.mu.Lock()
		h.repairHistory = append(h.repairHistory, *repairResult)
		if len(h.repairHistory) > h.maxHistorySize {
			h.repairHistory = h.repairHistory[1:]
		}
		h.mu.Unlock()

		// 再次检查健康状态
		newCheckResult, _ := h.registry.RunCheck(ctx, checkerName)
		checkResults[checkerName] = newCheckResult
	}

	return checkResults, repairResults, lastErr
}

// GetRepairHistory 获取修复历史
func (h *RepairSelfHealer) GetRepairHistory() []RepairResult {
	h.mu.RLock()
	defer h.mu.RUnlock()
	history := make([]RepairResult, len(h.repairHistory))
	copy(history, h.repairHistory)
	return history
}

// SetMaxHistorySize 设置最大历史记录大小
func (h *RepairSelfHealer) SetMaxHistorySize(size int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.maxHistorySize = size
	if len(h.repairHistory) > h.maxHistorySize {
		h.repairHistory = h.repairHistory[len(h.repairHistory)-h.maxHistorySize:]
	}
}
