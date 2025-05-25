package executor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lomehong/kennel/app/dlp/engine"
	"github.com/lomehong/kennel/pkg/logging"
)

// ExecutionManagerImpl 执行管理器实现
type ExecutionManagerImpl struct {
	executors         map[engine.PolicyAction]ActionExecutor
	config            ExecutorConfig
	logger            logging.Logger
	stats             ManagerStats
	metricsCollector  MetricsCollector
	notificationService NotificationService
	running           int32
	mu                sync.RWMutex
}

// NewExecutionManager 创建执行管理器
func NewExecutionManager(logger logging.Logger, config ExecutorConfig) ExecutionManager {
	return &ExecutionManagerImpl{
		executors:           make(map[engine.PolicyAction]ActionExecutor),
		config:              config,
		logger:              logger,
		metricsCollector:    NewMetricsCollector(),
		notificationService: NewNotificationService(logger),
		stats: ManagerStats{
			ExecutorStats:      make(map[string]ExecutorStats),
			ActionDistribution: make(map[string]uint64),
			StartTime:          time.Now(),
		},
	}
}

// RegisterExecutor 注册执行器
func (em *ExecutionManagerImpl) RegisterExecutor(actionType engine.PolicyAction, executor ActionExecutor) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.executors[actionType]; exists {
		return fmt.Errorf("执行器已存在: %s", actionType.String())
	}

	// 初始化执行器
	if err := executor.Initialize(em.config); err != nil {
		return fmt.Errorf("初始化执行器失败: %w", err)
	}

	em.executors[actionType] = executor
	em.stats.ExecutorStats[actionType.String()] = executor.GetStats()

	em.logger.Info("注册动作执行器", "action", actionType.String())
	return nil
}

// GetExecutor 获取执行器
func (em *ExecutionManagerImpl) GetExecutor(actionType engine.PolicyAction) (ActionExecutor, bool) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	executor, exists := em.executors[actionType]
	return executor, exists
}

// ExecuteDecision 执行决策
func (em *ExecutionManagerImpl) ExecuteDecision(ctx context.Context, decision *engine.PolicyDecision) (*ExecutionResult, error) {
	startTime := time.Now()
	atomic.AddUint64(&em.stats.TotalRequests, 1)

	// 获取对应的执行器
	executor, exists := em.GetExecutor(decision.Action)
	if !exists {
		atomic.AddUint64(&em.stats.FailedRequests, 1)
		em.stats.LastError = fmt.Errorf("未找到执行器: %s", decision.Action.String())
		return nil, em.stats.LastError
	}

	// 执行动作
	result, err := em.executeWithRetry(ctx, executor, decision)
	if err != nil {
		atomic.AddUint64(&em.stats.FailedRequests, 1)
		em.stats.LastError = err
		em.metricsCollector.RecordError(decision.Action, err.Error())
		return nil, fmt.Errorf("执行动作失败: %w", err)
	}

	// 更新统计信息
	atomic.AddUint64(&em.stats.ProcessedRequests, 1)
	processingTime := time.Since(startTime)
	em.updateAverageTime(processingTime)

	// 更新动作分布统计
	em.mu.Lock()
	em.stats.ActionDistribution[decision.Action.String()]++
	em.mu.Unlock()

	// 记录指标
	em.metricsCollector.RecordExecution(decision.Action, processingTime, result.Success)

	// 发送通知（如果需要）
	if em.shouldSendNotification(decision, result) {
		go em.sendNotification(decision, result)
	}

	em.logger.Debug("执行决策完成",
		"decision_id", decision.ID,
		"action", decision.Action.String(),
		"success", result.Success,
		"processing_time", processingTime)

	return result, nil
}

// GetSupportedActions 获取支持的动作类型
func (em *ExecutionManagerImpl) GetSupportedActions() []engine.PolicyAction {
	em.mu.RLock()
	defer em.mu.RUnlock()

	actions := make([]engine.PolicyAction, 0, len(em.executors))
	for action := range em.executors {
		actions = append(actions, action)
	}

	return actions
}

// GetStats 获取统计信息
func (em *ExecutionManagerImpl) GetStats() ManagerStats {
	em.mu.RLock()
	defer em.mu.RUnlock()

	stats := em.stats
	stats.Uptime = time.Since(em.stats.StartTime)

	// 更新执行器统计信息
	for action, executor := range em.executors {
		stats.ExecutorStats[action.String()] = executor.GetStats()
	}

	return stats
}

