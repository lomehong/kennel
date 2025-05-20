package sdk

import (
	"context"
	"fmt"
	"sync"
)

// Communication 定义了通信接口
// 提供了插件与主框架之间以及插件之间的通信能力
type Communication interface {
	// RegisterService 注册服务
	// service: 服务实例
	// 返回: 错误
	RegisterService(service interface{}) error

	// GetService 获取服务
	// name: 服务名称
	// 返回: 服务实例和错误
	GetService(name string) (interface{}, error)

	// SendMessage 发送消息
	// target: 目标
	// message: 消息内容
	// 返回: 错误
	SendMessage(target string, message interface{}) error

	// ReceiveMessage 接收消息
	// 返回: 源、消息内容和错误
	ReceiveMessage() (string, interface{}, error)

	// Subscribe 订阅主题
	// topic: 主题
	// handler: 处理函数
	// 返回: 错误
	Subscribe(topic string, handler func(message interface{})) error

	// Publish 发布消息到主题
	// topic: 主题
	// message: 消息内容
	// 返回: 错误
	Publish(topic string, message interface{}) error

	// Close 关闭通信
	// 返回: 错误
	Close() error
}

// CommunicationProtocol 定义了通信协议
type CommunicationProtocol string

// 预定义的通信协议
const (
	ProtocolGRPC      CommunicationProtocol = "grpc"      // gRPC协议
	ProtocolHTTP      CommunicationProtocol = "http"      // HTTP协议
	ProtocolWebSocket CommunicationProtocol = "websocket" // WebSocket协议
	ProtocolInProcess CommunicationProtocol = "inprocess" // 进程内通信
)

// CommunicationFactory 定义了通信工厂
// 用于创建不同协议的通信实例
type CommunicationFactory interface {
	// CreateCommunication 创建通信实例
	// protocol: 通信协议
	// options: 选项
	// 返回: 通信实例和错误
	CreateCommunication(protocol CommunicationProtocol, options map[string]interface{}) (Communication, error)
}

// DefaultCommunicationFactory 默认通信工厂实现
type DefaultCommunicationFactory struct {
	// 协议处理器映射
	handlers map[CommunicationProtocol]CommunicationHandler

	// 互斥锁
	mu sync.RWMutex
}

// CommunicationHandler 定义了通信处理器
type CommunicationHandler func(options map[string]interface{}) (Communication, error)

// NewCommunicationFactory 创建一个新的通信工厂
func NewCommunicationFactory() *DefaultCommunicationFactory {
	factory := &DefaultCommunicationFactory{
		handlers: make(map[CommunicationProtocol]CommunicationHandler),
	}

	// 注册默认处理器
	factory.RegisterHandler(ProtocolInProcess, newInProcessCommunication)

	return factory
}

// RegisterHandler 注册协议处理器
func (f *DefaultCommunicationFactory) RegisterHandler(protocol CommunicationProtocol, handler CommunicationHandler) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[protocol] = handler
}

// CreateCommunication 创建通信实例
func (f *DefaultCommunicationFactory) CreateCommunication(protocol CommunicationProtocol, options map[string]interface{}) (Communication, error) {
	f.mu.RLock()
	handler, ok := f.handlers[protocol]
	f.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("不支持的通信协议: %s", protocol)
	}

	return handler(options)
}

// Message 消息
type Message struct {
	// 源
	Source string

	// 目标
	Target string

	// 内容
	Content interface{}
}

// InProcessCommunication 进程内通信实现
type InProcessCommunication struct {
	// 服务映射
	services map[string]interface{}

	// 订阅映射
	subscriptions map[string][]func(message interface{})

	// 互斥锁
	mu sync.RWMutex

	// 消息通道
	messageCh chan Message

	// 上下文
	ctx context.Context

	// 取消函数
	cancel context.CancelFunc
}

// newInProcessCommunication 创建进程内通信实例
func newInProcessCommunication(options map[string]interface{}) (Communication, error) {
	ctx, cancel := context.WithCancel(context.Background())

	comm := &InProcessCommunication{
		services:      make(map[string]interface{}),
		subscriptions: make(map[string][]func(message interface{})),
		messageCh:     make(chan Message, 100),
		ctx:           ctx,
		cancel:        cancel,
	}

	// 启动消息处理
	go comm.processMessages()

	return comm, nil
}

// processMessages 处理消息
func (c *InProcessCommunication) processMessages() {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg := <-c.messageCh:
			// 处理消息
			c.handleMessage(msg)
		}
	}
}

// handleMessage 处理消息
func (c *InProcessCommunication) handleMessage(msg Message) {
	// 如果有目标，发送到目标
	if msg.Target != "" {
		// 这里可以实现目标路由逻辑
		return
	}

	// 否则作为主题消息处理
	c.mu.RLock()
	handlers, ok := c.subscriptions[msg.Source]
	c.mu.RUnlock()

	if !ok {
		return
	}

	// 调用所有处理器
	for _, handler := range handlers {
		go handler(msg.Content)
	}
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
	select {
	case c.messageCh <- Message{Target: target, Content: message}:
		return nil
	default:
		return fmt.Errorf("消息队列已满")
	}
}

// ReceiveMessage 接收消息
func (c *InProcessCommunication) ReceiveMessage() (string, interface{}, error) {
	select {
	case <-c.ctx.Done():
		return "", nil, fmt.Errorf("通信已关闭")
	case msg := <-c.messageCh:
		return msg.Source, msg.Content, nil
	}
}

// Subscribe 订阅主题
func (c *InProcessCommunication) Subscribe(topic string, handler func(message interface{})) error {
	if handler == nil {
		return fmt.Errorf("处理器不能为空")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.subscriptions[topic] = append(c.subscriptions[topic], handler)
	return nil
}

// Publish 发布消息到主题
func (c *InProcessCommunication) Publish(topic string, message interface{}) error {
	select {
	case c.messageCh <- Message{Source: topic, Content: message}:
		return nil
	default:
		return fmt.Errorf("消息队列已满")
	}
}

// Close 关闭通信
func (c *InProcessCommunication) Close() error {
	c.cancel()
	return nil
}
