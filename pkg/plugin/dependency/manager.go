package dependency

import (
	"fmt"
	"sort"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// DependencyManager 依赖管理器
// 负责管理插件之间的依赖关系
type DependencyManager struct {
	// 插件依赖图
	dependencyGraph map[string][]string
	
	// 插件版本映射
	versionMap map[string]string
	
	// 插件依赖映射
	dependencyMap map[string][]api.PluginDependency
	
	// 互斥锁
	mu sync.RWMutex
	
	// 日志记录器
	logger hclog.Logger
}

// NewDependencyManager 创建一个新的依赖管理器
func NewDependencyManager(logger hclog.Logger) *DependencyManager {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}
	
	return &DependencyManager{
		dependencyGraph: make(map[string][]string),
		versionMap:      make(map[string]string),
		dependencyMap:   make(map[string][]api.PluginDependency),
		logger:          logger.Named("dependency-manager"),
	}
}

// RegisterPlugin 注册插件
func (m *DependencyManager) RegisterPlugin(pluginID string, version string, dependencies []api.PluginDependency) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 检查插件是否已注册
	if _, exists := m.versionMap[pluginID]; exists {
		return fmt.Errorf("插件 %s 已注册", pluginID)
	}
	
	// 注册插件版本
	m.versionMap[pluginID] = version
	
	// 注册插件依赖
	m.dependencyMap[pluginID] = dependencies
	
	// 构建依赖图
	m.dependencyGraph[pluginID] = make([]string, 0)
	for _, dep := range dependencies {
		if !dep.Optional {
			m.dependencyGraph[pluginID] = append(m.dependencyGraph[pluginID], dep.ID)
		}
	}
	
	m.logger.Debug("注册插件", "id", pluginID, "version", version, "dependencies", len(dependencies))
	return nil
}

// UnregisterPlugin 注销插件
func (m *DependencyManager) UnregisterPlugin(pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 检查插件是否已注册
	if _, exists := m.versionMap[pluginID]; !exists {
		return fmt.Errorf("插件 %s 未注册", pluginID)
	}
	
	// 删除插件版本
	delete(m.versionMap, pluginID)
	
	// 删除插件依赖
	delete(m.dependencyMap, pluginID)
	
	// 删除依赖图中的插件
	delete(m.dependencyGraph, pluginID)
	
	// 删除依赖图中对该插件的依赖
	for id, deps := range m.dependencyGraph {
		newDeps := make([]string, 0)
		for _, dep := range deps {
			if dep != pluginID {
				newDeps = append(newDeps, dep)
			}
		}
		m.dependencyGraph[id] = newDeps
	}
	
	m.logger.Debug("注销插件", "id", pluginID)
	return nil
}

// CheckDependencies 检查依赖
func (m *DependencyManager) CheckDependencies(pluginID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 检查插件是否已注册
	if _, exists := m.versionMap[pluginID]; !exists {
		return nil, fmt.Errorf("插件 %s 未注册", pluginID)
	}
	
	// 获取插件依赖
	dependencies, exists := m.dependencyMap[pluginID]
	if !exists {
		return nil, fmt.Errorf("插件 %s 依赖未注册", pluginID)
	}
	
	// 检查依赖
	var missingDependencies []string
	for _, dep := range dependencies {
		// 如果是可选依赖，跳过
		if dep.Optional {
			continue
		}
		
		// 检查依赖是否存在
		depVersion, exists := m.versionMap[dep.ID]
		if !exists {
			missingDependencies = append(missingDependencies, dep.ID)
			continue
		}
		
		// 检查版本是否兼容
		if !m.checkVersionCompatibility(depVersion, dep.Version) {
			missingDependencies = append(missingDependencies, fmt.Sprintf("%s@%s", dep.ID, dep.Version))
		}
	}
	
	if len(missingDependencies) > 0 {
		return missingDependencies, fmt.Errorf("插件 %s 缺少依赖: %v", pluginID, missingDependencies)
	}
	
	return nil, nil
}

