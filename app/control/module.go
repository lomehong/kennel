package main

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/logger"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
	"github.com/lomehong/kennel/pkg/utils"
)

// ControlModule 实现了终端管控模块
type ControlModule struct {
	logger         logger.Logger
	config         map[string]interface{}
	processCache   *ProcessCache
	processManager *ProcessManager
	commandManager *CommandManager
}

// NewControlModule 创建一个新的终端管控模块
func NewControlModule() pluginLib.Module {
	// 创建日志器
	log := logger.NewLogger("control-module", hclog.Info)

	// 创建进程缓存
	processCache := NewProcessCache()

	// 创建模块
	module := &ControlModule{
		logger:       log,
		config:       make(map[string]interface{}),
		processCache: processCache,
	}

	// 创建进程管理器
	module.processManager = NewProcessManager(log)

	// 创建命令管理器
	module.commandManager = NewCommandManager(log, module.config)

	return module
}

// Init 初始化模块
func (m *ControlModule) Init(config map[string]interface{}) error {
	m.logger.Info("初始化终端管控模块")
	m.config = config

	// 更新命令管理器的配置
	m.commandManager = NewCommandManager(m.logger, config)

	return nil
}

// Execute 执行模块操作
func (m *ControlModule) Execute(action string, params map[string]interface{}) (map[string]interface{}, error) {
	m.logger.Info("执行操作", "action", action)

	switch action {
	case "list_processes":
		return m.listProcesses()
	case "kill_process":
		if pid, ok := params["pid"].(float64); ok {
			return m.processManager.KillProcess(int(pid))
		}
		return nil, fmt.Errorf("缺少进程ID参数")
	case "execute_command":
		if cmd, ok := params["command"].(string); ok {
			return m.commandManager.ExecuteCommand(cmd)
		}
		return nil, fmt.Errorf("缺少命令参数")
	case "install_software":
		if pkg, ok := params["package"].(string); ok {
			return m.commandManager.InstallSoftware(pkg)
		}
		return nil, fmt.Errorf("缺少软件包参数")
	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
}

// Shutdown 关闭模块
func (m *ControlModule) Shutdown() error {
	m.logger.Info("关闭终端管控模块")
	return nil
}

// GetInfo 获取模块信息
func (m *ControlModule) GetInfo() pluginLib.ModuleInfo {
	return pluginLib.ModuleInfo{
		Name:             "control",
		Version:          "0.1.0",
		Description:      "终端管控模块，用于管理终端进程和执行命令",
		SupportedActions: []string{"list_processes", "kill_process", "execute_command", "install_software"},
	}
}

// HandleMessage 处理消息
func (m *ControlModule) HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error) {
	m.logger.Info("处理消息", "type", messageType, "id", messageID)

	switch messageType {
	case "process_list_request":
		// 处理进程列表请求
		return m.listProcesses()
	case "process_kill_request":
		// 处理进程终止请求
		if pidFloat, ok := payload["pid"].(float64); ok {
			return m.processManager.KillProcess(int(pidFloat))
		}
		return nil, fmt.Errorf("缺少进程ID参数")
	case "command_execute_request":
		// 处理命令执行请求
		if cmd, ok := payload["command"].(string); ok {
			return m.commandManager.ExecuteCommand(cmd)
		}
		return nil, fmt.Errorf("缺少命令参数")
	case "software_install_request":
		// 处理软件安装请求
		if pkg, ok := payload["package"].(string); ok {
			return m.commandManager.InstallSoftware(pkg)
		}
		return nil, fmt.Errorf("缺少软件包参数")
	default:
		return nil, fmt.Errorf("不支持的消息类型: %s", messageType)
	}
}

// listProcesses 列出进程
func (m *ControlModule) listProcesses() (map[string]interface{}, error) {
	// 获取缓存过期时间（默认为10秒）
	cacheExpiration := 10 * time.Second
	if expStr := utils.GetString(m.config, "process_cache_interval", ""); expStr != "" {
		if exp, err := time.ParseDuration(expStr); err == nil {
			cacheExpiration = exp
		}
	} else if expSec := utils.GetFloat(m.config, "process_cache_interval", 0); expSec > 0 {
		cacheExpiration = time.Duration(expSec) * time.Second
	}

	// 检查缓存是否有效
	if processes, valid := m.processCache.GetCachedProcesses(cacheExpiration); valid {
		m.logger.Debug("使用缓存的进程信息")

		// 直接构建map，避免JSON序列化/反序列化
		result := make(map[string]interface{})
		result["processes"] = ProcessesToMap(processes)

		return result, nil
	}

	// 获取进程列表
	processes, err := m.processManager.ListProcesses()
	if err != nil {
		m.logger.Error("列出进程失败", "error", err)
		return nil, fmt.Errorf("列出进程失败: %w", err)
	}

	// 更新缓存
	m.processCache.SetCachedProcesses(processes)

	// 直接构建map，避免JSON序列化/反序列化
	result := make(map[string]interface{})
	result["processes"] = ProcessesToMap(processes)

	return result, nil
}
