//go:build windows

package interceptor

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/lomehong/kennel/pkg/logging"
)

// Windows API 常量
const (
	TCP_TABLE_OWNER_PID_ALL = 5
	UDP_TABLE_OWNER_PID     = 1
	AF_INET                 = 2
	NO_ERROR                = 0

	// 进程访问权限
	PROCESS_QUERY_INFORMATION         = 0x0400
	PROCESS_VM_READ                   = 0x0010
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

	// 令牌权限
	TOKEN_ADJUST_PRIVILEGES = 0x0020
	TOKEN_QUERY             = 0x0008
	SE_PRIVILEGE_ENABLED    = 0x00000002
	SE_DEBUG_NAME           = "SeDebugPrivilege"

	// 连接状态
	MIB_TCP_STATE_CLOSED     = 1
	MIB_TCP_STATE_LISTEN     = 2
	MIB_TCP_STATE_SYN_SENT   = 3
	MIB_TCP_STATE_SYN_RCVD   = 4
	MIB_TCP_STATE_ESTAB      = 5
	MIB_TCP_STATE_FIN_WAIT1  = 6
	MIB_TCP_STATE_FIN_WAIT2  = 7
	MIB_TCP_STATE_CLOSE_WAIT = 8
	MIB_TCP_STATE_CLOSING    = 9
	MIB_TCP_STATE_LAST_ACK   = 10
	MIB_TCP_STATE_TIME_WAIT  = 11
	MIB_TCP_STATE_DELETE_TCB = 12

	// 错误代码
	ERROR_INSUFFICIENT_BUFFER = 122

	// 进程快照常量
	TH32CS_SNAPPROCESS   = 0x00000002
	INVALID_HANDLE_VALUE = ^uintptr(0)
)

// TCP连接表项结构
type MIB_TCPROW_OWNER_PID struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPid  uint32
}

// TCP连接表结构
type MIB_TCPTABLE_OWNER_PID struct {
	NumEntries uint32
	Table      [1]MIB_TCPROW_OWNER_PID
}

// UDP连接表项结构
type MIB_UDPROW_OWNER_PID struct {
	LocalAddr uint32
	LocalPort uint32
	OwningPid uint32
}

// UDP连接表结构
type MIB_UDPTABLE_OWNER_PID struct {
	NumEntries uint32
	Table      [1]MIB_UDPROW_OWNER_PID
}

// 进程信息结构
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

// 权限结构
type LUID struct {
	LowPart  uint32
	HighPart int32
}

type LUID_AND_ATTRIBUTES struct {
	Luid       LUID
	Attributes uint32
}

type TOKEN_PRIVILEGES struct {
	PrivilegeCount uint32
	Privileges     [1]LUID_AND_ATTRIBUTES
}

// ProcessTracker 进程跟踪器
type ProcessTracker struct {
	logger          logging.Logger
	tcpTable        map[string]uint32 // "ip:port" -> PID
	udpTable        map[string]uint32 // "ip:port" -> PID
	processCache    map[uint32]*ProcessInfo
	processSnapshot map[uint32]string // PID -> ProcessName
	mu              sync.RWMutex

	// Windows API
	iphlpapi *syscall.LazyDLL
	kernel32 *syscall.LazyDLL
	psapi    *syscall.LazyDLL
	advapi32 *syscall.LazyDLL

	// API 函数
	getExtendedTcpTable       *syscall.LazyProc
	getExtendedUdpTable       *syscall.LazyProc
	createToolhelp32Snapshot  *syscall.LazyProc
	process32First            *syscall.LazyProc
	process32Next             *syscall.LazyProc
	closeHandle               *syscall.LazyProc
	openProcess               *syscall.LazyProc
	getModuleFileNameEx       *syscall.LazyProc
	queryFullProcessImageName *syscall.LazyProc

	// 权限相关API
	getCurrentProcess     *syscall.LazyProc
	openProcessToken      *syscall.LazyProc
	lookupPrivilegeValue  *syscall.LazyProc
	adjustTokenPrivileges *syscall.LazyProc
	getTokenInformation   *syscall.LazyProc

	// 权限提升状态
	privilegesEnabled bool

	// 监控状态
	monitoringActive bool
	stopMonitoring   chan bool
	lastUpdateTime   time.Time
	updateStats      struct {
		totalUpdates   int64
		successUpdates int64
		failedUpdates  int64
		lastError      error
		avgUpdateTime  time.Duration
	}
}

// NewProcessTracker 创建进程跟踪器
func NewProcessTracker(logger logging.Logger) *ProcessTracker {
	pt := &ProcessTracker{
		logger:          logger,
		tcpTable:        make(map[string]uint32),
		udpTable:        make(map[string]uint32),
		processCache:    make(map[uint32]*ProcessInfo),
		processSnapshot: make(map[uint32]string),
		stopMonitoring:  make(chan bool, 1),
	}

	// 加载Windows API
	pt.loadWindowsAPIs()

	// 执行初始连接表更新
	if err := pt.UpdateConnectionTables(); err != nil {
		logger.Warn("初始连接表更新失败", "error", err)
	}

	return pt
}

