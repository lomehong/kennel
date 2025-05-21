package mcp

import (
	"encoding/json"
	"time"
)

// MCPToolInfo 表示 MCP 工具信息
type MCPToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Returns     map[string]interface{} `json:"returns,omitempty"`
}

// MCPRequest 表示 MCP 请求
type MCPRequest struct {
	ID      string                 `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	JSONRPC string                 `json:"jsonrpc"`
}

// MCPResponse 表示 MCP 响应
type MCPResponse struct {
	ID      string                 `json:"id"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Error   *MCPError              `json:"error,omitempty"`
	JSONRPC string                 `json:"jsonrpc"`
}

// MCPError 表示 MCP 错误
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPNotification 表示 MCP 通知
type MCPNotification struct {
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	JSONRPC string                 `json:"jsonrpc"`
}

// ToolCallRequest 表示工具调用请求
type ToolCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolCallResponse 表示工具调用响应
type ToolCallResponse struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

// TextToolCallResponse 表示文本工具调用响应
type TextToolCallResponse struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// ErrorToolCallResponse 表示错误工具调用响应
type ErrorToolCallResponse struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// JSONToolCallResponse 表示 JSON 工具调用响应
type JSONToolCallResponse struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

// StreamChunk 表示流式响应的一个数据块
type StreamChunk struct {
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

// ResourceInfo 表示资源信息
type ResourceInfo struct {
	URI         string `json:"uri"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mime_type,omitempty"`
}

// ResourceContent 表示资源内容
type ResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mime_type"`
	Content  string `json:"content"`
}

// PromptInfo 表示提示信息
type PromptInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Arguments   map[string]interface{} `json:"arguments,omitempty"`
}

// PromptMessage 表示提示消息
type PromptMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ServerInfo 表示服务器信息
type ServerInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// SessionInfo 表示会话信息
type SessionInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MCPCapabilities 表示 MCP 服务器能力
type MCPCapabilities struct {
	Resources bool `json:"resources"`
	Tools     bool `json:"tools"`
	Prompts   bool `json:"prompts"`
	Sessions  bool `json:"sessions"`
}
