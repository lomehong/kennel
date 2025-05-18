package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lomehong/kennel/pkg/core"
)

// 本示例展示如何在AppFramework中使用配置热更新和动态配置
func main() {
	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config-example")
	if err != nil {
		fmt.Printf("创建临时目录失败: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "config.yaml")
	initialConfig := `
server:
  host: localhost
  port: 8080
  timeout: 30s
database:
  host: localhost
  port: 5432
  username: user
  password: password
  database: mydb
logging:
  level: info
  file: app.log
plugins:
  dir: plugins
  list:
    example-plugin:
      name: Example Plugin
      version: 1.0.0
      path: example-plugin
      isolation_level: basic
      auto_start: true
      auto_restart: true
      enabled: true
health:
  check_interval: 30s
  initial_delay: 5s
  auto_repair: true
  failure_threshold: 3
  success_threshold: 1
  memory_threshold: 80.0
  cpu_threshold: 70.0
  disk_threshold: 90.0
  goroutine_threshold: 1000
`
	err = os.WriteFile(configFile, []byte(initialConfig), 0644)
	if err != nil {
		fmt.Printf("创建配置文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 配置热更新和动态配置使用示例 ===")

	// 创建应用程序实例
	app := core.NewApp(configFile)

	// 初始化应用程序
	if err := app.Init(); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}

	// 示例1: 获取配置值
	fmt.Println("\n=== 示例1: 获取配置值 ===")
	getConfigValues(app)

	// 示例2: 修改配置
	fmt.Println("\n=== 示例2: 修改配置 ===")
	modifyConfig(app)

	// 示例3: 配置热更新
	fmt.Println("\n=== 示例3: 配置热更新 ===")
	hotReloadConfig(app, configFile)

	// 示例4: 配置版本和回滚
	fmt.Println("\n=== 示例4: 配置版本和回滚 ===")
	configVersionAndRollback(app)

	// 示例5: 导出和导入配置
	fmt.Println("\n=== 示例5: 导出和导入配置 ===")
	exportAndImportConfig(app, tempDir)

	// 停止应用程序
	app.Stop()
	fmt.Println("\n应用程序已停止")
}

// 获取配置值
func getConfigValues(app *core.App) {
	// 获取服务器配置
	server := app.GetConfigValue("server").(map[string]interface{})
	fmt.Printf("服务器配置:\n")
	fmt.Printf("- 主机: %s\n", server["host"])
	fmt.Printf("- 端口: %d\n", server["port"])
	fmt.Printf("- 超时: %s\n", server["timeout"])

	// 获取数据库配置
	database := app.GetConfigValue("database").(map[string]interface{})
	fmt.Printf("\n数据库配置:\n")
	fmt.Printf("- 主机: %s\n", database["host"])
	fmt.Printf("- 端口: %d\n", database["port"])
	fmt.Printf("- 用户名: %s\n", database["username"])
	fmt.Printf("- 数据库: %s\n", database["database"])

	// 获取日志配置
	logging := app.GetConfigValue("logging").(map[string]interface{})
	fmt.Printf("\n日志配置:\n")
	fmt.Printf("- 级别: %s\n", logging["level"])
	fmt.Printf("- 文件: %s\n", logging["file"])

	// 获取健康检查配置
	health := app.GetConfigValue("health").(map[string]interface{})
	fmt.Printf("\n健康检查配置:\n")
	fmt.Printf("- 检查间隔: %s\n", health["check_interval"])
	fmt.Printf("- 自动修复: %v\n", health["auto_repair"])
}

// 修改配置
func modifyConfig(app *core.App) {
	// 获取当前配置
	server := app.GetConfigValue("server").(map[string]interface{})
	oldPort := server["port"].(int)
	fmt.Printf("当前服务器端口: %d\n", oldPort)

	// 修改配置
	server["port"] = 9090
	app.SetConfigValue("server", server)
	fmt.Printf("修改服务器端口为: %d\n", 9090)

	// 保存配置
	err := app.SaveConfig()
	if err != nil {
		fmt.Printf("保存配置失败: %v\n", err)
		return
	}
	fmt.Println("配置已保存")

	// 重新加载配置
	err = app.ReloadConfig()
	if err != nil {
		fmt.Printf("重新加载配置失败: %v\n", err)
		return
	}
	fmt.Println("配置已重新加载")

	// 验证配置已更新
	server = app.GetConfigValue("server").(map[string]interface{})
	newPort := server["port"].(int)
	fmt.Printf("更新后的服务器端口: %d\n", newPort)
}

// 配置热更新
func hotReloadConfig(app *core.App, configFile string) {
	// 获取当前日志级别
	logging := app.GetConfigValue("logging").(map[string]interface{})
	oldLevel := logging["level"].(string)
	fmt.Printf("当前日志级别: %s\n", oldLevel)

	// 修改配置文件（外部修改）
	fmt.Println("修改配置文件...")
	updatedConfig := `
server:
  host: localhost
  port: 9090
  timeout: 60s
database:
  host: localhost
  port: 5432
  username: user
  password: password
  database: mydb
logging:
  level: debug
  file: app.log
plugins:
  dir: plugins
  list:
    example-plugin:
      name: Example Plugin
      version: 1.0.0
      path: example-plugin
      isolation_level: basic
      auto_start: true
      auto_restart: true
      enabled: true
health:
  check_interval: 15s
  initial_delay: 5s
  auto_repair: true
  failure_threshold: 3
  success_threshold: 1
  memory_threshold: 80.0
  cpu_threshold: 70.0
  disk_threshold: 90.0
  goroutine_threshold: 1000
`
	err := os.WriteFile(configFile, []byte(updatedConfig), 0644)
	if err != nil {
		fmt.Printf("修改配置文件失败: %v\n", err)
		return
	}

	// 等待配置热更新
	fmt.Println("等待配置热更新...")
	time.Sleep(2 * time.Second)

	// 验证配置已更新
	logging = app.GetConfigValue("logging").(map[string]interface{})
	newLevel := logging["level"].(string)
	fmt.Printf("更新后的日志级别: %s\n", newLevel)

	health := app.GetConfigValue("health").(map[string]interface{})
	checkInterval := health["check_interval"].(string)
	fmt.Printf("更新后的健康检查间隔: %s\n", checkInterval)
}

// 配置版本和回滚
func configVersionAndRollback(app *core.App) {
	// 获取当前配置版本
	version := app.GetConfigVersion()
	fmt.Printf("当前配置版本: %d (时间: %s)\n", version.Version, version.Timestamp.Format(time.RFC3339))

	// 修改配置
	server := app.GetConfigValue("server").(map[string]interface{})
	oldTimeout := server["timeout"].(string)
	fmt.Printf("当前服务器超时: %s\n", oldTimeout)

	// 修改配置
	server["timeout"] = "120s"
	app.SetConfigValue("server", server)
	fmt.Printf("修改服务器超时为: %s\n", "120s")

	// 保存配置
	err := app.SaveConfig()
	if err != nil {
		fmt.Printf("保存配置失败: %v\n", err)
		return
	}
	fmt.Println("配置已保存")

	// 获取新的配置版本
	newVersion := app.GetConfigVersion()
	fmt.Printf("新的配置版本: %d (时间: %s)\n", newVersion.Version, newVersion.Timestamp.Format(time.RFC3339))

	// 回滚配置
	fmt.Printf("回滚到版本 %d...\n", version.Version)
	err = app.RollbackConfig(version.Version)
	if err != nil {
		fmt.Printf("回滚配置失败: %v\n", err)
		return
	}
	fmt.Println("配置已回滚")

	// 验证配置已回滚
	server = app.GetConfigValue("server").(map[string]interface{})
	rolledBackTimeout := server["timeout"].(string)
	fmt.Printf("回滚后的服务器超时: %s\n", rolledBackTimeout)
}

// 导出和导入配置
func exportAndImportConfig(app *core.App, tempDir string) {
	// 导出配置
	exportPath := filepath.Join(tempDir, "exported_config.yaml")
	fmt.Printf("导出配置到: %s\n", exportPath)
	err := app.ExportConfig(exportPath)
	if err != nil {
		fmt.Printf("导出配置失败: %v\n", err)
		return
	}
	fmt.Println("配置已导出")

	// 修改当前配置
	server := app.GetConfigValue("server").(map[string]interface{})
	oldHost := server["host"].(string)
	fmt.Printf("当前服务器主机: %s\n", oldHost)

	server["host"] = "127.0.0.1"
	app.SetConfigValue("server", server)
	fmt.Printf("修改服务器主机为: %s\n", "127.0.0.1")

	// 保存配置
	err = app.SaveConfig()
	if err != nil {
		fmt.Printf("保存配置失败: %v\n", err)
		return
	}
	fmt.Println("配置已保存")

	// 导入配置
	fmt.Printf("导入配置从: %s\n", exportPath)
	err = app.ImportConfig(exportPath)
	if err != nil {
		fmt.Printf("导入配置失败: %v\n", err)
		return
	}
	fmt.Println("配置已导入")

	// 验证配置已导入
	server = app.GetConfigValue("server").(map[string]interface{})
	importedHost := server["host"].(string)
	fmt.Printf("导入后的服务器主机: %s\n", importedHost)
}
