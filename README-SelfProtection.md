# Kennel自我防护机制 🛡️

[![Go Version](https://img.shields.io/badge/Go-1.16+-blue.svg)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-green.svg)](https://github.com/lomehong/kennel)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](https://github.com/lomehong/kennel)

## 🎯 项目概述

Kennel自我防护机制是为终端安全管理系统设计的企业级自我保护解决方案，提供全方位的系统防护能力，确保主程序和关键插件在各种威胁环境下的持续稳定运行。

### ✨ 核心特性

- 🔒 **多层防护**: 进程、文件、注册表、服务四层全覆盖防护
- 🚀 **高性能**: 低资源消耗，异步处理，智能监控
- 🔧 **灵活配置**: 支持多种防护级别和详细配置选项
- 🌐 **跨平台**: Windows完整支持，Linux/macOS基础支持
- 📊 **可监控**: 完整的API接口、Web界面和事件系统
- 🛠️ **易集成**: 简单的集成方式和丰富的示例代码

## 🏗️ 架构设计

### 防护层次架构
```
┌─────────────────────────────────────────────────────────────┐
│                    应用程序层                                │
├─────────────────────────────────────────────────────────────┤
│                  自我防护管理层                              │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐   │
│  │  进程防护   │  文件防护   │ 注册表防护  │  服务防护   │   │
│  └─────────────┴─────────────┴─────────────┴─────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                    系统API层                                │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐   │
│  │ Windows API │ File System │  Registry   │  Services   │   │
│  └─────────────┴─────────────┴─────────────┴─────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                    操作系统层                               │
└─────────────────────────────────────────────────────────────┘
```

### 模块组织结构
```
pkg/core/selfprotect/
├── 核心模块
│   ├── types.go           # 类型定义
│   ├── protection.go      # 防护管理器
│   ├── config.go          # 配置管理
│   └── integration.go     # 集成服务
├── 防护实现
│   ├── process_protector_*.go   # 进程防护
│   ├── file_protector.go        # 文件防护
│   ├── registry_protector_*.go  # 注册表防护
│   └── service_protector_*.go   # 服务防护
├── 接口定义
│   ├── interfaces.go      # 防护接口
│   └── disabled.go        # 禁用实现
└── API服务
    └── api.go             # REST API和Web界面
```

## 🚀 快速开始

### 1. 编译启用自我防护

```bash
# 启用自我防护功能编译
go build -tags="selfprotect" -o bin/agent.exe cmd/agent/main.go

# 使用高级构建脚本
.\build-with-protection.ps1 -EnableProtection -Target all -Release
```

### 2. 配置自我防护

在 `config.yaml` 中添加自我防护配置：

```yaml
# 自我防护配置
self_protection:
  # 启用自我防护
  enabled: true
  # 防护级别：none, basic, standard, strict
  level: "basic"
  # 紧急禁用文件
  emergency_disable: ".emergency_disable"
  # 检查间隔
  check_interval: "5s"

  # 进程防护
  process_protection:
    enabled: true
    protected_processes:
      - "agent.exe"
      - "dlp.exe"
      - "audit.exe"
      - "device.exe"
    prevent_debug: true
    prevent_dump: true

  # 文件防护
  file_protection:
    enabled: true
    protected_files:
      - "config.yaml"
      - "agent.exe"
    protected_dirs:
      - "app"
    backup_enabled: true

  # 注册表防护（Windows）
  registry_protection:
    enabled: true
    protected_keys:
      - "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\KennelAgent"

  # 服务防护（Windows）
  service_protection:
    enabled: true
    service_name: "KennelAgent"
    auto_restart: true
```

### 3. 集成到应用程序

```go
package main

import (
    "github.com/lomehong/kennel/pkg/core/selfprotect"
    "github.com/hashicorp/go-hclog"
)

func main() {
    logger := hclog.Default()
    
    // 创建防护集成器
    integrator, err := selfprotect.NewProtectionIntegrator("config.yaml", logger)
    if err != nil {
        logger.Error("创建防护集成器失败", "error", err)
        return
    }
    
    // 初始化防护
    if err := integrator.Initialize(); err != nil {
        logger.Error("启动自我防护失败", "error", err)
        // 注意：防护失败不应阻止程序启动
    }
    
    // 优雅关闭
    defer integrator.Shutdown()
    
    // 你的应用程序逻辑...
    runApplication()
}
```

### 4. 运行测试验证

```bash
# 运行自我防护功能测试
go run -tags="selfprotect" cmd/selfprotect-test/main.go -verbose

# 运行集成示例
go run -tags="selfprotect" examples/selfprotect-integration/main.go
```

## 📋 功能特性

### 🔐 进程防护
- **实时监控**: 监控受保护进程的运行状态
- **自动重启**: 检测到进程终止时自动重启
- **调试防护**: 防止进程被调试器附加
- **内存保护**: 防止进程内存被转储
- **权限提升**: 自动提升必要的系统权限

### 📁 文件防护
- **文件监控**: 实时监控文件修改、删除、重命名
- **完整性检查**: MD5和SHA256双重校验
- **自动备份**: 自动备份受保护的文件
- **自动恢复**: 检测到篡改时自动恢复
- **目录保护**: 支持整个目录的递归保护

### 🗂️ 注册表防护（Windows）
- **注册表监控**: 监控关键注册表项变更
- **自动备份**: 备份重要注册表项
- **自动恢复**: 检测到篡改时自动恢复
- **多根键支持**: 支持HKLM、HKCU等多个根键

### ⚙️ 服务防护（Windows）
- **服务监控**: 监控服务状态变化
- **自动重启**: 服务被停止时自动重启
- **防止禁用**: 防止服务被恶意禁用
- **状态检查**: 定期检查服务运行状态

## 🔧 配置选项

### 防护级别

| 级别 | 说明 | 适用场景 |
|------|------|----------|
| `none` | 无防护 | 开发测试环境 |
| `basic` | 基础防护 | 一般生产环境（推荐） |
| `standard` | 标准防护 | 高安全要求环境 |
| `strict` | 严格防护 | 极高安全要求环境 |

### 白名单配置

```yaml
whitelist:
  enabled: true
  # 进程白名单
  processes:
    - "taskmgr.exe"
    - "procexp.exe"
  # 用户白名单
  users:
    - "SYSTEM"
    - "Administrator"
  # 签名白名单
  signatures:
    - "Microsoft Corporation"
```

### 紧急禁用机制

```bash
# 方法1: 创建紧急禁用文件
echo "emergency disable" > .emergency_disable

# 方法2: 修改配置文件
# 将 enabled: false

# 方法3: 使用API接口
curl -X POST http://localhost:8080/api/protection/disable \
  -H "Content-Type: application/json" \
  -d '{"reason": "maintenance", "temporary": true}'
```

## 📊 监控和管理

### REST API接口

```bash
# 获取防护状态
GET /api/protection/status

# 获取防护配置
GET /api/protection/config

# 获取防护事件
GET /api/protection/events?page=1&limit=20

# 获取防护统计
GET /api/protection/stats

# 生成防护报告
GET /api/protection/report

# 健康检查
GET /api/protection/health
```

### Web管理界面

访问 `http://localhost:8080/protection` 查看Web管理界面：

- **仪表板**: 防护状态概览
- **事件管理**: 防护事件查询和分析
- **配置管理**: 防护配置查看和修改
- **健康监控**: 系统健康状态监控

### 命令行工具

```bash
# 查看防护状态
.\bin\agent.exe protection status

# 查看防护事件
.\bin\agent.exe protection events --recent

# 生成防护报告
.\bin\agent.exe protection report --format json

# 运行健康检查
.\bin\agent.exe protection health
```

## 🧪 测试验证

### 功能测试

```bash
# 运行完整测试套件
go run -tags="selfprotect" cmd/selfprotect-test/main.go -verbose

# 测试结果示例
Kennel自我防护测试工具 v1.0.0
=====================================

1. 测试配置加载
   ✓ 配置加载测试通过

2. 测试防护管理器初始化
   ✓ 防护管理器初始化测试通过

3. 测试防护组件
   ✓ 防护组件测试通过

4. 测试紧急禁用机制
   ✓ 紧急禁用机制测试通过

5. 测试防护事件
   ✓ 防护事件测试通过

✓ 所有自我防护测试通过
```

### 性能测试

```bash
# 性能基准测试
go test -tags="selfprotect" -bench=. ./pkg/core/selfprotect/...

# 内存使用监控
go run -tags="selfprotect" examples/performance-monitor/main.go
```

## 📈 性能指标

### 资源消耗
- **内存占用**: +5-10MB（防护状态和事件缓存）
- **CPU开销**: +2-5%（监控和检查）
- **磁盘I/O**: 轻微增加（备份和日志）
- **网络影响**: 无

### 性能优化
- **异步处理**: 所有防护检查都在后台异步执行
- **智能间隔**: 可配置的检查间隔，平衡性能和实时性
- **事件缓存**: 限制事件数量，防止内存泄漏
- **批量操作**: 批量处理文件和注册表操作

## 🌍 平台兼容性

### Windows平台（完整支持）
- ✅ 进程防护（完整实现）
- ✅ 文件防护（完整实现）
- ✅ 注册表防护（完整实现）
- ✅ 服务防护（完整实现）
- 📋 支持版本：Windows 7, 8, 10, 11, Server 2008+

### Linux平台（基础支持）
- ⚠️ 进程防护（可扩展）
- ✅ 文件防护（完整实现）
- ❌ 注册表防护（不适用）
- ⚠️ 服务防护（可扩展）

### macOS平台（基础支持）
- ⚠️ 进程防护（可扩展）
- ✅ 文件防护（完整实现）
- ❌ 注册表防护（不适用）
- ⚠️ 服务防护（可扩展）

## 🔒 安全考虑

### 威胁模型
- **恶意进程终止**: 自动重启被终止的进程
- **文件篡改攻击**: 自动恢复被修改的文件
- **配置破坏**: 自动恢复被破坏的配置
- **服务攻击**: 自动重启被停止的服务
- **调试攻击**: 防止进程被调试器附加
- **内存转储**: 防止进程内存被转储

### 安全机制
- **多重验证**: 文件完整性双重校验
- **权限控制**: 最小权限原则
- **白名单机制**: 允许授权操作
- **审计日志**: 完整的操作审计
- **紧急禁用**: 安全的禁用机制

## 🛠️ 故障排除

### 常见问题

#### 1. 权限不足
```
错误: Access is denied.
解决: 以管理员权限运行程序
```

#### 2. 进程防护失败
```
错误: 设置进程保护失败
解决: 检查Windows版本兼容性，尝试降低防护级别
```

#### 3. 文件监控失败
```
错误: too many open files
解决: 减少监控文件数量，增加系统文件句柄限制
```

### 调试方法

```yaml
# 启用详细日志
logging:
  level: "debug"

# 使用测试工具
go run -tags="selfprotect" cmd/selfprotect-test/main.go -verbose

# 检查系统事件
# Windows: 事件查看器 -> 应用程序日志
# Linux: journalctl -u kennel-agent
```

## 📚 文档资源

- 📖 [实施报告](docs/self-protection-implementation-report.md) - 详细的技术实施报告
- 📋 [使用指南](docs/self-protection-usage-guide.md) - 完整的使用指南
- 📊 [功能总结](docs/self-protection-summary.md) - 功能特性总结
- 💡 [集成示例](examples/selfprotect-integration/) - 完整的集成示例
- 🔧 [API文档](docs/api-reference.md) - REST API参考文档

## 🤝 贡献指南

我们欢迎社区贡献！请遵循以下步骤：

1. Fork 项目仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

### 开发环境设置

```bash
# 克隆仓库
git clone https://github.com/lomehong/kennel.git
cd kennel

# 安装依赖
go mod download

# 运行测试
go test -tags="selfprotect" ./...

# 构建项目
.\build-with-protection.ps1 -EnableProtection -Target all
```

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

感谢以下开源项目的支持：
- [fsnotify](https://github.com/fsnotify/fsnotify) - 文件系统监控
- [go-hclog](https://github.com/hashicorp/go-hclog) - 结构化日志
- [gorilla/mux](https://github.com/gorilla/mux) - HTTP路由
- [golang.org/x/sys](https://golang.org/x/sys) - 系统调用

## 📞 联系我们

- 📧 Email: support@kennel-security.com
- 🐛 Issues: [GitHub Issues](https://github.com/lomehong/kennel/issues)
- 💬 Discussions: [GitHub Discussions](https://github.com/lomehong/kennel/discussions)

---

<div align="center">

**🛡️ Kennel自我防护机制 - 为您的终端安全保驾护航！**

[![Stars](https://img.shields.io/github/stars/lomehong/kennel?style=social)](https://github.com/lomehong/kennel/stargazers)
[![Forks](https://img.shields.io/github/forks/lomehong/kennel?style=social)](https://github.com/lomehong/kennel/network/members)
[![Issues](https://img.shields.io/github/issues/lomehong/kennel)](https://github.com/lomehong/kennel/issues)

</div>
