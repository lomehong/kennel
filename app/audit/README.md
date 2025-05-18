# 安全审计插件

## 功能概述

安全审计插件用于记录和管理系统安全事件，包括系统事件、用户事件、网络事件和文件事件等。

## 主要功能

1. **事件记录**：记录系统中的各类安全事件
2. **日志管理**：管理审计日志，包括查询、过滤和清理
3. **安全警报**：对重要安全事件发送警报通知

## 配置选项

```yaml
# 安全审计插件配置
enabled: true

# 是否记录系统事件
log_system_events: true

# 是否记录用户事件
log_user_events: true

# 是否记录网络事件
log_network_events: true

# 是否记录文件事件
log_file_events: true

# 日志保留天数
log_retention_days: 30

# 日志级别
log_level: "info"

# 是否启用实时警报
enable_alerts: false

# 警报接收者
alert_recipients:
  - "admin@example.com"

# 日志存储
storage:
  # 存储类型: file, database
  type: "file"
  
  # 文件存储配置
  file:
    # 日志目录
    dir: "data/audit/logs"
    
    # 日志文件名格式
    filename_format: "audit-%Y-%m-%d.log"
    
    # 是否压缩旧日志
    compress: true
```

## API

### 请求

#### 记录审计事件

```json
{
  "action": "log_event",
  "params": {
    "event_type": "user.login",
    "user": "admin",
    "details": {
      "ip": "192.168.1.100",
      "success": true
    }
  }
}
```

#### 获取审计日志

```json
{
  "action": "get_logs",
  "params": {
    "event_type": "user.login",
    "user": "admin",
    "start_time": "2023-01-01T00:00:00Z",
    "end_time": "2023-01-31T23:59:59Z"
  }
}
```

#### 清除审计日志

```json
{
  "action": "clear_logs",
  "params": {
    "user": "admin"
  }
}
```

### 事件

安全审计插件会监听并记录以下类型的事件：

- **系统事件**：`system.startup`, `system.shutdown`
- **用户事件**：`user.login`, `user.logout`
- **网络事件**：`network.connect`, `network.disconnect`
- **文件事件**：`file.create`, `file.modify`, `file.delete`

## 使用示例

```go
// 记录审计事件
resp, err := pluginManager.SendRequest("audit", &plugin.Request{
    ID:     "req-001",
    Action: "log_event",
    Params: map[string]interface{}{
        "event_type": "user.login",
        "user":       "admin",
        "details": map[string]interface{}{
            "ip":      "192.168.1.100",
            "success": true,
        },
    },
})

// 获取审计日志
resp, err := pluginManager.SendRequest("audit", &plugin.Request{
    ID:     "req-002",
    Action: "get_logs",
    Params: map[string]interface{}{
        "event_type": "user.login",
        "user":       "admin",
    },
})
```
