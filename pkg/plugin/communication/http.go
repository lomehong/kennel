package communication

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-hclog"
)

// HTTPCommunication HTTP通信实现
type HTTPCommunication struct {
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
	
	// HTTP客户端
	client *http.Client
	
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
	
	// 处理器映射
	handlers map[string]http.Handler
}

// HTTPOptions HTTP选项
type HTTPOptions struct {
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
	
	// 客户端选项
	ClientOptions *http.Client
	
	// 服务器选项
	ServerOptions *http.Server
	
	// 超时时间
	Timeout time.Duration
}

// NewHTTPCommunication 创建一个新的HTTP通信
func NewHTTPCommunication(options HTTPOptions) (*HTTPCommunication, error) {
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
	
	// 创建通信
	comm := &HTTPCommunication{
		services: make(map[string]interface{}),
		logger:   options.Logger.Named("http-communication"),
		ctx:      ctx,
		cancel:   cancel,
		eventBus: options.EventBus,
		address:  options.Address,
		isServer: options.IsServer,
		closed:   false,
		router:   mux.NewRouter(),
		handlers: make(map[string]http.Handler),
	}
	
	// 如果是服务器，启动服务器
	if options.IsServer {
		if err := comm.startServer(options); err != nil {
			cancel()
			return nil, fmt.Errorf("启动HTTP服务器失败: %w", err)
		}
	} else {
		// 否则，创建客户端
		comm.createClient(options)
	}
	
	return comm, nil
}

// startServer 启动服务器
func (c *HTTPCommunication) startServer(options HTTPOptions) error {
	// 注册路由
	if options.RouteRegistrar != nil {
		options.RouteRegistrar(c.router)
	}
	
	// 注册默认路由
	c.registerDefaultRoutes()
	
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
	c.logger.Info("启动HTTP服务器", "address", options.Address)
	go func() {
		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			c.logger.Error("HTTP服务器错误", "error", err)
		}
	}()
	
	return nil
}

// registerDefaultRoutes 注册默认路由
func (c *HTTPCommunication) registerDefaultRoutes() {
	// 注册健康检查路由
	c.router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	}).Methods("GET")
	
	// 注册消息发送路由
	c.router.HandleFunc("/message/{target}", func(w http.ResponseWriter, r *http.Request) {
		// 获取目标
		vars := mux.Vars(r)
		target := vars["target"]
		
		// 读取消息
		var message map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			http.Error(w, fmt.Sprintf("读取消息失败: %v", err), http.StatusBadRequest)
			return
		}
		
		// 发送消息
		if err := c.SendMessage(target, message); err != nil {
			http.Error(w, fmt.Sprintf("发送消息失败: %v", err), http.StatusInternalServerError)
			return
		}
		
		// 返回成功
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"time":   time.Now().Format(time.RFC3339),
		})
	}).Methods("POST")
	
	// 注册发布消息路由
	c.router.HandleFunc("/publish/{topic}", func(w http.ResponseWriter, r *http.Request) {
		// 获取主题
		vars := mux.Vars(r)
		topic := vars["topic"]
		
		// 读取消息
		var message map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			http.Error(w, fmt.Sprintf("读取消息失败: %v", err), http.StatusBadRequest)
			return
		}
		
		// 发布消息
		if err := c.Publish(topic, message); err != nil {
			http.Error(w, fmt.Sprintf("发布消息失败: %v", err), http.StatusInternalServerError)
			return
		}
		
		// 返回成功
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"time":   time.Now().Format(time.RFC3339),
		})
	}).Methods("POST")
}

// createClient 创建客户端
func (c *HTTPCommunication) createClient(options HTTPOptions) {
	// 创建客户端
	client := &http.Client{
		Timeout: options.Timeout,
	}
	
	// 如果有自定义选项，使用自定义选项
	if options.ClientOptions != nil {
		client = options.ClientOptions
	}
	
	c.client = client
}

// RegisterService 注册服务
func (c *HTTPCommunication) RegisterService(service interface{}) error {
	if service == nil {
		return fmt.Errorf("服务不能为空")
	}
	
	// 获取服务名称
	// 这里简化处理，实际应该通过反射获取
	name := fmt.Sprintf("%T", service)
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.services[name] = service
	
	// 如果服务实现了http.Handler接口，注册为处理器
	if handler, ok := service.(http.Handler); ok {
		c.handlers[name] = handler
		c.router.Handle("/service/"+name, handler)
	}
	
	return nil
}

// GetService 获取服务
func (c *HTTPCommunication) GetService(name string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	service, ok := c.services[name]
	if !ok {
		return nil, fmt.Errorf("服务不存在: %s", name)
	}
	
	return service, nil
}

// SendMessage 发送消息
func (c *HTTPCommunication) SendMessage(target string, message interface{}) error {
	// 如果是服务器，通过事件总线发送
	if c.isServer {
		return c.eventBus.Publish(Event{
			Type:      EventType(target),
			Source:    "http",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"message": message,
			},
		})
	}
	
	// 如果是客户端，通过HTTP发送
	// 序列化消息
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}
	
	// 创建请求
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/message/%s", c.address, target), bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	
	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	
	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		// 读取错误信息
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("请求失败: %s - %s", resp.Status, string(body))
	}
	
	return nil
}

// ReceiveMessage 接收消息
func (c *HTTPCommunication) ReceiveMessage() (string, interface{}, error) {
	// 在实际实现中，这里应该使用HTTP服务器接收消息
	// 这里简化处理，返回错误
	return "", nil, fmt.Errorf("不支持直接接收消息，请使用Subscribe")
}

// Subscribe 订阅主题
func (c *HTTPCommunication) Subscribe(topic string, handler func(message interface{})) error {
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
func (c *HTTPCommunication) Publish(topic string, message interface{}) error {
	// 如果是服务器，通过事件总线发布
	if c.isServer {
		return c.eventBus.Publish(Event{
			Type:      EventType(topic),
			Source:    "http",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"message": message,
			},
		})
	}
	
	// 如果是客户端，通过HTTP发布
	// 序列化消息
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}
	
	// 创建请求
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/publish/%s", c.address, topic), bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	
	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	
	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		// 读取错误信息
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("请求失败: %s - %s", resp.Status, string(body))
	}
	
	return nil
}

// Close 关闭通信
func (c *HTTPCommunication) Close() error {
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
	
	// 如果是服务器，停止服务器
	if c.isServer && c.server != nil {
		c.logger.Info("停止HTTP服务器")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.server.Shutdown(ctx); err != nil {
			c.logger.Error("关闭HTTP服务器失败", "error", err)
		}
	}
	
	c.logger.Info("HTTP通信已关闭")
	return nil
}
