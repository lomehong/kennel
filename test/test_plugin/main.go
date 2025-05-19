package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
)

// TestPlugin 是一个测试插件，用于测试优雅终止功能
type TestPlugin struct {
	logger hclog.Logger
	tasks  chan struct{}
}

// Init 初始化插件
func (p *TestPlugin) Init(config map[string]interface{}) error {
	p.logger.Info("初始化测试插件")

	// 创建任务通道
	p.tasks = make(chan struct{})

	// 启动后台任务
	go p.runBackgroundTask()

	return nil
}

// Execute 执行插件操作
func (p *TestPlugin) Execute(action string, params map[string]interface{}) (map[string]interface{}, error) {
	p.logger.Info("执行操作", "action", action)

	switch action {
	case "test":
		return map[string]interface{}{
			"success": true,
			"message": "测试成功",
		}, nil
	default:
		return nil, fmt.Errorf("不支持的操作: %s", action)
	}
}

// Shutdown 关闭插件
func (p *TestPlugin) Shutdown() error {
	p.logger.Info("开始关闭测试插件")

	// 关闭任务通道，通知后台任务停止
	close(p.tasks)

	// 模拟耗时操作，测试优雅终止
	p.logger.Info("模拟耗时操作，等待3秒...")
	time.Sleep(3 * time.Second)

	p.logger.Info("测试插件已关闭")
	return nil
}

// GetInfo 获取插件信息
func (p *TestPlugin) GetInfo() pluginLib.ModuleInfo {
	return pluginLib.ModuleInfo{
		Name:             "test",
		Version:          "1.0.0",
		Description:      "测试插件，用于测试优雅终止功能",
		SupportedActions: []string{"test"},
	}
}

// runBackgroundTask 运行后台任务
func (p *TestPlugin) runBackgroundTask() {
	p.logger.Info("启动后台任务")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.tasks:
			p.logger.Info("收到停止信号，后台任务退出")
			return
		case <-ticker.C:
			p.logger.Info("后台任务执行中...")
		}
	}
}

func main() {
	// 创建日志
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "test-plugin",
		Output: os.Stdout,
		Level:  hclog.Info,
	})

	// 创建插件
	testPlugin := &TestPlugin{
		logger: logger,
	}

	// 启动插件服务
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "PLUGIN_MAGIC_COOKIE",
			MagicCookieValue: "kennel",
		},
		Plugins: map[string]plugin.Plugin{
			"module": &pluginLib.ModulePlugin{Impl: testPlugin},
		},
		GRPCServer: plugin.DefaultGRPCServer,
		Logger:     logger,
	})
}
