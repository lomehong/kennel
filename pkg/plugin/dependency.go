package plugin

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
)

// PluginDependency 插件依赖
type PluginDependency struct {
	ID       string // 依赖的插件ID
	Version  string // 依赖的插件版本
	Optional bool   // 是否可选
}

// PluginDependencyManager 插件依赖管理器
type PluginDependencyManager struct {
	logger   hclog.Logger
	plugins  map[string]PluginMetadata
	graph    map[string][]string
	revGraph map[string][]string
	mu       sync.RWMutex
}

// NewPluginDependencyManager 创建插件依赖管理器
func NewPluginDependencyManager(logger hclog.Logger) *PluginDependencyManager {
	return &PluginDependencyManager{
		logger:   logger.Named("plugin-dependency-manager"),
		plugins:  make(map[string]PluginMetadata),
		graph:    make(map[string][]string),
		revGraph: make(map[string][]string),
	}
}

// AddPlugin 添加插件
func (pdm *PluginDependencyManager) AddPlugin(metadata PluginMetadata) error {
	pdm.mu.Lock()
	defer pdm.mu.Unlock()

	pdm.logger.Info("添加插件", "id", metadata.ID, "version", metadata.Version)

	// 检查插件是否已存在
	if _, exists := pdm.plugins[metadata.ID]; exists {
		return fmt.Errorf("插件已存在: %s", metadata.ID)
	}

	// 添加插件
	pdm.plugins[metadata.ID] = metadata

	// 初始化依赖图
	pdm.graph[metadata.ID] = make([]string, 0)
	if _, exists := pdm.revGraph[metadata.ID]; !exists {
		pdm.revGraph[metadata.ID] = make([]string, 0)
	}

	// 添加依赖关系
	for _, depID := range metadata.Dependencies {
		// 添加正向依赖
		pdm.graph[metadata.ID] = append(pdm.graph[metadata.ID], depID)

		// 添加反向依赖
		if _, exists := pdm.revGraph[depID]; !exists {
			pdm.revGraph[depID] = make([]string, 0)
		}
		pdm.revGraph[depID] = append(pdm.revGraph[depID], metadata.ID)
	}

	return nil
}

// RemovePlugin 移除插件
func (pdm *PluginDependencyManager) RemovePlugin(id string) error {
	pdm.mu.Lock()
	defer pdm.mu.Unlock()

	pdm.logger.Info("移除插件", "id", id)

	// 检查插件是否存在
	if _, exists := pdm.plugins[id]; !exists {
		return fmt.Errorf("插件不存在: %s", id)
	}

	// 检查是否有其他插件依赖此插件
	if deps, exists := pdm.revGraph[id]; exists && len(deps) > 0 {
		return fmt.Errorf("插件 %s 被其他插件依赖: %v", id, deps)
	}

	// 移除依赖关系
	for depID := range pdm.graph {
		for i, dep := range pdm.graph[depID] {
			if dep == id {
				pdm.graph[depID] = append(pdm.graph[depID][:i], pdm.graph[depID][i+1:]...)
				break
			}
		}
	}
	for depID := range pdm.revGraph {
		for i, dep := range pdm.revGraph[depID] {
			if dep == id {
				pdm.revGraph[depID] = append(pdm.revGraph[depID][:i], pdm.revGraph[depID][i+1:]...)
				break
			}
		}
	}

	// 移除插件
	delete(pdm.plugins, id)
	delete(pdm.graph, id)
	delete(pdm.revGraph, id)

	return nil
}

// GetPlugin 获取插件
func (pdm *PluginDependencyManager) GetPlugin(id string) (PluginMetadata, bool) {
	pdm.mu.RLock()
	defer pdm.mu.RUnlock()
	metadata, exists := pdm.plugins[id]
	return metadata, exists
}

// GetPlugins 获取所有插件
func (pdm *PluginDependencyManager) GetPlugins() map[string]PluginMetadata {
	pdm.mu.RLock()
	defer pdm.mu.RUnlock()

	// 复制插件映射
	plugins := make(map[string]PluginMetadata, len(pdm.plugins))
	for id, metadata := range pdm.plugins {
		plugins[id] = metadata
	}

	return plugins
}

// GetDependencies 获取插件的依赖
func (pdm *PluginDependencyManager) GetDependencies(id string) ([]string, error) {
	pdm.mu.RLock()
	defer pdm.mu.RUnlock()

	// 检查插件是否存在
	if _, exists := pdm.plugins[id]; !exists {
		return nil, fmt.Errorf("插件不存在: %s", id)
	}

	// 获取依赖
	deps := make([]string, len(pdm.graph[id]))
	copy(deps, pdm.graph[id])

	return deps, nil
}

