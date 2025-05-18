# 通讯模块

## 概述

通讯模块是框架的基础组件，负责与服务端进行双向通信。它提供了以下功能：

- 建立与服务端的WebSocket长连接
- 接收服务端下发的消息
- 向服务端发送消息
- 自动重连机制
- 心跳保活
- 消息分发处理

通讯模块位于`pkg/comm`目录下，是一个独立的包，可以被其他模块引用。

## 架构设计

通讯模块采用分层设计，主要包含以下组件：

1. **通讯管理器(Manager)**：提供高级API，负责消息分发和处理
2. **WebSocket客户端(Client)**：负责底层连接管理和消息收发
3. **消息处理器(Handler)**：负责处理不同类型的消息

### 消息类型

通讯模块定义了以下消息类型：

- **系统消息**：
  - `heartbeat`：心跳消息，用于保持连接活跃
  - `connect`：连接消息，用于建立连接
  - `ack`：确认消息，用于确认消息接收

- **业务消息**：
  - `command`：命令消息，服务端下发的指令
  - `data`：数据消息，用于传输数据
  - `event`：事件消息，用于通知事件
  - `response`：响应消息，用于响应请求

### 连接状态

通讯模块定义了以下连接状态：

- `StateDisconnected`：断开连接
- `StateConnecting`：正在连接
- `StateConnected`：已连接
- `StateReconnecting`：正在重连

## 使用方法

### 基本使用

```go
package main

import (
    "github.com/lomehong/kennel/pkg/comm"
    "github.com/lomehong/kennel/pkg/logger"
)

func main() {
    // 创建日志器
    log := logger.NewLogger("comm-example", nil)
    
    // 创建配置
    config := comm.DefaultConfig()
    config.ServerURL = "ws://example.com/ws"
    
    // 创建通讯管理器
    manager := comm.NewManager(config, log)
    
    // 设置客户端信息
    manager.SetClientInfo(map[string]interface{}{
        "client_id": "example-client",
        "version":   "1.0.0",
    })
    
    // 注册消息处理函数
    manager.RegisterHandler(comm.MessageTypeCommand, handleCommand)
    
    // 连接到服务器
    err := manager.Connect()
    if err != nil {
        log.Error("连接服务器失败", "error", err)
        return
    }
    
    // 发送事件
    manager.SendEvent("example_event", map[string]interface{}{
        "data": "这是一个示例事件",
    })
    
    // 使用完毕后断开连接
    manager.Disconnect()
}

// 处理命令消息
func handleCommand(msg *comm.Message) {
    // 处理命令
    command := msg.Payload["command"].(string)
    // ...
}
```

### 消息处理

通讯模块支持注册不同类型的消息处理函数：

```go
// 注册命令消息处理函数
manager.RegisterHandler(comm.MessageTypeCommand, func(msg *comm.Message) {
    command := msg.Payload["command"].(string)
    params := msg.Payload["params"].(map[string]interface{})
    
    switch command {
    case "restart":
        // 处理重启命令
    case "update":
        // 处理更新命令
    }
})

// 注册数据消息处理函数
manager.RegisterHandler(comm.MessageTypeData, func(msg *comm.Message) {
    dataType := msg.Payload["type"].(string)
    data := msg.Payload["data"]
    
    // 处理数据
})

// 注册事件消息处理函数
manager.RegisterHandler(comm.MessageTypeEvent, func(msg *comm.Message) {
    eventType := msg.Payload["event"].(string)
    details := msg.Payload["details"].(map[string]interface{})
    
    // 处理事件
})
```

### 发送消息

通讯模块提供了多种发送消息的方法：

```go
// 发送命令消息
manager.SendCommand("status", map[string]interface{}{
    "detail": true,
})

// 发送数据消息
manager.SendData("metrics", map[string]interface{}{
    "cpu": 30,
    "memory": 50,
})

// 发送事件消息
manager.SendEvent("file_changed", map[string]interface{}{
    "path": "/path/to/file",
    "action": "modified",
})

// 发送响应消息
manager.SendResponse("request-123", true, map[string]interface{}{
    "result": "success",
}, "")
```

## 配置选项

通讯模块支持以下配置选项：

| 配置项 | 说明 | 默认值 |
|-------|------|-------|
| ServerURL | 服务器URL | ws://localhost:8080/ws |
| ReconnectInterval | 重连间隔 | 5秒 |
| MaxReconnectAttempts | 最大重连次数 | 10 |
| HeartbeatInterval | 心跳间隔 | 30秒 |
| HandshakeTimeout | 握手超时 | 10秒 |
| WriteTimeout | 写超时 | 10秒 |
| ReadTimeout | 读超时 | 60秒 |
| MessageBufferSize | 消息缓冲区大小 | 100 |

## 高级功能

### 自定义消息类型

可以通过扩展`MessageType`类型来添加自定义消息类型：

```go
const (
    MessageTypeCustom comm.MessageType = "custom"
)

// 注册自定义消息处理函数
manager.RegisterHandler(MessageTypeCustom, handleCustomMessage)

// 发送自定义消息
manager.SendMessage(MessageTypeCustom, map[string]interface{}{
    "custom_data": "value",
})
```

### 连接状态监控

可以通过检查连接状态来监控连接：

```go
// 检查是否已连接
if manager.IsConnected() {
    // 执行需要连接的操作
}

// 获取当前连接状态
state := manager.GetState()
switch state {
case comm.StateConnected:
    // 已连接
case comm.StateDisconnected:
    // 已断开
case comm.StateConnecting:
    // 正在连接
case comm.StateReconnecting:
    // 正在重连
}
```

## 测试

通讯模块提供了测试工具，位于`test`目录下：

- `mock_server`：模拟服务端，用于测试客户端功能
- `comm_client`：测试客户端，用于测试与服务端的通信

### 运行测试

1. 启动模拟服务端：

```bash
go run test/mock_server/main.go
```

2. 运行测试客户端：

```bash
go run test/comm_client/main.go
```

## 注意事项

1. 通讯模块会自动处理重连，无需手动重连
2. 发送消息前应检查连接状态，避免在断开连接时发送消息
3. 消息处理函数应该是非阻塞的，如果需要执行耗时操作，应该在新的goroutine中执行
4. 通讯模块使用WebSocket协议，确保服务端支持WebSocket
