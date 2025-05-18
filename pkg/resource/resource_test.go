package resource

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestResourceUsageTracker(t *testing.T) {
	// 创建资源使用跟踪器
	tracker := NewResourceUsageTracker(int32(os.Getpid()), 10)
	assert.NotNil(t, tracker)

	// 设置磁盘路径
	tracker.SetDiskPaths([]string{"/"})

	// 设置网络接口
	tracker.SetNetworkInterfaces([]string{})

	// 设置资源限制
	tracker.SetResourceLimit(ResourceTypeCPU, 90)
	tracker.SetResourceLimit(ResourceTypeMemory, 1024*1024*1024)

	// 获取资源限制
	cpuLimit, exists := tracker.GetResourceLimit(ResourceTypeCPU)
	assert.True(t, exists)
	assert.Equal(t, uint64(90), cpuLimit)

	memLimit, exists := tracker.GetResourceLimit(ResourceTypeMemory)
	assert.True(t, exists)
	assert.Equal(t, uint64(1024*1024*1024), memLimit)

	// 移除资源限制
	tracker.RemoveResourceLimit(ResourceTypeCPU)
	_, exists = tracker.GetResourceLimit(ResourceTypeCPU)
	assert.False(t, exists)

	// 更新资源使用情况
	err := tracker.Update()
	assert.NoError(t, err)

	// 获取资源使用快照
	snapshot := tracker.GetSnapshot()
	assert.NotNil(t, snapshot)
	assert.NotNil(t, snapshot.Current)
	assert.NotNil(t, snapshot.Previous)
	assert.NotNil(t, snapshot.Delta)
	assert.NotEmpty(t, snapshot.History)
	assert.NotEmpty(t, snapshot.Stats)
	assert.NotEmpty(t, snapshot.Limits)

	// 再次更新资源使用情况
	err = tracker.Update()
	assert.NoError(t, err)

	// 获取资源使用快照
	snapshot = tracker.GetSnapshot()
	assert.NotNil(t, snapshot)
	assert.NotNil(t, snapshot.Current)
	assert.NotNil(t, snapshot.Previous)
	assert.NotNil(t, snapshot.Delta)
	assert.Equal(t, 2, len(snapshot.History))
	assert.NotEmpty(t, snapshot.Stats)
	assert.NotEmpty(t, snapshot.Limits)
}

func TestResourceLimiter(t *testing.T) {
	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建资源使用跟踪器
	tracker := NewResourceUsageTracker(int32(os.Getpid()), 10)
	assert.NotNil(t, tracker)

	// 创建资源限制器
	limiter := NewResourceLimiter(tracker, logger)
	assert.NotNil(t, limiter)

	// 添加资源限制
	limiter.AddLimit(ResourceLimit{
		ResourceType: ResourceTypeCPU,
		LimitType:    ResourceLimitTypeSoft,
		Value:        90,
		Action:       ResourceLimitActionLog,
	})

	limiter.AddLimit(ResourceLimit{
		ResourceType: ResourceTypeMemory,
		LimitType:    ResourceLimitTypeSoft,
		Value:        1024 * 1024 * 1024,
		Action:       ResourceLimitActionAlert,
	})

	// 获取资源限制
	limits := limiter.GetLimits()
	assert.NotEmpty(t, limits)
	assert.Equal(t, 1, len(limits[ResourceTypeCPU]))
	assert.Equal(t, 1, len(limits[ResourceTypeMemory]))

	// 注册告警处理器
	var alertMessage string
	var alertResourceType ResourceType
	limiter.RegisterAlertHandler(func(resourceType ResourceType, message string) {
		alertResourceType = resourceType
		alertMessage = message
	})

	// 更新资源使用情况
	err := tracker.Update()
	assert.NoError(t, err)

	// 检查资源限制
	err = limiter.Check()
	assert.NoError(t, err)

	// 移除资源限制
	limiter.RemoveLimit(ResourceTypeCPU, ResourceLimitTypeSoft)
	limits = limiter.GetLimits()
	assert.Empty(t, limits[ResourceTypeCPU])
	assert.Equal(t, 1, len(limits[ResourceTypeMemory]))

	// 使用便捷方法添加限制
	limiter.LimitCPU(80, ResourceLimitActionLog)
	limiter.LimitMemory(512*1024*1024, ResourceLimitActionAlert)
	limiter.LimitDisk(10*1024*1024*1024, ResourceLimitActionThrottle)
	limiter.LimitNetwork(1024*1024, ResourceLimitActionReject)

	// 获取资源限制
	limits = limiter.GetLimits()
	assert.Equal(t, 1, len(limits[ResourceTypeCPU]))
	assert.Equal(t, 1, len(limits[ResourceTypeMemory]))
	assert.Equal(t, 1, len(limits[ResourceTypeDisk]))
	assert.Equal(t, 1, len(limits[ResourceTypeNetwork]))

	// 启动资源限制器
	limiter.Start()

	// 等待一段时间
	time.Sleep(2 * time.Second)

	// 停止资源限制器
	limiter.Stop()
}

