package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// Status 表示健康状态
type Status string

// 预定义健康状态
const (
	StatusUnknown   Status = "unknown"   // 未知状态
	StatusHealthy   Status = "healthy"   // 健康状态
	StatusUnhealthy Status = "unhealthy" // 不健康状态
	StatusDegraded  Status = "degraded"  // 降级状态
	StatusStarting  Status = "starting"  // 启动中状态
	StatusStopping  Status = "stopping"  // 停止中状态
	StatusStopped   Status = "stopped"   // 已停止状态
)

// CheckResult 健康检查结果
type CheckResult struct {
	Status        Status                 // 健康状态
	Message       string                 // 状态消息
	Details       map[string]interface{} // 详细信息
	LastChecked   time.Time              // 最后检查时间
	CheckDuration time.Duration          // 检查耗时
	Error         error                  // 错误信息
}

// Checker 健康检查器接口
type Checker interface {
	// Check 执行健康检查
	Check(ctx context.Context) CheckResult
	// Name 返回检查器名称
	Name() string
	// Description 返回检查器描述
	Description() string
	// Type 返回检查器类型
	Type() string
}

// CheckerFunc 健康检查函数类型
type CheckerFunc func(ctx context.Context) CheckResult

// SimpleChecker 简单健康检查器
type SimpleChecker struct {
	name        string
	description string
	checkType   string
	checkFunc   CheckerFunc
}

// NewSimpleChecker 创建一个新的简单健康检查器
func NewSimpleChecker(name, description, checkType string, checkFunc CheckerFunc) *SimpleChecker {
	return &SimpleChecker{
		name:        name,
		description: description,
		checkType:   checkType,
		checkFunc:   checkFunc,
	}
}

// Check 执行健康检查
func (c *SimpleChecker) Check(ctx context.Context) CheckResult {
	return c.checkFunc(ctx)
}

// Name 返回检查器名称
func (c *SimpleChecker) Name() string {
	return c.name
}

// Description 返回检查器描述
func (c *SimpleChecker) Description() string {
	return c.description
}

// Type 返回检查器类型
func (c *SimpleChecker) Type() string {
	return c.checkType
}

// CompositeChecker 组合健康检查器
type CompositeChecker struct {
	name        string
	description string
	checkType   string
	checkers    []Checker
	aggregator  func([]CheckResult) CheckResult
}

// NewCompositeChecker 创建一个新的组合健康检查器
func NewCompositeChecker(name, description, checkType string, checkers []Checker, aggregator func([]CheckResult) CheckResult) *CompositeChecker {
	if aggregator == nil {
		aggregator = DefaultAggregator
	}
	return &CompositeChecker{
		name:        name,
		description: description,
		checkType:   checkType,
		checkers:    checkers,
		aggregator:  aggregator,
	}
}

// Check 执行健康检查
func (c *CompositeChecker) Check(ctx context.Context) CheckResult {
	results := make([]CheckResult, 0, len(c.checkers))
	for _, checker := range c.checkers {
		results = append(results, checker.Check(ctx))
	}
	return c.aggregator(results)
}

// Name 返回检查器名称
func (c *CompositeChecker) Name() string {
	return c.name
}

// Description 返回检查器描述
func (c *CompositeChecker) Description() string {
	return c.description
}

// Type 返回检查器类型
func (c *CompositeChecker) Type() string {
	return c.checkType
}

// AddChecker 添加检查器
func (c *CompositeChecker) AddChecker(checker Checker) {
	c.checkers = append(c.checkers, checker)
}

