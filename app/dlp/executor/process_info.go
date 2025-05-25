package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/lomehong/kennel/pkg/logging"
)

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID         int    `json:"pid"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	CommandLine string `json:"command_line"`
	ParentPID   int    `json:"parent_pid"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
}

// ProcessInfoCollector 进程信息收集器
type ProcessInfoCollector struct {
	logger logging.Logger
}

// NewProcessInfoCollector 创建进程信息收集器
func NewProcessInfoCollector(logger logging.Logger) *ProcessInfoCollector {
	return &ProcessInfoCollector{
		logger: logger,
	}
}

// GetCurrentProcessInfo 获取当前进程信息
func (pic *ProcessInfoCollector) GetCurrentProcessInfo() (*ProcessInfo, error) {
	return pic.GetProcessInfo(os.Getpid())
}

// GetProcessInfo 获取指定PID的进程信息
func (pic *ProcessInfoCollector) GetProcessInfo(pid int) (*ProcessInfo, error) {
	switch runtime.GOOS {
	case "windows":
		return pic.getProcessInfoWindows(pid)
	case "linux":
		return pic.getProcessInfoLinux(pid)
	case "darwin":
		return pic.getProcessInfoDarwin(pid)
	default:
		return pic.getProcessInfoGeneric(pid)
	}
}

// getProcessInfoWindows Windows平台进程信息获取
func (pic *ProcessInfoCollector) getProcessInfoWindows(pid int) (*ProcessInfo, error) {
	info := &ProcessInfo{
		PID: pid,
	}

	// 获取进程可执行文件路径
	if path, err := pic.getProcessPathWindows(pid); err == nil {
		info.Path = path
		info.Name = filepath.Base(path)
	} else {
		pic.logger.Debug("获取Windows进程路径失败", "pid", pid, "error", err)
		info.Name = "unknown"
		info.Path = "unknown"
	}

	// 获取命令行参数
	if cmdline, err := pic.getProcessCommandLineWindows(pid); err == nil {
		info.CommandLine = cmdline
	} else {
		pic.logger.Debug("获取Windows进程命令行失败", "pid", pid, "error", err)
		info.CommandLine = "unknown"
	}

	// 获取用户信息
	if userID, userName, err := pic.getProcessUserWindows(pid); err == nil {
		info.UserID = userID
		info.UserName = userName
	} else {
		pic.logger.Debug("获取Windows进程用户信息失败", "pid", pid, "error", err)
		info.UserID = "unknown"
		info.UserName = "unknown"
	}

	return info, nil
}

// getProcessInfoLinux Linux平台进程信息获取
func (pic *ProcessInfoCollector) getProcessInfoLinux(pid int) (*ProcessInfo, error) {
	info := &ProcessInfo{
		PID: pid,
	}

	// 从/proc/pid/目录获取进程信息
	procDir := fmt.Sprintf("/proc/%d", pid)

	// 检查进程是否存在
	if _, err := os.Stat(procDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("进程 %d 不存在", pid)
	}

	// 获取进程可执行文件路径
	if path, err := os.Readlink(fmt.Sprintf("%s/exe", procDir)); err == nil {
		info.Path = path
		info.Name = filepath.Base(path)
	} else {
		pic.logger.Debug("获取Linux进程路径失败", "pid", pid, "error", err)
		info.Name = "unknown"
		info.Path = "unknown"
	}

	// 获取命令行参数
	if cmdlineBytes, err := os.ReadFile(fmt.Sprintf("%s/cmdline", procDir)); err == nil {
		// /proc/pid/cmdline 使用null字符分隔参数
		cmdline := string(cmdlineBytes)
		cmdline = strings.ReplaceAll(cmdline, "\x00", " ")
		cmdline = strings.TrimSpace(cmdline)
		info.CommandLine = cmdline
	} else {
		pic.logger.Debug("获取Linux进程命令行失败", "pid", pid, "error", err)
		info.CommandLine = "unknown"
	}

	// 获取父进程ID
	if statBytes, err := os.ReadFile(fmt.Sprintf("%s/stat", procDir)); err == nil {
		statFields := strings.Fields(string(statBytes))
		if len(statFields) >= 4 {
			if ppid, err := strconv.Atoi(statFields[3]); err == nil {
				info.ParentPID = ppid
			}
		}
	}

	// 获取用户信息
	if statusBytes, err := os.ReadFile(fmt.Sprintf("%s/status", procDir)); err == nil {
		lines := strings.Split(string(statusBytes), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Uid:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					info.UserID = fields[1]
				}
				break
			}
		}
	}

	if info.UserID == "" {
		info.UserID = "unknown"
	}
	info.UserName = "unknown" // Linux下需要额外的系统调用获取用户名

	return info, nil
}

