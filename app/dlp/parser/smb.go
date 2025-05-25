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

// SMBParser SMB协议解析器
type SMBParser struct {
	logger            logging.Logger
	maxDataSize       int
	timeout           time.Duration
	enableFileLog     bool
	sensitivePatterns []*regexp.Regexp
	fileExtensions    map[string]bool
}

// SMB协议常量
const (
	SMBProtocolID  = "\xFFSMB" // SMB1协议标识
	SMB2ProtocolID = "\xFESMB" // SMB2/3协议标识

	// SMB命令类型
	SMBComNegotiate    = 0x72
	SMBComSessionSetup = 0x73
	SMBComTreeConnect  = 0x75
	SMBComOpen         = 0x02
	SMBComRead         = 0x0A
	SMBComWrite        = 0x0B
	SMBComClose        = 0x04
	SMBComCreateDir    = 0x00
	SMBComDeleteDir    = 0x01
	SMBComRename       = 0x07
	SMBComDelete       = 0x06
)

// SMB数据包结构
type SMBPacket struct {
	Protocol    string
	Command     uint8
	Status      uint32
	Flags       uint8
	Flags2      uint16
	TreeID      uint16
	ProcessID   uint16
	UserID      uint16
	MultiplexID uint16
	Filename    string
	ShareName   string
	Data        []byte
	Operation   string
	FileSize    uint64
	Sensitive   bool
}

