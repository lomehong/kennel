# 通讯模块集成

## 概述

通讯模块已经集成到框架的核心部分，使其成为框架的基础服务之一。通过这个集成，框架可以与服务端建立长连接，接收服务端下发的消息，并将消息路由到对应的插件进行处理。

## 架构设计

通讯模块的集成采用了分层设计：

1. **通讯管理器(CommManager)**：框架级别的通讯管理器，负责管理与服务端的通信
2. **消息路由**：将服务端下发的消息路由到对应的插件
3. **插件接口扩展**：扩展了插件接口，支持消息处理

### 通讯管理器

通讯管理器是框架级别的组件，负责管理与服务端的通信。它提供了以下功能：

- 初始化通讯模块
- 连接到服务端
- 接收服务端下发的消息
- 将消息路由到对应的插件
- 向服务端发送消息
- 断开与服务端的连接

### 消息路由

消息路由是通讯模块的核心功能之一，它负责将服务端下发的消息路由到对应的插件。消息路由的流程如下：

1. 通讯管理器接收到服务端下发的消息
2. 根据消息类型，调用对应的处理函数
3. 处理函数解析消息内容，获取目标插件
4. 将消息转发给目标插件
5. 插件处理消息，返回结果
6. 通讯管理器将结果发送给服务端

### 插件接口扩展

为了支持消息处理，我们扩展了插件接口，添加了 `HandleMessage` 方法：

```go
// Module 定义了插件模块的接口
type Module interface {
    // Init 初始化模块
    Init(config map[string]interface{}) error

    // Execute 执行模块操作
    Execute(action string, params map[string]interface{}) (map[string]interface{}, error)

    // Shutdown 关闭模块
    Shutdown() error

    // GetInfo 获取模块信息
    GetInfo() ModuleInfo

    // HandleMessage 处理消息
    HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error)
}
```

## 配置选项

通讯模块支持以下配置选项：

| 配置项 | 说明 | 默认值 |
|-------|------|-------|
| enable_comm | 是否启用通讯功能 | true |
| server_url | 服务器URL | ws://localhost:8080/ws |
| heartbeat_interval | 心跳间隔 | 30s |
| reconnect_interval | 重连间隔 | 5s |
| max_reconnect_attempts | 最大重连次数 | 10 |
| comm_shutdown_timeout | 通讯模块关闭超时时间（秒） | 5 |

### 安全配置选项

通讯模块支持以下安全配置选项：

| 配置项 | 说明 | 默认值 |
|-------|------|-------|
| comm_security.enable_tls | 是否启用TLS | false |
| comm_security.verify_server_cert | 是否验证服务器证书 | true |
| comm_security.client_cert_file | 客户端证书文件路径 | "" |
| comm_security.client_key_file | 客户端私钥文件路径 | "" |
| comm_security.ca_cert_file | CA证书文件路径 | "" |
| comm_security.enable_encryption | 是否启用消息加密 | false |
| comm_security.encryption_key | 加密密钥 | "" |
| comm_security.enable_auth | 是否启用认证 | false |
| comm_security.auth_token | 认证令牌 | "" |
| comm_security.auth_type | 认证类型 (basic, token, jwt) | "token" |
| comm_security.username | 用户名 (用于basic认证) | "" |
| comm_security.password | 密码 (用于basic认证) | "" |
| comm_security.enable_compression | 是否启用消息压缩 | false |
| comm_security.compression_level | 压缩级别 (1-9，1最快，9最高压缩率) | 6 |
| comm_security.compression_threshold | 压缩阈值，超过该大小才进行压缩（字节） | 1024 |

## 使用方法

### 在框架中使用通讯模块

通讯模块已经集成到框架的核心部分，在框架启动时会自动初始化和连接到服务端。你可以通过以下方式获取通讯管理器：

```go
// 获取通讯管理器
commManager := app.GetCommManager()

// 发送消息
commManager.SendMessage(comm.MessageTypeEvent, map[string]interface{}{
    "event": "app_started",
    "details": map[string]interface{}{
        "time": time.Now().Format(time.RFC3339),
    },
})
```

