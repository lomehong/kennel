package isolation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// IsolationLevel 定义了隔离级别
type IsolationLevel string

const (
	// IsolationLevelNone 无隔离
	IsolationLevelNone IsolationLevel = "none"
	
	// IsolationLevelBasic 基本隔离
	// 提供基本的错误隔离和资源限制
	IsolationLevelBasic IsolationLevel = "basic"
	
	// IsolationLevelStrict 严格隔离
	// 提供更严格的隔离，包括进程隔离
	IsolationLevelStrict IsolationLevel = "strict"
	
	// IsolationLevelComplete 完全隔离
	// 提供最高级别的隔离，包括容器隔离
	IsolationLevelComplete IsolationLevel = "complete"
)

// PluginIsolator 插件隔离器接口
// 负责提供插件的隔离环境
type PluginIsolator interface {
	// CreateSandbox 创建沙箱
	// id: 沙箱ID
	// config: 隔离配置
	// 返回: 沙箱和错误
	CreateSandbox(id string, config api.IsolationConfig) (PluginSandbox, error)
	
	// DestroySandbox 销毁沙箱
	// id: 沙箱ID
	// 返回: 错误
	DestroySandbox(id string) error
	
	// GetSandbox 获取沙箱
	// id: 沙箱ID
	// 返回: 沙箱和是否存在
	GetSandbox(id string) (PluginSandbox, bool)
	
	// ListSandboxes 列出所有沙箱
	// 返回: 沙箱ID列表
	ListSandboxes() []string
	
	// Close 关闭隔离器
	// 返回: 错误
	Close() error
}

// PluginSandbox 插件沙箱接口
// 提供隔离的执行环境
type PluginSandbox interface {
	// GetID 获取沙箱ID
	// 返回: 沙箱ID
	GetID() string
	
	// GetConfig 获取隔离配置
	// 返回: 隔离配置
	GetConfig() api.IsolationConfig
	
	// Execute 执行函数
	// f: 要执行的函数
	// 返回: 错误
	Execute(f func() error) error
	
	// ExecuteWithContext 执行带上下文的函数
	// ctx: 上下文
	// f: 要执行的函数
	// 返回: 错误
	ExecuteWithContext(ctx context.Context, f func(context.Context) error) error
	
	// ExecuteWithTimeout 执行带超时的函数
	// timeout: 超时时间
	// f: 要执行的函数
	// 返回: 错误
	ExecuteWithTimeout(timeout time.Duration, f func() error) error
	
	// Pause 暂停沙箱
	// 返回: 错误
	Pause() error
	
	// Resume 恢复沙箱
	// 返回: 错误
	Resume() error
	
	// Stop 停止沙箱
	// 返回: 错误
	Stop() error
	
	// IsHealthy 检查沙箱是否健康
	// 返回: 是否健康
	IsHealthy() bool
	
	// GetStats 获取统计信息
	// 返回: 统计信息
	GetStats() map[string]interface{}
}

// DefaultPluginIsolator 默认插件隔离器实现
type DefaultPluginIsolator struct {
	// 沙箱映射
	sandboxes map[string]PluginSandbox
	
	// 互斥锁
	mu sync.RWMutex
	
	// 日志记录器
	logger hclog.Logger
	
	// 上下文
	ctx context.Context
	
	// 取消函数
	cancel context.CancelFunc
	
	// 资源监控器
	resourceMonitor *ResourceMonitor
}

// NewPluginIsolator 创建一个新的插件隔离器
func NewPluginIsolator(logger hclog.Logger) *DefaultPluginIsolator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DefaultPluginIsolator{
		sandboxes:       make(map[string]PluginSandbox),
		logger:          logger.Named("isolator"),
		ctx:             ctx,
		cancel:          cancel,
		resourceMonitor: NewResourceMonitor(logger.Named("resource-monitor")),
	}
}

// CreateSandbox 创建沙箱
func (i *DefaultPluginIsolator) CreateSandbox(id string, config api.IsolationConfig) (PluginSandbox, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	// 检查沙箱是否已存在
	if _, exists := i.sandboxes[id]; exists {
		return nil, fmt.Errorf("沙箱 %s 已存在", id)
	}
	
	// 根据隔离级别创建沙箱
	var sandbox PluginSandbox
	var err error
	
	switch IsolationLevel(config.Level) {
	case IsolationLevelNone:
		sandbox = NewNoIsolationSandbox(id, config, i.logger.Named(fmt.Sprintf("sandbox-%s", id)))
	case IsolationLevelBasic:
		sandbox = NewBasicIsolationSandbox(id, config, i.logger.Named(fmt.Sprintf("sandbox-%s", id)))
	case IsolationLevelStrict:
		sandbox = NewStrictIsolationSandbox(id, config, i.logger.Named(fmt.Sprintf("sandbox-%s", id)))
	case IsolationLevelComplete:
		sandbox = NewCompleteIsolationSandbox(id, config, i.logger.Named(fmt.Sprintf("sandbox-%s", id)))
	default:
		// 默认使用基本隔离
		sandbox = NewBasicIsolationSandbox(id, config, i.logger.Named(fmt.Sprintf("sandbox-%s", id)))
	}
	
	if err != nil {
		return nil, fmt.Errorf("创建沙箱失败: %w", err)
	}
	
	// 存储沙箱
	i.sandboxes[id] = sandbox
	
	// 注册到资源监控器
	i.resourceMonitor.RegisterSandbox(sandbox)
	
	i.logger.Info("沙箱已创建", "id", id, "isolation_level", config.Level)
	return sandbox, nil
}

// DestroySandbox 销毁沙箱
func (i *DefaultPluginIsolator) DestroySandbox(id string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	// 检查沙箱是否存在
	sandbox, exists := i.sandboxes[id]
	if !exists {
		return fmt.Errorf("沙箱 %s 不存在", id)
	}
	
	// 停止沙箱
	if err := sandbox.Stop(); err != nil {
		i.logger.Error("停止沙箱失败", "id", id, "error", err)
		// 继续销毁过程
	}
	
	// 从资源监控器中注销
	i.resourceMonitor.UnregisterSandbox(id)
	
	// 删除沙箱
	delete(i.sandboxes, id)
	
	i.logger.Info("沙箱已销毁", "id", id)
	return nil
}

// GetSandbox 获取沙箱
func (i *DefaultPluginIsolator) GetSandbox(id string) (PluginSandbox, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	sandbox, exists := i.sandboxes[id]
	return sandbox, exists
}

// ListSandboxes 列出所有沙箱
func (i *DefaultPluginIsolator) ListSandboxes() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	ids := make([]string, 0, len(i.sandboxes))
	for id := range i.sandboxes {
		ids = append(ids, id)
	}
	
	return ids
}

// Close 关闭隔离器
func (i *DefaultPluginIsolator) Close() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	// 停止所有沙箱
	for id, sandbox := range i.sandboxes {
		i.logger.Info("停止沙箱", "id", id)
		if err := sandbox.Stop(); err != nil {
			i.logger.Error("停止沙箱失败", "id", id, "error", err)
		}
	}
	
	// 停止资源监控器
	i.resourceMonitor.Stop()
	
	// 取消上下文
	i.cancel()
	
	i.logger.Info("隔离器已关闭")
	return nil
}
