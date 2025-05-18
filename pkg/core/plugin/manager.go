package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// PluginManager 插件管理器
type PluginManager struct {
	plugins             map[string]*PluginInstance
	logger              hclog.Logger
	pluginsDir          string
	ctx                 context.Context
	cancel              context.CancelFunc
	mu                  sync.RWMutex
	healthCheckInterval time.Duration
	eventBus            EventBus
}

// PluginInstance 插件实例
type PluginInstance struct {
	// Metadata 插件元数据
	Metadata PluginMetadata

	// Instance 插件实例
	Instance Module

	// State 插件状态
	State PluginState

	// StartTime 启动时间
	StartTime time.Time

	// StopTime 停止时间
	StopTime time.Time

	// LastError 最后一次错误
	LastError error

	// Process 插件进程（如果是独立进程）
	Process *PluginProcess
}

// PluginProcess 插件进程
type PluginProcess struct {
	// PID 进程ID
	PID int

	// Cmd 命令
	Cmd interface{}

	// Client 客户端
	Client interface{}

	// Conn 连接
	Conn interface{}
}

// PluginManagerOption 插件管理器选项
type PluginManagerOption func(*PluginManager)

// WithPluginLogger 设置日志记录器
func WithPluginLogger(logger hclog.Logger) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.logger = logger
	}
}

// WithPluginsDir 设置插件目录
func WithPluginsDir(dir string) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.pluginsDir = dir
	}
}

// WithPluginContext 设置上下文
func WithPluginContext(ctx context.Context) PluginManagerOption {
	return func(pm *PluginManager) {
		if pm.cancel != nil {
			pm.cancel()
		}
		pm.ctx, pm.cancel = context.WithCancel(ctx)
	}
}

// WithHealthCheckInterval 设置健康检查间隔
func WithHealthCheckInterval(interval time.Duration) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.healthCheckInterval = interval
	}
}

// WithEventBus 设置事件总线
func WithEventBus(eventBus EventBus) PluginManagerOption {
	return func(pm *PluginManager) {
		pm.eventBus = eventBus
	}
}

// NewPluginManager 创建插件管理器
func NewPluginManager(options ...PluginManagerOption) *PluginManager {
	ctx, cancel := context.WithCancel(context.Background())

	pm := &PluginManager{
		plugins:             make(map[string]*PluginInstance),
		logger:              hclog.NewNullLogger(),
		pluginsDir:          "plugins",
		ctx:                 ctx,
		cancel:              cancel,
		healthCheckInterval: 30 * time.Second,
		eventBus:            NewDefaultEventBus(),
	}

	// 应用选项
	for _, option := range options {
		option(pm)
	}

	return pm
}

// SetPluginsDir 设置插件目录
func (pm *PluginManager) SetPluginsDir(dir string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.pluginsDir = dir
}

// ScanPluginsDir 扫描插件目录
func (pm *PluginManager) ScanPluginsDir() ([]PluginMetadata, error) {
	pm.logger.Info("扫描插件目录", "dir", pm.pluginsDir)

	// 确保插件目录存在
	if err := os.MkdirAll(pm.pluginsDir, 0755); err != nil {
		return nil, fmt.Errorf("创建插件目录失败: %w", err)
	}

	// 读取插件目录
	entries, err := os.ReadDir(pm.pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("读取插件目录失败: %w", err)
	}

	var metadataList []PluginMetadata

	// 遍历目录项
	for _, entry := range entries {
		// 跳过非目录
		if !entry.IsDir() {
			continue
		}

		// 构建插件目录路径
		pluginDir := filepath.Join(pm.pluginsDir, entry.Name())

		// 检查插件元数据文件
		metadataPath := filepath.Join(pluginDir, "plugin.json")
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			pm.logger.Warn("插件元数据文件不存在", "dir", pluginDir)
			continue
		}

		// 加载插件元数据
		metadata, err := pm.loadPluginMetadata(metadataPath)
		if err != nil {
			pm.logger.Error("加载插件元数据失败", "dir", pluginDir, "error", err)
			continue
		}

		// 设置插件路径
		metadata.Path = pluginDir

		// 添加到列表
		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

// loadPluginMetadata 加载插件元数据
func (pm *PluginManager) loadPluginMetadata(path string) (PluginMetadata, error) {
	// 读取元数据文件
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("读取元数据文件失败: %w", err)
	}

	// 解析元数据
	var metadata PluginMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return PluginMetadata{}, fmt.Errorf("解析元数据失败: %w", err)
	}

	// 验证元数据
	if metadata.ID == "" {
		return PluginMetadata{}, fmt.Errorf("插件ID不能为空")
	}
	if metadata.Name == "" {
		return PluginMetadata{}, fmt.Errorf("插件名称不能为空")
	}
	if metadata.Version == "" {
		return PluginMetadata{}, fmt.Errorf("插件版本不能为空")
	}
	if metadata.EntryPoint.Type == "" {
		return PluginMetadata{}, fmt.Errorf("插件入口点类型不能为空")
	}
	if metadata.EntryPoint.Path == "" {
		return PluginMetadata{}, fmt.Errorf("插件入口点路径不能为空")
	}

	return metadata, nil
}

