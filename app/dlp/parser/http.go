package parser

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

// HTTPParserImpl HTTP协议解析器实现
type HTTPParserImpl struct {
	config     ParserConfig
	logger     logging.Logger
	sessionMgr SessionManager
	stats      ParserStats
	mu         sync.RWMutex

	// TLS/SSL 支持
	tlsConfig *tls.Config
	certStore map[string]*tls.Certificate

	// 会话跟踪
	sessions  map[string]*SessionInfo
	sessionMu sync.RWMutex
}

// NewHTTPParser 创建HTTP解析器
func NewHTTPParser(logger logging.Logger) ProtocolParser {
	return &HTTPParserImpl{
		logger:    logger,
		certStore: make(map[string]*tls.Certificate),
		sessions:  make(map[string]*SessionInfo),
		tlsConfig: &tls.Config{
			InsecureSkipVerify: false, // 生产环境中应该验证证书
			MinVersion:         tls.VersionTLS12,
		},
	}
}

// GetParserInfo 获取解析器信息
func (h *HTTPParserImpl) GetParserInfo() ParserInfo {
	return ParserInfo{
		Name:               "HTTP Parser",
		Version:            "1.0.0",
		Description:        "HTTP协议解析器",
		SupportedProtocols: []string{"http"},
		Author:             "DLP Team",
		License:            "MIT",
	}
}

// CanParse 检查是否能解析指定的数据包
func (h *HTTPParserImpl) CanParse(packet *interceptor.PacketInfo) bool {
	// 检查是否是TCP协议
	if packet.Protocol != interceptor.ProtocolTCP {
		return false
	}

	// 检查端口（只处理HTTP端口，不处理HTTPS）
	if packet.DestPort == 80 || packet.DestPort == 8080 {
		return true
	}

	// 检查数据包内容是否包含HTTP特征（明文HTTP）
	payload := string(packet.Payload)
	return h.isHTTPTraffic(payload) && !h.isTLSTraffic(packet.Payload)
}

// Parse 解析数据包
func (h *HTTPParserImpl) Parse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	payload := packet.Payload
	if len(payload) == 0 {
		return nil, fmt.Errorf("数据包为空")
	}

	// 尝试解析HTTP请求
	if h.isHTTPRequest(payload) {
		return h.parseHTTPRequest(packet)
	}

	// 尝试解析HTTP响应
	if h.isHTTPResponse(payload) {
		return h.parseHTTPResponse(packet)
	}

	return nil, fmt.Errorf("不是有效的HTTP流量")
}

// GetSupportedProtocols 获取支持的协议列表
func (h *HTTPParserImpl) GetSupportedProtocols() []string {
	return []string{"http"}
}

// Initialize 初始化解析器
func (h *HTTPParserImpl) Initialize(config ParserConfig) error {
	h.config = config
	h.logger.Info("初始化HTTP解析器",
		"max_body_size", config.MaxBodySize,
		"timeout", config.Timeout)
	return nil
}

// Cleanup 清理资源
func (h *HTTPParserImpl) Cleanup() error {
	h.logger.Info("清理HTTP解析器资源")
	return nil
}

// ParseRequest 解析HTTP请求
func (h *HTTPParserImpl) ParseRequest(data []byte) (*http.Request, error) {
	reader := bufio.NewReader(bytes.NewReader(data))
	req, err := http.ReadRequest(reader)
	if err != nil {
		return nil, fmt.Errorf("解析HTTP请求失败: %w", err)
	}
	return req, nil
}

// ParseResponse 解析HTTP响应
func (h *HTTPParserImpl) ParseResponse(data []byte) (*http.Response, error) {
	reader := bufio.NewReader(bytes.NewReader(data))
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		return nil, fmt.Errorf("解析HTTP响应失败: %w", err)
	}
	return resp, nil
}

// ExtractHeaders 提取HTTP头部
func (h *HTTPParserImpl) ExtractHeaders(req *http.Request) map[string]string {
	headers := make(map[string]string)
	for name, values := range req.Header {
		headers[name] = strings.Join(values, ", ")
	}
	return headers
}

// ExtractBody 提取HTTP主体
func (h *HTTPParserImpl) ExtractBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}

	// 限制读取大小
	limitReader := io.LimitReader(req.Body, h.config.MaxBodySize)
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("读取HTTP主体失败: %w", err)
	}

	return body, nil
}

// isHTTPTraffic 检查是否是HTTP流量
func (h *HTTPParserImpl) isHTTPTraffic(payload string) bool {
	// 检查HTTP方法
	httpMethods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH", "TRACE", "CONNECT"}
	for _, method := range httpMethods {
		if strings.HasPrefix(payload, method+" ") {
			return true
		}
	}

	// 检查HTTP响应
	if strings.HasPrefix(payload, "HTTP/") {
		return true
	}

	return false
}

