//go:build !selfprotect
// +build !selfprotect

package selfprotect

import (
	"context"
	"time"

	"github.com/hashicorp/go-hclog"
)

// DisabledProtectionManager 禁用的防护管理器
type DisabledProtectionManager struct {
	logger hclog.Logger
}

// NewProtectionManager 创建禁用的防护管理器
func NewProtectionManager(config *ProtectionConfig, logger hclog.Logger) *DisabledProtectionManager {
	return &DisabledProtectionManager{
		logger: logger.Named("protection-manager-disabled"),
	}
}

// Start 启动防护（禁用状态）
func (dpm *DisabledProtectionManager) Start() error {
	dpm.logger.Info("自我防护功能已禁用（编译时未启用selfprotect标签）")
	return nil
}

// Stop 停止防护（禁用状态）
func (dpm *DisabledProtectionManager) Stop() {
	// 无操作
}

// IsEnabled 检查是否启用（始终返回false）
func (dpm *DisabledProtectionManager) IsEnabled() bool {
	return false
}

// GetStats 获取防护统计（返回空统计）
func (dpm *DisabledProtectionManager) GetStats() ProtectionStats {
	return ProtectionStats{
		StartTime: time.Now(),
	}
}

// GetEvents 获取防护事件（返回空列表）
func (dpm *DisabledProtectionManager) GetEvents() []ProtectionEvent {
	return []ProtectionEvent{}
}

// 禁用的防护器实现

// DisabledProcessProtector 禁用的进程防护器
type DisabledProcessProtector struct {
	logger hclog.Logger
}

// NewProcessProtector 创建禁用的进程防护器
func NewProcessProtector(config ProcessProtectionConfig, logger hclog.Logger) ProcessProtector {
	return &DisabledProcessProtector{
		logger: logger.Named("process-protector-disabled"),
	}
}

func (dpp *DisabledProcessProtector) Start(ctx context.Context) error {
	dpp.logger.Info("进程防护功能已禁用")
	return nil
}

func (dpp *DisabledProcessProtector) Stop() error                                      { return nil }
func (dpp *DisabledProcessProtector) IsEnabled() bool                                  { return false }
func (dpp *DisabledProcessProtector) PeriodicCheck() error                             { return nil }
func (dpp *DisabledProcessProtector) SetEventCallback(callback EventCallback)          {}
func (dpp *DisabledProcessProtector) ProtectProcess(processName string) error          { return nil }
func (dpp *DisabledProcessProtector) UnprotectProcess(processName string) error        { return nil }
func (dpp *DisabledProcessProtector) IsProcessProtected(processName string) bool       { return false }
func (dpp *DisabledProcessProtector) GetProtectedProcesses() []string                  { return []string{} }
func (dpp *DisabledProcessProtector) RestartProcess(processName string) error          { return nil }
func (dpp *DisabledProcessProtector) PreventProcessTermination(processID uint32) error { return nil }
func (dpp *DisabledProcessProtector) PreventProcessDebug(processID uint32) error       { return nil }

// DisabledFileProtector 禁用的文件防护器
type DisabledFileProtector struct {
	logger hclog.Logger
}

// NewFileProtector 创建禁用的文件防护器
func NewFileProtector(config FileProtectionConfig, logger hclog.Logger) FileProtector {
	return &DisabledFileProtector{
		logger: logger.Named("file-protector-disabled"),
	}
}

func (dfp *DisabledFileProtector) Start(ctx context.Context) error {
	dfp.logger.Info("文件防护功能已禁用")
	return nil
}

func (dfp *DisabledFileProtector) Stop() error                                      { return nil }
func (dfp *DisabledFileProtector) IsEnabled() bool                                  { return false }
func (dfp *DisabledFileProtector) PeriodicCheck() error                             { return nil }
func (dfp *DisabledFileProtector) SetEventCallback(callback EventCallback)          {}
func (dfp *DisabledFileProtector) ProtectFile(filePath string) error                { return nil }
func (dfp *DisabledFileProtector) UnprotectFile(filePath string) error              { return nil }
func (dfp *DisabledFileProtector) IsFileProtected(filePath string) bool             { return false }
func (dfp *DisabledFileProtector) GetProtectedFiles() []string                      { return []string{} }
func (dfp *DisabledFileProtector) CheckFileIntegrity(filePath string) (bool, error) { return true, nil }
func (dfp *DisabledFileProtector) BackupFile(filePath string) (string, error)       { return "", nil }
func (dfp *DisabledFileProtector) RestoreFile(filePath string) error                { return nil }
func (dfp *DisabledFileProtector) MonitorFileChanges(filePath string) error         { return nil }

