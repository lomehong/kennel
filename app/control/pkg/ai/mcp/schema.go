package mcp

// schema 包提供了与 MCP 协议相关的 JSON Schema 类型定义
// 这是一个自定义实现，用于替代 github.com/mark3labs/mcp-go/mcp/schema

// JSONSchemaProperty 表示 JSON Schema 属性
type JSONSchemaProperty struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// Message 表示 MCP 消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
