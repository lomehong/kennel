package plugin

import (
	"context"
	"time"
)

// Module 定义了插件模块的基础接口
type Module interface {
	// Init 初始化模块
	// ctx: 上下文，可用于取消操作
	// config: 模块配置
	Init(ctx context.Context, config *ModuleConfig) error

	// Start 启动模块
	// 返回nil表示成功启动
	Start() error

	// Stop 停止模块
	// 应该释放所有资源并优雅退出
	Stop() error

	// GetInfo 获取模块信息
	// 返回模块的元数据
	GetInfo() ModuleInfo

	// HandleRequest 处理请求
	// ctx: 请求上下文
	// req: 请求对象
	// 返回响应对象和错误
	HandleRequest(ctx context.Context, req *Request) (*Response, error)

	// HandleEvent 处理事件
	// ctx: 事件上下文
	// event: 事件对象
	// 返回处理错误
	HandleEvent(ctx context.Context, event *Event) error
}

// ModuleConfig 模块配置
type ModuleConfig struct {
	// ID 模块唯一标识符
	ID string `json:"id"`

	// Name 模块名称
	Name string `json:"name"`

	// Version 模块版本
	Version string `json:"version"`

	// Settings 模块特定设置
	Settings map[string]interface{} `json:"settings"`

	// Dependencies 依赖的其他模块
	Dependencies []string `json:"dependencies"`

	// Resources 资源限制
	Resources ResourceLimits `json:"resources"`
}

// ModuleInfo 模块信息
type ModuleInfo struct {
	// ID 模块唯一标识符
	ID string `json:"id"`

	// Name 模块名称
	Name string `json:"name"`

	// Version 模块版本
	Version string `json:"version"`

	// Description 模块描述
	Description string `json:"description"`

	// Author 作者信息
	Author string `json:"author"`

	// License 许可证信息
	License string `json:"license"`

	// Capabilities 模块能力
	Capabilities []string `json:"capabilities"`

	// SupportedPlatforms 支持的平台
	SupportedPlatforms []string `json:"supported_platforms"`

	// Language 实现语言
	Language string `json:"language"`
}

// Request 请求对象
type Request struct {
	// ID 请求唯一标识符
	ID string `json:"id"`

	// Action 请求的操作
	Action string `json:"action"`

	// Params 请求参数
	Params map[string]interface{} `json:"params"`

	// Metadata 请求元数据
	Metadata map[string]string `json:"metadata"`

	// Timeout 请求超时时间（毫秒）
	Timeout int64 `json:"timeout"`
}

// Response 响应对象
type Response struct {
	// ID 对应请求的ID
	ID string `json:"id"`

	// Success 是否成功
	Success bool `json:"success"`

	// Data 响应数据
	Data map[string]interface{} `json:"data"`

	// Error 错误信息
	Error *ErrorInfo `json:"error,omitempty"`

	// Metadata 响应元数据
	Metadata map[string]string `json:"metadata"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	// Code 错误代码
	Code string `json:"code"`

	// Message 错误消息
	Message string `json:"message"`

	// Details 错误详情
	Details map[string]interface{} `json:"details,omitempty"`
}

// Event 事件对象
type Event struct {
	// ID 事件唯一标识符
	ID string `json:"id"`

	// Type 事件类型
	Type string `json:"type"`

	// Source 事件源
	Source string `json:"source"`

	// Timestamp 事件时间戳
	Timestamp int64 `json:"timestamp"`

	// Data 事件数据
	Data map[string]interface{} `json:"data"`

	// Metadata 事件元数据
	Metadata map[string]string `json:"metadata"`
}

// HealthCheck 健康检查接口
type HealthCheck interface {
	// CheckHealth 检查健康状态
	// 返回健康状态信息
	CheckHealth() HealthStatus
}

// HealthStatus 健康状态
type HealthStatus struct {
	// Status 状态（"healthy", "degraded", "unhealthy"）
	Status string `json:"status"`

	// Details 详细状态信息
	Details map[string]interface{} `json:"details"`

	// Timestamp 状态时间戳
	Timestamp int64 `json:"timestamp"`
}

// ResourceManager 资源管理接口
type ResourceManager interface {
	// GetResourceUsage 获取资源使用情况
	GetResourceUsage() ResourceUsage

	// SetResourceLimits 设置资源限制
	SetResourceLimits(limits ResourceLimits) error
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	// CPU CPU使用率（百分比）
	CPU float64 `json:"cpu"`

	// Memory 内存使用量（字节）
	Memory int64 `json:"memory"`

	// Disk 磁盘使用量（字节）
	Disk int64 `json:"disk"`

	// Network 网络使用量（字节/秒）
	Network int64 `json:"network"`
}

// ResourceLimits 资源限制
type ResourceLimits struct {
	// MaxCPU 最大CPU使用率（百分比）
	MaxCPU float64 `json:"max_cpu"`

	// MaxMemory 最大内存使用量（字节）
	MaxMemory int64 `json:"max_memory"`

	// MaxDisk 最大磁盘使用量（字节）
	MaxDisk int64 `json:"max_disk"`

	// MaxNetwork 最大网络使用量（字节/秒）
	MaxNetwork int64 `json:"max_network"`
}

// PluginMetadata 插件元数据
type PluginMetadata struct {
	// ID 插件唯一标识符
	ID string `json:"id"`

	// Name 插件名称
	Name string `json:"name"`

	// Version 插件版本
	Version string `json:"version"`

	// Description 插件描述
	Description string `json:"description"`

	// EntryPoint 插件入口点
	EntryPoint PluginEntryPoint `json:"entry_point"`

	// Dependencies 依赖的其他插件
	Dependencies []string `json:"dependencies"`

	// Capabilities 插件能力
	Capabilities []string `json:"capabilities"`

	// SupportedPlatforms 支持的平台
	SupportedPlatforms []string `json:"supported_platforms"`

	// Language 实现语言
	Language string `json:"language"`

	// Author 作者信息
	Author string `json:"author"`

	// License 许可证信息
	License string `json:"license"`

	// MinFrameworkVersion 最低框架版本
	MinFrameworkVersion string `json:"min_framework_version"`

	// Path 插件路径（运行时填充）
	Path string `json:"-"`
}

// PluginEntryPoint 插件入口点
type PluginEntryPoint struct {
	// Type 入口点类型（"go", "python"）
	Type string `json:"type"`

	// Path 入口点路径
	Path string `json:"path"`

	// Interpreter 解释器（仅用于脚本语言）
	Interpreter string `json:"interpreter,omitempty"`
}

// PluginState 插件状态
type PluginState string

// 插件状态常量
const (
	PluginStateUnknown      PluginState = "unknown"
	PluginStateInitializing PluginState = "initializing"
	PluginStateRunning      PluginState = "running"
	PluginStatePaused       PluginState = "paused"
	PluginStateStopped      PluginState = "stopped"
	PluginStateError        PluginState = "error"
)

// PluginInfo 插件运行时信息
type PluginInfo struct {
	// Metadata 插件元数据
	Metadata PluginMetadata `json:"metadata"`

	// State 插件状态
	State PluginState `json:"state"`

	// StartTime 启动时间
	StartTime time.Time `json:"start_time"`

	// StopTime 停止时间
	StopTime time.Time `json:"stop_time,omitempty"`

	// LastError 最后一次错误
	LastError string `json:"last_error,omitempty"`

	// ResourceUsage 资源使用情况
	ResourceUsage ResourceUsage `json:"resource_usage"`
}
