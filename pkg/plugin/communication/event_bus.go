package communication

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// EventType 事件类型
type EventType string

// EventPriority 事件优先级
type EventPriority int

const (
	// PriorityLow 低优先级
	PriorityLow EventPriority = iota
	
	// PriorityNormal 普通优先级
	PriorityNormal
	
	// PriorityHigh 高优先级
	PriorityHigh
	
	// PriorityCritical 关键优先级
	PriorityCritical
)

// Event 事件
type Event struct {
	// 事件类型
	Type EventType
	
	// 事件源
	Source string
	
	// 事件ID
	ID string
	
	// 事件优先级
	Priority EventPriority
	
	// 事件时间
	Timestamp time.Time
	
	// 事件数据
	Data map[string]interface{}
}

// EventHandler 事件处理器
type EventHandler func(event Event) error

// EventFilter 事件过滤器
type EventFilter func(event Event) bool

// EventSubscription 事件订阅
type EventSubscription struct {
	// 订阅ID
	ID string
	
	// 事件类型
	Type EventType
	
	// 事件源
	Source string
	
	// 事件处理器
	Handler EventHandler
	
	// 事件过滤器
	Filter EventFilter
	
	// 是否同步处理
	Synchronous bool
	
	// 订阅时间
	SubscribedAt time.Time
}

// EventBus 事件总线
// 提供事件发布和订阅功能
type EventBus interface {
	// Publish 发布事件
	// event: 事件
	// 返回: 错误
	Publish(event Event) error
	
	// Subscribe 订阅事件
	// eventType: 事件类型
	// handler: 事件处理器
	// 返回: 订阅ID和错误
	Subscribe(eventType EventType, handler EventHandler) (string, error)
	
	// SubscribeWithFilter 带过滤器订阅事件
	// eventType: 事件类型
	// handler: 事件处理器
	// filter: 事件过滤器
	// 返回: 订阅ID和错误
	SubscribeWithFilter(eventType EventType, handler EventHandler, filter EventFilter) (string, error)
	
	// SubscribeWithOptions 带选项订阅事件
	// options: 订阅选项
	// 返回: 订阅ID和错误
	SubscribeWithOptions(options SubscriptionOptions) (string, error)
	
	// Unsubscribe 取消订阅
	// subscriptionID: 订阅ID
	// 返回: 错误
	Unsubscribe(subscriptionID string) error
	
	// Close 关闭事件总线
	// 返回: 错误
	Close() error
}

// SubscriptionOptions 订阅选项
type SubscriptionOptions struct {
	// 事件类型
	Type EventType
	
	// 事件源
	Source string
	
	// 事件处理器
	Handler EventHandler
	
	// 事件过滤器
	Filter EventFilter
	
	// 是否同步处理
	Synchronous bool
	
	// 订阅ID
	ID string
}

// DefaultEventBus 默认事件总线实现
type DefaultEventBus struct {
	// 订阅映射
	subscriptions map[EventType]map[string]*EventSubscription
	
	// 互斥锁
	mu sync.RWMutex
	
	// 日志记录器
	logger hclog.Logger
	
	// 上下文
	ctx context.Context
	
	// 取消函数
	cancel context.CancelFunc
	
	// 事件通道
	eventCh chan Event
	
	// 是否已关闭
	closed bool
}

// NewEventBus 创建一个新的事件总线
func NewEventBus(logger hclog.Logger) *DefaultEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	
	bus := &DefaultEventBus{
		subscriptions: make(map[EventType]map[string]*EventSubscription),
		logger:        logger.Named("event-bus"),
		ctx:           ctx,
		cancel:        cancel,
		eventCh:       make(chan Event, 100),
		closed:        false,
	}
	
	// 启动事件处理
	go bus.processEvents()
	
	return bus
}

// processEvents 处理事件
func (b *DefaultEventBus) processEvents() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case event := <-b.eventCh:
			b.handleEvent(event)
		}
	}
}

