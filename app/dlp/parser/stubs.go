package parser

import (
	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

// 这个文件包含其他协议解析器的存根实现
// 这些解析器提供基本的协议识别和元数据提取功能

// SFTPParser SFTP协议解析器存根
type SFTPParser struct {
	logger logging.Logger
}

func NewSFTPParser(logger logging.Logger) *SFTPParser {
	return &SFTPParser{logger: logger}
}

func (s *SFTPParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "SFTP Parser",
		Version:            "1.0.0",
		Description:        "SFTP协议解析器存根",
		SupportedProtocols: []string{"sftp"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

func (s *SFTPParser) CanParse(packet *interceptor.PacketInfo) bool {
	return packet != nil && (packet.DestPort == 22 || packet.SourcePort == 22)
}

func (s *SFTPParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	return &ParsedData{
		Protocol:    "sftp",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    map[string]any{"protocol": "sftp", "port": packet.DestPort},
		ContentType: "application/octet-stream",
	}, nil
}

func (s *SFTPParser) GetSupportedProtocols() []string {
	return []string{"sftp"}
}

func (s *SFTPParser) Initialize(config ParserConfig) error {
	return nil
}

func (s *SFTPParser) Cleanup() error {
	return nil
}

// POP3Parser POP3协议解析器存根
type POP3Parser struct {
	logger logging.Logger
}

func NewPOP3Parser(logger logging.Logger) *POP3Parser {
	return &POP3Parser{logger: logger}
}

func (p *POP3Parser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "POP3 Parser",
		Version:            "1.0.0",
		Description:        "POP3协议解析器存根",
		SupportedProtocols: []string{"pop3"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

func (p *POP3Parser) CanParse(packet *interceptor.PacketInfo) bool {
	return packet != nil && (packet.DestPort == 110 || packet.SourcePort == 110)
}

func (p *POP3Parser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	return &ParsedData{
		Protocol:    "pop3",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    map[string]any{"protocol": "pop3", "port": packet.DestPort},
		ContentType: "text/plain",
	}, nil
}

func (p *POP3Parser) GetSupportedProtocols() []string {
	return []string{"pop3"}
}

func (p *POP3Parser) Initialize(config ParserConfig) error {
	return nil
}

func (p *POP3Parser) Cleanup() error {
	return nil
}

// IMAPParser IMAP协议解析器存根
type IMAPParser struct {
	logger logging.Logger
}

func NewIMAPParser(logger logging.Logger) *IMAPParser {
	return &IMAPParser{logger: logger}
}

func (i *IMAPParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "IMAP Parser",
		Version:            "1.0.0",
		Description:        "IMAP协议解析器存根",
		SupportedProtocols: []string{"imap"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

func (i *IMAPParser) CanParse(packet *interceptor.PacketInfo) bool {
	return packet != nil && (packet.DestPort == 143 || packet.SourcePort == 143)
}

func (i *IMAPParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	return &ParsedData{
		Protocol:    "imap",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    map[string]any{"protocol": "imap", "port": packet.DestPort},
		ContentType: "text/plain",
	}, nil
}

func (i *IMAPParser) GetSupportedProtocols() []string {
	return []string{"imap"}
}

func (i *IMAPParser) Initialize(config ParserConfig) error {
	return nil
}

func (i *IMAPParser) Cleanup() error {
	return nil
}

// SQLServerParser SQL Server协议解析器存根
type SQLServerParser struct {
	logger logging.Logger
}

func NewSQLServerParser(logger logging.Logger) *SQLServerParser {
	return &SQLServerParser{logger: logger}
}

func (s *SQLServerParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "SQL Server Parser",
		Version:            "1.0.0",
		Description:        "SQL Server协议解析器存根",
		SupportedProtocols: []string{"sqlserver"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

func (s *SQLServerParser) CanParse(packet *interceptor.PacketInfo) bool {
	return packet != nil && (packet.DestPort == 1433 || packet.SourcePort == 1433)
}

func (s *SQLServerParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	return &ParsedData{
		Protocol:    "sqlserver",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    map[string]any{"protocol": "sqlserver", "port": packet.DestPort},
		ContentType: "application/sqlserver",
	}, nil
}

func (s *SQLServerParser) GetSupportedProtocols() []string {
	return []string{"sqlserver"}
}

func (s *SQLServerParser) Initialize(config ParserConfig) error {
	return nil
}

func (s *SQLServerParser) Cleanup() error {
	return nil
}

// MQTTParser MQTT协议解析器存根
type MQTTParser struct {
	logger logging.Logger
}

func NewMQTTParser(logger logging.Logger) *MQTTParser {
	return &MQTTParser{logger: logger}
}

func (m *MQTTParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "MQTT Parser",
		Version:            "1.0.0",
		Description:        "MQTT协议解析器存根",
		SupportedProtocols: []string{"mqtt"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

func (m *MQTTParser) CanParse(packet *interceptor.PacketInfo) bool {
	return packet != nil && (packet.DestPort == 1883 || packet.SourcePort == 1883 ||
		packet.DestPort == 8883 || packet.SourcePort == 8883)
}

func (m *MQTTParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	return &ParsedData{
		Protocol:    "mqtt",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    map[string]any{"protocol": "mqtt", "port": packet.DestPort},
		ContentType: "application/mqtt",
	}, nil
}

func (m *MQTTParser) GetSupportedProtocols() []string {
	return []string{"mqtt"}
}

func (m *MQTTParser) Initialize(config ParserConfig) error {
	return nil
}

func (m *MQTTParser) Cleanup() error {
	return nil
}

// AMQPParser AMQP协议解析器存根
type AMQPParser struct {
	logger logging.Logger
}

func NewAMQPParser(logger logging.Logger) *AMQPParser {
	return &AMQPParser{logger: logger}
}

func (a *AMQPParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "AMQP Parser",
		Version:            "1.0.0",
		Description:        "AMQP协议解析器存根",
		SupportedProtocols: []string{"amqp"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

func (a *AMQPParser) CanParse(packet *interceptor.PacketInfo) bool {
	return packet != nil && (packet.DestPort == 5672 || packet.SourcePort == 5672)
}

func (a *AMQPParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	return &ParsedData{
		Protocol:    "amqp",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    map[string]any{"protocol": "amqp", "port": packet.DestPort},
		ContentType: "application/amqp",
	}, nil
}

func (a *AMQPParser) GetSupportedProtocols() []string {
	return []string{"amqp"}
}

func (a *AMQPParser) Initialize(config ParserConfig) error {
	return nil
}

func (a *AMQPParser) Cleanup() error {
	return nil
}

// KafkaParser Kafka协议解析器存根
type KafkaParser struct {
	logger logging.Logger
}

func NewKafkaParser(logger logging.Logger) *KafkaParser {
	return &KafkaParser{logger: logger}
}

func (k *KafkaParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "Kafka Parser",
		Version:            "1.0.0",
		Description:        "Kafka协议解析器存根",
		SupportedProtocols: []string{"kafka"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

func (k *KafkaParser) CanParse(packet *interceptor.PacketInfo) bool {
	return packet != nil && (packet.DestPort == 9092 || packet.SourcePort == 9092)
}

func (k *KafkaParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	return &ParsedData{
		Protocol:    "kafka",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    map[string]any{"protocol": "kafka", "port": packet.DestPort},
		ContentType: "application/kafka",
	}, nil
}

func (k *KafkaParser) GetSupportedProtocols() []string {
	return []string{"kafka"}
}

func (k *KafkaParser) Initialize(config ParserConfig) error {
	return nil
}

func (k *KafkaParser) Cleanup() error {
	return nil
}

// GRPCParser gRPC协议解析器存根
type GRPCParser struct {
	logger logging.Logger
}

func NewGRPCParser(logger logging.Logger) *GRPCParser {
	return &GRPCParser{logger: logger}
}

func (g *GRPCParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "gRPC Parser",
		Version:            "1.0.0",
		Description:        "gRPC协议解析器存根",
		SupportedProtocols: []string{"grpc"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

func (g *GRPCParser) CanParse(packet *interceptor.PacketInfo) bool {
	// gRPC通常运行在HTTP/2上，需要检查内容
	return packet != nil && len(packet.Payload) > 0
}

func (g *GRPCParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	return &ParsedData{
		Protocol:    "grpc",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    map[string]any{"protocol": "grpc"},
		ContentType: "application/grpc",
	}, nil
}

func (g *GRPCParser) GetSupportedProtocols() []string {
	return []string{"grpc"}
}

func (g *GRPCParser) Initialize(config ParserConfig) error {
	return nil
}

func (g *GRPCParser) Cleanup() error {
	return nil
}

// GraphQLParser GraphQL协议解析器存根
type GraphQLParser struct {
	logger logging.Logger
}

func NewGraphQLParser(logger logging.Logger) *GraphQLParser {
	return &GraphQLParser{logger: logger}
}

func (g *GraphQLParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "GraphQL Parser",
		Version:            "1.0.0",
		Description:        "GraphQL协议解析器存根",
		SupportedProtocols: []string{"graphql"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

func (g *GraphQLParser) CanParse(packet *interceptor.PacketInfo) bool {
	// GraphQL通常通过HTTP传输，需要检查内容
	return packet != nil && len(packet.Payload) > 0
}

func (g *GraphQLParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	return &ParsedData{
		Protocol:    "graphql",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    map[string]any{"protocol": "graphql"},
		ContentType: "application/json",
	}, nil
}

func (g *GraphQLParser) GetSupportedProtocols() []string {
	return []string{"graphql"}
}

func (g *GraphQLParser) Initialize(config ParserConfig) error {
	return nil
}

func (g *GraphQLParser) Cleanup() error {
	return nil
}

// DefaultParser 默认协议解析器，用于处理未知协议
type DefaultParser struct {
	logger logging.Logger
}

func NewDefaultParser(logger logging.Logger) *DefaultParser {
	return &DefaultParser{logger: logger}
}

func (d *DefaultParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "Default Parser",
		Version:            "1.0.0",
		Description:        "默认协议解析器，处理未知协议",
		SupportedProtocols: []string{"unknown", "default"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

func (d *DefaultParser) CanParse(packet *interceptor.PacketInfo) bool {
	// 默认解析器总是可以解析任何数据包
	return packet != nil
}

func (d *DefaultParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	// 基本的数据包信息提取
	protocol := "unknown"
	contentType := "application/octet-stream"

	// 尝试基于端口推断协议
	if packet.DestPort != 0 {
		switch packet.DestPort {
		case 80:
			protocol = "http"
			contentType = "text/html"
		case 443:
			protocol = "https"
			contentType = "text/html"
		case 21:
			protocol = "ftp"
			contentType = "text/plain"
		case 22:
			protocol = "ssh"
			contentType = "application/octet-stream"
		case 25:
			protocol = "smtp"
			contentType = "text/plain"
		case 53:
			protocol = "dns"
			contentType = "application/dns"
		case 110:
			protocol = "pop3"
			contentType = "text/plain"
		case 143:
			protocol = "imap"
			contentType = "text/plain"
		case 993:
			protocol = "imaps"
			contentType = "text/plain"
		case 995:
			protocol = "pop3s"
			contentType = "text/plain"
		}
	}

	return &ParsedData{
		Protocol: protocol,
		Headers:  make(map[string]string),
		Body:     packet.Payload,
		Metadata: map[string]any{
			"protocol":    protocol,
			"port":        packet.DestPort,
			"source_port": packet.SourcePort,
			"size":        len(packet.Payload),
			"parser":      "default",
		},
		ContentType: contentType,
	}, nil
}

func (d *DefaultParser) GetSupportedProtocols() []string {
	return []string{"unknown", "default"}
}

func (d *DefaultParser) Initialize(config ParserConfig) error {
	d.logger.Info("初始化默认协议解析器")
	return nil
}

func (d *DefaultParser) Cleanup() error {
	d.logger.Info("清理默认协议解析器")
	return nil
}
