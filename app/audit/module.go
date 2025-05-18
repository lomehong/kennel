package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lomehong/kennel/pkg/core/plugin"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// AuditModule 实现了安全审计模块
type AuditModule struct {
	*sdk.BaseModule
	auditLogger *AuditLogger
}

// NewAuditModule 创建一个新的安全审计模块
func NewAuditModule() *AuditModule {
	// 创建基础模块
	base := sdk.NewBaseModule(
		"audit",
		"安全审计插件",
		"1.0.0",
		"安全审计模块，用于记录和管理系统安全事件",
	)

	// 创建模块
	module := &AuditModule{
		BaseModule: base,
	}

	return module
}

// Init 初始化模块
func (m *AuditModule) Init(ctx context.Context, config *plugin.ModuleConfig) error {
	// 调用基类初始化
	if err := m.BaseModule.Init(ctx, config); err != nil {
		return err
	}

	m.Logger.Info("初始化安全审计模块")

	// 设置日志级别
	logLevel := sdk.GetConfigString(m.Config, "log_level", "info")
	m.Logger.Debug("设置日志级别", "level", logLevel)

	// 创建审计日志记录器
	auditLogger, err := NewAuditLogger(m.Logger, m.Config)
	if err != nil {
		m.Logger.Error("创建审计日志记录器失败", "error", err)
		return fmt.Errorf("创建审计日志记录器失败: %w", err)
	}
	m.auditLogger = auditLogger

	// 记录初始化事件
	m.auditLogger.LogEvent("system.init", "system", map[string]interface{}{
		"module": "audit",
		"action": "initialize",
	})

	return nil
}

// Start 启动模块
func (m *AuditModule) Start() error {
	m.Logger.Info("启动安全审计模块")

	// 确保审计日志记录器已初始化
	if m.auditLogger == nil {
		m.Logger.Warn("审计日志记录器未初始化，尝试初始化")

		// 创建默认配置
		if m.Config == nil {
			m.Config = make(map[string]interface{})
		}

		// 创建审计日志记录器
		auditLogger, err := NewAuditLogger(m.Logger, m.Config)
		if err != nil {
			m.Logger.Error("创建审计日志记录器失败", "error", err)
			return fmt.Errorf("创建审计日志记录器失败: %w", err)
		}
		m.auditLogger = auditLogger
	}

	// 记录启动事件
	m.auditLogger.LogEvent("system.start", "system", map[string]interface{}{
		"module": "audit",
		"action": "start",
	})

	// 清理旧日志
	go func() {
		removedCount := m.auditLogger.CleanupOldLogs()
		m.Logger.Info("清理旧日志", "removed_count", removedCount)
	}()

	return nil
}

// Stop 停止模块
func (m *AuditModule) Stop() error {
	m.Logger.Info("停止安全审计模块")

	// 确保审计日志记录器已初始化
	if m.auditLogger != nil {
		// 记录停止事件
		m.auditLogger.LogEvent("system.stop", "system", map[string]interface{}{
			"module": "audit",
			"action": "stop",
		})

		// 关闭审计日志记录器
		if err := m.auditLogger.Close(); err != nil {
			m.Logger.Error("关闭审计日志记录器失败", "error", err)
			return fmt.Errorf("关闭审计日志记录器失败: %w", err)
		}
	} else {
		m.Logger.Warn("审计日志记录器未初始化，跳过关闭操作")
	}

	return nil
}

