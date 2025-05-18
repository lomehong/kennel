package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lomehong/kennel/pkg/core/plugin"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// AssetModule 实现了资产管理模块
type AssetModule struct {
	*sdk.BaseModule
	assetCache *AssetCache
	collector  *Collector
}

// NewAssetModule 创建一个新的资产管理模块
func NewAssetModule() *AssetModule {
	// 创建基础模块
	base := sdk.NewBaseModule(
		"assets",
		"资产管理插件",
		"1.0.0",
		"资产管理模块，用于收集和管理终端资产信息",
	)

	// 创建资产缓存
	assetCache := NewAssetCache()

	// 创建模块
	module := &AssetModule{
		BaseModule: base,
		assetCache: assetCache,
	}

	// 创建收集器
	module.collector = NewCollector(base.Logger)

	return module
}

// Init 初始化模块
func (m *AssetModule) Init(ctx context.Context, config *plugin.ModuleConfig) error {
	// 调用基类初始化
	if err := m.BaseModule.Init(ctx, config); err != nil {
		return err
	}

	m.Logger.Info("初始化资产管理模块")

	// 设置日志级别
	logLevel := sdk.GetConfigString(m.Config, "log_level", "info")
	m.Logger.Debug("设置日志级别", "level", logLevel)

	return nil
}

// Start 启动模块
func (m *AssetModule) Start() error {
	m.Logger.Info("启动资产管理模块")

	// 检查是否启用自动上报
	autoReport := sdk.GetConfigBool(m.Config, "auto_report", false)
	if autoReport {
		m.Logger.Info("启用自动上报")
		// 在实际应用中，这里应该启动一个后台协程定期上报资产信息
	}

	return nil
}

// Stop 停止模块
func (m *AssetModule) Stop() error {
	m.Logger.Info("停止资产管理模块")
	// 在实际应用中，这里应该停止所有后台协程
	return nil
}

// HandleRequest 处理请求
func (m *AssetModule) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	m.Logger.Info("处理请求", "action", req.Action)

	switch req.Action {
	case "collect":
		// 收集资产信息
		assetInfo, err := m.collectAssetInfo()
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "collect_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data:    assetInfo,
		}, nil

	case "report":
		// 上报资产信息
		server := sdk.GetConfigString(m.Config, "report_server", "")
		if server == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "config_error",
					Message: "未配置上报服务器",
				},
			}, nil
		}

		// 收集资产信息
		_, err := m.collectAssetInfo()
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "collect_error",
					Message: err.Error(),
				},
			}, nil
		}

		// 在实际应用中，这里应该将资产信息上报到服务器
		m.Logger.Info("上报资产信息", "server", server)

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"status":  "success",
				"message": "资产信息已上报",
				"server":  server,
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
func (m *AssetModule) HandleEvent(ctx context.Context, event *plugin.Event) error {
	m.Logger.Info("处理事件", "type", event.Type, "source", event.Source)

	switch event.Type {
	case "system.startup":
		// 系统启动事件
		m.Logger.Info("系统启动，执行资产信息收集")
		_, err := m.collectAssetInfo()
		return err

	case "system.shutdown":
		// 系统关闭事件
		m.Logger.Info("系统关闭")
		return nil

	case "asset.scan_request":
		// 资产扫描请求
		m.Logger.Info("收到资产扫描请求")
		_, err := m.collectAssetInfo()
		return err

	default:
		// 忽略其他事件
		return nil
	}
}

// collectAssetInfo 收集资产信息
func (m *AssetModule) collectAssetInfo() (map[string]interface{}, error) {
	// 获取缓存过期时间（默认为1小时）
	cacheExpiration := time.Hour
	if expSec := sdk.GetConfigInt(m.Config, "collect_interval", 3600); expSec > 0 {
		cacheExpiration = time.Duration(expSec) * time.Second
	}

	// 检查缓存是否启用
	cacheEnabled := sdk.GetConfigBool(m.Config, "cache.enabled", true)
	if cacheEnabled {
		// 检查缓存是否有效
		if assetInfo, valid := m.assetCache.GetCachedAsset(cacheExpiration); valid {
			m.Logger.Debug("使用缓存的资产信息")
			return AssetInfoToMap(assetInfo), nil
		}
	}

	// 收集资产信息
	m.Logger.Info("开始收集资产信息")
	assetInfo, err := m.collector.CollectAssetInfo()
	if err != nil {
		return nil, err
	}

	// 更新缓存
	if cacheEnabled {
		m.assetCache.SetCachedAsset(assetInfo)
	}

	// 转换为map
	return AssetInfoToMap(assetInfo), nil
}
