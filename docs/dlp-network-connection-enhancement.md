# DLP v2.0 网络连接信息增强

## 概述

成功为DLP v2.0审计日志系统增加了详细的网络连接信息，包括端口号、域名、完整URL和请求数据摘要，显著提升了安全审计和威胁溯源能力。

## 功能特性

### 1. 新增网络连接字段

#### AuditEvent结构体增强
```go
type AuditEvent struct {
    // ... 原有字段
    
    // 网络连接详细信息
    SourcePort  uint16 `json:"source_port"`           // 源进程使用的本地端口号
    DestPort    uint16 `json:"dest_port"`             // 目标服务器端口号
    DestDomain  string `json:"dest_domain,omitempty"` // 目标域名（如果可解析）
    RequestURL  string `json:"request_url,omitempty"` // 完整HTTP/HTTPS请求URL
    RequestData string `json:"request_data,omitempty"` // 发送的内容摘要或关键信息
    
    // ... 其他字段
}
```

#### 字段说明
- **source_port**: 源进程使用的本地端口号，用于连接追踪
- **dest_port**: 目标服务器端口号，识别服务类型（80/HTTP, 443/HTTPS等）
- **dest_domain**: 目标域名，便于识别访问的服务
- **request_url**: 完整的HTTP/HTTPS请求URL，包含路径和查询参数
- **request_data**: 请求数据摘要，经过脱敏处理的关键信息

### 2. 网络信息提取器

#### 核心组件
```go
type NetworkInfoExtractor struct {
    logger    logging.Logger
    dnsCache  map[string]string    // IP到域名的缓存
    cacheTime map[string]time.Time // 缓存时间戳
}
```

#### 主要功能
- **数据源整合**: 从PacketInfo和ParsedData中提取网络信息
- **URL构建**: 智能构建完整的HTTP/HTTPS请求URL
- **数据脱敏**: 自动识别和脱敏敏感信息
- **DNS缓存**: 异步DNS解析和缓存机制

### 3. 多协议数据提取

#### HTTP/HTTPS协议支持
- **GET请求**: 提取完整URL和查询参数
- **POST请求**: 提取表单数据、JSON数据、文件上传信息
- **头部信息**: 解析Host、User-Agent、Authorization等关键头部
- **内容类型**: 根据Content-Type进行不同的数据处理

#### 数据类型处理
```go
// JSON数据处理
func (nie *NetworkInfoExtractor) extractJSONData(data []byte) string

// 表单数据处理  
func (nie *NetworkInfoExtractor) extractFormData(data []byte) string

// 文件上传处理
func (nie *NetworkInfoExtractor) extractMultipartData(data []byte) string

// 文本数据处理
func (nie *NetworkInfoExtractor) extractTextData(data []byte) string
```

### 4. 安全脱敏机制

#### 敏感字段识别
自动识别并脱敏以下敏感信息：
- **密码相关**: password, pwd, secret, token, key, auth, credential
- **邮箱地址**: 部分脱敏显示
- **电话号码**: 格式化脱敏
- **API密钥**: 完全脱敏为 `***REDACTED***`

#### 脱敏示例
```json
// 原始数据
{"username":"alice","password":"secret123","api_key":"sk-1234567890abcdef"}

// 脱敏后
{"username":"alice","password":"***REDACTED***","api_key":"***REDACTED***"}
```

### 5. 模拟拦截器增强

#### 多样化测试数据
创建了4种不同类型的模拟网络请求：

1. **HTTPS API请求** (Chrome)
   - URL: `https://api.example.com/api/v1/users?page=1&limit=10`
   - 端口: 443
   - 包含Authorization头部

2. **HTTP表单提交** (Firefox)
   - URL: `http://www.example.com/login`
   - 端口: 80
   - 表单数据: 用户名、邮箱、密码（脱敏）

3. **文件上传** (Outlook)
   - URL: `http://files.example.com/upload`
   - 端口: 443
   - 多部分表单数据: document.pdf

