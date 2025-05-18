package health

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestSimpleRepairAction(t *testing.T) {
	// 创建一个简单的修复动作
	executed := false
	action := NewSimpleRepairAction(
		"test_action",
		"测试动作",
		func(ctx context.Context) error {
			executed = true
			return nil
		},
	)

	// 验证动作属性
	assert.Equal(t, "test_action", action.Name())
	assert.Equal(t, "测试动作", action.Description())

	// 执行动作
	err := action.Execute(context.Background())
	assert.NoError(t, err)
	assert.True(t, executed)

	// 创建一个返回错误的动作
	expectedErr := fmt.Errorf("测试错误")
	action = NewSimpleRepairAction(
		"error_action",
		"错误动作",
		func(ctx context.Context) error {
			return expectedErr
		},
	)

	// 执行动作
	err = action.Execute(context.Background())
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestSimpleRepairStrategy(t *testing.T) {
	// 创建一个简单的修复策略
	action := NewSimpleRepairAction(
		"test_action",
		"测试动作",
		func(ctx context.Context) error {
			return nil
		},
	)

	strategy := NewSimpleRepairStrategy(
		"test_strategy",
		func(result CheckResult) bool {
			return result.Status == StatusUnhealthy
		},
		func(result CheckResult) RepairAction {
			return action
		},
	)

	// 验证策略属性
	assert.Equal(t, "test_strategy", strategy.Name())

	// 测试应该修复的情况
	result := CheckResult{
		Status:  StatusUnhealthy,
		Message: "测试失败",
	}
	assert.True(t, strategy.ShouldRepair(result))
	assert.Equal(t, action, strategy.GetRepairAction(result))

	// 测试不应该修复的情况
	result = CheckResult{
		Status:  StatusHealthy,
		Message: "测试成功",
	}
	assert.False(t, strategy.ShouldRepair(result))
}

func TestSelfHealer(t *testing.T) {
	logger := hclog.NewNullLogger()
	registry := NewHealthCheckRegistry(logger)
	healer := NewSelfHealer(registry, logger)

	// 创建检查器
	checker := NewSimpleChecker(
		"test_checker",
		"测试检查器",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "测试失败",
				Details: map[string]interface{}{"test": false},
			}
		},
	)
	registry.RegisterChecker(checker)

	// 创建修复动作
	repaired := false
	action := NewSimpleRepairAction(
		"test_action",
		"测试动作",
		func(ctx context.Context) error {
			repaired = true
			return nil
		},
	)

	// 创建修复策略
	strategy := NewSimpleRepairStrategy(
		"test_strategy",
		func(result CheckResult) bool {
			return result.Status == StatusUnhealthy
		},
		func(result CheckResult) RepairAction {
			return action
		},
	)

	// 注册修复策略
	healer.RegisterStrategy("test_checker", strategy)

	// 获取修复策略
	s, ok := healer.GetStrategy("test_checker")
	assert.True(t, ok)
	assert.Equal(t, "test_strategy", s.Name())

	// 检查并修复
	checkResult, repairResult, err := healer.CheckAndRepair(context.Background(), "test_checker")
	assert.NoError(t, err)
	assert.Equal(t, StatusUnhealthy, checkResult.Status)
	assert.NotNil(t, repairResult)
	assert.True(t, repairResult.Success)
	assert.Equal(t, "test_action", repairResult.ActionName)
	assert.True(t, repaired)

	// 获取修复历史
	history := healer.GetRepairHistory()
	assert.Len(t, history, 1)
	assert.Equal(t, "test_checker", history[0].CheckerName)
	assert.Equal(t, "test_action", history[0].ActionName)
	assert.True(t, history[0].Success)

	// 设置最大历史记录大小
	healer.SetMaxHistorySize(10)

	// 注销修复策略
	healer.UnregisterStrategy("test_checker")
	_, ok = healer.GetStrategy("test_checker")
	assert.False(t, ok)
}

func TestCheckAndRepairAll(t *testing.T) {
	logger := hclog.NewNullLogger()
	registry := NewHealthCheckRegistry(logger)
	healer := NewSelfHealer(registry, logger)

	// 创建检查器
	checker1 := NewSimpleChecker(
		"test_checker_1",
		"测试检查器1",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "测试1失败",
				Details: map[string]interface{}{"test1": false},
			}
		},
	)

	checker2 := NewSimpleChecker(
		"test_checker_2",
		"测试检查器2",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusHealthy,
				Message: "测试2成功",
				Details: map[string]interface{}{"test2": true},
			}
		},
	)

	checker3 := NewSimpleChecker(
		"test_checker_3",
		"测试检查器3",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusDegraded,
				Message: "测试3降级",
				Details: map[string]interface{}{"test3": "degraded"},
			}
		},
	)

	registry.RegisterChecker(checker1)
	registry.RegisterChecker(checker2)
	registry.RegisterChecker(checker3)

	// 创建修复动作
	repaired1 := false
	action1 := NewSimpleRepairAction(
		"test_action_1",
		"测试动作1",
		func(ctx context.Context) error {
			repaired1 = true
			return nil
		},
	)

	repaired3 := false
	action3 := NewSimpleRepairAction(
		"test_action_3",
		"测试动作3",
		func(ctx context.Context) error {
			repaired3 = true
			return nil
		},
	)

	// 创建修复策略
	strategy1 := NewSimpleRepairStrategy(
		"test_strategy_1",
		func(result CheckResult) bool {
			return result.Status == StatusUnhealthy
		},
		func(result CheckResult) RepairAction {
			return action1
		},
	)

	strategy3 := NewSimpleRepairStrategy(
		"test_strategy_3",
		func(result CheckResult) bool {
			return result.Status == StatusDegraded
		},
		func(result CheckResult) RepairAction {
			return action3
		},
	)

	// 注册修复策略
	healer.RegisterStrategy("test_checker_1", strategy1)
	healer.RegisterStrategy("test_checker_3", strategy3)

	// 检查并修复所有
	checkResults, repairResults, err := healer.CheckAndRepairAll(context.Background())
	assert.NoError(t, err)
	assert.Len(t, checkResults, 3)
	assert.Len(t, repairResults, 2)
	assert.True(t, repaired1)
	assert.True(t, repaired3)
	assert.True(t, repairResults["test_checker_1"].Success)
	assert.True(t, repairResults["test_checker_3"].Success)

	// 获取修复历史
	history := healer.GetRepairHistory()
	assert.Len(t, history, 2)
}
