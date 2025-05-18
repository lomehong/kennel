package webconsole

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lomehong/kennel/pkg/comm"
)

// LogEntry 表示日志条目
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// SystemEvent 表示系统事件
type SystemEvent struct {
	Timestamp string                 `json:"timestamp"`
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// getPlugins 获取所有插件
func (c *Console) getPlugins(ctx *gin.Context) {
	pluginManager := c.app.GetPluginManager()
	if pluginManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "插件管理器未初始化",
		})
		return
	}

	plugins := pluginManager.GetAllPlugins()
	result := make([]map[string]interface{}, 0, len(plugins))

	for id, plugin := range plugins {
		info := plugin.GetInfo()

		// 获取插件状态
		status := pluginManager.GetPluginStatus(id)
		if status == "" || status == "unknown" {
			status = "running" // 默认状态为运行中
		}

		// 检查插件是否启用
		enabled := pluginManager.IsPluginEnabled(id)

		// 构建描述信息
		description := info.Description
		if description == "" {
			description = info.Name + " 插件"
		}

		// 构建插件信息
		result = append(result, map[string]interface{}{
			"id":          id,
			"name":        info.Name,
			"version":     info.Version,
			"description": description,
			"status":      status,
			"enabled":     enabled,
			"actions":     info.SupportedActions,
		})
	}

	// 如果没有插件，记录警告日志
	if len(result) == 0 {
		c.logger.Warn("没有找到插件")
	}

	// 直接返回插件数组，与前端期望的数据结构保持一致
	ctx.JSON(http.StatusOK, result)
}

// getPlugin 获取插件详情
func (c *Console) getPlugin(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "插件ID不能为空",
		})
		return
	}

	pluginManager := c.app.GetPluginManager()
	if pluginManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "插件管理器未初始化",
		})
		return
	}

	plugin, exists := pluginManager.GetPlugin(id)
	if !exists {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "插件不存在",
		})
		return
	}

	info := plugin.GetInfo()
	ctx.JSON(http.StatusOK, gin.H{
		"id":                id,
		"name":              info.Name,
		"version":           info.Version,
		"description":       info.Description,
		"supported_actions": info.SupportedActions,
		"status":            pluginManager.GetPluginStatus(id),
		"enabled":           pluginManager.IsPluginEnabled(id),
		"config":            pluginManager.GetPluginConfig(id),
	})
}

// updatePluginStatus 更新插件状态
func (c *Console) updatePluginStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "插件ID不能为空",
		})
		return
	}

	var request struct {
		Enabled bool `json:"enabled"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求参数",
		})
		return
	}

	pluginManager := c.app.GetPluginManager()
	if pluginManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "插件管理器未初始化",
		})
		return
	}

	_, exists := pluginManager.GetPlugin(id)
	if !exists {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "插件不存在",
		})
		return
	}

	var err error
	if request.Enabled {
		err = pluginManager.EnablePlugin(id)
	} else {
		err = pluginManager.DisablePlugin(id)
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("更新插件状态失败: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"id":      id,
		"enabled": request.Enabled,
		"status":  pluginManager.GetPluginStatus(id),
	})
}

// updatePluginConfig 更新插件配置
func (c *Console) updatePluginConfig(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "插件ID不能为空",
		})
		return
	}

	var request struct {
		Config map[string]interface{} `json:"config"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求参数",
		})
		return
	}

	pluginManager := c.app.GetPluginManager()
	if pluginManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "插件管理器未初始化",
		})
		return
	}

	_, exists := pluginManager.GetPlugin(id)
	if !exists {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "插件不存在",
		})
		return
	}

	err := pluginManager.UpdatePluginConfig(id, request.Config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("更新插件配置失败: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"id":     id,
		"config": pluginManager.GetPluginConfig(id),
	})
}

// getPluginLogs 获取插件日志
func (c *Console) getPluginLogs(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "插件ID不能为空",
		})
		return
	}

	limit := ctx.DefaultQuery("limit", "100")
	offset := ctx.DefaultQuery("offset", "0")
	level := ctx.DefaultQuery("level", "")

	pluginManager := c.app.GetPluginManager()
	if pluginManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "插件管理器未初始化",
		})
		return
	}

	_, exists := pluginManager.GetPlugin(id)
	if !exists {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "插件不存在",
		})
		return
	}

	logs, err := pluginManager.GetPluginLogs(id, limit, offset, level)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取插件日志失败: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"id":     id,
		"logs":   logs,
		"limit":  limit,
		"offset": offset,
		"level":  level,
	})
}

