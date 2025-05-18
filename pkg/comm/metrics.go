package comm

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector 收集通讯模块的指标
type MetricsCollector struct {
	// 连接指标
	connectCount       uint64 // 连接次数
	connectFailCount   uint64 // 连接失败次数
	disconnectCount    uint64 // 断开连接次数
	reconnectCount     uint64 // 重连次数
	lastConnectTime    int64  // 最后一次连接时间（Unix时间戳，毫秒）
	lastDisconnectTime int64  // 最后一次断开连接时间（Unix时间戳，毫秒）
	connectionDuration int64  // 连接持续时间（毫秒）

	// 消息指标
	sentMessageCount     uint64 // 发送消息数量
	receivedMessageCount uint64 // 接收消息数量
	sentBytes            uint64 // 发送字节数
	receivedBytes        uint64 // 接收字节数
	messageErrorCount    uint64 // 消息错误数量

	// 延迟指标
	totalLatency int64  // 总延迟（毫秒）
	latencyCount uint64 // 延迟计数
	maxLatency   int64  // 最大延迟（毫秒）
	minLatency   int64  // 最小延迟（毫秒）

	// 压缩指标
	compressedCount      uint64 // 压缩消息数量
	compressedBytes      uint64 // 压缩前字节数
	compressedBytesAfter uint64 // 压缩后字节数

	// 加密指标
	encryptedCount      uint64 // 加密消息数量
	encryptedBytes      uint64 // 加密前字节数
	encryptedBytesAfter uint64 // 加密后字节数

	// 心跳指标
	heartbeatSentCount     uint64 // 发送心跳数量
	heartbeatReceivedCount uint64 // 接收心跳数量
	heartbeatErrorCount    uint64 // 心跳错误数量
	lastHeartbeatTime      int64  // 最后一次心跳时间（Unix时间戳，毫秒）

	// 错误指标
	errorCount       uint64       // 错误数量
	lastErrorTime    int64        // 最后一次错误时间（Unix时间戳，毫秒）
	lastErrorMessage string       // 最后一次错误消息
	lastErrorMutex   sync.RWMutex // 保护lastErrorMessage的互斥锁

	// 状态指标
	currentState ConnectionState // 当前连接状态
	stateMutex   sync.RWMutex    // 保护currentState的互斥锁
}

// NewMetricsCollector 创建一个新的指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		minLatency: -1, // 初始化为-1，表示尚未设置
	}
}

// Reset 重置所有指标
func (mc *MetricsCollector) Reset() {
	atomic.StoreUint64(&mc.connectCount, 0)
	atomic.StoreUint64(&mc.connectFailCount, 0)
	atomic.StoreUint64(&mc.disconnectCount, 0)
	atomic.StoreUint64(&mc.reconnectCount, 0)
	atomic.StoreInt64(&mc.lastConnectTime, 0)
	atomic.StoreInt64(&mc.lastDisconnectTime, 0)
	atomic.StoreInt64(&mc.connectionDuration, 0)

	atomic.StoreUint64(&mc.sentMessageCount, 0)
	atomic.StoreUint64(&mc.receivedMessageCount, 0)
	atomic.StoreUint64(&mc.sentBytes, 0)
	atomic.StoreUint64(&mc.receivedBytes, 0)
	atomic.StoreUint64(&mc.messageErrorCount, 0)

	atomic.StoreInt64(&mc.totalLatency, 0)
	atomic.StoreUint64(&mc.latencyCount, 0)
	atomic.StoreInt64(&mc.maxLatency, 0)
	atomic.StoreInt64(&mc.minLatency, -1)

	atomic.StoreUint64(&mc.compressedCount, 0)
	atomic.StoreUint64(&mc.compressedBytes, 0)
	atomic.StoreUint64(&mc.compressedBytesAfter, 0)

	atomic.StoreUint64(&mc.encryptedCount, 0)
	atomic.StoreUint64(&mc.encryptedBytes, 0)
	atomic.StoreUint64(&mc.encryptedBytesAfter, 0)

	atomic.StoreUint64(&mc.heartbeatSentCount, 0)
	atomic.StoreUint64(&mc.heartbeatReceivedCount, 0)
	atomic.StoreUint64(&mc.heartbeatErrorCount, 0)
	atomic.StoreInt64(&mc.lastHeartbeatTime, 0)

	atomic.StoreUint64(&mc.errorCount, 0)
	atomic.StoreInt64(&mc.lastErrorTime, 0)

	mc.lastErrorMutex.Lock()
	mc.lastErrorMessage = ""
	mc.lastErrorMutex.Unlock()

	mc.stateMutex.Lock()
	mc.currentState = StateDisconnected
	mc.stateMutex.Unlock()
}

