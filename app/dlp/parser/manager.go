package parser

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lomehong/kennel/app/dlp/interceptor"

	"github.com/lomehong/kennel/pkg/logging"
)

// ProtocolManagerImpl 协议解析管理器实现
type ProtocolManagerImpl struct {
	parsers        map[string]ProtocolParser
	sessionManager SessionManager
	stats          ParserStats
	logger         logging.Logger
	config         ParserConfig
	running        int32
	mu             sync.RWMutex
}

// NewProtocolManager 创建协议解析管理器
func NewProtocolManager(logger logging.Logger, config ParserConfig) ProtocolManager {
	return &ProtocolManagerImpl{
		parsers:        make(map[string]ProtocolParser),
		sessionManager: NewSessionManager(logger, config),
		logger:         logger,
		config:         config,
		stats: ParserStats{
			ParserStats: make(map[string]uint64),
			StartTime:   time.Now(),
		},
	}
}

// RegisterParser 注册解析器
func (pm *ProtocolManagerImpl) RegisterParser(parser ProtocolParser) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	info := parser.GetParserInfo()
	for _, protocol := range info.SupportedProtocols {
		if _, exists := pm.parsers[protocol]; exists {
			return fmt.Errorf("协议解析器已存在: %s", protocol)
		}
		pm.parsers[protocol] = parser
		pm.stats.ParserStats[protocol] = 0
	}

	pm.logger.Info("注册协议解析器",
		"name", info.Name,
		"version", info.Version,
		"protocols", info.SupportedProtocols)

	return nil
}

// GetParser 获取解析器
func (pm *ProtocolManagerImpl) GetParser(protocol string) (ProtocolParser, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	parser, exists := pm.parsers[protocol]
	return parser, exists
}

// ParsePacket 解析数据包
func (pm *ProtocolManagerImpl) ParsePacket(packet *interceptor.PacketInfo) (*ParsedData, error) {
	atomic.AddUint64(&pm.stats.TotalPackets, 1)

	// 自动识别协议 - 使用优先级排序
	var parser ProtocolParser
	var protocol string

	pm.mu.RLock()

	// 定义协议解析器优先级顺序（从高到低）
	// 注意：http应该在https之前检查，避免HTTP流量被误判为TLS
	protocolPriority := []string{"http", "https", "ftp", "smtp", "mysql"}

	// 按优先级顺序查找匹配的解析器
	for _, proto := range protocolPriority {
		if p, exists := pm.parsers[proto]; exists && p.CanParse(packet) {
			parser = p
			protocol = proto
			pm.logger.Debug("找到匹配的协议解析器", "protocol", proto, "packet_size", packet.Size, "dest_port", packet.DestPort)
			break
		}
	}

	// 如果优先级列表中没有找到，再检查其他解析器
	if parser == nil {
		for proto, p := range pm.parsers {
			if proto != "unknown" && proto != "default" && !contains(protocolPriority, proto) && p.CanParse(packet) {
				parser = p
				protocol = proto
				pm.logger.Debug("找到其他协议解析器", "protocol", proto, "packet_size", packet.Size)
				break
			}
		}
	}

	// 如果没有找到特定解析器，使用默认解析器
	if parser == nil {
		// 优先使用"default"协议解析器
		if defaultParser, exists := pm.parsers["default"]; exists {
			parser = defaultParser
			protocol = "default"
			pm.logger.Debug("使用默认协议解析器",
				"packet_size", packet.Size,
				"dest_port", packet.DestPort,
				"source_port", packet.SourcePort)
		} else if unknownParser, exists := pm.parsers["unknown"]; exists {
			parser = unknownParser
			protocol = "unknown"
			pm.logger.Debug("使用未知协议解析器",
				"packet_size", packet.Size,
				"dest_port", packet.DestPort,
				"source_port", packet.SourcePort)
		}
	}
	pm.mu.RUnlock()

	// 如果仍然没有找到解析器，这是一个严重错误
	if parser == nil {
		atomic.AddUint64(&pm.stats.FailedPackets, 1)
		pm.logger.Error("严重错误：未找到任何协议解析器（包括默认解析器）",
			"dest_port", packet.DestPort,
			"source_port", packet.SourcePort,
			"payload_size", len(packet.Payload),
			"source_ip", packet.SourceIP.String(),
			"dest_ip", packet.DestIP.String(),
			"registered_parsers", pm.GetSupportedProtocols())
		return nil, fmt.Errorf("严重错误：未找到任何协议解析器（包括默认解析器）")
	}

	// 解析数据包
	data, err := parser.Parse(packet)
	if err != nil {
		// 如果特定协议解析失败，尝试使用默认解析器
		if protocol != "default" && protocol != "unknown" {
			pm.logger.Debug("特定协议解析失败，尝试使用默认解析器",
				"original_protocol", protocol,
				"error", err)

			// 尝试使用默认解析器
			pm.mu.RLock()
			if defaultParser, exists := pm.parsers["default"]; exists {
				pm.mu.RUnlock()
				defaultData, defaultErr := defaultParser.Parse(packet)
				if defaultErr == nil {
					atomic.AddUint64(&pm.stats.ParsedPackets, 1)
					atomic.AddUint64(&pm.stats.BytesProcessed, uint64(packet.Size))

					pm.mu.Lock()
					pm.stats.ParserStats["default"]++
					pm.mu.Unlock()

					pm.logger.Debug("默认解析器解析成功",
						"original_protocol", protocol,
						"dest_port", packet.DestPort)
					return defaultData, nil
				}
			} else {
				pm.mu.RUnlock()
			}
		}

		atomic.AddUint64(&pm.stats.FailedPackets, 1)
		pm.stats.LastError = err
		pm.logger.Error("协议解析失败",
			"protocol", protocol,
			"dest_port", packet.DestPort,
			"source_port", packet.SourcePort,
			"payload_size", len(packet.Payload),
			"source_ip", packet.SourceIP.String(),
			"dest_ip", packet.DestIP.String(),
			"error", err)
		return nil, fmt.Errorf("协议【%s】解析失败: %w", protocol, err)
	}

	// 更新统计信息
	atomic.AddUint64(&pm.stats.ParsedPackets, 1)
	atomic.AddUint64(&pm.stats.BytesProcessed, uint64(packet.Size))

	pm.mu.Lock()
	pm.stats.ParserStats[protocol]++
	pm.mu.Unlock()

	// 更新会话信息
	if data.Sessions != nil && len(data.Sessions) > 0 {
		for _, session := range data.Sessions {
			pm.sessionManager.UpdateSession(session.ID, packet)
		}
	}

	pm.logger.Debug("解析数据包成功",
		"protocol", protocol,
		"packet_id", packet.ID,
		"size", packet.Size)

	return data, nil
}

