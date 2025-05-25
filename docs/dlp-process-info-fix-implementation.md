# DLP进程信息获取修复实施方案

## 1. 问题定位总结

### 1.1 核心问题
通过详细分析发现，DLP系统中"进程名称显示为本应用自己的进程"的问题根源在于：

1. **ProcessTracker.GetProcessByConnection()** 方法返回PID为0
2. **审计日志记录** 缺少进程信息字段
3. **网络连接表查询** 逻辑存在缺陷
4. **WinDivert数据包** 与系统连接表关联失败

### 1.2 影响范围
- 审计日志无法追踪数据流量的源头进程
- 安全分析人员无法确定哪个应用程序发起了网络请求
- 违背了DLP系统的核心安全审计要求

## 2. 修复方案设计

### 2.1 ProcessTracker增强方案

#### 当前问题代码：
```go
// app/dlp/interceptor/process_tracker.go
func (pt *ProcessTracker) GetProcessByConnection(protocol interceptor.Protocol, ip net.IP, port uint16) uint32 {
    // 当前实现可能存在问题，总是返回0
    return 0
}
```

#### 修复方案：
1. **增强Windows API调用**：使用GetExtendedTcpTable和GetExtendedUdpTable
2. **实现精确匹配算法**：基于IP地址、端口、协议的多维度匹配
3. **添加连接状态跟踪**：实时监控网络连接变化

### 2.2 审计日志增强方案

#### 当前问题代码：
```go
// app/dlp/engine/audit.go:84-89
if decision.Context.PacketInfo != nil {
    auditLog.Details["source_ip"] = decision.Context.PacketInfo.SourceIP.String()
    auditLog.Details["dest_ip"] = decision.Context.PacketInfo.DestIP.String()
    auditLog.Details["protocol"] = decision.Context.PacketInfo.Protocol
    // 缺失：进程信息记录
}
```

#### 修复方案：
1. **扩展审计日志字段**：添加进程信息相关字段
2. **增强数据记录逻辑**：包含完整的网络连接和进程信息
3. **优化日志结构**：便于后续分析和查询

### 2.3 网络连接跟踪优化方案

#### 修复策略：
1. **多层次匹配**：精确匹配 → 模糊匹配 → 缓存查询
2. **实时连接监控**：定期更新连接表，捕获连接状态变化
3. **智能缓存机制**：缓存活跃连接，减少API调用开销

## 3. 具体实现方案

### 3.1 增强ProcessTracker实现

#### 新增Windows API调用：
```go
// 增强的网络连接表查询
func (pt *ProcessTracker) getNetworkConnections() error {
    // TCP连接表
    tcpTable, err := pt.getTcpTable()
    if err != nil {
        return err
    }
    
    // UDP连接表  
    udpTable, err := pt.getUdpTable()
    if err != nil {
        return err
    }
    
    // 更新内部连接映射
    pt.updateConnectionMap(tcpTable, udpTable)
    return nil
}

// 精确的连接匹配算法
func (pt *ProcessTracker) findProcessByConnection(protocol interceptor.Protocol, localIP net.IP, localPort uint16, remoteIP net.IP, remotePort uint16) uint32 {
    // 1. 精确匹配：协议+本地IP+本地端口+远程IP+远程端口
    if pid := pt.exactMatch(protocol, localIP, localPort, remoteIP, remotePort); pid != 0 {
        return pid
    }
    
    // 2. 本地匹配：协议+本地IP+本地端口
    if pid := pt.localMatch(protocol, localIP, localPort); pid != 0 {
        return pid
    }
    
    // 3. 端口匹配：协议+本地端口
    if pid := pt.portMatch(protocol, localPort); pid != 0 {
        return pid
    }
    
    return 0
}
```

#### 进程信息获取增强：
```go
// 获取详细进程信息
func (pt *ProcessTracker) getDetailedProcessInfo(pid uint32) *ProcessInfo {
    processInfo := &ProcessInfo{
        PID: pid,
    }
    
    // 获取进程句柄
    handle, err := pt.openProcess(pid)
    if err != nil {
        pt.logger.Debug("打开进程失败", "pid", pid, "error", err)
        return processInfo
    }
    defer pt.closeProcess(handle)
    
    // 获取进程名称
    if name, err := pt.getProcessName(handle); err == nil {
        processInfo.ProcessName = name
    }
    
    // 获取可执行文件路径
    if path, err := pt.getProcessPath(handle); err == nil {
        processInfo.ExecutePath = path
    }
    
    // 获取命令行参数
    if cmdline, err := pt.getProcessCommandLine(pid); err == nil {
        processInfo.CommandLine = cmdline
    }
    
    // 获取用户信息
    if user, err := pt.getProcessUser(handle); err == nil {
        processInfo.UserName = user
    }
    
    // 获取会话ID
    if sessionId, err := pt.getProcessSessionId(pid); err == nil {
        processInfo.SessionID = sessionId
    }
    
    return processInfo
}
```

