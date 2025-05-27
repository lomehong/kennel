//go:build selfprotect && windows
// +build selfprotect,windows

package selfprotect

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/sys/windows"
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	ntdll    = windows.NewLazySystemDLL("ntdll.dll")
	advapi32 = windows.NewLazySystemDLL("advapi32.dll")

	procOpenProcess                  = kernel32.NewProc("OpenProcess")
	procTerminateProcess             = kernel32.NewProc("TerminateProcess")
	procGetCurrentProcessId          = kernel32.NewProc("GetCurrentProcessId")
	procCreateToolhelp32Snapshot     = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First               = kernel32.NewProc("Process32FirstW")
	procProcess32Next                = kernel32.NewProc("Process32NextW")
	procCloseHandle                  = kernel32.NewProc("CloseHandle")
	procSetProcessShutdownParameters = kernel32.NewProc("SetProcessShutdownParameters")

	procNtSetInformationProcess   = ntdll.NewProc("NtSetInformationProcess")
	procNtQueryInformationProcess = ntdll.NewProc("NtQueryInformationProcess")

	procOpenProcessToken      = advapi32.NewProc("OpenProcessToken")
	procLookupPrivilegeValue  = advapi32.NewProc("LookupPrivilegeValueW")
	procAdjustTokenPrivileges = advapi32.NewProc("AdjustTokenPrivileges")
)

const (
	PROCESS_ALL_ACCESS        = 0x1F0FFF
	PROCESS_TERMINATE         = 0x0001
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_SET_INFORMATION   = 0x0200

	TH32CS_SNAPPROCESS = 0x00000002

	ProcessBreakOnTermination = 29
	ProcessDebugPort          = 7
	ProcessDebugObjectHandle  = 30
	ProcessDebugFlags         = 31

	TOKEN_ADJUST_PRIVILEGES = 0x0020
	TOKEN_QUERY             = 0x0008

	SE_DEBUG_PRIVILEGE   = "SeDebugPrivilege"
	SE_PRIVILEGE_ENABLED = 0x00000002
)

// PROCESSENTRY32 Windows进程信息结构
type PROCESSENTRY32 struct {
	Size              uint32
	Usage             uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	Threads           uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [260]uint16
}

// TOKEN_PRIVILEGES 权限结构
type TOKEN_PRIVILEGES struct {
	PrivilegeCount uint32
	Privileges     [1]LUID_AND_ATTRIBUTES
}

// LUID_AND_ATTRIBUTES 权限属性结构
type LUID_AND_ATTRIBUTES struct {
	Luid       LUID
	Attributes uint32
}

// LUID 本地唯一标识符
type LUID struct {
	LowPart  uint32
	HighPart int32
}

// WindowsProcessProtector Windows进程防护器
type WindowsProcessProtector struct {
	config ProcessProtectionConfig
	logger hclog.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	enabled            bool
	protectedProcesses map[string]*ProtectedProcess
	eventCallback      EventCallback

	// 监控状态
	monitoring         bool
	checkInterval      time.Duration
	restartAttempts    map[string]int
	maxRestartAttempts int
}

// ProtectedProcess 受保护的进程信息
type ProtectedProcess struct {
	Name            string
	Path            string
	ProcessID       uint32
	Handle          windows.Handle
	Protected       bool
	LastSeen        time.Time
	RestartCount    int
	CriticalProcess bool
}

// NewProcessProtector 创建Windows进程防护器
func NewProcessProtector(config ProcessProtectionConfig, logger hclog.Logger) ProcessProtector {
	ctx, cancel := context.WithCancel(context.Background())

	return &WindowsProcessProtector{
		config:             config,
		logger:             logger.Named("process-protector"),
		ctx:                ctx,
		cancel:             cancel,
		enabled:            config.Enabled,
		protectedProcesses: make(map[string]*ProtectedProcess),
		checkInterval:      5 * time.Second,
		restartAttempts:    make(map[string]int),
		maxRestartAttempts: 3,
	}
}

