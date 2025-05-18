package core

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/comm"
	"github.com/lomehong/kennel/pkg/logger"
)

// CommManager 管理与服务端的通信
type CommManager struct {
	// 通讯管理器
	manager *comm.Manager

	// 配置管理器
	configManager *ConfigManager

	// 日志
	logger hclog.Logger

	// 消息路由表，将消息类型映射到处理函数
	routeTable map[comm.MessageType][]comm.MessageHandler

	// 互斥锁，用于保护路由表
	routeMutex sync.RWMutex

	// 是否已初始化
	initialized bool

	// 是否已连接
	connected bool
}

// NewCommManager 创建一个新的通讯管理器
func NewCommManager(configManager *ConfigManager) *CommManager {
	// 创建日志
	log := hclog.New(&hclog.LoggerOptions{
		Name:   "comm-manager",
		Output: os.Stdout,
		Level:  hclog.Info,
	})

	return &CommManager{
		configManager: configManager,
		logger:        log,
		routeTable:    make(map[comm.MessageType][]comm.MessageHandler),
		initialized:   false,
		connected:     false,
	}
}

// Init 初始化通讯管理器
func (cm *CommManager) Init() error {
	if cm.initialized {
		return nil
	}

	cm.logger.Info("初始化通讯管理器")

	// 创建通讯配置
	config := comm.DefaultConfig()

	// 从配置中读取服务器地址和端口
	serverAddress := cm.configManager.GetString("comm.server_address")
	serverPort := cm.configManager.GetInt("comm.server_port")

	// 如果配置中有服务器地址和端口，则构建WebSocket URL
	if serverAddress != "" {
		// 默认端口为9000
		if serverPort == 0 {
			serverPort = 9000
		}

		// 构建WebSocket URL
		config.ServerURL = fmt.Sprintf("ws://%s:%d/ws", serverAddress, serverPort)
		cm.logger.Info("使用配置的服务器地址", "url", config.ServerURL)
	} else {
		// 从配置中读取完整的服务器URL
		serverURL := cm.configManager.GetString("server_url")
		if serverURL != "" {
			config.ServerURL = serverURL
			cm.logger.Info("使用配置的服务器URL", "url", config.ServerURL)
		} else {
			// 使用默认的本地WebSocket服务器
			config.ServerURL = "ws://localhost:9000/ws"
			cm.logger.Info("使用默认的服务器URL", "url", config.ServerURL)
		}
	}

	// 从配置中读取心跳间隔
	heartbeatInterval := cm.configManager.GetString("heartbeat_interval")
	if heartbeatInterval != "" {
		if interval, err := time.ParseDuration(heartbeatInterval); err == nil {
			config.HeartbeatInterval = interval
		}
	}

	// 从配置中读取重连间隔
	reconnectInterval := cm.configManager.GetString("reconnect_interval")
	if reconnectInterval != "" {
		if interval, err := time.ParseDuration(reconnectInterval); err == nil {
			config.ReconnectInterval = interval
		}
	}

	// 从配置中读取最大重连次数
	maxReconnectAttempts := cm.configManager.GetInt("max_reconnect_attempts")
	if maxReconnectAttempts > 0 {
		config.MaxReconnectAttempts = maxReconnectAttempts
	}

	// 从配置中读取安全配置
	securityConfig := cm.configManager.GetStringMap("comm_security")
	if securityConfig != nil {
		// TLS配置
		if enableTLS, ok := securityConfig["enable_tls"].(bool); ok {
			config.Security.EnableTLS = enableTLS
		}
		if verifyServerCert, ok := securityConfig["verify_server_cert"].(bool); ok {
			config.Security.VerifyServerCert = verifyServerCert
		}
		if clientCertFile, ok := securityConfig["client_cert_file"].(string); ok {
			config.Security.ClientCertFile = clientCertFile
		}
		if clientKeyFile, ok := securityConfig["client_key_file"].(string); ok {
			config.Security.ClientKeyFile = clientKeyFile
		}
		if caCertFile, ok := securityConfig["ca_cert_file"].(string); ok {
			config.Security.CACertFile = caCertFile
		}

		// 加密配置
		if enableEncryption, ok := securityConfig["enable_encryption"].(bool); ok {
			config.Security.EnableEncryption = enableEncryption
		}
		if encryptionKey, ok := securityConfig["encryption_key"].(string); ok {
			config.Security.EncryptionKey = encryptionKey
		}

		// 认证配置
		if enableAuth, ok := securityConfig["enable_auth"].(bool); ok {
			config.Security.EnableAuth = enableAuth
		}
		if authToken, ok := securityConfig["auth_token"].(string); ok {
			config.Security.AuthToken = authToken
		}
		if authType, ok := securityConfig["auth_type"].(string); ok {
			config.Security.AuthType = authType
		}
		if username, ok := securityConfig["username"].(string); ok {
			config.Security.Username = username
		}
		if password, ok := securityConfig["password"].(string); ok {
			config.Security.Password = password
		}

		// 压缩配置
		if enableCompression, ok := securityConfig["enable_compression"].(bool); ok {
			config.Security.EnableCompression = enableCompression
		}
		if compressionLevel, ok := securityConfig["compression_level"].(int); ok {
			config.Security.CompressionLevel = compressionLevel
		}
		if compressionThreshold, ok := securityConfig["compression_threshold"].(int); ok {
			config.Security.CompressionThreshold = compressionThreshold
		}
	}

	// 创建日志适配器
	logAdapter := logger.NewLogger("comm-client", hclog.Info)

	// 创建通讯管理器
	cm.manager = comm.NewManager(config, logAdapter)

	// 设置客户端信息
	clientInfo := cm.getClientInfo()
	cm.manager.SetClientInfo(clientInfo)

	// 注册消息处理函数
	cm.registerDefaultHandlers()

	cm.initialized = true
	return nil
}

