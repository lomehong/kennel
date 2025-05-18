package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// LogRotator 日志轮转器
type LogRotator struct {
	filePath   string        // 日志文件路径
	maxSize    int64         // 日志文件最大大小（字节）
	maxBackups int           // 日志文件最大备份数量
	maxAge     time.Duration // 日志文件最大保留时间
	size       int64         // 当前文件大小
	file       *os.File      // 当前文件
	mu         sync.Mutex    // 互斥锁
}

// NewLogRotator 创建一个新的日志轮转器
func NewLogRotator(filePath string, maxSize int64, maxBackups int, maxAge time.Duration) *LogRotator {
	return &LogRotator{
		filePath:   filePath,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
	}
}

// Write 实现io.Writer接口
func (r *LogRotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 如果文件未打开，则打开文件
	if r.file == nil {
		if err := r.openFile(); err != nil {
			return 0, err
		}
	}

	// 写入数据
	n, err = r.file.Write(p)
	if err != nil {
		return n, err
	}

	// 更新文件大小
	r.size += int64(n)

	// 检查是否需要轮转
	if r.size >= r.maxSize {
		if err := r.rotate(); err != nil {
			return n, err
		}
	}

	return n, nil
}

// Close 关闭日志轮转器
func (r *LogRotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 关闭文件
	if r.file != nil {
		err := r.file.Close()
		r.file = nil
		return err
	}

	return nil
}

// openFile 打开日志文件
func (r *LogRotator) openFile() error {
	// 确保目录存在
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 打开文件
	file, err := os.OpenFile(r.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	// 获取文件信息
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("获取日志文件信息失败: %w", err)
	}

	// 设置文件和大小
	r.file = file
	r.size = info.Size()

	return nil
}

// rotate 轮转日志文件
func (r *LogRotator) rotate() error {
	// 关闭当前文件
	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return fmt.Errorf("关闭日志文件失败: %w", err)
		}
		r.file = nil
	}

	// 轮转文件
	if err := r.rotateFile(); err != nil {
		return fmt.Errorf("轮转日志文件失败: %w", err)
	}

	// 清理旧文件
	if err := r.cleanOldFiles(); err != nil {
		return fmt.Errorf("清理旧日志文件失败: %w", err)
	}

	// 重新打开文件
	if err := r.openFile(); err != nil {
		return fmt.Errorf("重新打开日志文件失败: %w", err)
	}

	return nil
}

// rotateFile 轮转文件
func (r *LogRotator) rotateFile() error {
	// 生成新文件名
	timestamp := time.Now().Format("20060102-150405")
	ext := filepath.Ext(r.filePath)
	base := strings.TrimSuffix(r.filePath, ext)
	newPath := fmt.Sprintf("%s.%s%s", base, timestamp, ext)

	// 重命名文件
	if err := os.Rename(r.filePath, newPath); err != nil {
		return fmt.Errorf("重命名日志文件失败: %w", err)
	}

	return nil
}

// cleanOldFiles 清理旧文件
func (r *LogRotator) cleanOldFiles() error {
	// 获取目录和文件名
	dir := filepath.Dir(r.filePath)
	base := filepath.Base(r.filePath)
	ext := filepath.Ext(base)
	prefix := strings.TrimSuffix(base, ext)

	// 读取目录
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("读取日志目录失败: %w", err)
	}

	// 筛选匹配的文件
	var backups []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		if strings.HasPrefix(fileName, prefix) && fileName != base && strings.HasSuffix(fileName, ext) {
			backups = append(backups, filepath.Join(dir, fileName))
		}
	}

	// 按修改时间排序
	sort.Slice(backups, func(i, j int) bool {
		infoI, _ := os.Stat(backups[i])
		infoJ, _ := os.Stat(backups[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// 删除超过最大备份数量的文件
	if r.maxBackups > 0 && len(backups) > r.maxBackups {
		for i := r.maxBackups; i < len(backups); i++ {
			if err := os.Remove(backups[i]); err != nil {
				return fmt.Errorf("删除旧日志文件失败: %w", err)
			}
		}
		backups = backups[:r.maxBackups]
	}

	// 删除超过最大保留时间的文件
	if r.maxAge > 0 {
		cutoff := time.Now().Add(-r.maxAge)
		for _, backup := range backups {
			info, err := os.Stat(backup)
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				if err := os.Remove(backup); err != nil {
					return fmt.Errorf("删除过期日志文件失败: %w", err)
				}
			}
		}
	}

	return nil
}

// MultiWriter 多输出写入器
type MultiWriter struct {
	writers []io.Writer
}

// NewMultiWriter 创建一个新的多输出写入器
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write 实现io.Writer接口
func (w *MultiWriter) Write(p []byte) (n int, err error) {
	for _, writer := range w.writers {
		n, err = writer.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}

// Close 关闭多输出写入器
func (w *MultiWriter) Close() error {
	var lastErr error
	for _, writer := range w.writers {
		if closer, ok := writer.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}
