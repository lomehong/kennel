package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/lomehong/kennel/app/control/pkg/ai/mcp"
	"github.com/lomehong/kennel/pkg/logging"
	"github.com/shirou/gopsutil/v3/process"
)

// ProcessListTool 进程列表工具
type ProcessListTool struct {
	logger logging.Logger
}

// NewProcessListTool 创建一个新的进程列表工具
func NewProcessListTool(logger logging.Logger) *ProcessListTool {
	return &ProcessListTool{
		logger: logger,
	}
}

// GetName 返回工具的名称
func (t *ProcessListTool) GetName() string {
	return "get_processes"
}

// GetDescription 返回工具的描述
func (t *ProcessListTool) GetDescription() string {
	return "获取系统进程列表，可以按名称过滤"
}

// GetParameters 返回工具的参数定义
func (t *ProcessListTool) GetParameters() map[string]mcp.Parameter {
	return map[string]mcp.Parameter{
		"name_filter": {
			Type:        "string",
			Description: "进程名称过滤条件，可选",
			Required:    false,
		},
		"limit": {
			Type:        "number",
			Description: "返回的最大进程数量，默认为20",
			Required:    false,
			Default:     20,
		},
	}
}

// Execute 执行工具
func (t *ProcessListTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 解析参数
	nameFilter := ""
	if filter, ok := params["name_filter"].(string); ok {
		nameFilter = filter
	}

	limit := 20
	if limitVal, ok := params["limit"].(float64); ok {
		limit = int(limitVal)
	}

	t.logger.Info("获取进程列表", "filter", nameFilter, "limit", limit)

	// 获取进程列表
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}

	// 过滤和转换进程
	var result []map[string]interface{}
	count := 0

	for _, p := range processes {
		if count >= limit {
			break
		}

		name, err := p.Name()
		if err != nil {
			continue
		}

		// 应用名称过滤
		if nameFilter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(nameFilter)) {
			continue
		}

		pid := p.Pid

		// 获取CPU使用率
		cpu, _ := p.CPUPercent()

		// 获取内存使用率
		memInfo, _ := p.MemoryPercent()

		// 获取创建时间
		createTime, _ := p.CreateTime()
		startTime := time.Unix(0, createTime*int64(time.Millisecond)).Format(time.RFC3339)

		// 获取用户
		username, _ := p.Username()

		// 添加到结果
		result = append(result, map[string]interface{}{
			"pid":        int(pid),
			"name":       name,
			"cpu":        cpu,
			"memory":     float64(memInfo),
			"start_time": startTime,
			"user":       username,
		})

		count++
	}

	return result, nil
}

// Run 返回一个可执行的函数，用于与AI框架集成
func (t *ProcessListTool) Run() func(ctx context.Context, argumentsInJSON string) (string, error) {
	return func(ctx context.Context, argumentsInJSON string) (string, error) {
		// 解析参数
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", err
		}

		// 执行工具
		result, err := t.Execute(ctx, params)
		if err != nil {
			return "", err
		}

		// 序列化结果
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return "", err
		}

		return string(resultJSON), nil
	}
}

// CommandExecTool 命令执行工具
type CommandExecTool struct {
	logger logging.Logger
}

// NewCommandExecTool 创建一个新的命令执行工具
func NewCommandExecTool(logger logging.Logger) *CommandExecTool {
	return &CommandExecTool{
		logger: logger,
	}
}

// GetName 返回工具的名称
func (t *CommandExecTool) GetName() string {
	return "execute_command"
}

// GetDescription 返回工具的描述
func (t *CommandExecTool) GetDescription() string {
	return "执行系统命令，返回命令的输出结果"
}

// GetParameters 返回工具的参数定义
func (t *CommandExecTool) GetParameters() map[string]mcp.Parameter {
	return map[string]mcp.Parameter{
		"command": {
			Type:        "string",
			Description: "要执行的命令",
			Required:    true,
		},
		"args": {
			Type:        "array",
			Description: "命令参数列表",
			Required:    false,
		},
		"timeout": {
			Type:        "number",
			Description: "命令执行超时时间（秒），默认为30秒",
			Required:    false,
			Default:     30,
		},
	}
}

