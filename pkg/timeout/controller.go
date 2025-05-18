package timeout

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// OperationType 表示操作类型
type OperationType string

// 预定义操作类型
const (
	OperationTypeIO       OperationType = "io"
	OperationTypeNetwork  OperationType = "network"
	OperationTypeDatabase OperationType = "database"
	OperationTypePlugin   OperationType = "plugin"
	OperationTypeGeneric  OperationType = "generic"
)

// Operation 表示一个需要超时控制的操作
type Operation struct {
	ID          string
	Type        OperationType
	Description string
	StartTime   time.Time
	Timeout     time.Duration
	Context     context.Context
	Cancel      context.CancelFunc
}

// TimeoutController 管理操作的超时和取消
type TimeoutController struct {
	operations     map[string]*Operation
	mu             sync.RWMutex
	logger         hclog.Logger
	defaultTimeout map[OperationType]time.Duration
	monitorTicker  *time.Ticker
	stopChan       chan struct{}
}

// TimeoutControllerOption 超时控制器配置选项
type TimeoutControllerOption func(*TimeoutController)

// WithLogger 设置日志记录器
func WithLogger(logger hclog.Logger) TimeoutControllerOption {
	return func(tc *TimeoutController) {
		tc.logger = logger
	}
}

// WithDefaultTimeout 设置默认超时时间
func WithDefaultTimeout(opType OperationType, timeout time.Duration) TimeoutControllerOption {
	return func(tc *TimeoutController) {
		tc.defaultTimeout[opType] = timeout
	}
}

// WithMonitorInterval 设置监控间隔
func WithMonitorInterval(interval time.Duration) TimeoutControllerOption {
	return func(tc *TimeoutController) {
		if tc.monitorTicker != nil {
			tc.monitorTicker.Stop()
		}
		tc.monitorTicker = time.NewTicker(interval)
	}
}

// NewTimeoutController 创建一个新的超时控制器
func NewTimeoutController(options ...TimeoutControllerOption) *TimeoutController {
	tc := &TimeoutController{
		operations: make(map[string]*Operation),
		logger:     hclog.NewNullLogger(),
		defaultTimeout: map[OperationType]time.Duration{
			OperationTypeIO:       30 * time.Second,
			OperationTypeNetwork:  60 * time.Second,
			OperationTypeDatabase: 30 * time.Second,
			OperationTypePlugin:   60 * time.Second,
			OperationTypeGeneric:  30 * time.Second,
		},
		monitorTicker: time.NewTicker(5 * time.Second),
		stopChan:      make(chan struct{}),
	}

	// 应用选项
	for _, option := range options {
		option(tc)
	}

	// 启动监控协程
	go tc.monitorOperations()

	return tc
}

// CreateOperation 创建一个新的操作
func (tc *TimeoutController) CreateOperation(opType OperationType, id string, description string, timeout time.Duration) (*Operation, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// 检查操作是否已存在
	if _, exists := tc.operations[id]; exists {
		return nil, fmt.Errorf("操作已存在: %s", id)
	}

	// 如果未指定超时时间，使用默认值
	if timeout <= 0 {
		var ok bool
		timeout, ok = tc.defaultTimeout[opType]
		if !ok {
			timeout = tc.defaultTimeout[OperationTypeGeneric]
		}
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// 创建操作
	op := &Operation{
		ID:          id,
		Type:        opType,
		Description: description,
		StartTime:   time.Now(),
		Timeout:     timeout,
		Context:     ctx,
		Cancel:      cancel,
	}

	// 存储操作
	tc.operations[id] = op

	tc.logger.Debug("创建操作", "id", id, "type", opType, "timeout", timeout)
	return op, nil
}

// GetOperation 获取操作
func (tc *TimeoutController) GetOperation(id string) (*Operation, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	op, exists := tc.operations[id]
	return op, exists
}

// CancelOperation 取消操作
func (tc *TimeoutController) CancelOperation(id string) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	op, exists := tc.operations[id]
	if !exists {
		return fmt.Errorf("操作不存在: %s", id)
	}

	// 取消上下文
	op.Cancel()

	// 从映射中删除
	delete(tc.operations, id)

	tc.logger.Debug("取消操作", "id", id, "type", op.Type)
	return nil
}

// CompleteOperation 完成操作
func (tc *TimeoutController) CompleteOperation(id string) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	op, exists := tc.operations[id]
	if !exists {
		return fmt.Errorf("操作不存在: %s", id)
	}

	// 取消上下文
	op.Cancel()

	// 从映射中删除
	delete(tc.operations, id)

	tc.logger.Debug("完成操作", "id", id, "type", op.Type, "duration", time.Since(op.StartTime))
	return nil
}

// ListOperations 列出所有操作
func (tc *TimeoutController) ListOperations() []*Operation {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	operations := make([]*Operation, 0, len(tc.operations))
	for _, op := range tc.operations {
		operations = append(operations, op)
	}

	return operations
}

// monitorOperations 监控操作，清理已超时的操作
func (tc *TimeoutController) monitorOperations() {
	for {
		select {
		case <-tc.monitorTicker.C:
			tc.cleanupTimedOutOperations()
		case <-tc.stopChan:
			tc.monitorTicker.Stop()
			return
		}
	}
}

// cleanupTimedOutOperations 清理已超时的操作
func (tc *TimeoutController) cleanupTimedOutOperations() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for id, op := range tc.operations {
		select {
		case <-op.Context.Done():
			// 操作已超时或被取消
			if op.Context.Err() == context.DeadlineExceeded {
				tc.logger.Warn("操作超时", "id", id, "type", op.Type, "timeout", op.Timeout)
			} else {
				tc.logger.Debug("操作已取消", "id", id, "type", op.Type)
			}
			delete(tc.operations, id)
		default:
			// 操作仍在进行中
		}
	}
}

// Stop 停止超时控制器
func (tc *TimeoutController) Stop() {
	close(tc.stopChan)

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// 取消所有操作
	for id, op := range tc.operations {
		op.Cancel()
		tc.logger.Debug("停止时取消操作", "id", id, "type", op.Type)
	}

	// 清空操作映射
	tc.operations = make(map[string]*Operation)
}

// WithTimeout 使用超时执行函数
func (tc *TimeoutController) WithTimeout(opType OperationType, id string, description string, timeout time.Duration, fn func(context.Context) error) error {
	// 创建操作
	op, err := tc.CreateOperation(opType, id, description, timeout)
	if err != nil {
		return fmt.Errorf("创建操作失败: %w", err)
	}

	// 确保操作完成
	defer tc.CompleteOperation(op.ID)

	// 执行函数
	return fn(op.Context)
}

// ExecuteWithTimeout 使用超时执行函数（自动生成ID）
func (tc *TimeoutController) ExecuteWithTimeout(opType OperationType, description string, timeout time.Duration, fn func(context.Context) error) error {
	// 生成唯一ID
	id := fmt.Sprintf("%s-%d", opType, time.Now().UnixNano())
	return tc.WithTimeout(opType, id, description, timeout, fn)
}
