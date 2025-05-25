package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/app/dlp/parser"
	"github.com/lomehong/kennel/pkg/logging"
)

var (
	testProtocol = flag.String("protocol", "all", "要测试的协议 (http, https, ftp, smtp, mysql, all)")
	verbose      = flag.Bool("verbose", false, "详细输出")
	duration     = flag.Int("duration", 30, "测试持续时间（秒）")
)

func main() {
	flag.Parse()

	// 创建日志记录器
	logConfig := logging.DefaultLogConfig()
	logConfig.Level = logging.LogLevelInfo
	if *verbose {
		logConfig.Level = logging.LogLevelDebug
	}

	baseLogger, err := logging.NewEnhancedLogger(logConfig)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		os.Exit(1)
	}
	logger := baseLogger.Named("multiprotocol-test")

	logger.Info("启动DLP多协议测试程序", "protocol", *testProtocol, "duration", *duration)

	// 创建协议解析器工厂
	factory := parser.NewParserFactory(logger)

	// 创建协议检测器
	detector := parser.NewProtocolDetector(logger)

	// 测试协议解析器
	if err := testParsers(factory, detector, logger); err != nil {
		logger.Error("协议解析器测试失败", "error", err)
		os.Exit(1)
	}

	// 如果指定了特定协议，进行深度测试
	if *testProtocol != "all" {
		if err := testSpecificProtocol(*testProtocol, factory, logger); err != nil {
			logger.Error("特定协议测试失败", "protocol", *testProtocol, "error", err)
			os.Exit(1)
		}
	}

	// 模拟网络流量测试
	if err := simulateNetworkTraffic(factory, detector, logger); err != nil {
		logger.Error("网络流量模拟测试失败", "error", err)
		os.Exit(1)
	}

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动测试定时器
	timer := time.NewTimer(time.Duration(*duration) * time.Second)

	logger.Info("测试运行中，按Ctrl+C停止或等待超时")

	select {
	case <-sigChan:
		logger.Info("收到中断信号，停止测试")
	case <-timer.C:
		logger.Info("测试时间到，停止测试")
	}

	logger.Info("DLP多协议测试完成")
}

// testParsers 测试所有协议解析器
func testParsers(factory parser.ParserFactory, detector *parser.ProtocolDetector, logger logging.Logger) error {
	logger.Info("开始测试协议解析器")

	// 获取支持的协议列表
	protocols := factory.GetSupportedProtocols()
	logger.Info("支持的协议", "count", len(protocols), "protocols", protocols)

	// 测试每个协议解析器的创建
	for _, protocol := range protocols {
		config := parser.ParserConfig{
			Logger: logger.Named(protocol),
		}

		parser, err := factory.CreateParser(protocol, config)
		if err != nil {
			return fmt.Errorf("创建%s解析器失败: %w", protocol, err)
		}

		// 初始化解析器
		if err := parser.Initialize(config); err != nil {
			return fmt.Errorf("初始化%s解析器失败: %w", protocol, err)
		}

		// 获取解析器信息
		info := parser.GetParserInfo()
		logger.Info("解析器信息",
			"protocol", protocol,
			"name", info.Name,
			"version", info.Version,
			"description", info.Description)

		// 清理解析器
		if err := parser.Cleanup(); err != nil {
			logger.Warn("清理解析器失败", "protocol", protocol, "error", err)
		}
	}

	logger.Info("协议解析器测试完成")
	return nil
}

// testSpecificProtocol 测试特定协议
func testSpecificProtocol(protocol string, factory parser.ParserFactory, logger logging.Logger) error {
	logger.Info("开始特定协议测试", "protocol", protocol)

	config := parser.ParserConfig{
		Logger: logger.Named(protocol),
	}

	parser, err := factory.CreateParser(protocol, config)
	if err != nil {
		return fmt.Errorf("创建%s解析器失败: %w", protocol, err)
	}

	if err := parser.Initialize(config); err != nil {
		return fmt.Errorf("初始化%s解析器失败: %w", protocol, err)
	}

	// 根据协议类型生成测试数据
	testData := generateTestData(protocol)

	for i, data := range testData {
		packet := &interceptor.PacketInfo{
			Payload:    data.payload,
			SourceIP:   net.ParseIP(data.sourceIP),
			DestIP:     net.ParseIP(data.destIP),
			SourcePort: data.sourcePort,
			DestPort:   data.destPort,
			Protocol:   interceptor.ProtocolTCP,
			Timestamp:  time.Now(),
			ProcessInfo: &interceptor.ProcessInfo{
				PID:         1234,
				ProcessName: "test-process",
				ExecutePath: "/usr/bin/test",
			},
		}

		// 测试解析器是否能处理数据包
		if parser.CanParse(packet) {
			parsed, err := parser.Parse(packet)
			if err != nil {
				logger.Warn("解析数据包失败",
					"protocol", protocol,
					"test", i+1,
					"error", err)
				continue
			}

			logger.Info("解析成功",
				"protocol", protocol,
				"test", i+1,
				"content_type", parsed.ContentType,
				"body_size", len(parsed.Body),
				"headers_count", len(parsed.Headers),
				"metadata_count", len(parsed.Metadata))

			// 输出详细信息（如果启用详细模式）
			if *verbose {
				logger.Debug("解析详情",
					"headers", parsed.Headers,
					"metadata", parsed.Metadata)
			}
		} else {
			logger.Warn("解析器无法处理数据包", "protocol", protocol, "test", i+1)
		}
	}

	if err := parser.Cleanup(); err != nil {
		logger.Warn("清理解析器失败", "protocol", protocol, "error", err)
	}

	logger.Info("特定协议测试完成", "protocol", protocol)
	return nil
}