// Connect 连接到服务器
func (cm *CommManager) Connect() error {
	if !cm.initialized {
		if err := cm.Init(); err != nil {
			return err
		}
	}

	if cm.connected {
		return nil
	}

	cm.logger.Info("连接到服务器", "url", cm.manager.GetConfig().ServerURL)

	// 连接到服务器
	err := cm.manager.Connect()
	if err != nil {
		cm.logger.Error("连接服务器失败", "error", err)
		return fmt.Errorf("连接服务器失败: %w", err)
	}

	cm.connected = true
	cm.logger.Info("已连接到服务器")
	return nil
}

// Disconnect 断开连接
func (cm *CommManager) Disconnect() {
	if !cm.connected {
		return
	}

	cm.logger.Info("断开与服务器的连接")
	cm.manager.Disconnect()
	cm.connected = false
}

// IsConnected 检查是否已连接
func (cm *CommManager) IsConnected() bool {
	// 如果未初始化，返回false
	if !cm.initialized || cm.manager == nil {
		return false
	}

	// 返回实际的连接状态
	return cm.manager.IsConnected()
}

// RegisterHandler 注册消息处理函数
func (cm *CommManager) RegisterHandler(msgType comm.MessageType, handler comm.MessageHandler) {
	cm.routeMutex.Lock()
	defer cm.routeMutex.Unlock()

	if _, ok := cm.routeTable[msgType]; !ok {
		cm.routeTable[msgType] = make([]comm.MessageHandler, 0)
	}

	cm.routeTable[msgType] = append(cm.routeTable[msgType], handler)

	// 如果已初始化，则向通讯管理器注册处理函数
	if cm.initialized {
		cm.manager.RegisterHandler(msgType, handler)
	}
}

// SendMessage 发送消息
func (cm *CommManager) SendMessage(msgType comm.MessageType, payload map[string]interface{}) error {
	if !cm.initialized {
		return fmt.Errorf("通讯管理器未初始化")
	}

	if !cm.connected {
		return fmt.Errorf("未连接到服务器")
	}

	cm.manager.SendMessage(msgType, payload)
	return nil
}

