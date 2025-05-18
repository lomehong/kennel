package config

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestConfigWatcher(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "config-watcher-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.yaml")
	err = ioutil.WriteFile(testFile, []byte("key: value"), 0644)
	assert.NoError(t, err)

	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建配置监视器
	watcher, err := NewConfigWatcher(logger)
	assert.NoError(t, err)
	defer watcher.Stop()

	// 添加监视路径
	err = watcher.AddPath(testFile)
	assert.NoError(t, err)

	// 创建变更事件通道
	eventCh := make(chan ChangeEvent, 1)
	watcher.AddHandler(testFile, func(event ChangeEvent) error {
		eventCh <- event
		return nil
	})

	// 启动监视
	watcher.Start()

	// 修改文件
	time.Sleep(100 * time.Millisecond) // 等待监视器启动
	err = ioutil.WriteFile(testFile, []byte("key: new_value"), 0644)
	assert.NoError(t, err)

	// 等待事件
	select {
	case event := <-eventCh:
		assert.Equal(t, ChangeTypeUpdate, event.Type)
		assert.Equal(t, testFile, event.Path)
	case <-time.After(2 * time.Second):
		t.Fatal("超时等待事件")
	}

	// 移除监视路径
	err = watcher.RemovePath(testFile)
	assert.NoError(t, err)

	// 移除处理器
	watcher.RemoveHandlers(testFile)

	// 获取监视的路径
	paths := watcher.GetWatchedPaths()
	assert.Empty(t, paths)
}

func TestConfigWatcherMultipleHandlers(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "config-watcher-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.yaml")
	err = ioutil.WriteFile(testFile, []byte("key: value"), 0644)
	assert.NoError(t, err)

	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建配置监视器
	watcher, err := NewConfigWatcher(logger)
	assert.NoError(t, err)
	defer watcher.Stop()

	// 添加监视路径
	err = watcher.AddPath(testFile)
	assert.NoError(t, err)

	// 创建变更事件通道
	eventCh1 := make(chan ChangeEvent, 1)
	eventCh2 := make(chan ChangeEvent, 1)

	// 添加多个处理器
	watcher.AddHandler(testFile, func(event ChangeEvent) error {
		eventCh1 <- event
		return nil
	})
	watcher.AddHandler(testFile, func(event ChangeEvent) error {
		eventCh2 <- event
		return nil
	})

	// 启动监视
	watcher.Start()

	// 修改文件
	time.Sleep(100 * time.Millisecond) // 等待监视器启动
	err = ioutil.WriteFile(testFile, []byte("key: new_value"), 0644)
	assert.NoError(t, err)

	// 等待事件
	select {
	case event := <-eventCh1:
		assert.Equal(t, ChangeTypeUpdate, event.Type)
		assert.Equal(t, testFile, event.Path)
	case <-time.After(2 * time.Second):
		t.Fatal("超时等待事件1")
	}

	select {
	case event := <-eventCh2:
		assert.Equal(t, ChangeTypeUpdate, event.Type)
		assert.Equal(t, testFile, event.Path)
	case <-time.After(2 * time.Second):
		t.Fatal("超时等待事件2")
	}
}

func TestConfigWatcherWildcardHandler(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "config-watcher-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile1 := filepath.Join(tempDir, "test1.yaml")
	testFile2 := filepath.Join(tempDir, "test2.yaml")
	err = ioutil.WriteFile(testFile1, []byte("key: value1"), 0644)
	assert.NoError(t, err)
	err = ioutil.WriteFile(testFile2, []byte("key: value2"), 0644)
	assert.NoError(t, err)

	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建配置监视器
	watcher, err := NewConfigWatcher(logger)
	assert.NoError(t, err)
	defer watcher.Stop()

	// 添加监视路径
	err = watcher.AddPath(testFile1)
	assert.NoError(t, err)
	err = watcher.AddPath(testFile2)
	assert.NoError(t, err)

	// 创建变更事件通道
	eventCh := make(chan ChangeEvent, 2)

	// 添加通配符处理器
	watcher.AddHandler("*", func(event ChangeEvent) error {
		eventCh <- event
		return nil
	})

	// 启动监视
	watcher.Start()

	// 修改文件
	time.Sleep(100 * time.Millisecond) // 等待监视器启动
	err = ioutil.WriteFile(testFile1, []byte("key: new_value1"), 0644)
	assert.NoError(t, err)
	err = ioutil.WriteFile(testFile2, []byte("key: new_value2"), 0644)
	assert.NoError(t, err)

	// 等待事件
	receivedEvents := 0
	for i := 0; i < 2; i++ {
		select {
		case event := <-eventCh:
			assert.Equal(t, ChangeTypeUpdate, event.Type)
			assert.Contains(t, []string{testFile1, testFile2}, event.Path)
			receivedEvents++
		case <-time.After(2 * time.Second):
			t.Fatal("超时等待事件")
		}
	}
	assert.Equal(t, 2, receivedEvents)
}

func TestConfigWatcherDebounce(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "config-watcher-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.yaml")
	err = ioutil.WriteFile(testFile, []byte("key: value"), 0644)
	assert.NoError(t, err)

	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建配置监视器
	watcher, err := NewConfigWatcher(logger)
	assert.NoError(t, err)
	defer watcher.Stop()

	// 设置去抖时间
	watcher.SetDebounceTime(200 * time.Millisecond)

	// 添加监视路径
	err = watcher.AddPath(testFile)
	assert.NoError(t, err)

	// 创建变更事件通道和计数器
	var eventCount int
	var mu sync.Mutex
	watcher.AddHandler(testFile, func(event ChangeEvent) error {
		mu.Lock()
		eventCount++
		mu.Unlock()
		return nil
	})

	// 启动监视
	watcher.Start()

	// 多次修改文件
	time.Sleep(100 * time.Millisecond) // 等待监视器启动
	for i := 0; i < 5; i++ {
		err = ioutil.WriteFile(testFile, []byte(fmt.Sprintf("key: value%d", i)), 0644)
		assert.NoError(t, err)
		time.Sleep(50 * time.Millisecond)
	}

	// 等待去抖时间
	time.Sleep(300 * time.Millisecond)

	// 检查事件计数
	mu.Lock()
	count := eventCount
	mu.Unlock()
	assert.Equal(t, 1, count, "应该只收到一个事件（去抖后）")
}
