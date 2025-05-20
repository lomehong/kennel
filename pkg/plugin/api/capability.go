package api

import (
	"context"
)

// Configurable 定义了可配置的插件能力
// 实现此接口的插件支持动态配置更新
type Configurable interface {
	// UpdateConfig 更新插件配置
	// ctx: 上下文
	// config: 新的配置
	// 返回: 更新过程中的错误
	UpdateConfig(ctx context.Context, config map[string]interface{}) error

	// GetConfig 获取当前配置
	// 返回: 当前配置
	GetConfig() map[string]interface{}

	// ValidateConfig 验证配置
	// config: 待验证的配置
	// 返回: 验证结果和错误信息
	ValidateConfig(config map[string]interface{}) (bool, error)
}

// Versionable 定义了可版本化的插件能力
// 实现此接口的插件支持版本管理
type Versionable interface {
	// GetVersion 获取版本信息
	// 返回: 版本信息
	GetVersion() VersionInfo

	// CheckCompatibility 检查兼容性
	// requiredVersion: 所需版本
	// 返回: 是否兼容
	CheckCompatibility(requiredVersion string) bool
}

// VersionInfo 定义了版本信息
type VersionInfo struct {
	Version     string   // 版本号
	BuildTime   string   // 构建时间
	GitCommit   string   // Git提交哈希
	APIVersions []string // 支持的API版本
}

// Debuggable 定义了可调试的插件能力
// 实现此接口的插件支持调试功能
type Debuggable interface {
	// EnableDebug 启用调试
	// level: 调试级别
	// 返回: 错误
	EnableDebug(level int) error

	// DisableDebug 禁用调试
	// 返回: 错误
	DisableDebug() error

	// GetDebugInfo 获取调试信息
	// 返回: 调试信息
	GetDebugInfo() map[string]interface{}
}

// Monitorable 定义了可监控的插件能力
// 实现此接口的插件支持监控功能
type Monitorable interface {
	// GetMetrics 获取指标
	// 返回: 指标数据
	GetMetrics() map[string]interface{}

	// RegisterMetricsCollector 注册指标收集器
	// collector: 指标收集器
	// 返回: 错误
	RegisterMetricsCollector(collector MetricsCollector) error
}

// MetricsCollector 定义了指标收集器接口
type MetricsCollector interface {
	// CollectMetric 收集指标
	// name: 指标名称
	// value: 指标值
	// labels: 标签
	CollectMetric(name string, value float64, labels map[string]string) error
}

// Traceable 定义了可追踪的插件能力
// 实现此接口的插件支持分布式追踪
type Traceable interface {
	// StartSpan 开始一个追踪span
	// ctx: 上下文
	// operationName: 操作名称
	// 返回: 新的上下文和span
	StartSpan(ctx context.Context, operationName string) (context.Context, interface{})

	// FinishSpan 结束一个追踪span
	// span: 要结束的span
	FinishSpan(span interface{})

	// AddSpanTag 添加span标签
	// span: 目标span
	// key: 标签键
	// value: 标签值
	AddSpanTag(span interface{}, key string, value interface{})
}

// Loggable 定义了可记录日志的插件能力
// 实现此接口的插件支持结构化日志记录
type Loggable interface {
	// Log 记录日志
	// level: 日志级别
	// msg: 日志消息
	// fields: 日志字段
	Log(level string, msg string, fields map[string]interface{})

	// SetLogLevel 设置日志级别
	// level: 日志级别
	SetLogLevel(level string)

	// GetLogLevel 获取日志级别
	// 返回: 当前日志级别
	GetLogLevel() string
}

// Documentable 定义了可文档化的插件能力
// 实现此接口的插件提供自文档功能
type Documentable interface {
	// GetDocumentation 获取文档
	// 返回: 文档内容
	GetDocumentation() Documentation
}

// Documentation 定义了文档
type Documentation struct {
	Overview    string                   // 概述
	Usage       string                   // 使用说明
	API         map[string]APIDoc        // API文档
	Examples    []Example                // 示例
	References  []Reference              // 参考资料
}

// APIDoc 定义了API文档
type APIDoc struct {
	Name        string                   // API名称
	Description string                   // API描述
	Parameters  []Parameter              // 参数
	Returns     []Parameter              // 返回值
	Errors      []Error                  // 错误
}

// Parameter 定义了参数
type Parameter struct {
	Name        string                   // 参数名称
	Type        string                   // 参数类型
	Description string                   // 参数描述
	Required    bool                     // 是否必需
	Default     interface{}              // 默认值
}

// Error 定义了错误
type Error struct {
	Code        string                   // 错误码
	Description string                   // 错误描述
	Resolution  string                   // 解决方案
}

// Example 定义了示例
type Example struct {
	Title       string                   // 示例标题
	Description string                   // 示例描述
	Code        string                   // 示例代码
	Output      string                   // 示例输出
}

// Reference 定义了参考资料
type Reference struct {
	Title       string                   // 标题
	URL         string                   // URL
	Description string                   // 描述
}
