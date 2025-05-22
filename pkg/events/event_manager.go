package events

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lomehong/kennel/pkg/logging"
)

// Event 表示一个事件
type Event struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// EventHandler 事件处理函数
type EventHandler func(event Event) error

// EventManager 事件管理器，负责事件的发布和订阅
type EventManager struct {
	logger        logging.Logger
	handlers      map[string][]EventHandler
	handlersMutex sync.RWMutex
	events        []Event
	eventsMutex   sync.RWMutex
	maxEvents     int
}

// EventManagerOption 事件管理器选项
type EventManagerOption func(*EventManager)

// WithMaxEvents 设置最大事件数量
func WithMaxEvents(count int) EventManagerOption {
	return func(em *EventManager) {
		em.maxEvents = count
	}
}

// NewEventManager 创建一个新的事件管理器
func NewEventManager(log logging.Logger, options ...EventManagerOption) *EventManager {
	if log == nil {
		// 创建默认日志配置
		config := logging.DefaultLogConfig()
		config.Level = logging.LogLevelInfo
		
		// 创建增强日志记录器
		enhancedLogger, err := logging.NewEnhancedLogger(config)
		if err != nil {
			// 如果创建失败，使用默认配置
			enhancedLogger, _ = logging.NewEnhancedLogger(nil)
		}
		
		// 设置名称
		log = enhancedLogger.Named("event-manager")
	}

	// 创建事件管理器
	em := &EventManager{
		logger:    log,
		handlers:  make(map[string][]EventHandler),
		events:    make([]Event, 0, 1000),
		maxEvents: 10000,
	}

	// 应用选项
	for _, option := range options {
		option(em)
	}

	return em
}

// RegisterEventHandler 注册事件处理函数
func (em *EventManager) RegisterEventHandler(eventType string, handler EventHandler) error {
	if eventType == "" {
		return fmt.Errorf("事件类型不能为空")
	}
	if handler == nil {
		return fmt.Errorf("事件处理函数不能为空")
	}

	em.handlersMutex.Lock()
	defer em.handlersMutex.Unlock()

	// 添加处理函数
	em.handlers[eventType] = append(em.handlers[eventType], handler)
	em.logger.Info("注册事件处理函数", "type", eventType)

	return nil
}

// UnregisterEventHandler 注销事件处理函数
func (em *EventManager) UnregisterEventHandler(eventType string, handler EventHandler) error {
	if eventType == "" {
		return fmt.Errorf("事件类型不能为空")
	}
	if handler == nil {
		return fmt.Errorf("事件处理函数不能为空")
	}

	em.handlersMutex.Lock()
	defer em.handlersMutex.Unlock()

	// 查找处理函数
	handlers, ok := em.handlers[eventType]
	if !ok {
		return fmt.Errorf("未找到事件类型: %s", eventType)
	}

	// 移除处理函数
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			em.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			em.logger.Info("注销事件处理函数", "type", eventType)
			return nil
		}
	}

	return fmt.Errorf("未找到事件处理函数")
}

// PublishEvent 发布事件
func (em *EventManager) PublishEvent(event Event) error {
	if event.Type == "" {
		return fmt.Errorf("事件类型不能为空")
	}

	// 设置事件ID和时间戳
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 存储事件
	em.eventsMutex.Lock()
	em.events = append(em.events, event)
	// 如果超过最大事件数量，删除最旧的事件
	if len(em.events) > em.maxEvents {
		em.events = em.events[len(em.events)-em.maxEvents:]
	}
	em.eventsMutex.Unlock()

	// 调用处理函数
	em.handlersMutex.RLock()
	handlers := em.handlers[event.Type]
	allHandlers := em.handlers["*"] // 通配符处理函数
	em.handlersMutex.RUnlock()

	// 合并处理函数
	allHandlers = append(allHandlers, handlers...)

	// 异步调用处理函数
	for _, handler := range allHandlers {
		go func(h EventHandler, e Event) {
			if err := h(e); err != nil {
				em.logger.Error("事件处理失败", "type", e.Type, "id", e.ID, "error", err)
			}
		}(handler, event)
	}

	em.logger.Info("发布事件", "type", event.Type, "id", event.ID)
	return nil
}

// GetEvents 获取事件列表
func (em *EventManager) GetEvents(limit int, offset int, eventType string, source string) ([]interface{}, error) {
	em.eventsMutex.RLock()
	defer em.eventsMutex.RUnlock()

	// 过滤事件
	filtered := make([]Event, 0)
	for _, event := range em.events {
		if (eventType == "" || event.Type == eventType) && (source == "" || event.Source == source) {
			filtered = append(filtered, event)
		}
	}

	// 按时间倒序排序
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	// 应用分页
	total := len(filtered)
	if offset >= total {
		return []interface{}{}, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	// 转换为接口切片
	result := make([]interface{}, end-offset)
	for i, event := range filtered[offset:end] {
		result[i] = map[string]interface{}{
			"id":        event.ID,
			"timestamp": event.Timestamp.Format(time.RFC3339),
			"type":      event.Type,
			"message":   event.Message,
			"source":    event.Source,
			"data":      event.Data,
		}
	}

	return result, nil
}

// GetEventCount 获取事件数量
func (em *EventManager) GetEventCount() int {
	em.eventsMutex.RLock()
	defer em.eventsMutex.RUnlock()
	return len(em.events)
}

// ClearEvents 清除所有事件
func (em *EventManager) ClearEvents() {
	em.eventsMutex.Lock()
	defer em.eventsMutex.Unlock()
	em.events = make([]Event, 0, 1000)
	em.logger.Info("清除所有事件")
}

// GetEventTypes 获取所有事件类型
func (em *EventManager) GetEventTypes() []string {
	em.handlersMutex.RLock()
	defer em.handlersMutex.RUnlock()

	types := make([]string, 0, len(em.handlers))
	for eventType := range em.handlers {
		if eventType != "*" {
			types = append(types, eventType)
		}
	}

	return types
}
