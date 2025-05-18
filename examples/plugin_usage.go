package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lomehong/kennel/pkg/core"
	"github.com/lomehong/kennel/pkg/plugin"
)

// 本示例展示如何在AppFramework中使用增强的插件系统
func main() {
	// 创建应用程序实例
	app := core.NewApp("config.yaml")

	// 初始化应用程序
	if err := app.Init(); err != nil {
		fmt.Printf("初始化应用程序失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== 插件系统增强使用示例 ===")

	// 示例1: 创建和注册插件
	fmt.Println("\n=== 示例1: 创建和注册插件 ===")
	createAndRegisterPlugin(app)

	// 示例2: 加载和卸载插件
	fmt.Println("\n=== 示例2: 加载和卸载插件 ===")
	loadAndUnloadPlugin(app)

	// 示例3: 插件依赖管理
	fmt.Println("\n=== 示例3: 插件依赖管理 ===")
	pluginDependencyManagement(app)

	// 示例4: 插件通信
	fmt.Println("\n=== 示例4: 插件通信 ===")
	pluginCommunication(app)

	// 示例5: 插件沙箱和隔离
	fmt.Println("\n=== 示例5: 插件沙箱和隔离 ===")
	pluginSandboxAndIsolation(app)

	// 示例6: 插件生命周期管理
	fmt.Println("\n=== 示例6: 插件生命周期管理 ===")
	pluginLifecycleManagement(app)

	// 停止应用程序
	app.Stop()
	fmt.Println("\n应用程序已停止")
}

// 创建和注册插件
func createAndRegisterPlugin(app *core.App) {
	// 创建插件元数据
	metadata := plugin.PluginMetadata{
		ID:             "example-plugin",
		Name:           "Example Plugin",
		Version:        "1.0.0",
		Author:         "Example Author",
		Description:    "Example plugin for demonstration",
		Dependencies:   []string{},
		Tags:           []string{"example", "demo"},
		EntryPoint:     "example.so",
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

	// 创建插件目录
	pluginsDir := filepath.Join(".", "plugins")
	pluginDir := filepath.Join(pluginsDir, "example-plugin")
	os.MkdirAll(pluginDir, 0755)

	// 保存插件元数据
	metadataPath := filepath.Join(pluginDir, "metadata.json")
	err := plugin.SavePluginMetadata(metadata, metadataPath)
	if err != nil {
		fmt.Printf("保存插件元数据失败: %v\n", err)
		return
	}

	fmt.Println("插件元数据已保存到:", metadataPath)

	// 注册插件
	err = app.RegisterPlugin(metadata)
	if err != nil {
		fmt.Printf("注册插件失败: %v\n", err)
		return
	}

	fmt.Println("插件已注册:", metadata.ID)

	// 获取插件信息
	info, err := app.GetPluginInfo(metadata.ID)
	if err != nil {
		fmt.Printf("获取插件信息失败: %v\n", err)
		return
	}

	fmt.Printf("插件信息:\n")
	fmt.Printf("  - ID: %s\n", info.Metadata.ID)
	fmt.Printf("  - 名称: %s\n", info.Metadata.Name)
	fmt.Printf("  - 版本: %s\n", info.Metadata.Version)
	fmt.Printf("  - 作者: %s\n", info.Metadata.Author)
	fmt.Printf("  - 描述: %s\n", info.Metadata.Description)
	fmt.Printf("  - 状态: %s\n", info.State)
}

// 加载和卸载插件
func loadAndUnloadPlugin(app *core.App) {
	// 扫描插件目录
	plugins, err := app.ScanPlugins()
	if err != nil {
		fmt.Printf("扫描插件目录失败: %v\n", err)
		return
	}

	fmt.Printf("发现 %d 个插件\n", len(plugins))
	for i, metadata := range plugins {
		fmt.Printf("插件 #%d:\n", i+1)
		fmt.Printf("  - ID: %s\n", metadata.ID)
		fmt.Printf("  - 名称: %s\n", metadata.Name)
		fmt.Printf("  - 版本: %s\n", metadata.Version)
	}

	// 加载插件
	for _, metadata := range plugins {
		err := app.LoadPlugin(metadata.ID)
		if err != nil {
			fmt.Printf("加载插件失败: %v\n", err)
			continue
		}
		fmt.Printf("插件已加载: %s\n", metadata.ID)
	}

	// 获取已加载的插件
	loadedPlugins := app.GetLoadedPlugins()
	fmt.Printf("已加载 %d 个插件\n", len(loadedPlugins))
	for id := range loadedPlugins {
		fmt.Printf("  - %s\n", id)
	}

	// 卸载插件
	for id := range loadedPlugins {
		err := app.UnloadPlugin(id)
		if err != nil {
			fmt.Printf("卸载插件失败: %v\n", err)
			continue
		}
		fmt.Printf("插件已卸载: %s\n", id)
	}
}

// 插件依赖管理
func pluginDependencyManagement(app *core.App) {
	// 创建插件元数据
	metadata1 := plugin.PluginMetadata{
		ID:             "base-plugin",
		Name:           "Base Plugin",
		Version:        "1.0.0",
		Author:         "Example Author",
		Description:    "Base plugin",
		Dependencies:   []string{},
		Tags:           []string{"base"},
		EntryPoint:     "base.so",
		IsolationLevel: plugin.PluginIsolationNone,
		Enabled:        true,
	}

	metadata2 := plugin.PluginMetadata{
		ID:             "dependent-plugin",
		Name:           "Dependent Plugin",
		Version:        "1.0.0",
		Author:         "Example Author",
		Description:    "Dependent plugin",
		Dependencies:   []string{"base-plugin"},
		Tags:           []string{"dependent"},
		EntryPoint:     "dependent.so",
		IsolationLevel: plugin.PluginIsolationNone,
		Enabled:        true,
	}

	// 注册插件
	app.RegisterPlugin(metadata1)
	app.RegisterPlugin(metadata2)

	// 获取插件加载顺序
	loadOrder, err := app.GetPluginLoadOrder()
	if err != nil {
		fmt.Printf("获取插件加载顺序失败: %v\n", err)
		return
	}

	fmt.Println("插件加载顺序:")
	for i, id := range loadOrder {
		fmt.Printf("  %d. %s\n", i+1, id)
	}

	// 获取插件卸载顺序
	unloadOrder, err := app.GetPluginUnloadOrder()
	if err != nil {
		fmt.Printf("获取插件卸载顺序失败: %v\n", err)
		return
	}

	fmt.Println("插件卸载顺序:")
	for i, id := range unloadOrder {
		fmt.Printf("  %d. %s\n", i+1, id)
	}

	// 检查插件依赖
	ok, missing, err := app.CheckPluginDependencies("dependent-plugin")
	if err != nil {
		fmt.Printf("检查插件依赖失败: %v\n", err)
		return
	}

	if ok {
		fmt.Println("插件依赖检查通过")
	} else {
		fmt.Printf("插件依赖检查失败，缺少依赖: %v\n", missing)
	}

	// 获取依赖此插件的插件
	dependents, err := app.GetPluginDependents("base-plugin")
	if err != nil {
		fmt.Printf("获取依赖此插件的插件失败: %v\n", err)
		return
	}

	fmt.Printf("依赖 base-plugin 的插件: %v\n", dependents)
}

// 插件通信
func pluginCommunication(app *core.App) {
	// 创建消息
	msg := plugin.Message{
		ID:          "msg-1",
		Type:        plugin.MessageTypeRequest,
		Source:      "app",
		Destination: "example-plugin",
		Topic:       "example.request",
		Payload: map[string]interface{}{
			"action": "getData",
			"params": map[string]interface{}{
				"id": 123,
			},
		},
	}

	// 发送消息
	response, err := app.SendPluginMessage(context.Background(), msg)
	if err != nil {
		fmt.Printf("发送消息失败: %v\n", err)
		return
	}

	fmt.Println("收到响应:")
	fmt.Printf("  - ID: %s\n", response.ID)
	fmt.Printf("  - 类型: %s\n", response.Type)
	fmt.Printf("  - 源: %s\n", response.Source)
	fmt.Printf("  - 目标: %s\n", response.Destination)
	fmt.Printf("  - 主题: %s\n", response.Topic)
	fmt.Printf("  - 负载: %v\n", response.Payload)

	// 发布事件
	err = app.PublishPluginEvent(context.Background(), "example.event", map[string]interface{}{
		"event_type": "data_updated",
		"data": map[string]interface{}{
			"id":        123,
			"name":      "Example",
			"timestamp": time.Now().Unix(),
		},
	})
	if err != nil {
		fmt.Printf("发布事件失败: %v\n", err)
		return
	}

	fmt.Println("事件已发布")

	// 订阅主题
	err = app.SubscribePluginTopic("app", "example.event")
	if err != nil {
		fmt.Printf("订阅主题失败: %v\n", err)
		return
	}

	fmt.Println("已订阅主题: example.event")

	// 获取订阅
	subscriptions, err := app.GetPluginSubscriptions("app")
	if err != nil {
		fmt.Printf("获取订阅失败: %v\n", err)
		return
	}

	fmt.Println("订阅:")
	for _, topic := range subscriptions {
		fmt.Printf("  - %s\n", topic)
	}

	// 取消订阅
	err = app.UnsubscribePluginTopic("app", "example.event")
	if err != nil {
		fmt.Printf("取消订阅失败: %v\n", err)
		return
	}

	fmt.Println("已取消订阅主题: example.event")
}

// 插件沙箱和隔离
func pluginSandboxAndIsolation(app *core.App) {
	// 创建插件元数据
	metadata := plugin.PluginMetadata{
		ID:             "isolated-plugin",
		Name:           "Isolated Plugin",
		Version:        "1.0.0",
		Author:         "Example Author",
		Description:    "Isolated plugin for demonstration",
		Dependencies:   []string{},
		Tags:           []string{"isolated"},
		EntryPoint:     "isolated.so",
		IsolationLevel: plugin.PluginIsolationProcess,
		Enabled:        true,
		Config: map[string]interface{}{
			"resource_limits": map[string]interface{}{
				"cpu":    1.0,
				"memory": 1024 * 1024 * 1024, // 1GB
			},
		},
	}

	// 注册插件
	err := app.RegisterPlugin(metadata)
	if err != nil {
		fmt.Printf("注册插件失败: %v\n", err)
		return
	}

	fmt.Println("已注册隔离插件:", metadata.ID)

	// 获取插件沙箱配置
	config, err := app.GetPluginSandboxConfig(metadata.ID)
	if err != nil {
		fmt.Printf("获取插件沙箱配置失败: %v\n", err)
		return
	}

	fmt.Println("插件沙箱配置:")
	fmt.Printf("  - 隔离级别: %s\n", config.IsolationLevel)
	fmt.Printf("  - CPU限制: %v\n", config.ResourceLimits.CPU)
	fmt.Printf("  - 内存限制: %v\n", config.ResourceLimits.Memory)

	// 启动插件沙箱
	err = app.StartPluginSandbox(metadata.ID)
	if err != nil {
		fmt.Printf("启动插件沙箱失败: %v\n", err)
		return
	}

	fmt.Println("插件沙箱已启动:", metadata.ID)

	// 获取插件沙箱状态
	status, err := app.GetPluginSandboxStatus(metadata.ID)
	if err != nil {
		fmt.Printf("获取插件沙箱状态失败: %v\n", err)
		return
	}

	fmt.Println("插件沙箱状态:")
	fmt.Printf("  - 状态: %s\n", status.State)
	fmt.Printf("  - 健康: %v\n", status.Healthy)
	fmt.Printf("  - 最后活动时间: %s\n", status.LastActivity.Format(time.RFC3339))

	// 停止插件沙箱
	err = app.StopPluginSandbox(metadata.ID)
	if err != nil {
		fmt.Printf("停止插件沙箱失败: %v\n", err)
		return
	}

	fmt.Println("插件沙箱已停止:", metadata.ID)
}

// 插件生命周期管理
func pluginLifecycleManagement(app *core.App) {
	// 获取所有插件
	plugins := app.GetAllPlugins()
	fmt.Printf("共有 %d 个插件\n", len(plugins))

	// 启动所有插件
	for _, metadata := range plugins {
		err := app.StartPlugin(metadata.ID)
		if err != nil {
			fmt.Printf("启动插件失败: %v\n", err)
			continue
		}
		fmt.Printf("插件已启动: %s\n", metadata.ID)
	}

	// 获取插件状态
	for _, metadata := range plugins {
		status, err := app.GetPluginStatus(metadata.ID)
		if err != nil {
			fmt.Printf("获取插件状态失败: %v\n", err)
			continue
		}
		fmt.Printf("插件 %s 状态: %s\n", metadata.ID, status)
	}

	// 重新加载插件
	for _, metadata := range plugins {
		err := app.ReloadPlugin(metadata.ID)
		if err != nil {
			fmt.Printf("重新加载插件失败: %v\n", err)
			continue
		}
		fmt.Printf("插件已重新加载: %s\n", metadata.ID)
	}

	// 停止所有插件
	for _, metadata := range plugins {
		err := app.StopPlugin(metadata.ID)
		if err != nil {
			fmt.Printf("停止插件失败: %v\n", err)
			continue
		}
		fmt.Printf("插件已停止: %s\n", metadata.ID)
	}

	// 启用/禁用插件
	for _, metadata := range plugins {
		// 禁用插件
		err := app.DisablePlugin(metadata.ID)
		if err != nil {
			fmt.Printf("禁用插件失败: %v\n", err)
			continue
		}
		fmt.Printf("插件已禁用: %s\n", metadata.ID)

		// 启用插件
		err = app.EnablePlugin(metadata.ID)
		if err != nil {
			fmt.Printf("启用插件失败: %v\n", err)
			continue
		}
		fmt.Printf("插件已启用: %s\n", metadata.ID)
	}
}
