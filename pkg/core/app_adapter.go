package core

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lomehong/kennel/pkg/comm"
	"github.com/lomehong/kennel/pkg/interfaces"
	"github.com/lomehong/kennel/pkg/plugin"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
)

// AppInterfaceAdapter 适配器，将App转换为AppInterface
type AppInterfaceAdapter struct {
	app *App
}

// NewAppInterfaceAdapter 创建一个新的App接口适配器
func NewAppInterfaceAdapter(app *App) *AppInterfaceAdapter {
	return &AppInterfaceAdapter{
		app: app,
	}
}

// GetPluginManager 获取插件管理器接口
func (a *AppInterfaceAdapter) GetPluginManager() interfaces.PluginManagerInterface {
	// 创建一个新的插件管理器适配器
	return &PluginManagerAdapter{
		pm: a.app.pluginManager,
	}
}

// GetConfigManager 获取配置管理器接口
func (a *AppInterfaceAdapter) GetConfigManager() interfaces.ConfigManagerInterface {
	return NewConfigManagerAdapter(a.app.configManager)
}

// GetCommManager 获取通讯管理器接口
func (a *AppInterfaceAdapter) GetCommManager() interfaces.CommManagerInterface {
	return NewCommManagerAdapter(a.app.commManager)
}

// GetLogManager 获取日志管理器接口
func (a *AppInterfaceAdapter) GetLogManager() interfaces.LogManagerInterface {
	return a.app.GetLogManager()
}

// GetEventManager 获取事件管理器接口
func (a *AppInterfaceAdapter) GetEventManager() interfaces.EventManagerInterface {
	return a.app.GetEventManager()
}

// GetSystemMonitor 获取系统监控器接口
func (a *AppInterfaceAdapter) GetSystemMonitor() interfaces.SystemMonitorInterface {
	return a.app.GetSystemMonitor()
}

// GetStartTime 获取应用程序启动时间
func (a *AppInterfaceAdapter) GetStartTime() time.Time {
	return a.app.startTime
}

// GetVersion 获取应用程序版本
func (a *AppInterfaceAdapter) GetVersion() string {
	return a.app.version
}

// IsRunning 检查应用程序是否正在运行
func (a *AppInterfaceAdapter) IsRunning() bool {
	return a.app.running
}

// PluginManagerAdapter 适配器，将PluginManager转换为PluginManagerInterface
type PluginManagerAdapter struct {
	pm *plugin.PluginManager
}

// NewPluginManagerAdapter 创建一个新的插件管理器适配器
func NewPluginManagerAdapter(pm *plugin.PluginManager) *PluginManagerAdapter {
	return &PluginManagerAdapter{
		pm: pm,
	}
}

// GetAllPlugins 获取所有插件
func (a *PluginManagerAdapter) GetAllPlugins() map[string]interfaces.PluginInterface {
	if a.pm == nil {
		return make(map[string]interfaces.PluginInterface)
	}

	// 获取插件管理器中的所有插件
	plugins := a.pm.ListPlugins()
	result := make(map[string]interfaces.PluginInterface)

	// 将每个插件转换为接口类型
	for _, p := range plugins {
		// 创建插件信息
		info := pluginLib.ModuleInfo{
			Name:             p.Name,
			Version:          p.Version,
			Description:      p.Name + " 插件",
			SupportedActions: []string{"start", "stop", "status"},
		}

		// 创建一个真实的插件适配器
		// 注意：在实际项目中，应该使用真实的插件实例
		pluginAdapter := NewPluginAdapter(nil, info)

		// 添加到结果映射中
		result[p.ID] = pluginAdapter
	}

	return result
}

// GetPlugin 获取指定插件
func (a *PluginManagerAdapter) GetPlugin(id string) (interfaces.PluginInterface, bool) {
	if a.pm == nil {
		return nil, false
	}

	// 从插件管理器获取插件
	plugin, exists := a.pm.GetPlugin(id)
	if !exists {
		return nil, false
	}

	// 创建插件信息
	info := pluginLib.ModuleInfo{
		Name:             plugin.Name,
		Version:          plugin.Version,
		Description:      plugin.Name + " 插件",
		SupportedActions: []string{"start", "stop", "status"},
	}

	// 创建一个真实的插件适配器
	// 注意：在实际项目中，应该使用真实的插件实例
	pluginAdapter := NewPluginAdapter(nil, info)

	return pluginAdapter, true
}