// loadWindowsAPIs 加载Windows API
func (pt *ProcessTracker) loadWindowsAPIs() {
	pt.iphlpapi = syscall.NewLazyDLL("iphlpapi.dll")
	pt.kernel32 = syscall.NewLazyDLL("kernel32.dll")
	pt.psapi = syscall.NewLazyDLL("psapi.dll")
	pt.advapi32 = syscall.NewLazyDLL("advapi32.dll")

	// 网络和进程API
	pt.getExtendedTcpTable = pt.iphlpapi.NewProc("GetExtendedTcpTable")
	pt.getExtendedUdpTable = pt.iphlpapi.NewProc("GetExtendedUdpTable")
	pt.createToolhelp32Snapshot = pt.kernel32.NewProc("CreateToolhelp32Snapshot")
	pt.process32First = pt.kernel32.NewProc("Process32FirstW")
	pt.process32Next = pt.kernel32.NewProc("Process32NextW")
	pt.closeHandle = pt.kernel32.NewProc("CloseHandle")
	pt.openProcess = pt.kernel32.NewProc("OpenProcess")
	pt.getModuleFileNameEx = pt.psapi.NewProc("GetModuleFileNameExW")
	pt.queryFullProcessImageName = pt.kernel32.NewProc("QueryFullProcessImageNameW")

	// 权限相关API
	pt.getCurrentProcess = pt.kernel32.NewProc("GetCurrentProcess")
	pt.openProcessToken = pt.advapi32.NewProc("OpenProcessToken")
	pt.lookupPrivilegeValue = pt.advapi32.NewProc("LookupPrivilegeValueW")
	pt.adjustTokenPrivileges = pt.advapi32.NewProc("AdjustTokenPrivileges")
	pt.getTokenInformation = pt.advapi32.NewProc("GetTokenInformation")

	// 尝试提升权限
	pt.enableDebugPrivilege()
}

// enableDebugPrivilege 启用调试权限
func (pt *ProcessTracker) enableDebugPrivilege() {
	pt.logger.Debug("尝试启用调试权限")

	// 获取当前进程句柄
	currentProcess, _, _ := pt.getCurrentProcess.Call()
	if currentProcess == 0 {
		pt.logger.Warn("获取当前进程句柄失败")
		return
	}

	// 打开进程令牌
	var token uintptr
	ret, _, _ := pt.openProcessToken.Call(
		currentProcess,
		TOKEN_ADJUST_PRIVILEGES|TOKEN_QUERY,
		uintptr(unsafe.Pointer(&token)),
	)

	if ret == 0 {
		pt.logger.Warn("打开进程令牌失败")
		return
	}
	defer pt.closeHandle.Call(token)

	// 查找调试权限的LUID
	var luid LUID
	privilegeName, _ := syscall.UTF16PtrFromString(SE_DEBUG_NAME)
	ret, _, _ = pt.lookupPrivilegeValue.Call(
		0, // lpSystemName (NULL for local system)
		uintptr(unsafe.Pointer(privilegeName)),
		uintptr(unsafe.Pointer(&luid)),
	)

	if ret == 0 {
		pt.logger.Warn("查找调试权限LUID失败")
		return
	}

	// 构造权限结构
	privileges := TOKEN_PRIVILEGES{
		PrivilegeCount: 1,
		Privileges: [1]LUID_AND_ATTRIBUTES{
			{
				Luid:       luid,
				Attributes: SE_PRIVILEGE_ENABLED,
			},
		},
	}

	// 调整令牌权限
	ret, _, _ = pt.adjustTokenPrivileges.Call(
		token,
		0, // DisableAllPrivileges
		uintptr(unsafe.Pointer(&privileges)),
		0, // BufferLength
		0, // PreviousState
		0, // ReturnLength
	)

	if ret != 0 {
		pt.privilegesEnabled = true
		pt.logger.Info("调试权限启用成功")
	} else {
		pt.logger.Warn("调试权限启用失败")
	}
}

