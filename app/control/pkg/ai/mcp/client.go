package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// ClientConfig 定义了 MCP Client 的配置
type ClientConfig struct {
	ServerAddr    string        // 服务器地址，例如 http://localhost:8080
	Timeout       time.Duration // 请求超时，默认为 10 秒
	APIKey        string        // API 密钥，用于认证
	MaxRetries    int           // 最大重试次数，默认为 3
	RetryDelay    time.Duration // 重试延迟，默认为 1 秒
	RetryDelayMax time.Duration // 最大重试延迟，默认为 5 秒
	ModelName     string        // 模型名称，例如 "gpt-4"
	StreamMode    bool          // 是否使用流式模式
}

// Client 实现了 MCP Client
type Client struct {
	config     *ClientConfig
	httpClient *http.Client
	logger     logging.Logger
	mcpClient  *client.Client
}

// NewClient 创建一个新的 MCP Client
func NewClient(config *ClientConfig, logger logging.Logger) (*Client, error) {
	if config == nil {
		config = &ClientConfig{}
	}

	// 设置默认值
	if config.ServerAddr == "" {
		return nil, fmt.Errorf("服务器地址不能为空")
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.RetryDelayMax == 0 {
		config.RetryDelayMax = 5 * time.Second
	}
	if config.ModelName == "" {
		config.ModelName = "gpt-4"
	}

	// 创建 HTTP 客户端
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// 创建 MCP 客户端选项
	opts := []transport.ClientOption{
		transport.WithHTTPClient(httpClient),
	}

	// 如果提供了 API 密钥，添加认证头
	if config.APIKey != "" {
		headers := map[string]string{
			"X-API-Key": config.APIKey,
		}
		opts = append(opts, transport.WithHeaders(headers))
	}

	// 创建 MCP 客户端
	mcpClient, err := client.NewSSEMCPClient(config.ServerAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("创建 MCP 客户端失败: %w", err)
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
		mcpClient:  mcpClient,
	}, nil
}

// ListTools 列出所有工具
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	c.logger.Debug("获取工具列表")

	// 创建请求
	req := mcplib.ListToolsRequest{}

	// 使用 MCP 客户端获取工具列表
	result, err := c.mcpClient.ListTools(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("获取工具列表失败: %w", err)
	}

	// 转换为我们的 ToolInfo 类型
	tools := make([]ToolInfo, 0)
	if result != nil && result.Tools != nil {
		for _, tool := range result.Tools {
			params := make(map[string]Parameter)
			// 处理工具的输入模式
			if len(tool.InputSchema.Properties) > 0 {
				for name, propInterface := range tool.InputSchema.Properties {
					// 尝试从属性中提取类型和描述信息
					propMap, ok := propInterface.(map[string]interface{})
					if !ok {
						continue
					}

					// 提取类型
					typeVal, _ := propMap["type"].(string)

					// 提取描述
					descVal, _ := propMap["description"].(string)

					// 检查是否必需
					required := false
					for _, reqName := range tool.InputSchema.Required {
						if reqName == name {
							required = true
							break
						}
					}

					params[name] = Parameter{
						Type:        typeVal,
						Description: descVal,
						Required:    required,
					}
				}
			}

			tools = append(tools, ToolInfo{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			})
		}
	}

	return tools, nil
}

// GetTool 获取工具信息
func (c *Client) GetTool(ctx context.Context, name string) (*ToolInfo, error) {
	c.logger.Debug("获取工具信息", "name", name)

	// 获取所有工具
	tools, err := c.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工具列表失败: %w", err)
	}

	// 查找指定名称的工具
	for _, tool := range tools {
		if tool.Name == name {
			return &tool, nil
		}
	}

	return nil, fmt.Errorf("未找到工具: %s", name)
}

// ExecuteTool 执行工具
func (c *Client) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	c.logger.Debug("执行工具", "name", name, "params", params)

	// 创建请求
	req := mcplib.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = params

	// 使用 MCP 客户端执行工具
	result, err := c.mcpClient.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("执行工具失败: %w", err)
	}

	// 将结果序列化为 JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("序列化结果失败: %w", err)
	}

	// 解析 JSON 结果
	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultJSON, &resultMap); err != nil {
		return nil, fmt.Errorf("解析结果失败: %w", err)
	}

	return resultMap, nil
}

