# Kennel 模块化架构设计

## 架构概述

Kennel 采用模块化、可插拔的架构设计，主要包括以下几个核心组件：

1. **核心框架（Core Framework）**：提供基础设施和公共服务
2. **插件系统（Plugin System）**：管理插件的生命周期和通信
3. **配置系统（Configuration System）**：管理分层配置
4. **通信层（Communication Layer）**：处理模块间和远程通信
5. **功能模块（Functional Modules）**：实现具体业务功能的插件

## 核心组件

### 核心框架（Core Framework）

核心框架提供以下功能：

- 应用程序生命周期管理
- 日志记录和错误处理
- 资源管理和监控
- 事件总线和消息分发
- 健康检查和状态报告

### 插件系统（Plugin System）

插件系统负责：

- 插件发现和注册
- 插件加载和卸载
- 插件隔离和资源限制
- 插件间依赖管理
- 插件生命周期管理

### 配置系统（Configuration System）

配置系统实现：

- 分层配置管理（全局、插件管理、插件特定）
- 配置热加载和动态更新
- 配置验证和迁移
- 配置覆盖和合并

### 通信层（Communication Layer）

通信层处理：

- 模块间通信（进程内和跨进程）
- 远程通信（WebSocket、gRPC等）
- 消息序列化和反序列化
- 通信安全和认证

## 模块接口规范

### 基础模块接口

所有模块必须实现以下基础接口：

```go
type Module interface {
    // 初始化模块
    Init(ctx context.Context, config *ModuleConfig) error
    
    // 启动模块
    Start() error
    
    // 停止模块
    Stop() error
    
    // 获取模块信息
    GetInfo() ModuleInfo
    
    // 处理请求
    HandleRequest(ctx context.Context, req *Request) (*Response, error)
    
    // 处理事件
    HandleEvent(ctx context.Context, event *Event) error
}
```

### 扩展接口

模块可以选择性实现以下扩展接口：

```go
// 健康检查接口
type HealthCheck interface {
    CheckHealth() HealthStatus
}

// 指标收集接口
type MetricsProvider interface {
    GetMetrics() []Metric
}

// 资源管理接口
type ResourceManager interface {
    GetResourceUsage() ResourceUsage
    SetResourceLimits(limits ResourceLimits) error
}
```

## 模块通信模式

### 请求-响应模式

用于同步通信，模块直接调用其他模块的方法并等待响应。

### 事件-订阅模式

用于异步通信，模块发布事件到事件总线，其他模块订阅并处理这些事件。

### 流式通信模式

用于大量数据传输或实时数据流，模块之间建立流式通信通道。

## 跨语言支持

通过以下机制实现跨语言支持：

1. **语言无关的通信协议**：使用gRPC/Protobuf定义接口
2. **进程隔离**：每个插件作为独立进程运行
3. **标准化接口**：所有语言实现相同的接口规范
4. **语言特定SDK**：为每种支持的语言提供SDK

## 插件生命周期

1. **发现**：系统扫描插件目录，识别有效插件
2. **加载**：加载插件元数据和配置
3. **初始化**：调用插件的Init方法
4. **启动**：调用插件的Start方法
5. **运行**：插件正常运行，处理请求和事件
6. **停止**：调用插件的Stop方法
7. **卸载**：释放插件资源

## 部署模型

### 单体部署

所有插件作为主程序的子进程运行在同一台机器上。

### 分布式部署

插件可以部署在不同的机器上，通过网络通信。

### 混合部署

核心插件在本地运行，非核心插件可以远程部署。