4. **JSON API请求** (Postman)
   - URL: `http://hr.company.com/api/employees`
   - 端口: 443
   - JSON数据: 员工信息（敏感字段脱敏）

## 实际运行效果

### 审计日志示例

#### 1. HTTP表单登录 (Firefox)
```json
{
  "event_type": "dlp_decision",
  "action": "audit",
  "source_ip": "192.168.1.100",
  "source_port": 12347,
  "dest_ip": "203.0.113.2",
  "dest_port": 80,
  "request_url": "http://www.example.com/login",
  "request_data": "username=john.doe&email=john@example.com&password=***REDACTED***&remember=on",
  "process_name": "firefox.exe",
  "process_path": "C:\\Program Files\\Mozilla Firefox\\firefox.exe",
  "risk_level": "medium"
}
```

#### 2. 文件上传 (Outlook)
```json
{
  "event_type": "dlp_decision",
  "action": "audit",
  "source_ip": "192.168.1.100",
  "source_port": 12349,
  "dest_ip": "203.0.113.3",
  "dest_port": 443,
  "request_url": "http://files.example.com/upload",
  "request_data": "multipart/form-data with files: document.pdf",
  "process_name": "outlook.exe",
  "process_path": "C:\\Program Files\\Microsoft Office\\root\\Office16\\OUTLOOK.EXE",
  "risk_level": "low"
}
```

#### 3. JSON API请求 (Postman)
```json
{
  "event_type": "dlp_decision",
  "action": "audit",
  "source_ip": "192.168.1.100",
  "source_port": 12351,
  "dest_ip": "203.0.113.4",
  "dest_port": 443,
  "request_url": "http://hr.company.com/api/employees",
  "request_data": "{\"api_key\":\"***REDACTED***\",\"department\":\"engineering\",\"email\":\"alice@company.com\",\"salary\":75000,\"username\":\"alice\"}",
  "process_name": "postman.exe",
  "process_path": "C:\\Users\\user\\AppData\\Local\\Postman\\Postman.exe",
  "risk_level": "high"
}
```

## 技术实现

### 1. 架构设计

#### 数据流程
```
拦截器(PacketInfo) → 解析器(ParsedData) → 网络信息提取器 → 审计执行器 → 日志输出
```

#### 组件集成
- **拦截器**: 提供原始网络数据包信息
- **解析器**: 解析HTTP协议，提取结构化数据
- **提取器**: 整合多源数据，构建完整网络上下文
- **执行器**: 生成包含网络信息的审计事件

### 2. 性能优化

#### DNS解析优化
```go
// 异步DNS解析，不阻塞主流程
go func() {
    if names, err := net.LookupAddr(ip); err == nil && len(names) > 0 {
        domain := strings.TrimSuffix(names[0], ".")
        nie.dnsCache[ip] = domain
        nie.cacheTime[ip] = time.Now()
    }
}()
```

#### 数据大小限制
- **请求数据**: 限制在1KB以内，超出部分截断
- **URL长度**: 合理限制，避免过长URL影响性能
- **缓存管理**: DNS缓存5分钟过期，自动清理

### 3. 容错机制

#### 多层次后备方案
1. **优先使用ParsedData中的URL**
2. **从Headers构建URL**
3. **使用Metadata中的信息**
4. **提供基础网络信息**

#### 错误处理
- **解析失败**: 返回原始数据的安全摘要
- **网络超时**: 使用缓存或跳过DNS解析
- **数据异常**: 提供默认值确保系统稳定

## 安全价值

### 1. 增强的威胁检测

#### 网络行为分析
- **异常端口**: 识别非标准端口的通信
- **可疑域名**: 检测访问恶意或未授权域名
- **数据泄露**: 监控敏感数据的网络传输
- **协议滥用**: 发现HTTP/HTTPS协议的异常使用

