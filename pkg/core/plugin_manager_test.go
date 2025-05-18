package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lomehong/kennel/pkg/plugin"
	"github.com/stretchr/testify/assert"
)

// MockModule 是一个用于测试的模块实现
type MockModule struct {
	InitCalled     bool
	ExecuteCalled  bool
	ShutdownCalled bool
	GetInfoCalled  bool
	InitConfig     map[string]interface{}
	ExecuteAction  string
	ExecuteParams  map[string]interface{}
	ExecuteResult  map[string]interface{}
	ExecuteError   error
	ShutdownError  error
	Info           plugin.ModuleInfo
}

// Init 实现了Module接口的Init方法
func (m *MockModule) Init(config map[string]interface{}) error {
	m.InitCalled = true
	m.InitConfig = config
	return nil
}

// Execute 实现了Module接口的Execute方法
func (m *MockModule) Execute(action string, params map[string]interface{}) (map[string]interface{}, error) {
	m.ExecuteCalled = true
	m.ExecuteAction = action
	m.ExecuteParams = params
	return m.ExecuteResult, m.ExecuteError
}

// Shutdown 实现了Module接口的Shutdown方法
func (m *MockModule) Shutdown() error {
	m.ShutdownCalled = true
	return m.ShutdownError
}

// GetInfo 实现了Module接口的GetInfo方法
func (m *MockModule) GetInfo() plugin.ModuleInfo {
	m.GetInfoCalled = true
	return m.Info
}

// TestNewPluginManager 测试创建插件管理器
func TestNewPluginManager(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "plugin-manager-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建插件管理器
	pm := NewPluginManager(tempDir)

	// 验证插件管理器
	assert.NotNil(t, pm)
	assert.Equal(t, tempDir, pm.pluginDir)
	assert.NotNil(t, pm.plugins)
	assert.NotNil(t, pm.logger)
}

// TestListPlugins 测试列出插件
func TestListPlugins(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "plugin-manager-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建插件管理器
	pm := NewPluginManager(tempDir)

	// 添加一些模拟插件
	pm.plugins["test1"] = &loadedPlugin{
		name: "test1",
		path: filepath.Join(tempDir, "test1.exe"),
		info: plugin.ModuleInfo{
			Name:             "test1",
			Version:          "1.0.0",
			Description:      "Test Plugin 1",
			SupportedActions: []string{"action1", "action2"},
		},
	}

	pm.plugins["test2"] = &loadedPlugin{
		name: "test2",
		path: filepath.Join(tempDir, "test2.exe"),
		info: plugin.ModuleInfo{
			Name:             "test2",
			Version:          "2.0.0",
			Description:      "Test Plugin 2",
			SupportedActions: []string{"action3", "action4"},
		},
	}

	// 列出插件
	plugins := pm.ListPlugins()

	// 验证插件列表
	assert.Equal(t, 2, len(plugins))

	// 验证插件信息
	for _, p := range plugins {
		if p.Name == "test1" {
			assert.Equal(t, "1.0.0", p.Version)
			assert.Equal(t, "Test Plugin 1", p.Description)
			assert.Equal(t, []string{"action1", "action2"}, p.SupportedActions)
		} else if p.Name == "test2" {
			assert.Equal(t, "2.0.0", p.Version)
			assert.Equal(t, "Test Plugin 2", p.Description)
			assert.Equal(t, []string{"action3", "action4"}, p.SupportedActions)
		} else {
			t.Errorf("未知的插件: %s", p.Name)
		}
	}
}

// TestGetPlugin 测试获取插件
func TestGetPlugin(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "plugin-manager-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建插件管理器
	pm := NewPluginManager(tempDir)

	// 创建模拟模块
	mockModule := &MockModule{
		Info: plugin.ModuleInfo{
			Name:             "test",
			Version:          "1.0.0",
			Description:      "Test Plugin",
			SupportedActions: []string{"action1", "action2"},
		},
	}

	// 添加模拟插件
	pm.plugins["test"] = &loadedPlugin{
		name:     "test",
		path:     filepath.Join(tempDir, "test.exe"),
		instance: mockModule,
		info:     mockModule.Info,
	}

	// 获取插件
	module, ok := pm.GetPlugin("test")

	// 验证插件
	assert.True(t, ok)
	assert.Equal(t, mockModule, module)

	// 获取不存在的插件
	module, ok = pm.GetPlugin("nonexistent")

	// 验证结果
	assert.False(t, ok)
	assert.Nil(t, module)
}

// TestClosePlugin 测试关闭插件
func TestClosePlugin(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "plugin-manager-test")
	if err != nil {
		t.Fatalf("无法创建临时目录: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建插件管理器
	pm := NewPluginManager(tempDir)

	// 创建模拟模块
	mockModule := &MockModule{
		Info: plugin.ModuleInfo{
			Name:             "test",
			Version:          "1.0.0",
			Description:      "Test Plugin",
			SupportedActions: []string{"action1", "action2"},
		},
	}

	// 添加模拟插件
	pm.plugins["test"] = &loadedPlugin{
		name:     "test",
		path:     filepath.Join(tempDir, "test.exe"),
		instance: mockModule,
		info:     mockModule.Info,
	}

	// 关闭插件
	err = pm.ClosePlugin("test")

	// 验证结果
	assert.NoError(t, err)
	assert.True(t, mockModule.ShutdownCalled)
	assert.Empty(t, pm.plugins)

	// 关闭不存在的插件
	err = pm.ClosePlugin("nonexistent")

	// 验证结果
	assert.Error(t, err)
}
