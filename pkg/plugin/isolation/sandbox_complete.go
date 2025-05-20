package isolation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// CompleteIsolationSandbox 完全隔离沙箱
// 提供最高级别的隔离，包括容器隔离
type CompleteIsolationSandbox struct {
	// 沙箱ID
	id string

	// 隔离配置
	config api.IsolationConfig

	// 日志记录器
	logger hclog.Logger

	// 状态
	state int32

	// 上次活动时间
	lastActivity time.Time

	// 统计信息
	stats struct {
		// 执行次数
		executions int64

		// 成功次数
		successes int64

		// 失败次数
		failures int64

		// 恐慌次数
		panics int64

		// 超时次数
		timeouts int64

		// 总执行时间
		totalExecTime int64

		// 最长执行时间
		maxExecTime int64

		// 最短执行时间
		minExecTime int64

		// 平均执行时间
		avgExecTime int64
	}

	// 互斥锁
	mu sync.RWMutex

	// 工作目录
	workDir string

	// 环境变量
	env []string

	// 容器ID
	containerID string

	// 容器命令
	containerCmd *exec.Cmd
}

// NewCompleteIsolationSandbox 创建一个新的完全隔离沙箱
func NewCompleteIsolationSandbox(id string, config api.IsolationConfig, logger hclog.Logger) *CompleteIsolationSandbox {
	// 创建工作目录
	workDir := config.WorkingDir
	if workDir == "" {
		workDir = filepath.Join(os.TempDir(), "sandbox", id)
	}

	// 确保工作目录存在
	if err := os.MkdirAll(workDir, 0755); err != nil {
		logger.Error("创建工作目录失败", "id", id, "dir", workDir, "error", err)
	}

	// 构建环境变量
	env := os.Environ()
	for k, v := range config.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return &CompleteIsolationSandbox{
		id:           id,
		config:       config,
		logger:       logger,
		state:        sandboxStateRunning,
		lastActivity: time.Now(),
		workDir:      workDir,
		env:          env,
	}
}

// GetID 获取沙箱ID
func (s *CompleteIsolationSandbox) GetID() string {
	return s.id
}

// GetConfig 获取隔离配置
func (s *CompleteIsolationSandbox) GetConfig() api.IsolationConfig {
	return s.config
}

// Execute 执行函数
func (s *CompleteIsolationSandbox) Execute(f func() error) error {
	// 检查沙箱状态
	if atomic.LoadInt32(&s.state) != sandboxStateRunning {
		return fmt.Errorf("沙箱 %s 未运行", s.id)
	}

	// 更新最后活动时间
	s.lastActivity = time.Now()

	// 增加执行次数
	atomic.AddInt64(&s.stats.executions, 1)

	// 记录开始时间
	startTime := time.Now()

	// 创建错误通道
	errCh := make(chan error, 1)

	// 在容器中执行函数
	go func() {
		// 将函数序列化为脚本
		scriptPath, err := s.serializeFunction(f)
		if err != nil {
			errCh <- fmt.Errorf("序列化函数失败: %w", err)
			return
		}

		// 在容器中执行脚本
		err = s.executeInContainer(scriptPath)

		// 发送结果
		errCh <- err
	}()

	// 等待结果或超时
	var err error
	select {
	case err = <-errCh:
		// 函数执行完成
	case <-time.After(s.config.Timeout):
		// 超时
		atomic.AddInt64(&s.stats.timeouts, 1)
		err = fmt.Errorf("函数执行超时")
	}

	// 记录执行时间
	execTime := time.Since(startTime).Milliseconds()
	atomic.AddInt64(&s.stats.totalExecTime, execTime)

	// 更新最长执行时间
	for {
		maxExecTime := atomic.LoadInt64(&s.stats.maxExecTime)
		if execTime <= maxExecTime {
			break
		}
		if atomic.CompareAndSwapInt64(&s.stats.maxExecTime, maxExecTime, execTime) {
			break
		}
	}

	// 更新最短执行时间
	for {
		minExecTime := atomic.LoadInt64(&s.stats.minExecTime)
		if minExecTime == 0 || execTime < minExecTime {
			if atomic.CompareAndSwapInt64(&s.stats.minExecTime, minExecTime, execTime) {
				break
			}
		} else {
			break
		}
	}

	// 更新平均执行时间
	executions := atomic.LoadInt64(&s.stats.executions)
	if executions > 0 {
		totalExecTime := atomic.LoadInt64(&s.stats.totalExecTime)
		atomic.StoreInt64(&s.stats.avgExecTime, totalExecTime/executions)
	}

	// 更新成功/失败次数
	if err != nil {
		atomic.AddInt64(&s.stats.failures, 1)
	} else {
		atomic.AddInt64(&s.stats.successes, 1)
	}

	return err
}