// handleEvent 处理事件
func (b *DefaultEventBus) handleEvent(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// 获取事件类型的订阅
	subscriptions, exists := b.subscriptions[event.Type]
	if !exists {
		return
	}
	
	// 处理所有订阅
	for _, subscription := range subscriptions {
		// 检查过滤器
		if subscription.Filter != nil && !subscription.Filter(event) {
			continue
		}
		
		// 处理事件
		if subscription.Synchronous {
			// 同步处理
			if err := subscription.Handler(event); err != nil {
				b.logger.Error("处理事件失败", "type", event.Type, "id", event.ID, "error", err)
			}
		} else {
			// 异步处理
			go func(handler EventHandler, event Event) {
				if err := handler(event); err != nil {
					b.logger.Error("处理事件失败", "type", event.Type, "id", event.ID, "error", err)
				}
			}(subscription.Handler, event)
		}
	}
}

// Publish 发布事件
func (b *DefaultEventBus) Publish(event Event) error {
	// 检查是否已关闭
	if b.closed {
		return fmt.Errorf("事件总线已关闭")
	}
	
	// 设置事件时间
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	// 发送事件
	select {
	case b.eventCh <- event:
		return nil
	default:
		return fmt.Errorf("事件通道已满")
	}
}

// Subscribe 订阅事件
func (b *DefaultEventBus) Subscribe(eventType EventType, handler EventHandler) (string, error) {
	return b.SubscribeWithOptions(SubscriptionOptions{
		Type:        eventType,
		Handler:     handler,
		Synchronous: false,
		ID:          fmt.Sprintf("%s-%d", eventType, time.Now().UnixNano()),
	})
}

// SubscribeWithFilter 带过滤器订阅事件
func (b *DefaultEventBus) SubscribeWithFilter(eventType EventType, handler EventHandler, filter EventFilter) (string, error) {
	return b.SubscribeWithOptions(SubscriptionOptions{
		Type:        eventType,
		Handler:     handler,
		Filter:      filter,
		Synchronous: false,
		ID:          fmt.Sprintf("%s-%d", eventType, time.Now().UnixNano()),
	})
}

// SubscribeWithOptions 带选项订阅事件
func (b *DefaultEventBus) SubscribeWithOptions(options SubscriptionOptions) (string, error) {
	// 检查是否已关闭
	if b.closed {
		return "", fmt.Errorf("事件总线已关闭")
	}
	
	// 检查处理器
	if options.Handler == nil {
		return "", fmt.Errorf("处理器不能为空")
	}
	
	// 生成订阅ID
	subscriptionID := options.ID
	if subscriptionID == "" {
		subscriptionID = fmt.Sprintf("%s-%d", options.Type, time.Now().UnixNano())
	}
	
	// 创建订阅
	subscription := &EventSubscription{
		ID:           subscriptionID,
		Type:         options.Type,
		Source:       options.Source,
		Handler:      options.Handler,
		Filter:       options.Filter,
		Synchronous:  options.Synchronous,
		SubscribedAt: time.Now(),
	}
	
	// 添加订阅
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// 检查事件类型是否存在
	if _, exists := b.subscriptions[options.Type]; !exists {
		b.subscriptions[options.Type] = make(map[string]*EventSubscription)
	}
	
	// 添加订阅
	b.subscriptions[options.Type][subscriptionID] = subscription
	
	b.logger.Debug("订阅事件", "type", options.Type, "id", subscriptionID)
	return subscriptionID, nil
}

// Unsubscribe 取消订阅
func (b *DefaultEventBus) Unsubscribe(subscriptionID string) error {
	// 检查是否已关闭
	if b.closed {
		return fmt.Errorf("事件总线已关闭")
	}
	
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// 查找订阅
	for eventType, subscriptions := range b.subscriptions {
		if subscription, exists := subscriptions[subscriptionID]; exists {
			// 删除订阅
			delete(subscriptions, subscriptionID)
			
			// 如果没有订阅，删除事件类型
			if len(subscriptions) == 0 {
				delete(b.subscriptions, eventType)
			}
			
			b.logger.Debug("取消订阅事件", "type", subscription.Type, "id", subscriptionID)
			return nil
		}
	}
	
	return fmt.Errorf("订阅 %s 不存在", subscriptionID)
}

// Close 关闭事件总线
func (b *DefaultEventBus) Close() error {
	// 检查是否已关闭
	if b.closed {
		return nil
	}
	
	// 设置为已关闭
	b.closed = true
	
	// 取消上下文
	b.cancel()
	
	// 关闭事件通道
	close(b.eventCh)
	
	b.logger.Info("事件总线已关闭")
	return nil
}
