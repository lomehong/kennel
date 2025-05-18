package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewApp 测试创建应用程序
func TestNewApp(t *testing.T) {
	// 创建应用程序
	app := NewApp("test.yaml")

	// 验证应用程序
	assert.NotNil(t, app)
	assert.NotNil(t, app.configManager)
	assert.NotNil(t, app.logger)
	assert.False(t, app.running)
}

// TestInitApp 测试初始化应用程序
func TestInitApp(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "app-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建应用程序
	app := NewApp("")

	// 手动设置配置
	app.configManager.Set("plugin_dir", tempDir)
	app.configManager.Set("log_level", "debug")
	app.configManager.Set("log_file", "test.log")
	app.configManager.Set("enable_assets", true)
	app.configManager.Set("enable_device", true)
	app.configManager.Set("enable_dlp", true)
	app.configManager.Set("enable_control", true)

	// 初始化应用程序
	err = app.Init()

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, app.pluginManager)
	assert.Equal(t, tempDir, app.pluginManager.pluginDir)
}

// TestStartStopApp 测试启动和停止应用程序
func TestStartStopApp(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "app-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建应用程序
	app := NewApp("")

	// 手动设置配置
	app.configManager.Set("plugin_dir", tempDir)
	app.configManager.Set("log_level", "debug")
	app.configManager.Set("log_file", "test.log")
	app.configManager.Set("enable_assets", false)
	app.configManager.Set("enable_device", false)
	app.configManager.Set("enable_dlp", false)
	app.configManager.Set("enable_control", false)

	// 初始化应用程序
	err = app.Init()
	assert.NoError(t, err)

	// 启动应用程序
	err = app.Start()
	assert.NoError(t, err)
	assert.True(t, app.IsRunning())

	// 停止应用程序
	app.Stop()
	assert.False(t, app.IsRunning())
}

// TestGetManagers 测试获取管理器
func TestGetManagers(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "app-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建应用程序
	app := NewApp("")

	// 手动设置配置
	app.configManager.Set("plugin_dir", tempDir)
	app.configManager.Set("log_level", "debug")
	app.configManager.Set("log_file", "test.log")
	app.configManager.Set("enable_assets", false)
	app.configManager.Set("enable_device", false)
	app.configManager.Set("enable_dlp", false)
	app.configManager.Set("enable_control", false)

	// 初始化应用程序
	err = app.Init()
	assert.NoError(t, err)

	// 获取管理器
	pluginManager := app.GetPluginManager()
	configManager := app.GetConfigManager()

	// 验证管理器
	assert.NotNil(t, pluginManager)
	assert.NotNil(t, configManager)
	assert.Equal(t, app.pluginManager, pluginManager)
	assert.Equal(t, app.configManager, configManager)
}