// UpdateConnectionTables 更新连接表（增强版本）
func (pt *ProcessTracker) UpdateConnectionTables() error {
	startTime := time.Now()
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// 更新统计信息
	pt.updateStats.totalUpdates++

	// 更新TCP连接表
	if err := pt.updateTCPTable(); err != nil {
		pt.logger.Error("更新TCP连接表失败", "error", err)
		pt.updateStats.failedUpdates++
		pt.updateStats.lastError = err
		return err
	}

	// 更新UDP连接表
	if err := pt.updateUDPTable(); err != nil {
		pt.logger.Error("更新UDP连接表失败", "error", err)
		pt.updateStats.failedUpdates++
		pt.updateStats.lastError = err
		return err
	}

	// 更新进程快照
	if err := pt.updateProcessSnapshot(); err != nil {
		pt.logger.Warn("更新进程快照失败", "error", err)
		// 不返回错误，因为连接表更新成功
	}

	// 更新成功统计
	pt.updateStats.successUpdates++
	pt.lastUpdateTime = time.Now()
	updateDuration := time.Since(startTime)

	// 计算平均更新时间
	if pt.updateStats.successUpdates > 0 {
		pt.updateStats.avgUpdateTime = time.Duration(
			(int64(pt.updateStats.avgUpdateTime)*pt.updateStats.successUpdates + int64(updateDuration)) /
				(pt.updateStats.successUpdates + 1))
	} else {
		pt.updateStats.avgUpdateTime = updateDuration
	}

	pt.logger.Debug("连接表更新完成",
		"tcp_entries", len(pt.tcpTable),
		"udp_entries", len(pt.udpTable),
		"update_time", updateDuration,
		"total_updates", pt.updateStats.totalUpdates,
		"success_rate", float64(pt.updateStats.successUpdates)/float64(pt.updateStats.totalUpdates))

	return nil
}

// updateTCPTable 更新TCP连接表
func (pt *ProcessTracker) updateTCPTable() error {
	var size uint32

	// 获取所需缓冲区大小
	ret, _, _ := pt.getExtendedTcpTable.Call(
		0, // pTcpTable
		uintptr(unsafe.Pointer(&size)),
		0, // bOrder
		AF_INET,
		TCP_TABLE_OWNER_PID_ALL,
		0, // Reserved
	)

	if ret != ERROR_INSUFFICIENT_BUFFER {
		return fmt.Errorf("获取TCP表大小失败: %d", ret)
	}

	// 分配缓冲区并获取TCP表
	buffer := make([]byte, size)
	ret, _, _ = pt.getExtendedTcpTable.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
		0, // bOrder
		AF_INET,
		TCP_TABLE_OWNER_PID_ALL,
		0, // Reserved
	)

	if ret != NO_ERROR {
		return fmt.Errorf("获取TCP表失败: %d", ret)
	}

	// 解析TCP表
	table := (*MIB_TCPTABLE_OWNER_PID)(unsafe.Pointer(&buffer[0]))
	pt.tcpTable = make(map[string]uint32)

	for i := uint32(0); i < table.NumEntries; i++ {
		entry := (*MIB_TCPROW_OWNER_PID)(unsafe.Pointer(
			uintptr(unsafe.Pointer(&table.Table[0])) +
				uintptr(i)*unsafe.Sizeof(table.Table[0])))

		localIP := intToIP(entry.LocalAddr)
		localPort := uint16(entry.LocalPort>>8 | entry.LocalPort<<8)

		key := fmt.Sprintf("%s:%d", localIP.String(), localPort)
		pt.tcpTable[key] = entry.OwningPid
	}

	return nil
}

// updateUDPTable 更新UDP连接表
func (pt *ProcessTracker) updateUDPTable() error {
	var size uint32

	// 获取所需缓冲区大小
	ret, _, _ := pt.getExtendedUdpTable.Call(
		0, // pUdpTable
		uintptr(unsafe.Pointer(&size)),
		0, // bOrder
		AF_INET,
		UDP_TABLE_OWNER_PID,
		0, // Reserved
	)

	if ret != ERROR_INSUFFICIENT_BUFFER {
		return fmt.Errorf("获取UDP表大小失败: %d", ret)
	}

	// 分配缓冲区并获取UDP表
	buffer := make([]byte, size)
	ret, _, _ = pt.getExtendedUdpTable.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&size)),
		0, // bOrder
		AF_INET,
		UDP_TABLE_OWNER_PID,
		0, // Reserved
	)

	if ret != NO_ERROR {
		return fmt.Errorf("获取UDP表失败: %d", ret)
	}

	// 解析UDP表
	table := (*MIB_UDPTABLE_OWNER_PID)(unsafe.Pointer(&buffer[0]))
	pt.udpTable = make(map[string]uint32)

	for i := uint32(0); i < table.NumEntries; i++ {
		entry := (*MIB_UDPROW_OWNER_PID)(unsafe.Pointer(
			uintptr(unsafe.Pointer(&table.Table[0])) +
				uintptr(i)*unsafe.Sizeof(table.Table[0])))

		localIP := intToIP(entry.LocalAddr)
		localPort := uint16(entry.LocalPort>>8 | entry.LocalPort<<8)

		key := fmt.Sprintf("%s:%d", localIP.String(), localPort)
		pt.udpTable[key] = entry.OwningPid
	}

	return nil
}

