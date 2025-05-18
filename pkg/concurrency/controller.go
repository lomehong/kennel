package concurrency

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-hclog"
)

// ResourceType 表示资源类型
type ResourceType string

// 预定义资源类型
const (
	ResourceTypeCPU     ResourceType = "cpu"
	ResourceTypeMemory  ResourceType = "memory"
	ResourceTypeIO      ResourceType = "io"
	ResourceTypeNetwork ResourceType = "network"
	ResourceTypeGeneric ResourceType = "generic"
)

// ResourceLimit 表示资源限制
type ResourceLimit struct {
	Type  ResourceType
	Limit int
	Used  atomic.Int32
}

// ConcurrencyController 并发控制器
type ConcurrencyController struct {
	limits       map[ResourceType]*ResourceLimit
	pools        map[string]*WorkerPool
	mu           sync.RWMutex
	logger       hclog.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	defaultLimit int
}

// ConcurrencyControllerOption 并发控制器配置选项
type ConcurrencyControllerOption func(*ConcurrencyController)

// WithControllerLogger 设置日志记录器
func WithControllerLogger(logger hclog.Logger) ConcurrencyControllerOption {
	return func(cc *ConcurrencyController) {
		cc.logger = logger
	}
}

// WithControllerContext 设置上下文
func WithControllerContext(ctx context.Context) ConcurrencyControllerOption {
	return func(cc *ConcurrencyController) {
		if cc.cancel != nil {
			cc.cancel()
		}
		cc.ctx, cc.cancel = context.WithCancel(ctx)
	}
}

// WithResourceLimit 设置资源限制
func WithResourceLimit(resourceType ResourceType, limit int) ConcurrencyControllerOption {
	return func(cc *ConcurrencyController) {
		cc.limits[resourceType] = &ResourceLimit{
			Type:  resourceType,
			Limit: limit,
		}
	}
}

// WithDefaultLimit 设置默认限制
func WithDefaultLimit(limit int) ConcurrencyControllerOption {
	return func(cc *ConcurrencyController) {
		cc.defaultLimit = limit
	}
}

// NewConcurrencyController 创建一个新的并发控制器
func NewConcurrencyController(options ...ConcurrencyControllerOption) *ConcurrencyController {
	ctx, cancel := context.WithCancel(context.Background())

	cc := &ConcurrencyController{
		limits:       make(map[ResourceType]*ResourceLimit),
		pools:        make(map[string]*WorkerPool),
		logger:       hclog.NewNullLogger(),
		ctx:          ctx,
		cancel:       cancel,
		defaultLimit: runtime.NumCPU() * 2,
	}

	// 设置默认资源限制
	cc.limits[ResourceTypeCPU] = &ResourceLimit{
		Type:  ResourceTypeCPU,
		Limit: runtime.NumCPU() * 2,
	}
	cc.limits[ResourceTypeMemory] = &ResourceLimit{
		Type:  ResourceTypeMemory,
		Limit: 100,
	}
	cc.limits[ResourceTypeIO] = &ResourceLimit{
		Type:  ResourceTypeIO,
		Limit: 50,
	}
	cc.limits[ResourceTypeNetwork] = &ResourceLimit{
		Type:  ResourceTypeNetwork,
		Limit: 30,
	}
	cc.limits[ResourceTypeGeneric] = &ResourceLimit{
		Type:  ResourceTypeGeneric,
		Limit: 100,
	}

	// 应用选项
	for _, option := range options {
		option(cc)
	}

	return cc
}

// CreatePool 创建工作池
func (cc *ConcurrencyController) CreatePool(name string, workers int, options ...WorkerPoolOption) (*WorkerPool, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// 检查池是否已存在
	if _, exists := cc.pools[name]; exists {
		return nil, fmt.Errorf("工作池已存在: %s", name)
	}

	// 如果未指定工作协程数，使用默认限制
	if workers <= 0 {
		workers = cc.defaultLimit
	}

	// 创建工作池
	pool := NewWorkerPool(name, workers, append(options, WithContext(cc.ctx))...)

	// 存储工作池
	cc.pools[name] = pool

	cc.logger.Info("创建工作池", "name", name, "workers", workers)
	return pool, nil
}

// GetPool 获取工作池
func (cc *ConcurrencyController) GetPool(name string) (*WorkerPool, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	pool, exists := cc.pools[name]
	return pool, exists
}

// StartPool 启动工作池
func (cc *ConcurrencyController) StartPool(name string) error {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	pool, exists := cc.pools[name]
	if !exists {
		return fmt.Errorf("工作池不存在: %s", name)
	}

	pool.Start()
	return nil
}

