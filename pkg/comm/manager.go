package comm

import (
	"errors"
	"sync"
	"time"

	"github.com/lomehong/kennel/pkg/logger"
)

// Manager 通讯管理器，负责管理与服务端的通信
type Manager struct {
	client       *Client
	config       ConnectionConfig
	logger       logger.Logger
	handlers     map[MessageType][]MessageHandler
	handlerMutex sync.RWMutex
}

// NewManager 创建一个新的通讯管理器
func NewManager(config ConnectionConfig, log logger.Logger) *Manager {
	if log == nil {
		log = logger.NewLogger("comm-manager", logger.GetLogLevel("info"))
	}

	manager := &Manager{
		config:   config,
		logger:   log,
		handlers: make(map[MessageType][]MessageHandler),
	}

	// 创建客户端
	manager.client = NewClient(config, log)

	// 设置消息处理函数
	manager.client.SetMessageHandler(manager.dispatchMessage)

	// 设置状态变化处理函数
	manager.client.SetStateChangeHandler(manager.handleStateChange)

	return manager
}

// Connect 连接到服务器
func (m *Manager) Connect() error {
	return m.client.Connect()
}

// Disconnect 断开连接
func (m *Manager) Disconnect() {
	m.logger.Info("正在断开与服务器的连接...")

	// 等待所有消息处理完成
	m.waitForPendingMessages()

	// 断开连接
	m.client.Disconnect()

	m.logger.Info("已断开与服务器的连接")
}

// waitForPendingMessages 等待所有待处理的消息完成
func (m *Manager) waitForPendingMessages() {
	// 这里可以添加等待消息处理完成的逻辑
	// 例如，使用WaitGroup或者通道来等待所有消息处理完成

	// 简单起见，这里只等待一小段时间
	time.Sleep(200 * time.Millisecond)
}

// IsConnected 检查是否已连接
func (m *Manager) IsConnected() bool {
	return m.client.IsConnected()
}

// GetState 获取当前连接状态
func (m *Manager) GetState() ConnectionState {
	return m.client.GetState()
}

// GetConfig 获取通讯配置
func (m *Manager) GetConfig() ConnectionConfig {
	return m.config
}

// SetClientInfo 设置客户端信息
func (m *Manager) SetClientInfo(info map[string]interface{}) {
	m.client.SetClientInfo(info)
}

// RegisterHandler 注册消息处理函数
func (m *Manager) RegisterHandler(msgType MessageType, handler MessageHandler) {
	m.handlerMutex.Lock()
	defer m.handlerMutex.Unlock()

	if _, ok := m.handlers[msgType]; !ok {
		m.handlers[msgType] = make([]MessageHandler, 0)
	}

	m.handlers[msgType] = append(m.handlers[msgType], handler)
}

// UnregisterHandler 注销消息处理函数
func (m *Manager) UnregisterHandler(msgType MessageType, handler MessageHandler) error {
	m.handlerMutex.Lock()
	defer m.handlerMutex.Unlock()

	if handlers, ok := m.handlers[msgType]; ok {
		for i, h := range handlers {
			if &h == &handler {
				// 找到处理函数，从切片中删除
				m.handlers[msgType] = append(handlers[:i], handlers[i+1:]...)
				return nil
			}
		}
	}

	return errors.New("处理函数未注册")
}

// SendMessage 发送消息
func (m *Manager) SendMessage(msgType MessageType, payload map[string]interface{}) {
	msg := NewMessage(msgType, payload)
	m.client.Send(msg)
}

// SendCommand 发送命令消息
func (m *Manager) SendCommand(command string, params map[string]interface{}) {
	payload := map[string]interface{}{
		"command": command,
		"params":  params,
	}
	m.SendMessage(MessageTypeCommand, payload)
}

// SendData 发送数据消息
func (m *Manager) SendData(dataType string, data interface{}) {
	payload := map[string]interface{}{
		"type": dataType,
		"data": data,
	}
	m.SendMessage(MessageTypeData, payload)
}

// SendEvent 发送事件消息
func (m *Manager) SendEvent(eventType string, details map[string]interface{}) {
	payload := map[string]interface{}{
		"event":   eventType,
		"details": details,
	}
	m.SendMessage(MessageTypeEvent, payload)
}

// SendResponse 发送响应消息
func (m *Manager) SendResponse(requestID string, success bool, data interface{}, errorMsg string) {
	payload := map[string]interface{}{
		"request_id": requestID,
		"success":    success,
		"data":       data,
	}

	if errorMsg != "" {
		payload["error"] = errorMsg
	}

	m.SendMessage(MessageTypeResponse, payload)
}

// dispatchMessage 分发消息到对应的处理函数
func (m *Manager) dispatchMessage(msg *Message) {
	m.handlerMutex.RLock()
	defer m.handlerMutex.RUnlock()

	// 查找对应类型的处理函数
	if handlers, ok := m.handlers[msg.Type]; ok {
		for _, handler := range handlers {
			go handler(msg)
		}
	}
}

// handleStateChange 处理连接状态变化
func (m *Manager) handleStateChange(oldState, newState ConnectionState) {
	m.logger.Info("连接状态变化", "old", oldState, "new", newState)
}

// GetClient 获取通讯客户端
func (m *Manager) GetClient() *Client {
	return m.client
}

// GetMetrics 获取指标
func (m *Manager) GetMetrics() map[string]interface{} {
	if m.client == nil {
		return map[string]interface{}{
			"connected": false,
			"state":     "未初始化",
		}
	}

	metrics := m.client.GetMetrics()

	// 添加管理器级别的指标
	m.handlerMutex.RLock()
	handlerCount := make(map[string]int)
	for msgType, handlers := range m.handlers {
		handlerCount[string(msgType)] = len(handlers)
	}
	m.handlerMutex.RUnlock()
	metrics["handler_count"] = handlerCount

	return metrics
}

// GetMetricsReport 获取指标报告
func (m *Manager) GetMetricsReport() string {
	if m.client == nil {
		return "通讯客户端未初始化"
	}

	return m.client.GetMetricsReport()
}
