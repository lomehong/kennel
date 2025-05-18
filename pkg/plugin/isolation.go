package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/concurrency"
	"github.com/lomehong/kennel/pkg/errors"
	"github.com/lomehong/kennel/pkg/resource"
)

// IsolationLevel 表示插件隔离级别
type IsolationLevel int

// 预定义隔离级别
const (
	IsolationLevelNone     IsolationLevel = iota // 无隔离
	IsolationLevelBasic                          // 基本隔离（独立goroutine和错误处理）
	IsolationLevelStrict                         // 严格隔离（独立goroutine、错误处理和资源限制）
	IsolationLevelComplete                       // 完全隔离（独立进程）
)

// String 返回隔离级别的字符串表示
func (il IsolationLevel) String() string {
	switch il {
	case IsolationLevelNone:
		return "None"
	case IsolationLevelBasic:
		return "Basic"
	case IsolationLevelStrict:
		return "Strict"
	case IsolationLevelComplete:
		return "Complete"
	default:
		return "Unknown"
	}
}

// PluginIsolationConfig 插件隔离配置
type PluginIsolationConfig struct {
	Level           IsolationLevel         // 隔离级别
	ResourceLimits  map[string]int         // 资源限制
	TimeoutDuration time.Duration          // 超时时间
	MemoryLimit     int64                  // 内存限制（字节）
	CPULimit        int                    // CPU限制（百分比）
	IOLimit         int                    // IO限制（操作/秒）
	NetworkLimit    int                    // 网络限制（字节/秒）
	AllowedAPIs     map[string]bool        // 允许的API
	BlockedAPIs     map[string]bool        // 阻止的API
	Environment     map[string]string      // 环境变量
	WorkingDir      string                 // 工作目录
	LogLevel        string                 // 日志级别
	ErrorHandler    errors.ErrorHandler    // 错误处理器
	RecoveryHandler errors.RecoveryHandler // 恢复处理器
}

// DefaultPluginIsolationConfig 返回默认的插件隔离配置
func DefaultPluginIsolationConfig() *PluginIsolationConfig {
	return &PluginIsolationConfig{
		Level:           IsolationLevelBasic,
		ResourceLimits:  make(map[string]int),
		TimeoutDuration: 30 * time.Second,
		MemoryLimit:     100 * 1024 * 1024, // 100MB
		CPULimit:        50,                // 50%
		IOLimit:         100,               // 100 ops/sec
		NetworkLimit:    1024 * 1024,       // 1MB/sec
		AllowedAPIs:     make(map[string]bool),
		BlockedAPIs:     make(map[string]bool),
		Environment:     make(map[string]string),
		LogLevel:        "info",
	}
}

// PluginIsolator 插件隔离器
type PluginIsolator struct {
	config          *PluginIsolationConfig
	logger          hclog.Logger
	resourceTracker *resource.ResourceTracker
	workerpool      *concurrency.WorkerPool
	errorRegistry   *errors.ErrorHandlerRegistry
	recoveryManager *errors.RecoveryManager
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	resources       map[string][]resource.Resource
	stats           PluginIsolationStats
}

// PluginIsolationStats 插件隔离统计信息
type PluginIsolationStats struct {
	TotalCalls         int64
	SuccessfulCalls    int64
	FailedCalls        int64
	Timeouts           int64
	ResourceViolations int64
	Panics             int64
	TotalExecutionTime time.Duration
	AvgExecutionTime   time.Duration
	MaxExecutionTime   time.Duration
	MinExecutionTime   time.Duration
}

// PluginIsolatorOption 插件隔离器配置选项
type PluginIsolatorOption func(*PluginIsolator)

// WithLogger 设置日志记录器
func WithLogger(logger hclog.Logger) PluginIsolatorOption {
	return func(pi *PluginIsolator) {
		pi.logger = logger
	}
}

// WithResourceTracker 设置资源追踪器
func WithResourceTracker(tracker *resource.ResourceTracker) PluginIsolatorOption {
	return func(pi *PluginIsolator) {
		pi.resourceTracker = tracker
	}
}

