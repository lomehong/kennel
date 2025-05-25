//go:build windows

package interceptor

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/lomehong/kennel/pkg/logging"
)

const (
	WinDivertVersion     = "2.2.2"
	WinDivertDownloadURL = "https://github.com/basil00/Divert/releases/download/v2.2.2/WinDivert-2.2.2-A.zip"
	WinDivertInstallDir  = "C:\\Program Files\\WinDivert"
)

// WinDivertInstaller WinDivert安装器
type WinDivertInstaller struct {
	logger      logging.Logger
	installPath string
	version     string
}

// NewWinDivertInstaller 创建WinDivert安装器
func NewWinDivertInstaller(logger logging.Logger) *WinDivertInstaller {
	return &WinDivertInstaller{
		logger:      logger,
		installPath: WinDivertInstallDir,
		version:     WinDivertVersion,
	}
}

// CheckInstallation 检查WinDivert是否已安装
func (w *WinDivertInstaller) CheckInstallation() (bool, error) {
	// 检查关键文件是否存在
	requiredFiles := []string{
		"WinDivert.dll",
		"windivert.h",
		"WinDivert64.sys",
	}

	for _, file := range requiredFiles {
		filePath := filepath.Join(w.installPath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			w.logger.Debug("WinDivert文件不存在", "file", filePath)
			return false, nil
		}
	}

	// 尝试加载DLL验证
	dll := syscall.NewLazyDLL(filepath.Join(w.installPath, "WinDivert.dll"))
	if err := dll.Load(); err != nil {
		w.logger.Debug("无法加载WinDivert.dll", "error", err)
		return false, nil
	}

	w.logger.Info("WinDivert已正确安装", "path", w.installPath)
	return true, nil
}

// InstallWinDivert 安装WinDivert
func (w *WinDivertInstaller) InstallWinDivert() error {
	w.logger.Info("开始安装WinDivert", "version", w.version)

	// 检查管理员权限
	if !w.isAdmin() {
		return fmt.Errorf("安装WinDivert需要管理员权限")
	}

	// 创建临时目录
	tempDir := filepath.Join(os.TempDir(), "windivert-install")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 下载WinDivert
	zipFile := filepath.Join(tempDir, "windivert.zip")
	if err := w.downloadFile(WinDivertDownloadURL, zipFile); err != nil {
		return fmt.Errorf("下载WinDivert失败: %w", err)
	}

	// 解压文件
	extractDir := filepath.Join(tempDir, "extracted")
	if err := w.extractZip(zipFile, extractDir); err != nil {
		return fmt.Errorf("解压WinDivert失败: %w", err)
	}

	// 安装文件
	if err := w.installFiles(extractDir); err != nil {
		return fmt.Errorf("安装WinDivert文件失败: %w", err)
	}

	w.logger.Info("WinDivert安装完成", "path", w.installPath)
	return nil
}

// downloadFile 下载文件
func (w *WinDivertInstaller) downloadFile(url, filepath string) error {
	w.logger.Info("正在下载WinDivert", "url", url)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractZip 解压ZIP文件
func (w *WinDivertInstaller) extractZip(src, dest string) error {
	w.logger.Info("正在解压WinDivert文件")

	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	os.MkdirAll(dest, 0755)

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}

		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.FileInfo().Mode())
			rc.Close()
			continue
		}

		os.MkdirAll(filepath.Dir(path), 0755)
		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// installFiles 安装文件到目标目录
func (w *WinDivertInstaller) installFiles(extractDir string) error {
	w.logger.Info("正在安装WinDivert文件到系统目录")

	// 创建安装目录
	if err := os.MkdirAll(w.installPath, 0755); err != nil {
		return err
	}

	// 确定架构
	arch := "x64"
	if runtime.GOARCH == "386" {
		arch = "x86"
	}

	sourceDir := filepath.Join(extractDir, fmt.Sprintf("WinDivert-%s-A", w.version), arch)

	// 要复制的文件
	filesToCopy := []string{
		"WinDivert.dll",
		"WinDivert.lib",
		"WinDivert.sys",
		"WinDivert32.sys",
		"WinDivert64.sys",
	}

	for _, file := range filesToCopy {
		srcPath := filepath.Join(sourceDir, file)
		dstPath := filepath.Join(w.installPath, file)

		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			w.logger.Warn("源文件不存在，跳过", "file", file)
			continue
		}

		if err := w.copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("复制文件 %s 失败: %w", file, err)
		}

		w.logger.Debug("已复制文件", "file", file)
	}

	// 复制头文件
	headerSrc := filepath.Join(extractDir, fmt.Sprintf("WinDivert-%s-A", w.version), "include", "windivert.h")
	headerDst := filepath.Join(w.installPath, "windivert.h")
	if _, err := os.Stat(headerSrc); err == nil {
		w.copyFile(headerSrc, headerDst)
	}

	return nil
}

// copyFile 复制文件
func (w *WinDivertInstaller) copyFile(src, dst string) error {
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
	return err
}

// isAdmin 检查是否具有管理员权限
func (w *WinDivertInstaller) isAdmin() bool {
	// 简化的管理员权限检查
	// 尝试在系统目录创建文件来测试权限
	testFile := filepath.Join(w.installPath, "test_admin_access.tmp")

	// 确保目录存在
	os.MkdirAll(w.installPath, 0755)

	file, err := os.Create(testFile)
	if err != nil {
		w.logger.Debug("管理员权限检查失败", "error", err)
		return false
	}

	file.Close()
	os.Remove(testFile)

	w.logger.Debug("管理员权限检查通过")
	return true
}

// GetInstallationInfo 获取安装信息
func (w *WinDivertInstaller) GetInstallationInfo() map[string]interface{} {
	info := make(map[string]interface{})

	installed, _ := w.CheckInstallation()
	info["installed"] = installed
	info["version"] = w.version
	info["install_path"] = w.installPath
	info["architecture"] = runtime.GOARCH

	if installed {
		// 获取文件信息
		files := make(map[string]interface{})
		requiredFiles := []string{"WinDivert.dll", "WinDivert.sys"}

		for _, file := range requiredFiles {
			filePath := filepath.Join(w.installPath, file)
			if stat, err := os.Stat(filePath); err == nil {
				files[file] = map[string]interface{}{
					"size":     stat.Size(),
					"mod_time": stat.ModTime(),
				}
			}
		}
		info["files"] = files
	}

	return info
}

// AutoInstallIfNeeded 如果需要则自动安装
func (w *WinDivertInstaller) AutoInstallIfNeeded() error {
	installed, err := w.CheckInstallation()
	if err != nil {
		return err
	}

	if !installed {
		w.logger.Warn("WinDivert未安装，尝试自动安装")

		if !w.isAdmin() {
			return fmt.Errorf("WinDivert未安装且当前进程没有管理员权限，请以管理员身份运行或手动安装WinDivert")
		}

		return w.InstallWinDivert()
	}

	return nil
}
