package control

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// 辅助函数，用于从配置中获取字符串切片
func getConfigStringSliceFromCommand(config map[string]interface{}, key string) []string {
	if val, ok := config[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, len(slice))
			for i, v := range slice {
				if str, ok := v.(string); ok {
					result[i] = str
				}
			}
			return result
		}
	}
	return nil
}

// 辅助函数，用于从配置中获取布尔值
func getConfigBoolFromCommand(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := config[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// 辅助函数，用于从配置中获取整数值
func getConfigIntFromCommand(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
		if f, ok := val.(float64); ok {
			return int(f)
		}
	}
	return defaultValue
}

// CommandExecutor 命令执行器
type CommandExecutor struct {
	logger          logging.Logger
	config          map[string]interface{}
	allowedCommands map[string]bool
}

// NewCommandExecutor 创建一个新的命令执行器
func NewCommandExecutor(logger logging.Logger, config map[string]interface{}) *CommandExecutor {
	// 创建命令执行器
	executor := &CommandExecutor{
		logger:          logger,
		config:          config,
		allowedCommands: make(map[string]bool),
	}

	// 初始化允许的命令
	executor.initAllowedCommands()

	return executor
}

// initAllowedCommands 初始化允许的命令
func (e *CommandExecutor) initAllowedCommands() {
	// 获取允许的命令列表
	allowedCommands := getConfigStringSliceFromCommand(e.config, "allowed_commands")
	for _, cmd := range allowedCommands {
		e.allowedCommands[strings.ToLower(cmd)] = true
	}

	e.logger.Debug("初始化允许的命令", "count", len(e.allowedCommands))
}

// ExecuteCommand 执行命令
func (e *CommandExecutor) ExecuteCommand(command string, args []string, timeout int) (*CommandResult, error) {
	e.logger.Info("执行命令", "command", command, "args", args)

	// 检查是否允许执行命令
	if !getConfigBoolFromCommand(e.config, "allow_remote_command", true) {
		return nil, fmt.Errorf("不允许执行远程命令")
	}

	// 检查命令是否在白名单中
	if len(e.allowedCommands) > 0 {
		cmdLower := strings.ToLower(command)
		if !e.allowedCommands[cmdLower] {
			e.logger.Warn("尝试执行未授权的命令", "command", command)
			return nil, fmt.Errorf("命令不在白名单中: %s", command)
		}
	}

	// 设置超时
	if timeout <= 0 {
		timeout = getConfigIntFromCommand(e.config, "command_timeout", 30)
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// 创建命令
	cmd := exec.CommandContext(ctx, command, args...)

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
	result := &CommandResult{
		Command:  command + " " + strings.Join(args, " "),
		ExitCode: 0,
		Output:   stdout.String(),
		Error:    stderr.String(),
		Duration: duration,
	}

	// 检查错误
	if err != nil {
		e.logger.Error("执行命令失败", "command", command, "error", err)
		result.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	return result, nil
}

// InstallSoftware 安装软件
func (e *CommandExecutor) InstallSoftware(packageName string, timeout int) (*SoftwareInstallResult, error) {
	e.logger.Info("安装软件", "package", packageName)

	// 检查是否允许安装软件
	if !getConfigBoolFromCommand(e.config, "allow_software_install", true) {
		return nil, fmt.Errorf("不允许安装软件")
	}

	// 设置超时
	if timeout <= 0 {
		timeout = getConfigIntFromCommand(e.config, "install_timeout", 600)
	}

	// 根据操作系统选择安装命令
	var command string
	var args []string

	switch runtime.GOOS {
	case "windows":
		// 使用 Chocolatey 安装
		command = "choco"
		args = []string{"install", packageName, "-y"}
	case "darwin":
		// 使用 Homebrew 安装
		command = "brew"
		args = []string{"install", packageName}
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// 创建命令
	cmd := exec.CommandContext(ctx, command, args...)

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
	result := &SoftwareInstallResult{
		Package:  packageName,
		Success:  err == nil,
		ExitCode: 0,
		Output:   stdout.String(),
		Error:    stderr.String(),
		Duration: duration,
	}

	// 检查错误
	if err != nil {
		e.logger.Error("安装软件失败", "package", packageName, "error", err)
		result.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	return result, nil
}
