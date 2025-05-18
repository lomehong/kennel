package plugin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestPluginSandbox_Execute(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()
	isolator := NewPluginIsolator(config, WithLogger(logger))
	sandbox := NewPluginSandbox("test-plugin", isolator, WithSandboxLogger(logger))

	// 测试正常执行
	err := sandbox.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)

	// 测试返回错误
	expectedErr := fmt.Errorf("test error")
	err = sandbox.Execute(func() error {
		return expectedErr
	})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// 测试状态更新
	assert.Equal(t, PluginStateRunning, sandbox.GetState())

	// 测试统计信息
	stats := sandbox.GetStats()
	assert.Equal(t, "test-plugin", stats["plugin_id"])
	assert.Equal(t, "Running", stats["state"])
	assert.Equal(t, int64(2), stats["call_count"])
	assert.Equal(t, int64(1), stats["error_count"])
}

func TestPluginSandbox_ExecuteWithContext(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()
	isolator := NewPluginIsolator(config, WithLogger(logger))
	sandbox := NewPluginSandbox("test-plugin", isolator, WithSandboxLogger(logger))

	// 创建上下文
	ctx := context.Background()

	// 测试正常执行
	err := sandbox.ExecuteWithContext(ctx, func(ctx context.Context) error {
		return nil
	})
	assert.NoError(t, err)

	// 测试返回错误
	expectedErr := fmt.Errorf("test error")
	err = sandbox.ExecuteWithContext(ctx, func(ctx context.Context) error {
		return expectedErr
	})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// 测试上下文取消
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	err = sandbox.ExecuteWithContext(cancelCtx, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestPluginSandbox_StateManagement(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()
	isolator := NewPluginIsolator(config, WithLogger(logger))
	sandbox := NewPluginSandbox("test-plugin", isolator, WithSandboxLogger(logger))

	// 初始状态
	assert.Equal(t, PluginStateInitializing, sandbox.GetState())

	// 设置状态
	sandbox.SetState(PluginStateRunning)
	assert.Equal(t, PluginStateRunning, sandbox.GetState())

	// 暂停
	sandbox.Pause()
	assert.Equal(t, PluginStatePaused, sandbox.GetState())

	// 恢复
	sandbox.Resume()
	assert.Equal(t, PluginStateRunning, sandbox.GetState())

	// 停止
	sandbox.Stop()
	assert.Equal(t, PluginStateStopped, sandbox.GetState())

	// 尝试在停止状态下执行
	err := sandbox.Execute(func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不在运行状态")

	// 重置
	sandbox.Reset()
	assert.Equal(t, PluginStateInitializing, sandbox.GetState())

	// 重置后可以执行
	err = sandbox.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestPluginSandbox_HealthAndIdle(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()
	isolator := NewPluginIsolator(config, WithLogger(logger))
	sandbox := NewPluginSandbox("test-plugin", isolator, WithSandboxLogger(logger))

	// 初始状态应该是健康的
	assert.True(t, sandbox.IsHealthy())

	// 设置为错误状态
	sandbox.SetState(PluginStateError)
	assert.False(t, sandbox.IsHealthy())

	// 重置
	sandbox.Reset()
	assert.True(t, sandbox.IsHealthy())

	// 测试空闲检测
	assert.False(t, sandbox.IsIdle(100*time.Millisecond))

	// 等待一段时间
	time.Sleep(200 * time.Millisecond)
	assert.True(t, sandbox.IsIdle(100*time.Millisecond))

	// 执行操作后不再空闲
	sandbox.Execute(func() error {
		return nil
	})
	assert.False(t, sandbox.IsIdle(100*time.Millisecond))
}

func TestPluginSandbox_GetInfo(t *testing.T) {
	logger := hclog.NewNullLogger()
	config := DefaultPluginIsolationConfig()
	isolator := NewPluginIsolator(config, WithLogger(logger))
	sandbox := NewPluginSandbox("test-plugin", isolator, WithSandboxLogger(logger))

	// 获取ID
	assert.Equal(t, "test-plugin", sandbox.GetID())

	// 获取运行时间
	uptime := sandbox.GetUptime()
	assert.True(t, uptime >= 0)

	// 获取最后活动时间
	lastActivityTime := sandbox.GetLastActivityTime()
	assert.False(t, lastActivityTime.IsZero())

	// 执行操作后最后活动时间应该更新
	time.Sleep(10 * time.Millisecond)
	sandbox.Execute(func() error {
		return nil
	})
	newLastActivityTime := sandbox.GetLastActivityTime()
	assert.True(t, newLastActivityTime.After(lastActivityTime))
}