// checkVersionCompatibility 检查版本兼容性
func (m *DependencyManager) checkVersionCompatibility(version, constraint string) bool {
	// 如果约束为空，认为兼容
	if constraint == "" {
		return true
	}
	
	// 解析版本
	v, err := semver.NewVersion(version)
	if err != nil {
		m.logger.Error("解析版本失败", "version", version, "error", err)
		return false
	}
	
	// 解析约束
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		m.logger.Error("解析约束失败", "constraint", constraint, "error", err)
		return false
	}
	
	// 检查版本是否满足约束
	return c.Check(v)
}

// GetDependencyOrder 获取依赖顺序
func (m *DependencyManager) GetDependencyOrder() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 拓扑排序
	return m.topologicalSort()
}

// topologicalSort 拓扑排序
func (m *DependencyManager) topologicalSort() ([]string, error) {
	// 创建入度映射
	inDegree := make(map[string]int)
	for node := range m.dependencyGraph {
		inDegree[node] = 0
	}
	
	// 计算入度
	for _, deps := range m.dependencyGraph {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}
	
	// 创建队列，将入度为0的节点入队
	var queue []string
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}
	
	// 拓扑排序
	var result []string
	for len(queue) > 0 {
		// 出队
		node := queue[0]
		queue = queue[1:]
		
		// 添加到结果
		result = append(result, node)
		
		// 更新邻居的入度
		for _, neighbor := range m.dependencyGraph[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}
	
	// 检查是否有环
	if len(result) != len(m.dependencyGraph) {
		return nil, fmt.Errorf("依赖图中存在环")
	}
	
	// 反转结果，使依赖在前
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	
	return result, nil
}

// GetPluginDependencies 获取插件依赖
func (m *DependencyManager) GetPluginDependencies(pluginID string) ([]api.PluginDependency, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 检查插件是否已注册
	if _, exists := m.versionMap[pluginID]; !exists {
		return nil, fmt.Errorf("插件 %s 未注册", pluginID)
	}
	
	// 获取插件依赖
	dependencies, exists := m.dependencyMap[pluginID]
	if !exists {
		return nil, fmt.Errorf("插件 %s 依赖未注册", pluginID)
	}
	
	return dependencies, nil
}

// GetPluginVersion 获取插件版本
func (m *DependencyManager) GetPluginVersion(pluginID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 检查插件是否已注册
	version, exists := m.versionMap[pluginID]
	if !exists {
		return "", fmt.Errorf("插件 %s 未注册", pluginID)
	}
	
	return version, nil
}

// GetDependents 获取依赖于指定插件的插件
func (m *DependencyManager) GetDependents(pluginID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var dependents []string
	for id, deps := range m.dependencyGraph {
		for _, dep := range deps {
			if dep == pluginID {
				dependents = append(dependents, id)
				break
			}
		}
	}
	
	// 排序结果
	sort.Strings(dependents)
	
	return dependents
}

// GetAllPlugins 获取所有插件
func (m *DependencyManager) GetAllPlugins() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var plugins []string
	for id := range m.versionMap {
		plugins = append(plugins, id)
	}
	
	// 排序结果
	sort.Strings(plugins)
	
	return plugins
}

// GetDependencyGraph 获取依赖图
func (m *DependencyManager) GetDependencyGraph() map[string][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 复制依赖图
	graph := make(map[string][]string)
	for id, deps := range m.dependencyGraph {
		graph[id] = make([]string, len(deps))
		copy(graph[id], deps)
	}
	
	return graph
}

// GetVersionMap 获取版本映射
func (m *DependencyManager) GetVersionMap() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 复制版本映射
	versionMap := make(map[string]string)
	for id, version := range m.versionMap {
		versionMap[id] = version
	}
	
	return versionMap
}