// Start 启动管理器
func (em *ExecutionManagerImpl) Start() error {
	if !atomic.CompareAndSwapInt32(&em.running, 0, 1) {
		return fmt.Errorf("执行管理器已在运行")
	}

	em.logger.Info("启动执行管理器")

	// 注册默认执行器
	if err := em.registerDefaultExecutors(); err != nil {
		return fmt.Errorf("注册默认执行器失败: %w", err)
	}

	// 启动指标收集
	if em.config.EnableMetrics {
		go em.metricsWorker()
	}

	em.logger.Info("执行管理器已启动")
	return nil
}

// Stop 停止管理器
func (em *ExecutionManagerImpl) Stop() error {
	if !atomic.CompareAndSwapInt32(&em.running, 1, 0) {
		return fmt.Errorf("执行管理器未在运行")
	}

	em.logger.Info("停止执行管理器")

	// 清理所有执行器
	em.mu.RLock()
	for action, executor := range em.executors {
		if err := executor.Cleanup(); err != nil {
			em.logger.Error("清理执行器失败", "action", action.String(), "error", err)
		}
	}
	em.mu.RUnlock()

	em.logger.Info("执行管理器已停止")
	return nil
}

// HealthCheck 健康检查
func (em *ExecutionManagerImpl) HealthCheck() error {
	if atomic.LoadInt32(&em.running) == 0 {
		return fmt.Errorf("执行管理器未运行")
	}

	em.mu.RLock()
	executorCount := len(em.executors)
	em.mu.RUnlock()

	if executorCount == 0 {
		return fmt.Errorf("没有注册任何执行器")
	}

	return nil
}

// executeWithRetry 带重试的执行
func (em *ExecutionManagerImpl) executeWithRetry(ctx context.Context, executor ActionExecutor, decision *engine.PolicyDecision) (*ExecutionResult, error) {
	retryPolicy := DefaultRetryPolicy()
	var lastErr error

	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 执行动作
		result, err := executor.ExecuteAction(ctx, decision)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// 检查是否应该重试
		if attempt < retryPolicy.MaxRetries && em.shouldRetry(err, retryPolicy) {
			delay := em.calculateRetryDelay(attempt, retryPolicy)
			em.logger.Warn("执行失败，准备重试",
				"attempt", attempt+1,
				"max_retries", retryPolicy.MaxRetries,
				"delay", delay,
				"error", err)

			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		break
	}

	return nil, fmt.Errorf("执行失败，已达到最大重试次数: %w", lastErr)
}

// shouldRetry 检查是否应该重试
func (em *ExecutionManagerImpl) shouldRetry(err error, policy *RetryPolicy) bool {
	errorStr := err.Error()
	for _, retryableError := range policy.RetryableErrors {
		if errorStr == retryableError {
			return true
		}
	}
	return false
}

// calculateRetryDelay 计算重试延迟
func (em *ExecutionManagerImpl) calculateRetryDelay(attempt int, policy *RetryPolicy) time.Duration {
	delay := time.Duration(float64(policy.InitialDelay) * 
		(policy.BackoffFactor * float64(attempt)))
	
	if delay > policy.MaxDelay {
		delay = policy.MaxDelay
	}
	
	return delay
}

// shouldSendNotification 检查是否应该发送通知
func (em *ExecutionManagerImpl) shouldSendNotification(decision *engine.PolicyDecision, result *ExecutionResult) bool {
	// 对于阻断和告警动作，总是发送通知
	switch decision.Action {
	case engine.PolicyActionBlock, engine.PolicyActionAlert:
		return true
	case engine.PolicyActionQuarantine, engine.PolicyActionEncrypt:
		return true
	}

	// 对于执行失败的情况，发送通知
	if !result.Success {
		return true
	}

	return false
}

