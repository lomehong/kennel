package main

import (
	"sync"
	"time"
)

// NetworkInterface 表示网络接口
type NetworkInterface struct {
	Name       string   `json:"name"`
	MACAddress string   `json:"mac_address"`
	IPAddress  []string `json:"ip_address"`
	Status     string   `json:"status"`
}

// USBDevice 表示USB设备
type USBDevice struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	Status      string `json:"status"`
}

// DeviceInfo 表示设备信息
type DeviceInfo struct {
	NetworkInterfaces []NetworkInterface `json:"network_interfaces"`
	USBDevices        []USBDevice        `json:"usb_devices"`
	CollectTime       time.Time          `json:"collect_time"`
}

// DeviceCache 提供设备信息的缓存功能
type DeviceCache struct {
	mu         sync.RWMutex
	deviceInfo *DeviceInfo
}

// NewDeviceCache 创建一个新的设备缓存
func NewDeviceCache() *DeviceCache {
	return &DeviceCache{}
}

// GetCachedDevices 获取缓存的设备信息，如果缓存有效则返回true
func (c *DeviceCache) GetCachedDevices(expiration time.Duration) (*DeviceInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.deviceInfo == nil {
		return nil, false
	}

	// 检查缓存是否过期
	if time.Since(c.deviceInfo.CollectTime) > expiration {
		return nil, false
	}

	return c.deviceInfo, true
}

// SetCachedDevices 设置缓存的设备信息
func (c *DeviceCache) SetCachedDevices(deviceInfo *DeviceInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 设置收集时间
	deviceInfo.CollectTime = time.Now()
	c.deviceInfo = deviceInfo
}

// DeviceInfoToMap 将DeviceInfo转换为map
func DeviceInfoToMap(info *DeviceInfo) map[string]interface{} {
	// 转换网络接口
	networkInterfaces := make([]map[string]interface{}, len(info.NetworkInterfaces))
	for i, iface := range info.NetworkInterfaces {
		networkInterfaces[i] = map[string]interface{}{
			"name":        iface.Name,
			"mac_address": iface.MACAddress,
			"ip_address":  iface.IPAddress,
			"status":      iface.Status,
		}
	}

	// 转换USB设备
	usbDevices := make([]map[string]interface{}, len(info.USBDevices))
	for i, device := range info.USBDevices {
		usbDevices[i] = map[string]interface{}{
			"id":           device.ID,
			"name":         device.Name,
			"manufacturer": device.Manufacturer,
			"status":       device.Status,
		}
	}

	return map[string]interface{}{
		"network_interfaces": networkInterfaces,
		"usb_devices":        usbDevices,
		"collect_time":       info.CollectTime.Format(time.RFC3339),
	}
}
