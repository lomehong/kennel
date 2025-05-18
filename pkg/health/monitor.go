package health

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// MonitorConfig 监控配置
type MonitorConfig struct {
	CheckInterval    time.Duration // 检查间隔
	InitialDelay     time.Duration // 初始延迟
	AutoRepair       bool          // 是否自动修复
	FailureThreshold int           // 失败阈值
	SuccessThreshold int           // 成功阈值
}

// DefaultMonitorConfig 默认监控配置
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		CheckInterval:    30 * time.Second,
		InitialDelay:     5 * time.Second,
		AutoRepair:       true,
		FailureThreshold: 3,
		SuccessThreshold: 1,
	}
}

// CheckerStatus 检查器状态
type CheckerStatus struct {
	Name                 string        // 检查器名称
	Status               Status        // 健康状态
	Message              string        // 状态消息
	LastChecked          time.Time     // 最后检查时间
	LastSuccess          time.Time     // 最后成功时间
	LastFailure          time.Time     // 最后失败时间
	ConsecutiveSuccesses int           // 连续成功次数
	ConsecutiveFailures  int           // 连续失败次数
	TotalChecks          int           // 总检查次数
	TotalSuccesses       int           // 总成功次数
	TotalFailures        int           // 总失败次数
	LastRepair           time.Time     // 最后修复时间
	LastRepairResult     *RepairResult // 最后修复结果
}

// HealthMonitor 健康监控器
type HealthMonitor struct {
	registry       *CheckerRegistry
	healer         *RepairSelfHealer
	config         *MonitorConfig
	checkerStatus  map[string]*CheckerStatus
	statusHistory  map[string][]CheckResult
	maxHistorySize int
	mu             sync.RWMutex
	logger         hclog.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewHealthMonitor 创建一个新的健康监控器
func NewHealthMonitor(registry *CheckerRegistry, healer *RepairSelfHealer, config *MonitorConfig, logger hclog.Logger) *HealthMonitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &HealthMonitor{
		registry:       registry,
		healer:         healer,
		config:         config,
		checkerStatus:  make(map[string]*CheckerStatus),
		statusHistory:  make(map[string][]CheckResult),
		maxHistorySize: 100,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start 启动监控
func (m *HealthMonitor) Start() {
	m.logger.Info("启动健康监控", "interval", m.config.CheckInterval, "auto_repair", m.config.AutoRepair)

	// 初始化检查器状态
	checkers := m.registry.ListCheckers()
	for _, checker := range checkers {
		m.mu.Lock()
		m.checkerStatus[checker.Name()] = &CheckerStatus{
			Name:   checker.Name(),
			Status: StatusUnknown,
		}
		m.statusHistory[checker.Name()] = make([]CheckResult, 0)
		m.mu.Unlock()

		// 为每个检查器启动一个goroutine
		m.wg.Add(1)
		go m.monitorChecker(checker.Name())
	}
}

// Stop 停止监控
func (m *HealthMonitor) Stop() {
	m.logger.Info("停止健康监控")
	m.cancel()
	m.wg.Wait()
}

// monitorChecker 监控检查器
func (m *HealthMonitor) monitorChecker(checkerName string) {
	defer m.wg.Done()

	// 初始延迟
	select {
	case <-time.After(m.config.InitialDelay):
	case <-m.ctx.Done():
		return
	}

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAndUpdateStatus(checkerName)
		case <-m.ctx.Done():
			return
		}
	}
}

