package config

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/errors"
)

// PluginConfigErrorHandler 插件配置错误处理器
type PluginConfigErrorHandler struct {
	*ConfigErrorHandler
	pluginID string
}

// NewPluginConfigErrorHandler 创建插件配置错误处理器
func NewPluginConfigErrorHandler(logger hclog.Logger, pluginID string) *PluginConfigErrorHandler {
	baseHandler := NewConfigErrorHandler(logger, pluginID)
	// 插件配置错误不应该导致整个程序退出
	baseHandler.SetExitOnCritical(false)

	return &PluginConfigErrorHandler{
		ConfigErrorHandler: baseHandler,
		pluginID:           pluginID,
	}
}

// HandlePluginConfigError 处理插件配置错误
func (h *PluginConfigErrorHandler) HandlePluginConfigError(err error, configPath string) error {
	if err == nil {
		return nil
	}

	// 包装为插件配置错误
	var configErr *ConfigError
	if !errors.As(err, &configErr) {
		configErr = &ConfigError{
			Type:       ConfigErrorTypeValidationError,
			Component:  h.pluginID,
			ConfigPath: configPath,
			Message:    err.Error(),
			Cause:      err,
		}
	} else {
		// 确保组件名称正确
		configErr.Component = h.pluginID
		if configErr.ConfigPath == "" {
			configErr.ConfigPath = configPath
		}
	}

	return h.HandleError(configErr)
}

// HandlePluginInitError 处理插件初始化错误
func (h *PluginConfigErrorHandler) HandlePluginInitError(err error) error {
	if err == nil {
		return nil
	}

	h.logger.Error("插件初始化失败",
		"plugin", h.pluginID,
		"error", err,
	)

	// 插件初始化失败不应该导致程序退出，而是禁用该插件
	h.logger.Warn("插件将被禁用", "plugin", h.pluginID)

	return fmt.Errorf("插件 %s 初始化失败: %w", h.pluginID, err)
}

// HandlePluginStartError 处理插件启动错误
func (h *PluginConfigErrorHandler) HandlePluginStartError(err error) error {
	if err == nil {
		return nil
	}

	h.logger.Error("插件启动失败",
		"plugin", h.pluginID,
		"error", err,
	)

	// 提供重启建议
	h.logger.Info("插件启动失败修复建议:")
	h.logger.Info("  1. 检查插件配置是否正确")
	h.logger.Info("  2. 确认插件依赖是否满足")
	h.logger.Info("  3. 检查系统资源是否充足")
	h.logger.Info("  4. 查看详细错误日志")

	return fmt.Errorf("插件 %s 启动失败: %w", h.pluginID, err)
}

// HandlePluginStopError 处理插件停止错误
func (h *PluginConfigErrorHandler) HandlePluginStopError(err error) error {
	if err == nil {
		return nil
	}

	h.logger.Warn("插件停止时出现错误",
		"plugin", h.pluginID,
		"error", err,
	)

	// 插件停止错误通常不是致命的
	return nil
}

// StandardPluginErrorHandlers 标准插件错误处理器集合
var StandardPluginErrorHandlers = make(map[string]*PluginConfigErrorHandler)

// GetPluginErrorHandler 获取插件错误处理器
func GetPluginErrorHandler(pluginID string, logger hclog.Logger) *PluginConfigErrorHandler {
	if handler, exists := StandardPluginErrorHandlers[pluginID]; exists {
		return handler
	}

	// 创建新的处理器
	handler := NewPluginConfigErrorHandler(logger, pluginID)
	StandardPluginErrorHandlers[pluginID] = handler
	return handler
}

// ConfigErrorReporter 配置错误报告器
type ConfigErrorReporter struct {
	logger hclog.Logger
	errors []ConfigError
}

// NewConfigErrorReporter 创建配置错误报告器
func NewConfigErrorReporter(logger hclog.Logger) *ConfigErrorReporter {
	return &ConfigErrorReporter{
		logger: logger.Named("config-error-reporter"),
		errors: make([]ConfigError, 0),
	}
}

// ReportError 报告错误
func (r *ConfigErrorReporter) ReportError(err ConfigError) {
	r.errors = append(r.errors, err)

	// 立即记录错误
	r.logger.Error("配置错误报告",
		"type", string(err.Type),
		"component", err.Component,
		"path", err.ConfigPath,
		"field", err.Field,
		"message", err.Message,
	)
}

// GetErrors 获取所有错误
func (r *ConfigErrorReporter) GetErrors() []ConfigError {
	return r.errors
}

// GetErrorsByType 按类型获取错误
func (r *ConfigErrorReporter) GetErrorsByType(errorType ConfigErrorType) []ConfigError {
	var result []ConfigError
	for _, err := range r.errors {
		if err.Type == errorType {
			result = append(result, err)
		}
	}
	return result
}

// GetErrorsByComponent 按组件获取错误
func (r *ConfigErrorReporter) GetErrorsByComponent(component string) []ConfigError {
	var result []ConfigError
	for _, err := range r.errors {
		if err.Component == component {
			result = append(result, err)
		}
	}
	return result
}

