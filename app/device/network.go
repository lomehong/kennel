package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/lomehong/kennel/pkg/logger"
	"github.com/shirou/gopsutil/v3/net"
)

// NetworkManager 负责管理网络接口
type NetworkManager struct {
	logger logger.Logger
}

// NewNetworkManager 创建一个新的网络管理器
func NewNetworkManager(logger logger.Logger) *NetworkManager {
	return &NetworkManager{
		logger: logger,
	}
}

// GetNetworkInterfaces 获取网络接口信息
func (m *NetworkManager) GetNetworkInterfaces() ([]NetworkInterface, error) {
	m.logger.Info("获取网络接口信息")

	// 获取网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		m.logger.Error("获取网络接口失败", "error", err)
		return nil, fmt.Errorf("获取网络接口失败: %w", err)
	}

	// 预分配切片容量，避免动态扩容
	networkInterfaces := make([]NetworkInterface, 0, len(interfaces))
	for _, iface := range interfaces {
		// 检查是否是回环接口
		isLoopback := false
		for _, flag := range iface.Flags {
			if flag == "loopback" {
				isLoopback = true
				break
			}
		}
		if !isLoopback {
			// 获取IP地址
			// 预分配切片容量，避免动态扩容
			ipAddresses := make([]string, 0, 4) // 假设每个接口最多有4个IP地址

			// 使用系统命令获取IP地址
			if runtime.GOOS == "windows" {
				cmd := exec.Command("powershell", "-Command", fmt.Sprintf("(Get-NetIPAddress -InterfaceAlias '%s').IPAddress", iface.Name))
				output, err := cmd.Output()
				if err == nil {
					// 解析输出
					lines := strings.Split(string(output), "\r\n")
					for _, line := range lines {
						if line != "" {
							ipAddresses = append(ipAddresses, strings.TrimSpace(line))
						}
					}
				}
			} else {
				// 对于非Windows系统，可以使用其他方法获取IP地址
				// 这里简化处理，实际应用中可能需要更复杂的逻辑
			}

			// 确定状态
			status := "down"
			for _, flag := range iface.Flags {
				if flag == "up" {
					status = "up"
					break
				}
			}

			networkInterfaces = append(networkInterfaces, NetworkInterface{
				Name:       iface.Name,
				MACAddress: iface.HardwareAddr,
				IPAddress:  ipAddresses,
				Status:     status,
			})
		}
	}

	return networkInterfaces, nil
}

// EnableNetwork 启用网络接口
func (m *NetworkManager) EnableNetwork(interfaceName string) (map[string]interface{}, error) {
	m.logger.Info("启用网络接口", "interface", interfaceName)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell", "-Command", fmt.Sprintf("Enable-NetAdapter -Name '%s' -Confirm:$false", interfaceName))
	case "darwin":
		cmd = exec.Command("ifconfig", interfaceName, "up")
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		m.logger.Error("启用网络接口失败", "interface", interfaceName, "error", err)
		return nil, fmt.Errorf("启用网络接口失败: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("已启用网络接口 %s", interfaceName),
	}, nil
}

// DisableNetwork 禁用网络接口
func (m *NetworkManager) DisableNetwork(interfaceName string) (map[string]interface{}, error) {
	m.logger.Info("禁用网络接口", "interface", interfaceName)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell", "-Command", fmt.Sprintf("Disable-NetAdapter -Name '%s' -Confirm:$false", interfaceName))
	case "darwin":
		cmd = exec.Command("ifconfig", interfaceName, "down")
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		m.logger.Error("禁用网络接口失败", "interface", interfaceName, "error", err)
		return nil, fmt.Errorf("禁用网络接口失败: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("已禁用网络接口 %s", interfaceName),
	}, nil
}
