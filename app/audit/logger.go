package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lomehong/kennel/pkg/sdk/go"
)

// AuditLogger 负责记录审计日志
type AuditLogger struct {
	logger     sdk.Logger
	store      *AuditLogStore
	config     map[string]interface{}
	fileLogger *os.File
}

// NewAuditLogger 创建一个新的审计日志记录器
func NewAuditLogger(logger sdk.Logger, config map[string]interface{}) (*AuditLogger, error) {
	// 创建审计日志存储
	store := NewAuditLogStore()

	// 创建审计日志记录器
	auditLogger := &AuditLogger{
		logger: logger,
		store:  store,
		config: config,
	}

	// 初始化文件日志
	if err := auditLogger.initFileLogger(); err != nil {
		return nil, err
	}

	return auditLogger, nil
}

// initFileLogger 初始化文件日志
func (l *AuditLogger) initFileLogger() error {
	// 检查存储类型
	storageType := sdk.GetConfigString(l.config, "storage.type", "file")
	if storageType != "file" {
		// 不使用文件存储，直接返回
		return nil
	}

	// 获取日志目录
	logDir := sdk.GetConfigString(l.config, "storage.file.dir", "data/audit/logs")

	// 创建日志目录
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 获取日志文件名格式
	filenameFormat := sdk.GetConfigString(l.config, "storage.file.filename_format", "audit-%Y-%m-%d.log")

	// 替换日期格式
	now := time.Now()
	filename := now.Format(filenameFormat)
	filename = filepath.Join(logDir, filename)

	// 打开日志文件
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	l.fileLogger = file
	return nil
}

// Close 关闭日志记录器
func (l *AuditLogger) Close() error {
	if l.fileLogger != nil {
		return l.fileLogger.Close()
	}
	return nil
}

// LogEvent 记录审计事件
func (l *AuditLogger) LogEvent(eventType, user string, details map[string]interface{}) error {
	// 创建审计日志
	log := AuditLog{
		Timestamp: time.Now(),
		EventType: eventType,
		User:      user,
		Details:   details,
	}

	// 添加到存储
	l.store.AddLog(log)

	// 记录到文件
	if l.fileLogger != nil {
		// 格式化日志
		logLine := fmt.Sprintf("[%s] %s - %s - %v\n",
			log.Timestamp.Format(time.RFC3339),
			log.EventType,
			log.User,
			log.Details,
		)

		// 写入文件
		if _, err := l.fileLogger.WriteString(logLine); err != nil {
			l.logger.Error("写入审计日志文件失败", "error", err)
			return fmt.Errorf("写入审计日志文件失败: %w", err)
		}
	}

	// 检查是否需要发送警报
	if l.shouldSendAlert(eventType) {
		l.sendAlert(log)
	}

	return nil
}

// shouldSendAlert 检查是否需要发送警报
func (l *AuditLogger) shouldSendAlert(eventType string) bool {
	// 检查是否启用警报
	enableAlerts := sdk.GetConfigBool(l.config, "enable_alerts", false)
	if !enableAlerts {
		return false
	}

	// 在实际应用中，这里应该根据事件类型和严重性判断是否需要发送警报
	return eventType == "security.violation" || eventType == "system.critical"
}

// sendAlert 发送警报
func (l *AuditLogger) sendAlert(log AuditLog) {
	// 获取警报接收者
	recipients := sdk.GetConfigStringSlice(l.config, "alert_recipients")
	if len(recipients) == 0 {
		l.logger.Warn("未配置警报接收者，无法发送警报")
		return
	}

	// 在实际应用中，这里应该发送邮件或其他通知
	l.logger.Info("发送警报", "event", log.EventType, "recipients", recipients)
}

// GetLogs 获取所有审计日志
func (l *AuditLogger) GetLogs() []AuditLog {
	return l.store.GetLogs()
}

// FilterLogs 根据条件过滤审计日志
func (l *AuditLogger) FilterLogs(eventType, user string, startTime, endTime time.Time) []AuditLog {
	return l.store.FilterLogs(eventType, user, startTime, endTime)
}

// ClearLogs 清除所有审计日志
func (l *AuditLogger) ClearLogs(user string) error {
	// 创建清除事件
	clearEvent := AuditLog{
		Timestamp: time.Now(),
		EventType: "audit.clear",
		User:      user,
		Details: map[string]interface{}{
			"action": "clear_logs",
		},
	}

	// 清除日志
	l.store.ClearLogs(clearEvent)

	// 记录到文件
	if l.fileLogger != nil {
		// 关闭当前文件
		if err := l.fileLogger.Close(); err != nil {
			l.logger.Error("关闭审计日志文件失败", "error", err)
		}

		// 重新初始化文件日志
		if err := l.initFileLogger(); err != nil {
			return fmt.Errorf("重新初始化审计日志文件失败: %w", err)
		}

		// 记录清除事件
		logLine := fmt.Sprintf("[%s] %s - %s - %v\n",
			clearEvent.Timestamp.Format(time.RFC3339),
			clearEvent.EventType,
			clearEvent.User,
			clearEvent.Details,
		)

		// 写入文件
		if _, err := l.fileLogger.WriteString(logLine); err != nil {
			l.logger.Error("写入审计日志文件失败", "error", err)
			return fmt.Errorf("写入审计日志文件失败: %w", err)
		}
	}

	return nil
}

// CleanupOldLogs 清理超过保留期的旧日志
func (l *AuditLogger) CleanupOldLogs() int {
	// 获取日志保留天数
	retentionDays := sdk.GetConfigInt(l.config, "log_retention_days", 30)
	if retentionDays <= 0 {
		return 0
	}

	// 清理旧日志
	removedCount := l.store.CleanupOldLogs(retentionDays)

	// 记录清理事件
	if removedCount > 0 {
		l.LogEvent("audit.cleanup", "system", map[string]interface{}{
			"action":        "cleanup_logs",
			"removed_count": removedCount,
			"retention_days": retentionDays,
		})
	}

	return removedCount
}
