package selfprotect

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/hashicorp/go-hclog"
)

// ProtectionService 自我防护服务
type ProtectionService struct {
	manager *ProtectionManager
	config  *ProtectionConfig
	logger  hclog.Logger
	started bool
}

// NewProtectionService 创建自我防护服务
func NewProtectionService(configFile string, logger hclog.Logger) (*ProtectionService, error) {
	// 读取配置文件
	yamlData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 加载防护配置
	config, err := LoadProtectionConfigFromYAML(yamlData)
	if err != nil {
		return nil, fmt.Errorf("加载防护配置失败: %w", err)
	}

	// 验证配置
	if err := ValidateProtectionConfig(config); err != nil {
		return nil, fmt.Errorf("验证防护配置失败: %w", err)
	}

	// 创建防护管理器
	manager := NewProtectionManager(config, logger)

	return &ProtectionService{
		manager: manager,
		config:  config,
		logger:  logger.Named("protection-service"),
		started: false,
	}, nil
}

// Start 启动自我防护服务
func (ps *ProtectionService) Start() error {
	if ps.started {
		return fmt.Errorf("自我防护服务已经启动")
	}

	if !ps.config.Enabled {
		ps.logger.Info("自我防护功能已禁用")
		return nil
	}

	ps.logger.Info("启动自我防护服务", "level", ps.config.Level)

	// 启动防护管理器
	if err := ps.manager.Start(); err != nil {
		return fmt.Errorf("启动防护管理器失败: %w", err)
	}

	ps.started = true
	ps.logger.Info("自我防护服务已启动")

	return nil
}

// Stop 停止自我防护服务
func (ps *ProtectionService) Stop() {
	if !ps.started {
		return
	}

	ps.logger.Info("停止自我防护服务")
	ps.manager.Stop()
	ps.started = false
	ps.logger.Info("自我防护服务已停止")
}

// IsEnabled 检查是否启用
func (ps *ProtectionService) IsEnabled() bool {
	return ps.started && ps.manager.IsEnabled()
}

// GetStatus 获取防护状态
func (ps *ProtectionService) GetStatus() ProtectionStatus {
	if !ps.started {
		return ProtectionStatus{
			Enabled:   false,
			Level:     string(ProtectionLevelNone),
			StartTime: time.Time{},
			Stats:     ProtectionStats{},
		}
	}

	stats := ps.manager.GetStats()
	return ProtectionStatus{
		Enabled:   ps.manager.IsEnabled(),
		Level:     string(ps.config.Level),
		StartTime: stats.StartTime,
		Stats:     stats,
	}
}

// GetEvents 获取防护事件
func (ps *ProtectionService) GetEvents() []ProtectionEvent {
	if !ps.started {
		return []ProtectionEvent{}
	}

	return ps.manager.GetEvents()
}

// GetConfig 获取防护配置
func (ps *ProtectionService) GetConfig() *ProtectionConfig {
	return ps.config
}

// ProtectionStatus 防护状态
type ProtectionStatus struct {
	Enabled   bool            `json:"enabled"`
	Level     string          `json:"level"`
	StartTime time.Time       `json:"start_time"`
	Stats     ProtectionStats `json:"stats"`
}

// ProtectionIntegrator 防护集成器
type ProtectionIntegrator struct {
	service *ProtectionService
	logger  hclog.Logger
}

// NewProtectionIntegrator 创建防护集成器
func NewProtectionIntegrator(configFile string, logger hclog.Logger) (*ProtectionIntegrator, error) {
	service, err := NewProtectionService(configFile, logger)
	if err != nil {
		return nil, err
	}

	return &ProtectionIntegrator{
		service: service,
		logger:  logger.Named("protection-integrator"),
	}, nil
}

// Initialize 初始化防护
func (pi *ProtectionIntegrator) Initialize() error {
	pi.logger.Info("初始化自我防护")

	// 启动防护服务
	if err := pi.service.Start(); err != nil {
		return fmt.Errorf("启动防护服务失败: %w", err)
	}

	// 注册优雅关闭处理
	pi.registerShutdownHandler()

	return nil
}

// Shutdown 关闭防护
func (pi *ProtectionIntegrator) Shutdown() {
	pi.logger.Info("关闭自我防护")
	pi.service.Stop()
}

// GetService 获取防护服务
func (pi *ProtectionIntegrator) GetService() *ProtectionService {
	return pi.service
}

// registerShutdownHandler 注册关闭处理器
func (pi *ProtectionIntegrator) registerShutdownHandler() {
	// 这里可以注册信号处理器，确保程序退出时正确关闭防护
	// 实际实现中应该使用 signal.Notify 等机制
}

// ProtectionMiddleware 防护中间件
type ProtectionMiddleware struct {
	service *ProtectionService
	logger  hclog.Logger
}

// NewProtectionMiddleware 创建防护中间件
func NewProtectionMiddleware(service *ProtectionService, logger hclog.Logger) *ProtectionMiddleware {
	return &ProtectionMiddleware{
		service: service,
		logger:  logger.Named("protection-middleware"),
	}
}

// CheckProtection 检查防护状态
func (pm *ProtectionMiddleware) CheckProtection() error {
	if !pm.service.IsEnabled() {
		return fmt.Errorf("自我防护未启用")
	}

	status := pm.service.GetStatus()
	if !status.Enabled {
		return fmt.Errorf("自我防护已禁用")
	}

	return nil
}

