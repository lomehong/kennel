package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lomehong/kennel/pkg/core/plugin"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// DeviceModule 实现了设备管理模块
type DeviceModule struct {
	*sdk.BaseModule
	deviceCache    *DeviceCache
	networkManager *NetworkManager
	usbManager     *USBManager
	monitorCtx     context.Context
	monitorCancel  context.CancelFunc
}

// NewDeviceModule 创建一个新的设备管理模块
func NewDeviceModule() *DeviceModule {
	// 创建基础模块
	base := sdk.NewBaseModule(
		"device",
		"设备管理插件",
		"1.0.0",
		"设备管理模块，用于监控和管理网络接口和USB设备",
	)

	// 创建设备缓存
	deviceCache := NewDeviceCache()

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建模块
	module := &DeviceModule{
		BaseModule:    base,
		deviceCache:   deviceCache,
		monitorCtx:    ctx,
		monitorCancel: cancel,
	}

	return module
}

// Init 初始化模块
func (m *DeviceModule) Init(ctx context.Context, config *plugin.ModuleConfig) error {
	// 调用基类初始化
	if err := m.BaseModule.Init(ctx, config); err != nil {
		return err
	}

	m.Logger.Info("初始化设备管理模块")

	// 设置日志级别
	logLevel := sdk.GetConfigString(m.Config, "log_level", "info")
	m.Logger.Debug("设置日志级别", "level", logLevel)

	// 创建网络管理器
	m.networkManager = NewNetworkManager(m.Logger, m.Config)

	// 创建USB管理器
	m.usbManager = NewUSBManager(m.Logger, m.Config)

	return nil
}

// Start 启动模块
func (m *DeviceModule) Start() error {
	m.Logger.Info("启动设备管理模块")

	// 确保网络管理器和USB管理器已初始化
	if m.networkManager == nil {
		m.Logger.Warn("网络管理器未初始化，尝试初始化")
		m.networkManager = NewNetworkManager(m.Logger, m.Config)
	}

	if m.usbManager == nil {
		m.Logger.Warn("USB管理器未初始化，尝试初始化")
		m.usbManager = NewUSBManager(m.Logger, m.Config)
	}

	// 确保设备缓存已初始化
	if m.deviceCache == nil {
		m.Logger.Warn("设备缓存未初始化，尝试初始化")
		m.deviceCache = NewDeviceCache()
	}

	// 确保监控上下文已初始化
	if m.monitorCtx == nil {
		m.Logger.Warn("监控上下文未初始化，尝试初始化")
		m.monitorCtx, m.monitorCancel = context.WithCancel(context.Background())
	}

	// 启动设备监控
	go m.startDeviceMonitor()

	return nil
}

// Stop 停止模块
func (m *DeviceModule) Stop() error {
	m.Logger.Info("停止设备管理模块")

	// 停止设备监控
	if m.monitorCancel != nil {
		m.monitorCancel()
	} else {
		m.Logger.Warn("监控取消函数未初始化，跳过停止监控")
	}

	// 停止USB设备监控
	if m.usbManager != nil {
		if err := m.usbManager.StopMonitorUSBDevices(); err != nil {
			m.Logger.Error("停止USB设备监控失败", "error", err)
		}
	} else {
		m.Logger.Warn("USB管理器未初始化，跳过停止USB设备监控")
	}

	return nil
}

// startDeviceMonitor 启动设备监控
func (m *DeviceModule) startDeviceMonitor() {
	m.Logger.Info("启动设备监控")

	// 检查是否启用设备监控
	monitorNetwork := sdk.GetConfigBool(m.Config, "monitor_network", true)
	monitorUSB := sdk.GetConfigBool(m.Config, "monitor_usb", true)

	if !monitorNetwork && !monitorUSB {
		m.Logger.Info("设备监控已禁用")
		return
	}

	// 启动USB设备监控
	if monitorUSB {
		if err := m.usbManager.MonitorUSBDevices(); err != nil {
			m.Logger.Error("启动USB设备监控失败", "error", err)
		}
	}

	// 获取监控间隔
	monitorInterval := sdk.GetConfigInt(m.Config, "monitor_interval", 60)
	ticker := time.NewTicker(time.Duration(monitorInterval) * time.Second)
	defer ticker.Stop()

	// 监控循环
	for {
		select {
		case <-ticker.C:
			// 收集设备信息
			_, err := m.collectDeviceInfo()
			if err != nil {
				m.Logger.Error("收集设备信息失败", "error", err)
			}
		case <-m.monitorCtx.Done():
			m.Logger.Info("设备监控已停止")
			return
		}
	}
}

