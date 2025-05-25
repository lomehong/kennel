//go:build windows

package interceptor

import (
	"github.com/lomehong/kennel/pkg/logging"
)

// createRealInterceptor 创建真实的拦截器实现
func createRealInterceptor(logger logging.Logger) TrafficInterceptor {
	logger.Info("创建Windows WinDivert生产级拦截器")
	return NewWinDivertInterceptor(logger)
}
