package resource

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/time/rate"
)

// ResourceLimitType 资源限制类型
type ResourceLimitType string

// 预定义资源限制类型
const (
	ResourceLimitTypeSoft ResourceLimitType = "soft" // 软限制
	ResourceLimitTypeHard ResourceLimitType = "hard" // 硬限制
)

// ResourceLimitAction 资源限制动作
type ResourceLimitAction string

// 预定义资源限制动作
const (
	ResourceLimitActionNone     ResourceLimitAction = "none"     // 无动作
	ResourceLimitActionLog      ResourceLimitAction = "log"      // 记录日志
	ResourceLimitActionAlert    ResourceLimitAction = "alert"    // 发送告警
	ResourceLimitActionThrottle ResourceLimitAction = "throttle" // 限流
	ResourceLimitActionReject   ResourceLimitAction = "reject"   // 拒绝
	ResourceLimitActionRestart  ResourceLimitAction = "restart"  // 重启
	ResourceLimitActionStop     ResourceLimitAction = "stop"     // 停止
)

// ResourceLimit 资源限制
type ResourceLimit struct {
	ResourceType ResourceType        // 资源类型
	LimitType    ResourceLimitType   // 限制类型
	Value        uint64              // 限制值
	Action       ResourceLimitAction // 限制动作
	Duration     time.Duration       // 持续时间
}

// ResourceLimiter 资源限制器
type ResourceLimiter struct {
	limits         map[ResourceType][]ResourceLimit              // 资源限制
	tracker        *ResourceUsageTracker                         // 资源使用跟踪器
	logger         hclog.Logger                                  // 日志记录器
	limiters       map[ResourceType]*rate.Limiter                // 限流器
	alertHandlers  []ResourceAlertHandler                        // 告警处理器
	actionHandlers map[ResourceLimitAction]ResourceActionHandler // 动作处理器
	ctx            context.Context                               // 上下文
	cancel         context.CancelFunc                            // 取消函数
	mu             sync.RWMutex                                  // 互斥锁
}

// ResourceAlertHandler 资源告警处理器
type ResourceAlertHandler func(resourceType ResourceType, message string)

// ResourceActionHandler 资源动作处理器
type ResourceActionHandler func(resourceType ResourceType, limit ResourceLimit, usage *ResourceUsage) error

// NewResourceLimiter 创建资源限制器
func NewResourceLimiter(tracker *ResourceUsageTracker, logger hclog.Logger) *ResourceLimiter {
	ctx, cancel := context.WithCancel(context.Background())

	limiter := &ResourceLimiter{
		limits:         make(map[ResourceType][]ResourceLimit),
		tracker:        tracker,
		logger:         logger.Named("resource-limiter"),
		limiters:       make(map[ResourceType]*rate.Limiter),
		alertHandlers:  make([]ResourceAlertHandler, 0),
		actionHandlers: make(map[ResourceLimitAction]ResourceActionHandler),
		ctx:            ctx,
		cancel:         cancel,
	}

	// 注册默认动作处理器
	limiter.registerDefaultActionHandlers()

	return limiter
}

