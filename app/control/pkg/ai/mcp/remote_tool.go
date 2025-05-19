package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// RemoteTool 实现了远程工具适配器
type RemoteTool struct {
	client *Client
	name   string
	info   *ToolInfo
}

// NewRemoteTool 创建一个新的远程工具适配器
func NewRemoteTool(client *Client, name string) (*RemoteTool, error) {
	// 获取工具信息
	ctx := context.Background()
	info, err := client.GetTool(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("获取工具信息失败: %w", err)
	}

	return &RemoteTool{
		client: client,
		name:   name,
		info:   info,
	}, nil
}

// GetName 返回工具的名称
func (t *RemoteTool) GetName() string {
	return t.name
}

// GetDescription 返回工具的描述
func (t *RemoteTool) GetDescription() string {
	return t.info.Description
}

// GetParameters 返回工具的参数定义
func (t *RemoteTool) GetParameters() map[string]Parameter {
	return t.info.Parameters
}

// Execute 执行工具
func (t *RemoteTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 执行工具
	result, err := t.client.ExecuteTool(ctx, t.name, params)
	if err != nil {
		return nil, fmt.Errorf("执行工具失败: %w", err)
	}

	return result, nil
}

// Run 返回一个可执行的函数，用于与AI框架集成
func (t *RemoteTool) Run() func(ctx context.Context, argumentsInJSON string) (string, error) {
	return func(ctx context.Context, argumentsInJSON string) (string, error) {
		// 解析参数
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", fmt.Errorf("解析参数失败: %w", err)
		}

		// 执行工具
		result, err := t.Execute(ctx, params)
		if err != nil {
			return "", err
		}

		// 序列化结果
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("序列化结果失败: %w", err)
		}

		return string(resultJSON), nil
	}
}
