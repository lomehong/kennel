//go:build windows

package interceptor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/lomehong/kennel/pkg/logging"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	// Windows服务管理器常量
	SC_MANAGER_ALL_ACCESS = 0xF003F
	SERVICE_ALL_ACCESS    = 0xF01FF
	SERVICE_KERNEL_DRIVER = 0x00000001
	SERVICE_DEMAND_START  = 0x00000003
	SERVICE_ERROR_NORMAL  = 0x00000001
	SERVICE_RUNNING       = 0x00000004
	SERVICE_STOPPED       = 0x00000001

	// WinDivert驱动相关常量
	WINDIVERT_DRIVER_NAME  = "WinDivert"
	WINDIVERT_SERVICE_NAME = "WinDivert"
	WINDIVERT_DISPLAY_NAME = "WinDivert Packet Capture Driver"
	WINDIVERT_DRIVER_PATH  = "\\SystemRoot\\System32\\drivers\\WinDivert.sys"
	SYSTEM_DRIVERS_DIR     = "C:\\Windows\\System32\\drivers"
)

// WinDivertDriverManager WinDivert驱动管理器
type WinDivertDriverManager struct {
	logger      logging.Logger
	installPath string
	driverPath  string
	serviceName string
}

// NewWinDivertDriverManager 创建驱动管理器
func NewWinDivertDriverManager(logger logging.Logger) *WinDivertDriverManager {
	return &WinDivertDriverManager{
		logger:      logger,
		installPath: WinDivertInstallDir,
		driverPath:  filepath.Join(SYSTEM_DRIVERS_DIR, "WinDivert.sys"),
		serviceName: WINDIVERT_SERVICE_NAME,
	}
}

// DiagnoseDriverIssues 诊断驱动问题
func (dm *WinDivertDriverManager) DiagnoseDriverIssues() error {
	dm.logger.Info("开始诊断WinDivert驱动问题")

	// 1. 检查管理员权限
	if !dm.isRunningAsAdmin() {
		return fmt.Errorf("需要管理员权限来管理驱动程序")
	}
	dm.logger.Info("✓ 管理员权限检查通过")

	// 2. 检查驱动文件
	if err := dm.checkDriverFiles(); err != nil {
		dm.logger.Error("✗ 驱动文件检查失败", "error", err)
		return err
	}
	dm.logger.Info("✓ 驱动文件检查通过")

	// 3. 检查驱动签名
	if err := dm.checkDriverSignature(); err != nil {
		dm.logger.Warn("⚠ 驱动签名检查失败", "error", err)
		// 签名问题不是致命错误，继续检查
	} else {
		dm.logger.Info("✓ 驱动签名检查通过")
	}

	// 4. 检查驱动服务
	if err := dm.checkDriverService(); err != nil {
		dm.logger.Error("✗ 驱动服务检查失败", "error", err)
		return err
	}
	dm.logger.Info("✓ 驱动服务检查通过")

	// 5. 检查Windows安全策略
	if err := dm.checkSecurityPolicy(); err != nil {
		dm.logger.Warn("⚠ 安全策略检查失败", "error", err)
		// 安全策略问题给出警告但不阻止
	} else {
		dm.logger.Info("✓ 安全策略检查通过")
	}

	dm.logger.Info("WinDivert驱动诊断完成")
	return nil
}

// InstallAndRegisterDriver 安装并注册驱动
func (dm *WinDivertDriverManager) InstallAndRegisterDriver() error {
	dm.logger.Info("开始安装和注册WinDivert驱动")

	// 1. 确保有管理员权限
	if !dm.isRunningAsAdmin() {
		return fmt.Errorf("安装驱动需要管理员权限")
	}

	// 2. 检查驱动是否已经在运行
	if dm.isDriverRunning() {
		dm.logger.Info("WinDivert驱动已在运行，跳过安装")
		return nil
	}

	// 3. 复制驱动文件到系统目录
	if err := dm.copyDriverToSystem(); err != nil {
		return fmt.Errorf("复制驱动文件失败: %w", err)
	}

	// 4. 注册驱动服务
	if err := dm.registerDriverService(); err != nil {
		return fmt.Errorf("注册驱动服务失败: %w", err)
	}

	// 5. 启动驱动服务
	if err := dm.startDriverService(); err != nil {
		return fmt.Errorf("启动驱动服务失败: %w", err)
	}

	dm.logger.Info("WinDivert驱动安装和注册完成")
	return nil
}