// DefaultAggregator 默认聚合器
func DefaultAggregator(results []CheckResult) CheckResult {
	if len(results) == 0 {
		return CheckResult{
			Status:      StatusUnknown,
			Message:     "No health checks performed",
			Details:     make(map[string]interface{}),
			LastChecked: time.Now(),
		}
	}

	// 聚合状态
	status := StatusHealthy
	details := make(map[string]interface{})
	var messages []string
	var errors []error

	for _, result := range results {
		details[result.Message] = result.Details

		if result.Status == StatusUnhealthy {
			status = StatusUnhealthy
			messages = append(messages, result.Message)
			if result.Error != nil {
				errors = append(errors, result.Error)
			}
		} else if result.Status == StatusDegraded && status != StatusUnhealthy {
			status = StatusDegraded
			messages = append(messages, result.Message)
			if result.Error != nil {
				errors = append(errors, result.Error)
			}
		}
	}

	// 构建消息
	message := "All systems are healthy"
	if status == StatusUnhealthy {
		message = fmt.Sprintf("System is unhealthy: %v", messages)
	} else if status == StatusDegraded {
		message = fmt.Sprintf("System is degraded: %v", messages)
	}

	// 构建错误
	var err error
	if len(errors) > 0 {
		err = fmt.Errorf("health check errors: %v", errors)
	}

	return CheckResult{
		Status:      status,
		Message:     message,
		Details:     details,
		LastChecked: time.Now(),
		Error:       err,
	}
}

// CheckerRegistry 检查器注册表
type CheckerRegistry struct {
	checkers map[string]Checker
	mu       sync.RWMutex
	logger   hclog.Logger
}

// NewCheckerRegistry 创建一个新的检查器注册表
func NewCheckerRegistry(logger hclog.Logger) *CheckerRegistry {
	return &CheckerRegistry{
		checkers: make(map[string]Checker),
		logger:   logger,
	}
}

// RegisterChecker 注册健康检查器
func (r *CheckerRegistry) RegisterChecker(checker Checker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers[checker.Name()] = checker
	r.logger.Debug("注册健康检查器", "name", checker.Name(), "type", checker.Type())
}

// UnregisterChecker 注销健康检查器
func (r *CheckerRegistry) UnregisterChecker(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.checkers, name)
	r.logger.Debug("注销健康检查器", "name", name)
}

// GetChecker 获取健康检查器
func (r *CheckerRegistry) GetChecker(name string) (Checker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	checker, ok := r.checkers[name]
	return checker, ok
}

// ListCheckers 列出所有健康检查器
func (r *CheckerRegistry) ListCheckers() []Checker {
	r.mu.RLock()
	defer r.mu.RUnlock()
	checkers := make([]Checker, 0, len(r.checkers))
	for _, checker := range r.checkers {
		checkers = append(checkers, checker)
	}
	return checkers
}

// RunChecks 运行所有健康检查
func (r *CheckerRegistry) RunChecks(ctx context.Context) map[string]CheckResult {
	r.mu.RLock()
	checkers := make([]Checker, 0, len(r.checkers))
	for _, checker := range r.checkers {
		checkers = append(checkers, checker)
	}
	r.mu.RUnlock()

	results := make(map[string]CheckResult)
	for _, checker := range checkers {
		startTime := time.Now()
		result := checker.Check(ctx)
		result.CheckDuration = time.Since(startTime)
		result.LastChecked = time.Now()
		results[checker.Name()] = result
	}

	return results
}

// RunCheck 运行指定的健康检查
func (r *CheckerRegistry) RunCheck(ctx context.Context, name string) (CheckResult, bool) {
	r.mu.RLock()
	checker, ok := r.checkers[name]
	r.mu.RUnlock()

	if !ok {
		return CheckResult{
			Status:      StatusUnknown,
			Message:     fmt.Sprintf("Health checker %s not found", name),
			LastChecked: time.Now(),
		}, false
	}

	startTime := time.Now()
	result := checker.Check(ctx)
	result.CheckDuration = time.Since(startTime)
	result.LastChecked = time.Now()

	return result, true
}

// GetSystemStatus 获取系统状态
func (r *CheckerRegistry) GetSystemStatus(ctx context.Context) CheckResult {
	results := r.RunChecks(ctx)
	checkResults := make([]CheckResult, 0, len(results))
	for _, result := range results {
		checkResults = append(checkResults, result)
	}
	return DefaultAggregator(checkResults)
}
