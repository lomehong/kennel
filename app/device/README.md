# 设备管理插件

## 功能概述

设备管理插件用于监控和管理网络接口和USB设备，提供设备信息收集、网络接口管理等功能。

## 主要功能

1. **设备信息收集**：收集网络接口和USB设备信息
2. **网络接口管理**：启用/禁用网络接口
3. **设备监控**：监控设备变化并发送通知

## 配置选项

```yaml
# 设备管理插件配置
enabled: true

# 是否监控USB设备
monitor_usb: true

# 是否监控网络接口
monitor_network: true

# 是否允许禁用网络接口
allow_network_disable: true

# 设备缓存过期时间（秒）
device_cache_expiration: 30

# 设备监控间隔（秒）
monitor_interval: 60

# 日志级别
log_level: "info"

# 网络接口白名单（不允许禁用）
protected_interfaces:
  - "lo"
  - "eth0"
  - "en0"
```

## API

### 请求

#### 获取所有设备信息

```json
{
  "action": "get_devices",
  "params": {}
}
```

#### 获取网络接口列表

```json
{
  "action": "get_network_interfaces",
  "params": {}
}
```

#### 获取USB设备列表

```json
{
  "action": "get_usb_devices",
  "params": {}
}
```

#### 启用网络接口

```json
{
  "action": "enable_network_interface",
  "params": {
    "name": "eth0"
  }
}
```

#### 禁用网络接口

```json
{
  "action": "disable_network_interface",
  "params": {
    "name": "eth1"
  }
}
```

### 事件

设备管理插件会监听以下类型的事件：

- **系统事件**：`system.startup`, `system.shutdown`
- **设备扫描请求**：`device.scan_request`

## 使用示例

```go
// 获取所有设备信息
resp, err := pluginManager.SendRequest("device", &plugin.Request{
    ID:     "req-001",
    Action: "get_devices",
})

// 获取网络接口列表
resp, err := pluginManager.SendRequest("device", &plugin.Request{
    ID:     "req-002",
    Action: "get_network_interfaces",
})

// 禁用网络接口
resp, err := pluginManager.SendRequest("device", &plugin.Request{
    ID:     "req-003",
    Action: "disable_network_interface",
    Params: map[string]interface{}{
        "name": "eth1",
    },
})
```

## 安全注意事项

1. 使用白名单保护关键网络接口，防止被禁用
2. 根据需要禁用网络接口管理功能
3. 记录所有设备变化，便于审计
