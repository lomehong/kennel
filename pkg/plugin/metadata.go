package plugin

import (
	"time"
)

// PluginMetadata 插件元数据
type PluginMetadata struct {
	ID             string    // 插件ID
	Name           string    // 插件名称
	Version        string    // 插件版本
	Author         string    // 插件作者
	Description    string    // 插件描述
	Website        string    // 插件网站
	License        string    // 插件许可证
	Dependencies   []string  // 插件依赖
	Tags           []string  // 插件标签
	CreatedAt      time.Time // 创建时间
	UpdatedAt      time.Time // 更新时间
	Enabled        bool      // 是否启用
	Optional       bool      // 是否可选
	Path           string    // 插件路径
	EntryPoint     string    // 插件入口点
	IsolationLevel string    // 隔离级别
}

// NewPluginMetadata 创建插件元数据
func NewPluginMetadata(id, name, version string) PluginMetadata {
	now := time.Now()
	return PluginMetadata{
		ID:           id,
		Name:         name,
		Version:      version,
		CreatedAt:    now,
		UpdatedAt:    now,
		Dependencies: make([]string, 0),
		Tags:         make([]string, 0),
		Enabled:      true,
		Optional:     false,
	}
}

// AddDependency 添加依赖
func (pm *PluginMetadata) AddDependency(id string) {
	pm.Dependencies = append(pm.Dependencies, id)
}

// AddTag 添加标签
func (pm *PluginMetadata) AddTag(tag string) {
	pm.Tags = append(pm.Tags, tag)
}

// HasTag 是否有标签
func (pm *PluginMetadata) HasTag(tag string) bool {
	for _, t := range pm.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// HasDependency 是否有依赖
func (pm *PluginMetadata) HasDependency(id string) bool {
	for _, dep := range pm.Dependencies {
		if dep == id {
			return true
		}
	}
	return false
}

// Update 更新元数据
func (pm *PluginMetadata) Update() {
	pm.UpdatedAt = time.Now()
}
