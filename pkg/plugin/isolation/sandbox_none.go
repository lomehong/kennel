package isolation

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// NoIsolationSandbox 无隔离沙箱
// 不提供任何隔离，直接执行函数
type NoIsolationSandbox struct {
	// 沙箱ID
	id string
	
	// 隔离配置
	config api.IsolationConfig
	
	// 日志记录器
	logger hclog.Logger
	
	// 状态
	state int32
	
	// 上次活动时间
	lastActivity time.Time
	
	// 执行次数
	executions int64
}

// NewNoIsolationSandbox 创建一个新的无隔离沙箱
func NewNoIsolationSandbox(id string, config api.IsolationConfig, logger hclog.Logger) *NoIsolationSandbox {
	return &NoIsolationSandbox{
		id:           id,
		config:       config,
		logger:       logger,
		state:        sandboxStateRunning,
		lastActivity: time.Now(),
	}
}

// GetID 获取沙箱ID
func (s *NoIsolationSandbox) GetID() string {
	return s.id
}

// GetConfig 获取隔离配置
func (s *NoIsolationSandbox) GetConfig() api.IsolationConfig {
	return s.config
}

// Execute 执行函数
func (s *NoIsolationSandbox) Execute(f func() error) error {
	// 检查沙箱状态
	if atomic.LoadInt32(&s.state) != sandboxStateRunning {
		return fmt.Errorf("沙箱 %s 未运行", s.id)
	}
	
	// 更新最后活动时间
	s.lastActivity = time.Now()
	
	// 增加执行次数
	atomic.AddInt64(&s.executions, 1)
	
	// 直接执行函数
	return f()
}

// ExecuteWithContext 执行带上下文的函数
func (s *NoIsolationSandbox) ExecuteWithContext(ctx context.Context, f func(context.Context) error) error {
	// 检查沙箱状态
	if atomic.LoadInt32(&s.state) != sandboxStateRunning {
		return fmt.Errorf("沙箱 %s 未运行", s.id)
	}
	
	// 更新最后活动时间
	s.lastActivity = time.Now()
	
	// 增加执行次数
	atomic.AddInt64(&s.executions, 1)
	
	// 直接执行函数
	return f(ctx)
}

// ExecuteWithTimeout 执行带超时的函数
func (s *NoIsolationSandbox) ExecuteWithTimeout(timeout time.Duration, f func() error) error {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// 使用带上下文的执行
	return s.ExecuteWithContext(ctx, func(ctx context.Context) error {
		// 监听上下文取消
		done := make(chan error, 1)
		go func() {
			done <- f()
		}()
		
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

// Pause 暂停沙箱
func (s *NoIsolationSandbox) Pause() error {
	// 检查沙箱状态
	if !atomic.CompareAndSwapInt32(&s.state, sandboxStateRunning, sandboxStatePaused) {
		return fmt.Errorf("沙箱 %s 未运行", s.id)
	}
	
	s.logger.Info("沙箱已暂停", "id", s.id)
	return nil
}

// Resume 恢复沙箱
func (s *NoIsolationSandbox) Resume() error {
	// 检查沙箱状态
	if !atomic.CompareAndSwapInt32(&s.state, sandboxStatePaused, sandboxStateRunning) {
		return fmt.Errorf("沙箱 %s 未暂停", s.id)
	}
	
	s.logger.Info("沙箱已恢复", "id", s.id)
	return nil
}

// Stop 停止沙箱
func (s *NoIsolationSandbox) Stop() error {
	// 检查沙箱状态
	state := atomic.LoadInt32(&s.state)
	if state == sandboxStateStopped {
		return nil
	}
	
	// 设置状态为停止
	atomic.StoreInt32(&s.state, sandboxStateStopped)
	
	s.logger.Info("沙箱已停止", "id", s.id)
	return nil
}

// IsHealthy 检查沙箱是否健康
func (s *NoIsolationSandbox) IsHealthy() bool {
	// 检查沙箱状态
	return atomic.LoadInt32(&s.state) == sandboxStateRunning
}

// GetStats 获取统计信息
func (s *NoIsolationSandbox) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"id":             s.id,
		"state":          atomic.LoadInt32(&s.state),
		"executions":     atomic.LoadInt64(&s.executions),
		"last_activity":  s.lastActivity,
	}
}
