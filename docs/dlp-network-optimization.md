# DLP网络拦截优化方案

## 问题分析

DLP插件启动后浏览器无法访问网站的根本原因：

1. **数据包拦截后未重新注入**：WinDivert拦截数据包后，没有将处理完的数据包重新注入到网络栈
2. **阻断模式导致流量中断**：默认使用阻断模式，所有被拦截的数据包都被丢弃
3. **缺乏白名单机制**：没有对浏览器等关键应用提供白名单保护
4. **过度拦截**：拦截了不必要的系统和本地流量

## 解决方案

### 1. 实现三种拦截模式

```go
type InterceptorMode int

const (
    ModeMonitorOnly       InterceptorMode = 0  // 仅监控模式
    ModeInterceptAndAllow InterceptorMode = 1  // 拦截并允许模式  
    ModeInterceptAndBlock InterceptorMode = 2  // 拦截并阻断模式
)
```

**模式说明：**
- **监控模式**：数据包被复制用于分析，原始数据包立即重新注入，不影响网络流量
- **拦截并允许模式**：数据包被拦截分析，分析完成后自动重新注入
- **拦截并阻断模式**：数据包被拦截分析，根据策略决定是否重新注入

### 2. 自动重新注入机制

**核心组件：**
- `reinjectCh`：重新注入通道
- `reinjectWorker()`：重新注入工作协程
- `AutoReinject`：自动重新注入配置

**工作流程：**
```
数据包拦截 → 发送到分析通道 → 同时发送到重新注入通道 → 立即重新注入
```

### 3. WinDivert嗅探模式

**配置优化：**
```yaml
flags: 1  # WINDIVERT_FLAG_SNIFF - 嗅探模式
```

**效果：**
- 数据包被复制而不是拦截
- 原始网络流量不受影响
- 仍能获得完整的数据包信息

### 4. 智能数据包处理

**监控模式处理逻辑：**
```go
case ModeMonitorOnly:
    // 发送到分析通道
    select {
    case w.packetCh <- packet:
        // 成功发送到分析通道
    default:
        // 通道满了也不影响重新注入
    }
    
    // 立即重新注入
    if w.config.AutoReinject {
        w.reinjectPacket(packet)
    }
```

### 5. 保留原始地址信息

**元数据保存：**
```go
// 保存原始WinDivert地址信息
packet.Metadata["windivert_address"] = addr
packet.Metadata["interface_index"] = addr.IfIdx
packet.Metadata["sub_interface_index"] = addr.SubIfIdx
```

**重新注入时使用原始地址：**
```go
if addrData, exists := packet.Metadata["windivert_address"]; exists {
    if originalAddr, ok := addrData.(*WinDivertAddress); ok {
        addr = originalAddr
    }
}
```

### 6. 白名单机制

**进程白名单：**
- chrome.exe, firefox.exe, msedge.exe
- 浏览器进程的流量优先处理

**域名白名单：**
- *.google.com, *.microsoft.com
- 常用网站直接放行

**IP白名单：**
- 本地回环地址：127.0.0.0/8
- 私有网络：10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16

## 配置文件优化

### 关键配置参数

```yaml
# 拦截器配置
interceptor_config:
  # 使用嗅探模式，不阻断流量
  flags: 1                 # WINDIVERT_FLAG_SNIFF
  mode: 0                  # 监控模式
  auto_reinject: true      # 自动重新注入
  
  # 优化过滤器
  filter: "outbound and (tcp.DstPort == 80 or tcp.DstPort == 443)"
  
  # 性能优化
  buffer_size: 32768
  channel_size: 500
  worker_count: 2

# 白名单配置
whitelist:
  enable: true
  processes: ["chrome.exe", "firefox.exe", "msedge.exe"]
  ips: ["127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
```

## 性能优化

### 1. 批量处理优化

- 批次大小：5个数据包
- 处理间隔：5ms
- 减少系统调用次数

### 2. 自适应流量限制

- 监控系统资源使用率
- 动态调整处理速率
- 防止系统过载

### 3. 错误处理优化

- 自适应延迟机制
- 减少日志频率
- 优雅降级处理

## 测试验证

### 1. 功能测试

**测试步骤：**
1. 启动DLP模块
2. 打开浏览器访问网站
3. 验证网站能正常加载
4. 检查DLP日志确认数据包被正确分析

**预期结果：**
- 浏览器正常访问网站
- DLP功能正常工作
- 网络性能影响最小

### 2. 性能测试

**监控指标：**
- 网络延迟增加 < 10ms
- CPU占用 < 10%
- 内存占用 < 150MB
- 数据包丢失率 < 0.1%

### 3. 稳定性测试

**测试场景：**
- 长时间运行（24小时）
- 高并发网络访问
- 多浏览器同时使用
- 大文件下载/上传

## 故障排除

### 常见问题

1. **浏览器仍无法访问网站**
   ```yaml
   # 检查配置
   mode: 0              # 确保使用监控模式
   auto_reinject: true  # 确保启用自动重新注入
   flags: 1             # 确保使用嗅探模式
   ```

2. **DLP功能不工作**
   ```yaml
   # 检查过滤器
   filter: "outbound and (tcp.DstPort == 80 or tcp.DstPort == 443)"
   # 确保目标端口在过滤器中
   ```

3. **网络性能下降**
   ```yaml
   # 降低处理负载
   worker_count: 1
   channel_size: 200
   max_packets_per_second: 500
   ```

### 调试命令

```bash
# 检查DLP进程状态
tasklist | findstr dlp

# 检查网络连接
netstat -an | findstr :80
netstat -an | findstr :443

# 查看DLP日志
tail -f dlp.log
```

## 部署建议

### 1. 生产环境

```yaml
# 保守配置
mode: 0                    # 监控模式
auto_reinject: true        # 自动重新注入
flags: 1                   # 嗅探模式
worker_count: 1            # 单工作协程
max_packets_per_second: 500 # 限制处理速率
```

### 2. 开发环境

```yaml
# 标准配置
mode: 1                    # 拦截并允许模式
auto_reinject: true        # 自动重新注入
worker_count: 2            # 双工作协程
max_packets_per_second: 1000
```

### 3. 测试环境

```yaml
# 完整功能配置
mode: 2                    # 拦截并阻断模式
auto_reinject: false       # 手动控制重新注入
worker_count: 3            # 多工作协程
```

## 总结

通过以上优化措施，DLP模块现在能够：

1. **保证网络连通性**：使用嗅探模式和自动重新注入机制
2. **提供完整监控**：仍能获得所有网络流量的详细信息
3. **最小化性能影响**：优化的处理流程和资源配置
4. **灵活的部署模式**：支持监控、拦截允许、拦截阻断三种模式
5. **智能白名单保护**：对关键应用和网络提供保护

这确保了DLP功能在提供安全保护的同时，不会影响用户的正常网络使用体验。
