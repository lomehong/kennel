package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Parameter 定义了工具参数
type Parameter struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// ToolInfo 定义了工具信息
type ToolInfo struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Parameters  map[string]Parameter `json:"parameters"`
}

// Tool 定义了工具的接口
type Tool interface {
	// GetName 返回工具的名称
	GetName() string

	// GetDescription 返回工具的描述
	GetDescription() string

	// GetParameters 返回工具的参数定义
	GetParameters() map[string]Parameter

	// Execute 执行工具，返回结果或错误
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// BaseTool 提供了 Tool 接口的基础实现
type BaseTool struct {
	Name        string
	Description string
	Parameters  map[string]Parameter
	Handler     func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// GetName 返回工具的名称
func (t *BaseTool) GetName() string {
	return t.Name
}

// GetDescription 返回工具的描述
func (t *BaseTool) GetDescription() string {
	return t.Description
}

// GetParameters 返回工具的参数定义
func (t *BaseTool) GetParameters() map[string]Parameter {
	return t.Parameters
}

// Execute 执行工具，返回结果或错误
func (t *BaseTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 验证参数
	if err := validateParams(params, t.Parameters); err != nil {
		return nil, err
	}

	// 调用处理函数
	return t.Handler(ctx, params)
}

// validateParams 验证参数是否符合定义
func validateParams(params map[string]interface{}, paramDefs map[string]Parameter) error {
	// 检查必需参数
	for name, def := range paramDefs {
		if def.Required {
			if _, ok := params[name]; !ok {
				return fmt.Errorf("缺少必需参数: %s", name)
			}
		}
	}

	// 检查参数类型
	for name, value := range params {
		def, ok := paramDefs[name]
		if !ok {
			// 忽略未定义的参数
			continue
		}

		// 检查类型
		switch def.Type {
		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("参数 %s 类型错误，期望 string，实际 %T", name, value)
			}
		case "number":
			// JSON 解析可能将数字解析为 float64
			if _, ok := value.(float64); !ok {
				return fmt.Errorf("参数 %s 类型错误，期望 number，实际 %T", name, value)
			}
		case "boolean":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("参数 %s 类型错误，期望 boolean，实际 %T", name, value)
			}
		case "array":
			if _, ok := value.([]interface{}); !ok {
				return fmt.Errorf("参数 %s 类型错误，期望 array，实际 %T", name, value)
			}
		case "object":
			if _, ok := value.(map[string]interface{}); !ok {
				return fmt.Errorf("参数 %s 类型错误，期望 object，实际 %T", name, value)
			}
		}
	}

	return nil
}

// NewTool 创建一个新的工具
func NewTool(name, description string, parameters map[string]Parameter, handler func(ctx context.Context, params map[string]interface{}) (interface{}, error)) Tool {
	return &BaseTool{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		Handler:     handler,
	}
}

// ToToolInfo 将 Tool 转换为 ToolInfo
func ToToolInfo(tool Tool) ToolInfo {
	return ToolInfo{
		Name:        tool.GetName(),
		Description: tool.GetDescription(),
		Parameters:  tool.GetParameters(),
	}
}

// ToolToJSON 将工具转换为 JSON 字符串
func ToolToJSON(tool Tool) (string, error) {
	info := ToToolInfo(tool)
	data, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
