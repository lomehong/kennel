//go:build !windows && !linux && !darwin

package interceptor

import (
	"github.com/lomehong/kennel/pkg/logging"
)

// createPlatformInterceptor 创建平台特定的拦截器（默认实现）
func createPlatformInterceptor(logger logging.Logger) PlatformInterceptor {
	logger.Warn("不支持的平台，使用模拟实现")
	return NewMockPlatformInterceptor(logger)
}

// createRealInterceptor 创建真实的拦截器实现（默认实现）
func createRealInterceptor(logger logging.Logger) TrafficInterceptor {
	logger.Error("不支持的平台，无法创建生产级拦截器")
	panic("不支持的平台，无法创建生产级拦截器")
}
