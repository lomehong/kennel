package control

import (
	"fmt"
	"strings"
	"time"

	sdk "github.com/lomehong/kennel/pkg/sdk/go"
	"github.com/shirou/gopsutil/v3/process"
)

// ProcessManager 进程管理器
type ProcessManager struct {
	logger         sdk.Logger
	config         map[string]interface{}
	processCache   *ProcessCache
	protectedProcs map[string]bool
}

// NewProcessManager 创建一个新的进程管理器
func NewProcessManager(logger sdk.Logger, config map[string]interface{}) *ProcessManager {
	// 创建进程缓存
	processCache := NewProcessCache()

	// 创建进程管理器
	manager := &ProcessManager{
		logger:         logger,
		config:         config,
		processCache:   processCache,
		protectedProcs: make(map[string]bool),
	}

	// 初始化受保护的进程
	manager.initProtectedProcesses()

	return manager
}

// initProtectedProcesses 初始化受保护的进程
func (m *ProcessManager) initProtectedProcesses() {
	// 获取受保护的进程列表
	protectedProcs := sdk.GetConfigStringSlice(m.config, "protected_processes")
	for _, proc := range protectedProcs {
		m.protectedProcs[strings.ToLower(proc)] = true
	}

	m.logger.Debug("初始化受保护的进程", "count", len(m.protectedProcs))
}

// GetProcesses 获取进程列表
func (m *ProcessManager) GetProcesses() ([]ProcessInfo, error) {
	// 获取缓存过期时间
	cacheExpiration := time.Duration(sdk.GetConfigInt(m.config, "process_cache_expiration", 10)) * time.Second

	// 检查缓存是否有效
	if processes, valid := m.processCache.GetCachedProcesses(cacheExpiration); valid {
		m.logger.Debug("使用缓存的进程列表")
		return processes, nil
	}

	m.logger.Info("获取进程列表")

	// 获取所有进程
	procs, err := process.Processes()
	if err != nil {
		m.logger.Error("获取进程列表失败", "error", err)
		return nil, fmt.Errorf("获取进程列表失败: %w", err)
	}

	// 转换为ProcessInfo
	processes := make([]ProcessInfo, 0, len(procs))
	for _, p := range procs {
		// 获取进程名称
		name, err := p.Name()
		if err != nil {
			m.logger.Debug("获取进程名称失败", "pid", p.Pid, "error", err)
			continue
		}

		// 获取CPU使用率
		cpu, err := p.CPUPercent()
		if err != nil {
			m.logger.Debug("获取CPU使用率失败", "pid", p.Pid, "error", err)
			cpu = 0
		}

		// 获取内存使用率
		memInfo, err := p.MemoryPercent()
		if err != nil {
			m.logger.Debug("获取内存使用率失败", "pid", p.Pid, "error", err)
			memInfo = 0
		}

		// 获取创建时间
		createTime, err := p.CreateTime()
		if err != nil {
			m.logger.Debug("获取创建时间失败", "pid", p.Pid, "error", err)
			createTime = 0
		}
		startTime := time.Unix(0, createTime*int64(time.Millisecond)).Format(time.RFC3339)

		// 获取用户
		username, err := p.Username()
		if err != nil {
			m.logger.Debug("获取用户名失败", "pid", p.Pid, "error", err)
			username = ""
		}

		// 创建进程信息
		processInfo := ProcessInfo{
			PID:       int(p.Pid),
			Name:      name,
			CPU:       cpu,
			Memory:    float64(memInfo),
			StartTime: startTime,
			User:      username,
		}

		processes = append(processes, processInfo)
	}

	// 更新缓存
	m.processCache.SetCachedProcesses(processes)

	return processes, nil
}