### 3.2 审计日志增强实现

#### 扩展审计日志结构：
```go
// 增强的审计日志记录
func (al *AuditLoggerImpl) LogDecision(decision *PolicyDecision) error {
    al.mu.Lock()
    defer al.mu.Unlock()

    auditLog := &AuditLog{
        ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
        Timestamp: time.Now(),
        Type:      "policy_decision",
        Action:    decision.Action.String(),
        Result:    "success",
        Details:   make(map[string]interface{}),
    }

    // 基本决策信息
    auditLog.Details["decision_id"] = decision.ID
    auditLog.Details["risk_level"] = decision.RiskLevel.String()
    auditLog.Details["risk_score"] = decision.RiskScore
    auditLog.Details["confidence"] = decision.Confidence
    auditLog.Details["matched_rules"] = len(decision.MatchedRules)
    auditLog.Details["processing_time"] = decision.ProcessingTime.String()
    auditLog.Details["reason"] = decision.Reason

    // 从上下文中提取详细信息
    if decision.Context != nil {
        // 用户和设备信息
        if decision.Context.UserInfo != nil {
            auditLog.UserID = decision.Context.UserInfo.ID
            auditLog.Details["username"] = decision.Context.UserInfo.Username
            auditLog.Details["user_department"] = decision.Context.UserInfo.Department
            auditLog.Details["user_role"] = decision.Context.UserInfo.Role
        }
        
        if decision.Context.DeviceInfo != nil {
            auditLog.DeviceID = decision.Context.DeviceInfo.ID
            auditLog.Details["device_name"] = decision.Context.DeviceInfo.Name
            auditLog.Details["device_os"] = decision.Context.DeviceInfo.OS
        }

        // 网络数据包信息
        if decision.Context.PacketInfo != nil {
            packet := decision.Context.PacketInfo
            
            // 基本网络信息
            auditLog.Details["source_ip"] = packet.SourceIP.String()
            auditLog.Details["source_port"] = packet.SourcePort
            auditLog.Details["dest_ip"] = packet.DestIP.String()
            auditLog.Details["dest_port"] = packet.DestPort
            auditLog.Details["protocol"] = packet.Protocol.String()
            auditLog.Details["direction"] = packet.Direction.String()
            auditLog.Details["packet_size"] = packet.Size
            
            // 关键修复：添加进程信息
            if packet.ProcessInfo != nil {
                auditLog.Details["process_pid"] = packet.ProcessInfo.PID
                auditLog.Details["process_name"] = packet.ProcessInfo.ProcessName
                auditLog.Details["process_path"] = packet.ProcessInfo.ExecutePath
                auditLog.Details["process_cmdline"] = packet.ProcessInfo.CommandLine
                auditLog.Details["process_user"] = packet.ProcessInfo.UserName
                auditLog.Details["process_session_id"] = packet.ProcessInfo.SessionID
            } else {
                // 记录进程信息获取失败
                auditLog.Details["process_info_status"] = "failed_to_retrieve"
                auditLog.Details["process_error"] = "无法获取进程信息"
            }
        }

        // 解析数据信息
        if decision.Context.ParsedData != nil {
            parsed := decision.Context.ParsedData
            auditLog.Details["data_type"] = parsed.DataType.String()
            auditLog.Details["content_type"] = parsed.ContentType
            auditLog.Details["data_size"] = parsed.Size
            
            // 请求详情（如果是HTTP/HTTPS）
            if parsed.Metadata != nil {
                if url, exists := parsed.Metadata["url"]; exists {
                    auditLog.Details["request_url"] = url
                }
                if method, exists := parsed.Metadata["method"]; exists {
                    auditLog.Details["request_method"] = method
                }
                if headers, exists := parsed.Metadata["headers"]; exists {
                    auditLog.Details["request_headers"] = headers
                }
                if domain, exists := parsed.Metadata["domain"]; exists {
                    auditLog.Details["dest_domain"] = domain
                }
            }
            
            // 数据摘要（安全考虑，只记录摘要）
            if len(parsed.Content) > 0 {
                summary := al.generateDataSummary(parsed.Content)
                auditLog.Details["data_summary"] = summary
            }
        }

        // 分析结果信息
        if decision.Context.AnalysisResult != nil {
            analysis := decision.Context.AnalysisResult
            auditLog.Details["analysis_risk_score"] = analysis.RiskScore
            auditLog.Details["analysis_confidence"] = analysis.Confidence
            auditLog.Details["detected_patterns"] = len(analysis.DetectedPatterns)
            auditLog.Details["analysis_categories"] = analysis.Categories
        }
    }

    return al.writeLog(auditLog)
}

// 生成数据摘要（用于审计，不记录敏感内容）
func (al *AuditLoggerImpl) generateDataSummary(content []byte) string {
    const maxSummaryLength = 200
    
    if len(content) == 0 {
        return "empty"
    }
    
    // 生成内容摘要，避免记录敏感数据
    summary := fmt.Sprintf("size:%d bytes", len(content))
    
    // 检查是否为文本内容
    if al.isTextContent(content) {
        textContent := string(content)
        if len(textContent) > maxSummaryLength {
            summary += fmt.Sprintf(", preview:%s...", textContent[:maxSummaryLength])
        } else {
            summary += fmt.Sprintf(", content:%s", textContent)
        }
    } else {
        summary += ", type:binary"
    }
    
    return summary
}

// 检查是否为文本内容
func (al *AuditLoggerImpl) isTextContent(content []byte) bool {
    // 简单的文本检测逻辑
    for _, b := range content {
        if b < 32 && b != 9 && b != 10 && b != 13 { // 排除制表符、换行符、回车符
            return false
        }
    }
    return true
}
```