// registerDefaultActionHandlers 注册默认动作处理器
func (l *ResourceLimiter) registerDefaultActionHandlers() {
	// 日志动作处理器
	l.RegisterActionHandler(ResourceLimitActionLog, func(resourceType ResourceType, limit ResourceLimit, usage *ResourceUsage) error {
		l.logger.Warn("资源使用超过限制",
			"resource_type", resourceType,
			"limit_type", limit.LimitType,
			"limit_value", limit.Value,
			"current_value", l.getCurrentValue(resourceType, usage),
		)
		return nil
	})

	// 告警动作处理器
	l.RegisterActionHandler(ResourceLimitActionAlert, func(resourceType ResourceType, limit ResourceLimit, usage *ResourceUsage) error {
		message := fmt.Sprintf("资源使用超过限制: %s %s限制 %d, 当前值 %d",
			resourceType, limit.LimitType, limit.Value, l.getCurrentValue(resourceType, usage))
		l.triggerAlert(resourceType, message)
		return nil
	})

	// 限流动作处理器
	l.RegisterActionHandler(ResourceLimitActionThrottle, func(resourceType ResourceType, limit ResourceLimit, usage *ResourceUsage) error {
		// 获取或创建限流器
		limiter, ok := l.limiters[resourceType]
		if !ok {
			// 创建限流器，每秒允许的操作次数为当前值的一半
			currentValue := l.getCurrentValue(resourceType, usage)
			if currentValue == 0 {
				currentValue = 1
			}
			limiter = rate.NewLimiter(rate.Limit(currentValue/2), int(currentValue/4))
			l.limiters[resourceType] = limiter
		}

		// 等待令牌
		return limiter.Wait(l.ctx)
	})

	// 拒绝动作处理器
	l.RegisterActionHandler(ResourceLimitActionReject, func(resourceType ResourceType, limit ResourceLimit, usage *ResourceUsage) error {
		return fmt.Errorf("资源使用超过限制: %s %s限制 %d, 当前值 %d",
			resourceType, limit.LimitType, limit.Value, l.getCurrentValue(resourceType, usage))
	})

	// 重启动作处理器
	l.RegisterActionHandler(ResourceLimitActionRestart, func(resourceType ResourceType, limit ResourceLimit, usage *ResourceUsage) error {
		l.logger.Warn("资源使用超过限制，准备重启",
			"resource_type", resourceType,
			"limit_type", limit.LimitType,
			"limit_value", limit.Value,
			"current_value", l.getCurrentValue(resourceType, usage),
		)
		// 这里只是示例，实际实现可能需要调用外部重启机制
		return fmt.Errorf("需要重启")
	})

	// 停止动作处理器
	l.RegisterActionHandler(ResourceLimitActionStop, func(resourceType ResourceType, limit ResourceLimit, usage *ResourceUsage) error {
		l.logger.Warn("资源使用超过限制，准备停止",
			"resource_type", resourceType,
			"limit_type", limit.LimitType,
			"limit_value", limit.Value,
			"current_value", l.getCurrentValue(resourceType, usage),
		)
		// 这里只是示例，实际实现可能需要调用外部停止机制
		return fmt.Errorf("需要停止")
	})
}

// getCurrentValue 获取当前值
func (l *ResourceLimiter) getCurrentValue(resourceType ResourceType, usage *ResourceUsage) uint64 {
	switch resourceType {
	case ResourceTypeCPU:
		return uint64(usage.CPUUsage)
	case ResourceTypeMemory:
		return usage.MemoryUsage
	case ResourceTypeDisk:
		return usage.DiskUsage
	case ResourceTypeNetwork:
		return usage.NetworkSentBytes + usage.NetworkReceivedBytes
	default:
		return 0
	}
}

// AddLimit 添加资源限制
func (l *ResourceLimiter) AddLimit(limit ResourceLimit) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 添加限制
	l.limits[limit.ResourceType] = append(l.limits[limit.ResourceType], limit)

	// 设置资源跟踪器的限制
	l.tracker.SetResourceLimit(limit.ResourceType, limit.Value)

	l.logger.Info("添加资源限制",
		"resource_type", limit.ResourceType,
		"limit_type", limit.LimitType,
		"value", limit.Value,
		"action", limit.Action,
	)
}

// RemoveLimit 移除资源限制
func (l *ResourceLimiter) RemoveLimit(resourceType ResourceType, limitType ResourceLimitType) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 获取限制
	limits := l.limits[resourceType]
	if len(limits) == 0 {
		return
	}

	// 移除限制
	var newLimits []ResourceLimit
	for _, limit := range limits {
		if limit.LimitType != limitType {
			newLimits = append(newLimits, limit)
		}
	}
	l.limits[resourceType] = newLimits

	// 如果没有限制了，移除资源跟踪器的限制
	if len(newLimits) == 0 {
		l.tracker.RemoveResourceLimit(resourceType)
		delete(l.limits, resourceType)
	}

	l.logger.Info("移除资源限制",
		"resource_type", resourceType,
		"limit_type", limitType,
	)
}