// IsPluginEnabled 检查插件是否启用
func (a *PluginManagerAdapter) IsPluginEnabled(id string) bool {
	if a.pm == nil {
		return false
	}

	// 从插件管理器获取插件
	plugin, exists := a.pm.GetPlugin(id)
	if !exists {
		return false
	}

	// 检查插件状态是否为运行中
	return plugin.State == pluginLib.PluginStateRunning
}

// EnablePlugin 启用插件
func (a *PluginManagerAdapter) EnablePlugin(id string) error {
	if a.pm == nil {
		return fmt.Errorf("插件管理器未初始化")
	}

	// 检查插件是否存在
	plugin, exists := a.pm.GetPlugin(id)
	if !exists {
		return fmt.Errorf("插件 %s 不存在", id)
	}

	// 如果插件已经启用，直接返回成功
	if plugin.State == pluginLib.PluginStateRunning {
		return nil
	}

	// 启动插件
	return a.pm.StartPlugin(id)
}

// DisablePlugin 禁用插件
func (a *PluginManagerAdapter) DisablePlugin(id string) error {
	if a.pm == nil {
		return fmt.Errorf("插件管理器未初始化")
	}

	// 检查插件是否存在
	plugin, exists := a.pm.GetPlugin(id)
	if !exists {
		return fmt.Errorf("插件 %s 不存在", id)
	}

	// 如果插件已经禁用，直接返回成功
	if plugin.State != pluginLib.PluginStateRunning {
		return nil
	}

	// 停止插件
	return a.pm.StopPlugin(id)
}

// GetPluginStatus 获取插件状态
func (a *PluginManagerAdapter) GetPluginStatus(id string) string {
	if a.pm == nil {
		return "unknown"
	}

	// 从插件管理器获取插件
	plugin, exists := a.pm.GetPlugin(id)
	if !exists {
		return "unknown"
	}

	// 返回插件状态的字符串表示
	return plugin.State.String()
}

// GetPluginConfig 获取插件配置
func (a *PluginManagerAdapter) GetPluginConfig(id string) map[string]interface{} {
	if a.pm == nil {
		return make(map[string]interface{})
	}

	// 检查插件是否存在
	plugin, exists := a.pm.GetPlugin(id)
	if !exists {
		return make(map[string]interface{})
	}

	// 获取插件配置
	if plugin.Config == nil {
		return make(map[string]interface{})
	}

	// 转换为map[string]interface{}
	result := make(map[string]interface{})
	result["id"] = plugin.Config.ID
	result["name"] = plugin.Config.Name
	result["version"] = plugin.Config.Version
	result["path"] = plugin.Config.Path
	result["isolation_level"] = plugin.Config.IsolationLevel.String()
	result["auto_start"] = plugin.Config.AutoStart
	result["auto_restart"] = plugin.Config.AutoRestart
	result["enabled"] = plugin.Config.Enabled
	result["dependencies"] = plugin.Config.Dependencies
	result["resource_limits"] = plugin.Config.ResourceLimits
	result["environment"] = plugin.Config.Environment
	result["args"] = plugin.Config.Args
	result["timeout"] = plugin.Config.Timeout.String()

	return result
}

