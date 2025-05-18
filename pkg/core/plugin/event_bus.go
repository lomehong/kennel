package plugin

import (
	"context"
	"sync"
)

// EventBus 事件总线接口
type EventBus interface {
	// Publish 发布事件
	Publish(event *Event)

	// Subscribe 订阅事件
	Subscribe(eventType string, handler EventHandler) string

	// Unsubscribe 取消订阅
	Unsubscribe(subscriptionID string)

	// Close 关闭事件总线
	Close()
}

// EventHandler 事件处理器
type EventHandler func(ctx context.Context, event *Event) error

// DefaultEventBus 默认事件总线实现
type DefaultEventBus struct {
	handlers       map[string]map[string]EventHandler
	mu             sync.RWMutex
	nextHandlerID  int
	ctx            context.Context
	cancel         context.CancelFunc
	eventQueueSize int
	eventQueue     chan *Event
}

// NewDefaultEventBus 创建默认事件总线
func NewDefaultEventBus() *DefaultEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	bus := &DefaultEventBus{
		handlers:       make(map[string]map[string]EventHandler),
		nextHandlerID:  1,
		ctx:            ctx,
		cancel:         cancel,
		eventQueueSize: 100,
		eventQueue:     make(chan *Event, 100),
	}

	// 启动事件处理循环
	go bus.processEvents()

	return bus
}

// Publish 发布事件
func (b *DefaultEventBus) Publish(event *Event) {
	select {
	case b.eventQueue <- event:
		// 事件已入队
	default:
		// 队列已满，丢弃事件
	}
}

// Subscribe 订阅事件
func (b *DefaultEventBus) Subscribe(eventType string, handler EventHandler) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 创建事件类型的处理器映射（如果不存在）
	if _, exists := b.handlers[eventType]; !exists {
		b.handlers[eventType] = make(map[string]EventHandler)
	}

	// 生成处理器ID
	handlerID := generateHandlerID(eventType, b.nextHandlerID)
	b.nextHandlerID++

	// 存储处理器
	b.handlers[eventType][handlerID] = handler

	return handlerID
}

// Unsubscribe 取消订阅
func (b *DefaultEventBus) Unsubscribe(subscriptionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 解析订阅ID
	eventType, handlerID := parseSubscriptionID(subscriptionID)

	// 删除处理器
	if handlers, exists := b.handlers[eventType]; exists {
		delete(handlers, handlerID)
		// 如果没有处理器了，删除事件类型
		if len(handlers) == 0 {
			delete(b.handlers, eventType)
		}
	}
}

// Close 关闭事件总线
func (b *DefaultEventBus) Close() {
	b.cancel()
	close(b.eventQueue)
}

// processEvents 处理事件队列
func (b *DefaultEventBus) processEvents() {
	for {
		select {
		case event := <-b.eventQueue:
			b.dispatchEvent(event)
		case <-b.ctx.Done():
			return
		}
	}
}

// dispatchEvent 分发事件
func (b *DefaultEventBus) dispatchEvent(event *Event) {
	b.mu.RLock()
	// 获取特定事件类型的处理器
	typeHandlers := b.handlers[event.Type]
	// 获取通配符处理器
	wildcardHandlers := b.handlers["*"]
	b.mu.RUnlock()

	// 创建事件上下文
	ctx := context.Background()

	// 调用特定事件类型的处理器
	for _, handler := range typeHandlers {
		go func(h EventHandler) {
			_ = h(ctx, event)
		}(handler)
	}

	// 调用通配符处理器
	for _, handler := range wildcardHandlers {
		go func(h EventHandler) {
			_ = h(ctx, event)
		}(handler)
	}
}

// generateHandlerID 生成处理器ID
func generateHandlerID(eventType string, id int) string {
	return eventType + ":" + string(id)
}

// parseSubscriptionID 解析订阅ID
func parseSubscriptionID(subscriptionID string) (string, string) {
	// 简单实现，实际应该使用更健壮的方法
	return subscriptionID, subscriptionID
}
