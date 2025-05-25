// Package interceptor - Mock Implementation
//
// 警告：此文件包含模拟实现，仅用于以下场景：
// 1. 不支持的操作系统平台的后备实现
// 2. 开发和测试环境
// 3. 演示和概念验证
//
// 在生产环境中，应该使用平台特定的真实拦截器：
// - Windows: WinDivert
// - Linux: Netfilter
// - macOS: PF (Packet Filter)

package interceptor

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
)

// MockPlatformInterceptor 模拟平台拦截器（用于不支持的平台）
type MockPlatformInterceptor struct {
	config       InterceptorConfig
	packetCh     chan *PacketInfo
	stopCh       chan struct{}
	stats        InterceptorStats
	logger       logging.Logger
	processCache ProcessCache
	running      int32
	mu           sync.RWMutex
}

// NewMockPlatformInterceptor 创建模拟平台拦截器
func NewMockPlatformInterceptor(logger logging.Logger) PlatformInterceptor {
	return &MockPlatformInterceptor{
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// StartCapture 开始捕获数据包
func (m *MockPlatformInterceptor) StartCapture() error {
	if !atomic.CompareAndSwapInt32(&m.running, 0, 1) {
		return fmt.Errorf("拦截器已在运行")
	}

	m.logger.Info("启动模拟拦截器")
	m.packetCh = make(chan *PacketInfo, m.config.ChannelSize)

	// 启动模拟数据包生成
	go m.simulateTraffic()

	return nil
}

// StopCapture 停止捕获数据包
func (m *MockPlatformInterceptor) StopCapture() error {
	if !atomic.CompareAndSwapInt32(&m.running, 1, 0) {
		return fmt.Errorf("拦截器未在运行")
	}

	m.logger.Info("停止模拟拦截器")

	// 发送停止信号
	close(m.stopCh)

	// 关闭数据包通道
	if m.packetCh != nil {
		close(m.packetCh)
	}

	return nil
}

// SetFilter 设置过滤规则
func (m *MockPlatformInterceptor) SetFilter(filter string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.Filter = filter
	m.logger.Info("设置模拟过滤规则", "filter", filter)
	return nil
}

// GetPacketChannel 获取数据包通道
func (m *MockPlatformInterceptor) GetPacketChannel() <-chan *PacketInfo {
	return m.packetCh
}

// ReinjectPacket 重新注入数据包
func (m *MockPlatformInterceptor) ReinjectPacket(packet *PacketInfo) error {
	atomic.AddUint64(&m.stats.PacketsReinject, 1)
	m.logger.Debug("模拟重新注入数据包", "packet_id", packet.ID)
	return nil
}

// simulateTraffic 模拟流量生成
func (m *MockPlatformInterceptor) simulateTraffic() {
	m.logger.Debug("开始模拟流量生成")
	defer m.logger.Debug("模拟流量生成结束")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	packetID := 0
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			packetID++
			packet := m.createMockPacket(packetID)

			atomic.AddUint64(&m.stats.PacketsProcessed, 1)
			atomic.AddUint64(&m.stats.BytesProcessed, uint64(packet.Size))

			select {
			case m.packetCh <- packet:
				m.logger.Debug("生成模拟数据包", "packet_id", packet.ID)
			case <-m.stopCh:
				return
			default:
				atomic.AddUint64(&m.stats.PacketsDropped, 1)
				m.logger.Warn("数据包通道已满，丢弃数据包")
			}
		}
	}
}

// createMockPacket 创建模拟数据包
func (m *MockPlatformInterceptor) createMockPacket(id int) *PacketInfo {
	// 根据ID创建不同类型的模拟数据包
	switch id % 4 {
	case 0:
		return m.createHTTPSAPIRequest(id)
	case 1:
		return m.createHTTPFormSubmit(id)
	case 2:
		return m.createHTTPFileUpload(id)
	default:
		return m.createHTTPJSONRequest(id)
	}
}

// createHTTPSAPIRequest 创建HTTPS API请求
func (m *MockPlatformInterceptor) createHTTPSAPIRequest(id int) *PacketInfo {
	httpPayload := "GET /api/v1/users?page=1&limit=10 HTTP/1.1\r\nHost: api.example.com\r\nUser-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64)\r\nAuthorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9\r\nAccept: application/json\r\n\r\n"

	return &PacketInfo{
		ID:         fmt.Sprintf("mock_https_%d_%d", time.Now().Unix(), id),
		Timestamp:  time.Now(),
		Protocol:   ProtocolTCP,
		SourceIP:   net.ParseIP("192.168.1.100"),
		DestIP:     net.ParseIP("203.0.113.1"),
		SourcePort: uint16(12345 + id),
		DestPort:   443,
		Direction:  PacketDirectionOutbound,
		Payload:    []byte(httpPayload),
		Size:       len(httpPayload),
		ProcessInfo: &ProcessInfo{
			PID:         1234,
			ProcessName: "chrome.exe",
			ExecutePath: "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			User:        "user",
			CommandLine: "chrome.exe --new-window",
		},
		Metadata: map[string]interface{}{
			"mock":      true,
			"packet_id": id,
			"timestamp": time.Now().Unix(),
			"protocol":  "https",
		},
	}
}

