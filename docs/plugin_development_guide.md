# Kennel 插件开发指南

## 插件开发概述

Kennel 插件是独立的功能模块，可以动态加载到 Kennel 框架中。插件可以使用 Go 或 Python 语言开发，本指南将介绍插件开发的基本流程和最佳实践。

## 插件目录结构

一个标准的插件目录结构如下：

```
my-plugin/
├── plugin.json        # 插件元数据
├── config.yaml        # 插件默认配置
├── README.md          # 插件文档
├── LICENSE            # 许可证文件
├── src/               # 源代码目录
│   ├── main.go        # Go插件入口点
│   └── ...            # 其他源文件
└── resources/         # 资源文件目录
    ├── templates/     # 模板文件
    ├── static/        # 静态资源
    └── ...            # 其他资源
```

## 插件元数据

`plugin.json` 文件定义了插件的基本信息：

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
  "dependencies": [
    "core:>=1.0.0"
  ],
  "capabilities": [
    "example"
  ],
  "supported_platforms": [
    "windows",
    "linux",
    "darwin"
  ],
  "language": "go",
  "author": "开发者名称",
  "license": "MIT",
  "min_framework_version": "1.0.0"
}
```

## Go 插件开发

### 基本结构

Go 插件需要实现 `Module` 接口：

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/lomehong/kennel/pkg/plugin"
)

// MyPlugin 实现了 Module 接口
type MyPlugin struct {
    logger plugin.Logger
    config map[string]interface{}
}

// 创建插件实例
func NewMyPlugin() plugin.Module {
    return &MyPlugin{}
}

// Init 初始化插件
func (p *MyPlugin) Init(ctx context.Context, config *plugin.ModuleConfig) error {
    p.logger = plugin.NewLogger("my-plugin")
    p.config = config.Settings
    p.logger.Info("插件已初始化")
    return nil
}

// Start 启动插件
func (p *MyPlugin) Start() error {
    p.logger.Info("插件已启动")
    return nil
}

// Stop 停止插件
func (p *MyPlugin) Stop() error {
    p.logger.Info("插件已停止")
    return nil
}

// GetInfo 获取插件信息
func (p *MyPlugin) GetInfo() plugin.ModuleInfo {
    return plugin.ModuleInfo{
        ID:                "my-plugin",
        Name:              "我的插件",
        Version:           "1.0.0",
        Description:       "这是一个示例插件",
        Author:            "开发者名称",
        License:           "MIT",
        Capabilities:      []string{"example"},
        SupportedPlatforms: []string{"windows", "linux", "darwin"},
        Language:          "go",
    }
}

// HandleRequest 处理请求
func (p *MyPlugin) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
    p.logger.Info("收到请求", "action", req.Action)
    
    switch req.Action {
    case "hello":
        return &plugin.Response{
            ID:      req.ID,
            Success: true,
            Data: map[string]interface{}{
                "message": "Hello, World!",
            },
        }, nil
    default:
        return &plugin.Response{
            ID:      req.ID,
            Success: false,
            Error: &plugin.ErrorInfo{
                Code:    "unknown_action",
                Message: fmt.Sprintf("未知操作: %s", req.Action),
            },
        }, nil
    }
}

// HandleEvent 处理事件
func (p *MyPlugin) HandleEvent(ctx context.Context, event *plugin.Event) error {
    p.logger.Info("收到事件", "type", event.Type)
    return nil
}

// 插件入口点
func main() {
    plugin.Serve(NewMyPlugin())
}
```

### 构建 Go 插件

```bash
# 构建插件
go build -o my-plugin.exe src/main.go
```

## Python 插件开发

### 基本结构

Python 插件需要继承 `Module` 基类：

```python
#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from kennel.plugin import Module, ModuleInfo, Request, Response, Event
from typing import Dict, Any, Optional
import logging

class MyPlugin(Module):
    """示例Python插件"""
    
    def __init__(self):
        self.logger = logging.getLogger("my-plugin")
        self.config = {}
    
    def init(self, config: Dict[str, Any]) -> None:
        """初始化插件"""
        self.config = config
        self.logger.info("插件已初始化")
    
    def start(self) -> None:
        """启动插件"""
        self.logger.info("插件已启动")
    
    def stop(self) -> None:
        """停止插件"""
        self.logger.info("插件已停止")
    
    def get_info(self) -> ModuleInfo:
        """获取插件信息"""
        return ModuleInfo(
            id="my-python-plugin",
            name="我的Python插件",
            version="1.0.0",
            description="这是一个Python示例插件",
            author="开发者名称",
            license="MIT",
            capabilities=["example"],
            supported_platforms=["windows", "linux", "darwin"]
        )
    
    def handle_request(self, request: Request) -> Response:
        """处理请求"""
        self.logger.info(f"收到请求: {request.action}")
        
        if request.action == "hello":
            return Response(
                id=request.id,
                success=True,
                data={"message": "Hello from Python!"}
            )
        else:
            return Response(
                id=request.id,
                success=False,
                error={
                    "code": "unknown_action",
                    "message": f"未知操作: {request.action}"
                }
            )
    
    def handle_event(self, event: Event) -> bool:
        """处理事件"""
        self.logger.info(f"收到事件: {event.type}")
        return True

# 插件入口点
if __name__ == "__main__":
    from kennel.plugin import serve
    serve(MyPlugin())
```