// GetDependents 获取依赖此插件的插件
func (pdm *PluginDependencyManager) GetDependents(id string) ([]string, error) {
	pdm.mu.RLock()
	defer pdm.mu.RUnlock()

	// 检查插件是否存在
	if _, exists := pdm.plugins[id]; !exists {
		return nil, fmt.Errorf("插件不存在: %s", id)
	}

	// 获取依赖此插件的插件
	deps := make([]string, len(pdm.revGraph[id]))
	copy(deps, pdm.revGraph[id])

	return deps, nil
}

// GetLoadOrder 获取插件加载顺序
func (pdm *PluginDependencyManager) GetLoadOrder() ([]string, error) {
	pdm.mu.RLock()
	defer pdm.mu.RUnlock()

	// 创建临时图
	tempGraph := make(map[string][]string)
	for id, deps := range pdm.graph {
		tempGraph[id] = make([]string, len(deps))
		copy(tempGraph[id], deps)
	}

	// 创建入度映射
	inDegree := make(map[string]int)
	for id := range pdm.plugins {
		inDegree[id] = 0
	}
	for _, deps := range tempGraph {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// 创建队列
	var queue []string
	for id := range pdm.plugins {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	// 拓扑排序
	var order []string
	for len(queue) > 0 {
		// 出队
		id := queue[0]
		queue = queue[1:]

		// 添加到顺序
		order = append(order, id)

		// 更新入度
		for _, dep := range tempGraph[id] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// 检查是否有环
	if len(order) != len(pdm.plugins) {
		return nil, fmt.Errorf("依赖图中存在环")
	}

	// 反转顺序
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}

	return order, nil
}

// GetUnloadOrder 获取插件卸载顺序
func (pdm *PluginDependencyManager) GetUnloadOrder() ([]string, error) {
	pdm.mu.RLock()
	defer pdm.mu.RUnlock()

	// 创建临时图
	tempGraph := make(map[string][]string)
	for id, deps := range pdm.revGraph {
		tempGraph[id] = make([]string, len(deps))
		copy(tempGraph[id], deps)
	}

	// 创建入度映射
	inDegree := make(map[string]int)
	for id := range pdm.plugins {
		inDegree[id] = 0
	}
	for _, deps := range tempGraph {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// 创建队列
	var queue []string
	for id := range pdm.plugins {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	// 拓扑排序
	var order []string
	for len(queue) > 0 {
		// 出队
		id := queue[0]
		queue = queue[1:]

		// 添加到顺序
		order = append(order, id)

		// 更新入度
		for _, dep := range tempGraph[id] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// 检查是否有环
	if len(order) != len(pdm.plugins) {
		return nil, fmt.Errorf("依赖图中存在环")
	}

	return order, nil
}

// CheckDependencies 检查插件依赖
func (pdm *PluginDependencyManager) CheckDependencies(id string) (bool, []string, error) {
	pdm.mu.RLock()
	defer pdm.mu.RUnlock()

	// 检查插件是否存在
	metadata, exists := pdm.plugins[id]
	if !exists {
		return false, nil, fmt.Errorf("插件不存在: %s", id)
	}

	// 检查依赖
	var missingDeps []string
	for _, depID := range metadata.Dependencies {
		if _, exists := pdm.plugins[depID]; !exists {
			missingDeps = append(missingDeps, depID)
		}
	}

	return len(missingDeps) == 0, missingDeps, nil
}

// GetPluginsByTag 获取指定标签的插件
func (pdm *PluginDependencyManager) GetPluginsByTag(tag string) []PluginMetadata {
	pdm.mu.RLock()
	defer pdm.mu.RUnlock()

	var plugins []PluginMetadata
	for _, metadata := range pdm.plugins {
		for _, t := range metadata.Tags {
			if t == tag {
				plugins = append(plugins, metadata)
				break
			}
		}
	}

	return plugins
}

// GetPluginGraph 获取插件依赖图
func (pdm *PluginDependencyManager) GetPluginGraph() map[string][]string {
	pdm.mu.RLock()
	defer pdm.mu.RUnlock()

	// 复制依赖图
	graph := make(map[string][]string, len(pdm.graph))
	for id, deps := range pdm.graph {
		graph[id] = make([]string, len(deps))
		copy(graph[id], deps)
	}

	return graph
}

// GetPluginReverseGraph 获取插件反向依赖图
func (pdm *PluginDependencyManager) GetPluginReverseGraph() map[string][]string {
	pdm.mu.RLock()
	defer pdm.mu.RUnlock()

	// 复制反向依赖图
	revGraph := make(map[string][]string, len(pdm.revGraph))
	for id, deps := range pdm.revGraph {
		revGraph[id] = make([]string, len(deps))
		copy(revGraph[id], deps)
	}

	return revGraph
}
