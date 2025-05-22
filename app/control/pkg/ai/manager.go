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
	mcpServers  map[string]*mcp.Server // 多个MCP服务器，键为服务器名称
	modelClient *mcp.ModelClient       // 大语言模型客户端
	provider    AIProvider
	initialized bool
	initLock    sync.Mutex
}

// NewAIManager 创建一个新的AI管理器
func NewAIManager(logger sdk.Logger, config map[string]interface{}) *AIManager {
	return &AIManager{
		logger:      logger,
		config:      config,
		mcpServers:  make(map[string]*mcp.Server),
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
	// 首先尝试从 settings.ai 路径获取
	var aiConfig map[string]interface{}
	if settings, ok := m.config["settings"].(map[string]interface{}); ok {
		if ai, ok := settings["ai"].(map[string]interface{}); ok {
			aiConfig = ai
		}
	}

	// 如果从 settings.ai 路径获取失败，尝试从 ai 路径获取
	if aiConfig == nil {
		if ai, ok := m.config["ai"].(map[string]interface{}); ok {
			aiConfig = ai
		} else {
			aiConfig = make(map[string]interface{})
		}
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

	// 如果 MCP 是主要提供者，则初始化 MCP 相关组件
	if m.provider == AIProviderMCP {
		// 获取模型配置
		modelConfig, ok := aiConfig["model"].(map[string]interface{})
		if !ok {
			modelConfig = make(map[string]interface{})
		}

		// 获取MCP配置
		mcpConfig, ok := aiConfig["mcp"].(map[string]interface{})
		if !ok {
			mcpConfig = make(map[string]interface{})
		}

		// 检查MCP是否启用
		mcpEnabled := getBoolConfig(mcpConfig, "enabled", true)
		if !mcpEnabled {
			m.logger.Info("MCP功能未启用")
			return nil
		}

		// 创建大语言模型客户端
		modelClientConfig := &mcp.ModelClientConfig{
			ModelName:     getConfigString(modelConfig, "name", "gpt-3.5-turbo"),
			Temperature:   getConfigFloat(modelConfig, "temperature", 0.7),
			MaxTokens:     getConfigInt(modelConfig, "max_tokens", 2000),
			Timeout:       getConfigDuration(modelConfig, "timeout", 30*time.Second),
			MaxRetries:    getConfigInt(mcpConfig, "client.max_retries", 3),
			RetryDelay:    getConfigDuration(mcpConfig, "client.retry_delay", 1*time.Second),
			RetryDelayMax: getConfigDuration(mcpConfig, "client.retry_delay_max", 5*time.Second),
			APIKey:        getConfigString(mcpConfig, "client.api_key", ""),
		}

		m.logger.Info("初始化大语言模型客户端", "model", modelClientConfig.ModelName)

		modelClient, err := mcp.NewModelClient(modelClientConfig, m.logger)
		if err != nil {
			m.logger.Error("创建大语言模型客户端失败", "error", err)
			return fmt.Errorf("创建大语言模型客户端失败: %w", err)
		}

		m.modelClient = modelClient

		// 获取MCP服务器配置
		mcpServers, ok := mcpConfig["servers"].(map[string]interface{})
		if !ok {
			mcpServers = make(map[string]interface{})
		}

		// 如果没有服务器配置，则使用旧的配置方式
		if len(mcpServers) == 0 {
			m.logger.Warn("未找到MCP服务器配置，将使用默认配置")

			// 创建默认服务器配置
			serverConfig := &mcp.ServerConfig{
				Addr:         ":8080",
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
				APIKey:       modelClientConfig.APIKey,
			}

			// 创建默认服务器
			server, err := mcp.NewServer(serverConfig, m.logger)
			if err != nil {
				m.logger.Error("创建默认MCP服务器失败", "error", err)
			} else {
				m.mcpServers["default"] = server
				m.modelClient.RegisterServer("default", server)

				// 注册默认工具
				registerDefaultTools(server, m.logger)
			}
		} else {
			// 遍历所有MCP服务器配置
			for serverName, serverConfig := range mcpServers {
				serverConfigMap, ok := serverConfig.(map[string]interface{})
				if !ok {
					m.logger.Warn("无效的服务器配置", "server", serverName)
					continue
				}

				// 获取服务器类型
				serverType := getConfigString(serverConfigMap, "type", "remote")

				// 根据服务器类型创建不同的配置
				if serverType == "local" {
					// 本地服务器配置
					command := getConfigString(serverConfigMap, "command", "")
					if command == "" {
						m.logger.Warn("本地服务器缺少命令", "server", serverName)
						continue
					}

					// 获取参数
					var args []string
					if argsValue, ok := serverConfigMap["args"].([]interface{}); ok {
						for _, arg := range argsValue {
							if argStr, ok := arg.(string); ok {
								args = append(args, argStr)
							}
						}
					}

					m.logger.Info("初始化本地MCP服务器", "server", serverName, "command", command)

					// 创建服务器配置
					serverConfig := &mcp.ServerConfig{
						Addr:         ":8080", // 本地服务器地址
						ReadTimeout:  getConfigDuration(serverConfigMap, "timeout", 10*time.Second),
						WriteTimeout: getConfigDuration(serverConfigMap, "timeout", 10*time.Second),
						APIKey:       modelClientConfig.APIKey,
					}

					// 创建服务器
					server, err := mcp.NewServer(serverConfig, m.logger)
					if err != nil {
						m.logger.Error("创建本地MCP服务器失败", "server", serverName, "error", err)
						continue
					}

					// 添加到服务器映射
					m.mcpServers[serverName] = server
					m.modelClient.RegisterServer(serverName, server)

					// TODO: 实现本地服务器启动逻辑
					// 这里需要实现启动本地服务器的逻辑

					// 注册工具
					// 这里应该根据服务器类型注册不同的工具
					registerDefaultTools(server, m.logger)
				} else {
					// 远程服务器配置
					serverAddr := getConfigString(serverConfigMap, "server_addr", "")
					if serverAddr == "" {
						m.logger.Warn("远程服务器缺少地址", "server", serverName)
						continue
					}

					m.logger.Info("初始化远程MCP服务器", "server", serverName, "addr", serverAddr)

					// 创建远程工具
					// 这里应该根据远程服务器的API创建相应的工具
					// 暂时不实现
				}
			}
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
		// 使用大语言模型客户端处理请求
		if m.modelClient != nil {
			response, err := m.modelClient.QueryAI(ctx, query)
			if err != nil {
				m.logger.Error("MCP 查询失败", "error", err)
				return "", fmt.Errorf("MCP 查询失败: %w", err)
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
		// 使用大语言模型客户端处理流式请求
		if m.modelClient != nil {
			err := m.modelClient.QueryAIStream(ctx, query, callback)
			if err != nil {
				m.logger.Error("MCP 流式查询失败", "error", err)
				return fmt.Errorf("MCP 流式查询失败: %w", err)
			}
			return nil
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

// 辅助函数：从配置中获取整数值
func getConfigInt(config map[string]interface{}, key string, defaultValue int) int {
	if value, ok := config[key].(int); ok {
		return value
	}
	if value, ok := config[key].(float64); ok {
		return int(value)
	}
	return defaultValue
}

// 辅助函数：从配置中获取浮点数值
func getConfigFloat(config map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := config[key].(float64); ok {
		return value
	}
	if value, ok := config[key].(int); ok {
		return float64(value)
	}
	return defaultValue
}

// 辅助函数：从配置中获取时间间隔
func getConfigDuration(config map[string]interface{}, key string, defaultValue time.Duration) time.Duration {
	if value, ok := config[key].(string); ok {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	if value, ok := config[key].(int); ok {
		return time.Duration(value) * time.Second
	}
	if value, ok := config[key].(float64); ok {
		return time.Duration(value) * time.Second
	}
	return defaultValue
}

// 注册默认工具
func registerDefaultTools(server *mcp.Server, logger sdk.Logger) {
	// 注册进程终止工具
	processKillTool := &mcp.ProcessKillTool{
		Logger: logger,
	}
	if err := server.RegisterTool(processKillTool); err != nil {
		logger.Error("注册进程终止工具失败", "error", err)
	}

	// 注册命令执行工具
	commandExecuteTool := &mcp.CommandExecuteTool{
		Logger: logger,
	}
	if err := server.RegisterTool(commandExecuteTool); err != nil {
		logger.Error("注册命令执行工具失败", "error", err)
	}

	// 注册文件读取工具
	fileReadTool := &mcp.FileReadTool{
		Logger: logger,
	}
	if err := server.RegisterTool(fileReadTool); err != nil {
		logger.Error("注册文件读取工具失败", "error", err)
	}
}

// Stop 停止AI管理器
func (m *AIManager) Stop() error {
	m.initLock.Lock()
	defer m.initLock.Unlock()

	// 关闭所有 MCP 服务器
	for serverName, server := range m.mcpServers {
		if server != nil {
			if err := server.Shutdown(context.Background()); err != nil {
				m.logger.Warn("关闭 MCP 服务器失败", "server", serverName, "error", err)
			}
		}
	}

	// 关闭大语言模型客户端
	if m.modelClient != nil {
		if err := m.modelClient.Close(); err != nil {
			m.logger.Warn("关闭大语言模型客户端失败", "error", err)
		}
		m.modelClient = nil
	}

	// 清空服务器映射
	m.mcpServers = make(map[string]*mcp.Server)

	m.initialized = false
	m.logger.Info("AI管理器已停止")
	return nil
}
