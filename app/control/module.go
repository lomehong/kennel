package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/lomehong/kennel/pkg/core/plugin"
	"github.com/lomehong/kennel/pkg/sdk/go"
)

// ControlModule 实现了终端管控模块
type ControlModule struct {
	*sdk.BaseModule
	processManager *ProcessManager
	commandExecutor *CommandExecutor
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

	return nil
}

// Start 启动模块
func (m *ControlModule) Start() error {
	m.Logger.Info("启动终端管控模块")
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

	case "find_process":
		// 根据名称查找进程
		name, ok := req.Params["name"].(string)
		if !ok {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "缺少进程名称参数",
				},
			}, nil
		}

		// 查找进程
		processes, err := m.processManager.FindProcessByName(name)
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "find_error",
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

	case "execute_command":
		// 执行命令
		command, ok := req.Params["command"].(string)
		if !ok {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "缺少命令参数",
				},
			}, nil
		}

		// 获取参数
		argsInterface, ok := req.Params["args"].([]interface{})
		args := make([]string, 0)
		if ok {
			for _, arg := range argsInterface {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		}

		// 获取超时
		timeout := 0
		if timeoutFloat, ok := req.Params["timeout"].(float64); ok {
			timeout = int(timeoutFloat)
		}

		// 执行命令
		result, err := m.commandExecutor.ExecuteCommand(command, args, timeout)
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "execute_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data:    CommandResultToMap(result),
		}, nil

	case "install_software":
		// 安装软件
		packageName, ok := req.Params["package"].(string)
		if !ok {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "缺少软件包参数",
				},
			}, nil
		}

		// 获取超时
		timeout := 0
		if timeoutFloat, ok := req.Params["timeout"].(float64); ok {
			timeout = int(timeoutFloat)
		}

		// 安装软件
		result, err := m.commandExecutor.InstallSoftware(packageName, timeout)
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "install_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: result.Success,
			Data:    SoftwareInstallResultToMap(result),
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
