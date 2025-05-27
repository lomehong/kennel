package config

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// HotReloadType 热更新类型
type HotReloadType string

const (
	HotReloadTypeGlobal     HotReloadType = "global"
	HotReloadTypePlugin     HotReloadType = "plugin"
	HotReloadTypeWebConsole HotReloadType = "web_console"
	HotReloadTypeComm       HotReloadType = "comm"
	HotReloadTypeLogging    HotReloadType = "logging"
)

// HotReloadSupport 热更新支持级别
type HotReloadSupport string

const (
	HotReloadSupportFull    HotReloadSupport = "full"    // 完全支持
	HotReloadSupportPartial HotReloadSupport = "partial" // 部分支持
	HotReloadSupportNone    HotReloadSupport = "none"    // 不支持
)

// HotReloadConfig 热更新配置
type HotReloadConfig struct {
	Enabled           bool          `yaml:"enabled"`
	DebounceTime      time.Duration `yaml:"debounce_time"`
	MaxRetries        int           `yaml:"max_retries"`
	RetryInterval     time.Duration `yaml:"retry_interval"`
	ValidationTimeout time.Duration `yaml:"validation_timeout"`
	RollbackOnFailure bool          `yaml:"rollback_on_failure"`
	NotifyOnSuccess   bool          `yaml:"notify_on_success"`
	NotifyOnFailure   bool          `yaml:"notify_on_failure"`
}

// DefaultHotReloadConfig 默认热更新配置
func DefaultHotReloadConfig() *HotReloadConfig {
	return &HotReloadConfig{
		Enabled:           true,
		DebounceTime:      2 * time.Second,
		MaxRetries:        3,
		RetryInterval:     5 * time.Second,
		ValidationTimeout: 10 * time.Second,
		RollbackOnFailure: true,
		NotifyOnSuccess:   true,
		NotifyOnFailure:   true,
	}
}

// HotReloadEvent 热更新事件
type HotReloadEvent struct {
	Type       HotReloadType          `json:"type"`
	Component  string                 `json:"component"`
	ConfigPath string                 `json:"config_path"`
	OldConfig  map[string]interface{} `json:"old_config"`
	NewConfig  map[string]interface{} `json:"new_config"`
	Changes    map[string]interface{} `json:"changes"`
	Timestamp  time.Time              `json:"timestamp"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration"`
	Retries    int                    `json:"retries"`
}

// HotReloadHandler 热更新处理器
type HotReloadHandler interface {
	// GetSupportLevel 获取支持级别
	GetSupportLevel() HotReloadSupport

	// CanReload 检查是否可以热更新
	CanReload(oldConfig, newConfig map[string]interface{}) bool

	// Reload 执行热更新
	Reload(ctx context.Context, oldConfig, newConfig map[string]interface{}) error

	// Validate 验证配置
	Validate(config map[string]interface{}) error

	// Rollback 回滚配置
	Rollback(ctx context.Context, config map[string]interface{}) error
}