// getMetrics 获取所有指标
func (c *Console) getMetrics(ctx *gin.Context) {
	// 获取通讯指标
	commMetrics := c.getCommMetricsData()

	// 获取系统指标
	systemMetrics := c.getSystemMetricsData()

	ctx.JSON(http.StatusOK, gin.H{
		"comm":   commMetrics,
		"system": systemMetrics,
		"time":   time.Now().Format(time.RFC3339),
	})
}

// getCommMetrics 获取通讯指标
func (c *Console) getCommMetrics(ctx *gin.Context) {
	metrics := c.getCommMetricsData()
	ctx.JSON(http.StatusOK, metrics)
}

// getCommMetricsData 获取通讯指标数据
func (c *Console) getCommMetricsData() map[string]interface{} {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		return map[string]interface{}{
			"error": "通讯管理器未初始化",
		}
	}

	return commManager.GetMetrics()
}

// getSystemMetrics 获取系统指标
func (c *Console) getSystemMetrics(ctx *gin.Context) {
	metrics := c.getSystemMetricsData()
	ctx.JSON(http.StatusOK, metrics)
}

// getSystemMetricsData 获取系统指标数据
func (c *Console) getSystemMetricsData() map[string]interface{} {
	// 获取系统监控器
	systemMonitor := c.app.GetSystemMonitor()
	if systemMonitor == nil {
		c.logger.Error("系统监控器未初始化")
		return map[string]interface{}{
			"error": "系统监控器未初始化",
		}
	}

	// 获取系统指标
	metrics, err := systemMonitor.GetSystemMetrics()
	if err != nil {
		c.logger.Error("获取系统指标失败", "error", err)
		return map[string]interface{}{
			"error": "获取系统指标失败: " + err.Error(),
		}
	}

	return metrics
}

// getSystemStatus 获取系统状态
func (c *Console) getSystemStatus(ctx *gin.Context) {
	// 获取系统监控器
	systemMonitor := c.app.GetSystemMonitor()
	if systemMonitor == nil {
		c.logger.Error("系统监控器未初始化")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "系统监控器未初始化",
		})
		return
	}

	// 获取系统状态
	status, err := systemMonitor.GetSystemStatus()
	if err != nil {
		c.logger.Error("获取系统状态失败", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取系统状态失败: " + err.Error(),
		})
		return
	}

	// 获取插件信息
	status["plugins"] = func() map[string]interface{} {
		pluginManager := c.app.GetPluginManager()
		if pluginManager == nil {
			return map[string]interface{}{
				"total":    0,
				"enabled":  0,
				"disabled": 0,
			}
		}

		// 获取所有插件
		plugins := pluginManager.GetAllPlugins()
		enabledCount := pluginManager.GetEnabledPluginCount()

		return map[string]interface{}{
			"total":    len(plugins),
			"enabled":  enabledCount,
			"disabled": len(plugins) - enabledCount,
		}
	}()

	// 获取通讯状态
	status["comm"] = func() map[string]interface{} {
		commManager := c.app.GetCommManager()
		if commManager == nil {
			return map[string]interface{}{
				"connected":    false,
				"status":       "disconnected",
				"last_connect": "",
			}
		}

		metrics := commManager.GetMetrics()
		lastConnectTime, ok := metrics["last_connect_time"].(int64)
		lastConnectTimeStr := ""
		if ok && lastConnectTime > 0 {
			lastConnectTimeStr = time.Unix(0, lastConnectTime*int64(time.Millisecond)).Format(time.RFC3339)
		}

		return map[string]interface{}{
			"connected":    commManager.IsConnected(),
			"status":       commManager.GetState(),
			"last_connect": lastConnectTimeStr,
		}
	}()

	// 添加时间戳
	status["timestamp"] = time.Now().Format(time.RFC3339)

	ctx.JSON(http.StatusOK, status)
}