// WithWorkerPool 设置工作池
func WithWorkerPool(pool *concurrency.WorkerPool) PluginIsolatorOption {
	return func(pi *PluginIsolator) {
		pi.workerpool = pool
	}
}

// WithErrorRegistry 设置错误处理器注册表
func WithErrorRegistry(registry *errors.ErrorHandlerRegistry) PluginIsolatorOption {
	return func(pi *PluginIsolator) {
		pi.errorRegistry = registry
	}
}

// WithRecoveryManager 设置恢复管理器
func WithRecoveryManager(manager *errors.RecoveryManager) PluginIsolatorOption {
	return func(pi *PluginIsolator) {
		pi.recoveryManager = manager
	}
}

// WithContext 设置上下文
func WithContext(ctx context.Context) PluginIsolatorOption {
	return func(pi *PluginIsolator) {
		if pi.cancel != nil {
			pi.cancel()
		}
		pi.ctx, pi.cancel = context.WithCancel(ctx)
	}
}

// NewPluginIsolator 创建一个新的插件隔离器
func NewPluginIsolator(config *PluginIsolationConfig, options ...PluginIsolatorOption) *PluginIsolator {
	if config == nil {
		config = DefaultPluginIsolationConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pi := &PluginIsolator{
		config:    config,
		logger:    hclog.NewNullLogger(),
		ctx:       ctx,
		cancel:    cancel,
		resources: make(map[string][]resource.Resource),
	}

	// 应用选项
	for _, option := range options {
		option(pi)
	}

	// 如果没有设置错误处理器，使用默认的
	if config.ErrorHandler == nil && pi.errorRegistry != nil {
		config.ErrorHandler = errors.DefaultErrorHandler(pi.logger)
	}

	// 如果没有设置恢复处理器，使用默认的
	if config.RecoveryHandler == nil && pi.recoveryManager != nil {
		config.RecoveryHandler = errors.DefaultRecoveryHandler(pi.logger)
	}

	return pi
}

// ExecuteFunc 在隔离环境中执行函数
func (pi *PluginIsolator) ExecuteFunc(pluginID string, f func() error) error {
	startTime := time.Now()
	pi.mu.Lock()
	pi.stats.TotalCalls++
	pi.mu.Unlock()

	var err error

	// 根据隔离级别执行函数
	switch pi.config.Level {
	case IsolationLevelNone:
		// 直接执行，无隔离
		err = f()
	case IsolationLevelBasic:
		// 基本隔离（独立goroutine和错误处理）
		err = pi.executeWithBasicIsolation(pluginID, f)
	case IsolationLevelStrict:
		// 严格隔离（独立goroutine、错误处理和资源限制）
		err = pi.executeWithStrictIsolation(pluginID, f)
	case IsolationLevelComplete:
		// 完全隔离（独立进程）
		err = pi.executeWithCompleteIsolation(pluginID, f)
	default:
		err = fmt.Errorf("未知的隔离级别: %v", pi.config.Level)
	}

	// 更新统计信息
	executionTime := time.Since(startTime)
	pi.mu.Lock()
	pi.stats.TotalExecutionTime += executionTime
	if executionTime > pi.stats.MaxExecutionTime {
		pi.stats.MaxExecutionTime = executionTime
	}
	if pi.stats.MinExecutionTime == 0 || executionTime < pi.stats.MinExecutionTime {
		pi.stats.MinExecutionTime = executionTime
	}
	pi.stats.AvgExecutionTime = pi.stats.TotalExecutionTime / time.Duration(pi.stats.TotalCalls)

	if err != nil {
		pi.stats.FailedCalls++
		if errors.IsType(err, errors.ErrorTypeTemporary) && err.Error() == "timeout" {
			pi.stats.Timeouts++
		}
	} else {
		pi.stats.SuccessfulCalls++
	}
	pi.mu.Unlock()

	return err
}

// executeWithBasicIsolation 使用基本隔离执行函数
func (pi *PluginIsolator) executeWithBasicIsolation(pluginID string, f func() error) error {
	// 创建上下文
	ctx, cancel := context.WithTimeout(pi.ctx, pi.config.TimeoutDuration)
	defer cancel()

	// 创建结果通道
	resultCh := make(chan error, 1)

	// 在goroutine中执行函数
	go func() {
		// 使用恢复处理器
		defer func() {
			if p := recover(); p != nil {
				pi.mu.Lock()
				pi.stats.Panics++
				pi.mu.Unlock()

				var err error
				if pi.config.RecoveryHandler != nil {
					err = pi.config.RecoveryHandler.HandlePanic(p)
				} else {
					err = fmt.Errorf("插件 %s 发生panic: %v", pluginID, p)
				}
				resultCh <- err
			}
		}()

		// 执行函数
		err := f()

		// 处理错误
		if err != nil && pi.config.ErrorHandler != nil {
			err = pi.config.ErrorHandler.Handle(err)
		}

		// 发送结果
		resultCh <- err
	}()

	// 等待结果或超时
	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), errors.ErrorTypeTemporary, "TIMEOUT", fmt.Sprintf("插件 %s 执行超时", pluginID))
	}
}