### 运行 Python 插件

```bash
# 安装依赖
pip install kennel-plugin-sdk

# 运行插件
python src/main.py
```

## 插件配置

插件配置使用 YAML 格式，放在 `config.yaml` 文件中：

```yaml
# 插件默认配置
enabled: true
log_level: "info"

# 插件特定设置
settings:
  option1: "value1"
  option2: 42
  nested:
    key1: "value1"
    key2: "value2"
```

## 插件通信

### 请求-响应模式

插件可以处理来自框架的请求：

```go
// 处理请求
func (p *MyPlugin) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
    switch req.Action {
    case "get_data":
        // 获取参数
        id := req.Params["id"].(string)
        
        // 处理请求
        data := p.getData(id)
        
        // 返回响应
        return &plugin.Response{
            ID:      req.ID,
            Success: true,
            Data:    data,
        }, nil
    }
    
    return nil, fmt.Errorf("未知操作: %s", req.Action)
}
```

### 事件处理

插件可以处理框架发布的事件：

```go
// 处理事件
func (p *MyPlugin) HandleEvent(ctx context.Context, event *plugin.Event) error {
    switch event.Type {
    case "system.startup":
        // 处理系统启动事件
        p.onSystemStartup()
    case "user.login":
        // 处理用户登录事件
        username := event.Data["username"].(string)
        p.onUserLogin(username)
    }
    
    return nil
}
```

### 发布事件

插件可以通过事件总线发布事件：

```go
// 发布事件
func (p *MyPlugin) publishEvent() {
    event := &plugin.Event{
        Type:   "my-plugin.something_happened",
        Source: "my-plugin",
        Data: map[string]interface{}{
            "timestamp": time.Now().Unix(),
            "message":   "Something interesting happened",
        },
    }
    
    p.eventBus.Publish(event)
}
```

## 插件依赖管理

插件可以声明对其他插件的依赖：

```json
"dependencies": [
  "core:>=1.0.0",
  "assets:>=0.5.0",
  "device:>=0.3.0"
]
```

## 插件资源管理

插件应该遵循以下资源管理最佳实践：

1. **优雅关闭**：在 `Stop()` 方法中释放所有资源
2. **资源限制**：尊重框架设置的资源限制
3. **临时文件**：使用系统临时目录存储临时文件
4. **持久化数据**：使用框架提供的数据目录存储持久化数据

## 插件测试

### 单元测试

为插件编写单元测试：

```go
func TestMyPlugin_HandleRequest(t *testing.T) {
    p := NewMyPlugin()
    
    // 初始化插件
    err := p.Init(context.Background(), &plugin.ModuleConfig{
        ID:       "my-plugin",
        Name:     "我的插件",
        Version:  "1.0.0",
        Settings: map[string]interface{}{},
    })
    if err != nil {
        t.Fatalf("初始化插件失败: %v", err)
    }
    
    // 测试请求处理
    req := &plugin.Request{
        ID:     "test-1",
        Action: "hello",
        Params: map[string]interface{}{},
    }
    
    resp, err := p.HandleRequest(context.Background(), req)
    if err != nil {
        t.Fatalf("处理请求失败: %v", err)
    }
    
    if !resp.Success {
        t.Errorf("请求失败: %v", resp.Error)
    }
    
    message, ok := resp.Data["message"].(string)
    if !ok {
        t.Errorf("响应数据类型错误")
    }
    
    if message != "Hello, World!" {
        t.Errorf("响应消息错误: %s", message)
    }
}
```

### 集成测试

使用框架提供的测试工具进行集成测试：

```go
func TestMyPluginIntegration(t *testing.T) {
    // 创建测试环境
    env := plugin.NewTestEnvironment()
    
    // 加载插件
    p, err := env.LoadPlugin("my-plugin")
    if err != nil {
        t.Fatalf("加载插件失败: %v", err)
    }
    
    // 发送请求
    resp, err := env.SendRequest(p.ID, "hello", nil)
    if err != nil {
        t.Fatalf("发送请求失败: %v", err)
    }
    
    // 验证响应
    if !resp.Success {
        t.Errorf("请求失败: %v", resp.Error)
    }
}
```
