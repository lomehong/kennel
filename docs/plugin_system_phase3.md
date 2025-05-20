# 插件系统优化 - 第三阶段

## 1. 概述

本文档描述了插件系统优化的第三阶段工作，包括增强插件SDK、创建插件测试框架、实现通信机制和文档生成器等。

## 2. 主要工作

### 2.1 增强插件SDK

在这个阶段，我们增强了插件SDK，提供了更多的辅助函数和工具，简化插件开发流程：

1. **配置管理**：实现了配置加载、保存和访问功能，支持YAML和JSON格式。
2. **调试支持**：添加了调试服务器，提供健康检查、状态查询和性能分析功能。
3. **工具函数**：提供了文件操作、日志记录、重试机制等常用工具函数。

### 2.2 创建插件测试框架

为了方便插件的测试，我们创建了插件测试框架：

1. **单元测试**：提供了插件单元测试的辅助工具，支持模拟插件和断言功能。
2. **集成测试**：实现了插件集成测试的辅助工具，支持插件管理器的测试。
3. **端到端测试**：提供了插件端到端测试的辅助工具，支持插件进程的启动和停止。

### 2.3 实现通信机制

为了支持插件之间的通信，我们实现了多种通信机制：

1. **事件总线**：实现了事件发布和订阅功能，支持插件之间的事件通信。
2. **HTTP通信**：实现了基于HTTP的通信机制，支持RESTful API。
3. **WebSocket通信**：实现了基于WebSocket的通信机制，支持双向通信。
4. **通信工厂**：实现了通信工厂，支持创建不同类型的通信实例。

### 2.4 实现文档生成器

为了方便插件的文档生成，我们实现了文档生成器：

1. **插件列表文档**：生成插件列表文档，包括插件的基本信息和依赖关系。
2. **插件详情文档**：生成插件详情文档，包括插件的详细信息和配置说明。
3. **依赖关系图**：生成插件依赖关系图，直观展示插件之间的依赖关系。

### 2.5 更新插件管理器

为了支持新的功能，我们更新了插件管理器：

1. **集成通信工厂**：集成通信工厂，支持创建不同类型的通信实例。
2. **集成文档生成器**：集成文档生成器，支持生成插件文档。
3. **提供辅助方法**：提供了获取各种组件的辅助方法，方便使用。

### 2.6 创建示例插件

为了展示如何使用新的插件SDK，我们创建了示例插件：

1. **Hello插件**：实现了一个简单的Hello插件，展示了插件的基本功能。
2. **配置示例**：提供了插件配置的示例，展示了如何配置插件。
3. **测试示例**：提供了插件测试的示例，展示了如何测试插件。

## 3. 技术细节

### 3.1 插件SDK

插件SDK提供了以下主要功能：

```go
// 配置管理
config, err := sdk.NewConfigManager("my-plugin", logger)
config.Load()
config.Set("key", "value")
value := config.GetString("key")
config.Save()

// 调试支持
debugServer := sdk.NewDebugServer("my-plugin", logger, sdk.WithDebugPort(8080), sdk.WithDebugEnabled(true))
debugServer.Start()
debugServer.RegisterHandler("/custom", customHandler)
debugServer.Stop()

// 工具函数
sdk.LoadConfig("config.yaml", &config)
sdk.SaveConfig("config.yaml", config)
sdk.CreateLogger("my-plugin", "info", "logs/my-plugin.log")
sdk.RetryWithBackoff(ctx, fn, 3, 1*time.Second, 10*time.Second)
```

### 3.2 插件测试框架

插件测试框架提供了以下主要功能：

```go
// 单元测试
suite := testing.NewPluginTestSuite(t, plugin)
suite.SetConfig("key", "value")
suite.Run(func(s *testing.TestSuite) {
    // 测试代码
})

// 集成测试
suite := testing.NewManagerTestSuite(t)
suite.AddMockPlugin("my-plugin")
suite.Run(func(s *testing.ManagerTestSuite) {
    // 测试代码
})

// 端到端测试
suite := testing.NewIntegrationTestSuite(t)
suite.BuildPlugin("my-plugin", "src", "bin")
suite.StartPluginProcess("my-plugin", "bin")
suite.Run(func(s *testing.IntegrationTestSuite) {
    // 测试代码
})
```

### 3.3 通信机制

通信机制提供了以下主要功能：

