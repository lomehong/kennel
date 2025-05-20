# 插件开发指南

## 1. 概述

本文档提供了开发插件的详细指南，包括插件的创建、配置、测试和部署。

### 1.1 什么是插件？

插件是一个独立的模块，可以扩展应用程序的功能。插件通过标准接口与应用程序通信，可以在不修改应用程序核心代码的情况下添加新功能。

### 1.2 插件系统特点

- **模块化**: 将功能拆分为独立的插件，便于维护和扩展
- **松耦合**: 插件之间通过标准接口通信，减少依赖
- **可扩展**: 支持动态加载和卸载插件，无需修改核心代码
- **隔离性**: 插件运行在隔离环境中，避免影响主应用程序
- **依赖管理**: 支持插件之间的依赖关系管理
- **版本兼容**: 支持插件版本管理和兼容性检查

## 2. 插件开发环境

### 2.1 开发环境要求

- Go 1.16 或更高版本
- Git
- 编辑器（推荐 VS Code）
- 插件SDK

### 2.2 安装插件SDK

```bash
go get github.com/lomehong/kennel/pkg/plugin/sdk
```

### 2.3 开发工具

- **VS Code 插件**:
  - Go 插件
  - Go Test Explorer
  - Go Outline
  - Go Doc

- **命令行工具**:
  - `go build`: 编译插件
  - `go test`: 测试插件
  - `go mod`: 管理依赖

## 3. 创建插件

### 3.1 插件项目结构

推荐的插件项目结构：

```
my-plugin/
├── cmd/
│   └── main.go         # 插件入口点
├── pkg/
│   ├── api/            # 插件API
│   ├── service/        # 插件服务
│   └── utils/          # 工具函数
├── config/
│   └── config.yaml     # 插件配置
├── tests/              # 测试
├── go.mod              # Go模块文件
├── go.sum              # Go依赖校验文件
└── README.md           # 插件文档
```

### 3.2 插件基础代码

#### 3.2.1 使用构建器创建插件

```go
package main

import (
    "context"
    "fmt"
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
        WithHealthCheckFunc(func(ctx context.Context) (sdk.HealthStatus, error) {
            return sdk.HealthStatus{
                Status:      "healthy",
                Details:     make(map[string]interface{}),
                LastChecked: time.Now(),
            }, nil
        }).
        Build()

    // 创建插件运行器配置
    config := sdk.DefaultRunnerConfig()
    config.PluginID = "my-plugin"
    config.LogLevel = "info"
    config.LogFile = "logs/my-plugin.log"
    config.ShutdownTimeout = 30 * time.Second
    config.HealthCheckInterval = 30 * time.Second

    // 运行插件
    if err := sdk.Run(plugin, config); err != nil {
        fmt.Fprintf(os.Stderr, "运行插件失败: %v\n", err)
        os.Exit(1)
    }
}
```

#### 3.2.2 自定义插件实现

```go
package myplugin

import (
    "context"
    "time"

    "github.com/hashicorp/go-hclog"
    "github.com/lomehong/kennel/pkg/plugin/api"
    "github.com/lomehong/kennel/pkg/plugin/sdk"
)

// MyPlugin 自定义插件实现
type MyPlugin struct {
    // 基础插件
    *sdk.BasePlugin

    // 自定义字段
    config *Config
    service *Service
}

// Config 插件配置
type Config struct {
    Option1 string `json:"option1"`
    Option2 int    `json:"option2"`
}

// Service 插件服务
type Service struct {
    // 服务字段
}

// NewMyPlugin 创建一个新的插件
func NewMyPlugin(logger hclog.Logger) *MyPlugin {
    // 创建插件信息
    info := api.PluginInfo{
        ID:          "my-plugin",
        Name:        "我的插件",
        Version:     "1.0.0",
        Description: "这是一个示例插件",
        Author:      "示例作者",
        License:     "MIT",
        Tags:        []string{"示例"},
        Capabilities: map[string]bool{
            "example": true,
        },
    }

    // 创建基础插件
    basePlugin := sdk.NewBasePlugin(info, logger)

    // 创建插件
    return &MyPlugin{
        BasePlugin: basePlugin,
    }
}

// Init 初始化插件
func (p *MyPlugin) Init(ctx context.Context, config api.PluginConfig) error {
    // 调用基类初始化
    if err := p.BasePlugin.Init(ctx, config); err != nil {
        return err
    }

    p.GetLogger().Info("初始化插件")

    // 解析配置
    p.config = &Config{
        Option1: "default",
        Option2: 42,
    }

    if option1, ok := config.Settings["option1"].(string); ok {
        p.config.Option1 = option1
    }

    if option2, ok := config.Settings["option2"].(int); ok {
        p.config.Option2 = option2
    }

    // 创建服务
    p.service = &Service{}

    return nil
}

// Start 启动插件
func (p *MyPlugin) Start(ctx context.Context) error {
    // 调用基类启动
    if err := p.BasePlugin.Start(ctx); err != nil {
        return err
    }

    p.GetLogger().Info("启动插件")

    // 启动服务
    // ...

    return nil
}

// Stop 停止插件
func (p *MyPlugin) Stop(ctx context.Context) error {
    // 调用基类停止
    if err := p.BasePlugin.Stop(ctx); err != nil {
        return err
    }

    p.GetLogger().Info("停止插件")

    // 停止服务
    // ...

    return nil
}

// HealthCheck 执行健康检查
func (p *MyPlugin) HealthCheck(ctx context.Context) (api.HealthStatus, error) {
    // 检查服务健康状态
    // ...

    return api.HealthStatus{
        Status:      "healthy",
        Details:     make(map[string]interface{}),
        LastChecked: time.Now(),
    }, nil
}
```

