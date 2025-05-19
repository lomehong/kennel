package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin"
)

// 定义隔离级别常量（如果pkg/plugin包中没有定义）
const (
	IsolationLevelNone      = "none"
	IsolationLevelProcess   = "process"
	IsolationLevelContainer = "container"
)

// PluginRegistry 插件注册表，管理可用的插件
type PluginRegistry struct {
	// 已注册的插件
	plugins map[string]*plugin.PluginConfig

	// 插件目录
	pluginDir string

	// 日志记录器
	logger hclog.Logger

	// 互斥锁
	mu sync.RWMutex

	// 上次扫描时间
	lastScanTime time.Time

	// 自动发现配置
	autoDiscover bool
	scanInterval time.Duration
}

// NewPluginRegistry 创建一个新的插件注册表
func NewPluginRegistry(logger hclog.Logger, pluginDir string) *PluginRegistry {
	return &PluginRegistry{
		plugins:      make(map[string]*plugin.PluginConfig),
		pluginDir:    pluginDir,
		logger:       logger.Named("plugin-registry"),
		autoDiscover: true,
		scanInterval: 60 * time.Second,
	}
}

// SetAutoDiscover 设置是否自动发现插件
func (pr *PluginRegistry) SetAutoDiscover(auto bool) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.autoDiscover = auto
}

// SetScanInterval 设置扫描间隔
func (pr *PluginRegistry) SetScanInterval(interval time.Duration) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.scanInterval = interval
}

// RegisterPlugin 注册插件
func (pr *PluginRegistry) RegisterPlugin(config *plugin.PluginConfig) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if config.ID == "" {
		return fmt.Errorf("插件ID不能为空")
	}

	if _, exists := pr.plugins[config.ID]; exists {
		return fmt.Errorf("插件 %s 已注册", config.ID)
	}

	pr.plugins[config.ID] = config
	pr.logger.Info("插件已注册", "id", config.ID, "name", config.Name)
	return nil
}

// UnregisterPlugin 注销插件
func (pr *PluginRegistry) UnregisterPlugin(id string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.plugins[id]; !exists {
		return fmt.Errorf("插件 %s 未注册", id)
	}

	delete(pr.plugins, id)
	pr.logger.Info("插件已注销", "id", id)
	return nil
}

// GetPlugin 获取插件配置
func (pr *PluginRegistry) GetPlugin(id string) (*plugin.PluginConfig, bool) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	config, exists := pr.plugins[id]
	return config, exists
}

// ListPlugins 列出所有已注册的插件
func (pr *PluginRegistry) ListPlugins() []*plugin.PluginConfig {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	plugins := make([]*plugin.PluginConfig, 0, len(pr.plugins))
	for _, config := range pr.plugins {
		plugins = append(plugins, config)
	}
	return plugins
}

// DiscoverPlugins 发现插件
func (pr *PluginRegistry) DiscoverPlugins() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.logger.Info("开始发现插件", "dir", pr.pluginDir)
	pr.lastScanTime = time.Now()

	// 检查插件目录是否存在
	if _, err := os.Stat(pr.pluginDir); os.IsNotExist(err) {
		pr.logger.Warn("插件目录不存在", "dir", pr.pluginDir)
		return nil
	}

	// 读取插件目录
	entries, err := os.ReadDir(pr.pluginDir)
	if err != nil {
		pr.logger.Error("读取插件目录失败", "error", err)
		return fmt.Errorf("读取插件目录失败: %w", err)
	}

	// 遍历目录
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginID := entry.Name()
		pluginPath := filepath.Join(pr.pluginDir, pluginID)

		// 检查是否已注册
		if _, exists := pr.plugins[pluginID]; exists {
			continue
		}

		// 创建插件配置
		config := &plugin.PluginConfig{
			ID:        pluginID,
			Name:      pluginID,
			Version:   "1.0.0",
			Path:      pluginID,
			AutoStart: false,
			Enabled:   true,
		}

		// 注册插件
		pr.plugins[pluginID] = config
		pr.logger.Info("发现新插件", "id", pluginID, "path", pluginPath)
	}

	pr.logger.Info("插件发现完成", "count", len(pr.plugins))
	return nil
}

// StartDiscovery 启动插件发现
func (pr *PluginRegistry) StartDiscovery() {
	if !pr.autoDiscover {
		return
	}

	go func() {
		ticker := time.NewTicker(pr.scanInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := pr.DiscoverPlugins(); err != nil {
				pr.logger.Error("插件发现失败", "error", err)
			}
		}
	}()
}