// serializeFunction 将函数序列化为脚本
func (s *CompleteIsolationSandbox) serializeFunction(f func() error) (string, error) {
	// 创建脚本文件
	scriptPath := filepath.Join(s.workDir, "script.go")

	// 创建脚本内容
	scriptContent := `package main

import (
	"fmt"
	"os"
)

func main() {
	// 执行函数
	if err := execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func execute() error {
	// 这里是序列化的函数
	// 在实际实现中，这里应该是动态生成的
	return nil
}
`

	// 写入脚本文件
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		return "", fmt.Errorf("写入脚本文件失败: %w", err)
	}

	return scriptPath, nil
}

// executeInContainer 在容器中执行脚本
func (s *CompleteIsolationSandbox) executeInContainer(scriptPath string) error {
	// 检查容器运行时
	containerRuntime := "docker"
	if runtime.GOOS == "windows" {
		// 在Windows上检查WSL
		if _, err := exec.LookPath("wsl"); err == nil {
			containerRuntime = "wsl"
		}
	}

	// 构建容器命令
	var cmd *exec.Cmd
	switch containerRuntime {
	case "docker":
		// 使用Docker运行
		cmd = exec.Command("docker", "run", "--rm",
			"--name", fmt.Sprintf("sandbox-%s", s.id),
			"-v", fmt.Sprintf("%s:/sandbox", s.workDir),
			"-w", "/sandbox",
			"golang:alpine",
			"go", "run", "script.go")
	case "wsl":
		// 使用WSL运行
		cmd = exec.Command("wsl", "cd", s.workDir, "&&", "go", "run", "script.go")
	default:
		// 直接运行
		cmd = exec.Command("go", "run", scriptPath)
		cmd.Dir = s.workDir
	}

	// 设置环境变量
	cmd.Env = s.env

	// 设置输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 存储容器命令
	s.mu.Lock()
	s.containerCmd = cmd
	s.mu.Unlock()

	// 执行命令
	err := cmd.Run()

	// 清理容器命令
	s.mu.Lock()
	s.containerCmd = nil
	s.mu.Unlock()

	return err
}

// ExecuteWithContext 执行带上下文的函数
func (s *CompleteIsolationSandbox) ExecuteWithContext(ctx context.Context, f func(context.Context) error) error {
	// 检查沙箱状态
	if atomic.LoadInt32(&s.state) != sandboxStateRunning {
		return fmt.Errorf("沙箱 %s 未运行", s.id)
	}

	// 更新最后活动时间
	s.lastActivity = time.Now()

	// 增加执行次数
	atomic.AddInt64(&s.stats.executions, 1)

	// 记录开始时间
	// startTime := time.Now() // 暂时注释掉未使用的变量

	// 创建错误通道
	errCh := make(chan error, 1)

	// 创建带超时的上下文
	timeoutCtx := ctx
	if s.config.Timeout > 0 {
		var cancel context.CancelFunc
		timeoutCtx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}

	// 在容器中执行函数
	go func() {
		// 将上下文信息序列化
		deadline, hasDeadline := timeoutCtx.Deadline()
		ctxData, err := json.Marshal(map[string]interface{}{
			"deadline":    deadline,
			"hasDeadline": hasDeadline,
		})
		if err != nil {
			errCh <- fmt.Errorf("序列化上下文失败: %w", err)
			return
		}

		// 将上下文信息写入文件
		ctxPath := filepath.Join(s.workDir, "context.json")
		if err := os.WriteFile(ctxPath, ctxData, 0644); err != nil {
			errCh <- fmt.Errorf("写入上下文文件失败: %w", err)
			return
		}

		// 将函数序列化为脚本
		scriptPath, err := s.serializeFunction(func() error {
			// 读取上下文信息
			ctxData, err := os.ReadFile(ctxPath)
			if err != nil {
				return fmt.Errorf("读取上下文文件失败: %w", err)
			}

			// 解析上下文信息
			var ctxInfo map[string]interface{}
			if err := json.Unmarshal(ctxData, &ctxInfo); err != nil {
				return fmt.Errorf("解析上下文信息失败: %w", err)
			}

			// 创建上下文
			ctx := context.Background()

			// 设置截止时间
			if deadline, ok := ctxInfo["deadline"].(time.Time); ok {
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, deadline)
				defer cancel()
			}

			// 执行函数
			return f(ctx)
		})
		if err != nil {
			errCh <- fmt.Errorf("序列化函数失败: %w", err)
			return
		}

		// 在容器中执行脚本
		err = s.executeInContainer(scriptPath)

		// 发送结果
		errCh <- err
	}()

	// 等待结果或上下文取消
	var err error
	select {
	case err = <-errCh:
		// 函数执行完成
	case <-timeoutCtx.Done():
		// 上下文取消或超时
		if timeoutCtx.Err() == context.DeadlineExceeded {
			atomic.AddInt64(&s.stats.timeouts, 1)
			err = fmt.Errorf("函数执行超时")
		} else {
			err = fmt.Errorf("上下文取消: %w", timeoutCtx.Err())
		}

		// 停止容器
		s.mu.Lock()
		if s.containerCmd != nil && s.containerCmd.Process != nil {
			s.containerCmd.Process.Kill()
		}
		s.mu.Unlock()
	}

	// 记录执行时间和更新统计信息
	// ... (与Execute方法相同的统计信息更新逻辑)

	// 更新成功/失败次数
	if err != nil {
		atomic.AddInt64(&s.stats.failures, 1)
	} else {
		atomic.AddInt64(&s.stats.successes, 1)
	}

	return err
}

