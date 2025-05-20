package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
)

// PluginDiscoverer 插件发现器接口
// 负责发现可用的插件
type PluginDiscoverer interface {
	// DiscoverPlugins 发现插件
	// ctx: 上下文
	// 返回: 插件元数据列表和错误
	DiscoverPlugins(ctx context.Context) ([]api.PluginMetadata, error)

	// WatchPlugins 监听插件变化
	// ctx: 上下文
	// 返回: 插件事件通道和错误
	WatchPlugins(ctx context.Context) (<-chan api.PluginEvent, error)

	// Close 关闭发现器
	// 返回: 错误
	Close() error
}

// FileSystemDiscoverer 文件系统插件发现器
// 从文件系统中发现插件
type FileSystemDiscoverer struct {
	// 插件目录
	directories []string

	// 文件模式
	patterns []string

	// 递归扫描
	recursive bool

	// 日志记录器
	logger hclog.Logger

	// 上次扫描时间
	lastScanTime time.Time

	// 已发现的插件
	discoveredPlugins map[string]api.PluginMetadata

	// 互斥锁
	mu sync.RWMutex

	// 事件通道
	eventCh chan api.PluginEvent

	// 上下文
	ctx context.Context

	// 取消函数
	cancel context.CancelFunc
}

// FileSystemDiscovererOption 文件系统插件发现器选项
type FileSystemDiscovererOption func(*FileSystemDiscoverer)

// WithLogger 设置日志记录器
func WithLogger(logger hclog.Logger) FileSystemDiscovererOption {
	return func(d *FileSystemDiscoverer) {
		if logger != nil {
			d.logger = logger
		}
	}
}

// WithRecursive 设置是否递归扫描
func WithRecursive(recursive bool) FileSystemDiscovererOption {
	return func(d *FileSystemDiscoverer) {
		d.recursive = recursive
	}
}

// WithPatterns 设置文件模式
func WithPatterns(patterns []string) FileSystemDiscovererOption {
	return func(d *FileSystemDiscoverer) {
		if len(patterns) > 0 {
			d.patterns = patterns
		}
	}
}

// NewFileSystemDiscoverer 创建一个新的文件系统插件发现器
func NewFileSystemDiscoverer(directories []string, options ...FileSystemDiscovererOption) *FileSystemDiscoverer {
	ctx, cancel := context.WithCancel(context.Background())

	d := &FileSystemDiscoverer{
		directories:       directories,
		patterns:          []string{"*.exe", "*.so", "*.dll", "*.dylib"},
		recursive:         true,
		logger:            hclog.NewNullLogger(),
		discoveredPlugins: make(map[string]api.PluginMetadata),
		eventCh:           make(chan api.PluginEvent, 100),
		ctx:               ctx,
		cancel:            cancel,
	}

	// 应用选项
	for _, option := range options {
		option(d)
	}

	return d
}

// DiscoverPlugins 发现插件
func (d *FileSystemDiscoverer) DiscoverPlugins(ctx context.Context) ([]api.PluginMetadata, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Info("开始发现插件", "directories", d.directories)
	d.lastScanTime = time.Now()

	// 清空已发现的插件
	d.discoveredPlugins = make(map[string]api.PluginMetadata)

	// 遍历目录
	for _, dir := range d.directories {
		if err := d.scanDirectory(dir); err != nil {
			d.logger.Error("扫描目录失败", "directory", dir, "error", err)
			continue
		}
	}

	// 转换为列表
	plugins := make([]api.PluginMetadata, 0, len(d.discoveredPlugins))
	for _, metadata := range d.discoveredPlugins {
		plugins = append(plugins, metadata)
	}

	d.logger.Info("插件发现完成", "count", len(plugins))
	return plugins, nil
}

// scanDirectory 扫描目录
func (d *FileSystemDiscoverer) scanDirectory(dir string) error {
	// 检查目录是否存在
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			d.logger.Warn("目录不存在", "directory", dir)
			return nil
		}
		return err
	}

	// 检查是否为目录
	if !info.IsDir() {
		d.logger.Warn("不是目录", "path", dir)
		return nil
	}

	// 读取目录
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// 遍历目录项
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		// 如果是目录且允许递归，则递归扫描
		if entry.IsDir() && d.recursive {
			if err := d.scanDirectory(path); err != nil {
				d.logger.Error("递归扫描目录失败", "directory", path, "error", err)
				continue
			}
			continue
		}

		// 检查是否匹配模式
		matched := false
		for _, pattern := range d.patterns {
			if match, _ := filepath.Match(pattern, entry.Name()); match {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		// 尝试加载插件元数据
		metadata, err := d.loadPluginMetadata(path)
		if err != nil {
			d.logger.Debug("加载插件元数据失败", "path", path, "error", err)
			continue
		}

		// 存储插件元数据
		d.discoveredPlugins[metadata.ID] = metadata

		// 发送插件发现事件
		d.sendPluginEvent(api.PluginEvent{
			Type:      "discovered",
			PluginID:  metadata.ID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"metadata": metadata,
			},
		})
	}

	return nil
}

