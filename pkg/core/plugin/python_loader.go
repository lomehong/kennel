package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// PythonPluginLoader Python插件加载器
type PythonPluginLoader struct {
	logger hclog.Logger
}

// NewPythonPluginLoader 创建Python插件加载器
func NewPythonPluginLoader(logger hclog.Logger) *PythonPluginLoader {
	return &PythonPluginLoader{
		logger: logger.Named("python-loader"),
	}
}

// PythonPluginInstance Python插件实例
type PythonPluginInstance struct {
	metadata PluginMetadata
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	scanner  *bufio.Scanner
	info     ModuleInfo
	logger   hclog.Logger
	mu       sync.Mutex
}

// LoadPlugin 加载Python插件
func (l *PythonPluginLoader) LoadPlugin(metadata PluginMetadata) (Module, *PluginProcess, error) {
	l.logger.Info("加载Python插件", "id", metadata.ID, "path", metadata.Path)

	// 构建插件路径
	scriptPath := filepath.Join(metadata.Path, metadata.EntryPoint.Path)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("Python脚本不存在: %s", scriptPath)
	}

	// 确定Python解释器
	interpreter := metadata.EntryPoint.Interpreter
	if interpreter == "" {
		interpreter = "python"
	}

	// 创建命令
	cmd := exec.Command(interpreter, scriptPath)

	// 设置环境变量
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KENNEL_PLUGIN_ID=%s", metadata.ID),
		"KENNEL_PLUGIN_CONFIG={}",
	)

	// 创建管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("创建标准输入管道失败: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("创建标准输出管道失败: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("创建标准错误管道失败: %w", err)
	}

	// 创建插件实例
	instance := &PythonPluginInstance{
		metadata: metadata,
		cmd:      cmd,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		scanner:  bufio.NewScanner(stdout),
		logger:   l.logger.Named(metadata.ID),
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("启动Python进程失败: %w", err)
	}

	// 处理标准错误
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			instance.logger.Error("Python错误", "message", scanner.Text())
		}
	}()

	// 等待就绪信息
	ready, err := instance.waitForReady()
	if err != nil {
		// 杀死进程
		_ = cmd.Process.Kill()
		return nil, nil, fmt.Errorf("等待Python插件就绪失败: %w", err)
	}

	// 解析模块信息
	if err := json.Unmarshal([]byte(ready), &instance.info); err != nil {
		// 杀死进程
		_ = cmd.Process.Kill()
		return nil, nil, fmt.Errorf("解析模块信息失败: %w", err)
	}

	// 创建插件进程
	process := &PluginProcess{
		PID:    cmd.Process.Pid,
		Cmd:    cmd,
		Client: nil,
		Conn:   nil,
	}

	return instance, process, nil
}

// waitForReady 等待Python插件就绪
func (p *PythonPluginInstance) waitForReady() (string, error) {
	// 设置超时
	timeout := time.After(30 * time.Second)
	readyCh := make(chan string, 1)
	errorCh := make(chan error, 1)

	// 读取输出
	go func() {
		for p.scanner.Scan() {
			line := p.scanner.Text()
			if strings.HasPrefix(line, "KENNEL_PLUGIN_READY:") {
				readyCh <- line[len("KENNEL_PLUGIN_READY:"):]
				return
			} else if strings.HasPrefix(line, "KENNEL_PLUGIN_ERROR:") {
				errorCh <- fmt.Errorf("Python插件错误: %s", line[len("KENNEL_PLUGIN_ERROR:"):])
				return
			}
		}
		if err := p.scanner.Err(); err != nil {
			errorCh <- fmt.Errorf("读取Python输出失败: %w", err)
		} else {
			errorCh <- fmt.Errorf("Python进程意外退出")
		}
	}()

	// 等待就绪或超时
	select {
	case ready := <-readyCh:
		return ready, nil
	case err := <-errorCh:
		return "", err
	case <-timeout:
		return "", fmt.Errorf("等待Python插件就绪超时")
	}
}

// Init 初始化模块
func (p *PythonPluginInstance) Init(ctx context.Context, config *ModuleConfig) error {
	// 已经在加载时初始化
	return nil
}

// Start 启动模块
func (p *PythonPluginInstance) Start() error {
	// 已经在加载时启动
	return nil
}

// Stop 停止模块
func (p *PythonPluginInstance) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 发送停止命令
	if _, err := fmt.Fprintln(p.stdin, "KENNEL_STOP"); err != nil {
		return fmt.Errorf("发送停止命令失败: %w", err)
	}

	// 等待进程退出
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	// 设置超时
	timeout := time.After(5 * time.Second)

	// 等待进程退出或超时
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("Python进程退出错误: %w", err)
		}
		return nil
	case <-timeout:
		// 超时，强制杀死进程
		if err := p.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("杀死Python进程失败: %w", err)
		}
		return fmt.Errorf("Python进程停止超时，已强制杀死")
	}
}