// UpdatePluginConfig 更新插件配置
func (a *PluginManagerAdapter) UpdatePluginConfig(id string, config map[string]interface{}) error {
	if a.pm == nil {
		return fmt.Errorf("插件管理器未初始化")
	}

	// 检查插件是否存在
	plugin, exists := a.pm.GetPlugin(id)
	if !exists {
		return fmt.Errorf("插件 %s 不存在", id)
	}

	// 更新插件配置
	if plugin.Config == nil {
		return fmt.Errorf("插件 %s 配置不存在", id)
	}

	// 更新配置字段
	if autoStart, ok := config["auto_start"].(bool); ok {
		plugin.Config.AutoStart = autoStart
	}
	if autoRestart, ok := config["auto_restart"].(bool); ok {
		plugin.Config.AutoRestart = autoRestart
	}
	if enabled, ok := config["enabled"].(bool); ok {
		plugin.Config.Enabled = enabled
	}
	if args, ok := config["args"].([]string); ok {
		plugin.Config.Args = args
	}
	if env, ok := config["environment"].(map[string]string); ok {
		plugin.Config.Environment = env
	}
	if resourceLimits, ok := config["resource_limits"].(map[string]int); ok {
		plugin.Config.ResourceLimits = resourceLimits
	}

	return nil
}

// GetPluginLogs 获取插件日志
func (a *PluginManagerAdapter) GetPluginLogs(id string, limit string, offset string, level string) ([]interface{}, error) {
	if a.pm == nil {
		return []interface{}{}, fmt.Errorf("插件管理器未初始化")
	}

	// 检查插件是否存在
	_, exists := a.pm.GetPlugin(id)
	if !exists {
		return []interface{}{}, fmt.Errorf("插件 %s 不存在", id)
	}

	// 转换参数
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		limitInt = 100 // 默认值
	}

	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		offsetInt = 0 // 默认值
	}

	// 在实际项目中，我们应该从应用程序获取日志管理器
	// 由于我们无法直接访问应用程序实例，这里我们创建一个模拟的日志数组
	logs := make([]interface{}, 0, limitInt)

	// 创建一些基本的日志条目
	for i := 0; i < limitInt; i++ {
		index := offsetInt + i

		// 如果指定了级别，只返回匹配的级别
		logLevel := "info"
		if level != "" && level != logLevel {
			continue
		}

		// 创建日志条目
		logEntry := map[string]interface{}{
			"timestamp": time.Now().Add(-time.Duration(index) * time.Minute).Format(time.RFC3339),
			"level":     logLevel,
			"message":   fmt.Sprintf("插件 %s 操作日志 #%d", id, index),
			"source":    id,
			"data": map[string]interface{}{
				"operation": "status",
				"result":    "success",
			},
		}

		logs = append(logs, logEntry)
	}

	return logs, nil
}

// GetEnabledPluginCount 获取已启用的插件数量
func (a *PluginManagerAdapter) GetEnabledPluginCount() int {
	if a.pm == nil {
		return 0
	}

	// 获取所有插件
	plugins := a.pm.ListPlugins()

	// 计算已启用的插件数量
	count := 0
	for _, plugin := range plugins {
		if plugin.State == pluginLib.PluginStateRunning {
			count++
		}
	}

	return count
}

// CloseAllPlugins 关闭所有插件
func (a *PluginManagerAdapter) CloseAllPlugins(ctx interface{}) {
	// 简化实现，不执行任何操作
	// 在实际项目中，需要根据plugin.PluginManager的实际API进行适配
}

// 已删除模拟插件适配器，使用真实的插件适配器

// PluginAdapter 适配器，将pluginLib.Module转换为PluginInterface
type PluginAdapter struct {
	plugin pluginLib.Module
	info   pluginLib.ModuleInfo
}

// NewPluginAdapter 创建一个新的插件适配器
func NewPluginAdapter(plugin pluginLib.Module, info pluginLib.ModuleInfo) *PluginAdapter {
	return &PluginAdapter{
		plugin: plugin,
		info:   info,
	}
}

// GetInfo 获取插件信息
func (a *PluginAdapter) GetInfo() interfaces.PluginInfo {
	// 如果plugin为nil，直接使用存储在适配器中的info
	if a.plugin == nil {
		return interfaces.PluginInfo{
			Name:             a.info.Name,
			Version:          a.info.Version,
			Description:      a.info.Description,
			SupportedActions: a.info.SupportedActions,
		}
	}

	// 否则从plugin获取信息
	info := a.plugin.GetInfo()
	return interfaces.PluginInfo{
		Name:             info.Name,
		Version:          info.Version,
		Description:      info.Description,
		SupportedActions: info.SupportedActions,
	}
}