// ExecuteToolStream 执行工具并返回流式结果
func (c *Client) ExecuteToolStream(ctx context.Context, name string, params map[string]interface{}, callback func(chunk StreamChunk) error) error {
	c.logger.Debug("执行工具（流式）", "name", name, "params", params)

	// 创建请求
	req := mcplib.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = params

	// 使用 MCP 客户端执行工具
	result, err := c.mcpClient.CallTool(ctx, req)
	if err != nil {
		return fmt.Errorf("执行工具失败: %w", err)
	}

	// 将结果序列化为 JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("序列化结果失败: %w", err)
	}

	// 创建流式块
	chunk := StreamChunk{
		Type:    "text",
		Content: string(resultJSON),
	}

	// 调用回调函数
	if err := callback(chunk); err != nil {
		return fmt.Errorf("处理流式响应失败: %w", err)
	}

	return nil
}

// QueryAI 向 AI 发送查询
func (c *Client) QueryAI(ctx context.Context, query string) (string, error) {
	c.logger.Debug("向 AI 发送查询", "query", query)

	// 创建请求
	req := mcplib.CompleteRequest{}

	// 添加用户消息
	userMessage := mcplib.PromptMessage{
		Role:    mcplib.RoleUser,
		Content: mcplib.NewTextContent(query),
	}

	// 设置请求参数
	paramsData := struct {
		Messages []mcplib.PromptMessage `json:"messages"`
		Model    string                 `json:"model,omitempty"`
	}{
		Messages: []mcplib.PromptMessage{userMessage},
		Model:    c.config.ModelName,
	}

	// 将参数转换为 JSON，然后设置到 Ref 字段
	paramsJSON, err := json.Marshal(paramsData)
	if err != nil {
		return "", fmt.Errorf("序列化参数失败: %w", err)
	}

	var paramsMap map[string]interface{}
	if err := json.Unmarshal(paramsJSON, &paramsMap); err != nil {
		return "", fmt.Errorf("反序列化参数失败: %w", err)
	}

	req.Params.Ref = paramsMap

	// 使用 MCP 客户端发送查询
	result, err := c.mcpClient.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("发送查询失败: %w", err)
	}

	// 将结果序列化为 JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// QueryAIStream 向 AI 发送查询并返回流式结果
func (c *Client) QueryAIStream(ctx context.Context, query string, callback func(chunk string) error) error {
	c.logger.Debug("向 AI 发送查询（流式）", "query", query)

	// 创建请求
	req := mcplib.CompleteRequest{}

	// 添加用户消息
	userMessage := mcplib.PromptMessage{
		Role:    mcplib.RoleUser,
		Content: mcplib.NewTextContent(query),
	}

	// 设置请求参数
	paramsData := struct {
		Messages []mcplib.PromptMessage `json:"messages"`
		Model    string                 `json:"model,omitempty"`
		Stream   bool                   `json:"stream,omitempty"`
	}{
		Messages: []mcplib.PromptMessage{userMessage},
		Model:    c.config.ModelName,
		Stream:   true,
	}

	// 将参数转换为 JSON，然后设置到 Ref 字段
	paramsJSON, err := json.Marshal(paramsData)
	if err != nil {
		return fmt.Errorf("序列化参数失败: %w", err)
	}

	var paramsMap map[string]interface{}
	if err := json.Unmarshal(paramsJSON, &paramsMap); err != nil {
		return fmt.Errorf("反序列化参数失败: %w", err)
	}

	req.Params.Ref = paramsMap

	// 使用 MCP 客户端发送查询
	// 注意：当前版本的 MCP Go 库不直接支持流式完成
	// 这里我们使用非流式方法，然后模拟流式响应
	result, err := c.mcpClient.Complete(ctx, req)
	if err != nil {
		return fmt.Errorf("发送查询失败: %w", err)
	}

	// 将结果序列化为 JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("序列化结果失败: %w", err)
	}

	// 将结果作为字符串返回
	if err := callback(string(resultJSON)); err != nil {
		return fmt.Errorf("处理流式响应失败: %w", err)
	}

	return nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	// 关闭 HTTP 客户端
	c.httpClient.CloseIdleConnections()
	return nil
}

// GetServerInfo 获取服务器信息
func (c *Client) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	c.logger.Debug("获取服务器信息")

	// 创建一个默认的服务器信息
	// 注意：当前版本的 MCP Go 库不支持获取服务器信息
	// 这里我们返回一个硬编码的默认值
	result := &ServerInfo{
		Name:        "MCP Server",
		Version:     "1.0.0",
		Description: "MCP API Server",
	}

	return result, nil
}