func TestResourceManager(t *testing.T) {
	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建资源管理器
	manager := NewResourceManager(
		WithLogger(logger),
		WithHistoryLimit(10),
		WithUpdateInterval(1*time.Second),
		WithDiskPaths([]string{"/"}),
		WithNetworkInterfaces([]string{}),
		WithProcessID(int32(os.Getpid())),
		WithContext(context.Background()),
	)
	assert.NotNil(t, manager)

	// 启动资源管理器
	manager.Start()

	// 等待一段时间
	time.Sleep(2 * time.Second)

	// 获取资源使用情况
	usage := manager.GetResourceUsage()
	assert.NotNil(t, usage)

	// 获取资源使用历史
	history := manager.GetResourceUsageHistory()
	assert.NotEmpty(t, history)

	// 获取资源统计信息
	stats := manager.GetResourceStats()
	assert.NotEmpty(t, stats)

	// 获取资源告警
	alerts := manager.GetResourceAlerts()
	assert.NotNil(t, alerts)

	// 获取资源限制
	limits := manager.GetResourceLimits()
	assert.NotNil(t, limits)

	// 添加资源限制
	manager.LimitCPU(80, ResourceLimitActionLog)
	manager.LimitMemory(512*1024*1024, ResourceLimitActionAlert)
	manager.LimitDisk(10*1024*1024*1024, ResourceLimitActionThrottle)
	manager.LimitNetwork(1024*1024, ResourceLimitActionReject)

	// 获取资源限制
	limits = manager.GetResourceLimits()
	assert.Equal(t, 1, len(limits[ResourceTypeCPU]))
	assert.Equal(t, 1, len(limits[ResourceTypeMemory]))
	assert.Equal(t, 1, len(limits[ResourceTypeDisk]))
	assert.Equal(t, 1, len(limits[ResourceTypeNetwork]))

	// 移除资源限制
	manager.RemoveLimit(ResourceTypeCPU, ResourceLimitTypeSoft)
	limits = manager.GetResourceLimits()
	assert.Empty(t, limits[ResourceTypeCPU])

	// 设置进程优先级
	err := manager.SetProcessPriority(0)
	if err != nil {
		t.Logf("设置进程优先级失败: %v", err)
	}

	// 设置GOMAXPROCS
	manager.SetGOMAXPROCS(2)

	// 获取进程信息
	procInfo, err := manager.GetProcessInfo()
	assert.NoError(t, err)
	assert.NotEmpty(t, procInfo)

	// 获取系统信息
	sysInfo := manager.GetSystemInfo()
	assert.NotEmpty(t, sysInfo)

	// 优化资源使用
	manager.OptimizeResourceUsage()

	// 停止资源管理器
	manager.Stop()
}

func TestResourceLimitActions(t *testing.T) {
	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建资源使用跟踪器
	tracker := NewResourceUsageTracker(int32(os.Getpid()), 10)
	assert.NotNil(t, tracker)

	// 创建资源限制器
	limiter := NewResourceLimiter(tracker, logger)
	assert.NotNil(t, limiter)

	// 注册自定义动作处理器
	var customActionCalled bool
	limiter.RegisterActionHandler("custom_action", func(resourceType ResourceType, limit ResourceLimit, usage *ResourceUsage) error {
		customActionCalled = true
		return nil
	})

	// 添加使用自定义动作的资源限制
	limiter.AddLimit(ResourceLimit{
		ResourceType: ResourceTypeCPU,
		LimitType:    ResourceLimitTypeSoft,
		Value:        1, // 设置一个很小的值，确保会触发
		Action:       "custom_action",
	})

	// 更新资源使用情况
	err := tracker.Update()
	assert.NoError(t, err)

	// 检查资源限制
	err = limiter.Check()
	assert.NoError(t, err)

	// 验证自定义动作是否被调用
	assert.True(t, customActionCalled)
}

func TestResourceUsageTrackerWithLimits(t *testing.T) {
	// 创建资源使用跟踪器
	tracker := NewResourceUsageTracker(int32(os.Getpid()), 10)
	assert.NotNil(t, tracker)

	// 设置资源限制
	tracker.SetResourceLimit(ResourceTypeCPU, 1) // 设置一个很小的值，确保会触发
	tracker.SetResourceLimit(ResourceTypeMemory, 1) // 设置一个很小的值，确保会触发

	// 更新资源使用情况
	err := tracker.Update()
	assert.NoError(t, err)

	// 获取资源使用快照
	snapshot := tracker.GetSnapshot()
	assert.NotNil(t, snapshot)

	// 验证是否有告警
	assert.NotEmpty(t, snapshot.Alerts)
}
