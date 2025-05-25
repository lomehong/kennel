package interceptor

import (
	"net"
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// ETWDataSource ETW数据源实现
type ETWDataSource struct {
	logger           logging.Logger
	etwMonitor       ETWNetworkMonitor
	connectionMapper ConnectionMapper
	running          bool
	mu               sync.RWMutex
	
	// 统计信息
	stats struct {
		queriesHandled   int64
		successfulLookups int64
		failedLookups    int64
		lastQueryTime    time.Time
		mu               sync.RWMutex
	}
}

// NewETWDataSource 创建新的ETW数据源
func NewETWDataSource(logger logging.Logger, etwMonitor ETWNetworkMonitor, connectionMapper ConnectionMapper) ProcessDataSource {
	ds := &ETWDataSource{
		logger:           logger,
		etwMonitor:       etwMonitor,
		connectionMapper: connectionMapper,
	}
	
	// 启动ETW事件处理
	ds.startEventProcessing()
	
	return ds
}

// GetProcessInfo 获取进程信息
func (ds *ETWDataSource) GetProcessInfo(packet *PacketInfo) *ProcessInfo {
	ds.stats.mu.Lock()
	ds.stats.queriesHandled++
	ds.stats.lastQueryTime = time.Now()
	ds.stats.mu.Unlock()
	
	if packet == nil {
		return nil
	}
	
	// 创建连接信息
	conn := &ConnectionInfo{
		Protocol:  packet.Protocol,
		LocalAddr: &net.TCPAddr{IP: packet.SourceIP, Port: int(packet.SourcePort)},
		RemoteAddr: &net.TCPAddr{IP: packet.DestIP, Port: int(packet.DestPort)},
		State:     ConnectionStateEstablished,
		Timestamp: packet.Timestamp,
	}
	
	// 从连接映射器查找进程信息
	processInfo := ds.connectionMapper.GetProcessByConnection(conn)
	
	if processInfo != nil {
		ds.stats.mu.Lock()
		ds.stats.successfulLookups++
		ds.stats.mu.Unlock()
		
		ds.logger.Debug("ETW数据源找到进程信息",
			"pid", processInfo.PID,
			"process_name", processInfo.ProcessName,
			"source_ip", packet.SourceIP.String(),
			"dest_ip", packet.DestIP.String(),
			"source_port", packet.SourcePort,
			"dest_port", packet.DestPort,
		)
		
		return processInfo
	}
	
	// 尝试反向查找（交换源和目标地址）
	reverseConn := &ConnectionInfo{
		Protocol:  packet.Protocol,
		LocalAddr: &net.TCPAddr{IP: packet.DestIP, Port: int(packet.DestPort)},
		RemoteAddr: &net.TCPAddr{IP: packet.SourceIP, Port: int(packet.SourcePort)},
		State:     ConnectionStateEstablished,
		Timestamp: packet.Timestamp,
	}
	
	processInfo = ds.connectionMapper.GetProcessByConnection(reverseConn)
	
	if processInfo != nil {
		ds.stats.mu.Lock()
		ds.stats.successfulLookups++
		ds.stats.mu.Unlock()
		
		ds.logger.Debug("ETW数据源反向查找找到进程信息",
			"pid", processInfo.PID,
			"process_name", processInfo.ProcessName,
			"source_ip", packet.SourceIP.String(),
			"dest_ip", packet.DestIP.String(),
			"source_port", packet.SourcePort,
			"dest_port", packet.DestPort,
		)
		
		return processInfo
	}
	
	ds.stats.mu.Lock()
	ds.stats.failedLookups++
	ds.stats.mu.Unlock()
	
	ds.logger.Debug("ETW数据源未找到进程信息",
		"source_ip", packet.SourceIP.String(),
		"dest_ip", packet.DestIP.String(),
		"source_port", packet.SourcePort,
		"dest_port", packet.DestPort,
	)
	
	return nil
}

// Priority 返回数据源优先级
func (ds *ETWDataSource) Priority() int {
	return 100 // 高优先级，ETW数据最准确
}

// Name 返回数据源名称
func (ds *ETWDataSource) Name() string {
	return "ETW"
}

// startEventProcessing 启动ETW事件处理
func (ds *ETWDataSource) startEventProcessing() {
	ds.mu.Lock()
	if ds.running {
		ds.mu.Unlock()
		return
	}
	ds.running = true
	ds.mu.Unlock()
	
	go ds.eventProcessingLoop()
	ds.logger.Info("ETW数据源事件处理已启动")
}

// eventProcessingLoop ETW事件处理循环
func (ds *ETWDataSource) eventProcessingLoop() {
	defer func() {
		ds.mu.Lock()
		ds.running = false
		ds.mu.Unlock()
		ds.logger.Info("ETW数据源事件处理已停止")
	}()
	
	if !ds.etwMonitor.IsRunning() {
		ds.logger.Warn("ETW监听器未运行，启动ETW监听器")
		if err := ds.etwMonitor.Start(); err != nil {
			ds.logger.Error("启动ETW监听器失败", "error", err)
			return
		}
	}
	
	eventChan := ds.etwMonitor.GetEventChannel()
	
	for event := range eventChan {
		ds.processETWEvent(event)
	}
}

// processETWEvent 处理ETW事件
func (ds *ETWDataSource) processETWEvent(event *ETWNetworkEvent) {
	if event == nil || event.Connection == nil {
		return
	}
	
	// 创建进程信息
	processInfo := &ProcessInfo{
		PID:         int(event.ProcessID),
		ProcessName: event.ProcessName,
		ExecutePath: event.ProcessPath,
		User:        "unknown", // TODO: 从ETW事件中获取用户信息
		CommandLine: "",        // TODO: 从ETW事件中获取命令行信息
	}
	
	// 如果进程名为空，尝试从路径提取
	if processInfo.ProcessName == "" && processInfo.ExecutePath != "" {
		processInfo.ProcessName = extractProcessNameFromPath(processInfo.ExecutePath)
	}
	
	// 添加到连接映射器
	ds.connectionMapper.AddMapping(event.Connection, processInfo)
	
	ds.logger.Debug("处理ETW网络事件",
		"event_type", event.EventType.String(),
		"pid", event.ProcessID,
		"process_name", processInfo.ProcessName,
		"local_addr", event.Connection.LocalAddr.String(),
		"remote_addr", event.Connection.RemoteAddr.String(),
	)
}

// extractProcessNameFromPath 从路径中提取进程名
func extractProcessNameFromPath(path string) string {
	if path == "" {
		return ""
	}
	
	// 查找最后一个反斜杠
	lastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '\\' || path[i] == '/' {
			lastSlash = i
			break
		}
	}
	
	if lastSlash >= 0 && lastSlash < len(path)-1 {
		return path[lastSlash+1:]
	}
	
	return path
}

