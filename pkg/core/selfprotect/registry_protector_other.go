//go:build selfprotect && !windows
// +build selfprotect,!windows

package selfprotect

import (
	"context"

	"github.com/hashicorp/go-hclog"
)

// NewRegistryProtector 创建注册表防护器（非Windows平台）
func NewRegistryProtector(config RegistryProtectionConfig, logger hclog.Logger) RegistryProtector {
	return &EmptyRegistryProtector{
		logger: logger.Named("registry-protector"),
	}
}

// EmptyRegistryProtector 空的注册表防护器实现
type EmptyRegistryProtector struct {
	logger hclog.Logger
}

func (erp *EmptyRegistryProtector) Start(ctx context.Context) error {
	erp.logger.Info("注册表防护在此平台上不可用")
	return nil
}

func (erp *EmptyRegistryProtector) Stop() error                                        { return nil }
func (erp *EmptyRegistryProtector) IsEnabled() bool                                    { return false }
func (erp *EmptyRegistryProtector) PeriodicCheck() error                               { return nil }
func (erp *EmptyRegistryProtector) SetEventCallback(callback EventCallback)            {}
func (erp *EmptyRegistryProtector) ProtectRegistryKey(keyPath string) error            { return nil }
func (erp *EmptyRegistryProtector) UnprotectRegistryKey(keyPath string) error          { return nil }
func (erp *EmptyRegistryProtector) IsRegistryKeyProtected(keyPath string) bool         { return false }
func (erp *EmptyRegistryProtector) GetProtectedRegistryKeys() []string                 { return []string{} }
func (erp *EmptyRegistryProtector) BackupRegistryKey(keyPath string) error             { return nil }
func (erp *EmptyRegistryProtector) RestoreRegistryKey(keyPath string) error            { return nil }
func (erp *EmptyRegistryProtector) MonitorRegistryChanges(keyPath string) error        { return nil }
