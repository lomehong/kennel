package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/lomehong/kennel/pkg/core/plugin"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
)

// ModuleAdapter 是一个适配器，将 core/plugin.Module 接口转换为 plugin.Module 接口
type ModuleAdapter struct {
	Module plugin.Module
}

// Init 实现了 plugin.Module 接口的 Init 方法
func (a *ModuleAdapter) Init(config map[string]interface{}) error {
	// 将配置转换为 ModuleConfig
	moduleConfig := &plugin.ModuleConfig{
		Settings: config,
	}

	// 调用原始模块的 Init 方法
	return a.Module.Init(context.Background(), moduleConfig)
}

// Execute 实现了 plugin.Module 接口的 Execute 方法
func (a *ModuleAdapter) Execute(action string, params map[string]interface{}) (map[string]interface{}, error) {
	// 创建请求
	req := &plugin.Request{
		ID:     fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Action: action,
		Params: params,
	}

	// 调用原始模块的 HandleRequest 方法
	resp, err := a.Module.HandleRequest(context.Background(), req)
	if err != nil {
		return nil, err
	}

	// 检查响应
	if !resp.Success {
		if resp.Error != nil {
			return nil, fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}
		return nil, fmt.Errorf("执行失败: %s", action)
	}

	return resp.Data, nil
}

// Shutdown 实现了 plugin.Module 接口的 Shutdown 方法
func (a *ModuleAdapter) Shutdown() error {
	// 调用原始模块的 Stop 方法
	return a.Module.Stop()
}

// GetInfo 实现了 plugin.Module 接口的 GetInfo 方法
func (a *ModuleAdapter) GetInfo() pluginLib.ModuleInfo {
	// 获取原始模块的信息
	info := a.Module.GetInfo()

	// 转换为 plugin.ModuleInfo
	return pluginLib.ModuleInfo{
		Name:             info.Name,
		Version:          info.Version,
		Description:      info.Description,
		SupportedActions: info.Capabilities,
	}
}

// HandleMessage 实现了 plugin.Module 接口的 HandleMessage 方法
func (a *ModuleAdapter) HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error) {
	// 创建事件
	event := &plugin.Event{
		ID:        messageID,
		Type:      messageType,
		Source:    "host",
		Timestamp: timestamp,
		Data:      payload,
	}

	// 调用原始模块的 HandleEvent 方法
	if err := a.Module.HandleEvent(context.Background(), event); err != nil {
		return nil, err
	}

	// 返回空响应
	return map[string]interface{}{
		"success": true,
	}, nil
}
