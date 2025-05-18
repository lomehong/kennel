package health

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ServiceCheckType 服务检查类型
const ServiceCheckType = "service"

// 服务健康检查名称
const (
	HTTPEndpointCheckName = "http_endpoint"
	TCPPortCheckName      = "tcp_port"
	ProcessCheckName      = "process"
	DatabaseCheckName     = "database"
	FileExistsCheckName   = "file_exists"
	CommandCheckName      = "command"
)

// NewHTTPEndpointCheck 创建HTTP端点检查
func NewHTTPEndpointCheck(
	name string,
	endpoint string,
	method string,
	headers map[string]string,
	expectedStatus int,
	expectedBody string,
	timeout time.Duration,
	interval time.Duration,
) HealthCheck {
	if method == "" {
		method = http.MethodGet
	}
	if expectedStatus == 0 {
		expectedStatus = http.StatusOK
	}

	return &BaseHealthCheck{
		name:             name,
		checkType:        ServiceCheckType,
		component:        "http",
		timeout:          timeout,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 创建请求
			req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("创建HTTP请求失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error":    err.Error(),
						"endpoint": endpoint,
						"method":   method,
					},
				}
			}

			// 添加请求头
			for key, value := range headers {
				req.Header.Set(key, value)
			}

			// 发送请求
			client := &http.Client{
				Timeout: timeout,
			}
			resp, err := client.Do(req)
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("HTTP请求失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error":    err.Error(),
						"endpoint": endpoint,
						"method":   method,
					},
				}
			}
			defer resp.Body.Close()

			// 检查状态码
			if resp.StatusCode != expectedStatus {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("HTTP状态码不匹配: %d != %d", resp.StatusCode, expectedStatus),
					Details: map[string]interface{}{
						"endpoint":        endpoint,
						"method":          method,
						"status_code":     resp.StatusCode,
						"expected_status": expectedStatus,
					},
				}
			}

			// 检查响应体
			if expectedBody != "" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return &HealthCheckResult{
						Status:  HealthStatusUnknown,
						Message: fmt.Sprintf("读取HTTP响应体失败: %s", err.Error()),
						Error:   err,
						Details: map[string]interface{}{
							"error":    err.Error(),
							"endpoint": endpoint,
							"method":   method,
						},
					}
				}

				bodyStr := string(body)
				if !strings.Contains(bodyStr, expectedBody) {
					return &HealthCheckResult{
						Status:  HealthStatusUnhealthy,
						Message: fmt.Sprintf("HTTP响应体不包含预期内容"),
						Details: map[string]interface{}{
							"endpoint":      endpoint,
							"method":        method,
							"body":          bodyStr,
							"expected_body": expectedBody,
						},
					}
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("HTTP端点检查成功: %s", endpoint),
				Details: map[string]interface{}{
					"endpoint":    endpoint,
					"method":      method,
					"status_code": resp.StatusCode,
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewTCPPortCheck 创建TCP端口检查
func NewTCPPortCheck(
	name string,
	host string,
	port int,
	timeout time.Duration,
	interval time.Duration,
) HealthCheck {
	return &BaseHealthCheck{
		name:             name,
		checkType:        ServiceCheckType,
		component:        "tcp",
		timeout:          timeout,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 创建地址
			address := fmt.Sprintf("%s:%d", host, port)

			// 创建连接
			var d net.Dialer
			conn, err := d.DialContext(ctx, "tcp", address)
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("TCP连接失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error":   err.Error(),
						"host":    host,
						"port":    port,
						"address": address,
					},
				}
			}
			defer conn.Close()

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("TCP端口检查成功: %s", address),
				Details: map[string]interface{}{
					"host":    host,
					"port":    port,
					"address": address,
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewProcessCheck 创建进程检查
func NewProcessCheck(
	name string,
	processName string,
	timeout time.Duration,
	interval time.Duration,
) HealthCheck {
	return &BaseHealthCheck{
		name:             name,
		checkType:        ServiceCheckType,
		component:        "process",
		timeout:          timeout,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 创建命令
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.CommandContext(ctx, "tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName))
			} else {
				cmd = exec.CommandContext(ctx, "pgrep", processName)
			}

			// 执行命令
			output, err := cmd.CombinedOutput()
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("进程 %s 未运行", processName),
					Error:   err,
					Details: map[string]interface{}{
						"error":        err.Error(),
						"process_name": processName,
						"output":       string(output),
					},
				}
			}

			// 检查输出
			outputStr := string(output)
			if runtime.GOOS == "windows" {
				if !strings.Contains(outputStr, processName) {
					return &HealthCheckResult{
						Status:  HealthStatusUnhealthy,
						Message: fmt.Sprintf("进程 %s 未运行", processName),
						Details: map[string]interface{}{
							"process_name": processName,
							"output":       outputStr,
						},
					}
				}
			} else {
				if outputStr == "" {
					return &HealthCheckResult{
						Status:  HealthStatusUnhealthy,
						Message: fmt.Sprintf("进程 %s 未运行", processName),
						Details: map[string]interface{}{
							"process_name": processName,
							"output":       outputStr,
						},
					}
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("进程 %s 正在运行", processName),
				Details: map[string]interface{}{
					"process_name": processName,
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewDatabaseCheck 创建数据库检查
func NewDatabaseCheck(
	name string,
	driverName string,
	dsn string,
	query string,
	timeout time.Duration,
	interval time.Duration,
) HealthCheck {
	return &BaseHealthCheck{
		name:             name,
		checkType:        ServiceCheckType,
		component:        "database",
		timeout:          timeout,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 解析DSN
			parsedDSN := dsn
			if u, err := url.Parse(dsn); err == nil {
				// 隐藏密码
				if u.User != nil {
					password, _ := u.User.Password()
					if password != "" {
						username := u.User.Username()
						u.User = url.UserPassword(username, "******")
					}
				}
				parsedDSN = u.String()
			}

			// 尝试连接数据库并执行查询
			var db *sql.DB
			var err error

			// 根据驱动类型连接数据库
			switch driverName {
			case "mysql":
				db, err = sql.Open("mysql", dsn)
			case "postgres", "postgresql":
				db, err = sql.Open("postgres", dsn)
			case "sqlite", "sqlite3":
				db, err = sql.Open("sqlite3", dsn)
			case "sqlserver", "mssql":
				db, err = sql.Open("sqlserver", dsn)
			default:
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("不支持的数据库驱动: %s", driverName),
					Details: map[string]interface{}{
						"driver": driverName,
						"dsn":    parsedDSN,
					},
				}
			}

			// 检查连接错误
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("数据库连接失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"driver": driverName,
						"dsn":    parsedDSN,
						"error":  err.Error(),
					},
				}
			}
			defer db.Close()

			// 设置连接超时
			db.SetConnMaxLifetime(timeout)

			// 执行查询
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			var result interface{}
			err = db.QueryRowContext(ctx, query).Scan(&result)
			if err != nil && err != sql.ErrNoRows {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("数据库查询失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"driver": driverName,
						"dsn":    parsedDSN,
						"query":  query,
						"error":  err.Error(),
					},
				}
			}

			// 返回健康状态
			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("数据库连接成功: %s", driverName),
				Details: map[string]interface{}{
					"driver": driverName,
					"dsn":    parsedDSN,
					"query":  query,
					"result": result,
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewFileExistsCheck 创建文件存在检查
func NewFileExistsCheck(
	name string,
	filePath string,
	maxAge time.Duration,
	timeout time.Duration,
	interval time.Duration,
) HealthCheck {
	return &BaseHealthCheck{
		name:             name,
		checkType:        ServiceCheckType,
		component:        "file",
		timeout:          timeout,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 检查文件是否存在
			info, err := os.Stat(filePath)
			if os.IsNotExist(err) {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("文件不存在: %s", filePath),
					Error:   err,
					Details: map[string]interface{}{
						"error":     err.Error(),
						"file_path": filePath,
					},
				}
			}
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnknown,
					Message: fmt.Sprintf("检查文件失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error":     err.Error(),
						"file_path": filePath,
					},
				}
			}

			// 检查文件是否过期
			if maxAge > 0 {
				modTime := info.ModTime()
				age := time.Since(modTime)
				if age > maxAge {
					return &HealthCheckResult{
						Status:  HealthStatusUnhealthy,
						Message: fmt.Sprintf("文件过期: %s (已过期 %s)", filePath, age),
						Details: map[string]interface{}{
							"file_path": filePath,
							"mod_time":  modTime,
							"age":       age.String(),
							"max_age":   maxAge.String(),
						},
					}
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("文件存在: %s", filePath),
				Details: map[string]interface{}{
					"file_path": filePath,
					"size":      info.Size(),
					"mod_time":  info.ModTime(),
				},
			}
		},
		recoverFunc: nil,
	}
}

// NewCommandCheck 创建命令检查
func NewCommandCheck(
	name string,
	command string,
	args []string,
	expectedOutput string,
	timeout time.Duration,
	interval time.Duration,
) HealthCheck {
	return &BaseHealthCheck{
		name:             name,
		checkType:        ServiceCheckType,
		component:        "command",
		timeout:          timeout,
		interval:         interval,
		failureThreshold: 3,
		successThreshold: 1,
		recoverable:      false,
		checkFunc: func(ctx context.Context) *HealthCheckResult {
			// 创建命令
			cmd := exec.CommandContext(ctx, command, args...)

			// 执行命令
			output, err := cmd.CombinedOutput()
			if err != nil {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("命令执行失败: %s", err.Error()),
					Error:   err,
					Details: map[string]interface{}{
						"error":   err.Error(),
						"command": command,
						"args":    args,
						"output":  string(output),
					},
				}
			}

			// 检查输出
			outputStr := string(output)
			if expectedOutput != "" && !strings.Contains(outputStr, expectedOutput) {
				return &HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: fmt.Sprintf("命令输出不包含预期内容"),
					Details: map[string]interface{}{
						"command":         command,
						"args":            args,
						"output":          outputStr,
						"expected_output": expectedOutput,
					},
				}
			}

			return &HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: fmt.Sprintf("命令执行成功: %s", command),
				Details: map[string]interface{}{
					"command": command,
					"args":    args,
					"output":  outputStr,
				},
			}
		},
		recoverFunc: nil,
	}
}