### 3.3 插件配置

插件配置示例：

```yaml
# 插件配置
id: my-plugin
name: 我的插件
version: 1.0.0
enabled: true
log_level: info

# 隔离配置
isolation:
  level: basic
  resources:
    memory: 256000000  # 256MB
    cpu: 50            # 50% CPU
  timeout: 30s
  working_dir: ./data
  environment:
    DEBUG: "false"
    PLUGIN_MODE: "production"

# 插件特定配置
settings:
  option1: "custom value"
  option2: 100
```

### 3.4 插件通信

#### 3.4.1 注册服务

```go
// 创建服务
service := &MyService{}

// 注册服务
err := comm.RegisterService(service)
```

#### 3.4.2 获取服务

```go
// 获取服务
service, err := comm.GetService("other-plugin-service")
```

#### 3.4.3 发送消息

```go
// 发送消息
err := comm.SendMessage("target-plugin", map[string]interface{}{
    "action": "doSomething",
    "data": "some data",
})
```

#### 3.4.4 订阅主题

```go
// 订阅主题
err := comm.Subscribe("topic", func(message interface{}) {
    // 处理消息
    msg, ok := message.(map[string]interface{})
    if !ok {
        return
    }

    action, ok := msg["action"].(string)
    if !ok {
        return
    }

    switch action {
    case "doSomething":
        // 处理操作
    }
})
```

#### 3.4.5 发布消息

```go
// 发布消息
err := comm.Publish("topic", map[string]interface{}{
    "action": "doSomething",
    "data": "some data",
})
```

## 4. 插件测试

### 4.1 单元测试

```go
package myplugin

import (
    "context"
    "testing"
    "time"

    "github.com/hashicorp/go-hclog"
    "github.com/lomehong/kennel/pkg/plugin/api"
    "github.com/stretchr/testify/assert"
)

func TestMyPlugin(t *testing.T) {
    // 创建日志记录器
    logger := hclog.NewNullLogger()

    // 创建插件
    plugin := NewMyPlugin(logger)

    // 创建配置
    config := api.PluginConfig{
        ID:      "my-plugin",
        Enabled: true,
        Settings: map[string]interface{}{
            "option1": "test",
            "option2": 123,
        },
    }

    // 测试初始化
    err := plugin.Init(context.Background(), config)
    assert.NoError(t, err)
    assert.Equal(t, "test", plugin.config.Option1)
    assert.Equal(t, 123, plugin.config.Option2)

    // 测试启动
    err = plugin.Start(context.Background())
    assert.NoError(t, err)
    assert.Equal(t, api.PluginStateRunning, plugin.GetState())

    // 测试健康检查
    health, err := plugin.HealthCheck(context.Background())
    assert.NoError(t, err)
    assert.Equal(t, "healthy", health.Status)

    // 测试停止
    err = plugin.Stop(context.Background())
    assert.NoError(t, err)
    assert.Equal(t, api.PluginStateStopped, plugin.GetState())
}
```

### 4.2 集成测试

```go
package integration

import (
    "context"
    "testing"
    "time"

    "github.com/hashicorp/go-hclog"
    "github.com/lomehong/kennel/pkg/plugin"
    "github.com/lomehong/kennel/pkg/plugin/api"
    "github.com/stretchr/testify/assert"
)

func TestPluginIntegration(t *testing.T) {
    // 创建日志记录器
    logger := hclog.NewNullLogger()

    // 创建插件管理器
    manager := plugin.NewPluginManagerV3(logger, plugin.DefaultManagerConfigV3())

    // 启动插件管理器
    err := manager.Start()
    assert.NoError(t, err)
    defer manager.Stop()

    // 加载插件
    p, err := manager.LoadPlugin("my-plugin")
    assert.NoError(t, err)
    assert.NotNil(t, p)

    // 获取插件状态
    status, err := manager.GetPluginStatus("my-plugin")
    assert.NoError(t, err)
    assert.Equal(t, api.PluginStateInitialized, status.State)

    // 启动插件
    err = manager.StartPlugin("my-plugin")
    assert.NoError(t, err)

    // 获取插件状态
    status, err = manager.GetPluginStatus("my-plugin")
    assert.NoError(t, err)
    assert.Equal(t, api.PluginStateRunning, status.State)

    // 停止插件
    err = manager.StopPlugin("my-plugin")
    assert.NoError(t, err)

    // 获取插件状态
    status, err = manager.GetPluginStatus("my-plugin")
    assert.NoError(t, err)
    assert.Equal(t, api.PluginStateStopped, status.State)

    // 卸载插件
    err = manager.UnloadPlugin("my-plugin")
    assert.NoError(t, err)
}
```

