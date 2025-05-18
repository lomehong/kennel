package health

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/lomehong/kennel/pkg/errors"
)

// NewRestartServiceAction 创建重启服务的修复动作
func NewRestartServiceAction(serviceName string, restartFunc func(ctx context.Context) error) RepairAction {
	return NewSimpleRepairAction(
		fmt.Sprintf("restart_%s", serviceName),
		fmt.Sprintf("重启服务 %s", serviceName),
		func(ctx context.Context) error {
			if restartFunc != nil {
				return restartFunc(ctx)
			}
			return fmt.Errorf("未提供重启函数")
		},
	)
}

// NewRunCommandAction 创建运行命令的修复动作
func NewRunCommandAction(name string, command string, args ...string) RepairAction {
	return NewSimpleRepairAction(
		fmt.Sprintf("run_command_%s", name),
		fmt.Sprintf("运行命令 %s %v", command, args),
		func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, command, args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("命令执行失败: %w, 输出: %s", err, string(output))
			}
			return nil
		},
	)
}

// NewCleanupTempFilesAction 创建清理临时文件的修复动作
func NewCleanupTempFilesAction(directory string, pattern string, olderThan time.Duration) RepairAction {
	return NewSimpleRepairAction(
		fmt.Sprintf("cleanup_temp_files_%s", directory),
		fmt.Sprintf("清理临时文件 %s/%s (超过 %v)", directory, pattern, olderThan),
		func(ctx context.Context) error {
			cutoff := time.Now().Add(-olderThan)
			return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// 检查上下文是否已取消
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				// 跳过目录
				if info.IsDir() {
					return nil
				}

				// 检查文件名是否匹配模式
				matched, err := filepath.Match(pattern, filepath.Base(path))
				if err != nil {
					return err
				}
				if !matched {
					return nil
				}

				// 检查文件是否超过指定时间
				if info.ModTime().Before(cutoff) {
					return os.Remove(path)
				}

				return nil
			})
		},
	)
}

// NewFreeMemoryAction 创建释放内存的修复动作
func NewFreeMemoryAction() RepairAction {
	return NewSimpleRepairAction(
		"free_memory",
		"释放内存",
		func(ctx context.Context) error {
			// 强制进行垃圾回收
			runtime.GC()
			// 释放操作系统持有的内存
			debug.FreeOSMemory()
			return nil
		},
	)
}

// NewRestartComponentAction 创建重启组件的修复动作
func NewRestartComponentAction(componentName string, stopFunc, startFunc func(ctx context.Context) error) RepairAction {
	return NewSimpleRepairAction(
		fmt.Sprintf("restart_component_%s", componentName),
		fmt.Sprintf("重启组件 %s", componentName),
		func(ctx context.Context) error {
			// 停止组件
			if stopFunc != nil {
				if err := stopFunc(ctx); err != nil {
					return fmt.Errorf("停止组件失败: %w", err)
				}
			}

			// 等待一段时间
			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}

			// 启动组件
			if startFunc != nil {
				if err := startFunc(ctx); err != nil {
					return fmt.Errorf("启动组件失败: %w", err)
				}
			}

			return nil
		},
	)
}

// NewRetryAction 创建重试操作的修复动作
func NewRetryAction(name string, operation func(ctx context.Context) error, maxRetries int, delay time.Duration) RepairAction {
	return NewSimpleRepairAction(
		fmt.Sprintf("retry_%s", name),
		fmt.Sprintf("重试操作 %s (最多 %d 次, 延迟 %v)", name, maxRetries, delay),
		func(ctx context.Context) error {
			var lastErr error
			for i := 0; i < maxRetries; i++ {
				// 检查上下文是否已取消
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				// 执行操作
				err := operation(ctx)
				if err == nil {
					return nil
				}
				lastErr = err

				// 如果不是最后一次重试，等待一段时间
				if i < maxRetries-1 {
					select {
					case <-time.After(delay):
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			return fmt.Errorf("重试 %d 次后仍然失败: %w", maxRetries, lastErr)
		},
	)
}

// NewRecoverPanicAction 创建恢复panic的修复动作
func NewRecoverPanicAction(name string, operation func(ctx context.Context) error) RepairAction {
	return NewSimpleRepairAction(
		fmt.Sprintf("recover_panic_%s", name),
		fmt.Sprintf("恢复panic %s", name),
		func(ctx context.Context) error {
			return errors.SafeExecWithContext(ctx, func(ctx context.Context) error {
				return operation(ctx)
			})
		},
	)
}

// NewCreateDirectoryAction 创建目录的修复动作
func NewCreateDirectoryAction(path string, perm os.FileMode) RepairAction {
	return NewSimpleRepairAction(
		fmt.Sprintf("create_directory_%s", path),
		fmt.Sprintf("创建目录 %s", path),
		func(ctx context.Context) error {
			return os.MkdirAll(path, perm)
		},
	)
}

// NewCreateFileAction 创建文件的修复动作
func NewCreateFileAction(path string, content []byte, perm os.FileMode) RepairAction {
	return NewSimpleRepairAction(
		fmt.Sprintf("create_file_%s", path),
		fmt.Sprintf("创建文件 %s", path),
		func(ctx context.Context) error {
			// 确保目录存在
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			return os.WriteFile(path, content, perm)
		},
	)
}

// NewRotateLogAction 创建日志轮转的修复动作
func NewRotateLogAction(logPath string, maxSize int64, keepCount int) RepairAction {
	return NewSimpleRepairAction(
		fmt.Sprintf("rotate_log_%s", logPath),
		fmt.Sprintf("轮转日志 %s (最大大小 %d 字节, 保留 %d 个)", logPath, maxSize, keepCount),
		func(ctx context.Context) error {
			// 检查日志文件是否存在
			info, err := os.Stat(logPath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil // 文件不存在，不需要轮转
				}
				return fmt.Errorf("获取日志文件信息失败: %w", err)
			}

			// 检查文件大小
			if info.Size() < maxSize {
				return nil // 文件大小未超过阈值，不需要轮转
			}

			// 轮转日志文件
			for i := keepCount - 1; i > 0; i-- {
				oldPath := fmt.Sprintf("%s.%d", logPath, i)
				newPath := fmt.Sprintf("%s.%d", logPath, i+1)

				// 检查上下文是否已取消
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				// 如果目标文件存在，则删除
				if _, err := os.Stat(newPath); err == nil {
					if err := os.Remove(newPath); err != nil {
						return fmt.Errorf("删除日志文件失败: %w", err)
					}
				}

				// 如果源文件存在，则重命名
				if _, err := os.Stat(oldPath); err == nil {
					if err := os.Rename(oldPath, newPath); err != nil {
						return fmt.Errorf("重命名日志文件失败: %w", err)
					}
				}
			}

			// 重命名当前日志文件
			newPath := fmt.Sprintf("%s.1", logPath)
			if err := os.Rename(logPath, newPath); err != nil {
				return fmt.Errorf("重命名当前日志文件失败: %w", err)
			}

			// 创建新的日志文件
			file, err := os.Create(logPath)
			if err != nil {
				return fmt.Errorf("创建新日志文件失败: %w", err)
			}
			file.Close()

			return nil
		},
	)
}