```go
// 创建通信
comm, err := manager.CreateCommunication(sdk.ProtocolHTTP, map[string]interface{}{
    "address":  "localhost:8080",
    "is_server": true,
})

// 注册服务
comm.RegisterService(service)

// 获取服务
service, err := comm.GetService("service-name")

// 发送消息
comm.SendMessage("target", message)

// 订阅主题
comm.Subscribe("topic", func(message interface{}) {
    // 处理消息
})

// 发布消息
comm.Publish("topic", message)
```

### 3.4 文档生成器

文档生成器提供了以下主要功能：

```go
// 生成插件文档
manager.GeneratePluginDocs("docs")

// 生成插件依赖关系图
manager.GeneratePluginDiagram("docs/dependencies.dot")
```

## 4. 使用示例

### 4.1 创建插件

```go
package main

import (
    "context"
    "os"
    "time"

    "github.com/hashicorp/go-hclog"
    "github.com/lomehong/kennel/pkg/plugin/sdk"
)

func main() {
    // 创建日志记录器
    logger := hclog.New(&hclog.LoggerOptions{
        Name:   "my-plugin",
        Level:  hclog.LevelFromString("info"),
        Output: os.Stdout,
    })

    // 使用构建器创建插件
    plugin := sdk.NewPluginBuilder("my-plugin").
        WithName("我的插件").
        WithVersion("1.0.0").
        WithDescription("这是一个示例插件").
        WithAuthor("示例作者").
        WithLicense("MIT").
        WithTag("示例").
        WithCapability("example").
        WithLogger(logger).
        WithInitFunc(func(ctx context.Context, config sdk.PluginConfig) error {
            logger.Info("初始化插件")
            return nil
        }).
        WithStartFunc(func(ctx context.Context) error {
            logger.Info("启动插件")
            return nil
        }).
        WithStopFunc(func(ctx context.Context) error {
            logger.Info("停止插件")
            return nil
        }).
        Build()

    // 创建插件运行器配置
    config := sdk.DefaultRunnerConfig()
    config.PluginID = "my-plugin"
    config.LogLevel = "info"

    // 运行插件
    if err := sdk.Run(plugin, config); err != nil {
        fmt.Fprintf(os.Stderr, "运行插件失败: %v\n", err)
        os.Exit(1)
    }
}
```

### 4.2 测试插件

```go
func TestMyPlugin(t *testing.T) {
    // 创建日志记录器
    logger := hclog.NewNullLogger()

    // 创建插件
    plugin := NewMyPlugin(logger)

    // 创建测试套件
    suite := testing.NewPluginTestSuite(t, plugin)

    // 设置配置
    suite.SetConfig("key", "value")

    // 运行测试
    suite.Run(func(s *testing.TestSuite) {
        // 检查插件信息
        info := plugin.GetInfo()
        assert.Equal(t, "my-plugin", info.ID)

        // 执行健康检查
        health, err := plugin.HealthCheck(context.Background())
        assert.NoError(t, err)
        assert.Equal(t, "healthy", health.Status)
    })
}
```

### 4.3 使用通信

```go
// 创建通信
comm, err := manager.CreateCommunication(sdk.ProtocolHTTP, map[string]interface{}{
    "address":  "localhost:8080",
    "is_server": true,
})

// 注册服务
comm.RegisterService(&MyService{})

// 订阅主题
comm.Subscribe("events", func(message interface{}) {
    // 处理消息
    fmt.Printf("收到消息: %v\n", message)
})

// 发布消息
comm.Publish("events", map[string]interface{}{
    "type": "notification",
    "content": "Hello, World!",
})
```

## 5. 下一步工作

1. **完善插件安全性**：实现插件签名验证、权限控制和沙箱机制。
2. **优化插件性能**：减少插件通信开销、优化资源使用和实现缓存机制。
3. **增强插件管理**：实现插件版本管理、升级机制和兼容性检查。
4. **改进插件发现**：支持远程插件仓库、插件搜索和自动更新。
5. **集成到主框架**：更新主应用程序以使用新的插件系统，提供向后兼容层。

## 6. 总结

在这个阶段，我们完成了插件系统优化的第三阶段工作，包括增强插件SDK、创建插件测试框架、实现通信机制和文档生成器等。这些工作为插件系统提供了更好的开发体验、更完善的测试支持和更丰富的功能。下一步，我们将继续完善插件系统，提高其安全性、性能和可管理性。
