# 终端管控插件AI功能集成指南

本文档介绍了如何在终端管控插件中使用AI功能，包括配置、使用方法和示例查询。

## 1. 功能概述

终端管控插件集成了CloudWego的Eino框架，提供了以下AI功能：

- 自然语言处理：理解用户的自然语言请求，执行相应的操作
- 进程管理：通过自然语言查询和管理系统进程
- 命令执行：通过自然语言执行系统命令
- 流式响应：支持流式响应，提供更好的用户体验

## 2. 配置方法

### 2.1 配置文件

在终端管控插件的配置文件中，添加以下AI相关配置：

```json
{
  "id": "control",
  "name": "终端管控插件",
  "version": "1.0.0",
  "settings": {
    "log_level": "info",
    "allow_remote_command": true,
    "command_timeout": 30,
    "allowed_commands": ["ipconfig", "ping", "netstat", "tasklist", "dir", "systeminfo", "echo", "whoami", "hostname"],
    "protected_processes": ["system", "explorer", "winlogon", "services", "lsass"],
    "process_cache_expiration": 10,
    "ai": {
      "enabled": true,
      "model_type": "openai",
      "model_name": "gpt-3.5-turbo",
      "api_key": "your-api-key-here",
      "base_url": "",
      "max_tokens": 2000,
      "temperature": 0.7
    }
  },
  "dependencies": []
}
```

如果要使用CloudWego的Ark模型，可以使用以下配置：

```json
{
  "settings": {
    "ai": {
      "enabled": true,
      "model_type": "ark",
      "model_name": "ark-model-name",
      "api_key": "your-ark-api-key-here",
      "base_url": "https://api.ark-example.com",
      "max_tokens": 2000,
      "temperature": 0.7
    }
  }
}
```

### 2.2 配置项说明

| 配置项 | 类型 | 说明 |
| --- | --- | --- |
| ai.enabled | boolean | 是否启用AI功能 |
| ai.model_type | string | 模型类型，支持"openai"和"ark" |
| ai.model_name | string | 模型名称，如"gpt-3.5-turbo"、"gpt-4"等 |
| ai.api_key | string | API密钥 |
| ai.base_url | string | 可选，API基础URL，用于自定义API端点或使用代理 |
| ai.max_tokens | number | 最大生成令牌数 |
| ai.temperature | number | 生成温度，值越高，生成的内容越随机 |

### 2.3 环境变量

也可以通过环境变量设置API密钥：

```bash
# OpenAI模型 - Windows
set OPENAI_API_KEY=your-api-key-here

# OpenAI模型 - Linux/macOS
export OPENAI_API_KEY=your-api-key-here

# Ark模型 - Windows
set ARK_API_KEY=your-ark-api-key-here

# Ark模型 - Linux/macOS
export ARK_API_KEY=your-ark-api-key-here
```

注意：环境变量设置的API密钥会覆盖配置文件中的设置。

## 3. 使用方法

### 3.1 通过API调用

可以通过终端管控插件的API调用AI功能：

```go
// 创建请求
req := &plugin.Request{
    ID:     "req-001",
    Action: "ai_query",
    Params: map[string]interface{}{
        "query":     "列出当前运行的进程",
        "streaming": false,
    },
}

// 发送请求
resp, err := module.HandleRequest(ctx, req)
if err != nil {
    // 处理错误
}

// 处理响应
if resp.Success {
    response := resp.Data["response"].(string)
    fmt.Println(response)
} else {
    fmt.Printf("请求失败: %s\n", resp.Error.Message)
}
```

### 3.2 流式响应

如果需要流式响应，可以设置`streaming`参数为`true`：

```go
// 创建请求
req := &plugin.Request{
    ID:     "req-001",
    Action: "ai_query",
    Params: map[string]interface{}{
        "query":     "列出当前运行的进程",
        "streaming": true,
    },
}

// 发送请求
resp, err := module.HandleRequest(ctx, req)
if err != nil {
    // 处理错误
}

// 处理流式响应
if resp.Success {
    responseChan := resp.Data["channel"].(chan string)
    errorChan := resp.Data["error"].(chan error)

    for {
        select {
        case response, ok := <-responseChan:
            if !ok {
                // 响应结束
                return
            }
            fmt.Print(response)
        case err := <-errorChan:
            fmt.Printf("处理请求失败: %v\n", err)
            return
        }
    }
} else {
    fmt.Printf("请求失败: %s\n", resp.Error.Message)
}
```

