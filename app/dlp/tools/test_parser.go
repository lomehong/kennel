package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dlp/interceptor"
	"dlp/parser"
	"github.com/lomehong/kennel/pkg/logging"
)

func main() {
	// 初始化日志
	config := &logging.LogConfig{
		Level:            logging.LogLevelDebug,
		Format:           logging.LogFormatText,
		Output:           logging.LogOutputStdout,
		IncludeLocation:  true,
		IncludeTimestamp: true,
		TimeFormat:       "2006-01-02 15:04:05",
	}

	logger, err := logging.NewEnhancedLogger(config)
	if err != nil {
		log.Fatalf("创建日志记录器失败: %v", err)
	}
	defer logger.Close()

	// 创建解析器配置
	parserConfig := parser.ParserConfig{
		MaxSessions:    1000,
		SessionTimeout: 30 * 60,   // 30分钟
		BufferSize:     64 * 1024, // 64KB
	}

	// 创建解析器管理器
	parserManager := parser.NewProtocolManager(logger, parserConfig)

	// 注册解析器
	registerParsers(parserManager, logger)

	// 启动管理器
	if err := parserManager.Start(); err != nil {
		log.Fatalf("启动解析器管理器失败: %v", err)
	}
	defer parserManager.Stop()

	// 测试HTTP请求解析
	testHTTPParsing(parserManager, logger)

	// 测试HTTPS请求解析
	testHTTPSParsing(parserManager, logger)

	// 测试未知协议解析
	testUnknownProtocolParsing(parserManager, logger)
}

func registerParsers(pm parser.ProtocolManager, logger logging.Logger) {
	// 注册HTTP解析器
	httpParser := parser.NewHTTPParser(logger)
	if err := pm.RegisterParser(httpParser); err != nil {
		log.Fatalf("注册HTTP解析器失败: %v", err)
	}

	// 注册HTTPS解析器
	tlsConfig := &parser.TLSConfig{
		InsecureSkipVerify: true, // 测试时跳过证书验证
		CertFile:           "",
		KeyFile:            "",
	}
	httpsParser := parser.NewHTTPSParser(logger, tlsConfig)
	if err := pm.RegisterParser(httpsParser); err != nil {
		log.Fatalf("注册HTTPS解析器失败: %v", err)
	}

	// 注册MySQL解析器
	mysqlParser := parser.NewMySQLParser(logger)
	if err := pm.RegisterParser(mysqlParser); err != nil {
		log.Fatalf("注册MySQL解析器失败: %v", err)
	}

	// 注册SMTP解析器
	smtpParser := parser.NewSMTPParser(logger)
	if err := pm.RegisterParser(smtpParser); err != nil {
		log.Fatalf("注册SMTP解析器失败: %v", err)
	}

	// 注册默认解析器
	defaultParser := parser.NewDefaultParser(logger)
	if err := pm.RegisterParser(defaultParser); err != nil {
		log.Fatalf("注册默认解析器失败: %v", err)
	}

	logger.Info("所有解析器注册完成")
}

func testHTTPParsing(pm parser.ProtocolManager, logger logging.Logger) {
	logger.Info("测试HTTP解析器")

	// 构造HTTP请求数据包
	httpRequest := "GET /api/users?page=1&limit=10 HTTP/1.1\r\n" +
		"Host: api.example.com\r\n" +
		"User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36\r\n" +
		"Accept: application/json\r\n" +
		"Authorization: Bearer token123\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 0\r\n" +
		"\r\n"

	packet := &interceptor.PacketInfo{
		SourceIP:   net.ParseIP("192.168.1.100"),
		DestIP:     net.ParseIP("203.0.113.10"),
		SourcePort: 54321,
		DestPort:   80,
		Protocol:   interceptor.ProtocolTCP,
		Direction:  interceptor.PacketDirectionOutbound,
		Size:       len(httpRequest),
		Payload:    []byte(httpRequest),
		Timestamp:  time.Now(),
	}

	// 解析数据包
	parsedData, err := pm.ParsePacket(packet)
	if err != nil {
		logger.Error("HTTP解析失败", "error", err)
		return
	}

	// 输出解析结果
	printParsedData("HTTP", parsedData, logger)
}

