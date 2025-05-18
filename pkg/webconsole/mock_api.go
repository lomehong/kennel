package webconsole

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 注册模拟API路由
func (c *Console) registerMockAPIRoutes(router *gin.Engine) {
	// 创建模拟API组
	mockAPI := router.Group("/mock-api")
	{
		// 通讯管理API
		comm := mockAPI.Group("/comm")
		{
			comm.GET("/status", c.mockGetCommStatus)
			comm.POST("/connect", c.mockConnectComm)
			comm.POST("/disconnect", c.mockDisconnectComm)
			comm.GET("/config", c.mockGetCommConfig)
			comm.GET("/stats", c.mockGetCommStats)
			comm.GET("/logs", c.mockGetCommLogs)

			// 通讯测试API
			commTest := comm.Group("/test")
			{
				commTest.POST("/connection", c.mockTestCommConnection)
				commTest.POST("/send-receive", c.mockTestCommSendReceive)
				commTest.POST("/encryption", c.mockTestCommEncryption)
				commTest.POST("/compression", c.mockTestCommCompression)
				commTest.POST("/performance", c.mockTestCommPerformance)
				commTest.GET("/history", c.mockGetCommTestHistory)
			}
		}
	}
}

// 模拟获取通讯状态
func (c *Console) mockGetCommStatus(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"connected": true,
		"status":    "已连接",
		"server":    "ws://mock-server:9000/ws",
		"uptime":    "1h 30m 15s",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// 模拟连接通讯
func (c *Console) mockConnectComm(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "连接成功",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// 模拟断开通讯
func (c *Console) mockDisconnectComm(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "断开连接成功",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// 模拟获取通讯配置
func (c *Console) mockGetCommConfig(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"config": map[string]interface{}{
			"server_url":          "ws://mock-server:9000/ws",
			"heartbeat_interval":  "30s",
			"reconnect_interval":  "5s",
			"max_reconnect_attempts": 10,
			"security": map[string]interface{}{
				"enable_tls":         false,
				"enable_encryption":  true,
				"enable_compression": true,
				"compression_level":  6,
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// 模拟获取通讯统计信息
func (c *Console) mockGetCommStats(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"messages_sent":     1024,
		"messages_received": 896,
		"bytes_sent":        102400,
		"bytes_received":    89600,
		"errors":            5,
		"reconnects":        2,
		"uptime":            "1h 30m 15s",
		"timestamp":         time.Now().Format(time.RFC3339),
	})
}

// 模拟获取通讯日志
func (c *Console) mockGetCommLogs(ctx *gin.Context) {
	logs := make([]map[string]interface{}, 0)
	for i := 0; i < 10; i++ {
		logs = append(logs, map[string]interface{}{
			"timestamp": time.Now().Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			"level":     "info",
			"message":   "模拟通讯日志 " + time.Now().Add(-time.Duration(i)*time.Minute).Format("15:04:05"),
			"source":    "comm-manager",
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"limit":  10,
		"offset": 0,
		"level":  "",
		"total":  10,
	})
}

// 模拟测试通讯连接
func (c *Console) mockTestCommConnection(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "连接测试成功",
		"duration":  "0.567s",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// 模拟测试通讯发送和接收
func (c *Console) mockTestCommSendReceive(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "发送消息成功",
		"duration": "0.789s",
		"response": map[string]interface{}{
			"request_id": "req-" + time.Now().Format("20060102150405"),
			"success":    true,
			"data":       "模拟响应数据",
			"timestamp":  time.Now().Format(time.RFC3339),
		},
	})
}

// 模拟测试通讯加密
func (c *Console) mockTestCommEncryption(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"success":          true,
		"message":          "加密测试成功",
		"duration":         "0.345s",
		"data_size":        1024,
		"encrypted_size":   1040,
		"decrypted_size":   1024,
		"encryption_ratio": 1.015625,
		"timestamp":        time.Now().Format(time.RFC3339),
	})
}

// 模拟测试通讯压缩
func (c *Console) mockTestCommCompression(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"success":           true,
		"message":           "压缩测试成功",
		"duration":          "0.456s",
		"data_size":         1024,
		"compressed_size":   512,
		"decompressed_size": 1024,
		"compression_ratio": 0.5,
		"timestamp":         time.Now().Format(time.RFC3339),
	})
}

// 模拟测试通讯性能
func (c *Console) mockTestCommPerformance(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "性能测试成功",
		"duration": "2.345s",
		"result": map[string]interface{}{
			"message_count":       1000,
			"message_size":        1024,
			"send_duration":       "1.234s",
			"send_throughput":     810.3728,
			"send_size":           1024000,
			"send_compressed_size": 512000,
			"compression_ratio":   0.5,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// 模拟获取通讯测试历史记录
func (c *Console) mockGetCommTestHistory(ctx *gin.Context) {
	history := []map[string]interface{}{
		{
			"type":         "connection",
			"timestamp":    time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			"server_url":   "ws://mock-server:9000/ws",
			"timeout":      "10s",
			"success":      true,
			"duration":     "0.567s",
		},
		{
			"type":         "send-receive",
			"timestamp":    time.Now().Add(-25 * time.Minute).Format(time.RFC3339),
			"message_type": "command",
			"payload": map[string]interface{}{
				"command":    "ping",
				"request_id": "req-20230517153112",
			},
			"response": map[string]interface{}{
				"request_id": "req-20230517153112",
				"success":    true,
				"data":       "pong",
			},
			"success":      true,
			"duration":     "0.789s",
		},
		{
			"type":            "encryption",
			"timestamp":       time.Now().Add(-20 * time.Minute).Format(time.RFC3339),
			"data_size":       1024,
			"encryption_key":  "test-key",
			"encrypted_size":  1040,
			"decrypted_size":  1024,
			"success":         true,
			"duration":        "0.345s",
			"encryption_ratio": 1.015625,
		},
		{
			"type":              "compression",
			"timestamp":         time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
			"data_size":         1024,
			"compression_level": 6,
			"compressed_size":   512,
			"decompressed_size": 1024,
			"success":           true,
			"duration":          "0.456s",
			"compression_ratio": 0.5,
		},
		{
			"type":               "performance",
			"timestamp":          time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
			"message_count":      1000,
			"message_size":       1024,
			"enable_encryption":  true,
			"enable_compression": true,
			"compression_level":  6,
			"success":            true,
			"duration":           "2.345s",
			"result": map[string]interface{}{
				"send_duration":       "1.234s",
				"send_throughput":     810.3728,
				"send_size":           1024000,
				"send_compressed_size": 512000,
				"compression_ratio":   0.5,
			},
		},
	}

	ctx.JSON(http.StatusOK, history)
}
