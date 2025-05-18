package main

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/logger"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
	"github.com/lomehong/kennel/pkg/utils"
)

// AssetModule 实现了资产管理模块
type AssetModule struct {
	logger     logger.Logger
	config     map[string]interface{}
	assetCache *AssetCache
	collector  *Collector
}

// NewAssetModule 创建一个新的资产管理模块
func NewAssetModule() pluginLib.Module {
	// 创建日志器
	log := logger.NewLogger("asset-module", hclog.Info)

	// 创建资产缓存
	assetCache := NewAssetCache()

	// 创建模块
	module := &AssetModule{
		logger:     log,
		config:     make(map[string]interface{}),
		assetCache: assetCache,
	}

	// 创建收集器
	module.collector = NewCollector(log)

	return module
}

// Init 初始化模块
func (m *AssetModule) Init(config map[string]interface{}) error {
	m.logger.Info("初始化资产管理模块")
	m.config = config
	return nil
}

// Execute 执行模块操作
func (m *AssetModule) Execute(action string, params map[string]interface{}) (map[string]interface{}, error) {
	m.logger.Info("执行操作", "action", action)

	switch action {
	case "collect":
		return m.collectAssetInfo()
	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
}

// Shutdown 关闭模块
func (m *AssetModule) Shutdown() error {
	m.logger.Info("关闭资产管理模块")
	return nil
}

// GetInfo 获取模块信息
func (m *AssetModule) GetInfo() pluginLib.ModuleInfo {
	return pluginLib.ModuleInfo{
		Name:             "assets",
		Version:          "0.1.0",
		Description:      "资产管理模块，用于收集和管理终端资产信息",
		SupportedActions: []string{"collect"},
	}
}

// HandleMessage 处理消息
func (m *AssetModule) HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error) {
	m.logger.Info("处理消息", "type", messageType, "id", messageID)

	switch messageType {
	case "scan_request":
		// 处理扫描请求
		return m.collectAssetInfo()
	case "update_request":
		// 处理更新请求
		return map[string]interface{}{
			"status":  "success",
			"message": "资产信息已更新",
		}, nil
	default:
		return nil, fmt.Errorf("不支持的消息类型: %s", messageType)
	}
}

// collectAssetInfo 收集资产信息
func (m *AssetModule) collectAssetInfo() (map[string]interface{}, error) {
	// 获取缓存过期时间（默认为1小时）
	cacheExpiration := time.Hour
	if expStr := utils.GetString(m.config, "collect_interval", ""); expStr != "" {
		if exp, err := time.ParseDuration(expStr); err == nil {
			cacheExpiration = exp
		}
	} else if expSec := utils.GetFloat(m.config, "collect_interval", 0); expSec > 0 {
		cacheExpiration = time.Duration(expSec) * time.Second
	}

	// 检查缓存是否有效
	if assetInfo, valid := m.assetCache.GetCachedAsset(cacheExpiration); valid {
		m.logger.Debug("使用缓存的资产信息")
		return AssetInfoToMap(assetInfo), nil
	}

	// 收集资产信息
	assetInfo, err := m.collector.CollectAssetInfo()
	if err != nil {
		return nil, err
	}

	// 更新缓存
	m.assetCache.SetCachedAsset(assetInfo)

	// 转换为map
	return AssetInfoToMap(assetInfo), nil
}