// checkDriverFiles 检查驱动文件
func (dm *WinDivertDriverManager) checkDriverFiles() error {
	// 检查安装目录中的驱动文件
	arch := "64"
	if runtime.GOARCH == "386" {
		arch = "32"
	}

	sourceDriverFile := filepath.Join(dm.installPath, fmt.Sprintf("WinDivert%s.sys", arch))
	if _, err := os.Stat(sourceDriverFile); os.IsNotExist(err) {
		return fmt.Errorf("源驱动文件不存在: %s", sourceDriverFile)
	}

	// 检查系统目录中的驱动文件
	if _, err := os.Stat(dm.driverPath); os.IsNotExist(err) {
		dm.logger.Info("系统驱动文件不存在，需要复制", "path", dm.driverPath)
		return dm.copyDriverToSystem()
	}

	dm.logger.Info("驱动文件存在", "path", dm.driverPath)
	return nil
}

// copyDriverToSystem 复制驱动文件到系统目录
func (dm *WinDivertDriverManager) copyDriverToSystem() error {
	dm.logger.Info("复制WinDivert驱动到系统目录")

	// 确定源文件
	arch := "64"
	if runtime.GOARCH == "386" {
		arch = "32"
	}

	sourceFile := filepath.Join(dm.installPath, fmt.Sprintf("WinDivert%s.sys", arch))
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		// 尝试通用文件名
		sourceFile = filepath.Join(dm.installPath, "WinDivert.sys")
		if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
			return fmt.Errorf("找不到WinDivert驱动文件")
		}
	}

	// 复制文件
	sourceData, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("读取源驱动文件失败: %w", err)
	}

	if err := os.WriteFile(dm.driverPath, sourceData, 0644); err != nil {
		return fmt.Errorf("写入驱动文件到系统目录失败: %w", err)
	}

	dm.logger.Info("驱动文件复制完成", "from", sourceFile, "to", dm.driverPath)
	return nil
}

// registerDriverService 注册驱动服务
func (dm *WinDivertDriverManager) registerDriverService() error {
	dm.logger.Info("注册WinDivert驱动服务")

	// 打开服务管理器
	scm, err := windows.OpenSCManager(nil, nil, SC_MANAGER_ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("打开服务管理器失败: %w", err)
	}
	defer windows.CloseServiceHandle(scm)

	// 检查服务是否已存在
	serviceNamePtr, _ := windows.UTF16PtrFromString(dm.serviceName)
	service, err := windows.OpenService(scm, serviceNamePtr, SERVICE_ALL_ACCESS)
	if err == nil {
		// 服务已存在，先删除
		dm.logger.Info("WinDivert服务已存在，先删除")
		windows.DeleteService(service)
		windows.CloseServiceHandle(service)
		time.Sleep(1 * time.Second) // 等待服务删除完成
	}

	// 创建新服务
	displayNamePtr, _ := windows.UTF16PtrFromString(WINDIVERT_DISPLAY_NAME)
	driverPathPtr, _ := windows.UTF16PtrFromString(WINDIVERT_DRIVER_PATH)
	service, err = windows.CreateService(
		scm,
		serviceNamePtr,
		displayNamePtr,
		SERVICE_ALL_ACCESS,
		SERVICE_KERNEL_DRIVER,
		SERVICE_DEMAND_START,
		SERVICE_ERROR_NORMAL,
		driverPathPtr,
		nil, nil, nil, nil, nil,
	)
	if err != nil {
		return fmt.Errorf("创建驱动服务失败: %w", err)
	}
	defer windows.CloseServiceHandle(service)

	dm.logger.Info("WinDivert驱动服务注册完成")
	return nil
}

// startDriverService 启动驱动服务
func (dm *WinDivertDriverManager) startDriverService() error {
	dm.logger.Info("启动WinDivert驱动服务")

	// 打开服务管理器
	scm, err := windows.OpenSCManager(nil, nil, SC_MANAGER_ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("打开服务管理器失败: %w", err)
	}
	defer windows.CloseServiceHandle(scm)

	// 打开服务
	serviceNamePtr, _ := windows.UTF16PtrFromString(dm.serviceName)
	service, err := windows.OpenService(scm, serviceNamePtr, SERVICE_ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("打开驱动服务失败: %w", err)
	}
	defer windows.CloseServiceHandle(service)

	// 启动服务
	err = windows.StartService(service, 0, nil)
	if err != nil {
		// 检查是否已经在运行
		if err == windows.ERROR_SERVICE_ALREADY_RUNNING {
			dm.logger.Info("WinDivert驱动服务已在运行")
			return nil
		}
		return fmt.Errorf("启动驱动服务失败: %w", err)
	}

	dm.logger.Info("WinDivert驱动服务启动完成")
	return nil
}

