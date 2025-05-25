package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// ProcessKillToolResult 是进程终止工具的结果
type ProcessKillToolResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// CommandExecuteToolResult 是命令执行工具的结果
type CommandExecuteToolResult struct {
	Success  bool   `json:"success"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error,omitempty"`
}

// FileReadToolResult 是文件读取工具的结果
type FileReadToolResult struct {
	Success bool   `json:"success"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ToolExecutor 是工具执行器接口
type ToolExecutor interface {
	// Execute 执行工具
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// ProcessKillToolExecutor 是进程终止工具执行器
type ProcessKillToolExecutor struct {
	logger logging.Logger
}

// NewProcessKillToolExecutor 创建一个新的进程终止工具执行器
func NewProcessKillToolExecutor(logger logging.Logger) *ProcessKillToolExecutor {
	return &ProcessKillToolExecutor{
		logger: logger,
	}
}

// Execute 执行进程终止工具
func (e *ProcessKillToolExecutor) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	e.logger.Debug("执行进程终止工具", "params", params)

	// 获取进程 ID
	pidValue, ok := params["pid"]
	if !ok {
		return &ProcessKillToolResult{
			Success: false,
			Error:   "缺少参数: pid",
		}, nil
	}

	var pid int
	switch v := pidValue.(type) {
	case float64:
		pid = int(v)
	case int:
		pid = v
	case string:
		var err error
		pid, err = strconv.Atoi(v)
		if err != nil {
			return &ProcessKillToolResult{
				Success: false,
				Error:   fmt.Sprintf("无效的进程 ID: %s", v),
			}, nil
		}
	default:
		return &ProcessKillToolResult{
			Success: false,
			Error:   fmt.Sprintf("无效的进程 ID 类型: %T", pidValue),
		}, nil
	}

	// 获取进程
	process, err := os.FindProcess(pid)
	if err != nil {
		return &ProcessKillToolResult{
			Success: false,
			Error:   fmt.Sprintf("找不到进程: %d, %v", pid, err),
		}, nil
	}

	// 终止进程
	err = process.Kill()
	if err != nil {
		return &ProcessKillToolResult{
			Success: false,
			Error:   fmt.Sprintf("终止进程失败: %d, %v", pid, err),
		}, nil
	}

	return &ProcessKillToolResult{
		Success: true,
	}, nil
}

// CommandExecuteToolExecutor 是命令执行工具执行器
type CommandExecuteToolExecutor struct {
	logger logging.Logger
}

// NewCommandExecuteToolExecutor 创建一个新的命令执行工具执行器
func NewCommandExecuteToolExecutor(logger logging.Logger) *CommandExecuteToolExecutor {
	return &CommandExecuteToolExecutor{
		logger: logger,
	}
}

// Execute 执行命令执行工具
func (e *CommandExecuteToolExecutor) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	e.logger.Debug("执行命令执行工具", "params", params)

	// 获取命令
	commandValue, ok := params["command"]
	if !ok {
		return &CommandExecuteToolResult{
			Success:  false,
			ExitCode: -1,
			Error:    "缺少参数: command",
		}, nil
	}

	command, ok := commandValue.(string)
	if !ok {
		return &CommandExecuteToolResult{
			Success:  false,
			ExitCode: -1,
			Error:    fmt.Sprintf("无效的命令类型: %T", commandValue),
		}, nil
	}

	// 获取参数
	var args []string
	if argsValue, ok := params["args"]; ok {
		if argsArray, ok := argsValue.([]interface{}); ok {
			for _, arg := range argsArray {
				if argStr, ok := arg.(string); ok {
					args = append(args, argStr)
				}
			}
		}
	}

	// 获取工作目录
	workDir := ""
	if workDirValue, ok := params["workDir"]; ok {
		if workDirStr, ok := workDirValue.(string); ok {
			workDir = workDirStr
		}
	}

	// 获取环境变量
	env := os.Environ()
	if envValue, ok := params["env"]; ok {
		if envMap, ok := envValue.(map[string]interface{}); ok {
			for key, value := range envMap {
				if valueStr, ok := value.(string); ok {
					env = append(env, fmt.Sprintf("%s=%s", key, valueStr))
				}
			}
		}
	}

	// 获取超时
	timeout := 30 // 默认 30 秒
	if timeoutValue, ok := params["timeout"]; ok {
		switch v := timeoutValue.(type) {
		case float64:
			timeout = int(v)
		case int:
			timeout = v
		case string:
			var err error
			timeout, err = strconv.Atoi(v)
			if err != nil {
				timeout = 30
			}
		}
	}

	// 创建上下文
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 创建命令
	cmd := exec.CommandContext(execCtx, command, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}
	cmd.Env = env

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	err := cmd.Run()

	// 检查结果
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		} else {
			return &CommandExecuteToolResult{
				Success:  false,
				ExitCode: -1,
				Error:    fmt.Sprintf("执行命令失败: %v", err),
			}, nil
		}
	}

	return &CommandExecuteToolResult{
		Success:  exitCode == 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}, nil
}

// FileReadToolExecutor 是文件读取工具执行器
type FileReadToolExecutor struct {
	logger logging.Logger
}

// NewFileReadToolExecutor 创建一个新的文件读取工具执行器
func NewFileReadToolExecutor(logger logging.Logger) *FileReadToolExecutor {
	return &FileReadToolExecutor{
		logger: logger,
	}
}

// Execute 执行文件读取工具
func (e *FileReadToolExecutor) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	e.logger.Debug("执行文件读取工具", "params", params)

	// 获取文件路径
	pathValue, ok := params["path"]
	if !ok {
		return &FileReadToolResult{
			Success: false,
			Error:   "缺少参数: path",
		}, nil
	}

	path, ok := pathValue.(string)
	if !ok {
		return &FileReadToolResult{
			Success: false,
			Error:   fmt.Sprintf("无效的路径类型: %T", pathValue),
		}, nil
	}

	// 规范化路径
	path = filepath.Clean(path)

	// 检查文件是否存在
	info, err := os.Stat(path)
	if err != nil {
		return &FileReadToolResult{
			Success: false,
			Error:   fmt.Sprintf("文件不存在: %s, %v", path, err),
		}, nil
	}

	// 检查是否是目录
	if info.IsDir() {
		return &FileReadToolResult{
			Success: false,
			Error:   fmt.Sprintf("路径是目录: %s", path),
		}, nil
	}

	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		return &FileReadToolResult{
			Success: false,
			Error:   fmt.Sprintf("打开文件失败: %s, %v", path, err),
		}, nil
	}
	defer file.Close()

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		return &FileReadToolResult{
			Success: false,
			Error:   fmt.Sprintf("读取文件失败: %s, %v", path, err),
		}, nil
	}

	// 获取开始行和结束行
	startLine := 1
	endLine := -1

	if startLineValue, ok := params["startLine"]; ok {
		switch v := startLineValue.(type) {
		case float64:
			startLine = int(v)
		case int:
			startLine = v
		case string:
			var err error
			startLine, err = strconv.Atoi(v)
			if err != nil {
				startLine = 1
			}
		}
	}

	if endLineValue, ok := params["endLine"]; ok {
		switch v := endLineValue.(type) {
		case float64:
			endLine = int(v)
		case int:
			endLine = v
		case string:
			var err error
			endLine, err = strconv.Atoi(v)
			if err != nil {
				endLine = -1
			}
		}
	}

	// 如果指定了行范围，则只返回指定行
	if startLine > 1 || endLine > 0 {
		lines := strings.Split(string(content), "\n")
		if startLine > len(lines) {
			return &FileReadToolResult{
				Success: false,
				Error:   fmt.Sprintf("开始行超出文件行数: %d > %d", startLine, len(lines)),
			}, nil
		}

		if endLine == -1 || endLine > len(lines) {
			endLine = len(lines)
		}

		if startLine > endLine {
			return &FileReadToolResult{
				Success: false,
				Error:   fmt.Sprintf("开始行大于结束行: %d > %d", startLine, endLine),
			}, nil
		}

		// 提取指定行
		selectedLines := lines[startLine-1 : endLine]
		content = []byte(strings.Join(selectedLines, "\n"))
	}

	return &FileReadToolResult{
		Success: true,
		Content: string(content),
	}, nil
}