// checkAndUpdateStatus 检查并更新状态
func (m *HealthMonitor) checkAndUpdateStatus(checkerName string) {
	ctx, cancel := context.WithTimeout(m.ctx, m.config.CheckInterval/2)
	defer cancel()

	// 运行健康检查
	checkResult, ok := m.registry.RunCheck(ctx, checkerName)
	if !ok {
		m.logger.Warn("健康检查器不存在", "name", checkerName)
		return
	}

	// 更新状态
	m.mu.Lock()
	status, exists := m.checkerStatus[checkerName]
	if !exists {
		status = &CheckerStatus{
			Name:   checkerName,
			Status: StatusUnknown,
		}
		m.checkerStatus[checkerName] = status
	}

	status.LastChecked = time.Now()
	status.TotalChecks++

	// 更新历史记录
	history, exists := m.statusHistory[checkerName]
	if !exists {
		history = make([]CheckResult, 0)
		m.statusHistory[checkerName] = history
	}
	m.statusHistory[checkerName] = append(history, checkResult)
	if len(m.statusHistory[checkerName]) > m.maxHistorySize {
		m.statusHistory[checkerName] = m.statusHistory[checkerName][1:]
	}

	// 更新成功/失败计数
	if checkResult.Status == StatusHealthy {
		status.Status = StatusHealthy
		status.Message = checkResult.Message
		status.LastSuccess = time.Now()
		status.TotalSuccesses++
		status.ConsecutiveSuccesses++
		status.ConsecutiveFailures = 0
	} else {
		status.Status = checkResult.Status
		status.Message = checkResult.Message
		status.LastFailure = time.Now()
		status.TotalFailures++
		status.ConsecutiveFailures++
		status.ConsecutiveSuccesses = 0
	}
	m.mu.Unlock()

	// 记录状态
	m.logger.Debug("健康检查结果",
		"name", checkerName,
		"status", checkResult.Status,
		"message", checkResult.Message,
		"consecutive_failures", status.ConsecutiveFailures,
		"consecutive_successes", status.ConsecutiveSuccesses,
	)

	// 检查是否需要修复
	if m.config.AutoRepair && status.ConsecutiveFailures >= m.config.FailureThreshold {
		m.logger.Info("尝试自动修复",
			"name", checkerName,
			"consecutive_failures", status.ConsecutiveFailures,
			"threshold", m.config.FailureThreshold,
		)

		// 执行修复
		_, repairResult, err := m.healer.CheckAndRepair(ctx, checkerName)
		if err != nil {
			m.logger.Error("自动修复失败", "name", checkerName, "error", err)
		} else if repairResult != nil {
			m.mu.Lock()
			status.LastRepair = time.Now()
			status.LastRepairResult = repairResult
			m.mu.Unlock()

			if repairResult.Success {
				m.logger.Info("自动修复成功", "name", checkerName, "action", repairResult.ActionName)
			} else {
				m.logger.Warn("自动修复未成功", "name", checkerName, "action", repairResult.ActionName, "message", repairResult.Message)
			}
		}
	}
}

// GetStatus 获取状态
func (m *HealthMonitor) GetStatus(checkerName string) (*CheckerStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status, ok := m.checkerStatus[checkerName]
	return status, ok
}

// GetAllStatus 获取所有状态
func (m *HealthMonitor) GetAllStatus() map[string]*CheckerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	statuses := make(map[string]*CheckerStatus)
	for name, status := range m.checkerStatus {
		statuses[name] = status
	}
	return statuses
}

// GetStatusHistory 获取状态历史
func (m *HealthMonitor) GetStatusHistory(checkerName string) ([]CheckResult, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	history, ok := m.statusHistory[checkerName]
	if !ok {
		return nil, false
	}
	result := make([]CheckResult, len(history))
	copy(result, history)
	return result, true
}

// SetMaxHistorySize 设置最大历史记录大小
func (m *HealthMonitor) SetMaxHistorySize(size int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxHistorySize = size
	for name, history := range m.statusHistory {
		if len(history) > m.maxHistorySize {
			m.statusHistory[name] = history[len(history)-m.maxHistorySize:]
		}
	}
}

// GetSystemHealth 获取系统健康状态
func (m *HealthMonitor) GetSystemHealth() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.checkerStatus) == 0 {
		return StatusUnknown
	}

	unhealthyCount := 0
	degradedCount := 0
	unknownCount := 0

	for _, status := range m.checkerStatus {
		switch status.Status {
		case StatusUnhealthy:
			unhealthyCount++
		case StatusDegraded:
			degradedCount++
		case StatusUnknown:
			unknownCount++
		}
	}

	if unhealthyCount > 0 {
		return StatusUnhealthy
	}
	if degradedCount > 0 {
		return StatusDegraded
	}
	if unknownCount == len(m.checkerStatus) {
		return StatusUnknown
	}
	return StatusHealthy
}
