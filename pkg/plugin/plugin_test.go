package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestBasePlugin(t *testing.T) {
	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建插件元数据
	metadata := PluginMetadata{
		ID:             "test-plugin",
		Name:           "Test Plugin",
		Version:        "1.0.0",
		Author:         "Test Author",
		Description:    "Test Description",
		Dependencies:   []string{},
		Tags:           []string{"test", "example"},
		Path:           "/path/to/plugin",
		EntryPoint:     "plugin.so",
		IsolationLevel: PluginIsolationNone,
		AutoStart:      true,
		AutoRestart:    false,
		Enabled:        true,
		Config:         map[string]interface{}{"key": "value"},
	}

	// 创建基础插件
	plugin := NewBasePlugin(metadata, logger)

	// 验证基本属性
	assert.Equal(t, "test-plugin", plugin.ID())
	assert.Equal(t, metadata, plugin.Metadata())
	assert.Equal(t, PluginStateCreated, plugin.GetState())
	assert.Nil(t, plugin.GetError())
	assert.False(t, plugin.IsRunning())

	// 初始化插件
	err := plugin.Init(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, PluginStateLoaded, plugin.GetState())

	// 启动插件
	err = plugin.Start(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, PluginStateStarted, plugin.GetState())
	assert.True(t, plugin.IsRunning())

	// 停止插件
	err = plugin.Stop(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, PluginStateStopped, plugin.GetState())
	assert.False(t, plugin.IsRunning())

	// 重新加载插件
	err = plugin.Reload(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, PluginStateLoaded, plugin.GetState())

	// 获取配置
	config := plugin.GetConfig()
	assert.Equal(t, "value", config["key"])

	// 设置配置
	err = plugin.SetConfig(map[string]interface{}{"key": "new-value"})
	assert.NoError(t, err)
	config = plugin.GetConfig()
	assert.Equal(t, "new-value", config["key"])

	// 获取接口
	_, err = plugin.GetInterface("test")
	assert.Error(t, err)

	// 注册接口
	plugin.registerInterface("test", "test-interface")
	iface, err := plugin.GetInterface("test")
	assert.NoError(t, err)
	assert.Equal(t, "test-interface", iface)

	// 获取资源使用情况
	usage := plugin.GetResourceUsage()
	assert.NotNil(t, usage)

	// 设置错误
	plugin.setError(assert.AnError)
	assert.Equal(t, PluginStateError, plugin.GetState())
	assert.Equal(t, assert.AnError, plugin.GetError())
}

func TestPluginDependencyManager(t *testing.T) {
	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建插件依赖管理器
	pdm := NewPluginDependencyManager(logger)

	// 创建插件元数据
	metadata1 := PluginMetadata{
		ID:           "plugin1",
		Name:         "Plugin 1",
		Version:      "1.0.0",
		Dependencies: []string{},
		Tags:         []string{"tag1"},
	}

	metadata2 := PluginMetadata{
		ID:           "plugin2",
		Name:         "Plugin 2",
		Version:      "1.0.0",
		Dependencies: []string{"plugin1"},
		Tags:         []string{"tag2"},
	}

	metadata3 := PluginMetadata{
		ID:           "plugin3",
		Name:         "Plugin 3",
		Version:      "1.0.0",
		Dependencies: []string{"plugin2"},
		Tags:         []string{"tag1", "tag2"},
	}

	// 添加插件
	err := pdm.AddPlugin(metadata1)
	assert.NoError(t, err)
	err = pdm.AddPlugin(metadata2)
	assert.NoError(t, err)
	err = pdm.AddPlugin(metadata3)
	assert.NoError(t, err)

	// 获取插件
	plugin, exists := pdm.GetPlugin("plugin1")
	assert.True(t, exists)
	assert.Equal(t, metadata1, plugin)

	// 获取所有插件
	plugins := pdm.GetPlugins()
	assert.Equal(t, 3, len(plugins))
	assert.Contains(t, plugins, "plugin1")
	assert.Contains(t, plugins, "plugin2")
	assert.Contains(t, plugins, "plugin3")

	// 获取依赖
	deps, err := pdm.GetDependencies("plugin2")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(deps))
	assert.Equal(t, "plugin1", deps[0])

	// 获取依赖此插件的插件
	deps, err = pdm.GetDependents("plugin2")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(deps))
	assert.Equal(t, "plugin3", deps[0])

	// 获取插件加载顺序
	order, err := pdm.GetLoadOrder()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(order))
	assert.Equal(t, "plugin1", order[0])
	assert.Equal(t, "plugin2", order[1])
	assert.Equal(t, "plugin3", order[2])

	// 获取插件卸载顺序
	order, err = pdm.GetUnloadOrder()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(order))
	assert.Equal(t, "plugin3", order[0])
	assert.Equal(t, "plugin2", order[1])
	assert.Equal(t, "plugin1", order[2])

	// 检查依赖
	ok, missing, err := pdm.CheckDependencies("plugin2")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Empty(t, missing)

	// 获取指定标签的插件
	tagPlugins := pdm.GetPluginsByTag("tag1")
	assert.Equal(t, 2, len(tagPlugins))
	assert.Contains(t, []string{tagPlugins[0].ID, tagPlugins[1].ID}, "plugin1")
	assert.Contains(t, []string{tagPlugins[0].ID, tagPlugins[1].ID}, "plugin3")

	// 获取插件依赖图
	graph := pdm.GetPluginGraph()
	assert.Equal(t, 3, len(graph))
	assert.Empty(t, graph["plugin1"])
	assert.Equal(t, 1, len(graph["plugin2"]))
	assert.Equal(t, "plugin1", graph["plugin2"][0])
	assert.Equal(t, 1, len(graph["plugin3"]))
	assert.Equal(t, "plugin2", graph["plugin3"][0])

	// 获取插件反向依赖图
	revGraph := pdm.GetPluginReverseGraph()
	assert.Equal(t, 3, len(revGraph))
	assert.Equal(t, 1, len(revGraph["plugin1"]))
	assert.Equal(t, "plugin2", revGraph["plugin1"][0])
	assert.Equal(t, 1, len(revGraph["plugin2"]))
	assert.Equal(t, "plugin3", revGraph["plugin2"][0])
	assert.Empty(t, revGraph["plugin3"])

	// 移除插件
	err = pdm.RemovePlugin("plugin3")
	assert.NoError(t, err)
	plugins = pdm.GetPlugins()
	assert.Equal(t, 2, len(plugins))
	assert.NotContains(t, plugins, "plugin3")

	// 尝试移除有依赖的插件
	err = pdm.RemovePlugin("plugin1")
	assert.Error(t, err)
	plugins = pdm.GetPlugins()
	assert.Equal(t, 2, len(plugins))
	assert.Contains(t, plugins, "plugin1")
}

