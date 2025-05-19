# 插件动态加载机制指南

## 概述

AppFramework 的插件动态加载机制允许应用程序在运行时发现、加载和管理插件，无需在框架代码中硬编码插件名称。这种机制提供了更好的灵活性和可扩展性，使第三方开发者能够轻松地开发和集成自己的插件。

## 核心组件

### 1. 插件注册表 (PluginRegistry)

插件注册表是插件动态加载机制的核心组件，负责管理可用的插件。它提供以下功能：

- **插件注册**：注册新的插件配置
- **插件发现**：自动扫描插件目录，发现可用的插件
- **插件查询**：获取已注册的插件信息
- **插件列表**：列出所有已注册的插件

### 2. 插件配置 (PluginConfig)

插件配置包含插件的元数据和运行时配置，包括：

- **ID**：插件的唯一标识符
- **名称**：插件的显示名称
- **版本**：插件的版本号
- **路径**：插件的路径
- **隔离级别**：插件的隔离级别（无隔离、进程隔离、容器隔离）
- **自动启动**：是否在加载后自动启动
- **自动重启**：是否在崩溃后自动重启
- **启用状态**：是否启用

### 3. 插件管理器 (PluginManager)

插件管理器负责插件的生命周期管理，包括：

- **加载插件**：加载插件到内存
- **启动插件**：启动插件
- **停止插件**：停止插件
- **重启插件**：重启插件
- **卸载插件**：卸载插件
- **健康检查**：检查插件的健康状态

## 配置方式

AppFramework 支持两种插件配置方式：

### 1. 新版配置方式

新版配置方式使用 `plugins` 配置节，每个插件有自己的配置节，包含插件的元数据和特定配置：

```yaml
# 插件管理配置
plugin_manager:
  # 插件目录
  plugin_dir: "plugins"
  
  # 插件发现
  discovery:
    scan_interval: 60  # 秒
    auto_load: true
    follow_symlinks: false

# 插件配置
plugins:
  # 资产管理插件
  assets:
    enabled: true
    name: "资产管理插件"
    version: "1.0.0"
    path: "assets"
    auto_start: true
    auto_restart: true
    isolation_level: "none"
    # 插件特定配置
    settings:
      collect_interval: 3600
      report_server: "https://example.com/api/assets"
      auto_report: false
```

### 2. 旧版配置方式（兼容模式）

旧版配置方式使用 `enable_xxx` 配置项来控制插件的启用状态：

```yaml
# 插件目录
plugin_dir: "app"

# 模块启用配置
enable_assets: true
enable_device: true
enable_dlp: true
enable_control: true
enable_audit: true
```

## 插件发现机制

插件发现机制会自动扫描插件目录，发现可用的插件：

1. 扫描插件目录中的所有子目录
2. 检查每个子目录是否包含有效的插件
3. 创建插件配置并注册到插件注册表

插件发现可以通过以下方式配置：

```yaml
plugin_manager:
  discovery:
    scan_interval: 60  # 扫描间隔（秒）
    auto_load: true    # 是否自动加载发现的插件
    follow_symlinks: false  # 是否跟随符号链接
```

## 插件加载流程

1. 应用程序启动时，初始化插件注册表和插件管理器
2. 从配置中加载插件配置
3. 如果使用新版配置方式，直接使用配置中的插件配置
4. 如果使用旧版配置方式，通过插件发现机制发现可用的插件
5. 根据配置的启用状态，加载和启动插件

## 插件隔离

AppFramework 支持三种插件隔离级别：

1. **无隔离 (none)**：插件在主进程中运行，共享主进程的内存空间
2. **进程隔离 (process)**：插件在独立的进程中运行，通过 IPC 与主进程通信
3. **容器隔离 (container)**：插件在独立的容器中运行，提供更强的隔离性

隔离级别可以在插件配置中指定：

```yaml
plugins:
  example:
    isolation_level: "process"  # none, process, container
```

## 插件生命周期

插件的生命周期包括以下阶段：

1. **初始化**：插件被加载到内存
2. **启动**：插件开始运行
3. **运行**：插件正常运行
4. **暂停**：插件暂时停止运行
5. **停止**：插件停止运行
6. **卸载**：插件从内存中卸载

插件生命周期可以通过以下方式配置：

```yaml
plugin_manager:
  lifecycle:
    startup_timeout: 30  # 启动超时（秒）
    shutdown_timeout: 30  # 关闭超时（秒）
    health_check_interval: 60  # 健康检查间隔（秒）
    auto_restart: true  # 是否自动重启
    max_restarts: 3  # 最大重启次数
    restart_delay: 5  # 重启延迟（秒）
```

## 开发自定义插件

开发自定义插件需要实现 `plugin.Module` 接口：

```go
type Module interface {
    // GetInfo 返回插件信息
    GetInfo() ModuleInfo
    
    // Init 初始化插件
    Init(ctx context.Context, config *ModuleConfig) error
    
    // Start 启动插件
    Start() error
    
    // Stop 停止插件
    Stop() error
    
    // Shutdown 关闭插件
    Shutdown() error
}
```

插件目录结构示例：

```
plugins/
  my-plugin/
    plugin.json  # 插件元数据
    main.go      # 插件入口点
    ...          # 其他插件文件
```

插件元数据示例：

```json
{
  "id": "my-plugin",
  "name": "我的插件",
  "version": "1.0.0",
  "description": "这是一个示例插件",
  "entry_point": {
    "type": "go",
    "path": "main.go"
  },
  "dependencies": [],
  "isolation_level": "process"
}
```

## 最佳实践

1. **使用新版配置方式**：新版配置方式提供了更多的灵活性和功能
2. **启用插件发现**：启用插件发现可以自动发现新的插件
3. **使用适当的隔离级别**：根据插件的需求选择适当的隔离级别
4. **实现健康检查**：实现健康检查可以提高插件的可靠性
5. **处理依赖关系**：正确处理插件之间的依赖关系
6. **提供完整的元数据**：提供完整的插件元数据可以帮助用户理解插件的功能和用途
7. **遵循命名约定**：遵循命名约定可以提高代码的可读性和可维护性