// SendCommand 发送命令消息
func (cm *CommManager) SendCommand(command string, params map[string]interface{}) error {
	if !cm.initialized {
		return fmt.Errorf("通讯管理器未初始化")
	}

	if !cm.connected {
		return fmt.Errorf("未连接到服务器")
	}

	cm.manager.SendCommand(command, params)
	return nil
}

// SendData 发送数据消息
func (cm *CommManager) SendData(dataType string, data interface{}) error {
	if !cm.initialized {
		return fmt.Errorf("通讯管理器未初始化")
	}

	if !cm.connected {
		return fmt.Errorf("未连接到服务器")
	}

	cm.manager.SendData(dataType, data)
	return nil
}

// SendEvent 发送事件消息
func (cm *CommManager) SendEvent(eventType string, details map[string]interface{}) error {
	if !cm.initialized {
		return fmt.Errorf("通讯管理器未初始化")
	}

	if !cm.connected {
		return fmt.Errorf("未连接到服务器")
	}

	cm.manager.SendEvent(eventType, details)
	return nil
}

// getClientInfo 获取客户端信息
func (cm *CommManager) getClientInfo() map[string]interface{} {
	hostname, _ := os.Hostname()

	return map[string]interface{}{
		"client_id":    hostname,
		"version":      "1.0.0", // 应该从配置或常量中获取
		"os":           cm.configManager.GetString("os"),
		"arch":         cm.configManager.GetString("arch"),
		"connect_time": time.Now().Format(time.RFC3339),
	}
}

// registerDefaultHandlers 注册默认的消息处理函数
func (cm *CommManager) registerDefaultHandlers() {
	// 注册命令消息处理函数
	cm.manager.RegisterHandler(comm.MessageTypeCommand, cm.handleCommand)

	// 注册数据消息处理函数
	cm.manager.RegisterHandler(comm.MessageTypeData, cm.handleData)

	// 注册事件消息处理函数
	cm.manager.RegisterHandler(comm.MessageTypeEvent, cm.handleEvent)
}

// handleCommand 处理命令消息
func (cm *CommManager) handleCommand(msg *comm.Message) {
	command, ok := msg.Payload["command"].(string)
	if !ok {
		cm.logger.Warn("收到无效的命令消息")
		return
	}

	params, _ := msg.Payload["params"].(map[string]interface{})
	cm.logger.Info("收到命令", "command", command)

	// 根据命令类型路由到对应的插件
	switch command {
	case "execute_plugin":
		cm.handlePluginCommand(params, msg.ID)
	case "restart":
		cm.handleRestartCommand(params)
	case "update":
		cm.handleUpdateCommand(params)
	default:
		cm.logger.Info("未知命令", "command", command)
	}
}

// handleData 处理数据消息
func (cm *CommManager) handleData(msg *comm.Message) {
	dataType, ok := msg.Payload["type"].(string)
	if !ok {
		cm.logger.Warn("收到无效的数据消息")
		return
	}

	cm.logger.Info("收到数据", "type", dataType)
}

// handleEvent 处理事件消息
func (cm *CommManager) handleEvent(msg *comm.Message) {
	eventType, ok := msg.Payload["event"].(string)
	if !ok {
		cm.logger.Warn("收到无效的事件消息")
		return
	}

	cm.logger.Info("收到事件", "event", eventType)
}

// handlePluginCommand 处理插件命令
func (cm *CommManager) handlePluginCommand(params map[string]interface{}, messageID string) {
	pluginName, ok := params["plugin"].(string)
	if !ok {
		cm.logger.Warn("缺少插件名称")
		return
	}

	action, ok := params["action"].(string)
	if !ok {
		cm.logger.Warn("缺少操作名称")
		return
	}

	actionParams, _ := params["params"].(map[string]interface{})
	if actionParams == nil {
		actionParams = make(map[string]interface{})
	}

	// 获取插件管理器
	pluginManager := GetPluginManager()
	if pluginManager == nil {
		cm.logger.Error("无法获取插件管理器")
		return
	}

	// 获取插件
	plugin, ok := pluginManager.GetPlugin(pluginName)
	if !ok {
		cm.logger.Warn("插件未找到", "plugin", pluginName)
		return
	}

	// 执行插件操作
	cm.logger.Info("执行插件操作", "plugin", pluginName, "action", action)
	result, err := plugin.Execute(action, actionParams)
	if err != nil {
		cm.logger.Error("执行插件操作失败", "plugin", pluginName, "action", action, "error", err)
		return
	}

	// 发送响应
	cm.SendResponse(messageID, true, result, "")
}

