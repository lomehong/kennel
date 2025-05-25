package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

// ProcessInfoDiagnostic 进程信息诊断工具
type ProcessInfoDiagnostic struct {
	logger         logging.Logger
	processTracker *interceptor.ProcessTracker
}

// DiagnosticResult 诊断结果
type DiagnosticResult struct {
	Timestamp          time.Time                  `json:"timestamp"`
	SystemInfo         SystemInfo                 `json:"system_info"`
	ProcessTrackerTest ProcessTrackerTestResult   `json:"process_tracker_test"`
	NetworkConnections []NetworkConnectionInfo    `json:"network_connections"`
	ProcessList        []ProcessInfo              `json:"process_list"`
	ConnectionMapping  []ConnectionProcessMapping `json:"connection_mapping"`
	Issues             []DiagnosticIssue          `json:"issues"`
	Recommendations    []string                   `json:"recommendations"`
}

// SystemInfo 系统信息
type SystemInfo struct {
	OS              string `json:"os"`
	Architecture    string `json:"architecture"`
	IsAdmin         bool   `json:"is_admin"`
	ProcessCount    int    `json:"process_count"`
	ConnectionCount int    `json:"connection_count"`
}

// ProcessTrackerTestResult 进程跟踪器测试结果
type ProcessTrackerTestResult struct {
	Initialized         bool                   `json:"initialized"`
	ConnectionTableSize int                    `json:"connection_table_size"`
	ProcessCacheSize    int                    `json:"process_cache_size"`
	LastUpdateTime      time.Time              `json:"last_update_time"`
	TestConnections     []ConnectionTestResult `json:"test_connections"`
}

// NetworkConnectionInfo 网络连接信息
type NetworkConnectionInfo struct {
	Protocol    string `json:"protocol"`
	LocalIP     string `json:"local_ip"`
	LocalPort   uint16 `json:"local_port"`
	RemoteIP    string `json:"remote_ip"`
	RemotePort  uint16 `json:"remote_port"`
	State       string `json:"state"`
	PID         uint32 `json:"pid"`
	ProcessName string `json:"process_name,omitempty"`
}

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID         uint32 `json:"pid"`
	ProcessName string `json:"process_name"`
	ExecutePath string `json:"execute_path"`
	UserName    string `json:"user_name,omitempty"`
	SessionID   uint32 `json:"session_id,omitempty"`
}

// ConnectionProcessMapping 连接进程映射
type ConnectionProcessMapping struct {
	Connection NetworkConnectionInfo `json:"connection"`
	Process    ProcessInfo           `json:"process"`
	MappingOK  bool                  `json:"mapping_ok"`
	Error      string                `json:"error,omitempty"`
}

