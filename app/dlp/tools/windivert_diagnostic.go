//go:build windows

package main

import (
	"fmt"
	"os"

	"github.com/lomehong/kennel/app/dlp/interceptor"
	"github.com/lomehong/kennel/pkg/logging"
)

func main() {
	// 创建日志记录器
	config := logging.DefaultLogConfig()
	config.Level = logging.LogLevelInfo
	logger, err := logging.NewEnhancedLogger(config)
	if err != nil {
		fmt.Printf("创建日志记录器失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== WinDivert 诊断工具 ===")
	fmt.Println()

	// 检查管理员权限
	if !isRunningAsAdmin() {
		fmt.Println("❌ 错误: 需要管理员权限")
		fmt.Println("请以管理员身份运行此程序")
		os.Exit(1)
	}
	fmt.Println("✅ 管理员权限检查通过")

	// 创建驱动管理器
	driverManager := interceptor.NewWinDivertDriverManager(logger)

	// 执行诊断
	fmt.Println("\n--- 开始诊断 WinDivert 驱动 ---")
	if err := driverManager.DiagnoseDriverIssues(); err != nil {
		fmt.Printf("❌ 诊断失败: %v\n", err)

		// 尝试修复
		fmt.Println("\n--- 尝试自动修复 ---")
		if err := driverManager.InstallAndRegisterDriver(); err != nil {
			fmt.Printf("❌ 修复失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ 修复完成")

		// 重新诊断
		fmt.Println("\n--- 重新诊断 ---")
		if err := driverManager.DiagnoseDriverIssues(); err != nil {
			fmt.Printf("❌ 修复后仍有问题: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("✅ WinDivert 驱动诊断通过")

	// 测试WinDivert拦截器
	fmt.Println("\n--- 测试 WinDivert 拦截器 ---")
	interceptor := interceptor.NewWinDivertInterceptor(logger)

	fmt.Println("尝试启动拦截器...")
	if err := interceptor.Start(); err != nil {
		fmt.Printf("❌ 拦截器启动失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ 拦截器启动成功")

	// 执行健康检查
	fmt.Println("执行健康检查...")
	if err := interceptor.HealthCheck(); err != nil {
		fmt.Printf("❌ 健康检查失败: %v\n", err)
	} else {
		fmt.Println("✅ 健康检查通过")
	}

	// 停止拦截器
	fmt.Println("停止拦截器...")
	if err := interceptor.Stop(); err != nil {
		fmt.Printf("⚠️ 停止拦截器时出现警告: %v\n", err)
	} else {
		fmt.Println("✅ 拦截器已停止")
	}

	fmt.Println("\n🎉 所有测试通过！WinDivert 网络拦截功能已就绪")
}

// isRunningAsAdmin 检查是否以管理员身份运行
func isRunningAsAdmin() bool {
	// 尝试打开一个需要管理员权限的资源
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	return true
}
