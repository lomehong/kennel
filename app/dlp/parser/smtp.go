package parser

import (
	"bufio"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

// SMTPParser SMTP协议解析器
type SMTPParser struct {
	logger   logging.Logger
	sessions map[string]*SMTPSession
}

// SMTPSession SMTP会话信息
type SMTPSession struct {
	SessionID   string
	State       SMTPState
	From        string
	Recipients  []string
	DataMode    bool
	MessageData []byte
	CreatedAt   time.Time
	LastUsed    time.Time
}

// SMTPState SMTP会话状态
type SMTPState int

const (
	SMTPStateInit SMTPState = iota
	SMTPStateGreeting
	SMTPStateAuth
	SMTPStateReady
	SMTPStateData
	SMTPStateQuit
)

// SMTPCommand SMTP命令结构
type SMTPCommand struct {
	Command    string
	Parameters []string
	RawLine    string
}

// SMTPResponse SMTP响应结构
type SMTPResponse struct {
	Code    int
	Message string
	RawLine string
}

// NewSMTPParser 创建SMTP解析器
func NewSMTPParser(logger logging.Logger) *SMTPParser {
	return &SMTPParser{
		logger:   logger,
		sessions: make(map[string]*SMTPSession),
	}
}

// GetParserInfo 获取解析器信息
func (s *SMTPParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "SMTP Parser",
		Version:            "1.0.0",
		Description:        "SMTP协议解析器，支持邮件传输协议解析",
		SupportedProtocols: []string{"smtp"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

// CanParse 检查是否能解析指定的数据包
func (s *SMTPParser) CanParse(packet *interceptor.PacketInfo) bool {
	if packet == nil || len(packet.Payload) == 0 {
		return false
	}

	// 检查端口
	if packet.DestPort == 25 || packet.SourcePort == 25 ||
		packet.DestPort == 587 || packet.SourcePort == 587 ||
		packet.DestPort == 465 || packet.SourcePort == 465 {
		return true
	}

	// 检查内容
	return s.isSMTPData(packet.Payload)
}

// Parse 解析数据包
func (s *SMTPParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	if !s.CanParse(packet) {
		return nil, fmt.Errorf("不是有效的SMTP数据包")
	}

	parsedData := &ParsedData{
		Protocol:    "smtp",
		Headers:     make(map[string]string),
		Body:        packet.Payload,
		Metadata:    make(map[string]any),
		ContentType: "text/plain",
	}

	// 获取或创建会话
	sessionID := s.getSessionID(packet)
	session := s.getOrCreateSession(sessionID, packet)

	// 解析SMTP数据
	return s.parseSMTPData(packet.Payload, parsedData, session)
}

// GetSupportedProtocols 获取支持的协议列表
func (s *SMTPParser) GetSupportedProtocols() []string {
	return []string{"smtp"}
}

// Initialize 初始化解析器
func (s *SMTPParser) Initialize(config ParserConfig) error {
	s.logger.Info("初始化SMTP解析器")
	return nil
}

// Cleanup 清理资源
func (s *SMTPParser) Cleanup() error {
	s.logger.Info("清理SMTP解析器资源")
	s.sessions = make(map[string]*SMTPSession)
	return nil
}

// isSMTPData 检查是否为SMTP数据
func (s *SMTPParser) isSMTPData(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	dataStr := strings.TrimSpace(string(data))

	// 检查SMTP响应码格式 (3位数字 + 空格或-)
	if len(dataStr) >= 4 {
		if dataStr[0] >= '2' && dataStr[0] <= '5' &&
			dataStr[1] >= '0' && dataStr[1] <= '9' &&
			dataStr[2] >= '0' && dataStr[2] <= '9' &&
			(dataStr[3] == ' ' || dataStr[3] == '-') {
			return true
		}
	}

	// 检查SMTP命令
	smtpCommands := []string{
		"HELO", "EHLO", "MAIL", "RCPT", "DATA", "RSET", "VRFY", "EXPN",
		"HELP", "NOOP", "QUIT", "AUTH", "STARTTLS",
	}

	upperData := strings.ToUpper(dataStr)
	for _, cmd := range smtpCommands {
		if strings.HasPrefix(upperData, cmd+" ") || upperData == cmd {
			return true
		}
	}

	return false
}

// parseSMTPData 解析SMTP数据
func (s *SMTPParser) parseSMTPData(data []byte, parsedData *ParsedData, session *SMTPSession) (*ParsedData, error) {
	dataStr := strings.TrimSpace(string(data))
	lines := strings.Split(dataStr, "\r\n")

	commands := make([]SMTPCommand, 0)
	responses := make([]SMTPResponse, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 如果会话处于DATA模式，所有数据都是邮件内容
		if session.DataMode {
			if line == "." {
				// 邮件结束标记
				session.DataMode = false
				session.State = SMTPStateReady

				// 解析邮件内容
				if err := s.parseEmailContent(session.MessageData, parsedData); err != nil {
					s.logger.Warn("解析邮件内容失败", "error", err)
				}

				session.MessageData = []byte{}
			} else {
				// 累积邮件数据
				session.MessageData = append(session.MessageData, []byte(line+"\r\n")...)
			}
			continue
		}

		// 尝试解析为响应
		if response, isResponse := s.parseResponse(line); isResponse {
			responses = append(responses, response)
		} else {
			// 解析为命令
			if command := s.parseCommand(line); command.Command != "" {
				commands = append(commands, command)
				s.handleCommand(command, session)
			}
		}
	}

	// 设置元数据
	if len(commands) > 0 {
		parsedData.Metadata["smtp_commands"] = commands
		parsedData.Metadata["command_count"] = len(commands)

		// 提取敏感信息
		for _, cmd := range commands {
			switch strings.ToUpper(cmd.Command) {
			case "MAIL":
				if len(cmd.Parameters) > 0 {
					from := s.extractEmailFromParam(cmd.Parameters[0])
					if from != "" {
						parsedData.Headers["From"] = from
						parsedData.Metadata["sender"] = from
					}
				}
			case "RCPT":
				if len(cmd.Parameters) > 0 {
					to := s.extractEmailFromParam(cmd.Parameters[0])
					if to != "" {
						if recipients, exists := parsedData.Metadata["recipients"]; exists {
							if recipientList, ok := recipients.([]string); ok {
								parsedData.Metadata["recipients"] = append(recipientList, to)
							}
						} else {
							parsedData.Metadata["recipients"] = []string{to}
						}
					}
				}
			case "AUTH":
				parsedData.Metadata["authentication"] = true
				if len(cmd.Parameters) > 0 {
					parsedData.Metadata["auth_method"] = cmd.Parameters[0]
				}
			case "DATA":
				parsedData.Metadata["has_email_data"] = true
			}
		}
	}

	if len(responses) > 0 {
		parsedData.Metadata["smtp_responses"] = responses
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

	// 如果有邮件数据，设置相关信息
	if len(session.MessageData) > 0 {
		parsedData.Metadata["email_size"] = len(session.MessageData)
		parsedData.Metadata["has_email_content"] = true
	}

	// 更新会话信息
	session.LastUsed = time.Now()

	return parsedData, nil
}

// parseCommand 解析SMTP命令
func (s *SMTPParser) parseCommand(line string) SMTPCommand {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return SMTPCommand{}
	}

	command := SMTPCommand{
		Command:    strings.ToUpper(parts[0]),
		Parameters: parts[1:],
		RawLine:    line,
	}

	return command
}

// parseResponse 解析SMTP响应
func (s *SMTPParser) parseResponse(line string) (SMTPResponse, bool) {
	// SMTP响应格式: 3位数字 + 空格或- + 消息
	if len(line) < 4 {
		return SMTPResponse{}, false
	}

	codeStr := line[:3]
	code, err := strconv.Atoi(codeStr)
	if err != nil {
		return SMTPResponse{}, false
	}

	if line[3] != ' ' && line[3] != '-' {
		return SMTPResponse{}, false
	}

	response := SMTPResponse{
		Code:    code,
		Message: strings.TrimSpace(line[4:]),
		RawLine: line,
	}

	return response, true
}

// handleCommand 处理SMTP命令，更新会话状态
func (s *SMTPParser) handleCommand(command SMTPCommand, session *SMTPSession) {
	switch strings.ToUpper(command.Command) {
	case "HELO", "EHLO":
		session.State = SMTPStateGreeting
	case "AUTH":
		session.State = SMTPStateAuth
	case "MAIL":
		if len(command.Parameters) > 0 {
			session.From = s.extractEmailFromParam(command.Parameters[0])
		}
		session.State = SMTPStateReady
	case "RCPT":
		if len(command.Parameters) > 0 {
			recipient := s.extractEmailFromParam(command.Parameters[0])
			if recipient != "" {
				session.Recipients = append(session.Recipients, recipient)
			}
		}
	case "DATA":
		session.State = SMTPStateData
		session.DataMode = true
	case "RSET":
		session.From = ""
		session.Recipients = []string{}
		session.MessageData = []byte{}
		session.State = SMTPStateReady
	case "QUIT":
		session.State = SMTPStateQuit
	}
}

// extractEmailFromParam 从参数中提取邮件地址
func (s *SMTPParser) extractEmailFromParam(param string) string {
	// 处理 "FROM:<email>" 或 "TO:<email>" 格式
	if strings.Contains(param, ":") {
		parts := strings.Split(param, ":")
		if len(parts) > 1 {
			email := strings.Trim(parts[1], "<>")
			return email
		}
	}

	// 直接提取尖括号中的邮件地址
	if strings.Contains(param, "<") && strings.Contains(param, ">") {
		start := strings.Index(param, "<")
		end := strings.Index(param, ">")
		if start < end {
			return param[start+1 : end]
		}
	}

	return param
}

// parseEmailContent 解析邮件内容
func (s *SMTPParser) parseEmailContent(data []byte, parsedData *ParsedData) error {
	if len(data) == 0 {
		return nil
	}

	// 使用Go标准库解析邮件
	msg, err := mail.ReadMessage(strings.NewReader(string(data)))
	if err != nil {
		// 如果无法解析为标准邮件格式，尝试简单解析
		return s.parseEmailContentSimple(data, parsedData)
	}

	// 提取邮件头
	for key, values := range msg.Header {
		if len(values) > 0 {
			parsedData.Headers[key] = values[0]
		}
	}

	// 提取常见头部到元数据
	if from := msg.Header.Get("From"); from != "" {
		parsedData.Metadata["email_from"] = from
	}
	if to := msg.Header.Get("To"); to != "" {
		parsedData.Metadata["email_to"] = to
	}
	if cc := msg.Header.Get("Cc"); cc != "" {
		parsedData.Metadata["email_cc"] = cc
	}
	if subject := msg.Header.Get("Subject"); subject != "" {
		parsedData.Metadata["email_subject"] = subject
	}
	if date := msg.Header.Get("Date"); date != "" {
		parsedData.Metadata["email_date"] = date
	}
	if messageID := msg.Header.Get("Message-ID"); messageID != "" {
		parsedData.Metadata["email_message_id"] = messageID
	}

	// 读取邮件正文
	scanner := bufio.NewScanner(msg.Body)
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}

	body := strings.Join(bodyLines, "\n")
	parsedData.Body = []byte(body)
	parsedData.Metadata["email_body"] = body
	parsedData.Metadata["email_body_size"] = len(body)

	// 检测内容类型
	contentType := msg.Header.Get("Content-Type")
	if contentType != "" {
		parsedData.ContentType = contentType
		parsedData.Metadata["email_content_type"] = contentType

		// 检查是否为HTML邮件
		if strings.Contains(strings.ToLower(contentType), "text/html") {
			parsedData.Metadata["is_html_email"] = true
		}
	}

	// 检查是否有附件
	if strings.Contains(strings.ToLower(contentType), "multipart") {
		parsedData.Metadata["has_attachments"] = true
	}

	return nil
}

// parseEmailContentSimple 简单解析邮件内容
func (s *SMTPParser) parseEmailContentSimple(data []byte, parsedData *ParsedData) error {
	content := string(data)
	lines := strings.Split(content, "\n")

	headerEnd := -1
	headers := make(map[string]string)

	// 查找头部结束位置
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			headerEnd = i
			break
		}

		// 解析头部
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				headers[key] = value
				parsedData.Headers[key] = value
			}
		}
	}

	// 提取常见头部
	if from, exists := headers["From"]; exists {
		parsedData.Metadata["email_from"] = from
	}
	if to, exists := headers["To"]; exists {
		parsedData.Metadata["email_to"] = to
	}
	if subject, exists := headers["Subject"]; exists {
		parsedData.Metadata["email_subject"] = subject
	}

	// 提取正文
	if headerEnd >= 0 && headerEnd < len(lines)-1 {
		bodyLines := lines[headerEnd+1:]
		body := strings.Join(bodyLines, "\n")
		parsedData.Body = []byte(body)
		parsedData.Metadata["email_body"] = body
		parsedData.Metadata["email_body_size"] = len(body)
	}

	return nil
}

// getSessionID 获取会话ID
func (s *SMTPParser) getSessionID(packet *interceptor.PacketInfo) string {
	return fmt.Sprintf("%s:%d-%s:%d",
		packet.SourceIP.String(), packet.SourcePort,
		packet.DestIP.String(), packet.DestPort)
}

// getOrCreateSession 获取或创建会话
func (s *SMTPParser) getOrCreateSession(sessionID string, packet *interceptor.PacketInfo) *SMTPSession {
	if session, exists := s.sessions[sessionID]; exists {
		session.LastUsed = time.Now()
		return session
	}

	session := &SMTPSession{
		SessionID:   sessionID,
		State:       SMTPStateInit,
		Recipients:  make([]string, 0),
		DataMode:    false,
		MessageData: make([]byte, 0),
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
	}

	s.sessions[sessionID] = session
	return session
}