// GetSupportedProtocols 获取支持的协议列表
func (pm *ProtocolManagerImpl) GetSupportedProtocols() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	protocols := make([]string, 0, len(pm.parsers))
	for protocol := range pm.parsers {
		protocols = append(protocols, protocol)
	}

	return protocols
}

// GetStats 获取统计信息
func (pm *ProtocolManagerImpl) GetStats() ParserStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := pm.stats
	stats.Uptime = time.Since(pm.stats.StartTime)
	stats.ActiveSessions = uint64(len(pm.sessionManager.GetActiveSessions()))

	return stats
}

// Start 启动管理器
func (pm *ProtocolManagerImpl) Start() error {
	if !atomic.CompareAndSwapInt32(&pm.running, 0, 1) {
		return fmt.Errorf("协议管理器已在运行")
	}

	pm.logger.Info("启动协议解析管理器")

	// 初始化所有解析器
	pm.mu.RLock()
	for protocol, parser := range pm.parsers {
		if err := parser.Initialize(pm.config); err != nil {
			pm.mu.RUnlock()
			return fmt.Errorf("初始化解析器失败 [%s]: %w", protocol, err)
		}
	}
	pm.mu.RUnlock()

	// 启动会话清理协程
	go pm.sessionCleanupWorker()

	pm.logger.Info("协议解析管理器已启动")
	return nil
}

// Stop 停止管理器
func (pm *ProtocolManagerImpl) Stop() error {
	if !atomic.CompareAndSwapInt32(&pm.running, 1, 0) {
		return fmt.Errorf("协议管理器未在运行")
	}

	pm.logger.Info("停止协议解析管理器")

	// 清理所有解析器
	pm.mu.RLock()
	for protocol, parser := range pm.parsers {
		if err := parser.Cleanup(); err != nil {
			pm.logger.Error("清理解析器失败", "protocol", protocol, "error", err)
		}
	}
	pm.mu.RUnlock()

	pm.logger.Info("协议解析管理器已停止")
	return nil
}

// sessionCleanupWorker 会话清理工作协程
func (pm *ProtocolManagerImpl) sessionCleanupWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		if atomic.LoadInt32(&pm.running) == 0 {
			return
		}

		select {
		case <-ticker.C:
			cleaned := pm.sessionManager.CleanupExpiredSessions()
			if cleaned > 0 {
				pm.logger.Debug("清理过期会话", "count", cleaned)
			}
		}
	}
}