// LoadPlugin 加载插件
func (pm *PluginManager) LoadPlugin(metadata PluginMetadata) (*PluginInstance, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.logger.Info("加载插件", "id", metadata.ID, "name", metadata.Name, "version", metadata.Version)

	// 检查插件是否已加载
	if _, exists := pm.plugins[metadata.ID]; exists {
		return nil, fmt.Errorf("插件已加载: %s", metadata.ID)
	}

	// 创建插件实例
	instance := &PluginInstance{
		Metadata:  metadata,
		State:     PluginStateInitializing,
		StartTime: time.Now(),
	}

	// 根据插件类型加载
	switch metadata.EntryPoint.Type {
	case "go":
		// 加载Go插件
		module, err := pm.loadGoPlugin(metadata)
		if err != nil {
			return nil, err
		}
		instance.Instance = module

	case "python":
		// 加载Python插件
		module, process, err := pm.loadPythonPlugin(metadata)
		if err != nil {
			return nil, err
		}
		instance.Instance = module
		instance.Process = process

	default:
		return nil, fmt.Errorf("不支持的插件类型: %s", metadata.EntryPoint.Type)
	}

	// 存储插件实例
	pm.plugins[metadata.ID] = instance

	pm.logger.Info("插件已加载", "id", metadata.ID)
	return instance, nil
}

// loadGoPlugin 加载Go插件
func (pm *PluginManager) loadGoPlugin(metadata PluginMetadata) (Module, error) {
	// 这里实现Go插件加载逻辑
	// 在实际实现中，这里应该使用go-plugin库加载插件
	return nil, fmt.Errorf("Go插件加载尚未实现")
}

// loadPythonPlugin 加载Python插件
func (pm *PluginManager) loadPythonPlugin(metadata PluginMetadata) (Module, *PluginProcess, error) {
	// 这里实现Python插件加载逻辑
	// 在实际实现中，这里应该启动Python解释器并建立gRPC连接
	return nil, nil, fmt.Errorf("Python插件加载尚未实现")
}

// StartPlugin 启动插件
func (pm *PluginManager) StartPlugin(id string) error {
	pm.mu.Lock()
	plugin, exists := pm.plugins[id]
	if !exists {
		pm.mu.Unlock()
		return fmt.Errorf("插件不存在: %s", id)
	}

	// 检查插件状态
	if plugin.State == PluginStateRunning {
		pm.mu.Unlock()
		return fmt.Errorf("插件已在运行: %s", id)
	}

	// 更新插件状态
	plugin.State = PluginStateInitializing
	plugin.StartTime = time.Now()
	pm.mu.Unlock()

	// 启动插件
	if err := plugin.Instance.Start(); err != nil {
		plugin.State = PluginStateError
		plugin.LastError = err
		return fmt.Errorf("启动插件失败: %w", err)
	}

	// 更新状态
	plugin.State = PluginStateRunning

	// 发布插件启动事件
	pm.publishPluginEvent(id, "plugin.started")

	pm.logger.Info("插件已启动", "id", id)
	return nil
}