### 在插件中处理消息

插件需要实现 `HandleMessage` 方法来处理服务端下发的消息：

```go
// HandleMessage 处理消息
func (m *MyModule) HandleMessage(messageType string, messageID string, timestamp int64, payload map[string]interface{}) (map[string]interface{}, error) {
    // 处理消息
    switch messageType {
    case "command":
        // 处理命令消息
        command, _ := payload["command"].(string)
        params, _ := payload["params"].(map[string]interface{})

        // 处理命令
        // ...

    case "data":
        // 处理数据消息
        // ...

    case "event":
        // 处理事件消息
        // ...
    }

    // 返回结果
    return map[string]interface{}{
        "success": true,
        "message": "消息处理成功",
    }, nil
}
```

## 消息格式

### 命令消息

命令消息用于服务端向客户端发送命令，格式如下：

```json
{
    "type": "command",
    "id": "msg-123456",
    "timestamp": 1623456789000,
    "payload": {
        "command": "execute_plugin",
        "params": {
            "plugin": "dlp",
            "action": "scan",
            "params": {
                "path": "C:\\Users\\user\\Documents",
                "recursive": true
            }
        }
    }
}
```

### 数据消息

数据消息用于传输数据，格式如下：

```json
{
    "type": "data",
    "id": "msg-123456",
    "timestamp": 1623456789000,
    "payload": {
        "type": "config_update",
        "data": {
            "enable_dlp": true,
            "enable_device": true,
            "scan_interval": 3600
        }
    }
}
```

### 事件消息

事件消息用于通知事件，格式如下：

```json
{
    "type": "event",
    "id": "msg-123456",
    "timestamp": 1623456789000,
    "payload": {
        "event": "file_changed",
        "details": {
            "path": "C:\\Users\\user\\Documents\\file.txt",
            "action": "modified"
        }
    }
}
```

### 响应消息

响应消息用于响应请求，格式如下：

```json
{
    "type": "response",
    "id": "msg-123456",
    "timestamp": 1623456789000,
    "payload": {
        "request_id": "msg-123456",
        "success": true,
        "data": {
            "result": "success"
        }
    }
}
```

## 安全功能

通讯模块提供了完整的安全功能，包括TLS加密、消息加密、消息压缩和认证。

### TLS加密

通讯模块支持TLS加密，确保通信过程中的数据安全。可以通过以下配置启用TLS：

```yaml
comm_security:
  enable_tls: true
  verify_server_cert: true
  client_cert_file: "certs/client.crt"
  client_key_file: "certs/client.key"
  ca_cert_file: "certs/ca.crt"
```

TLS配置选项说明：

- `enable_tls`：是否启用TLS加密
- `verify_server_cert`：是否验证服务器证书
- `client_cert_file`：客户端证书文件路径
- `client_key_file`：客户端私钥文件路径
- `ca_cert_file`：CA证书文件路径

### 消息加密

通讯模块支持消息级别的加密，确保消息内容的安全。可以通过以下配置启用消息加密：

```yaml
comm_security:
  enable_encryption: true
  encryption_key: "your-encryption-key"
```

消息加密配置选项说明：

- `enable_encryption`：是否启用消息加密
- `encryption_key`：加密密钥

#### 加密实现

通讯模块使用AES-GCM算法进行消息加密，这是一种高安全性的对称加密算法，具有以下特点：

1. **认证加密**：AES-GCM不仅提供加密，还提供消息认证，可以检测消息是否被篡改
2. **高性能**：AES-GCM具有高性能，适合加密大量数据
3. **安全性**：AES-GCM是一种广泛使用的加密算法，安全性已经得到验证

加密过程如下：

1. 使用SHA-256哈希密码，得到32字节的密钥
2. 使用AES-GCM模式加密消息
3. 将随机数（nonce）附加到密文前面，形成最终的加密数据

解密过程如下：

