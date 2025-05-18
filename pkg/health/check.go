package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// HealthStatus 健康状态
type HealthStatus string

// 预定义健康状态
const (
	HealthStatusHealthy   HealthStatus = "healthy"   // 健康
	HealthStatusUnhealthy HealthStatus = "unhealthy" // 不健康
	HealthStatusDegraded  HealthStatus = "degraded"  // 降级
	HealthStatusUnknown   HealthStatus = "unknown"   // 未知
)

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Status      HealthStatus       // 健康状态
	Message     string             // 消息
	Details     map[string]interface{} // 详细信息
	Timestamp   time.Time          // 时间戳
	Duration    time.Duration      // 检查耗时
	CheckName   string             // 检查名称
	CheckType   string             // 检查类型
	Component   string             // 组件名称
	Error       error              // 错误信息
	Recoverable bool               // 是否可恢复
}

// HealthCheck 健康检查接口
type HealthCheck interface {
	// Name 返回健康检查的名称
	Name() string

	// Type 返回健康检查的类型
	Type() string

	// Component 返回健康检查的组件
	Component() string

	// Check 执行健康检查
	Check(ctx context.Context) *HealthCheckResult

	// IsRecoverable 检查是否可恢复
	IsRecoverable() bool

	// Recover 尝试恢复
	Recover(ctx context.Context) error

	// Timeout 返回健康检查的超时时间
	Timeout() time.Duration

	// Interval 返回健康检查的间隔时间
	Interval() time.Duration

	// FailureThreshold 返回健康检查的失败阈值
	FailureThreshold() int

	// SuccessThreshold 返回健康检查的成功阈值
	SuccessThreshold() int
}

// BaseHealthCheck 基础健康检查
type BaseHealthCheck struct {
	name             string        // 名称
	checkType        string        // 类型
	component        string        // 组件
	timeout          time.Duration // 超时时间
	interval         time.Duration // 间隔时间
	failureThreshold int           // 失败阈值
	successThreshold int           // 成功阈值
	recoverable      bool          // 是否可恢复
	checkFunc        func(ctx context.Context) *HealthCheckResult // 检查函数
	recoverFunc      func(ctx context.Context) error              // 恢复函数
}

// NewBaseHealthCheck 创建基础健康检查
func NewBaseHealthCheck(
	name string,
	checkType string,
	component string,
	timeout time.Duration,
	interval time.Duration,
	failureThreshold int,
	successThreshold int,
	recoverable bool,
	checkFunc func(ctx context.Context) *HealthCheckResult,
	recoverFunc func(ctx context.Context) error,
) *BaseHealthCheck {
	return &BaseHealthCheck{
		name:             name,
		checkType:        checkType,
		component:        component,
		timeout:          timeout,
		interval:         interval,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		recoverable:      recoverable,
		checkFunc:        checkFunc,
		recoverFunc:      recoverFunc,
	}
}

// Name 返回健康检查的名称
func (c *BaseHealthCheck) Name() string {
	return c.name
}

// Type 返回健康检查的类型
func (c *BaseHealthCheck) Type() string {
	return c.checkType
}

// Component 返回健康检查的组件
func (c *BaseHealthCheck) Component() string {
	return c.component
}

// Check 执行健康检查
func (c *BaseHealthCheck) Check(ctx context.Context) *HealthCheckResult {
	if c.checkFunc == nil {
		return &HealthCheckResult{
			Status:      HealthStatusUnknown,
			Message:     "检查函数未定义",
			Timestamp:   time.Now(),
			CheckName:   c.name,
			CheckType:   c.checkType,
			Component:   c.component,
			Recoverable: c.recoverable,
		}
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 记录开始时间
	startTime := time.Now()

	// 执行检查
	result := c.checkFunc(ctx)

	// 设置结果信息
	result.Timestamp = startTime
	result.Duration = time.Since(startTime)
	result.CheckName = c.name
	result.CheckType = c.checkType
	result.Component = c.component
	result.Recoverable = c.recoverable

	return result
}

// IsRecoverable 检查是否可恢复
func (c *BaseHealthCheck) IsRecoverable() bool {
	return c.recoverable && c.recoverFunc != nil
}

// Recover 尝试恢复
func (c *BaseHealthCheck) Recover(ctx context.Context) error {
	if !c.IsRecoverable() {
		return fmt.Errorf("健康检查 %s 不可恢复", c.name)
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 执行恢复
	return c.recoverFunc(ctx)
}

// Timeout 返回健康检查的超时时间
func (c *BaseHealthCheck) Timeout() time.Duration {
	return c.timeout
}

// Interval 返回健康检查的间隔时间
func (c *BaseHealthCheck) Interval() time.Duration {
	return c.interval
}

// FailureThreshold 返回健康检查的失败阈值
func (c *BaseHealthCheck) FailureThreshold() int {
	return c.failureThreshold
}

// SuccessThreshold 返回健康检查的成功阈值
func (c *BaseHealthCheck) SuccessThreshold() int {
	return c.successThreshold
}

// HealthCheckRegistry 健康检查注册表
type HealthCheckRegistry struct {
	checks map[string]HealthCheck // 健康检查
	mu     sync.RWMutex           // 互斥锁
	logger hclog.Logger           // 日志记录器
}

// NewHealthCheckRegistry 创建健康检查注册表
func NewHealthCheckRegistry(logger hclog.Logger) *HealthCheckRegistry {
	return &HealthCheckRegistry{
		checks: make(map[string]HealthCheck),
		logger: logger.Named("health-registry"),
	}
}

// Register 注册健康检查
func (r *HealthCheckRegistry) Register(check HealthCheck) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := check.Name()
	if _, exists := r.checks[name]; exists {
		return fmt.Errorf("健康检查 %s 已存在", name)
	}

	r.checks[name] = check
	r.logger.Info("注册健康检查", "name", name, "type", check.Type(), "component", check.Component())
	return nil
}

// Unregister 注销健康检查
func (r *HealthCheckRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.checks[name]; !exists {
		return fmt.Errorf("健康检查 %s 不存在", name)
	}

	delete(r.checks, name)
	r.logger.Info("注销健康检查", "name", name)
	return nil
}

// Get 获取健康检查
func (r *HealthCheckRegistry) Get(name string) (HealthCheck, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	check, exists := r.checks[name]
	return check, exists
}

// GetAll 获取所有健康检查
func (r *HealthCheckRegistry) GetAll() map[string]HealthCheck {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 复制健康检查
	checks := make(map[string]HealthCheck, len(r.checks))
	for name, check := range r.checks {
		checks[name] = check
	}

	return checks
}

// GetByType 获取指定类型的健康检查
func (r *HealthCheckRegistry) GetByType(checkType string) []HealthCheck {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var checks []HealthCheck
	for _, check := range r.checks {
		if check.Type() == checkType {
			checks = append(checks, check)
		}
	}

	return checks
}

// GetByComponent 获取指定组件的健康检查
func (r *HealthCheckRegistry) GetByComponent(component string) []HealthCheck {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var checks []HealthCheck
	for _, check := range r.checks {
		if check.Component() == component {
			checks = append(checks, check)
		}
	}

	return checks
}

// Count 获取健康检查数量
func (r *HealthCheckRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.checks)
}

// Clear 清空健康检查
func (r *HealthCheckRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.checks = make(map[string]HealthCheck)
	r.logger.Info("清空健康检查")
}