#### 攻击链重建
- **完整URL**: 重建攻击者的访问路径
- **请求数据**: 分析攻击载荷和数据窃取
- **时间序列**: 构建攻击时间线
- **进程关联**: 将网络活动与具体进程关联

### 2. 合规性支持

#### 详细审计记录
- **数据传输**: 记录所有敏感数据的网络传输
- **访问控制**: 监控对受保护资源的访问
- **用户行为**: 追踪用户的网络活动模式
- **证据保全**: 提供完整的事件证据链

#### 监管要求
- **GDPR**: 个人数据处理的详细记录
- **SOX**: 财务数据访问的审计追踪
- **HIPAA**: 医疗数据传输的合规记录
- **PCI DSS**: 支付卡数据的安全监控

### 3. 运维价值

#### 网络可视化
- **流量分析**: 了解网络流量的构成和模式
- **应用识别**: 识别网络中活跃的应用程序
- **性能监控**: 监控网络性能和异常
- **容量规划**: 基于实际使用情况进行规划

#### 故障排查
- **连接问题**: 快速定位网络连接问题
- **应用错误**: 分析应用程序的网络行为
- **性能瓶颈**: 识别网络性能瓶颈
- **安全事件**: 快速响应安全事件

## 扩展功能

### 1. 高级分析能力

#### 流量模式分析
- **基线建立**: 建立正常网络行为基线
- **异常检测**: 基于机器学习的异常检测
- **趋势分析**: 网络流量的趋势和预测
- **关联分析**: 多维度数据关联分析

#### 威胁情报集成
- **恶意域名**: 集成威胁情报数据库
- **IP信誉**: 检查IP地址的信誉评分
- **URL分类**: 自动分类和风险评估
- **实时更新**: 威胁情报的实时更新

### 2. 可视化展示

#### 网络拓扑
- **连接图**: 可视化网络连接关系
- **流量图**: 显示数据流向和大小
- **时间线**: 网络事件的时间序列
- **热力图**: 网络活动的热力分布

#### 仪表板
- **实时监控**: 实时网络活动监控
- **统计报表**: 网络使用统计和报表
- **告警面板**: 安全告警和事件面板
- **趋势图表**: 长期趋势和模式图表

## 配置选项

### 1. 网络信息提取配置
```yaml
network_extractor:
  enabled: true
  dns_resolution:
    enabled: true
    cache_duration: "5m"
    timeout: "2s"
  data_extraction:
    max_request_data_size: 1024
    max_url_length: 2048
    enable_content_analysis: true
  security:
    enable_data_masking: true
    sensitive_fields:
      - "password"
      - "token"
      - "secret"
      - "key"
      - "auth"
```

### 2. 日志输出配置
```yaml
audit_logging:
  network_fields:
    include_ports: true
    include_domain: true
    include_url: true
    include_request_data: true
  output_format:
    json_pretty: false
    field_ordering: ["timestamp", "event_type", "source_ip", "dest_ip"]
```

## 总结

网络连接信息增强功能为DLP v2.0带来了显著的能力提升：

### ✅ 核心成果
- **完整网络上下文**: 提供端口、域名、URL、请求数据等完整信息
- **智能数据提取**: 支持多种协议和数据格式的智能解析
- **安全脱敏机制**: 自动识别和脱敏敏感信息
- **高性能设计**: 异步处理和缓存机制确保性能

### 🎯 安全价值
- **威胁检测增强**: 基于网络行为的高级威胁检测
- **攻击链重建**: 完整的攻击路径和数据流分析
- **合规性支持**: 满足各种监管要求的详细审计
- **运维可视化**: 网络活动的全面可视化和分析

### 🚀 技术特点
- **生产级实现**: 真实可用的网络信息提取
- **多协议支持**: HTTP/HTTPS协议的完整支持
- **容错设计**: 多层次后备方案确保稳定性
- **扩展性**: 支持未来协议和功能扩展

这一增强功能使DLP v2.0在企业网络安全防护中更加强大和实用，为安全团队提供了前所未有的网络活动洞察能力！