// getSystemResources 获取系统资源
func (c *Console) getSystemResources(ctx *gin.Context) {
	// 获取系统监控器
	systemMonitor := c.app.GetSystemMonitor()
	if systemMonitor == nil {
		c.logger.Error("系统监控器未初始化")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "系统监控器未初始化",
		})
		return
	}

	// 获取系统资源
	resources, err := systemMonitor.GetSystemResources()
	if err != nil {
		c.logger.Error("获取系统资源失败", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取系统资源失败: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, resources)
}

// getSystemLogs 获取系统日志
func (c *Console) getSystemLogs(ctx *gin.Context) {
	// 获取查询参数
	limitStr := ctx.DefaultQuery("limit", "100")
	offsetStr := ctx.DefaultQuery("offset", "0")
	level := ctx.DefaultQuery("level", "")
	source := ctx.DefaultQuery("source", "")

	// 转换参数
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	// 获取日志管理器
	logManager := c.app.GetLogManager()
	if logManager == nil {
		c.logger.Error("日志管理器未初始化")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "日志管理器未初始化",
		})
		return
	}

	// 获取系统日志
	logs, err := logManager.GetLogs(limit, offset, level, source)
	if err != nil {
		c.logger.Error("获取系统日志失败", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取系统日志失败: " + err.Error(),
		})
		return
	}

	// 如果没有日志，返回空数组
	if logs == nil {
		logs = []interface{}{}
	}

	ctx.JSON(http.StatusOK, logs)
}

// getSystemEvents 获取系统事件
func (c *Console) getSystemEvents(ctx *gin.Context) {
	// 获取查询参数
	limitStr := ctx.DefaultQuery("limit", "100")
	offsetStr := ctx.DefaultQuery("offset", "0")
	eventType := ctx.DefaultQuery("type", "")
	source := ctx.DefaultQuery("source", "")

	// 转换参数
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	// 获取事件管理器
	eventManager := c.app.GetEventManager()
	if eventManager == nil {
		c.logger.Error("事件管理器未初始化")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "事件管理器未初始化",
		})
		return
	}

	// 获取系统事件
	events, err := eventManager.GetEvents(limit, offset, eventType, source)
	if err != nil {
		c.logger.Error("获取系统事件失败", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取系统事件失败: " + err.Error(),
		})
		return
	}

	// 如果没有事件，返回空数组
	if events == nil {
		events = []interface{}{}
	}

	ctx.JSON(http.StatusOK, events)
}

// getConfig 获取配置
func (c *Console) getConfig(ctx *gin.Context) {
	configManager := c.app.GetConfigManager()
	if configManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "配置管理器未初始化",
		})
		return
	}

	config := configManager.GetAllConfig()
	ctx.JSON(http.StatusOK, gin.H{
		"config": config,
	})
}