### 3.3 命令行测试程序

可以使用提供的命令行测试程序测试AI功能：

```bash
# 使用OpenAI模型运行测试程序
go run app/control/cmd/ai_test/simple_test.go -api-key=your-api-key-here -model=gpt-3.5-turbo

# 使用Ark模型运行测试程序
go run app/control/cmd/ai_test/simple_test.go -api-key=your-ark-api-key-here -model=ark-model-name -model-type=ark -base-url=https://api.ark-example.com

# 启用流式响应
go run app/control/cmd/ai_test/simple_test.go -api-key=your-api-key-here -stream=true

# 完整参数说明
# -api-key: API密钥
# -model: 模型名称，默认为gpt-3.5-turbo
# -model-type: 模型类型，可选值为openai或ark，默认为openai
# -base-url: API基础URL，可选
# -stream: 是否使用流式响应，默认为false
```

## 4. 示例查询

以下是一些示例查询和预期的响应：

### 4.1 进程管理

**查询**：列出当前运行的进程

**响应**：
```
当前系统中运行的进程如下：

1. System (PID: 4)
2. explorer.exe (PID: 1234)
3. chrome.exe (PID: 5678)
4. ...

共有XX个进程正在运行。
```

**查询**：查找名称包含"chrome"的进程

**响应**：
```
找到以下包含"chrome"的进程：

1. chrome.exe (PID: 5678, CPU: 2.5%, 内存: 150MB)
2. chrome.exe (PID: 5679, CPU: 1.2%, 内存: 120MB)
3. ...

共找到XX个匹配的进程。
```

**查询**：终止PID为5678的进程

**响应**：
```
已成功终止PID为5678的进程(chrome.exe)。
```

### 4.2 命令执行

**查询**：显示当前网络配置

**响应**：
```
当前网络配置信息如下：

Windows IP 配置

以太网适配器 以太网:
   连接特定的 DNS 后缀 . . . . . . . :
   IPv4 地址 . . . . . . . . . . . . : 192.168.1.100
   子网掩码  . . . . . . . . . . . . : 255.255.255.0
   默认网关. . . . . . . . . . . . . : 192.168.1.1

...
```

**查询**：检查与百度的连接

**响应**：
```
正在检查与百度的连接...

正在 Ping baidu.com [39.156.69.79] 具有 32 字节的数据:
来自 39.156.69.79 的回复: 字节=32 时间=36ms TTL=51
来自 39.156.69.79 的回复: 字节=32 时间=36ms TTL=51
来自 39.156.69.79 的回复: 字节=32 时间=36ms TTL=51
来自 39.156.69.79 的回复: 字节=32 时间=36ms TTL=51

39.156.69.79 的 Ping 统计信息:
    数据包: 已发送 = 4，已接收 = 4，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 36ms，最长 = 36ms，平均 = 36ms

连接正常，网络状态良好。
```

## 5. 注意事项

1. **API密钥安全**：请妥善保管API密钥，不要将其硬编码在代码中或提交到版本控制系统。
2. **命令限制**：为了安全起见，只允许执行配置文件中指定的命令。
3. **进程保护**：为了系统安全，不允许终止配置文件中指定的受保护进程。
4. **流量消耗**：使用AI功能会消耗API调用次数，请合理使用。
5. **响应时间**：AI响应可能需要一定时间，特别是在网络不稳定的情况下。
6. **模型选择**：OpenAI模型和Ark模型各有优势，可以根据需要选择合适的模型。
7. **API兼容性**：确保使用的Ark模型API与OpenAI API兼容，否则可能需要额外的适配工作。

## 6. 故障排除

1. **初始化失败**：检查API密钥是否正确，网络是否正常。
2. **响应超时**：检查网络连接，或者尝试使用流式响应。
3. **命令执行失败**：检查命令是否在允许列表中，参数是否正确。
4. **进程终止失败**：检查进程是否存在，是否是受保护的进程。
5. **模型类型错误**：确保配置的模型类型正确，目前支持"openai"和"ark"。
6. **基础URL错误**：如果使用自定义基础URL，确保URL格式正确且可访问。
7. **Ark模型问题**：如果使用Ark模型遇到问题，检查Ark模型的API是否与OpenAI API兼容。

如果遇到其他问题，可以查看日志文件获取更详细的错误信息。