// simulateNetworkTraffic 模拟网络流量测试
func simulateNetworkTraffic(factory parser.ParserFactory, detector *parser.ProtocolDetector, logger logging.Logger) error {
	logger.Info("开始网络流量模拟测试")

	// 模拟不同协议的网络数据包
	testPackets := []struct {
		name     string
		payload  []byte
		destPort uint16
		expected string
	}{
		{
			name:     "HTTP GET请求",
			payload:  []byte("GET /api/users HTTP/1.1\r\nHost: example.com\r\n\r\n"),
			destPort: 80,
			expected: "http",
		},
		{
			name:     "HTTPS握手",
			payload:  []byte{0x16, 0x03, 0x01, 0x00, 0x05, 0x01, 0x00, 0x00, 0x01, 0x00},
			destPort: 443,
			expected: "https",
		},
		{
			name:     "FTP命令",
			payload:  []byte("USER anonymous\r\n"),
			destPort: 21,
			expected: "ftp",
		},
		{
			name:     "SMTP命令",
			payload:  []byte("HELO example.com\r\n"),
			destPort: 25,
			expected: "smtp",
		},
		{
			name:     "MySQL握手",
			payload:  []byte{0x4a, 0x00, 0x00, 0x00, 0x0a, 0x35, 0x2e, 0x37, 0x2e, 0x32, 0x39},
			destPort: 3306,
			expected: "mysql",
		},
	}

	for _, test := range testPackets {
		logger.Info("测试数据包", "name", test.name)

		// 协议检测测试
		detected := detector.DetectProtocol(test.payload, test.destPort)
		if detected != test.expected {
			logger.Warn("协议检测不匹配",
				"name", test.name,
				"expected", test.expected,
				"detected", detected)
		} else {
			logger.Info("协议检测成功", "name", test.name, "protocol", detected)
		}

		// 尝试解析数据包
		config := parser.ParserConfig{
			Logger: logger.Named(detected),
		}

		if parser, err := factory.CreateParser(detected, config); err == nil {
			packet := &interceptor.PacketInfo{
				Payload:    test.payload,
				SourceIP:   net.ParseIP("192.168.1.100"),
				DestIP:     net.ParseIP("192.168.1.1"),
				SourcePort: 12345,
				DestPort:   test.destPort,
				Protocol:   interceptor.ProtocolTCP,
				Timestamp:  time.Now(),
			}

			if parser.CanParse(packet) {
				if parsed, err := parser.Parse(packet); err == nil {
					logger.Info("数据包解析成功",
						"name", test.name,
						"protocol", parsed.Protocol,
						"content_type", parsed.ContentType)
				} else {
					logger.Warn("数据包解析失败", "name", test.name, "error", err)
				}
			}

			parser.Cleanup()
		}
	}

	logger.Info("网络流量模拟测试完成")
	return nil
}

// TestData 测试数据结构
type TestData struct {
	payload    []byte
	sourceIP   string
	destIP     string
	sourcePort uint16
	destPort   uint16
}

// generateTestData 生成测试数据
func generateTestData(protocol string) []TestData {
	switch protocol {
	case "http":
		return []TestData{
			{
				payload:    []byte("GET /index.html HTTP/1.1\r\nHost: example.com\r\n\r\n"),
				sourceIP:   "192.168.1.100",
				destIP:     "192.168.1.1",
				sourcePort: 12345,
				destPort:   80,
			},
			{
				payload:    []byte("POST /api/login HTTP/1.1\r\nHost: example.com\r\nContent-Length: 25\r\n\r\nusername=admin&password=123"),
				sourceIP:   "192.168.1.100",
				destIP:     "192.168.1.1",
				sourcePort: 12346,
				destPort:   80,
			},
		}

	case "ftp":
		return []TestData{
			{
				payload:    []byte("USER anonymous\r\n"),
				sourceIP:   "192.168.1.100",
				destIP:     "192.168.1.1",
				sourcePort: 12345,
				destPort:   21,
			},
			{
				payload:    []byte("PASS guest@example.com\r\n"),
				sourceIP:   "192.168.1.100",
				destIP:     "192.168.1.1",
				sourcePort: 12345,
				destPort:   21,
			},
		}

	case "smtp":
		return []TestData{
			{
				payload:    []byte("HELO example.com\r\n"),
				sourceIP:   "192.168.1.100",
				destIP:     "192.168.1.1",
				sourcePort: 12345,
				destPort:   25,
			},
			{
				payload:    []byte("MAIL FROM:<sender@example.com>\r\n"),
				sourceIP:   "192.168.1.100",
				destIP:     "192.168.1.1",
				sourcePort: 12345,
				destPort:   25,
			},
		}

	case "mysql":
		return []TestData{
			{
				payload:    []byte{0x4a, 0x00, 0x00, 0x00, 0x0a, 0x35, 0x2e, 0x37, 0x2e, 0x32, 0x39},
				sourceIP:   "192.168.1.100",
				destIP:     "192.168.1.1",
				sourcePort: 12345,
				destPort:   3306,
			},
		}

	default:
		return []TestData{
			{
				payload:    []byte("test data"),
				sourceIP:   "192.168.1.100",
				destIP:     "192.168.1.1",
				sourcePort: 12345,
				destPort:   8080,
			},
		}
	}
}