// updateConfig 更新配置
func (c *Console) updateConfig(ctx *gin.Context) {
	var request struct {
		Config map[string]interface{} `json:"config"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求参数",
		})
		return
	}

	configManager := c.app.GetConfigManager()
	if configManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "配置管理器未初始化",
		})
		return
	}

	err := configManager.UpdateConfig(request.Config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("更新配置失败: %v", err),
		})
		return
	}

	err = configManager.SaveConfig()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("保存配置失败: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "配置已更新",
		"config":  configManager.GetAllConfig(),
	})
}

// resetConfig 重置配置
func (c *Console) resetConfig(ctx *gin.Context) {
	configManager := c.app.GetConfigManager()
	if configManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "配置管理器未初始化",
		})
		return
	}

	err := configManager.ResetConfig()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("重置配置失败: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "配置已重置",
		"config":  configManager.GetAllConfig(),
	})
}

// getCommStatus 获取通讯状态
func (c *Console) getCommStatus(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":    commManager.GetState(),
		"connected": commManager.IsConnected(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// connectComm 连接到服务器
func (c *Console) connectComm(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 如果已经连接，返回成功
	if commManager.IsConnected() {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "已经连接到服务器",
			"status":  commManager.GetState(),
		})
		return
	}

	// 连接到服务器
	err := commManager.Connect()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("连接服务器失败: %v", err),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "成功连接到服务器",
		"status":  commManager.GetState(),
	})
}

// disconnectComm 断开连接
func (c *Console) disconnectComm(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 如果已经断开连接，返回成功
	if !commManager.IsConnected() {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "已经断开连接",
			"status":  commManager.GetState(),
		})
		return
	}

	// 断开连接
	commManager.Disconnect()

	ctx.JSON(http.StatusOK, gin.H{
		"message": "成功断开连接",
		"status":  commManager.GetState(),
	})
}

// getCommConfig 获取通讯配置
func (c *Console) getCommConfig(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 获取通讯配置
	config := commManager.GetConfig()
	if config == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取通讯配置失败",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"config": config,
	})
}

// getCommStats 获取通讯统计信息
func (c *Console) getCommStats(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 获取通讯统计信息
	metrics := commManager.GetMetrics()

	// 添加当前状态信息
	metrics["status"] = commManager.GetState()
	metrics["connected"] = commManager.IsConnected()
	metrics["timestamp"] = time.Now().Format(time.RFC3339)

	ctx.JSON(http.StatusOK, metrics)
}

// getCommLogs 获取通讯日志
func (c *Console) getCommLogs(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 获取查询参数
	limitStr := ctx.DefaultQuery("limit", "100")
	offsetStr := ctx.DefaultQuery("offset", "0")
	level := ctx.DefaultQuery("level", "")

	// 转换参数
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	// 获取通讯日志
	logs := commManager.GetLogs(limit, offset, level)
	if logs == nil {
		logs = []interface{}{}
	}

	// 获取日志总数
	// 注意：这里应该从通讯管理器获取日志总数，但接口中没有提供该方法
	// 因此，我们使用日志数组的长度作为总数
	total := len(logs)

	ctx.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"limit":  limit,
		"offset": offset,
		"level":  level,
		"total":  total,
	})
}

// testCommConnection 测试通讯连接
func (c *Console) testCommConnection(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 获取请求参数
	var req struct {
		ServerURL string `json:"server_url"`
		Timeout   int    `json:"timeout"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求参数: " + err.Error(),
		})
		return
	}

	// 设置默认超时
	if req.Timeout <= 0 {
		req.Timeout = 10
	}

	// 测试连接
	startTime := time.Now()
	success, err := commManager.TestConnection(req.ServerURL, time.Duration(req.Timeout)*time.Second)
	duration := time.Since(startTime)

	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"success":  false,
			"message":  "连接失败: " + err.Error(),
			"duration": duration.String(),
		})
		return
	}

	if !success {
		ctx.JSON(http.StatusOK, gin.H{
			"success":  false,
			"message":  "连接失败",
			"duration": duration.String(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "连接成功",
		"duration": duration.String(),
	})
}

// testCommSendReceive 测试通讯发送和接收
func (c *Console) testCommSendReceive(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 获取请求参数
	var req struct {
		MessageType string                 `json:"message_type"`
		Payload     map[string]interface{} `json:"payload"`
		Timeout     int                    `json:"timeout"`
		UseMock     bool                   `json:"use_mock"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求参数: " + err.Error(),
		})
		return
	}

	// 设置默认超时
	if req.Timeout <= 0 {
		req.Timeout = 10
	}

	// 解析消息类型
	var msgType comm.MessageType
	switch req.MessageType {
	case "command":
		msgType = comm.MessageTypeCommand
	case "data":
		msgType = comm.MessageTypeData
	case "event":
		msgType = comm.MessageTypeEvent
	case "response":
		msgType = comm.MessageTypeResponse
	default:
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的消息类型: " + req.MessageType,
		})
		return
	}

	// 如果未连接且启用了模拟模式，添加mock标志
	if !commManager.IsConnected() && req.UseMock {
		if req.Payload == nil {
			req.Payload = make(map[string]interface{})
		}
		req.Payload["mock"] = true
	}

	// 发送消息并等待响应
	startTime := time.Now()
	response, err := commManager.SendMessageAndWaitResponse(msgType, req.Payload, time.Duration(req.Timeout)*time.Second)
	duration := time.Since(startTime)

	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"success":  false,
			"message":  "发送消息失败: " + err.Error(),
			"duration": duration.String(),
		})
		return
	}

	// 检查是否使用了模拟响应
	message := "发送消息成功"
	if _, ok := response["mock"]; ok {
		message = "使用模拟响应成功"
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  message,
		"response": response,
		"duration": duration.String(),
	})
}

// testCommEncryption 测试通讯加密
func (c *Console) testCommEncryption(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 获取请求参数
	var req struct {
		Data          string `json:"data"`
		EncryptionKey string `json:"encryption_key"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求参数: " + err.Error(),
		})
		return
	}

	// 测试加密和解密
	startTime := time.Now()
	encryptedData, decryptedData, err := commManager.TestEncryption([]byte(req.Data), req.EncryptionKey)
	duration := time.Since(startTime)

	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"success":  false,
			"message":  "加密测试失败: " + err.Error(),
			"duration": duration.String(),
		})
		return
	}

	// 计算加密率
	originalSize := len(req.Data)
	encryptedSize := len(encryptedData)
	ratio := float64(encryptedSize) / float64(originalSize)

	ctx.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        "加密测试成功",
		"original_data":  req.Data,
		"encrypted_data": base64.StdEncoding.EncodeToString(encryptedData),
		"decrypted_data": string(decryptedData),
		"original_size":  originalSize,
		"encrypted_size": encryptedSize,
		"ratio":          ratio,
		"duration":       duration.String(),
	})
}

