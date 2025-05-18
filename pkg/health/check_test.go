package health

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestBaseHealthCheck(t *testing.T) {
	// 创建基础健康检查
	check := NewBaseHealthCheck(
		"test_check",
		"test_type",
		"test_component",
		5*time.Second,
		30*time.Second,
		3,
		1,
		true,
		func(ctx context.Context) *HealthCheckResult {
			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "测试成功",
				Details: map[string]interface{}{
					"test": true,
				},
			}
		},
		func(ctx context.Context) error {
			return nil
		},
	)

	// 验证基本属性
	assert.Equal(t, "test_check", check.Name())
	assert.Equal(t, "test_type", check.Type())
	assert.Equal(t, "test_component", check.Component())
	assert.Equal(t, 5*time.Second, check.Timeout())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.Equal(t, 3, check.FailureThreshold())
	assert.Equal(t, 1, check.SuccessThreshold())
	assert.True(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.Equal(t, HealthStatusHealthy, result.Status)
	assert.Equal(t, "测试成功", result.Message)
	assert.Equal(t, "test_check", result.CheckName)
	assert.Equal(t, "test_type", result.CheckType)
	assert.Equal(t, "test_component", result.Component)
	assert.True(t, result.Recoverable)
	assert.Equal(t, true, result.Details["test"])

	// 执行恢复
	err := check.Recover(context.Background())
	assert.NoError(t, err)
}

func TestHealthCheckRegistry(t *testing.T) {
	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建健康检查注册表
	registry := NewHealthCheckRegistry(logger)

	// 创建健康检查
	check1 := NewBaseHealthCheck(
		"check1",
		"type1",
		"component1",
		5*time.Second,
		30*time.Second,
		3,
		1,
		true,
		func(ctx context.Context) *HealthCheckResult {
			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "检查1成功",
			}
		},
		nil,
	)

	check2 := NewBaseHealthCheck(
		"check2",
		"type2",
		"component2",
		5*time.Second,
		30*time.Second,
		3,
		1,
		false,
		func(ctx context.Context) *HealthCheckResult {
			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "检查2成功",
			}
		},
		nil,
	)

	check3 := NewBaseHealthCheck(
		"check3",
		"type1",
		"component2",
		5*time.Second,
		30*time.Second,
		3,
		1,
		true,
		func(ctx context.Context) *HealthCheckResult {
			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "检查3成功",
			}
		},
		nil,
	)

	// 注册健康检查
	err := registry.Register(check1)
	assert.NoError(t, err)
	err = registry.Register(check2)
	assert.NoError(t, err)
	err = registry.Register(check3)
	assert.NoError(t, err)

	// 验证注册表
	assert.Equal(t, 3, registry.Count())

	// 获取健康检查
	check, exists := registry.Get("check1")
	assert.True(t, exists)
	assert.Equal(t, "check1", check.Name())

	// 获取不存在的健康检查
	check, exists = registry.Get("not_exists")
	assert.False(t, exists)
	assert.Nil(t, check)

	// 获取所有健康检查
	checks := registry.GetAll()
	assert.Equal(t, 3, len(checks))
	assert.Contains(t, checks, "check1")
	assert.Contains(t, checks, "check2")
	assert.Contains(t, checks, "check3")

	// 获取指定类型的健康检查
	typeChecks := registry.GetByType("type1")
	assert.Equal(t, 2, len(typeChecks))
	assert.Equal(t, "check1", typeChecks[0].Name())
	assert.Equal(t, "check3", typeChecks[1].Name())

	// 获取指定组件的健康检查
	componentChecks := registry.GetByComponent("component2")
	assert.Equal(t, 2, len(componentChecks))
	assert.Equal(t, "check2", componentChecks[0].Name())
	assert.Equal(t, "check3", componentChecks[1].Name())

	// 注销健康检查
	err = registry.Unregister("check1")
	assert.NoError(t, err)
	assert.Equal(t, 2, registry.Count())

	// 注销不存在的健康检查
	err = registry.Unregister("not_exists")
	assert.Error(t, err)

	// 清空健康检查
	registry.Clear()
	assert.Equal(t, 0, registry.Count())
}

func TestHealthCheckResult(t *testing.T) {
	// 创建健康检查结果
	result := &HealthCheckResult{
		Status:      HealthStatusHealthy,
		Message:     "测试成功",
		Details:     map[string]interface{}{"test": true},
		Timestamp:   time.Now(),
		Duration:    100 * time.Millisecond,
		CheckName:   "test_check",
		CheckType:   "test_type",
		Component:   "test_component",
		Error:       nil,
		Recoverable: true,
	}

	// 验证结果
	assert.Equal(t, HealthStatusHealthy, result.Status)
	assert.Equal(t, "测试成功", result.Message)
	assert.Equal(t, true, result.Details["test"])
	assert.Equal(t, "test_check", result.CheckName)
	assert.Equal(t, "test_type", result.CheckType)
	assert.Equal(t, "test_component", result.Component)
	assert.Nil(t, result.Error)
	assert.True(t, result.Recoverable)
	assert.Equal(t, 100*time.Millisecond, result.Duration)
}

func TestHealthCheckWithNoCheckFunc(t *testing.T) {
	// 创建没有检查函数的健康检查
	check := NewBaseHealthCheck(
		"test_check",
		"test_type",
		"test_component",
		5*time.Second,
		30*time.Second,
		3,
		1,
		true,
		nil,
		nil,
	)

	// 执行检查
	result := check.Check(context.Background())
	assert.Equal(t, HealthStatusUnknown, result.Status)
	assert.Equal(t, "检查函数未定义", result.Message)
	assert.Equal(t, "test_check", result.CheckName)
	assert.Equal(t, "test_type", result.CheckType)
	assert.Equal(t, "test_component", result.Component)
	assert.True(t, result.Recoverable)
}

func TestHealthCheckNotRecoverable(t *testing.T) {
	// 创建不可恢复的健康检查
	check := NewBaseHealthCheck(
		"test_check",
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
			}
		},
		nil,
	)

	// 验证不可恢复
	assert.False(t, check.IsRecoverable())

	// 尝试恢复
	err := check.Recover(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不可恢复")
}

func TestHealthCheckRecoverableButNoRecoverFunc(t *testing.T) {
	// 创建可恢复但没有恢复函数的健康检查
	check := NewBaseHealthCheck(
		"test_check",
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
		nil,
	)

	// 验证不可恢复
	assert.False(t, check.IsRecoverable())

	// 尝试恢复
	err := check.Recover(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不可恢复")
}