// handleRestartCommand 处理重启命令
func (cm *CommManager) handleRestartCommand(params map[string]interface{}) {
	cm.logger.Info("收到重启命令")
	// TODO: 实现重启逻辑
}

// handleUpdateCommand 处理更新命令
func (cm *CommManager) handleUpdateCommand(params map[string]interface{}) {
	cm.logger.Info("收到更新命令")
	// TODO: 实现更新逻辑
}

// SendResponse 发送响应消息
func (cm *CommManager) SendResponse(requestID string, success bool, data interface{}, errorMsg string) error {
	if !cm.initialized {
		return fmt.Errorf("通讯管理器未初始化")
	}

	if !cm.connected {
		return fmt.Errorf("未连接到服务器")
	}

	cm.manager.SendResponse(requestID, success, data, errorMsg)
	return nil
}

// GetManager 获取底层通讯管理器
func (cm *CommManager) GetManager() *comm.Manager {
	return cm.manager
}

// GetConfig 获取通讯配置
func (cm *CommManager) GetConfig() comm.ConnectionConfig {
	return cm.manager.GetConfig()
}

// GetMetrics 获取通讯模块的指标
func (cm *CommManager) GetMetrics() map[string]interface{} {
	if cm.manager == nil {
		return map[string]interface{}{
			"error":     "通讯管理器未初始化",
			"connected": false,
			"status":    "未初始化",
			"timestamp": time.Now().Format(time.RFC3339),
		}
	}

	// 获取真实的通讯指标
	now := time.Now()
	metrics := cm.manager.GetMetrics()

	// 添加管理器级别的指标
	metrics["status"] = cm.GetState()
	metrics["connected"] = cm.IsConnected()
	metrics["timestamp"] = now.Format(time.RFC3339)

	// 添加连接持续时间
	if cm.IsConnected() && metrics["last_connect_time"] != nil {
		if lastConnectTime, ok := metrics["last_connect_time"].(int64); ok {
			connectTime := time.Unix(0, lastConnectTime*int64(time.Millisecond))
			metrics["connection_duration"] = int64(time.Since(connectTime).Milliseconds())
		}
	}

	return metrics
}

// GetMetricsReport 获取通讯模块的指标报告
func (cm *CommManager) GetMetricsReport() string {
	if cm.manager == nil {
		return "通讯模块未初始化"
	}

	return cm.manager.GetClient().GetMetricsReport()
}

// GetState 获取当前状态
func (cm *CommManager) GetState() string {
	if cm.manager == nil {
		return "未初始化"
	}

	// 获取真实的连接状态
	state := cm.manager.GetState()

	// 直接使用状态的字符串表示
	return state.String()
}

// GetLogs 获取通讯日志
func (cm *CommManager) GetLogs(limit int, offset int, level string) []interface{} {
	if cm.manager == nil {
		return []interface{}{}
	}

	// TODO: 实现日志获取逻辑
	// 这里应该从日志系统中获取通讯日志
	// 目前返回模拟数据
	logs := make([]interface{}, 0)
	for i := 0; i < 10; i++ {
		logs = append(logs, map[string]interface{}{
			"timestamp": time.Now().Add(-time.Duration(i) * time.Minute).Format(time.RFC3339),
			"level":     "info",
			"message":   fmt.Sprintf("通讯日志 %d", i),
			"source":    "comm-manager",
		})
	}

	return logs
}

// 测试历史记录
var testHistory []map[string]interface{}
var testHistoryMutex sync.RWMutex