### 4.3 性能测试

```go
package performance

import (
    "context"
    "testing"
    "time"

    "github.com/hashicorp/go-hclog"
    "github.com/lomehong/kennel/pkg/plugin/api"
    "github.com/lomehong/kennel/pkg/plugin/sdk"
)

func BenchmarkPluginInit(b *testing.B) {
    // 创建日志记录器
    logger := hclog.NewNullLogger()

    // 创建配置
    config := api.PluginConfig{
        ID:      "my-plugin",
        Enabled: true,
        Settings: map[string]interface{}{
            "option1": "test",
            "option2": 123,
        },
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // 创建插件
        plugin := sdk.NewPluginBuilder("my-plugin").
            WithName("我的插件").
            WithVersion("1.0.0").
            WithLogger(logger).
            WithInitFunc(func(ctx context.Context, config api.PluginConfig) error {
                return nil
            }).
            Build()

        // 初始化插件
        plugin.Init(context.Background(), config)
    }
}
```

## 5. 插件部署

### 5.1 编译插件

```bash
# 编译插件
go build -o my-plugin cmd/main.go
```

### 5.2 打包插件

```bash
# 创建插件目录
mkdir -p dist/my-plugin

# 复制插件文件
cp my-plugin dist/my-plugin/
cp config/config.yaml dist/my-plugin/

# 打包插件
cd dist
zip -r my-plugin.zip my-plugin/
```

### 5.3 安装插件

```bash
# 解压插件
unzip my-plugin.zip -d plugins/

# 设置权限
chmod +x plugins/my-plugin/my-plugin
```

### 5.4 配置插件

编辑插件配置文件：

```bash
vim plugins/my-plugin/config.yaml
```

### 5.5 启动插件

```bash
# 直接启动插件
./plugins/my-plugin/my-plugin

# 或者通过插件管理器启动
# 在应用程序中加载插件
```

## 6. 插件最佳实践

### 6.1 插件设计原则

- **单一职责原则**: 每个插件只负责一个功能
- **接口隔离原则**: 只实现需要的接口
- **依赖倒置原则**: 依赖于抽象而非具体实现
- **开闭原则**: 对扩展开放，对修改关闭

### 6.2 插件性能优化

- **减少通信开销**: 批量处理消息，减少通信次数
- **资源管理**: 合理使用资源，避免资源泄漏
- **并发控制**: 使用适当的并发模型，避免竞态条件
- **缓存**: 缓存频繁访问的数据，减少计算开销

### 6.3 插件安全性

- **输入验证**: 验证所有输入，防止注入攻击
- **权限控制**: 限制插件权限，遵循最小权限原则
- **资源限制**: 限制插件资源使用，防止资源耗尽
- **错误处理**: 正确处理错误，不泄露敏感信息

### 6.4 插件文档

- **README**: 提供插件的基本信息和使用说明
- **API文档**: 详细描述插件的API
- **配置说明**: 说明插件的配置选项
- **示例代码**: 提供插件的使用示例
- **变更日志**: 记录插件的版本变更

## 7. 常见问题

### 7.1 插件加载失败

可能的原因：
- 插件文件不存在
- 插件格式不正确
- 插件依赖缺失
- 插件版本不兼容

解决方法：
- 检查插件文件是否存在
- 检查插件格式是否正确
- 安装缺失的依赖
- 使用兼容的插件版本

### 7.2 插件通信失败

可能的原因：
- 通信协议不匹配
- 网络问题
- 序列化/反序列化错误
- 权限问题

解决方法：
- 检查通信协议是否匹配
- 检查网络连接
- 检查序列化/反序列化逻辑
- 检查权限设置

### 7.3 插件资源泄漏

可能的原因：
- 插件未正确释放资源
- 插件未正确处理异常
- 插件未正确实现Stop方法

解决方法：
- 确保在Stop方法中释放所有资源
- 使用defer确保资源释放
- 实现正确的异常处理
- 使用资源监控工具检测泄漏

### 7.4 插件冲突

可能的原因：
- 插件ID冲突
- 插件依赖冲突
- 插件资源冲突

解决方法：
- 使用唯一的插件ID
- 解决依赖冲突
- 避免资源冲突

## 8. 参考资料

- [Go Plugin Package](https://golang.org/pkg/plugin/)
- [Hashicorp Go Plugin](https://github.com/hashicorp/go-plugin)
- [gRPC](https://grpc.io/)
- [WebSocket](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API)
- [Semver](https://semver.org/)
