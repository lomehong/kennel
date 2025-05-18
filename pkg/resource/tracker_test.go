package resource

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestResourceTracker_Track(t *testing.T) {
	logger := hclog.NewNullLogger()
	tracker := NewResourceTracker(WithLogger(logger))

	// 创建临时文件
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatal(err)
	}

	// 追踪文件资源
	fileResource := NewFileResource(tmpfile)
	tracker.Track(fileResource)

	// 验证资源是否被正确追踪
	resource, exists := tracker.Get(fileResource.ID())
	assert.True(t, exists)
	assert.Equal(t, fileResource.ID(), resource.ID())
	assert.Equal(t, TypeFile, resource.Type())

	// 验证统计信息
	stats := tracker.GetStats()
	assert.Equal(t, int64(1), stats.TotalCreated)
	assert.Equal(t, int64(1), stats.CurrentActive)
	assert.Equal(t, int64(0), stats.TotalClosed)

	// 清理
	tracker.ReleaseAll()
}

func TestResourceTracker_Release(t *testing.T) {
	logger := hclog.NewNullLogger()
	tracker := NewResourceTracker(WithLogger(logger))

	// 创建临时文件
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatal(err)
	}

	// 追踪文件资源
	fileResource := NewFileResource(tmpfile)
	tracker.Track(fileResource)

	// 释放资源
	err = tracker.Release(fileResource.ID())
	assert.NoError(t, err)

	// 验证资源是否被正确释放
	_, exists := tracker.Get(fileResource.ID())
	assert.False(t, exists)

	// 验证统计信息
	stats := tracker.GetStats()
	assert.Equal(t, int64(1), stats.TotalCreated)
	assert.Equal(t, int64(0), stats.CurrentActive)
	assert.Equal(t, int64(1), stats.TotalClosed)

	// 尝试再次释放，应该返回错误
	err = tracker.Release(fileResource.ID())
	assert.Error(t, err)
}

func TestResourceTracker_ReleaseAll(t *testing.T) {
	logger := hclog.NewNullLogger()
	tracker := NewResourceTracker(WithLogger(logger))

	// 创建多个临时文件
	tmpfile1, err := ioutil.TempFile("", "example1")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile2, err := ioutil.TempFile("", "example2")
	if err != nil {
		t.Fatal(err)
	}

	// 追踪文件资源
	fileResource1 := NewFileResource(tmpfile1)
	fileResource2 := NewFileResource(tmpfile2)
	tracker.Track(fileResource1)
	tracker.Track(fileResource2)

	// 释放所有资源
	errors := tracker.ReleaseAll()
	assert.Empty(t, errors)

	// 验证所有资源是否被正确释放
	resources := tracker.ListResources()
	assert.Empty(t, resources)

	// 验证统计信息
	stats := tracker.GetStats()
	assert.Equal(t, int64(2), stats.TotalCreated)
	assert.Equal(t, int64(0), stats.CurrentActive)
	assert.Equal(t, int64(2), stats.TotalClosed)
}

func TestResourceTracker_CleanupIdleResources(t *testing.T) {
	logger := hclog.NewNullLogger()
	tracker := NewResourceTracker(WithLogger(logger))

	// 创建临时文件
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatal(err)
	}

	// 追踪文件资源
	fileResource := NewFileResource(tmpfile)
	tracker.Track(fileResource)

	// 设置最后使用时间为过去
	fileResource.lastUsedAt = time.Now().Add(-time.Hour)

	// 清理空闲资源
	errors := tracker.CleanupIdleResources(30 * time.Minute)
	assert.Empty(t, errors)

	// 验证资源是否被正确清理
	_, exists := tracker.Get(fileResource.ID())
	assert.False(t, exists)

	// 验证统计信息
	stats := tracker.GetStats()
	assert.Equal(t, int64(1), stats.TotalCreated)
	assert.Equal(t, int64(0), stats.CurrentActive)
	assert.Equal(t, int64(1), stats.TotalClosed)
}

func TestContextResourceTracker(t *testing.T) {
	logger := hclog.NewNullLogger()
	tracker := NewResourceTracker(WithLogger(logger))

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	ctxTracker := WithContext(ctx, tracker)

	// 创建临时文件
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatal(err)
	}

	// 追踪文件资源
	fileResource := ctxTracker.TrackFile(tmpfile)
	assert.NotNil(t, fileResource)

	// 验证资源是否被正确追踪
	resource, exists := tracker.Get(fileResource.ID())
	assert.True(t, exists)
	assert.Equal(t, fileResource.ID(), resource.ID())

	// 取消上下文
	cancel()

	// 等待资源释放
	time.Sleep(100 * time.Millisecond)

	// 验证资源是否被正确释放
	_, exists = tracker.Get(fileResource.ID())
	assert.False(t, exists)

	// 验证统计信息
	stats := tracker.GetStats()
	assert.Equal(t, int64(1), stats.TotalCreated)
	assert.Equal(t, int64(0), stats.CurrentActive)
	assert.Equal(t, int64(1), stats.TotalClosed)
}

func TestResourceTracker_ErrorHandling(t *testing.T) {
	logger := hclog.NewNullLogger()
	tracker := NewResourceTracker(WithLogger(logger))

	// 创建一个会在关闭时返回错误的资源
	errCloser := errors.New("close error")
	resource := NewGenericResource("test", "test-resource", func() error {
		return errCloser
	})
	tracker.Track(resource)

	// 释放资源，应该返回错误
	err := tracker.Release(resource.ID())
	assert.Error(t, err)
	assert.Equal(t, errCloser, errors.Unwrap(err))

	// 验证统计信息
	stats := tracker.GetStats()
	assert.Equal(t, int64(1), stats.ClosureErrors)
}

func TestResourceTracker_AutoCleanup(t *testing.T) {
	logger := hclog.NewNullLogger()
	// 设置清理间隔为100毫秒
	tracker := NewResourceTracker(
		WithLogger(logger),
		WithCleanupInterval(100*time.Millisecond),
	)

	// 创建临时文件
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatal(err)
	}

	// 追踪文件资源
	fileResource := NewFileResource(tmpfile)
	tracker.Track(fileResource)

	// 设置最后使用时间为过去
	fileResource.lastUsedAt = time.Now().Add(-time.Hour)

	// 等待自动清理
	time.Sleep(200 * time.Millisecond)

	// 验证资源是否被正确清理
	_, exists := tracker.Get(fileResource.ID())
	assert.False(t, exists)

	// 停止追踪器
	tracker.Stop()
}