1. 使用SHA-256哈希密码，得到32字节的密钥
2. 从加密数据中分离出随机数和密文
3. 使用AES-GCM模式解密密文

#### 安全注意事项

1. **密钥管理**：加密密钥应该妥善保管，不要硬编码在代码中，建议使用环境变量或配置文件
2. **密钥强度**：加密密钥应该足够强，建议使用随机生成的密钥
3. **密钥轮换**：定期更换加密密钥，提高安全性
4. **传输安全**：即使消息已经加密，也建议使用TLS加密传输，提供双重保护

### 消息压缩

通讯模块支持消息压缩，减少网络传输的数据量，提高性能。可以通过以下配置启用消息压缩：

```yaml
comm_security:
  enable_compression: true
  compression_level: 6
  compression_threshold: 1024
```

压缩配置选项说明：

- `enable_compression`：是否启用消息压缩
- `compression_level`：压缩级别，范围1-9，1表示最快压缩速度，9表示最高压缩率
- `compression_threshold`：压缩阈值，只有当消息大小超过该阈值时才进行压缩，单位为字节

#### 压缩实现

通讯模块使用gzip算法进行消息压缩，这是一种广泛使用的压缩算法，具有以下特点：

1. **高压缩率**：gzip可以显著减少数据大小，特别是对于文本数据
2. **快速解压**：gzip解压速度快，适合客户端使用
3. **广泛支持**：gzip是一种标准的压缩格式，被广泛支持

压缩过程如下：

1. 检查消息大小是否超过压缩阈值，如果没有超过，则不进行压缩
2. 使用gzip算法压缩消息
3. 比较压缩前后的大小，如果压缩后的大小大于或等于原始大小，则不使用压缩结果
4. 添加压缩标记，标识消息是否已压缩

解压过程如下：

1. 检查压缩标记，判断消息是否已压缩
2. 如果已压缩，使用gzip算法解压消息
3. 如果未压缩，直接返回原始消息

#### 性能考虑

1. **压缩阈值**：设置合适的压缩阈值可以避免对小消息进行压缩，因为压缩小消息可能会增加数据大小
2. **压缩级别**：根据需求选择合适的压缩级别，较低的级别压缩速度快但压缩率低，较高的级别压缩率高但压缩速度慢
3. **消息类型**：文本数据通常有较高的压缩率，而二进制数据的压缩效果可能不明显

### 认证

通讯模块支持多种认证方式，包括基本认证、令牌认证和JWT认证。可以通过以下配置启用认证：

```yaml
comm_security:
  enable_auth: true
  auth_type: "token"
  auth_token: "your-auth-token"
```

或者使用基本认证：

```yaml
comm_security:
  enable_auth: true
  auth_type: "basic"
  username: "your-username"
  password: "your-password"
```

认证配置选项说明：

- `enable_auth`：是否启用认证
- `auth_type`：认证类型，支持 "basic"、"token" 和 "jwt"
- `auth_token`：认证令牌，用于 "token" 和 "jwt" 认证
- `username`：用户名，用于 "basic" 认证
- `password`：密码，用于 "basic" 认证

## 优雅终止

通讯模块实现了完整的优雅终止机制，确保在应用程序关闭时，与服务器的连接能够正确地关闭，避免连接泄漏和数据丢失。

优雅终止流程如下：

1. **停止接收新消息**：通讯模块停止接收新的消息
2. **等待消息处理完成**：通讯模块等待所有正在处理的消息完成
3. **发送关闭消息**：通讯模块向服务器发送关闭消息，通知服务器客户端即将关闭
4. **关闭WebSocket连接**：通讯模块发送WebSocket关闭帧，然后关闭连接
5. **释放资源**：通讯模块释放相关资源

通讯模块的优雅终止是有超时控制的，如果在指定时间内未能完成关闭，将强制关闭连接。默认超时时间为5秒，可以在配置文件中修改：

```yaml
# 通讯模块关闭超时时间（秒）
comm_shutdown_timeout: 5
```

