package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lomehong/kennel/pkg/sdk/go"
)

// Scanner 扫描器
type Scanner struct {
	logger       sdk.Logger
	ruleManager  *RuleManager
	alertManager *AlertManager
	config       map[string]interface{}
}

// NewScanner 创建一个新的扫描器
func NewScanner(logger sdk.Logger, ruleManager *RuleManager, alertManager *AlertManager, config map[string]interface{}) *Scanner {
	return &Scanner{
		logger:       logger,
		ruleManager:  ruleManager,
		alertManager: alertManager,
		config:       config,
	}
}

// ScanFile 扫描文件
func (s *Scanner) ScanFile(path string) ([]DLPAlert, error) {
	s.logger.Info("扫描文件", "path", path)
	
	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", path)
	}
	
	// 读取文件内容
	content, err := ioutil.ReadFile(path)
	if err != nil {
		s.logger.Error("读取文件失败", "path", path, "error", err)
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	
	// 扫描内容
	alerts := s.ScanContent(string(content), path, "file")
	
	// 添加警报
	for _, alert := range alerts {
		s.alertManager.AddAlert(alert)
	}
	
	return alerts, nil
}

// ScanDirectory 扫描目录
func (s *Scanner) ScanDirectory(dir string) ([]DLPAlert, error) {
	s.logger.Info("扫描目录", "dir", dir)
	
	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("目录不存在: %s", dir)
	}
	
	// 获取监控的文件类型
	fileTypes := sdk.GetConfigStringSlice(s.config, "monitored_file_types")
	if len(fileTypes) == 0 {
		s.logger.Warn("未配置监控的文件类型")
		return nil, nil
	}
	
	// 扫描目录
	var alerts []DLPAlert
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			s.logger.Error("访问文件失败", "path", path, "error", err)
			return nil
		}
		
		// 跳过目录
		if info.IsDir() {
			return nil
		}
		
		// 检查文件类型
		matched := false
		for _, pattern := range fileTypes {
			if matched, _ = filepath.Match(pattern, filepath.Base(path)); matched {
				break
			}
		}
		
		if !matched {
			return nil
		}
		
		// 扫描文件
		fileAlerts, err := s.ScanFile(path)
		if err != nil {
			s.logger.Error("扫描文件失败", "path", path, "error", err)
			return nil
		}
		
		alerts = append(alerts, fileAlerts...)
		return nil
	})
	
	if err != nil {
		s.logger.Error("扫描目录失败", "dir", dir, "error", err)
		return nil, fmt.Errorf("扫描目录失败: %w", err)
	}
	
	return alerts, nil
}

// ScanClipboard 扫描剪贴板
func (s *Scanner) ScanClipboard() ([]DLPAlert, error) {
	s.logger.Info("扫描剪贴板")
	
	// 获取剪贴板内容
	content, err := s.getClipboardContent()
	if err != nil {
		s.logger.Error("获取剪贴板内容失败", "error", err)
		return nil, fmt.Errorf("获取剪贴板内容失败: %w", err)
	}
	
	// 扫描内容
	alerts := s.ScanContent(content, "clipboard", "clipboard")
	
	// 添加警报
	for _, alert := range alerts {
		s.alertManager.AddAlert(alert)
	}
	
	return alerts, nil
}

// getClipboardContent 获取剪贴板内容
func (s *Scanner) getClipboardContent() (string, error) {
	// 在实际应用中，这里应该调用系统API获取剪贴板内容
	// 由于这是平台相关的，这里只是一个示例
	switch runtime.GOOS {
	case "windows":
		// 使用Windows API获取剪贴板内容
		return "这是一个示例剪贴板内容，包含一个信用卡号：4111-1111-1111-1111", nil
	case "darwin":
		// 使用macOS API获取剪贴板内容
		return "这是一个示例剪贴板内容，包含一个信用卡号：4111-1111-1111-1111", nil
	default:
		return "", fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// ScanContent 扫描内容
func (s *Scanner) ScanContent(content, source, sourceType string) []DLPAlert {
	// 获取所有规则
	rules := s.ruleManager.GetRules()
	
	// 预分配切片容量，避免动态扩容
	alerts := make([]DLPAlert, 0, len(rules))
	
	// 对每个规则进行扫描
	for _, rule := range rules {
		// 跳过禁用的规则
		if !rule.Enabled {
			continue
		}
		
		// 查找匹配项
		matches := rule.regex.FindAllString(content, -1)
		for _, match := range matches {
			// 创建警报
			alert := DLPAlert{
				RuleID:      rule.ID,
				RuleName:    rule.Name,
				Content:     match,
				Source:      source,
				Destination: sourceType,
				Action:      rule.Action,
				Timestamp:   time.Now(),
			}
			
			// 添加警报
			alerts = append(alerts, alert)
			
			// 记录日志
			s.logger.Info("发现敏感数据", "rule", rule.Name, "source", source, "action", rule.Action)
		}
	}
	
	return alerts
}

// MonitorClipboard 监控剪贴板
func (s *Scanner) MonitorClipboard() error {
	s.logger.Info("开始监控剪贴板")
	
	// 检查是否启用剪贴板监控
	if !sdk.GetConfigBool(s.config, "monitor_clipboard", false) {
		s.logger.Info("剪贴板监控已禁用")
		return nil
	}
	
	// 在实际应用中，这里应该启动一个后台协程监控剪贴板变化
	// 由于这是平台相关的，这里只是一个示例
	s.logger.Info("剪贴板监控已启动")
	
	return nil
}

// MonitorFiles 监控文件
func (s *Scanner) MonitorFiles() error {
	s.logger.Info("开始监控文件")
	
	// 检查是否启用文件监控
	if !sdk.GetConfigBool(s.config, "monitor_files", false) {
		s.logger.Info("文件监控已禁用")
		return nil
	}
	
	// 获取监控的目录
	dirs := sdk.GetConfigStringSlice(s.config, "monitored_directories")
	if len(dirs) == 0 {
		s.logger.Warn("未配置监控的目录")
		return nil
	}
	
	// 在实际应用中，这里应该启动一个后台协程监控文件变化
	// 由于这是平台相关的，这里只是一个示例
	s.logger.Info("文件监控已启动", "directories", strings.Join(dirs, ", "))
	
	return nil
}

// StopMonitoring 停止监控
func (s *Scanner) StopMonitoring() error {
	s.logger.Info("停止监控")
	
	// 在实际应用中，这里应该停止所有监控
	// 由于这是平台相关的，这里只是一个示例
	s.logger.Info("监控已停止")
	
	return nil
}
