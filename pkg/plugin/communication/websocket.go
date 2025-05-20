package communication

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-hclog"
)

// WebSocketCommunication WebSocket通信实现
type WebSocketCommunication struct {
	// 服务映射
	services map[string]interface{}
	
	// 互斥锁
	mu sync.RWMutex
	
	// 日志记录器
	logger hclog.Logger
	
	// 上下文
	ctx context.Context
	
	// 取消函数
	cancel context.CancelFunc
	
	// HTTP服务器
	server *http.Server
	
	// WebSocket升级器
	upgrader websocket.Upgrader
	
	// WebSocket连接
	conn *websocket.Conn
	
	// 事件总线
	eventBus EventBus
	
	// 地址
	address string
	
	// 是否为服务器
	isServer bool
	
	// 是否已关闭
	closed bool
	
	// 路由器
	router *mux.Router
	
	// 连接映射
	connections map[string]*websocket.Conn
	
	// 连接互斥锁
	connMu sync.RWMutex
	
	// 消息通道
	messageCh chan Message
}

// Message WebSocket消息
type Message struct {
	// 消息类型
	Type string `json:"type"`
	
	// 消息目标
	Target string `json:"target"`
	
	// 消息源
	Source string `json:"source"`
	
	// 消息ID
	ID string `json:"id"`
	
	// 消息时间
	Timestamp time.Time `json:"timestamp"`
	
	// 消息数据
	Data map[string]interface{} `json:"data"`
}

// WebSocketOptions WebSocket选项
type WebSocketOptions struct {
	// 地址
	Address string
	
	// 是否为服务器
	IsServer bool
	
	// 日志记录器
	Logger hclog.Logger
	
	// 事件总线
	EventBus EventBus
	
	// 路由注册函数
	RouteRegistrar func(*mux.Router)
	
	// 服务器选项
	ServerOptions *http.Server
	
	// 超时时间
	Timeout time.Duration
	
	// 心跳间隔
	HeartbeatInterval time.Duration
}

// NewWebSocketCommunication 创建一个新的WebSocket通信
func NewWebSocketCommunication(options WebSocketOptions) (*WebSocketCommunication, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// 设置默认值
	if options.Logger == nil {
		options.Logger = hclog.NewNullLogger()
	}
	
	if options.EventBus == nil {
		options.EventBus = NewEventBus(options.Logger)
	}
	
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}
	
	if options.HeartbeatInterval == 0 {
		options.HeartbeatInterval = 30 * time.Second
	}
	
	// 创建通信
	comm := &WebSocketCommunication{
		services:    make(map[string]interface{}),
		logger:      options.Logger.Named("websocket-communication"),
		ctx:         ctx,
		cancel:      cancel,
		eventBus:    options.EventBus,
		address:     options.Address,
		isServer:    options.IsServer,
		closed:      false,
		router:      mux.NewRouter(),
		connections: make(map[string]*websocket.Conn),
		messageCh:   make(chan Message, 100),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
	
	// 如果是服务器，启动服务器
	if options.IsServer {
		if err := comm.startServer(options); err != nil {
			cancel()
			return nil, fmt.Errorf("启动WebSocket服务器失败: %w", err)
		}
	} else {
		// 否则，连接服务器
		if err := comm.connectServer(options); err != nil {
			cancel()
			return nil, fmt.Errorf("连接WebSocket服务器失败: %w", err)
		}
	}
	
	// 启动消息处理
	go comm.processMessages()
	
	// 启动心跳
	go comm.heartbeat(options.HeartbeatInterval)
	
	return comm, nil
}

// startServer 启动服务器
func (c *WebSocketCommunication) startServer(options WebSocketOptions) error {
	// 注册路由
	if options.RouteRegistrar != nil {
		options.RouteRegistrar(c.router)
	}
	
	// 注册WebSocket路由
	c.router.HandleFunc("/ws", c.handleWebSocket)
	
	// 创建服务器
	server := &http.Server{
		Addr:         options.Address,
		Handler:      c.router,
		ReadTimeout:  options.Timeout,
		WriteTimeout: options.Timeout,
	}
	
	// 如果有自定义选项，使用自定义选项
	if options.ServerOptions != nil {
		server = options.ServerOptions
		server.Addr = options.Address
		server.Handler = c.router
	}
	
	c.server = server
	
	// 启动服务器
	c.logger.Info("启动WebSocket服务器", "address", options.Address)
	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.logger.Error("WebSocket服务器错误", "error", err)
		}
	}()
	
	return nil
}

