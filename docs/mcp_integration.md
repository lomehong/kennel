# 终端管控插件AI模块与MCP Server集成方案

## 概述

本文档描述了终端管控插件AI模块与MCP Server的集成方案。MCP (Model Context Protocol) 是一种用于与 AI 模型交互的协议，它提供了一种标准化的方式来与各种 AI 模型进行通信。MCP Server 是一个轻量级的 HTTP 服务器，用于提供 AI 模型控制能力，允许 AI 模型通过 HTTP API 调用终端管控插件的功能。

本集成方案支持 CloudWego 的 Eino 框架，同时保持与现有 OpenAI 和 Ark 模型的兼容性。

## 架构设计

### 组件关系

```
+----------------+     +----------------+     +----------------+
|                |     |                |     |                |
|   AI模型服务   | <-> |   MCP Server   | <-> |  终端管控插件  |
|                |     |                |     |                |
+----------------+     +----------------+     +----------------+
                                ^
                                |
                                v
                       +----------------+
                       |                |
                       |   Eino 框架    |
                       |                |
                       +----------------+
```

### 主要组件

1. **MCP Server**：提供 HTTP API，允许 AI 模型调用终端管控插件的功能
2. **MCP Client**：终端管控插件中的客户端组件，用于与 MCP Server 通信
3. **MCP 管理器**：管理 MCP 客户端的生命周期，提供统一的接口
4. **AI 管理器**：管理 AI 相关功能，包括 MCP、OpenAI 和 Ark 等提供者
5. **工具执行器**：执行特定工具的组件，如进程终止工具、命令执行工具等

## 功能特性

### MCP Server

- 提供工具注册和管理功能
- 提供工具执行 API
- 支持参数验证和错误处理
- 支持 API 密钥认证
- 支持 CloudWego Eino 框架集成

### MCP Client

- 提供与 MCP Server 通信的能力
- 支持工具列表获取
- 支持工具信息获取
- 支持工具执行
- 支持流式响应
- 支持重试机制

### MCP 管理器

- 管理 MCP 客户端的生命周期
- 提供统一的接口
- 支持配置管理
- 支持工具管理
- 支持错误处理

### AI 管理器

- 管理 AI 相关功能
- 支持多种 AI 提供者（MCP、OpenAI、Ark）
- 提供统一的查询接口
- 支持流式响应
- 支持配置管理

### 工具执行器

- 执行特定工具的组件
- 支持进程终止工具
- 支持命令执行工具
- 支持文件读取工具
- 支持参数验证和错误处理

## 工具列表

目前，MCP Server 提供以下工具：

1. **process_kill**：终止指定的进程
   - 参数：
     - `pid`：进程 ID（整数，必需）
   - 返回值：
     - `success`：是否成功（布尔值）
     - `error`：错误信息（字符串，可选）

2. **command_execute**：执行系统命令，返回命令的输出结果
   - 参数：
     - `command`：要执行的命令（字符串，必需）
     - `args`：命令参数（字符串数组，可选）
     - `workDir`：工作目录（字符串，可选）
     - `timeout`：超时时间（整数，可选，单位：秒）
   - 返回值：
     - `success`：是否成功（布尔值）
     - `stdout`：标准输出（字符串，可选）
     - `stderr`：标准错误（字符串，可选）
     - `exitCode`：退出码（整数）
     - `error`：错误信息（字符串，可选）

3. **file_read**：读取指定的文件
   - 参数：
     - `path`：文件路径（字符串，必需）
     - `startLine`：开始行（整数，可选，从 1 开始）
     - `endLine`：结束行（整数，可选，从 1 开始）
   - 返回值：
     - `success`：是否成功（布尔值）
     - `content`：文件内容（字符串，可选）
     - `error`：错误信息（字符串，可选）

## 使用方法

### 配置 AI 管理器

在终端管控插件的配置文件中，添加以下配置：

```json
{
  "ai": {
    "enabled": true,
    "provider": "mcp",
    "mcp_enabled": true,
    "mcp_server_addr": "http://localhost:8080",
    "mcp_api_key": "your-api-key",
    "mcp_model_name": "gpt-4"
  }
}
```

