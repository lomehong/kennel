package sdk

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// PluginRunner 用于运行插件
// 处理插件的启动、停止和生命周期管理
type PluginRunner struct {
	// 插件实例
	plugin api.Plugin

	// 日志记录器
	logger hclog.Logger

	// 插件服务器
	server *plugin.GRPCServer

	// 通信实例
	comm Communication

	// 上下文
	ctx context.Context

	// 取消函数
	cancel context.CancelFunc

	// 配置
	config RunnerConfig
}

// RunnerConfig 定义了运行器配置
type RunnerConfig struct {
	// 插件ID
	PluginID string

	// 通信协议
	Protocol CommunicationProtocol

	// 通信选项
	CommOptions map[string]interface{}

	// 日志级别
	LogLevel string

	// 日志文件
	LogFile string

	// 优雅关闭超时
	ShutdownTimeout time.Duration

	// 健康检查间隔
	HealthCheckInterval time.Duration
}

// DefaultRunnerConfig 返回默认运行器配置
func DefaultRunnerConfig() RunnerConfig {
	return RunnerConfig{
		Protocol:            ProtocolGRPC,
		CommOptions:         make(map[string]interface{}),
		LogLevel:            "info",
		ShutdownTimeout:     30 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}
}

// NewPluginRunner 创建一个新的插件运行器
func NewPluginRunner(plugin api.Plugin, config RunnerConfig) *PluginRunner {
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建日志记录器
	var logger hclog.Logger
	if config.LogFile != "" {
		logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			logger = hclog.New(&hclog.LoggerOptions{
				Name:   plugin.GetInfo().ID,
				Level:  hclog.LevelFromString(config.LogLevel),
				Output: logFile,
			})
		}
	}

	if logger == nil {
		logger = hclog.New(&hclog.LoggerOptions{
			Name:   plugin.GetInfo().ID,
			Level:  hclog.LevelFromString(config.LogLevel),
			Output: os.Stderr,
		})
	}

	return &PluginRunner{
		plugin: plugin,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
		config: config,
	}
}

// Run 运行插件
func (r *PluginRunner) Run() error {
	r.logger.Info("启动插件", "id", r.plugin.GetInfo().ID, "version", r.plugin.GetInfo().Version)

	// 设置信号处理
	r.setupSignalHandling()

	// 创建通信实例
	factory := NewCommunicationFactory()
	comm, err := factory.CreateCommunication(r.config.Protocol, r.config.CommOptions)
	if err != nil {
		r.logger.Error("创建通信实例失败", "error", err)
		return fmt.Errorf("创建通信实例失败: %w", err)
	}
	r.comm = comm

	// 初始化插件
	pluginConfig := api.PluginConfig{
		ID:       r.plugin.GetInfo().ID,
		Enabled:  true,
		LogLevel: r.config.LogLevel,
		Settings: make(map[string]interface{}),
	}

	r.logger.Info("初始化插件")
	if err := r.plugin.Init(r.ctx, pluginConfig); err != nil {
		r.logger.Error("初始化插件失败", "error", err)
		return fmt.Errorf("初始化插件失败: %w", err)
	}

	// 启动插件
	r.logger.Info("启动插件")
	if err := r.plugin.Start(r.ctx); err != nil {
		r.logger.Error("启动插件失败", "error", err)
		return fmt.Errorf("启动插件失败: %w", err)
	}

	// 启动健康检查
	if r.config.HealthCheckInterval > 0 {
		go r.runHealthCheck()
	}

	// 如果使用gRPC协议，启动gRPC服务器
	if r.config.Protocol == ProtocolGRPC {
		r.startGRPCServer()
	}

	// 等待上下文取消
	<-r.ctx.Done()

	// 优雅关闭
	r.shutdown()

	return nil
}

// setupSignalHandling 设置信号处理
func (r *PluginRunner) setupSignalHandling() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signalCh
		r.logger.Info("收到信号", "signal", sig)
		r.cancel()
	}()
}

// runHealthCheck 运行健康检查
func (r *PluginRunner) runHealthCheck() {
	ticker := time.NewTicker(r.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			// 执行健康检查
			status, err := r.plugin.HealthCheck(r.ctx)
			if err != nil {
				r.logger.Warn("健康检查失败", "error", err)
			} else {
				r.logger.Debug("健康检查", "status", status.Status)
			}
		}
	}
}

// startGRPCServer 启动gRPC服务器
func (r *PluginRunner) startGRPCServer() {
	// 创建插件映射
	// 注意：这里的映射在简化实现中未使用
	// pluginMap := map[string]plugin.Plugin{
	//     r.plugin.GetInfo().ID: &GRPCPluginAdapter{Impl: r.plugin},
	// }

	// 注意：这里简化了gRPC服务器的实现
	// 在实际实现中，应该使用适当的gRPC服务器
	r.logger.Info("启动gRPC服务器")
	r.logger.Warn("gRPC服务器实现被简化，仅用于演示")
}

// shutdown 关闭插件
func (r *PluginRunner) shutdown() {
	r.logger.Info("关闭插件")

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), r.config.ShutdownTimeout)
	defer cancel()

	// 停止插件
	if err := r.plugin.Stop(ctx); err != nil {
		r.logger.Error("停止插件失败", "error", err)
	}

	// 关闭通信
	if r.comm != nil {
		if err := r.comm.Close(); err != nil {
			r.logger.Error("关闭通信失败", "error", err)
		}
	}

	// 停止gRPC服务器
	// 注意：在简化实现中，server字段未初始化
	// if r.server != nil {
	//     r.server.Stop()
	// }

	r.logger.Info("插件已关闭")
}

// GRPCPluginAdapter gRPC插件适配器
// 用于将我们的插件接口适配到Hashicorp的插件接口
type GRPCPluginAdapter struct {
	plugin.Plugin
	Impl api.Plugin
}

// GRPCServer 实现Plugin接口
func (p *GRPCPluginAdapter) GRPCServer(broker *plugin.GRPCBroker, s interface{}) error {
	// 在实际实现中，这里需要注册gRPC服务
	return nil
}

// GRPCClient 实现Plugin接口
func (p *GRPCPluginAdapter) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c interface{}) (interface{}, error) {
	// 在实际实现中，这里需要创建gRPC客户端
	return p.Impl, nil
}

// Run 运行插件的便捷函数
func Run(plugin api.Plugin, config RunnerConfig) error {
	runner := NewPluginRunner(plugin, config)
	return runner.Run()
}
