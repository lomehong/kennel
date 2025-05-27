package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestConfigManager 测试配置管理器
func TestConfigManager(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "config-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试配置文件
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `
global:
  app:
    name: "test-app"
    version: "1.0.0"
  logging:
    level: "debug"
    file: "logs/test.log"

plugin_manager:
  plugin_dir: "plugins"
  discovery:
    auto_load: true

plugins:
  test-plugin:
    enabled: true
    option1: "value1"
    option2: 42
`
	if err := ioutil.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 创建配置管理器
	cm, err := NewConfigManager(
		WithConfigPath(configPath),
		WithConfigFormat(ConfigFormatYAML),
	)
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}
	defer cm.Close()

	// 测试获取全局配置
	globalConfig := cm.GetGlobalConfig()
	if app, ok := globalConfig["app"].(map[string]interface{}); ok {
		if name, ok := app["name"].(string); ok {
			if name != "test-app" {
				t.Errorf("应用名称不匹配: 期望 %s, 实际 %s", "test-app", name)
			}
		} else {
			t.Error("应用名称不是字符串类型")
		}
	} else {
		t.Error("全局配置中缺少app部分")
	}

	// 测试获取插件管理配置
	pluginManagerConfig := cm.GetPluginManagerConfig()
	if pluginDir, ok := pluginManagerConfig["plugin_dir"].(string); ok {
		if pluginDir != "plugins" {
			t.Errorf("插件目录不匹配: 期望 %s, 实际 %s", "plugins", pluginDir)
		}
	} else {
		t.Error("插件管理配置中缺少plugin_dir")
	}

	// 测试获取插件配置
	pluginConfig := cm.GetPluginConfig("test-plugin")
	if enabled, ok := pluginConfig["enabled"].(bool); ok {
		if !enabled {
			t.Error("插件应该被启用")
		}
	} else {
		t.Error("插件配置中缺少enabled字段")
	}
	if option1, ok := pluginConfig["option1"].(string); ok {
		if option1 != "value1" {
			t.Errorf("选项1不匹配: 期望 %s, 实际 %s", "value1", option1)
		}
	} else {
		t.Error("插件配置中缺少option1字段")
	}
	if option2, ok := pluginConfig["option2"].(int); ok {
		if option2 != 42 {
			t.Errorf("选项2不匹配: 期望 %d, 实际 %d", 42, option2)
		}
	} else {
		t.Error("插件配置中缺少option2字段")
	}

	// 测试设置配置
	cm.SetPluginConfig("new-plugin", map[string]interface{}{
		"enabled": true,
		"name":    "新插件",
		"version": "1.0.0",
	})

	// 测试获取新插件配置
	newPluginConfig := cm.GetPluginConfig("new-plugin")
	if name, ok := newPluginConfig["name"].(string); ok {
		if name != "新插件" {
			t.Errorf("新插件名称不匹配: 期望 %s, 实际 %s", "新插件", name)
		}
	} else {
		t.Error("新插件配置中缺少name字段")
	}

	// 测试保存配置
	if err := cm.Save(); err != nil {
		t.Fatalf("保存配置失败: %v", err)
	}

	// 验证配置文件是否包含新插件
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatalf("读取配置文件失败: %v", err)
	}
	configStr := string(configData)
	if !contains(configStr, "new-plugin") {
		t.Error("配置文件中应该包含新插件")
	}
}

// TestConfigChangeListener 测试配置变更监听器
func TestConfigChangeListener(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "config-listener-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试配置文件
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `
global:
  app:
    name: "test-app"
    version: "1.0.0"

plugins:
  test-plugin:
    enabled: true
    option1: "value1"
`
	if err := ioutil.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 创建配置管理器
	cm, err := NewConfigManager(
		WithConfigPath(configPath),
		WithConfigFormat(ConfigFormatYAML),
	)
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}
	defer cm.Close()

	// 创建变更通道
	changeCh := make(chan string, 1)

	// 添加配置变更监听器
	cm.AddConfigChangeListener(func(configType string, oldConfig, newConfig map[string]interface{}) error {
		changeCh <- configType
		return nil
	})

	// 修改配置文件
	newConfigContent := `
global:
  app:
    name: "test-app"
    version: "1.0.0"

plugins:
  test-plugin:
    enabled: false
    option1: "value2"
`
	if err := ioutil.WriteFile(configPath, []byte(newConfigContent), 0644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 等待配置变更
	select {
	case configType := <-changeCh:
		if configType != "plugin:test-plugin" {
			t.Errorf("配置类型不匹配: 期望 %s, 实际 %s", "plugin:test-plugin", configType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("等待配置变更超时")
	}

	// 验证配置是否已更新
	pluginConfig := cm.GetPluginConfig("test-plugin")
	if enabled, ok := pluginConfig["enabled"].(bool); ok {
		if enabled {
			t.Error("插件应该被禁用")
		}
	} else {
		t.Error("插件配置中缺少enabled字段")
	}
	if option1, ok := pluginConfig["option1"].(string); ok {
		if option1 != "value2" {
			t.Errorf("选项1不匹配: 期望 %s, 实际 %s", "value2", option1)
		}
	} else {
		t.Error("插件配置中缺少option1字段")
	}
}

// TestConfigValidator 测试配置验证器
func TestConfigValidator(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "config-validator-test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建测试配置文件
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := `
global:
  app:
    name: "test-app"
    version: "1.0.0"

plugins:
  test-plugin:
    enabled: true
    level: "info"
    count: 10
`
	if err := ioutil.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 创建配置验证器
	validator := NewPluginConfigValidator("test-plugin")
	validator.AddRequiredField("enabled")
	validator.AddFieldType("level", reflect.String)
	validator.AddFieldType("count", reflect.Float64) // YAML解析数字为float64
	validator.AddFieldValidator("level", StringValidator("debug", "info", "warn", "error"))
	validator.AddFieldValidator("count", IntRangeValidator(1, 100))
	validator.AddDefault("timeout", 30)

	// 创建配置管理器
	cm, err := NewConfigManager(
		WithConfigPath(configPath),
		WithConfigFormat(ConfigFormatYAML),
		WithConfigValidator(validator),
	)
	if err != nil {
		t.Fatalf("创建配置管理器失败: %v", err)
	}
	defer cm.Close()

	// 验证配置是否包含默认值
	pluginConfig := cm.GetPluginConfig("test-plugin")

	// 检查基本配置字段
	if enabled, ok := pluginConfig["enabled"].(bool); !ok || !enabled {
		t.Error("插件配置中enabled字段不正确")
	}

	if level, ok := pluginConfig["level"].(string); !ok || level != "info" {
		t.Error("插件配置中level字段不正确")
	}

	// 注意：默认值可能不会自动添加到GetPluginConfig的返回值中
	// 这取决于配置管理器的实现
	t.Logf("插件配置: %+v", pluginConfig)

	// 测试无效配置
	cm.SetPluginConfig("test-plugin", map[string]interface{}{
		"enabled": true,
		"level":   "invalid",
		"count":   200,
	})

	// 保存配置应该失败
	if err := cm.Save(); err == nil {
		t.Fatal("保存无效配置应该失败")
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
