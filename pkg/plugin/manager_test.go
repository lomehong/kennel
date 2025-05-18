package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestPluginManager_LoadPlugin(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewPluginManager(WithPluginManagerLogger(logger))

	// 创建插件配置
	config := &PluginConfig{
		ID:             "test-plugin",
		Name:           "Test Plugin",
		Version:        "1.0.0",
		Path:           "test-plugin",
		IsolationLevel: IsolationLevelBasic,
		AutoStart:      false,
	}

	// 加载插件
	plugin, err := manager.LoadPlugin(config)
	assert.NoError(t, err)
	assert.NotNil(t, plugin)
	assert.Equal(t, "test-plugin", plugin.ID)
	assert.Equal(t, "Test Plugin", plugin.Name)
	assert.Equal(t, "1.0.0", plugin.Version)
	assert.Equal(t, PluginStateInitializing, plugin.State)

	// 尝试加载重复的插件
	_, err = manager.LoadPlugin(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已加载")

	// 获取插件
	loadedPlugin, exists := manager.GetPlugin("test-plugin")
	assert.True(t, exists)
	assert.Equal(t, plugin, loadedPlugin)

	// 列出插件
	plugins := manager.ListPlugins()
	assert.Len(t, plugins, 1)
	assert.Equal(t, plugin, plugins[0])
}

func TestPluginManager_StartStopPlugin(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewPluginManager(WithPluginManagerLogger(logger))

	// 创建插件配置
	config := &PluginConfig{
		ID:             "test-plugin",
		Name:           "Test Plugin",
		Version:        "1.0.0",
		Path:           "test-plugin",
		IsolationLevel: IsolationLevelBasic,
		AutoStart:      false,
	}

	// 加载插件
	plugin, err := manager.LoadPlugin(config)
	assert.NoError(t, err)

	// 启动插件
	err = manager.StartPlugin("test-plugin")
	assert.NoError(t, err)
	assert.Equal(t, PluginStateRunning, plugin.State)
	assert.Equal(t, PluginStateRunning, plugin.Sandbox.GetState())

	// 尝试启动已运行的插件
	err = manager.StartPlugin("test-plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已在运行")

	// 停止插件
	err = manager.StopPlugin("test-plugin")
	assert.NoError(t, err)
	assert.Equal(t, PluginStateStopped, plugin.State)
	assert.Equal(t, PluginStateStopped, plugin.Sandbox.GetState())

	// 尝试停止未运行的插件
	err = manager.StopPlugin("test-plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未在运行")

	// 尝试操作不存在的插件
	err = manager.StartPlugin("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")

	err = manager.StopPlugin("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestPluginManager_RestartPlugin(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewPluginManager(WithPluginManagerLogger(logger))

	// 创建插件配置
	config := &PluginConfig{
		ID:             "test-plugin",
		Name:           "Test Plugin",
		Version:        "1.0.0",
		Path:           "test-plugin",
		IsolationLevel: IsolationLevelBasic,
		AutoStart:      false,
	}

	// 加载并启动插件
	_, err := manager.LoadPlugin(config)
	assert.NoError(t, err)
	err = manager.StartPlugin("test-plugin")
	assert.NoError(t, err)

	// 重启插件
	err = manager.RestartPlugin("test-plugin")
	assert.NoError(t, err)

	// 获取插件
	plugin, exists := manager.GetPlugin("test-plugin")
	assert.True(t, exists)
	assert.Equal(t, PluginStateRunning, plugin.State)
	assert.Equal(t, PluginStateRunning, plugin.Sandbox.GetState())
}

func TestPluginManager_UnloadPlugin(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewPluginManager(WithPluginManagerLogger(logger))

	// 创建插件配置
	config := &PluginConfig{
		ID:             "test-plugin",
		Name:           "Test Plugin",
		Version:        "1.0.0",
		Path:           "test-plugin",
		IsolationLevel: IsolationLevelBasic,
		AutoStart:      false,
	}

	// 加载插件
	_, err := manager.LoadPlugin(config)
	assert.NoError(t, err)

	// 卸载插件
	err = manager.UnloadPlugin("test-plugin")
	assert.NoError(t, err)

	// 验证插件已卸载
	_, exists := manager.GetPlugin("test-plugin")
	assert.False(t, exists)

	// 尝试卸载不存在的插件
	err = manager.UnloadPlugin("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestPluginManager_ExecutePluginFunc(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewPluginManager(WithPluginManagerLogger(logger))

	// 创建插件配置
	config := &PluginConfig{
		ID:             "test-plugin",
		Name:           "Test Plugin",
		Version:        "1.0.0",
		Path:           "test-plugin",
		IsolationLevel: IsolationLevelBasic,
		AutoStart:      true,
	}

	// 加载插件
	_, err := manager.LoadPlugin(config)
	assert.NoError(t, err)

	// 执行函数
	executed := false
	err = manager.ExecutePluginFunc("test-plugin", func() error {
		executed = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, executed)

	// 执行带上下文的函数
	ctx := context.Background()
	executed = false
	err = manager.ExecutePluginFuncWithContext("test-plugin", ctx, func(ctx context.Context) error {
		executed = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, executed)

	// 尝试在不存在的插件中执行函数
	err = manager.ExecutePluginFunc("non-existent", func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不存在")
}

func TestPluginManager_AutoStartPlugin(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewPluginManager(WithPluginManagerLogger(logger))

	// 创建插件配置
	config := &PluginConfig{
		ID:             "test-plugin",
		Name:           "Test Plugin",
		Version:        "1.0.0",
		Path:           "test-plugin",
		IsolationLevel: IsolationLevelBasic,
		AutoStart:      true,
	}

	// 加载插件
	plugin, err := manager.LoadPlugin(config)
	assert.NoError(t, err)

	// 等待自动启动
	time.Sleep(100 * time.Millisecond)

	// 验证插件已启动
	assert.Equal(t, PluginStateRunning, plugin.State)
	assert.Equal(t, PluginStateRunning, plugin.Sandbox.GetState())
}

func TestPluginManager_HealthCheck(t *testing.T) {
	logger := hclog.NewNullLogger()
	manager := NewPluginManager(
		WithPluginManagerLogger(logger),
		WithHealthCheckInterval(100*time.Millisecond),
		WithIdleTimeout(200*time.Millisecond),
	)

	// 创建插件配置
	config := &PluginConfig{
		ID:             "test-plugin",
		Name:           "Test Plugin",
		Version:        "1.0.0",
		Path:           "test-plugin",
		IsolationLevel: IsolationLevelBasic,
		AutoStart:      true,
		AutoRestart:    true,
	}

	// 加载插件
	plugin, err := manager.LoadPlugin(config)
	assert.NoError(t, err)

	// 等待自动启动
	time.Sleep(100 * time.Millisecond)

	// 启动健康检查
	manager.StartHealthCheck()

	// 验证插件已启动
	assert.Equal(t, PluginStateRunning, plugin.State)

	// 等待插件空闲
	time.Sleep(300 * time.Millisecond)

	// 验证插件已暂停
	plugin, exists := manager.GetPlugin("test-plugin")
	assert.True(t, exists)
	assert.Equal(t, PluginStatePaused, plugin.State)
	assert.Equal(t, PluginStatePaused, plugin.Sandbox.GetState())

	// 停止插件管理器
	manager.Stop()
}
