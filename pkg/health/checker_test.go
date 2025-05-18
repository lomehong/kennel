package health

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestSimpleChecker(t *testing.T) {
	// 创建一个简单的健康检查器
	checker := NewSimpleChecker(
		"test_checker",
		"测试检查器",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusHealthy,
				Message: "测试成功",
				Details: map[string]interface{}{
					"test": true,
				},
			}
		},
	)

	// 验证检查器属性
	assert.Equal(t, "test_checker", checker.Name())
	assert.Equal(t, "测试检查器", checker.Description())
	assert.Equal(t, "test", checker.Type())

	// 执行检查
	result := checker.Check(context.Background())
	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, "测试成功", result.Message)
	assert.Equal(t, true, result.Details["test"])
}

func TestCompositeChecker(t *testing.T) {
	// 创建两个简单的健康检查器
	checker1 := NewSimpleChecker(
		"test_checker_1",
		"测试检查器1",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusHealthy,
				Message: "测试1成功",
				Details: map[string]interface{}{
					"test1": true,
				},
			}
		},
	)

	checker2 := NewSimpleChecker(
		"test_checker_2",
		"测试检查器2",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "测试2失败",
				Details: map[string]interface{}{
					"test2": false,
				},
			}
		},
	)

	// 创建组合检查器
	compositeChecker := NewCompositeChecker(
		"composite_checker",
		"组合检查器",
		"test",
		[]Checker{checker1, checker2},
		nil,
	)

	// 验证检查器属性
	assert.Equal(t, "composite_checker", compositeChecker.Name())
	assert.Equal(t, "组合检查器", compositeChecker.Description())
	assert.Equal(t, "test", compositeChecker.Type())

	// 执行检查
	result := compositeChecker.Check(context.Background())
	assert.Equal(t, StatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "System is unhealthy")
	assert.Contains(t, result.Details, "测试1成功")
	assert.Contains(t, result.Details, "测试2失败")

	// 添加另一个检查器
	checker3 := NewSimpleChecker(
		"test_checker_3",
		"测试检查器3",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusDegraded,
				Message: "测试3降级",
				Details: map[string]interface{}{
					"test3": "degraded",
				},
			}
		},
	)
	compositeChecker.AddChecker(checker3)

	// 再次执行检查
	result = compositeChecker.Check(context.Background())
	assert.Equal(t, StatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "System is unhealthy")
	assert.Contains(t, result.Details, "测试3降级")
}

func TestDefaultAggregator(t *testing.T) {
	// 测试空结果
	result := DefaultAggregator([]CheckResult{})
	assert.Equal(t, StatusUnknown, result.Status)
	assert.Equal(t, "No health checks performed", result.Message)

	// 测试全部健康
	result = DefaultAggregator([]CheckResult{
		{
			Status:  StatusHealthy,
			Message: "测试1成功",
			Details: map[string]interface{}{"test1": true},
		},
		{
			Status:  StatusHealthy,
			Message: "测试2成功",
			Details: map[string]interface{}{"test2": true},
		},
	})
	assert.Equal(t, StatusHealthy, result.Status)
	assert.Equal(t, "All systems are healthy", result.Message)

	// 测试部分降级
	result = DefaultAggregator([]CheckResult{
		{
			Status:  StatusHealthy,
			Message: "测试1成功",
			Details: map[string]interface{}{"test1": true},
		},
		{
			Status:  StatusDegraded,
			Message: "测试2降级",
			Details: map[string]interface{}{"test2": "degraded"},
		},
	})
	assert.Equal(t, StatusDegraded, result.Status)
	assert.Contains(t, result.Message, "System is degraded")
	assert.Contains(t, result.Message, "测试2降级")

	// 测试部分不健康
	result = DefaultAggregator([]CheckResult{
		{
			Status:  StatusHealthy,
			Message: "测试1成功",
			Details: map[string]interface{}{"test1": true},
		},
		{
			Status:  StatusDegraded,
			Message: "测试2降级",
			Details: map[string]interface{}{"test2": "degraded"},
		},
		{
			Status:  StatusUnhealthy,
			Message: "测试3失败",
			Details: map[string]interface{}{"test3": false},
			Error:   assert.AnError,
		},
	})
	assert.Equal(t, StatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "System is unhealthy")
	assert.Contains(t, result.Message, "测试3失败")
	assert.NotNil(t, result.Error)
}

func TestHealthCheckRegistry(t *testing.T) {
	logger := hclog.NewNullLogger()
	registry := NewHealthCheckRegistry(logger)

	// 创建检查器
	checker1 := NewSimpleChecker(
		"test_checker_1",
		"测试检查器1",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusHealthy,
				Message: "测试1成功",
				Details: map[string]interface{}{"test1": true},
			}
		},
	)

	checker2 := NewSimpleChecker(
		"test_checker_2",
		"测试检查器2",
		"test",
		func(ctx context.Context) CheckResult {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "测试2失败",
				Details: map[string]interface{}{"test2": false},
			}
		},
	)

	// 注册检查器
	registry.RegisterChecker(checker1)
	registry.RegisterChecker(checker2)

	// 获取检查器
	c1, ok := registry.GetChecker("test_checker_1")
	assert.True(t, ok)
	assert.Equal(t, "test_checker_1", c1.Name())

	// 列出所有检查器
	checkers := registry.ListCheckers()
	assert.Len(t, checkers, 2)

	// 运行所有检查
	results := registry.RunChecks(context.Background())
	assert.Len(t, results, 2)
	assert.Equal(t, StatusHealthy, results["test_checker_1"].Status)
	assert.Equal(t, StatusUnhealthy, results["test_checker_2"].Status)

	// 运行单个检查
	result, ok := registry.RunCheck(context.Background(), "test_checker_1")
	assert.True(t, ok)
	assert.Equal(t, StatusHealthy, result.Status)

	// 运行不存在的检查
	result, ok = registry.RunCheck(context.Background(), "non_existent")
	assert.False(t, ok)
	assert.Equal(t, StatusUnknown, result.Status)

	// 获取系统状态
	systemStatus := registry.GetSystemStatus(context.Background())
	assert.Equal(t, StatusUnhealthy, systemStatus.Status)
	assert.Contains(t, systemStatus.Message, "System is unhealthy")

	// 注销检查器
	registry.UnregisterChecker("test_checker_2")
	checkers = registry.ListCheckers()
	assert.Len(t, checkers, 1)

	// 再次获取系统状态
	systemStatus = registry.GetSystemStatus(context.Background())
	assert.Equal(t, StatusHealthy, systemStatus.Status)
	assert.Equal(t, "All systems are healthy", systemStatus.Message)
}
