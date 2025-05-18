package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// MessageType 消息类型
type MessageType string

// 预定义消息类型
const (
	MessageTypeRequest  MessageType = "request"  // 请求
	MessageTypeResponse MessageType = "response" // 响应
	MessageTypeEvent    MessageType = "event"    // 事件
	MessageTypeError    MessageType = "error"    // 错误
)

// Message 消息
type Message struct {
	ID          string                 // 消息ID
	Type        MessageType            // 消息类型
	Source      string                 // 消息源
	Destination string                 // 消息目标
	Topic       string                 // 消息主题
	Payload     map[string]interface{} // 消息负载
	Timestamp   time.Time              // 时间戳
}

// MessageHandler 消息处理器
type MessageHandler func(ctx context.Context, msg Message) (Message, error)

// PluginCommunicator 插件通信器
type PluginCommunicator struct {
	logger         hclog.Logger
	handlers       map[string]MessageHandler
	subscriptions  map[string][]string
	pluginManager  *PluginManager
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	subMu          sync.RWMutex
}

// NewPluginCommunicator 创建插件通信器
func NewPluginCommunicator(logger hclog.Logger, pluginManager *PluginManager) *PluginCommunicator {
	ctx, cancel := context.WithCancel(context.Background())

	return &PluginCommunicator{
		logger:         logger.Named("plugin-communicator"),
		handlers:       make(map[string]MessageHandler),
		subscriptions:  make(map[string][]string),
		pluginManager:  pluginManager,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// RegisterHandler 注册消息处理器
func (pc *PluginCommunicator) RegisterHandler(pluginID string, handler MessageHandler) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.handlers[pluginID] = handler
	pc.logger.Info("注册消息处理器", "plugin_id", pluginID)
}

// UnregisterHandler 注销消息处理器
func (pc *PluginCommunicator) UnregisterHandler(pluginID string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	delete(pc.handlers, pluginID)
	pc.logger.Info("注销消息处理器", "plugin_id", pluginID)
}

// Subscribe 订阅主题
func (pc *PluginCommunicator) Subscribe(pluginID string, topic string) {
	pc.subMu.Lock()
	defer pc.subMu.Unlock()

	// 检查订阅是否已存在
	for _, t := range pc.subscriptions[pluginID] {
		if t == topic {
			return
		}
	}

	// 添加订阅
	pc.subscriptions[pluginID] = append(pc.subscriptions[pluginID], topic)
	pc.logger.Info("订阅主题", "plugin_id", pluginID, "topic", topic)
}

// Unsubscribe 取消订阅主题
func (pc *PluginCommunicator) Unsubscribe(pluginID string, topic string) {
	pc.subMu.Lock()
	defer pc.subMu.Unlock()

	// 查找订阅
	for i, t := range pc.subscriptions[pluginID] {
		if t == topic {
			// 移除订阅
			pc.subscriptions[pluginID] = append(pc.subscriptions[pluginID][:i], pc.subscriptions[pluginID][i+1:]...)
			pc.logger.Info("取消订阅主题", "plugin_id", pluginID, "topic", topic)
			return
		}
	}
}

// UnsubscribeAll 取消所有订阅
func (pc *PluginCommunicator) UnsubscribeAll(pluginID string) {
	pc.subMu.Lock()
	defer pc.subMu.Unlock()
	delete(pc.subscriptions, pluginID)
	pc.logger.Info("取消所有订阅", "plugin_id", pluginID)
}

// GetSubscriptions 获取订阅
func (pc *PluginCommunicator) GetSubscriptions(pluginID string) []string {
	pc.subMu.RLock()
	defer pc.subMu.RUnlock()

	// 复制订阅
	subs := make([]string, len(pc.subscriptions[pluginID]))
	copy(subs, pc.subscriptions[pluginID])

	return subs
}

// GetSubscribers 获取订阅者
func (pc *PluginCommunicator) GetSubscribers(topic string) []string {
	pc.subMu.RLock()
	defer pc.subMu.RUnlock()

	var subscribers []string
	for pluginID, topics := range pc.subscriptions {
		for _, t := range topics {
			if t == topic {
				subscribers = append(subscribers, pluginID)
				break
			}
		}
	}

	return subscribers
}

// SendMessage 发送消息
func (pc *PluginCommunicator) SendMessage(ctx context.Context, msg Message) (Message, error) {
	pc.logger.Debug("发送消息",
		"id", msg.ID,
		"type", msg.Type,
		"source", msg.Source,
		"destination", msg.Destination,
		"topic", msg.Topic,
	)

	// 设置时间戳
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// 检查消息类型
	if msg.Type == MessageTypeEvent {
		// 广播事件
		return Message{}, pc.broadcastEvent(ctx, msg)
	}

	// 获取目标处理器
	pc.mu.RLock()
	handler, exists := pc.handlers[msg.Destination]
	pc.mu.RUnlock()

	if !exists {
		return Message{}, fmt.Errorf("目标插件不存在或未注册处理器: %s", msg.Destination)
	}

	// 处理消息
	response, err := handler(ctx, msg)
	if err != nil {
		pc.logger.Error("处理消息失败",
			"id", msg.ID,
			"type", msg.Type,
			"source", msg.Source,
			"destination", msg.Destination,
			"topic", msg.Topic,
			"error", err,
		)
		return Message{}, err
	}

	// 设置响应字段
	response.Type = MessageTypeResponse
	response.Source = msg.Destination
	response.Destination = msg.Source
	response.Timestamp = time.Now()

	pc.logger.Debug("接收响应",
		"id", response.ID,
		"type", response.Type,
		"source", response.Source,
		"destination", response.Destination,
		"topic", response.Topic,
	)

	return response, nil
}

// PublishEvent 发布事件
func (pc *PluginCommunicator) PublishEvent(ctx context.Context, source string, topic string, payload map[string]interface{}) error {
	// 创建事件消息
	msg := Message{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Type:      MessageTypeEvent,
		Source:    source,
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	// 广播事件
	return pc.broadcastEvent(ctx, msg)
}

// broadcastEvent 广播事件
func (pc *PluginCommunicator) broadcastEvent(ctx context.Context, msg Message) error {
	// 获取订阅者
	subscribers := pc.GetSubscribers(msg.Topic)
	if len(subscribers) == 0 {
		pc.logger.Debug("没有订阅者", "topic", msg.Topic)
		return nil
	}

	pc.logger.Debug("广播事件",
		"id", msg.ID,
		"source", msg.Source,
		"topic", msg.Topic,
		"subscribers", subscribers,
	)

	// 广播事件
	var wg sync.WaitGroup
	for _, pluginID := range subscribers {
		// 跳过源插件
		if pluginID == msg.Source {
			continue
		}

		wg.Add(1)
		go func(pluginID string) {
			defer wg.Done()

			// 获取处理器
			pc.mu.RLock()
			handler, exists := pc.handlers[pluginID]
			pc.mu.RUnlock()

			if !exists {
				pc.logger.Warn("订阅者未注册处理器", "plugin_id", pluginID)
				return
			}

			// 设置目标
			eventMsg := msg
			eventMsg.Destination = pluginID

			// 处理事件
			_, err := handler(ctx, eventMsg)
			if err != nil {
				pc.logger.Error("处理事件失败",
					"id", eventMsg.ID,
					"source", eventMsg.Source,
					"destination", eventMsg.Destination,
					"topic", eventMsg.Topic,
					"error", err,
				)
			}
		}(pluginID)
	}

	// 等待所有处理完成
	wg.Wait()

	return nil
}

// Close 关闭通信器
func (pc *PluginCommunicator) Close() error {
	pc.cancel()
	return nil
}

// PluginCommunicationInterface 插件通信接口
type PluginCommunicationInterface interface {
	// SendMessage 发送消息
	SendMessage(ctx context.Context, msg Message) (Message, error)

	// PublishEvent 发布事件
	PublishEvent(ctx context.Context, topic string, payload map[string]interface{}) error

	// Subscribe 订阅主题
	Subscribe(topic string) error

	// Unsubscribe 取消订阅主题
	Unsubscribe(topic string) error

	// GetSubscriptions 获取订阅
	GetSubscriptions() []string
}

// PluginCommunicationClient 插件通信客户端
type PluginCommunicationClient struct {
	pluginID     string
	communicator *PluginCommunicator
	logger       hclog.Logger
}

// NewPluginCommunicationClient 创建插件通信客户端
func NewPluginCommunicationClient(pluginID string, communicator *PluginCommunicator, logger hclog.Logger) *PluginCommunicationClient {
	return &PluginCommunicationClient{
		pluginID:     pluginID,
		communicator: communicator,
		logger:       logger.Named(fmt.Sprintf("plugin-comm-client-%s", pluginID)),
	}
}

// SendMessage 发送消息
func (pcc *PluginCommunicationClient) SendMessage(ctx context.Context, msg Message) (Message, error) {
	// 设置源
	msg.Source = pcc.pluginID

	// 发送消息
	return pcc.communicator.SendMessage(ctx, msg)
}

// PublishEvent 发布事件
func (pcc *PluginCommunicationClient) PublishEvent(ctx context.Context, topic string, payload map[string]interface{}) error {
	// 发布事件
	return pcc.communicator.PublishEvent(ctx, pcc.pluginID, topic, payload)
}

// Subscribe 订阅主题
func (pcc *PluginCommunicationClient) Subscribe(topic string) error {
	// 订阅主题
	pcc.communicator.Subscribe(pcc.pluginID, topic)
	return nil
}

// Unsubscribe 取消订阅主题
func (pcc *PluginCommunicationClient) Unsubscribe(topic string) error {
	// 取消订阅主题
	pcc.communicator.Unsubscribe(pcc.pluginID, topic)
	return nil
}

// GetSubscriptions 获取订阅
func (pcc *PluginCommunicationClient) GetSubscriptions() []string {
	// 获取订阅
	return pcc.communicator.GetSubscriptions(pcc.pluginID)
}
