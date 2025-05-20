package testing

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"github.com/lomehong/kennel/pkg/plugin/sdk"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// IntegrationTestSuite 集成测试套件
// 提供插件集成测试功能
type IntegrationTestSuite struct {
	// 测试对象
	t *testing.T

	// 插件管理器
	manager *plugin.PluginManagerV3

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

	// 插件进程
	processes map[string]*os.Process
}

// NewIntegrationTestSuite 创建一个新的集成测试套件
func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "integration-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "integration-test",
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
	config.AutoDiscover = true
	config.AutoStart = false

	// 创建插件管理器
	manager := plugin.NewPluginManagerV3(logger, config)

	return &IntegrationTestSuite{
		t:         t,
		manager:   manager,
		logger:    logger,
		tempDir:   tempDir,
		ctx:       ctx,
		cancel:    cancel,
		cleanups:  make([]func(), 0),
		processes: make(map[string]*os.Process),
	}
}

// AddCleanup 添加清理函数
func (s *IntegrationTestSuite) AddCleanup(cleanup func()) *IntegrationTestSuite {
	s.cleanups = append(s.cleanups, cleanup)
	return s
}

// CreateTempFile 创建临时文件
func (s *IntegrationTestSuite) CreateTempFile(name string, content []byte) string {
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
func (s *IntegrationTestSuite) CreateTempDir(name string) string {
	path := filepath.Join(s.tempDir, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		s.t.Fatalf("创建临时目录失败: %v", err)
	}

	return path
}

// GetTempDir 获取临时目录
func (s *IntegrationTestSuite) GetTempDir() string {
	return s.tempDir
}

// GetLogger 获取日志记录器
func (s *IntegrationTestSuite) GetLogger() hclog.Logger {
	return s.logger
}

// GetContext 获取上下文
func (s *IntegrationTestSuite) GetContext() context.Context {
	return s.ctx
}

// GetManager 获取插件管理器
func (s *IntegrationTestSuite) GetManager() *plugin.PluginManagerV3 {
	return s.manager
}

// Start 启动管理器
func (s *IntegrationTestSuite) Start() error {
	return s.manager.Start()
}

// Stop 停止管理器
func (s *IntegrationTestSuite) Stop() error {
	return s.manager.Stop()
}

// BuildPlugin 构建插件
func (s *IntegrationTestSuite) BuildPlugin(id, sourcePath, targetPath string) error {
	// 创建目标目录
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 构建插件
	cmd := exec.Command("go", "build", "-o", targetPath, sourcePath)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("构建插件失败: %w", err)
	}

	return nil
}

// StartPluginProcess 启动插件进程
func (s *IntegrationTestSuite) StartPluginProcess(id, path string) error {
	// 检查插件是否已启动
	if _, exists := s.processes[id]; exists {
		return fmt.Errorf("插件进程已启动: %s", id)
	}

	// 启动插件进程
	cmd := exec.Command(path)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动插件进程失败: %w", err)
	}

	// 存储进程
	s.processes[id] = cmd.Process

	// 等待插件启动
	time.Sleep(1 * time.Second)

	return nil
}

// StopPluginProcess 停止插件进程
func (s *IntegrationTestSuite) StopPluginProcess(id string) error {
	// 检查插件是否已启动
	process, exists := s.processes[id]
	if !exists {
		return fmt.Errorf("插件进程未启动: %s", id)
	}

	// 停止进程
	if err := process.Kill(); err != nil {
		return fmt.Errorf("停止插件进程失败: %w", err)
	}

	// 删除进程
	delete(s.processes, id)

	return nil
}

