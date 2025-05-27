//go:build selfprotect
// +build selfprotect

package selfprotect

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// ProtectionManager 防护管理器
type ProtectionManager struct {
	config *ProtectionConfig
	logger hclog.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// 防护组件
	processProtector  ProcessProtector
	fileProtector     FileProtector
	registryProtector RegistryProtector
	serviceProtector  ServiceProtector

	// 状态
	enabled       bool
	emergencyMode bool
	events        []ProtectionEvent
	maxEvents     int

	// 统计
	stats ProtectionStats
}

// DefaultProtectionConfig 默认防护配置
func DefaultProtectionConfig() *ProtectionConfig {
	return &ProtectionConfig{
		Enabled:            false, // 默认禁用，需要显式启用
		Level:              ProtectionLevelBasic,
		EmergencyDisable:   ".emergency_disable",
		CheckInterval:      5 * time.Second,
		RestartDelay:       3 * time.Second,
		MaxRestartAttempts: 3,
		Whitelist: WhitelistConfig{
			Enabled: true,
			Processes: []string{
				"taskmgr.exe",
				"procexp.exe",
				"procexp64.exe",
			},
			Users: []string{
				"SYSTEM",
				"Administrator",
			},
		},
		ProcessProtection: ProcessProtectionConfig{
			Enabled: true,
			ProtectedProcesses: []string{
				"agent.exe",
				"dlp.exe",
				"audit.exe",
				"device.exe",
			},
			MonitorChildren: true,
			PreventDebug:    true,
			PreventDump:     true,
		},
		FileProtection: FileProtectionConfig{
			Enabled: true,
			ProtectedFiles: []string{
				"config.yaml",
				"agent.exe",
			},
			ProtectedDirs: []string{
				"app",
			},
			CheckIntegrity: true,
			BackupEnabled:  true,
			BackupDir:      "backup",
		},
		RegistryProtection: RegistryProtectionConfig{
			Enabled: true,
			ProtectedKeys: []string{
				`HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\KennelAgent`,
			},
			MonitorChanges: true,
		},
		ServiceProtection: ServiceProtectionConfig{
			Enabled:        true,
			ServiceName:    "KennelAgent",
			AutoRestart:    true,
			PreventDisable: true,
		},
	}
}

// NewProtectionManager 创建防护管理器
func NewProtectionManager(config *ProtectionConfig, logger hclog.Logger) *ProtectionManager {
	if config == nil {
		config = DefaultProtectionConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pm := &ProtectionManager{
		config:    config,
		logger:    logger.Named("protection-manager"),
		ctx:       ctx,
		cancel:    cancel,
		enabled:   config.Enabled,
		maxEvents: 10000,
		stats: ProtectionStats{
			StartTime:         time.Now(),
			ConfigHealthScore: 100.0, // 初始健康分数
		},
	}

	// 初始化防护组件
	pm.initProtectors()

	return pm
}

// initProtectors 初始化防护组件
func (pm *ProtectionManager) initProtectors() {
	// 初始化进程防护器
	if pm.config.ProcessProtection.Enabled {
		pm.processProtector = NewProcessProtector(pm.config.ProcessProtection, pm.logger)
	}

	// 初始化文件防护器
	if pm.config.FileProtection.Enabled {
		pm.fileProtector = NewFileProtector(pm.config.FileProtection, pm.logger)
	}

	// 初始化注册表防护器
	if pm.config.RegistryProtection.Enabled {
		pm.registryProtector = NewRegistryProtector(pm.config.RegistryProtection, pm.logger)
	}

	// 初始化服务防护器
	if pm.config.ServiceProtection.Enabled {
		pm.serviceProtector = NewServiceProtector(pm.config.ServiceProtection, pm.logger)
	}
}

// Start 启动防护
func (pm *ProtectionManager) Start() error {
	if !pm.enabled {
		pm.logger.Info("自我防护已禁用")
		return nil
	}

	// 检查紧急禁用文件
	if pm.checkEmergencyDisable() {
		pm.logger.Warn("检测到紧急禁用文件，自我防护已禁用")
		pm.emergencyMode = true
		return nil
	}

	pm.logger.Info("启动自我防护", "level", pm.config.Level)

	// 启动各个防护组件
	if pm.processProtector != nil {
		pm.wg.Add(1)
		go pm.runProcessProtection()
	}

	if pm.fileProtector != nil {
		pm.wg.Add(1)
		go pm.runFileProtection()
	}

	if pm.registryProtector != nil {
		pm.wg.Add(1)
		go pm.runRegistryProtection()
	}

	if pm.serviceProtector != nil {
		pm.wg.Add(1)
		go pm.runServiceProtection()
	}

	// 启动主监控循环
	pm.wg.Add(1)
	go pm.runMainLoop()

	return nil
}

// Stop 停止防护
func (pm *ProtectionManager) Stop() {
	pm.logger.Info("停止自我防护")
	pm.cancel()
	pm.wg.Wait()
}

// IsEnabled 检查是否启用
func (pm *ProtectionManager) IsEnabled() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.enabled && !pm.emergencyMode
}