// RecordConnect 记录连接事件
func (mc *MetricsCollector) RecordConnect(success bool) {
	if success {
		atomic.AddUint64(&mc.connectCount, 1)
		atomic.StoreInt64(&mc.lastConnectTime, time.Now().UnixNano()/int64(time.Millisecond))
	} else {
		atomic.AddUint64(&mc.connectFailCount, 1)
	}
}

// RecordDisconnect 记录断开连接事件
func (mc *MetricsCollector) RecordDisconnect() {
	atomic.AddUint64(&mc.disconnectCount, 1)
	now := time.Now().UnixNano() / int64(time.Millisecond)
	atomic.StoreInt64(&mc.lastDisconnectTime, now)

	// 计算连接持续时间
	lastConnect := atomic.LoadInt64(&mc.lastConnectTime)
	if lastConnect > 0 {
		duration := now - lastConnect
		atomic.StoreInt64(&mc.connectionDuration, duration)
	}
}

// RecordReconnect 记录重连事件
func (mc *MetricsCollector) RecordReconnect() {
	atomic.AddUint64(&mc.reconnectCount, 1)
}

// RecordSentMessage 记录发送消息事件
func (mc *MetricsCollector) RecordSentMessage(bytes int) {
	atomic.AddUint64(&mc.sentMessageCount, 1)
	atomic.AddUint64(&mc.sentBytes, uint64(bytes))
}

// RecordReceivedMessage 记录接收消息事件
func (mc *MetricsCollector) RecordReceivedMessage(bytes int) {
	atomic.AddUint64(&mc.receivedMessageCount, 1)
	atomic.AddUint64(&mc.receivedBytes, uint64(bytes))
}

// RecordMessageError 记录消息错误事件
func (mc *MetricsCollector) RecordMessageError() {
	atomic.AddUint64(&mc.messageErrorCount, 1)
}

// RecordLatency 记录延迟
func (mc *MetricsCollector) RecordLatency(latency int64) {
	atomic.AddInt64(&mc.totalLatency, latency)
	atomic.AddUint64(&mc.latencyCount, 1)

	// 更新最大延迟
	for {
		currentMax := atomic.LoadInt64(&mc.maxLatency)
		if latency <= currentMax {
			break
		}
		if atomic.CompareAndSwapInt64(&mc.maxLatency, currentMax, latency) {
			break
		}
	}

	// 更新最小延迟
	for {
		currentMin := atomic.LoadInt64(&mc.minLatency)
		if currentMin != -1 && latency >= currentMin {
			break
		}
		if atomic.CompareAndSwapInt64(&mc.minLatency, currentMin, latency) {
			break
		}
	}
}

// RecordCompression 记录压缩事件
func (mc *MetricsCollector) RecordCompression(before, after int) {
	atomic.AddUint64(&mc.compressedCount, 1)
	atomic.AddUint64(&mc.compressedBytes, uint64(before))
	atomic.AddUint64(&mc.compressedBytesAfter, uint64(after))
}

// RecordEncryption 记录加密事件
func (mc *MetricsCollector) RecordEncryption(before, after int) {
	atomic.AddUint64(&mc.encryptedCount, 1)
	atomic.AddUint64(&mc.encryptedBytes, uint64(before))
	atomic.AddUint64(&mc.encryptedBytesAfter, uint64(after))
}

// RecordHeartbeatSent 记录发送心跳事件
func (mc *MetricsCollector) RecordHeartbeatSent() {
	atomic.AddUint64(&mc.heartbeatSentCount, 1)
	atomic.StoreInt64(&mc.lastHeartbeatTime, time.Now().UnixNano()/int64(time.Millisecond))
}

// RecordHeartbeatReceived 记录接收心跳事件
func (mc *MetricsCollector) RecordHeartbeatReceived() {
	atomic.AddUint64(&mc.heartbeatReceivedCount, 1)
	atomic.StoreInt64(&mc.lastHeartbeatTime, time.Now().UnixNano()/int64(time.Millisecond))
}

// RecordHeartbeatError 记录心跳错误事件
func (mc *MetricsCollector) RecordHeartbeatError() {
	atomic.AddUint64(&mc.heartbeatErrorCount, 1)
}

// RecordError 记录错误事件
func (mc *MetricsCollector) RecordError(message string) {
	atomic.AddUint64(&mc.errorCount, 1)
	atomic.StoreInt64(&mc.lastErrorTime, time.Now().UnixNano()/int64(time.Millisecond))

	mc.lastErrorMutex.Lock()
	mc.lastErrorMessage = message
	mc.lastErrorMutex.Unlock()
}

