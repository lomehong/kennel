package testing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"github.com/lomehong/kennel/pkg/plugin/sdk"
	"github.com/stretchr/testify/assert"
)

// PluginTestSuite 插件测试套件
// 提供插件测试功能
type PluginTestSuite struct {
	// 测试对象
	t *testing.T

	// 插件
	plugin api.Plugin

	// 插件ID
	pluginID string

	// 插件配置
	config api.PluginConfig

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
}

// NewPluginTestSuite 创建一个新的插件测试套件
func NewPluginTestSuite(t *testing.T, plugin api.Plugin) *PluginTestSuite {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "plugin-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin-test",
		Level:  hclog.Debug,
		Output: os.Stdout,
	})

	// 获取插件信息
	info := plugin.GetInfo()

	// 创建插件配置
	config := api.PluginConfig{
		ID:       info.ID,
		Enabled:  true,
		LogLevel: "debug",
		Settings: make(map[string]interface{}),
	}

	return &PluginTestSuite{
		t:        t,
		plugin:   plugin,
		pluginID: info.ID,
		config:   config,
		logger:   logger,
		tempDir:  tempDir,
		ctx:      ctx,
		cancel:   cancel,
		cleanups: make([]func(), 0),
	}
}

// SetConfig 设置插件配置
func (s *PluginTestSuite) SetConfig(key string, value interface{}) *PluginTestSuite {
	s.config.Settings[key] = value
	return s
}

// SetEnabled 设置插件启用状态
func (s *PluginTestSuite) SetEnabled(enabled bool) *PluginTestSuite {
	s.config.Enabled = enabled
	return s
}

// SetLogLevel 设置日志级别
func (s *PluginTestSuite) SetLogLevel(level string) *PluginTestSuite {
	s.config.LogLevel = level
	return s
}

// AddCleanup 添加清理函数
func (s *PluginTestSuite) AddCleanup(cleanup func()) *PluginTestSuite {
	s.cleanups = append(s.cleanups, cleanup)
	return s
}

