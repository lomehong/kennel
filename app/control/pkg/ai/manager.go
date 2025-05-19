package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lomehong/kennel/app/control/pkg/ai/mcp"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// AIManager 管理AI相关功能
type AIManager struct {
	logger      sdk.Logger
	config      map[string]interface{}
	mcpClient   *mcp.Client
	initialized bool
	initLock    sync.Mutex
}

// NewAIManager 创建一个新的AI管理器
func NewAIManager(logger sdk.Logger, config map[string]interface{}) *AIManager {
	return &AIManager{
		logger:      logger,
		config:      config,
		initialized: false,
	}
}

// Init 初始化AI管理器
func (m *AIManager) Init(ctx context.Context) error {
	m.initLock.Lock()
	defer m.initLock.Unlock()

	if m.initialized {
		return nil
	}

	// 获取AI配置
	aiConfig, ok := m.config["ai"].(map[string]interface{})
	if !ok {
		aiConfig = make(map[string]interface{})
	}

	// 检查AI是否启用
	enabled, _ := aiConfig["enabled"].(bool)
	if !enabled {
		m.logger.Info("AI功能未启用")
		return nil
	}

	// 初始化 MCP Client
	mcpEnabled := getBoolConfig(aiConfig, "mcp_enabled", false)
	if mcpEnabled {
		mcpServerAddr := getConfigString(aiConfig, "mcp_server_addr", "http://localhost:8080")
		mcpAPIKey := getConfigString(aiConfig, "mcp_api_key", "")

		m.logger.Info("初始化 MCP Client", "server", mcpServerAddr)

		// 创建 MCP Client 配置
		mcpConfig := &mcp.ClientConfig{
			ServerAddr: mcpServerAddr,
			APIKey:     mcpAPIKey,
			Timeout:    30 * time.Second,
			MaxRetries: 3,
		}

		// 创建 MCP Client
		mcpClient, err := mcp.NewClient(mcpConfig, m.logger)
		if err != nil {
			m.logger.Error("初始化 MCP Client 失败", "error", err)
			// 不返回错误，允许继续初始化
		} else {
			m.mcpClient = mcpClient
		}
	}

	m.initialized = true
	m.logger.Info("AI管理器初始化完成")

	return nil
}

// HandleRequest 处理AI请求
func (m *AIManager) HandleRequest(ctx context.Context, query string) (string, error) {
	if !m.initialized {
		if err := m.Init(ctx); err != nil {
			return "", err
		}
	}

	m.logger.Info("处理AI请求", "query", query)

	// 如果MCP Client可用，可以通过它获取远程工具列表
	if m.mcpClient != nil {
		tools, err := m.mcpClient.ListTools(ctx)
		if err != nil {
			m.logger.Warn("获取远程工具列表失败", "error", err)
		} else {
			m.logger.Info("获取远程工具列表成功", "count", len(tools))
			// 这里可以根据需要使用远程工具
		}
	}

	// 简单返回一个固定的响应，实际实现中应该调用AI模型处理请求
	return fmt.Sprintf("收到请求: %s", query), nil
}

// HandleStreamRequest 处理流式AI请求
func (m *AIManager) HandleStreamRequest(ctx context.Context, query string, callback func(string) error) error {
	if !m.initialized {
		if err := m.Init(ctx); err != nil {
			return err
		}
	}

	m.logger.Info("处理流式AI请求", "query", query)

	// 简单返回一个固定的响应，实际实现中应该调用AI模型处理请求
	return callback(fmt.Sprintf("收到流式请求: %s", query))
}

// 辅助函数：从配置中获取字符串值
func getConfigString(config map[string]interface{}, key string, defaultValue string) string {
	if value, ok := config[key].(string); ok {
		return value
	}
	return defaultValue
}

// 辅助函数：从配置中获取布尔值
func getBoolConfig(config map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := config[key].(bool); ok {
		return value
	}
	return defaultValue
}

// Stop 停止AI管理器
func (m *AIManager) Stop() error {
	m.initLock.Lock()
	defer m.initLock.Unlock()

	// 关闭 MCP Client
	if m.mcpClient != nil {
		if err := m.mcpClient.Close(); err != nil {
			m.logger.Warn("关闭 MCP Client 失败", "error", err)
		}
		m.mcpClient = nil
	}

	m.initialized = false
	return nil
}
