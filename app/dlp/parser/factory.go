package parser

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lomehong/kennel/pkg/logging"
)

// ParserFactoryImpl 协议解析器工厂实现
type ParserFactoryImpl struct {
	creators map[string]ParserCreator
	mu       sync.RWMutex
	logger   logging.Logger
}

// NewParserFactory 创建协议解析器工厂
func NewParserFactory(logger logging.Logger) ParserFactory {
	factory := &ParserFactoryImpl{
		creators: make(map[string]ParserCreator),
		logger:   logger,
	}

	// 注册内置解析器
	factory.registerBuiltinParsers()

	return factory
}

// CreateParser 创建解析器
func (f *ParserFactoryImpl) CreateParser(protocol string, config ParserConfig) (ProtocolParser, error) {
	f.mu.RLock()
	creator, exists := f.creators[protocol]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("不支持的协议: %s", protocol)
	}

	parser, err := creator(config)
	if err != nil {
		return nil, fmt.Errorf("创建%s解析器失败: %w", protocol, err)
	}

	f.logger.Info("创建协议解析器", "protocol", protocol)
	return parser, nil
}

// GetSupportedProtocols 获取支持的协议
func (f *ParserFactoryImpl) GetSupportedProtocols() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	protocols := make([]string, 0, len(f.creators))
	for protocol := range f.creators {
		protocols = append(protocols, protocol)
	}
	return protocols
}

