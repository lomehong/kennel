# AppFramework 性能分析工具使用指南

## 简介

AppFramework 性能分析工具是一个用于收集和分析应用程序性能数据的工具，它提供了多种性能分析类型，包括 CPU 分析、内存分析、阻塞分析等，可以帮助开发人员和运维人员快速定位和解决性能瓶颈。

## 功能特点

- **多种性能分析类型**：支持 CPU、堆内存、阻塞、协程、线程创建、互斥锁、执行追踪和内存分配等多种性能分析类型
- **多种输出格式**：支持 pprof、JSON、文本、SVG、PDF 和 HTML 等多种输出格式
- **HTTP 接口**：提供 Web 界面和 HTTP API，方便远程访问和集成
- **命令行工具**：提供命令行工具，方便本地使用和脚本集成
- **自动清理**：支持自动清理过期的性能分析数据，避免磁盘空间占用过大
- **结果管理**：支持管理和查询历史性能分析结果
- **数据分析**：支持分析性能分析数据，提供热点函数、内存分配等信息

## 安装和配置

### 在应用程序中集成

```go
import (
    "github.com/hashicorp/go-hclog"
    "github.com/lomehong/kennel/pkg/profiler"
)

func main() {
    // 创建日志记录器
    logger := hclog.New(&hclog.LoggerOptions{
        Name:   "app",
        Level:  hclog.Info,
        Output: os.Stdout,
    })

    // 创建性能分析器
    p := profiler.NewStandardProfiler("profiles", 100, logger)

    // 注册 HTTP 处理器
    handler := profiler.NewHTTPHandler(p, "/debug/pprof")
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, World!")
    })
    handler.RegisterHandlers(http.DefaultServeMux)

    // 启动 HTTP 服务器
    http.ListenAndServe(":8080", nil)
}
```

### 在命令行工具中集成

```go
import (
    "github.com/hashicorp/go-hclog"
    "github.com/spf13/cobra"
    "github.com/lomehong/kennel/pkg/profiler"
)

func main() {
    // 创建日志记录器
    logger := hclog.New(&hclog.LoggerOptions{
        Name:   "app",
        Level:  hclog.Info,
        Output: os.Stdout,
    })

    // 创建性能分析器
    p := profiler.NewStandardProfiler("profiles", 100, logger)

    // 创建根命令
    rootCmd := &cobra.Command{
        Use:   "app",
        Short: "应用程序",
    }

    // 注册命令行处理器
    handler := profiler.NewCommandHandler(p, logger)
    handler.RegisterCommands(rootCmd)

    // 执行命令
    rootCmd.Execute()
}
```

## 使用方法

### HTTP 接口

启动应用程序后，可以通过浏览器访问 `http://localhost:8080/debug/pprof/` 查看性能分析工具的 Web 界面。

Web 界面提供了以下功能：

- 查看可用的性能分析类型
- 启动和停止性能分析
- 查看正在运行的性能分析
- 查看性能分析结果
- 下载和分析性能分析数据

也可以通过 HTTP API 直接访问性能分析功能：

- `GET /debug/pprof/cpu?seconds=30`：收集 30 秒的 CPU 性能分析
- `GET /debug/pprof/heap`：收集堆内存性能分析
- `GET /debug/pprof/block`：收集阻塞性能分析
- `GET /debug/pprof/goroutine`：收集协程性能分析
- `GET /debug/pprof/threadcreate`：收集线程创建性能分析
- `GET /debug/pprof/mutex`：收集互斥锁性能分析
- `GET /debug/pprof/trace?seconds=5`：收集 5 秒的执行追踪
- `GET /debug/pprof/allocs`：收集内存分配性能分析

### 命令行工具

命令行工具提供了以下命令：

- `app profile start [type]`：启动性能分析
- `app profile stop [type]`：停止性能分析
- `app profile list`：列出性能分析结果
- `app profile running`：列出正在运行的性能分析
- `app profile analyze [type]`：分析性能分析数据
- `app profile cleanup`：清理性能分析数据

也可以使用快捷命令直接收集性能分析数据：

- `app profile cpu [seconds]`：收集 CPU 性能分析
- `app profile heap`：收集堆内存性能分析
- `app profile block`：收集阻塞性能分析
- `app profile goroutine`：收集协程性能分析
- `app profile threadcreate`：收集线程创建性能分析
- `app profile mutex`：收集互斥锁性能分析
- `app profile trace [seconds]`：收集执行追踪
- `app profile allocs`：收集内存分配性能分析

### 编程接口

也可以通过编程接口直接使用性能分析功能：

```go
// 创建性能分析器
p := profiler.NewStandardProfiler("profiles", 100, logger)

// 创建上下文
ctx := context.Background()

// 创建性能分析选项
options := profiler.DefaultProfileOptions()
options.Duration = 30 * time.Second

// 启动 CPU 性能分析
err := p.Start(ctx, profiler.ProfileTypeCPU, options)
if err != nil {
    log.Fatalf("启动 CPU 性能分析失败: %v", err)
}

// 执行一些操作...

// 停止 CPU 性能分析
result, err := p.Stop(profiler.ProfileTypeCPU)
if err != nil {
    log.Fatalf("停止 CPU 性能分析失败: %v", err)
}

// 分析性能分析数据
analysis, err := p.AnalyzeProfile(profiler.ProfileTypeCPU, result.FilePath)
if err != nil {
    log.Fatalf("分析性能分析数据失败: %v", err)
}

// 打印分析结果
fmt.Printf("性能分析结果: %+v\n", analysis)
```