// HandleRequest 处理请求
func (m *AuditModule) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	m.Logger.Info("处理请求", "action", req.Action)

	// 确保审计日志记录器已初始化
	if m.auditLogger == nil {
		m.Logger.Warn("审计日志记录器未初始化，尝试初始化")

		// 创建默认配置
		if m.Config == nil {
			m.Config = make(map[string]interface{})
		}

		// 创建审计日志记录器
		auditLogger, err := NewAuditLogger(m.Logger, m.Config)
		if err != nil {
			m.Logger.Error("创建审计日志记录器失败", "error", err)
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "init_error",
					Message: fmt.Sprintf("初始化审计日志记录器失败: %v", err),
				},
			}, nil
		}
		m.auditLogger = auditLogger
	}

	switch req.Action {
	case "log_event":
		// 记录审计事件
		eventType, ok := req.Params["event_type"].(string)
		if !ok {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "缺少事件类型参数",
				},
			}, nil
		}

		user, ok := req.Params["user"].(string)
		if !ok {
			user = "unknown"
		}

		details, ok := req.Params["details"].(map[string]interface{})
		if !ok {
			details = make(map[string]interface{})
		}

		// 记录事件
		if err := m.auditLogger.LogEvent(eventType, user, details); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "log_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"status":  "success",
				"message": "事件已记录",
			},
		}, nil

	case "get_logs":
		// 获取审计日志
		eventType, _ := req.Params["event_type"].(string)
		user, _ := req.Params["user"].(string)

		var startTime, endTime time.Time
		if startTimeStr, ok := req.Params["start_time"].(string); ok && startTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
				startTime = t
			}
		}

		if endTimeStr, ok := req.Params["end_time"].(string); ok && endTimeStr != "" {
			if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
				endTime = t
			}
		}

		// 过滤日志
		logs := m.auditLogger.FilterLogs(eventType, user, startTime, endTime)

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"logs":  LogsToMap(logs),
				"count": len(logs),
			},
		}, nil

	case "clear_logs":
		// 清除审计日志
		user, ok := req.Params["user"].(string)
		if !ok {
			user = "unknown"
		}

		// 清除日志
		if err := m.auditLogger.ClearLogs(user); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "clear_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"status":  "success",
				"message": "日志已清除",
			},
		}, nil

	default:
		return &plugin.Response{
			ID:      req.ID,
			Success: false,
			Error: &plugin.ErrorInfo{
				Code:    "unknown_action",
				Message: fmt.Sprintf("不支持的操作: %s", req.Action),
			},
		}, nil
	}
}

// HandleEvent 处理事件
func (m *AuditModule) HandleEvent(ctx context.Context, event *plugin.Event) error {
	m.Logger.Info("处理事件", "type", event.Type, "source", event.Source)

	// 确保审计日志记录器已初始化
	if m.auditLogger == nil {
		m.Logger.Warn("审计日志记录器未初始化，尝试初始化")

		// 创建默认配置
		if m.Config == nil {
			m.Config = make(map[string]interface{})
		}

		// 创建审计日志记录器
		auditLogger, err := NewAuditLogger(m.Logger, m.Config)
		if err != nil {
			m.Logger.Error("创建审计日志记录器失败", "error", err)
			return fmt.Errorf("创建审计日志记录器失败: %w", err)
		}
		m.auditLogger = auditLogger
	}

	// 检查是否需要记录事件
	shouldLog := false

	switch {
	case event.Type == "system.startup" || event.Type == "system.shutdown":
		shouldLog = sdk.GetConfigBool(m.Config, "log_system_events", true)
	case event.Type == "user.login" || event.Type == "user.logout":
		shouldLog = sdk.GetConfigBool(m.Config, "log_user_events", true)
	case event.Type == "network.connect" || event.Type == "network.disconnect":
		shouldLog = sdk.GetConfigBool(m.Config, "log_network_events", true)
	case event.Type == "file.create" || event.Type == "file.modify" || event.Type == "file.delete":
		shouldLog = sdk.GetConfigBool(m.Config, "log_file_events", true)
	}

	if shouldLog {
		// 获取用户信息
		user := "system"
		if userVal, ok := event.Data["user"]; ok {
			if userStr, ok := userVal.(string); ok {
				user = userStr
			}
		}

		// 记录事件
		if err := m.auditLogger.LogEvent(event.Type, user, event.Data); err != nil {
			m.Logger.Error("记录事件失败", "error", err)
			return fmt.Errorf("记录事件失败: %w", err)
		}
	}

	return nil
}