// HotReloadManager 热更新管理器
type HotReloadManager struct {
	config   *HotReloadConfig
	handlers map[string]HotReloadHandler
	events   []HotReloadEvent
	logger   hclog.Logger
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewHotReloadManager 创建热更新管理器
func NewHotReloadManager(config *HotReloadConfig, logger hclog.Logger) *HotReloadManager {
	if config == nil {
		config = DefaultHotReloadConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &HotReloadManager{
		config:   config,
		handlers: make(map[string]HotReloadHandler),
		events:   make([]HotReloadEvent, 0),
		logger:   logger.Named("hot-reload-manager"),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// RegisterHandler 注册热更新处理器
func (hrm *HotReloadManager) RegisterHandler(component string, handler HotReloadHandler) {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	hrm.handlers[component] = handler
	hrm.logger.Info("注册热更新处理器", "component", component, "support", handler.GetSupportLevel())
}

// UnregisterHandler 注销热更新处理器
func (hrm *HotReloadManager) UnregisterHandler(component string) {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	delete(hrm.handlers, component)
	hrm.logger.Info("注销热更新处理器", "component", component)
}

// Reload 执行热更新
func (hrm *HotReloadManager) Reload(reloadType HotReloadType, component string, configPath string, oldConfig, newConfig map[string]interface{}) error {
	if !hrm.config.Enabled {
		return fmt.Errorf("热更新已禁用")
	}

	startTime := time.Now()
	event := HotReloadEvent{
		Type:       reloadType,
		Component:  component,
		ConfigPath: configPath,
		OldConfig:  copyConfigMap(oldConfig),
		NewConfig:  copyConfigMap(newConfig),
		Changes:    calculateChanges(oldConfig, newConfig),
		Timestamp:  startTime,
	}

	hrm.logger.Info("开始热更新",
		"type", reloadType,
		"component", component,
		"config_path", configPath,
	)

	// 获取处理器
	hrm.mu.RLock()
	handler, exists := hrm.handlers[component]
	hrm.mu.RUnlock()

	if !exists {
		err := fmt.Errorf("未找到组件 %s 的热更新处理器", component)
		event.Success = false
		event.Error = err.Error()
		event.Duration = time.Since(startTime)
		hrm.addEvent(event)
		return err
	}

	// 检查支持级别
	supportLevel := handler.GetSupportLevel()
	if supportLevel == HotReloadSupportNone {
		err := fmt.Errorf("组件 %s 不支持热更新", component)
		event.Success = false
		event.Error = err.Error()
		event.Duration = time.Since(startTime)
		hrm.addEvent(event)
		return err
	}

	// 检查是否可以热更新
	if !handler.CanReload(oldConfig, newConfig) {
		err := fmt.Errorf("组件 %s 当前状态不支持热更新", component)
		event.Success = false
		event.Error = err.Error()
		event.Duration = time.Since(startTime)
		hrm.addEvent(event)
		return err
	}

	// 执行热更新（带重试）
	var lastErr error
	for retry := 0; retry <= hrm.config.MaxRetries; retry++ {
		event.Retries = retry

		// 创建超时上下文
		ctx, cancel := context.WithTimeout(hrm.ctx, hrm.config.ValidationTimeout)

		// 验证新配置
		if err := handler.Validate(newConfig); err != nil {
			cancel()
			lastErr = fmt.Errorf("配置验证失败: %w", err)
			hrm.logger.Warn("配置验证失败", "component", component, "retry", retry, "error", err)

			if retry < hrm.config.MaxRetries {
				time.Sleep(hrm.config.RetryInterval)
				continue
			}
			break
		}

		// 执行热更新
		if err := handler.Reload(ctx, oldConfig, newConfig); err != nil {
			cancel()
			lastErr = fmt.Errorf("热更新执行失败: %w", err)
			hrm.logger.Warn("热更新执行失败", "component", component, "retry", retry, "error", err)

			// 如果启用回滚，尝试回滚
			if hrm.config.RollbackOnFailure && retry == hrm.config.MaxRetries {
				if rollbackErr := handler.Rollback(hrm.ctx, oldConfig); rollbackErr != nil {
					hrm.logger.Error("回滚失败", "component", component, "error", rollbackErr)
				} else {
					hrm.logger.Info("回滚成功", "component", component)
				}
			}

			if retry < hrm.config.MaxRetries {
				time.Sleep(hrm.config.RetryInterval)
				continue
			}
			break
		}

		cancel()
		// 成功
		event.Success = true
		event.Duration = time.Since(startTime)
		hrm.addEvent(event)

		hrm.logger.Info("热更新成功",
			"component", component,
			"retries", retry,
			"duration", event.Duration,
		)

		if hrm.config.NotifyOnSuccess {
			hrm.notifySuccess(event)
		}

		return nil
	}

	// 失败
	event.Success = false
	event.Error = lastErr.Error()
	event.Duration = time.Since(startTime)
	hrm.addEvent(event)

	hrm.logger.Error("热更新失败",
		"component", component,
		"retries", event.Retries,
		"duration", event.Duration,
		"error", lastErr,
	)

	if hrm.config.NotifyOnFailure {
		hrm.notifyFailure(event)
	}

	return lastErr
}

// GetSupportInfo 获取热更新支持信息
func (hrm *HotReloadManager) GetSupportInfo() map[string]HotReloadSupport {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()

	info := make(map[string]HotReloadSupport)
	for component, handler := range hrm.handlers {
		info[component] = handler.GetSupportLevel()
	}
	return info
}

// GetEvents 获取热更新事件
func (hrm *HotReloadManager) GetEvents() []HotReloadEvent {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()

	// 返回副本
	events := make([]HotReloadEvent, len(hrm.events))
	copy(events, hrm.events)
	return events
}

// GetEventsByComponent 按组件获取事件
func (hrm *HotReloadManager) GetEventsByComponent(component string) []HotReloadEvent {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()

	var events []HotReloadEvent
	for _, event := range hrm.events {
		if event.Component == component {
			events = append(events, event)
		}
	}
	return events
}

// GetSuccessRate 获取成功率
func (hrm *HotReloadManager) GetSuccessRate() float64 {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()

	if len(hrm.events) == 0 {
		return 0.0
	}

	successCount := 0
	for _, event := range hrm.events {
		if event.Success {
			successCount++
		}
	}

	return float64(successCount) / float64(len(hrm.events))
}

// addEvent 添加事件
func (hrm *HotReloadManager) addEvent(event HotReloadEvent) {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	hrm.events = append(hrm.events, event)

	// 限制事件数量
	maxEvents := 1000
	if len(hrm.events) > maxEvents {
		hrm.events = hrm.events[len(hrm.events)-maxEvents:]
	}
}

// notifySuccess 通知成功
func (hrm *HotReloadManager) notifySuccess(event HotReloadEvent) {
	// 这里可以集成通知系统
	hrm.logger.Info("热更新成功通知", "component", event.Component)
}

// notifyFailure 通知失败
func (hrm *HotReloadManager) notifyFailure(event HotReloadEvent) {
	// 这里可以集成通知系统
	hrm.logger.Error("热更新失败通知", "component", event.Component, "error", event.Error)
}

// Stop 停止热更新管理器
func (hrm *HotReloadManager) Stop() {
	hrm.cancel()
	hrm.logger.Info("热更新管理器已停止")
}

// calculateChanges 计算配置变更
func calculateChanges(oldConfig, newConfig map[string]interface{}) map[string]interface{} {
	changes := make(map[string]interface{})

	// 检查新增和修改的配置
	for key, newValue := range newConfig {
		if oldValue, exists := oldConfig[key]; !exists {
			changes[key] = map[string]interface{}{
				"type": "added",
				"new":  newValue,
			}
		} else if !deepEqual(oldValue, newValue) {
			changes[key] = map[string]interface{}{
				"type": "modified",
				"old":  oldValue,
				"new":  newValue,
			}
		}
	}

	// 检查删除的配置
	for key, oldValue := range oldConfig {
		if _, exists := newConfig[key]; !exists {
			changes[key] = map[string]interface{}{
				"type": "removed",
				"old":  oldValue,
			}
		}
	}

	return changes
}

// deepEqual 深度比较两个值
func deepEqual(a, b interface{}) bool {
	// 简化实现，实际应该使用更完善的深度比较
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// copyConfigMap 复制配置map
func copyConfigMap(original map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for key, value := range original {
		copy[key] = value
	}
	return copy
}

// LoggingHotReloadHandler 日志配置热更新处理器
type LoggingHotReloadHandler struct {
	logger hclog.Logger
}

// NewLoggingHotReloadHandler 创建日志热更新处理器
func NewLoggingHotReloadHandler(logger hclog.Logger) *LoggingHotReloadHandler {
	return &LoggingHotReloadHandler{
		logger: logger.Named("logging-hot-reload"),
	}
}

// GetSupportLevel 获取支持级别
func (h *LoggingHotReloadHandler) GetSupportLevel() HotReloadSupport {
	return HotReloadSupportFull
}

// CanReload 检查是否可以热更新
func (h *LoggingHotReloadHandler) CanReload(oldConfig, newConfig map[string]interface{}) bool {
	// 日志配置总是可以热更新
	return true
}

// Reload 执行热更新
func (h *LoggingHotReloadHandler) Reload(ctx context.Context, oldConfig, newConfig map[string]interface{}) error {
	h.logger.Info("执行日志配置热更新")

	// 这里应该实际更新日志配置
	// 例如：更新日志级别、输出文件等

	return nil
}

// Validate 验证配置
func (h *LoggingHotReloadHandler) Validate(config map[string]interface{}) error {
	// 验证日志配置
	if level, ok := config["level"].(string); ok {
		validLevels := []string{"trace", "debug", "info", "warn", "error"}
		valid := false
		for _, validLevel := range validLevels {
			if level == validLevel {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("无效的日志级别: %s", level)
		}
	}

	return nil
}

// Rollback 回滚配置
func (h *LoggingHotReloadHandler) Rollback(ctx context.Context, config map[string]interface{}) error {
	h.logger.Info("回滚日志配置")
	return h.Reload(ctx, nil, config)
}

// PluginHotReloadHandler 插件配置热更新处理器
type PluginHotReloadHandler struct {
	pluginID string
	logger   hclog.Logger
}

// NewPluginHotReloadHandler 创建插件热更新处理器
func NewPluginHotReloadHandler(pluginID string, logger hclog.Logger) *PluginHotReloadHandler {
	return &PluginHotReloadHandler{
		pluginID: pluginID,
		logger:   logger.Named("plugin-hot-reload"),
	}
}

// GetSupportLevel 获取支持级别
func (h *PluginHotReloadHandler) GetSupportLevel() HotReloadSupport {
	return HotReloadSupportPartial
}

// CanReload 检查是否可以热更新
func (h *PluginHotReloadHandler) CanReload(oldConfig, newConfig map[string]interface{}) bool {
	// 检查是否只是配置参数变更，而不是启用状态变更
	oldEnabled := getConfigBool(oldConfig, "enabled", true)
	newEnabled := getConfigBool(newConfig, "enabled", true)

	// 如果启用状态发生变化，需要重启插件
	if oldEnabled != newEnabled {
		return false
	}

	return true
}

// Reload 执行热更新
func (h *PluginHotReloadHandler) Reload(ctx context.Context, oldConfig, newConfig map[string]interface{}) error {
	h.logger.Info("执行插件配置热更新", "plugin", h.pluginID)

	// 这里应该通知插件重新加载配置
	// 例如：调用插件的配置更新接口

	return nil
}

// Validate 验证配置
func (h *PluginHotReloadHandler) Validate(config map[string]interface{}) error {
	// 验证插件配置
	// 这里应该根据具体插件的配置规则进行验证

	return nil
}

// Rollback 回滚配置
func (h *PluginHotReloadHandler) Rollback(ctx context.Context, config map[string]interface{}) error {
	h.logger.Info("回滚插件配置", "plugin", h.pluginID)
	return h.Reload(ctx, nil, config)
}

// getConfigBool 获取布尔配置值
func getConfigBool(config map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := config[key].(bool); ok {
		return value
	}
	return defaultValue
}
