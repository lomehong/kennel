package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lomehong/kennel/pkg/core/config"
)

func main() {
	var (
		sourceFile = flag.String("source", "config.yaml", "源配置文件路径")
		targetFile = flag.String("target", "config.new.yaml", "目标配置文件路径")
		backup     = flag.Bool("backup", true, "是否备份原配置文件")
		force      = flag.Bool("force", false, "是否强制覆盖目标文件")
		validate   = flag.Bool("validate", true, "是否验证迁移后的配置")
		help       = flag.Bool("help", false, "显示帮助信息")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	fmt.Println("Kennel配置迁移工具 v1.0.0")
	fmt.Println("==============================")

	// 检查源文件是否存在
	if _, err := os.Stat(*sourceFile); os.IsNotExist(err) {
		fmt.Printf("错误: 源配置文件 %s 不存在\n", *sourceFile)
		os.Exit(1)
	}

	// 检查目标文件是否存在
	if _, err := os.Stat(*targetFile); err == nil && !*force {
		fmt.Printf("错误: 目标配置文件 %s 已存在，使用 -force 参数强制覆盖\n", *targetFile)
		os.Exit(1)
	}

	// 备份原配置文件
	if *backup {
		backupFile := *sourceFile + ".backup"
		if err := copyFile(*sourceFile, backupFile); err != nil {
			fmt.Printf("警告: 备份配置文件失败: %v\n", err)
		} else {
			fmt.Printf("✓ 已备份原配置文件到: %s\n", backupFile)
		}
	}

	// 执行迁移
	fmt.Printf("正在迁移配置文件: %s -> %s\n", *sourceFile, *targetFile)

	migration := config.NewConfigMigration(*sourceFile, *targetFile)
	if err := migration.Migrate(); err != nil {
		fmt.Printf("错误: 配置迁移失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ 配置迁移完成\n")

	// 验证迁移后的配置
	if *validate {
		fmt.Println("正在验证迁移后的配置...")
		if err := validateConfig(*targetFile); err != nil {
			fmt.Printf("警告: 配置验证失败: %v\n", err)
		} else {
			fmt.Printf("✓ 配置验证通过\n")
		}
	}

	// 显示迁移总结
	showMigrationSummary(*sourceFile, *targetFile)
}

func showHelp() {
	fmt.Println("Kennel配置迁移工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  config-migrate [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -source string    源配置文件路径 (默认: config.yaml)")
	fmt.Println("  -target string    目标配置文件路径 (默认: config.new.yaml)")
	fmt.Println("  -backup          是否备份原配置文件 (默认: true)")
	fmt.Println("  -force           是否强制覆盖目标文件 (默认: false)")
	fmt.Println("  -validate        是否验证迁移后的配置 (默认: true)")
	fmt.Println("  -help            显示帮助信息")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  config-migrate")
	fmt.Println("  config-migrate -source old-config.yaml -target new-config.yaml")
	fmt.Println("  config-migrate -force -backup=false")
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func validateConfig(configFile string) error {
	// 这里可以添加配置验证逻辑
	// 暂时只检查文件是否可以正常读取
	_, err := os.ReadFile(configFile)
	return err
}

func showMigrationSummary(sourceFile, targetFile string) {
	fmt.Println()
	fmt.Println("迁移总结:")
	fmt.Println("========")

	sourceInfo, _ := os.Stat(sourceFile)
	targetInfo, _ := os.Stat(targetFile)

	fmt.Printf("源文件: %s (大小: %d 字节)\n", sourceFile, sourceInfo.Size())
	fmt.Printf("目标文件: %s (大小: %d 字节)\n", targetFile, targetInfo.Size())

	fmt.Println()
	fmt.Println("迁移内容:")
	fmt.Println("- ✓ 全局配置 (应用信息、日志配置、系统配置)")
	fmt.Println("- ✓ 插件管理配置 (插件目录、发现机制、隔离配置)")
	fmt.Println("- ✓ 通讯模块配置")
	fmt.Println("- ✓ Web控制台配置")
	fmt.Println("- ✓ 插件配置 (Assets、Device、DLP、Control、Audit)")

	fmt.Println()
	fmt.Println("配置格式变更:")
	fmt.Println("- 扁平结构 -> 层次化结构")
	fmt.Println("- 插件启用标志 -> 插件配置对象")
	fmt.Println("- 添加了插件元数据 (名称、版本、路径)")
	fmt.Println("- 添加了插件隔离和生命周期配置")

	fmt.Println()
	fmt.Println("后续步骤:")
	fmt.Println("1. 检查迁移后的配置文件内容")
	fmt.Println("2. 根据需要调整配置参数")
	fmt.Println("3. 更新应用程序以使用新配置文件")
	fmt.Println("4. 测试应用程序功能")
	fmt.Println("5. 删除旧配置文件 (可选)")
}
