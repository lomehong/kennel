package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/hashicorp/go-hclog"
	hashiPlugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// PluginLoader 插件加载器
type PluginLoader struct {
	logger        hclog.Logger
	pluginsDir    string
	loadedPlugins map[string]*LoadedPlugin
	mu            sync.RWMutex
}

// LoadedPlugin 已加载的插件
type LoadedPlugin struct {
	Metadata  PluginMetadata
	Instance  interface{}
	Client    *hashiPlugin.Client
	RawPlugin *plugin.Plugin
}

// NewPluginLoader 创建插件加载器
func NewPluginLoader(pluginsDir string, logger hclog.Logger) *PluginLoader {
	return &PluginLoader{
		logger:        logger.Named("plugin-loader"),
		pluginsDir:    pluginsDir,
		loadedPlugins: make(map[string]*LoadedPlugin),
	}
}

// LoadPluginMetadata 加载插件元数据
func LoadPluginMetadata(path string) (PluginMetadata, error) {
	// 读取元数据文件
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("读取元数据文件失败: %w", err)
	}

	// 解析元数据
	var metadata PluginMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return PluginMetadata{}, fmt.Errorf("解析元数据失败: %w", err)
	}

	// 验证元数据
	if metadata.ID == "" {
		return PluginMetadata{}, fmt.Errorf("插件ID不能为空")
	}
	if metadata.Name == "" {
		return PluginMetadata{}, fmt.Errorf("插件名称不能为空")
	}
	if metadata.Version == "" {
		return PluginMetadata{}, fmt.Errorf("插件版本不能为空")
	}
	if metadata.EntryPoint == "" {
		return PluginMetadata{}, fmt.Errorf("插件入口点不能为空")
	}

	return metadata, nil
}

// SavePluginMetadata 保存插件元数据
func SavePluginMetadata(metadata PluginMetadata, path string) error {
	// 序列化元数据
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	// 写入元数据文件
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入元数据文件失败: %w", err)
	}

	return nil
}

// ScanPluginsDir 扫描插件目录
func (pl *PluginLoader) ScanPluginsDir() ([]PluginMetadata, error) {
	pl.logger.Info("扫描插件目录", "dir", pl.pluginsDir)

	// 确保插件目录存在
	if err := os.MkdirAll(pl.pluginsDir, 0755); err != nil {
		return nil, fmt.Errorf("创建插件目录失败: %w", err)
	}

	// 读取插件目录
	entries, err := os.ReadDir(pl.pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("读取插件目录失败: %w", err)
	}

	var metadataList []PluginMetadata

	// 遍历目录项
	for _, entry := range entries {
		// 跳过非目录
		if !entry.IsDir() {
			continue
		}

		// 构建插件目录路径
		pluginDir := filepath.Join(pl.pluginsDir, entry.Name())

		// 检查插件元数据文件
		metadataPath := filepath.Join(pluginDir, "metadata.json")
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			pl.logger.Warn("插件元数据文件不存在", "dir", pluginDir)
			continue
		}

		// 加载插件元数据
		metadata, err := LoadPluginMetadata(metadataPath)
		if err != nil {
			pl.logger.Error("加载插件元数据失败", "dir", pluginDir, "error", err)
			continue
		}

		// 设置插件路径
		metadata.Path = pluginDir

		// 添加到列表
		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

// LoadPlugin 加载插件
func (pl *PluginLoader) LoadPlugin(metadata PluginMetadata) (*LoadedPlugin, error) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	pl.logger.Info("加载插件", "id", metadata.ID, "name", metadata.Name, "version", metadata.Version)

	// 检查插件是否已加载
	if _, exists := pl.loadedPlugins[metadata.ID]; exists {
		return nil, fmt.Errorf("插件已加载: %s", metadata.ID)
	}

	// 构建插件路径
	pluginPath := filepath.Join(metadata.Path, metadata.EntryPoint)
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("插件文件不存在: %s", pluginPath)
	}

	var loadedPlugin *LoadedPlugin

	// 根据隔离级别加载插件
	switch metadata.IsolationLevel {
	case PluginIsolationLevelNone:
		// 直接加载插件
		rawPlugin, err := plugin.Open(pluginPath)
		if err != nil {
			return nil, fmt.Errorf("打开插件失败: %w", err)
		}

		// 查找插件符号
		symPlugin, err := rawPlugin.Lookup("Plugin")
		if err != nil {
			return nil, fmt.Errorf("查找插件符号失败: %w", err)
		}

		// 转换为插件实例
		instance, ok := symPlugin.(Module)
		if !ok {
			return nil, fmt.Errorf("插件符号类型错误")
		}

		loadedPlugin = &LoadedPlugin{
			Metadata:  metadata,
			Instance:  instance,
			RawPlugin: rawPlugin,
		}

	case PluginIsolationLevelBasic, PluginIsolationLevelStrict:
		// 使用Hashicorp插件系统加载插件
		client := hashiPlugin.NewClient(&hashiPlugin.ClientConfig{
			HandshakeConfig: hashiPlugin.HandshakeConfig{
				ProtocolVersion:  1,
				MagicCookieKey:   "APPFRAMEWORK_PLUGIN",
				MagicCookieValue: "appframework",
			},
			Plugins: map[string]hashiPlugin.Plugin{
				metadata.ID: &AppFrameworkPlugin{},
			},
			Cmd:              exec.Command(pluginPath),
			AllowedProtocols: []hashiPlugin.Protocol{hashiPlugin.ProtocolGRPC},
			Logger:           pl.logger.Named(metadata.ID),
		})

		// 连接到插件
		rpcClient, err := client.Client()
		if err != nil {
			return nil, fmt.Errorf("连接到插件失败: %w", err)
		}

		// 获取插件实例
		raw, err := rpcClient.Dispense(metadata.ID)
		if err != nil {
			client.Kill()
			return nil, fmt.Errorf("获取插件实例失败: %w", err)
		}

		// 转换为插件实例
		instance, ok := raw.(Module)
		if !ok {
			client.Kill()
			return nil, fmt.Errorf("插件实例类型错误")
		}

		loadedPlugin = &LoadedPlugin{
			Metadata: metadata,
			Instance: instance,
			Client:   client,
		}

	default:
		return nil, fmt.Errorf("不支持的隔离级别: %s", metadata.IsolationLevel)
	}

	// 存储已加载的插件
	pl.loadedPlugins[metadata.ID] = loadedPlugin

	pl.logger.Info("插件已加载", "id", metadata.ID)
	return loadedPlugin, nil
}

