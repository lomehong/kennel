package testing

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"github.com/lomehong/kennel/pkg/plugin/registry"
	"github.com/stretchr/testify/assert"
)

// ManagerTestSuite 管理器测试套件
// 提供插件管理器测试功能
type ManagerTestSuite struct {
	// 测试对象
	t *testing.T

	// 插件管理器
	manager *plugin.PluginManagerV3

	// 插件注册表
	registry registry.PluginRegistry

	// 日志记录器
	logger hclog.Logger

	// 临时目录
	tempDir string

	// 上下文
	ctx context.Context

	// 取消函数
	cancel context.CancelFunc

	// 清理函数
	cleanups []func()

	// 模拟插件
	mockPlugins map[string]*MockPlugin
}

// NewManagerTestSuite 创建一个新的管理器测试套件
func NewManagerTestSuite(t *testing.T) *ManagerTestSuite {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "manager-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "manager-test",
		Level:  hclog.Debug,
		Output: os.Stdout,
	})

	// 创建插件目录
	pluginsDir := filepath.Join(tempDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatalf("创建插件目录失败: %v", err)
	}

	// 创建管理器配置
	config := plugin.DefaultManagerConfigV3()
	config.PluginsDir = pluginsDir
	config.AutoDiscover = false
	config.AutoStart = false

	// 创建插件管理器
	manager := plugin.NewPluginManagerV3(logger, config)

	return &ManagerTestSuite{
		t:           t,
		manager:     manager,
		registry:    manager.GetRegistry(),
		logger:      logger,
		tempDir:     tempDir,
		ctx:         ctx,
		cancel:      cancel,
		cleanups:    make([]func(), 0),
		mockPlugins: make(map[string]*MockPlugin),
	}
}

// AddMockPlugin 添加模拟插件
func (s *ManagerTestSuite) AddMockPlugin(id string) *MockPlugin {
	// 创建模拟插件
	mockPlugin := NewMockPlugin(id)

	// 注册插件元数据
	metadata := api.PluginMetadata{
		ID:          id,
		Name:        mockPlugin.info.Name,
		Version:     mockPlugin.info.Version,
		Description: mockPlugin.info.Description,
		Author:      mockPlugin.info.Author,
		License:     mockPlugin.info.License,
		Tags:        mockPlugin.info.Tags,
		Capabilities: mockPlugin.info.Capabilities,
		Dependencies: mockPlugin.info.Dependencies,
		Location: api.PluginLocation{
			Type: "memory",
			Path: id,
		},
	}

	// 注册插件
	if err := s.registry.RegisterPlugin(metadata); err != nil {
		s.t.Fatalf("注册插件失败: %v", err)
	}

	// 存储模拟插件
	s.mockPlugins[id] = mockPlugin

	return mockPlugin
}

// AddCleanup 添加清理函数
func (s *ManagerTestSuite) AddCleanup(cleanup func()) *ManagerTestSuite {
	s.cleanups = append(s.cleanups, cleanup)
	return s
}

