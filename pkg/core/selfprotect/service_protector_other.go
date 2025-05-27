//go:build selfprotect && !windows
// +build selfprotect,!windows

package selfprotect

import (
	"context"

	"github.com/hashicorp/go-hclog"
)

// NewServiceProtector 创建服务防护器（非Windows平台）
func NewServiceProtector(config ServiceProtectionConfig, logger hclog.Logger) ServiceProtector {
	return &EmptyServiceProtector{
		logger: logger.Named("service-protector"),
	}
}

// EmptyServiceProtector 空的服务防护器实现
type EmptyServiceProtector struct {
	logger hclog.Logger
}

func (esp *EmptyServiceProtector) Start(ctx context.Context) error {
	esp.logger.Info("服务防护在此平台上不可用")
	return nil
}

func (esp *EmptyServiceProtector) Stop() error                                         { return nil }
func (esp *EmptyServiceProtector) IsEnabled() bool                                     { return false }
func (esp *EmptyServiceProtector) PeriodicCheck() error                                { return nil }
func (esp *EmptyServiceProtector) SetEventCallback(callback EventCallback)             {}
func (esp *EmptyServiceProtector) ProtectService(serviceName string) error             { return nil }
func (esp *EmptyServiceProtector) UnprotectService(serviceName string) error           { return nil }
func (esp *EmptyServiceProtector) IsServiceProtected(serviceName string) bool          { return false }
func (esp *EmptyServiceProtector) GetProtectedServices() []string                      { return []string{} }
func (esp *EmptyServiceProtector) RestartService(serviceName string) error             { return nil }
func (esp *EmptyServiceProtector) PreventServiceStop(serviceName string) error         { return nil }
func (esp *EmptyServiceProtector) PreventServiceDisable(serviceName string) error      { return nil }
func (esp *EmptyServiceProtector) GetServiceStatus(serviceName string) (ServiceStatus, error) {
	return ServiceStatus{}, nil
}
