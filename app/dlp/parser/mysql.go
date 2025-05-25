package parser

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

// MySQLParser MySQL协议解析器
type MySQLParser struct {
	logger   logging.Logger
	sessions map[string]*MySQLSession
}

// MySQLSession MySQL会话信息
type MySQLSession struct {
	SessionID          string
	State              MySQLState
	Database           string
	Username           string
	ConnectionID       uint32
	ServerVersion      string
	CharacterSet       uint8
	ServerCapabilities uint32
	CreatedAt          time.Time
	LastUsed           time.Time
}

// MySQLState MySQL会话状态
type MySQLState int

const (
	MySQLStateInit MySQLState = iota
	MySQLStateHandshake
	MySQLStateAuth
	MySQLStateConnected
	MySQLStateQuery
	MySQLStateResult
	MySQLStateClosed
)

// MySQLPacket MySQL数据包结构
type MySQLPacket struct {
	Length     uint32
	SequenceID uint8
	Payload    []byte
}

// MySQLCommand MySQL命令类型
type MySQLCommand uint8

const (
	ComSleep MySQLCommand = iota
	ComQuit
	ComInitDB
	ComQuery
	ComFieldList
	ComCreateDB
	ComDropDB
	ComRefresh
	ComShutdown
	ComStatistics
	ComProcessInfo
	ComConnect
	ComProcessKill
	ComDebug
	ComPing
	ComTime
	ComDelayedInsert
	ComChangeUser
	ComBinlogDump
	ComTableDump
	ComConnectOut
	ComRegisterSlave
	ComStmtPrepare
	ComStmtExecute
	ComStmtSendLongData
	ComStmtClose
	ComStmtReset
	ComSetOption
	ComStmtFetch
)

// NewMySQLParser 创建MySQL解析器
func NewMySQLParser(logger logging.Logger) *MySQLParser {
	return &MySQLParser{
		logger:   logger,
		sessions: make(map[string]*MySQLSession),
	}
}

