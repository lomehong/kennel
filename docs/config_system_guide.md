# Kennel 配置系统使用指南

## 概述

Kennel 配置系统采用分层设计，支持全局配置、插件管理配置和插件特定配置。本指南将帮助您理解和使用 Kennel 的配置系统。

## 配置文件格式

Kennel 支持 YAML 和 JSON 格式的配置文件，推荐使用 YAML 格式，因为它更易于阅读和编辑。

### YAML 格式示例

```yaml
# 全局配置
global:
  app:
    name: "kennel"
    version: "1.0.0"
  logging:
    level: "info"
    file: "logs/kennel.log"

# 插件管理配置
plugin_manager:
  plugin_dir: "plugins"
  discovery:
    auto_load: true

# 插件配置
plugins:
  my-plugin:
    enabled: true
    option1: "value1"
    option2: 42
```

### JSON 格式示例

```json
{
  "global": {
    "app": {
      "name": "kennel",
      "version": "1.0.0"
    },
    "logging": {
      "level": "info",
      "file": "logs/kennel.log"
    }
  },
  "plugin_manager": {
    "plugin_dir": "plugins",
    "discovery": {
      "auto_load": true
    }
  },
  "plugins": {
    "my-plugin": {
      "enabled": true,
      "option1": "value1",
      "option2": 42
    }
  }
}
```

## 配置层次结构

Kennel 配置系统分为三个层次：

1. **全局配置**：适用于整个应用程序的基础配置
2. **插件管理配置**：控制插件系统行为的配置
3. **插件特定配置**：每个插件的独立配置

### 全局配置

全局配置包含以下内容：

```yaml
global:
  # 应用程序基本信息
  app:
    name: "kennel"
    version: "1.0.0"
    description: "跨平台终端代理框架"
  
  # 日志配置
  logging:
    level: "info"  # trace, debug, info, warn, error, off
    file: "logs/kennel.log"
    format: "json"  # json, text
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
plugins:
  # 插件ID
  my-plugin:
    # 是否启用插件
    enabled: true
    
    # 插件特定配置
    option1: "value1"
    option2: 42
    nested:
      key1: "value1"
      key2: "value2"
```

## 配置文件位置

配置文件按以下优先级顺序加载：

1. 命令行指定的配置文件
2. 当前目录下的 `config.yaml`
3. 用户目录下的 `.kennel/config.yaml`
4. 系统配置目录下的 `kennel/config.yaml`

## 命令行参数

可以通过命令行参数指定配置文件路径：

```bash
kennel --config path/to/config.yaml
```

## 环境变量

环境变量可以覆盖配置，格式为 `KENNEL_SECTION_KEY`，例如：

- `KENNEL_GLOBAL_LOGGING_LEVEL=debug`
- `KENNEL_PLUGINS_MY_PLUGIN_ENABLED=false`

## 配置热加载

Kennel 支持配置热加载，当配置文件发生变化时，自动重新加载配置。这意味着您可以在不重启应用程序的情况下修改配置。

### 热加载行为

1. 当配置文件发生变化时，配置管理器会检测到变化并重新加载配置
2. 配置变更监听器会收到通知，并根据变更类型执行相应操作
3. 对于插件配置的变更，如果插件已加载，会重新初始化插件

### 配置变更监听器

您可以注册配置变更监听器来响应配置变更：

```go
// 添加配置变更监听器
configManager.AddConfigChangeListener(func(configType string, oldConfig, newConfig map[string]interface{}) error {
    fmt.Printf("配置变更: %s\n", configType)
    return nil
})
```

## 配置验证

Kennel 提供配置验证机制，确保配置符合预期：

```go
// 创建插件配置验证器
validator := config.NewPluginConfigValidator("my-plugin")

// 添加必需字段
validator.AddRequiredField("enabled")

// 添加字段类型
validator.AddFieldType("option1", reflect.String)
validator.AddFieldType("option2", reflect.Int)

// 添加字段验证器
validator.AddFieldValidator("option1", config.StringValidator("value1", "value2"))
validator.AddFieldValidator("option2", config.IntRangeValidator(1, 100))

// 添加默认值
validator.AddDefault("timeout", 30)

// 注册验证器
configManager.AddValidator(validator)
```

## 配置访问

### 在代码中访问配置

在代码中可以通过配置管理器访问配置：

```go
// 获取全局配置
globalConfig := configManager.GetGlobalConfig()

// 获取插件管理配置
pluginManagerConfig := configManager.GetPluginManagerConfig()

// 获取插件配置
pluginConfig := configManager.GetPluginConfig("my-plugin")
```

### 在插件中访问配置

在插件中可以通过初始化参数访问配置：

```go
// 在插件初始化时接收配置
func (p *MyPlugin) Init(ctx context.Context, config *plugin.ModuleConfig) error {
    // 访问插件特定配置
    settings := config.Settings
    
    // 获取配置值
    option1, ok := settings["option1"].(string)
    if !ok {
        option1 = "default"
    }
    
    option2, ok := settings["option2"].(int)
    if !ok {
        option2 = 0
    }
    
    // 使用配置
    p.logger.Info("配置已加载", "option1", option1, "option2", option2)
    
    return nil
}
```

## 配置最佳实践

1. **使用分层配置**：将配置分为全局配置、插件管理配置和插件特定配置
2. **提供默认值**：为所有配置项提供合理的默认值
3. **验证配置**：使用配置验证器验证配置
4. **使用环境变量**：使用环境变量覆盖敏感配置
5. **配置热加载**：利用配置热加载功能动态更新配置
6. **文档化配置**：为所有配置项提供详细的文档
7. **版本控制配置**：将配置文件纳入版本控制
8. **使用配置模板**：提供配置模板，方便用户创建配置文件

## 配置示例

### 完整配置示例

```yaml
# Kennel 配置文件

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