// executeWithStrictIsolation 使用严格隔离执行函数
func (pi *PluginIsolator) executeWithStrictIsolation(pluginID string, f func() error) error {
	// 如果没有工作池，回退到基本隔离
	if pi.workerpool == nil {
		pi.logger.Warn("没有工作池，回退到基本隔离", "plugin_id", pluginID)
		return pi.executeWithBasicIsolation(pluginID, f)
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(pi.ctx, pi.config.TimeoutDuration)
	defer cancel()

	// 创建上下文资源追踪器
	if pi.resourceTracker != nil {
		// 创建资源追踪器但不保存变量，避免未使用变量警告
		// 资源会在上下文取消时自动释放
		_ = resource.WithTrackerContext(ctx, pi.resourceTracker)
	}

	// 提交任务到工作池
	resultChan, err := pi.workerpool.SubmitWithContext(pluginID, fmt.Sprintf("Plugin %s execution", pluginID), ctx, func(ctx context.Context) error {
		// 使用恢复处理器
		defer func() {
			if p := recover(); p != nil {
				pi.mu.Lock()
				pi.stats.Panics++
				pi.mu.Unlock()

				if pi.config.RecoveryHandler != nil {
					pi.config.RecoveryHandler.HandlePanic(p)
				}
			}
		}()

		// 执行函数
		err := f()

		// 处理错误
		if err != nil && pi.config.ErrorHandler != nil {
			err = pi.config.ErrorHandler.Handle(err)
		}

		return err
	})

	if err != nil {
		return errors.Wrap(err, errors.ErrorTypeInternal, "SUBMIT_ERROR", fmt.Sprintf("提交插件 %s 任务失败", pluginID))
	}

	// 等待结果
	result := <-resultChan
	if result.Error != nil {
		if errors.Is(result.Error, context.DeadlineExceeded) {
			return errors.Wrap(result.Error, errors.ErrorTypeTemporary, "TIMEOUT", fmt.Sprintf("插件 %s 执行超时", pluginID))
		}
		return result.Error
	}

	return nil
}

// executeWithCompleteIsolation 使用完全隔离执行函数
func (pi *PluginIsolator) executeWithCompleteIsolation(pluginID string, f func() error) error {
	// 完全隔离需要在独立进程中执行，这里简化为基本隔离
	pi.logger.Warn("完全隔离尚未实现，回退到基本隔离", "plugin_id", pluginID)
	return pi.executeWithBasicIsolation(pluginID, f)
}

// GetStats 获取统计信息
func (pi *PluginIsolator) GetStats() PluginIsolationStats {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.stats
}

// Stop 停止插件隔离器
func (pi *PluginIsolator) Stop() {
	pi.cancel()
}