// handleWebSocket 处理WebSocket连接
func (c *WebSocketCommunication) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.Error("升级WebSocket连接失败", "error", err)
		return
	}
	
	// 生成连接ID
	connID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())
	
	// 存储连接
	c.connMu.Lock()
	c.connections[connID] = conn
	c.connMu.Unlock()
	
	c.logger.Info("WebSocket连接已建立", "id", connID, "remote_addr", r.RemoteAddr)
	
	// 处理连接
	go c.handleConnection(connID, conn)
}

// handleConnection 处理连接
func (c *WebSocketCommunication) handleConnection(connID string, conn *websocket.Conn) {
	defer func() {
		// 关闭连接
		conn.Close()
		
		// 删除连接
		c.connMu.Lock()
		delete(c.connections, connID)
		c.connMu.Unlock()
		
		c.logger.Info("WebSocket连接已关闭", "id", connID)
	}()
	
	for {
		// 读取消息
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("读取WebSocket消息失败", "id", connID, "error", err)
			}
			break
		}
		
		// 解析消息
		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			c.logger.Error("解析WebSocket消息失败", "id", connID, "error", err)
			continue
		}
		
		// 设置消息源
		message.Source = connID
		
		// 设置消息时间
		if message.Timestamp.IsZero() {
			message.Timestamp = time.Now()
		}
		
		// 处理消息
		c.messageCh <- message
	}
}

// connectServer 连接服务器
func (c *WebSocketCommunication) connectServer(options WebSocketOptions) error {
	// 连接WebSocket服务器
	c.logger.Info("连接WebSocket服务器", "address", options.Address)
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/ws", options.Address), nil)
	if err != nil {
		return fmt.Errorf("连接WebSocket服务器失败: %w", err)
	}
	
	c.conn = conn
	
	// 处理连接
	go c.handleClientConnection()
	
	return nil
}

// handleClientConnection 处理客户端连接
func (c *WebSocketCommunication) handleClientConnection() {
	defer func() {
		// 关闭连接
		c.conn.Close()
		c.logger.Info("WebSocket客户端连接已关闭")
	}()
	
	for {
		// 读取消息
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("读取WebSocket消息失败", "error", err)
			}
			break
		}
		
		// 解析消息
		var message Message
		if err := json.Unmarshal(data, &message); err != nil {
			c.logger.Error("解析WebSocket消息失败", "error", err)
			continue
		}
		
		// 处理消息
		c.messageCh <- message
	}
}

// processMessages 处理消息
func (c *WebSocketCommunication) processMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case message := <-c.messageCh:
			c.handleMessage(message)
		}
	}
}

// handleMessage 处理消息
func (c *WebSocketCommunication) handleMessage(message Message) {
	// 处理心跳消息
	if message.Type == "heartbeat" {
		// 发送心跳响应
		response := Message{
			Type:      "heartbeat_response",
			Target:    message.Source,
			Source:    "server",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"status": "ok"},
		}
		
		// 发送响应
		if c.isServer {
			// 如果是服务器，通过连接发送
			c.connMu.RLock()
			conn, exists := c.connections[message.Source]
			c.connMu.RUnlock()
			
			if exists {
				data, _ := json.Marshal(response)
				conn.WriteMessage(websocket.TextMessage, data)
			}
		} else {
			// 如果是客户端，通过连接发送
			data, _ := json.Marshal(response)
			c.conn.WriteMessage(websocket.TextMessage, data)
		}
		
		return
	}
	
	// 处理普通消息
	if message.Type == "message" {
		// 通过事件总线发布
		c.eventBus.Publish(Event{
			Type:      EventType(message.Target),
			Source:    message.Source,
			Timestamp: message.Timestamp,
			Data:      message.Data,
		})
		
		return
	}
}

// heartbeat 心跳
func (c *WebSocketCommunication) heartbeat(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// 发送心跳
			message := Message{
				Type:      "heartbeat",
				Source:    "client",
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"status": "ok"},
			}
			
			// 序列化消息
			data, _ := json.Marshal(message)
			
			// 发送心跳
			if c.isServer {
				// 如果是服务器，向所有连接发送
				c.connMu.RLock()
				for id, conn := range c.connections {
					if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
						c.logger.Error("发送心跳失败", "id", id, "error", err)
					}
				}
				c.connMu.RUnlock()
			} else {
				// 如果是客户端，向服务器发送
				if c.conn != nil {
					if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
						c.logger.Error("发送心跳失败", "error", err)
					}
				}
			}
		}
	}
}