// CreateTempFile 创建临时文件
func (s *ManagerTestSuite) CreateTempFile(name string, content []byte) string {
	path := filepath.Join(s.tempDir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		s.t.Fatalf("创建临时文件目录失败: %v", err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		s.t.Fatalf("创建临时文件失败: %v", err)
	}

	return path
}

// CreateTempDir 创建临时目录
func (s *ManagerTestSuite) CreateTempDir(name string) string {
	path := filepath.Join(s.tempDir, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		s.t.Fatalf("创建临时目录失败: %v", err)
	}

	return path
}

// GetTempDir 获取临时目录
func (s *ManagerTestSuite) GetTempDir() string {
	return s.tempDir
}

// GetLogger 获取日志记录器
func (s *ManagerTestSuite) GetLogger() hclog.Logger {
	return s.logger
}

// GetContext 获取上下文
func (s *ManagerTestSuite) GetContext() context.Context {
	return s.ctx
}

// GetManager 获取插件管理器
func (s *ManagerTestSuite) GetManager() *plugin.PluginManagerV3 {
	return s.manager
}

// GetRegistry 获取插件注册表
func (s *ManagerTestSuite) GetRegistry() registry.PluginRegistry {
	return s.registry
}

// GetMockPlugin 获取模拟插件
func (s *ManagerTestSuite) GetMockPlugin(id string) *MockPlugin {
	return s.mockPlugins[id]
}

// Start 启动管理器
func (s *ManagerTestSuite) Start() error {
	return s.manager.Start()
}

// Stop 停止管理器
func (s *ManagerTestSuite) Stop() error {
	return s.manager.Stop()
}

// LoadPlugin 加载插件
func (s *ManagerTestSuite) LoadPlugin(id string) (api.Plugin, error) {
	// 获取模拟插件
	mockPlugin, exists := s.mockPlugins[id]
	if !exists {
		return s.manager.LoadPlugin(id)
	}

	// 注入模拟插件
	s.manager.InjectPlugin(id, mockPlugin)

	return mockPlugin, nil
}

// Run 运行测试
func (s *ManagerTestSuite) Run(fn func(*ManagerTestSuite)) {
	defer s.Cleanup()

	// 启动管理器
	if err := s.Start(); err != nil {
		s.t.Fatalf("启动管理器失败: %v", err)
	}

	// 运行测试函数
	fn(s)

	// 停止管理器
	if err := s.Stop(); err != nil {
		s.t.Fatalf("停止管理器失败: %v", err)
	}
}

// Cleanup 清理资源
func (s *ManagerTestSuite) Cleanup() {
	// 执行清理函数
	for _, cleanup := range s.cleanups {
		cleanup()
	}

	// 取消上下文
	s.cancel()

	// 删除临时目录
	os.RemoveAll(s.tempDir)
}

// AssertStartSuccess 断言启动成功
func (s *ManagerTestSuite) AssertStartSuccess() *ManagerTestSuite {
	err := s.Start()
	assert.NoError(s.t, err, "启动管理器应该成功")
	return s
}

// AssertStopSuccess 断言停止成功
func (s *ManagerTestSuite) AssertStopSuccess() *ManagerTestSuite {
	err := s.Stop()
	assert.NoError(s.t, err, "停止管理器应该成功")
	return s
}

// AssertLoadPluginSuccess 断言加载插件成功
func (s *ManagerTestSuite) AssertLoadPluginSuccess(id string) *ManagerTestSuite {
	plugin, err := s.LoadPlugin(id)
	assert.NoError(s.t, err, "加载插件应该成功")
	assert.NotNil(s.t, plugin, "插件不应该为空")
	return s
}

// AssertStartPluginSuccess 断言启动插件成功
func (s *ManagerTestSuite) AssertStartPluginSuccess(id string) *ManagerTestSuite {
	err := s.manager.StartPlugin(id)
	assert.NoError(s.t, err, "启动插件应该成功")
	return s
}

// AssertStopPluginSuccess 断言停止插件成功
func (s *ManagerTestSuite) AssertStopPluginSuccess(id string) *ManagerTestSuite {
	err := s.manager.StopPlugin(id)
	assert.NoError(s.t, err, "停止插件应该成功")
	return s
}

// AssertUnloadPluginSuccess 断言卸载插件成功
func (s *ManagerTestSuite) AssertUnloadPluginSuccess(id string) *ManagerTestSuite {
	err := s.manager.UnloadPlugin(id)
	assert.NoError(s.t, err, "卸载插件应该成功")
	return s
}

// AssertPluginHealthy 断言插件健康
func (s *ManagerTestSuite) AssertPluginHealthy(id string) *ManagerTestSuite {
	status, err := s.manager.GetPluginStatus(id)
	assert.NoError(s.t, err, "获取插件状态应该成功")
	assert.Equal(s.t, "healthy", status.Health.Status, "插件应该健康")
	return s
}

// AssertPluginState 断言插件状态
func (s *ManagerTestSuite) AssertPluginState(id string, state api.PluginState) *ManagerTestSuite {
	status, err := s.manager.GetPluginStatus(id)
	assert.NoError(s.t, err, "获取插件状态应该成功")
	assert.Equal(s.t, state, status.State, "插件状态应该匹配")
	return s
}

// AssertPluginCount 断言插件数量
func (s *ManagerTestSuite) AssertPluginCount(count int) *ManagerTestSuite {
	plugins := s.manager.ListPlugins()
	assert.Equal(s.t, count, len(plugins), "插件数量应该匹配")
	return s
}

// AssertPluginExists 断言插件存在
func (s *ManagerTestSuite) AssertPluginExists(id string) *ManagerTestSuite {
	plugin, exists := s.manager.GetPlugin(id)
	assert.True(s.t, exists, "插件应该存在")
	assert.NotNil(s.t, plugin, "插件不应该为空")
	return s
}

// AssertPluginNotExists 断言插件不存在
func (s *ManagerTestSuite) AssertPluginNotExists(id string) *ManagerTestSuite {
	_, exists := s.manager.GetPlugin(id)
	assert.False(s.t, exists, "插件不应该存在")
	return s
}

// WaitForPluginState 等待插件状态
func (s *ManagerTestSuite) WaitForPluginState(id string, state api.PluginState, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := s.manager.GetPluginStatus(id)
		if err != nil {
			return err
		}

		if status.State == state {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("等待插件状态超时: %s", id)
}

// WaitForPluginHealth 等待插件健康
func (s *ManagerTestSuite) WaitForPluginHealth(id string, status string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pluginStatus, err := s.manager.GetPluginStatus(id)
		if err != nil {
			return err
		}

		if pluginStatus.Health.Status == status {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("等待插件健康状态超时: %s", id)
}