// StopPlugin 停止插件
func (pm *PluginManager) StopPlugin(id string) error {
	pm.mu.Lock()
	plugin, exists := pm.plugins[id]
	if !exists {
		pm.mu.Unlock()
		return fmt.Errorf("插件不存在: %s", id)
	}

	// 检查插件状态
	if plugin.State != PluginStateRunning && plugin.State != PluginStatePaused {
		pm.mu.Unlock()
		return fmt.Errorf("插件未在运行: %s", id)
	}

	// 更新插件状态
	oldState := plugin.State
	plugin.State = PluginStateStopped
	plugin.StopTime = time.Now()
	pm.mu.Unlock()

	// 停止插件
	if err := plugin.Instance.Stop(); err != nil {
		plugin.State = oldState
		plugin.LastError = err
		return fmt.Errorf("停止插件失败: %w", err)
	}

	// 发布插件停止事件
	pm.publishPluginEvent(id, "plugin.stopped")

	pm.logger.Info("插件已停止", "id", id)
	return nil
}

// UnloadPlugin 卸载插件
func (pm *PluginManager) UnloadPlugin(id string) error {
	pm.mu.Lock()
	plugin, exists := pm.plugins[id]
	if !exists {
		pm.mu.Unlock()
		return fmt.Errorf("插件不存在: %s", id)
	}

	// 如果插件正在运行，先停止它
	if plugin.State == PluginStateRunning || plugin.State == PluginStatePaused {
		pm.mu.Unlock()
		if err := pm.StopPlugin(id); err != nil {
			return fmt.Errorf("停止插件失败: %w", err)
		}
		pm.mu.Lock()
	}

	// 删除插件
	delete(pm.plugins, id)
	pm.mu.Unlock()

	// 发布插件卸载事件
	pm.publishPluginEvent(id, "plugin.unloaded")

	pm.logger.Info("插件已卸载", "id", id)
	return nil
}

// GetPlugin 获取插件
func (pm *PluginManager) GetPlugin(id string) (*PluginInstance, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	plugin, exists := pm.plugins[id]
	return plugin, exists
}

// ListPlugins 列出所有插件
func (pm *PluginManager) ListPlugins() []*PluginInstance {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	plugins := make([]*PluginInstance, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// publishPluginEvent 发布插件事件
func (pm *PluginManager) publishPluginEvent(id string, eventType string) {
	if pm.eventBus == nil {
		return
	}

	event := &Event{
		ID:        fmt.Sprintf("%s-%d", eventType, time.Now().UnixNano()),
		Type:      eventType,
		Source:    "plugin-manager",
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Data: map[string]interface{}{
			"plugin_id": id,
		},
		Metadata: map[string]string{
			"component": "plugin-manager",
		},
	}

	pm.eventBus.Publish(event)
}

// StartHealthCheck 启动健康检查
func (pm *PluginManager) StartHealthCheck() {
	go func() {
		ticker := time.NewTicker(pm.healthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pm.checkPluginsHealth()
			case <-pm.ctx.Done():
				return
			}
		}
	}()
}

// checkPluginsHealth 检查插件健康状态
func (pm *PluginManager) checkPluginsHealth() {
	pm.mu.RLock()
	plugins := make([]*PluginInstance, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}
	pm.mu.RUnlock()

	for _, plugin := range plugins {
		if plugin.State != PluginStateRunning {
			continue
		}

		// 检查插件是否实现了健康检查接口
		if healthCheck, ok := plugin.Instance.(HealthCheck); ok {
			status := healthCheck.CheckHealth()
			if status.Status != "healthy" {
				pm.logger.Warn("插件不健康", "id", plugin.Metadata.ID, "status", status.Status)
				pm.publishPluginEvent(plugin.Metadata.ID, "plugin.unhealthy")
			}
		}
	}
}

// Stop 停止插件管理器
func (pm *PluginManager) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 停止所有插件
	for id, plugin := range pm.plugins {
		if plugin.State == PluginStateRunning || plugin.State == PluginStatePaused {
			pm.logger.Info("停止插件", "id", id)
			if err := plugin.Instance.Stop(); err != nil {
				pm.logger.Error("停止插件失败", "id", id, "error", err)
			}
			plugin.State = PluginStateStopped
			plugin.StopTime = time.Now()
		}
	}

	// 取消上下文
	pm.cancel()

	pm.logger.Info("插件管理器已停止")
}
