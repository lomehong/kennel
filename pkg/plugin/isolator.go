package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/lomehong/kennel/pkg/concurrency"
	"github.com/lomehong/kennel/pkg/errors"
	"github.com/lomehong/kennel/pkg/resource"
)

// PluginExecutor 插件执行器
type PluginExecutor struct {
	pluginID        string
	pluginPath      string
	logger          hclog.Logger
	resourceTracker *resource.ResourceTracker
	workerpool      *concurrency.WorkerPool
	errorRegistry   *errors.ErrorHandlerRegistry
	recoveryManager *errors.RecoveryManager
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	client          *plugin.Client
	rpcClient       plugin.ClientProtocol
	instance        interface{}
	started         bool
	exited          bool
	exitCode        int
	lastError       error
	startTime       time.Time
	stopTime        time.Time
}

// PluginExecutorOption 插件执行器配置选项
type PluginExecutorOption func(*PluginExecutor)

// WithExecutorLogger 设置日志记录器
func WithExecutorLogger(logger hclog.Logger) PluginExecutorOption {
	return func(pe *PluginExecutor) {
		pe.logger = logger
	}
}

// WithExecutorResourceTracker 设置资源追踪器
func WithExecutorResourceTracker(tracker *resource.ResourceTracker) PluginExecutorOption {
	return func(pe *PluginExecutor) {
		pe.resourceTracker = tracker
	}
}

// WithExecutorWorkerPool 设置工作池
func WithExecutorWorkerPool(pool *concurrency.WorkerPool) PluginExecutorOption {
	return func(pe *PluginExecutor) {
		pe.workerpool = pool
	}
}

// WithExecutorErrorRegistry 设置错误处理器注册表
func WithExecutorErrorRegistry(registry *errors.ErrorHandlerRegistry) PluginExecutorOption {
	return func(pe *PluginExecutor) {
		pe.errorRegistry = registry
	}
}

// WithExecutorRecoveryManager 设置恢复管理器
func WithExecutorRecoveryManager(manager *errors.RecoveryManager) PluginExecutorOption {
	return func(pe *PluginExecutor) {
		pe.recoveryManager = manager
	}
}

// WithExecutorContext 设置上下文
func WithExecutorContext(ctx context.Context) PluginExecutorOption {
	return func(pe *PluginExecutor) {
		if pe.cancel != nil {
			pe.cancel()
		}
		pe.ctx, pe.cancel = context.WithCancel(ctx)
	}
}

// NewPluginExecutor 创建一个新的插件执行器
func NewPluginExecutor(pluginID string, pluginPath string, options ...PluginExecutorOption) *PluginExecutor {
	ctx, cancel := context.WithCancel(context.Background())

	pe := &PluginExecutor{
		pluginID:   pluginID,
		pluginPath: pluginPath,
		logger:     hclog.NewNullLogger(),
		ctx:        ctx,
		cancel:     cancel,
		startTime:  time.Now(),
	}

	// 应用选项
	for _, option := range options {
		option(pe)
	}

	return pe
}

// Start 启动插件
func (pe *PluginExecutor) Start() error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if pe.started {
		return fmt.Errorf("插件 %s 已启动", pe.pluginID)
	}

	// 检查插件可执行文件是否存在
	if _, err := os.Stat(pe.pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("插件可执行文件不存在: %s", pe.pluginPath)
	}

	// 创建插件客户端
	pe.client = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "PLUGIN_MAGIC_COOKIE",
			MagicCookieValue: "kennel",
		},
		Plugins:  PluginMap,
		Cmd:      exec.Command(pe.pluginPath),
		Logger:   pe.logger,
		AutoMTLS: true,
	})

	// 连接到插件
	rpcClient, err := pe.client.Client()
	if err != nil {
		pe.lastError = err
		return fmt.Errorf("连接到插件失败: %w", err)
	}
	pe.rpcClient = rpcClient

	// 获取插件实例
	instance, err := rpcClient.Dispense("module")
	if err != nil {
		pe.lastError = err
		return fmt.Errorf("获取插件实例失败: %w", err)
	}
	pe.instance = instance

	pe.started = true
	pe.startTime = time.Now()
	pe.logger.Info("插件已启动", "id", pe.pluginID, "path", pe.pluginPath)

	return nil
}

// Stop 停止插件
func (pe *PluginExecutor) Stop() error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if !pe.started {
		return fmt.Errorf("插件 %s 未启动", pe.pluginID)
	}

	if pe.exited {
		return nil
	}

	// 关闭客户端
	pe.client.Kill()

	pe.exited = true
	pe.stopTime = time.Now()
	pe.logger.Info("插件已停止", "id", pe.pluginID)

	return nil
}

// GetInstance 获取插件实例
func (pe *PluginExecutor) GetInstance() interface{} {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.instance
}

// IsRunning 检查插件是否正在运行
func (pe *PluginExecutor) IsRunning() bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.started && !pe.exited
}

// GetLastError 获取最后一个错误
func (pe *PluginExecutor) GetLastError() error {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.lastError
}

// GetUptime 获取插件运行时间
func (pe *PluginExecutor) GetUptime() time.Duration {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	if !pe.started {
		return 0
	}
	if pe.exited {
		return pe.stopTime.Sub(pe.startTime)
	}
	return time.Since(pe.startTime)
}

// ExecuteFunc 在插件中执行函数
func (pe *PluginExecutor) ExecuteFunc(f func(interface{}) error) error {
	pe.mu.RLock()
	if !pe.started {
		pe.mu.RUnlock()
		return fmt.Errorf("插件 %s 未启动", pe.pluginID)
	}
	instance := pe.instance
	pe.mu.RUnlock()

	return f(instance)
}
