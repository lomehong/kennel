package config

import (
	"reflect"
)

// CreateAssetsValidator 创建资产管理插件配置验证器
func CreateAssetsValidator() *PluginConfigValidator {
	validator := NewPluginConfigValidator("assets")

	// 基础字段
	validator.AddRequiredField("enabled")
	validator.AddFieldType("enabled", reflect.Bool)
	validator.AddDefault("enabled", true)

	// 收集间隔
	validator.AddRequiredField("collect_interval")
	validator.AddFieldType("collect_interval", reflect.Float64) // YAML解析数字为float64
	validator.AddFieldValidator("collect_interval", DurationValidator(60, 86400)) // 1分钟到1天
	validator.AddDefault("collect_interval", 3600)

	// 上报服务器
	validator.AddFieldType("report_server", reflect.String)
	validator.AddFieldValidator("report_server", URLValidator())
	validator.AddDefault("report_server", "")

	// 自动上报
	validator.AddFieldType("auto_report", reflect.Bool)
	validator.AddDefault("auto_report", false)

	// 日志级别
	validator.AddFieldType("log_level", reflect.String)
	validator.AddFieldValidator("log_level", StringEnumValidator("debug", "info", "warn", "error"))
	validator.AddDefault("log_level", "info")

	// 缓存配置
	validator.AddFieldType("cache", reflect.Map)

	return validator
}

// CreateAuditValidator 创建安全审计插件配置验证器
func CreateAuditValidator() *PluginConfigValidator {
	validator := NewPluginConfigValidator("audit")

	// 基础字段
	validator.AddRequiredField("enabled")
	validator.AddFieldType("enabled", reflect.Bool)
	validator.AddDefault("enabled", true)

	// 系统事件记录
	validator.AddFieldType("log_system_events", reflect.Bool)
	validator.AddDefault("log_system_events", true)

	// 用户事件记录
	validator.AddFieldType("log_user_events", reflect.Bool)
	validator.AddDefault("log_user_events", true)

	// 网络事件记录
	validator.AddFieldType("log_network_events", reflect.Bool)
	validator.AddDefault("log_network_events", true)

	// 文件事件记录
	validator.AddFieldType("log_file_events", reflect.Bool)
	validator.AddDefault("log_file_events", true)

	// 日志保留天数
	validator.AddFieldType("log_retention_days", reflect.Float64)
	validator.AddFieldValidator("log_retention_days", IntRangeValidator(1, 365))
	validator.AddDefault("log_retention_days", 30)

	// 日志级别
	validator.AddFieldType("log_level", reflect.String)
	validator.AddFieldValidator("log_level", StringEnumValidator("debug", "info", "warn", "error"))
	validator.AddDefault("log_level", "info")

	// 实时警报
	validator.AddFieldType("enable_alerts", reflect.Bool)
	validator.AddDefault("enable_alerts", false)

	// 警报接收者
	validator.AddFieldType("alert_recipients", reflect.Slice)
	validator.AddFieldValidator("alert_recipients", ArrayValidator(0, 10))

	// 存储配置
	validator.AddFieldType("storage", reflect.Map)

	return validator
}

// CreateDeviceValidator 创建设备管理插件配置验证器
func CreateDeviceValidator() *PluginConfigValidator {
	validator := NewPluginConfigValidator("device")

	// 基础字段
	validator.AddRequiredField("enabled")
	validator.AddFieldType("enabled", reflect.Bool)
	validator.AddDefault("enabled", true)

	// USB监控
	validator.AddFieldType("monitor_usb", reflect.Bool)
	validator.AddDefault("monitor_usb", true)

	// 网络监控
	validator.AddFieldType("monitor_network", reflect.Bool)
	validator.AddDefault("monitor_network", true)

	// 允许禁用网络接口
	validator.AddFieldType("allow_network_disable", reflect.Bool)
	validator.AddDefault("allow_network_disable", true)

	// 设备缓存过期时间
	validator.AddFieldType("device_cache_expiration", reflect.Float64)
	validator.AddFieldValidator("device_cache_expiration", IntRangeValidator(10, 300))
	validator.AddDefault("device_cache_expiration", 30)

	// 监控间隔
	validator.AddFieldType("monitor_interval", reflect.Float64)
	validator.AddFieldValidator("monitor_interval", IntRangeValidator(10, 3600))
	validator.AddDefault("monitor_interval", 60)

	// 日志级别
	validator.AddFieldType("log_level", reflect.String)
	validator.AddFieldValidator("log_level", StringEnumValidator("debug", "info", "warn", "error"))
	validator.AddDefault("log_level", "info")

	// 受保护的网络接口
	validator.AddFieldType("protected_interfaces", reflect.Slice)
	validator.AddFieldValidator("protected_interfaces", ArrayValidator(0, 20))

	return validator
}

