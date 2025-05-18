# Kennel 配置系统设计

## 配置系统概述

Kennel 的配置系统采用分层设计，支持全局配置、插件管理配置和插件特定配置。配置系统具有以下特性：

- **分层结构**：不同级别的配置有明确的优先级
- **命名空间隔离**：每个插件拥有独立的配置命名空间
- **动态加载**：支持配置热加载和动态更新
- **格式支持**：支持YAML和JSON格式
- **验证机制**：提供配置验证和错误报告
- **默认值**：为所有配置项提供合理的默认值
- **环境变量覆盖**：支持通过环境变量覆盖配置

## 配置层次结构

配置系统分为三个层次：

1. **全局配置**：适用于整个应用程序的基础配置
2. **插件管理配置**：控制插件系统行为的配置
3. **插件特定配置**：每个插件的独立配置

### 全局配置

全局配置包含以下内容：

```yaml
# 全局配置
global:
  # 应用程序基本信息
  app:
    name: "kennel"
    version: "1.0.0"
    description: "跨平台终端代理框架"
  
  # 日志配置
  logging:
    level: "info"
    file: "logs/kennel.log"
    format: "json"
    max_size: 10  # MB
    max_backups: 5
    max_age: 30  # 天
    compress: true
  
  # 系统配置
  system:
    temp_dir: "tmp"
    data_dir: "data"
    pid_file: "kennel.pid"
    graceful_timeout: 30  # 秒
  
  # Web控制台配置
  web_console:
    enabled: true
    host: "0.0.0.0"
    port: 8088
    enable_https: false
    cert_file: ""
    key_file: ""
    static_dir: "web/dist"
```

### 插件管理配置

插件管理配置控制插件系统的行为：

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
  
  # 插件隔离
  isolation:
    default_level: "process"  # none, process, container
    resource_limits:
      cpu: 50  # 百分比
      memory: 100  # MB
      disk: 1000  # MB
  
  # 插件生命周期
  lifecycle:
    startup_timeout: 30  # 秒
    shutdown_timeout: 30  # 秒
    health_check_interval: 60  # 秒
    auto_restart: true
    max_restarts: 3
    restart_delay: 5  # 秒
```

### 插件特定配置

每个插件拥有独立的配置命名空间：

```yaml
# 插件配置
plugins:
  # 资产管理插件
  assets:
    enabled: true
    collect_interval: 3600  # 秒
    report_server: "https://example.com/api/assets"
    auto_report: false
  
  # 设备管理插件
  device:
    enabled: true
    monitor_usb: true
    monitor_network: true
    allow_network_disable: true
  
  # 数据防泄漏插件
  dlp:
    enabled: true
    rules:
      - id: "credit-card"
        name: "信用卡号"
        pattern: "\\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13})\\b"
        action: "alert"
    monitor_clipboard: false
    monitor_files: false
```

## 配置文件位置

配置文件按以下优先级顺序加载：

1. 命令行指定的配置文件
2. 当前目录下的 `config.yaml`
3. 用户目录下的 `.kennel/config.yaml`
4. 系统配置目录下的 `kennel/config.yaml`

## 配置热加载

配置系统支持热加载，当配置文件发生变化时，自动重新加载配置：

```go
// 配置变更监听器
type ConfigChangeListener func(oldConfig, newConfig map[string]interface{}) error

// 添加配置变更监听器
func (c *ConfigManager) AddChangeListener(listener ConfigChangeListener)
```

## 配置覆盖机制

配置覆盖按以下优先级（从高到低）：

1. 命令行参数
2. 环境变量
3. 用户配置文件
4. 默认配置

### 环境变量覆盖

环境变量可以覆盖配置，格式为 `KENNEL_SECTION_KEY`，例如：

- `KENNEL_GLOBAL_LOGGING_LEVEL=debug`
- `KENNEL_PLUGINS_ASSETS_ENABLED=false`

## 插件配置访问

插件可以通过以下方式访问配置：

```go
// 在插件初始化时接收配置
func (m *MyPlugin) Init(ctx context.Context, config *ModuleConfig) error {
    // 访问插件特定配置
    collectInterval := config.Settings["collect_interval"].(int)
    
    // 使用配置
    // ...
    
    return nil
}
```

## 配置验证

配置系统提供验证机制，确保配置符合预期：

```go
// 配置验证器
type ConfigValidator interface {
    // 验证配置
    Validate(config map[string]interface{}) error
    
    // 获取默认配置
    GetDefaults() map[string]interface{}
    
    // 获取配置架构
    GetSchema() map[string]interface{}
}
```

## 配置迁移

支持配置版本迁移，处理配置格式变更：

```go
// 配置迁移器
type ConfigMigrator interface {
    // 获取源版本
    GetSourceVersion() string
    
    // 获取目标版本
    GetTargetVersion() string
    
    // 执行迁移
    Migrate(config map[string]interface{}) (map[string]interface{}, error)
}
```
