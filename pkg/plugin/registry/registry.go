package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// PluginRegistry 插件注册表接口
// 管理已注册的插件
type PluginRegistry interface {
	// RegisterPlugin 注册插件
	// metadata: 插件元数据
	// 返回: 错误
	RegisterPlugin(metadata api.PluginMetadata) error

	// UnregisterPlugin 注销插件
	// id: 插件ID
	// 返回: 错误
	UnregisterPlugin(id string) error

	// GetPluginMetadata 获取插件元数据
	// id: 插件ID
	// 返回: 插件元数据和是否存在
	GetPluginMetadata(id string) (api.PluginMetadata, bool)

	// ListPlugins 列出所有插件
	// 返回: 插件元数据列表
	ListPlugins() []api.PluginMetadata

	// QueryPlugins 查询插件
	// query: 查询条件
	// 返回: 插件元数据列表
	QueryPlugins(query PluginQuery) []api.PluginMetadata

	// WatchRegistry 监听注册表变化
	// handler: 处理函数
	// 返回: 观察者和错误
	WatchRegistry(handler func(event RegistryEvent)) (api.Watcher, error)
}

// PluginQuery 插件查询条件
type PluginQuery struct {
	// 插件ID
	ID string

	// 插件名称
	Name string

	// 插件标签
	Tags []string

	// 插件能力
	Capabilities []string

	// 插件作者
	Author string

	// 插件许可证
	License string
}

// RegistryEvent 注册表事件
type RegistryEvent struct {
	// 事件类型: registered, unregistered, updated
	Type string

	// 插件ID
	PluginID string

	// 时间戳
	Timestamp time.Time

	// 插件元数据
	Metadata api.PluginMetadata
}

// DefaultPluginRegistry 默认插件注册表实现
type DefaultPluginRegistry struct {
	// 插件映射
	plugins map[string]api.PluginMetadata

	// 互斥锁
	mu sync.RWMutex

	// 日志记录器
	logger hclog.Logger

	// 事件处理器
	eventHandlers []func(event RegistryEvent)

	// 事件处理器互斥锁
	eventMu sync.RWMutex
}

// registryWatcher 注册表观察者
type registryWatcher struct {
	// 注册表
	registry *DefaultPluginRegistry

	// 处理函数
	handler func(event RegistryEvent)

	// 是否已停止
	stopped bool

	// 互斥锁
	mu sync.Mutex
}

// Stop 停止观察
func (w *registryWatcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return nil
	}

	w.stopped = true
	w.registry.removeEventHandler(w.handler)
	return nil
}

// NewPluginRegistry 创建一个新的插件注册表
func NewPluginRegistry(logger hclog.Logger) *DefaultPluginRegistry {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}

	return &DefaultPluginRegistry{
		plugins: make(map[string]api.PluginMetadata),
		logger:  logger.Named("plugin-registry"),
	}
}

// RegisterPlugin 注册插件
func (r *DefaultPluginRegistry) RegisterPlugin(metadata api.PluginMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查插件ID是否为空
	if metadata.ID == "" {
		return fmt.Errorf("插件ID不能为空")
	}

	// 检查插件是否已注册
	if _, exists := r.plugins[metadata.ID]; exists {
		return fmt.Errorf("插件 %s 已注册", metadata.ID)
	}

	// 注册插件
	r.plugins[metadata.ID] = metadata
	r.logger.Info("插件已注册", "id", metadata.ID, "name", metadata.Name)

	// 发送注册事件
	r.publishEvent(RegistryEvent{
		Type:      "registered",
		PluginID:  metadata.ID,
		Timestamp: time.Now(),
		Metadata:  metadata,
	})

	return nil
}

// UnregisterPlugin 注销插件
func (r *DefaultPluginRegistry) UnregisterPlugin(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查插件是否已注册
	metadata, exists := r.plugins[id]
	if !exists {
		return fmt.Errorf("插件 %s 未注册", id)
	}

	// 注销插件
	delete(r.plugins, id)
	r.logger.Info("插件已注销", "id", id)

	// 发送注销事件
	r.publishEvent(RegistryEvent{
		Type:      "unregistered",
		PluginID:  id,
		Timestamp: time.Now(),
		Metadata:  metadata,
	})

	return nil
}

// GetPluginMetadata 获取插件元数据
func (r *DefaultPluginRegistry) GetPluginMetadata(id string) (api.PluginMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	metadata, exists := r.plugins[id]
	return metadata, exists
}

// ListPlugins 列出所有插件
func (r *DefaultPluginRegistry) ListPlugins() []api.PluginMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 转换为列表
	plugins := make([]api.PluginMetadata, 0, len(r.plugins))
	for _, metadata := range r.plugins {
		plugins = append(plugins, metadata)
	}

	return plugins
}

// QueryPlugins 查询插件
func (r *DefaultPluginRegistry) QueryPlugins(query PluginQuery) []api.PluginMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 结果列表
	var results []api.PluginMetadata

	// 遍历所有插件
	for _, metadata := range r.plugins {
		// 检查是否匹配查询条件
		if !r.matchQuery(metadata, query) {
			continue
		}

		// 添加到结果列表
		results = append(results, metadata)
	}

	return results
}

// matchQuery 检查插件是否匹配查询条件
func (r *DefaultPluginRegistry) matchQuery(metadata api.PluginMetadata, query PluginQuery) bool {
	// 检查ID
	if query.ID != "" && metadata.ID != query.ID {
		return false
	}

	// 检查名称
	if query.Name != "" && metadata.Name != query.Name {
		return false
	}

	// 检查作者
	if query.Author != "" && metadata.Author != query.Author {
		return false
	}

	// 检查许可证
	if query.License != "" && metadata.License != query.License {
		return false
	}

	// 检查标签
	if len(query.Tags) > 0 {
		matched := false
		for _, tag := range query.Tags {
			for _, metadataTag := range metadata.Tags {
				if tag == metadataTag {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查能力
	if len(query.Capabilities) > 0 {
		matched := false
		for _, capability := range query.Capabilities {
			if enabled, ok := metadata.Capabilities[capability]; ok && enabled {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// WatchRegistry 监听注册表变化
func (r *DefaultPluginRegistry) WatchRegistry(handler func(event RegistryEvent)) (api.Watcher, error) {
	if handler == nil {
		return nil, fmt.Errorf("处理函数不能为空")
	}

	r.eventMu.Lock()
	r.eventHandlers = append(r.eventHandlers, handler)
	r.eventMu.Unlock()

	return &registryWatcher{
		registry: r,
		handler:  handler,
	}, nil
}

// publishEvent 发布事件
func (r *DefaultPluginRegistry) publishEvent(event RegistryEvent) {
	r.eventMu.RLock()
	handlers := make([]func(event RegistryEvent), len(r.eventHandlers))
	copy(handlers, r.eventHandlers)
	r.eventMu.RUnlock()

	// 调用所有处理器
	for _, handler := range handlers {
		go handler(event)
	}
}

// removeEventHandler 移除事件处理器
func (r *DefaultPluginRegistry) removeEventHandler(handler func(event RegistryEvent)) {
	r.eventMu.Lock()
	defer r.eventMu.Unlock()

	// 查找处理器
	for i, h := range r.eventHandlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			// 移除处理器
			r.eventHandlers = append(r.eventHandlers[:i], r.eventHandlers[i+1:]...)
			break
		}
	}
}
