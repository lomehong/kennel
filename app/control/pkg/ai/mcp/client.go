package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// ClientConfig 定义了 MCP Client 的配置
type ClientConfig struct {
	ServerAddr    string        // 服务器地址，例如 http://localhost:8080
	Timeout       time.Duration // 请求超时，默认为 10 秒
	APIKey        string        // API 密钥，用于认证
	MaxRetries    int           // 最大重试次数，默认为 3
	RetryDelay    time.Duration // 重试延迟，默认为 1 秒
	RetryDelayMax time.Duration // 最大重试延迟，默认为 5 秒
}

// Client 实现了 MCP Client
type Client struct {
	config     *ClientConfig
	httpClient *http.Client
	logger     sdk.Logger
}

// NewClient 创建一个新的 MCP Client
func NewClient(config *ClientConfig, logger sdk.Logger) (*Client, error) {
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

	// 创建 HTTP 客户端
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

// ListTools 列出所有工具
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	url := fmt.Sprintf("%s/tools", c.config.ServerAddr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 添加 API 密钥
	if c.config.APIKey != "" {
		req.Header.Set("X-API-Key", c.config.APIKey)
	}

	// 发送请求
	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("服务器返回错误: %s, %s", resp.Status, string(body))
	}

	// 解析响应
	var tools []ToolInfo
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return tools, nil
}

// GetTool 获取工具信息
func (c *Client) GetTool(ctx context.Context, name string) (*ToolInfo, error) {
	url := fmt.Sprintf("%s/tools/%s", c.config.ServerAddr, name)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 添加 API 密钥
	if c.config.APIKey != "" {
		req.Header.Set("X-API-Key", c.config.APIKey)
	}

	// 发送请求
	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("工具 %s 不存在", name)
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("服务器返回错误: %s, %s", resp.Status, string(body))
	}

	// 解析响应
	var tool ToolInfo
	if err := json.NewDecoder(resp.Body).Decode(&tool); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &tool, nil
}

// ExecuteTool 执行工具
func (c *Client) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	url := fmt.Sprintf("%s/tools/%s/execute", c.config.ServerAddr, name)

	// 编码参数
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("编码参数失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("X-API-Key", c.config.APIKey)
	}

	// 发送请求
	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("工具 %s 不存在", name)
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("服务器返回错误: %s, %s", resp.Status, string(body))
	}

	// 解析响应
	var result struct {
		Result interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return result.Result, nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	// 关闭 HTTP 客户端
	c.httpClient.CloseIdleConnections()
	return nil
}

// doWithRetry 发送请求并重试
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	retryDelay := c.config.RetryDelay

	for i := 0; i <= c.config.MaxRetries; i++ {
		// 克隆请求，因为请求体可能已经被读取
		reqClone := req.Clone(req.Context())
		if req.Body != nil {
			body, _ := io.ReadAll(req.Body)
			reqClone.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		resp, err = c.httpClient.Do(reqClone)
		if err == nil && resp.StatusCode < 500 {
			// 成功或客户端错误，不重试
			return resp, nil
		}

		if i < c.config.MaxRetries {
			// 记录错误并重试
			if err != nil {
				c.logger.Warn("请求失败，将重试", "error", err, "retry", i+1)
			} else {
				c.logger.Warn("服务器错误，将重试", "status", resp.Status, "retry", i+1)
				resp.Body.Close()
			}

			// 等待重试
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(retryDelay):
				// 指数退避
				retryDelay = time.Duration(float64(retryDelay) * 1.5)
				if retryDelay > c.config.RetryDelayMax {
					retryDelay = c.config.RetryDelayMax
				}
			}
		}
	}

	// 所有重试都失败
	return resp, err
}