// DisabledRegistryProtector 禁用的注册表防护器
type DisabledRegistryProtector struct {
	logger hclog.Logger
}

// NewRegistryProtector 创建禁用的注册表防护器
func NewRegistryProtector(config RegistryProtectionConfig, logger hclog.Logger) RegistryProtector {
	return &DisabledRegistryProtector{
		logger: logger.Named("registry-protector-disabled"),
	}
}

func (drp *DisabledRegistryProtector) Start(ctx context.Context) error {
	drp.logger.Info("注册表防护功能已禁用")
	return nil
}

func (drp *DisabledRegistryProtector) Stop() error                                 { return nil }
func (drp *DisabledRegistryProtector) IsEnabled() bool                             { return false }
func (drp *DisabledRegistryProtector) PeriodicCheck() error                        { return nil }
func (drp *DisabledRegistryProtector) SetEventCallback(callback EventCallback)     {}
func (drp *DisabledRegistryProtector) ProtectRegistryKey(keyPath string) error     { return nil }
func (drp *DisabledRegistryProtector) UnprotectRegistryKey(keyPath string) error   { return nil }
func (drp *DisabledRegistryProtector) IsRegistryKeyProtected(keyPath string) bool  { return false }
func (drp *DisabledRegistryProtector) GetProtectedRegistryKeys() []string          { return []string{} }
func (drp *DisabledRegistryProtector) BackupRegistryKey(keyPath string) error      { return nil }
func (drp *DisabledRegistryProtector) RestoreRegistryKey(keyPath string) error     { return nil }
func (drp *DisabledRegistryProtector) MonitorRegistryChanges(keyPath string) error { return nil }

// DisabledServiceProtector 禁用的服务防护器
type DisabledServiceProtector struct {
	logger hclog.Logger
}

// NewServiceProtector 创建禁用的服务防护器
func NewServiceProtector(config ServiceProtectionConfig, logger hclog.Logger) ServiceProtector {
	return &DisabledServiceProtector{
		logger: logger.Named("service-protector-disabled"),
	}
}

func (dsp *DisabledServiceProtector) Start(ctx context.Context) error {
	dsp.logger.Info("服务防护功能已禁用")
	return nil
}

func (dsp *DisabledServiceProtector) Stop() error                                    { return nil }
func (dsp *DisabledServiceProtector) IsEnabled() bool                                { return false }
func (dsp *DisabledServiceProtector) PeriodicCheck() error                           { return nil }
func (dsp *DisabledServiceProtector) SetEventCallback(callback EventCallback)        {}
func (dsp *DisabledServiceProtector) ProtectService(serviceName string) error        { return nil }
func (dsp *DisabledServiceProtector) UnprotectService(serviceName string) error      { return nil }
func (dsp *DisabledServiceProtector) IsServiceProtected(serviceName string) bool     { return false }
func (dsp *DisabledServiceProtector) GetProtectedServices() []string                 { return []string{} }
func (dsp *DisabledServiceProtector) RestartService(serviceName string) error        { return nil }
func (dsp *DisabledServiceProtector) PreventServiceStop(serviceName string) error    { return nil }
func (dsp *DisabledServiceProtector) PreventServiceDisable(serviceName string) error { return nil }
func (dsp *DisabledServiceProtector) GetServiceStatus(serviceName string) (ServiceStatus, error) {
	return ServiceStatus{}, nil
}

// 禁用的配置函数

// LoadProtectionConfigFromYAML 从YAML配置中加载自我防护配置（禁用版本）
func LoadProtectionConfigFromYAML(yamlData []byte) (*ProtectionConfig, error) {
	return DefaultProtectionConfig(), nil
}

// ValidateProtectionConfig 验证防护配置（禁用版本）
func ValidateProtectionConfig(config *ProtectionConfig) error {
	return nil
}

// MergeProtectionConfigs 合并防护配置（禁用版本）
func MergeProtectionConfigs(base, override *ProtectionConfig) *ProtectionConfig {
	return DefaultProtectionConfig()
}

// GetProtectionConfigSummary 获取防护配置摘要（禁用版本）
func GetProtectionConfigSummary(config *ProtectionConfig) map[string]interface{} {
	return map[string]interface{}{
		"enabled": false,
		"level":   "none",
	}
}

// DefaultProtectionConfig 默认防护配置（禁用版本）
func DefaultProtectionConfig() *ProtectionConfig {
	return &ProtectionConfig{
		Enabled: false,
		Level:   ProtectionLevelNone,
	}
}
