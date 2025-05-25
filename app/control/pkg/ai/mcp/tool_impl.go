package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/lomehong/kennel/pkg/logging"
)

// ProcessKillTool 是进程终止工具
type ProcessKillTool struct {
	Logger logging.Logger
}

// GetName 返回工具名称
func (t *ProcessKillTool) GetName() string {
	return "process_kill"
}

// GetDescription 返回工具描述
func (t *ProcessKillTool) GetDescription() string {
	return "终止指定的进程"
}

// GetParameters 返回工具参数
func (t *ProcessKillTool) GetParameters() map[string]Parameter {
	return map[string]Parameter{
		"pid": {
			Type:        "integer",
			Description: "进程ID",
			Required:    true,
		},
		"force": {
			Type:        "boolean",
			Description: "是否强制终止",
			Required:    false,
		},
	}
}

// Execute 执行工具
func (t *ProcessKillTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 获取参数
	pidValue, ok := params["pid"]
	if !ok {
		return nil, fmt.Errorf("缺少必要参数: pid")
	}

	var pid int
	switch v := pidValue.(type) {
	case int:
		pid = v
	case float64:
		pid = int(v)
	case string:
		var err error
		pid, err = strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("无效的进程ID: %s", v)
		}
	default:
		return nil, fmt.Errorf("无效的进程ID类型")
	}

	// 获取是否强制终止
	force := false
	if forceValue, ok := params["force"].(bool); ok {
		force = forceValue
	}

	// 获取进程
	process, err := os.FindProcess(pid)
	if err != nil {
		return &ProcessKillToolResult{
			Success: false,
			Error:   fmt.Sprintf("找不到进程: %v", err),
		}, nil
	}

	// 终止进程
	var signal syscall.Signal
	if force {
		signal = syscall.SIGKILL
	} else {
		signal = syscall.SIGTERM
	}

	if err := process.Signal(signal); err != nil {
		return &ProcessKillToolResult{
			Success: false,
			Error:   fmt.Sprintf("终止进程失败: %v", err),
		}, nil
	}

	return &ProcessKillToolResult{
		Success: true,
	}, nil
}

// CommandExecuteTool 是命令执行工具
type CommandExecuteTool struct {
	Logger logging.Logger
}

// GetName 返回工具名称
func (t *CommandExecuteTool) GetName() string {
	return "command_execute"
}

// GetDescription 返回工具描述
func (t *CommandExecuteTool) GetDescription() string {
	return "执行指定的命令"
}

// GetParameters 返回工具参数
func (t *CommandExecuteTool) GetParameters() map[string]Parameter {
	return map[string]Parameter{
		"command": {
			Type:        "string",
			Description: "要执行的命令",
			Required:    true,
		},
		"args": {
			Type:        "array",
			Description: "命令参数",
			Required:    false,
		},
		"timeout": {
			Type:        "integer",
			Description: "超时时间（秒）",
			Required:    false,
		},
	}
}

// Execute 执行工具
func (t *CommandExecuteTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 获取参数
	commandValue, ok := params["command"]
	if !ok {
		return nil, fmt.Errorf("缺少必要参数: command")
	}

	command, ok := commandValue.(string)
	if !ok {
		return nil, fmt.Errorf("无效的命令类型")
	}

	// 获取参数
	var args []string
	if argsValue, ok := params["args"].([]interface{}); ok {
		for _, arg := range argsValue {
			if argStr, ok := arg.(string); ok {
				args = append(args, argStr)
			}
		}
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, command, args...)

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	err := cmd.Run()

	// 获取退出码
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	return &CommandExecuteToolResult{
		Success:  err == nil,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Error:    err.Error(),
	}, nil
}

// FileReadTool 是文件读取工具
type FileReadTool struct {
	Logger logging.Logger
}

// GetName 返回工具名称
func (t *FileReadTool) GetName() string {
	return "file_read"
}

// GetDescription 返回工具描述
func (t *FileReadTool) GetDescription() string {
	return "读取指定的文件"
}

// GetParameters 返回工具参数
func (t *FileReadTool) GetParameters() map[string]Parameter {
	return map[string]Parameter{
		"path": {
			Type:        "string",
			Description: "文件路径",
			Required:    true,
		},
	}
}

// Execute 执行工具
func (t *FileReadTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 获取参数
	pathValue, ok := params["path"]
	if !ok {
		return nil, fmt.Errorf("缺少必要参数: path")
	}

	path, ok := pathValue.(string)
	if !ok {
		return nil, fmt.Errorf("无效的路径类型")
	}

	// 读取文件
	content, err := os.ReadFile(path)
	if err != nil {
		return &FileReadToolResult{
			Success: false,
			Error:   fmt.Sprintf("读取文件失败: %v", err),
		}, nil
	}

	return &FileReadToolResult{
		Success: true,
		Content: string(content),
	}, nil
}