// Start 启动进程防护
func (pp *WindowsProcessProtector) Start(ctx context.Context) error {
	if !pp.enabled {
		return nil
	}

	pp.logger.Info("启动Windows进程防护")

	// 提升权限
	if err := pp.enableDebugPrivilege(); err != nil {
		pp.logger.Warn("提升调试权限失败", "error", err)
	}

	// 设置当前进程为关键进程
	if err := pp.setCriticalProcess(); err != nil {
		pp.logger.Warn("设置关键进程失败", "error", err)
	}

	// 初始化受保护的进程
	for _, processName := range pp.config.ProtectedProcesses {
		if err := pp.ProtectProcess(processName); err != nil {
			pp.logger.Error("保护进程失败", "process", processName, "error", err)
		}
	}

	// 启动监控
	pp.monitoring = true
	pp.wg.Add(1)
	go pp.monitorProcesses()

	return nil
}

// Stop 停止进程防护
func (pp *WindowsProcessProtector) Stop() error {
	pp.logger.Info("停止Windows进程防护")

	pp.monitoring = false
	pp.cancel()
	pp.wg.Wait()

	// 清理受保护的进程
	pp.mu.Lock()
	for _, process := range pp.protectedProcesses {
		if process.Handle != 0 {
			windows.CloseHandle(process.Handle)
		}
	}
	pp.protectedProcesses = make(map[string]*ProtectedProcess)
	pp.mu.Unlock()

	return nil
}

// IsEnabled 检查是否启用
func (pp *WindowsProcessProtector) IsEnabled() bool {
	return pp.enabled
}

// PeriodicCheck 定期检查
func (pp *WindowsProcessProtector) PeriodicCheck() error {
	if !pp.enabled || !pp.monitoring {
		return nil
	}

	// 检查受保护的进程状态
	pp.mu.RLock()
	processes := make([]*ProtectedProcess, 0, len(pp.protectedProcesses))
	for _, process := range pp.protectedProcesses {
		processes = append(processes, process)
	}
	pp.mu.RUnlock()

	for _, process := range processes {
		if err := pp.checkProcessStatus(process); err != nil {
			pp.logger.Error("检查进程状态失败", "process", process.Name, "error", err)
		}
	}

	return nil
}

// SetEventCallback 设置事件回调
func (pp *WindowsProcessProtector) SetEventCallback(callback EventCallback) {
	pp.eventCallback = callback
}

// ProtectProcess 保护进程
func (pp *WindowsProcessProtector) ProtectProcess(processName string) error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	// 查找进程
	processID, processPath, err := pp.findProcessByName(processName)
	if err != nil {
		return fmt.Errorf("查找进程失败: %w", err)
	}

	if processID == 0 {
		pp.logger.Warn("进程未运行", "process", processName)
		// 仍然添加到保护列表，以便后续监控
		pp.protectedProcesses[processName] = &ProtectedProcess{
			Name:     processName,
			Path:     processPath,
			LastSeen: time.Now(),
		}
		return nil
	}

	// 打开进程句柄
	handle, err := pp.openProcessHandle(processID)
	if err != nil {
		return fmt.Errorf("打开进程句柄失败: %w", err)
	}

	// 设置进程保护
	if err := pp.setProcessProtection(handle); err != nil {
		windows.CloseHandle(handle)
		return fmt.Errorf("设置进程保护失败: %w", err)
	}

	// 添加到保护列表
	pp.protectedProcesses[processName] = &ProtectedProcess{
		Name:      processName,
		Path:      processPath,
		ProcessID: processID,
		Handle:    handle,
		Protected: true,
		LastSeen:  time.Now(),
	}

	pp.logger.Info("进程已保护", "process", processName, "pid", processID)

	// 记录事件
	if pp.eventCallback != nil {
		pp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeProcess,
			Action:      "protect",
			Target:      processName,
			Description: fmt.Sprintf("进程 %s (PID: %d) 已被保护", processName, processID),
			Details: map[string]interface{}{
				"process_id":   processID,
				"process_path": processPath,
			},
		})
	}

	return nil
}

