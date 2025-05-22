package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogEntry 表示一条日志记录
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// LogManager 日志管理器，负责收集和管理系统日志
type LogManager struct {
	logger       Logger
	logDir       string
	maxLogSize   int64
	maxLogFiles  int
	logEntries   []LogEntry
	entriesMutex sync.RWMutex
	maxEntries   int
}

// LogManagerOption 日志管理器选项
type LogManagerOption func(*LogManager)

// WithLogDir 设置日志目录
func WithLogDir(dir string) LogManagerOption {
	return func(lm *LogManager) {
		lm.logDir = dir
	}
}

// WithMaxLogSize 设置单个日志文件的最大大小（字节）
func WithMaxLogSize(size int64) LogManagerOption {
	return func(lm *LogManager) {
		lm.maxLogSize = size
	}
}

// WithMaxLogFiles 设置最大日志文件数量
func WithMaxLogFiles(count int) LogManagerOption {
	return func(lm *LogManager) {
		lm.maxLogFiles = count
	}
}

// WithMaxEntries 设置内存中保存的最大日志条目数
func WithMaxEntries(count int) LogManagerOption {
	return func(lm *LogManager) {
		lm.maxEntries = count
	}
}

// NewLogManager 创建一个新的日志管理器
func NewLogManager(log Logger, options ...LogManagerOption) *LogManager {
	if log == nil {
		// 创建默认日志配置
		config := DefaultLogConfig()
		config.Level = LogLevelInfo

		// 创建增强日志记录器
		enhancedLogger, err := NewEnhancedLogger(config)
		if err != nil {
			// 如果创建失败，使用默认配置
			enhancedLogger, _ = NewEnhancedLogger(nil)
		}

		// 设置名称
		log = enhancedLogger.Named("log-manager")
	}

	// 创建日志管理器
	lm := &LogManager{
		logger:      log,
		logDir:      "logs",
		maxLogSize:  10 * 1024 * 1024, // 10MB
		maxLogFiles: 10,
		maxEntries:  10000,
		logEntries:  make([]LogEntry, 0, 1000),
	}

	// 应用选项
	for _, option := range options {
		option(lm)
	}

	// 确保日志目录存在
	if err := os.MkdirAll(lm.logDir, 0755); err != nil {
		lm.logger.Error("创建日志目录失败", "error", err)
	}

	return lm
}

// Log 记录一条日志
func (lm *LogManager) Log(level, message, source string, data map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    source,
		Data:      data,
	}

	// 添加到内存中
	lm.entriesMutex.Lock()
	lm.logEntries = append(lm.logEntries, entry)

	// 如果超过最大条目数，删除最旧的条目
	if len(lm.logEntries) > lm.maxEntries {
		lm.logEntries = lm.logEntries[len(lm.logEntries)-lm.maxEntries:]
	}
	lm.entriesMutex.Unlock()

	// 写入文件
	go lm.writeLogToFile(entry)
}

// GetLogs 获取日志
func (lm *LogManager) GetLogs(limit int, offset int, level string, source string) ([]interface{}, error) {
	lm.entriesMutex.RLock()
	defer lm.entriesMutex.RUnlock()

	// 过滤日志
	filtered := make([]LogEntry, 0)
	for _, entry := range lm.logEntries {
		if (level == "" || entry.Level == level) && (source == "" || entry.Source == source) {
			filtered = append(filtered, entry)
		}
	}

	// 按时间倒序排序
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	// 应用分页
	total := len(filtered)
	if offset >= total {
		return []interface{}{}, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	// 转换为接口切片
	result := make([]interface{}, end-offset)
	for i, entry := range filtered[offset:end] {
		result[i] = map[string]interface{}{
			"timestamp": entry.Timestamp.Format(time.RFC3339),
			"level":     entry.Level,
			"message":   entry.Message,
			"source":    entry.Source,
			"data":      entry.Data,
		}
	}

	return result, nil
}

// writeLogToFile 将日志写入文件
func (lm *LogManager) writeLogToFile(entry LogEntry) {
	// 构建日志文件名
	logFile := filepath.Join(lm.logDir, fmt.Sprintf("%s.log", entry.Source))

	// 检查文件大小
	if info, err := os.Stat(logFile); err == nil && info.Size() > lm.maxLogSize {
		// 文件已存在且超过大小限制，进行轮转
		lm.rotateLogFile(logFile)
	}

	// 打开日志文件（追加模式）
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		lm.logger.Error("打开日志文件失败", "error", err)
		return
	}
	defer file.Close()

	// 格式化日志条目
	logLine := fmt.Sprintf("[%s] [%s] %s\n", entry.Timestamp.Format(time.RFC3339), strings.ToUpper(entry.Level), entry.Message)

	// 写入日志
	if _, err := file.WriteString(logLine); err != nil {
		lm.logger.Error("写入日志文件失败", "error", err)
	}
}

// rotateLogFile 轮转日志文件
func (lm *LogManager) rotateLogFile(logFile string) {
	// 获取当前时间
	timestamp := time.Now().Format("20060102-150405")

	// 构建新文件名
	newLogFile := fmt.Sprintf("%s.%s", logFile, timestamp)

	// 重命名文件
	if err := os.Rename(logFile, newLogFile); err != nil {
		lm.logger.Error("轮转日志文件失败", "error", err)
		return
	}

	// 清理旧日志文件
	lm.cleanupOldLogFiles(logFile)
}

// cleanupOldLogFiles 清理旧日志文件
func (lm *LogManager) cleanupOldLogFiles(baseLogFile string) {
	// 获取日志文件所在目录
	dir := filepath.Dir(baseLogFile)
	base := filepath.Base(baseLogFile)

	// 读取目录
	files, err := os.ReadDir(dir)
	if err != nil {
		lm.logger.Error("读取日志目录失败", "error", err)
		return
	}

	// 收集匹配的日志文件
	var logFiles []string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), base+".") {
			logFiles = append(logFiles, filepath.Join(dir, file.Name()))
		}
	}

	// 如果文件数量超过限制，删除最旧的文件
	if len(logFiles) > lm.maxLogFiles {
		// 按修改时间排序
		sort.Slice(logFiles, func(i, j int) bool {
			infoI, _ := os.Stat(logFiles[i])
			infoJ, _ := os.Stat(logFiles[j])
			return infoI.ModTime().Before(infoJ.ModTime())
		})

		// 删除最旧的文件
		for i := 0; i < len(logFiles)-lm.maxLogFiles; i++ {
			if err := os.Remove(logFiles[i]); err != nil {
				lm.logger.Error("删除旧日志文件失败", "file", logFiles[i], "error", err)
			}
		}
	}
}