// createHTTPFormSubmit 创建HTTP表单提交
func (m *MockPlatformInterceptor) createHTTPFormSubmit(id int) *PacketInfo {
	formData := "username=john.doe&email=john%40example.com&password=secret123&remember=on"
	httpPayload := fmt.Sprintf("POST /login HTTP/1.1\r\nHost: www.example.com\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: %d\r\nUser-Agent: Mozilla/5.0\r\n\r\n%s", len(formData), formData)

	return &PacketInfo{
		ID:         fmt.Sprintf("mock_form_%d_%d", time.Now().Unix(), id),
		Timestamp:  time.Now(),
		Protocol:   ProtocolTCP,
		SourceIP:   net.ParseIP("192.168.1.100"),
		DestIP:     net.ParseIP("203.0.113.2"),
		SourcePort: uint16(12346 + id),
		DestPort:   80,
		Direction:  PacketDirectionOutbound,
		Payload:    []byte(httpPayload),
		Size:       len(httpPayload),
		ProcessInfo: &ProcessInfo{
			PID:         5678,
			ProcessName: "firefox.exe",
			ExecutePath: "C:\\Program Files\\Mozilla Firefox\\firefox.exe",
			User:        "user",
			CommandLine: "firefox.exe -new-tab",
		},
		Metadata: map[string]interface{}{
			"mock":      true,
			"packet_id": id,
			"timestamp": time.Now().Unix(),
		},
	}
}

// createHTTPFileUpload 创建HTTP文件上传
func (m *MockPlatformInterceptor) createHTTPFileUpload(id int) *PacketInfo {
	boundary := "----WebKitFormBoundary7MA4YWxkTrZu0gW"
	multipartData := fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"file\"; filename=\"document.pdf\"\r\nContent-Type: application/pdf\r\n\r\n%%PDF-1.4 binary data...\r\n--%s--\r\n", boundary, boundary)
	httpPayload := fmt.Sprintf("POST /upload HTTP/1.1\r\nHost: files.example.com\r\nContent-Type: multipart/form-data; boundary=%s\r\nContent-Length: %d\r\n\r\n%s", boundary, len(multipartData), multipartData)

	return &PacketInfo{
		ID:         fmt.Sprintf("mock_upload_%d_%d", time.Now().Unix(), id),
		Timestamp:  time.Now(),
		Protocol:   ProtocolTCP,
		SourceIP:   net.ParseIP("192.168.1.100"),
		DestIP:     net.ParseIP("203.0.113.3"),
		SourcePort: uint16(12347 + id),
		DestPort:   443,
		Direction:  PacketDirectionOutbound,
		Payload:    []byte(httpPayload),
		Size:       len(httpPayload),
		ProcessInfo: &ProcessInfo{
			PID:         9012,
			ProcessName: "outlook.exe",
			ExecutePath: "C:\\Program Files\\Microsoft Office\\root\\Office16\\OUTLOOK.EXE",
			User:        "user",
			CommandLine: "outlook.exe",
		},
		Metadata: map[string]interface{}{
			"mock":      true,
			"packet_id": id,
			"timestamp": time.Now().Unix(),
			"protocol":  "https",
		},
	}
}

// createHTTPJSONRequest 创建HTTP JSON请求
func (m *MockPlatformInterceptor) createHTTPJSONRequest(id int) *PacketInfo {
	jsonData := `{"username":"alice","email":"alice@company.com","department":"engineering","salary":75000,"api_key":"sk-1234567890abcdef"}`
	httpPayload := fmt.Sprintf("POST /api/employees HTTP/1.1\r\nHost: hr.company.com\r\nContent-Type: application/json\r\nContent-Length: %d\r\nAuthorization: Bearer company-token-xyz\r\n\r\n%s", len(jsonData), jsonData)

	return &PacketInfo{
		ID:         fmt.Sprintf("mock_json_%d_%d", time.Now().Unix(), id),
		Timestamp:  time.Now(),
		Protocol:   ProtocolTCP,
		SourceIP:   net.ParseIP("192.168.1.100"),
		DestIP:     net.ParseIP("203.0.113.4"),
		SourcePort: uint16(12348 + id),
		DestPort:   443,
		Direction:  PacketDirectionOutbound,
		Payload:    []byte(httpPayload),
		Size:       len(httpPayload),
		ProcessInfo: &ProcessInfo{
			PID:         3456,
			ProcessName: "postman.exe",
			ExecutePath: "C:\\Users\\user\\AppData\\Local\\Postman\\Postman.exe",
			User:        "user",
			CommandLine: "postman.exe --no-sandbox",
		},
		Metadata: map[string]interface{}{
			"mock":      true,
			"packet_id": id,
			"timestamp": time.Now().Unix(),
			"protocol":  "https",
		},
	}
}