// UnprotectProcess 取消保护进程
func (pp *WindowsProcessProtector) UnprotectProcess(processName string) error {
	pp.mu.Lock()
	defer pp.mu.Unlock()

	process, exists := pp.protectedProcesses[processName]
	if !exists {
		return fmt.Errorf("进程未受保护: %s", processName)
	}

	// 关闭进程句柄
	if process.Handle != 0 {
		windows.CloseHandle(process.Handle)
	}

	// 从保护列表移除
	delete(pp.protectedProcesses, processName)

	pp.logger.Info("取消进程保护", "process", processName)

	// 记录事件
	if pp.eventCallback != nil {
		pp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeProcess,
			Action:      "unprotect",
			Target:      processName,
			Description: fmt.Sprintf("取消保护进程 %s", processName),
		})
	}

	return nil
}

// IsProcessProtected 检查进程是否受保护
func (pp *WindowsProcessProtector) IsProcessProtected(processName string) bool {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	_, exists := pp.protectedProcesses[processName]
	return exists
}

// GetProtectedProcesses 获取受保护的进程列表
func (pp *WindowsProcessProtector) GetProtectedProcesses() []string {
	pp.mu.RLock()
	defer pp.mu.RUnlock()

	processes := make([]string, 0, len(pp.protectedProcesses))
	for name := range pp.protectedProcesses {
		processes = append(processes, name)
	}
	return processes
}

// RestartProcess 重启进程
func (pp *WindowsProcessProtector) RestartProcess(processName string) error {
	pp.mu.Lock()
	process, exists := pp.protectedProcesses[processName]
	if !exists {
		pp.mu.Unlock()
		return fmt.Errorf("进程未受保护: %s", processName)
	}

	processPath := process.Path
	pp.mu.Unlock()

	if processPath == "" {
		return fmt.Errorf("进程路径未知: %s", processName)
	}

	// 检查重启次数
	attempts := pp.restartAttempts[processName]
	if attempts >= pp.maxRestartAttempts {
		pp.logger.Error("进程重启次数超限", "process", processName, "attempts", attempts)
		return fmt.Errorf("进程重启次数超限: %s", processName)
	}

	pp.logger.Info("重启进程", "process", processName, "path", processPath, "attempt", attempts+1)

	// 启动进程
	cmd := &syscall.ProcAttr{
		Files: []uintptr{0, 1, 2},
	}

	pid, handle, err := syscall.StartProcess(processPath, []string{processPath}, cmd)
	if err != nil {
		pp.restartAttempts[processName]++
		return fmt.Errorf("启动进程失败: %w", err)
	}

	// 关闭进程句柄（我们不需要保持它）
	syscall.CloseHandle(syscall.Handle(handle))

	pp.logger.Info("进程已重启", "process", processName, "pid", pid)

	// 重置重启计数
	delete(pp.restartAttempts, processName)

	// 重新保护进程
	time.Sleep(1 * time.Second) // 等待进程启动
	if err := pp.ProtectProcess(processName); err != nil {
		pp.logger.Error("重新保护进程失败", "process", processName, "error", err)
	}

	// 记录事件
	if pp.eventCallback != nil {
		pp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeProcess,
			Action:      "restart",
			Target:      processName,
			Description: fmt.Sprintf("进程 %s 已重启 (PID: %d)", processName, pid),
			Details: map[string]interface{}{
				"new_process_id": pid,
				"process_path":   processPath,
				"restart_count":  attempts + 1,
			},
		})
	}

	return nil
}

// PreventProcessTermination 防止进程终止
func (pp *WindowsProcessProtector) PreventProcessTermination(processID uint32) error {
	handle, err := pp.openProcessHandle(processID)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	return pp.setProcessProtection(handle)
}