// ConnectionTestResult 连接测试结果
type ConnectionTestResult struct {
	TestName    string `json:"test_name"`
	Protocol    string `json:"protocol"`
	LocalIP     string `json:"local_ip"`
	LocalPort   uint16 `json:"local_port"`
	RemoteIP    string `json:"remote_ip"`
	RemotePort  uint16 `json:"remote_port"`
	ExpectedPID uint32 `json:"expected_pid"`
	ActualPID   uint32 `json:"actual_pid"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// DiagnosticIssue 诊断问题
type DiagnosticIssue struct {
	Severity    string `json:"severity"` // "critical", "warning", "info"
	Category    string `json:"category"` // "process_tracker", "permissions", "api_calls"
	Description string `json:"description"`
	Impact      string `json:"impact"`
	Solution    string `json:"solution"`
}

// NewProcessInfoDiagnostic 创建进程信息诊断工具
func NewProcessInfoDiagnostic() *ProcessInfoDiagnostic {
	logConfig := &logging.LogConfig{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	}
	logger, _ := logging.NewEnhancedLogger(logConfig)

	return &ProcessInfoDiagnostic{
		logger:         logger,
		processTracker: interceptor.NewProcessTracker(logger),
	}
}

// RunDiagnostic 运行诊断
func (d *ProcessInfoDiagnostic) RunDiagnostic() (*DiagnosticResult, error) {
	d.logger.Info("开始进程信息诊断")

	result := &DiagnosticResult{
		Timestamp:       time.Now(),
		Issues:          make([]DiagnosticIssue, 0),
		Recommendations: make([]string, 0),
	}

	// 1. 收集系统信息
	d.logger.Info("收集系统信息")
	result.SystemInfo = d.collectSystemInfo()

	// 2. 测试进程跟踪器
	d.logger.Info("测试进程跟踪器")
	result.ProcessTrackerTest = d.testProcessTracker()

	// 3. 收集网络连接信息
	d.logger.Info("收集网络连接信息")
	result.NetworkConnections = d.collectNetworkConnections()

	// 4. 收集进程列表
	d.logger.Info("收集进程列表")
	result.ProcessList = d.collectProcessList()

	// 5. 测试连接进程映射
	d.logger.Info("测试连接进程映射")
	result.ConnectionMapping = d.testConnectionMapping()

	// 6. 分析问题
	d.logger.Info("分析诊断结果")
	d.analyzeIssues(result)

	// 7. 生成建议
	d.generateRecommendations(result)

	d.logger.Info("诊断完成")
	return result, nil
}

// collectSystemInfo 收集系统信息
func (d *ProcessInfoDiagnostic) collectSystemInfo() SystemInfo {
	return SystemInfo{
		OS:           "Windows", // 简化实现
		Architecture: "x64",     // 简化实现
		IsAdmin:      d.isRunningAsAdmin(),
		// ProcessCount 和 ConnectionCount 将在后续步骤中填充
	}
}

// testProcessTracker 测试进程跟踪器
func (d *ProcessInfoDiagnostic) testProcessTracker() ProcessTrackerTestResult {
	result := ProcessTrackerTestResult{
		Initialized:     true,
		TestConnections: make([]ConnectionTestResult, 0),
	}

	// 测试常见的网络连接
	testCases := []struct {
		name       string
		protocol   interceptor.Protocol
		localIP    string
		localPort  uint16
		remoteIP   string
		remotePort uint16
	}{
		{"HTTP连接", interceptor.ProtocolTCP, "127.0.0.1", 12345, "8.8.8.8", 80},
		{"HTTPS连接", interceptor.ProtocolTCP, "192.168.1.100", 54321, "1.1.1.1", 443},
		{"DNS查询", interceptor.ProtocolUDP, "192.168.1.100", 53001, "8.8.8.8", 53},
	}

	for _, tc := range testCases {
		testResult := ConnectionTestResult{
			TestName:   tc.name,
			Protocol:   d.protocolToString(tc.protocol),
			LocalIP:    tc.localIP,
			LocalPort:  tc.localPort,
			RemoteIP:   tc.remoteIP,
			RemotePort: tc.remotePort,
		}

		// 测试进程查找
		localIP := net.ParseIP(tc.localIP)
		pid := d.processTracker.GetProcessByConnection(tc.protocol, localIP, tc.localPort)
		testResult.ActualPID = pid
		testResult.Success = pid != 0

		if !testResult.Success {
			testResult.Error = "未找到对应进程"
		}

		result.TestConnections = append(result.TestConnections, testResult)
	}

	return result
}

// collectNetworkConnections 收集网络连接信息
func (d *ProcessInfoDiagnostic) collectNetworkConnections() []NetworkConnectionInfo {
	// 简化实现，实际应该调用Windows API获取真实连接信息
	connections := make([]NetworkConnectionInfo, 0)

	// 示例连接（实际实现需要调用GetExtendedTcpTable等API）
	sampleConnections := []NetworkConnectionInfo{
		{
			Protocol:    "TCP",
			LocalIP:     "127.0.0.1",
			LocalPort:   80,
			RemoteIP:    "0.0.0.0",
			RemotePort:  0,
			State:       "LISTENING",
			PID:         1234,
			ProcessName: "httpd.exe",
		},
		{
			Protocol:    "TCP",
			LocalIP:     "192.168.1.100",
			LocalPort:   12345,
			RemoteIP:    "8.8.8.8",
			RemotePort:  443,
			State:       "ESTABLISHED",
			PID:         5678,
			ProcessName: "chrome.exe",
		},
	}

	connections = append(connections, sampleConnections...)
	return connections
}

// collectProcessList 收集进程列表
func (d *ProcessInfoDiagnostic) collectProcessList() []ProcessInfo {
	// 简化实现，实际应该调用Windows API获取真实进程信息
	processes := make([]ProcessInfo, 0)

	// 示例进程（实际实现需要调用EnumProcesses等API）
	sampleProcesses := []ProcessInfo{
		{
			PID:         1234,
			ProcessName: "httpd.exe",
			ExecutePath: "C:\\Apache\\bin\\httpd.exe",
			UserName:    "SYSTEM",
			SessionID:   0,
		},
		{
			PID:         5678,
			ProcessName: "chrome.exe",
			ExecutePath: "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			UserName:    "DOMAIN\\user",
			SessionID:   1,
		},
	}

	processes = append(processes, sampleProcesses...)
	return processes
}

// testConnectionMapping 测试连接进程映射
func (d *ProcessInfoDiagnostic) testConnectionMapping() []ConnectionProcessMapping {
	mappings := make([]ConnectionProcessMapping, 0)

	// 获取网络连接
	connections := d.collectNetworkConnections()
	processes := d.collectProcessList()

	// 创建进程映射
	processMap := make(map[uint32]ProcessInfo)
	for _, proc := range processes {
		processMap[proc.PID] = proc
	}

	// 测试每个连接的进程映射
	for _, conn := range connections {
		mapping := ConnectionProcessMapping{
			Connection: conn,
		}

		// 查找对应的进程
		if proc, exists := processMap[conn.PID]; exists {
			mapping.Process = proc
			mapping.MappingOK = true
		} else {
			mapping.MappingOK = false
			mapping.Error = fmt.Sprintf("未找到PID %d 对应的进程信息", conn.PID)
		}

		mappings = append(mappings, mapping)
	}

	return mappings
}

// analyzeIssues 分析问题
func (d *ProcessInfoDiagnostic) analyzeIssues(result *DiagnosticResult) {
	// 检查管理员权限
	if !result.SystemInfo.IsAdmin {
		result.Issues = append(result.Issues, DiagnosticIssue{
			Severity:    "critical",
			Category:    "permissions",
			Description: "程序未以管理员权限运行",
			Impact:      "无法获取完整的进程和网络连接信息",
			Solution:    "请以管理员身份重新运行程序",
		})
	}

	// 检查进程跟踪器测试结果
	failedTests := 0
	for _, test := range result.ProcessTrackerTest.TestConnections {
		if !test.Success {
			failedTests++
		}
	}

	if failedTests > 0 {
		result.Issues = append(result.Issues, DiagnosticIssue{
			Severity:    "critical",
			Category:    "process_tracker",
			Description: fmt.Sprintf("进程跟踪器测试失败 %d/%d", failedTests, len(result.ProcessTrackerTest.TestConnections)),
			Impact:      "无法正确关联网络连接到进程",
			Solution:    "检查ProcessTracker.GetProcessByConnection()方法实现",
		})
	}

	// 检查连接进程映射
	failedMappings := 0
	for _, mapping := range result.ConnectionMapping {
		if !mapping.MappingOK {
			failedMappings++
		}
	}

	if failedMappings > 0 {
		result.Issues = append(result.Issues, DiagnosticIssue{
			Severity:    "warning",
			Category:    "process_tracker",
			Description: fmt.Sprintf("连接进程映射失败 %d/%d", failedMappings, len(result.ConnectionMapping)),
			Impact:      "部分网络连接无法追踪到源进程",
			Solution:    "优化进程信息获取和缓存机制",
		})
	}
}

// generateRecommendations 生成建议
func (d *ProcessInfoDiagnostic) generateRecommendations(result *DiagnosticResult) {
	if !result.SystemInfo.IsAdmin {
		result.Recommendations = append(result.Recommendations, "以管理员身份运行程序以获取完整权限")
	}

	if len(result.Issues) > 0 {
		result.Recommendations = append(result.Recommendations, "修复ProcessTracker.GetProcessByConnection()方法实现")
		result.Recommendations = append(result.Recommendations, "增强Windows API调用权限处理")
		result.Recommendations = append(result.Recommendations, "实现实时网络连接表监控")
		result.Recommendations = append(result.Recommendations, "优化进程信息缓存策略")
	}

	result.Recommendations = append(result.Recommendations, "在审计日志中添加进程信息字段")
	result.Recommendations = append(result.Recommendations, "实现多策略进程查找算法")
}

// isRunningAsAdmin 检查是否以管理员身份运行
func (d *ProcessInfoDiagnostic) isRunningAsAdmin() bool {
	// 简化实现，尝试访问需要管理员权限的资源
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

// SaveReport 保存诊断报告
func (d *ProcessInfoDiagnostic) SaveReport(result *DiagnosticResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化诊断结果失败: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("保存诊断报告失败: %w", err)
	}

	d.logger.Info("诊断报告已保存", "filename", filename)
	return nil
}

// main 主函数
func main() {
	fmt.Println("=== DLP进程信息诊断工具 ===")
	fmt.Println()

	diagnostic := NewProcessInfoDiagnostic()

	// 运行诊断
	result, err := diagnostic.RunDiagnostic()
	if err != nil {
		fmt.Printf("诊断失败: %v\n", err)
		os.Exit(1)
	}

	// 显示结果摘要
	fmt.Printf("诊断时间: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("系统信息: %s %s (管理员权限: %v)\n",
		result.SystemInfo.OS,
		result.SystemInfo.Architecture,
		result.SystemInfo.IsAdmin)
	fmt.Printf("发现问题: %d 个\n", len(result.Issues))
	fmt.Printf("建议措施: %d 条\n", len(result.Recommendations))
	fmt.Println()

	// 显示问题
	if len(result.Issues) > 0 {
		fmt.Println("发现的问题:")
		for i, issue := range result.Issues {
			fmt.Printf("%d. [%s] %s\n", i+1, issue.Severity, issue.Description)
			fmt.Printf("   影响: %s\n", issue.Impact)
			fmt.Printf("   解决方案: %s\n", issue.Solution)
			fmt.Println()
		}
	}

	// 显示建议
	if len(result.Recommendations) > 0 {
		fmt.Println("建议措施:")
		for i, rec := range result.Recommendations {
			fmt.Printf("%d. %s\n", i+1, rec)
		}
		fmt.Println()
	}

	// 保存详细报告
	reportFile := fmt.Sprintf("process_info_diagnostic_%s.json",
		time.Now().Format("20060102_150405"))

	if err := diagnostic.SaveReport(result, reportFile); err != nil {
		fmt.Printf("保存报告失败: %v\n", err)
	} else {
		fmt.Printf("详细诊断报告已保存到: %s\n", reportFile)
	}

	fmt.Println("诊断完成。")
}

// protocolToString 将协议转换为字符串
func (d *ProcessInfoDiagnostic) protocolToString(protocol interceptor.Protocol) string {
	switch protocol {
	case interceptor.ProtocolTCP:
		return "TCP"
	case interceptor.ProtocolUDP:
		return "UDP"
	case 1: // ICMP
		return "ICMP"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(protocol))
	}
}