// isHTTPRequest 检查是否是HTTP请求
func (h *HTTPParserImpl) isHTTPRequest(payload []byte) bool {
	payloadStr := string(payload)
	httpMethods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH", "TRACE", "CONNECT"}

	for _, method := range httpMethods {
		if strings.HasPrefix(payloadStr, method+" ") {
			return true
		}
	}

	return false
}

// isHTTPResponse 检查是否是HTTP响应
func (h *HTTPParserImpl) isHTTPResponse(payload []byte) bool {
	return strings.HasPrefix(string(payload), "HTTP/")
}

// parseHTTPRequest 解析HTTP请求
func (h *HTTPParserImpl) parseHTTPRequest(packet *interceptor.PacketInfo) (*ParsedData, error) {
	req, err := h.ParseRequest(packet.Payload)
	if err != nil {
		return nil, err
	}

	// 提取头部信息
	headers := h.ExtractHeaders(req)

	// 提取主体内容
	body, err := h.ExtractBody(req)
	if err != nil {
		h.logger.Warn("提取HTTP主体失败", "error", err)
		body = nil
	}

	// 构建解析结果
	data := &ParsedData{
		Protocol:    "HTTP",
		Headers:     headers,
		Body:        body,
		ContentType: req.Header.Get("Content-Type"),
		URL:         req.URL.String(),
		Method:      req.Method,
		Metadata: map[string]interface{}{
			"host":           req.Host,
			"user_agent":     req.Header.Get("User-Agent"),
			"content_length": req.ContentLength,
			"remote_addr":    req.RemoteAddr,
			"request_uri":    req.RequestURI,
		},
	}

	// 创建会话信息
	sessionID := fmt.Sprintf("%s:%d-%s:%d",
		packet.SourceIP.String(), packet.SourcePort,
		packet.DestIP.String(), packet.DestPort)

	session := &SessionInfo{
		ID:         sessionID,
		Protocol:   "HTTP",
		SourceIP:   packet.SourceIP.String(),
		DestIP:     packet.DestIP.String(),
		SourcePort: packet.SourcePort,
		DestPort:   packet.DestPort,
		StartTime:  packet.Timestamp,
		LastSeen:   packet.Timestamp,
		BytesSent:  uint64(packet.Size),
		State:      SessionStateEstablished,
		Metadata: map[string]interface{}{
			"method": req.Method,
			"url":    req.URL.String(),
			"host":   req.Host,
		},
	}

	data.Sessions = []*SessionInfo{session}

	h.logger.Debug("解析HTTP请求成功",
		"method", req.Method,
		"url", req.URL.String(),
		"host", req.Host,
		"content_length", len(body))

	return data, nil
}

// parseHTTPResponse 解析HTTP响应
func (h *HTTPParserImpl) parseHTTPResponse(packet *interceptor.PacketInfo) (*ParsedData, error) {
	resp, err := h.ParseResponse(packet.Payload)
	if err != nil {
		return nil, err
	}

	// 提取头部信息
	headers := make(map[string]string)
	for name, values := range resp.Header {
		headers[name] = strings.Join(values, ", ")
	}

	// 提取主体内容
	var body []byte
	if resp.Body != nil {
		limitReader := io.LimitReader(resp.Body, h.config.MaxBodySize)
		body, err = io.ReadAll(limitReader)
		if err != nil {
			h.logger.Warn("读取HTTP响应主体失败", "error", err)
			body = nil
		}
		resp.Body.Close()
	}

	// 构建解析结果
	data := &ParsedData{
		Protocol:    "HTTP",
		Headers:     headers,
		Body:        body,
		ContentType: resp.Header.Get("Content-Type"),
		StatusCode:  resp.StatusCode,
		Metadata: map[string]interface{}{
			"status":         resp.Status,
			"status_code":    resp.StatusCode,
			"content_length": resp.ContentLength,
			"server":         resp.Header.Get("Server"),
			"location":       resp.Header.Get("Location"),
		},
	}

	// 创建会话信息
	sessionID := fmt.Sprintf("%s:%d-%s:%d",
		packet.DestIP.String(), packet.DestPort,
		packet.SourceIP.String(), packet.SourcePort)

	session := &SessionInfo{
		ID:         sessionID,
		Protocol:   "HTTP",
		SourceIP:   packet.DestIP.String(),
		DestIP:     packet.SourceIP.String(),
		SourcePort: packet.DestPort,
		DestPort:   packet.SourcePort,
		StartTime:  packet.Timestamp,
		LastSeen:   packet.Timestamp,
		BytesRecv:  uint64(packet.Size),
		State:      SessionStateEstablished,
		Metadata: map[string]interface{}{
			"status_code": resp.StatusCode,
			"status":      resp.Status,
		},
	}

	data.Sessions = []*SessionInfo{session}

	h.logger.Debug("解析HTTP响应成功",
		"status_code", resp.StatusCode,
		"content_type", resp.Header.Get("Content-Type"),
		"content_length", len(body))

	return data, nil
}