// NewSMBParser 创建SMB解析器
func NewSMBParser(logger logging.Logger) *SMBParser {
	parser := &SMBParser{
		logger:        logger,
		maxDataSize:   1048576, // 1MB
		timeout:       30 * time.Second,
		enableFileLog: true,
		sensitivePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\.(doc|docx|xls|xlsx|ppt|pptx|pdf)$`),
			regexp.MustCompile(`(?i)\.(txt|log|conf|config|ini)$`),
			regexp.MustCompile(`(?i)\.(key|pem|p12|pfx|crt|cer)$`),
			regexp.MustCompile(`(?i)(password|secret|confidential|private)`),
		},
		fileExtensions: map[string]bool{
			".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
			".ppt": true, ".pptx": true, ".pdf": true, ".txt": true,
			".log": true, ".conf": true, ".ini": true, ".key": true,
			".pem": true, ".p12": true, ".pfx": true, ".crt": true,
			".cer": true, ".zip": true, ".rar": true, ".7z": true,
		},
	}

	parser.logger.Info("初始化SMB解析器",
		"max_data_size", parser.maxDataSize,
		"timeout", parser.timeout,
		"enable_file_log", parser.enableFileLog)

	return parser
}

// GetParserInfo 获取解析器信息
func (s *SMBParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "SMB Parser",
		Version:            "1.0.0",
		Description:        "SMB协议解析器",
		SupportedProtocols: []string{"smb", "smb2", "smb3", "cifs"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

// GetSupportedProtocols 获取支持的协议
func (s *SMBParser) GetSupportedProtocols() []string {
	return []string{"smb", "smb2", "smb3", "cifs"}
}

// CanParse 检查是否可以解析数据
func (s *SMBParser) CanParse(packet *interceptor.PacketInfo) bool {
	if len(packet.Payload) < 8 {
		return false
	}

	// 检查端口
	if packet.DestPort == 445 || packet.DestPort == 139 || packet.SourcePort == 445 || packet.SourcePort == 139 {
		return true
	}

	// 检查SMB协议特征
	return s.isSMBProtocol(packet.Payload)
}

// Initialize 初始化解析器
func (s *SMBParser) Initialize(config ParserConfig) error {
	s.logger.Info("初始化SMB解析器", "config", config)
	return nil
}

// Parse 解析SMB数据包
func (s *SMBParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		if duration > s.timeout {
			s.logger.Warn("SMB解析超时", "duration", duration)
		}
	}()

	data := packet.Payload
	if len(data) > s.maxDataSize {
		s.logger.Warn("SMB数据包过大，截断处理", "size", len(data), "max_size", s.maxDataSize)
		data = data[:s.maxDataSize]
	}

	smbPacket, err := s.parsePacket(data)
	if err != nil {
		return nil, fmt.Errorf("解析SMB数据包失败: %w", err)
	}

	// 构建解析结果
	result := &ParsedData{
		Protocol:    smbPacket.Protocol,
		ContentType: "application/smb",
		Headers:     make(map[string]string),
		Body:        smbPacket.Data,
		Metadata:    make(map[string]interface{}),
	}

	// 填充元数据
	result.Metadata["command"] = smbPacket.Command
	result.Metadata["operation"] = smbPacket.Operation
	result.Metadata["filename"] = smbPacket.Filename
	result.Metadata["share_name"] = smbPacket.ShareName
	result.Metadata["file_size"] = smbPacket.FileSize
	result.Metadata["sensitive"] = smbPacket.Sensitive
	result.Metadata["tree_id"] = smbPacket.TreeID
	result.Metadata["user_id"] = smbPacket.UserID

	// 填充头部信息
	result.Headers["SMB-Command"] = fmt.Sprintf("0x%02X", smbPacket.Command)
	result.Headers["SMB-Operation"] = smbPacket.Operation
	if smbPacket.Filename != "" {
		result.Headers["SMB-Filename"] = smbPacket.Filename
	}
	if smbPacket.ShareName != "" {
		result.Headers["SMB-Share"] = smbPacket.ShareName
	}
	if smbPacket.FileSize > 0 {
		result.Headers["SMB-File-Size"] = fmt.Sprintf("%d", smbPacket.FileSize)
	}

	// 敏感数据检测
	if smbPacket.Sensitive {
		result.Headers["Sensitive-Data"] = "true"
		s.logger.Warn("检测到敏感SMB文件操作",
			"operation", smbPacket.Operation,
			"filename", smbPacket.Filename,
			"share", smbPacket.ShareName)
	}

	if s.enableFileLog && smbPacket.Filename != "" {
		s.logger.Debug("SMB文件操作解析",
			"operation", smbPacket.Operation,
			"filename", smbPacket.Filename,
			"share", smbPacket.ShareName,
			"size", smbPacket.FileSize)
	}

	return result, nil
}

// parsePacket 解析SMB数据包
func (s *SMBParser) parsePacket(data []byte) (*SMBPacket, error) {
	packet := &SMBPacket{}

	if len(data) < 32 {
		return nil, fmt.Errorf("SMB数据包太短")
	}

	// 检查协议标识
	if bytes.HasPrefix(data, []byte(SMBProtocolID)) {
		packet.Protocol = "smb1"
		return s.parseSMB1Packet(data, packet)
	} else if bytes.HasPrefix(data, []byte(SMB2ProtocolID)) {
		packet.Protocol = "smb2"
		return s.parseSMB2Packet(data, packet)
	}

	return nil, fmt.Errorf("未知的SMB协议格式")
}

// parseSMB1Packet 解析SMB1数据包
func (s *SMBParser) parseSMB1Packet(data []byte, packet *SMBPacket) (*SMBPacket, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("SMB1数据包太短")
	}

	reader := bytes.NewReader(data)

	// 跳过协议标识 (4字节)
	reader.Seek(4, 0)

	// 读取命令 (1字节)
	binary.Read(reader, binary.LittleEndian, &packet.Command)

	// 读取状态 (4字节)
	binary.Read(reader, binary.LittleEndian, &packet.Status)

	// 读取标志 (1字节)
	binary.Read(reader, binary.LittleEndian, &packet.Flags)

	// 读取标志2 (2字节)
	binary.Read(reader, binary.LittleEndian, &packet.Flags2)

	// 跳过一些字段到重要的ID字段
	reader.Seek(24, 0)

	// 读取TreeID (2字节)
	binary.Read(reader, binary.LittleEndian, &packet.TreeID)

	// 读取ProcessID (2字节)
	binary.Read(reader, binary.LittleEndian, &packet.ProcessID)

	// 读取UserID (2字节)
	binary.Read(reader, binary.LittleEndian, &packet.UserID)

	// 读取MultiplexID (2字节)
	binary.Read(reader, binary.LittleEndian, &packet.MultiplexID)

	// 解析操作类型和文件信息
	packet.Operation = s.getSMBOperation(packet.Command)

	// 尝试提取文件名（简化处理）
	if len(data) > 32 {
		packet.Filename, packet.ShareName = s.extractFileInfo(data[32:], packet.Command)
	}

	// 检测敏感数据
	packet.Sensitive = s.detectSensitiveFile(packet.Filename)

	return packet, nil
}

// parseSMB2Packet 解析SMB2数据包
func (s *SMBParser) parseSMB2Packet(data []byte, packet *SMBPacket) (*SMBPacket, error) {
	if len(data) < 64 {
		return nil, fmt.Errorf("SMB2数据包太短")
	}

	reader := bytes.NewReader(data)

	// 跳过协议标识 (4字节)
	reader.Seek(4, 0)

	// 跳过结构大小 (2字节)
	reader.Seek(6, 0)

	// 跳过信用费用 (2字节)
	reader.Seek(8, 0)

	// 跳过状态 (4字节)
	reader.Seek(12, 0)

	// 读取命令 (2字节)
	var command uint16
	binary.Read(reader, binary.LittleEndian, &command)
	packet.Command = uint8(command)

	// SMB2的其他字段解析...
	packet.Operation = s.getSMB2Operation(command)

	// 尝试提取文件名（简化处理）
	if len(data) > 64 {
		packet.Filename, packet.ShareName = s.extractSMB2FileInfo(data[64:], command)
	}

	// 检测敏感数据
	packet.Sensitive = s.detectSensitiveFile(packet.Filename)

	return packet, nil
}

// getSMBOperation 获取SMB1操作类型
func (s *SMBParser) getSMBOperation(command uint8) string {
	switch command {
	case SMBComNegotiate:
		return "NEGOTIATE"
	case SMBComSessionSetup:
		return "SESSION_SETUP"
	case SMBComTreeConnect:
		return "TREE_CONNECT"
	case SMBComOpen:
		return "OPEN"
	case SMBComRead:
		return "READ"
	case SMBComWrite:
		return "WRITE"
	case SMBComClose:
		return "CLOSE"
	case SMBComCreateDir:
		return "CREATE_DIR"
	case SMBComDeleteDir:
		return "DELETE_DIR"
	case SMBComRename:
		return "RENAME"
	case SMBComDelete:
		return "DELETE"
	default:
		return fmt.Sprintf("UNKNOWN_0x%02X", command)
	}
}

// getSMB2Operation 获取SMB2操作类型
func (s *SMBParser) getSMB2Operation(command uint16) string {
	switch command {
	case 0x0000:
		return "NEGOTIATE"
	case 0x0001:
		return "SESSION_SETUP"
	case 0x0003:
		return "TREE_CONNECT"
	case 0x0005:
		return "CREATE"
	case 0x0008:
		return "READ"
	case 0x0009:
		return "WRITE"
	case 0x0006:
		return "CLOSE"
	case 0x0010:
		return "QUERY_INFO"
	case 0x0011:
		return "SET_INFO"
	default:
		return fmt.Sprintf("UNKNOWN_0x%04X", command)
	}
}

// extractFileInfo 提取SMB1文件信息
func (s *SMBParser) extractFileInfo(data []byte, command uint8) (filename, shareName string) {
	// 简化的文件名提取逻辑
	// 在实际实现中需要根据具体的SMB命令格式来解析

	// 查找可能的文件路径字符串
	for i := 0; i < len(data)-1; i++ {
		if data[i] == '\\' || data[i] == '/' {
			// 找到路径分隔符，尝试提取文件名
			start := i
			end := start
			for end < len(data) && data[end] != 0 && data[end] != '\r' && data[end] != '\n' {
				end++
			}
			if end > start {
				path := string(data[start:end])
				if strings.Contains(path, "\\") {
					parts := strings.Split(path, "\\")
					if len(parts) > 1 {
						shareName = parts[1]
						if len(parts) > 2 {
							filename = parts[len(parts)-1]
						}
					}
				}
				break
			}
		}
	}

	return filename, shareName
}

// extractSMB2FileInfo 提取SMB2文件信息
func (s *SMBParser) extractSMB2FileInfo(data []byte, command uint16) (filename, shareName string) {
	// SMB2的文件信息提取逻辑
	// 这里简化处理，实际需要根据SMB2协议规范来解析
	return s.extractFileInfo(data, uint8(command))
}

// detectSensitiveFile 检测敏感文件
func (s *SMBParser) detectSensitiveFile(filename string) bool {
	if filename == "" {
		return false
	}

	filenameLower := strings.ToLower(filename)

	// 检查文件扩展名
	for ext := range s.fileExtensions {
		if strings.HasSuffix(filenameLower, ext) {
			return true
		}
	}

	// 检查敏感模式
	for _, pattern := range s.sensitivePatterns {
		if pattern.MatchString(filenameLower) {
			return true
		}
	}

	return false
}

// isSMBProtocol 检查是否为SMB协议
func (s *SMBParser) isSMBProtocol(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// 检查SMB1协议标识
	if bytes.HasPrefix(data, []byte(SMBProtocolID)) {
		return true
	}

	// 检查SMB2/3协议标识
	if bytes.HasPrefix(data, []byte(SMB2ProtocolID)) {
		return true
	}

	return false
}

// Cleanup 清理资源
func (s *SMBParser) Cleanup() error {
	s.logger.Info("清理SMB解析器资源")
	return nil
}
