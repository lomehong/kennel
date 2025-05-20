package control

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/app/control/pkg/ai"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"github.com/lomehong/kennel/pkg/plugin/sdk"
)

// ControlModuleV2 实现了终端管控模块的V2版本
// 使用新的插件SDK
type ControlModuleV2 struct {
	// 基础插件
	*sdk.BasePlugin

	// 进程管理器
	processManager *ProcessManager

	// 命令执行器
	commandExecutor *CommandExecutor

	// AI管理器
	aiManager *ai.AIManager
}

// NewControlModuleV2 创建一个新的终端管控模块V2
func NewControlModuleV2(logger hclog.Logger) *ControlModuleV2 {
	// 创建插件信息
	info := api.PluginInfo{
		ID:          "control",
		Name:        "终端管控插件",
		Version:     "2.0.0",
		Description: "终端管控模块V2，用于远程执行命令、管理进程和安装软件",
		Author:      "Kennel Team",
		License:     "MIT",
		Tags:        []string{"control", "terminal", "process"},
		Capabilities: map[string]bool{
			"process_management": true,
			"command_execution": true,
			"ai_assistant":      true,
		},
	}

	// 创建基础插件
	basePlugin := sdk.NewBasePlugin(info, logger)

	// 创建模块
	module := &ControlModuleV2{
		BasePlugin: basePlugin,
	}

	return module
}

// Init 初始化模块
func (m *ControlModuleV2) Init(ctx context.Context, config api.PluginConfig) error {
	// 调用基类初始化
	if err := m.BasePlugin.Init(ctx, config); err != nil {
		return api.NewInitError(m.GetInfo().ID, "初始化基础插件失败", err)
	}

	m.GetLogger().Info("初始化终端管控模块V2")

	// 获取配置
	settings := config.Settings

	// 创建进程管理器
	m.processManager = NewProcessManager(m.GetLogger(), nil)

	// 创建命令执行器
	m.commandExecutor = NewCommandExecutor(m.GetLogger(), nil)

	// 创建AI管理器
	m.aiManager = ai.NewAIManager(m.GetLogger(), nil)

	return nil
}

// Start 启动模块
func (m *ControlModuleV2) Start(ctx context.Context) error {
	// 调用基类启动
	if err := m.BasePlugin.Start(ctx); err != nil {
		return api.NewStartError(m.GetInfo().ID, "启动基础插件失败", err)
	}

	m.GetLogger().Info("启动终端管控模块V2")

	// 初始化AI管理器
	if err := m.aiManager.Init(ctx); err != nil {
		m.GetLogger().Error("初始化AI管理器失败", "error", err)
		// 不返回错误，允许模块继续启动
	}

	return nil
}

// Stop 停止模块
func (m *ControlModuleV2) Stop(ctx context.Context) error {
	// 调用基类停止
	if err := m.BasePlugin.Stop(ctx); err != nil {
		return api.NewStopError(m.GetInfo().ID, "停止基础插件失败", err)
	}

	m.GetLogger().Info("停止终端管控模块V2")
	return nil
}

// HandleRequest 处理请求
func (m *ControlModuleV2) HandleRequest(ctx context.Context, action string, params map[string]interface{}) (map[string]interface{}, error) {
	m.GetLogger().Info("处理请求", "action", action)

	// 处理AI相关请求
	if action == "ai_query" {
		// 获取查询参数
		query, ok := params["query"].(string)
		if !ok {
			return nil, fmt.Errorf("缺少查询参数")
		}

		// 处理流式请求
		streaming, _ := params["streaming"].(bool)
		if streaming {
			// 创建流式响应通道
			responseChan := make(chan string)
			errorChan := make(chan error, 1)

			// 启动流式处理
			go func() {
				err := m.aiManager.HandleStreamRequest(ctx, query, func(content string) error {
					responseChan <- content
					return nil
				})
				if err != nil {
					errorChan <- err
				}
				close(responseChan)
			}()

			// 返回流式响应
			return map[string]interface{}{
				"streaming": true,
				"channel":   responseChan,
				"error":     errorChan,
			}, nil
		}

		// 处理非流式请求
		response, err := m.aiManager.HandleRequest(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("AI处理失败: %w", err)
		}

		return map[string]interface{}{
			"response": response,
		}, nil
	}

	switch action {
	case "get_processes":
		// 获取进程列表
		processes, err := m.processManager.GetProcesses()
		if err != nil {
			return nil, fmt.Errorf("获取进程失败: %w", err)
		}

		return map[string]interface{}{
			"processes": ProcessesToMap(processes),
			"count":     len(processes),
		}, nil

	case "kill_process":
		// 终止进程
		pidStr, ok := params["pid"].(string)
		if !ok {
			pidFloat, ok := params["pid"].(float64)
			if !ok {
				return nil, fmt.Errorf("缺少进程ID参数")
			}
			pidStr = fmt.Sprintf("%d", int(pidFloat))
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return nil, fmt.Errorf("无效的进程ID: %w", err)
		}

		// 终止进程
		if err := m.processManager.KillProcess(pid); err != nil {
			return nil, fmt.Errorf("终止进程失败: %w", err)
		}

		return map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("进程 %d 已终止", pid),
		}, nil

	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
}

// HandleEvent 处理事件
func (m *ControlModuleV2) HandleEvent(ctx context.Context, eventType string, eventData map[string]interface{}) error {
	m.GetLogger().Info("处理事件", "type", eventType)

	switch eventType {
	case "system.startup":
		// 系统启动事件
		m.GetLogger().Info("系统启动")

		// 初始化AI管理器
		if err := m.aiManager.Init(ctx); err != nil {
			m.GetLogger().Error("初始化AI管理器失败", "error", err)
		}

		return nil

	case "system.shutdown":
		// 系统关闭事件
		m.GetLogger().Info("系统关闭")
		return nil

	case "process.monitor":
		// 进程监控事件
		m.GetLogger().Info("进程监控")
		// 在实际应用中，这里可以执行进程监控逻辑
		return nil

	default:
		// 忽略其他事件
		return nil
	}
}

// 实现HTTPHandlerPlugin接口
func (m *ControlModuleV2) GetHTTPHandler() (interface{}, error) {
	// 在实际实现中，这里应该返回一个HTTP处理器
	return nil, fmt.Errorf("HTTP处理器未实现")
}

// 实现HTTPHandlerPlugin接口
func (m *ControlModuleV2) GetBasePath() string {
	return "/api/control"
}