// sendNotification 发送通知
func (em *ExecutionManagerImpl) sendNotification(decision *engine.PolicyDecision, result *ExecutionResult) {
	notification := &Notification{
		ID:        fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		Title:     fmt.Sprintf("DLP动作执行: %s", decision.Action.String()),
		Message:   em.buildNotificationMessage(decision, result),
		Level:     em.getNotificationLevel(decision),
		Channel:   "default",
		Recipients: []string{"admin@example.com"},
		Metadata: map[string]interface{}{
			"decision_id": decision.ID,
			"action":      decision.Action.String(),
			"risk_level":  decision.RiskLevel.String(),
			"success":     result.Success,
		},
		Timestamp: time.Now(),
	}

	if err := em.notificationService.SendNotification(notification); err != nil {
		em.logger.Error("发送通知失败", "error", err)
	}
}

// buildNotificationMessage 构建通知消息
func (em *ExecutionManagerImpl) buildNotificationMessage(decision *engine.PolicyDecision, result *ExecutionResult) string {
	if result.Success {
		return fmt.Sprintf("成功执行%s动作，风险级别: %s，匹配规则: %d个",
			decision.Action.String(),
			decision.RiskLevel.String(),
			len(decision.MatchedRules))
	} else {
		return fmt.Sprintf("执行%s动作失败: %v",
			decision.Action.String(),
			result.Error)
	}
}

// getNotificationLevel 获取通知级别
func (em *ExecutionManagerImpl) getNotificationLevel(decision *engine.PolicyDecision) AlertLevel {
	switch decision.Action {
	case engine.PolicyActionBlock:
		return AlertLevelError
	case engine.PolicyActionAlert:
		return AlertLevelWarning
	case engine.PolicyActionQuarantine:
		return AlertLevelWarning
	default:
		return AlertLevelInfo
	}
}

// updateAverageTime 更新平均处理时间
func (em *ExecutionManagerImpl) updateAverageTime(duration time.Duration) {
	// 简化的平均时间计算
	em.stats.AverageTime = (em.stats.AverageTime + duration) / 2
}

// registerDefaultExecutors 注册默认执行器
func (em *ExecutionManagerImpl) registerDefaultExecutors() error {
	// 注册阻断执行器
	blockExecutor := NewBlockExecutor(em.logger)
	if err := em.RegisterExecutor(engine.PolicyActionBlock, blockExecutor); err != nil {
		return fmt.Errorf("注册阻断执行器失败: %w", err)
	}

	// 注册告警执行器
	alertExecutor := NewAlertExecutor(em.logger)
	if err := em.RegisterExecutor(engine.PolicyActionAlert, alertExecutor); err != nil {
		return fmt.Errorf("注册告警执行器失败: %w", err)
	}

	// 注册审计执行器
	auditExecutor := NewAuditExecutor(em.logger)
	if err := em.RegisterExecutor(engine.PolicyActionAudit, auditExecutor); err != nil {
		return fmt.Errorf("注册审计执行器失败: %w", err)
	}

	// 注册加密执行器
	encryptExecutor := NewEncryptExecutor(em.logger)
	if err := em.RegisterExecutor(engine.PolicyActionEncrypt, encryptExecutor); err != nil {
		return fmt.Errorf("注册加密执行器失败: %w", err)
	}

	// 注册隔离执行器
	quarantineExecutor := NewQuarantineExecutor(em.logger)
	if err := em.RegisterExecutor(engine.PolicyActionQuarantine, quarantineExecutor); err != nil {
		return fmt.Errorf("注册隔离执行器失败: %w", err)
	}

	// 注册重定向执行器
	redirectExecutor := NewRedirectExecutor(em.logger)
	if err := em.RegisterExecutor(engine.PolicyActionRedirect, redirectExecutor); err != nil {
		return fmt.Errorf("注册重定向执行器失败: %w", err)
	}

	// 注册允许执行器（默认动作）
	allowExecutor := NewAllowExecutor(em.logger)
	if err := em.RegisterExecutor(engine.PolicyActionAllow, allowExecutor); err != nil {
		return fmt.Errorf("注册允许执行器失败: %w", err)
	}

	return nil
}

// metricsWorker 指标收集工作协程
func (em *ExecutionManagerImpl) metricsWorker() {
	ticker := time.NewTicker(em.config.MetricsInterval)
	defer ticker.Stop()

	for {
		if atomic.LoadInt32(&em.running) == 0 {
			return
		}

		select {
		case <-ticker.C:
			metrics := em.metricsCollector.GetMetrics()
			em.logger.Debug("执行器指标", "metrics", metrics)
		}
	}
}