// UnloadPlugin 卸载插件
func (pl *PluginLoader) UnloadPlugin(id string) error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	pl.logger.Info("卸载插件", "id", id)

	// 获取已加载的插件
	loadedPlugin, exists := pl.loadedPlugins[id]
	if !exists {
		return fmt.Errorf("插件未加载: %s", id)
	}

	// 根据隔离级别卸载插件
	switch loadedPlugin.Metadata.IsolationLevel {
	case PluginIsolationLevelNone:
		// 无需特殊处理

	case PluginIsolationLevelBasic, PluginIsolationLevelStrict:
		// 杀死插件进程
		if loadedPlugin.Client != nil {
			loadedPlugin.Client.Kill()
		}
	}

	// 删除已加载的插件
	delete(pl.loadedPlugins, id)

	pl.logger.Info("插件已卸载", "id", id)
	return nil
}

// GetLoadedPlugin 获取已加载的插件
func (pl *PluginLoader) GetLoadedPlugin(id string) (*LoadedPlugin, bool) {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	loadedPlugin, exists := pl.loadedPlugins[id]
	return loadedPlugin, exists
}

// GetLoadedPlugins 获取所有已加载的插件
func (pl *PluginLoader) GetLoadedPlugins() map[string]*LoadedPlugin {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	// 复制插件映射
	plugins := make(map[string]*LoadedPlugin, len(pl.loadedPlugins))
	for id, plugin := range pl.loadedPlugins {
		plugins[id] = plugin
	}

	return plugins
}

// Close 关闭插件加载器
func (pl *PluginLoader) Close() error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	pl.logger.Info("关闭插件加载器")

	// 卸载所有插件
	for id, loadedPlugin := range pl.loadedPlugins {
		// 根据隔离级别卸载插件
		switch loadedPlugin.Metadata.IsolationLevel {
		case PluginIsolationLevelNone:
			// 无需特殊处理

		case PluginIsolationLevelBasic, PluginIsolationLevelStrict:
			// 杀死插件进程
			if loadedPlugin.Client != nil {
				loadedPlugin.Client.Kill()
			}
		}

		pl.logger.Info("插件已卸载", "id", id)
	}

	// 清空已加载的插件
	pl.loadedPlugins = make(map[string]*LoadedPlugin)

	return nil
}

// AppFrameworkPlugin Hashicorp插件实现
type AppFrameworkPlugin struct {
	hashiPlugin.Plugin
	Impl Module
}

// GRPCServer 实现GRPCPlugin接口
func (p *AppFrameworkPlugin) GRPCServer(broker *hashiPlugin.GRPCBroker, s *grpc.Server) error {
	// 注册gRPC服务
	// 在实际应用中，这里需要实现gRPC服务
	return nil
}

// GRPCClient 实现GRPCPlugin接口
func (p *AppFrameworkPlugin) GRPCClient(ctx context.Context, broker *hashiPlugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	// 创建gRPC客户端
	// 在实际应用中，这里需要创建gRPC客户端
	return p.Impl, nil
}