// getProcessInfoDarwin macOS平台进程信息获取
func (pic *ProcessInfoCollector) getProcessInfoDarwin(pid int) (*ProcessInfo, error) {
	info := &ProcessInfo{
		PID: pid,
	}

	// macOS下可以使用ps命令获取进程信息
	// 这里提供一个简化的实现
	info.Name = "unknown"
	info.Path = "unknown"
	info.CommandLine = "unknown"
	info.UserID = "unknown"
	info.UserName = "unknown"

	pic.logger.Debug("macOS进程信息获取需要完整实现", "pid", pid)

	return info, nil
}

// getProcessInfoGeneric 通用进程信息获取
func (pic *ProcessInfoCollector) getProcessInfoGeneric(pid int) (*ProcessInfo, error) {
	return &ProcessInfo{
		PID:         pid,
		Name:        "unknown",
		Path:        "unknown",
		CommandLine: "unknown",
		ParentPID:   0,
		UserID:      "unknown",
		UserName:    "unknown",
	}, nil
}

// Windows特定的进程信息获取函数
func (pic *ProcessInfoCollector) getProcessPathWindows(pid int) (string, error) {
	// 这里需要使用Windows API获取进程路径
	// 简化实现，返回当前进程路径
	if pid == os.Getpid() {
		if exe, err := os.Executable(); err == nil {
			return exe, nil
		}
	}
	return "unknown", fmt.Errorf("Windows进程路径获取需要完整实现")
}

func (pic *ProcessInfoCollector) getProcessCommandLineWindows(pid int) (string, error) {
	// 这里需要使用Windows API获取进程命令行
	// 简化实现
	if pid == os.Getpid() {
		return strings.Join(os.Args, " "), nil
	}
	return "unknown", fmt.Errorf("Windows进程命令行获取需要完整实现")
}

func (pic *ProcessInfoCollector) getProcessUserWindows(pid int) (string, string, error) {
	// 这里需要使用Windows API获取进程用户信息
	// 简化实现
	return "unknown", "unknown", fmt.Errorf("Windows进程用户信息获取需要完整实现")
}

// GetProcessesByName 根据进程名称获取进程列表
func (pic *ProcessInfoCollector) GetProcessesByName(name string) ([]*ProcessInfo, error) {
	var processes []*ProcessInfo

	switch runtime.GOOS {
	case "linux":
		// 在Linux上遍历/proc目录
		entries, err := os.ReadDir("/proc")
		if err != nil {
			return nil, fmt.Errorf("读取/proc目录失败: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			// 检查是否是数字目录名（PID）
			if pid, err := strconv.Atoi(entry.Name()); err == nil {
				if info, err := pic.GetProcessInfo(pid); err == nil {
					if strings.Contains(info.Name, name) {
						processes = append(processes, info)
					}
				}
			}
		}
	default:
		pic.logger.Warn("按名称获取进程列表在当前平台未完全实现", "platform", runtime.GOOS)
	}

	return processes, nil
}

// GetAllProcesses 获取所有进程信息
func (pic *ProcessInfoCollector) GetAllProcesses() ([]*ProcessInfo, error) {
	var processes []*ProcessInfo

	switch runtime.GOOS {
	case "linux":
		// 在Linux上遍历/proc目录
		entries, err := os.ReadDir("/proc")
		if err != nil {
			return nil, fmt.Errorf("读取/proc目录失败: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			// 检查是否是数字目录名（PID）
			if pid, err := strconv.Atoi(entry.Name()); err == nil {
				if info, err := pic.GetProcessInfo(pid); err == nil {
					processes = append(processes, info)
				}
			}
		}
	default:
		pic.logger.Warn("获取所有进程信息在当前平台未完全实现", "platform", runtime.GOOS)
	}

	return processes, nil
}

// IsProcessRunning 检查进程是否正在运行
func (pic *ProcessInfoCollector) IsProcessRunning(pid int) bool {
	switch runtime.GOOS {
	case "linux":
		_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
		return err == nil
	case "windows":
		// Windows下需要使用API检查
		return true // 简化实现
	default:
		return true // 简化实现
	}
}

// GetProcessParent 获取进程的父进程信息
func (pic *ProcessInfoCollector) GetProcessParent(pid int) (*ProcessInfo, error) {
	info, err := pic.GetProcessInfo(pid)
	if err != nil {
		return nil, err
	}

	if info.ParentPID == 0 {
		return nil, fmt.Errorf("无法获取进程 %d 的父进程ID", pid)
	}

	return pic.GetProcessInfo(info.ParentPID)
}

// GetProcessChildren 获取进程的子进程列表
func (pic *ProcessInfoCollector) GetProcessChildren(pid int) ([]*ProcessInfo, error) {
	var children []*ProcessInfo

	// 获取所有进程
	allProcesses, err := pic.GetAllProcesses()
	if err != nil {
		return nil, err
	}

	// 查找父进程ID匹配的进程
	for _, process := range allProcesses {
		if process.ParentPID == pid {
			children = append(children, process)
		}
	}

	return children, nil
}