### 使用 MCP 管理器

```go
// 创建 MCP 管理器配置
config := &mcp.ManagerConfig{
    Enabled:    true,
    ServerAddr: "http://localhost:8080",
    APIKey:     "your-api-key",
    ModelName:  "gpt-4",
    Timeout:    30 * time.Second,
    MaxRetries: 3,
}

// 创建 MCP 管理器
manager, err := mcp.NewManager(config, logger)
if err != nil {
    logger.Error("创建 MCP 管理器失败", "error", err)
    return
}

// 启动 MCP 管理器
if err := manager.Start(ctx); err != nil {
    logger.Error("启动 MCP 管理器失败", "error", err)
    return
}
defer manager.Stop()

// 查询 AI
response, err := manager.QueryAI(ctx, "你好，请介绍一下自己")
if err != nil {
    logger.Error("查询 AI 失败", "error", err)
    return
}
fmt.Println(response)

// 执行工具
result, err := manager.ExecuteTool(ctx, "process_kill", map[string]interface{}{
    "pid": 1234,
})
if err != nil {
    logger.Error("执行工具失败", "error", err)
    return
}
fmt.Printf("%+v\n", result)
```

### 使用 MCP 客户端

```go
// 创建 MCP 客户端配置
config := &mcp.SimpleClientConfig{
    ServerAddr: "http://localhost:8080",
    APIKey:     "your-api-key",
    Timeout:    30 * time.Second,
    MaxRetries: 3,
    ModelName:  "gpt-4",
}

// 创建 MCP 客户端
client, err := mcp.NewSimpleClient(config, logger)
if err != nil {
    logger.Error("创建 MCP 客户端失败", "error", err)
    return
}
defer client.Close()

// 获取工具列表
tools, err := client.ListTools(ctx)
if err != nil {
    logger.Error("获取工具列表失败", "error", err)
    return
}

// 打印工具列表
for _, tool := range tools {
    fmt.Printf("工具: %s\n", tool.Name)
    fmt.Printf("描述: %s\n", tool.Description)
    fmt.Printf("参数: %+v\n", tool.Parameters)
}

// 执行工具
params := map[string]interface{}{
    "pid": 1234,
}
result, err := client.ExecuteTool(ctx, "process_kill", params)
if err != nil {
    logger.Error("执行工具失败", "error", err)
    return
}
fmt.Printf("执行结果: %+v\n", result)

// 查询 AI
response, err := client.QueryAI(ctx, "你好，请介绍一下自己")
if err != nil {
    logger.Error("查询 AI 失败", "error", err)
    return
}
fmt.Println(response)
```

### 使用 AI 管理器

```go
// 创建 AI 管理器
aiManager := ai.NewAIManager(logger, config)

// 初始化 AI 管理器
if err := aiManager.Init(ctx); err != nil {
    logger.Error("初始化 AI 管理器失败", "error", err)
    return
}
defer aiManager.Stop()

// 处理 AI 请求
response, err := aiManager.HandleRequest(ctx, "你好，请介绍一下自己")
if err != nil {
    logger.Error("处理 AI 请求失败", "error", err)
    return
}
fmt.Println(response)

// 处理流式 AI 请求
err = aiManager.HandleStreamRequest(ctx, "你好，请介绍一下自己", func(chunk string) error {
    fmt.Print(chunk)
    return nil
})
if err != nil {
    logger.Error("处理流式 AI 请求失败", "error", err)
    return
}
```

## 配置选项

### MCP 服务器配置

```json
{
  "enabled": true,
  "listen_addr": ":8080",
  "api_key": "your-api-key",
  "timeout": 30
}
```

### MCP 客户端配置

```json
{
  "server_addr": "http://localhost:8080",
  "api_key": "your-api-key",
  "timeout": 10,
  "max_retries": 3,
  "retry_delay": 1,
  "retry_delay_max": 5,
  "model_name": "gpt-4",
  "stream_mode": false
}
```

### MCP 管理器配置

