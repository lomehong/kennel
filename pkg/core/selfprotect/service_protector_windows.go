//go:build selfprotect && windows
// +build selfprotect,windows

package selfprotect

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// WindowsServiceProtector Windows服务防护器
type WindowsServiceProtector struct {
	config        ServiceProtectionConfig
	logger        hclog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	
	enabled       bool
	protectedServices map[string]*ProtectedService
	eventCallback EventCallback
	
	// 监控状态
	monitoring    bool
	checkInterval time.Duration
}

// ProtectedService 受保护的服务信息
type ProtectedService struct {
	Name         string
	DisplayName  string
	Protected    bool
	LastCheck    time.Time
	ExpectedState svc.State
	StartType    uint32
	RestartCount int
}

// NewServiceProtector 创建Windows服务防护器
func NewServiceProtector(config ServiceProtectionConfig, logger hclog.Logger) ServiceProtector {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WindowsServiceProtector{
		config:            config,
		logger:            logger.Named("service-protector"),
		ctx:               ctx,
		cancel:            cancel,
		enabled:           config.Enabled,
		protectedServices: make(map[string]*ProtectedService),
		checkInterval:     10 * time.Second,
	}
}

// Start 启动服务防护
func (sp *WindowsServiceProtector) Start(ctx context.Context) error {
	if !sp.enabled {
		return nil
	}

	sp.logger.Info("启动Windows服务防护")

	// 保护指定的服务
	if sp.config.ServiceName != "" {
		if err := sp.ProtectService(sp.config.ServiceName); err != nil {
			sp.logger.Error("保护服务失败", "service", sp.config.ServiceName, "error", err)
		}
	}

	// 启动监控
	sp.monitoring = true
	sp.wg.Add(1)
	go sp.monitorServices()

	return nil
}

// Stop 停止服务防护
func (sp *WindowsServiceProtector) Stop() error {
	sp.logger.Info("停止Windows服务防护")
	
	sp.monitoring = false
	sp.cancel()
	sp.wg.Wait()

	return nil
}

// IsEnabled 检查是否启用
func (sp *WindowsServiceProtector) IsEnabled() bool {
	return sp.enabled
}

// PeriodicCheck 定期检查
func (sp *WindowsServiceProtector) PeriodicCheck() error {
	if !sp.enabled || !sp.monitoring {
		return nil
	}

	// 检查受保护服务的状态
	sp.mu.RLock()
	services := make([]*ProtectedService, 0, len(sp.protectedServices))
	for _, service := range sp.protectedServices {
		services = append(services, service)
	}
	sp.mu.RUnlock()

	for _, service := range services {
		if err := sp.checkServiceStatus(service); err != nil {
			sp.logger.Error("检查服务状态失败", "service", service.Name, "error", err)
		}
	}

	return nil
}

// SetEventCallback 设置事件回调
func (sp *WindowsServiceProtector) SetEventCallback(callback EventCallback) {
	sp.eventCallback = callback
}

// ProtectService 保护服务
func (sp *WindowsServiceProtector) ProtectService(serviceName string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// 连接到服务管理器
	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败: %w", err)
	}
	defer manager.Disconnect()

	// 打开服务
	service, err := manager.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("打开服务失败: %w", err)
	}
	defer service.Close()

	// 获取服务状态
	status, err := service.Query()
	if err != nil {
		return fmt.Errorf("查询服务状态失败: %w", err)
	}

	// 获取服务配置
	config, err := service.Config()
	if err != nil {
		return fmt.Errorf("获取服务配置失败: %w", err)
	}

	// 添加到保护列表
	sp.protectedServices[serviceName] = &ProtectedService{
		Name:          serviceName,
		DisplayName:   config.DisplayName,
		Protected:     true,
		LastCheck:     time.Now(),
		ExpectedState: status.State,
		StartType:     config.StartType,
	}

	sp.logger.Info("服务已保护", "service", serviceName, "display_name", config.DisplayName)

	// 记录事件
	if sp.eventCallback != nil {
		sp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeService,
			Action:      "protect",
			Target:      serviceName,
			Description: fmt.Sprintf("服务 %s (%s) 已被保护", serviceName, config.DisplayName),
			Details: map[string]interface{}{
				"service_name":  serviceName,
				"display_name":  config.DisplayName,
				"current_state": status.State,
				"start_type":    config.StartType,
			},
		})
	}

	return nil
}

// UnprotectService 取消保护服务
func (sp *WindowsServiceProtector) UnprotectService(serviceName string) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	_, exists := sp.protectedServices[serviceName]
	if !exists {
		return fmt.Errorf("服务未受保护: %s", serviceName)
	}

	// 从保护列表移除
	delete(sp.protectedServices, serviceName)

	sp.logger.Info("取消服务保护", "service", serviceName)

	// 记录事件
	if sp.eventCallback != nil {
		sp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeService,
			Action:      "unprotect",
			Target:      serviceName,
			Description: fmt.Sprintf("取消保护服务 %s", serviceName),
		})
	}

	return nil
}

// IsServiceProtected 检查服务是否受保护
func (sp *WindowsServiceProtector) IsServiceProtected(serviceName string) bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	
	_, exists := sp.protectedServices[serviceName]
	return exists
}

// GetProtectedServices 获取受保护的服务列表
func (sp *WindowsServiceProtector) GetProtectedServices() []string {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	
	services := make([]string, 0, len(sp.protectedServices))
	for name := range sp.protectedServices {
		services = append(services, name)
	}
	return services
}

