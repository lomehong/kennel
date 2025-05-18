# 数据防泄漏插件

## 功能概述

数据防泄漏插件用于检测和防止敏感数据泄漏，支持文件扫描、剪贴板监控和自定义规则管理。

## 主要功能

1. **敏感数据检测**：使用正则表达式检测敏感数据
2. **文件扫描**：扫描文件中的敏感数据
3. **剪贴板监控**：监控剪贴板中的敏感数据
4. **规则管理**：添加、更新、删除和启用/禁用规则

## 配置选项

```yaml
# 数据防泄漏插件配置
enabled: true

# 是否启用剪贴板监控
monitor_clipboard: false

# 是否启用文件监控
monitor_files: false

# 监控的文件类型
monitored_file_types:
  - "*.doc"
  - "*.docx"
  - "*.xls"
  - "*.xlsx"
  - "*.pdf"
  - "*.txt"

# 监控的目录
monitored_directories:
  - "C:/Users/*/Documents"
  - "C:/Users/*/Desktop"

# 日志级别
log_level: "info"

# 规则配置
rules:
  - id: "credit-card"
    name: "信用卡号"
    description: "检测信用卡号"
    pattern: "\\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12}|(?:2131|1800|35\\d{3})\\d{11})\\b"
    action: "alert"
    enabled: true
  
  - id: "email"
    name: "电子邮件地址"
    description: "检测电子邮件地址"
    pattern: "\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}\\b"
    action: "alert"
    enabled: true
  
  - id: "ip-address"
    name: "IP地址"
    description: "检测IP地址"
    pattern: "\\b(?:\\d{1,3}\\.){3}\\d{1,3}\\b"
    action: "alert"
    enabled: true
```

## API

### 请求

#### 获取规则列表

```json
{
  "action": "get_rules",
  "params": {}
}
```

#### 添加规则

```json
{
  "action": "add_rule",
  "params": {
    "id": "ssn",
    "name": "社会安全号码",
    "description": "检测美国社会安全号码",
    "pattern": "\\b\\d{3}-\\d{2}-\\d{4}\\b",
    "action": "block",
    "enabled": true
  }
}
```

#### 更新规则

```json
{
  "action": "update_rule",
  "params": {
    "id": "ssn",
    "name": "社会安全号码",
    "description": "检测美国社会安全号码",
    "pattern": "\\b\\d{3}-\\d{2}-\\d{4}\\b",
    "action": "alert",
    "enabled": true
  }
}
```

#### 删除规则

```json
{
  "action": "delete_rule",
  "params": {
    "id": "ssn"
  }
}
```

#### 扫描文件

```json
{
  "action": "scan_file",
  "params": {
    "path": "C:/Users/user/Documents/file.txt"
  }
}
```

#### 扫描目录

```json
{
  "action": "scan_directory",
  "params": {
    "directory": "C:/Users/user/Documents"
  }
}
```

#### 扫描剪贴板

```json
{
  "action": "scan_clipboard",
  "params": {}
}
```

#### 获取警报列表

```json
{
  "action": "get_alerts",
  "params": {}
}
```

#### 清除警报

```json
{
  "action": "clear_alerts",
  "params": {}
}
```

### 事件

数据防泄漏插件会监听以下类型的事件：

- **系统事件**：`system.startup`, `system.shutdown`
- **扫描请求**：`dlp.scan_request`

## 使用示例

```go
// 获取规则列表
resp, err := pluginManager.SendRequest("dlp", &plugin.Request{
    ID:     "req-001",
    Action: "get_rules",
})

// 扫描文件
resp, err := pluginManager.SendRequest("dlp", &plugin.Request{
    ID:     "req-002",
    Action: "scan_file",
    Params: map[string]interface{}{
        "path": "C:/Users/user/Documents/file.txt",
    },
})

// 扫描剪贴板
resp, err := pluginManager.SendRequest("dlp", &plugin.Request{
    ID:     "req-003",
    Action: "scan_clipboard",
})
```
