# 插件系统设计文档

## 1. 概述

插件系统是一个模块化、可扩展的框架，允许开发者通过插件机制扩展应用程序的功能。本文档描述了插件系统的设计、架构和使用方法。

### 1.1 设计目标

- **模块化**: 将功能拆分为独立的插件，便于维护和扩展
- **松耦合**: 插件之间通过标准接口通信，减少依赖
- **可扩展**: 支持动态加载和卸载插件，无需修改核心代码
- **隔离性**: 插件运行在隔离环境中，避免影响主应用程序
- **依赖管理**: 支持插件之间的依赖关系管理
- **版本兼容**: 支持插件版本管理和兼容性检查

### 1.2 核心概念

- **插件**: 实现特定功能的独立模块
- **插件管理器**: 负责插件的加载、卸载、启动和停止
- **插件注册表**: 存储已注册插件的元数据
- **插件发现器**: 发现可用的插件
- **生命周期管理器**: 管理插件的生命周期
- **依赖管理器**: 管理插件之间的依赖关系
- **隔离器**: 提供插件的隔离环境
- **通信层**: 提供插件之间的通信机制

## 2. 架构设计

### 2.1 整体架构

插件系统的整体架构如下图所示：

```
+----------------------------------+
|           应用程序               |
+----------------------------------+
|          插件管理器              |
+----------------------------------+
|                                  |
| +------------+ +---------------+ |
| | 插件注册表 | | 插件发现器    | |
| +------------+ +---------------+ |
|                                  |
| +------------+ +---------------+ |
| | 依赖管理器 | | 生命周期管理器| |
| +------------+ +---------------+ |
|                                  |
| +------------+ +---------------+ |
| |  隔离器    | |   通信层      | |
| +------------+ +---------------+ |
|                                  |
+----------------------------------+
|                                  |
| +--------+ +--------+ +--------+ |
| | 插件1  | | 插件2  | | 插件3  | |
| +--------+ +--------+ +--------+ |
|                                  |
+----------------------------------+
```

### 2.2 组件说明

#### 2.2.1 插件管理器

插件管理器是插件系统的核心组件，负责协调其他组件的工作。主要功能包括：

- 加载和卸载插件
- 启动和停止插件
- 管理插件的生命周期
- 处理插件之间的依赖关系
- 提供插件的隔离环境
- 管理插件之间的通信

#### 2.2.2 插件注册表

插件注册表存储已注册插件的元数据，包括：

- 插件ID
- 插件名称
- 插件版本
- 插件描述
- 插件作者
- 插件许可证
- 插件标签
- 插件能力
- 插件依赖

#### 2.2.3 插件发现器

插件发现器负责发现可用的插件，支持多种发现方式：

- 文件系统发现
- 网络发现
- 注册表发现

#### 2.2.4 生命周期管理器

生命周期管理器负责管理插件的生命周期，包括：

- 初始化
- 启动
- 停止
- 卸载

#### 2.2.5 依赖管理器

依赖管理器负责管理插件之间的依赖关系，包括：

- 依赖解析
- 依赖注入
- 版本兼容性检查

#### 2.2.6 隔离器

隔离器提供插件的隔离环境，支持多种隔离级别：

- 无隔离
- 基本隔离
- 严格隔离
- 完全隔离

#### 2.2.7 通信层

通信层提供插件之间的通信机制，支持多种通信协议：

- 进程内通信
- gRPC
- HTTP
- WebSocket

## 3. 插件接口

### 3.1 基础插件接口

所有插件都必须实现的基础接口：

```go
type Plugin interface {
    // GetInfo 返回插件信息
    GetInfo() PluginInfo
    
    // Init 初始化插件
    Init(ctx context.Context, config PluginConfig) error
    
    // Start 启动插件
    Start(ctx context.Context) error
    
    // Stop 停止插件
    Stop(ctx context.Context) error
    
    // HealthCheck 执行健康检查
    HealthCheck(ctx context.Context) (HealthStatus, error)
}
```

### 3.2 领域特定接口

针对不同领域的插件接口：

