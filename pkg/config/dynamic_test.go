package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestDynamicConfig(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "dynamic-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.yaml")
	err = ioutil.WriteFile(testFile, []byte("key: value\nnum: 42\nbool: true"), 0644)
	assert.NoError(t, err)

	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建动态配置
	config, err := NewDynamicConfig(testFile, logger)
	assert.NoError(t, err)

	// 验证配置
	assert.Equal(t, "value", config.Get("key"))
	assert.Equal(t, 42, config.Get("num"))
	assert.Equal(t, true, config.Get("bool"))

	// 修改配置
	config.Set("key", "new_value")
	assert.Equal(t, "new_value", config.Get("key"))

	// 保存配置
	err = config.Save()
	assert.NoError(t, err)

	// 重新加载配置
	err = config.Reload()
	assert.NoError(t, err)
	assert.Equal(t, "new_value", config.Get("key"))

	// 获取所有配置
	allConfig := config.GetAll()
	assert.Equal(t, "new_value", allConfig["key"])
	assert.Equal(t, 42, allConfig["num"])
	assert.Equal(t, true, allConfig["bool"])

	// 更新配置
	config.Update(map[string]interface{}{
		"key":  "updated_value",
		"num":  100,
		"bool": false,
	})
	assert.Equal(t, "updated_value", config.Get("key"))
	assert.Equal(t, 100, config.Get("num"))
	assert.Equal(t, false, config.Get("bool"))

	// 删除配置
	config.Delete("bool")
	assert.Nil(t, config.Get("bool"))

	// 获取版本
	version := config.GetVersion()
	assert.Equal(t, 1, version.Version)

	// 获取所有版本
	versions := config.GetVersions()
	assert.Len(t, versions, 1)
}

func TestDynamicConfigWithValidator(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "dynamic-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.yaml")
	err = ioutil.WriteFile(testFile, []byte("key: value\nnum: 42\nbool: true"), 0644)
	assert.NoError(t, err)

	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建验证器
	validator := func(config map[string]interface{}) error {
		if _, ok := config["key"]; !ok {
			return fmt.Errorf("缺少必需字段: key")
		}
		if num, ok := config["num"].(int); ok && num < 0 {
			return fmt.Errorf("num 应该大于等于 0")
		}
		return nil
	}

	// 创建动态配置
	config, err := NewDynamicConfig(testFile, logger, WithValidator(validator))
	assert.NoError(t, err)

	// 验证配置
	assert.Equal(t, "value", config.Get("key"))
	assert.Equal(t, 42, config.Get("num"))

	// 修改配置（有效）
	config.Set("num", 100)
	err = config.Save()
	assert.NoError(t, err)

	// 修改配置（无效）
	config.Set("num", -1)
	err = config.Save()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "num 应该大于等于 0")

	// 删除必需字段
	config.Delete("key")
	err = config.Save()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "缺少必需字段: key")
}

func TestDynamicConfigWithListener(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "dynamic-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.yaml")
	err = ioutil.WriteFile(testFile, []byte("key: value\nnum: 42\nbool: true"), 0644)
	assert.NoError(t, err)

	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建监听器
	listenerCalled := false
	listener := func(oldConfig, newConfig map[string]interface{}) error {
		listenerCalled = true
		assert.Equal(t, "value", oldConfig["key"])
		assert.Equal(t, "new_value", newConfig["key"])
		return nil
	}

	// 创建动态配置
	config, err := NewDynamicConfig(testFile, logger, WithListener(listener))
	assert.NoError(t, err)

	// 修改配置
	config.Set("key", "new_value")
	err = config.Save()
	assert.NoError(t, err)

	// 重新加载配置
	err = config.Reload()
	assert.NoError(t, err)
	assert.True(t, listenerCalled)
}

func TestDynamicConfigWithWatcher(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "dynamic-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.yaml")
	err = ioutil.WriteFile(testFile, []byte("key: value\nnum: 42\nbool: true"), 0644)
	assert.NoError(t, err)

	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建配置监视器
	watcher, err := NewConfigWatcher(logger)
	assert.NoError(t, err)
	defer watcher.Stop()

	// 创建监听器
	listenerCalled := false
	listener := func(oldConfig, newConfig map[string]interface{}) error {
		listenerCalled = true
		return nil
	}

	// 创建动态配置
	config, err := NewDynamicConfig(testFile, logger,
		WithWatcher(watcher),
		WithListener(listener),
	)
	assert.NoError(t, err)

	// 启动监视
	watcher.Start()

	// 修改文件
	time.Sleep(100 * time.Millisecond) // 等待监视器启动
	err = ioutil.WriteFile(testFile, []byte("key: new_value\nnum: 100\nbool: false"), 0644)
	assert.NoError(t, err)

	// 等待配置重新加载
	time.Sleep(500 * time.Millisecond)
	assert.True(t, listenerCalled)
	assert.Equal(t, "new_value", config.Get("key"))
	assert.Equal(t, 100, config.Get("num"))
	assert.Equal(t, false, config.Get("bool"))
}

func TestDynamicConfigRollback(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "dynamic-config-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建测试文件
	testFile := filepath.Join(tempDir, "test.yaml")
	err = ioutil.WriteFile(testFile, []byte("key: value\nnum: 42\nbool: true"), 0644)
	assert.NoError(t, err)

	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建动态配置
	config, err := NewDynamicConfig(testFile, logger, WithMaxHistorySize(5))
	assert.NoError(t, err)

	// 修改配置多次
	for i := 1; i <= 3; i++ {
		config.Set("key", fmt.Sprintf("value%d", i))
		config.Set("num", 42+i)
		err = config.Save()
		assert.NoError(t, err)
		err = config.Reload()
		assert.NoError(t, err)
	}

	// 获取版本
	versions := config.GetVersions()
	assert.Len(t, versions, 4) // 初始版本 + 3个修改版本

	// 回滚到第2个版本
	err = config.Rollback(2)
	assert.NoError(t, err)
	assert.Equal(t, "value1", config.Get("key"))
	assert.Equal(t, 43, config.Get("num"))

	// 获取版本
	versions = config.GetVersions()
	assert.Len(t, versions, 5) // 初始版本 + 3个修改版本 + 1个回滚版本
	assert.Equal(t, 5, versions[4].Version)
}
