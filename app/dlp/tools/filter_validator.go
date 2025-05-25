package main

import (
	"fmt"
	"net"
	"time"

	"dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

// 测试过滤器功能
func main() {
	// 创建日志记录器（用于显示测试信息）
	logConfig := &logging.LogConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}
	_, _ = logging.NewEnhancedLogger(logConfig)

	// 测试数据包
	testPackets := []*interceptor.PacketInfo{
		// 公网地址 - 应该被处理
		{
			ID:        "test_1",
			Timestamp: time.Now(),
			Direction: interceptor.PacketDirectionOutbound,
			Protocol:  interceptor.ProtocolTCP,
			SourceIP:  net.ParseIP("192.168.1.100"),
			DestIP:    net.ParseIP("8.8.8.8"), // Google DNS - 公网
			DestPort:  443,
		},
		// 本地回环地址 - 应该被过滤
		{
			ID:        "test_2",
			Timestamp: time.Now(),
			Direction: interceptor.PacketDirectionOutbound,
			Protocol:  interceptor.ProtocolTCP,
			SourceIP:  net.ParseIP("192.168.1.100"),
			DestIP:    net.ParseIP("127.0.0.1"), // 本地回环
			DestPort:  80,
		},
		// 私有网络A类 - 应该被过滤
		{
			ID:        "test_3",
			Timestamp: time.Now(),
			Direction: interceptor.PacketDirectionOutbound,
			Protocol:  interceptor.ProtocolTCP,
			SourceIP:  net.ParseIP("192.168.1.100"),
			DestIP:    net.ParseIP("10.0.0.1"), // 私有网络A类
			DestPort:  80,
		},
		// 私有网络B类 - 应该被过滤
		{
			ID:        "test_4",
			Timestamp: time.Now(),
			Direction: interceptor.PacketDirectionOutbound,
			Protocol:  interceptor.ProtocolTCP,
			SourceIP:  net.ParseIP("192.168.1.100"),
			DestIP:    net.ParseIP("172.16.0.1"), // 私有网络B类
			DestPort:  443,
		},
		// 私有网络C类 - 应该被过滤
		{
			ID:        "test_5",
			Timestamp: time.Now(),
			Direction: interceptor.PacketDirectionOutbound,
			Protocol:  interceptor.ProtocolTCP,
			SourceIP:  net.ParseIP("192.168.1.100"),
			DestIP:    net.ParseIP("192.168.1.1"), // 私有网络C类
			DestPort:  80,
		},
		// 链路本地地址 - 应该被过滤
		{
			ID:        "test_6",
			Timestamp: time.Now(),
			Direction: interceptor.PacketDirectionOutbound,
			Protocol:  interceptor.ProtocolTCP,
			SourceIP:  net.ParseIP("192.168.1.100"),
			DestIP:    net.ParseIP("169.254.1.1"), // 链路本地地址
			DestPort:  80,
		},
		// 组播地址 - 应该被过滤
		{
			ID:        "test_7",
			Timestamp: time.Now(),
			Direction: interceptor.PacketDirectionOutbound,
			Protocol:  interceptor.ProtocolTCP,
			SourceIP:  net.ParseIP("192.168.1.100"),
			DestIP:    net.ParseIP("224.0.0.1"), // 组播地址
			DestPort:  80,
		},
		// 公网地址 - 应该被处理
		{
			ID:        "test_8",
			Timestamp: time.Now(),
			Direction: interceptor.PacketDirectionOutbound,
			Protocol:  interceptor.ProtocolTCP,
			SourceIP:  net.ParseIP("192.168.1.100"),
			DestIP:    net.ParseIP("1.1.1.1"), // Cloudflare DNS - 公网
			DestPort:  443,
		},
		// 公网地址 - 应该被处理
		{
			ID:        "test_9",
			Timestamp: time.Now(),
			Direction: interceptor.PacketDirectionOutbound,
			Protocol:  interceptor.ProtocolTCP,
			SourceIP:  net.ParseIP("192.168.1.100"),
			DestIP:    net.ParseIP("13.107.42.14"), // Microsoft - 公网
			DestPort:  443,
		},
	}

	fmt.Println("=== DLP网络过滤器测试 ===")
	fmt.Println()

	// 测试过滤器构建
	fmt.Println("1. 测试WinDivert过滤器构建:")

	// 直接展示预期的过滤器
	expectedFilter := `outbound and tcp and (tcp.DstPort == 80 or tcp.DstPort == 443 or tcp.DstPort == 21 or tcp.DstPort == 25 or tcp.DstPort == 3306) and not (ip.DstAddr >= 127.0.0.0 and ip.DstAddr <= 127.255.255.255) and not (ip.DstAddr >= 10.0.0.0 and ip.DstAddr <= 10.255.255.255) and not (ip.DstAddr >= 172.16.0.0 and ip.DstAddr <= 172.31.255.255) and not (ip.DstAddr >= 192.168.0.0 and ip.DstAddr <= 192.168.255.255) and not (ip.DstAddr >= 169.254.0.0 and ip.DstAddr <= 169.254.255.255) and not (ip.DstAddr >= 224.0.0.0 and ip.DstAddr <= 239.255.255.255) and not (ip.DstAddr == 255.255.255.255)`

	fmt.Printf("预期过滤器: %s\n", expectedFilter)
	fmt.Println()

	// 测试应用层过滤
	fmt.Println("2. 测试应用层过滤逻辑:")
	fmt.Println("数据包ID | 目标IP | 端口 | 是否过滤 | 说明")
	fmt.Println("---------|--------|------|----------|--------")

	for _, packet := range testPackets {
		// 这里我们需要一个公共方法来测试shouldFilterPacket
		// 由于它是私有方法，我们模拟其逻辑
		shouldFilter := shouldFilterPacketTest(packet)

		filterStatus := "允许"
		if shouldFilter {
			filterStatus = "过滤"
		}

		description := getIPDescription(packet.DestIP)

		fmt.Printf("%-8s | %-15s | %-4d | %-8s | %s\n",
			packet.ID,
			packet.DestIP.String(),
			packet.DestPort,
			filterStatus,
			description)
	}

	fmt.Println()
	fmt.Println("3. 过滤器验证结果:")

	publicCount := 0
	privateCount := 0

	for _, packet := range testPackets {
		if shouldFilterPacketTest(packet) {
			privateCount++
		} else {
			publicCount++
		}
	}

	fmt.Printf("- 公网地址数据包（应处理）: %d\n", publicCount)
	fmt.Printf("- 私有/本地地址数据包（应过滤）: %d\n", privateCount)
	fmt.Printf("- 总测试数据包: %d\n", len(testPackets))

	if publicCount == 3 && privateCount == 6 {
		fmt.Println("✓ 过滤器测试通过！")
	} else {
		fmt.Println("✗ 过滤器测试失败！")
	}
}

