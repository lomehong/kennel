package main

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/logger"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
	"github.com/lomehong/kennel/pkg/utils"
)

// DeviceModule 实现了设备管理模块
type DeviceModule struct {
	logger         logger.Logger
	config         map[string]interface{}
	deviceCache    *DeviceCache
	networkManager *NetworkManager
	usbManager     *USBManager
}

// NewDeviceModule 创建一个新的设备管理模块
func NewDeviceModule() pluginLib.Module {
	// 创建日志器
	log := logger.NewLogger("device-module", hclog.Info)

	// 创建设备缓存
	deviceCache := NewDeviceCache()

	// 创建网络管理器
	networkManager := NewNetworkManager(log)

	// 创建USB管理器
	usbManager := NewUSBManager(log)

	return &DeviceModule{
		logger:         log,
		config:         make(map[string]interface{}),
		deviceCache:    deviceCache,
		networkManager: networkManager,
		usbManager:     usbManager,
	}
}

// Init 初始化模块
func (m *DeviceModule) Init(config map[string]interface{}) error {
	m.logger.Info("初始化设备管理模块")
	m.config = config
	return nil
}

// Execute 执行模块操作
func (m *DeviceModule) Execute(action string, params map[string]interface{}) (map[string]interface{}, error) {
	m.logger.Info("执行操作", "action", action)

	switch action {
	case "list_devices":
		return m.listDevices()
	case "enable_network":
		if name, ok := params["interface"].(string); ok {
			return m.networkManager.EnableNetwork(name)
		}
		return nil, fmt.Errorf("缺少接口名称参数")
	case "disable_network":
		if name, ok := params["interface"].(string); ok {
			return m.networkManager.DisableNetwork(name)
		}
		return nil, fmt.Errorf("缺少接口名称参数")
	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
}

// Shutdown 关闭模块
func (m *DeviceModule) Shutdown() error {
	m.logger.Info("关闭设备管理模块")
	return nil
}

// GetInfo 获取模块信息
func (m *DeviceModule) GetInfo() pluginLib.ModuleInfo {
	return pluginLib.ModuleInfo{
		Name:             "device",
		Version:          "0.1.0",
		Description:      "设备管理模块，用于管理终端设备",
		SupportedActions: []string{"list_devices", "enable_network", "disable_network"},
	}
}

// HandleMessage 处理消息
func (m *DeviceModule) HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error) {
	m.logger.Info("处理消息", "type", messageType, "id", messageID)

	switch messageType {
	case "device_scan_request":
		// 处理设备扫描请求
		return m.listDevices()
	case "network_control":
		// 处理网络控制请求
		if action, ok := payload["action"].(string); ok {
			if action == "enable" {
				if iface, ok := payload["interface"].(string); ok {
					return m.networkManager.EnableNetwork(iface)
				}
			} else if action == "disable" {
				if iface, ok := payload["interface"].(string); ok {
					return m.networkManager.DisableNetwork(iface)
				}
			}
		}
		return nil, fmt.Errorf("无效的网络控制请求")
	default:
		return nil, fmt.Errorf("不支持的消息类型: %s", messageType)
	}
}

// listDevices 列出设备
func (m *DeviceModule) listDevices() (map[string]interface{}, error) {
	// 获取缓存过期时间（默认为5分钟）
	cacheExpiration := 5 * time.Minute
	if expStr := utils.GetString(m.config, "device_cache_interval", ""); expStr != "" {
		if exp, err := time.ParseDuration(expStr); err == nil {
			cacheExpiration = exp
		}
	} else if expSec := utils.GetFloat(m.config, "device_cache_interval", 0); expSec > 0 {
		cacheExpiration = time.Duration(expSec) * time.Second
	}

	// 检查缓存是否有效
	if deviceInfo, valid := m.deviceCache.GetCachedDevices(cacheExpiration); valid {
		m.logger.Debug("使用缓存的设备信息")
		return DeviceInfoToMap(deviceInfo), nil
	}

	m.logger.Info("收集设备信息")

	// 获取网络接口
	networkInterfaces, err := m.networkManager.GetNetworkInterfaces()
	if err != nil {
		m.logger.Error("获取网络接口失败", "error", err)
		// 继续执行，不返回错误
		networkInterfaces = []NetworkInterface{} // 确保不是nil
	}

	// 获取USB设备
	usbDevices, err := m.usbManager.GetUSBDevices()
	if err != nil {
		m.logger.Warn("获取USB设备失败", "error", err)
		// 继续执行，不返回错误
		usbDevices = []USBDevice{} // 确保不是nil
	}

	// 构建设备信息
	deviceInfo := &DeviceInfo{
		NetworkInterfaces: networkInterfaces,
		USBDevices:        usbDevices,
	}

	// 更新缓存
	m.deviceCache.SetCachedDevices(deviceInfo)

	// 转换为map
	return DeviceInfoToMap(deviceInfo), nil
}