// Stop 停止ETW数据源
func (ds *ETWDataSource) Stop() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	
	if !ds.running {
		return nil
	}
	
	ds.logger.Info("停止ETW数据源")
	
	// 停止ETW监听器
	if ds.etwMonitor.IsRunning() {
		if err := ds.etwMonitor.Stop(); err != nil {
			ds.logger.Error("停止ETW监听器失败", "error", err)
			return err
		}
	}
	
	ds.running = false
	ds.logger.Info("ETW数据源已停止")
	return nil
}

// GetStats 获取统计信息
func (ds *ETWDataSource) GetStats() map[string]interface{} {
	ds.stats.mu.RLock()
	defer ds.stats.mu.RUnlock()
	
	ds.mu.RLock()
	isRunning := ds.running
	ds.mu.RUnlock()
	
	successRate := float64(0)
	if ds.stats.queriesHandled > 0 {
		successRate = float64(ds.stats.successfulLookups) / float64(ds.stats.queriesHandled) * 100
	}
	
	stats := map[string]interface{}{
		"name":               ds.Name(),
		"priority":           ds.Priority(),
		"is_running":         isRunning,
		"queries_handled":    ds.stats.queriesHandled,
		"successful_lookups": ds.stats.successfulLookups,
		"failed_lookups":     ds.stats.failedLookups,
		"success_rate":       successRate,
		"last_query_time":    ds.stats.lastQueryTime,
	}
	
	// 添加ETW监听器统计信息
	if etwImpl, ok := ds.etwMonitor.(*ETWNetworkMonitorImpl); ok {
		etwStats := etwImpl.GetStats()
		stats["etw_monitor"] = etwStats
	}
	
	// 添加连接映射器统计信息
	if mapperImpl, ok := ds.connectionMapper.(*ConnectionMapperImpl); ok {
		mapperStats := mapperImpl.GetStats()
		stats["connection_mapper"] = mapperStats
	}
	
	return stats
}

// IsRunning 检查是否正在运行
func (ds *ETWDataSource) IsRunning() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.running
}
