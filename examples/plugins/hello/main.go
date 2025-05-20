package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"github.com/lomehong/kennel/pkg/plugin/sdk"
)

// HelloPlugin 示例插件
type HelloPlugin struct {
	// 基础插件
	*sdk.BasePlugin

	// 配置
	config *HelloConfig

	// 调试服务器
	debugServer *sdk.DebugServer

	// 配置管理器
	configManager *sdk.ConfigManager

	// 是否运行中
	running bool

	// 停止通道
	stopCh chan struct{}
}

// HelloConfig 插件配置
type HelloConfig struct {
	// 消息
	Message string `json:"message" yaml:"message"`

	// 间隔
	Interval time.Duration `json:"interval" yaml:"interval"`

	// 调试端口
	DebugPort int `json:"debug_port" yaml:"debug_port"`

	// 是否启用调试
	DebugEnabled bool `json:"debug_enabled" yaml:"debug_enabled"`
}

// NewHelloPlugin 创建一个新的示例插件
func NewHelloPlugin(logger hclog.Logger) *HelloPlugin {
	// 创建插件信息
	info := api.PluginInfo{
		ID:          "hello",
		Name:        "Hello Plugin",
		Version:     "1.0.0",
		Description: "A simple hello plugin",
		Author:      "Example Author",
		License:     "MIT",
		Tags:        []string{"example", "hello"},
		Capabilities: map[string]bool{
			"hello": true,
		},
	}

	// 创建基础插件
	basePlugin := sdk.NewBasePlugin(info, logger)

	return &HelloPlugin{
		BasePlugin: basePlugin,
		stopCh:     make(chan struct{}),
	}
}

// Init 初始化插件
func (p *HelloPlugin) Init(ctx context.Context, config api.PluginConfig) error {
	// 调用基类初始化
	if err := p.BasePlugin.Init(ctx, config); err != nil {
		return err
	}

	p.GetLogger().Info("初始化插件")

	// 创建配置管理器
	configManager, err := sdk.NewConfigManager(p.GetInfo().ID, p.GetLogger())
	if err != nil {
		return fmt.Errorf("创建配置管理器失败: %w", err)
	}

	// 加载配置
	if err := configManager.Load(); err != nil {
		p.GetLogger().Warn("加载配置失败", "error", err)
	}

	// 设置插件配置
	configManager.SetPluginConfig(config)

	// 保存配置
	if err := configManager.Save(); err != nil {
		p.GetLogger().Warn("保存配置失败", "error", err)
	}

	p.configManager = configManager

	// 解析配置
	p.config = &HelloConfig{
		Message:      "Hello, World!",
		Interval:     5 * time.Second,
		DebugPort:    8080,
		DebugEnabled: false,
	}

	// 从配置中获取消息
	if message := configManager.GetString("settings.message"); message != "" {
		p.config.Message = message
	}

	// 从配置中获取间隔
	if interval := configManager.GetInt("settings.interval"); interval > 0 {
		p.config.Interval = time.Duration(interval) * time.Second
	}

	// 从配置中获取调试端口
	if debugPort := configManager.GetInt("settings.debug_port"); debugPort > 0 {
		p.config.DebugPort = debugPort
	}

	// 从配置中获取是否启用调试
	p.config.DebugEnabled = configManager.GetBool("settings.debug_enabled")

	// 创建调试服务器
	p.debugServer = sdk.NewDebugServer(
		p.GetInfo().ID,
		p.GetLogger(),
		sdk.WithDebugPort(p.config.DebugPort),
		sdk.WithDebugEnabled(p.config.DebugEnabled),
	)

	// 注册自定义处理器
	p.debugServer.RegisterHandler("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": p.config.Message,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	return nil
}

// Start 启动插件
func (p *HelloPlugin) Start(ctx context.Context) error {
	// 调用基类启动
	if err := p.BasePlugin.Start(ctx); err != nil {
		return err
	}

	p.GetLogger().Info("启动插件")

	// 启动调试服务器
	if err := p.debugServer.Start(); err != nil {
		p.GetLogger().Error("启动调试服务器失败", "error", err)
	}

	// 启动后台任务
	p.running = true
	go p.run()

	return nil
}

// Stop 停止插件
func (p *HelloPlugin) Stop(ctx context.Context) error {
	// 调用基类停止
	if err := p.BasePlugin.Stop(ctx); err != nil {
		return err
	}

	p.GetLogger().Info("停止插件")

	// 停止后台任务
	if p.running {
		p.running = false
		close(p.stopCh)
	}

	// 停止调试服务器
	if err := p.debugServer.Stop(); err != nil {
		p.GetLogger().Error("停止调试服务器失败", "error", err)
	}

	return nil
}

// HealthCheck 执行健康检查
func (p *HelloPlugin) HealthCheck(ctx context.Context) (api.HealthStatus, error) {
	// 检查是否运行中
	status := "healthy"
	if !p.running {
		status = "unhealthy"
	}

	return api.HealthStatus{
		Status:  status,
		Details: map[string]interface{}{"running": p.running},
		LastChecked: time.Now(),
	}, nil
}

// run 运行后台任务
func (p *HelloPlugin) run() {
	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.GetLogger().Info(p.config.Message)
		}
	}
}

func main() {
	// 创建日志记录器
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "hello-plugin",
		Level:  hclog.LevelFromString("info"),
		Output: os.Stdout,
	})

	// 创建插件
	plugin := NewHelloPlugin(logger)

	// 创建插件运行器配置
	config := sdk.DefaultRunnerConfig()
	config.PluginID = "hello"
	config.LogLevel = "info"
	config.LogFile = "logs/hello.log"
	config.ShutdownTimeout = 30 * time.Second
	config.HealthCheckInterval = 30 * time.Second

	// 运行插件
	if err := sdk.Run(plugin, config); err != nil {
		fmt.Fprintf(os.Stderr, "运行插件失败: %v\n", err)
		os.Exit(1)
	}
}
