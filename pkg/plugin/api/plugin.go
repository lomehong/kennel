package api

import (
	"context"
	"time"
)

// Plugin 定义了所有插件必须实现的基础接口
// 这是插件系统的核心接口，所有插件都必须实现这个接口
type Plugin interface {
	// GetInfo 返回插件的基本信息
	GetInfo() PluginInfo

	// Init 初始化插件
	// ctx: 上下文，可用于超时控制和取消操作
	// config: 插件配置
	// 返回: 初始化过程中的错误
	Init(ctx context.Context, config PluginConfig) error

	// Start 启动插件
	// ctx: 上下文，可用于超时控制和取消操作
	// 返回: 启动过程中的错误
	Start(ctx context.Context) error

	// Stop 停止插件
	// ctx: 上下文，可用于超时控制和取消操作
	// 返回: 停止过程中的错误
	Stop(ctx context.Context) error

	// HealthCheck 执行健康检查
	// ctx: 上下文，可用于超时控制和取消操作
	// 返回: 健康状态和检查过程中的错误
	HealthCheck(ctx context.Context) (HealthStatus, error)
}

// PluginInfo 包含插件的基本信息
type PluginInfo struct {
	ID           string             // 插件唯一标识符
	Name         string             // 插件名称
	Version      string             // 插件版本
	Description  string             // 插件描述
	Author       string             // 插件作者
	License      string             // 插件许可证
	Tags         []string           // 插件标签，用于分类和筛选
	Capabilities map[string]bool    // 插件能力，表示插件支持的功能
	Dependencies []PluginDependency // 插件依赖，表示插件依赖的其他插件
}

// PluginDependency 定义了插件的依赖关系
type PluginDependency struct {
	ID       string // 依赖的插件ID
	Version  string // 依赖的插件版本要求（语义化版本表达式）
	Optional bool   // 是否为可选依赖
}

// PluginConfig 定义了插件的配置
type PluginConfig struct {
	// 基本配置
	ID       string // 插件ID
	Enabled  bool   // 是否启用
	LogLevel string // 日志级别

	// 运行时配置
	AutoStart   bool // 是否自动启动
	AutoRestart bool // 是否自动重启

	// 隔离配置
	Isolation IsolationConfig // 隔离配置

	// 自定义配置
	Settings map[string]interface{} // 自定义设置，插件特定的配置项
}

// IsolationConfig 定义了插件的隔离配置
type IsolationConfig struct {
	Level       string            // 隔离级别: none, basic, strict, complete
	Resources   map[string]int64  // 资源限制: memory, cpu, etc.
	Timeout     time.Duration     // 操作超时时间
	WorkingDir  string            // 工作目录
	Environment map[string]string // 环境变量
}

// HealthStatus 定义了插件的健康状态
type HealthStatus struct {
	Status      string                 // 状态: healthy, degraded, unhealthy
	Details     map[string]interface{} // 详细信息
	LastChecked time.Time              // 最后检查时间
}

// PluginState 表示插件的状态
type PluginState string

// 预定义的插件状态
const (
	PluginStateUnknown     PluginState = "unknown"     // 未知状态
	PluginStateDiscovered  PluginState = "discovered"  // 已发现
	PluginStateRegistered  PluginState = "registered"  // 已注册
	PluginStateLoaded      PluginState = "loaded"      // 已加载
	PluginStateInitialized PluginState = "initialized" // 已初始化
	PluginStateStarting    PluginState = "starting"    // 启动中
	PluginStateRunning     PluginState = "running"     // 运行中
	PluginStateStopping    PluginState = "stopping"    // 停止中
	PluginStateStopped     PluginState = "stopped"     // 已停止
	PluginStateFailed      PluginState = "failed"      // 失败
	PluginStateUnloading   PluginState = "unloading"   // 卸载中
	PluginStateUnloaded    PluginState = "unloaded"    // 已卸载
)

// PluginStatus 定义了插件的状态信息
type PluginStatus struct {
	ID         string                 // 插件ID
	State      PluginState            // 状态
	Health     HealthStatus           // 健康状态
	StartTime  time.Time              // 启动时间
	StopTime   time.Time              // 停止时间
	Error      string                 // 错误信息
	Statistics map[string]interface{} // 统计信息
}

// PluginEvent 定义了插件事件
type PluginEvent struct {
	Type      string                 // 事件类型: loaded, unloaded, started, stopped, error
	PluginID  string                 // 插件ID
	Timestamp time.Time              // 时间戳
	Data      map[string]interface{} // 事件数据
}

// Watcher 定义了观察者接口
type Watcher interface {
	// Stop 停止观察
	Stop() error
}

// PluginMetadata 插件元数据
type PluginMetadata struct {
	// 插件ID
	ID string

	// 插件名称
	Name string

	// 插件版本
	Version string

	// 插件描述
	Description string

	// 插件作者
	Author string

	// 插件许可证
	License string

	// 插件标签
	Tags []string

	// 插件能力
	Capabilities map[string]bool

	// 插件依赖
	Dependencies []PluginDependency

	// 插件位置
	Location PluginLocation

	// 插件签名
	Signature string

	// 注册时间
	Timestamp time.Time
}

// PluginLocation 插件位置
type PluginLocation struct {
	// 位置类型: local, remote, registry
	Type string

	// 路径
	Path string

	// URL
	URL string

	// 版本
	Version string
}

// PluginStateInitializing 初始化中状态
const PluginStateInitializing PluginState = "initializing"
