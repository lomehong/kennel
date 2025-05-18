package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	hplugin "github.com/hashicorp/go-plugin"
	pluginLib "github.com/lomehong/kennel/pkg/plugin"
)

// 全局插件管理器实例
var globalPluginManager *PluginManager

// GetPluginManager 获取全局插件管理器实例
func GetPluginManager() *PluginManager {
	return globalPluginManager
}

// PluginManager 管理插件的加载和生命周期
type PluginManager struct {
	// 插件目录
	pluginDir string

	// 已加载的插件
	plugins map[string]*loadedPlugin

	// 互斥锁，用于保护plugins map
	mu sync.RWMutex

	// 日志
	logger hclog.Logger

	// 通讯管理器
	commManager *CommManager
}

// loadedPlugin 表示已加载的插件
type loadedPlugin struct {
	// 插件名称
	name string

	// 插件路径
	path string

	// 插件客户端
	client *hplugin.Client

	// 插件实例
	instance pluginLib.Module

	// 插件信息
	info pluginLib.ModuleInfo
}

// NewPluginManager 创建一个新的插件管理器
func NewPluginManager(pluginDir string) *PluginManager {
	// 创建日志
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin-manager",
		Output: os.Stdout,
		Level:  hclog.Info,
	})

	// 创建插件管理器
	pm := &PluginManager{
		pluginDir: pluginDir,
		plugins:   make(map[string]*loadedPlugin),
		logger:    logger,
	}

	// 设置全局插件管理器实例
	globalPluginManager = pm

	return pm
}

// LoadPlugin 加载指定路径的插件
func (pm *PluginManager) LoadPlugin(pluginPath string) error {
	pm.logger.Info("加载插件", "path", pluginPath)

	// 创建插件客户端
	client := hplugin.NewClient(&hplugin.ClientConfig{
		HandshakeConfig: hplugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "APPFRAMEWORK_PLUGIN",
			MagicCookieValue: "appframework",
		},
		Plugins: pluginLib.PluginMap,
		Cmd:     exec.Command(pluginPath),
		Logger:  pm.logger,
		AllowedProtocols: []hplugin.Protocol{
			hplugin.ProtocolGRPC,
		},
		// 设置启动超时，避免插件启动时间过长
		StartTimeout: 10 * time.Second,
	})

	// 添加超时控制，避免连接时间过长
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 连接到插件
	connCh := make(chan struct {
		client *hplugin.Client
		rpc    hplugin.ClientProtocol
		err    error
	}, 1)

	go func() {
		rpcClient, err := client.Client()
		connCh <- struct {
			client *hplugin.Client
			rpc    hplugin.ClientProtocol
			err    error
		}{client, rpcClient, err}
	}()

	// 等待连接或超时
	var rpcClient hplugin.ClientProtocol
	select {
	case <-ctx.Done():
		client.Kill()
		pm.logger.Error("连接插件超时")
		return fmt.Errorf("连接插件超时")
	case conn := <-connCh:
		if conn.err != nil {
			pm.logger.Error("无法连接到插件", "error", conn.err)
			return fmt.Errorf("无法连接到插件: %w", conn.err)
		}
		rpcClient = conn.rpc
	}

	// 获取插件实例
	raw, err := rpcClient.Dispense("module")
	if err != nil {
		pm.logger.Error("无法获取插件实例", "error", err)
		return fmt.Errorf("无法获取插件实例: %w", err)
	}

	// 类型断言
	instance, ok := raw.(pluginLib.Module)
	if !ok {
		pm.logger.Error("插件不是有效的Module类型")
		return fmt.Errorf("插件不是有效的Module类型")
	}

	// 获取插件信息
	info := instance.GetInfo()

	// 存储插件
	pm.mu.Lock()
	pm.plugins[info.Name] = &loadedPlugin{
		name:     info.Name,
		path:     pluginPath,
		client:   client,
		instance: instance,
		info:     info,
	}
	pm.mu.Unlock()

	pm.logger.Info("插件加载成功", "name", info.Name, "version", info.Version)
	return nil
}

