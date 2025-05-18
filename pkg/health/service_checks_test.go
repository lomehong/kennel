package health

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTPEndpointCheck(t *testing.T) {
	// 创建HTTP服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		} else if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"status":"error"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 创建HTTP端点检查（成功）
	check := NewHTTPEndpointCheck(
		"http_check",
		server.URL+"/health",
		http.MethodGet,
		map[string]string{"User-Agent": "HealthCheck"},
		http.StatusOK,
		"ok",
		5*time.Second,
		30*time.Second,
	)

	// 验证基本属性
	assert.Equal(t, "http_check", check.Name())
	assert.Equal(t, ServiceCheckType, check.Type())
	assert.Equal(t, "http", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.False(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusHealthy, result.Status)
	assert.Contains(t, result.Message, "HTTP端点检查成功")
	assert.Contains(t, result.Details, "endpoint")
	assert.Contains(t, result.Details, "method")
	assert.Contains(t, result.Details, "status_code")

	// 创建HTTP端点检查（失败 - 状态码不匹配）
	check = NewHTTPEndpointCheck(
		"http_check_error",
		server.URL+"/error",
		http.MethodGet,
		nil,
		http.StatusOK,
		"",
		5*time.Second,
		30*time.Second,
	)

	// 执行检查
	result = check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "HTTP状态码不匹配")
	assert.Contains(t, result.Details, "endpoint")
	assert.Contains(t, result.Details, "method")
	assert.Contains(t, result.Details, "status_code")
	assert.Contains(t, result.Details, "expected_status")

	// 创建HTTP端点检查（失败 - 响应体不匹配）
	check = NewHTTPEndpointCheck(
		"http_check_body",
		server.URL+"/health",
		http.MethodGet,
		nil,
		http.StatusOK,
		"not_found",
		5*time.Second,
		30*time.Second,
	)

	// 执行检查
	result = check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "HTTP响应体不包含预期内容")
	assert.Contains(t, result.Details, "endpoint")
	assert.Contains(t, result.Details, "method")
	assert.Contains(t, result.Details, "body")
	assert.Contains(t, result.Details, "expected_body")

	// 创建HTTP端点检查（失败 - 无效端点）
	check = NewHTTPEndpointCheck(
		"http_check_invalid",
		"http://invalid.example.com",
		http.MethodGet,
		nil,
		http.StatusOK,
		"",
		1*time.Second,
		30*time.Second,
	)

	// 执行检查
	result = check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "HTTP请求失败")
	assert.Contains(t, result.Details, "error")
	assert.Contains(t, result.Details, "endpoint")
	assert.Contains(t, result.Details, "method")
}

func TestTCPPortCheck(t *testing.T) {
	// 创建TCP监听器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer listener.Close()

	// 获取端口
	port := listener.Addr().(*net.TCPAddr).Port

	// 创建TCP端口检查（成功）
	check := NewTCPPortCheck(
		"tcp_check",
		"127.0.0.1",
		port,
		5*time.Second,
		30*time.Second,
	)

	// 验证基本属性
	assert.Equal(t, "tcp_check", check.Name())
	assert.Equal(t, ServiceCheckType, check.Type())
	assert.Equal(t, "tcp", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.False(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusHealthy, result.Status)
	assert.Contains(t, result.Message, "TCP端口检查成功")
	assert.Contains(t, result.Details, "host")
	assert.Contains(t, result.Details, "port")
	assert.Contains(t, result.Details, "address")

	// 创建TCP端口检查（失败）
	check = NewTCPPortCheck(
		"tcp_check_error",
		"127.0.0.1",
		12345, // 假设这个端口没有监听
		1*time.Second,
		30*time.Second,
	)

	// 执行检查
	result = check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "TCP连接失败")
	assert.Contains(t, result.Details, "error")
	assert.Contains(t, result.Details, "host")
	assert.Contains(t, result.Details, "port")
	assert.Contains(t, result.Details, "address")
}

