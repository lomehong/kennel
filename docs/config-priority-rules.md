# Kennel项目配置优先级规则文档

## 文档信息
- **版本**: v1.0
- **创建日期**: 2024年12月
- **文档类型**: 配置优先级规则说明
- **适用范围**: Kennel项目配置管理

## 配置优先级概述

Kennel项目采用多层次配置系统，支持多种配置来源和格式。配置加载遵循明确的优先级规则，确保配置的可预测性和一致性。

## 配置文件优先级

### 主程序配置文件优先级（从高到低）

1. **命令行指定的配置文件** (最高优先级)
   ```bash
   ./kennel --config /path/to/custom-config.yaml
   ```

2. **统一配置文件**: `config.unified.yaml`
   - 通过配置迁移工具生成的统一格式配置文件
   - 包含完整的层次化配置结构
   - 推荐的生产环境配置文件

3. **新版配置文件**: `config.new.yaml`
   - 新版层次化配置格式
   - 支持完整的插件配置管理

4. **旧版配置文件**: `config.yaml`
   - 向后兼容的扁平配置格式
   - 逐步废弃，建议迁移到新格式

5. **备用配置文件**: `config.yml`
   - YAML格式的备用配置文件

6. **用户目录配置**: `~/.appframework/config.yaml`
   - 用户级别的配置文件
   - 当前目录没有配置文件时使用

### 插件配置文件优先级（从高到低）

1. **主配置文件中的插件配置段**
   ```yaml
   plugins:
     dlp:
       settings:
         monitor_network: true
   ```

2. **插件独立配置文件**: `app/{plugin_id}/config.yaml`
   ```yaml
   # app/dlp/config.yaml
   monitor_network: true
   max_concurrency: 4
   ```

3. **插件默认配置**
   - 插件代码中定义的默认配置值

## 环境变量覆盖规则

### 主程序环境变量

环境变量具有最高优先级，可以覆盖配置文件中的任何设置。

**命名规则**: `APPFW_{配置路径}`

```bash
# 全局配置
export APPFW_GLOBAL_LOGGING_LEVEL=debug
export APPFW_GLOBAL_SYSTEM_MAX_CONCURRENCY=20

# 插件管理配置
export APPFW_PLUGIN_MANAGER_PLUGIN_DIR=custom_plugins

# Web控制台配置
export APPFW_WEB_CONSOLE_PORT=9090
export APPFW_WEB_CONSOLE_ENABLED=true

# 插件配置
export APPFW_PLUGINS_DLP_ENABLED=false
export APPFW_PLUGINS_ASSETS_SETTINGS_COLLECT_INTERVAL=7200
```

### 插件环境变量

**命名规则**: `{PLUGIN_ID}_{配置键}`

```bash
# DLP插件环境变量
export DLP_MONITOR_NETWORK=false
export DLP_MAX_CONCURRENCY=8
export DLP_BUFFER_SIZE=1000

# Assets插件环境变量
export ASSETS_COLLECT_INTERVAL=7200
export ASSETS_AUTO_REPORT=true

# Control插件环境变量
export CONTROL_ALLOW_REMOTE_COMMAND=false
```

## 配置合并策略

### 主程序配置合并

配置合并按以下顺序进行：

1. **加载默认配置**: 系统内置的默认值
2. **加载配置文件**: 按优先级加载配置文件
3. **应用环境变量**: 环境变量覆盖配置文件值
4. **应用命令行参数**: 命令行参数具有最高优先级

### 插件配置合并

插件配置合并策略：

```go
// 配置合并伪代码
func mergePluginConfig(pluginID string) map[string]interface{} {
    config := make(map[string]interface{})
    
    // 1. 加载插件默认配置
    mergeMap(config, getPluginDefaults(pluginID))
    
    // 2. 加载插件独立配置文件
    if pluginConfig := loadPluginConfigFile(pluginID); pluginConfig != nil {
        mergeMap(config, pluginConfig)
    }
    
    // 3. 加载主配置文件中的插件配置
    if mainConfig := getMainConfigPluginSection(pluginID); mainConfig != nil {
        mergeMap(config, mainConfig)
    }
    
    // 4. 应用环境变量
    applyEnvironmentVariables(config, pluginID)
    
    return config
}
```

## 配置格式转换

### 旧版到新版格式转换

**旧版格式** (config.yaml):
```yaml
# 扁平结构
plugin_dir: "app"
log_level: "debug"
enable_dlp: true
dlp:
  monitor_network: true
  rules: [...]
```

**新版格式** (config.unified.yaml):
```yaml
# 层次化结构
global:
  logging:
    level: "debug"
plugin_manager:
  plugin_dir: "app"
plugins:
  dlp:
    enabled: true
    settings:
      monitor_network: true
      rules: [...]
```