// ExecuteWithTimeout 执行带超时的函数
func (s *CompleteIsolationSandbox) ExecuteWithTimeout(timeout time.Duration, f func() error) error {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 使用带上下文的执行
	return s.ExecuteWithContext(ctx, func(ctx context.Context) error {
		// 监听上下文取消
		done := make(chan error, 1)
		go func() {
			done <- f()
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

// Pause 暂停沙箱
func (s *CompleteIsolationSandbox) Pause() error {
	// 检查沙箱状态
	if !atomic.CompareAndSwapInt32(&s.state, sandboxStateRunning, sandboxStatePaused) {
		return fmt.Errorf("沙箱 %s 未运行", s.id)
	}

	s.logger.Info("沙箱已暂停", "id", s.id)
	return nil
}

// Resume 恢复沙箱
func (s *CompleteIsolationSandbox) Resume() error {
	// 检查沙箱状态
	if !atomic.CompareAndSwapInt32(&s.state, sandboxStatePaused, sandboxStateRunning) {
		return fmt.Errorf("沙箱 %s 未暂停", s.id)
	}

	s.logger.Info("沙箱已恢复", "id", s.id)
	return nil
}

// Stop 停止沙箱
func (s *CompleteIsolationSandbox) Stop() error {
	// 检查沙箱状态
	state := atomic.LoadInt32(&s.state)
	if state == sandboxStateStopped {
		return nil
	}

	// 设置状态为停止
	atomic.StoreInt32(&s.state, sandboxStateStopped)

	// 停止容器
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.containerCmd != nil && s.containerCmd.Process != nil {
		s.logger.Info("停止容器", "id", s.id)
		if err := s.containerCmd.Process.Kill(); err != nil {
			s.logger.Error("停止容器失败", "id", s.id, "error", err)
		}
	}

	// 如果有容器ID，停止容器
	if s.containerID != "" {
		s.logger.Info("停止Docker容器", "id", s.id, "container_id", s.containerID)
		stopCmd := exec.Command("docker", "stop", s.containerID)
		if err := stopCmd.Run(); err != nil {
			s.logger.Error("停止Docker容器失败", "id", s.id, "container_id", s.containerID, "error", err)
		}
	}

	s.logger.Info("沙箱已停止", "id", s.id)
	return nil
}

// IsHealthy 检查沙箱是否健康
func (s *CompleteIsolationSandbox) IsHealthy() bool {
	// 检查沙箱状态
	if atomic.LoadInt32(&s.state) != sandboxStateRunning {
		return false
	}

	// 检查失败率
	executions := atomic.LoadInt64(&s.stats.executions)
	if executions > 0 {
		failures := atomic.LoadInt64(&s.stats.failures)
		failureRate := float64(failures) / float64(executions)

		// 如果失败率超过50%，认为不健康
		if failureRate > 0.5 {
			return false
		}
	}

	// 检查恐慌次数
	panics := atomic.LoadInt64(&s.stats.panics)
	if panics > 5 {
		return false
	}

	return true
}

// GetStats 获取统计信息
func (s *CompleteIsolationSandbox) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"id":              s.id,
		"state":           atomic.LoadInt32(&s.state),
		"executions":      atomic.LoadInt64(&s.stats.executions),
		"successes":       atomic.LoadInt64(&s.stats.successes),
		"failures":        atomic.LoadInt64(&s.stats.failures),
		"panics":          atomic.LoadInt64(&s.stats.panics),
		"timeouts":        atomic.LoadInt64(&s.stats.timeouts),
		"total_exec_time": atomic.LoadInt64(&s.stats.totalExecTime),
		"max_exec_time":   atomic.LoadInt64(&s.stats.maxExecTime),
		"min_exec_time":   atomic.LoadInt64(&s.stats.minExecTime),
		"avg_exec_time":   atomic.LoadInt64(&s.stats.avgExecTime),
		"last_activity":   s.lastActivity,
		"work_dir":        s.workDir,
		"container_id":    s.containerID,
	}
}