// SessionManagerImpl 会话管理器实现
type SessionManagerImpl struct {
	sessions map[string]*SessionInfo
	config   ParserConfig
	logger   logging.Logger
	stats    SessionStats
	mu       sync.RWMutex
}

// NewSessionManager 创建会话管理器
func NewSessionManager(logger logging.Logger, config ParserConfig) SessionManager {
	return &SessionManagerImpl{
		sessions: make(map[string]*SessionInfo),
		config:   config,
		logger:   logger,
	}
}

// CreateSession 创建会话
func (sm *SessionManagerImpl) CreateSession(packet *interceptor.PacketInfo) *SessionInfo {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionID := fmt.Sprintf("%s:%d-%s:%d",
		packet.SourceIP.String(), packet.SourcePort,
		packet.DestIP.String(), packet.DestPort)

	session := &SessionInfo{
		ID:          sessionID,
		Protocol:    getProtocolName(packet.Protocol),
		SourceIP:    packet.SourceIP.String(),
		DestIP:      packet.DestIP.String(),
		SourcePort:  packet.SourcePort,
		DestPort:    packet.DestPort,
		StartTime:   packet.Timestamp,
		LastSeen:    packet.Timestamp,
		BytesSent:   0,
		BytesRecv:   0,
		PacketCount: 1,
		State:       SessionStateNew,
		Metadata:    make(map[string]interface{}),
	}

	// 检查会话数量限制
	if len(sm.sessions) >= sm.config.MaxSessions {
		// 删除最旧的会话
		sm.removeOldestSession()
	}

	sm.sessions[sessionID] = session
	sm.stats.TotalSessions++
	sm.stats.ActiveSessions++

	sm.logger.Debug("创建会话", "session_id", sessionID)
	return session
}

// GetSession 获取会话
func (sm *SessionManagerImpl) GetSession(sessionID string) (*SessionInfo, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

// UpdateSession 更新会话
func (sm *SessionManagerImpl) UpdateSession(sessionID string, packet *interceptor.PacketInfo) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}

	session.LastSeen = packet.Timestamp
	session.PacketCount++

	// 更新字节统计
	if packet.Direction == interceptor.PacketDirectionOutbound {
		session.BytesSent += uint64(packet.Size)
	} else {
		session.BytesRecv += uint64(packet.Size)
	}

	// 更新会话状态
	if session.State == SessionStateNew {
		session.State = SessionStateEstablished
	}

	return nil
}

// CloseSession 关闭会话
func (sm *SessionManagerImpl) CloseSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("会话不存在: %s", sessionID)
	}

	session.State = SessionStateClosed
	delete(sm.sessions, sessionID)
	sm.stats.ActiveSessions--
	sm.stats.ClosedSessions++

	sm.logger.Debug("关闭会话", "session_id", sessionID)
	return nil
}

// GetActiveSessions 获取活跃会话
func (sm *SessionManagerImpl) GetActiveSessions() []*SessionInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*SessionInfo, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		if session.State == SessionStateEstablished {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// CleanupExpiredSessions 清理过期会话
func (sm *SessionManagerImpl) CleanupExpiredSessions() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for sessionID, session := range sm.sessions {
		if now.Sub(session.LastSeen) > sm.config.SessionTimeout {
			expired = append(expired, sessionID)
		}
	}

	for _, sessionID := range expired {
		delete(sm.sessions, sessionID)
		sm.stats.ActiveSessions--
		sm.stats.ExpiredSessions++
	}

	return len(expired)
}

// GetStats 获取统计信息
func (sm *SessionManagerImpl) GetStats() SessionStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.stats
}

// removeOldestSession 删除最旧的会话
func (sm *SessionManagerImpl) removeOldestSession() {
	var oldestID string
	var oldestTime time.Time

	for sessionID, session := range sm.sessions {
		if oldestID == "" || session.StartTime.Before(oldestTime) {
			oldestID = sessionID
			oldestTime = session.StartTime
		}
	}

	if oldestID != "" {
		delete(sm.sessions, oldestID)
		sm.stats.ActiveSessions--
		sm.stats.ExpiredSessions++
	}
}

// getProtocolName 获取协议名称
func getProtocolName(protocol interceptor.Protocol) string {
	switch protocol {
	case interceptor.ProtocolTCP:
		return "TCP"
	case interceptor.ProtocolUDP:
		return "UDP"
	default:
		return "Unknown"
	}
}

// contains 检查字符串切片是否包含指定字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