// extractURLInfo 提取URL信息
func (h *HTTPParserImpl) extractURLInfo(rawURL string) map[string]interface{} {
	info := make(map[string]interface{})

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return info
	}

	info["scheme"] = parsedURL.Scheme
	info["host"] = parsedURL.Host
	info["path"] = parsedURL.Path
	info["query"] = parsedURL.RawQuery
	info["fragment"] = parsedURL.Fragment

	// 提取查询参数
	if parsedURL.RawQuery != "" {
		params := make(map[string]string)
		for key, values := range parsedURL.Query() {
			params[key] = strings.Join(values, ", ")
		}
		info["params"] = params
	}

	return info
}

// detectContentType 检测内容类型
func (h *HTTPParserImpl) detectContentType(body []byte, contentType string) string {
	if contentType != "" {
		return contentType
	}

	// 使用Go标准库检测内容类型
	return http.DetectContentType(body)
}

// LoadTLSCertificate 加载TLS证书用于解密
func (h *HTTPParserImpl) LoadTLSCertificate(certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("加载TLS证书失败: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// 解析证书以获取域名
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("解析证书失败: %w", err)
	}

	// 存储证书，使用域名作为键
	for _, name := range x509Cert.DNSNames {
		h.certStore[name] = &cert
	}

	// 如果有通用名称，也存储
	if x509Cert.Subject.CommonName != "" {
		h.certStore[x509Cert.Subject.CommonName] = &cert
	}

	h.logger.Info("加载TLS证书成功",
		"domains", x509Cert.DNSNames,
		"common_name", x509Cert.Subject.CommonName)

	return nil
}

// DecryptTLSTraffic 解密TLS流量
func (h *HTTPParserImpl) DecryptTLSTraffic(packet *interceptor.PacketInfo) ([]byte, error) {
	// 检查是否是TLS流量
	if !h.isTLSTraffic(packet.Payload) {
		return packet.Payload, nil
	}

	// 这里应该实现TLS解密逻辑
	// 实际实现需要：
	// 1. 解析TLS握手
	// 2. 提取服务器名称指示(SNI)
	// 3. 查找对应的私钥
	// 4. 解密应用数据

	h.logger.Debug("检测到TLS流量，需要解密", "packet_id", packet.ID)

	// 简化实现：返回原始数据
	// 生产环境中需要实现完整的TLS解密
	return packet.Payload, fmt.Errorf("TLS解密功能需要完整实现")
}

// isTLSTraffic 检查是否是TLS流量
func (h *HTTPParserImpl) isTLSTraffic(payload []byte) bool {
	if len(payload) < 5 {
		return false
	}

	// TLS记录头部：类型(1字节) + 版本(2字节) + 长度(2字节)
	// 类型：20(ChangeCipherSpec), 21(Alert), 22(Handshake), 23(ApplicationData)
	recordType := payload[0]
	version := uint16(payload[1])<<8 | uint16(payload[2])

	// 检查TLS版本
	validVersions := []uint16{
		0x0301, // TLS 1.0
		0x0302, // TLS 1.1
		0x0303, // TLS 1.2
		0x0304, // TLS 1.3
	}

	validType := recordType >= 20 && recordType <= 23
	validVersion := false
	for _, v := range validVersions {
		if version == v {
			validVersion = true
			break
		}
	}

	return validType && validVersion
}

// GetSessionInfo 获取会话信息
func (h *HTTPParserImpl) GetSessionInfo(sessionID string) *SessionInfo {
	h.sessionMu.RLock()
	defer h.sessionMu.RUnlock()

	return h.sessions[sessionID]
}

// UpdateSession 更新会话信息
func (h *HTTPParserImpl) UpdateSession(session *SessionInfo) {
	h.sessionMu.Lock()
	defer h.sessionMu.Unlock()

	if existing, exists := h.sessions[session.ID]; exists {
		// 更新现有会话
		existing.LastSeen = session.LastSeen
		existing.BytesSent += session.BytesSent
		existing.BytesRecv += session.BytesRecv
		existing.State = session.State

		// 合并元数据
		for k, v := range session.Metadata {
			existing.Metadata[k] = v
		}
	} else {
		// 创建新会话
		h.sessions[session.ID] = session
	}
}

// CleanupExpiredSessions 清理过期会话
func (h *HTTPParserImpl) CleanupExpiredSessions(maxAge time.Duration) {
	h.sessionMu.Lock()
	defer h.sessionMu.Unlock()

	now := time.Now()
	for id, session := range h.sessions {
		if now.Sub(session.LastSeen) > maxAge {
			delete(h.sessions, id)
		}
	}
}

// GetStats 获取解析器统计信息
func (h *HTTPParserImpl) GetStats() ParserStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.stats
}