// LoadPluginsFromDir 从目录加载所有插件
func (pm *PluginManager) LoadPluginsFromDir() error {
	pm.logger.Info("从目录加载插件", "dir", pm.pluginDir)

	// 检查目录是否存在
	if _, err := os.Stat(pm.pluginDir); os.IsNotExist(err) {
		pm.logger.Warn("插件目录不存在", "dir", pm.pluginDir)
		return nil
	}

	// 遍历目录
	entries, err := os.ReadDir(pm.pluginDir)
	if err != nil {
		pm.logger.Error("无法读取插件目录", "error", err)
		return fmt.Errorf("无法读取插件目录: %w", err)
	}

	// 创建一个工作池，限制并发加载的插件数量
	// 避免同时启动太多进程导致系统负载过高
	const maxConcurrent = 4
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	// 用于收集错误
	errCh := make(chan error, len(entries))

	// 遍历每个应用目录
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 获取应用目录
		appDir := filepath.Join(pm.pluginDir, entry.Name())

		// 查找可执行文件
		appFiles, err := os.ReadDir(appDir)
		if err != nil {
			pm.logger.Warn("无法读取应用目录", "dir", appDir, "error", err)
			continue
		}

		// 查找可执行文件
		for _, file := range appFiles {
			if file.IsDir() {
				continue
			}

			// 检查文件是否可执行
			path := filepath.Join(appDir, file.Name())
			info, err := os.Stat(path)
			if err != nil {
				pm.logger.Warn("无法获取文件信息", "path", path, "error", err)
				continue
			}

			// 在Windows上，我们检查文件扩展名是否为.exe
			// 在Unix系统上，我们检查文件是否有执行权限
			if filepath.Ext(path) == ".exe" || info.Mode()&0111 != 0 {
				// 并发加载插件
				wg.Add(1)
				go func(pluginPath string) {
					defer wg.Done()

					// 获取信号量，限制并发数
					sem <- struct{}{}
					defer func() { <-sem }()

					// 加载插件
					if err := pm.LoadPlugin(pluginPath); err != nil {
						pm.logger.Warn("加载插件失败", "path", pluginPath, "error", err)
						errCh <- fmt.Errorf("加载插件失败 %s: %w", pluginPath, err)
					}
				}(path)
			}
		}
	}

	// 等待所有插件加载完成
	wg.Wait()
	close(errCh)

	// 检查是否有错误
	var loadErrors []error
	for err := range errCh {
		loadErrors = append(loadErrors, err)
	}

	if len(loadErrors) > 0 {
		pm.logger.Warn("部分插件加载失败", "error_count", len(loadErrors))
		// 只返回第一个错误，其他错误已经记录在日志中
		return loadErrors[0]
	}

	return nil
}

// GetPlugin 获取指定名称的插件
func (pm *PluginManager) GetPlugin(name string) (pluginLib.Module, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	p, ok := pm.plugins[name]
	if !ok {
		return nil, false
	}

	return p.instance, true
}

// ListPlugins 列出所有已加载的插件
func (pm *PluginManager) ListPlugins() []pluginLib.ModuleInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]pluginLib.ModuleInfo, 0, len(pm.plugins))
	for _, p := range pm.plugins {
		result = append(result, p.info)
	}

	return result
}

// ClosePlugin 关闭指定名称的插件
func (pm *PluginManager) ClosePlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	p, ok := pm.plugins[name]
	if !ok {
		return fmt.Errorf("插件 %s 未加载", name)
	}

	// 关闭插件
	p.instance.Shutdown()
	p.client.Kill()

	// 从map中删除
	delete(pm.plugins, name)

	pm.logger.Info("插件已关闭", "name", name)
	return nil
}

// CloseAllPlugins 关闭所有插件，支持优雅终止
func (pm *PluginManager) CloseAllPlugins(ctx context.Context) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 如果没有插件，直接返回
	if len(pm.plugins) == 0 {
		pm.logger.Info("没有需要关闭的插件")
		return
	}

	// 创建一个WaitGroup，用于等待所有插件关闭
	var wg sync.WaitGroup

	// 创建一个通道，用于收集错误
	errCh := make(chan error, len(pm.plugins))

	// 并发关闭所有插件
	for name, p := range pm.plugins {
		wg.Add(1)
		go func(name string, plugin *loadedPlugin) {
			defer wg.Done()

			// 创建一个带超时的上下文
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			// 创建一个通道，用于等待Shutdown完成
			done := make(chan struct{})

			// 在后台执行Shutdown
			go func() {
				pm.logger.Debug("开始关闭插件", "name", name)
				err := plugin.instance.Shutdown()
				if err != nil {
					pm.logger.Error("插件关闭出错", "name", name, "error", err)
					errCh <- fmt.Errorf("插件 %s 关闭失败: %w", name, err)
				}
				close(done)
			}()

			// 等待Shutdown完成或超时
			select {
			case <-shutdownCtx.Done():
				pm.logger.Warn("插件关闭超时，强制终止", "name", name)
			case <-done:
				pm.logger.Debug("插件已正常关闭", "name", name)
			}

			// 无论如何，都要终止插件进程
			plugin.client.Kill()
			pm.logger.Info("插件已关闭", "name", name)
		}(name, p)
	}

	// 等待所有插件关闭
	wg.Wait()
	close(errCh)

	// 检查是否有错误
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		pm.logger.Warn("部分插件关闭失败", "error_count", len(errors))
	}

	// 清空map
	pm.plugins = make(map[string]*loadedPlugin)
}

// SetCommManager 设置通讯管理器
func (pm *PluginManager) SetCommManager(commManager *CommManager) {
	pm.commManager = commManager
}

// RouteMessage 将消息路由到对应的插件
func (pm *PluginManager) RouteMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error) {
	// 检查消息是否包含目标插件
	targetPlugin, ok := payload["plugin"].(string)
	if !ok {
		return nil, fmt.Errorf("消息缺少目标插件")
	}

	// 获取插件
	pm.mu.RLock()
	plugin, ok := pm.plugins[targetPlugin]
	pm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("插件 %s 未找到", targetPlugin)
	}

	// 将消息转发给插件
	pm.logger.Info("将消息路由到插件", "plugin", targetPlugin, "message_type", messageType)
	return plugin.instance.HandleMessage(messageType, messageID, timestamp, payload)
}
