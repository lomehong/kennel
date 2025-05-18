package main

import (
	"fmt"
	"runtime"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/logger"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
)

// AuditModule 是安全审计模块的实现
type AuditModule struct {
	logger      logger.Logger
	config      map[string]interface{}
	logStore    *AuditLogStore
	auditLogger *AuditLogger
}

// NewAuditModule 创建一个新的安全审计模块
func NewAuditModule() pluginLib.Module {
	// 创建日志器
	log := logger.NewLogger("audit-module", hclog.Info)

	// 创建日志存储
	logStore := NewAuditLogStore()

	// 创建模块
	module := &AuditModule{
		logger:   log,
		config:   make(map[string]interface{}),
		logStore: logStore,
	}

	// 创建审计日志记录器
	module.auditLogger = NewAuditLogger(log, logStore, module.config)

	return module
}

// Init 初始化模块
func (m *AuditModule) Init(config map[string]interface{}) error {
	m.config = config

	// 更新审计日志记录器的配置
	m.auditLogger = NewAuditLogger(m.logger, m.logStore, config)

	// 记录初始化事件
	m.auditLogger.LogInitEvent(runtime.GOOS, runtime.GOARCH, "0.1.0")

	return nil
}

// Execute 执行模块操作
func (m *AuditModule) Execute(action string, params map[string]interface{}) (map[string]interface{}, error) {
	m.logger.Info("执行操作", "action", action)

	switch action {
	case "log_event":
		return m.auditLogger.LogEvent(params)
	case "get_logs":
		return m.auditLogger.GetLogs(params)
	case "clear_logs":
		return m.auditLogger.ClearLogs(params)
	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
}

// Shutdown 关闭模块
func (m *AuditModule) Shutdown() error {
	m.logger.Info("关闭安全审计模块")

	// 记录关闭事件
	m.auditLogger.LogShutdownEvent()

	return nil
}

// GetInfo 获取模块信息
func (m *AuditModule) GetInfo() pluginLib.ModuleInfo {
	return pluginLib.ModuleInfo{
		Name:        "audit",
		Version:     "0.1.0",
		Description: "安全审计模块，用于记录和查询系统安全事件",
		SupportedActions: []string{
			"log_event",
			"get_logs",
			"clear_logs",
		},
	}
}

// HandleMessage 处理消息
func (m *AuditModule) HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error) {
	m.logger.Info("处理消息", "type", messageType, "id", messageID)

	switch messageType {
	case "audit_event":
		// 处理审计事件
		return m.auditLogger.LogEvent(payload)
	case "get_audit_logs":
		// 处理获取日志请求
		return m.auditLogger.GetLogs(payload)
	case "clear_audit_logs":
		// 处理清除日志请求
		return m.auditLogger.ClearLogs(payload)
	default:
		return nil, fmt.Errorf("不支持的消息类型: %s", messageType)
	}
}