// HandleRequest 处理请求
func (m *DeviceModule) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	m.Logger.Info("处理请求", "action", req.Action)

	// 确保网络管理器和USB管理器已初始化
	if m.networkManager == nil {
		m.Logger.Warn("网络管理器未初始化，尝试初始化")
		m.networkManager = NewNetworkManager(m.Logger, m.Config)
	}

	if m.usbManager == nil {
		m.Logger.Warn("USB管理器未初始化，尝试初始化")
		m.usbManager = NewUSBManager(m.Logger, m.Config)
	}

	// 确保设备缓存已初始化
	if m.deviceCache == nil {
		m.Logger.Warn("设备缓存未初始化，尝试初始化")
		m.deviceCache = NewDeviceCache()
	}

	switch req.Action {
	case "get_devices":
		// 获取设备信息
		deviceInfo, err := m.collectDeviceInfo()
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "device_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data:    deviceInfo,
		}, nil

	case "get_network_interfaces":
		// 获取网络接口列表
		interfaces, err := m.networkManager.GetNetworkInterfaces()
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "network_error",
					Message: err.Error(),
				},
			}, nil
		}

		// 转换为map切片
		interfacesMap := make([]map[string]interface{}, len(interfaces))
		for i, iface := range interfaces {
			interfacesMap[i] = map[string]interface{}{
				"name":        iface.Name,
				"mac_address": iface.MACAddress,
				"ip_address":  iface.IPAddress,
				"status":      iface.Status,
			}
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"interfaces": interfacesMap,
				"count":      len(interfaces),
			},
		}, nil

	case "get_usb_devices":
		// 获取USB设备列表
		devices, err := m.usbManager.GetUSBDevices()
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "usb_error",
					Message: err.Error(),
				},
			}, nil
		}

		// 转换为map切片
		devicesMap := make([]map[string]interface{}, len(devices))
		for i, device := range devices {
			devicesMap[i] = map[string]interface{}{
				"id":           device.ID,
				"name":         device.Name,
				"manufacturer": device.Manufacturer,
				"status":       device.Status,
			}
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"devices": devicesMap,
				"count":   len(devices),
			},
		}, nil

	case "enable_network_interface":
		// 启用网络接口
		name, ok := req.Params["name"].(string)
		if !ok {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "缺少网络接口名称参数",
				},
			}, nil
		}

		// 启用网络接口
		if err := m.networkManager.EnableNetworkInterface(name); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "enable_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"status":  "success",
				"message": fmt.Sprintf("网络接口 %s 已启用", name),
			},
		}, nil

	case "disable_network_interface":
		// 禁用网络接口
		name, ok := req.Params["name"].(string)
		if !ok {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "缺少网络接口名称参数",
				},
			}, nil
		}

		// 禁用网络接口
		if err := m.networkManager.DisableNetworkInterface(name); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "disable_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"status":  "success",
				"message": fmt.Sprintf("网络接口 %s 已禁用", name),
			},
		}, nil

	default:
		return &plugin.Response{
			ID:      req.ID,
			Success: false,
			Error: &plugin.ErrorInfo{
				Code:    "unknown_action",
				Message: fmt.Sprintf("不支持的操作: %s", req.Action),
			},
		}, nil
	}
}

// HandleEvent 处理事件
func (m *DeviceModule) HandleEvent(ctx context.Context, event *plugin.Event) error {
	m.Logger.Info("处理事件", "type", event.Type, "source", event.Source)

	// 确保网络管理器和USB管理器已初始化
	if m.networkManager == nil {
		m.Logger.Warn("网络管理器未初始化，尝试初始化")
		m.networkManager = NewNetworkManager(m.Logger, m.Config)
	}

	if m.usbManager == nil {
		m.Logger.Warn("USB管理器未初始化，尝试初始化")
		m.usbManager = NewUSBManager(m.Logger, m.Config)
	}

	// 确保设备缓存已初始化
	if m.deviceCache == nil {
		m.Logger.Warn("设备缓存未初始化，尝试初始化")
		m.deviceCache = NewDeviceCache()
	}

	switch event.Type {
	case "system.startup":
		// 系统启动事件
		m.Logger.Info("系统启动")
		return nil

	case "system.shutdown":
		// 系统关闭事件
		m.Logger.Info("系统关闭")
		return nil

	case "device.scan_request":
		// 设备扫描请求
		m.Logger.Info("收到设备扫描请求")
		_, err := m.collectDeviceInfo()
		return err

	default:
		// 忽略其他事件
		return nil
	}
}

// collectDeviceInfo 收集设备信息
func (m *DeviceModule) collectDeviceInfo() (map[string]interface{}, error) {
	// 获取缓存过期时间
	cacheExpiration := time.Duration(sdk.GetConfigInt(m.Config, "device_cache_expiration", 30)) * time.Second

	// 检查缓存是否有效
	if deviceInfo, valid := m.deviceCache.GetCachedDevices(cacheExpiration); valid {
		m.Logger.Debug("使用缓存的设备信息")
		return DeviceInfoToMap(deviceInfo), nil
	}

	m.Logger.Info("收集设备信息")

	// 创建设备信息
	deviceInfo := &DeviceInfo{
		NetworkInterfaces: []NetworkInterface{},
		USBDevices:        []USBDevice{},
	}

	// 收集网络接口信息
	if sdk.GetConfigBool(m.Config, "monitor_network", true) {
		interfaces, err := m.networkManager.GetNetworkInterfaces()
		if err != nil {
			m.Logger.Error("获取网络接口信息失败", "error", err)
		} else {
			deviceInfo.NetworkInterfaces = interfaces
		}
	}

	// 收集USB设备信息
	if sdk.GetConfigBool(m.Config, "monitor_usb", true) {
		devices, err := m.usbManager.GetUSBDevices()
		if err != nil {
			m.Logger.Error("获取USB设备信息失败", "error", err)
		} else {
			deviceInfo.USBDevices = devices
		}
	}

	// 更新缓存
	m.deviceCache.SetCachedDevices(deviceInfo)

	// 转换为map
	return DeviceInfoToMap(deviceInfo), nil
}
