//go:build selfprotect && !windows
// +build selfprotect,!windows

package selfprotect

import (
	"context"

	"github.com/hashicorp/go-hclog"
)

// NewProcessProtector 创建进程防护器（非Windows平台）
func NewProcessProtector(config ProcessProtectionConfig, logger hclog.Logger) ProcessProtector {
	return &EmptyProcessProtector{
		logger: logger.Named("process-protector"),
	}
}

// EmptyProcessProtector 空的进程防护器实现
type EmptyProcessProtector struct {
	logger hclog.Logger
}

func (epp *EmptyProcessProtector) Start(ctx context.Context) error {
	epp.logger.Info("进程防护在此平台上不可用")
	return nil
}

func (epp *EmptyProcessProtector) Stop() error                                      { return nil }
func (epp *EmptyProcessProtector) IsEnabled() bool                                  { return false }
func (epp *EmptyProcessProtector) PeriodicCheck() error                             { return nil }
func (epp *EmptyProcessProtector) SetEventCallback(callback EventCallback)          {}
func (epp *EmptyProcessProtector) ProtectProcess(processName string) error          { return nil }
func (epp *EmptyProcessProtector) UnprotectProcess(processName string) error        { return nil }
func (epp *EmptyProcessProtector) IsProcessProtected(processName string) bool       { return false }
func (epp *EmptyProcessProtector) GetProtectedProcesses() []string                  { return []string{} }
func (epp *EmptyProcessProtector) RestartProcess(processName string) error          { return nil }
func (epp *EmptyProcessProtector) PreventProcessTermination(processID uint32) error { return nil }
func (epp *EmptyProcessProtector) PreventProcessDebug(processID uint32) error       { return nil }