func TestPluginCommunicator(t *testing.T) {
	// 创建日志记录器
	logger := hclog.NewNullLogger()

	// 创建插件管理器
	pluginManager := NewPluginManager(WithPluginManagerLogger(logger))

	// 创建插件通信器
	communicator := NewPluginCommunicator(logger, pluginManager)

	// 创建消息处理器
	handler1 := func(ctx context.Context, msg Message) (Message, error) {
		return Message{
			ID:      "response-1",
			Payload: map[string]interface{}{"response": "ok"},
		}, nil
	}

	handler2 := func(ctx context.Context, msg Message) (Message, error) {
		return Message{
			ID:      "response-2",
			Payload: map[string]interface{}{"response": "ok"},
		}, nil
	}

	// 注册消息处理器
	communicator.RegisterHandler("plugin1", handler1)
	communicator.RegisterHandler("plugin2", handler2)

	// 订阅主题
	communicator.Subscribe("plugin1", "topic1")
	communicator.Subscribe("plugin2", "topic1")
	communicator.Subscribe("plugin2", "topic2")

	// 获取订阅
	subs := communicator.GetSubscriptions("plugin1")
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, "topic1", subs[0])

	subs = communicator.GetSubscriptions("plugin2")
	assert.Equal(t, 2, len(subs))
	assert.Contains(t, subs, "topic1")
	assert.Contains(t, subs, "topic2")

	// 获取订阅者
	subscribers := communicator.GetSubscribers("topic1")
	assert.Equal(t, 2, len(subscribers))
	assert.Contains(t, subscribers, "plugin1")
	assert.Contains(t, subscribers, "plugin2")

	subscribers = communicator.GetSubscribers("topic2")
	assert.Equal(t, 1, len(subscribers))
	assert.Equal(t, "plugin2", subscribers[0])

	// 发送消息
	msg := Message{
		ID:          "request-1",
		Type:        MessageTypeRequest,
		Source:      "plugin1",
		Destination: "plugin2",
		Topic:       "test",
		Payload:     map[string]interface{}{"request": "test"},
	}

	response, err := communicator.SendMessage(context.Background(), msg)
	assert.NoError(t, err)
	assert.Equal(t, "response-2", response.ID)
	assert.Equal(t, MessageTypeResponse, response.Type)
	assert.Equal(t, "plugin2", response.Source)
	assert.Equal(t, "plugin1", response.Destination)
	assert.Equal(t, "ok", response.Payload["response"])

	// 发布事件
	err = communicator.PublishEvent(context.Background(), "plugin1", "topic1", map[string]interface{}{"event": "test"})
	assert.NoError(t, err)

	// 取消订阅
	communicator.Unsubscribe("plugin2", "topic1")
	subs = communicator.GetSubscriptions("plugin2")
	assert.Equal(t, 1, len(subs))
	assert.Equal(t, "topic2", subs[0])

	// 取消所有订阅
	communicator.UnsubscribeAll("plugin2")
	subs = communicator.GetSubscriptions("plugin2")
	assert.Empty(t, subs)

	// 注销消息处理器
	communicator.UnregisterHandler("plugin1")
	_, err = communicator.SendMessage(context.Background(), Message{
		ID:          "request-2",
		Type:        MessageTypeRequest,
		Source:      "plugin2",
		Destination: "plugin1",
		Topic:       "test",
		Payload:     map[string]interface{}{"request": "test"},
	})
	assert.Error(t, err)

	// 关闭通信器
	err = communicator.Close()
	assert.NoError(t, err)
}