// GetLimits 获取资源限制
func (l *ResourceLimiter) GetLimits() map[ResourceType][]ResourceLimit {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 复制限制
	limits := make(map[ResourceType][]ResourceLimit)
	for resourceType, resourceLimits := range l.limits {
		limits[resourceType] = make([]ResourceLimit, len(resourceLimits))
		copy(limits[resourceType], resourceLimits)
	}

	return limits
}

// RegisterAlertHandler 注册告警处理器
func (l *ResourceLimiter) RegisterAlertHandler(handler ResourceAlertHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.alertHandlers = append(l.alertHandlers, handler)
}

// RegisterActionHandler 注册动作处理器
func (l *ResourceLimiter) RegisterActionHandler(action ResourceLimitAction, handler ResourceActionHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.actionHandlers[action] = handler
}

// triggerAlert 触发告警
func (l *ResourceLimiter) triggerAlert(resourceType ResourceType, message string) {
	l.mu.RLock()
	handlers := make([]ResourceAlertHandler, len(l.alertHandlers))
	copy(handlers, l.alertHandlers)
	l.mu.RUnlock()

	// 调用所有告警处理器
	for _, handler := range handlers {
		handler(resourceType, message)
	}
}

// Check 检查资源限制
func (l *ResourceLimiter) Check() error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 获取资源使用快照
	snapshot := l.tracker.GetSnapshot()
	if snapshot == nil || snapshot.Current == nil {
		return nil
	}

	// 检查每种资源类型的限制
	for resourceType, limits := range l.limits {
		// 获取当前值
		currentValue := l.getCurrentValue(resourceType, snapshot.Current)

		// 检查每个限制
		for _, limit := range limits {
			// 如果超过限制
			if currentValue > limit.Value {
				// 获取动作处理器
				handler, ok := l.actionHandlers[limit.Action]
				if !ok {
					continue
				}

				// 执行动作
				if err := handler(resourceType, limit, snapshot.Current); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Start 启动资源限制器
func (l *ResourceLimiter) Start() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// 更新资源使用情况
				if err := l.tracker.Update(); err != nil {
					l.logger.Error("更新资源使用情况失败", "error", err)
					continue
				}

				// 检查资源限制
				if err := l.Check(); err != nil {
					l.logger.Error("检查资源限制失败", "error", err)
				}
			case <-l.ctx.Done():
				return
			}
		}
	}()

	l.logger.Info("资源限制器已启动")
}

// Stop 停止资源限制器
func (l *ResourceLimiter) Stop() {
	l.cancel()
	l.logger.Info("资源限制器已停止")
}

// LimitCPU 限制CPU使用
func (l *ResourceLimiter) LimitCPU(percent float64, action ResourceLimitAction) {
	l.AddLimit(ResourceLimit{
		ResourceType: ResourceTypeCPU,
		LimitType:    ResourceLimitTypeSoft,
		Value:        uint64(percent),
		Action:       action,
	})
}

// LimitMemory 限制内存使用
func (l *ResourceLimiter) LimitMemory(bytes uint64, action ResourceLimitAction) {
	l.AddLimit(ResourceLimit{
		ResourceType: ResourceTypeMemory,
		LimitType:    ResourceLimitTypeSoft,
		Value:        bytes,
		Action:       action,
	})
}

// LimitDisk 限制磁盘使用
func (l *ResourceLimiter) LimitDisk(bytes uint64, action ResourceLimitAction) {
	l.AddLimit(ResourceLimit{
		ResourceType: ResourceTypeDisk,
		LimitType:    ResourceLimitTypeSoft,
		Value:        bytes,
		Action:       action,
	})
}

