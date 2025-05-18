package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/lomehong/kennel/pkg/logger"
)

// USBManager 负责管理USB设备
type USBManager struct {
	logger logger.Logger
}

// NewUSBManager 创建一个新的USB管理器
func NewUSBManager(logger logger.Logger) *USBManager {
	return &USBManager{
		logger: logger,
	}
}

// GetUSBDevices 获取USB设备信息
func (m *USBManager) GetUSBDevices() ([]USBDevice, error) {
	// 预分配切片容量，避免动态扩容
	usbDevices := make([]USBDevice, 0, 10) // 假设最多有10个USB设备

	// 根据操作系统执行不同的命令
	switch runtime.GOOS {
	case "windows":
		return m.getWindowsUSBDevices()
	case "darwin":
		return m.getDarwinUSBDevices()
	default:
		m.logger.Warn("不支持的操作系统", "os", runtime.GOOS)
		return usbDevices, nil
	}
}

// getWindowsUSBDevices 获取Windows系统的USB设备
func (m *USBManager) getWindowsUSBDevices() ([]USBDevice, error) {
	// 预分配切片容量，避免动态扩容
	usbDevices := make([]USBDevice, 0, 10) // 假设最多有10个USB设备

	// 使用PowerShell获取USB设备信息
	// 使用更高效的命令，减少输出数据量
	cmd := exec.Command("powershell", "-Command", "Get-PnpDevice -Class USB | Where-Object { $_.FriendlyName -ne $null } | Select-Object FriendlyName, InstanceId, Status | ConvertTo-Json")
	output, err := cmd.Output()
	if err != nil {
		return usbDevices, fmt.Errorf("执行PowerShell命令失败: %w", err)
	}

	// 解析输出
	var devices []map[string]interface{}
	if err := json.Unmarshal(output, &devices); err != nil {
		return usbDevices, fmt.Errorf("解析PowerShell输出失败: %w", err)
	}

	// 预分配map容量
	for _, device := range devices {
		name, _ := device["FriendlyName"].(string)
		instanceID, _ := device["InstanceId"].(string)
		status, _ := device["Status"].(string)

		// 从InstanceID中提取VendorID和ProductID
		vendorID := ""
		productID := ""
		if strings.Contains(instanceID, "VID_") && strings.Contains(instanceID, "PID_") {
			// 使用更高效的字符串处理方法
			vidIndex := strings.Index(instanceID, "VID_")
			pidIndex := strings.Index(instanceID, "PID_")

			if vidIndex >= 0 {
				vidStart := vidIndex + 4 // 跳过"VID_"
				vidEnd := strings.IndexByte(instanceID[vidStart:], '&')
				if vidEnd >= 0 {
					vendorID = instanceID[vidStart : vidStart+vidEnd]
				} else {
					vendorID = instanceID[vidStart:]
				}
			}

			if pidIndex >= 0 {
				pidStart := pidIndex + 4 // 跳过"PID_"
				pidEnd := strings.IndexByte(instanceID[pidStart:], '&')
				if pidEnd >= 0 {
					productID = instanceID[pidStart : pidStart+pidEnd]
				} else {
					productID = instanceID[pidStart:]
				}
			}
		}

		usbDevices = append(usbDevices, USBDevice{
			ID:           instanceID,
			Name:         name,
			Manufacturer: vendorID + " " + productID,
			Status:       status,
		})
	}

	return usbDevices, nil
}

// getDarwinUSBDevices 获取macOS系统的USB设备
func (m *USBManager) getDarwinUSBDevices() ([]USBDevice, error) {
	// 预分配切片容量，避免动态扩容
	usbDevices := make([]USBDevice, 0, 10) // 假设最多有10个USB设备

	// 使用system_profiler获取USB设备信息
	// 添加超时控制，避免命令执行时间过长
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "system_profiler", "SPUSBDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return usbDevices, fmt.Errorf("执行system_profiler命令失败: %w", err)
	}

	// 解析输出
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return usbDevices, fmt.Errorf("解析system_profiler输出失败: %w", err)
	}

	// 提取USB设备信息
	if usbData, ok := result["SPUSBDataType"].([]interface{}); ok && len(usbData) > 0 {
		if items, ok := usbData[0].(map[string]interface{})["_items"].([]interface{}); ok {
			// 预分配切片容量
			for _, item := range items {
				itemMap, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				name, _ := itemMap["_name"].(string)
				vendorID, _ := itemMap["vendor_id"].(string)
				productID, _ := itemMap["product_id"].(string)
				serial, _ := itemMap["serial_num"].(string)

				usbDevices = append(usbDevices, USBDevice{
					ID:           serial,
					Name:         name,
					Manufacturer: vendorID + " " + productID,
					Status:       "connected",
				})
			}
		}
	}

	return usbDevices, nil
}
