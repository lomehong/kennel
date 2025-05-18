# 终端管控插件

## 功能概述

终端管控插件用于远程执行命令、管理进程和安装软件，提供对终端的远程管理能力。

## 主要功能

1. **进程管理**：获取进程列表、查找进程、终止进程
2. **命令执行**：远程执行命令并获取结果
3. **软件安装**：远程安装软件包

## 配置选项

```yaml
# 终端管控插件配置
enabled: true

# 是否允许远程执行命令
allow_remote_command: true

# 是否允许远程安装软件
allow_software_install: true

# 是否允许远程终止进程
allow_process_kill: true

# 命令执行超时时间（秒）
command_timeout: 30

# 软件安装超时时间（秒）
install_timeout: 600

# 进程缓存过期时间（秒）
process_cache_expiration: 10

# 日志级别
log_level: "info"

# 白名单进程（不允许终止）
protected_processes:
  - "agent.exe"
  - "system"
  - "explorer.exe"

# 白名单命令（允许执行）
allowed_commands:
  - "ipconfig"
  - "ping"
  - "dir"
  - "ls"
  - "ps"
```

## API

### 请求

#### 获取进程列表

```json
{
  "action": "get_processes",
  "params": {}
}
```

#### 终止进程

```json
{
  "action": "kill_process",
  "params": {
    "pid": "1234"
  }
}
```

#### 查找进程

```json
{
  "action": "find_process",
  "params": {
    "name": "chrome"
  }
}
```

#### 执行命令

```json
{
  "action": "execute_command",
  "params": {
    "command": "ipconfig",
    "args": ["/all"],
    "timeout": 30
  }
}
```

#### 安装软件

```json
{
  "action": "install_software",
  "params": {
    "package": "git",
    "timeout": 600
  }
}
```

### 事件

终端管控插件会监听以下类型的事件：

- **系统事件**：`system.startup`, `system.shutdown`
- **进程监控事件**：`process.monitor`

## 使用示例

```go
// 获取进程列表
resp, err := pluginManager.SendRequest("control", &plugin.Request{
    ID:     "req-001",
    Action: "get_processes",
})

// 执行命令
resp, err := pluginManager.SendRequest("control", &plugin.Request{
    ID:     "req-002",
    Action: "execute_command",
    Params: map[string]interface{}{
        "command": "ipconfig",
        "args":    []string{"/all"},
        "timeout": 30,
    },
})

// 安装软件
resp, err := pluginManager.SendRequest("control", &plugin.Request{
    ID:     "req-003",
    Action: "install_software",
    Params: map[string]interface{}{
        "package": "git",
        "timeout": 600,
    },
})
```

## 安全注意事项

1. 使用白名单限制可执行的命令
2. 保护关键系统进程，防止被终止
3. 根据需要禁用远程命令执行、进程终止或软件安装功能
4. 记录所有操作，便于审计