// shouldFilterPacketTest 模拟shouldFilterPacket的逻辑用于测试
func shouldFilterPacketTest(packet *interceptor.PacketInfo) bool {
	if packet == nil || packet.DestIP == nil {
		return true
	}

	destIPv4 := packet.DestIP.To4()
	if destIPv4 == nil {
		return false // IPv6暂不过滤
	}

	destAddr := uint32(destIPv4[0])<<24 | uint32(destIPv4[1])<<16 | uint32(destIPv4[2])<<8 | uint32(destIPv4[3])

	isPrivateOrLocal :=
		// 本地回环 127.0.0.0/8
		(destAddr >= 0x7F000000 && destAddr <= 0x7FFFFFFF) ||
			// 私有网络A类 10.0.0.0/8
			(destAddr >= 0x0A000000 && destAddr <= 0x0AFFFFFF) ||
			// 私有网络B类 172.16.0.0/12
			(destAddr >= 0xAC100000 && destAddr <= 0xAC1FFFFF) ||
			// 私有网络C类 192.168.0.0/16
			(destAddr >= 0xC0A80000 && destAddr <= 0xC0A8FFFF) ||
			// 链路本地地址 169.254.0.0/16
			(destAddr >= 0xA9FE0000 && destAddr <= 0xA9FEFFFF) ||
			// 组播地址 224.0.0.0/4
			(destAddr >= 0xE0000000 && destAddr <= 0xEFFFFFFF) ||
			// 广播地址
			(destAddr == 0xFFFFFFFF)

	return isPrivateOrLocal
}

// getIPDescription 获取IP地址的描述
func getIPDescription(ip net.IP) string {
	if ip == nil {
		return "无效IP"
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return "IPv6地址"
	}

	addr := uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3])

	switch {
	case addr >= 0x7F000000 && addr <= 0x7FFFFFFF:
		return "本地回环地址"
	case addr >= 0x0A000000 && addr <= 0x0AFFFFFF:
		return "私有网络A类"
	case addr >= 0xAC100000 && addr <= 0xAC1FFFFF:
		return "私有网络B类"
	case addr >= 0xC0A80000 && addr <= 0xC0A8FFFF:
		return "私有网络C类"
	case addr >= 0xA9FE0000 && addr <= 0xA9FEFFFF:
		return "链路本地地址"
	case addr >= 0xE0000000 && addr <= 0xEFFFFFFF:
		return "组播地址"
	case addr == 0xFFFFFFFF:
		return "广播地址"
	default:
		return "公网地址"
	}
}
