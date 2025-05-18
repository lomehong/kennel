package logging

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogRotator(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "rotator-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建日志文件路径
	logPath := filepath.Join(tempDir, "test.log")

	// 创建日志轮转器
	rotator := NewLogRotator(logPath, 100, 3, 7*24*time.Hour)

	// 写入数据
	data := []byte("这是一条测试日志\n")
	for i := 0; i < 10; i++ {
		n, err := rotator.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
	}

	// 关闭轮转器
	err = rotator.Close()
	assert.NoError(t, err)

	// 验证日志文件存在
	_, err = os.Stat(logPath)
	assert.NoError(t, err)

	// 读取日志文件
	content, err := ioutil.ReadFile(logPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestLogRotatorRotate(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "rotator-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建日志文件路径
	logPath := filepath.Join(tempDir, "test.log")

	// 创建日志轮转器（小文件大小，触发轮转）
	rotator := NewLogRotator(logPath, 10, 3, 7*24*time.Hour)

	// 写入数据（触发轮转）
	data := []byte("这是一条测试日志\n")
	for i := 0; i < 10; i++ {
		n, err := rotator.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
	}

	// 关闭轮转器
	err = rotator.Close()
	assert.NoError(t, err)

	// 验证日志文件存在
	_, err = os.Stat(logPath)
	assert.NoError(t, err)

	// 读取目录
	files, err := ioutil.ReadDir(tempDir)
	assert.NoError(t, err)

	// 验证有多个日志文件（原始文件 + 轮转文件）
	assert.Greater(t, len(files), 1)
}

func TestLogRotatorMaxBackups(t *testing.T) {
	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "rotator-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建日志文件路径
	logPath := filepath.Join(tempDir, "test.log")

	// 创建日志轮转器（小文件大小，触发轮转，最多保留2个备份）
	rotator := NewLogRotator(logPath, 10, 2, 7*24*time.Hour)

	// 写入数据（触发多次轮转）
	data := []byte("这是一条测试日志\n")
	for i := 0; i < 50; i++ {
		n, err := rotator.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, len(data), n)
	}

	// 关闭轮转器
	err = rotator.Close()
	assert.NoError(t, err)

	// 读取目录
	files, err := ioutil.ReadDir(tempDir)
	assert.NoError(t, err)

	// 验证文件数量（原始文件 + 最多2个备份）
	assert.LessOrEqual(t, len(files), 3)
}

func TestMultiWriter(t *testing.T) {
	// 创建缓冲区
	var buf1, buf2 []byte

	// 创建写入器
	writer1 := &testWriter{&buf1}
	writer2 := &testWriter{&buf2}

	// 创建多输出写入器
	multiWriter := NewMultiWriter(writer1, writer2)

	// 写入数据
	data := []byte("这是一条测试日志\n")
	n, err := multiWriter.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)

	// 验证数据
	assert.Equal(t, data, buf1)
	assert.Equal(t, data, buf2)
}

// testWriter 测试写入器
type testWriter struct {
	buf *[]byte
}

// Write 实现io.Writer接口
func (w *testWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}