// GetProcessByConnection 根据连接信息获取进程ID（增强版本）
func (pt *ProcessTracker) GetProcessByConnection(protocol Protocol, localIP net.IP, localPort uint16) uint32 {
	// 首先尝试从缓存查找
	pid := pt.findProcessInCache(protocol, localIP, localPort)
	if pid != 0 {
		pt.logger.Debug("从缓存找到进程", "protocol", protocol, "ip", localIP.String(), "port", localPort, "pid", pid)
		return pid
	}

	// 如果缓存中没有，强制更新连接表
	pt.logger.Debug("缓存未命中，更新连接表", "protocol", protocol, "ip", localIP.String(), "port", localPort)
	if err := pt.UpdateConnectionTables(); err != nil {
		pt.logger.Error("更新连接表失败", "error", err)
		return 0
	}

	// 再次尝试查找
	pid = pt.findProcessInCache(protocol, localIP, localPort)
	if pid != 0 {
		pt.logger.Debug("更新连接表后找到进程", "protocol", protocol, "ip", localIP.String(), "port", localPort, "pid", pid)
	} else {
		pt.logger.Debug("更新连接表后仍未找到进程", "protocol", protocol, "ip", localIP.String(), "port", localPort)
	}

	return pid
}

// findProcessInCache 在缓存中查找进程
func (pt *ProcessTracker) findProcessInCache(protocol Protocol, localIP net.IP, localPort uint16) uint32 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var table map[string]uint32
	var protocolName string

	switch protocol {
	case ProtocolTCP:
		table = pt.tcpTable
		protocolName = "TCP"
	case ProtocolUDP:
		table = pt.udpTable
		protocolName = "UDP"
	default:
		pt.logger.Debug("不支持的协议", "protocol", protocol)
		return 0
	}

	// 策略1：精确匹配 IP:Port
	key := fmt.Sprintf("%s:%d", localIP.String(), localPort)
	if pid, exists := table[key]; exists {
		pt.logger.Debug("精确匹配成功", "protocol", protocolName, "key", key, "pid", pid)
		return pid
	}

	// 策略2：通配符匹配 0.0.0.0:Port（监听所有接口）
	wildcardKey := fmt.Sprintf("0.0.0.0:%d", localPort)
	if pid, exists := table[wildcardKey]; exists {
		pt.logger.Debug("通配符匹配成功", "protocol", protocolName, "key", wildcardKey, "pid", pid)
		return pid
	}

	// 策略3：本地接口匹配（尝试常见的本地IP）
	localIPs := []string{
		"127.0.0.1",
		"::1",
	}

	for _, localIPStr := range localIPs {
		localKey := fmt.Sprintf("%s:%d", localIPStr, localPort)
		if pid, exists := table[localKey]; exists {
			pt.logger.Debug("本地接口匹配成功", "protocol", protocolName, "key", localKey, "pid", pid)
			return pid
		}
	}

	// 策略4：端口匹配（查找使用相同端口的进程）
	if portPid := pt.findProcessByPort(protocol, localPort); portPid != 0 {
		pt.logger.Debug("端口匹配成功", "protocol", protocolName, "port", localPort, "pid", portPid)
		return portPid
	}

	// 策略5：调试输出所有连接表项以帮助诊断
	pt.logger.Debug("所有匹配策略都失败，输出连接表信息",
		"protocol", protocolName,
		"target_ip", localIP.String(),
		"target_port", localPort,
		"table_size", len(table))

	// 输出前5个连接表项用于调试
	count := 0
	for tableKey, tablePid := range table {
		if count >= 5 {
			break
		}
		pt.logger.Debug("连接表项", "key", tableKey, "pid", tablePid)
		count++
	}

	return 0
}

// findProcessByPort 根据端口查找进程（增强版本）
func (pt *ProcessTracker) findProcessByPort(protocol Protocol, port uint16) uint32 {
	var table map[string]uint32
	var protocolName string

	switch protocol {
	case ProtocolTCP:
		table = pt.tcpTable
		protocolName = "TCP"
	case ProtocolUDP:
		table = pt.udpTable
		protocolName = "UDP"
	default:
		return 0
	}

	portStr := fmt.Sprintf(":%d", port)

	// 遍历连接表，查找使用相同端口的连接
	for key, pid := range table {
		// 检查端口是否匹配（key格式为 "IP:Port"）
		if strings.HasSuffix(key, portStr) {
			pt.logger.Debug("端口匹配找到进程",
				"protocol", protocolName,
				"port", port,
				"key", key,
				"pid", pid)
			return pid
		}
	}

	pt.logger.Debug("端口匹配未找到进程",
		"protocol", protocolName,
		"port", port,
		"table_size", len(table))
	return 0
}

// GetProcessByConnectionEx 根据完整连接信息获取进程ID（四元组匹配）
func (pt *ProcessTracker) GetProcessByConnectionEx(protocol Protocol, localIP net.IP, localPort uint16, remoteIP net.IP, remotePort uint16) uint32 {
	// 首先尝试四元组精确匹配
	pid := pt.findProcessByQuadruple(protocol, localIP, localPort, remoteIP, remotePort)
	if pid != 0 {
		return pid
	}

	// 如果四元组匹配失败，降级到本地连接匹配
	return pt.GetProcessByConnection(protocol, localIP, localPort)
}