```json
{
  "enabled": true,
  "server_addr": "http://localhost:8080",
  "api_key": "your-api-key",
  "model_name": "gpt-4",
  "timeout": 10,
  "max_retries": 3,
  "retry_delay": 1,
  "retry_delay_max": 5,
  "tools": {
    "process_kill": "终止指定的进程",
    "command_execute": "执行指定的命令",
    "file_read": "读取指定的文件"
  }
}
```

### AI 管理器配置

```json
{
  "enabled": true,
  "provider": "mcp",
  "mcp_enabled": true,
  "mcp_server_addr": "http://localhost:8080",
  "mcp_api_key": "your-api-key",
  "mcp_model_name": "gpt-4",
  "openai_enabled": false,
  "openai_api_key": "",
  "ark_enabled": false,
  "ark_api_key": ""
}
```

## 示例查询

以下是一些示例查询，展示了如何使用 MCP 提供的功能：

### 基本查询

```
你好，请介绍一下自己
```

### 终止进程

```
请终止进程 ID 为 1234 的进程
```

对应的工具调用：

```json
{
  "name": "process_kill",
  "params": {
    "pid": 1234
  }
}
```

### 执行命令

```
请执行 ipconfig /all 命令，并显示结果
```

对应的工具调用：

```json
{
  "name": "command_execute",
  "params": {
    "command": "ipconfig",
    "args": ["/all"],
    "timeout": 30
  }
}
```

### 读取文件

```
请读取 C:\Windows\System32\drivers\etc\hosts 文件的内容
```

对应的工具调用：

```json
{
  "name": "file_read",
  "params": {
    "path": "C:\\Windows\\System32\\drivers\\etc\\hosts"
  }
}
```

### 复杂查询

```
请帮我分析一下系统中的进程，找出占用 CPU 最高的进程，并提供终止该进程的命令
```

## 错误处理

MCP 集成提供了统一的错误处理机制，错误信息包含以下字段：

- **success**：是否成功（布尔值）
- **error**：错误消息（字符串）
- **code**：错误代码（字符串，可选）
- **details**：错误详情（对象，可选）

常见错误类型：

- **工具不存在**：请求的工具不存在
- **参数无效**：提供的参数无效
- **执行错误**：工具执行过程中发生错误
- **未授权**：API 密钥无效或缺失
- **内部错误**：服务器内部错误
- **超时错误**：请求超时
- **网络错误**：网络连接问题

错误处理示例：

```go
// 执行工具
result, err := manager.ExecuteTool(ctx, "process_kill", map[string]interface{}{
    "pid": 1234,
})
if err != nil {
    // 处理错误
    logger.Error("执行工具失败", "error", err)
    return
}

// 检查结果
if result, ok := result.(map[string]interface{}); ok {
    if success, ok := result["success"].(bool); ok && !success {
        // 处理工具执行失败
        if errMsg, ok := result["error"].(string); ok {
            logger.Error("工具执行失败", "error", errMsg)
        }
        return
    }
}
```

## 安全性考虑

- **认证**：
  - 使用 API 密钥进行认证
  - 支持 HTTPS 进行安全通信（生产环境）
  - 考虑使用更强的认证机制，如 OAuth 2.0

- **授权**：
  - 限制允许执行的命令
  - 保护敏感进程不被终止
  - 实现细粒度的权限控制

- **数据安全**：
  - 敏感数据加密存储
  - 避免在日志中记录敏感信息
  - 定期清理临时文件和日志

- **网络安全**：
  - 限制 API 访问频率
  - 实现 IP 白名单
  - 使用防火墙保护服务器

- **审计**：
  - 记录所有 API 调用
  - 监控异常活动
  - 定期审查日志

## 最佳实践

- **配置管理**：
  - 使用环境变量或配置文件管理敏感信息
  - 不同环境使用不同的配置
  - 定期更新 API 密钥

- **错误处理**：
  - 实现全面的错误处理
  - 提供有用的错误消息
  - 避免暴露敏感信息

- **性能优化**：
  - 使用连接池
  - 实现缓存机制
  - 优化查询性能

- **可观测性**：
  - 实现详细的日志记录
  - 添加性能指标
  - 设置监控和告警