### 3.3 WinDivert拦截器优化

#### 增强getProcessInfo方法：
```go
// 增强的进程信息获取
func (w *WinDivertInterceptorImpl) getProcessInfo(packet *PacketInfo) *ProcessInfo {
    // 多策略进程查找
    var pid uint32
    
    // 策略1：基于连接四元组精确匹配
    if packet.Direction == PacketDirectionOutbound {
        pid = w.processTracker.GetProcessByConnection(
            packet.Protocol, 
            packet.SourceIP, packet.SourcePort,
            packet.DestIP, packet.DestPort,
        )
    } else {
        pid = w.processTracker.GetProcessByConnection(
            packet.Protocol,
            packet.DestIP, packet.DestPort,
            packet.SourceIP, packet.SourcePort,
        )
    }
    
    // 策略2：如果精确匹配失败，尝试本地端口匹配
    if pid == 0 {
        if packet.Direction == PacketDirectionOutbound {
            pid = w.processTracker.GetProcessByLocalPort(packet.Protocol, packet.SourcePort)
        } else {
            pid = w.processTracker.GetProcessByLocalPort(packet.Protocol, packet.DestPort)
        }
    }
    
    // 策略3：如果仍然失败，尝试从缓存查找
    if pid == 0 {
        pid = w.processTracker.GetProcessFromCache(packet.SourceIP, packet.SourcePort)
    }
    
    if pid == 0 {
        w.logger.Debug("未找到对应的进程",
            "direction", packet.Direction,
            "protocol", packet.Protocol,
            "source", fmt.Sprintf("%s:%d", packet.SourceIP, packet.SourcePort),
            "dest", fmt.Sprintf("%s:%d", packet.DestIP, packet.DestPort))
        return nil
    }
    
    // 获取详细进程信息
    processInfo := w.processTracker.GetDetailedProcessInfo(pid)
    if processInfo == nil {
        w.logger.Debug("获取进程详细信息失败", "pid", pid)
        return nil
    }
    
    w.logger.Debug("成功获取进程信息",
        "pid", processInfo.PID,
        "name", processInfo.ProcessName,
        "path", processInfo.ExecutePath,
        "user", processInfo.UserName)
    
    return processInfo
}
```

## 4. 实施步骤

### 步骤1：修复ProcessTracker（优先级：高）
1. 实现增强的Windows API调用
2. 添加多策略连接匹配算法
3. 实现详细进程信息获取

### 步骤2：增强审计日志（优先级：高）
1. 扩展审计日志数据结构
2. 添加进程信息记录逻辑
3. 增加网络连接详细信息

### 步骤3：优化WinDivert拦截器（优先级：中）
1. 改进getProcessInfo方法
2. 添加多策略进程查找
3. 优化错误处理和日志记录

### 步骤4：测试验证（优先级：高）
1. 单元测试：各个组件功能验证
2. 集成测试：端到端流程验证
3. 性能测试：确保不影响系统性能

## 5. 预期效果

修复完成后，审计日志将包含完整的进程信息：

```json
{
  "id": "audit_1748095943539331600",
  "timestamp": "2025-05-24T22:12:23.5393316+08:00",
  "type": "policy_decision",
  "action": "audit",
  "user_id": "user123",
  "device_id": "device456",
  "details": {
    "source_ip": "192.168.1.100",
    "source_port": 12345,
    "dest_ip": "8.8.8.8",
    "dest_port": 443,
    "protocol": "TCP",
    "direction": "outbound",
    "process_pid": 1234,
    "process_name": "chrome.exe",
    "process_path": "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
    "process_user": "DOMAIN\\username",
    "request_url": "https://www.google.com",
    "request_method": "GET"
  }
}
```

这将使DLP系统能够准确追踪每个网络请求的源头进程，满足生产级安全审计的要求。
