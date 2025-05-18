package comm

import (
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// readPump 从WebSocket连接读取消息
func (c *Client) readPump() {
	defer func() {
		c.reconnect()
	}()

	// 设置读取超时
	c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
		return nil
	})

	for {
		// 检查是否需要停止
		select {
		case <-c.stopChan:
			return
		default:
			// 继续读取
		}

		// 读取消息
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.handleError(err)
			}
			return
		}

		// 记录接收字节数
		c.metrics.RecordReceivedMessage(len(data))

		// 重置读取超时
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))

		// 如果启用了加密，解密消息
		if c.config.Security.EnableEncryption {
			data, err = c.decryptMessage(data)
			if err != nil {
				c.handleError(fmt.Errorf("解密消息失败: %w", err))
				continue
			}
		}

		// 如果启用了压缩，解压缩消息
		if c.config.Security.EnableCompression {
			data, err = c.decompressData(data)
			if err != nil {
				c.handleError(fmt.Errorf("解压缩消息失败: %w", err))
				continue
			}
		}

		// 解析消息
		msg, err := decodeMessage(data)
		if err != nil {
			c.handleError(err)
			continue
		}

		// 处理系统消息
		if c.handleSystemMessage(msg) {
			continue
		}

		// 将消息放入接收队列
		select {
		case c.receiveChan <- msg:
			// 消息已加入接收队列
		default:
			c.logger.Warn("接收队列已满，消息被丢弃")
		}
	}
}

// writePump 向WebSocket连接写入消息
func (c *Client) writePump() {
	defer func() {
		c.reconnect()
	}()

	for {
		select {
		case <-c.stopChan:
			return
		case msg := <-c.sendChan:
			// 编码消息
			data, err := encodeMessage(msg)
			if err != nil {
				c.handleError(err)
				continue
			}

			// 记录发送字节数
			c.metrics.RecordSentMessage(len(data))

			// 如果启用了压缩，压缩消息
			if c.config.Security.EnableCompression {
				beforeSize := len(data)
				data, err = c.compressData(data)
				if err != nil {
					c.handleError(fmt.Errorf("压缩消息失败: %w", err))
					continue
				}
				// 记录压缩指标
				c.metrics.RecordCompression(beforeSize, len(data))
			}

			// 如果启用了加密，加密消息
			if c.config.Security.EnableEncryption {
				beforeSize := len(data)
				data, err = c.encryptMessage(data)
				if err != nil {
					c.handleError(fmt.Errorf("加密消息失败: %w", err))
					continue
				}
				// 记录加密指标
				c.metrics.RecordEncryption(beforeSize, len(data))
			}

			// 设置写入超时
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))

			// 写入消息
			err = c.conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				c.handleError(err)
				c.metrics.RecordMessageError()
				return
			}

			c.logger.Debug("消息已发送", "type", msg.Type, "id", msg.ID)
		}
	}
}

// processPump 处理接收到的消息
func (c *Client) processPump() {
	for {
		select {
		case <-c.stopChan:
			return
		case msg := <-c.receiveChan:
			// 调用消息处理函数
			if c.messageHandler != nil {
				go c.messageHandler(msg)
			}
			c.logger.Debug("消息已处理", "type", msg.Type, "id", msg.ID)
		}
	}
}

// handleSystemMessage 处理系统消息
func (c *Client) handleSystemMessage(msg *Message) bool {
	switch msg.Type {
	case MessageTypeHeartbeat:
		// 收到心跳消息，回复确认
		c.metrics.RecordHeartbeatReceived()
		c.Send(createAckMessage(msg.ID))
		return true
	case MessageTypeAck:
		// 收到确认消息，不需要特殊处理
		return true
	default:
		return false
	}
}

// startHeartbeat 启动心跳
func (c *Client) startHeartbeat() {
	// 停止现有的心跳定时器
	if c.heartbeatTimer != nil {
		c.heartbeatTimer.Stop()
	}

	// 创建新的心跳定时器
	c.heartbeatTimer = time.NewTimer(c.config.HeartbeatInterval)

	// 启动心跳协程
	go func() {
		for {
			select {
			case <-c.stopChan:
				return
			case <-c.heartbeatTimer.C:
				// 发送心跳消息
				c.Send(createHeartbeatMessage())
				c.metrics.RecordHeartbeatSent()
				// 重置定时器
				c.heartbeatTimer.Reset(c.config.HeartbeatInterval)
			}
		}
	}()
}

// reconnect 重新连接
func (c *Client) reconnect() {
	c.stateMutex.Lock()

	// 如果已经是断开连接状态，不需要重连
	if c.state == StateDisconnected {
		c.stateMutex.Unlock()
		return
	}

	// 如果已经达到最大重连次数，放弃重连
	if c.reconnectCount >= c.config.MaxReconnectAttempts {
		c.logger.Error("达到最大重连次数，放弃重连")
		c.setState(StateDisconnected)
		c.stateMutex.Unlock()
		return
	}

	// 设置重连状态
	c.setState(StateReconnecting)
	c.reconnectCount++
	c.metrics.RecordReconnect()
	c.stateMutex.Unlock()

	// 关闭现有连接
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// 等待重连间隔
	time.Sleep(c.config.ReconnectInterval)

	// 尝试重新连接
	c.logger.Info("尝试重新连接", "attempt", c.reconnectCount)
	err := c.Connect()
	if err != nil {
		c.handleError(errors.New("重连失败: " + err.Error()))
	}
}