// CreatePluginConfig 创建插件配置
func (s *IntegrationTestSuite) CreatePluginConfig(id string, config map[string]interface{}) error {
	// 设置基本配置
	if _, ok := config["id"]; !ok {
		config["id"] = id
	}

	if _, ok := config["enabled"]; !ok {
		config["enabled"] = true
	}

	// 创建插件目录
	pluginDir := filepath.Join(s.tempDir, "plugins", id)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("创建插件目录失败: %w", err)
	}

	// 序列化配置
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入配置文件
	configPath := filepath.Join(pluginDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// Run 运行测试
func (s *IntegrationTestSuite) Run(fn func(*IntegrationTestSuite)) {
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
func (s *IntegrationTestSuite) Cleanup() {
	// 停止所有插件进程
	for id := range s.processes {
		s.StopPluginProcess(id)
	}

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
func (s *IntegrationTestSuite) AssertStartSuccess() *IntegrationTestSuite {
	err := s.Start()
	assert.NoError(s.t, err, "启动管理器应该成功")
	return s
}

// AssertStopSuccess 断言停止成功
func (s *IntegrationTestSuite) AssertStopSuccess() *IntegrationTestSuite {
	err := s.Stop()
	assert.NoError(s.t, err, "停止管理器应该成功")
	return s
}

// AssertLoadPluginSuccess 断言加载插件成功
func (s *IntegrationTestSuite) AssertLoadPluginSuccess(id string) *IntegrationTestSuite {
	plugin, err := s.manager.LoadPlugin(id)
	assert.NoError(s.t, err, "加载插件应该成功")
	assert.NotNil(s.t, plugin, "插件不应该为空")
	return s
}

// AssertStartPluginSuccess 断言启动插件成功
func (s *IntegrationTestSuite) AssertStartPluginSuccess(id string) *IntegrationTestSuite {
	err := s.manager.StartPlugin(id)
	assert.NoError(s.t, err, "启动插件应该成功")
	return s
}

// AssertStopPluginSuccess 断言停止插件成功
func (s *IntegrationTestSuite) AssertStopPluginSuccess(id string) *IntegrationTestSuite {
	err := s.manager.StopPlugin(id)
	assert.NoError(s.t, err, "停止插件应该成功")
	return s
}

// AssertUnloadPluginSuccess 断言卸载插件成功
func (s *IntegrationTestSuite) AssertUnloadPluginSuccess(id string) *IntegrationTestSuite {
	err := s.manager.UnloadPlugin(id)
	assert.NoError(s.t, err, "卸载插件应该成功")
	return s
}

// AssertPluginHealthy 断言插件健康
func (s *IntegrationTestSuite) AssertPluginHealthy(id string) *IntegrationTestSuite {
	status, err := s.manager.GetPluginStatus(id)
	assert.NoError(s.t, err, "获取插件状态应该成功")
	assert.Equal(s.t, "healthy", status.Health.Status, "插件应该健康")
	return s
}

// AssertPluginState 断言插件状态
func (s *IntegrationTestSuite) AssertPluginState(id string, state api.PluginState) *IntegrationTestSuite {
	status, err := s.manager.GetPluginStatus(id)
	assert.NoError(s.t, err, "获取插件状态应该成功")
	assert.Equal(s.t, state, status.State, "插件状态应该匹配")
	return s
}

// AssertPluginCount 断言插件数量
func (s *IntegrationTestSuite) AssertPluginCount(count int) *IntegrationTestSuite {
	plugins := s.manager.ListPlugins()
	assert.Equal(s.t, count, len(plugins), "插件数量应该匹配")
	return s
}

// AssertPluginExists 断言插件存在
func (s *IntegrationTestSuite) AssertPluginExists(id string) *IntegrationTestSuite {
	plugin, exists := s.manager.GetPlugin(id)
	assert.True(s.t, exists, "插件应该存在")
	assert.NotNil(s.t, plugin, "插件不应该为空")
	return s
}

// AssertPluginNotExists 断言插件不存在
func (s *IntegrationTestSuite) AssertPluginNotExists(id string) *IntegrationTestSuite {
	_, exists := s.manager.GetPlugin(id)
	assert.False(s.t, exists, "插件不应该存在")
	return s
}

// WaitForPluginState 等待插件状态
func (s *IntegrationTestSuite) WaitForPluginState(id string, state api.PluginState, timeout time.Duration) error {
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
func (s *IntegrationTestSuite) WaitForPluginHealth(id string, status string, timeout time.Duration) error {
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