func TestFileExistsCheck(t *testing.T) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "health-test-*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// 创建文件存在检查（成功）
	check := NewFileExistsCheck(
		"file_check",
		tempFile.Name(),
		0,
		5*time.Second,
		30*time.Second,
	)

	// 验证基本属性
	assert.Equal(t, "file_check", check.Name())
	assert.Equal(t, ServiceCheckType, check.Type())
	assert.Equal(t, "file", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.False(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusHealthy, result.Status)
	assert.Contains(t, result.Message, "文件存在")
	assert.Contains(t, result.Details, "file_path")
	assert.Contains(t, result.Details, "size")
	assert.Contains(t, result.Details, "mod_time")

	// 创建文件存在检查（失败 - 文件不存在）
	check = NewFileExistsCheck(
		"file_check_error",
		"not_exists.txt",
		0,
		5*time.Second,
		30*time.Second,
	)

	// 执行检查
	result = check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "文件不存在")
	assert.Contains(t, result.Details, "error")
	assert.Contains(t, result.Details, "file_path")

	// 修改文件时间
	oldTime := time.Now().Add(-2 * time.Hour)
	err = os.Chtimes(tempFile.Name(), oldTime, oldTime)
	assert.NoError(t, err)

	// 创建文件存在检查（失败 - 文件过期）
	check = NewFileExistsCheck(
		"file_check_expired",
		tempFile.Name(),
		1*time.Hour,
		5*time.Second,
		30*time.Second,
	)

	// 执行检查
	result = check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "文件过期")
	assert.Contains(t, result.Details, "file_path")
	assert.Contains(t, result.Details, "mod_time")
	assert.Contains(t, result.Details, "age")
	assert.Contains(t, result.Details, "max_age")
}

func TestCommandCheck(t *testing.T) {
	// 创建命令检查（成功）
	var command string
	var args []string
	if isWindows() {
		command = "cmd"
		args = []string{"/c", "echo", "hello"}
	} else {
		command = "echo"
		args = []string{"hello"}
	}

	check := NewCommandCheck(
		"command_check",
		command,
		args,
		"hello",
		5*time.Second,
		30*time.Second,
	)

	// 验证基本属性
	assert.Equal(t, "command_check", check.Name())
	assert.Equal(t, ServiceCheckType, check.Type())
	assert.Equal(t, "command", check.Component())
	assert.Equal(t, 30*time.Second, check.Interval())
	assert.False(t, check.IsRecoverable())

	// 执行检查
	result := check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusHealthy, result.Status)
	assert.Contains(t, result.Message, "命令执行成功")
	assert.Contains(t, result.Details, "command")
	assert.Contains(t, result.Details, "args")
	assert.Contains(t, result.Details, "output")

	// 创建命令检查（失败 - 命令不存在）
	check = NewCommandCheck(
		"command_check_error",
		"not_exists_command",
		nil,
		"",
		5*time.Second,
		30*time.Second,
	)

	// 执行检查
	result = check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "命令执行失败")
	assert.Contains(t, result.Details, "error")
	assert.Contains(t, result.Details, "command")
	assert.Contains(t, result.Details, "args")
	assert.Contains(t, result.Details, "output")

	// 创建命令检查（失败 - 输出不匹配）
	check = NewCommandCheck(
		"command_check_output",
		command,
		args,
		"not_match",
		5*time.Second,
		30*time.Second,
	)

	// 执行检查
	result = check.Check(context.Background())
	assert.NotNil(t, result)
	assert.Equal(t, HealthStatusUnhealthy, result.Status)
	assert.Contains(t, result.Message, "命令输出不包含预期内容")
	assert.Contains(t, result.Details, "command")
	assert.Contains(t, result.Details, "args")
	assert.Contains(t, result.Details, "output")
	assert.Contains(t, result.Details, "expected_output")
}

// isWindows 检查是否是Windows系统
func isWindows() bool {
	return filepath.Separator == '\\' && filepath.ListSeparator == ';'
}
