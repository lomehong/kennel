package communication

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/sdk"
)

// CommunicationFactory 通信工厂
// 用于创建不同类型的通信实例
type CommunicationFactory interface {
	// CreateCommunication 创建通信实例
	// protocol: 通信协议
	// options: 选项
	// 返回: 通信实例和错误
	CreateCommunication(protocol sdk.CommunicationProtocol, options map[string]interface{}) (sdk.Communication, error)
	
	// RegisterHandler 注册协议处理器
	// protocol: 通信协议
	// handler: 处理器
	RegisterHandler(protocol sdk.CommunicationProtocol, handler sdk.CommunicationHandler)
}

// DefaultCommunicationFactory 默认通信工厂实现
type DefaultCommunicationFactory struct {
	// 协议处理器映射
	handlers map[sdk.CommunicationProtocol]sdk.CommunicationHandler
	
	// 互斥锁
	mu sync.RWMutex
	
	// 日志记录器
	logger hclog.Logger
}

// NewCommunicationFactory 创建一个新的通信工厂
func NewCommunicationFactory(logger hclog.Logger) *DefaultCommunicationFactory {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}
	
	factory := &DefaultCommunicationFactory{
		handlers: make(map[sdk.CommunicationProtocol]sdk.CommunicationHandler),
		logger:   logger.Named("communication-factory"),
	}
	
	// 注册默认处理器
	factory.RegisterHandler(sdk.ProtocolGRPC, newGRPCCommunication)
	factory.RegisterHandler(sdk.ProtocolHTTP, newHTTPCommunication)
	factory.RegisterHandler(sdk.ProtocolWebSocket, newWebSocketCommunication)
	factory.RegisterHandler(sdk.ProtocolInProcess, newInProcessCommunication)
	
	return factory
}

// RegisterHandler 注册协议处理器
func (f *DefaultCommunicationFactory) RegisterHandler(protocol sdk.CommunicationProtocol, handler sdk.CommunicationHandler) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[protocol] = handler
	f.logger.Debug("注册通信协议处理器", "protocol", protocol)
}

// CreateCommunication 创建通信实例
func (f *DefaultCommunicationFactory) CreateCommunication(protocol sdk.CommunicationProtocol, options map[string]interface{}) (sdk.Communication, error) {
	f.mu.RLock()
	handler, ok := f.handlers[protocol]
	f.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("不支持的通信协议: %s", protocol)
	}
	
	f.logger.Debug("创建通信实例", "protocol", protocol)
	return handler(options)
}

// newGRPCCommunication 创建gRPC通信实例
func newGRPCCommunication(options map[string]interface{}) (sdk.Communication, error) {
	// 获取选项
	address, _ := options["address"].(string)
	isServer, _ := options["is_server"].(bool)
	logger, _ := options["logger"].(hclog.Logger)
	eventBus, _ := options["event_bus"].(EventBus)
	
	// 创建gRPC选项
	grpcOptions := GRPCOptions{
		Address:  address,
		IsServer: isServer,
		Logger:   logger,
		EventBus: eventBus,
	}
	
	// 创建gRPC通信
	comm, err := NewGRPCCommunication(grpcOptions)
	if err != nil {
		return nil, err
	}
	
	// 包装为SDK通信接口
	return &communicationAdapter{
		comm: comm,
	}, nil
}

// newHTTPCommunication 创建HTTP通信实例
func newHTTPCommunication(options map[string]interface{}) (sdk.Communication, error) {
	// 获取选项
	address, _ := options["address"].(string)
	isServer, _ := options["is_server"].(bool)
	logger, _ := options["logger"].(hclog.Logger)
	eventBus, _ := options["event_bus"].(EventBus)
	
	// 创建HTTP选项
	httpOptions := HTTPOptions{
		Address:  address,
		IsServer: isServer,
		Logger:   logger,
		EventBus: eventBus,
	}
	
	// 创建HTTP通信
	comm, err := NewHTTPCommunication(httpOptions)
	if err != nil {
		return nil, err
	}
	
	// 包装为SDK通信接口
	return &communicationAdapter{
		comm: comm,
	}, nil
}

// newWebSocketCommunication 创建WebSocket通信实例
func newWebSocketCommunication(options map[string]interface{}) (sdk.Communication, error) {
	// 获取选项
	address, _ := options["address"].(string)
	isServer, _ := options["is_server"].(bool)
	logger, _ := options["logger"].(hclog.Logger)
	eventBus, _ := options["event_bus"].(EventBus)
	
	// 创建WebSocket选项
	wsOptions := WebSocketOptions{
		Address:  address,
		IsServer: isServer,
		Logger:   logger,
		EventBus: eventBus,
	}
	
	// 创建WebSocket通信
	comm, err := NewWebSocketCommunication(wsOptions)
	if err != nil {
		return nil, err
	}
	
	// 包装为SDK通信接口
	return &communicationAdapter{
		comm: comm,
	}, nil
}

