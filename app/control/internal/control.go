package internal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/lomehong/kennel/app/control/ai"
	"github.com/lomehong/kennel/pkg/core/plugin"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// ControlModule 实现了终端管控模块
type ControlModule struct {
	*sdk.BaseModule
	processManager  *ProcessManager
	commandExecutor *CommandExecutor
	aiManager       *ai.AIManager
}

// NewControlModule 创建一个新的终端管控模块
func NewControlModule() *ControlModule {
	// 创建基础模块
	base := sdk.NewBaseModule(
		"control",
		"终端管控插件",
		"1.0.0",
		"终端管控模块，用于远程执行命令、管理进程和安装软件",
	)

	// 创建模块
	module := &ControlModule{
		BaseModule: base,
	}

	return module
}

// Init 初始化模块
func (m *ControlModule) Init(ctx context.Context, config *plugin.ModuleConfig) error {
	// 调用基类初始化
	if err := m.BaseModule.Init(ctx, config); err != nil {
		return err
	}

	m.Logger.Info("初始化终端管控模块")

	// 设置日志级别
	logLevel := sdk.GetConfigString(m.Config, "log_level", "info")
	m.Logger.Debug("设置日志级别", "level", logLevel)

	// 创建进程管理器
	m.processManager = NewProcessManager(m.Logger, m.Config)

	// 创建命令执行器
	m.commandExecutor = NewCommandExecutor(m.Logger, m.Config)

	// 创建AI管理器
	m.aiManager = ai.NewAIManager(m.Logger, m.Config)

	return nil
}

// Start 启动模块
func (m *ControlModule) Start() error {
	m.Logger.Info("启动终端管控模块")

	// 初始化AI管理器
	if err := m.aiManager.Init(context.Background()); err != nil {
		m.Logger.Error("初始化AI管理器失败", "error", err)
		// 不返回错误，允许模块继续启动
	}

	return nil
}

// Stop 停止模块
func (m *ControlModule) Stop() error {
	m.Logger.Info("停止终端管控模块")
	return nil
}

// HandleRequest 处理请求
func (m *ControlModule) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	m.Logger.Info("处理请求", "action", req.Action)

	// 处理AI相关请求
	if req.Action == "ai_query" {
		// 获取查询参数
		query, ok := req.Params["query"].(string)
		if !ok {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "缺少查询参数",
				},
			}, nil
		}

		// 处理流式请求
		streaming, _ := req.Params["streaming"].(bool)
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
					close(responseChan)
					return
				}
				close(responseChan)
			}()

			// 返回流式响应
			return &plugin.Response{
				ID:      req.ID,
				Success: true,
				Data: map[string]interface{}{
					"streaming": true,
					"channel":   responseChan,
					"error":     errorChan,
				},
			}, nil
		}

		// 处理非流式请求
		response, err := m.aiManager.HandleRequest(ctx, query)
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "ai_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"response": response,
			},
		}, nil
	}

	switch req.Action {
	case "get_processes":
		// 获取进程列表
		processes, err := m.processManager.GetProcesses()
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "process_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"processes": ProcessesToMap(processes),
				"count":     len(processes),
			},
		}, nil

	case "kill_process":
		// 终止进程
		pidStr, ok := req.Params["pid"].(string)
		if !ok {
			pidFloat, ok := req.Params["pid"].(float64)
			if !ok {
				return &plugin.Response{
					ID:      req.ID,
					Success: false,
					Error: &plugin.ErrorInfo{
						Code:    "invalid_param",
						Message: "缺少进程ID参数",
					},
				}, nil
			}
			pidStr = fmt.Sprintf("%d", int(pidFloat))
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "无效的进程ID",
				},
			}, nil
		}

		// 终止进程
		if err := m.processManager.KillProcess(pid); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "kill_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"status":  "success",
				"message": fmt.Sprintf("进程 %d 已终止", pid),
			},
		}, nil

	default:
		return &plugin.Response{
			ID:      req.ID,
			Success: false,
			Error: &plugin.ErrorInfo{
				Code:    "unknown_action",
				Message: fmt.Sprintf("不支持的操作: %s", req.Action),
			},
		}, nil
	}
}

// HandleEvent 处理事件
func (m *ControlModule) HandleEvent(ctx context.Context, event *plugin.Event) error {
	m.Logger.Info("处理事件", "type", event.Type, "source", event.Source)

	switch event.Type {
	case "system.startup":
		// 系统启动事件
		m.Logger.Info("系统启动")

		// 初始化AI管理器
		if err := m.aiManager.Init(ctx); err != nil {
			m.Logger.Error("初始化AI管理器失败", "error", err)
		}

		return nil

	case "system.shutdown":
		// 系统关闭事件
		m.Logger.Info("系统关闭")
		return nil

	case "process.monitor":
		// 进程监控事件
		m.Logger.Info("进程监控")
		// 在实际应用中，这里可以执行进程监控逻辑
		return nil

	default:
		// 忽略其他事件
		return nil
	}
}