// recordTestHistory 记录测试历史
func (cm *CommManager) recordTestHistory(testType string, data map[string]interface{}) {
	testHistoryMutex.Lock()
	defer testHistoryMutex.Unlock()

	// 创建测试记录
	record := map[string]interface{}{
		"type":      testType,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 添加测试数据
	for k, v := range data {
		record[k] = v
	}

	// 添加到历史记录
	testHistory = append(testHistory, record)

	// 限制历史记录数量
	if len(testHistory) > 100 {
		testHistory = testHistory[len(testHistory)-100:]
	}
}

// GetTestHistory 获取测试历史记录
func (cm *CommManager) GetTestHistory() []interface{} {
	testHistoryMutex.RLock()
	defer testHistoryMutex.RUnlock()

	// 复制历史记录
	result := make([]interface{}, len(testHistory))
	for i, record := range testHistory {
		result[i] = record
	}

	return result
}

// TestConnection 测试通讯连接
func (cm *CommManager) TestConnection(serverURL string, timeout time.Duration) (bool, error) {
	// 记录测试开始时间
	startTime := time.Now()

	// 如果通讯管理器已初始化，使用当前连接
	if cm.initialized {
		// 检查当前连接状态
		if cm.connected {
			// 记录测试历史
			cm.recordTestHistory("connection", map[string]interface{}{
				"server_url": serverURL,
				"timeout":    timeout.String(),
				"success":    true,
				"duration":   time.Since(startTime).String(),
			})
			return true, nil
		}

		// 尝试连接
		err := cm.Connect()
		if err != nil {
			// 记录测试历史
			cm.recordTestHistory("connection", map[string]interface{}{
				"server_url": serverURL,
				"timeout":    timeout.String(),
				"success":    false,
				"error":      err.Error(),
				"duration":   time.Since(startTime).String(),
			})
			return false, err
		}

		// 记录测试历史
		cm.recordTestHistory("connection", map[string]interface{}{
			"server_url": serverURL,
			"timeout":    timeout.String(),
			"success":    true,
			"duration":   time.Since(startTime).String(),
		})
		return true, nil
	}

	// 创建临时配置
	config := comm.DefaultConfig()
	config.ServerURL = serverURL
	config.HandshakeTimeout = timeout

	// 创建临时客户端
	client := comm.NewClient(config, logger.NewLogger("test-client", hclog.Info))

	// 连接到服务器
	err := client.Connect()
	if err != nil {
		// 记录测试历史
		cm.recordTestHistory("connection", map[string]interface{}{
			"server_url": serverURL,
			"timeout":    timeout.String(),
			"success":    false,
			"error":      err.Error(),
			"duration":   time.Since(startTime).String(),
		})
		return false, err
	}

	// 等待一段时间，确保连接稳定
	time.Sleep(500 * time.Millisecond)

	// 检查连接状态
	if !client.IsConnected() {
		// 记录测试历史
		cm.recordTestHistory("connection", map[string]interface{}{
			"server_url": serverURL,
			"timeout":    timeout.String(),
			"success":    false,
			"error":      "连接不稳定",
			"duration":   time.Since(startTime).String(),
		})
		return false, fmt.Errorf("连接不稳定")
	}

	// 断开连接
	client.Disconnect()

	// 记录测试历史
	cm.recordTestHistory("connection", map[string]interface{}{
		"server_url": serverURL,
		"timeout":    timeout.String(),
		"success":    true,
		"duration":   time.Since(startTime).String(),
	})

	return true, nil
}

// SendMessageAndWaitResponse 发送消息并等待响应
func (cm *CommManager) SendMessageAndWaitResponse(msgType comm.MessageType, payload map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	if !cm.initialized {
		return nil, fmt.Errorf("通讯管理器未初始化")
	}

	// 记录测试开始时间
	startTime := time.Now()

	// 如果未连接，并且请求中包含mock=true，使用模拟响应
	if !cm.connected {
		if mock, ok := payload["mock"].(bool); ok && mock {
			// 创建模拟响应
			response := createMockResponse(cm.messageTypeToString(msgType), payload)

			// 记录测试历史
			cm.recordTestHistory("send-receive", map[string]interface{}{
				"message_type": cm.messageTypeToString(msgType),
				"payload":      payload,
				"response":     response,
				"success":      true,
				"duration":     time.Since(startTime).String(),
				"mock":         true,
			})

			// 添加一些延迟，模拟网络延迟
			time.Sleep(200 * time.Millisecond)

			return response, nil
		}

		return nil, fmt.Errorf("未连接到服务器")
	}

	// 创建响应通道
	responseChan := make(chan map[string]interface{}, 1)
	errorChan := make(chan error, 1)

	// 生成请求ID
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())
	payload["request_id"] = requestID

	// 创建响应处理函数
	responseHandler := func(msg *comm.Message) {
		// 检查是否是对应的响应
		if respID, ok := msg.Payload["request_id"].(string); ok && respID == requestID {
			// 提取响应数据
			response := make(map[string]interface{})
			for k, v := range msg.Payload {
				response[k] = v
			}
			responseChan <- response
		}
	}

	// 注册临时响应处理函数
	cm.manager.RegisterHandler(comm.MessageTypeResponse, responseHandler)

	// 确保在函数返回时取消注册处理函数
	defer func() {
		// 尝试取消注册处理函数
		err := cm.manager.UnregisterHandler(comm.MessageTypeResponse, responseHandler)
		if err != nil {
			cm.logger.Warn("取消注册响应处理函数失败", "error", err)
		}
	}()

	// 发送消息
	cm.manager.SendMessage(msgType, payload)
	cm.logger.Debug("已发送消息", "type", msgType, "request_id", requestID)

	// 等待响应或超时
	select {
	case response := <-responseChan:
		// 记录测试历史
		cm.recordTestHistory("send-receive", map[string]interface{}{
			"message_type": cm.messageTypeToString(msgType),
			"payload":      payload,
			"response":     response,
			"success":      true,
			"duration":     time.Since(startTime).String(),
		})
		return response, nil
	case err := <-errorChan:
		// 记录测试历史
		cm.recordTestHistory("send-receive", map[string]interface{}{
			"message_type": cm.messageTypeToString(msgType),
			"payload":      payload,
			"success":      false,
			"error":        err.Error(),
			"duration":     time.Since(startTime).String(),
		})
		return nil, err
	case <-time.After(timeout):
		// 记录测试历史
		cm.recordTestHistory("send-receive", map[string]interface{}{
			"message_type": cm.messageTypeToString(msgType),
			"payload":      payload,
			"success":      false,
			"error":        "超时",
			"duration":     time.Since(startTime).String(),
		})
		return nil, fmt.Errorf("等待响应超时")
	}
}

