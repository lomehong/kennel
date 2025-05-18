# 资产管理插件

## 功能概述

资产管理插件用于收集和管理终端资产信息，包括主机信息、CPU、内存、磁盘和网络接口等。

## 主要功能

1. **资产信息收集**：收集终端的硬件和软件信息
2. **资产信息缓存**：缓存资产信息，减少重复收集
3. **资产信息上报**：将资产信息上报到指定服务器

## 配置选项

```yaml
# 资产管理插件配置
enabled: true

# 收集间隔（秒）
collect_interval: 3600

# 上报服务器
report_server: "https://example.com/api/assets"

# 是否启用自动上报
auto_report: false

# 日志级别
log_level: "info"

# 缓存设置
cache:
  # 是否启用缓存
  enabled: true
  
  # 缓存目录
  dir: "data/assets/cache"
```

## API

### 请求

#### 收集资产信息

```json
{
  "action": "collect",
  "params": {}
}
```

#### 上报资产信息

```json
{
  "action": "report",
  "params": {}
}
```

### 事件

#### 系统启动事件

```json
{
  "type": "system.startup",
  "source": "system"
}
```

#### 资产扫描请求事件

```json
{
  "type": "asset.scan_request",
  "source": "user"
}
```

## 使用示例

```go
// 收集资产信息
resp, err := pluginManager.SendRequest("assets", &plugin.Request{
    ID:     "req-001",
    Action: "collect",
})

// 上报资产信息
resp, err := pluginManager.SendRequest("assets", &plugin.Request{
    ID:     "req-002",
    Action: "report",
})
```
