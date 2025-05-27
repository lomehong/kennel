# Kennel自我防护使用指南

## 概述

Kennel自我防护机制为终端安全管理系统提供了全面的自我保护能力，防止主程序和关键插件被恶意或异常终止。本指南将详细介绍如何配置、启用和使用自我防护功能。

## 快速开始

### 1. 编译启用自我防护

#### 启用自我防护功能
```bash
# 编译主程序并启用自我防护
go build -tags="selfprotect" -o bin/agent.exe cmd/agent/main.go

# 编译所有插件并启用自我防护
go build -tags="selfprotect" -o bin/app/dlp/dlp.exe app/dlp/main.go
go build -tags="selfprotect" -o bin/app/audit/audit.exe app/audit/main.go
go build -tags="selfprotect" -o bin/app/device/device.exe app/device/main.go

# 或者使用构建脚本
.\build.ps1 -Tags "selfprotect"
```

#### 禁用自我防护功能（默认）
```bash
# 正常编译，自我防护功能被禁用
go build -o bin/agent.exe cmd/agent/main.go
```

### 2. 配置自我防护

在`config.yaml`文件中配置自我防护：

```yaml
# 自我防护配置
self_protection:
  # 启用自我防护
  enabled: true
  # 防护级别：basic（推荐）
  level: "basic"
  # 紧急禁用文件
  emergency_disable: ".emergency_disable"
  # 检查间隔
  check_interval: "5s"
  # 重启延迟
  restart_delay: "3s"
  # 最大重启尝试次数
  max_restart_attempts: 3

  # 进程防护配置
  process_protection:
    enabled: true
    protected_processes:
      - "agent.exe"
      - "dlp.exe"
      - "audit.exe"
      - "device.exe"

  # 文件防护配置
  file_protection:
    enabled: true
    protected_files:
      - "config.yaml"
      - "agent.exe"
    protected_dirs:
      - "app"
    backup_enabled: true
```

### 3. 启动系统

```bash
# 以管理员权限启动
.\bin\agent.exe
```

## 详细配置

### 防护级别

| 级别 | 说明 | 适用场景 |
|------|------|----------|
| `none` | 无防护 | 开发测试环境 |
| `basic` | 基础防护 | 一般生产环境（推荐） |
| `standard` | 标准防护 | 高安全要求环境 |
| `strict` | 严格防护 | 极高安全要求环境 |

### 进程防护配置

```yaml
process_protection:
  enabled: true
  # 受保护的进程列表
  protected_processes:
    - "agent.exe"          # 主程序
    - "dlp.exe"           # DLP插件
    - "audit.exe"         # 审计插件
    - "device.exe"        # 设备管理插件
    - "control.exe"       # 终端管控插件
    - "assets.exe"        # 资产管理插件
  
  # 是否监控子进程
  monitor_children: true
  
  # 是否防止调试
  prevent_debug: true
  
  # 是否防止内存转储
  prevent_dump: true
```

### 文件防护配置

```yaml
file_protection:
  enabled: true
  
  # 受保护的文件
  protected_files:
    - "config.yaml"       # 主配置文件
    - "agent.exe"         # 主程序可执行文件
    - "license.key"       # 许可证文件
  
  # 受保护的目录
  protected_dirs:
    - "app"               # 插件目录
    - "config"            # 配置目录
    - "data"              # 数据目录
  
  # 是否检查文件完整性
  check_integrity: true
  
  # 是否启用自动备份
  backup_enabled: true
  
  # 备份目录
  backup_dir: "backup"
```

### 注册表防护配置（仅Windows）

```yaml
registry_protection:
  enabled: true
  
  # 受保护的注册表键
  protected_keys:
    - "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\KennelAgent"
    - "HKEY_LOCAL_MACHINE\\SOFTWARE\\Kennel"
    - "HKEY_CURRENT_USER\\SOFTWARE\\Kennel"
  
  # 是否监控注册表变更
  monitor_changes: true
```

### 服务防护配置（仅Windows）

```yaml
service_protection:
  enabled: true
  
  # 服务名称
  service_name: "KennelAgent"
  
  # 是否自动重启服务
  auto_restart: true
  
  # 是否防止服务被禁用
  prevent_disable: true
```

### 白名单配置

```yaml
whitelist:
  enabled: true
  
  # 白名单进程（允许这些进程操作受保护资源）
  processes:
    - "taskmgr.exe"       # 任务管理器
    - "procexp.exe"       # Process Explorer
    - "procexp64.exe"     # Process Explorer 64位
    - "perfmon.exe"       # 性能监视器
  
  # 白名单用户（允许这些用户执行管理操作）
  users:
    - "SYSTEM"            # 系统账户
    - "Administrator"     # 管理员账户
    - "Domain\\AdminUser" # 域管理员
  
  # 白名单数字签名
  signatures:
    - "Microsoft Corporation"
    - "Your Company Name"
```

## 使用场景

### 场景1: 开发环境

```yaml
self_protection:
  enabled: false  # 开发时通常禁用
  level: "none"
```

### 场景2: 测试环境

```yaml
self_protection:
  enabled: true
  level: "basic"
  
  # 只保护关键进程
  process_protection:
    enabled: true
    protected_processes:
      - "agent.exe"
  
  # 只保护关键文件
  file_protection:
    enabled: true
    protected_files:
      - "config.yaml"
```

### 场景3: 生产环境

