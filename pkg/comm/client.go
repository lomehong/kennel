package comm

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lomehong/kennel/pkg/logging"
)

// Client 定义WebSocket客户端
type Client struct {
	config         ConnectionConfig
	conn           *websocket.Conn
	state          ConnectionState
	stateMutex     sync.RWMutex
	reconnectCount int

	// 消息处理
	sendChan    chan *Message
	receiveChan chan *Message

	// 处理器
	messageHandler     MessageHandler
	stateChangeHandler ConnectionStateHandler
	errorHandler       ErrorHandler

	// 控制
	stopChan       chan struct{}
	heartbeatTimer *time.Timer

	// 日志
	logger logging.Logger

	// 客户端信息
	clientInfo map[string]interface{}

	// 指标收集器
	metrics *MetricsCollector
}

// NewClient 创建一个新的WebSocket客户端
func NewClient(config ConnectionConfig, log logging.Logger) *Client {
	if log == nil {
		// 创建默认日志配置
		logConfig := logging.DefaultLogConfig()
		logConfig.Level = logging.LogLevelInfo

		// 创建增强日志记录器
		enhancedLogger, err := logging.NewEnhancedLogger(logConfig)
		if err != nil {
			// 如果创建失败，使用默认配置
			enhancedLogger, _ = logging.NewEnhancedLogger(nil)
		}

		// 设置名称
		log = enhancedLogger.Named("comm-client")
	}

	return &Client{
		config:      config,
		state:       StateDisconnected,
		sendChan:    make(chan *Message, config.MessageBufferSize),
		receiveChan: make(chan *Message, config.MessageBufferSize),
		stopChan:    make(chan struct{}),
		logger:      log,
		clientInfo:  make(map[string]interface{}),
		metrics:     NewMetricsCollector(),
	}
}

// Connect 连接到服务器
func (c *Client) Connect() error {
	c.stateMutex.Lock()
	if c.state == StateConnecting || c.state == StateConnected {
		c.stateMutex.Unlock()
		return errors.New("已经连接或正在连接中")
	}

	c.setState(StateConnecting)
	c.stateMutex.Unlock()

	// 设置连接超时和TLS配置
	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.HandshakeTimeout,
	}

	// 如果启用了TLS，设置TLS配置
	if c.config.Security.EnableTLS {
		tlsConfig, err := c.createTLSConfig()
		if err != nil {
			c.setState(StateDisconnected)
			c.logger.Error("创建TLS配置失败", "error", err)
			return err
		}
		dialer.TLSClientConfig = tlsConfig
	}

	// 准备HTTP头
	header := http.Header{}

	// 如果启用了认证，添加认证头
	if c.config.Security.EnableAuth {
		authHeader, err := c.createAuthHeader()
		if err != nil {
			c.setState(StateDisconnected)
			c.logger.Error("创建认证头失败", "error", err)
			return err
		}
		for k, v := range authHeader {
			header.Set(k, v)
		}
	}

	// 连接到服务器
	conn, _, err := dialer.Dial(c.config.ServerURL, header)
	if err != nil {
		c.setState(StateDisconnected)
		c.logger.Error("连接服务器失败", "error", err)
		c.metrics.RecordConnect(false)
		return err
	}

	c.conn = conn
	c.setState(StateConnected)
	c.reconnectCount = 0
	c.metrics.RecordConnect(true)

	// 启动处理协程
	go c.readPump()
	go c.writePump()
	go c.processPump()

	// 发送连接消息
	c.Send(createConnectMessage(c.clientInfo))

	// 启动心跳
	c.startHeartbeat()

	c.logger.Info("已连接到服务器", "url", c.config.ServerURL)
	return nil
}

// Disconnect 断开连接
func (c *Client) Disconnect() {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	if c.state == StateDisconnected {
		return
	}

	c.logger.Info("正在断开连接...")

	// 停止所有协程
	close(c.stopChan)

	// 停止心跳
	if c.heartbeatTimer != nil {
		c.heartbeatTimer.Stop()
	}

	// 发送关闭消息
	if c.conn != nil {
		// 创建关闭消息
		closeMsg := NewMessage(MessageTypeEvent, map[string]interface{}{
			"event": "client_disconnect",
			"details": map[string]interface{}{
				"reason": "graceful_shutdown",
				"time":   time.Now().Format(time.RFC3339),
			},
		})

		// 编码消息
		data, err := encodeMessage(closeMsg)
		if err == nil {
			// 设置写入超时
			c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))

			// 写入消息
			c.conn.WriteMessage(websocket.TextMessage, data)

			// 等待一小段时间，确保消息发送出去
			time.Sleep(100 * time.Millisecond)
		}

		// 关闭连接
		c.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "客户端正常关闭"))
		c.conn.Close()
		c.conn = nil
	}

	c.setState(StateDisconnected)
	c.logger.Info("已断开连接")
	c.metrics.RecordDisconnect()

	// 重新初始化通道
	c.stopChan = make(chan struct{})
	c.sendChan = make(chan *Message, c.config.MessageBufferSize)
	c.receiveChan = make(chan *Message, c.config.MessageBufferSize)
}

// Send 发送消息
func (c *Client) Send(msg *Message) {
	select {
	case c.sendChan <- msg:
		// 消息已加入发送队列
	default:
		c.logger.Warn("发送队列已满，消息被丢弃")
	}
}

// SetMessageHandler 设置消息处理函数
func (c *Client) SetMessageHandler(handler MessageHandler) {
	c.messageHandler = handler
}

// SetStateChangeHandler 设置状态变化处理函数
func (c *Client) SetStateChangeHandler(handler ConnectionStateHandler) {
	c.stateChangeHandler = handler
}

// SetErrorHandler 设置错误处理函数
func (c *Client) SetErrorHandler(handler ErrorHandler) {
	c.errorHandler = handler
}

// SetClientInfo 设置客户端信息
func (c *Client) SetClientInfo(info map[string]interface{}) {
	c.clientInfo = info
}

// GetState 获取当前连接状态
func (c *Client) GetState() ConnectionState {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	return c.state
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	return c.GetState() == StateConnected
}

// setState 设置连接状态
func (c *Client) setState(newState ConnectionState) {
	oldState := c.state
	c.state = newState

	// 调用状态变化处理函数
	if c.stateChangeHandler != nil && oldState != newState {
		go c.stateChangeHandler(oldState, newState)
	}
}

// handleError 处理错误
func (c *Client) handleError(err error) {
	if c.errorHandler != nil {
		go c.errorHandler(err)
	}
	c.logger.Error("发生错误", "error", err)
	c.metrics.RecordError(err.Error())
}

// GetMetrics 获取指标
func (c *Client) GetMetrics() map[string]interface{} {
	return c.metrics.GetMetrics()
}

// GetMetricsReport 获取指标报告
func (c *Client) GetMetricsReport() string {
	return c.metrics.GetMetricsReport()
}
