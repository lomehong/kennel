package main

import (
	"fmt"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// Collector 负责收集资产信息
type Collector struct {
	logger logging.Logger
}

// NewCollector 创建一个新的收集器
func NewCollector(logger logging.Logger) *Collector {
	return &Collector{
		logger: logger,
	}
}

// CollectAssetInfo 收集资产信息
func (c *Collector) CollectAssetInfo() (*AssetInfo, error) {
	c.logger.Info("收集资产信息")

	// 获取主机信息
	hostInfo, err := host.Info()
	if err != nil {
		c.logger.Error("获取主机信息失败", "error", err)
		return nil, fmt.Errorf("获取主机信息失败: %w", err)
	}

	// 获取CPU信息
	cpuInfo, err := cpu.Info()
	if err != nil {
		c.logger.Error("获取CPU信息失败", "error", err)
		return nil, fmt.Errorf("获取CPU信息失败: %w", err)
	}

	// 获取内存信息
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		c.logger.Error("获取内存信息失败", "error", err)
		return nil, fmt.Errorf("获取内存信息失败: %w", err)
	}

	// 获取磁盘信息
	parts, err := disk.Partitions(false)
	if err != nil {
		c.logger.Error("获取磁盘分区信息失败", "error", err)
		return nil, fmt.Errorf("获取磁盘分区信息失败: %w", err)
	}

	var diskTotal, diskFree uint64
	for _, part := range parts {
		usage, err := disk.Usage(part.Mountpoint)
		if err != nil {
			c.logger.Warn("获取磁盘使用情况失败", "mountpoint", part.Mountpoint, "error", err)
			continue
		}
		diskTotal += usage.Total
		diskFree += usage.Free
	}

	// 获取网络接口信息
	interfaces, err := net.Interfaces()
	if err != nil {
		c.logger.Error("获取网络接口信息失败", "error", err)
		return nil, fmt.Errorf("获取网络接口信息失败: %w", err)
	}

	// 预分配切片容量，避免动态扩容
	networkCards := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		// 检查接口是否启用且不是回环接口
		isUp := false
		isLoopback := false

		for _, flag := range iface.Flags {
			if flag == "up" {
				isUp = true
			}
			if flag == "loopback" {
				isLoopback = true
			}
		}

		if isUp && !isLoopback {
			networkCards = append(networkCards, iface.Name)
		}
	}

	// 构建资产信息
	assetInfo := &AssetInfo{
		Hostname:     hostInfo.Hostname,
		Platform:     hostInfo.Platform,
		PlatformVer:  hostInfo.PlatformVersion,
		CPUModel:     cpuInfo[0].ModelName,
		CPUCores:     len(cpuInfo),
		MemoryTotal:  memInfo.Total,
		MemoryFree:   memInfo.Free,
		DiskTotal:    diskTotal,
		DiskFree:     diskFree,
		NetworkCards: networkCards,
		CollectTime:  time.Now(),
	}

	return assetInfo, nil
}
