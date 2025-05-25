//go:build windows

package interceptor

import (
	"fmt"
	"net"
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

// ProcessTracker 进程跟踪器
type ProcessTracker struct {
	logger       logging.Logger
	tcpTable     map[string]uint32 // "ip:port" -> PID
	udpTable     map[string]uint32 // "ip:port" -> PID
	processCache map[uint32]*ProcessInfo
	mu           sync.RWMutex

	// Windows API
	iphlpapi *syscall.LazyDLL
	kernel32 *syscall.LazyDLL
	psapi    *syscall.LazyDLL

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
}

// NewProcessTracker 创建进程跟踪器
func NewProcessTracker(logger logging.Logger) *ProcessTracker {
	pt := &ProcessTracker{
		logger:       logger,
		tcpTable:     make(map[string]uint32),
		udpTable:     make(map[string]uint32),
		processCache: make(map[uint32]*ProcessInfo),
	}

	// 加载Windows API
	pt.loadWindowsAPIs()

	return pt
}

// loadWindowsAPIs 加载Windows API
func (pt *ProcessTracker) loadWindowsAPIs() {
	pt.iphlpapi = syscall.NewLazyDLL("iphlpapi.dll")
	pt.kernel32 = syscall.NewLazyDLL("kernel32.dll")
	pt.psapi = syscall.NewLazyDLL("psapi.dll")

	pt.getExtendedTcpTable = pt.iphlpapi.NewProc("GetExtendedTcpTable")
	pt.getExtendedUdpTable = pt.iphlpapi.NewProc("GetExtendedUdpTable")
	pt.createToolhelp32Snapshot = pt.kernel32.NewProc("CreateToolhelp32Snapshot")
	pt.process32First = pt.kernel32.NewProc("Process32FirstW")
	pt.process32Next = pt.kernel32.NewProc("Process32NextW")
	pt.closeHandle = pt.kernel32.NewProc("CloseHandle")
	pt.openProcess = pt.kernel32.NewProc("OpenProcess")
	pt.getModuleFileNameEx = pt.psapi.NewProc("GetModuleFileNameExW")
	pt.queryFullProcessImageName = pt.kernel32.NewProc("QueryFullProcessImageNameW")
}

// UpdateConnectionTables 更新连接表
func (pt *ProcessTracker) UpdateConnectionTables() error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// 更新TCP连接表
	if err := pt.updateTCPTable(); err != nil {
		pt.logger.Error("更新TCP连接表失败", "error", err)
		return err
	}

	// 更新UDP连接表
	if err := pt.updateUDPTable(); err != nil {
		pt.logger.Error("更新UDP连接表失败", "error", err)
		return err
	}

	pt.logger.Debug("连接表更新完成",
		"tcp_entries", len(pt.tcpTable),
		"udp_entries", len(pt.udpTable))

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

	if ret != 122 { // ERROR_INSUFFICIENT_BUFFER
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

	if ret != 122 { // ERROR_INSUFFICIENT_BUFFER
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

// GetProcessByConnection 根据连接信息获取进程ID
func (pt *ProcessTracker) GetProcessByConnection(protocol Protocol, localIP net.IP, localPort uint16) uint32 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", localIP.String(), localPort)

	switch protocol {
	case ProtocolTCP:
		if pid, exists := pt.tcpTable[key]; exists {
			return pid
		}
	case ProtocolUDP:
		if pid, exists := pt.udpTable[key]; exists {
			return pid
		}
	}

	return 0
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

// getProcessDetails 获取进程详细信息
func (pt *ProcessTracker) getProcessDetails(pid uint32) *ProcessInfo {
	const PROCESS_QUERY_INFORMATION = 0x0400
	const PROCESS_VM_READ = 0x0010

	// 打开进程句柄
	handle, _, _ := pt.openProcess.Call(
		PROCESS_QUERY_INFORMATION|PROCESS_VM_READ,
		0, // bInheritHandle
		uintptr(pid),
	)

	if handle == 0 {
		return nil
	}

	defer pt.closeHandle.Call(handle)

	// 获取进程可执行文件路径
	var pathBuffer [260]uint16
	var pathSize uint32 = 260

	ret, _, _ := pt.queryFullProcessImageName.Call(
		handle,
		0, // dwFlags
		uintptr(unsafe.Pointer(&pathBuffer[0])),
		uintptr(unsafe.Pointer(&pathSize)),
	)

	var execPath string
	if ret != 0 {
		execPath = syscall.UTF16ToString(pathBuffer[:pathSize])
	}

	// 获取进程名称
	processName := "unknown"
	if execPath != "" {
		// 从路径中提取文件名
		for i := len(execPath) - 1; i >= 0; i-- {
			if execPath[i] == '\\' || execPath[i] == '/' {
				processName = execPath[i+1:]
				break
			}
		}
	}

	return &ProcessInfo{
		PID:         int(pid),
		ProcessName: processName,
		ExecutePath: execPath,
		User:        "unknown", // 需要额外API获取用户信息
		CommandLine: "",        // 需要额外API获取命令行
	}
}

// StartPeriodicUpdate 启动定期更新
func (pt *ProcessTracker) StartPeriodicUpdate(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := pt.UpdateConnectionTables(); err != nil {
				pt.logger.Error("定期更新连接表失败", "error", err)
			}
		}
	}()
}

// ClearCache 清理缓存
func (pt *ProcessTracker) ClearCache() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.processCache = make(map[uint32]*ProcessInfo)
	pt.logger.Debug("进程缓存已清理")
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
