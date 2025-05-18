package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin"
)

// ExamplePlugin 示例插件
type ExamplePlugin struct {
	*plugin.BasePlugin
	dataStore map[string]interface{}
}

// NewExamplePlugin 创建示例插件
func NewExamplePlugin() *ExamplePlugin {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "example-plugin",
		Level:  hclog.Info,
		Output: os.Stdout,
	})

	// 创建插件元数据
	metadata := plugin.PluginMetadata{
		ID:             "example-plugin",
		Name:           "Example Plugin",
		Version:        "1.0.0",
		Author:         "Example Author",
		Description:    "Example plugin for demonstration",
		Dependencies:   []string{},
		Tags:           []string{"example", "demo"},
		EntryPoint:     "main.go",
		IsolationLevel: plugin.PluginIsolationNone,
		AutoStart:      true,
		AutoRestart:    false,
		Enabled:        true,
		Config: map[string]interface{}{
			"setting1": "value1",
			"setting2": 42,
			"setting3": true,
		},
	}

	// 创建基础插件
	basePlugin := plugin.NewBasePlugin(metadata, logger)

	return &ExamplePlugin{
		BasePlugin: basePlugin,
		dataStore:  make(map[string]interface{}),
	}
}

// Init 初始化插件
func (p *ExamplePlugin) Init(ctx context.Context, config map[string]interface{}) error {
	// 调用基础插件的初始化方法
	if err := p.BasePlugin.Init(ctx, config); err != nil {
		return err
	}

	// 初始化数据存储
	p.dataStore["example"] = "Hello, World!"
	p.dataStore["timestamp"] = time.Now().Unix()

	// 注册接口
	p.registerInterface("data_store", p.dataStore)
	p.registerInterface("message_handler", p.HandleMessage)

	p.logger.Info("示例插件已初始化", "id", p.ID())
	return nil
}

// Start 启动插件
func (p *ExamplePlugin) Start(ctx context.Context) error {
	// 调用基础插件的启动方法
	if err := p.BasePlugin.Start(ctx); err != nil {
		return err
	}

	// 启动后台任务
	go p.backgroundTask(ctx)

	p.logger.Info("示例插件已启动", "id", p.ID())
	return nil
}

// Stop 停止插件
func (p *ExamplePlugin) Stop(ctx context.Context) error {
	// 调用基础插件的停止方法
	if err := p.BasePlugin.Stop(ctx); err != nil {
		return err
	}

	p.logger.Info("示例插件已停止", "id", p.ID())
	return nil
}

// backgroundTask 后台任务
func (p *ExamplePlugin) backgroundTask(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("后台任务已停止", "id", p.ID())
			return
		case <-ticker.C:
			// 更新时间戳
			p.dataStore["timestamp"] = time.Now().Unix()
			p.logger.Info("后台任务已执行", "id", p.ID(), "timestamp", p.dataStore["timestamp"])
		}
	}
}

// HandleMessage 处理消息
func (p *ExamplePlugin) HandleMessage(ctx context.Context, msg plugin.Message) (plugin.Message, error) {
	p.logger.Info("收到消息",
		"id", msg.ID,
		"type", msg.Type,
		"source", msg.Source,
		"topic", msg.Topic,
	)

	// 检查消息类型
	if msg.Type == plugin.MessageTypeEvent {
		// 处理事件
		return p.handleEvent(ctx, msg)
	}

	// 检查消息主题
	switch msg.Topic {
	case "example.request":
		// 处理请求
		return p.handleRequest(ctx, msg)
	default:
		// 未知主题
		return plugin.Message{}, fmt.Errorf("未知主题: %s", msg.Topic)
	}
}

// handleEvent 处理事件
func (p *ExamplePlugin) handleEvent(ctx context.Context, msg plugin.Message) (plugin.Message, error) {
	// 获取事件类型
	eventType, ok := msg.Payload["event_type"].(string)
	if !ok {
		return plugin.Message{}, fmt.Errorf("事件类型无效")
	}

	p.logger.Info("处理事件", "event_type", eventType)

	// 处理不同类型的事件
	switch eventType {
	case "data_updated":
		// 更新数据
		if data, ok := msg.Payload["data"].(map[string]interface{}); ok {
			for key, value := range data {
				p.dataStore[key] = value
			}
		}
	}

	// 事件不需要响应
	return plugin.Message{}, nil
}

// handleRequest 处理请求
func (p *ExamplePlugin) handleRequest(ctx context.Context, msg plugin.Message) (plugin.Message, error) {
	// 获取操作
	action, ok := msg.Payload["action"].(string)
	if !ok {
		return plugin.Message{}, fmt.Errorf("操作无效")
	}

	p.logger.Info("处理请求", "action", action)

	// 处理不同类型的操作
	switch action {
	case "getData":
		// 获取数据
		params, ok := msg.Payload["params"].(map[string]interface{})
		if !ok {
			return plugin.Message{}, fmt.Errorf("参数无效")
		}

		// 获取ID
		id, ok := params["id"].(float64)
		if !ok {
			return plugin.Message{}, fmt.Errorf("ID无效")
		}

		// 创建响应
		return plugin.Message{
			ID:     fmt.Sprintf("resp-%s", msg.ID),
			Type:   plugin.MessageTypeResponse,
			Source: p.ID(),
			Topic:  msg.Topic,
			Payload: map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"id":        id,
					"name":      "Example",
					"value":     p.dataStore["example"],
					"timestamp": p.dataStore["timestamp"],
				},
			},
		}, nil

	case "setData":
		// 设置数据
		params, ok := msg.Payload["params"].(map[string]interface{})
		if !ok {
			return plugin.Message{}, fmt.Errorf("参数无效")
		}

		// 获取键和值
		key, ok := params["key"].(string)
		if !ok {
			return plugin.Message{}, fmt.Errorf("键无效")
		}

		// 设置数据
		p.dataStore[key] = params["value"]

		// 创建响应
		return plugin.Message{
			ID:     fmt.Sprintf("resp-%s", msg.ID),
			Type:   plugin.MessageTypeResponse,
			Source: p.ID(),
			Topic:  msg.Topic,
			Payload: map[string]interface{}{
				"success": true,
				"key":     key,
				"value":   params["value"],
			},
		}, nil

	default:
		// 未知操作
		return plugin.Message{}, fmt.Errorf("未知操作: %s", action)
	}
}

// Plugin 导出插件
var Plugin plugin.Plugin = NewExamplePlugin()

func main() {
	// 创建插件
	p := NewExamplePlugin()

	// 初始化插件
	if err := p.Init(context.Background(), nil); err != nil {
		fmt.Printf("初始化插件失败: %v\n", err)
		os.Exit(1)
	}

	// 启动插件
	if err := p.Start(context.Background()); err != nil {
		fmt.Printf("启动插件失败: %v\n", err)
		os.Exit(1)
	}

	// 等待信号
	fmt.Println("插件已启动，按Ctrl+C停止")
	select {}
}
