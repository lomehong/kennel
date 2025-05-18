package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/core/plugin"
)

// ExampleGoPlugin 示例Go插件
type ExampleGoPlugin struct {
	logger    hclog.Logger
	config    map[string]interface{}
	startTime time.Time
}

// NewExampleGoPlugin 创建示例Go插件
func NewExampleGoPlugin() plugin.Module {
	return &ExampleGoPlugin{
		logger: hclog.New(&hclog.LoggerOptions{
			Name:  "example-go",
			Level: hclog.Debug,
		}),
		startTime: time.Now(),
	}
}

// Init 初始化模块
func (p *ExampleGoPlugin) Init(ctx context.Context, config *plugin.ModuleConfig) error {
	p.logger.Info("初始化Go示例插件")
	p.config = config.Settings
	p.logger.Debug("配置", "config", config.Settings)
	return nil
}

// Start 启动模块
func (p *ExampleGoPlugin) Start() error {
	p.logger.Info("启动Go示例插件")
	p.startTime = time.Now()
	return nil
}

// Stop 停止模块
func (p *ExampleGoPlugin) Stop() error {
	p.logger.Info("停止Go示例插件")
	uptime := time.Since(p.startTime)
	p.logger.Info("运行时间", "uptime", uptime.String())
	return nil
}

// GetInfo 获取模块信息
func (p *ExampleGoPlugin) GetInfo() plugin.ModuleInfo {
	return plugin.ModuleInfo{
		ID:                "example-go",
		Name:              "Go示例插件",
		Version:           "1.0.0",
		Description:       "使用Go实现的示例插件",
		Author:            "Kennel Team",
		License:           "MIT",
		Capabilities:      []string{"example"},
		SupportedPlatforms: []string{"windows", "linux", "darwin"},
		Language:          "go",
	}
}

// HandleRequest 处理请求
func (p *ExampleGoPlugin) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	p.logger.Info("处理请求", "action", req.Action)

	switch req.Action {
	case "hello":
		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"message":   "Hello from Go!",
				"timestamp": time.Now().Unix(),
			},
		}, nil
	case "get_system_info":
		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data:    p.getSystemInfo(),
		}, nil
	case "echo":
		message, _ := req.Params["message"].(string)
		return &plugin.Response{
			ID:      req.ID,
			Success: true,
			Data: map[string]interface{}{
				"message":   message,
				"timestamp": time.Now().Unix(),
			},
		}, nil
	default:
		return &plugin.Response{
			ID:      req.ID,
			Success: false,
			Error: &plugin.ErrorInfo{
				Code:    "unknown_action",
				Message: fmt.Sprintf("未知操作: %s", req.Action),
			},
		}, nil
	}
}

// HandleEvent 处理事件
func (p *ExampleGoPlugin) HandleEvent(ctx context.Context, event *plugin.Event) error {
	p.logger.Info("处理事件", "type", event.Type, "source", event.Source)

	switch event.Type {
	case "system.startup":
		p.logger.Info("系统启动事件")
		return nil
	case "system.shutdown":
		p.logger.Info("系统关闭事件")
		return nil
	default:
		p.logger.Warn("未处理的事件类型", "type", event.Type)
		return nil
	}
}

// getSystemInfo 获取系统信息
func (p *ExampleGoPlugin) getSystemInfo() map[string]interface{} {
	return map[string]interface{}{
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
		"num_cpu":         runtime.NumCPU(),
		"go_version":      runtime.Version(),
		"num_goroutines":  runtime.NumGoroutine(),
		"uptime":          time.Since(p.startTime).String(),
		"uptime_seconds":  time.Since(p.startTime).Seconds(),
		"memory_alloc":    runtime.MemStats{}.Alloc,
		"memory_total":    runtime.MemStats{}.TotalAlloc,
		"memory_sys":      runtime.MemStats{}.Sys,
		"memory_num_gc":   runtime.MemStats{}.NumGC,
		"memory_next_gc":  runtime.MemStats{}.NextGC,
		"memory_last_gc":  runtime.MemStats{}.LastGC,
		"memory_pause_ns": runtime.MemStats{}.PauseNs,
	}
}

// CheckHealth 检查健康状态
func (p *ExampleGoPlugin) CheckHealth() plugin.HealthStatus {
	return plugin.HealthStatus{
		Status: "healthy",
		Details: map[string]interface{}{
			"uptime":         time.Since(p.startTime).String(),
			"num_goroutines": runtime.NumGoroutine(),
		},
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}
}

func main() {
	// 创建插件
	plugin := NewExampleGoPlugin()

	// 这里应该使用SDK提供的Serve函数启动插件
	// 由于SDK尚未实现，这里只是示例
	fmt.Println("Go示例插件已启动")
	select {}
}