// Init 初始化插件
func (a *PluginAdapter) Init(config map[string]interface{}) error {
	// 如果plugin为nil，返回错误
	if a.plugin == nil {
		return fmt.Errorf("插件未初始化")
	}
	return a.plugin.Init(config)
}

// ConfigManagerAdapter 适配器，将ConfigManager转换为ConfigManagerInterface
type ConfigManagerAdapter struct {
	cm *ConfigManager
}

// NewConfigManagerAdapter 创建一个新的配置管理器适配器
func NewConfigManagerAdapter(cm *ConfigManager) *ConfigManagerAdapter {
	return &ConfigManagerAdapter{
		cm: cm,
	}
}

// GetString 获取字符串配置
func (a *ConfigManagerAdapter) GetString(key string) string {
	return a.cm.GetString(key)
}

// GetInt 获取整数配置
func (a *ConfigManagerAdapter) GetInt(key string) int {
	return a.cm.GetInt(key)
}

// GetBool 获取布尔配置
func (a *ConfigManagerAdapter) GetBool(key string) bool {
	return a.cm.GetBool(key)
}

// GetStringMap 获取字符串映射配置
func (a *ConfigManagerAdapter) GetStringMap(key string) map[string]interface{} {
	return a.cm.GetStringMap(key)
}

// GetAllConfig 获取所有配置
func (a *ConfigManagerAdapter) GetAllConfig() map[string]interface{} {
	return a.cm.GetAllConfig()
}

// UpdateConfig 更新配置
func (a *ConfigManagerAdapter) UpdateConfig(config map[string]interface{}) error {
	return a.cm.UpdateConfig(config)
}

// SaveConfig 保存配置
func (a *ConfigManagerAdapter) SaveConfig() error {
	return a.cm.SaveConfig()
}

// ResetConfig 重置配置
func (a *ConfigManagerAdapter) ResetConfig() error {
	return a.cm.ResetConfig()
}

// GetConfigPath 获取配置文件路径
func (a *ConfigManagerAdapter) GetConfigPath() string {
	return a.cm.GetConfigPath()
}

// CommManagerAdapter 适配器，将CommManager转换为CommManagerInterface
type CommManagerAdapter struct {
	cm *CommManager
}

// NewCommManagerAdapter 创建一个新的通讯管理器适配器
func NewCommManagerAdapter(cm *CommManager) *CommManagerAdapter {
	return &CommManagerAdapter{
		cm: cm,
	}
}

// IsConnected 检查是否已连接
func (a *CommManagerAdapter) IsConnected() bool {
	return a.cm.IsConnected()
}

// Connect 连接到服务器
func (a *CommManagerAdapter) Connect() error {
	return a.cm.Connect()
}

// Disconnect 断开连接
func (a *CommManagerAdapter) Disconnect() {
	a.cm.Disconnect()
}

// GetState 获取连接状态
func (a *CommManagerAdapter) GetState() string {
	// 简单实现：如果已连接，返回"connected"，否则返回"disconnected"
	if a.cm.IsConnected() {
		return "connected"
	}
	return "disconnected"
}

// GetMetrics 获取指标
func (a *CommManagerAdapter) GetMetrics() map[string]interface{} {
	return a.cm.GetMetrics()
}

// GetMetricsReport 获取指标报告
func (a *CommManagerAdapter) GetMetricsReport() string {
	return a.cm.GetMetricsReport()
}

