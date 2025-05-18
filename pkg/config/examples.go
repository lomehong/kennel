package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/go-hclog"
)

// 以下是配置热更新和动态配置的使用示例

// ExampleConfigWatcher 展示配置监视器的基本用法
func ExampleConfigWatcher() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "config-watcher",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建配置监视器
	watcher, err := NewConfigWatcher(logger)
	if err != nil {
		logger.Error("创建配置监视器失败", "error", err)
		return
	}
	defer watcher.Stop()

	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config-example")
	if err != nil {
		logger.Error("创建临时目录失败", "error", err)
		return
	}
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configFile, []byte("key: value"), 0644)
	if err != nil {
		logger.Error("创建配置文件失败", "error", err)
		return
	}

	// 添加监视路径
	err = watcher.AddPath(configFile)
	if err != nil {
		logger.Error("添加监视路径失败", "error", err)
		return
	}

	// 添加变更处理器
	watcher.AddHandler(configFile, func(event ChangeEvent) error {
		logger.Info("配置文件变更",
			"path", event.Path,
			"type", event.Type,
			"time", event.Time,
		)
		return nil
	})

	// 启动监视
	watcher.Start()
	logger.Info("配置监视已启动")

	// 修改配置文件
	logger.Info("修改配置文件")
	err = os.WriteFile(configFile, []byte("key: new_value"), 0644)
	if err != nil {
		logger.Error("修改配置文件失败", "error", err)
		return
	}

	// 等待一段时间
	time.Sleep(1 * time.Second)

	// 获取监视的路径
	paths := watcher.GetWatchedPaths()
	logger.Info("监视的路径", "paths", paths)
}

// ExampleDynamicConfig 展示动态配置的基本用法
func ExampleDynamicConfig() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "dynamic-config",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config-example")
	if err != nil {
		logger.Error("创建临时目录失败", "error", err)
		return
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
`
	err = os.WriteFile(configFile, []byte(initialConfig), 0644)
	if err != nil {
		logger.Error("创建配置文件失败", "error", err)
		return
	}

	// 创建配置验证器
	validator := func(config map[string]interface{}) error {
		// 验证服务器配置
		server, ok := config["server"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("缺少服务器配置")
		}

		port, ok := server["port"].(int)
		if !ok {
			return fmt.Errorf("服务器端口应该是整数")
		}

		if port < 1024 || port > 65535 {
			return fmt.Errorf("服务器端口应该在 1024-65535 范围内")
		}

		return nil
	}

	// 创建配置变更监听器
	listener := func(oldConfig, newConfig map[string]interface{}) error {
		logger.Info("配置已变更")

		// 获取旧的服务器配置
		oldServer, _ := oldConfig["server"].(map[string]interface{})
		oldPort, _ := oldServer["port"].(int)

		// 获取新的服务器配置
		newServer, _ := newConfig["server"].(map[string]interface{})
		newPort, _ := newServer["port"].(int)

		// 检查端口是否变更
		if oldPort != newPort {
			logger.Info("服务器端口已变更", "old", oldPort, "new", newPort)
			// 在实际应用中，这里可能需要重启服务器或更新监听端口
		}

		return nil
	}

	// 创建动态配置
	config := NewDynamicConfig(
		configFile,
		logger,
		WithValidator(validator),
		WithListener(listener),
		WithMaxHistorySize(5),
	)

	// 获取配置值
	server := config.Get("server").(map[string]interface{})
	logger.Info("服务器配置",
		"host", server["host"],
		"port", server["port"],
		"timeout", server["timeout"],
	)

	// 修改配置
	serverConfig := config.Get("server").(map[string]interface{})
	serverConfig["port"] = 9090
	config.Set("server", serverConfig)

	// 保存配置
	if err := config.Save(); err != nil {
		logger.Error("保存配置失败", "error", err)
		return
	}

	// 重新加载配置
	if err := config.Reload(); err != nil {
		logger.Error("重新加载配置失败", "error", err)
		return
	}

	// 获取版本信息
	version := config.GetVersion()
	logger.Info("当前配置版本", "version", version.Version, "time", version.Timestamp)

	// 获取所有版本
	versions := config.GetVersions()
	logger.Info("配置版本历史", "count", len(versions))
	for i, v := range versions {
		logger.Info(fmt.Sprintf("版本 %d", i+1), "version", v.Version, "time", v.Timestamp)
	}
}

// ExampleConfigWatcherWithDynamicConfig 展示配置监视器和动态配置结合使用
func ExampleConfigWatcherWithDynamicConfig() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "config-example",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建临时配置文件
	tempDir, err := os.MkdirTemp("", "config-example")
	if err != nil {
		logger.Error("创建临时目录失败", "error", err)
		return
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
`
	err = os.WriteFile(configFile, []byte(initialConfig), 0644)
	if err != nil {
		logger.Error("创建配置文件失败", "error", err)
		return
	}

	// 创建配置监视器
	watcher, err := NewConfigWatcher(logger)
	if err != nil {
		logger.Error("创建配置监视器失败", "error", err)
		return
	}
	defer watcher.Stop()

	// 创建配置变更监听器
	listener := func(oldConfig, newConfig map[string]interface{}) error {
		logger.Info("配置已变更")

		// 获取旧的服务器配置
		oldServer, _ := oldConfig["server"].(map[string]interface{})
		oldPort, _ := oldServer["port"].(int)

		// 获取新的服务器配置
		newServer, _ := newConfig["server"].(map[string]interface{})
		newPort, _ := newServer["port"].(int)

		// 检查端口是否变更
		if oldPort != newPort {
			logger.Info("服务器端口已变更", "old", oldPort, "new", newPort)
			// 在实际应用中，这里可能需要重启服务器或更新监听端口
		}

		return nil
	}

	// 创建动态配置
	config := NewDynamicConfig(
		configFile,
		logger,
		WithWatcher(watcher),
		WithListener(listener),
	)

	// 启动监视
	watcher.Start()
	logger.Info("配置监视已启动")

	// 获取初始配置
	server := config.Get("server").(map[string]interface{})
	logger.Info("初始服务器配置",
		"host", server["host"],
		"port", server["port"],
		"timeout", server["timeout"],
	)

	// 修改配置文件（外部修改）
	logger.Info("修改配置文件")
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
`
	err = os.WriteFile(configFile, []byte(updatedConfig), 0644)
	if err != nil {
		logger.Error("修改配置文件失败", "error", err)
		return
	}

	// 等待配置重新加载
	time.Sleep(1 * time.Second)

	// 获取更新后的配置
	server = config.Get("server").(map[string]interface{})
	logger.Info("更新后的服务器配置",
		"host", server["host"],
		"port", server["port"],
		"timeout", server["timeout"],
	)

	// 获取版本信息
	version := config.GetVersion()
	logger.Info("当前配置版本", "version", version.Version, "time", version.Timestamp)
}