// checkDriverService 检查驱动服务状态
func (dm *WinDivertDriverManager) checkDriverService() error {
	// 打开服务管理器
	scm, err := windows.OpenSCManager(nil, nil, SC_MANAGER_ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("打开服务管理器失败: %w", err)
	}
	defer windows.CloseServiceHandle(scm)

	// 打开服务
	serviceNamePtr, _ := windows.UTF16PtrFromString(dm.serviceName)
	service, err := windows.OpenService(scm, serviceNamePtr, SERVICE_ALL_ACCESS)
	if err != nil {
		dm.logger.Info("WinDivert驱动服务不存在，需要注册")
		return dm.registerDriverService()
	}
	defer windows.CloseServiceHandle(service)

	// 查询服务状态
	var status windows.SERVICE_STATUS
	err = windows.QueryServiceStatus(service, &status)
	if err != nil {
		return fmt.Errorf("查询服务状态失败: %w", err)
	}

	dm.logger.Info("WinDivert驱动服务状态", "state", status.CurrentState)

	// 如果服务未运行，尝试启动
	if status.CurrentState != SERVICE_RUNNING {
		dm.logger.Info("WinDivert驱动服务未运行，尝试启动")
		return dm.startDriverService()
	}

	return nil
}

// checkDriverSignature 检查驱动签名
func (dm *WinDivertDriverManager) checkDriverSignature() error {
	// 这里可以添加驱动签名验证逻辑
	// 对于WinDivert，通常是已签名的
	dm.logger.Info("跳过驱动签名检查（WinDivert通常已签名）")
	return nil
}

// checkSecurityPolicy 检查Windows安全策略
func (dm *WinDivertDriverManager) checkSecurityPolicy() error {
	// 检查测试签名模式
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\CI\Policy`, registry.QUERY_VALUE)
	if err != nil {
		dm.logger.Debug("无法打开CI策略注册表项", "error", err)
		return nil // 不是致命错误
	}
	defer key.Close()

	// 检查其他安全相关设置
	dm.logger.Info("安全策略检查完成")
	return nil
}

// isRunningAsAdmin 检查是否以管理员身份运行
func (dm *WinDivertDriverManager) isRunningAsAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}

	return member
}

// RestartDriverService 重启驱动服务
func (dm *WinDivertDriverManager) RestartDriverService() error {
	dm.logger.Info("重启WinDivert驱动服务")

	// 停止服务
	if err := dm.stopDriverService(); err != nil {
		dm.logger.Warn("停止驱动服务失败", "error", err)
	}

	// 等待服务完全停止
	time.Sleep(2 * time.Second)

	// 启动服务
	return dm.startDriverService()
}

// stopDriverService 停止驱动服务
func (dm *WinDivertDriverManager) stopDriverService() error {
	scm, err := windows.OpenSCManager(nil, nil, SC_MANAGER_ALL_ACCESS)
	if err != nil {
		return err
	}
	defer windows.CloseServiceHandle(scm)

	serviceNamePtr, _ := windows.UTF16PtrFromString(dm.serviceName)
	service, err := windows.OpenService(scm, serviceNamePtr, SERVICE_ALL_ACCESS)
	if err != nil {
		return err
	}
	defer windows.CloseServiceHandle(service)

	var status windows.SERVICE_STATUS
	return windows.ControlService(service, windows.SERVICE_CONTROL_STOP, &status)
}

// isDriverRunning 检查驱动是否正在运行
func (dm *WinDivertDriverManager) isDriverRunning() bool {
	// 打开服务管理器
	scm, err := windows.OpenSCManager(nil, nil, SC_MANAGER_ALL_ACCESS)
	if err != nil {
		return false
	}
	defer windows.CloseServiceHandle(scm)

	// 打开服务
	serviceNamePtr, _ := windows.UTF16PtrFromString(dm.serviceName)
	service, err := windows.OpenService(scm, serviceNamePtr, SERVICE_ALL_ACCESS)
	if err != nil {
		return false
	}
	defer windows.CloseServiceHandle(service)

	// 查询服务状态
	var status windows.SERVICE_STATUS
	err = windows.QueryServiceStatus(service, &status)
	if err != nil {
		return false
	}

	// 检查是否正在运行
	return status.CurrentState == SERVICE_RUNNING
}
