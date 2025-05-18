package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestNewConfigManager 测试创建配置管理器
func TestNewConfigManager(t *testing.T) {
	// 创建配置管理器
	cm := NewConfigManager("test.yaml")

	// 验证配置管理器
	assert.NotNil(t, cm)
	assert.Equal(t, "test.yaml", cm.configFile)
	assert.NotNil(t, cm.defaults)
}

// TestInitConfig 测试初始化配置
func TestInitConfig(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "config-manager-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建配置文件
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `plugin_dir: plugins
log_level: debug
log_file: test.log
enable_assets: true
enable_device: false`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("无法创建配置文件: %v", err)
	}

	// 创建配置管理器
	cm := NewConfigManager(configPath)

	// 初始化配置
	err = cm.InitConfig()

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, "plugins", cm.GetString("plugin_dir"))
	assert.Equal(t, "debug", cm.GetString("log_level"))
	assert.Equal(t, "test.log", cm.GetString("log_file"))
	assert.True(t, cm.GetBool("enable_assets"))
	assert.False(t, cm.GetBool("enable_device"))
}

// TestCreateDefaultConfig 测试创建默认配置
func TestCreateDefaultConfig(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "config-manager-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建配置文件路径
	configPath := filepath.Join(tempDir, "config.yaml")

	// 创建配置管理器
	cm := NewConfigManager(configPath)

	// 创建默认配置
	err = cm.CreateDefaultConfig()

	// 验证结果
	assert.NoError(t, err)
	assert.FileExists(t, configPath)

	// 读取配置文件
	viper.Reset()
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	assert.NoError(t, err)

	// 验证配置内容
	assert.Equal(t, "plugins", viper.GetString("plugin_dir"))
	assert.Equal(t, "info", viper.GetString("log_level"))
	assert.Equal(t, "agent.log", viper.GetString("log_file"))
	assert.True(t, viper.GetBool("enable_assets"))
	assert.True(t, viper.GetBool("enable_device"))
	assert.True(t, viper.GetBool("enable_dlp"))
	assert.True(t, viper.GetBool("enable_control"))
}

// TestGetSetConfig 测试获取和设置配置
func TestGetSetConfig(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "config-manager-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建配置文件路径
	configPath := filepath.Join(tempDir, "config.yaml")

	// 创建配置管理器
	cm := NewConfigManager(configPath)

	// 设置配置
	cm.Set("test_key", "test_value")
	cm.Set("test_bool", true)
	cm.Set("test_int", 42)
	cm.Set("test_map", map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	})

	// 获取配置
	assert.Equal(t, "test_value", cm.GetString("test_key"))
	assert.True(t, cm.GetBool("test_bool"))
	assert.Equal(t, 42, cm.GetInt("test_int"))
	assert.Equal(t, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}, cm.GetStringMap("test_map"))
}