// loadPluginMetadata 加载插件元数据
func (d *FileSystemDiscoverer) loadPluginMetadata(path string) (api.PluginMetadata, error) {
	// 在实际实现中，这里应该执行插件的元数据加载逻辑
	// 可能涉及到执行插件并获取其元数据，或者读取元数据文件

	// 这里简化处理，使用文件名作为插件ID
	id := filepath.Base(path)
	ext := filepath.Ext(id)
	if ext != "" {
		id = id[:len(id)-len(ext)]
	}

	return api.PluginMetadata{
		ID:      id,
		Name:    id,
		Version: "1.0.0",
		Location: api.PluginLocation{
			Type: "local",
			Path: path,
		},
		Timestamp: time.Now(),
	}, nil
}

// sendPluginEvent 发送插件事件
func (d *FileSystemDiscoverer) sendPluginEvent(event api.PluginEvent) {
	select {
	case d.eventCh <- event:
		// 事件已发送
	default:
		d.logger.Warn("事件通道已满，丢弃事件", "type", event.Type, "plugin_id", event.PluginID)
	}
}

// WatchPlugins 监听插件变化
func (d *FileSystemDiscoverer) WatchPlugins(ctx context.Context) (<-chan api.PluginEvent, error) {
	return d.eventCh, nil
}

// Close 关闭发现器
func (d *FileSystemDiscoverer) Close() error {
	d.cancel()
	close(d.eventCh)
	return nil
}

// CompositeDiscoverer 复合插件发现器
// 组合多个插件发现器
type CompositeDiscoverer struct {
	// 子发现器
	discoverers []PluginDiscoverer

	// 日志记录器
	logger hclog.Logger

	// 事件通道
	eventCh chan api.PluginEvent

	// 上下文
	ctx context.Context

	// 取消函数
	cancel context.CancelFunc
}

// NewCompositeDiscoverer 创建一个新的复合插件发现器
func NewCompositeDiscoverer(discoverers ...PluginDiscoverer) *CompositeDiscoverer {
	ctx, cancel := context.WithCancel(context.Background())

	return &CompositeDiscoverer{
		discoverers: discoverers,
		logger:      hclog.NewNullLogger(),
		eventCh:     make(chan api.PluginEvent, 100),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// DiscoverPlugins 发现插件
func (d *CompositeDiscoverer) DiscoverPlugins(ctx context.Context) ([]api.PluginMetadata, error) {
	var allPlugins []api.PluginMetadata
	var errs []error

	// 遍历所有发现器
	for _, discoverer := range d.discoverers {
		plugins, err := discoverer.DiscoverPlugins(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("发现器错误: %w", err))
			continue
		}

		allPlugins = append(allPlugins, plugins...)
	}

	// 如果有错误，返回复合错误
	if len(errs) > 0 {
		return allPlugins, fmt.Errorf("部分发现器失败: %v", errs)
	}

	return allPlugins, nil
}

// WatchPlugins 监听插件变化
func (d *CompositeDiscoverer) WatchPlugins(ctx context.Context) (<-chan api.PluginEvent, error) {
	// 启动所有发现器的监听
	for _, discoverer := range d.discoverers {
		eventCh, err := discoverer.WatchPlugins(ctx)
		if err != nil {
			d.logger.Error("启动发现器监听失败", "error", err)
			continue
		}

		// 转发事件
		go func(ch <-chan api.PluginEvent) {
			for event := range ch {
				select {
				case d.eventCh <- event:
					// 事件已转发
				case <-d.ctx.Done():
					return
				}
			}
		}(eventCh)
	}

	return d.eventCh, nil
}

// Close 关闭发现器
func (d *CompositeDiscoverer) Close() error {
	d.cancel()

	// 关闭所有发现器
	for _, discoverer := range d.discoverers {
		if err := discoverer.Close(); err != nil {
			d.logger.Error("关闭发现器失败", "error", err)
		}
	}

	close(d.eventCh)
	return nil
}
