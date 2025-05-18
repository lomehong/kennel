package comm

import (
	"bytes"
	"testing"
)

// TestCompressDecompress 测试压缩和解压缩
func TestCompressDecompress(t *testing.T) {
	// 创建客户端
	client := &Client{
		config: ConnectionConfig{
			Security: SecurityConfig{
				EnableCompression:    true,
				CompressionLevel:     6,
				CompressionThreshold: 10, // 设置较小的阈值，确保测试数据会被压缩
			},
		},
	}

	// 测试数据
	testData := []byte("这是一条测试消息，它应该足够长以便被压缩。这是一条测试消息，它应该足够长以便被压缩。")

	// 压缩数据
	compressedData, err := client.compressData(testData)
	if err != nil {
		t.Fatalf("压缩数据失败: %v", err)
	}

	// 检查压缩标记
	if compressedData[0] != 1 {
		t.Errorf("压缩标记应该是1，但是 %d", compressedData[0])
	}

	// 检查压缩后的数据小于原始数据
	if len(compressedData) >= len(testData)+1 {
		t.Errorf("压缩后的数据应该小于原始数据: %d >= %d", len(compressedData), len(testData)+1)
	}

	// 解压缩数据
	decompressedData, err := client.decompressData(compressedData)
	if err != nil {
		t.Fatalf("解压缩数据失败: %v", err)
	}

	// 检查解压缩后的数据等于原始数据
	if !bytes.Equal(decompressedData, testData) {
		t.Errorf("解压缩后的数据不等于原始数据: %s != %s", decompressedData, testData)
	}
}

// TestCompressDecompressSmallData 测试压缩和解压缩小数据
func TestCompressDecompressSmallData(t *testing.T) {
	// 创建客户端
	client := &Client{
		config: ConnectionConfig{
			Security: SecurityConfig{
				EnableCompression:    true,
				CompressionLevel:     6,
				CompressionThreshold: 100, // 设置较大的阈值，确保测试数据不会被压缩
			},
		},
	}

	// 测试数据
	testData := []byte("小数据")

	// 压缩数据
	compressedData, err := client.compressData(testData)
	if err != nil {
		t.Fatalf("压缩数据失败: %v", err)
	}

	// 检查压缩标记
	if compressedData[0] != 0 {
		t.Errorf("压缩标记应该是0，但是 %d", compressedData[0])
	}

	// 检查数据没有被压缩（除了添加的压缩标记）
	if len(compressedData) != len(testData)+1 {
		t.Errorf("小数据不应该被压缩: %d != %d", len(compressedData), len(testData)+1)
	}

	// 解压缩数据
	decompressedData, err := client.decompressData(compressedData)
	if err != nil {
		t.Fatalf("解压缩数据失败: %v", err)
	}

	// 检查解压缩后的数据等于原始数据
	if !bytes.Equal(decompressedData, testData) {
		t.Errorf("解压缩后的数据不等于原始数据: %s != %s", decompressedData, testData)
	}
}

// TestCompressDecompressDisabled 测试禁用压缩
func TestCompressDecompressDisabled(t *testing.T) {
	// 创建客户端
	client := &Client{
		config: ConnectionConfig{
			Security: SecurityConfig{
				EnableCompression: false,
			},
		},
	}

	// 测试数据
	testData := []byte("这是一条测试消息，它应该足够长以便被压缩。")

	// 压缩数据
	compressedData, err := client.compressData(testData)
	if err != nil {
		t.Fatalf("压缩数据失败: %v", err)
	}

	// 检查数据没有被压缩
	if !bytes.Equal(compressedData, testData) {
		t.Errorf("禁用压缩时，数据不应该被压缩: %v != %v", compressedData, testData)
	}

	// 解压缩数据
	decompressedData, err := client.decompressData(compressedData)
	if err != nil {
		t.Fatalf("解压缩数据失败: %v", err)
	}

	// 检查解压缩后的数据等于原始数据
	if !bytes.Equal(decompressedData, testData) {
		t.Errorf("解压缩后的数据不等于原始数据: %s != %s", decompressedData, testData)
	}
}
