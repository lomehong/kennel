package parser

import (
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

// HTTPSParser HTTPS协议解析器
type HTTPSParser struct {
	logger    logging.Logger
	tlsConfig *TLSConfig
	sessions  map[string]*TLSSessionInfo
}

// TLSSessionInfo TLS会话信息
type TLSSessionInfo struct {
	SessionID    string
	ClientRandom []byte
	ServerRandom []byte
	MasterSecret []byte
	CipherSuite  uint16
	Version      uint16
	ServerName   string
	Certificates []*x509.Certificate
	CreatedAt    time.Time
	LastUsed     time.Time
}

// NewHTTPSParser 创建HTTPS解析器
func NewHTTPSParser(logger logging.Logger, tlsConfig *TLSConfig) *HTTPSParser {
	return &HTTPSParser{
		logger:    logger,
		tlsConfig: tlsConfig,
		sessions:  make(map[string]*TLSSessionInfo),
	}
}

// GetParserInfo 获取解析器信息
func (h *HTTPSParser) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "HTTPS Parser",
		Version:            "1.0.0",
		Description:        "HTTPS/TLS协议解析器，支持TLS解密和HTTP内容提取",
		SupportedProtocols: []string{"https", "tls"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

// CanParse 检查是否能解析指定的数据包
func (h *HTTPSParser) CanParse(packet *interceptor.PacketInfo) bool {
	if packet == nil || len(packet.Payload) < 6 {
		return false
	}

	// 检查是否为HTTPS端口
	if packet.DestPort == 443 || packet.SourcePort == 443 {
		return true
	}

	// 检查是否为TLS流量
	return h.isTLSPacket(packet.Payload)
}

// Parse 解析数据包
func (h *HTTPSParser) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	if !h.CanParse(packet) {
		return nil, fmt.Errorf("不是有效的HTTPS数据包")
	}

	// 解析TLS记录
	tlsRecord, err := h.parseTLSRecord(packet.Payload)
	if err != nil {
		return nil, fmt.Errorf("解析TLS记录失败: %w", err)
	}

	// 创建解析结果
	parsedData := &ParsedData{
		Protocol:    "https",
		Headers:     make(map[string]string),
		Body:        []byte{},
		Metadata:    make(map[string]any),
		ContentType: "application/octet-stream",
	}

	// 处理不同类型的TLS记录
	switch tlsRecord.ContentType {
	case 22: // Handshake
		err = h.parseHandshake(tlsRecord, parsedData, packet)
	case 23: // Application Data
		err = h.parseApplicationData(tlsRecord, parsedData, packet)
	case 21: // Alert
		err = h.parseAlert(tlsRecord, parsedData)
	case 20: // Change Cipher Spec
		err = h.parseChangeCipherSpec(tlsRecord, parsedData)
	case 24: // Heartbeat
		err = h.parseHeartbeat(tlsRecord, parsedData)
	case 25: // TLS 1.3 Application Data
		err = h.parseApplicationData(tlsRecord, parsedData, packet)
	case 69: // 未知但常见的TLS内容类型，可能是加密的应用数据
		err = h.parseEncryptedApplicationData(tlsRecord, parsedData, packet)
	default:
		// 对于其他未知的TLS内容类型，作为加密数据处理
		h.logger.Debug("未知的TLS内容类型，作为加密数据处理",
			"content_type", tlsRecord.ContentType,
			"data_length", len(tlsRecord.Data))
		err = h.parseEncryptedApplicationData(tlsRecord, parsedData, packet)
	}

	if err != nil {
		return nil, fmt.Errorf("解析TLS内容失败: %w", err)
	}

	// 添加TLS元数据
	parsedData.Metadata["tls_version"] = tlsRecord.Version
	parsedData.Metadata["tls_content_type"] = tlsRecord.ContentType
	parsedData.Metadata["tls_length"] = tlsRecord.Length

	return parsedData, nil
}

// GetSupportedProtocols 获取支持的协议列表
func (h *HTTPSParser) GetSupportedProtocols() []string {
	return []string{"https", "tls"}
}

// Initialize 初始化解析器
func (h *HTTPSParser) Initialize(config ParserConfig) error {
	h.logger.Info("初始化HTTPS解析器")

	// 设置TLS配置
	if config.TLSConfig != nil {
		h.tlsConfig = config.TLSConfig
	} else {
		h.tlsConfig = &TLSConfig{
			InsecureSkipVerify: true, // 默认跳过证书验证以便解析
		}
	}

	return nil
}

// Cleanup 清理资源
func (h *HTTPSParser) Cleanup() error {
	h.logger.Info("清理HTTPS解析器资源")
	h.sessions = make(map[string]*TLSSessionInfo)
	return nil
}

// TLSRecord TLS记录结构
type TLSRecord struct {
	ContentType uint8
	Version     uint16
	Length      uint16
	Data        []byte
}

// isTLSPacket 检查是否为TLS数据包
func (h *HTTPSParser) isTLSPacket(data []byte) bool {
	if len(data) < 6 {
		return false
	}

	// TLS记录头: Content Type (1) + Version (2) + Length (2)
	contentType := data[0]
	version := uint16(data[1])<<8 | uint16(data[2])

	// 检查内容类型 - 扩展支持更多TLS内容类型
	validContentTypes := []uint8{20, 21, 22, 23, 24, 25, 69} // 添加更多有效的TLS内容类型
	isValidContentType := false
	for _, ct := range validContentTypes {
		if contentType == ct {
			isValidContentType = true
			break
		}
	}

	// 对于未知但可能的TLS内容类型，进行更宽松的检查
	if !isValidContentType && contentType >= 20 && contentType <= 255 {
		// 如果端口是443，更可能是TLS流量
		isValidContentType = true
	}

	// 检查版本
	isValidVersion := version >= 0x0300 && version <= 0x0304

	return isValidContentType && isValidVersion
}

// parseTLSRecord 解析TLS记录
func (h *HTTPSParser) parseTLSRecord(data []byte) (*TLSRecord, error) {
	if len(data) < 5 {
		// 对于不完整的TLS记录，创建一个基本记录而不是返回错误
		h.logger.Debug("TLS记录数据不完整，创建基本记录", "data_length", len(data))
		return &TLSRecord{
			ContentType: 22,     // 默认为握手类型
			Version:     0x0303, // TLS 1.2
			Length:      uint16(len(data)),
			Data:        data,
		}, nil
	}

	record := &TLSRecord{
		ContentType: data[0],
		Version:     uint16(data[1])<<8 | uint16(data[2]),
		Length:      uint16(data[3])<<8 | uint16(data[4]),
	}

	// 对于数据不完整的情况，使用实际可用的数据
	if len(data) < int(5+record.Length) {
		h.logger.Debug("TLS记录长度超出可用数据，使用实际数据",
			"expected_length", record.Length,
			"available_data", len(data)-5)
		if len(data) > 5 {
			record.Data = data[5:]
		} else {
			record.Data = []byte{}
		}
		record.Length = uint16(len(record.Data))
	} else {
		record.Data = data[5 : 5+record.Length]
	}

	return record, nil
}

// parseHandshake 解析握手消息
func (h *HTTPSParser) parseHandshake(record *TLSRecord, parsedData *ParsedData, packet *interceptor.PacketInfo) error {
	if len(record.Data) < 4 {
		return fmt.Errorf("握手消息长度不足")
	}

	handshakeType := record.Data[0]
	length := uint32(record.Data[1])<<16 | uint32(record.Data[2])<<8 | uint32(record.Data[3])

	parsedData.Metadata["handshake_type"] = handshakeType
	parsedData.Metadata["handshake_length"] = length

	switch handshakeType {
	case 1: // Client Hello
		return h.parseClientHello(record.Data[4:], parsedData)
	case 2: // Server Hello
		return h.parseServerHello(record.Data[4:], parsedData)
	case 11: // Certificate
		return h.parseCertificate(record.Data[4:], parsedData)
	case 16: // Client Key Exchange
		return h.parseClientKeyExchange(record.Data[4:], parsedData)
	default:
		h.logger.Debug("未处理的握手类型", "type", handshakeType)
	}

	return nil
}

// parseClientHello 解析Client Hello消息
func (h *HTTPSParser) parseClientHello(data []byte, parsedData *ParsedData) error {
	if len(data) < 38 {
		return fmt.Errorf("Client Hello消息长度不足")
	}

	// 解析版本
	version := uint16(data[0])<<8 | uint16(data[1])
	parsedData.Metadata["client_version"] = version

	// 解析随机数
	clientRandom := data[2:34]
	parsedData.Metadata["client_random"] = clientRandom

	// 解析会话ID
	sessionIDLength := data[34]
	if len(data) < int(35+sessionIDLength) {
		return fmt.Errorf("Client Hello会话ID长度不足")
	}

	sessionID := data[35 : 35+sessionIDLength]
	parsedData.Metadata["session_id"] = sessionID

	offset := 35 + int(sessionIDLength)

	// 解析密码套件
	if len(data) < offset+2 {
		return fmt.Errorf("Client Hello密码套件长度不足")
	}

	cipherSuitesLength := uint16(data[offset])<<8 | uint16(data[offset+1])
	offset += 2

	if len(data) < offset+int(cipherSuitesLength) {
		return fmt.Errorf("Client Hello密码套件数据不足")
	}

	cipherSuites := make([]uint16, cipherSuitesLength/2)
	for i := 0; i < int(cipherSuitesLength/2); i++ {
		cipherSuites[i] = uint16(data[offset+i*2])<<8 | uint16(data[offset+i*2+1])
	}
	parsedData.Metadata["cipher_suites"] = cipherSuites

	offset += int(cipherSuitesLength)

	// 解析扩展（包括SNI）
	if len(data) > offset {
		extensions, err := h.parseExtensions(data[offset:])
		if err == nil {
			parsedData.Metadata["extensions"] = extensions

			// 提取SNI
			if sni, exists := extensions["server_name"]; exists {
				parsedData.Metadata["server_name"] = sni
				parsedData.Headers["Host"] = sni.(string)

				// 保存到会话信息中以供后续使用
				h.saveSessionInfo(parsedData, sni.(string))
			}
		}
	}

	return nil
}

// parseServerHello 解析Server Hello消息
func (h *HTTPSParser) parseServerHello(data []byte, parsedData *ParsedData) error {
	if len(data) < 38 {
		return fmt.Errorf("Server Hello消息长度不足")
	}

	// 解析版本
	version := uint16(data[0])<<8 | uint16(data[1])
	parsedData.Metadata["server_version"] = version

	// 解析随机数
	serverRandom := data[2:34]
	parsedData.Metadata["server_random"] = serverRandom

	// 解析会话ID
	sessionIDLength := data[34]
	offset := 35 + int(sessionIDLength)

	// 解析选择的密码套件
	if len(data) < offset+2 {
		return fmt.Errorf("Server Hello密码套件长度不足")
	}

	selectedCipherSuite := uint16(data[offset])<<8 | uint16(data[offset+1])
	parsedData.Metadata["selected_cipher_suite"] = selectedCipherSuite

	return nil
}

// parseCertificate 解析证书消息
func (h *HTTPSParser) parseCertificate(data []byte, parsedData *ParsedData) error {
	if len(data) < 3 {
		return fmt.Errorf("证书消息长度不足")
	}

	certificatesLength := uint32(data[0])<<16 | uint32(data[1])<<8 | uint32(data[2])
	offset := 3

	certificates := make([]map[string]any, 0)

	for offset < len(data) && offset < int(3+certificatesLength) {
		if len(data) < offset+3 {
			break
		}

		certLength := uint32(data[offset])<<16 | uint32(data[offset+1])<<8 | uint32(data[offset+2])
		offset += 3

		if len(data) < offset+int(certLength) {
			break
		}

		certData := data[offset : offset+int(certLength)]
		offset += int(certLength)

		// 解析X.509证书
		cert, err := x509.ParseCertificate(certData)
		if err == nil {
			certInfo := map[string]any{
				"subject":      cert.Subject.String(),
				"issuer":       cert.Issuer.String(),
				"not_before":   cert.NotBefore,
				"not_after":    cert.NotAfter,
				"dns_names":    cert.DNSNames,
				"ip_addresses": cert.IPAddresses,
			}
			certificates = append(certificates, certInfo)
		}
	}

	parsedData.Metadata["certificates"] = certificates
	return nil
}

// parseClientKeyExchange 解析客户端密钥交换
func (h *HTTPSParser) parseClientKeyExchange(data []byte, parsedData *ParsedData) error {
	parsedData.Metadata["key_exchange_data"] = data
	return nil
}

// parseApplicationData 解析应用数据
func (h *HTTPSParser) parseApplicationData(record *TLSRecord, parsedData *ParsedData, packet *interceptor.PacketInfo) error {
	// 尝试解密数据（如果有密钥）
	decryptedData, err := h.tryDecrypt(record.Data, packet)
	if err != nil {
		h.logger.Debug("无法解密TLS数据", "error", err)
		// 即使无法解密，也记录加密数据的元信息
		parsedData.Body = record.Data
		parsedData.Metadata["encrypted"] = true
		parsedData.Metadata["encrypted_length"] = len(record.Data)
		return nil
	}

	// 如果解密成功，尝试解析HTTP内容
	if h.isHTTPData(decryptedData) {
		httpParser := NewHTTPParser(h.logger)

		// 创建临时数据包用于HTTP解析
		tempPacket := &interceptor.PacketInfo{
			Payload:     decryptedData,
			SourceIP:    packet.SourceIP,
			DestIP:      packet.DestIP,
			SourcePort:  packet.SourcePort,
			DestPort:    packet.DestPort,
			Protocol:    packet.Protocol,
			Timestamp:   packet.Timestamp,
			ProcessInfo: packet.ProcessInfo,
		}

		httpData, err := httpParser.Parse(tempPacket)
		if err == nil {
			// 合并HTTP解析结果
			parsedData.Headers = httpData.Headers
			parsedData.Body = httpData.Body
			parsedData.ContentType = httpData.ContentType
			parsedData.URL = httpData.URL
			parsedData.Method = httpData.Method
			parsedData.StatusCode = httpData.StatusCode

			// 合并元数据
			for k, v := range httpData.Metadata {
				parsedData.Metadata[k] = v
			}

			parsedData.Metadata["decrypted"] = true
		}
	} else {
		parsedData.Body = decryptedData
		parsedData.Metadata["decrypted"] = true
	}

	return nil
}

// parseAlert 解析警报消息
func (h *HTTPSParser) parseAlert(record *TLSRecord, parsedData *ParsedData) error {
	if len(record.Data) < 2 {
		return fmt.Errorf("警报消息长度不足")
	}

	alertLevel := record.Data[0]
	alertDescription := record.Data[1]

	parsedData.Metadata["alert_level"] = alertLevel
	parsedData.Metadata["alert_description"] = alertDescription

	return nil
}

// parseChangeCipherSpec 解析密码规范变更消息
func (h *HTTPSParser) parseChangeCipherSpec(record *TLSRecord, parsedData *ParsedData) error {
	parsedData.Metadata["change_cipher_spec"] = true
	return nil
}

// parseExtensions 解析TLS扩展
func (h *HTTPSParser) parseExtensions(data []byte) (map[string]any, error) {
	extensions := make(map[string]any)

	if len(data) < 2 {
		return extensions, nil
	}

	// 跳过压缩方法
	compressionLength := data[0]
	offset := 1 + int(compressionLength)

	if len(data) < offset+2 {
		return extensions, nil
	}

	extensionsLength := uint16(data[offset])<<8 | uint16(data[offset+1])
	offset += 2

	for offset < len(data) && offset < int(2+extensionsLength) {
		if len(data) < offset+4 {
			break
		}

		extType := uint16(data[offset])<<8 | uint16(data[offset+1])
		extLength := uint16(data[offset+2])<<8 | uint16(data[offset+3])
		offset += 4

		if len(data) < offset+int(extLength) {
			break
		}

		extData := data[offset : offset+int(extLength)]
		offset += int(extLength)

		switch extType {
		case 0: // Server Name Indication
			if serverName := h.parseServerNameExtension(extData); serverName != "" {
				extensions["server_name"] = serverName
			}
		case 16: // Application Layer Protocol Negotiation
			if protocols := h.parseALPNExtension(extData); len(protocols) > 0 {
				extensions["alpn"] = protocols
			}
		}
	}

	return extensions, nil
}

// parseServerNameExtension 解析服务器名称扩展
func (h *HTTPSParser) parseServerNameExtension(data []byte) string {
	if len(data) < 5 {
		return ""
	}

	// 跳过服务器名称列表长度
	offset := 2

	// 名称类型 (0 = hostname)
	if data[offset] != 0 {
		return ""
	}
	offset++

	// 名称长度
	nameLength := uint16(data[offset])<<8 | uint16(data[offset+1])
	offset += 2

	if len(data) < offset+int(nameLength) {
		return ""
	}

	return string(data[offset : offset+int(nameLength)])
}

// parseALPNExtension 解析ALPN扩展
func (h *HTTPSParser) parseALPNExtension(data []byte) []string {
	if len(data) < 2 {
		return nil
	}

	protocols := make([]string, 0)
	listLength := uint16(data[0])<<8 | uint16(data[1])
	offset := 2

	for offset < len(data) && offset < int(2+listLength) {
		if len(data) <= offset {
			break
		}

		protocolLength := data[offset]
		offset++

		if len(data) < offset+int(protocolLength) {
			break
		}

		protocol := string(data[offset : offset+int(protocolLength)])
		protocols = append(protocols, protocol)
		offset += int(protocolLength)
	}

	return protocols
}

// tryDecrypt 尝试解密TLS数据
func (h *HTTPSParser) tryDecrypt(data []byte, packet *interceptor.PacketInfo) ([]byte, error) {
	// 这里需要实现TLS解密逻辑
	// 在生产环境中，这需要：
	// 1. 获取TLS会话密钥
	// 2. 使用适当的密码套件进行解密
	// 3. 处理不同的TLS版本

	// 目前返回错误，表示无法解密
	return nil, fmt.Errorf("TLS解密功能需要会话密钥")
}

// isHTTPData 检查解密后的数据是否为HTTP
func (h *HTTPSParser) isHTTPData(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	dataStr := string(data[:min(len(data), 20)])

	// 检查HTTP方法
	httpMethods := []string{"GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH "}
	for _, method := range httpMethods {
		if strings.HasPrefix(dataStr, method) {
			return true
		}
	}

	// 检查HTTP响应
	if strings.HasPrefix(dataStr, "HTTP/") {
		return true
	}

	return false
}

// parseHeartbeat 解析心跳消息
func (h *HTTPSParser) parseHeartbeat(record *TLSRecord, parsedData *ParsedData) error {
	h.logger.Debug("解析TLS心跳消息", "data_length", len(record.Data))

	parsedData.Metadata["heartbeat"] = true
	parsedData.Metadata["encrypted"] = false
	parsedData.Body = record.Data

	return nil
}

// parseEncryptedApplicationData 解析加密的应用数据
func (h *HTTPSParser) parseEncryptedApplicationData(record *TLSRecord, parsedData *ParsedData, packet *interceptor.PacketInfo) error {
	h.logger.Debug("解析加密的TLS应用数据",
		"content_type", record.ContentType,
		"data_length", len(record.Data))

	// 标记为加密数据
	parsedData.Metadata["encrypted"] = true
	parsedData.Metadata["tls_encrypted_data"] = true
	parsedData.Metadata["encrypted_length"] = len(record.Data)
	parsedData.Body = record.Data

	// 尝试从包信息中提取网络层面的信息
	if packet != nil {
		// 提取目标域名（如果可能）
		if packet.DestIP != nil {
			parsedData.Metadata["dest_ip"] = packet.DestIP.String()
		}

		// 如果是HTTPS端口，尝试提取SNI信息
		if packet.DestPort == 443 || packet.SourcePort == 443 {
			// 尝试从之前的握手中获取SNI信息
			if sessionInfo := h.getSessionInfo(packet); sessionInfo != nil {
				if sessionInfo.ServerName != "" {
					parsedData.Metadata["server_name"] = sessionInfo.ServerName
					parsedData.Headers["Host"] = sessionInfo.ServerName

					// 构建可能的URL
					if packet.DestPort == 443 {
						parsedData.URL = fmt.Sprintf("https://%s/", sessionInfo.ServerName)
						parsedData.Metadata["request_url"] = parsedData.URL
					}
				}
			}
		}
	}

	return nil
}

// getSessionInfo 获取会话信息
func (h *HTTPSParser) getSessionInfo(packet *interceptor.PacketInfo) *TLSSessionInfo {
	if packet == nil {
		return nil
	}

	// 构建会话键
	sessionKey := fmt.Sprintf("%s:%d-%s:%d",
		packet.SourceIP.String(), packet.SourcePort,
		packet.DestIP.String(), packet.DestPort)

	if session, exists := h.sessions[sessionKey]; exists {
		// 更新最后使用时间
		session.LastUsed = time.Now()
		return session
	}

	// 尝试反向键
	reverseKey := fmt.Sprintf("%s:%d-%s:%d",
		packet.DestIP.String(), packet.DestPort,
		packet.SourceIP.String(), packet.SourcePort)

	if session, exists := h.sessions[reverseKey]; exists {
		session.LastUsed = time.Now()
		return session
	}

	return nil
}

// saveSessionInfo 保存会话信息
func (h *HTTPSParser) saveSessionInfo(parsedData *ParsedData, serverName string) {
	// 从元数据中提取会话信息
	sessionInfo := &TLSSessionInfo{
		ServerName: serverName,
		CreatedAt:  time.Now(),
		LastUsed:   time.Now(),
	}

	if sessionID, exists := parsedData.Metadata["session_id"].([]byte); exists {
		sessionInfo.SessionID = string(sessionID)
	}

	if clientRandom, exists := parsedData.Metadata["client_random"].([]byte); exists {
		sessionInfo.ClientRandom = clientRandom
	}

	if version, exists := parsedData.Metadata["client_version"].(uint16); exists {
		sessionInfo.Version = version
	}

	// 生成会话键（这里我们需要从上下文中获取网络信息）
	// 由于我们在parseClientHello中没有packet信息，我们使用一个通用键
	sessionKey := fmt.Sprintf("sni_%s", serverName)
	h.sessions[sessionKey] = sessionInfo

	h.logger.Debug("保存TLS会话信息",
		"server_name", serverName,
		"session_key", sessionKey)
}