// GetStats 获取防护统计
func (pm *ProtectionManager) GetStats() ProtectionStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.stats
}

// GetEvents 获取防护事件
func (pm *ProtectionManager) GetEvents() []ProtectionEvent {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	events := make([]ProtectionEvent, len(pm.events))
	copy(events, pm.events)
	return events
}

// checkEmergencyDisable 检查紧急禁用
func (pm *ProtectionManager) checkEmergencyDisable() bool {
	if pm.config.EmergencyDisable == "" {
		return false
	}

	_, err := os.Stat(pm.config.EmergencyDisable)
	return err == nil
}

// recordEvent 记录防护事件
func (pm *ProtectionManager) recordEvent(event ProtectionEvent) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 设置事件ID和时间戳
	event.ID = fmt.Sprintf("event_%d", time.Now().UnixNano())
	event.Timestamp = time.Now()

	// 添加事件
	pm.events = append(pm.events, event)

	// 限制事件数量
	if len(pm.events) > pm.maxEvents {
		pm.events = pm.events[len(pm.events)-pm.maxEvents:]
	}

	// 更新统计
	pm.stats.TotalEvents++
	pm.stats.LastEvent = event.Timestamp

	if event.Blocked {
		pm.stats.BlockedEvents++
	}

	switch event.Type {
	case ProtectionTypeProcess:
		pm.stats.ProcessEvents++
	case ProtectionTypeFile:
		pm.stats.FileEvents++
	case ProtectionTypeRegistry:
		pm.stats.RegistryEvents++
	case ProtectionTypeService:
		pm.stats.ServiceEvents++
	}

	pm.logger.Info("记录防护事件",
		"type", event.Type,
		"action", event.Action,
		"target", event.Target,
		"blocked", event.Blocked,
	)
}

// runMainLoop 运行主监控循环
func (pm *ProtectionManager) runMainLoop() {
	defer pm.wg.Done()

	ticker := time.NewTicker(pm.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			// 检查紧急禁用
			if pm.checkEmergencyDisable() && !pm.emergencyMode {
				pm.logger.Warn("检测到紧急禁用文件，进入紧急模式")
				pm.mu.Lock()
				pm.emergencyMode = true
				pm.mu.Unlock()
			}

			// 如果在紧急模式，跳过防护检查
			if pm.emergencyMode {
				continue
			}

			// 执行定期检查
			pm.performPeriodicChecks()
		}
	}
}

// performPeriodicChecks 执行定期检查
func (pm *ProtectionManager) performPeriodicChecks() {
	// 检查各个防护组件的状态
	if pm.processProtector != nil {
		pm.processProtector.PeriodicCheck()
	}

	if pm.fileProtector != nil {
		pm.fileProtector.PeriodicCheck()
	}

	if pm.registryProtector != nil {
		pm.registryProtector.PeriodicCheck()
	}

	if pm.serviceProtector != nil {
		pm.serviceProtector.PeriodicCheck()
	}
}

// runProcessProtection 运行进程防护
func (pm *ProtectionManager) runProcessProtection() {
	defer pm.wg.Done()

	if pm.processProtector == nil {
		return
	}

	pm.logger.Info("启动进程防护")

	// 设置事件回调
	pm.processProtector.SetEventCallback(func(event ProtectionEvent) {
		pm.recordEvent(event)
	})

	// 启动进程防护
	pm.processProtector.Start(pm.ctx)
}

// runFileProtection 运行文件防护
func (pm *ProtectionManager) runFileProtection() {
	defer pm.wg.Done()

	if pm.fileProtector == nil {
		return
	}

	pm.logger.Info("启动文件防护")

	// 设置事件回调
	pm.fileProtector.SetEventCallback(func(event ProtectionEvent) {
		pm.recordEvent(event)
	})

	// 启动文件防护
	pm.fileProtector.Start(pm.ctx)
}

// runRegistryProtection 运行注册表防护
func (pm *ProtectionManager) runRegistryProtection() {
	defer pm.wg.Done()

	if pm.registryProtector == nil {
		return
	}

	pm.logger.Info("启动注册表防护")

	// 设置事件回调
	pm.registryProtector.SetEventCallback(func(event ProtectionEvent) {
		pm.recordEvent(event)
	})

	// 启动注册表防护
	pm.registryProtector.Start(pm.ctx)
}

// runServiceProtection 运行服务防护
func (pm *ProtectionManager) runServiceProtection() {
	defer pm.wg.Done()

	if pm.serviceProtector == nil {
		return
	}

	pm.logger.Info("启动服务防护")

	// 设置事件回调
	pm.serviceProtector.SetEventCallback(func(event ProtectionEvent) {
		pm.recordEvent(event)
	})

	// 启动服务防护
	pm.serviceProtector.Start(pm.ctx)
}