- `ServicePlugin`: 提供服务的插件
- `UIPlugin`: 提供UI组件的插件
- `DataProcessorPlugin`: 提供数据处理功能的插件
- `SecurityPlugin`: 提供安全功能的插件
- `HTTPHandlerPlugin`: 提供HTTP处理功能的插件
- `StoragePlugin`: 提供存储功能的插件
- `EventHandlerPlugin`: 提供事件处理功能的插件

### 3.3 能力接口

插件可以实现的额外能力接口：

- `Configurable`: 可配置的插件
- `Versionable`: 可版本化的插件
- `Debuggable`: 可调试的插件
- `Monitorable`: 可监控的插件
- `Traceable`: 可追踪的插件
- `Loggable`: 可记录日志的插件
- `Documentable`: 可文档化的插件

## 4. 插件开发指南

### 4.1 创建插件

使用插件SDK创建插件：

```go
package main

import (
    "context"
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

### 4.2 插件配置

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
  option1: value1
  option2: value2
```

### 4.3 插件通信

插件之间的通信示例：

```go
// 注册服务
err := comm.RegisterService(myService)

// 获取服务
service, err := comm.GetService("other-plugin-service")

// 发送消息
err := comm.SendMessage("target-plugin", message)

// 订阅主题
err := comm.Subscribe("topic", func(message interface{}) {
    // 处理消息
})

// 发布消息
err := comm.Publish("topic", message)
```

### 4.4 依赖注入

使用依赖注入：

```go
type MyPlugin struct {
    // 依赖注入
    Logger    hclog.Logger  `inject:"logger"`
    Config    *Config       `inject:"config"`
    Database  *Database     `inject:"database"`
}

// 注入依赖
err := injector.Inject(plugin)
```

## 5. 插件管理

### 5.1 加载插件

```go
// 创建插件管理器
manager := plugin.NewPluginManagerV3(logger, plugin.DefaultManagerConfigV3())

// 启动插件管理器
err := manager.Start()

// 加载插件
plugin, err := manager.LoadPlugin("my-plugin")
```

### 5.2 卸载插件

```go
// 卸载插件
err := manager.UnloadPlugin("my-plugin")
```

### 5.3 启动插件

```go
// 启动插件
err := manager.StartPlugin("my-plugin")
```

### 5.4 停止插件

```go
// 停止插件
err := manager.StopPlugin("my-plugin")
```

### 5.5 获取插件状态

```go
// 获取插件状态
status, err := manager.GetPluginStatus("my-plugin")
```

## 6. 最佳实践

### 6.1 插件设计原则

- 单一职责原则：每个插件只负责一个功能
- 接口隔离原则：只实现需要的接口
- 依赖倒置原则：依赖于抽象而非具体实现
- 开闭原则：对扩展开放，对修改关闭

### 6.2 插件性能优化

- 减少插件之间的通信
- 使用适当的隔离级别
- 优化资源使用
- 实现健康检查

### 6.3 插件安全性

- 验证插件签名
- 限制插件权限
- 监控插件资源使用
- 实现插件沙箱

### 6.4 插件测试

- 单元测试
- 集成测试
- 性能测试
- 安全测试

## 7. 常见问题

### 7.1 插件加载失败

可能的原因：
- 插件文件不存在
- 插件格式不正确
- 插件依赖缺失
- 插件版本不兼容

### 7.2 插件通信失败

可能的原因：
- 通信协议不匹配
- 网络问题
- 序列化/反序列化错误
- 权限问题

### 7.3 插件资源泄漏

可能的原因：
- 插件未正确释放资源
- 插件未正确处理异常
- 插件未正确实现Stop方法

### 7.4 插件冲突

可能的原因：
- 插件ID冲突
- 插件依赖冲突
- 插件资源冲突

## 8. 参考资料

- [Go Plugin Package](https://golang.org/pkg/plugin/)
- [Hashicorp Go Plugin](https://github.com/hashicorp/go-plugin)
- [gRPC](https://grpc.io/)
- [WebSocket](https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API)
- [Semver](https://semver.org/)