// HasCriticalErrors 检查是否有关键错误
func (r *ConfigErrorReporter) HasCriticalErrors() bool {
	for _, err := range r.errors {
		if err.Type == ConfigErrorTypeParseError ||
			err.Type == ConfigErrorTypeValidationError ||
			err.Type == ConfigErrorTypePermissionError {
			return true
		}
	}
	return false
}

// Clear 清空错误列表
func (r *ConfigErrorReporter) Clear() {
	r.errors = r.errors[:0]
}

// GenerateReport 生成错误报告
func (r *ConfigErrorReporter) GenerateReport() string {
	if len(r.errors) == 0 {
		return "没有配置错误"
	}

	report := fmt.Sprintf("配置错误报告 (共 %d 个错误):\n", len(r.errors))
	report += "=" + fmt.Sprintf("%*s", 50, "") + "\n"

	// 按类型分组错误
	errorsByType := make(map[ConfigErrorType][]ConfigError)
	for _, err := range r.errors {
		errorsByType[err.Type] = append(errorsByType[err.Type], err)
	}

	// 输出每种类型的错误
	for errorType, errors := range errorsByType {
		report += fmt.Sprintf("\n%s (%d 个):\n", string(errorType), len(errors))
		for i, err := range errors {
			report += fmt.Sprintf("  %d. [%s] %s\n", i+1, err.Component, err.Message)
			if err.ConfigPath != "" {
				report += fmt.Sprintf("     文件: %s\n", err.ConfigPath)
			}
			if err.Field != "" {
				report += fmt.Sprintf("     字段: %s\n", err.Field)
			}
		}
	}

	return report
}

// StandardConfigErrorReporter 标准配置错误报告器
var StandardConfigErrorReporter *ConfigErrorReporter

// InitStandardConfigErrorReporter 初始化标准配置错误报告器
func InitStandardConfigErrorReporter(logger hclog.Logger) {
	StandardConfigErrorReporter = NewConfigErrorReporter(logger)
}

// ReportConfigError 报告配置错误（便捷函数）
func ReportConfigError(errorType ConfigErrorType, component, configPath, field, message string, cause error) {
	if StandardConfigErrorReporter == nil {
		return
	}

	err := ConfigError{
		Type:       errorType,
		Component:  component,
		ConfigPath: configPath,
		Field:      field,
		Message:    message,
		Cause:      cause,
	}

	StandardConfigErrorReporter.ReportError(err)
}

// StandardizedConfigErrorHandling 标准化配置错误处理
type StandardizedConfigErrorHandling struct {
	mainHandler    *ConfigErrorHandler
	pluginHandlers map[string]*PluginConfigErrorHandler
	reporter       *ConfigErrorReporter
	logger         hclog.Logger
}

// NewStandardizedConfigErrorHandling 创建标准化配置错误处理
func NewStandardizedConfigErrorHandling(logger hclog.Logger) *StandardizedConfigErrorHandling {
	return &StandardizedConfigErrorHandling{
		mainHandler:    NewConfigErrorHandler(logger, "main"),
		pluginHandlers: make(map[string]*PluginConfigErrorHandler),
		reporter:       NewConfigErrorReporter(logger),
		logger:         logger.Named("config-error-handling"),
	}
}

// HandleMainConfigError 处理主程序配置错误
func (s *StandardizedConfigErrorHandling) HandleMainConfigError(err error) error {
	return s.mainHandler.HandleError(err)
}

// HandlePluginConfigError 处理插件配置错误
func (s *StandardizedConfigErrorHandling) HandlePluginConfigError(pluginID string, err error, configPath string) error {
	handler := s.getPluginHandler(pluginID)
	return handler.HandlePluginConfigError(err, configPath)
}

// HandlePluginInitError 处理插件初始化错误
func (s *StandardizedConfigErrorHandling) HandlePluginInitError(pluginID string, err error) error {
	handler := s.getPluginHandler(pluginID)
	return handler.HandlePluginInitError(err)
}

// getPluginHandler 获取插件处理器
func (s *StandardizedConfigErrorHandling) getPluginHandler(pluginID string) *PluginConfigErrorHandler {
	if handler, exists := s.pluginHandlers[pluginID]; exists {
		return handler
	}

	handler := NewPluginConfigErrorHandler(s.logger, pluginID)
	s.pluginHandlers[pluginID] = handler
	return handler
}

// GetErrorReport 获取错误报告
func (s *StandardizedConfigErrorHandling) GetErrorReport() string {
	return s.reporter.GenerateReport()
}

// HasCriticalErrors 检查是否有关键错误
func (s *StandardizedConfigErrorHandling) HasCriticalErrors() bool {
	return s.reporter.HasCriticalErrors()
}

// 全局标准化错误处理实例
var GlobalConfigErrorHandling *StandardizedConfigErrorHandling

// InitGlobalConfigErrorHandling 初始化全局配置错误处理
func InitGlobalConfigErrorHandling(logger hclog.Logger) {
	GlobalConfigErrorHandling = NewStandardizedConfigErrorHandling(logger)

	// 同时初始化其他全局处理器
	InitStandardConfigErrorHandler(logger)
	InitStandardConfigErrorReporter(logger)
}
