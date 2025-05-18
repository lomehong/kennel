package main

import (
	"fmt"
	"os/user"
	"time"

	"github.com/lomehong/kennel/pkg/logger"
)

// AuditLogger 负责记录审计日志
type AuditLogger struct {
	logger   logger.Logger
	logStore *AuditLogStore
	config   map[string]interface{}
}

// NewAuditLogger 创建一个新的审计日志记录器
func NewAuditLogger(logger logger.Logger, logStore *AuditLogStore, config map[string]interface{}) *AuditLogger {
	return &AuditLogger{
		logger:   logger,
		logStore: logStore,
		config:   config,
	}
}

// LogEvent 记录一个审计事件
func (l *AuditLogger) LogEvent(params map[string]interface{}) (map[string]interface{}, error) {
	eventType, ok := params["event_type"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少必要参数: event_type")
	}

	details, ok := params["details"].(map[string]interface{})
	if !ok {
		details = make(map[string]interface{})
	}

	// 获取当前用户
	currentUser, err := user.Current()
	username := "unknown"
	if err == nil {
		username = currentUser.Username
	}

	// 如果参数中指定了用户，则使用参数中的用户
	if userParam, ok := params["user"].(string); ok && userParam != "" {
		username = userParam
	}

	// 记录事件
	log := AuditLog{
		Timestamp: time.Now(),
		EventType: eventType,
		User:      username,
		Details:   details,
	}

	l.logStore.AddLog(log)
	l.logger.Info("记录审计事件", "event_type", eventType, "user", username)

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("已记录事件: %s", eventType),
	}, nil
}

// GetLogs 获取审计日志
func (l *AuditLogger) GetLogs(params map[string]interface{}) (map[string]interface{}, error) {
	// 过滤条件
	eventType, _ := params["event_type"].(string)
	user, _ := params["user"].(string)

	// 过滤日志
	filteredLogs := l.logStore.FilterLogs(eventType, user)
	l.logger.Info("获取审计日志", "count", len(filteredLogs), "event_type", eventType, "user", user)

	return map[string]interface{}{
		"logs": LogsToMap(filteredLogs),
	}, nil
}

// ClearLogs 清除审计日志
func (l *AuditLogger) ClearLogs(params map[string]interface{}) (map[string]interface{}, error) {
	// 记录清除事件
	currentUser, err := user.Current()
	username := "unknown"
	if err == nil {
		username = currentUser.Username
	}

	// 保存清除事件
	clearEvent := AuditLog{
		Timestamp: time.Now(),
		EventType: "logs_cleared",
		User:      username,
		Details: map[string]interface{}{
			"count": len(l.logStore.GetLogs()),
		},
	}

	// 清除日志
	l.logStore.ClearLogs(clearEvent)
	l.logger.Info("清除审计日志", "user", username)

	return map[string]interface{}{
		"success": true,
		"message": "已清除所有日志",
	}, nil
}

// LogInitEvent 记录初始化事件
func (l *AuditLogger) LogInitEvent(osName, arch, version string) {
	// 获取当前用户
	currentUser, err := user.Current()
	username := "unknown"
	if err == nil {
		username = currentUser.Username
	}

	// 记录初始化事件
	l.logStore.AddLog(AuditLog{
		Timestamp: time.Now(),
		EventType: "module_init",
		User:      username,
		Details: map[string]interface{}{
			"os":      osName,
			"arch":    arch,
			"version": version,
		},
	})

	l.logger.Info("模块初始化", "user", username, "os", osName, "arch", arch)
}

// LogShutdownEvent 记录关闭事件
func (l *AuditLogger) LogShutdownEvent() {
	// 获取当前用户
	currentUser, err := user.Current()
	username := "unknown"
	if err == nil {
		username = currentUser.Username
	}

	// 记录关闭事件
	l.logStore.AddLog(AuditLog{
		Timestamp: time.Now(),
		EventType: "module_shutdown",
		User:      username,
		Details:   map[string]interface{}{},
	})

	l.logger.Info("模块关闭", "user", username)
}