// testCommCompression 测试通讯压缩
func (c *Console) testCommCompression(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 获取请求参数
	var req struct {
		Data             string `json:"data"`
		CompressionLevel int    `json:"compression_level"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求参数: " + err.Error(),
		})
		return
	}

	// 设置默认压缩级别
	if req.CompressionLevel <= 0 || req.CompressionLevel > 9 {
		req.CompressionLevel = 6
	}

	// 测试压缩和解压缩
	startTime := time.Now()
	compressedData, decompressedData, err := commManager.TestCompression([]byte(req.Data), req.CompressionLevel)
	duration := time.Since(startTime)

	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"success":  false,
			"message":  "压缩测试失败: " + err.Error(),
			"duration": duration.String(),
		})
		return
	}

	// 计算压缩率
	originalSize := len(req.Data)
	compressedSize := len(compressedData)
	ratio := float64(compressedSize) / float64(originalSize)

	ctx.JSON(http.StatusOK, gin.H{
		"success":           true,
		"message":           "压缩测试成功",
		"original_data":     req.Data,
		"compressed_data":   base64.StdEncoding.EncodeToString(compressedData),
		"decompressed_data": string(decompressedData),
		"original_size":     originalSize,
		"compressed_size":   compressedSize,
		"ratio":             ratio,
		"duration":          duration.String(),
	})
}

// testCommPerformance 测试通讯性能
func (c *Console) testCommPerformance(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 获取请求参数
	var req struct {
		MessageCount      int    `json:"message_count"`
		MessageSize       int    `json:"message_size"`
		EnableEncryption  bool   `json:"enable_encryption"`
		EncryptionKey     string `json:"encryption_key"`
		EnableCompression bool   `json:"enable_compression"`
		CompressionLevel  int    `json:"compression_level"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求参数: " + err.Error(),
		})
		return
	}

	// 设置默认值
	if req.MessageCount <= 0 {
		req.MessageCount = 100
	}
	if req.MessageSize <= 0 {
		req.MessageSize = 1024
	}
	if req.CompressionLevel <= 0 || req.CompressionLevel > 9 {
		req.CompressionLevel = 6
	}

	// 执行性能测试
	startTime := time.Now()
	result, err := commManager.TestPerformance(req.MessageCount, req.MessageSize, req.EnableEncryption, req.EncryptionKey, req.EnableCompression, req.CompressionLevel)
	duration := time.Since(startTime)

	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"success":  false,
			"message":  "性能测试失败: " + err.Error(),
			"duration": duration.String(),
		})
		return
	}

	// 添加总测试时间
	result["total_duration"] = duration.String()

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "性能测试成功",
		"result":  result,
	})
}

// getCommTestHistory 获取通讯测试历史记录
func (c *Console) getCommTestHistory(ctx *gin.Context) {
	commManager := c.app.GetCommManager()
	if commManager == nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "通讯管理器未初始化",
		})
		return
	}

	// 获取测试历史记录
	history := commManager.GetTestHistory()
	if history == nil {
		history = []interface{}{}
	}

	ctx.JSON(http.StatusOK, history)
}
