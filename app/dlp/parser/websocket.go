package parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// WebSocketParser WebSocket协议解析器
type WebSocketParser struct {
	logger            logging.Logger
	maxMessageSize    int
	timeout           time.Duration
	enableContentLog  bool
	sensitivePatterns []*regexp.Regexp
}

// WebSocket操作码
const (
	WSOpcodeContinuation = 0x0
	WSOpcodText          = 0x1
	WSOpcodeBinary       = 0x2
	WSOpcodClose         = 0x8
	WSOpcodePing         = 0x9
	WSOpcodePong         = 0xA
)

// WebSocket魔法字符串
const WebSocketMagicString = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// WebSocket数据包结构
type WebSocketPacket struct {
	IsHandshake   bool
	Opcode        uint8
	Masked        bool
	PayloadLength uint64
	MaskingKey    []byte
	Payload       []byte
	MessageType   string
	Content       string
	URL           string
	Origin        string
	Protocol      string
	Extensions    []string
	Sensitive     bool
}

// NewWebSocketParser 创建WebSocket解析器
func NewWebSocketParser(logger logging.Logger) *WebSocketParser {
	parser := &WebSocketParser{
		logger:           logger,
		maxMessageSize:   1048576, // 1MB
		timeout:          30 * time.Second,
		enableContentLog: true,
		sensitivePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(password|pwd|secret|token|key|auth)\b`),
			regexp.MustCompile(`(?i)\b(credit_card|ssn|social_security|phone|email)\b`),
			regexp.MustCompile(`(?i)\b(api_key|access_token|refresh_token|session_id)\b`),
			regexp.MustCompile(`(?i)"(password|token|secret|key)"\s*:\s*"[^"]+"`),
		},
	}

	parser.logger.Info("初始化WebSocket解析器",
		"max_message_size", parser.maxMessageSize,
		"timeout", parser.timeout,
		"enable_content_log", parser.enableContentLog)

	return parser
}

// GetName 获取解析器名称
func (w *WebSocketParser) GetName() string {
	return "WebSocket Parser"
}

// GetVersion 获取解析器版本
func (w *WebSocketParser) GetVersion() string {
	return "1.0.0"
}

// GetSupportedProtocols 获取支持的协议
func (w *WebSocketParser) GetSupportedProtocols() []string {
	return []string{"websocket", "ws", "wss"}
}

// CanParse 检查是否可以解析数据
func (w *WebSocketParser) CanParse(data []byte, metadata map[string]interface{}) bool {
	if len(data) < 2 {
		return false
	}

	// 检查是否为WebSocket握手
	if w.isWebSocketHandshake(data) {
		return true
	}

	// 检查是否为WebSocket帧
	return w.isWebSocketFrame(data)
}