// CreateControlValidator 创建终端管控插件配置验证器
func CreateControlValidator() *PluginConfigValidator {
	validator := NewPluginConfigValidator("control")

	// 基础字段
	validator.AddRequiredField("enabled")
	validator.AddFieldType("enabled", reflect.Bool)
	validator.AddDefault("enabled", true)

	// 日志级别
	validator.AddFieldType("log_level", reflect.String)
	validator.AddFieldValidator("log_level", StringEnumValidator("debug", "info", "warn", "error"))
	validator.AddDefault("log_level", "info")

	// 自动启动
	validator.AddFieldType("auto_start", reflect.Bool)
	validator.AddDefault("auto_start", true)

	// 自动重启
	validator.AddFieldType("auto_restart", reflect.Bool)
	validator.AddDefault("auto_restart", true)

	// 隔离配置
	validator.AddFieldType("isolation", reflect.Map)

	// 插件特定配置
	validator.AddFieldType("settings", reflect.Map)

	return validator
}

// CreateDLPValidator 创建DLP插件配置验证器
func CreateDLPValidator() *PluginConfigValidator {
	validator := NewPluginConfigValidator("dlp")

	// 基础字段
	validator.AddRequiredField("enabled")
	validator.AddFieldType("enabled", reflect.Bool)
	validator.AddDefault("enabled", true)

	// 插件信息
	validator.AddFieldType("name", reflect.String)
	validator.AddDefault("name", "dlp")

	validator.AddFieldType("version", reflect.String)
	validator.AddDefault("version", "2.0.0")

	// 监控开关
	validator.AddFieldType("monitor_network", reflect.Bool)
	validator.AddDefault("monitor_network", true)

	validator.AddFieldType("monitor_files", reflect.Bool)
	validator.AddDefault("monitor_files", true)

	validator.AddFieldType("monitor_clipboard", reflect.Bool)
	validator.AddDefault("monitor_clipboard", true)

	// 性能配置
	validator.AddFieldType("max_concurrency", reflect.Float64)
	validator.AddFieldValidator("max_concurrency", IntRangeValidator(1, 16))
	validator.AddDefault("max_concurrency", 4)

	validator.AddFieldType("buffer_size", reflect.Float64)
	validator.AddFieldValidator("buffer_size", IntRangeValidator(100, 2000))
	validator.AddDefault("buffer_size", 500)

	// 网络协议
	validator.AddFieldType("network_protocols", reflect.Slice)
	validator.AddFieldValidator("network_protocols", ArrayValidator(1, 20))

	// 拦截器配置
	validator.AddFieldType("interceptor_config", reflect.Map)

	// 解析器配置
	validator.AddFieldType("parser_config", reflect.Map)

	// 分析器配置
	validator.AddFieldType("analyzer_config", reflect.Map)

	// 策略引擎配置
	validator.AddFieldType("engine_config", reflect.Map)

	// 执行器配置
	validator.AddFieldType("executor_config", reflect.Map)

	// 监控目录
	validator.AddFieldType("monitored_directories", reflect.Slice)
	validator.AddFieldValidator("monitored_directories", ArrayValidator(0, 50))

	// 监控文件类型
	validator.AddFieldType("monitored_file_types", reflect.Slice)
	validator.AddFieldValidator("monitored_file_types", ArrayValidator(0, 100))

	// 日志配置
	validator.AddFieldType("logging", reflect.Map)

	// 性能监控
	validator.AddFieldType("performance", reflect.Map)

	// 自适应配置
	validator.AddFieldType("adaptive", reflect.Map)

	// 流量限制
	validator.AddFieldType("traffic_limit", reflect.Map)

	// 优先级配置
	validator.AddFieldType("priority", reflect.Map)

	// OCR配置
	validator.AddFieldType("ocr", reflect.Map)

	// 机器学习配置
	validator.AddFieldType("ml", reflect.Map)

	// 文件检测配置
	validator.AddFieldType("file_detection", reflect.Map)

	// 规则配置
	validator.AddFieldType("rules", reflect.Map)

	// 告警配置
	validator.AddFieldType("alerts", reflect.Map)

	// 审计配置
	validator.AddFieldType("audit", reflect.Map)

	return validator
}

// GetAllPluginValidators 获取所有插件验证器
func GetAllPluginValidators() map[string]*PluginConfigValidator {
	return map[string]*PluginConfigValidator{
		"assets":  CreateAssetsValidator(),
		"audit":   CreateAuditValidator(),
		"device":  CreateDeviceValidator(),
		"control": CreateControlValidator(),
		"dlp":     CreateDLPValidator(),
	}
}