// RegisterParserType 注册解析器类型
func (f *ParserFactoryImpl) RegisterParserType(protocol string, creator ParserCreator) error {
	if protocol == "" {
		return fmt.Errorf("协议名称不能为空")
	}
	if creator == nil {
		return fmt.Errorf("解析器创建函数不能为空")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.creators[protocol] = creator
	f.logger.Info("注册协议解析器", "protocol", protocol)
	return nil
}

// registerBuiltinParsers 注册内置解析器
func (f *ParserFactoryImpl) registerBuiltinParsers() {
	// HTTP/HTTPS 解析器
	f.creators["http"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewHTTPParser(config.Logger), nil
	}
	f.creators["https"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewHTTPSParser(config.Logger, config.TLSConfig), nil
	}

	// FTP 解析器
	f.creators["ftp"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewFTPParser(config.Logger), nil
	}
	f.creators["sftp"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewSFTPParser(config.Logger), nil
	}

	// 邮件协议解析器
	f.creators["smtp"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewSMTPParser(config.Logger), nil
	}
	f.creators["pop3"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewPOP3Parser(config.Logger), nil
	}
	f.creators["imap"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewIMAPParser(config.Logger), nil
	}

	// 文件共享协议解析器
	f.creators["smb"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewSMBParser(config.Logger), nil
	}
	f.creators["cifs"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewSMBParser(config.Logger), nil // CIFS使用SMB解析器
	}

	// WebSocket 解析器 - 暂时注释掉，等待修复
	// f.creators["websocket"] = func(config ParserConfig) (ProtocolParser, error) {
	//	return NewWebSocketParser(config.Logger), nil
	// }

	// 数据库协议解析器
	f.creators["mysql"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewMySQLParser(config.Logger), nil
	}
	f.creators["postgresql"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewPostgreSQLParser(config.Logger), nil
	}
	f.creators["sqlserver"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewSQLServerParser(config.Logger), nil
	}

	// 消息队列协议解析器
	f.creators["mqtt"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewMQTTParser(config.Logger), nil
	}
	f.creators["amqp"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewAMQPParser(config.Logger), nil
	}
	f.creators["kafka"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewKafkaParser(config.Logger), nil
	}

	// API 协议解析器
	f.creators["grpc"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewGRPCParser(config.Logger), nil
	}
	f.creators["graphql"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewGraphQLParser(config.Logger), nil
	}

	// 默认解析器
	f.creators["unknown"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewDefaultParser(config.Logger), nil
	}
	f.creators["default"] = func(config ParserConfig) (ProtocolParser, error) {
		return NewDefaultParser(config.Logger), nil
	}

	f.logger.Info("注册内置协议解析器完成", "count", len(f.creators))
}

// ProtocolDetector 协议检测器
type ProtocolDetector struct {
	logger logging.Logger
}

// NewProtocolDetector 创建协议检测器
func NewProtocolDetector(logger logging.Logger) *ProtocolDetector {
	return &ProtocolDetector{
		logger: logger,
	}
}

// DetectProtocol 检测协议类型
func (d *ProtocolDetector) DetectProtocol(data []byte, port uint16) string {
	if len(data) == 0 {
		return "unknown"
	}

	// 基于数据内容的深度检测（优先级最高）
	protocolByContent := d.detectByContent(data)

	// 基于端口的初步判断
	protocolByPort := d.detectByPort(port)

	// 内容检测结果优先，但需要验证一致性
	if protocolByContent != "unknown" {
		// 如果内容检测和端口检测结果一致，直接返回
		if protocolByContent == protocolByPort {
			d.logger.Debug("协议检测一致", "protocol", protocolByContent, "port", port)
			return protocolByContent
		}

		// 如果不一致，进行冲突解决
		resolved := d.resolveProtocolConflict(protocolByContent, protocolByPort, data, port)
		d.logger.Debug("协议检测冲突已解决",
			"content_detected", protocolByContent,
			"port_detected", protocolByPort,
			"resolved", resolved,
			"port", port)
		return resolved
	}

	// 如果内容检测失败，使用端口检测结果
	if protocolByPort != "unknown" {
		d.logger.Debug("使用端口检测结果", "protocol", protocolByPort, "port", port)
		return protocolByPort
	}

	return "unknown"
}

// detectByPort 基于端口检测协议
func (d *ProtocolDetector) detectByPort(port uint16) string {
	portMap := map[uint16]string{
		80:   "http",
		443:  "https",
		21:   "ftp",
		22:   "sftp",
		25:   "smtp",
		110:  "pop3",
		143:  "imap",
		445:  "smb",
		139:  "smb",
		3306: "mysql",
		5432: "postgresql",
		1433: "sqlserver",
		1883: "mqtt",
		5672: "amqp",
		9092: "kafka",
	}

	if protocol, exists := portMap[port]; exists {
		return protocol
	}
	return "unknown"
}

// detectByContent 基于内容检测协议
func (d *ProtocolDetector) detectByContent(data []byte) string {
	if len(data) < 4 {
		return "unknown"
	}

	// HTTP 检测
	if d.isHTTP(data) {
		return "http"
	}

	// TLS/SSL 检测
	if d.isTLS(data) {
		return "https"
	}

	// FTP 检测
	if d.isFTP(data) {
		return "ftp"
	}

	// SMTP 检测
	if d.isSMTP(data) {
		return "smtp"
	}

	// MySQL 检测
	if d.isMySQL(data) {
		return "mysql"
	}

	// MQTT 检测
	if d.isMQTT(data) {
		return "mqtt"
	}

	return "unknown"
}

// resolveProtocolConflict 解决协议检测冲突
func (d *ProtocolDetector) resolveProtocolConflict(contentProtocol, portProtocol string, data []byte, port uint16) string {
	// 特殊情况处理：HTTP在非标准端口上
	if contentProtocol == "http" && portProtocol != "http" {
		// 如果内容明确是HTTP，优先使用内容检测结果
		if d.isHTTPStrict(data) {
			return "http"
		}
	}

	// 特殊情况处理：MySQL误识别
	if contentProtocol == "mysql" && portProtocol == "http" {
		// 重新严格检查是否真的是MySQL
		if !d.isMySQLStrict(data) {
			return "http"
		}
	}

	// 特殊情况处理：HTTPS在标准端口
	if contentProtocol == "https" && port == 443 {
		return "https"
	}

	// 默认情况：内容检测优先
	return contentProtocol
}

// isHTTP 检测是否为HTTP协议
func (d *ProtocolDetector) isHTTP(data []byte) bool {
	return d.isHTTPStrict(data) || d.isHTTPLoose(data)
}

// isHTTPStrict 严格的HTTP检测
func (d *ProtocolDetector) isHTTPStrict(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	dataStr := string(data[:min(len(data), 100)])

	// HTTP请求方法检测（严格）
	httpMethods := []string{"GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH ", "TRACE ", "CONNECT "}
	for _, method := range httpMethods {
		if len(dataStr) >= len(method) && dataStr[:len(method)] == method {
			// 进一步验证：检查是否包含HTTP版本
			if strings.Contains(dataStr, "HTTP/1.") || strings.Contains(dataStr, "HTTP/2") {
				return true
			}
		}
	}

	// HTTP响应检测（严格）
	if strings.HasPrefix(dataStr, "HTTP/1.0 ") ||
		strings.HasPrefix(dataStr, "HTTP/1.1 ") ||
		strings.HasPrefix(dataStr, "HTTP/2 ") {
		return true
	}

	return false
}

// isHTTPLoose 宽松的HTTP检测
func (d *ProtocolDetector) isHTTPLoose(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	dataStr := string(data[:min(len(data), 50)])

	// 检查常见HTTP头部
	httpHeaders := []string{"Host:", "User-Agent:", "Content-Type:", "Content-Length:", "Accept:"}
	for _, header := range httpHeaders {
		if strings.Contains(dataStr, header) {
			return true
		}
	}

	return false
}

// isTLS 检测是否为TLS协议
func (d *ProtocolDetector) isTLS(data []byte) bool {
	if len(data) < 6 {
		return false
	}

	// TLS记录头: Content Type (1 byte) + Version (2 bytes) + Length (2 bytes)
	// Content Type: 22 (Handshake), 23 (Application Data), 21 (Alert), 20 (Change Cipher Spec)
	contentType := data[0]
	version := uint16(data[1])<<8 | uint16(data[2])

	// 检查内容类型
	validContentTypes := []byte{20, 21, 22, 23}
	isValidContentType := false
	for _, ct := range validContentTypes {
		if contentType == ct {
			isValidContentType = true
			break
		}
	}

	// 检查版本 (TLS 1.0-1.3: 0x0301-0x0304, SSL 3.0: 0x0300)
	isValidVersion := version >= 0x0300 && version <= 0x0304

	return isValidContentType && isValidVersion
}

// isFTP 检测是否为FTP协议
func (d *ProtocolDetector) isFTP(data []byte) bool {
	dataStr := string(data)

	// FTP响应码格式: 3位数字 + 空格或-
	if len(dataStr) >= 4 {
		if dataStr[0] >= '1' && dataStr[0] <= '5' &&
			dataStr[1] >= '0' && dataStr[1] <= '9' &&
			dataStr[2] >= '0' && dataStr[2] <= '9' &&
			(dataStr[3] == ' ' || dataStr[3] == '-') {
			return true
		}
	}

	// FTP命令检测
	ftpCommands := []string{"USER ", "PASS ", "QUIT", "LIST", "RETR ", "STOR "}
	for _, cmd := range ftpCommands {
		if len(dataStr) >= len(cmd) && dataStr[:len(cmd)] == cmd {
			return true
		}
	}

	return false
}

// isSMTP 检测是否为SMTP协议
func (d *ProtocolDetector) isSMTP(data []byte) bool {
	dataStr := string(data)

	// SMTP响应码
	if len(dataStr) >= 4 {
		if dataStr[0] >= '2' && dataStr[0] <= '5' &&
			dataStr[1] >= '0' && dataStr[1] <= '9' &&
			dataStr[2] >= '0' && dataStr[2] <= '9' &&
			dataStr[3] == ' ' {
			return true
		}
	}

	// SMTP命令
	smtpCommands := []string{"HELO ", "EHLO ", "MAIL ", "RCPT ", "DATA", "QUIT"}
	for _, cmd := range smtpCommands {
		if len(dataStr) >= len(cmd) && dataStr[:len(cmd)] == cmd {
			return true
		}
	}

	return false
}

// isMySQL 检测是否为MySQL协议
func (d *ProtocolDetector) isMySQL(data []byte) bool {
	return d.isMySQLStrict(data)
}

// isMySQLStrict 严格的MySQL协议检测
func (d *ProtocolDetector) isMySQLStrict(data []byte) bool {
	if len(data) < 5 {
		return false
	}

	// MySQL握手包检测
	// 包长度 (3 bytes) + 序列号 (1 byte) + 协议版本 (1 byte)

	// 检查包长度是否合理（MySQL握手包通常较长）
	packetLength := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16
	if packetLength < 20 || packetLength > 1000 {
		return false
	}

	// 检查序列号（握手包序列号通常为0）
	sequenceNumber := data[3]
	if sequenceNumber != 0 {
		return false
	}

	// 检查协议版本（MySQL协议版本10）
	if data[4] != 0x0a {
		return false
	}

	// 进一步验证：检查服务器版本字符串
	if len(data) > 10 {
		// MySQL服务器版本字符串应该包含可打印字符
		versionStart := 5
		versionEnd := versionStart
		for i := versionStart; i < len(data) && i < versionStart+20; i++ {
			if data[i] == 0 {
				versionEnd = i
				break
			}
			// 检查是否为可打印字符
			if data[i] < 32 || data[i] > 126 {
				return false
			}
		}

		// 版本字符串应该有合理的长度
		if versionEnd-versionStart < 3 || versionEnd-versionStart > 20 {
			return false
		}
	}

	return true
}

// isMQTT 检测是否为MQTT协议
func (d *ProtocolDetector) isMQTT(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	// MQTT固定头部: 消息类型 (4 bits) + 标志 (4 bits) + 剩余长度
	messageType := (data[0] >> 4) & 0x0F

	// MQTT消息类型: 1-14 (0和15保留)
	return messageType >= 1 && messageType <= 14
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
