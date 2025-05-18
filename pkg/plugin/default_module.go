package plugin

import (
	"fmt"
)

// DefaultModule 提供了Module接口的默认实现
type DefaultModule struct {
	// 模块名称
	Name string

	// 模块版本
	Version string

	// 模块描述
	Description string

	// 支持的操作列表
	SupportedActions []string

	// 配置
	Config map[string]interface{}
}

// NewDefaultModule 创建一个新的默认模块
func NewDefaultModule(name, version, description string, supportedActions []string) *DefaultModule {
	return &DefaultModule{
		Name:             name,
		Version:          version,
		Description:      description,
		SupportedActions: supportedActions,
		Config:           make(map[string]interface{}),
	}
}

// Init 初始化模块
func (m *DefaultModule) Init(config map[string]interface{}) error {
	m.Config = config
	return nil
}

// Execute 执行模块操作
func (m *DefaultModule) Execute(action string, params map[string]interface{}) (map[string]interface{}, error) {
	// 检查操作是否支持
	supported := false
	for _, a := range m.SupportedActions {
		if a == action {
			supported = true
			break
		}
	}

	if !supported {
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}

	// 默认实现只返回一个空的结果
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("执行操作 %s 成功", action),
	}, nil
}

// Shutdown 关闭模块
func (m *DefaultModule) Shutdown() error {
	return nil
}

// GetInfo 获取模块信息
func (m *DefaultModule) GetInfo() ModuleInfo {
	return ModuleInfo{
		Name:             m.Name,
		Version:          m.Version,
		Description:      m.Description,
		SupportedActions: m.SupportedActions,
	}
}

// HandleMessage 处理消息
func (m *DefaultModule) HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error) {
	// 默认实现只返回一个空的结果
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("处理消息 %s 成功", messageType),
		"message_id": messageID,
	}, nil
}
