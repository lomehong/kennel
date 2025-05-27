package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/errors"
)

// ConfigErrorType 配置错误类型
type ConfigErrorType string

const (
	ConfigErrorTypeFileNotFound    ConfigErrorType = "file_not_found"
	ConfigErrorTypeParseError      ConfigErrorType = "parse_error"
	ConfigErrorTypeValidationError ConfigErrorType = "validation_error"
	ConfigErrorTypePermissionError ConfigErrorType = "permission_error"
	ConfigErrorTypeFormatError     ConfigErrorType = "format_error"
	ConfigErrorTypeConflictError   ConfigErrorType = "conflict_error"
	ConfigErrorTypeHotReloadError  ConfigErrorType = "hot_reload_error"
)

// ConfigError 配置错误
type ConfigError struct {
	Type        ConfigErrorType
	Component   string // 组件名称（主程序、插件ID等）
	ConfigPath  string // 配置文件路径
	Field       string // 配置字段
	Message     string // 错误消息
	Cause       error  // 原始错误
	Suggestions []string // 修复建议
}

// Error 实现error接口
func (e *ConfigError) Error() string {
	var parts []string
	
	if e.Component != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Component))
	}
	
	if e.ConfigPath != "" {
		parts = append(parts, fmt.Sprintf("配置文件: %s", e.ConfigPath))
	}
	
	if e.Field != "" {
		parts = append(parts, fmt.Sprintf("字段: %s", e.Field))
	}
	
	parts = append(parts, e.Message)
	
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("原因: %v", e.Cause))
	}
	
	return strings.Join(parts, " - ")
}

// Unwrap 返回原始错误
func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// NewConfigError 创建配置错误
func NewConfigError(errorType ConfigErrorType, component, configPath, field, message string, cause error) *ConfigError {
	return &ConfigError{
		Type:       errorType,
		Component:  component,
		ConfigPath: configPath,
		Field:      field,
		Message:    message,
		Cause:      cause,
	}
}

// ConfigErrorHandler 配置错误处理器
type ConfigErrorHandler struct {
	logger    hclog.Logger
	component string
	exitOnCritical bool
}

// NewConfigErrorHandler 创建配置错误处理器
func NewConfigErrorHandler(logger hclog.Logger, component string) *ConfigErrorHandler {
	return &ConfigErrorHandler{
		logger:         logger.Named("config-error-handler"),
		component:      component,
		exitOnCritical: true,
	}
}

// SetExitOnCritical 设置是否在关键错误时退出
func (h *ConfigErrorHandler) SetExitOnCritical(exit bool) {
	h.exitOnCritical = exit
}

// HandleError 处理配置错误
func (h *ConfigErrorHandler) HandleError(err error) error {
	if err == nil {
		return nil
	}

	// 检查是否是配置错误
	var configErr *ConfigError
	if errors.As(err, &configErr) {
		return h.handleConfigError(configErr)
	}

	// 包装为通用配置错误
	configErr = &ConfigError{
		Type:      ConfigErrorTypeParseError,
		Component: h.component,
		Message:   err.Error(),
		Cause:     err,
	}

	return h.handleConfigError(configErr)
}

// handleConfigError 处理具体的配置错误
func (h *ConfigErrorHandler) handleConfigError(err *ConfigError) error {
	// 添加修复建议
	h.addSuggestions(err)

	// 根据错误类型决定处理方式
	switch err.Type {
	case ConfigErrorTypeFileNotFound:
		return h.handleFileNotFoundError(err)
	case ConfigErrorTypeParseError:
		return h.handleParseError(err)
	case ConfigErrorTypeValidationError:
		return h.handleValidationError(err)
	case ConfigErrorTypePermissionError:
		return h.handlePermissionError(err)
	case ConfigErrorTypeFormatError:
		return h.handleFormatError(err)
	case ConfigErrorTypeConflictError:
		return h.handleConflictError(err)
	case ConfigErrorTypeHotReloadError:
		return h.handleHotReloadError(err)
	default:
		return h.handleGenericError(err)
	}
}

// handleFileNotFoundError 处理文件未找到错误
func (h *ConfigErrorHandler) handleFileNotFoundError(err *ConfigError) error {
	h.logger.Warn("配置文件未找到",
		"component", err.Component,
		"path", err.ConfigPath,
		"message", err.Message,
	)

	// 输出修复建议
	h.outputSuggestions(err)

	// 文件未找到通常不是致命错误，使用默认配置
	return nil
}

// handleParseError 处理解析错误
func (h *ConfigErrorHandler) handleParseError(err *ConfigError) error {
	h.logger.Error("配置文件解析失败",
		"component", err.Component,
		"path", err.ConfigPath,
		"field", err.Field,
		"message", err.Message,
		"cause", err.Cause,
	)

	// 输出修复建议
	h.outputSuggestions(err)

	// 解析错误是致命错误
	if h.exitOnCritical {
		h.logger.Error("配置解析失败，程序退出")
		os.Exit(1)
	}

	return err
}

// handleValidationError 处理验证错误
func (h *ConfigErrorHandler) handleValidationError(err *ConfigError) error {
	h.logger.Error("配置验证失败",
		"component", err.Component,
		"path", err.ConfigPath,
		"field", err.Field,
		"message", err.Message,
	)

	// 输出修复建议
	h.outputSuggestions(err)

	// 验证错误是致命错误
	if h.exitOnCritical {
		h.logger.Error("配置验证失败，程序退出")
		os.Exit(1)
	}

	return err
}

