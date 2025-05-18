package main

import (
	"context"
	"fmt"

	"github.com/lomehong/kennel/pkg/core/plugin"
	sdk "github.com/lomehong/kennel/pkg/sdk/go"
)

// DLPModule 实现了数据防泄漏模块
type DLPModule struct {
	*sdk.BaseModule
	ruleManager   *RuleManager
	alertManager  *AlertManager
	scanner       *Scanner
	monitorCtx    context.Context
	monitorCancel context.CancelFunc
}

// NewDLPModule 创建一个新的数据防泄漏模块
func NewDLPModule() *DLPModule {
	// 创建基础模块
	base := sdk.NewBaseModule(
		"dlp",
		"数据防泄漏插件",
		"1.0.0",
		"数据防泄漏模块，用于检测和防止敏感数据泄漏",
	)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建模块
	module := &DLPModule{
		BaseModule:    base,
		monitorCtx:    ctx,
		monitorCancel: cancel,
	}

	return module
}

// Init 初始化模块
func (m *DLPModule) Init(ctx context.Context, config *plugin.ModuleConfig) error {
	// 调用基类初始化
	if err := m.BaseModule.Init(ctx, config); err != nil {
		return err
	}

	m.Logger.Info("初始化数据防泄漏模块")

	// 设置日志级别
	logLevel := sdk.GetConfigString(m.Config, "log_level", "info")
	m.Logger.Debug("设置日志级别", "level", logLevel)

	// 创建规则管理器
	m.ruleManager = NewRuleManager(m.Logger)

	// 创建警报管理器
	m.alertManager = NewAlertManager()

	// 创建扫描器
	m.scanner = NewScanner(m.Logger, m.ruleManager, m.alertManager, m.Config)

	// 加载规则
	if err := m.ruleManager.LoadRules(m.Config); err != nil {
		m.Logger.Error("加载规则失败", "error", err)
		return fmt.Errorf("加载规则失败: %w", err)
	}

	return nil
}

// Start 启动模块
func (m *DLPModule) Start() error {
	m.Logger.Info("启动数据防泄漏模块")

	// 确保规则管理器已初始化
	if m.ruleManager == nil {
		m.Logger.Warn("规则管理器未初始化，尝试初始化")
		m.ruleManager = NewRuleManager(m.Logger)

		// 加载规则
		if err := m.ruleManager.LoadRules(m.Config); err != nil {
			m.Logger.Error("加载规则失败", "error", err)
		}
	}

	// 确保警报管理器已初始化
	if m.alertManager == nil {
		m.Logger.Warn("警报管理器未初始化，尝试初始化")
		m.alertManager = NewAlertManager()
	}

	// 确保扫描器已初始化
	if m.scanner == nil {
		m.Logger.Warn("扫描器未初始化，尝试初始化")
		m.scanner = NewScanner(m.Logger, m.ruleManager, m.alertManager, m.Config)
	}

	// 确保监控上下文已初始化
	if m.monitorCtx == nil {
		m.Logger.Warn("监控上下文未初始化，尝试初始化")
		m.monitorCtx, m.monitorCancel = context.WithCancel(context.Background())
	}

	// 启动剪贴板监控
	if err := m.scanner.MonitorClipboard(); err != nil {
		m.Logger.Error("启动剪贴板监控失败", "error", err)
	}

	// 启动文件监控
	if err := m.scanner.MonitorFiles(); err != nil {
		m.Logger.Error("启动文件监控失败", "error", err)
	}

	return nil
}

// Stop 停止模块
func (m *DLPModule) Stop() error {
	m.Logger.Info("停止数据防泄漏模块")

	// 停止监控
	if m.monitorCancel != nil {
		m.monitorCancel()
	} else {
		m.Logger.Warn("监控取消函数未初始化，跳过停止监控")
	}

	// 停止扫描器
	if m.scanner != nil {
		if err := m.scanner.StopMonitoring(); err != nil {
			m.Logger.Error("停止监控失败", "error", err)
		}
	} else {
		m.Logger.Warn("扫描器未初始化，跳过停止监控")
	}

	return nil
}

