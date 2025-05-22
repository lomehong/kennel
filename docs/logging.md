# 日志系统使用指南

本文档介绍了项目中统一的日志系统使用方法。项目使用 `pkg/logging` 包作为唯一的日志实现，提供了丰富的功能和灵活的配置选项。

## 基本用法

### 创建日志记录器

```go
import "github.com/lomehong/kennel/pkg/logging"

// 使用默认配置创建日志记录器
logger, err := logging.NewEnhancedLogger(nil)
if err != nil {
    // 处理错误
}

// 使用自定义配置创建日志记录器
config := logging.DefaultLogConfig()
config.Level = logging.LogLevelDebug
config.Format = logging.LogFormatJSON
config.Output = "file"
config.FilePath = "logs/app.log"

logger, err := logging.NewEnhancedLogger(config)
if err != nil {
    // 处理错误
}
```

### 记录日志

```go
// 记录不同级别的日志
logger.Trace("这是跟踪级别日志")
logger.Debug("这是调试级别日志")
logger.Info("这是信息级别日志")
logger.Warn("这是警告级别日志")
logger.Error("这是错误级别日志")
logger.Fatal("这是致命级别日志") // 会导致程序退出

// 记录带有上下文的日志
logger.Info("用户登录成功", "user_id", 123, "username", "admin")

// 记录带有错误的日志
err := someFunction()
if err != nil {
    logger.Error("操作失败", "error", err)
}
```

### 创建子日志记录器

```go
// 创建带有名称的子日志记录器
userLogger := logger.Named("user")
userLogger.Info("用户相关操作")

// 创建带有固定字段的子日志记录器
dbLogger := logger.WithFields(map[string]interface{}{
    "component": "database",
    "version": "1.0",
})
dbLogger.Info("数据库操作")
```

## 高级功能

### 日志轮转

日志系统支持基于大小和时间的日志轮转：

```go
config := logging.DefaultLogConfig()
config.Output = "file"
config.FilePath = "logs/app.log"
config.MaxSize = 10    // 单个日志文件最大大小（MB）
config.MaxBackups = 5  // 保留的旧日志文件数量
config.MaxAge = 30     // 保留的旧日志文件最大天数
config.Compress = true // 是否压缩旧日志文件
```

### 多输出目标

日志系统支持同时输出到多个目标：

```go
config := logging.DefaultLogConfig()
config.Output = "multi"
config.Outputs = []logging.OutputConfig{
    {
        Type:     "console",
        Format:   "text",
        Level:    "debug",
    },
    {
        Type:     "file",
        Format:   "json",
        Level:    "info",
        FilePath: "logs/app.log",
    },
}
```

### 上下文跟踪

日志系统支持通过上下文传递日志字段：

```go
// 创建带有请求ID的上下文
ctx := logging.WithValue(context.Background(), "request_id", "req-123")

// 从上下文中获取日志记录器
logger := logging.FromContext(ctx)
logger.Info("处理请求") // 自动包含 request_id 字段

// 向上下文添加更多字段
ctx = logging.WithFields(ctx, map[string]interface{}{
    "user_id": 456,
})
```

### HTTP 中间件

日志系统提供了 HTTP 中间件，用于记录请求日志：

```go
// 创建 HTTP 处理器
handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // 处理请求
})

// 应用日志中间件
loggedHandler := logging.HTTPMiddleware(logger)(handler)
http.Handle("/api", loggedHandler)
```

## 与旧版日志系统的兼容性

为了平滑迁移，日志系统提供了与旧版 `pkg/logger` 包兼容的接口：

```go
import "github.com/lomehong/kennel/pkg/logging"

// 创建兼容旧版接口的日志记录器
logger := logging.NewLegacyLogger("module-name", logging.GetLegacyLogLevel("info"))

// 使用旧版接口记录日志
logger.Info("这是信息级别日志")
logger.Error("这是错误级别日志", "error", err)

// 创建带有上下文的子日志记录器
subLogger := logger.With("user_id", 123)
```

## 最佳实践

1. **使用结构化日志**：始终使用键值对记录上下文信息，而不是拼接字符串。
2. **合理设置日志级别**：生产环境通常使用 Info 级别，开发环境使用 Debug 级别。
3. **使用有意义的日志消息**：日志消息应该简洁明了，描述发生了什么，而不是如何发生的。
4. **记录关键操作**：记录所有关键操作，如用户登录、重要配置更改、系统启动和关闭等。
5. **使用子日志记录器**：为不同的组件创建子日志记录器，以便更好地组织和过滤日志。
6. **包含错误详情**：记录错误时，始终包含完整的错误信息。
7. **避免敏感信息**：不要记录密码、令牌等敏感信息。

## 配置参考

### 日志级别

- `trace`：最详细的日志级别，用于追踪程序执行流程。
- `debug`：调试信息，用于开发和故障排除。
- `info`：一般信息，用于记录正常操作。
- `warn`：警告信息，表示可能的问题。
- `error`：错误信息，表示操作失败。
- `fatal`：致命错误，会导致程序退出。

### 日志格式

- `text`：人类可读的文本格式。
- `json`：结构化的 JSON 格式，便于机器处理。

### 输出类型

- `console`：输出到标准输出（stdout）。
- `file`：输出到文件。
- `multi`：同时输出到多个目标。