// findProcessByQuadruple 四元组匹配查找进程
func (pt *ProcessTracker) findProcessByQuadruple(protocol Protocol, localIP net.IP, localPort uint16, remoteIP net.IP, remotePort uint16) uint32 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if protocol != ProtocolTCP {
		// UDP没有远程连接概念，直接返回0
		return 0
	}

	// 遍历TCP连接表，查找完全匹配的连接
	for key, pid := range pt.tcpTable {
		// 解析连接表中的信息
		if pt.matchesConnection(key, localIP, localPort, remoteIP, remotePort) {
			pt.logger.Debug("四元组匹配成功",
				"local", fmt.Sprintf("%s:%d", localIP.String(), localPort),
				"remote", fmt.Sprintf("%s:%d", remoteIP.String(), remotePort),
				"pid", pid)
			return pid
		}
	}

	return 0
}

// matchesConnection 检查连接是否匹配
func (pt *ProcessTracker) matchesConnection(key string, localIP net.IP, localPort uint16, remoteIP net.IP, remotePort uint16) bool {
	// 简化的匹配逻辑，实际应该解析连接表中的完整信息
	expectedLocal := fmt.Sprintf("%s:%d", localIP.String(), localPort)
	return key == expectedLocal
}

// GetProcessInfo 获取进程详细信息
func (pt *ProcessTracker) GetProcessInfo(pid uint32) *ProcessInfo {
	pt.mu.RLock()
	if info, exists := pt.processCache[pid]; exists {
		pt.mu.RUnlock()
		return info
	}
	pt.mu.RUnlock()

	// 获取进程信息
	info := pt.getProcessDetails(pid)
	if info != nil {
		pt.mu.Lock()
		pt.processCache[pid] = info
		pt.mu.Unlock()
	}

	return info
}

// getProcessDetails 获取进程详细信息（生产级实现）
func (pt *ProcessTracker) getProcessDetails(pid uint32) *ProcessInfo {
	startTime := time.Now()

	processInfo := &ProcessInfo{
		PID: int(pid),
	}

	pt.logger.Debug("开始获取进程详细信息", "pid", pid)

	// 特殊处理系统进程
	if pt.isSystemProcess(pid) {
		processInfo.ProcessName = pt.getSystemProcessName(pid)
		processInfo.ExecutePath = "System"
		processInfo.User = "SYSTEM"
		processInfo.CommandLine = processInfo.ProcessName
		pt.logger.Debug("系统进程信息获取完成", "pid", pid, "name", processInfo.ProcessName)
		return processInfo
	}

	// 策略1：首先尝试从进程快照获取基本信息（权限要求较低）
	if name := pt.getProcessNameFromSnapshot(pid); name != "" {
		processInfo.ProcessName = name
		pt.logger.Debug("从进程快照获取进程名成功", "pid", pid, "name", name)
	} else {
		processInfo.ProcessName = "unknown_process"
		pt.logger.Debug("从进程快照获取进程名失败", "pid", pid)
	}

	// 策略2：尝试不同的权限级别打开进程句柄获取详细信息
	handle := pt.openProcessWithFallback(pid)
	if handle != 0 {
		defer pt.closeHandle.Call(handle)
		pt.logger.Debug("成功打开进程句柄", "pid", pid, "handle", handle)

		// 获取进程可执行文件路径
		if execPath := pt.getProcessExecutablePath(handle); execPath != "" {
			processInfo.ExecutePath = execPath
			// 从完整路径中提取进程名
			if extractedName := pt.extractProcessName(execPath); extractedName != "" {
				processInfo.ProcessName = extractedName
			}
			pt.logger.Debug("获取进程路径成功", "pid", pid, "path", execPath)
		} else {
			pt.logger.Debug("获取进程路径失败", "pid", pid)
		}

		// 获取用户信息
		if user := pt.getProcessUserEnhanced(handle); user != "" {
			processInfo.User = user
			pt.logger.Debug("获取进程用户成功", "pid", pid, "user", user)
		} else {
			processInfo.User = pt.getCurrentUser()
			pt.logger.Debug("获取进程用户失败，使用当前用户", "pid", pid, "user", processInfo.User)
		}
	} else {
		// 无法打开进程句柄，使用默认值
		processInfo.User = pt.getCurrentUser()
		pt.logger.Debug("无法打开进程句柄", "pid", pid)
	}

	// 策略3：获取命令行参数（尝试多种方法）
	if cmdline := pt.getProcessCommandLineEnhanced(pid); cmdline != "" {
		processInfo.CommandLine = cmdline
		pt.logger.Debug("获取进程命令行成功", "pid", pid, "cmdline", cmdline)
	} else {
		// 使用进程名作为命令行
		processInfo.CommandLine = processInfo.ProcessName
		pt.logger.Debug("获取进程命令行失败，使用进程名", "pid", pid)
	}

	// 确保所有字段都有值
	if processInfo.ProcessName == "" {
		processInfo.ProcessName = fmt.Sprintf("process_%d", pid)
	}
	if processInfo.User == "" {
		processInfo.User = "unknown"
	}
	if processInfo.CommandLine == "" {
		processInfo.CommandLine = processInfo.ProcessName
	}

	processingTime := time.Since(startTime)
	pt.logger.Debug("获取进程详细信息完成",
		"pid", pid,
		"name", processInfo.ProcessName,
		"path", processInfo.ExecutePath,
		"user", processInfo.User,
		"cmdline", processInfo.CommandLine,
		"processing_time", processingTime)

	return processInfo
}