// GetParserInfo 获取解析器信息
func (m *MySQLParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "MySQL Parser",
		Version:            "1.0.0",
		Description:        "MySQL协议解析器，支持SQL查询和结果解析",
		SupportedProtocols: []string{"mysql"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

// CanParse 检查是否能解析指定的数据包
func (m *MySQLParser) CanParse(packet *interceptor.PacketInfo) bool {
	if packet == nil || len(packet.Payload) < 5 {
		return false
	}

	// 检查是否是TCP协议
	if packet.Protocol != interceptor.ProtocolTCP {
		return false
	}

	// 首先检查端口（MySQL标准端口）
	if packet.DestPort == 3306 || packet.SourcePort == 3306 {
		// 对于MySQL端口，进一步验证协议特征
		return m.isMySQLPacket(packet.Payload) && m.validateMySQLContent(packet.Payload)
	}

	// 对于非标准端口，需要严格的协议特征检查
	if m.isMySQLPacket(packet.Payload) {
		// 额外验证：确保不是HTTP或其他协议的误判
		return m.validateMySQLContent(packet.Payload) && !m.isHTTPLikeContent(packet.Payload)
	}

	return false
}

// Parse 解析数据包
func (m *MySQLParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	if !m.CanParse(packet) {
		return nil, fmt.Errorf("不是有效的MySQL数据包")
	}

	parsedData := &ParsedData{
		Protocol:    "mysql",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    make(map[string]any),
		ContentType: "application/mysql",
	}

	// 获取或创建会话
	sessionID := m.getSessionID(packet)
	session := m.getOrCreateSession(sessionID, packet)

	// 解析MySQL数据包
	return m.parseMySQLPacket(packet.Payload, parsedData, session)
}

// GetSupportedProtocols 获取支持的协议列表
func (m *MySQLParser) GetSupportedProtocols() []string {
	return []string{"mysql"}
}

// Initialize 初始化解析器
func (m *MySQLParser) Initialize(config ParserConfig) error {
	m.logger.Info("初始化MySQL解析器")
	return nil
}

// Cleanup 清理资源
func (m *MySQLParser) Cleanup() error {
	m.logger.Info("清理MySQL解析器资源")
	m.sessions = make(map[string]*MySQLSession)
	return nil
}

// isMySQLPacket 检查是否为MySQL数据包
func (m *MySQLParser) isMySQLPacket(data []byte) bool {
	if len(data) < 5 {
		return false
	}

	// MySQL数据包格式: 长度(3字节) + 序列号(1字节) + 载荷
	length := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16

	// 检查长度是否合理
	if length > 16777215 || length == 0 {
		return false
	}

	// 检查是否为握手包 (协议版本10)
	if len(data) > 4 && data[4] == 0x0a {
		return true
	}

	// 检查是否为命令包
	if len(data) > 4 {
		command := data[4]
		return command <= uint8(ComStmtFetch)
	}

	return false
}

// validateMySQLContent 验证MySQL内容的有效性
func (m *MySQLParser) validateMySQLContent(data []byte) bool {
	if len(data) < 5 {
		return false
	}

	// 解析数据包头
	length := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
	sequenceID := data[3]

	// 验证长度字段
	if length == 0 || length > 16777215 {
		return false
	}

	// 验证数据包完整性
	if len(data) < int(4+length) {
		return false
	}

	// 验证序列号（通常从0开始，递增）
	if sequenceID > 100 { // 合理的序列号范围
		return false
	}

	// 检查载荷内容
	if len(data) > 4 {
		payload := data[4:]
		firstByte := payload[0]

		// 验证MySQL协议特定的字节模式
		switch firstByte {
		case 0x0a: // 握手包
			return m.validateHandshakePacket(payload)
		case 0x00: // OK包
			return len(payload) >= 7
		case 0xff: // 错误包
			return len(payload) >= 3
		default:
			// 命令包验证
			if firstByte <= uint8(ComStmtFetch) {
				return true
			}
		}
	}

	return false
}

// validateHandshakePacket 验证握手包
func (m *MySQLParser) validateHandshakePacket(payload []byte) bool {
	if len(payload) < 20 {
		return false
	}

	// 检查协议版本（应该是10）
	if payload[0] != 0x0a {
		return false
	}

	// 检查服务器版本字符串（应该以null结尾）
	versionEnd := 1
	for versionEnd < len(payload) && payload[versionEnd] != 0 {
		versionEnd++
		if versionEnd > 50 { // 版本字符串不应该太长
			return false
		}
	}

	return versionEnd < len(payload) && payload[versionEnd] == 0
}

// isHTTPLikeContent 检查是否像HTTP内容
func (m *MySQLParser) isHTTPLikeContent(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	content := string(data)

	// 检查HTTP方法
	httpMethods := []string{"GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH "}
	for _, method := range httpMethods {
		if strings.HasPrefix(content, method) {
			return true
		}
	}

	// 检查HTTP响应
	if strings.HasPrefix(content, "HTTP/") {
		return true
	}

	// 检查HTTP头部特征
	if strings.Contains(content, "Content-Type:") ||
		strings.Contains(content, "User-Agent:") ||
		strings.Contains(content, "Host:") {
		return true
	}

	return false
}

// parseMySQLPacket 解析MySQL数据包
func (m *MySQLParser) parseMySQLPacket(data []byte, parsedData *ParsedData, session *MySQLSession) (*ParsedData, error) {
	packet, err := m.parsePacketHeader(data)
	if err != nil {
		return nil, fmt.Errorf("解析MySQL数据包头失败: %w", err)
	}

	parsedData.Metadata["packet_length"] = packet.Length
	parsedData.Metadata["sequence_id"] = packet.SequenceID

	// 根据会话状态和数据包内容判断类型
	if len(packet.Payload) == 0 {
		return parsedData, nil
	}

	firstByte := packet.Payload[0]

	// 握手包检测 (协议版本10)
	if firstByte == 0x0a && session.State == MySQLStateInit {
		return m.parseHandshakePacket(packet.Payload, parsedData, session)
	}

	// 错误包检测
	if firstByte == 0xff {
		return m.parseErrorPacket(packet.Payload, parsedData)
	}

	// OK包检测
	if firstByte == 0x00 && len(packet.Payload) >= 7 {
		return m.parseOKPacket(packet.Payload, parsedData)
	}

	// 命令包检测
	if session.State == MySQLStateConnected {
		command := MySQLCommand(firstByte)
		return m.parseCommandPacket(command, packet.Payload[1:], parsedData, session)
	}

	// 认证响应包
	if session.State == MySQLStateHandshake {
		return m.parseAuthPacket(packet.Payload, parsedData, session)
	}

	// 结果集包
	if session.State == MySQLStateQuery {
		return m.parseResultPacket(packet.Payload, parsedData)
	}

	return parsedData, nil
}

// parsePacketHeader 解析数据包头
func (m *MySQLParser) parsePacketHeader(data []byte) (*MySQLPacket, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("数据包长度不足")
	}

	length := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
	sequenceID := data[3]

	if len(data) < int(4+length) {
		return nil, fmt.Errorf("数据包数据不完整")
	}

	packet := &MySQLPacket{
		Length:     length,
		SequenceID: sequenceID,
		Payload:    data[4 : 4+length],
	}

	return packet, nil
}

// parseHandshakePacket 解析握手包
func (m *MySQLParser) parseHandshakePacket(payload []byte, parsedData *ParsedData, session *MySQLSession) (*ParsedData, error) {
	if len(payload) < 20 {
		return nil, fmt.Errorf("握手包长度不足")
	}

	offset := 1 // 跳过协议版本

	// 解析服务器版本
	versionEnd := offset
	for versionEnd < len(payload) && payload[versionEnd] != 0 {
		versionEnd++
	}

	if versionEnd < len(payload) {
		session.ServerVersion = string(payload[offset:versionEnd])
		parsedData.Metadata["server_version"] = session.ServerVersion
		offset = versionEnd + 1
	}

	// 解析连接ID
	if offset+4 <= len(payload) {
		session.ConnectionID = binary.LittleEndian.Uint32(payload[offset : offset+4])
		parsedData.Metadata["connection_id"] = session.ConnectionID
		offset += 4
	}

	// 跳过认证数据和其他字段
	session.State = MySQLStateHandshake
	parsedData.Metadata["packet_type"] = "handshake"

	return parsedData, nil
}

// parseQueryCommand 解析查询命令
func (m *MySQLParser) parseQueryCommand(payload []byte, parsedData *ParsedData, session *MySQLSession) (*ParsedData, error) {
	if len(payload) == 0 {
		return parsedData, nil
	}

	sql := string(payload)
	parsedData.Metadata["sql"] = sql
	parsedData.Body = payload

	// 解析SQL语句
	sqlInfo := m.parseSQL(sql)
	for k, v := range sqlInfo {
		parsedData.Metadata[k] = v
	}

	session.State = MySQLStateQuery
	return parsedData, nil
}

// parseInitDBCommand 解析初始化数据库命令
func (m *MySQLParser) parseInitDBCommand(payload []byte, parsedData *ParsedData, session *MySQLSession) (*ParsedData, error) {
	if len(payload) == 0 {
		return parsedData, nil
	}

	database := string(payload)
	session.Database = database
	parsedData.Metadata["database"] = database
	parsedData.Headers["Database"] = database

	return parsedData, nil
}

// parseErrorPacket 解析错误包
func (m *MySQLParser) parseErrorPacket(payload []byte, parsedData *ParsedData) (*ParsedData, error) {
	if len(payload) < 3 {
		return parsedData, nil
	}

	errorCode := binary.LittleEndian.Uint16(payload[1:3])
	parsedData.Metadata["error_code"] = errorCode
	parsedData.Metadata["packet_type"] = "error"

	if len(payload) > 3 {
		errorMessage := string(payload[3:])
		parsedData.Metadata["error_message"] = errorMessage
	}

	return parsedData, nil
}

// parseOKPacket 解析OK包
func (m *MySQLParser) parseOKPacket(payload []byte, parsedData *ParsedData) (*ParsedData, error) {
	parsedData.Metadata["packet_type"] = "ok"

	if len(payload) >= 7 {
		// 解析受影响行数
		affectedRows, offset := m.readLengthEncodedInteger(payload, 1)
		parsedData.Metadata["affected_rows"] = affectedRows

		// 解析插入ID
		if offset < len(payload) {
			insertID, _ := m.readLengthEncodedInteger(payload, offset)
			parsedData.Metadata["insert_id"] = insertID
		}
	}

	return parsedData, nil
}

// parseResultPacket 解析结果包
func (m *MySQLParser) parseResultPacket(payload []byte, parsedData *ParsedData) (*ParsedData, error) {
	parsedData.Metadata["packet_type"] = "result"

	// 简单解析列数
	if len(payload) > 0 {
		columnCount, _ := m.readLengthEncodedInteger(payload, 0)
		parsedData.Metadata["column_count"] = columnCount
	}

	return parsedData, nil
}

// parseSQL 解析SQL语句
func (m *MySQLParser) parseSQL(sql string) map[string]any {
	result := make(map[string]any)

	// 清理SQL语句
	sql = strings.TrimSpace(sql)
	upperSQL := strings.ToUpper(sql)

	// 检测SQL类型
	if strings.HasPrefix(upperSQL, "SELECT") {
		result["sql_type"] = "SELECT"
		result["operation"] = "read"

		// 提取表名
		if tables := m.extractTablesFromSelect(sql); len(tables) > 0 {
			result["tables"] = tables
		}

	} else if strings.HasPrefix(upperSQL, "INSERT") {
		result["sql_type"] = "INSERT"
		result["operation"] = "write"

		if table := m.extractTableFromInsert(sql); table != "" {
			result["table"] = table
		}

	} else if strings.HasPrefix(upperSQL, "UPDATE") {
		result["sql_type"] = "UPDATE"
		result["operation"] = "write"

		if table := m.extractTableFromUpdate(sql); table != "" {
			result["table"] = table
		}

	} else if strings.HasPrefix(upperSQL, "DELETE") {
		result["sql_type"] = "DELETE"
		result["operation"] = "write"

		if table := m.extractTableFromDelete(sql); table != "" {
			result["table"] = table
		}

	} else if strings.HasPrefix(upperSQL, "CREATE") {
		result["sql_type"] = "CREATE"
		result["operation"] = "ddl"

	} else if strings.HasPrefix(upperSQL, "DROP") {
		result["sql_type"] = "DROP"
		result["operation"] = "ddl"

	} else if strings.HasPrefix(upperSQL, "ALTER") {
		result["sql_type"] = "ALTER"
		result["operation"] = "ddl"
	}

	// 检测敏感操作
	if m.containsSensitiveKeywords(upperSQL) {
		result["contains_sensitive"] = true
	}

	return result
}

// extractTablesFromSelect 从SELECT语句中提取表名
func (m *MySQLParser) extractTablesFromSelect(sql string) []string {
	// 简单的正则表达式匹配FROM子句
	re := regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`)
	matches := re.FindAllStringSubmatch(sql, -1)

	tables := make([]string, 0)
	for _, match := range matches {
		if len(match) > 1 {
			tables = append(tables, match[1])
		}
	}

	return tables
}

// extractTableFromInsert 从INSERT语句中提取表名
func (m *MySQLParser) extractTableFromInsert(sql string) string {
	re := regexp.MustCompile(`(?i)INSERT\s+INTO\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`)
	matches := re.FindStringSubmatch(sql)

	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// extractTableFromUpdate 从UPDATE语句中提取表名
func (m *MySQLParser) extractTableFromUpdate(sql string) string {
	re := regexp.MustCompile(`(?i)UPDATE\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`)
	matches := re.FindStringSubmatch(sql)

	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// extractTableFromDelete 从DELETE语句中提取表名
func (m *MySQLParser) extractTableFromDelete(sql string) string {
	re := regexp.MustCompile(`(?i)DELETE\s+FROM\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`)
	matches := re.FindStringSubmatch(sql)

	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// containsSensitiveKeywords 检查是否包含敏感关键词
func (m *MySQLParser) containsSensitiveKeywords(sql string) bool {
	sensitiveKeywords := []string{
		"PASSWORD", "PASSWD", "SECRET", "TOKEN", "KEY",
		"CREDIT", "CARD", "SSN", "SOCIAL", "SECURITY",
		"PHONE", "EMAIL", "ADDRESS", "PERSONAL",
	}

	for _, keyword := range sensitiveKeywords {
		if strings.Contains(sql, keyword) {
			return true
		}
	}

	return false
}

// getCommandName 获取命令名称
func (m *MySQLParser) getCommandName(command MySQLCommand) string {
	commandNames := map[MySQLCommand]string{
		ComSleep:            "COM_SLEEP",
		ComQuit:             "COM_QUIT",
		ComInitDB:           "COM_INIT_DB",
		ComQuery:            "COM_QUERY",
		ComFieldList:        "COM_FIELD_LIST",
		ComCreateDB:         "COM_CREATE_DB",
		ComDropDB:           "COM_DROP_DB",
		ComRefresh:          "COM_REFRESH",
		ComShutdown:         "COM_SHUTDOWN",
		ComStatistics:       "COM_STATISTICS",
		ComProcessInfo:      "COM_PROCESS_INFO",
		ComConnect:          "COM_CONNECT",
		ComProcessKill:      "COM_PROCESS_KILL",
		ComDebug:            "COM_DEBUG",
		ComPing:             "COM_PING",
		ComTime:             "COM_TIME",
		ComDelayedInsert:    "COM_DELAYED_INSERT",
		ComChangeUser:       "COM_CHANGE_USER",
		ComBinlogDump:       "COM_BINLOG_DUMP",
		ComTableDump:        "COM_TABLE_DUMP",
		ComConnectOut:       "COM_CONNECT_OUT",
		ComRegisterSlave:    "COM_REGISTER_SLAVE",
		ComStmtPrepare:      "COM_STMT_PREPARE",
		ComStmtExecute:      "COM_STMT_EXECUTE",
		ComStmtSendLongData: "COM_STMT_SEND_LONG_DATA",
		ComStmtClose:        "COM_STMT_CLOSE",
		ComStmtReset:        "COM_STMT_RESET",
		ComSetOption:        "COM_SET_OPTION",
		ComStmtFetch:        "COM_STMT_FETCH",
	}

	if name, exists := commandNames[command]; exists {
		return name
	}

	return fmt.Sprintf("UNKNOWN_%d", command)
}

// readLengthEncodedInteger 读取长度编码整数
func (m *MySQLParser) readLengthEncodedInteger(data []byte, offset int) (uint64, int) {
	if offset >= len(data) {
		return 0, offset
	}

	firstByte := data[offset]

	if firstByte < 251 {
		return uint64(firstByte), offset + 1
	} else if firstByte == 252 {
		if offset+2 >= len(data) {
			return 0, offset
		}
		return uint64(binary.LittleEndian.Uint16(data[offset+1 : offset+3])), offset + 3
	} else if firstByte == 253 {
		if offset+3 >= len(data) {
			return 0, offset
		}
		return uint64(data[offset+1]) | uint64(data[offset+2])<<8 | uint64(data[offset+3])<<16, offset + 4
	} else if firstByte == 254 {
		if offset+8 >= len(data) {
			return 0, offset
		}
		return binary.LittleEndian.Uint64(data[offset+1 : offset+9]), offset + 9
	}

	return 0, offset
}

// getSessionID 获取会话ID
func (m *MySQLParser) getSessionID(packet *interceptor.PacketInfo) string {
	return fmt.Sprintf("%s:%d-%s:%d",
		packet.SourceIP.String(), packet.SourcePort,
		packet.DestIP.String(), packet.DestPort)
}

// getOrCreateSession 获取或创建会话
func (m *MySQLParser) getOrCreateSession(sessionID string, packet *interceptor.PacketInfo) *MySQLSession {
	if session, exists := m.sessions[sessionID]; exists {
		session.LastUsed = time.Now()
		return session
	}

	session := &MySQLSession{
		SessionID: sessionID,
		State:     MySQLStateInit,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	m.sessions[sessionID] = session
	return session
}

// parseAuthPacket 解析认证包
func (m *MySQLParser) parseAuthPacket(payload []byte, parsedData *ParsedData, session *MySQLSession) (*ParsedData, error) {
	if len(payload) < 32 {
		return parsedData, nil
	}

	// 解析客户端能力标志
	capabilities := binary.LittleEndian.Uint32(payload[0:4])
	parsedData.Metadata["client_capabilities"] = capabilities

	// 解析最大数据包大小
	maxPacketSize := binary.LittleEndian.Uint32(payload[4:8])
	parsedData.Metadata["max_packet_size"] = maxPacketSize

	// 解析字符集
	charset := payload[8]
	parsedData.Metadata["charset"] = charset

	offset := 32 // 跳过保留字段

	// 解析用户名
	if offset < len(payload) {
		usernameEnd := offset
		for usernameEnd < len(payload) && payload[usernameEnd] != 0 {
			usernameEnd++
		}

		if usernameEnd < len(payload) {
			session.Username = string(payload[offset:usernameEnd])
			parsedData.Metadata["username"] = session.Username
			parsedData.Headers["Username"] = session.Username
		}
	}

	session.State = MySQLStateAuth
	parsedData.Metadata["packet_type"] = "auth"

	return parsedData, nil
}

// parseCommandPacket 解析命令包
func (m *MySQLParser) parseCommandPacket(command MySQLCommand, payload []byte, parsedData *ParsedData, session *MySQLSession) (*ParsedData, error) {
	commandName := m.getCommandName(command)
	parsedData.Metadata["command"] = commandName
	parsedData.Metadata["packet_type"] = "command"

	switch command {
	case ComQuery:
		return m.parseQueryCommand(payload, parsedData, session)
	case ComInitDB:
		return m.parseInitDBCommand(payload, parsedData, session)
	case ComQuit:
		session.State = MySQLStateClosed
		parsedData.Metadata["action"] = "disconnect"
	case ComPing:
		parsedData.Metadata["action"] = "ping"
	}

	return parsedData, nil
}