// CreateTempFile 创建临时文件
func (s *PluginTestSuite) CreateTempFile(name string, content []byte) string {
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
func (s *PluginTestSuite) CreateTempDir(name string) string {
	path := filepath.Join(s.tempDir, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		s.t.Fatalf("创建临时目录失败: %v", err)
	}

	return path
}

// GetTempDir 获取临时目录
func (s *PluginTestSuite) GetTempDir() string {
	return s.tempDir
}

// GetLogger 获取日志记录器
func (s *PluginTestSuite) GetLogger() hclog.Logger {
	return s.logger
}

// GetContext 获取上下文
func (s *PluginTestSuite) GetContext() context.Context {
	return s.ctx
}

// GetPlugin 获取插件
func (s *PluginTestSuite) GetPlugin() api.Plugin {
	return s.plugin
}

// GetPluginID 获取插件ID
func (s *PluginTestSuite) GetPluginID() string {
	return s.pluginID
}

// GetConfig 获取插件配置
func (s *PluginTestSuite) GetConfig() api.PluginConfig {
	return s.config
}

// Init 初始化插件
func (s *PluginTestSuite) Init() error {
	return s.plugin.Init(s.ctx, s.config)
}

// Start 启动插件
func (s *PluginTestSuite) Start() error {
	return s.plugin.Start(s.ctx)
}

// Stop 停止插件
func (s *PluginTestSuite) Stop() error {
	return s.plugin.Stop(s.ctx)
}

// HealthCheck 执行健康检查
func (s *PluginTestSuite) HealthCheck() (api.HealthStatus, error) {
	return s.plugin.HealthCheck(s.ctx)
}

// Run 运行测试
func (s *PluginTestSuite) Run(fn func(*PluginTestSuite)) {
	defer s.Cleanup()

	// 初始化插件
	if err := s.Init(); err != nil {
		s.t.Fatalf("初始化插件失败: %v", err)
	}

	// 启动插件
	if err := s.Start(); err != nil {
		s.t.Fatalf("启动插件失败: %v", err)
	}

	// 运行测试函数
	fn(s)

	// 停止插件
	if err := s.Stop(); err != nil {
		s.t.Fatalf("停止插件失败: %v", err)
	}
}

// Cleanup 清理资源
func (s *PluginTestSuite) Cleanup() {
	// 执行清理函数
	for _, cleanup := range s.cleanups {
		cleanup()
	}

	// 取消上下文
	s.cancel()

	// 删除临时目录
	os.RemoveAll(s.tempDir)
}

// AssertInitSuccess 断言初始化成功
func (s *PluginTestSuite) AssertInitSuccess() *PluginTestSuite {
	err := s.Init()
	assert.NoError(s.t, err, "初始化插件应该成功")
	return s
}

// AssertStartSuccess 断言启动成功
func (s *PluginTestSuite) AssertStartSuccess() *PluginTestSuite {
	err := s.Start()
	assert.NoError(s.t, err, "启动插件应该成功")
	return s
}

// AssertStopSuccess 断言停止成功
func (s *PluginTestSuite) AssertStopSuccess() *PluginTestSuite {
	err := s.Stop()
	assert.NoError(s.t, err, "停止插件应该成功")
	return s
}

// AssertHealthy 断言插件健康
func (s *PluginTestSuite) AssertHealthy() *PluginTestSuite {
	health, err := s.HealthCheck()
	assert.NoError(s.t, err, "健康检查应该成功")
	assert.Equal(s.t, "healthy", health.Status, "插件应该健康")
	return s
}

// MockPlugin 模拟插件
// 用于测试
type MockPlugin struct {
	// 插件信息
	info api.PluginInfo

	// 初始化函数
	initFunc func(ctx context.Context, config api.PluginConfig) error

	// 启动函数
	startFunc func(ctx context.Context) error

	// 停止函数
	stopFunc func(ctx context.Context) error

	// 健康检查函数
	healthCheckFunc func(ctx context.Context) (api.HealthStatus, error)

	// 是否已初始化
	initialized bool

	// 是否已启动
	started bool

	// 配置
	config api.PluginConfig

	// 日志记录器
	logger hclog.Logger
}

// NewMockPlugin 创建一个新的模拟插件
func NewMockPlugin(id string) *MockPlugin {
	return &MockPlugin{
		info: api.PluginInfo{
			ID:          id,
			Name:        "Mock Plugin",
			Version:     "1.0.0",
			Description: "Mock plugin for testing",
			Author:      "Test",
			License:     "MIT",
		},
		initFunc: func(ctx context.Context, config api.PluginConfig) error {
			return nil
		},
		startFunc: func(ctx context.Context) error {
			return nil
		},
		stopFunc: func(ctx context.Context) error {
			return nil
		},
		healthCheckFunc: func(ctx context.Context) (api.HealthStatus, error) {
			return api.HealthStatus{
				Status:      "healthy",
				Details:     make(map[string]interface{}),
				LastChecked: time.Now(),
			}, nil
		},
		logger: hclog.NewNullLogger(),
	}
}

// GetInfo 获取插件信息
func (p *MockPlugin) GetInfo() api.PluginInfo {
	return p.info
}

// Init 初始化插件
func (p *MockPlugin) Init(ctx context.Context, config api.PluginConfig) error {
	p.config = config
	p.initialized = true
	return p.initFunc(ctx, config)
}

// Start 启动插件
func (p *MockPlugin) Start(ctx context.Context) error {
	if !p.initialized {
		return fmt.Errorf("插件未初始化")
	}
	p.started = true
	return p.startFunc(ctx)
}

// Stop 停止插件
func (p *MockPlugin) Stop(ctx context.Context) error {
	if !p.started {
		return fmt.Errorf("插件未启动")
	}
	p.started = false
	return p.stopFunc(ctx)
}

// HealthCheck 执行健康检查
func (p *MockPlugin) HealthCheck(ctx context.Context) (api.HealthStatus, error) {
	return p.healthCheckFunc(ctx)
}

// WithInitFunc 设置初始化函数
func (p *MockPlugin) WithInitFunc(fn func(ctx context.Context, config api.PluginConfig) error) *MockPlugin {
	p.initFunc = fn
	return p
}

// WithStartFunc 设置启动函数
func (p *MockPlugin) WithStartFunc(fn func(ctx context.Context) error) *MockPlugin {
	p.startFunc = fn
	return p
}

// WithStopFunc 设置停止函数
func (p *MockPlugin) WithStopFunc(fn func(ctx context.Context) error) *MockPlugin {
	p.stopFunc = fn
	return p
}

// WithHealthCheckFunc 设置健康检查函数
func (p *MockPlugin) WithHealthCheckFunc(fn func(ctx context.Context) (api.HealthStatus, error)) *MockPlugin {
	p.healthCheckFunc = fn
	return p
}

// WithInfo 设置插件信息
func (p *MockPlugin) WithInfo(info api.PluginInfo) *MockPlugin {
	p.info = info
	return p
}

// WithLogger 设置日志记录器
func (p *MockPlugin) WithLogger(logger hclog.Logger) *MockPlugin {
	p.logger = logger
	return p
}

// IsInitialized 检查插件是否已初始化
func (p *MockPlugin) IsInitialized() bool {
	return p.initialized
}

// IsStarted 检查插件是否已启动
func (p *MockPlugin) IsStarted() bool {
	return p.started
}

// GetConfig 获取插件配置
func (p *MockPlugin) GetConfig() api.PluginConfig {
	return p.config
}

// GetLogger 获取日志记录器
func (p *MockPlugin) GetLogger() hclog.Logger {
	return p.logger
}