// RecordState 记录状态变化
func (mc *MetricsCollector) RecordState(state ConnectionState) {
	mc.stateMutex.Lock()
	mc.currentState = state
	mc.stateMutex.Unlock()
}

// GetMetrics 获取所有指标
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// 连接指标
	metrics["connect_count"] = atomic.LoadUint64(&mc.connectCount)
	metrics["connect_fail_count"] = atomic.LoadUint64(&mc.connectFailCount)
	metrics["disconnect_count"] = atomic.LoadUint64(&mc.disconnectCount)
	metrics["reconnect_count"] = atomic.LoadUint64(&mc.reconnectCount)
	metrics["last_connect_time"] = atomic.LoadInt64(&mc.lastConnectTime)
	metrics["last_disconnect_time"] = atomic.LoadInt64(&mc.lastDisconnectTime)
	metrics["connection_duration"] = atomic.LoadInt64(&mc.connectionDuration)

	// 消息指标
	metrics["sent_message_count"] = atomic.LoadUint64(&mc.sentMessageCount)
	metrics["received_message_count"] = atomic.LoadUint64(&mc.receivedMessageCount)
	metrics["sent_bytes"] = atomic.LoadUint64(&mc.sentBytes)
	metrics["received_bytes"] = atomic.LoadUint64(&mc.receivedBytes)
	metrics["message_error_count"] = atomic.LoadUint64(&mc.messageErrorCount)

	// 延迟指标
	metrics["total_latency"] = atomic.LoadInt64(&mc.totalLatency)
	latencyCount := atomic.LoadUint64(&mc.latencyCount)
	metrics["latency_count"] = latencyCount
	metrics["max_latency"] = atomic.LoadInt64(&mc.maxLatency)
	minLatency := atomic.LoadInt64(&mc.minLatency)
	metrics["min_latency"] = minLatency

	// 计算平均延迟
	if latencyCount > 0 {
		metrics["avg_latency"] = float64(atomic.LoadInt64(&mc.totalLatency)) / float64(latencyCount)
	} else {
		metrics["avg_latency"] = 0.0
	}

	// 压缩指标
	metrics["compressed_count"] = atomic.LoadUint64(&mc.compressedCount)
	compressedBytes := atomic.LoadUint64(&mc.compressedBytes)
	compressedBytesAfter := atomic.LoadUint64(&mc.compressedBytesAfter)
	metrics["compressed_bytes"] = compressedBytes
	metrics["compressed_bytes_after"] = compressedBytesAfter

	// 计算压缩率
	if compressedBytes > 0 {
		metrics["compression_ratio"] = float64(compressedBytes-compressedBytesAfter) / float64(compressedBytes)
	} else {
		metrics["compression_ratio"] = 0.0
	}

	// 加密指标
	metrics["encrypted_count"] = atomic.LoadUint64(&mc.encryptedCount)
	metrics["encrypted_bytes"] = atomic.LoadUint64(&mc.encryptedBytes)
	metrics["encrypted_bytes_after"] = atomic.LoadUint64(&mc.encryptedBytesAfter)

	// 心跳指标
	metrics["heartbeat_sent_count"] = atomic.LoadUint64(&mc.heartbeatSentCount)
	metrics["heartbeat_received_count"] = atomic.LoadUint64(&mc.heartbeatReceivedCount)
	metrics["heartbeat_error_count"] = atomic.LoadUint64(&mc.heartbeatErrorCount)
	metrics["last_heartbeat_time"] = atomic.LoadInt64(&mc.lastHeartbeatTime)

	// 错误指标
	metrics["error_count"] = atomic.LoadUint64(&mc.errorCount)
	metrics["last_error_time"] = atomic.LoadInt64(&mc.lastErrorTime)

	mc.lastErrorMutex.RLock()
	metrics["last_error_message"] = mc.lastErrorMessage
	mc.lastErrorMutex.RUnlock()

	// 状态指标
	mc.stateMutex.RLock()
	metrics["current_state"] = mc.currentState.String()
	mc.stateMutex.RUnlock()

	return metrics
}

