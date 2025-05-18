package comm

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// compressData 压缩数据
func (c *Client) compressData(data []byte) ([]byte, error) {
	// 如果未启用压缩，直接返回原始数据
	if !c.config.Security.EnableCompression {
		return data, nil
	}

	// 如果数据大小小于压缩阈值，直接返回原始数据，添加未压缩标记
	if len(data) < c.config.Security.CompressionThreshold {
		// 添加未压缩标记
		result := make([]byte, len(data)+1)
		result[0] = 0 // 未压缩标记
		copy(result[1:], data)
		return result, nil
	}

	// 创建一个缓冲区
	var buf bytes.Buffer

	// 创建一个gzip写入器
	gzipWriter, err := gzip.NewWriterLevel(&buf, c.config.Security.CompressionLevel)
	if err != nil {
		return nil, fmt.Errorf("创建gzip写入器失败: %w", err)
	}

	// 写入数据
	_, err = gzipWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("写入数据失败: %w", err)
	}

	// 关闭写入器
	err = gzipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("关闭gzip写入器失败: %w", err)
	}

	// 获取压缩后的数据
	compressedData := buf.Bytes()

	// 如果压缩后的数据大于原始数据，返回原始数据
	if len(compressedData) >= len(data) {
		return data, nil
	}

	// 添加压缩标记
	// 格式：[1字节压缩标记][压缩后的数据]
	// 压缩标记：1表示已压缩，0表示未压缩
	result := make([]byte, len(compressedData)+1)
	result[0] = 1 // 压缩标记
	copy(result[1:], compressedData)

	return result, nil
}

// decompressData 解压缩数据
func (c *Client) decompressData(data []byte) ([]byte, error) {
	// 如果未启用压缩，直接返回原始数据
	if !c.config.Security.EnableCompression {
		return data, nil
	}

	// 检查数据长度
	if len(data) < 1 {
		return nil, fmt.Errorf("数据太短")
	}

	// 检查压缩标记
	compressionFlag := data[0]
	if compressionFlag == 0 {
		// 未压缩，返回原始数据（去掉压缩标记）
		return data[1:], nil
	}

	// 创建一个gzip读取器
	gzipReader, err := gzip.NewReader(bytes.NewReader(data[1:]))
	if err != nil {
		return nil, fmt.Errorf("创建gzip读取器失败: %w", err)
	}
	defer gzipReader.Close()

	// 读取解压缩后的数据
	var buf bytes.Buffer
	_, err = io.Copy(&buf, gzipReader)
	if err != nil {
		return nil, fmt.Errorf("读取解压缩数据失败: %w", err)
	}

	return buf.Bytes(), nil
}

// CompressData 压缩数据（公共方法）
func CompressData(data []byte, compressionLevel int) ([]byte, error) {
	// 创建一个缓冲区
	var buf bytes.Buffer

	// 创建一个gzip写入器
	gzipWriter, err := gzip.NewWriterLevel(&buf, compressionLevel)
	if err != nil {
		return nil, fmt.Errorf("创建gzip写入器失败: %w", err)
	}

	// 写入数据
	_, err = gzipWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("写入数据失败: %w", err)
	}

	// 关闭写入器
	err = gzipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("关闭gzip写入器失败: %w", err)
	}

	// 获取压缩后的数据
	compressedData := buf.Bytes()

	// 如果压缩后的数据大于原始数据，返回原始数据
	if len(compressedData) >= len(data) {
		return data, nil
	}

	// 添加压缩标记
	result := make([]byte, len(compressedData)+1)
	result[0] = 1 // 压缩标记
	copy(result[1:], compressedData)

	return result, nil
}

// DecompressData 解压缩数据（公共方法）
func DecompressData(data []byte) ([]byte, error) {
	// 检查数据长度
	if len(data) < 1 {
		return nil, fmt.Errorf("数据太短")
	}

	// 检查压缩标记
	compressionFlag := data[0]
	if compressionFlag == 0 {
		// 未压缩，返回原始数据（去掉压缩标记）
		return data[1:], nil
	}

	// 创建一个gzip读取器
	gzipReader, err := gzip.NewReader(bytes.NewReader(data[1:]))
	if err != nil {
		return nil, fmt.Errorf("创建gzip读取器失败: %w", err)
	}
	defer gzipReader.Close()

	// 读取解压缩后的数据
	var buf bytes.Buffer
	_, err = io.Copy(&buf, gzipReader)
	if err != nil {
		return nil, fmt.Errorf("读取解压缩数据失败: %w", err)
	}

	return buf.Bytes(), nil
}
