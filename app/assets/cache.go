package main

import (
	"sync"
	"time"
)

// AssetInfo 表示资产信息
type AssetInfo struct {
	Hostname     string    `json:"hostname"`
	Platform     string    `json:"platform"`
	PlatformVer  string    `json:"platform_version"`
	CPUModel     string    `json:"cpu_model"`
	CPUCores     int       `json:"cpu_cores"`
	MemoryTotal  uint64    `json:"memory_total"`
	MemoryFree   uint64    `json:"memory_free"`
	DiskTotal    uint64    `json:"disk_total"`
	DiskFree     uint64    `json:"disk_free"`
	NetworkCards []string  `json:"network_cards"`
	CollectTime  time.Time `json:"collect_time"`
}

// AssetCache 提供资产信息的缓存功能
type AssetCache struct {
	mu        sync.RWMutex
	assetInfo *AssetInfo
}

// NewAssetCache 创建一个新的资产缓存
func NewAssetCache() *AssetCache {
	return &AssetCache{}
}

// GetCachedAsset 获取缓存的资产信息，如果缓存有效则返回true
func (c *AssetCache) GetCachedAsset(expiration time.Duration) (*AssetInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.assetInfo == nil {
		return nil, false
	}

	// 检查缓存是否过期
	if time.Since(c.assetInfo.CollectTime) > expiration {
		return nil, false
	}

	return c.assetInfo, true
}

// SetCachedAsset 设置缓存的资产信息
func (c *AssetCache) SetCachedAsset(assetInfo *AssetInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.assetInfo = assetInfo
}

// AssetInfoToMap 将AssetInfo转换为map
func AssetInfoToMap(info *AssetInfo) map[string]interface{} {
	return map[string]interface{}{
		"hostname":         info.Hostname,
		"platform":         info.Platform,
		"platform_version": info.PlatformVer,
		"cpu": map[string]interface{}{
			"model": info.CPUModel,
			"cores": info.CPUCores,
		},
		"memory": map[string]interface{}{
			"total": info.MemoryTotal,
			"free":  info.MemoryFree,
		},
		"disk": map[string]interface{}{
			"total": info.DiskTotal,
			"free":  info.DiskFree,
		},
		"network": map[string]interface{}{
			"cards": info.NetworkCards,
		},
		"collect_time": info.CollectTime.Format(time.RFC3339),
	}
}