// newInProcessCommunication 创建进程内通信实例
func newInProcessCommunication(options map[string]interface{}) (sdk.Communication, error) {
	// 获取选项
	logger, _ := options["logger"].(hclog.Logger)
	eventBus, _ := options["event_bus"].(EventBus)
	
	// 如果没有事件总线，创建一个
	if eventBus == nil {
		eventBus = NewEventBus(logger)
	}
	
	// 创建进程内通信
	comm := &InProcessCommunication{
		services:      make(map[string]interface{}),
		subscriptions: make(map[string][]func(message interface{})),
		eventBus:      eventBus,
		logger:        logger,
	}
	
	// 包装为SDK通信接口
	return &communicationAdapter{
		comm: comm,
	}, nil
}

// InProcessCommunication 进程内通信实现
type InProcessCommunication struct {
	// 服务映射
	services map[string]interface{}
	
	// 订阅映射
	subscriptions map[string][]func(message interface{})
	
	// 互斥锁
	mu sync.RWMutex
	
	// 日志记录器
	logger hclog.Logger
	
	// 事件总线
	eventBus EventBus
}

// RegisterService 注册服务
func (c *InProcessCommunication) RegisterService(service interface{}) error {
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
func (c *InProcessCommunication) GetService(name string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	service, ok := c.services[name]
	if !ok {
		return nil, fmt.Errorf("服务不存在: %s", name)
	}
	
	return service, nil
}

// SendMessage 发送消息
func (c *InProcessCommunication) SendMessage(target string, message interface{}) error {
	// 通过事件总线发送
	return c.eventBus.Publish(Event{
		Type:   EventType(target),
		Source: "in-process",
		Data: map[string]interface{}{
			"message": message,
		},
	})
}

// ReceiveMessage 接收消息
func (c *InProcessCommunication) ReceiveMessage() (string, interface{}, error) {
	// 在实际实现中，这里应该使用事件总线接收消息
	// 这里简化处理，返回错误
	return "", nil, fmt.Errorf("不支持直接接收消息，请使用Subscribe")
}

// Subscribe 订阅主题
func (c *InProcessCommunication) Subscribe(topic string, handler func(message interface{})) error {
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
func (c *InProcessCommunication) Publish(topic string, message interface{}) error {
	// 通过事件总线发布
	return c.eventBus.Publish(Event{
		Type:   EventType(topic),
		Source: "in-process",
		Data: map[string]interface{}{
			"message": message,
		},
	})
}

// Close 关闭通信
func (c *InProcessCommunication) Close() error {
	// 关闭事件总线
	if c.eventBus != nil {
		return c.eventBus.Close()
	}
	return nil
}

// communicationAdapter 通信适配器
// 将内部通信实现适配为SDK通信接口
type communicationAdapter struct {
	// 通信实现
	comm interface {
		RegisterService(service interface{}) error
		GetService(name string) (interface{}, error)
		SendMessage(target string, message interface{}) error
		ReceiveMessage() (string, interface{}, error)
		Subscribe(topic string, handler func(message interface{})) error
		Publish(topic string, message interface{}) error
		Close() error
	}
}

// RegisterService 注册服务
func (a *communicationAdapter) RegisterService(service interface{}) error {
	return a.comm.RegisterService(service)
}

// GetService 获取服务
func (a *communicationAdapter) GetService(name string) (interface{}, error) {
	return a.comm.GetService(name)
}

// SendMessage 发送消息
func (a *communicationAdapter) SendMessage(target string, message interface{}) error {
	return a.comm.SendMessage(target, message)
}

// ReceiveMessage 接收消息
func (a *communicationAdapter) ReceiveMessage() (string, interface{}, error) {
	return a.comm.ReceiveMessage()
}

// Subscribe 订阅主题
func (a *communicationAdapter) Subscribe(topic string, handler func(message interface{})) error {
	return a.comm.Subscribe(topic, handler)
}

// Publish 发布消息到主题
func (a *communicationAdapter) Publish(topic string, message interface{}) error {
	return a.comm.Publish(topic, message)
}

// Close 关闭通信
func (a *communicationAdapter) Close() error {
	return a.comm.Close()
}
