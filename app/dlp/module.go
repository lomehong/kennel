package main

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/logger"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
)

// DLPModule 实现了数据防泄漏模块
type DLPModule struct {
	logger      logger.Logger
	config      map[string]interface{}
	ruleManager *RuleManager
	scanner     *Scanner
}

// NewDLPModule 创建一个新的数据防泄漏模块
func NewDLPModule() pluginLib.Module {
	// 使用新的日志包创建日志器
	log := logger.NewLogger("dlp-module", hclog.Info)

	ruleManager := NewRuleManager()

	module := &DLPModule{
		logger:      log,
		config:      make(map[string]interface{}),
		ruleManager: ruleManager,
	}

	// 创建扫描器
	module.scanner = NewScanner(ruleManager, log)

	return module
}

// Init 初始化模块
func (m *DLPModule) Init(config map[string]interface{}) error {
	m.logger.Info("初始化数据防泄漏模块")
	m.config = config

	// 从配置中加载规则
	m.ruleManager.LoadRules(config)

	m.logger.Info("已加载规则", "count", len(m.ruleManager.GetRules()))
	return nil
}

// Execute 执行模块操作
func (m *DLPModule) Execute(action string, params map[string]interface{}) (map[string]interface{}, error) {
	m.logger.Info("执行操作", "action", action)

	switch action {
	case "list_rules":
		return m.ruleManager.ListRules()
	case "add_rule":
		return m.ruleManager.AddRule(params)
	case "delete_rule":
		if id, ok := params["id"].(string); ok {
			return m.ruleManager.DeleteRule(id)
		}
		return nil, fmt.Errorf("缺少规则ID参数")
	case "scan_file":
		if path, ok := params["path"].(string); ok {
			return m.scanner.ScanFile(path)
		}
		return nil, fmt.Errorf("缺少文件路径参数")
	case "scan_clipboard":
		return m.scanner.ScanClipboard()
	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
}

// Shutdown 关闭模块
func (m *DLPModule) Shutdown() error {
	m.logger.Info("关闭数据防泄漏模块")
	return nil
}

// GetInfo 获取模块信息
func (m *DLPModule) GetInfo() pluginLib.ModuleInfo {
	return pluginLib.ModuleInfo{
		Name:             "dlp",
		Version:          "0.1.0",
		Description:      "数据防泄漏模块，用于检测和防止敏感数据泄漏",
		SupportedActions: []string{"list_rules", "add_rule", "delete_rule", "scan_file", "scan_clipboard"},
	}
}

// HandleMessage 处理消息
func (m *DLPModule) HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error) {
	m.logger.Info("处理消息", "type", messageType, "id", messageID)

	switch messageType {
	case "scan_file_request":
		// 处理文件扫描请求
		if path, ok := payload["path"].(string); ok {
			return m.scanner.ScanFile(path)
		}
		return nil, fmt.Errorf("缺少文件路径参数")
	case "scan_clipboard_request":
		// 处理剪贴板扫描请求
		return m.scanner.ScanClipboard()
	case "rule_update":
		// 处理规则更新
		return m.ruleManager.AddRule(payload)
	default:
		return nil, fmt.Errorf("不支持的消息类型: %s", messageType)
	}
}