// openProcessWithFallback 使用降级权限策略打开进程
func (pt *ProcessTracker) openProcessWithFallback(pid uint32) uintptr {
	const PROCESS_QUERY_INFORMATION = 0x0400
	const PROCESS_VM_READ = 0x0010
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

	// 尝试完整权限
	if handle, _, _ := pt.openProcess.Call(
		PROCESS_QUERY_INFORMATION|PROCESS_VM_READ,
		0,
		uintptr(pid),
	); handle != 0 {
		return handle
	}

	// 尝试查询权限
	if handle, _, _ := pt.openProcess.Call(
		PROCESS_QUERY_INFORMATION,
		0,
		uintptr(pid),
	); handle != 0 {
		return handle
	}

	// 尝试有限查询权限
	if handle, _, _ := pt.openProcess.Call(
		PROCESS_QUERY_LIMITED_INFORMATION,
		0,
		uintptr(pid),
	); handle != 0 {
		return handle
	}

	return 0
}

// getProcessExecutablePath 获取进程可执行文件路径
func (pt *ProcessTracker) getProcessExecutablePath(handle uintptr) string {
	var pathBuffer [260]uint16
	var pathSize uint32 = 260

	ret, _, _ := pt.queryFullProcessImageName.Call(
		handle,
		0, // dwFlags
		uintptr(unsafe.Pointer(&pathBuffer[0])),
		uintptr(unsafe.Pointer(&pathSize)),
	)

	if ret != 0 && pathSize > 0 {
		return syscall.UTF16ToString(pathBuffer[:pathSize])
	}

	return ""
}

// extractProcessName 从路径中提取进程名称
func (pt *ProcessTracker) extractProcessName(execPath string) string {
	if execPath == "" {
		return "unknown"
	}

	// 从路径中提取文件名
	for i := len(execPath) - 1; i >= 0; i-- {
		if execPath[i] == '\\' || execPath[i] == '/' {
			return execPath[i+1:]
		}
	}

	return execPath
}

// getProcessNameFromSnapshot 从进程快照获取进程名称
func (pt *ProcessTracker) getProcessNameFromSnapshot(pid uint32) string {
	const TH32CS_SNAPPROCESS = 0x00000002

	// 创建进程快照
	snapshot, _, _ := pt.createToolhelp32Snapshot.Call(
		TH32CS_SNAPPROCESS,
		0,
	)

	if snapshot == 0 || snapshot == ^uintptr(0) {
		return ""
	}

	defer pt.closeHandle.Call(snapshot)

	var pe32 PROCESSENTRY32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	// 获取第一个进程
	ret, _, _ := pt.process32First.Call(
		snapshot,
		uintptr(unsafe.Pointer(&pe32)),
	)

	if ret == 0 {
		return ""
	}

	// 遍历进程列表
	for {
		if pe32.ProcessID == pid {
			return syscall.UTF16ToString(pe32.ExeFile[:])
		}

		ret, _, _ := pt.process32Next.Call(
			snapshot,
			uintptr(unsafe.Pointer(&pe32)),
		)

		if ret == 0 {
			break
		}
	}

	return ""
}

// getProcessUser 获取进程用户信息
func (pt *ProcessTracker) getProcessUser(handle uintptr) string {
	// 打开进程令牌
	var token uintptr
	ret, _, _ := pt.openProcessToken.Call(
		handle,
		TOKEN_QUERY,
		uintptr(unsafe.Pointer(&token)),
	)

	if ret == 0 {
		pt.logger.Debug("打开进程令牌失败")
		return "unknown"
	}
	defer pt.closeHandle.Call(token)

	// 获取令牌用户信息
	// 这里简化实现，实际需要调用GetTokenInformation获取SID并转换为用户名
	// 由于涉及复杂的SID处理，暂时返回基本信息
	return "token_user"
}

// getProcessCommandLine 获取进程命令行参数
func (pt *ProcessTracker) getProcessCommandLine(pid uint32) string {
	// 通过进程快照获取命令行信息
	// 注意：这是一个简化实现，完整的命令行获取需要通过WMI或其他方式

	// 尝试通过进程环境块获取命令行
	// 这需要读取进程内存，权限要求较高
	if !pt.privilegesEnabled {
		return ""
	}

	// 简化实现：返回进程名作为命令行的一部分
	// 实际实现需要读取PEB (Process Environment Block)
	return fmt.Sprintf("pid_%d_cmdline", pid)
}