// PreventProcessDebug 防止进程调试
func (pp *WindowsProcessProtector) PreventProcessDebug(processID uint32) error {
	handle, err := pp.openProcessHandle(processID)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	// 设置调试端口为无效值
	debugPort := uintptr(0xFFFFFFFF)
	ret, _, err := procNtSetInformationProcess.Call(
		uintptr(handle),
		ProcessDebugPort,
		uintptr(unsafe.Pointer(&debugPort)),
		unsafe.Sizeof(debugPort),
	)

	if ret != 0 {
		return fmt.Errorf("设置调试端口失败: %v", err)
	}

	// 设置调试标志
	debugFlags := uint32(1)
	ret, _, err = procNtSetInformationProcess.Call(
		uintptr(handle),
		ProcessDebugFlags,
		uintptr(unsafe.Pointer(&debugFlags)),
		unsafe.Sizeof(debugFlags),
	)

	if ret != 0 {
		return fmt.Errorf("设置调试标志失败: %v", err)
	}

	return nil
}

// enableDebugPrivilege 提升调试权限
func (pp *WindowsProcessProtector) enableDebugPrivilege() error {
	currentProcess := windows.CurrentProcess()
	var token windows.Token
	err := windows.OpenProcessToken(currentProcess, TOKEN_ADJUST_PRIVILEGES|TOKEN_QUERY, &token)
	if err != nil {
		return fmt.Errorf("打开进程令牌失败: %w", err)
	}
	defer token.Close()

	var luid LUID
	privilegeName, _ := syscall.UTF16PtrFromString(SE_DEBUG_PRIVILEGE)
	ret, _, err := procLookupPrivilegeValue.Call(
		0,
		uintptr(unsafe.Pointer(privilegeName)),
		uintptr(unsafe.Pointer(&luid)),
	)

	if ret == 0 {
		return fmt.Errorf("查找权限值失败: %v", err)
	}

	privileges := TOKEN_PRIVILEGES{
		PrivilegeCount: 1,
		Privileges: [1]LUID_AND_ATTRIBUTES{
			{
				Luid:       luid,
				Attributes: SE_PRIVILEGE_ENABLED,
			},
		},
	}

	ret, _, err = procAdjustTokenPrivileges.Call(
		uintptr(token),
		0,
		uintptr(unsafe.Pointer(&privileges)),
		0,
		0,
		0,
	)

	if ret == 0 {
		return fmt.Errorf("调整令牌权限失败: %v", err)
	}

	return nil
}

// setCriticalProcess 设置当前进程为关键进程
func (pp *WindowsProcessProtector) setCriticalProcess() error {
	currentProcess := windows.CurrentProcess()

	breakOnTermination := uint32(1)
	ret, _, err := procNtSetInformationProcess.Call(
		uintptr(currentProcess),
		ProcessBreakOnTermination,
		uintptr(unsafe.Pointer(&breakOnTermination)),
		unsafe.Sizeof(breakOnTermination),
	)

	if ret != 0 {
		return fmt.Errorf("设置关键进程失败: %v", err)
	}

	// 设置关闭优先级
	ret, _, err = procSetProcessShutdownParameters.Call(
		0x3FF, // 最高优先级
		0,     // 不强制关闭
	)

	if ret == 0 {
		return fmt.Errorf("设置关闭参数失败: %v", err)
	}

	pp.logger.Info("当前进程已设置为关键进程")
	return nil
}

// openProcessHandle 打开进程句柄
func (pp *WindowsProcessProtector) openProcessHandle(processID uint32) (windows.Handle, error) {
	ret, _, err := procOpenProcess.Call(
		PROCESS_ALL_ACCESS,
		0,
		uintptr(processID),
	)

	if ret == 0 {
		return 0, fmt.Errorf("打开进程句柄失败: %v", err)
	}

	return windows.Handle(ret), nil
}