// createMockResponse 创建模拟响应
func createMockResponse(messageType string, payload map[string]interface{}) map[string]interface{} {
	response := make(map[string]interface{})

	// 复制请求ID
	if requestID, ok := payload["request_id"].(string); ok {
		response["request_id"] = requestID
	} else {
		response["request_id"] = fmt.Sprintf("resp-%d", time.Now().UnixNano())
	}

	// 根据消息类型创建不同的响应
	switch messageType {
	case "command":
		// 获取命令
		command, _ := payload["command"].(string)
		response["success"] = true
		response["command"] = command

		// 根据不同命令返回不同响应
		switch command {
		case "ping":
			response["data"] = "pong"
		case "echo":
			if data, ok := payload["data"]; ok {
				response["data"] = data
			} else {
				response["data"] = "echo"
			}
		case "status":
			response["data"] = map[string]interface{}{
				"status":  "running",
				"uptime":  "3h 24m 15s",
				"version": "1.0.0",
			}
		default:
			response["data"] = fmt.Sprintf("执行命令: %s", command)
		}

	case "data":
		// 获取数据类型
		dataType, _ := payload["type"].(string)
		response["success"] = true
		response["type"] = dataType

		// 根据不同数据类型返回不同响应
		switch dataType {
		case "user":
			response["data"] = map[string]interface{}{
				"id":   1001,
				"name": "测试用户",
				"role": "admin",
			}
		case "device":
			response["data"] = map[string]interface{}{
				"id":     "DEV-2001",
				"name":   "测试设备",
				"status": "online",
			}
		default:
			response["data"] = fmt.Sprintf("接收数据: %s", dataType)
		}

	case "event":
		// 获取事件类型
		eventType, _ := payload["event"].(string)
		response["success"] = true
		response["event"] = eventType
		response["message"] = fmt.Sprintf("事件已处理: %s", eventType)

	default:
		response["success"] = true
		response["message"] = "未知消息类型，已接收"
	}

	// 添加时间戳
	response["timestamp"] = time.Now().Format(time.RFC3339)

	return response
}

