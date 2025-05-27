//go:build selfprotect
// +build selfprotect

package selfprotect

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/go-hclog"
)

// FileProtectorImpl 文件防护器实现
type FileProtectorImpl struct {
	config FileProtectionConfig
	logger hclog.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	enabled        bool
	protectedFiles map[string]*ProtectedFile
	protectedDirs  map[string]*ProtectedDir
	eventCallback  EventCallback

	// 文件监控
	watcher    *fsnotify.Watcher
	monitoring bool

	// 文件完整性
	checksums map[string]FileChecksum
}

// ProtectedFile 受保护的文件信息
type ProtectedFile struct {
	Path         string
	OriginalPath string
	BackupPath   string
	Checksum     FileChecksum
	Protected    bool
	LastCheck    time.Time
	Attributes   FileAttributes
}

// ProtectedDir 受保护的目录信息
type ProtectedDir struct {
	Path      string
	Recursive bool
	Protected bool
	LastCheck time.Time
	FileCount int
}

// FileChecksum 文件校验和
type FileChecksum struct {
	MD5     string    `json:"md5"`
	SHA256  string    `json:"sha256"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// FileAttributes 文件属性
type FileAttributes struct {
	ReadOnly bool        `json:"read_only"`
	Hidden   bool        `json:"hidden"`
	System   bool        `json:"system"`
	Archive  bool        `json:"archive"`
	ModTime  time.Time   `json:"mod_time"`
	Size     int64       `json:"size"`
	Mode     os.FileMode `json:"mode"`
}

// NewFileProtector 创建文件防护器
func NewFileProtector(config FileProtectionConfig, logger hclog.Logger) FileProtector {
	ctx, cancel := context.WithCancel(context.Background())

	return &FileProtectorImpl{
		config:         config,
		logger:         logger.Named("file-protector"),
		ctx:            ctx,
		cancel:         cancel,
		enabled:        config.Enabled,
		protectedFiles: make(map[string]*ProtectedFile),
		protectedDirs:  make(map[string]*ProtectedDir),
		checksums:      make(map[string]FileChecksum),
	}
}

// Start 启动文件防护
func (fp *FileProtectorImpl) Start(ctx context.Context) error {
	if !fp.enabled {
		return nil
	}

	fp.logger.Info("启动文件防护")

	// 创建文件监控器
	var err error
	fp.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建文件监控器失败: %w", err)
	}

	// 创建备份目录
	if fp.config.BackupEnabled && fp.config.BackupDir != "" {
		if err := os.MkdirAll(fp.config.BackupDir, 0755); err != nil {
			fp.logger.Warn("创建备份目录失败", "dir", fp.config.BackupDir, "error", err)
		}
	}

	// 保护指定的文件
	for _, filePath := range fp.config.ProtectedFiles {
		if err := fp.ProtectFile(filePath); err != nil {
			fp.logger.Error("保护文件失败", "file", filePath, "error", err)
		}
	}

	// 保护指定的目录
	for _, dirPath := range fp.config.ProtectedDirs {
		if err := fp.protectDirectory(dirPath); err != nil {
			fp.logger.Error("保护目录失败", "dir", dirPath, "error", err)
		}
	}

	// 启动文件监控
	fp.monitoring = true
	fp.wg.Add(1)
	go fp.monitorFiles()

	return nil
}

// Stop 停止文件防护
func (fp *FileProtectorImpl) Stop() error {
	fp.logger.Info("停止文件防护")

	fp.monitoring = false
	fp.cancel()

	if fp.watcher != nil {
		fp.watcher.Close()
	}

	fp.wg.Wait()

	return nil
}

// IsEnabled 检查是否启用
func (fp *FileProtectorImpl) IsEnabled() bool {
	return fp.enabled
}

// PeriodicCheck 定期检查
func (fp *FileProtectorImpl) PeriodicCheck() error {
	if !fp.enabled || !fp.monitoring {
		return nil
	}

	// 检查受保护文件的完整性
	fp.mu.RLock()
	files := make([]*ProtectedFile, 0, len(fp.protectedFiles))
	for _, file := range fp.protectedFiles {
		files = append(files, file)
	}
	fp.mu.RUnlock()

	for _, file := range files {
		if fp.config.CheckIntegrity {
			if valid, err := fp.CheckFileIntegrity(file.Path); err != nil {
				fp.logger.Error("检查文件完整性失败", "file", file.Path, "error", err)
			} else if !valid {
				fp.logger.Warn("文件完整性验证失败", "file", file.Path)

				// 记录事件
				if fp.eventCallback != nil {
					fp.eventCallback(ProtectionEvent{
						Type:        ProtectionTypeFile,
						Action:      "integrity_violation",
						Target:      file.Path,
						Description: fmt.Sprintf("文件 %s 完整性验证失败", file.Path),
						Details: map[string]interface{}{
							"file_path": file.Path,
						},
					})
				}

				// 尝试恢复文件
				if fp.config.BackupEnabled {
					if err := fp.RestoreFile(file.Path); err != nil {
						fp.logger.Error("恢复文件失败", "file", file.Path, "error", err)
					}
				}
			}
		}
	}

	return nil
}

// SetEventCallback 设置事件回调
func (fp *FileProtectorImpl) SetEventCallback(callback EventCallback) {
	fp.eventCallback = callback
}

// ProtectFile 保护文件
func (fp *FileProtectorImpl) ProtectFile(filePath string) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// 获取绝对路径
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 检查文件是否存在
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			fp.logger.Warn("文件不存在", "file", absPath)
			// 仍然添加到保护列表，以便监控文件创建
			fp.protectedFiles[absPath] = &ProtectedFile{
				Path:         absPath,
				OriginalPath: filePath,
				Protected:    true,
				LastCheck:    time.Now(),
			}
			return nil
		}
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 计算文件校验和
	checksum, err := fp.calculateFileChecksum(absPath)
	if err != nil {
		return fmt.Errorf("计算文件校验和失败: %w", err)
	}

	// 获取文件属性
	attributes, err := fp.getFileAttributes(fileInfo)
	if err != nil {
		return fmt.Errorf("获取文件属性失败: %w", err)
	}

	// 备份文件
	var backupPath string
	if fp.config.BackupEnabled {
		backupPath, err = fp.BackupFile(absPath)
		if err != nil {
			fp.logger.Warn("备份文件失败", "file", absPath, "error", err)
		}
	}

	// 添加到保护列表
	fp.protectedFiles[absPath] = &ProtectedFile{
		Path:         absPath,
		OriginalPath: filePath,
		BackupPath:   backupPath,
		Checksum:     checksum,
		Protected:    true,
		LastCheck:    time.Now(),
		Attributes:   attributes,
	}

	// 添加到文件监控
	if fp.watcher != nil {
		if err := fp.watcher.Add(absPath); err != nil {
			fp.logger.Warn("添加文件监控失败", "file", absPath, "error", err)
		}
	}

	fp.logger.Info("文件已保护", "file", absPath)

	// 记录事件
	if fp.eventCallback != nil {
		fp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeFile,
			Action:      "protect",
			Target:      absPath,
			Description: fmt.Sprintf("文件 %s 已被保护", absPath),
			Details: map[string]interface{}{
				"file_path":   absPath,
				"file_size":   checksum.Size,
				"backup_path": backupPath,
			},
		})
	}

	return nil
}

// UnprotectFile 取消保护文件
func (fp *FileProtectorImpl) UnprotectFile(filePath string) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %w", err)
	}

	_, exists := fp.protectedFiles[absPath]
	if !exists {
		return fmt.Errorf("文件未受保护: %s", absPath)
	}

	// 从文件监控移除
	if fp.watcher != nil {
		fp.watcher.Remove(absPath)
	}

	// 从保护列表移除
	delete(fp.protectedFiles, absPath)

	fp.logger.Info("取消文件保护", "file", absPath)

	// 记录事件
	if fp.eventCallback != nil {
		fp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeFile,
			Action:      "unprotect",
			Target:      absPath,
			Description: fmt.Sprintf("取消保护文件 %s", absPath),
		})
	}

	return nil
}

// IsFileProtected 检查文件是否受保护
func (fp *FileProtectorImpl) IsFileProtected(filePath string) bool {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	_, exists := fp.protectedFiles[absPath]
	return exists
}

// GetProtectedFiles 获取受保护的文件列表
func (fp *FileProtectorImpl) GetProtectedFiles() []string {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	files := make([]string, 0, len(fp.protectedFiles))
	for path := range fp.protectedFiles {
		files = append(files, path)
	}
	return files
}

// CheckFileIntegrity 检查文件完整性
func (fp *FileProtectorImpl) CheckFileIntegrity(filePath string) (bool, error) {
	fp.mu.RLock()
	file, exists := fp.protectedFiles[filePath]
	fp.mu.RUnlock()

	if !exists {
		return false, fmt.Errorf("文件未受保护: %s", filePath)
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	// 计算当前文件校验和
	currentChecksum, err := fp.calculateFileChecksum(filePath)
	if err != nil {
		return false, err
	}

	// 比较校验和
	if currentChecksum.MD5 != file.Checksum.MD5 ||
		currentChecksum.SHA256 != file.Checksum.SHA256 ||
		currentChecksum.Size != file.Checksum.Size {
		return false, nil
	}

	return true, nil
}

// BackupFile 备份文件
func (fp *FileProtectorImpl) BackupFile(filePath string) (string, error) {
	if !fp.config.BackupEnabled || fp.config.BackupDir == "" {
		return "", fmt.Errorf("备份功能未启用")
	}

	// 检查源文件是否存在
	if _, err := os.Stat(filePath); err != nil {
		return "", fmt.Errorf("源文件不存在: %w", err)
	}

	// 生成备份文件路径
	fileName := filepath.Base(filePath)
	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("%s.%s.backup", fileName, timestamp)
	backupPath := filepath.Join(fp.config.BackupDir, backupFileName)

	// 复制文件
	if err := fp.copyFile(filePath, backupPath); err != nil {
		return "", fmt.Errorf("复制文件失败: %w", err)
	}

	fp.logger.Info("文件已备份", "source", filePath, "backup", backupPath)
	return backupPath, nil
}

// RestoreFile 恢复文件
func (fp *FileProtectorImpl) RestoreFile(filePath string) error {
	fp.mu.RLock()
	file, exists := fp.protectedFiles[filePath]
	fp.mu.RUnlock()

	if !exists {
		return fmt.Errorf("文件未受保护: %s", filePath)
	}

	if file.BackupPath == "" {
		return fmt.Errorf("文件无备份: %s", filePath)
	}

	// 检查备份文件是否存在
	if _, err := os.Stat(file.BackupPath); err != nil {
		return fmt.Errorf("备份文件不存在: %w", err)
	}

	// 恢复文件
	if err := fp.copyFile(file.BackupPath, filePath); err != nil {
		return fmt.Errorf("恢复文件失败: %w", err)
	}

	fp.logger.Info("文件已恢复", "file", filePath, "backup", file.BackupPath)

	// 记录事件
	if fp.eventCallback != nil {
		fp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeFile,
			Action:      "restore",
			Target:      filePath,
			Description: fmt.Sprintf("文件 %s 已从备份恢复", filePath),
			Details: map[string]interface{}{
				"file_path":   filePath,
				"backup_path": file.BackupPath,
			},
		})
	}

	return nil
}

// MonitorFileChanges 监控文件变更
func (fp *FileProtectorImpl) MonitorFileChanges(filePath string) error {
	if fp.watcher == nil {
		return fmt.Errorf("文件监控器未初始化")
	}

	return fp.watcher.Add(filePath)
}

// protectDirectory 保护目录
func (fp *FileProtectorImpl) protectDirectory(dirPath string) error {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 检查目录是否存在
	dirInfo, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("获取目录信息失败: %w", err)
	}

	if !dirInfo.IsDir() {
		return fmt.Errorf("路径不是目录: %s", absPath)
	}

	// 添加到保护列表
	fp.protectedDirs[absPath] = &ProtectedDir{
		Path:      absPath,
		Recursive: true,
		Protected: true,
		LastCheck: time.Now(),
	}

	// 添加到文件监控
	if fp.watcher != nil {
		if err := fp.watcher.Add(absPath); err != nil {
			fp.logger.Warn("添加目录监控失败", "dir", absPath, "error", err)
		}
	}

	// 保护目录中的所有文件
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if err := fp.ProtectFile(path); err != nil {
				fp.logger.Warn("保护目录文件失败", "file", path, "error", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("遍历目录失败: %w", err)
	}

	fp.logger.Info("目录已保护", "dir", absPath)
	return nil
}

// calculateFileChecksum 计算文件校验和
func (fp *FileProtectorImpl) calculateFileChecksum(filePath string) (FileChecksum, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return FileChecksum{}, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return FileChecksum{}, err
	}

	md5Hash := md5.New()
	sha256Hash := sha256.New()

	multiWriter := io.MultiWriter(md5Hash, sha256Hash)
	if _, err := io.Copy(multiWriter, file); err != nil {
		return FileChecksum{}, err
	}

	return FileChecksum{
		MD5:     fmt.Sprintf("%x", md5Hash.Sum(nil)),
		SHA256:  fmt.Sprintf("%x", sha256Hash.Sum(nil)),
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime(),
	}, nil
}

// getFileAttributes 获取文件属性
func (fp *FileProtectorImpl) getFileAttributes(fileInfo os.FileInfo) (FileAttributes, error) {
	return FileAttributes{
		ModTime: fileInfo.ModTime(),
		Size:    fileInfo.Size(),
		Mode:    fileInfo.Mode(),
	}, nil
}

// copyFile 复制文件
func (fp *FileProtectorImpl) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// 复制文件权限
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// monitorFiles 监控文件变更
func (fp *FileProtectorImpl) monitorFiles() {
	defer fp.wg.Done()

	for {
		select {
		case <-fp.ctx.Done():
			return
		case event, ok := <-fp.watcher.Events:
			if !ok {
				return
			}
			fp.handleFileEvent(event)
		case err, ok := <-fp.watcher.Errors:
			if !ok {
				return
			}
			fp.logger.Error("文件监控错误", "error", err)
		}
	}
}

// handleFileEvent 处理文件事件
func (fp *FileProtectorImpl) handleFileEvent(event fsnotify.Event) {
	fp.logger.Debug("文件事件", "event", event.String())

	// 检查是否是受保护的文件
	if !fp.IsFileProtected(event.Name) {
		return
	}

	var action string
	var blocked bool

	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		action = "modify"
		blocked = fp.handleFileModification(event.Name)
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		action = "delete"
		blocked = fp.handleFileDeletion(event.Name)
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		action = "rename"
		blocked = fp.handleFileRename(event.Name)
	case event.Op&fsnotify.Chmod == fsnotify.Chmod:
		action = "chmod"
		blocked = fp.handleFilePermissionChange(event.Name)
	default:
		return
	}

	// 记录事件
	if fp.eventCallback != nil {
		fp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeFile,
			Action:      action,
			Target:      event.Name,
			Blocked:     blocked,
			Description: fmt.Sprintf("检测到文件 %s 操作: %s", event.Name, action),
			Details: map[string]interface{}{
				"file_path": event.Name,
				"operation": event.Op.String(),
			},
		})
	}
}

// handleFileModification 处理文件修改
func (fp *FileProtectorImpl) handleFileModification(filePath string) bool {
	fp.logger.Warn("检测到受保护文件被修改", "file", filePath)

	// 检查文件完整性
	if valid, err := fp.CheckFileIntegrity(filePath); err != nil {
		fp.logger.Error("检查文件完整性失败", "file", filePath, "error", err)
		return false
	} else if !valid {
		fp.logger.Warn("文件完整性验证失败，尝试恢复", "file", filePath)

		// 尝试恢复文件
		if fp.config.BackupEnabled {
			if err := fp.RestoreFile(filePath); err != nil {
				fp.logger.Error("恢复文件失败", "file", filePath, "error", err)
				return false
			}
			return true
		}
	}

	return false
}

// handleFileDeletion 处理文件删除
func (fp *FileProtectorImpl) handleFileDeletion(filePath string) bool {
	fp.logger.Warn("检测到受保护文件被删除", "file", filePath)

	// 尝试恢复文件
	if fp.config.BackupEnabled {
		if err := fp.RestoreFile(filePath); err != nil {
			fp.logger.Error("恢复被删除的文件失败", "file", filePath, "error", err)
			return false
		}
		return true
	}

	return false
}

// handleFileRename 处理文件重命名
func (fp *FileProtectorImpl) handleFileRename(filePath string) bool {
	fp.logger.Warn("检测到受保护文件被重命名", "file", filePath)

	// 对于重命名，我们可以尝试恢复原文件
	if fp.config.BackupEnabled {
		if err := fp.RestoreFile(filePath); err != nil {
			fp.logger.Error("恢复被重命名的文件失败", "file", filePath, "error", err)
			return false
		}
		return true
	}

	return false
}

// handleFilePermissionChange 处理文件权限变更
func (fp *FileProtectorImpl) handleFilePermissionChange(filePath string) bool {
	fp.logger.Warn("检测到受保护文件权限被修改", "file", filePath)

	// 对于权限变更，我们可以尝试恢复原始权限
	fp.mu.RLock()
	file, exists := fp.protectedFiles[filePath]
	fp.mu.RUnlock()

	if exists && file.Attributes.Mode != 0 {
		if err := os.Chmod(filePath, file.Attributes.Mode); err != nil {
			fp.logger.Error("恢复文件权限失败", "file", filePath, "error", err)
			return false
		}
		return true
	}

	return false
}