// handlePermissionError 处理权限错误
func (h *ConfigErrorHandler) handlePermissionError(err *ConfigError) error {
	h.logger.Error("配置文件权限错误",
		"component", err.Component,
		"path", err.ConfigPath,
		"message", err.Message,
		"cause", err.Cause,
	)

	// 输出修复建议
	h.outputSuggestions(err)

	// 权限错误是致命错误
	if h.exitOnCritical {
		h.logger.Error("配置文件权限错误，程序退出")
		os.Exit(1)
	}

	return err
}

// handleFormatError 处理格式错误
func (h *ConfigErrorHandler) handleFormatError(err *ConfigError) error {
	h.logger.Error("配置文件格式错误",
		"component", err.Component,
		"path", err.ConfigPath,
		"field", err.Field,
		"message", err.Message,
	)

	// 输出修复建议
	h.outputSuggestions(err)

	return err
}

// handleConflictError 处理冲突错误
func (h *ConfigErrorHandler) handleConflictError(err *ConfigError) error {
	h.logger.Warn("配置冲突",
		"component", err.Component,
		"field", err.Field,
		"message", err.Message,
	)

	// 输出修复建议
	h.outputSuggestions(err)

	// 冲突错误通常可以自动解决
	return nil
}

// handleHotReloadError 处理热更新错误
func (h *ConfigErrorHandler) handleHotReloadError(err *ConfigError) error {
	h.logger.Warn("配置热更新失败",
		"component", err.Component,
		"path", err.ConfigPath,
		"message", err.Message,
		"cause", err.Cause,
	)

	// 输出修复建议
	h.outputSuggestions(err)

	// 热更新失败不是致命错误
	return nil
}

// handleGenericError 处理通用错误
func (h *ConfigErrorHandler) handleGenericError(err *ConfigError) error {
	h.logger.Error("配置错误",
		"component", err.Component,
		"type", string(err.Type),
		"path", err.ConfigPath,
		"field", err.Field,
		"message", err.Message,
		"cause", err.Cause,
	)

	// 输出修复建议
	h.outputSuggestions(err)

	return err
}

// addSuggestions 添加修复建议
func (h *ConfigErrorHandler) addSuggestions(err *ConfigError) {
	switch err.Type {
	case ConfigErrorTypeFileNotFound:
		err.Suggestions = []string{
			"检查配置文件路径是否正确",
			"确认配置文件是否存在",
			"使用配置迁移工具创建配置文件",
			"检查文件权限",
		}
	case ConfigErrorTypeParseError:
		err.Suggestions = []string{
			"检查YAML/JSON语法是否正确",
			"使用在线YAML/JSON验证器检查格式",
			"检查文件编码是否为UTF-8",
			"确认文件没有被截断",
		}
	case ConfigErrorTypeValidationError:
		err.Suggestions = []string{
			"检查配置值是否在允许范围内",
			"确认配置类型是否正确",
			"参考配置文档和示例",
			"使用配置验证工具检查",
		}
	case ConfigErrorTypePermissionError:
		err.Suggestions = []string{
			"检查文件读写权限",
			"确认当前用户有访问权限",
			"使用sudo或管理员权限运行",
			"检查SELinux或其他安全策略",
		}
	case ConfigErrorTypeFormatError:
		err.Suggestions = []string{
			"使用推荐的配置文件格式",
			"运行配置迁移工具",
			"参考最新的配置文档",
			"检查配置文件版本兼容性",
		}
	case ConfigErrorTypeConflictError:
		err.Suggestions = []string{
			"检查配置优先级规则",
			"解决配置键名冲突",
			"使用环境变量覆盖冲突配置",
			"参考配置优先级文档",
		}
	case ConfigErrorTypeHotReloadError:
		err.Suggestions = []string{
			"检查配置文件是否被锁定",
			"确认配置变更是否有效",
			"重启应用程序",
			"检查热更新支持范围",
		}
	}
}

// outputSuggestions 输出修复建议
func (h *ConfigErrorHandler) outputSuggestions(err *ConfigError) {
	if len(err.Suggestions) == 0 {
		return
	}

	h.logger.Info("修复建议:")
	for i, suggestion := range err.Suggestions {
		h.logger.Info(fmt.Sprintf("  %d. %s", i+1, suggestion))
	}
}

// StandardConfigErrorHandler 标准配置错误处理器
var StandardConfigErrorHandler *ConfigErrorHandler

// InitStandardConfigErrorHandler 初始化标准配置错误处理器
func InitStandardConfigErrorHandler(logger hclog.Logger) {
	StandardConfigErrorHandler = NewConfigErrorHandler(logger, "main")
}

// HandleConfigError 处理配置错误（便捷函数）
func HandleConfigError(err error) error {
	if StandardConfigErrorHandler == nil {
		// 如果没有初始化标准处理器，创建一个临时的
		handler := NewConfigErrorHandler(hclog.NewNullLogger(), "unknown")
		return handler.HandleError(err)
	}
	return StandardConfigErrorHandler.HandleError(err)
}