// messageTypeToString 将消息类型转换为字符串
func (cm *CommManager) messageTypeToString(msgType comm.MessageType) string {
	switch msgType {
	case comm.MessageTypeConnect:
		return "connect"
	case comm.MessageTypeHeartbeat:
		return "heartbeat"
	case comm.MessageTypeCommand:
		return "command"
	case comm.MessageTypeData:
		return "data"
	case comm.MessageTypeEvent:
		return "event"
	case comm.MessageTypeResponse:
		return "response"
	case comm.MessageTypeAck:
		return "ack"
	default:
		return fmt.Sprintf("unknown(%d)", msgType)
	}
}

// TestEncryption 测试通讯加密
func (cm *CommManager) TestEncryption(data []byte, encryptionKey string) ([]byte, []byte, error) {
	// 记录测试开始时间
	startTime := time.Now()

	// 使用AES加密
	encryptedData, err := comm.EncryptAES(data, []byte(encryptionKey))
	if err != nil {
		// 记录测试历史
		cm.recordTestHistory("encryption", map[string]interface{}{
			"data_size":      len(data),
			"encryption_key": encryptionKey,
			"success":        false,
			"error":          err.Error(),
			"duration":       time.Since(startTime).String(),
		})
		return nil, nil, err
	}

	// 使用AES解密
	decryptedData, err := comm.DecryptAES(encryptedData, []byte(encryptionKey))
	if err != nil {
		// 记录测试历史
		cm.recordTestHistory("encryption", map[string]interface{}{
			"data_size":      len(data),
			"encryption_key": encryptionKey,
			"success":        false,
			"error":          err.Error(),
			"duration":       time.Since(startTime).String(),
		})
		return nil, nil, err
	}

	// 记录测试历史
	cm.recordTestHistory("encryption", map[string]interface{}{
		"data_size":        len(data),
		"encryption_key":   encryptionKey,
		"encrypted_size":   len(encryptedData),
		"decrypted_size":   len(decryptedData),
		"success":          true,
		"duration":         time.Since(startTime).String(),
		"encryption_ratio": float64(len(encryptedData)) / float64(len(data)),
	})

	return encryptedData, decryptedData, nil
}

// TestCompression 测试通讯压缩
func (cm *CommManager) TestCompression(data []byte, compressionLevel int) ([]byte, []byte, error) {
	// 记录测试开始时间
	startTime := time.Now()

	// 压缩数据
	compressedData, err := comm.CompressData(data, compressionLevel)
	if err != nil {
		// 记录测试历史
		cm.recordTestHistory("compression", map[string]interface{}{
			"data_size":         len(data),
			"compression_level": compressionLevel,
			"success":           false,
			"error":             err.Error(),
			"duration":          time.Since(startTime).String(),
		})
		return nil, nil, err
	}

	// 解压缩数据
	decompressedData, err := comm.DecompressData(compressedData)
	if err != nil {
		// 记录测试历史
		cm.recordTestHistory("compression", map[string]interface{}{
			"data_size":         len(data),
			"compression_level": compressionLevel,
			"success":           false,
			"error":             err.Error(),
			"duration":          time.Since(startTime).String(),
		})
		return nil, nil, err
	}

	// 记录测试历史
	cm.recordTestHistory("compression", map[string]interface{}{
		"data_size":         len(data),
		"compression_level": compressionLevel,
		"compressed_size":   len(compressedData),
		"decompressed_size": len(decompressedData),
		"success":           true,
		"duration":          time.Since(startTime).String(),
		"compression_ratio": float64(len(compressedData)) / float64(len(data)),
	})

	return compressedData, decompressedData, nil
}

