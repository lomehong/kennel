# 终端管控插件AI模块与MCP Server集成方案

## 概述

本文档描述了终端管控插件AI模块与MCP Server的集成方案。MCP (Model Control Protocol) Server是一个轻量级的HTTP服务器，用于提供AI模型控制能力，允许AI模型通过HTTP API调用终端管控插件的功能。

## 架构设计

### 组件关系

```
+----------------+     +----------------+     +----------------+
|                |     |                |     |                |
|   AI模型服务   | <-> |   MCP Server   | <-> |  终端管控插件  |
|                |     |                |     |                |
+----------------+     +----------------+     +----------------+
```

### 主要组件

1. **MCP Server**：提供HTTP API，允许AI模型调用终端管控插件的功能
2. **MCP Client**：终端管控插件中的客户端组件，用于与MCP Server通信
3. **远程工具适配器**：将MCP Server提供的工具适配为终端管控插件可用的工具

## 功能特性

### MCP Server

- 提供工具注册和管理功能
- 提供工具执行API
- 支持参数验证和错误处理
- 支持API密钥认证

### MCP Client

- 提供与MCP Server通信的能力
- 支持工具列表获取
- 支持工具信息获取
- 支持工具执行

### 远程工具适配器

- 将MCP Server提供的工具适配为终端管控插件可用的工具
- 支持参数转换和结果处理

## 工具列表

目前，MCP Server提供以下工具：

1. **get_processes**：获取系统进程列表，可以按名称过滤
2. **kill_process**：终止指定的进程
3. **execute_command**：执行系统命令，返回命令的输出结果

## 使用方法

### 启动MCP Server

```bash
# 使用默认配置启动
./mcp_server.exe

# 指定监听地址和端口
./mcp_server.exe -addr :8080

# 指定API密钥
./mcp_server.exe -api-key your-api-key

# 指定日志级别
./mcp_server.exe -log-level debug
```

### 使用MCP Client

```go
// 创建MCP Client配置
mcpConfig := &mcp.ClientConfig{
    ServerAddr: "http://localhost:8080",
    APIKey:     "your-api-key",
    Timeout:    30 * time.Second,
    MaxRetries: 3,
}

// 创建MCP Client
mcpClient, err := mcp.NewClient(mcpConfig, logger)
if err != nil {
    logger.Error("初始化 MCP Client 失败", "error", err)
    return
}

// 获取工具列表
tools, err := mcpClient.ListTools(ctx)
if err != nil {
    logger.Error("获取工具列表失败", "error", err)
    return
}

// 获取工具信息
toolInfo, err := mcpClient.GetTool(ctx, "get_processes")
if err != nil {
    logger.Error("获取工具信息失败", "error", err)
    return
}

// 执行工具
params := map[string]interface{}{
    "name_filter": "chrome",
    "limit":       10,
}
result, err := mcpClient.ExecuteTool(ctx, "get_processes", params)
if err != nil {
    logger.Error("执行工具失败", "error", err)
    return
}
```

### 使用远程工具适配器

```go
// 创建远程工具适配器
remoteTool, err := mcp.NewRemoteTool(mcpClient, "get_processes")
if err != nil {
    logger.Error("创建远程工具失败", "error", err)
    return
}

// 获取工具名称
name := remoteTool.GetName()

// 获取工具描述
description := remoteTool.GetDescription()

// 获取工具参数定义
parameters := remoteTool.GetParameters()

// 执行工具
params := map[string]interface{}{
    "name_filter": "chrome",
    "limit":       10,
}
result, err := remoteTool.Execute(ctx, params)
if err != nil {
    logger.Error("执行工具失败", "error", err)
    return
}
```

## 配置选项

### MCP Server配置

```json
{
  "addr": ":8080",
  "read_timeout": 10,
  "write_timeout": 10,
  "max_header_bytes": 1048576,
  "api_key": "your-api-key"
}
```

### MCP Client配置

```json
{
  "server_addr": "http://localhost:8080",
  "api_key": "your-api-key",
  "timeout": 30,
  "max_retries": 3
}
```

## 示例查询

以下是一些示例查询，展示了如何使用MCP Server提供的工具：

### 获取进程列表

```json
{
  "name": "get_processes",
  "params": {
    "name_filter": "chrome",
    "limit": 10
  }
}
```

### 终止进程

```json
{
  "name": "kill_process",
  "params": {
    "pid": 1234,
    "force": false
  }
}
```

### 执行命令

```json
{
  "name": "execute_command",
  "params": {
    "command": "ipconfig",
    "args": ["/all"],
    "timeout": 30
  }
}
```

## 错误处理

MCP Server和Client提供了统一的错误处理机制，错误信息包含以下字段：

- **code**：错误代码
- **message**：错误消息
- **details**：错误详情（可选）

常见错误代码：

- **tool_not_found**：工具不存在
- **invalid_params**：参数无效
- **execution_error**：执行错误
- **unauthorized**：未授权
- **internal_error**：内部错误

## 安全性考虑

- 使用API密钥进行认证
- 限制允许执行的命令
- 保护敏感进程不被终止
- 使用HTTPS进行通信（生产环境）
- 限制API访问频率
