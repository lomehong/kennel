package isolation

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// BasicIsolationSandbox 基本隔离沙箱
// 提供基本的错误隔离和资源限制
type BasicIsolationSandbox struct {
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
	
	// 统计信息
	stats struct {
		// 执行次数
		executions int64
		
		// 成功次数
		successes int64
		
		// 失败次数
		failures int64
		
		// 恐慌次数
		panics int64
		
		// 超时次数
		timeouts int64
		
		// 总执行时间
		totalExecTime int64
		
		// 最长执行时间
		maxExecTime int64
		
		// 最短执行时间
		minExecTime int64
		
		// 平均执行时间
		avgExecTime int64
	}
	
	// 互斥锁
	mu sync.RWMutex
}

// 沙箱状态
const (
	// 沙箱状态：运行中
	sandboxStateRunning int32 = iota
	
	// 沙箱状态：暂停
	sandboxStatePaused
	
	// 沙箱状态：停止
	sandboxStateStopped
)

// NewBasicIsolationSandbox 创建一个新的基本隔离沙箱
func NewBasicIsolationSandbox(id string, config api.IsolationConfig, logger hclog.Logger) *BasicIsolationSandbox {
	return &BasicIsolationSandbox{
		id:           id,
		config:       config,
		logger:       logger,
		state:        sandboxStateRunning,
		lastActivity: time.Now(),
	}
}

// GetID 获取沙箱ID
func (s *BasicIsolationSandbox) GetID() string {
	return s.id
}

// GetConfig 获取隔离配置
func (s *BasicIsolationSandbox) GetConfig() api.IsolationConfig {
	return s.config
}

// Execute 执行函数
func (s *BasicIsolationSandbox) Execute(f func() error) error {
	// 检查沙箱状态
	if atomic.LoadInt32(&s.state) != sandboxStateRunning {
		return fmt.Errorf("沙箱 %s 未运行", s.id)
	}
	
	// 更新最后活动时间
	s.lastActivity = time.Now()
	
	// 增加执行次数
	atomic.AddInt64(&s.stats.executions, 1)
	
	// 记录开始时间
	startTime := time.Now()
	
	// 创建错误通道
	errCh := make(chan error, 1)
	
	// 在goroutine中执行函数，捕获恐慌
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// 记录恐慌
				atomic.AddInt64(&s.stats.panics, 1)
				
				// 记录堆栈跟踪
				stack := debug.Stack()
				s.logger.Error("函数执行恐慌", "id", s.id, "panic", r, "stack", string(stack))
				
				// 发送错误
				errCh <- fmt.Errorf("函数执行恐慌: %v", r)
			}
		}()
		
		// 执行函数
		err := f()
		
		// 发送结果
		errCh <- err
	}()
	
	// 等待结果或超时
	var err error
	select {
	case err = <-errCh:
		// 函数执行完成
	case <-time.After(s.config.Timeout):
		// 超时
		atomic.AddInt64(&s.stats.timeouts, 1)
		err = fmt.Errorf("函数执行超时")
	}
	
	// 记录执行时间
	execTime := time.Since(startTime).Milliseconds()
	atomic.AddInt64(&s.stats.totalExecTime, execTime)
	
	// 更新最长执行时间
	for {
		maxExecTime := atomic.LoadInt64(&s.stats.maxExecTime)
		if execTime <= maxExecTime {
			break
		}
		if atomic.CompareAndSwapInt64(&s.stats.maxExecTime, maxExecTime, execTime) {
			break
		}
	}
	
	// 更新最短执行时间
	for {
		minExecTime := atomic.LoadInt64(&s.stats.minExecTime)
		if minExecTime == 0 || execTime < minExecTime {
			if atomic.CompareAndSwapInt64(&s.stats.minExecTime, minExecTime, execTime) {
				break
			}
		} else {
			break
		}
	}
	
	// 更新平均执行时间
	executions := atomic.LoadInt64(&s.stats.executions)
	if executions > 0 {
		totalExecTime := atomic.LoadInt64(&s.stats.totalExecTime)
		atomic.StoreInt64(&s.stats.avgExecTime, totalExecTime/executions)
	}
	
	// 更新成功/失败次数
	if err != nil {
		atomic.AddInt64(&s.stats.failures, 1)
	} else {
		atomic.AddInt64(&s.stats.successes, 1)
	}
	
	return err
}

