package health

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestSelfHealer(t *testing.T) {
	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建健康检查注册表
	registry := NewHealthCheckRegistry(logger)

	// 创建自我修复器
	healer := NewSelfHealer(registry, logger)

	// 创建可恢复的健康检查
	recoverableCheck := NewBaseHealthCheck(
		"recoverable_check",
		"test_type",
		"test_component",
		5*time.Second,
		30*time.Second,
		3,
		1,
		true,
		func(ctx context.Context) *HealthCheckResult {
			return &HealthCheckResult{
				Status:  HealthStatusUnhealthy,
				Message: "测试失败",
				Details: map[string]interface{}{
					"test": true,
				},
			}
		},
		func(ctx context.Context) error {
			return nil
		},
	)

	// 创建不可恢复的健康检查
	unrecoverableCheck := NewBaseHealthCheck(
		"unrecoverable_check",
		"test_type",
		"test_component",
		5*time.Second,
		30*time.Second,
		3,
		1,
		false,
		func(ctx context.Context) *HealthCheckResult {
			return &HealthCheckResult{
				Status:  HealthStatusUnhealthy,
				Message: "测试失败",
				Details: map[string]interface{}{
					"test": true,
				},
			}
		},
		nil,
	)

	// 创建恢复失败的健康检查
	failingCheck := NewBaseHealthCheck(
		"failing_check",
		"test_type",
		"test_component",
		5*time.Second,
		30*time.Second,
		3,
		1,
		true,
		func(ctx context.Context) *HealthCheckResult {
			return &HealthCheckResult{
				Status:  HealthStatusUnhealthy,
				Message: "测试失败",
				Details: map[string]interface{}{
					"test": true,
				},
			}
		},
		func(ctx context.Context) error {
			return fmt.Errorf("恢复失败")
		},
	)

	// 注册健康检查
	err := registry.Register(recoverableCheck)
	assert.NoError(t, err)
	err = registry.Register(unrecoverableCheck)
	assert.NoError(t, err)
	err = registry.Register(failingCheck)
	assert.NoError(t, err)

	// 修复可恢复的健康检查
	result, err := healer.Heal(recoverableCheck)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "修复成功", result.Message)
	assert.Equal(t, "recoverable_check", result.CheckName)
	assert.Equal(t, "test_type", result.CheckType)
	assert.Equal(t, "test_component", result.Component)

	// 修复不可恢复的健康检查
	result, err = healer.Heal(unrecoverableCheck)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不可恢复")

	// 修复失败的健康检查
	result, err = healer.Heal(failingCheck)
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "修复失败")
	assert.Equal(t, "failing_check", result.CheckName)
	assert.Equal(t, "test_type", result.CheckType)
	assert.Equal(t, "test_component", result.Component)
	assert.Equal(t, "恢复失败", result.Error.Error())

	// 根据名称修复健康检查
	result, err = healer.HealByName("recoverable_check")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// 修复不存在的健康检查
	result, err = healer.HealByName("not_exists")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不存在")

	// 获取修复历史
	history, exists := healer.GetHealHistory("recoverable_check")
	assert.True(t, exists)
	assert.NotNil(t, history)
	assert.Equal(t, 2, history.TotalHeals)
	assert.Equal(t, 2, history.SuccessHeals)
	assert.Equal(t, 0, history.FailedHeals)
	assert.Equal(t, 2, len(history.Results))
	assert.Equal(t, "recoverable_check", history.Check.Name())

	// 获取不存在的修复历史
	history, exists = healer.GetHealHistory("not_exists")
	assert.False(t, exists)
	assert.Nil(t, history)

	// 获取所有修复历史
	allHistory := healer.GetAllHealHistory()
	assert.Equal(t, 2, len(allHistory))
	assert.Contains(t, allHistory, "recoverable_check")
	assert.Contains(t, allHistory, "failing_check")

	// 检查是否正在修复
	assert.False(t, healer.IsHealing("recoverable_check"))

	// 设置历史限制
	healer.SetHistoryLimit(5)

	// 清空修复历史
	healer.ClearHealHistory("recoverable_check")
	history, exists = healer.GetHealHistory("recoverable_check")
	assert.False(t, exists)
	assert.Nil(t, history)

	// 清空所有修复历史
	healer.ClearAllHealHistory()
	allHistory = healer.GetAllHealHistory()
	assert.Equal(t, 0, len(allHistory))

	// 获取修复统计信息
	stats := healer.GetHealStats()
	assert.Equal(t, 0, stats["total_heals"])
	assert.Equal(t, 0, stats["success_heals"])
	assert.Equal(t, 0, stats["failed_heals"])
}

func TestConcurrentHealing(t *testing.T) {
	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建健康检查注册表
	registry := NewHealthCheckRegistry(logger)

	// 创建自我修复器
	healer := NewSelfHealer(registry, logger)

	// 创建耗时的健康检查
	slowCheck := NewBaseHealthCheck(
		"slow_check",
		"test_type",
		"test_component",
		5*time.Second,
		30*time.Second,
		3,
		1,
		true,
		func(ctx context.Context) *HealthCheckResult {
			return &HealthCheckResult{
				Status:  HealthStatusUnhealthy,
				Message: "测试失败",
			}
		},
		func(ctx context.Context) error {
			time.Sleep(500 * time.Millisecond)
			return nil
		},
	)

	// 注册健康检查
	err := registry.Register(slowCheck)
	assert.NoError(t, err)

	// 启动第一个修复
	go func() {
		healer.Heal(slowCheck)
	}()

	// 等待第一个修复开始
	time.Sleep(100 * time.Millisecond)

	// 尝试同时修复同一个健康检查
	result, err := healer.Heal(slowCheck)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "正在修复中")

	// 等待第一个修复完成
	time.Sleep(500 * time.Millisecond)

	// 现在应该可以修复了
	result, err = healer.Heal(slowCheck)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}
