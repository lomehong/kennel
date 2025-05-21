package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	sdk "github.com/lomehong/kennel/pkg/sdk/go"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// ClientV3Config 定义了 MCP Client V3 的配置
type ClientV3Config struct {
	ServerAddr    string        // 服务器地址，例如 http://localhost:8080
	Timeout       time.Duration // 请求超时，默认为 10 秒
	APIKey        string        // API 密钥，用于认证
	MaxRetries    int           // 最大重试次数，默认为 3
	RetryDelay    time.Duration // 重试延迟，默认为 1 秒
	RetryDelayMax time.Duration // 最大重试延迟，默认为 5 秒
	ModelName     string        // 模型名称，例如 "gpt-4"
	StreamMode    bool          // 是否使用流式模式
}

// ClientV3 实现了 MCP Client V3
type ClientV3 struct {
	config     *ClientV3Config
	httpClient *http.Client
	logger     sdk.Logger
	mcpClient  *client.Client
}

// NewClientV3 创建一个新的 MCP Client V3
func NewClientV3(config *ClientV3Config, logger sdk.Logger) (*ClientV3, error) {
	if config == nil {
		config = &ClientV3Config{}
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

	return &ClientV3{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
		mcpClient:  mcpClient,
	}, nil
}

// Start 启动客户端
func (c *ClientV3) Start(ctx context.Context) error {
	c.logger.Debug("启动 MCP 客户端")

	// 启动 MCP 客户端
	if err := c.mcpClient.Start(ctx); err != nil {
		return fmt.Errorf("启动 MCP 客户端失败: %w", err)
	}

	// 初始化 MCP 客户端
	initReq := mcplib.InitializeRequest{}

	// 设置客户端信息
	initReq.Params.ClientInfo = mcplib.Implementation{
		Name:    "Kennel Control Plugin",
		Version: "1.0.0",
	}

	// 设置协议版本和能力
	initReq.Params.ProtocolVersion = "0.4.0"
	initReq.Params.Capabilities = mcplib.ClientCapabilities{}

	_, err := c.mcpClient.Initialize(ctx, initReq)
	if err != nil {
		return fmt.Errorf("初始化 MCP 客户端失败: %w", err)
	}

	return nil
}

// Close 关闭客户端
func (c *ClientV3) Close() error {
	c.logger.Debug("关闭 MCP 客户端")
	return c.mcpClient.Close()
}

// ListTools 列出所有工具
func (c *ClientV3) ListTools(ctx context.Context) ([]ToolInfo, error) {
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
				for name, prop := range tool.InputSchema.Properties {
					// 尝试从属性中提取类型和描述信息
					propMap, ok := prop.(map[string]interface{})
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

// ExecuteTool 执行工具
func (c *ClientV3) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
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

	// 根据结果类型返回不同的值
	if result != nil {
		// 尝试获取文本内容
		if len(result.Content) > 0 {
			// 尝试获取第一个内容元素的文本
			textContent, ok := mcplib.AsTextContent(result.Content[0])
			if ok && textContent != nil {
				return textContent.Text, nil
			}
		}

		// 如果无法获取文本内容，则返回整个结果对象
		return result, nil
	}

	return nil, fmt.Errorf("工具执行返回空结果")
}

// QueryAI 向 AI 发送查询
func (c *ClientV3) QueryAI(ctx context.Context, query string) (string, error) {
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

	// 返回响应
	if result != nil {
		// 将结果序列化为 JSON
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("序列化结果失败: %w", err)
		}

		return string(resultJSON), nil
	}

	return "", fmt.Errorf("AI 查询返回空结果")
}

// QueryAIStream 向 AI 发送查询并返回流式结果
func (c *ClientV3) QueryAIStream(ctx context.Context, query string, callback func(chunk string) error) error {
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

	// 调用回调函数
	if result != nil {
		// 将结果序列化为 JSON
		content, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("解析响应失败: %w", err)
		}

		// 将结果作为字符串返回
		if err := callback(string(content)); err != nil {
			return fmt.Errorf("处理流式响应失败: %w", err)
		}
	} else {
		return fmt.Errorf("AI 查询返回空结果")
	}

	return nil
}

// GetServerInfo 获取服务器信息
func (c *ClientV3) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	c.logger.Debug("获取服务器信息")

	// 创建服务器信息
	info := &ServerInfo{
		Name:        "MCP Server",
		Version:     "1.0.0",
		Description: "MCP 服务器",
	}

	return info, nil
}