func testHTTPSParsing(pm parser.ProtocolManager, logger logging.Logger) {
	logger.Info("测试HTTPS解析器")

	// 构造HTTPS TLS握手数据包（简化版本）
	tlsHandshake := []byte{
		0x16, 0x03, 0x01, 0x00, 0x2f, // TLS Record Header (Content Type: Handshake, Version: TLS 1.0, Length: 47)
		0x01, 0x00, 0x00, 0x2b, // Handshake Header (Type: Client Hello, Length: 43)
		0x03, 0x03, // Version: TLS 1.2
		// Random (32 bytes) - simplified
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
		0x00,       // Session ID Length: 0
		0x00, 0x02, // Cipher Suites Length: 2
		0x00, 0x35, // Cipher Suite: TLS_RSA_WITH_AES_256_CBC_SHA
		0x01, 0x00, // Compression Methods Length: 1, Method: null
	}

	packet := &interceptor.PacketInfo{
		SourceIP:   net.ParseIP("192.168.1.100"),
		DestIP:     net.ParseIP("203.0.113.10"),
		SourcePort: 54322,
		DestPort:   443,
		Protocol:   interceptor.ProtocolTCP,
		Direction:  interceptor.PacketDirectionOutbound,
		Size:       len(tlsHandshake),
		Payload:    tlsHandshake,
		Timestamp:  time.Now(),
	}

	// 解析数据包
	parsedData, err := pm.ParsePacket(packet)
	if err != nil {
		logger.Error("HTTPS解析失败", "error", err)
		return
	}

	// 输出解析结果
	printParsedData("HTTPS", parsedData, logger)
}

func testUnknownProtocolParsing(pm parser.ProtocolManager, logger logging.Logger) {
	logger.Info("测试未知协议解析器")

	// 构造未知协议数据包
	unknownData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	packet := &interceptor.PacketInfo{
		SourceIP:   net.ParseIP("192.168.1.100"),
		DestIP:     net.ParseIP("203.0.113.10"),
		SourcePort: 54323,
		DestPort:   9999,
		Protocol:   interceptor.ProtocolTCP,
		Direction:  interceptor.PacketDirectionOutbound,
		Size:       len(unknownData),
		Payload:    unknownData,
		Timestamp:  time.Now(),
	}

	// 解析数据包
	parsedData, err := pm.ParsePacket(packet)
	if err != nil {
		logger.Error("未知协议解析失败", "error", err)
		return
	}

	// 输出解析结果
	printParsedData("Unknown", parsedData, logger)
}

func printParsedData(testName string, data *parser.ParsedData, logger logging.Logger) {
	fmt.Printf("\n=== %s 解析结果 ===\n", testName)
	fmt.Printf("协议: %s\n", data.Protocol)
	fmt.Printf("URL: %s\n", data.URL)
	fmt.Printf("方法: %s\n", data.Method)
	fmt.Printf("内容类型: %s\n", data.ContentType)
	fmt.Printf("状态码: %d\n", data.StatusCode)
	fmt.Printf("数据大小: %d bytes\n", len(data.Body))

	if data.Headers != nil && len(data.Headers) > 0 {
		fmt.Printf("头部信息:\n")
		for key, value := range data.Headers {
			if strings.ToLower(key) == "authorization" || strings.ToLower(key) == "cookie" {
				fmt.Printf("  %s: [REDACTED]\n", key)
			} else {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
	}

	if data.Metadata != nil && len(data.Metadata) > 0 {
		fmt.Printf("元数据:\n")
		for key, value := range data.Metadata {
			if strings.Contains(strings.ToLower(key), "password") ||
				strings.Contains(strings.ToLower(key), "token") ||
				strings.Contains(strings.ToLower(key), "secret") {
				fmt.Printf("  %s: [REDACTED]\n", key)
			} else {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}
	}

	fmt.Printf("=== 结束 ===\n\n")
}

func init() {
	// 确保日志目录存在
	logDir := filepath.Join("..", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("创建日志目录失败: %v", err)
	}
}