// RecordEvent 记录防护事件
func (pm *ProtectionMiddleware) RecordEvent(eventType ProtectionType, action, target, message string, details map[string]interface{}) {
	if !pm.service.IsEnabled() {
		return
	}

	// 这里应该调用防护管理器的事件记录方法
	// 简化实现，实际应该通过防护管理器记录
	pm.logger.Info("记录防护事件",
		"type", eventType,
		"action", action,
		"target", target,
		"message", message,
	)
}

// ProtectionHealthChecker 防护健康检查器
type ProtectionHealthChecker struct {
	service *ProtectionService
	logger  hclog.Logger
}

// NewProtectionHealthChecker 创建防护健康检查器
func NewProtectionHealthChecker(service *ProtectionService, logger hclog.Logger) *ProtectionHealthChecker {
	return &ProtectionHealthChecker{
		service: service,
		logger:  logger.Named("protection-health-checker"),
	}
}

// CheckHealth 检查防护健康状态
func (phc *ProtectionHealthChecker) CheckHealth() HealthCheckResult {
	if !phc.service.started {
		return HealthCheckResult{
			Status:  "unhealthy",
			Message: "自我防护服务未启动",
			Details: map[string]interface{}{
				"started": false,
			},
		}
	}

	if !phc.service.IsEnabled() {
		return HealthCheckResult{
			Status:  "disabled",
			Message: "自我防护功能已禁用",
			Details: map[string]interface{}{
				"enabled": false,
			},
		}
	}

	status := phc.service.GetStatus()
	stats := status.Stats

	// 检查健康指标
	healthScore := stats.ConfigHealthScore
	if healthScore < 50 {
		return HealthCheckResult{
			Status:  "unhealthy",
			Message: fmt.Sprintf("防护健康分数过低: %.2f", healthScore),
			Details: map[string]interface{}{
				"health_score": healthScore,
				"stats":        stats,
			},
		}
	}

	return HealthCheckResult{
		Status:  "healthy",
		Message: "自我防护运行正常",
		Details: map[string]interface{}{
			"enabled":      true,
			"level":        status.Level,
			"start_time":   status.StartTime,
			"health_score": healthScore,
			"stats":        stats,
		},
	}
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details"`
}

// ProtectionReporter 防护报告器
type ProtectionReporter struct {
	service *ProtectionService
	logger  hclog.Logger
}

// NewProtectionReporter 创建防护报告器
func NewProtectionReporter(service *ProtectionService, logger hclog.Logger) *ProtectionReporter {
	return &ProtectionReporter{
		service: service,
		logger:  logger.Named("protection-reporter"),
	}
}

// GenerateReport 生成防护报告
func (pr *ProtectionReporter) GenerateReport() ProtectionReport {
	status := pr.service.GetStatus()
	events := pr.service.GetEvents()
	config := pr.service.GetConfig()

	// 分析事件
	eventsByType := make(map[ProtectionType]int)
	recentEvents := []ProtectionEvent{}

	for _, event := range events {
		eventsByType[event.Type]++

		// 获取最近24小时的事件
		if time.Since(event.Timestamp) < 24*time.Hour {
			recentEvents = append(recentEvents, event)
		}
	}

	// 生成建议
	recommendations := pr.generateRecommendations(status, events, config)

	return ProtectionReport{
		GeneratedAt:     time.Now(),
		Status:          status,
		Config:          config,
		TotalEvents:     len(events),
		RecentEvents:    len(recentEvents),
		EventsByType:    eventsByType,
		Recommendations: recommendations,
		Events:          recentEvents,
	}
}

// generateRecommendations 生成建议
func (pr *ProtectionReporter) generateRecommendations(status ProtectionStatus, events []ProtectionEvent, config *ProtectionConfig) []string {
	var recommendations []string

	// 检查防护状态
	if !status.Enabled {
		recommendations = append(recommendations, "建议启用自我防护功能以提高系统安全性")
	}

	// 检查健康分数
	if status.Stats.ConfigHealthScore < 80 {
		recommendations = append(recommendations, "配置健康分数较低，建议检查配置文件和系统状态")
	}

	// 检查错误事件
	if status.Stats.ConfigErrors > 10 {
		recommendations = append(recommendations, "配置错误次数较多，建议检查配置文件的正确性")
	}

	// 检查重启次数
	if status.Stats.HotReloadFailures > 5 {
		recommendations = append(recommendations, "热更新失败次数较多，建议检查配置文件格式")
	}

	// 检查活跃告警
	if status.Stats.ActiveAlerts > 0 {
		recommendations = append(recommendations, fmt.Sprintf("当前有 %d 个活跃告警，建议及时处理", status.Stats.ActiveAlerts))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "系统运行正常，无特殊建议")
	}

	return recommendations
}

// ProtectionReport 防护报告
type ProtectionReport struct {
	GeneratedAt     time.Time              `json:"generated_at"`
	Status          ProtectionStatus       `json:"status"`
	Config          *ProtectionConfig      `json:"config"`
	TotalEvents     int                    `json:"total_events"`
	RecentEvents    int                    `json:"recent_events"`
	EventsByType    map[ProtectionType]int `json:"events_by_type"`
	Recommendations []string               `json:"recommendations"`
	Events          []ProtectionEvent      `json:"events"`
}