// TestPerformance 测试通讯性能
func (cm *CommManager) TestPerformance(messageCount int, messageSize int, enableEncryption bool, encryptionKey string, enableCompression bool, compressionLevel int) (map[string]interface{}, error) {
	// 记录测试开始时间
	startTime := time.Now()

	// 创建测试数据
	testData := make([]byte, messageSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// 性能测试结果
	result := map[string]interface{}{
		"message_count":      messageCount,
		"message_size":       messageSize,
		"enable_encryption":  enableEncryption,
		"enable_compression": enableCompression,
		"compression_level":  compressionLevel,
	}

	// 测试发送性能
	sendStartTime := time.Now()
	totalSendTime := int64(0)
	totalSendSize := int64(0)
	totalSendCompressedSize := int64(0)
	totalSendEncryptedSize := int64(0)

	for i := 0; i < messageCount; i++ {
		// 使用测试数据
		data := testData

		// 记录原始大小
		totalSendSize += int64(len(data))

		// 压缩数据
		if enableCompression {
			compressedData, err := comm.CompressData(data, compressionLevel)
			if err != nil {
				// 记录测试历史
				cm.recordTestHistory("performance", map[string]interface{}{
					"message_count":      messageCount,
					"message_size":       messageSize,
					"enable_encryption":  enableEncryption,
					"enable_compression": enableCompression,
					"compression_level":  compressionLevel,
					"success":            false,
					"error":              err.Error(),
					"duration":           time.Since(startTime).String(),
				})
				return nil, err
			}
			data = compressedData
			totalSendCompressedSize += int64(len(data))
		}

		// 加密数据
		if enableEncryption {
			encryptedData, err := comm.EncryptAES(data, []byte(encryptionKey))
			if err != nil {
				// 记录测试历史
				cm.recordTestHistory("performance", map[string]interface{}{
					"message_count":      messageCount,
					"message_size":       messageSize,
					"enable_encryption":  enableEncryption,
					"enable_compression": enableCompression,
					"compression_level":  compressionLevel,
					"success":            false,
					"error":              err.Error(),
					"duration":           time.Since(startTime).String(),
				})
				return nil, err
			}
			data = encryptedData
			totalSendEncryptedSize += int64(len(data))
		}

		// 记录发送时间
		totalSendTime += time.Since(sendStartTime).Nanoseconds()
	}

	// 计算发送性能指标
	sendDuration := time.Since(sendStartTime)
	result["send_duration"] = sendDuration.String()
	result["send_throughput"] = float64(messageCount) / sendDuration.Seconds()
	result["send_size"] = totalSendSize
	if enableCompression {
		result["send_compressed_size"] = totalSendCompressedSize
		result["send_compression_ratio"] = float64(totalSendCompressedSize) / float64(totalSendSize)
	}
	if enableEncryption {
		result["send_encrypted_size"] = totalSendEncryptedSize
		if enableCompression {
			result["send_encryption_ratio"] = float64(totalSendEncryptedSize) / float64(totalSendCompressedSize)
		} else {
			result["send_encryption_ratio"] = float64(totalSendEncryptedSize) / float64(totalSendSize)
		}
	}

	// 记录测试历史
	cm.recordTestHistory("performance", map[string]interface{}{
		"message_count":      messageCount,
		"message_size":       messageSize,
		"enable_encryption":  enableEncryption,
		"enable_compression": enableCompression,
		"compression_level":  compressionLevel,
		"success":            true,
		"duration":           time.Since(startTime).String(),
		"result":             result,
	})

	return result, nil
}
