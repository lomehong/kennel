package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/lomehong/kennel/pkg/logging"
)

// 辅助函数，用于从配置中获取字符串切片
func getConfigStringSliceFromNetwork(config map[string]interface{}, key string) []string {
	if val, ok := config[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, len(slice))
			for i, v := range slice {
				if str, ok := v.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}
	return nil
}

// 辅助函数，用于从配置中获取布尔值
func getConfigBoolFromNetwork(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := config[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// NetworkManager 网络管理器
type NetworkManager struct {
	logger              logging.Logger
	config              map[string]interface{}
	protectedInterfaces map[string]bool
}

// NewNetworkManager 创建一个新的网络管理器
func NewNetworkManager(logger logging.Logger, config map[string]interface{}) *NetworkManager {
	// 创建网络管理器
	manager := &NetworkManager{
		logger:              logger,
		config:              config,
		protectedInterfaces: make(map[string]bool),
	}

	// 初始化受保护的网络接口
	manager.initProtectedInterfaces()

	return manager
}

// initProtectedInterfaces 初始化受保护的网络接口
func (m *NetworkManager) initProtectedInterfaces() {
	// 获取受保护的网络接口列表
	protectedInterfaces := getConfigStringSliceFromNetwork(m.config, "protected_interfaces")
	for _, iface := range protectedInterfaces {
		m.protectedInterfaces[strings.ToLower(iface)] = true
	}

	m.logger.Debug("初始化受保护的网络接口", "count", len(m.protectedInterfaces))
}

// GetNetworkInterfaces 获取网络接口列表
func (m *NetworkManager) GetNetworkInterfaces() ([]NetworkInterface, error) {
	m.logger.Info("获取网络接口列表")

	// 获取所有网络接口
	ifaces, err := net.Interfaces()
	if err != nil {
		m.logger.Error("获取网络接口列表失败", "error", err)
		return nil, fmt.Errorf("获取网络接口列表失败: %w", err)
	}

	// 转换为NetworkInterface
	networkInterfaces := make([]NetworkInterface, 0, len(ifaces))
	for _, iface := range ifaces {
		// 获取IP地址
		addrs, err := iface.Addrs()
		if err != nil {
			m.logger.Debug("获取网络接口地址失败", "interface", iface.Name, "error", err)
			continue
		}

		// 提取IP地址
		ipAddresses := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ipAddresses = append(ipAddresses, ipNet.IP.String())
		}

		// 确定接口状态
		status := "down"
		if iface.Flags&net.FlagUp != 0 {
			status = "up"
		}

		// 创建网络接口信息
		networkInterface := NetworkInterface{
			Name:       iface.Name,
			MACAddress: iface.HardwareAddr.String(),
			IPAddress:  ipAddresses,
			Status:     status,
		}

		networkInterfaces = append(networkInterfaces, networkInterface)
	}

	return networkInterfaces, nil
}

// EnableNetworkInterface 启用网络接口
func (m *NetworkManager) EnableNetworkInterface(name string) error {
	m.logger.Info("启用网络接口", "interface", name)

	// 检查是否允许禁用网络接口
	if !getConfigBoolFromNetwork(m.config, "allow_network_disable", true) {
		return fmt.Errorf("不允许修改网络接口状态")
	}

	// 在实际应用中，这里应该调用系统API启用网络接口
	// 由于这是平台相关的，这里只是一个示例
	m.logger.Info("网络接口已启用", "interface", name)

	return nil
}

// DisableNetworkInterface 禁用网络接口
func (m *NetworkManager) DisableNetworkInterface(name string) error {
	m.logger.Info("禁用网络接口", "interface", name)

	// 检查是否允许禁用网络接口
	if !getConfigBoolFromNetwork(m.config, "allow_network_disable", true) {
		return fmt.Errorf("不允许修改网络接口状态")
	}

	// 检查是否是受保护的网络接口
	if m.protectedInterfaces[strings.ToLower(name)] {
		m.logger.Warn("尝试禁用受保护的网络接口", "interface", name)
		return fmt.Errorf("不允许禁用受保护的网络接口: %s", name)
	}

	// 在实际应用中，这里应该调用系统API禁用网络接口
	// 由于这是平台相关的，这里只是一个示例
	m.logger.Info("网络接口已禁用", "interface", name)

	return nil
}

// GetNetworkInterfaceInfo 获取网络接口信息
func (m *NetworkManager) GetNetworkInterfaceInfo(name string) (*NetworkInterface, error) {
	m.logger.Debug("获取网络接口信息", "interface", name)

	// 获取所有网络接口
	interfaces, err := m.GetNetworkInterfaces()
	if err != nil {
		return nil, err
	}

	// 查找指定的网络接口
	for _, iface := range interfaces {
		if iface.Name == name {
			return &iface, nil
		}
	}

	return nil, fmt.Errorf("未找到网络接口: %s", name)
}