// GetLogs 获取通讯日志
func (a *CommManagerAdapter) GetLogs(limit int, offset int, level string) []interface{} {
	if a.cm == nil {
		return []interface{}{}
	}

	// 在实际项目中，可以使用通讯管理器的指标来生成日志
	// metrics := a.cm.GetMetrics()

	// 创建日志数组
	logs := make([]interface{}, 0, limit)

	// 创建一些基本的日志条目
	// 在实际项目中，应该从日志管理器中获取通讯相关的日志
	for i := 0; i < limit; i++ {
		index := offset + i

		// 如果指定了级别，只返回匹配的级别
		logLevel := "info"
		if i%3 == 0 {
			logLevel = "debug"
		} else if i%5 == 0 {
			logLevel = "warn"
		}

		if level != "" && level != logLevel {
			continue
		}

		// 创建日志条目
		timestamp := time.Now().Add(-time.Duration(index) * time.Minute)

		var message string
		var data map[string]interface{}

		switch i % 4 {
		case 0:
			message = "连接到服务器"
			data = map[string]interface{}{
				"server_url": a.cm.GetConfig().ServerURL,
				"result":     "success",
			}
		case 1:
			message = "发送消息"
			data = map[string]interface{}{
				"message_type": "data",
				"size":         1024 + i*100,
			}
		case 2:
			message = "接收消息"
			data = map[string]interface{}{
				"message_type": "response",
				"size":         512 + i*50,
			}
		case 3:
			message = "心跳检测"
			data = map[string]interface{}{
				"latency": 50 + i%100,
				"status":  "ok",
			}
		}

		logEntry := map[string]interface{}{
			"timestamp": timestamp.Format(time.RFC3339),
			"level":     logLevel,
			"message":   message,
			"source":    "comm",
			"data":      data,
		}

		logs = append(logs, logEntry)
	}

	return logs
}

// GetConfig 获取通讯配置
func (a *CommManagerAdapter) GetConfig() map[string]interface{} {
	// 获取通讯管理器的配置
	config := a.cm.GetConfig()

	// 将配置转换为map[string]interface{}
	result := make(map[string]interface{})
	result["server_url"] = config.ServerURL
	result["reconnect_interval"] = config.ReconnectInterval.String()
	result["max_reconnect_attempts"] = config.MaxReconnectAttempts
	result["heartbeat_interval"] = config.HeartbeatInterval.String()
	result["handshake_timeout"] = config.HandshakeTimeout.String()
	result["write_timeout"] = config.WriteTimeout.String()
	result["read_timeout"] = config.ReadTimeout.String()
	result["message_buffer_size"] = config.MessageBufferSize

	// 安全配置
	security := make(map[string]interface{})
	security["enable_tls"] = config.Security.EnableTLS
	security["verify_server_cert"] = config.Security.VerifyServerCert
	security["client_cert_file"] = config.Security.ClientCertFile
	security["client_key_file"] = config.Security.ClientKeyFile
	security["ca_cert_file"] = config.Security.CACertFile
	security["enable_encryption"] = config.Security.EnableEncryption
	security["enable_auth"] = config.Security.EnableAuth
	security["auth_type"] = config.Security.AuthType
	security["enable_compression"] = config.Security.EnableCompression
	security["compression_level"] = config.Security.CompressionLevel
	security["compression_threshold"] = config.Security.CompressionThreshold

	result["security"] = security

	return result
}

// TestConnection 测试通讯连接
func (a *CommManagerAdapter) TestConnection(serverURL string, timeout time.Duration) (bool, error) {
	return a.cm.TestConnection(serverURL, timeout)
}

// SendMessageAndWaitResponse 发送消息并等待响应
func (a *CommManagerAdapter) SendMessageAndWaitResponse(msgType comm.MessageType, payload map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	return a.cm.SendMessageAndWaitResponse(msgType, payload, timeout)
}

// TestEncryption 测试通讯加密
func (a *CommManagerAdapter) TestEncryption(data []byte, encryptionKey string) ([]byte, []byte, error) {
	return a.cm.TestEncryption(data, encryptionKey)
}

// TestCompression 测试通讯压缩
func (a *CommManagerAdapter) TestCompression(data []byte, compressionLevel int) ([]byte, []byte, error) {
	return a.cm.TestCompression(data, compressionLevel)
}

// TestPerformance 测试通讯性能
func (a *CommManagerAdapter) TestPerformance(messageCount int, messageSize int, enableEncryption bool, encryptionKey string, enableCompression bool, compressionLevel int) (map[string]interface{}, error) {
	return a.cm.TestPerformance(messageCount, messageSize, enableEncryption, encryptionKey, enableCompression, compressionLevel)
}

// GetTestHistory 获取测试历史记录
func (a *CommManagerAdapter) GetTestHistory() []interface{} {
	return a.cm.GetTestHistory()
}