// Execute 执行工具
func (t *CommandExecTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 解析参数
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return nil, fmt.Errorf("命令不能为空")
	}

	// 解析参数列表
	var args []string
	if argsParam, ok := params["args"].([]interface{}); ok {
		for _, arg := range argsParam {
			if argStr, ok := arg.(string); ok {
				args = append(args, argStr)
			}
		}
	}

	// 解析超时时间
	timeout := 30
	if timeoutParam, ok := params["timeout"].(float64); ok {
		timeout = int(timeoutParam)
	}
	if timeout <= 0 {
		timeout = 30
	}

	// 检查命令是否在允许列表中
	allowedCommands := []string{"ipconfig", "ping", "netstat", "tasklist", "dir", "systeminfo", "echo", "whoami", "hostname"}
	commandAllowed := false
	for _, cmd := range allowedCommands {
		if strings.EqualFold(command, cmd) {
			commandAllowed = true
			break
		}
	}

	if !commandAllowed {
		return nil, fmt.Errorf("不允许执行命令: %s", command)
	}

	t.logger.Info("执行命令", "command", command, "args", args)

	// 创建上下文
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// 创建命令
	cmd := exec.CommandContext(execCtx, command, args...)

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 记录开始时间
	startTime := time.Now()

	// 执行命令
	err := cmd.Run()

	// 计算执行时间
	duration := time.Since(startTime).Milliseconds()

	// 创建结果
	result := map[string]interface{}{
		"command":     command + " " + strings.Join(args, " "),
		"exit_code":   0,
		"output":      stdout.String(),
		"error":       stderr.String(),
		"duration_ms": duration,
	}

	// 检查错误
	if err != nil {
		t.logger.Error("执行命令失败", "command", command, "error", err)
		result["error"] = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exit_code"] = exitErr.ExitCode()
		} else {
			result["exit_code"] = -1
		}
	}

	return result, nil
}

// Run 返回一个可执行的函数，用于与AI框架集成
func (t *CommandExecTool) Run() func(ctx context.Context, argumentsInJSON string) (string, error) {
	return func(ctx context.Context, argumentsInJSON string) (string, error) {
		// 解析参数
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", err
		}

		// 执行工具
		result, err := t.Execute(ctx, params)
		if err != nil {
			return "", err
		}

		// 序列化结果
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return "", err
		}

		return string(resultJSON), nil
	}
}

// ProcessKillTool 进程终止工具
type ProcessKillTool struct {
	logger logging.Logger
}

// NewProcessKillTool 创建一个新的进程终止工具
func NewProcessKillTool(logger logging.Logger) *ProcessKillTool {
	return &ProcessKillTool{
		logger: logger,
	}
}

// GetName 返回工具的名称
func (t *ProcessKillTool) GetName() string {
	return "kill_process"
}

// GetDescription 返回工具的描述
func (t *ProcessKillTool) GetDescription() string {
	return "终止指定的进程"
}

// GetParameters 返回工具的参数定义
func (t *ProcessKillTool) GetParameters() map[string]mcp.Parameter {
	return map[string]mcp.Parameter{
		"pid": {
			Type:        "number",
			Description: "进程ID",
			Required:    true,
		},
		"force": {
			Type:        "boolean",
			Description: "是否强制终止进程",
			Required:    false,
			Default:     false,
		},
	}
}

// Execute 执行工具
func (t *ProcessKillTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// 解析参数
	pidFloat, ok := params["pid"].(float64)
	if !ok {
		return nil, fmt.Errorf("无效的进程ID")
	}
	pid := int(pidFloat)

	if pid <= 0 {
		return nil, fmt.Errorf("无效的进程ID")
	}

	// 解析强制终止标志
	force := false
	if forceParam, ok := params["force"].(bool); ok {
		force = forceParam
	}

	t.logger.Info("终止进程", "pid", pid, "force", force)

	// 获取进程
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, fmt.Errorf("获取进程失败: %w", err)
	}

	// 获取进程名称
	name, err := p.Name()
	if err != nil {
		return nil, fmt.Errorf("获取进程名称失败: %w", err)
	}

	// 检查是否是受保护的进程
	protectedProcs := []string{"system", "explorer", "winlogon", "services", "lsass"}
	for _, protectedProc := range protectedProcs {
		if strings.ToLower(name) == protectedProc {
			return nil, fmt.Errorf("不允许终止受保护的进程: %s", name)
		}
	}

	// 终止进程
	var killErr error
	if force {
		killErr = p.Kill()
	} else {
		killErr = p.Terminate()
	}

	if killErr != nil {
		return nil, fmt.Errorf("终止进程失败: %w", killErr)
	}

	// 创建结果
	result := map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("进程 %d (%s) 已终止", pid, name),
	}

	return result, nil
}

// Run 返回一个可执行的函数，用于与AI框架集成
func (t *ProcessKillTool) Run() func(ctx context.Context, argumentsInJSON string) (string, error) {
	return func(ctx context.Context, argumentsInJSON string) (string, error) {
		// 解析参数
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", err
		}

		// 执行工具
		result, err := t.Execute(ctx, params)
		if err != nil {
			return "", err
		}

		// 序列化结果
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return "", err
		}

		return string(resultJSON), nil
	}
}