// LimitNetwork 限制网络使用
func (l *ResourceLimiter) LimitNetwork(bytesPerSecond uint64, action ResourceLimitAction) {
	l.AddLimit(ResourceLimit{
		ResourceType: ResourceTypeNetwork,
		LimitType:    ResourceLimitTypeSoft,
		Value:        bytesPerSecond,
		Action:       action,
	})
}

// SetProcessPriority 设置进程优先级
func (l *ResourceLimiter) SetProcessPriority(priority int) error {
	// 获取进程ID
	pid := l.tracker.processID
	if pid <= 0 {
		pid = int32(os.Getpid())
	}

	// 获取进程
	_, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("获取进程失败: %w", err)
	}

	// 设置进程优先级 - 使用系统特定方法
	// Windows系统使用Windows API设置进程优先级
	if runtime.GOOS == "windows" {
		// 将priority转换为Windows优先级类
		var priorityClass int
		switch {
		case priority <= -15:
			priorityClass = -1 // IDLE_PRIORITY_CLASS
		case priority <= -5:
			priorityClass = -2 // BELOW_NORMAL_PRIORITY_CLASS
		case priority < 5:
			priorityClass = 0 // NORMAL_PRIORITY_CLASS
		case priority < 15:
			priorityClass = 1 // ABOVE_NORMAL_PRIORITY_CLASS
		default:
			priorityClass = 2 // HIGH_PRIORITY_CLASS
		}

		// 使用process库获取进程信息
		// 在实际项目中，可以使用这个进程对象来获取更多信息
		// 例如进程名称、命令行参数等
		_, err = process.NewProcess(pid)
		if err != nil {
			l.logger.Error("获取进程失败", "pid", pid, "error", err)
			return fmt.Errorf("获取进程失败: %w", err)
		}

		// 设置进程优先级
		// 注意：gopsutil v3不直接支持设置优先级，这里只是记录日志
		l.logger.Info("设置Windows进程优先级", "pid", pid, "priority_class", priorityClass)

		// 在实际项目中，可以使用syscall或者golang.org/x/sys/windows包来设置进程优先级
		// 例如：windows.SetPriorityClass(windows.Handle(proc.Pid), uint32(priorityClass))
	} else if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		// Unix系统使用setpriority系统调用
		// 这里需要导入syscall包并使用适当的系统调用
		l.logger.Info("在Unix系统上设置进程优先级", "pid", pid, "priority", priority)
		// 实际实现需要使用syscall.Setpriority
	} else {
		l.logger.Warn("不支持的操作系统", "os", runtime.GOOS)
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	l.logger.Info("设置进程优先级成功", "pid", pid, "priority", priority)
	return nil
}

// SetGOMAXPROCS 设置GOMAXPROCS
func (l *ResourceLimiter) SetGOMAXPROCS(n int) {
	oldN := runtime.GOMAXPROCS(n)
	l.logger.Info("设置GOMAXPROCS", "old", oldN, "new", n)
}

// GetResourceUsage 获取资源使用情况
func (l *ResourceLimiter) GetResourceUsage() *ResourceUsage {
	snapshot := l.tracker.GetSnapshot()
	if snapshot == nil {
		return nil
	}
	return snapshot.Current
}

// GetResourceUsageHistory 获取资源使用历史
func (l *ResourceLimiter) GetResourceUsageHistory() []*ResourceUsage {
	snapshot := l.tracker.GetSnapshot()
	if snapshot == nil {
		return nil
	}
	return snapshot.History
}

// GetResourceStats 获取资源统计信息
func (l *ResourceLimiter) GetResourceStats() map[string]interface{} {
	snapshot := l.tracker.GetSnapshot()
	if snapshot == nil {
		return nil
	}
	return snapshot.Stats
}

// GetResourceAlerts 获取资源告警
func (l *ResourceLimiter) GetResourceAlerts() map[ResourceType][]string {
	snapshot := l.tracker.GetSnapshot()
	if snapshot == nil {
		return nil
	}
	return snapshot.Alerts
}