// KillProcess 终止进程
func (m *ProcessManager) KillProcess(pid int) error {
	m.logger.Info("终止进程", "pid", pid)

	// 检查是否允许终止进程
	if !sdk.GetConfigBool(m.config, "allow_process_kill", true) {
		return fmt.Errorf("不允许终止进程")
	}

	// 获取进程
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		m.logger.Error("获取进程失败", "pid", pid, "error", err)
		return fmt.Errorf("获取进程失败: %w", err)
	}

	// 获取进程名称
	name, err := p.Name()
	if err != nil {
		m.logger.Error("获取进程名称失败", "pid", pid, "error", err)
		return fmt.Errorf("获取进程名称失败: %w", err)
	}

	// 检查是否是受保护的进程
	if m.protectedProcs[strings.ToLower(name)] {
		m.logger.Warn("尝试终止受保护的进程", "pid", pid, "name", name)
		return fmt.Errorf("不允许终止受保护的进程: %s", name)
	}

	// 终止进程
	if err := p.Kill(); err != nil {
		m.logger.Error("终止进程失败", "pid", pid, "error", err)
		return fmt.Errorf("终止进程失败: %w", err)
	}

	// 清除缓存
	m.processCache.SetCachedProcesses(nil)

	return nil
}

// IsProcessRunning 检查进程是否运行
func (m *ProcessManager) IsProcessRunning(pid int) (bool, error) {
	m.logger.Debug("检查进程是否运行", "pid", pid)

	// 获取进程
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return false, nil // 进程不存在
	}

	// 检查进程是否运行
	running, err := p.IsRunning()
	if err != nil {
		m.logger.Error("检查进程是否运行失败", "pid", pid, "error", err)
		return false, fmt.Errorf("检查进程是否运行失败: %w", err)
	}

	return running, nil
}

// FindProcessByName 根据名称查找进程
func (m *ProcessManager) FindProcessByName(name string) ([]ProcessInfo, error) {
	m.logger.Debug("根据名称查找进程", "name", name)

	// 获取所有进程
	processes, err := m.GetProcesses()
	if err != nil {
		return nil, err
	}

	// 查找匹配的进程
	matchedProcesses := make([]ProcessInfo, 0)
	for _, p := range processes {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(name)) {
			matchedProcesses = append(matchedProcesses, p)
		}
	}

	return matchedProcesses, nil
}

// GetProcessInfo 获取进程信息
func (m *ProcessManager) GetProcessInfo(pid int) (*ProcessInfo, error) {
	m.logger.Debug("获取进程信息", "pid", pid)

	// 获取进程
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		m.logger.Error("获取进程失败", "pid", pid, "error", err)
		return nil, fmt.Errorf("获取进程失败: %w", err)
	}

	// 获取进程名称
	name, err := p.Name()
	if err != nil {
		m.logger.Error("获取进程名称失败", "pid", pid, "error", err)
		return nil, fmt.Errorf("获取进程名称失败: %w", err)
	}

	// 获取CPU使用率
	cpu, err := p.CPUPercent()
	if err != nil {
		m.logger.Debug("获取CPU使用率失败", "pid", pid, "error", err)
		cpu = 0
	}

	// 获取内存使用率
	memInfo, err := p.MemoryPercent()
	if err != nil {
		m.logger.Debug("获取内存使用率失败", "pid", pid, "error", err)
		memInfo = 0
	}

	// 获取创建时间
	createTime, err := p.CreateTime()
	if err != nil {
		m.logger.Debug("获取创建时间失败", "pid", pid, "error", err)
		createTime = 0
	}
	startTime := time.Unix(0, createTime*int64(time.Millisecond)).Format(time.RFC3339)

	// 获取用户
	username, err := p.Username()
	if err != nil {
		m.logger.Debug("获取用户名失败", "pid", pid, "error", err)
		username = ""
	}

	// 创建进程信息
	processInfo := &ProcessInfo{
		PID:       int(p.Pid),
		Name:      name,
		CPU:       cpu,
		Memory:    float64(memInfo),
		StartTime: startTime,
		User:      username,
	}

	return processInfo, nil
}
