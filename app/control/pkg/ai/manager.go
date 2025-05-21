package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lomehong/kennel/app/control/pkg/ai/mcp"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// AIProvider 表示 AI 提供者类型
type AIProvider string

const (
	// AIProviderOpenAI 表示 OpenAI 提供者
	AIProviderOpenAI AIProvider = "openai"
	// AIProviderArk 表示 Ark 提供者
	AIProviderArk AIProvider = "ark"
	// AIProviderMCP 表示 MCP 提供者
	AIProviderMCP AIProvider = "mcp"
)

// AIManager 管理AI相关功能
type AIManager struct {
	logger      sdk.Logger
	config      map[string]interface{}
	mcpClient   *mcp.SimpleClient
	mcpManager  *mcp.Manager
	provider    AIProvider
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

	// 获取 AI 提供者
	providerStr := getConfigString(aiConfig, "provider", "openai")
	switch providerStr {
	case "openai":
		m.provider = AIProviderOpenAI
	case "ark":
		m.provider = AIProviderArk
	case "mcp":
		m.provider = AIProviderMCP
	default:
		m.provider = AIProviderOpenAI
	}

	// 初始化 MCP
	mcpEnabled := getBoolConfig(aiConfig, "mcp_enabled", false)
	if mcpEnabled && m.provider == AIProviderMCP {
		mcpServerAddr := getConfigString(aiConfig, "mcp_server_addr", "http://localhost:8080")
		mcpAPIKey := getConfigString(aiConfig, "mcp_api_key", "")
		mcpModelName := getConfigString(aiConfig, "mcp_model_name", "gpt-4")

		m.logger.Info("初始化 MCP 管理器", "server", mcpServerAddr, "model", mcpModelName)

		// 创建 MCP 管理器配置
		mcpConfig := &mcp.ManagerConfig{
			Enabled:    true,
			ServerAddr: mcpServerAddr,
			APIKey:     mcpAPIKey,
			ModelName:  mcpModelName,
			Timeout:    30 * time.Second,
			MaxRetries: 3,
		}

		// 创建 MCP 管理器
		mcpManager, err := mcp.NewManager(mcpConfig, m.logger)
		if err != nil {
			m.logger.Error("创建 MCP 管理器失败", "error", err)
			// 不返回错误，允许继续初始化
		} else {
			m.mcpManager = mcpManager

			// 启动 MCP 管理器
			if err := m.mcpManager.Start(ctx); err != nil {
				m.logger.Error("启动 MCP 管理器失败", "error", err)
				// 不返回错误，允许继续初始化
			}
		}
	} else if mcpEnabled {
		// 如果启用了 MCP 但不是主要提供者，则使用 SimpleClient
		mcpServerAddr := getConfigString(aiConfig, "mcp_server_addr", "http://localhost:8080")
		mcpAPIKey := getConfigString(aiConfig, "mcp_api_key", "")

		m.logger.Info("初始化 MCP 客户端", "server", mcpServerAddr)

		// 创建 MCP 客户端配置
		mcpConfig := &mcp.SimpleClientConfig{
			ServerAddr: mcpServerAddr,
			APIKey:     mcpAPIKey,
			Timeout:    30 * time.Second,
			MaxRetries: 3,
		}

		// 创建 MCP 客户端
		mcpClient, err := mcp.NewSimpleClient(mcpConfig, m.logger)
		if err != nil {
			m.logger.Error("初始化 MCP 客户端失败", "error", err)
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

	m.logger.Info("处理AI请求", "query", query, "provider", m.provider)

	// 根据提供者选择不同的处理方式
	switch m.provider {
	case AIProviderMCP:
		// 使用 MCP 管理器处理请求
		if m.mcpManager != nil && m.mcpManager.IsRunning() {
			response, err := m.mcpManager.QueryAI(ctx, query)
			if err != nil {
				m.logger.Error("MCP 查询失败", "error", err)
				return "", fmt.Errorf("MCP 查询失败: %w", err)
			}
			return response, nil
		} else if m.mcpClient != nil {
			// 使用 MCP 客户端处理请求
			response, err := m.mcpClient.QueryAI(ctx, query)
			if err != nil {
				m.logger.Error("MCP 客户端查询失败", "error", err)
				return "", fmt.Errorf("MCP 客户端查询失败: %w", err)
			}
			return response, nil
		}
		return "", fmt.Errorf("MCP 提供者未初始化")

	case AIProviderOpenAI:
		// 使用 OpenAI 处理请求
		// TODO: 实现 OpenAI 处理逻辑
		return fmt.Sprintf("OpenAI 处理请求: %s", query), nil

	case AIProviderArk:
		// 使用 Ark 处理请求
		// TODO: 实现 Ark 处理逻辑
		return fmt.Sprintf("Ark 处理请求: %s", query), nil

	default:
		return fmt.Sprintf("未知提供者 %s 处理请求: %s", m.provider, query), nil
	}
}

// HandleStreamRequest 处理流式AI请求
func (m *AIManager) HandleStreamRequest(ctx context.Context, query string, callback func(string) error) error {
	if !m.initialized {
		if err := m.Init(ctx); err != nil {
			return err
		}
	}

	m.logger.Info("处理流式AI请求", "query", query, "provider", m.provider)

	// 根据提供者选择不同的处理方式
	switch m.provider {
	case AIProviderMCP:
		// 使用 MCP 管理器处理流式请求
		if m.mcpManager != nil && m.mcpManager.IsRunning() {
			err := m.mcpManager.QueryAIStream(ctx, query, callback)
			if err != nil {
				m.logger.Error("MCP 流式查询失败", "error", err)
				return fmt.Errorf("MCP 流式查询失败: %w", err)
			}
			return nil
		} else if m.mcpClient != nil {
			// 使用 MCP 客户端处理流式请求
			response, err := m.mcpClient.QueryAI(ctx, query)
			if err != nil {
				m.logger.Error("MCP 客户端查询失败", "error", err)
				return fmt.Errorf("MCP 客户端查询失败: %w", err)
			}
			// 模拟流式响应
			return callback(response)
		}
		return fmt.Errorf("MCP 提供者未初始化")

	case AIProviderOpenAI:
		// 使用 OpenAI 处理流式请求
		// TODO: 实现 OpenAI 流式处理逻辑
		return callback(fmt.Sprintf("OpenAI 处理流式请求: %s", query))

	case AIProviderArk:
		// 使用 Ark 处理流式请求
		// TODO: 实现 Ark 流式处理逻辑
		return callback(fmt.Sprintf("Ark 处理流式请求: %s", query))

	default:
		return callback(fmt.Sprintf("未知提供者 %s 处理流式请求: %s", m.provider, query))
	}
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

	// 关闭 MCP 管理器
	if m.mcpManager != nil {
		if err := m.mcpManager.Stop(); err != nil {
			m.logger.Warn("关闭 MCP 管理器失败", "error", err)
		}
		m.mcpManager = nil
	}

	// 关闭 MCP 客户端
	if m.mcpClient != nil {
		if err := m.mcpClient.Close(); err != nil {
			m.logger.Warn("关闭 MCP 客户端失败", "error", err)
		}
		m.mcpClient = nil
	}

	m.initialized = false
	m.logger.Info("AI管理器已停止")
	return nil
}