// RegisterService 注册服务
func (c *WebSocketCommunication) RegisterService(service interface{}) error {
	if service == nil {
		return fmt.Errorf("服务不能为空")
	}
	
	// 获取服务名称
	// 这里简化处理，实际应该通过反射获取
	name := fmt.Sprintf("%T", service)
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.services[name] = service
	return nil
}

// GetService 获取服务
func (c *WebSocketCommunication) GetService(name string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	service, ok := c.services[name]
	if !ok {
		return nil, fmt.Errorf("服务不存在: %s", name)
	}
	
	return service, nil
}

// SendMessage 发送消息
func (c *WebSocketCommunication) SendMessage(target string, message interface{}) error {
	// 创建消息
	wsMessage := Message{
		Type:      "message",
		Target:    target,
		Source:    "client",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"message": message},
	}
	
	// 序列化消息
	data, err := json.Marshal(wsMessage)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}
	
	// 发送消息
	if c.isServer {
		// 如果是服务器，向所有连接发送
		c.connMu.RLock()
		for id, conn := range c.connections {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				c.logger.Error("发送消息失败", "id", id, "error", err)
			}
		}
		c.connMu.RUnlock()
	} else {
		// 如果是客户端，向服务器发送
		if c.conn != nil {
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return fmt.Errorf("发送消息失败: %w", err)
			}
		} else {
			return fmt.Errorf("WebSocket连接未建立")
		}
	}
	
	return nil
}

// ReceiveMessage 接收消息
func (c *WebSocketCommunication) ReceiveMessage() (string, interface{}, error) {
	// 在实际实现中，这里应该使用WebSocket接收消息
	// 这里简化处理，返回错误
	return "", nil, fmt.Errorf("不支持直接接收消息，请使用Subscribe")
}

// Subscribe 订阅主题
func (c *WebSocketCommunication) Subscribe(topic string, handler func(message interface{})) error {
	// 通过事件总线订阅
	_, err := c.eventBus.Subscribe(EventType(topic), func(event Event) error {
		// 获取消息
		message, ok := event.Data["message"]
		if !ok {
			return fmt.Errorf("消息不存在")
		}
		
		// 调用处理器
		handler(message)
		return nil
	})
	
	return err
}

// Publish 发布消息到主题
func (c *WebSocketCommunication) Publish(topic string, message interface{}) error {
	// 创建消息
	wsMessage := Message{
		Type:      "message",
		Target:    topic,
		Source:    "client",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"message": message},
	}
	
	// 序列化消息
	data, err := json.Marshal(wsMessage)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}
	
	// 发送消息
	if c.isServer {
		// 如果是服务器，向所有连接发送
		c.connMu.RLock()
		for id, conn := range c.connections {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				c.logger.Error("发送消息失败", "id", id, "error", err)
			}
		}
		c.connMu.RUnlock()
		
		// 通过事件总线发布
		c.eventBus.Publish(Event{
			Type:      EventType(topic),
			Source:    "server",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"message": message},
		})
	} else {
		// 如果是客户端，向服务器发送
		if c.conn != nil {
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return fmt.Errorf("发送消息失败: %w", err)
			}
		} else {
			return fmt.Errorf("WebSocket连接未建立")
		}
	}
	
	return nil
}

// Close 关闭通信
func (c *WebSocketCommunication) Close() error {
	// 检查是否已关闭
	if c.closed {
		return nil
	}
	
	// 设置为已关闭
	c.closed = true
	
	// 取消上下文
	c.cancel()
	
	// 关闭事件总线
	if err := c.eventBus.Close(); err != nil {
		c.logger.Error("关闭事件总线失败", "error", err)
	}
	
	// 关闭所有连接
	c.connMu.Lock()
	for id, conn := range c.connections {
		conn.Close()
		c.logger.Debug("关闭WebSocket连接", "id", id)
	}
	c.connections = make(map[string]*websocket.Conn)
	c.connMu.Unlock()
	
	// 如果是客户端，关闭连接
	if !c.isServer && c.conn != nil {
		c.conn.Close()
	}
	
	// 如果是服务器，停止服务器
	if c.isServer && c.server != nil {
		c.logger.Info("停止WebSocket服务器")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.server.Shutdown(ctx); err != nil {
			c.logger.Error("关闭WebSocket服务器失败", "error", err)
		}
	}
	
	c.logger.Info("WebSocket通信已关闭")
	return nil
}
