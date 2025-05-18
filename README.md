# Kennel

一个基于Go的跨平台（Windows + macOS）终端代理框架，具有模块化、可插拔的特性，包括终端资产管理、设备管理、数据防泄漏（DLP）、终端管控等功能，统一在一个CLI应用程序下。

## 项目状态

当前项目处于开发阶段，基本框架已经搭建完成，但插件系统的gRPC接口实现尚未完成。主程序可以正常运行，但无法加载插件。

## 特性

- **模块化设计**：核心框架与功能模块分离，支持动态加载
- **跨平台支持**：同时支持Windows和macOS系统
- **插件系统**：基于HashiCorp go-plugin，通过子进程+RPC/gRPC实现插件加载
- **优雅终止**：支持信号处理和资源清理，确保应用程序安全退出
- **通讯模块**：支持与服务端的WebSocket长连接，实现双向通信
- **功能丰富**：包含资产管理、设备管理、数据防泄漏、终端管控等模块

## 架构

Kennel采用模块化架构，主要包括以下组件：

1. **核心框架（Host CLI）**：基于Cobra/Viper实现的命令行界面和配置管理
2. **插件系统**：使用HashiCorp go-plugin实现的插件加载机制
3. **通信接口**：基于gRPC/Protobuf定义的模块间通信接口
4. **通讯模块**：基于WebSocket实现的与服务端的双向通信
5. **功能模块**：独立的插件进程，实现特定功能

## 安装

### 从源码构建

1. 克隆仓库：

```bash
git clone https://github.com/lomehong/kennel.git
cd kennel
```

2. 构建主程序：

```bash
go build -o agent.exe cmd/agent/main.go
```

3. 构建插件模块：

```bash
# 资产管理模块
go build -o app/assets/assets.exe app/assets/main.go

# 设备管理模块
go build -o app/device/device.exe app/device/main.go

# 数据防泄漏模块
go build -o app/dlp/dlp.exe app/dlp/main.go

# 终端管控模块
go build -o app/control/control.exe app/control/main.go
```

## 使用方法

### 基本命令

- 显示版本信息：

```bash
agent.exe version
```

- 列出已加载的插件：

```bash
agent.exe plugin list
```

- 加载插件：

```bash
agent.exe plugin load app/assets/assets.exe
```

- 启动代理：

```bash
agent.exe start
```

- 停止代理：

```bash
agent.exe stop
```

### 配置文件

AppFramework使用YAML格式的配置文件，默认位置为当前目录下的`config.yaml`。您也可以使用`-c`或`--config`参数指定配置文件路径：

```bash
agent.exe -c path/to/config.yaml start
```

配置文件示例：

```yaml
# 插件目录
plugin_dir: "app"

# 日志配置
log_level: "info"
log_file: "agent.log"

# 模块启用配置
enable_assets: true
enable_device: true
enable_dlp: true
enable_control: true

# 各模块的具体配置...
```

## 模块功能

### 资产管理（Asset Management）

- 收集主机硬件/系统信息（CPU、内存、磁盘、网络）
- 定期上报资产状态至后台服务器

### 设备管理（Device Management）

- 网络流量捕获与管理
- 控制网络接口（启用/禁用）
- 收集外设状态

### 数据防泄漏（DLP）

- 内容检测（正则表达式、指纹）
- 文件监控
- 剪贴板监控

### 终端管控（Endpoint Control）

- 进程管理（列出、终止）
- 远程命令执行
- 软件安装

### 安全审计（Security Audit）

- 系统事件记录
- 用户操作审计
- 安全事件警报
- 审计日志查询

## 目录结构

项目采用以下目录结构：

```
AppFramework/
├── app/                    # 插件应用目录
│   ├── assets/             # 资产管理模块
│   ├── device/             # 设备管理模块
│   ├── dlp/                # 数据防泄漏模块
│   ├── control/            # 终端管控模块
│   └── audit/              # 安全审计模块
├── bin/                    # 构建输出目录
├── cmd/                    # 命令行程序
│   └── agent/              # 主程序
├── pkg/                    # 共享包
│   ├── core/               # 核心功能
│   ├── plugin/             # 插件接口
├── build.ps1               # 构建脚本
├── release.ps1             # 发布脚本
├── .goreleaser.yml         # GoReleaser配置
├── config.yaml             # 配置文件
└── README.md               # 说明文档
```

## 开发指南

### 创建新模块

1. 在`app`目录下创建新的模块目录
2. 实现`Module`接口
3. 使用go-plugin框架注册插件
4. 构建插件可执行文件

模块接口定义：

```go
type Module interface {
    // 初始化模块
    Init(config map[string]interface{}) error

    // 执行模块操作
    Execute(action string, params map[string]interface{}) (map[string]interface{}, error)

    // 关闭模块
    Shutdown() error

    // 获取模块信息
    GetInfo() ModuleInfo
}
```

## 文档

- [优雅终止功能](docs/graceful_shutdown.md)：详细介绍了框架的优雅终止机制和插件开发指南
- [通讯模块](docs/comm.md)：详细介绍了框架的通讯模块和使用方法


## 许可证

[AGPL-3.0 license](LICENSE)

## 贡献

欢迎提交问题和拉取请求！
