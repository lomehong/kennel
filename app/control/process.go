package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/lomehong/kennel/pkg/logger"
)

// ProcessManager 负责管理进程
type ProcessManager struct {
	logger logger.Logger
}

// NewProcessManager 创建一个新的进程管理器
func NewProcessManager(logger logger.Logger) *ProcessManager {
	return &ProcessManager{
		logger: logger,
	}
}

// ListProcesses 列出进程
func (m *ProcessManager) ListProcesses() ([]ProcessInfo, error) {
	// 根据操作系统执行不同的命令
	switch runtime.GOOS {
	case "windows":
		return m.listProcessesWindows()
	case "darwin":
		return m.listProcessesMacOS()
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// listProcessesWindows 列出Windows进程
func (m *ProcessManager) listProcessesWindows() ([]ProcessInfo, error) {
	// 添加超时控制，避免命令执行时间过长
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 使用更高效的命令，限制返回的进程数量，避免处理过多数据
	// 使用Where-Object过滤掉CPU使用率为0的进程，减少数据量
	cmd := exec.CommandContext(ctx, "powershell", "-Command",
		"Get-Process | Where-Object {$_.CPU -gt 0 -or $_.WorkingSet -gt 10000000} | Sort-Object -Property CPU -Descending | Select-Object -First 100 Id, ProcessName, CPU, WorkingSet, StartTime | ConvertTo-Json")

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			m.logger.Error("获取进程列表超时")
			return nil, fmt.Errorf("获取进程列表超时")
		}
		return nil, fmt.Errorf("执行PowerShell命令失败: %w", err)
	}

	// 解析输出
	var processes []map[string]interface{}
	if err := json.Unmarshal(output, &processes); err != nil {
		return nil, fmt.Errorf("解析PowerShell输出失败: %w", err)
	}

	// 预分配切片容量，避免动态扩容
	result := make([]ProcessInfo, 0, len(processes))

	for _, process := range processes {
		pid, _ := process["Id"].(float64)
		name, _ := process["ProcessName"].(string)
		cpu, _ := process["CPU"].(float64)
		memory, _ := process["WorkingSet"].(float64)
		startTime, _ := process["StartTime"].(string)

		result = append(result, ProcessInfo{
			PID:       int(pid),
			Name:      name,
			CPU:       cpu,
			Memory:    memory / 1024 / 1024, // 转换为MB
			StartTime: startTime,
			User:      "",
		})
	}

	return result, nil
}

// listProcessesMacOS 列出macOS进程
func (m *ProcessManager) listProcessesMacOS() ([]ProcessInfo, error) {
	// 添加超时控制，避免命令执行时间过长
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 使用更高效的命令，过滤掉CPU使用率低的进程，减少数据量
	// -o 指定输出格式，-r 按CPU使用率排序，head -100 限制返回的进程数量
	cmd := exec.CommandContext(ctx, "sh", "-c", "ps -eo pid,comm,%cpu,%mem,lstart,user -r | head -100")

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			m.logger.Error("获取进程列表超时")
			return nil, fmt.Errorf("获取进程列表超时")
		}
		return nil, fmt.Errorf("执行ps命令失败: %w", err)
	}

	// 解析输出
	lines := strings.Split(string(output), "\n")

	// 预分配切片容量，避免动态扩容
	result := make([]ProcessInfo, 0, len(lines))

	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		pid := 0
		fmt.Sscanf(fields[0], "%d", &pid)

		cpu := 0.0
		fmt.Sscanf(fields[2], "%f", &cpu)

		memory := 0.0
		fmt.Sscanf(fields[3], "%f", &memory)

		startTime := strings.Join(fields[4:9], " ")

		result = append(result, ProcessInfo{
			PID:       pid,
			Name:      fields[1],
			CPU:       cpu,
			Memory:    memory,
			StartTime: startTime,
			User:      fields[9],
		})
	}

	return result, nil
}

// KillProcess 终止进程
func (m *ProcessManager) KillProcess(pid int) (map[string]interface{}, error) {
	m.logger.Info("终止进程", "pid", pid)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", pid))
	default:
		cmd = exec.Command("kill", "-9", fmt.Sprintf("%d", pid))
	}

	if err := cmd.Run(); err != nil {
		m.logger.Error("终止进程失败", "pid", pid, "error", err)
		return nil, fmt.Errorf("终止进程失败: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("已终止进程 %d", pid),
	}, nil
}