// HandleRequest 处理请求
func (m *DLPModule) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	m.Logger.Info("处理请求", "action", req.Action)

	// 确保规则管理器已初始化
	if m.ruleManager == nil {
		m.Logger.Warn("规则管理器未初始化，尝试初始化")
		m.ruleManager = NewRuleManager(m.Logger)

		// 加载规则
		if err := m.ruleManager.LoadRules(m.Config); err != nil {
			m.Logger.Error("加载规则失败", "error", err)
		}
	}

	// 确保警报管理器已初始化
	if m.alertManager == nil {
		m.Logger.Warn("警报管理器未初始化，尝试初始化")
		m.alertManager = NewAlertManager()
	}

	// 确保扫描器已初始化
	if m.scanner == nil {
		m.Logger.Warn("扫描器未初始化，尝试初始化")
		m.scanner = NewScanner(m.Logger, m.ruleManager, m.alertManager, m.Config)
	}

	switch req.Action {
	case "get_rules":
		// 获取规则列表
		rules := m.ruleManager.GetRules()
		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"rules": RulesToMap(rules),
				"count": len(rules),
			},
		}, nil

	case "add_rule":
		// 添加规则
		rule := &DLPRule{
			ID:          sdk.GetConfigString(req.Params, "id", ""),
			Name:        sdk.GetConfigString(req.Params, "name", ""),
			Description: sdk.GetConfigString(req.Params, "description", ""),
			Pattern:     sdk.GetConfigString(req.Params, "pattern", ""),
			Action:      sdk.GetConfigString(req.Params, "action", "alert"),
			Enabled:     sdk.GetConfigBool(req.Params, "enabled", true),
		}

		// 检查必要字段
		if rule.ID == "" || rule.Pattern == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "规则ID和模式不能为空",
				},
			}, nil
		}

		// 添加规则
		if err := m.ruleManager.AddRule(rule); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "add_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"rule": RuleToMap(rule),
			},
		}, nil

	case "update_rule":
		// 更新规则
		rule := &DLPRule{
			ID:          sdk.GetConfigString(req.Params, "id", ""),
			Name:        sdk.GetConfigString(req.Params, "name", ""),
			Description: sdk.GetConfigString(req.Params, "description", ""),
			Pattern:     sdk.GetConfigString(req.Params, "pattern", ""),
			Action:      sdk.GetConfigString(req.Params, "action", "alert"),
			Enabled:     sdk.GetConfigBool(req.Params, "enabled", true),
		}

		// 检查必要字段
		if rule.ID == "" || rule.Pattern == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "规则ID和模式不能为空",
				},
			}, nil
		}

		// 更新规则
		if err := m.ruleManager.UpdateRule(rule); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "update_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"rule": RuleToMap(rule),
			},
		}, nil

	case "delete_rule":
		// 删除规则
		id := sdk.GetConfigString(req.Params, "id", "")
		if id == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "规则ID不能为空",
				},
			}, nil
		}

		// 删除规则
		if err := m.ruleManager.DeleteRule(id); err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "delete_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"id": id,
			},
		}, nil

	case "scan_file":
		// 扫描文件
		path := sdk.GetConfigString(req.Params, "path", "")
		if path == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "文件路径不能为空",
				},
			}, nil
		}

		// 扫描文件
		alerts, err := m.scanner.ScanFile(path)
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "scan_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"alerts": AlertsToMap(alerts),
				"count":  len(alerts),
			},
		}, nil

	case "scan_directory":
		// 扫描目录
		dir := sdk.GetConfigString(req.Params, "directory", "")
		if dir == "" {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "invalid_param",
					Message: "目录路径不能为空",
				},
			}, nil
		}

		// 扫描目录
		alerts, err := m.scanner.ScanDirectory(dir)
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "scan_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"alerts": AlertsToMap(alerts),
				"count":  len(alerts),
			},
		}, nil

	case "scan_clipboard":
		// 扫描剪贴板
		alerts, err := m.scanner.ScanClipboard()
		if err != nil {
			return &plugin.Response{
				ID:      req.ID,
				Success: false,
				Error: &plugin.ErrorInfo{
					Code:    "scan_error",
					Message: err.Error(),
				},
			}, nil
		}

		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"alerts": AlertsToMap(alerts),
				"count":  len(alerts),
			},
		}, nil

	case "get_alerts":
		// 获取警报列表
		alerts := m.alertManager.GetAlerts()
		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"alerts": AlertsToMap(alerts),
				"count":  len(alerts),
			},
		}, nil

	case "clear_alerts":
		// 清除警报
		m.alertManager.ClearAlerts()
		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"status":  "success",
				"message": "警报已清除",
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
func (m *DLPModule) HandleEvent(ctx context.Context, event *plugin.Event) error {
	m.Logger.Info("处理事件", "type", event.Type, "source", event.Source)

	// 确保规则管理器已初始化
	if m.ruleManager == nil {
		m.Logger.Warn("规则管理器未初始化，尝试初始化")
		m.ruleManager = NewRuleManager(m.Logger)

		// 加载规则
		if err := m.ruleManager.LoadRules(m.Config); err != nil {
			m.Logger.Error("加载规则失败", "error", err)
		}
	}

	// 确保警报管理器已初始化
	if m.alertManager == nil {
		m.Logger.Warn("警报管理器未初始化，尝试初始化")
		m.alertManager = NewAlertManager()
	}

	// 确保扫描器已初始化
	if m.scanner == nil {
		m.Logger.Warn("扫描器未初始化，尝试初始化")
		m.scanner = NewScanner(m.Logger, m.ruleManager, m.alertManager, m.Config)
	}

	switch event.Type {
	case "system.startup":
		// 系统启动事件
		m.Logger.Info("系统启动")
		return nil

	case "system.shutdown":
		// 系统关闭事件
		m.Logger.Info("系统关闭")
		return nil

	case "dlp.scan_request":
		// 扫描请求
		m.Logger.Info("收到扫描请求")
		if path, ok := event.Data["path"].(string); ok && path != "" {
			_, err := m.scanner.ScanFile(path)
			return err
		}
		return nil

	default:
		// 忽略其他事件
		return nil
	}
}
