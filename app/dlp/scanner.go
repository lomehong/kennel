package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/lomehong/kennel/pkg/logger"
)

// Scanner 负责扫描内容
type Scanner struct {
	ruleManager *RuleManager
	logger      logger.Logger
}

// NewScanner 创建一个新的扫描器
func NewScanner(ruleManager *RuleManager, logger logger.Logger) *Scanner {
	return &Scanner{
		ruleManager: ruleManager,
		logger:      logger,
	}
}

// ScanFile 扫描文件
func (s *Scanner) ScanFile(path string) (map[string]interface{}, error) {
	s.logger.Info("扫描文件", "path", path)

	// 添加超时控制，避免命令执行时间过长
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 读取文件内容
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "powershell", "-Command", fmt.Sprintf("Get-Content '%s'", path))
	default:
		cmd = exec.CommandContext(ctx, "cat", path)
	}

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.logger.Error("读取文件超时", "path", path)
			return nil, fmt.Errorf("读取文件超时: %s", path)
		}
		s.logger.Error("读取文件失败", "path", path, "error", err)
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	// 扫描内容
	content := string(output)
	alerts := s.ScanContent(content, path, "file")

	// 直接构建map，避免JSON序列化/反序列化
	result := make(map[string]interface{})
	result["alerts"] = convertAlertsToMap(alerts)

	return result, nil
}

// ScanClipboard 扫描剪贴板
func (s *Scanner) ScanClipboard() (map[string]interface{}, error) {
	s.logger.Info("扫描剪贴板")

	// 添加超时控制，避免命令执行时间过长
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 读取剪贴板内容
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.CommandContext(ctx, "powershell", "-Command", "Get-Clipboard")
	case "darwin":
		cmd = exec.CommandContext(ctx, "pbpaste")
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.logger.Error("读取剪贴板超时")
			return nil, fmt.Errorf("读取剪贴板超时")
		}
		s.logger.Error("读取剪贴板失败", "error", err)
		return nil, fmt.Errorf("读取剪贴板失败: %w", err)
	}

	// 扫描内容
	content := string(output)
	alerts := s.ScanContent(content, "clipboard", "clipboard")

	// 直接构建map，避免JSON序列化/反序列化
	result := make(map[string]interface{})
	result["alerts"] = convertAlertsToMap(alerts)

	return result, nil
}

// ScanContent 扫描内容
func (s *Scanner) ScanContent(content, source, sourceType string) []DLPAlert {
	// 预分配切片容量，避免动态扩容
	alerts := make([]DLPAlert, 0, 10) // 假设最多有10个警报

	// 获取当前时间戳
	timestamp := time.Now().Format(time.RFC3339)

	// 获取所有规则
	rules := s.ruleManager.GetRules()

	// 对每个规则进行扫描
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// 获取编译后的正则表达式
		compiledRules := s.ruleManager.GetCompiledRules()
		if patterns, ok := compiledRules[rule.ID]; ok {
			for _, re := range patterns {
				// 查找匹配项（限制最大匹配数量，避免DoS攻击）
				matches := re.FindAllString(content, 100) // 最多返回100个匹配项
				for _, match := range matches {
					alert := DLPAlert{
						RuleID:      rule.ID,
						RuleName:    rule.Name,
						Content:     match,
						Source:      source,
						Destination: sourceType,
						Action:      rule.Action,
						Timestamp:   timestamp,
					}
					alerts = append(alerts, alert)
				}
			}
		}
	}

	return alerts
}