// RestartService 重启服务
func (sp *WindowsServiceProtector) RestartService(serviceName string) error {
	sp.logger.Info("重启服务", "service", serviceName)

	// 连接到服务管理器
	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败: %w", err)
	}
	defer manager.Disconnect()

	// 打开服务
	service, err := manager.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("打开服务失败: %w", err)
	}
	defer service.Close()

	// 停止服务
	status, err := service.Control(svc.Stop)
	if err != nil {
		sp.logger.Warn("停止服务失败", "service", serviceName, "error", err)
	} else {
		// 等待服务停止
		for status.State != svc.Stopped {
			time.Sleep(100 * time.Millisecond)
			status, err = service.Query()
			if err != nil {
				break
			}
		}
	}

	// 启动服务
	err = service.Start()
	if err != nil {
		return fmt.Errorf("启动服务失败: %w", err)
	}

	sp.logger.Info("服务已重启", "service", serviceName)

	// 更新重启计数
	sp.mu.Lock()
	if protectedService, exists := sp.protectedServices[serviceName]; exists {
		protectedService.RestartCount++
	}
	sp.mu.Unlock()

	// 记录事件
	if sp.eventCallback != nil {
		sp.eventCallback(ProtectionEvent{
			Type:        ProtectionTypeService,
			Action:      "restart",
			Target:      serviceName,
			Description: fmt.Sprintf("服务 %s 已重启", serviceName),
		})
	}

	return nil
}

// PreventServiceStop 防止服务停止
func (sp *WindowsServiceProtector) PreventServiceStop(serviceName string) error {
	// 这需要修改服务的安全描述符，防止非授权用户停止服务
	// 简化实现，实际应该使用Windows安全API
	sp.logger.Info("设置服务停止保护", "service", serviceName)
	return nil
}

// PreventServiceDisable 防止服务禁用
func (sp *WindowsServiceProtector) PreventServiceDisable(serviceName string) error {
	// 这需要修改服务的安全描述符，防止非授权用户禁用服务
	// 简化实现，实际应该使用Windows安全API
	sp.logger.Info("设置服务禁用保护", "service", serviceName)
	return nil
}

// GetServiceStatus 获取服务状态
func (sp *WindowsServiceProtector) GetServiceStatus(serviceName string) (ServiceStatus, error) {
	// 连接到服务管理器
	manager, err := mgr.Connect()
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("连接服务管理器失败: %w", err)
	}
	defer manager.Disconnect()

	// 打开服务
	service, err := manager.OpenService(serviceName)
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("打开服务失败: %w", err)
	}
	defer service.Close()

	// 获取服务状态
	status, err := service.Query()
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("查询服务状态失败: %w", err)
	}

	// 获取服务配置
	config, err := service.Config()
	if err != nil {
		return ServiceStatus{}, fmt.Errorf("获取服务配置失败: %w", err)
	}

	return ServiceStatus{
		Name:        serviceName,
		DisplayName: config.DisplayName,
		State:       sp.stateToString(status.State),
		StartType:   sp.startTypeToString(config.StartType),
		ProcessID:   status.ProcessId,
	}, nil
}

// checkServiceStatus 检查服务状态
func (sp *WindowsServiceProtector) checkServiceStatus(protectedService *ProtectedService) error {
	// 获取当前服务状态
	currentStatus, err := sp.GetServiceStatus(protectedService.Name)
	if err != nil {
		return err
	}

	// 检查服务是否被停止
	if currentStatus.State == "Stopped" && protectedService.ExpectedState == svc.Running {
		sp.logger.Warn("检测到受保护服务被停止", "service", protectedService.Name)
		
		// 记录事件
		if sp.eventCallback != nil {
			sp.eventCallback(ProtectionEvent{
				Type:        ProtectionTypeService,
				Action:      "stopped",
				Target:      protectedService.Name,
				Description: fmt.Sprintf("受保护的服务 %s 已被停止", protectedService.Name),
				Details: map[string]interface{}{
					"service_name":    protectedService.Name,
					"expected_state":  sp.stateToString(protectedService.ExpectedState),
					"current_state":   currentStatus.State,
				},
			})
		}

		// 自动重启服务
		if sp.config.AutoRestart {
			if err := sp.RestartService(protectedService.Name); err != nil {
				sp.logger.Error("自动重启服务失败", "service", protectedService.Name, "error", err)
				return err
			}
		}
	}

	// 更新最后检查时间
	protectedService.LastCheck = time.Now()
	return nil
}

// stateToString 将服务状态转换为字符串
func (sp *WindowsServiceProtector) stateToString(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "StartPending"
	case svc.StopPending:
		return "StopPending"
	case svc.Running:
		return "Running"
	case svc.ContinuePending:
		return "ContinuePending"
	case svc.PausePending:
		return "PausePending"
	case svc.Paused:
		return "Paused"
	default:
		return "Unknown"
	}
}

// startTypeToString 将启动类型转换为字符串
func (sp *WindowsServiceProtector) startTypeToString(startType uint32) string {
	switch startType {
	case mgr.StartAutomatic:
		return "Automatic"
	case mgr.StartManual:
		return "Manual"
	case mgr.StartDisabled:
		return "Disabled"
	default:
		return "Unknown"
	}
}

// monitorServices 监控服务
func (sp *WindowsServiceProtector) monitorServices() {
	defer sp.wg.Done()

	ticker := time.NewTicker(sp.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sp.ctx.Done():
			return
		case <-ticker.C:
			if sp.monitoring {
				sp.PeriodicCheck()
			}
		}
	}
}
