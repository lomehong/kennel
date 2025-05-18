package resource

import (
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// Resource 表示一个需要追踪的资源
type Resource interface {
	// Close 关闭资源
	Close() error
	// ID 返回资源的唯一标识符
	ID() string
	// Type 返回资源的类型
	Type() string
	// CreatedAt 返回资源的创建时间
	CreatedAt() time.Time
	// LastUsedAt 返回资源的最后使用时间
	LastUsedAt() time.Time
	// UpdateLastUsed 更新资源的最后使用时间
	UpdateLastUsed()
}

// BaseResource 提供Resource接口的基本实现
type BaseResource struct {
	id           string
	resourceType string
	createdAt    time.Time
	lastUsedAt   time.Time
	mu           sync.RWMutex
	closer       func() error
}

// NewBaseResource 创建一个新的基础资源
func NewBaseResource(id, resourceType string, closer func() error) *BaseResource {
	now := time.Now()
	return &BaseResource{
		id:           id,
		resourceType: resourceType,
		createdAt:    now,
		lastUsedAt:   now,
		closer:       closer,
	}
}

// ID 返回资源的唯一标识符
func (r *BaseResource) ID() string {
	return r.id
}

// Type 返回资源的类型
func (r *BaseResource) Type() string {
	return r.resourceType
}

// CreatedAt 返回资源的创建时间
func (r *BaseResource) CreatedAt() time.Time {
	return r.createdAt
}

// LastUsedAt 返回资源的最后使用时间
func (r *BaseResource) LastUsedAt() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastUsedAt
}

// UpdateLastUsed 更新资源的最后使用时间
func (r *BaseResource) UpdateLastUsed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastUsedAt = time.Now()
}

// Close 关闭资源
func (r *BaseResource) Close() error {
	if r.closer != nil {
		return r.closer()
	}
	return nil
}

// ResourceTracker 管理和追踪系统中的资源
type ResourceTracker struct {
	resources     map[string]Resource
	mu            sync.RWMutex
	logger        hclog.Logger
	cleanupTicker *time.Ticker
	stopChan      chan struct{}

	// 资源统计
	stats ResourceStats
}

// ResourceStats 资源统计信息
type ResourceStats struct {
	TotalCreated  int64
	TotalClosed   int64
	CurrentActive int64
	ClosureErrors int64
}

// ResourceTrackerOption 资源追踪器配置选项
type ResourceTrackerOption func(*ResourceTracker)

// WithTrackerLogger 设置日志记录器
func WithTrackerLogger(logger hclog.Logger) ResourceTrackerOption {
	return func(rt *ResourceTracker) {
		rt.logger = logger
	}
}

// WithCleanupInterval 设置自动清理间隔
func WithCleanupInterval(interval time.Duration) ResourceTrackerOption {
	return func(rt *ResourceTracker) {
		if rt.cleanupTicker != nil {
			rt.cleanupTicker.Stop()
		}
		rt.cleanupTicker = time.NewTicker(interval)
	}
}

// NewResourceTracker 创建一个新的资源追踪器
func NewResourceTracker(options ...ResourceTrackerOption) *ResourceTracker {
	rt := &ResourceTracker{
		resources:     make(map[string]Resource),
		logger:        hclog.NewNullLogger(),
		cleanupTicker: time.NewTicker(10 * time.Minute), // 默认10分钟清理一次
		stopChan:      make(chan struct{}),
	}

	// 应用选项
	for _, option := range options {
		option(rt)
	}

	// 启动自动清理协程
	go rt.autoCleanupLoop()

	return rt
}

// Track 追踪一个资源
func (rt *ResourceTracker) Track(resource Resource) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	id := resource.ID()
	if _, exists := rt.resources[id]; exists {
		rt.logger.Warn("资源已存在，将被覆盖", "id", id, "type", resource.Type())
	}

	rt.resources[id] = resource
	rt.stats.TotalCreated++
	rt.stats.CurrentActive++

	rt.logger.Debug("资源已追踪", "id", id, "type", resource.Type())
}

// Release 释放一个资源
func (rt *ResourceTracker) Release(id string) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	return rt.releaseResource(id)
}

// 内部方法，不加锁，调用者需要确保已加锁
func (rt *ResourceTracker) releaseResource(id string) error {
	resource, exists := rt.resources[id]
	if !exists {
		return fmt.Errorf("资源不存在: %s", id)
	}

	err := resource.Close()
	delete(rt.resources, id)

	rt.stats.TotalClosed++
	rt.stats.CurrentActive--

	if err != nil {
		rt.stats.ClosureErrors++
		rt.logger.Error("关闭资源失败", "id", id, "type", resource.Type(), "error", err)
		return fmt.Errorf("关闭资源失败: %w", err)
	}

	rt.logger.Debug("资源已释放", "id", id, "type", resource.Type())
	return nil
}

// Get 获取一个资源
func (rt *ResourceTracker) Get(id string) (Resource, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	resource, exists := rt.resources[id]
	if exists {
		resource.UpdateLastUsed()
	}
	return resource, exists
}

// ReleaseAll 释放所有资源
func (rt *ResourceTracker) ReleaseAll() []error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	return rt.releaseAllNoLock()
}

// releaseAllNoLock 释放所有资源（内部方法，不加锁）
// 调用者必须确保已经获取了互斥锁
func (rt *ResourceTracker) releaseAllNoLock() []error {
	var errors []error
	for id := range rt.resources {
		if err := rt.releaseResource(id); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// ReleaseByType 释放指定类型的所有资源
func (rt *ResourceTracker) ReleaseByType(resourceType string) []error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	var errors []error
	for id, resource := range rt.resources {
		if resource.Type() == resourceType {
			if err := rt.releaseResource(id); err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}

// GetStats 获取资源统计信息
func (rt *ResourceTracker) GetStats() ResourceStats {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	return rt.stats
}

// ListResources 列出所有资源
func (rt *ResourceTracker) ListResources() []Resource {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	resources := make([]Resource, 0, len(rt.resources))
	for _, resource := range rt.resources {
		resources = append(resources, resource)
	}

	return resources
}

// autoCleanupLoop 自动清理循环
func (rt *ResourceTracker) autoCleanupLoop() {
	for {
		select {
		case <-rt.cleanupTicker.C:
			rt.CleanupIdleResources(30 * time.Minute) // 清理30分钟未使用的资源
		case <-rt.stopChan:
			rt.cleanupTicker.Stop()
			return
		}
	}
}

// CleanupIdleResources 清理空闲资源
func (rt *ResourceTracker) CleanupIdleResources(idleTimeout time.Duration) []error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	var errors []error
	now := time.Now()

	for id, resource := range rt.resources {
		if now.Sub(resource.LastUsedAt()) > idleTimeout {
			rt.logger.Info("清理空闲资源", "id", id, "type", resource.Type(), "idle_time", now.Sub(resource.LastUsedAt()))
			if err := rt.releaseResource(id); err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}

// Stop 停止资源追踪器
func (rt *ResourceTracker) Stop() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// 检查stopChan是否已关闭，避免重复关闭
	select {
	case <-rt.stopChan:
		// 通道已关闭，不需要再次关闭
		rt.logger.Debug("资源追踪器已经停止，跳过重复停止")
		return
	default:
		// 通道未关闭，可以安全关闭
		close(rt.stopChan)
		// 释放所有资源
		rt.releaseAllNoLock()
		rt.logger.Debug("资源追踪器已停止")
	}
}