### 配置迁移命令

```bash
# 基本迁移
go run cmd/config-migrate/main.go

# 自定义源和目标文件
go run cmd/config-migrate/main.go \
  -source old-config.yaml \
  -target new-config.yaml

# 强制覆盖，不备份
go run cmd/config-migrate/main.go \
  -force \
  -backup=false
```

## 配置验证规则

### 验证优先级

1. **类型验证**: 检查配置值的数据类型
2. **范围验证**: 检查数值是否在允许范围内
3. **格式验证**: 检查字符串格式（URL、路径等）
4. **自定义验证**: 业务逻辑相关的验证规则

### 验证失败处理

```yaml
# 验证失败时的处理策略
validation:
  on_error: "use_default"  # 选项: "fail", "use_default", "skip"
  strict_mode: false       # 严格模式：任何验证失败都会导致启动失败
  log_warnings: true       # 记录验证警告
```

## 配置热更新机制

### 热更新支持范围

| 配置类型 | 热更新支持 | 重启要求 | 说明 |
|----------|-----------|----------|------|
| 全局日志配置 | ✅ 支持 | ❌ 不需要 | 实时生效 |
| 插件启用状态 | ✅ 支持 | ❌ 不需要 | 自动启停插件 |
| 插件业务配置 | 🟡 部分支持 | ✅ 需要 | 重新初始化插件 |
| 网络端口配置 | ❌ 不支持 | ✅ 需要 | 重启应用程序 |
| 插件目录配置 | ❌ 不支持 | ✅ 需要 | 重启应用程序 |

### 热更新触发方式

1. **文件监视**: 自动检测配置文件变化
2. **API调用**: 通过Web控制台或API触发
3. **信号处理**: 发送SIGHUP信号触发重载

```bash
# 发送重载信号
kill -HUP $(pidof kennel)

# 通过API触发
curl -X POST http://localhost:8088/api/config/reload
```

## 配置冲突解决

### 冲突检测

系统会自动检测以下配置冲突：

1. **端口冲突**: 多个服务使用相同端口
2. **路径冲突**: 多个组件使用相同的文件路径
3. **资源冲突**: 资源限制超出系统能力
4. **依赖冲突**: 插件依赖关系冲突

### 冲突解决策略

```yaml
# 冲突解决配置
conflict_resolution:
  strategy: "priority_based"  # 选项: "priority_based", "fail_fast", "merge"
  auto_resolve: true          # 自动解决冲突
  log_conflicts: true         # 记录冲突信息
```

## 最佳实践

### 生产环境配置

1. **使用统一配置文件**: 优先使用`config.unified.yaml`
2. **环境变量管理**: 敏感信息通过环境变量配置
3. **配置验证**: 启用严格的配置验证
4. **配置备份**: 定期备份配置文件
5. **版本控制**: 将配置文件纳入版本控制

### 开发环境配置

1. **使用本地配置**: 创建`config.local.yaml`用于开发
2. **详细日志**: 启用debug级别日志
3. **快速迭代**: 使用配置热更新功能
4. **配置模板**: 使用配置模板快速生成配置

### 配置安全

1. **敏感信息**: 不要在配置文件中存储密码等敏感信息
2. **文件权限**: 设置适当的配置文件权限（600或644）
3. **环境隔离**: 不同环境使用不同的配置文件
4. **配置加密**: 对敏感配置进行加密存储

## 故障排除

### 常见配置问题

1. **配置文件未找到**
   ```
   错误: 无法读取配置文件
   解决: 检查配置文件路径和权限
   ```

2. **配置格式错误**
   ```
   错误: 解析YAML配置失败
   解决: 检查YAML语法，使用在线YAML验证器
   ```

3. **配置验证失败**
   ```
   错误: 配置验证失败: 端口号必须在1-65535范围内
   解决: 修正配置值，确保符合验证规则
   ```

4. **环境变量未生效**
   ```
   问题: 设置了环境变量但配置未更新
   解决: 检查环境变量命名是否正确，重启应用程序
   ```

### 调试配置加载

```bash
# 启用配置调试日志
export APPFW_GLOBAL_LOGGING_LEVEL=debug

# 查看当前使用的配置文件
./kennel --show-config-path

# 验证配置文件
./kennel --validate-config
```

## 总结

Kennel项目的配置优先级规则确保了：

1. **可预测性**: 明确的优先级顺序，配置行为可预测
2. **灵活性**: 支持多种配置来源和覆盖方式
3. **向后兼容**: 支持旧版配置格式的平滑迁移
4. **独立性**: 插件配置相互独立，不会产生冲突
5. **安全性**: 支持环境变量和配置验证机制

通过遵循这些规则，可以确保配置管理的一致性和可维护性。