// GetInfo 获取模块信息
func (p *PythonPluginInstance) GetInfo() ModuleInfo {
	return p.info
}

// HandleRequest 处理请求
func (p *PythonPluginInstance) HandleRequest(ctx context.Context, req *Request) (*Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 构建命令
	command := map[string]interface{}{
		"type": "request",
		"data": map[string]interface{}{
			"id":       req.ID,
			"action":   req.Action,
			"params":   req.Params,
			"metadata": req.Metadata,
			"timeout":  req.Timeout,
		},
	}

	// 序列化命令
	commandJSON, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送命令
	if _, err := fmt.Fprintf(p.stdin, "KENNEL_COMMAND:%s\n", commandJSON); err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	// 等待响应
	responseCh := make(chan *Response, 1)
	errorCh := make(chan error, 1)

	go func() {
		for p.scanner.Scan() {
			line := p.scanner.Text()
			if strings.HasPrefix(line, "KENNEL_RESPONSE:") {
				responseJSON := line[len("KENNEL_RESPONSE:"):]
				var response Response
				if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
					errorCh <- fmt.Errorf("解析响应失败: %w", err)
					return
				}
				responseCh <- &response
				return
			} else if strings.HasPrefix(line, "KENNEL_ERROR:") {
				errorJSON := line[len("KENNEL_ERROR:"):]
				var errorData struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal([]byte(errorJSON), &errorData); err != nil {
					errorCh <- fmt.Errorf("解析错误失败: %w", err)
					return
				}
				errorCh <- fmt.Errorf("Python插件错误: %s", errorData.Error)
				return
			}
		}
		if err := p.scanner.Err(); err != nil {
			errorCh <- fmt.Errorf("读取Python输出失败: %w", err)
		} else {
			errorCh <- fmt.Errorf("Python进程意外退出")
		}
	}()

	// 设置超时
	timeout := time.After(time.Duration(req.Timeout) * time.Millisecond)

	// 等待响应或超时
	select {
	case response := <-responseCh:
		return response, nil
	case err := <-errorCh:
		return nil, err
	case <-timeout:
		return nil, fmt.Errorf("请求超时")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// HandleEvent 处理事件
func (p *PythonPluginInstance) HandleEvent(ctx context.Context, event *Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 构建命令
	command := map[string]interface{}{
		"type": "event",
		"data": map[string]interface{}{
			"id":        event.ID,
			"type":      event.Type,
			"source":    event.Source,
			"timestamp": event.Timestamp,
			"data":      event.Data,
			"metadata":  event.Metadata,
		},
	}

	// 序列化命令
	commandJSON, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("序列化事件失败: %w", err)
	}

	// 发送命令
	if _, err := fmt.Fprintf(p.stdin, "KENNEL_COMMAND:%s\n", commandJSON); err != nil {
		return fmt.Errorf("发送事件失败: %w", err)
	}

	// 等待响应
	responseCh := make(chan bool, 1)
	errorCh := make(chan error, 1)

	go func() {
		for p.scanner.Scan() {
			line := p.scanner.Text()
			if strings.HasPrefix(line, "KENNEL_EVENT_RESPONSE:") {
				responseJSON := line[len("KENNEL_EVENT_RESPONSE:"):]
				var response struct {
					EventID string `json:"event_id"`
					Success bool   `json:"success"`
				}
				if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
					errorCh <- fmt.Errorf("解析事件响应失败: %w", err)
					return
				}
				responseCh <- response.Success
				return
			} else if strings.HasPrefix(line, "KENNEL_ERROR:") {
				errorJSON := line[len("KENNEL_ERROR:"):]
				var errorData struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal([]byte(errorJSON), &errorData); err != nil {
					errorCh <- fmt.Errorf("解析错误失败: %w", err)
					return
				}
				errorCh <- fmt.Errorf("Python插件错误: %s", errorData.Error)
				return
			}
		}
		if err := p.scanner.Err(); err != nil {
			errorCh <- fmt.Errorf("读取Python输出失败: %w", err)
		} else {
			errorCh <- fmt.Errorf("Python进程意外退出")
		}
	}()

	// 设置超时
	timeout := time.After(30 * time.Second)

	// 等待响应或超时
	select {
	case success := <-responseCh:
		if !success {
			return fmt.Errorf("事件处理失败")
		}
		return nil
	case err := <-errorCh:
		return err
	case <-timeout:
		return fmt.Errorf("事件处理超时")
	case <-ctx.Done():
		return ctx.Err()
	}
}
