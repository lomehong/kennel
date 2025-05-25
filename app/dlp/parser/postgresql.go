package parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

// PostgreSQLParser PostgreSQL协议解析器
type PostgreSQLParser struct {
	logger            logging.Logger
	maxQuerySize      int
	timeout           time.Duration
	enableQueryLog    bool
	sensitivePatterns []*regexp.Regexp
}

// PostgreSQL消息类型
const (
	PGMsgQuery          = 'Q' // 简单查询
	PGMsgParse          = 'P' // 解析语句
	PGMsgBind           = 'B' // 绑定参数
	PGMsgExecute        = 'E' // 执行语句
	PGMsgClose          = 'C' // 关闭语句
	PGMsgDescribe       = 'D' // 描述语句
	PGMsgSync           = 'S' // 同步
	PGMsgTerminate      = 'X' // 终止连接
	PGMsgStartupMessage = 0   // 启动消息（无类型字节）
)

// PostgreSQL数据包结构
type PostgreSQLPacket struct {
	MessageType byte
	Length      uint32
	Payload     []byte
	Query       string
	Parameters  []string
	Database    string
	User        string
	Operation   string
	Tables      []string
	Sensitive   bool
}

// NewPostgreSQLParser 创建PostgreSQL解析器
func NewPostgreSQLParser(logger logging.Logger) *PostgreSQLParser {
	parser := &PostgreSQLParser{
		logger:         logger,
		maxQuerySize:   65536, // 64KB
		timeout:        30 * time.Second,
		enableQueryLog: true,
		sensitivePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(password|pwd|secret|token|key)\b`),
			regexp.MustCompile(`(?i)\b(credit_card|ssn|social_security)\b`),
			regexp.MustCompile(`(?i)\b(email|phone|address)\b`),
		},
	}

	parser.logger.Info("初始化PostgreSQL解析器",
		"max_query_size", parser.maxQuerySize,
		"timeout", parser.timeout,
		"enable_query_log", parser.enableQueryLog)

	return parser
}

// GetParserInfo 获取解析器信息
func (p *PostgreSQLParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "PostgreSQL Parser",
		Version:            "1.0.0",
		Description:        "PostgreSQL协议解析器",
		SupportedProtocols: []string{"postgresql", "postgres", "pgsql"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

// GetSupportedProtocols 获取支持的协议
func (p *PostgreSQLParser) GetSupportedProtocols() []string {
	return []string{"postgresql", "postgres", "pgsql"}
}

// CanParse 检查是否可以解析数据
func (p *PostgreSQLParser) CanParse(packet *interceptor.PacketInfo) bool {
	if len(packet.Payload) < 5 {
		return false
	}

	// 检查端口
	if packet.DestPort == 5432 || packet.SourcePort == 5432 { // PostgreSQL默认端口
		return true
	}

	// 检查PostgreSQL协议特征
	return p.isPostgreSQLProtocol(packet.Payload)
}

// Initialize 初始化解析器
func (p *PostgreSQLParser) Initialize(config ParserConfig) error {
	p.logger.Info("初始化PostgreSQL解析器", "config", config)
	return nil
}

// Parse 解析PostgreSQL数据包
func (p *PostgreSQLParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		if duration > p.timeout {
			p.logger.Warn("PostgreSQL解析超时", "duration", duration)
		}
	}()

	data := packet.Payload
	if len(data) > p.maxQuerySize {
		p.logger.Warn("PostgreSQL数据包过大，截断处理", "size", len(data), "max_size", p.maxQuerySize)
		data = data[:p.maxQuerySize]
	}

	pgPacket, err := p.parsePacket(data)
	if err != nil {
		return nil, fmt.Errorf("解析PostgreSQL数据包失败: %w", err)
	}

	// 构建解析结果
	result := &ParsedData{
		Protocol:    "postgresql",
		ContentType: "application/postgresql",
		Headers:     make(map[string]string),
		Body:        []byte(pgPacket.Query),
		Metadata:    make(map[string]interface{}),
	}

	// 填充元数据
	result.Metadata["message_type"] = string(pgPacket.MessageType)
	result.Metadata["operation"] = pgPacket.Operation
	result.Metadata["database"] = pgPacket.Database
	result.Metadata["user"] = pgPacket.User
	result.Metadata["tables"] = pgPacket.Tables
	result.Metadata["sensitive"] = pgPacket.Sensitive
	result.Metadata["query_length"] = len(pgPacket.Query)

	// 填充头部信息
	if pgPacket.Database != "" {
		result.Headers["Database"] = pgPacket.Database
	}
	if pgPacket.User != "" {
		result.Headers["User"] = pgPacket.User
	}
	if pgPacket.Operation != "" {
		result.Headers["Operation"] = pgPacket.Operation
	}

	// 参数处理
	if len(pgPacket.Parameters) > 0 {
		result.Metadata["parameters"] = pgPacket.Parameters
		result.Headers["Parameter-Count"] = fmt.Sprintf("%d", len(pgPacket.Parameters))
	}

	// 敏感数据检测
	if pgPacket.Sensitive {
		result.Headers["Sensitive-Data"] = "true"
		p.logger.Warn("检测到敏感PostgreSQL查询",
			"operation", pgPacket.Operation,
			"database", pgPacket.Database,
			"user", pgPacket.User)
	}

	if p.enableQueryLog && pgPacket.Query != "" {
		p.logger.Debug("PostgreSQL查询解析",
			"operation", pgPacket.Operation,
			"database", pgPacket.Database,
			"query_preview", p.truncateString(pgPacket.Query, 100))
	}

	return result, nil
}

// parsePacket 解析PostgreSQL数据包
func (p *PostgreSQLParser) parsePacket(data []byte) (*PostgreSQLPacket, error) {
	packet := &PostgreSQLPacket{}

	if len(data) < 5 {
		return nil, fmt.Errorf("数据包太短")
	}

	reader := bytes.NewReader(data)

	// 读取消息类型（第一个字节）
	var msgType byte
	if err := binary.Read(reader, binary.BigEndian, &msgType); err != nil {
		return nil, fmt.Errorf("读取消息类型失败: %w", err)
	}
	packet.MessageType = msgType

	// 读取消息长度（4字节，大端序）
	var length uint32
	if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("读取消息长度失败: %w", err)
	}
	packet.Length = length

	// 读取载荷
	payloadSize := int(length) - 4 // 长度字段本身占4字节
	if payloadSize > 0 && payloadSize <= len(data)-5 {
		payload := make([]byte, payloadSize)
		if _, err := reader.Read(payload); err != nil {
			return nil, fmt.Errorf("读取载荷失败: %w", err)
		}
		packet.Payload = payload
	}

	// 根据消息类型解析内容
	switch msgType {
	case PGMsgQuery:
		p.parseQuery(packet)
	case PGMsgParse:
		p.parsePreparedStatement(packet)
	case PGMsgBind:
		p.parseBindMessage(packet)
	case PGMsgExecute:
		p.parseExecuteMessage(packet)
	default:
		// 尝试解析为启动消息
		if msgType == 0 {
			p.parseStartupMessage(data, packet)
		}
	}

	// 检测敏感数据
	packet.Sensitive = p.detectSensitiveData(packet.Query)

	return packet, nil
}

// parseQuery 解析简单查询消息
func (p *PostgreSQLParser) parseQuery(packet *PostgreSQLPacket) {
	if len(packet.Payload) == 0 {
		return
	}

	// 查询字符串以null结尾
	queryBytes := packet.Payload
	if queryBytes[len(queryBytes)-1] == 0 {
		queryBytes = queryBytes[:len(queryBytes)-1]
	}

	packet.Query = string(queryBytes)
	packet.Operation = p.extractOperation(packet.Query)
	packet.Tables = p.extractTables(packet.Query)
}

// parsePreparedStatement 解析预处理语句
func (p *PostgreSQLParser) parsePreparedStatement(packet *PostgreSQLPacket) {
	if len(packet.Payload) < 2 {
		return
	}

	payload := packet.Payload
	// 跳过语句名称（以null结尾）
	nameEnd := bytes.IndexByte(payload, 0)
	if nameEnd == -1 {
		return
	}

	// 获取查询字符串
	queryStart := nameEnd + 1
	if queryStart < len(payload) {
		queryEnd := bytes.IndexByte(payload[queryStart:], 0)
		if queryEnd != -1 {
			packet.Query = string(payload[queryStart : queryStart+queryEnd])
			packet.Operation = p.extractOperation(packet.Query)
			packet.Tables = p.extractTables(packet.Query)
		}
	}
}

// parseBindMessage 解析绑定消息
func (p *PostgreSQLParser) parseBindMessage(packet *PostgreSQLPacket) {
	// 绑定消息包含参数值，这里简化处理
	packet.Operation = "BIND"
}

// parseExecuteMessage 解析执行消息
func (p *PostgreSQLParser) parseExecuteMessage(packet *PostgreSQLPacket) {
	packet.Operation = "EXECUTE"
}

// parseStartupMessage 解析启动消息
func (p *PostgreSQLParser) parseStartupMessage(data []byte, packet *PostgreSQLPacket) {
	if len(data) < 8 {
		return
	}

	// 启动消息格式：长度(4) + 协议版本(4) + 参数键值对
	reader := bytes.NewReader(data[8:]) // 跳过长度和版本

	for reader.Len() > 0 {
		// 读取键
		key, err := p.readCString(reader)
		if err != nil || key == "" {
			break
		}

		// 读取值
		value, err := p.readCString(reader)
		if err != nil {
			break
		}

		switch key {
		case "database":
			packet.Database = value
		case "user":
			packet.User = value
		}
	}

	packet.Operation = "STARTUP"
}

// readCString 读取C风格字符串（以null结尾）
func (p *PostgreSQLParser) readCString(reader *bytes.Reader) (string, error) {
	var result []byte
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		if b == 0 {
			break
		}
		result = append(result, b)
	}
	return string(result), nil
}

// extractOperation 提取SQL操作类型
func (p *PostgreSQLParser) extractOperation(query string) string {
	if query == "" {
		return ""
	}

	// 移除前导空白和注释
	query = strings.TrimSpace(query)
	if strings.HasPrefix(query, "--") || strings.HasPrefix(query, "/*") {
		return "COMMENT"
	}

	// 提取第一个单词作为操作类型
	words := strings.Fields(query)
	if len(words) > 0 {
		return strings.ToUpper(words[0])
	}

	return "UNKNOWN"
}

// extractTables 提取涉及的表名
func (p *PostgreSQLParser) extractTables(query string) []string {
	if query == "" {
		return nil
	}

	// 简化的表名提取逻辑
	tables := make([]string, 0)
	query = strings.ToLower(query)

	// 匹配 FROM、JOIN、UPDATE、INSERT INTO 等关键字后的表名
	patterns := []string{
		`from\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
		`join\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
		`update\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
		`insert\s+into\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(query, -1)
		for _, match := range matches {
			if len(match) > 1 {
				tableName := strings.TrimSpace(match[1])
				if tableName != "" {
					tables = append(tables, tableName)
				}
			}
		}
	}

	return p.uniqueStrings(tables)
}

// detectSensitiveData 检测敏感数据
func (p *PostgreSQLParser) detectSensitiveData(query string) bool {
	if query == "" {
		return false
	}

	queryLower := strings.ToLower(query)
	for _, pattern := range p.sensitivePatterns {
		if pattern.MatchString(queryLower) {
			return true
		}
	}

	return false
}

// isPostgreSQLProtocol 检查是否为PostgreSQL协议
func (p *PostgreSQLParser) isPostgreSQLProtocol(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	// 检查启动消息的协议版本号
	if data[0] == 0 && len(data) >= 8 {
		// 读取协议版本（大端序）
		version := binary.BigEndian.Uint32(data[4:8])
		// PostgreSQL协议版本通常是3.0 (0x00030000)
		if version == 0x00030000 {
			return true
		}
	}

	// 检查常见的PostgreSQL消息类型
	msgType := data[0]
	switch msgType {
	case PGMsgQuery, PGMsgParse, PGMsgBind, PGMsgExecute, PGMsgClose, PGMsgDescribe, PGMsgSync, PGMsgTerminate:
		return true
	}

	return false
}

// uniqueStrings 去重字符串切片
func (p *PostgreSQLParser) uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, str := range slice {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

// truncateString 截断字符串
func (p *PostgreSQLParser) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Cleanup 清理资源
func (p *PostgreSQLParser) Cleanup() error {
	p.logger.Info("清理PostgreSQL解析器资源")
	return nil
}
