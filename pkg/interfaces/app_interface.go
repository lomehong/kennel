package interfaces

import (
	"time"

	"github.com/lomehong/kennel/pkg/comm"
)

// AppInterface 定义应用程序接口，用于解耦core和webconsole包
type AppInterface interface {
	// GetPluginManager 获取插件管理器接口
	GetPluginManager() PluginManagerInterface

	// GetConfigManager 获取配置管理器接口
	GetConfigManager() ConfigManagerInterface

	// GetCommManager 获取通讯管理器接口
	GetCommManager() CommManagerInterface

	// GetLogManager 获取日志管理器接口
	GetLogManager() LogManagerInterface

	// GetEventManager 获取事件管理器接口
	GetEventManager() EventManagerInterface

	// GetSystemMonitor 获取系统监控器接口
	GetSystemMonitor() SystemMonitorInterface

	// GetStartTime 获取应用程序启动时间
	GetStartTime() time.Time

	// GetVersion 获取应用程序版本
	GetVersion() string

	// IsRunning 检查应用程序是否正在运行
	IsRunning() bool
}

// PluginManagerInterface 定义插件管理器接口
type PluginManagerInterface interface {
	// GetAllPlugins 获取所有插件
	GetAllPlugins() map[string]PluginInterface

	// GetPlugin 获取指定插件
	GetPlugin(id string) (PluginInterface, bool)

	// IsPluginEnabled 检查插件是否启用
	IsPluginEnabled(id string) bool

	// EnablePlugin 启用插件
	EnablePlugin(id string) error

	// DisablePlugin 禁用插件
	DisablePlugin(id string) error

	// GetPluginStatus 获取插件状态
	GetPluginStatus(id string) string

	// GetPluginConfig 获取插件配置
	GetPluginConfig(id string) map[string]interface{}

	// UpdatePluginConfig 更新插件配置
	UpdatePluginConfig(id string, config map[string]interface{}) error

	// GetPluginLogs 获取插件日志
	GetPluginLogs(id string, limit string, offset string, level string) ([]interface{}, error)

	// GetEnabledPluginCount 获取已启用的插件数量
	GetEnabledPluginCount() int

	// CloseAllPlugins 关闭所有插件
	CloseAllPlugins(ctx interface{})
}

// PluginInterface 定义插件接口
type PluginInterface interface {
	// GetInfo 获取插件信息
	GetInfo() PluginInfo

	// Init 初始化插件
	Init(config map[string]interface{}) error
}

// PluginInfo 定义插件信息
type PluginInfo struct {
	Name             string
	Version          string
	Description      string
	SupportedActions []string
}

// ConfigManagerInterface 定义配置管理器接口
type ConfigManagerInterface interface {
	// GetString 获取字符串配置
	GetString(key string) string

	// GetInt 获取整数配置
	GetInt(key string) int

	// GetBool 获取布尔配置
	GetBool(key string) bool

	// GetStringMap 获取字符串映射配置
	GetStringMap(key string) map[string]interface{}

	// GetAllConfig 获取所有配置
	GetAllConfig() map[string]interface{}

	// UpdateConfig 更新配置
	UpdateConfig(config map[string]interface{}) error

	// SaveConfig 保存配置
	SaveConfig() error

	// ResetConfig 重置配置
	ResetConfig() error

	// GetConfigPath 获取配置文件路径
	GetConfigPath() string
}

// CommManagerInterface 定义通讯管理器接口
type CommManagerInterface interface {
	// IsConnected 检查是否已连接
	IsConnected() bool

	// Connect 连接到服务器
	Connect() error

	// Disconnect 断开连接
	Disconnect()

	// GetState 获取连接状态
	GetState() string

	// GetMetrics 获取指标
	GetMetrics() map[string]interface{}

	// GetMetricsReport 获取指标报告
	GetMetricsReport() string

	// GetLogs 获取通讯日志
	GetLogs(limit int, offset int, level string) []interface{}

	// GetConfig 获取通讯配置
	GetConfig() map[string]interface{}

	// TestConnection 测试通讯连接
	TestConnection(serverURL string, timeout time.Duration) (bool, error)

	// SendMessageAndWaitResponse 发送消息并等待响应
	SendMessageAndWaitResponse(msgType comm.MessageType, payload map[string]interface{}, timeout time.Duration) (map[string]interface{}, error)

	// TestEncryption 测试通讯加密
	TestEncryption(data []byte, encryptionKey string) ([]byte, []byte, error)

	// TestCompression 测试通讯压缩
	TestCompression(data []byte, compressionLevel int) ([]byte, []byte, error)

	// TestPerformance 测试通讯性能
	TestPerformance(messageCount int, messageSize int, enableEncryption bool, encryptionKey string, enableCompression bool, compressionLevel int) (map[string]interface{}, error)

	// GetTestHistory 获取测试历史记录
	GetTestHistory() []interface{}
}

// LogManagerInterface 定义日志管理器接口
type LogManagerInterface interface {
	// GetLogs 获取日志
	GetLogs(limit int, offset int, level string, source string) ([]interface{}, error)
}

// EventManagerInterface 定义事件管理器接口
type EventManagerInterface interface {
	// GetEvents 获取事件
	GetEvents(limit int, offset int, eventType string, source string) ([]interface{}, error)
}

// SystemMonitorInterface 定义系统监控器接口
type SystemMonitorInterface interface {
	// GetSystemMetrics 获取系统指标
	GetSystemMetrics() (map[string]interface{}, error)

	// GetSystemResources 获取系统资源详细信息
	GetSystemResources() (map[string]interface{}, error)

	// GetSystemStatus 获取系统状态
	GetSystemStatus() (map[string]interface{}, error)

	// GetHostname 获取主机名
	GetHostname() string
}