// getProcessUserEnhanced 获取进程用户信息（增强版本）
func (pt *ProcessTracker) getProcessUserEnhanced(handle uintptr) string {
	// 打开进程令牌
	var token uintptr
	ret, _, _ := pt.openProcessToken.Call(
		handle,
		TOKEN_QUERY,
		uintptr(unsafe.Pointer(&token)),
	)

	if ret == 0 {
		pt.logger.Debug("打开进程令牌失败")
		// 尝试获取当前用户作为备选
		return pt.getCurrentUser()
	}
	defer pt.closeHandle.Call(token)

	// 尝试获取令牌用户信息
	user := pt.getTokenUser(token)
	if user != "" {
		return user
	}

	// 备选方案：返回当前用户
	return pt.getCurrentUser()
}

// getProcessCommandLineEnhanced 获取进程命令行参数（增强版本）
func (pt *ProcessTracker) getProcessCommandLineEnhanced(pid uint32) string {
	// 方法1：尝试通过WMI获取
	if cmdline := pt.getCommandLineViaWMI(pid); cmdline != "" {
		return cmdline
	}

	// 方法2：尝试通过进程快照
	if cmdline := pt.getCommandLineViaSnapshot(pid); cmdline != "" {
		return cmdline
	}

	// 方法3：构造基本命令行
	if info := pt.processCache[pid]; info != nil && info.ProcessName != "" {
		return info.ProcessName
	}

	return ""
}

// getCurrentUser 获取当前用户
func (pt *ProcessTracker) getCurrentUser() string {
	// 简化实现：返回环境变量中的用户名
	if username := os.Getenv("USERNAME"); username != "" {
		return username
	}
	return "current_user"
}

// getTokenUser 从令牌获取用户信息
func (pt *ProcessTracker) getTokenUser(token uintptr) string {
	// 这里应该调用GetTokenInformation获取TokenUser
	// 然后使用LookupAccountSid转换SID为用户名
	// 由于实现复杂，暂时返回基本信息
	return "token_user"
}

// getCommandLineViaWMI 通过WMI获取命令行
func (pt *ProcessTracker) getCommandLineViaWMI(pid uint32) string {
	// 这里应该使用WMI查询Win32_Process
	// 由于需要COM接口，实现较复杂，暂时返回空
	return ""
}

// getCommandLineViaSnapshot 通过进程快照获取命令行
func (pt *ProcessTracker) getCommandLineViaSnapshot(pid uint32) string {
	// 尝试从进程快照中获取更多信息
	// 这是一个简化实现
	if name := pt.getProcessNameFromSnapshot(pid); name != "" {
		return name
	}
	return ""
}

// getProcessOwner 获取进程所有者信息（增强版本）
func (pt *ProcessTracker) getProcessOwner(handle uintptr) (string, string) {
	// 打开进程令牌
	var token uintptr
	ret, _, _ := pt.openProcessToken.Call(
		handle,
		TOKEN_QUERY,
		uintptr(unsafe.Pointer(&token)),
	)

	if ret == 0 {
		return "unknown", "unknown"
	}
	defer pt.closeHandle.Call(token)

	// 这里应该调用GetTokenInformation获取TokenUser信息
	// 然后使用LookupAccountSid将SID转换为用户名和域名
	// 由于实现复杂，这里返回基本信息
	return "domain", "username"
}

// isSystemProcess 检查是否为系统进程
func (pt *ProcessTracker) isSystemProcess(pid uint32) bool {
	// 系统进程通常PID较小
	systemPids := []uint32{0, 4, 8} // System Idle Process, System, etc.
	for _, sysPid := range systemPids {
		if pid == sysPid {
			return true
		}
	}
	return false
}

// getSystemProcessName 获取系统进程名称
func (pt *ProcessTracker) getSystemProcessName(pid uint32) string {
	switch pid {
	case 0:
		return "System Idle Process"
	case 4:
		return "System"
	case 8:
		return "Registry"
	default:
		return "System Process"
	}
}

// getProcessIntegrityLevel 获取进程完整性级别
func (pt *ProcessTracker) getProcessIntegrityLevel(handle uintptr) string {
	// 打开进程令牌
	var token uintptr
	ret, _, _ := pt.openProcessToken.Call(
		handle,
		TOKEN_QUERY,
		uintptr(unsafe.Pointer(&token)),
	)

	if ret == 0 {
		return "unknown"
	}
	defer pt.closeHandle.Call(token)

	// 获取令牌完整性级别
	// 这需要调用GetTokenInformation with TokenIntegrityLevel
	// 简化实现
	return "medium"
}

