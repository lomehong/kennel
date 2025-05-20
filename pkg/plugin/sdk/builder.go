package sdk

import (
	"context"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// InitFunc 定义了初始化函数类型
type InitFunc func(ctx context.Context, config api.PluginConfig) error

// StartFunc 定义了启动函数类型
type StartFunc func(ctx context.Context) error

// StopFunc 定义了停止函数类型
type StopFunc func(ctx context.Context) error

// HealthCheckFunc 定义了健康检查函数类型
type HealthCheckFunc func(ctx context.Context) (api.HealthStatus, error)

// PluginBuilder 用于构建插件
// 提供了流式API，简化插件的创建过程
type PluginBuilder struct {
	// 插件信息
	info api.PluginInfo

	// 日志记录器
	logger hclog.Logger

	// 初始化函数
	initFunc InitFunc

	// 启动函数
	startFunc StartFunc

	// 停止函数
	stopFunc StopFunc

	// 健康检查函数
	healthCheckFunc HealthCheckFunc
}

// NewPluginBuilder 创建一个新的插件构建器
func NewPluginBuilder(id string) *PluginBuilder {
	return &PluginBuilder{
		info: api.PluginInfo{
			ID:           id,
			Name:         id,
			Version:      "1.0.0",
			Dependencies: []api.PluginDependency{},
			Tags:         []string{},
			Capabilities: make(map[string]bool),
		},
		logger: hclog.NewNullLogger(),
	}
}

// WithName 设置插件名称
func (b *PluginBuilder) WithName(name string) *PluginBuilder {
	b.info.Name = name
	return b
}

// WithVersion 设置插件版本
func (b *PluginBuilder) WithVersion(version string) *PluginBuilder {
	b.info.Version = version
	return b
}

// WithDescription 设置插件描述
func (b *PluginBuilder) WithDescription(description string) *PluginBuilder {
	b.info.Description = description
	return b
}

// WithAuthor 设置插件作者
func (b *PluginBuilder) WithAuthor(author string) *PluginBuilder {
	b.info.Author = author
	return b
}

// WithLicense 设置插件许可证
func (b *PluginBuilder) WithLicense(license string) *PluginBuilder {
	b.info.License = license
	return b
}

// WithTag 添加插件标签
func (b *PluginBuilder) WithTag(tag string) *PluginBuilder {
	b.info.Tags = append(b.info.Tags, tag)
	return b
}

// WithCapability 添加插件能力
func (b *PluginBuilder) WithCapability(capability string) *PluginBuilder {
	b.info.Capabilities[capability] = true
	return b
}

// WithDependency 添加插件依赖
func (b *PluginBuilder) WithDependency(id, version string, optional bool) *PluginBuilder {
	b.info.Dependencies = append(b.info.Dependencies, api.PluginDependency{
		ID:       id,
		Version:  version,
		Optional: optional,
	})
	return b
}

// WithLogger 设置日志记录器
func (b *PluginBuilder) WithLogger(logger hclog.Logger) *PluginBuilder {
	if logger != nil {
		b.logger = logger
	}
	return b
}

// WithInitFunc 设置初始化函数
func (b *PluginBuilder) WithInitFunc(initFunc InitFunc) *PluginBuilder {
	b.initFunc = initFunc
	return b
}

// WithStartFunc 设置启动函数
func (b *PluginBuilder) WithStartFunc(startFunc StartFunc) *PluginBuilder {
	b.startFunc = startFunc
	return b
}

// WithStopFunc 设置停止函数
func (b *PluginBuilder) WithStopFunc(stopFunc StopFunc) *PluginBuilder {
	b.stopFunc = stopFunc
	return b
}

// WithHealthCheckFunc 设置健康检查函数
func (b *PluginBuilder) WithHealthCheckFunc(healthCheckFunc HealthCheckFunc) *PluginBuilder {
	b.healthCheckFunc = healthCheckFunc
	return b
}

// Build 构建插件
func (b *PluginBuilder) Build() api.Plugin {
	// 创建基础插件
	basePlugin := NewBasePlugin(b.info, b.logger)

	// 创建自定义插件
	plugin := &customPlugin{
		BasePlugin:     basePlugin,
		initFunc:       b.initFunc,
		startFunc:      b.startFunc,
		stopFunc:       b.stopFunc,
		healthCheckFunc: b.healthCheckFunc,
	}

	return plugin
}

// customPlugin 自定义插件实现
type customPlugin struct {
	*BasePlugin
	initFunc       InitFunc
	startFunc      StartFunc
	stopFunc       StopFunc
	healthCheckFunc HealthCheckFunc
}

// Init 初始化插件
func (p *customPlugin) Init(ctx context.Context, config api.PluginConfig) error {
	// 调用基础实现
	if err := p.BasePlugin.Init(ctx, config); err != nil {
		return err
	}

	// 调用自定义初始化函数
	if p.initFunc != nil {
		return p.initFunc(ctx, config)
	}

	return nil
}

// Start 启动插件
func (p *customPlugin) Start(ctx context.Context) error {
	// 调用基础实现
	if err := p.BasePlugin.Start(ctx); err != nil {
		return err
	}

	// 调用自定义启动函数
	if p.startFunc != nil {
		return p.startFunc(ctx)
	}

	return nil
}

// Stop 停止插件
func (p *customPlugin) Stop(ctx context.Context) error {
	// 调用基础实现
	if err := p.BasePlugin.Stop(ctx); err != nil {
		return err
	}

	// 调用自定义停止函数
	if p.stopFunc != nil {
		return p.stopFunc(ctx)
	}

	return nil
}

// HealthCheck 执行健康检查
func (p *customPlugin) HealthCheck(ctx context.Context) (api.HealthStatus, error) {
	// 如果有自定义健康检查函数，调用它
	if p.healthCheckFunc != nil {
		return p.healthCheckFunc(ctx)
	}

	// 否则调用基础实现
	return p.BasePlugin.HealthCheck(ctx)
}
