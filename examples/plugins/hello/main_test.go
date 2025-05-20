package main

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"github.com/lomehong/kennel/pkg/plugin/testing"
	"github.com/stretchr/testify/assert"
)

func TestHelloPlugin(t *testing.T) {
	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建插件
	plugin := NewHelloPlugin(logger)

	// 创建测试套件
	suite := testing.NewPluginTestSuite(t, plugin)

	// 设置配置
	suite.SetConfig("message", "Hello, Test!")
	suite.SetConfig("interval", 1)
	suite.SetConfig("debug_port", 8081)
	suite.SetConfig("debug_enabled", false)

	// 运行测试
	suite.Run(func(s *testing.TestSuite) {
		// 检查插件信息
		info := plugin.GetInfo()
		assert.Equal(t, "hello", info.ID)
		assert.Equal(t, "Hello Plugin", info.Name)
		assert.Equal(t, "1.0.0", info.Version)

		// 检查插件配置
		assert.Equal(t, "Hello, Test!", plugin.config.Message)
		assert.Equal(t, 1*time.Second, plugin.config.Interval)
		assert.Equal(t, 8081, plugin.config.DebugPort)
		assert.Equal(t, false, plugin.config.DebugEnabled)

		// 检查插件状态
		assert.True(t, plugin.running)

		// 执行健康检查
		health, err := plugin.HealthCheck(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "healthy", health.Status)
		assert.Equal(t, true, health.Details["running"])
	})
}

func TestHelloPluginIntegration(t *testing.T) {
	// 创建集成测试套件
	suite := testing.NewIntegrationTestSuite(t)

	// 创建插件配置
	err := suite.CreatePluginConfig("hello", map[string]interface{}{
		"name":        "Hello Plugin",
		"version":     "1.0.0",
		"description": "A simple hello plugin",
		"author":      "Example Author",
		"license":     "MIT",
		"tags":        []string{"example", "hello"},
		"capabilities": map[string]bool{
			"hello": true,
		},
		"settings": map[string]interface{}{
			"message":       "Hello, Integration!",
			"interval":      1,
			"debug_port":    8082,
			"debug_enabled": false,
		},
	})
	assert.NoError(t, err)

	// 构建插件
	sourcePath := "."
	targetPath := suite.CreateTempDir("plugins/hello") + "/hello"
	err = suite.BuildPlugin("hello", sourcePath, targetPath)
	assert.NoError(t, err)

	// 启动插件进程
	err = suite.StartPluginProcess("hello", targetPath)
	assert.NoError(t, err)

	// 运行测试
	suite.Run(func(s *testing.IntegrationTestSuite) {
		// 加载插件
		s.AssertLoadPluginSuccess("hello")

		// 启动插件
		s.AssertStartPluginSuccess("hello")

		// 等待插件状态
		err := s.WaitForPluginState("hello", api.PluginStateRunning, 5*time.Second)
		assert.NoError(t, err)

		// 检查插件健康
		s.AssertPluginHealthy("hello")

		// 停止插件
		s.AssertStopPluginSuccess("hello")

		// 等待插件状态
		err = s.WaitForPluginState("hello", api.PluginStateStopped, 5*time.Second)
		assert.NoError(t, err)

		// 卸载插件
		s.AssertUnloadPluginSuccess("hello")
	})
}