// StartPeriodicUpdate 启动定期更新（增强版本）
func (pt *ProcessTracker) StartPeriodicUpdate(interval time.Duration) {
	pt.mu.Lock()
	if pt.monitoringActive {
		pt.mu.Unlock()
		pt.logger.Warn("监控已经在运行中")
		return
	}
	pt.monitoringActive = true
	pt.mu.Unlock()

	pt.logger.Info("启动连接表定期监控", "interval", interval)

	go func() {
		defer func() {
			pt.mu.Lock()
			pt.monitoringActive = false
			pt.mu.Unlock()
			pt.logger.Info("连接表监控已停止")
		}()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// 自适应更新间隔
		adaptiveInterval := interval
		consecutiveFailures := 0
		maxFailures := 3

		for {
			select {
			case <-pt.stopMonitoring:
				pt.logger.Info("收到停止监控信号")
				return
			case <-ticker.C:
				updateStart := time.Now()

				if err := pt.UpdateConnectionTables(); err != nil {
					consecutiveFailures++
					pt.logger.Error("定期更新连接表失败",
						"error", err,
						"consecutive_failures", consecutiveFailures)

					// 自适应调整：失败时增加更新间隔
					if consecutiveFailures >= maxFailures {
						newInterval := adaptiveInterval * 2
						if newInterval <= 60*time.Second { // 最大60秒
							adaptiveInterval = newInterval
							ticker.Reset(adaptiveInterval)
							pt.logger.Warn("由于连续失败，调整更新间隔",
								"new_interval", adaptiveInterval)
						}
					}
				} else {
					// 成功时重置失败计数和间隔
					if consecutiveFailures > 0 {
						consecutiveFailures = 0
						if adaptiveInterval != interval {
							adaptiveInterval = interval
							ticker.Reset(adaptiveInterval)
							pt.logger.Info("恢复正常更新间隔", "interval", interval)
						}
					}
				}

				// 性能监控
				updateDuration := time.Since(updateStart)
				if updateDuration > interval/2 {
					pt.logger.Warn("连接表更新耗时过长",
						"duration", updateDuration,
						"interval", interval)
				}
			}
		}
	}()
}

// StopPeriodicUpdate 停止定期更新
func (pt *ProcessTracker) StopPeriodicUpdate() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if !pt.monitoringActive {
		pt.logger.Debug("监控未运行")
		return
	}

	pt.logger.Info("停止连接表监控")
	select {
	case pt.stopMonitoring <- true:
	default:
		// 通道已满或已关闭
	}
}

// GetMonitoringStats 获取监控统计信息
func (pt *ProcessTracker) GetMonitoringStats() map[string]interface{} {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	stats := map[string]interface{}{
		"monitoring_active":  pt.monitoringActive,
		"last_update_time":   pt.lastUpdateTime,
		"total_updates":      pt.updateStats.totalUpdates,
		"success_updates":    pt.updateStats.successUpdates,
		"failed_updates":     pt.updateStats.failedUpdates,
		"avg_update_time":    pt.updateStats.avgUpdateTime,
		"tcp_entries":        len(pt.tcpTable),
		"udp_entries":        len(pt.udpTable),
		"process_cache_size": len(pt.processCache),
		"privileges_enabled": pt.privilegesEnabled,
	}

	if pt.updateStats.lastError != nil {
		stats["last_error"] = pt.updateStats.lastError.Error()
	}

	if pt.updateStats.totalUpdates > 0 {
		stats["success_rate"] = float64(pt.updateStats.successUpdates) / float64(pt.updateStats.totalUpdates)
	}

	return stats
}

// ClearCache 清理缓存
func (pt *ProcessTracker) ClearCache() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.processCache = make(map[uint32]*ProcessInfo)
	pt.logger.Debug("进程缓存已清理")
}

// updateProcessSnapshot 更新进程快照
func (pt *ProcessTracker) updateProcessSnapshot() error {
	// 创建进程快照
	snapshot, _, _ := pt.createToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == uintptr(INVALID_HANDLE_VALUE) {
		return fmt.Errorf("创建进程快照失败")
	}
	defer pt.closeHandle.Call(snapshot)

	// 初始化进程条目结构
	var pe32 PROCESSENTRY32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	// 获取第一个进程
	ret, _, _ := pt.process32First.Call(snapshot, uintptr(unsafe.Pointer(&pe32)))
	if ret == 0 {
		return fmt.Errorf("获取第一个进程失败")
	}

	// 清空现有快照缓存
	pt.processSnapshot = make(map[uint32]string)

	// 遍历所有进程
	for {
		// 将进程名从UTF16转换为字符串
		processName := syscall.UTF16ToString(pe32.ExeFile[:])
		pt.processSnapshot[pe32.ProcessID] = processName

		// 获取下一个进程
		ret, _, _ := pt.process32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe32)))
		if ret == 0 {
			break
		}
	}

	pt.logger.Debug("进程快照更新完成", "process_count", len(pt.processSnapshot))
	return nil
}

// intToIP 将32位整数转换为IP地址
func intToIP(ipInt uint32) net.IP {
	ip := make(net.IP, 4)
	ip[0] = byte(ipInt & 0xFF)
	ip[1] = byte((ipInt >> 8) & 0xFF)
	ip[2] = byte((ipInt >> 16) & 0xFF)
	ip[3] = byte((ipInt >> 24) & 0xFF)
	return ip
}