// Parse 解析WebSocket数据包
func (w *WebSocketParser) Parse(data []byte, metadata map[string]interface{}) (*ParsedData, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		if duration > w.timeout {
			w.logger.Warn("WebSocket解析超时", "duration", duration)
		}
	}()

	if len(data) > w.maxMessageSize {
		w.logger.Warn("WebSocket数据包过大，截断处理", "size", len(data), "max_size", w.maxMessageSize)
		data = data[:w.maxMessageSize]
	}

	packet, err := w.parsePacket(data)
	if err != nil {
		return nil, fmt.Errorf("解析WebSocket数据包失败: %w", err)
	}

	// 构建解析结果
	result := &ParsedData{
		Protocol:    "websocket",
		ContentType: "application/websocket",
		Headers:     make(map[string]string),
		Body:        []byte(packet.Content),
		Metadata:    make(map[string]interface{}),
	}

	// 填充元数据
	result.Metadata["is_handshake"] = packet.IsHandshake
	result.Metadata["opcode"] = packet.Opcode
	result.Metadata["message_type"] = packet.MessageType
	result.Metadata["masked"] = packet.Masked
	result.Metadata["payload_length"] = packet.PayloadLength
	result.Metadata["sensitive"] = packet.Sensitive

	// 填充头部信息
	result.Headers["WebSocket-Type"] = packet.MessageType
	if packet.IsHandshake {
		result.Headers["WebSocket-Handshake"] = "true"
		if packet.URL != "" {
			result.Headers["WebSocket-URL"] = packet.URL
		}
		if packet.Origin != "" {
			result.Headers["WebSocket-Origin"] = packet.Origin
		}
		if packet.Protocol != "" {
			result.Headers["WebSocket-Protocol"] = packet.Protocol
		}
		if len(packet.Extensions) > 0 {
			result.Headers["WebSocket-Extensions"] = strings.Join(packet.Extensions, ", ")
		}
	} else {
		result.Headers["WebSocket-Opcode"] = fmt.Sprintf("0x%X", packet.Opcode)
		result.Headers["WebSocket-Masked"] = fmt.Sprintf("%t", packet.Masked)
		result.Headers["WebSocket-Length"] = fmt.Sprintf("%d", packet.PayloadLength)
	}

	// 敏感数据检测
	if packet.Sensitive {
		result.Headers["Sensitive-Data"] = "true"
		w.logger.Warn("检测到敏感WebSocket内容",
			"message_type", packet.MessageType,
			"url", packet.URL,
			"content_preview", w.truncateString(packet.Content, 100))
	}

	if w.enableContentLog && packet.Content != "" {
		w.logger.Debug("WebSocket消息解析",
			"message_type", packet.MessageType,
			"url", packet.URL,
			"content_preview", w.truncateString(packet.Content, 200))
	}

	return result, nil
}

// parsePacket 解析WebSocket数据包
func (w *WebSocketParser) parsePacket(data []byte) (*WebSocketPacket, error) {
	packet := &WebSocketPacket{}

	// 检查是否为握手
	if w.isWebSocketHandshake(data) {
		return w.parseHandshake(data, packet)
	}

	// 解析WebSocket帧
	return w.parseFrame(data, packet)
}

// parseHandshake 解析WebSocket握手
func (w *WebSocketParser) parseHandshake(data []byte, packet *WebSocketPacket) (*WebSocketPacket, error) {
	packet.IsHandshake = true
	packet.MessageType = "HANDSHAKE"

	content := string(data)
	lines := strings.Split(content, "\r\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析请求行
		if strings.HasPrefix(line, "GET ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				packet.URL = parts[1]
			}
		}

		// 解析头部
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch strings.ToLower(key) {
				case "origin":
					packet.Origin = value
				case "sec-websocket-protocol":
					packet.Protocol = value
				case "sec-websocket-extensions":
					packet.Extensions = strings.Split(value, ",")
					for i := range packet.Extensions {
						packet.Extensions[i] = strings.TrimSpace(packet.Extensions[i])
					}
				}
			}
		}
	}

	packet.Content = content
	packet.Sensitive = w.detectSensitiveContent(packet.Content)

	return packet, nil
}