// setProcessProtection 设置进程保护
func (pp *WindowsProcessProtector) setProcessProtection(handle windows.Handle) error {
	// 设置进程为关键进程
	breakOnTermination := uint32(1)
	ret, _, err := procNtSetInformationProcess.Call(
		uintptr(handle),
		ProcessBreakOnTermination,
		uintptr(unsafe.Pointer(&breakOnTermination)),
		unsafe.Sizeof(breakOnTermination),
	)

	if ret != 0 {
		return fmt.Errorf("设置进程保护失败: %v", err)
	}

	return nil
}

// findProcessByName 根据名称查找进程
func (pp *WindowsProcessProtector) findProcessByName(processName string) (uint32, string, error) {
	snapshot, _, err := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == uintptr(syscall.InvalidHandle) {
		return 0, "", fmt.Errorf("创建进程快照失败: %v", err)
	}
	defer procCloseHandle.Call(snapshot)

	var pe32 PROCESSENTRY32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&pe32)))
	if ret == 0 {
		return 0, "", fmt.Errorf("获取第一个进程失败")
	}

	for {
		exeFile := syscall.UTF16ToString(pe32.ExeFile[:])
		if strings.EqualFold(exeFile, processName) {
			// 获取进程完整路径
			processPath, _ := pp.getProcessPath(pe32.ProcessID)
			return pe32.ProcessID, processPath, nil
		}

		ret, _, _ := procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe32)))
		if ret == 0 {
			break
		}
	}

	return 0, "", nil
}

// getProcessPath 获取进程完整路径
func (pp *WindowsProcessProtector) getProcessPath(processID uint32) (string, error) {
	handle, err := pp.openProcessHandle(processID)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)

	var buffer [windows.MAX_PATH]uint16
	size := uint32(len(buffer))

	err = windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size)
	if err != nil {
		return "", err
	}

	return syscall.UTF16ToString(buffer[:size]), nil
}

// checkProcessStatus 检查进程状态
func (pp *WindowsProcessProtector) checkProcessStatus(process *ProtectedProcess) error {
	if process.ProcessID == 0 {
		// 进程未运行，尝试查找
		processID, _, err := pp.findProcessByName(process.Name)
		if err != nil {
			return err
		}

		if processID != 0 {
			// 进程已启动，重新保护
			pp.logger.Info("检测到进程启动", "process", process.Name, "pid", processID)
			return pp.ProtectProcess(process.Name)
		}

		// 进程仍未运行，检查是否需要重启
		if time.Since(process.LastSeen) > 30*time.Second {
			pp.logger.Warn("进程长时间未运行，尝试重启", "process", process.Name)
			return pp.RestartProcess(process.Name)
		}

		return nil
	}

	// 检查进程是否仍在运行
	if process.Handle != 0 {
		var exitCode uint32
		err := windows.GetExitCodeProcess(process.Handle, &exitCode)
		if err != nil || exitCode != 259 { // 259 = STILL_ACTIVE
			// 进程已终止
			pp.logger.Warn("检测到进程终止", "process", process.Name, "pid", process.ProcessID)

			// 记录事件
			if pp.eventCallback != nil {
				pp.eventCallback(ProtectionEvent{
					Type:        ProtectionTypeProcess,
					Action:      "terminated",
					Target:      process.Name,
					Description: fmt.Sprintf("受保护的进程 %s (PID: %d) 已终止", process.Name, process.ProcessID),
					Details: map[string]interface{}{
						"process_id": process.ProcessID,
						"exit_code":  exitCode,
					},
				})
			}

			// 清理句柄
			windows.CloseHandle(process.Handle)
			process.Handle = 0
			process.ProcessID = 0
			process.Protected = false

			// 尝试重启进程
			return pp.RestartProcess(process.Name)
		}
	}

	// 更新最后检查时间
	process.LastSeen = time.Now()
	return nil
}

// monitorProcesses 监控进程
func (pp *WindowsProcessProtector) monitorProcesses() {
	defer pp.wg.Done()

	ticker := time.NewTicker(pp.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pp.ctx.Done():
			return
		case <-ticker.C:
			if pp.monitoring {
				pp.PeriodicCheck()
			}
		}
	}
}
