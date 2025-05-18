package comm

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// generateID 生成唯一ID
func generateID() string {
	// 使用当前时间戳作为前缀
	timestamp := time.Now().UnixNano()
	prefix := fmt.Sprintf("%d", timestamp)

	// 生成8字节的随机数
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// 如果随机数生成失败，使用时间戳作为ID
		return prefix
	}

	// 将随机数转换为十六进制字符串
	randomHex := hex.EncodeToString(randomBytes)

	// 组合时间戳和随机数
	return prefix + "-" + randomHex
}

// encodeMessage 将消息编码为JSON字符串
func encodeMessage(msg *Message) ([]byte, error) {
	// 将消息编码为JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("编码消息失败: %w", err)
	}

	return data, nil
}

// decodeMessage 将JSON字符串解码为消息
func decodeMessage(data []byte) (*Message, error) {
	// 解码JSON
	var msg Message
	err := json.Unmarshal(data, &msg)
	if err != nil {
		return nil, fmt.Errorf("解码消息失败: %w", err)
	}

	return &msg, nil
}

// createHeartbeatMessage 创建心跳消息
func createHeartbeatMessage() *Message {
	return NewMessage(MessageTypeHeartbeat, map[string]interface{}{
		"time": time.Now().UnixNano() / int64(time.Millisecond),
	})
}

// createConnectMessage 创建连接消息
func createConnectMessage(clientInfo map[string]interface{}) *Message {
	return NewMessage(MessageTypeConnect, clientInfo)
}

// createAckMessage 创建确认消息
func createAckMessage(messageID string) *Message {
	return NewMessage(MessageTypeAck, map[string]interface{}{
		"message_id": messageID,
		"time":       time.Now().UnixNano() / int64(time.Millisecond),
	})
}
