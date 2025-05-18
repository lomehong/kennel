package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/lomehong/kennel/pkg/logger"
	"github.com/lomehong/kennel/pkg/utils"
)

// CommandManager 负责执行命令
type CommandManager struct {
	logger logger.Logger
	config map[string]interface{}
}

// NewCommandManager 创建一个新的命令管理器
func NewCommandManager(logger logger.Logger, config map[string]interface{}) *CommandManager {
	return &CommandManager{
		logger: logger,
		config: config,
	}
}

// ExecuteCommand 执行命令
func (m *CommandManager) ExecuteCommand(command string) (map[string]interface{}, error) {
	m.logger.Info("执行命令", "command", command)

	// 获取超时时间（默认为30秒）
	timeout := 30 * time.Second
	if timeoutStr := utils.GetString(m.config, "command_timeout", ""); timeoutStr != "" {
		if t, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = t
		}
	} else if timeoutSec := utils.GetFloat(m.config, "command_timeout", 0); timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}

	// 添加超时控制，避免命令执行时间过长
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "powershell", "-Command", command)
	default:
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			m.logger.Error("执行命令超时", "command", command)
			return nil, fmt.Errorf("执行命令超时: %s", command)
		}
		m.logger.Error("执行命令失败", "command", command, "error", err)
		return nil, fmt.Errorf("执行命令失败: %w", err)
	}

	// 直接构建map，避免JSON序列化/反序列化
	result := make(map[string]interface{})
	result["success"] = true
	result["output"] = string(output)

	return result, nil
}

// InstallSoftware 安装软件
func (m *CommandManager) InstallSoftware(pkg string) (map[string]interface{}, error) {
	m.logger.Info("安装软件", "package", pkg)

	// 获取超时时间（默认为10分钟）
	timeout := 10 * time.Minute
	if timeoutStr := utils.GetString(m.config, "install_timeout", ""); timeoutStr != "" {
		if t, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = t
		}
	} else if timeoutSec := utils.GetFloat(m.config, "install_timeout", 0); timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}

	// 添加超时控制，避免命令执行时间过长
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// 使用Chocolatey安装软件
		cmd = exec.CommandContext(ctx, "powershell", "-Command", fmt.Sprintf("choco install %s -y", pkg))
	case "darwin":
		// 使用Homebrew安装软件
		cmd = exec.CommandContext(ctx, "brew", "install", pkg)
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			m.logger.Error("安装软件超时", "package", pkg)
			return nil, fmt.Errorf("安装软件超时: %s", pkg)
		}
		m.logger.Error("安装软件失败", "package", pkg, "error", err)
		return nil, fmt.Errorf("安装软件失败: %w", err)
	}

	// 直接构建map，避免JSON序列化/反序列化
	result := make(map[string]interface{})
	result["success"] = true
	result["output"] = string(output)

	return result, nil
}
