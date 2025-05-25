package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

// FTPParser FTP协议解析器
type FTPParser struct {
	logger   logging.Logger
	sessions map[string]*FTPSession
}

// FTPSession FTP会话信息
type FTPSession struct {
	SessionID    string
	ControlConn  *FTPConnection
	DataConn     *FTPConnection
	CurrentDir   string
	TransferMode string
	LastCommand  string
	CreatedAt    time.Time
	LastUsed     time.Time
}

// FTPConnection FTP连接信息
type FTPConnection struct {
	SourceIP   string
	DestIP     string
	SourcePort uint16
	DestPort   uint16
	IsActive   bool
}

// FTPCommand FTP命令结构
type FTPCommand struct {
	Command    string
	Parameters []string
	RawLine    string
}

// FTPResponse FTP响应结构
type FTPResponse struct {
	Code       int
	Message    string
	IsMultiple bool
	RawLine    string
}

// NewFTPParser 创建FTP解析器
func NewFTPParser(logger logging.Logger) *FTPParser {
	return &FTPParser{
		logger:   logger,
		sessions: make(map[string]*FTPSession),
	}
}

// GetParserInfo 获取解析器信息
func (f *FTPParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "FTP Parser",
		Version:            "1.0.0",
		Description:        "FTP协议解析器，支持命令和数据传输解析",
		SupportedProtocols: []string{"ftp"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

// CanParse 检查是否能解析指定的数据包
func (f *FTPParser) CanParse(packet *interceptor.PacketInfo) bool {
	if packet == nil || len(packet.Payload) == 0 {
		return false
	}

	// 检查端口
	if packet.DestPort == 21 || packet.SourcePort == 21 {
		return true
	}

	// 检查内容
	return f.isFTPData(packet.Payload)
}

// Parse 解析数据包
func (f *FTPParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	if !f.CanParse(packet) {
		return nil, fmt.Errorf("不是有效的FTP数据包")
	}

	parsedData := &ParsedData{
		Protocol:    "ftp",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    make(map[string]any),
		ContentType: "text/plain",
	}

	// 获取或创建会话
	sessionID := f.getSessionID(packet)
	session := f.getOrCreateSession(sessionID, packet)

	// 判断是控制连接还是数据连接
	if packet.DestPort == 21 || packet.SourcePort == 21 {
		return f.parseControlData(packet.Payload, parsedData, session)
	} else {
		return f.parseDataTransfer(packet.Payload, parsedData, session)
	}
}

// GetSupportedProtocols 获取支持的协议列表
func (f *FTPParser) GetSupportedProtocols() []string {
	return []string{"ftp"}
}

// Initialize 初始化解析器
func (f *FTPParser) Initialize(config ParserConfig) error {
	f.logger.Info("初始化FTP解析器")
	return nil
}

// Cleanup 清理资源
func (f *FTPParser) Cleanup() error {
	f.logger.Info("清理FTP解析器资源")
	f.sessions = make(map[string]*FTPSession)
	return nil
}

// isFTPData 检查是否为FTP数据
func (f *FTPParser) isFTPData(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	dataStr := strings.TrimSpace(string(data))

	// 检查FTP响应码格式 (3位数字 + 空格或-)
	if len(dataStr) >= 4 {
		if dataStr[0] >= '1' && dataStr[0] <= '5' &&
			dataStr[1] >= '0' && dataStr[1] <= '9' &&
			dataStr[2] >= '0' && dataStr[2] <= '9' &&
			(dataStr[3] == ' ' || dataStr[3] == '-') {
			return true
		}
	}

	// 检查FTP命令
	ftpCommands := []string{
		"USER", "PASS", "ACCT", "CWD", "CDUP", "SMNT", "QUIT", "REIN",
		"PORT", "PASV", "TYPE", "STRU", "MODE", "RETR", "STOR", "STOU",
		"APPE", "ALLO", "REST", "RNFR", "RNTO", "ABOR", "DELE", "RMD",
		"MKD", "PWD", "LIST", "NLST", "SITE", "SYST", "STAT", "HELP",
		"NOOP", "FEAT", "OPTS", "SIZE", "MDTM",
	}

	upperData := strings.ToUpper(dataStr)
	for _, cmd := range ftpCommands {
		if strings.HasPrefix(upperData, cmd+" ") || upperData == cmd {
			return true
		}
	}

	return false
}

// parseControlData 解析控制连接数据
func (f *FTPParser) parseControlData(data []byte, parsedData *ParsedData, session *FTPSession) (*ParsedData, error) {
	dataStr := strings.TrimSpace(string(data))
	lines := strings.Split(dataStr, "\r\n")

	commands := make([]FTPCommand, 0)
	responses := make([]FTPResponse, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 尝试解析为响应
		if response, isResponse := f.parseResponse(line); isResponse {
			responses = append(responses, response)
		} else {
			// 解析为命令
			if command := f.parseCommand(line); command.Command != "" {
				commands = append(commands, command)
				session.LastCommand = command.Command
			}
		}
	}

	// 设置元数据
	if len(commands) > 0 {
		parsedData.Metadata["ftp_commands"] = commands
		parsedData.Metadata["command_count"] = len(commands)

		// 提取敏感信息
		for _, cmd := range commands {
			switch strings.ToUpper(cmd.Command) {
			case "USER":
				if len(cmd.Parameters) > 0 {
					parsedData.Headers["Username"] = cmd.Parameters[0]
					parsedData.Metadata["username"] = cmd.Parameters[0]
				}
			case "PASS":
				if len(cmd.Parameters) > 0 {
					parsedData.Headers["Password"] = "***REDACTED***"
					parsedData.Metadata["password_provided"] = true
				}
			case "CWD":
				if len(cmd.Parameters) > 0 {
					session.CurrentDir = cmd.Parameters[0]
					parsedData.Metadata["current_directory"] = cmd.Parameters[0]
				}
			case "RETR", "STOR":
				if len(cmd.Parameters) > 0 {
					parsedData.Metadata["file_operation"] = cmd.Command
					parsedData.Metadata["filename"] = cmd.Parameters[0]
				}
			case "PORT":
				if len(cmd.Parameters) > 0 {
					if dataConn := f.parsePortCommand(cmd.Parameters[0]); dataConn != nil {
						session.DataConn = dataConn
						parsedData.Metadata["data_connection"] = "active"
					}
				}
			case "PASV":
				parsedData.Metadata["data_connection"] = "passive"
			}
		}
	}

	if len(responses) > 0 {
		parsedData.Metadata["ftp_responses"] = responses
		parsedData.Metadata["response_count"] = len(responses)

		// 分析响应
		for _, resp := range responses {
			if resp.Code >= 200 && resp.Code < 300 {
				parsedData.Metadata["status"] = "success"
			} else if resp.Code >= 400 {
				parsedData.Metadata["status"] = "error"
				parsedData.Metadata["error_code"] = resp.Code
				parsedData.Metadata["error_message"] = resp.Message
			}
		}
	}

	// 更新会话信息
	session.LastUsed = time.Now()

	return parsedData, nil
}

// parseDataTransfer 解析数据传输
func (f *FTPParser) parseDataTransfer(data []byte, parsedData *ParsedData, session *FTPSession) (*ParsedData, error) {
	parsedData.Metadata["transfer_type"] = "data"
	parsedData.Metadata["data_size"] = len(data)
	parsedData.Metadata["last_command"] = session.LastCommand

	// 根据最后的命令类型分析数据
	switch strings.ToUpper(session.LastCommand) {
	case "LIST", "NLST":
		// 目录列表
		parsedData.ContentType = "text/plain"
		parsedData.Metadata["content_type"] = "directory_listing"

		// 解析目录列表
		if listing := f.parseDirectoryListing(data); len(listing) > 0 {
			parsedData.Metadata["directory_entries"] = listing
		}

	case "RETR":
		// 文件下载
		parsedData.ContentType = f.detectFileContentType(data)
		parsedData.Metadata["content_type"] = "file_download"
		parsedData.Metadata["operation"] = "download"

	case "STOR", "STOU", "APPE":
		// 文件上传
		parsedData.ContentType = f.detectFileContentType(data)
		parsedData.Metadata["content_type"] = "file_upload"
		parsedData.Metadata["operation"] = "upload"

	default:
		parsedData.Metadata["content_type"] = "unknown_data"
	}

	return parsedData, nil
}

// parseCommand 解析FTP命令
func (f *FTPParser) parseCommand(line string) FTPCommand {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return FTPCommand{}
	}

	command := FTPCommand{
		Command:    strings.ToUpper(parts[0]),
		Parameters: parts[1:],
		RawLine:    line,
	}

	return command
}

// parseResponse 解析FTP响应
func (f *FTPParser) parseResponse(line string) (FTPResponse, bool) {
	// FTP响应格式: 3位数字 + 空格或- + 消息
	if len(line) < 4 {
		return FTPResponse{}, false
	}

	codeStr := line[:3]
	code, err := strconv.Atoi(codeStr)
	if err != nil {
		return FTPResponse{}, false
	}

	if line[3] != ' ' && line[3] != '-' {
		return FTPResponse{}, false
	}

	response := FTPResponse{
		Code:       code,
		Message:    strings.TrimSpace(line[4:]),
		IsMultiple: line[3] == '-',
		RawLine:    line,
	}

	return response, true
}

// parsePortCommand 解析PORT命令
func (f *FTPParser) parsePortCommand(portParam string) *FTPConnection {
	// PORT命令格式: h1,h2,h3,h4,p1,p2
	parts := strings.Split(portParam, ",")
	if len(parts) != 6 {
		return nil
	}

	// 解析IP地址
	ip := strings.Join(parts[:4], ".")

	// 解析端口
	p1, err1 := strconv.Atoi(parts[4])
	p2, err2 := strconv.Atoi(parts[5])
	if err1 != nil || err2 != nil {
		return nil
	}

	port := uint16(p1*256 + p2)

	return &FTPConnection{
		DestIP:   ip,
		DestPort: port,
		IsActive: true,
	}
}

// parseDirectoryListing 解析目录列表
func (f *FTPParser) parseDirectoryListing(data []byte) []map[string]any {
	lines := strings.Split(string(data), "\n")
	entries := make([]map[string]any, 0)

	// Unix风格的ls -l输出正则表达式
	unixRegex := regexp.MustCompile(`^([drwx-]{10})\s+\d+\s+(\S+)\s+(\S+)\s+(\d+)\s+(\S+\s+\S+\s+\S+)\s+(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		entry := make(map[string]any)

		// 尝试解析Unix风格
		if matches := unixRegex.FindStringSubmatch(line); len(matches) == 7 {
			entry["permissions"] = matches[1]
			entry["owner"] = matches[2]
			entry["group"] = matches[3]
			entry["size"] = matches[4]
			entry["date"] = matches[5]
			entry["name"] = matches[6]
			entry["type"] = "file"
			if strings.HasPrefix(matches[1], "d") {
				entry["type"] = "directory"
			}
		} else {
			// 简单解析
			entry["name"] = line
			entry["type"] = "unknown"
		}

		entries = append(entries, entry)
	}

	return entries
}

// detectFileContentType 检测文件内容类型
func (f *FTPParser) detectFileContentType(data []byte) string {
	if len(data) == 0 {
		return "application/octet-stream"
	}

	// 检查文件头
	if len(data) >= 4 {
		header := data[:4]

		// PDF
		if string(header) == "%PDF" {
			return "application/pdf"
		}

		// ZIP
		if header[0] == 0x50 && header[1] == 0x4B {
			return "application/zip"
		}

		// JPEG
		if header[0] == 0xFF && header[1] == 0xD8 {
			return "image/jpeg"
		}

		// PNG
		if header[0] == 0x89 && header[1] == 0x50 && header[2] == 0x4E && header[3] == 0x47 {
			return "image/png"
		}
	}

	// 检查是否为文本
	if f.isTextData(data) {
		return "text/plain"
	}

	return "application/octet-stream"
}

// isTextData 检查是否为文本数据
func (f *FTPParser) isTextData(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// 检查前1024字节
	checkSize := min(len(data), 1024)
	for i := 0; i < checkSize; i++ {
		b := data[i]
		// 检查是否为可打印字符或常见控制字符
		if b < 32 && b != 9 && b != 10 && b != 13 {
			return false
		}
		if b > 126 {
			return false
		}
	}

	return true
}

// getSessionID 获取会话ID
func (f *FTPParser) getSessionID(packet *interceptor.PacketInfo) string {
	return fmt.Sprintf("%s:%d-%s:%d",
		packet.SourceIP.String(), packet.SourcePort,
		packet.DestIP.String(), packet.DestPort)
}

// getOrCreateSession 获取或创建会话
func (f *FTPParser) getOrCreateSession(sessionID string, packet *interceptor.PacketInfo) *FTPSession {
	if session, exists := f.sessions[sessionID]; exists {
		session.LastUsed = time.Now()
		return session
	}

	session := &FTPSession{
		SessionID: sessionID,
		ControlConn: &FTPConnection{
			SourceIP:   packet.SourceIP.String(),
			DestIP:     packet.DestIP.String(),
			SourcePort: packet.SourcePort,
			DestPort:   packet.DestPort,
			IsActive:   true,
		},
		CurrentDir:   "/",
		TransferMode: "binary",
		CreatedAt:    time.Now(),
		LastUsed:     time.Now(),
	}

	f.sessions[sessionID] = session
	return session
}
