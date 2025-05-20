package dependency

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// DependencyResolver 依赖解析器
// 负责解析插件依赖
type DependencyResolver struct {
	// 依赖管理器
	manager *DependencyManager
	
	// 依赖注入器
	injector *DependencyInjector
	
	// 插件提供者
	pluginProvider PluginProvider
	
	// 互斥锁
	mu sync.RWMutex
	
	// 日志记录器
	logger hclog.Logger
	
	// 已解析的插件
	resolvedPlugins map[string]api.Plugin
}

// PluginProvider 插件提供者接口
// 用于获取插件实例
type PluginProvider interface {
	// GetPlugin 获取插件
	// id: 插件ID
	// 返回: 插件实例和是否存在
	GetPlugin(id string) (api.Plugin, bool)
}

// NewDependencyResolver 创建一个新的依赖解析器
func NewDependencyResolver(manager *DependencyManager, injector *DependencyInjector, provider PluginProvider, logger hclog.Logger) *DependencyResolver {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}
	
	return &DependencyResolver{
		manager:         manager,
		injector:        injector,
		pluginProvider:  provider,
		logger:          logger.Named("dependency-resolver"),
		resolvedPlugins: make(map[string]api.Plugin),
	}
}

// ResolvePluginDependencies 解析插件依赖
func (r *DependencyResolver) ResolvePluginDependencies(pluginID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 检查插件是否已解析
	if _, exists := r.resolvedPlugins[pluginID]; exists {
		return nil
	}
	
	// 获取插件
	plugin, exists := r.pluginProvider.GetPlugin(pluginID)
	if !exists {
		return fmt.Errorf("插件 %s 不存在", pluginID)
	}
	
	// 获取插件依赖
	dependencies, err := r.manager.GetPluginDependencies(pluginID)
	if err != nil {
		return fmt.Errorf("获取插件 %s 依赖失败: %w", pluginID, err)
	}
	
	// 解析依赖
	for _, dep := range dependencies {
		// 如果是可选依赖，跳过
		if dep.Optional {
			continue
		}
		
		// 解析依赖
		if err := r.ResolvePluginDependencies(dep.ID); err != nil {
			return fmt.Errorf("解析插件 %s 依赖 %s 失败: %w", pluginID, dep.ID, err)
		}
		
		// 获取依赖插件
		depPlugin, exists := r.resolvedPlugins[dep.ID]
		if !exists {
			return fmt.Errorf("依赖插件 %s 未解析", dep.ID)
		}
		
		// 注册依赖服务
		if err := r.injector.RegisterService(dep.ID, depPlugin); err != nil {
			return fmt.Errorf("注册依赖服务 %s 失败: %w", dep.ID, err)
		}
	}
	
	// 注入依赖
	if err := r.injector.Inject(plugin); err != nil {
		return fmt.Errorf("注入插件 %s 依赖失败: %w", pluginID, err)
	}
	
	// 标记为已解析
	r.resolvedPlugins[pluginID] = plugin
	
	r.logger.Debug("解析插件依赖", "id", pluginID, "dependencies", len(dependencies))
	return nil
}

// ResolveAllPluginDependencies 解析所有插件依赖
func (r *DependencyResolver) ResolveAllPluginDependencies() error {
	// 获取依赖顺序
	order, err := r.manager.GetDependencyOrder()
	if err != nil {
		return fmt.Errorf("获取依赖顺序失败: %w", err)
	}
	
	// 按顺序解析依赖
	for _, pluginID := range order {
		if err := r.ResolvePluginDependencies(pluginID); err != nil {
			return fmt.Errorf("解析插件 %s 依赖失败: %w", pluginID, err)
		}
	}
	
	return nil
}

// GetResolvedPlugin 获取已解析的插件
func (r *DependencyResolver) GetResolvedPlugin(pluginID string) (api.Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugin, exists := r.resolvedPlugins[pluginID]
	return plugin, exists
}

// GetAllResolvedPlugins 获取所有已解析的插件
func (r *DependencyResolver) GetAllResolvedPlugins() map[string]api.Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// 复制插件映射
	plugins := make(map[string]api.Plugin)
	for id, plugin := range r.resolvedPlugins {
		plugins[id] = plugin
	}
	
	return plugins
}

// Clear 清除所有已解析的插件
func (r *DependencyResolver) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.resolvedPlugins = make(map[string]api.Plugin)
	r.injector.Clear()
	
	r.logger.Debug("清除所有已解析的插件")
}

// RegisterPluginService 注册插件服务
func (r *DependencyResolver) RegisterPluginService(pluginID string, serviceName string, service interface{}) error {
	// 注册服务
	if err := r.injector.RegisterService(serviceName, service); err != nil {
		return fmt.Errorf("注册服务 %s 失败: %w", serviceName, err)
	}
	
	r.logger.Debug("注册插件服务", "plugin_id", pluginID, "service_name", serviceName)
	return nil
}

// RegisterPluginFactory 注册插件工厂
func (r *DependencyResolver) RegisterPluginFactory(pluginID string, serviceName string, factory Factory) error {
	// 注册工厂
	if err := r.injector.RegisterFactory(serviceName, factory); err != nil {
		return fmt.Errorf("注册工厂 %s 失败: %w", serviceName, err)
	}
	
	r.logger.Debug("注册插件工厂", "plugin_id", pluginID, "service_name", serviceName)
	return nil
}

// GetPluginService 获取插件服务
func (r *DependencyResolver) GetPluginService(serviceName string) (interface{}, error) {
	// 获取服务
	service, err := r.injector.GetService(serviceName)
	if err != nil {
		return nil, fmt.Errorf("获取服务 %s 失败: %w", serviceName, err)
	}
	
	return service, nil
}

// InjectPluginDependencies 注入插件依赖
func (r *DependencyResolver) InjectPluginDependencies(target interface{}) error {
	// 注入依赖
	if err := r.injector.Inject(target); err != nil {
		return fmt.Errorf("注入依赖失败: %w", err)
	}
	
	return nil
}

// InjectPluginMethod 注入插件方法参数
func (r *DependencyResolver) InjectPluginMethod(target interface{}, methodName string, args ...interface{}) ([]interface{}, error) {
	// 注入方法参数
	results, err := r.injector.InjectMethod(target, methodName, args...)
	if err != nil {
		return nil, fmt.Errorf("注入方法 %s 参数失败: %w", methodName, err)
	}
	
	// 转换结果
	var returnValues []interface{}
	for _, result := range results {
		returnValues = append(returnValues, result.Interface())
	}
	
	return returnValues, nil
}