// StopPool 停止工作池
func (cc *ConcurrencyController) StopPool(name string) error {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	pool, exists := cc.pools[name]
	if !exists {
		return fmt.Errorf("工作池不存在: %s", name)
	}

	pool.Stop()
	return nil
}

// RemovePool 移除工作池
func (cc *ConcurrencyController) RemovePool(name string) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	pool, exists := cc.pools[name]
	if !exists {
		return fmt.Errorf("工作池不存在: %s", name)
	}

	// 停止工作池
	pool.Stop()

	// 移除工作池
	delete(cc.pools, name)

	cc.logger.Info("移除工作池", "name", name)
	return nil
}

// ListPools 列出所有工作池
func (cc *ConcurrencyController) ListPools() []string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	pools := make([]string, 0, len(cc.pools))
	for name := range cc.pools {
		pools = append(pools, name)
	}

	return pools
}

// GetPoolStats 获取工作池统计信息
func (cc *ConcurrencyController) GetPoolStats(name string) (map[string]interface{}, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	pool, exists := cc.pools[name]
	if !exists {
		return nil, fmt.Errorf("工作池不存在: %s", name)
	}

	return pool.Stats(), nil
}

// GetAllPoolStats 获取所有工作池统计信息
func (cc *ConcurrencyController) GetAllPoolStats() map[string]map[string]interface{} {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	stats := make(map[string]map[string]interface{})
	for name, pool := range cc.pools {
		stats[name] = pool.Stats()
	}

	return stats
}

// AcquireResource 获取资源
func (cc *ConcurrencyController) AcquireResource(resourceType ResourceType, count int) bool {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	limit, exists := cc.limits[resourceType]
	if !exists {
		limit = cc.limits[ResourceTypeGeneric]
	}

	// 检查是否有足够的资源
	for {
		used := limit.Used.Load()
		if used+int32(count) > int32(limit.Limit) {
			return false
		}

		// 尝试原子更新
		if limit.Used.CompareAndSwap(used, used+int32(count)) {
			cc.logger.Debug("获取资源", "type", resourceType, "count", count, "used", used+int32(count), "limit", limit.Limit)
			return true
		}
	}
}

// ReleaseResource 释放资源
func (cc *ConcurrencyController) ReleaseResource(resourceType ResourceType, count int) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	limit, exists := cc.limits[resourceType]
	if !exists {
		limit = cc.limits[ResourceTypeGeneric]
	}

	// 释放资源
	for {
		used := limit.Used.Load()
		newUsed := used - int32(count)
		if newUsed < 0 {
			newUsed = 0
		}

		// 尝试原子更新
		if limit.Used.CompareAndSwap(used, newUsed) {
			cc.logger.Debug("释放资源", "type", resourceType, "count", count, "used", newUsed, "limit", limit.Limit)
			return
		}
	}
}

// WithResource 使用资源执行函数
func (cc *ConcurrencyController) WithResource(resourceType ResourceType, count int, fn func() error) error {
	// 获取资源
	if !cc.AcquireResource(resourceType, count) {
		return fmt.Errorf("无法获取资源: %s, 请求数量: %d", resourceType, count)
	}

	// 确保释放资源
	defer cc.ReleaseResource(resourceType, count)

	// 执行函数
	return fn()
}

// WithResourceContext 使用资源执行带上下文的函数
func (cc *ConcurrencyController) WithResourceContext(ctx context.Context, resourceType ResourceType, count int, fn func(context.Context) error) error {
	// 获取资源
	if !cc.AcquireResource(resourceType, count) {
		return fmt.Errorf("无法获取资源: %s, 请求数量: %d", resourceType, count)
	}

	// 确保释放资源
	defer cc.ReleaseResource(resourceType, count)

	// 执行函数
	return fn(ctx)
}

// Stop 停止并发控制器
func (cc *ConcurrencyController) Stop() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.logger.Info("停止并发控制器")

	// 停止所有工作池
	for name, pool := range cc.pools {
		cc.logger.Info("停止工作池", "name", name)
		pool.Stop()
	}

	// 取消上下文
	cc.cancel()
}

// GetResourceUsage 获取资源使用情况
func (cc *ConcurrencyController) GetResourceUsage() map[string]map[string]int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	usage := make(map[string]map[string]int)
	for typeName, limit := range cc.limits {
		usage[string(typeName)] = map[string]int{
			"limit": limit.Limit,
			"used":  int(limit.Used.Load()),
		}
	}

	return usage
}
