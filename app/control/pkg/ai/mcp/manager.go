package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// ManagerConfig 定义了 MCP 管理器的配置
type ManagerConfig struct {
	Enabled       bool              // 是否启用 MCP
	ServerAddr    string            // 服务器地址，例如 http://localhost:8080
	APIKey        string            // API 密钥，用于认证
	ModelName     string            // 模型名称，例如 "gpt-4"
	Timeout       time.Duration     // 请求超时，默认为 10 秒
	MaxRetries    int               // 最大重试次数，默认为 3
	RetryDelay    time.Duration     // 重试延迟，默认为 1 秒
	RetryDelayMax time.Duration     // 最大重试延迟，默认为 5 秒
	Tools         map[string]string // 工具名称到描述的映射
}

// Manager 实现了 MCP 管理器
type Manager struct {
	config    *ManagerConfig
	logger    logging.Logger
	client    *Client
	mutex     sync.RWMutex
	tools     map[string]ToolInfo
	isRunning bool
}

// NewManager 创建一个新的 MCP 管理器
func NewManager(config *ManagerConfig, logger logging.Logger) (*Manager, error) {
	if config == nil {
		config = &ManagerConfig{}
	}

	// 设置默认值
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

	return &Manager{
		config:    config,
		logger:    logger,
		tools:     make(map[string]ToolInfo),
		isRunning: false,
	}, nil
}

// Start 启动 MCP 管理器
func (m *Manager) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.config.Enabled {
		m.logger.Info("MCP 管理器已禁用")
		return nil
	}

	if m.isRunning {
		m.logger.Info("MCP 管理器已经在运行")
		return nil
	}

	// 创建客户端配置
	clientConfig := &ClientConfig{
		ServerAddr:    m.config.ServerAddr,
		Timeout:       m.config.Timeout,
		APIKey:        m.config.APIKey,
		MaxRetries:    m.config.MaxRetries,
		RetryDelay:    m.config.RetryDelay,
		RetryDelayMax: m.config.RetryDelayMax,
		ModelName:     m.config.ModelName,
	}

	// 创建客户端
	client, err := NewClient(clientConfig, m.logger)
	if err != nil {
		return fmt.Errorf("创建 MCP 客户端失败: %w", err)
	}

	m.client = client
	m.isRunning = true

	// 获取工具列表
	go func() {
		tools, err := m.client.ListTools(ctx)
		if err != nil {
			m.logger.Error("获取工具列表失败", "error", err)
			return
		}

		m.mutex.Lock()
		defer m.mutex.Unlock()

		for _, tool := range tools {
			m.tools[tool.Name] = tool
		}

		m.logger.Info("已获取工具列表", "count", len(tools))
	}()

	m.logger.Info("MCP 管理器已启动", "server", m.config.ServerAddr)
	return nil
}

// Stop 停止 MCP 管理器
func (m *Manager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isRunning {
		m.logger.Info("MCP 管理器未运行")
		return nil
	}

	if m.client != nil {
		if err := m.client.Close(); err != nil {
			m.logger.Warn("关闭 MCP 客户端失败", "error", err)
		}
		m.client = nil
	}

	m.isRunning = false
	m.logger.Info("MCP 管理器已停止")
	return nil
}

// IsRunning 检查 MCP 管理器是否正在运行
func (m *Manager) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.isRunning
}

// ExecuteTool 执行工具
func (m *Manager) ExecuteTool(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.isRunning {
		return nil, fmt.Errorf("MCP 管理器未运行")
	}

	if m.client != nil {
		return m.client.ExecuteTool(ctx, name, params)
	}

	return nil, fmt.Errorf("MCP 客户端未初始化")
}

// QueryAI 向 AI 发送查询
func (m *Manager) QueryAI(ctx context.Context, query string) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.isRunning {
		return "", fmt.Errorf("MCP 管理器未运行")
	}

	if m.client != nil {
		return m.client.QueryAI(ctx, query)
	}

	return "", fmt.Errorf("MCP 客户端未初始化")
}

// QueryAIStream 向 AI 发送查询并返回流式结果
func (m *Manager) QueryAIStream(ctx context.Context, query string, callback func(chunk string) error) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.isRunning {
		return fmt.Errorf("MCP 管理器未运行")
	}

	if m.client != nil {
		return m.client.QueryAIStream(ctx, query, callback)
	}

	return fmt.Errorf("MCP 客户端未初始化")
}

// GetTools 获取工具列表
func (m *Manager) GetTools() map[string]ToolInfo {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	tools := make(map[string]ToolInfo, len(m.tools))
	for name, tool := range m.tools {
		tools[name] = tool
	}

	return tools
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *ManagerConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config
}

// SetConfig 设置配置
func (m *Manager) SetConfig(config *ManagerConfig) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config = config
}