## 性能分析类型

### CPU 分析

CPU 分析用于收集 CPU 使用情况，可以帮助发现 CPU 密集型操作和热点函数。

```bash
# HTTP 接口
curl http://localhost:8080/debug/pprof/cpu?seconds=30

# 命令行工具
app profile cpu 30

# 编程接口
p.Start(ctx, profiler.ProfileTypeCPU, options)
```

### 堆内存分析

堆内存分析用于收集堆内存使用情况，可以帮助发现内存泄漏和内存使用过高的问题。

```bash
# HTTP 接口
curl http://localhost:8080/debug/pprof/heap

# 命令行工具
app profile heap

# 编程接口
p.Start(ctx, profiler.ProfileTypeHeap, options)
```

### 阻塞分析

阻塞分析用于收集 goroutine 阻塞情况，可以帮助发现死锁和性能瓶颈。

```bash
# HTTP 接口
curl http://localhost:8080/debug/pprof/block

# 命令行工具
app profile block

# 编程接口
p.Start(ctx, profiler.ProfileTypeBlock, options)
```

### 协程分析

协程分析用于收集 goroutine 信息，可以帮助发现 goroutine 泄漏和过多的问题。

```bash
# HTTP 接口
curl http://localhost:8080/debug/pprof/goroutine

# 命令行工具
app profile goroutine

# 编程接口
p.Start(ctx, profiler.ProfileTypeGoroutine, options)
```

### 线程创建分析

线程创建分析用于收集线程创建情况，可以帮助发现线程创建过多的问题。

```bash
# HTTP 接口
curl http://localhost:8080/debug/pprof/threadcreate

# 命令行工具
app profile threadcreate

# 编程接口
p.Start(ctx, profiler.ProfileTypeThreadcreate, options)
```

### 互斥锁分析

互斥锁分析用于收集互斥锁争用情况，可以帮助发现锁竞争和死锁问题。

```bash
# HTTP 接口
curl http://localhost:8080/debug/pprof/mutex

# 命令行工具
app profile mutex

# 编程接口
p.Start(ctx, profiler.ProfileTypeMutex, options)
```

### 执行追踪

执行追踪用于收集程序执行追踪，可以帮助了解程序的执行流程和性能瓶颈。

```bash
# HTTP 接口
curl http://localhost:8080/debug/pprof/trace?seconds=5

# 命令行工具
app profile trace 5

# 编程接口
p.Start(ctx, profiler.ProfileTypeTrace, options)
```

### 内存分配分析

内存分配分析用于收集内存分配情况，可以帮助发现内存分配过多的问题。

```bash
# HTTP 接口
curl http://localhost:8080/debug/pprof/allocs

# 命令行工具
app profile allocs

# 编程接口
p.Start(ctx, profiler.ProfileTypeAllocs, options)
```

## 最佳实践

### 性能分析时机

- **开发阶段**：在开发阶段进行性能分析，可以及早发现和解决性能问题
- **测试阶段**：在测试阶段进行性能分析，可以验证性能优化效果
- **生产环境**：在生产环境进行性能分析，可以发现实际使用中的性能问题

### 性能分析持续时间

- **CPU 分析**：建议 30 秒左右，太短可能无法收集到足够的样本，太长可能会产生过多的数据
- **堆内存分析**：可以在任何时候收集，建议在怀疑内存泄漏时收集多个样本进行比较
- **阻塞分析**：建议在系统负载较高时收集，可以发现阻塞问题
- **执行追踪**：建议 5-10 秒，太长会产生大量数据，难以分析

### 性能分析数据分析

- 使用 `go tool pprof` 分析 pprof 格式的性能分析数据
- 使用 `go tool trace` 分析执行追踪数据
- 使用 Web 界面查看性能分析结果
- 使用命令行工具分析性能分析数据

### 性能优化建议

- **CPU 优化**：减少不必要的计算，使用缓存，使用并发处理
- **内存优化**：减少内存分配，使用对象池，避免内存泄漏
- **并发优化**：控制并发数量，避免过多的 goroutine，使用适当的同步机制
- **I/O 优化**：使用缓冲 I/O，使用异步 I/O，减少系统调用

## 故障排除

### 常见问题

- **性能分析启动失败**：检查权限和磁盘空间
- **性能分析数据过大**：减少持续时间或采样率
- **性能分析数据分析失败**：检查 Go 工具链是否安装正确
- **Web 界面无法访问**：检查网络和防火墙设置

### 日志和调试

- 查看应用程序日志，了解性能分析工具的运行状态
- 使用 `--debug` 参数启动命令行工具，查看详细日志
- 使用 `?debug=1` 参数访问 HTTP 接口，查看详细日志

## 参考资料

- [Go pprof 文档](https://golang.org/pkg/runtime/pprof/)
- [Go trace 文档](https://golang.org/pkg/runtime/trace/)
- [Go 性能优化指南](https://github.com/dgryski/go-perfbook)
- [Go 性能分析工具](https://blog.golang.org/pprof)