```yaml
self_protection:
  enabled: true
  level: "standard"
  
  # 完整的进程防护
  process_protection:
    enabled: true
    protected_processes:
      - "agent.exe"
      - "dlp.exe"
      - "audit.exe"
      - "device.exe"
      - "control.exe"
    monitor_children: true
    prevent_debug: true
    prevent_dump: true
  
  # 完整的文件防护
  file_protection:
    enabled: true
    protected_files:
      - "config.yaml"
      - "agent.exe"
      - "license.key"
    protected_dirs:
      - "app"
      - "config"
      - "data"
    check_integrity: true
    backup_enabled: true
  
  # 注册表和服务防护
  registry_protection:
    enabled: true
  service_protection:
    enabled: true
```

### 场景4: 高安全环境

```yaml
self_protection:
  enabled: true
  level: "strict"
  
  # 最严格的防护配置
  check_interval: "3s"    # 更频繁的检查
  max_restart_attempts: 5 # 更多重启尝试
  
  # 扩展的白名单
  whitelist:
    enabled: true
    processes:
      - "taskmgr.exe"
      # 只允许必要的系统工具
    users:
      - "SYSTEM"
      # 只允许系统账户
```

## 管理操作

### 查看防护状态

```bash
# 查看防护状态（需要实现管理接口）
.\bin\agent.exe status --protection

# 查看防护事件
.\bin\agent.exe events --protection

# 查看防护统计
.\bin\agent.exe stats --protection
```

### 紧急禁用防护

#### 方法1: 创建紧急禁用文件
```bash
# 创建紧急禁用文件
echo "emergency disable" > .emergency_disable

# 系统检测到文件后会自动进入紧急模式
```

#### 方法2: 修改配置文件
```yaml
self_protection:
  enabled: false  # 禁用自我防护
```

#### 方法3: 使用管理命令
```bash
# 临时禁用防护（需要实现管理接口）
.\bin\agent.exe protection disable --temporary

# 永久禁用防护
.\bin\agent.exe protection disable --permanent
```

### 备份和恢复

#### 查看备份
```bash
# 查看备份文件
dir backup\

# 备份文件命名格式：原文件名.时间戳.backup
# 例如：config.yaml.20241227_143022.backup
```

#### 手动恢复
```bash
# 从备份恢复文件
copy backup\config.yaml.20241227_143022.backup config.yaml
```

## 监控和告警

### 防护事件类型

| 事件类型 | 说明 | 处理建议 |
|----------|------|----------|
| `process_terminated` | 受保护进程被终止 | 检查是否为恶意攻击 |
| `file_modified` | 受保护文件被修改 | 验证修改的合法性 |
| `file_deleted` | 受保护文件被删除 | 立即恢复文件 |
| `registry_modified` | 受保护注册表被修改 | 检查修改来源 |
| `service_stopped` | 受保护服务被停止 | 检查停止原因 |

### 日志记录

防护事件会记录在系统日志中：

```
2024-12-27T14:30:22.123+0800 [WARN]  protection-manager.process-protector: 检测到受保护进程被终止: process=dlp.exe pid=1234
2024-12-27T14:30:22.124+0800 [INFO]  protection-manager.process-protector: 进程已重启: process=dlp.exe pid=5678
2024-12-27T14:30:25.456+0800 [WARN]  protection-manager.file-protector: 检测到受保护文件被修改: file=config.yaml
2024-12-27T14:30:25.457+0800 [INFO]  protection-manager.file-protector: 文件已从备份恢复: file=config.yaml
```

## 故障排除

### 常见问题

#### 1. 权限不足错误
```
错误: 连接服务管理器失败: Access is denied.
```

**解决方案**:
- 确保以管理员权限运行程序
- 检查用户账户控制(UAC)设置

#### 2. 进程防护失败
```
错误: 设置进程保护失败: The operation completed successfully.
```

**解决方案**:
- 检查Windows版本兼容性
- 确保系统支持关键进程设置
- 尝试降低防护级别

#### 3. 文件监控失败
```
错误: 创建文件监控器失败: too many open files
```

**解决方案**:
- 减少监控的文件和目录数量
- 增加系统文件句柄限制
- 重启系统释放资源

#### 4. 注册表访问失败
```
错误: 打开注册表键失败: Access is denied.
```

**解决方案**:
- 确保有注册表访问权限
- 检查注册表键路径是否正确
- 以管理员权限运行

### 调试方法

#### 1. 启用详细日志
```yaml
logging:
  level: "debug"  # 启用调试日志
```

#### 2. 使用测试工具
```bash
# 运行自我防护测试
go run -tags="selfprotect" cmd/selfprotect-test/main.go -verbose
```

#### 3. 检查系统事件
- 查看Windows事件查看器
- 检查应用程序日志
- 查看系统日志

## 性能优化

### 配置优化

```yaml
self_protection:
  # 根据系统性能调整检查间隔
  check_interval: "10s"  # 降低检查频率以减少CPU使用
  
  # 限制事件数量
  max_events: 1000       # 减少内存使用
  
  # 优化文件监控
  file_protection:
    check_integrity: false  # 禁用完整性检查以提高性能
```

### 监控性能影响

```bash
# 监控CPU使用率
Get-Process agent | Select-Object CPU

# 监控内存使用
Get-Process agent | Select-Object WorkingSet

# 监控文件句柄数
Get-Process agent | Select-Object Handles
```

## 最佳实践

### 1. 配置管理
- 使用版本控制管理配置文件
- 定期备份配置文件
- 在测试环境验证配置变更

### 2. 监控告警
- 集成监控系统监控防护状态
- 设置关键事件告警
- 定期审查防护日志

### 3. 安全管理
- 定期更新白名单
- 审查防护事件
- 保护紧急禁用机制

### 4. 性能管理
- 监控系统性能影响
- 根据需要调整配置
- 定期清理备份文件

## 总结

Kennel自我防护机制提供了全面的系统保护能力，通过合理的配置和使用，可以显著提升系统的安全性和稳定性。建议在生产环境中启用基础或标准级别的防护，并根据实际需求调整具体配置。