框架会在收到终止信号时自动调用通讯模块的优雅终止流程，无需手动处理。

## 监控

通讯模块提供了完整的监控功能，可以实时监控通讯模块的状态和性能。

### 监控指标

通讯模块收集以下指标：

1. **连接指标**：
   - 连接次数
   - 连接失败次数
   - 断开连接次数
   - 重连次数
   - 最后一次连接时间
   - 最后一次断开连接时间
   - 连接持续时间

2. **消息指标**：
   - 发送消息数量
   - 接收消息数量
   - 发送字节数
   - 接收字节数
   - 消息错误数量

3. **延迟指标**：
   - 平均延迟
   - 最大延迟
   - 最小延迟

4. **压缩指标**：
   - 压缩消息数量
   - 压缩前字节数
   - 压缩后字节数
   - 压缩率

5. **加密指标**：
   - 加密消息数量
   - 加密前字节数
   - 加密后字节数

6. **心跳指标**：
   - 发送心跳数量
   - 接收心跳数量
   - 心跳错误数量
   - 最后一次心跳时间

7. **错误指标**：
   - 错误数量
   - 最后一次错误时间
   - 最后一次错误消息

8. **状态指标**：
   - 当前连接状态

### 获取监控指标

可以通过以下方式获取监控指标：

```go
// 获取通讯管理器
commManager := app.GetCommManager()

// 获取指标
metrics := commManager.GetMetrics()

// 获取指标报告
report := commManager.GetMetricsReport()
fmt.Println(report)
```

### 监控工具

通讯模块提供了一个命令行监控工具，可以实时监控通讯模块的状态和性能：

```bash
# 构建监控工具
cd tools/comm_monitor
go build

# 运行监控工具
./comm_monitor -addr localhost:8080 -path /ws -interval 5

# 输出JSON格式
./comm_monitor -addr localhost:8080 -path /ws -interval 5 -json

# 监控特定指标
./comm_monitor -addr localhost:8080 -path /ws -interval 5 -watch "sent_bytes"

# 设置监控持续时间
./comm_monitor -addr localhost:8080 -path /ws -interval 5 -duration 60
```

监控工具支持以下参数：

- `-addr`：服务器地址，默认为 `localhost:8080`
- `-path`：WebSocket路径，默认为 `/ws`
- `-interval`：监控间隔（秒），默认为 `5`
- `-json`：输出JSON格式，默认为 `false`
- `-watch`：监控特定指标，默认为空（监控所有指标）
- `-duration`：监控持续时间（秒），默认为 `0`（一直运行）
- `-log-level`：日志级别，默认为 `info`

## 测试

通讯模块提供了完整的测试功能，包括单元测试、集成测试和测试工具。详细的测试指南请参考[通讯模块测试指南](comm_testing.md)。

### 单元测试

通讯模块包含了完整的单元测试，覆盖了核心功能，包括连接、消息发送和接收、断开连接等。

```bash
# 运行所有通讯模块的单元测试
cd pkg/comm
go test -v
```

### 集成测试

集成测试验证通讯模块与框架的集成，确保通讯模块能够正确地工作在框架环境中。

```bash
# 运行所有集成测试
cd test/integration
go test -v
```

### 测试工具

通讯模块提供了一个测试工具，用于手动测试通讯功能。测试工具支持服务器模式和客户端模式。

```bash
# 构建测试工具
cd tools/comm_tester
go build

# 启动服务器
./comm_tester -server -addr localhost:8080 -path /ws

# 启动客户端
./comm_tester -client -addr localhost:8080 -path /ws -interactive
```

## 注意事项

1. 通讯模块会自动处理重连，无需手动重连
2. 发送消息前应检查连接状态，避免在断开连接时发送消息
3. 消息处理函数应该是非阻塞的，如果需要执行耗时操作，应该在新的goroutine中执行
4. 通讯模块使用WebSocket协议，确保服务端支持WebSocket
5. 在插件的Shutdown方法中，确保所有通过通讯模块发送的消息都已经处理完成