// parseFrame 解析WebSocket帧
func (w *WebSocketParser) parseFrame(data []byte, packet *WebSocketPacket) (*WebSocketPacket, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("WebSocket帧太短")
	}

	packet.IsHandshake = false

	reader := bytes.NewReader(data)

	// 读取第一个字节
	var firstByte uint8
	binary.Read(reader, binary.BigEndian, &firstByte)

	// 提取操作码
	packet.Opcode = firstByte & 0x0F
	packet.MessageType = w.getOpcodeString(packet.Opcode)

	// 读取第二个字节
	var secondByte uint8
	binary.Read(reader, binary.BigEndian, &secondByte)

	// 检查是否有掩码
	packet.Masked = (secondByte & 0x80) != 0

	// 获取载荷长度
	payloadLen := uint64(secondByte & 0x7F)

	if payloadLen == 126 {
		// 16位长度
		var extLen uint16
		binary.Read(reader, binary.BigEndian, &extLen)
		payloadLen = uint64(extLen)
	} else if payloadLen == 127 {
		// 64位长度
		binary.Read(reader, binary.BigEndian, &payloadLen)
	}

	packet.PayloadLength = payloadLen

	// 读取掩码键（如果存在）
	if packet.Masked {
		packet.MaskingKey = make([]byte, 4)
		reader.Read(packet.MaskingKey)
	}

	// 读取载荷数据
	if payloadLen > 0 && reader.Len() > 0 {
		remainingData := make([]byte, reader.Len())
		reader.Read(remainingData)

		if uint64(len(remainingData)) > payloadLen {
			remainingData = remainingData[:payloadLen]
		}

		packet.Payload = remainingData

		// 如果有掩码，解码数据
		if packet.Masked && len(packet.MaskingKey) == 4 {
			for i := range packet.Payload {
				packet.Payload[i] ^= packet.MaskingKey[i%4]
			}
		}

		// 根据操作码处理内容
		switch packet.Opcode {
		case WSOpcodText:
			packet.Content = string(packet.Payload)
		case WSOpcodeBinary:
			packet.Content = fmt.Sprintf("Binary data (%d bytes)", len(packet.Payload))
		case WSOpcodClose:
			packet.Content = "Connection close"
		case WSOpcodePing:
			packet.Content = "Ping"
		case WSOpcodePong:
			packet.Content = "Pong"
		default:
			packet.Content = fmt.Sprintf("Unknown opcode 0x%X", packet.Opcode)
		}
	}

	packet.Sensitive = w.detectSensitiveContent(packet.Content)

	return packet, nil
}

// getOpcodeString 获取操作码字符串
func (w *WebSocketParser) getOpcodeString(opcode uint8) string {
	switch opcode {
	case WSOpcodeContinuation:
		return "CONTINUATION"
	case WSOpcodText:
		return "TEXT"
	case WSOpcodeBinary:
		return "BINARY"
	case WSOpcodClose:
		return "CLOSE"
	case WSOpcodePing:
		return "PING"
	case WSOpcodePong:
		return "PONG"
	default:
		return fmt.Sprintf("UNKNOWN_0x%X", opcode)
	}
}

// isWebSocketHandshake 检查是否为WebSocket握手
func (w *WebSocketParser) isWebSocketHandshake(data []byte) bool {
	content := string(data)

	// 检查HTTP升级请求
	if strings.Contains(content, "GET ") &&
		strings.Contains(strings.ToLower(content), "upgrade: websocket") &&
		strings.Contains(strings.ToLower(content), "sec-websocket-key:") {
		return true
	}

	// 检查HTTP升级响应
	if strings.Contains(content, "HTTP/1.1 101") &&
		strings.Contains(strings.ToLower(content), "upgrade: websocket") &&
		strings.Contains(strings.ToLower(content), "sec-websocket-accept:") {
		return true
	}

	return false
}

// isWebSocketFrame 检查是否为WebSocket帧
func (w *WebSocketParser) isWebSocketFrame(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	// 检查第一个字节的操作码
	opcode := data[0] & 0x0F
	switch opcode {
	case WSOpcodeContinuation, WSOpcodText, WSOpcodeBinary, WSOpcodClose, WSOpcodePing, WSOpcodePong:
		return true
	}

	return false
}

// detectSensitiveContent 检测敏感内容
func (w *WebSocketParser) detectSensitiveContent(content string) bool {
	if content == "" {
		return false
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range w.sensitivePatterns {
		if pattern.MatchString(contentLower) {
			return true
		}
	}

	return false
}

// truncateString 截断字符串
func (w *WebSocketParser) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Cleanup 清理资源
func (w *WebSocketParser) Cleanup() error {
	w.logger.Info("清理WebSocket解析器资源")
	return nil
}
