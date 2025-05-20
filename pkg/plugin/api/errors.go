package api

import (
	"fmt"
)

// ErrorType 定义了错误类型
type ErrorType string

// 预定义的错误类型
const (
	ErrorTypePlugin      ErrorType = "plugin"       // 插件错误
	ErrorTypeInit        ErrorType = "init"         // 初始化错误
	ErrorTypeStart       ErrorType = "start"        // 启动错误
	ErrorTypeStop        ErrorType = "stop"         // 停止错误
	ErrorTypeConfig      ErrorType = "config"       // 配置错误
	ErrorTypeDependency  ErrorType = "dependency"   // 依赖错误
	ErrorTypeIsolation   ErrorType = "isolation"    // 隔离错误
	ErrorTypeTimeout     ErrorType = "timeout"      // 超时错误
	ErrorTypeResource    ErrorType = "resource"     // 资源错误
	ErrorTypePermission  ErrorType = "permission"   // 权限错误
	ErrorTypeValidation  ErrorType = "validation"   // 验证错误
	ErrorTypeInternal    ErrorType = "internal"     // 内部错误
	ErrorTypeExternal    ErrorType = "external"     // 外部错误
	ErrorTypeUnknown     ErrorType = "unknown"      // 未知错误
)

// ErrorSeverity 定义了错误严重程度
type ErrorSeverity string

// 预定义的错误严重程度
const (
	ErrorSeverityInfo     ErrorSeverity = "info"     // 信息
	ErrorSeverityWarning  ErrorSeverity = "warning"  // 警告
	ErrorSeverityError    ErrorSeverity = "error"    // 错误
	ErrorSeverityCritical ErrorSeverity = "critical" // 严重
	ErrorSeverityFatal    ErrorSeverity = "fatal"    // 致命
)

// PluginError 定义了插件错误
type PluginError struct {
	Type        ErrorType       // 错误类型
	Severity    ErrorSeverity   // 错误严重程度
	Code        string          // 错误代码
	Message     string          // 错误消息
	PluginID    string          // 插件ID
	Details     map[string]interface{} // 错误详情
	Cause       error           // 原因
}

// Error 实现error接口
func (e *PluginError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (plugin: %s, code: %s) - caused by: %v", 
			e.Type, e.Severity, e.Message, e.PluginID, e.Code, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s (plugin: %s, code: %s)", 
		e.Type, e.Severity, e.Message, e.PluginID, e.Code)
}

// Unwrap 实现errors.Unwrap接口
func (e *PluginError) Unwrap() error {
	return e.Cause
}

// NewPluginError 创建一个新的插件错误
func NewPluginError(pluginID string, errType ErrorType, severity ErrorSeverity, code string, message string, cause error) *PluginError {
	return &PluginError{
		Type:     errType,
		Severity: severity,
		Code:     code,
		Message:  message,
		PluginID: pluginID,
		Details:  make(map[string]interface{}),
		Cause:    cause,
	}
}

// WithDetails 添加错误详情
func (e *PluginError) WithDetails(details map[string]interface{}) *PluginError {
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// IsPluginError 检查错误是否为插件错误
func IsPluginError(err error) bool {
	_, ok := err.(*PluginError)
	return ok
}

// GetPluginError 获取插件错误
func GetPluginError(err error) (*PluginError, bool) {
	if err == nil {
		return nil, false
	}
	
	pe, ok := err.(*PluginError)
	return pe, ok
}

// IsErrorType 检查错误是否为指定类型
func IsErrorType(err error, errType ErrorType) bool {
	pe, ok := err.(*PluginError)
	if !ok {
		return false
	}
	return pe.Type == errType
}

// IsErrorSeverity 检查错误是否为指定严重程度
func IsErrorSeverity(err error, severity ErrorSeverity) bool {
	pe, ok := err.(*PluginError)
	if !ok {
		return false
	}
	return pe.Severity == severity
}

// NewInitError 创建初始化错误
func NewInitError(pluginID string, message string, cause error) *PluginError {
	return NewPluginError(pluginID, ErrorTypeInit, ErrorSeverityError, "INIT_FAILED", message, cause)
}

// NewStartError 创建启动错误
func NewStartError(pluginID string, message string, cause error) *PluginError {
	return NewPluginError(pluginID, ErrorTypeStart, ErrorSeverityError, "START_FAILED", message, cause)
}

// NewStopError 创建停止错误
func NewStopError(pluginID string, message string, cause error) *PluginError {
	return NewPluginError(pluginID, ErrorTypeStop, ErrorSeverityError, "STOP_FAILED", message, cause)
}

// NewConfigError 创建配置错误
func NewConfigError(pluginID string, message string, cause error) *PluginError {
	return NewPluginError(pluginID, ErrorTypeConfig, ErrorSeverityError, "CONFIG_ERROR", message, cause)
}

// NewDependencyError 创建依赖错误
func NewDependencyError(pluginID string, message string, cause error) *PluginError {
	return NewPluginError(pluginID, ErrorTypeDependency, ErrorSeverityError, "DEPENDENCY_ERROR", message, cause)
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(pluginID string, message string, cause error) *PluginError {
	return NewPluginError(pluginID, ErrorTypeTimeout, ErrorSeverityWarning, "TIMEOUT", message, cause)
}

// NewResourceError 创建资源错误
func NewResourceError(pluginID string, message string, cause error) *PluginError {
	return NewPluginError(pluginID, ErrorTypeResource, ErrorSeverityError, "RESOURCE_ERROR", message, cause)
}

// NewValidationError 创建验证错误
func NewValidationError(pluginID string, message string, cause error) *PluginError {
	return NewPluginError(pluginID, ErrorTypeValidation, ErrorSeverityWarning, "VALIDATION_ERROR", message, cause)
}