// GetMetricsReport 获取指标报告
func (mc *MetricsCollector) GetMetricsReport() string {
	metrics := mc.GetMetrics()

	// 格式化报告
	report := "通讯模块指标报告:\n"
	report += "==================\n"

	// 连接状态
	report += "连接状态: " + metrics["current_state"].(string) + "\n"

	// 连接指标
	report += "\n连接指标:\n"
	report += "  连接次数: " + formatUint64(metrics["connect_count"].(uint64)) + "\n"
	report += "  连接失败次数: " + formatUint64(metrics["connect_fail_count"].(uint64)) + "\n"
	report += "  断开连接次数: " + formatUint64(metrics["disconnect_count"].(uint64)) + "\n"
	report += "  重连次数: " + formatUint64(metrics["reconnect_count"].(uint64)) + "\n"

	lastConnectTime := metrics["last_connect_time"].(int64)
	if lastConnectTime > 0 {
		report += "  最后连接时间: " + formatTime(lastConnectTime) + "\n"
	}

	lastDisconnectTime := metrics["last_disconnect_time"].(int64)
	if lastDisconnectTime > 0 {
		report += "  最后断开时间: " + formatTime(lastDisconnectTime) + "\n"
	}

	connectionDuration := metrics["connection_duration"].(int64)
	if connectionDuration > 0 {
		report += "  连接持续时间: " + formatDuration(connectionDuration) + "\n"
	}

	// 消息指标
	report += "\n消息指标:\n"
	report += "  发送消息数: " + formatUint64(metrics["sent_message_count"].(uint64)) + "\n"
	report += "  接收消息数: " + formatUint64(metrics["received_message_count"].(uint64)) + "\n"
	report += "  发送字节数: " + formatBytes(metrics["sent_bytes"].(uint64)) + "\n"
	report += "  接收字节数: " + formatBytes(metrics["received_bytes"].(uint64)) + "\n"
	report += "  消息错误数: " + formatUint64(metrics["message_error_count"].(uint64)) + "\n"

	// 延迟指标
	report += "\n延迟指标:\n"
	if metrics["latency_count"].(uint64) > 0 {
		report += "  平均延迟: " + formatFloat(metrics["avg_latency"].(float64)) + " ms\n"
		report += "  最大延迟: " + formatInt64(metrics["max_latency"].(int64)) + " ms\n"

		minLatency := metrics["min_latency"].(int64)
		if minLatency >= 0 {
			report += "  最小延迟: " + formatInt64(minLatency) + " ms\n"
		}
	} else {
		report += "  尚无延迟数据\n"
	}

	// 压缩指标
	report += "\n压缩指标:\n"
	compressedCount := metrics["compressed_count"].(uint64)
	if compressedCount > 0 {
		report += "  压缩消息数: " + formatUint64(compressedCount) + "\n"
		report += "  压缩前字节数: " + formatBytes(metrics["compressed_bytes"].(uint64)) + "\n"
		report += "  压缩后字节数: " + formatBytes(metrics["compressed_bytes_after"].(uint64)) + "\n"
		report += "  压缩率: " + formatPercentage(metrics["compression_ratio"].(float64)) + "\n"
	} else {
		report += "  尚无压缩数据\n"
	}

	// 加密指标
	report += "\n加密指标:\n"
	encryptedCount := metrics["encrypted_count"].(uint64)
	if encryptedCount > 0 {
		report += "  加密消息数: " + formatUint64(encryptedCount) + "\n"
		report += "  加密前字节数: " + formatBytes(metrics["encrypted_bytes"].(uint64)) + "\n"
		report += "  加密后字节数: " + formatBytes(metrics["encrypted_bytes_after"].(uint64)) + "\n"
	} else {
		report += "  尚无加密数据\n"
	}

	// 心跳指标
	report += "\n心跳指标:\n"
	report += "  发送心跳数: " + formatUint64(metrics["heartbeat_sent_count"].(uint64)) + "\n"
	report += "  接收心跳数: " + formatUint64(metrics["heartbeat_received_count"].(uint64)) + "\n"
	report += "  心跳错误数: " + formatUint64(metrics["heartbeat_error_count"].(uint64)) + "\n"

	lastHeartbeatTime := metrics["last_heartbeat_time"].(int64)
	if lastHeartbeatTime > 0 {
		report += "  最后心跳时间: " + formatTime(lastHeartbeatTime) + "\n"
	}

	// 错误指标
	report += "\n错误指标:\n"
	errorCount := metrics["error_count"].(uint64)
	if errorCount > 0 {
		report += "  错误数: " + formatUint64(errorCount) + "\n"

		lastErrorTime := metrics["last_error_time"].(int64)
		if lastErrorTime > 0 {
			report += "  最后错误时间: " + formatTime(lastErrorTime) + "\n"
		}

		lastErrorMessage := metrics["last_error_message"].(string)
		if lastErrorMessage != "" {
			report += "  最后错误信息: " + lastErrorMessage + "\n"
		}
	} else {
		report += "  尚无错误\n"
	}

	return report
}
