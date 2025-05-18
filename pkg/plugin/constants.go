package plugin

// 插件隔离级别
const (
	PluginIsolationLevelNone   = "none"   // 无隔离
	PluginIsolationLevelBasic  = "basic"  // 基本隔离
	PluginIsolationLevelStrict = "strict" // 严格隔离
)

// 插件状态
const (
	PluginStatusUnloaded = "unloaded" // 未加载
	PluginStatusLoaded   = "loaded"   // 已加载
	PluginStatusRunning  = "running"  // 运行中
	PluginStatusStopped  = "stopped"  // 已停止
	PluginStatusError    = "error"    // 错误
)

// 插件类型
const (
	PluginTypeCore    = "core"    // 核心插件
	PluginTypeService = "service" // 服务插件
	PluginTypeUI      = "ui"      // UI插件
	PluginTypeUtil    = "util"    // 工具插件
)

// 插件事件
const (
	PluginEventLoaded   = "loaded"   // 已加载
	PluginEventUnloaded = "unloaded" // 已卸载
	PluginEventStarted  = "started"  // 已启动
	PluginEventStopped  = "stopped"  // 已停止
	PluginEventError    = "error"    // 错误
)

// 插件权限
const (
	PluginPermissionNone      = "none"      // 无权限
	PluginPermissionReadOnly  = "readonly"  // 只读权限
	PluginPermissionReadWrite = "readwrite" // 读写权限
	PluginPermissionAdmin     = "admin"     // 管理员权限
)