// ExecuteWithContext 执行带上下文的函数
func (s *BasicIsolationSandbox) ExecuteWithContext(ctx context.Context, f func(context.Context) error) error {
	// 检查沙箱状态
	if atomic.LoadInt32(&s.state) != sandboxStateRunning {
		return fmt.Errorf("沙箱 %s 未运行", s.id)
	}
	
	// 更新最后活动时间
	s.lastActivity = time.Now()
	
	// 增加执行次数
	atomic.AddInt64(&s.stats.executions, 1)
	
	// 记录开始时间
	startTime := time.Now()
	
	// 创建错误通道
	errCh := make(chan error, 1)
	
	// 创建带超时的上下文
	timeoutCtx := ctx
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		timeoutCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}
	
	// 在goroutine中执行函数，捕获恐慌
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// 记录恐慌
				atomic.AddInt64(&s.stats.panics, 1)
				
				// 记录堆栈跟踪
				stack := debug.Stack()
				s.logger.Error("函数执行恐慌", "id", s.id, "panic", r, "stack", string(stack))
				
				// 发送错误
				errCh <- fmt.Errorf("函数执行恐慌: %v", r)
			}
		}()
		
		// 执行函数
		err := f(timeoutCtx)
		
		// 发送结果
		errCh <- err
	}()
	
	// 等待结果或上下文取消
	var err error
	select {
	case err = <-errCh:
		// 函数执行完成
	case <-timeoutCtx.Done():
		// 上下文取消或超时
		if timeoutCtx.Err() == context.DeadlineExceeded {
			atomic.AddInt64(&s.stats.timeouts, 1)
			err = fmt.Errorf("函数执行超时")
		} else {
			err = fmt.Errorf("上下文取消: %w", timeoutCtx.Err())
		}
	}
	
	// 记录执行时间
	execTime := time.Since(startTime).Milliseconds()
	atomic.AddInt64(&s.stats.totalExecTime, execTime)
	
	// 更新最长执行时间
	for {
		maxExecTime := atomic.LoadInt64(&s.stats.maxExecTime)
		if execTime <= maxExecTime {
			break
		}
		if atomic.CompareAndSwapInt64(&s.stats.maxExecTime, maxExecTime, execTime) {
			break
		}
	}
	
	// 更新最短执行时间
	for {
		minExecTime := atomic.LoadInt64(&s.stats.minExecTime)
		if minExecTime == 0 || execTime < minExecTime {
			if atomic.CompareAndSwapInt64(&s.stats.minExecTime, minExecTime, execTime) {
				break
			}
		} else {
			break
		}
	}
	
	// 更新平均执行时间
	executions := atomic.LoadInt64(&s.stats.executions)
	if executions > 0 {
		totalExecTime := atomic.LoadInt64(&s.stats.totalExecTime)
		atomic.StoreInt64(&s.stats.avgExecTime, totalExecTime/executions)
	}
	
	// 更新成功/失败次数
	if err != nil {
		atomic.AddInt64(&s.stats.failures, 1)
	} else {
		atomic.AddInt64(&s.stats.successes, 1)
	}
	
	return err
}

// ExecuteWithTimeout 执行带超时的函数
func (s *BasicIsolationSandbox) ExecuteWithTimeout(timeout time.Duration, f func() error) error {
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
func (s *BasicIsolationSandbox) Pause() error {
	// 检查沙箱状态
	if !atomic.CompareAndSwapInt32(&s.state, sandboxStateRunning, sandboxStatePaused) {
		return fmt.Errorf("沙箱 %s 未运行", s.id)
	}
	
	s.logger.Info("沙箱已暂停", "id", s.id)
	return nil
}

// Resume 恢复沙箱
func (s *BasicIsolationSandbox) Resume() error {
	// 检查沙箱状态
	if !atomic.CompareAndSwapInt32(&s.state, sandboxStatePaused, sandboxStateRunning) {
		return fmt.Errorf("沙箱 %s 未暂停", s.id)
	}
	
	s.logger.Info("沙箱已恢复", "id", s.id)
	return nil
}

// Stop 停止沙箱
func (s *BasicIsolationSandbox) Stop() error {
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
func (s *BasicIsolationSandbox) IsHealthy() bool {
	// 检查沙箱状态
	if atomic.LoadInt32(&s.state) != sandboxStateRunning {
		return false
	}
	
	// 检查失败率
	executions := atomic.LoadInt64(&s.stats.executions)
	if executions > 0 {
		failures := atomic.LoadInt64(&s.stats.failures)
		failureRate := float64(failures) / float64(executions)
		
		// 如果失败率超过50%，认为不健康
		if failureRate > 0.5 {
			return false
		}
	}
	
	// 检查恐慌次数
	panics := atomic.LoadInt64(&s.stats.panics)
	if panics > 5 {
		return false
	}
	
	return true
}

// GetStats 获取统计信息
func (s *BasicIsolationSandbox) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"id":             s.id,
		"state":          atomic.LoadInt32(&s.state),
		"executions":     atomic.LoadInt64(&s.stats.executions),
		"successes":      atomic.LoadInt64(&s.stats.successes),
		"failures":       atomic.LoadInt64(&s.stats.failures),
		"panics":         atomic.LoadInt64(&s.stats.panics),
		"timeouts":       atomic.LoadInt64(&s.stats.timeouts),
		"total_exec_time": atomic.LoadInt64(&s.stats.totalExecTime),
		"max_exec_time":  atomic.LoadInt64(&s.stats.maxExecTime),
		"min_exec_time":  atomic.LoadInt64(&s.stats.minExecTime),
		"avg_exec_time":  atomic.LoadInt64(&s.stats.avgExecTime),
		"last_activity":  s.lastActivity,
	}
}
