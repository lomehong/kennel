# Kennel 插件开发指南（更新版）

## 概述

Kennel 是一个模块化、可插拔的跨平台终端代理框架。本指南将帮助您开发 Kennel 插件，包括 Go 和 Python 两种语言的插件开发流程。

## 插件架构

Kennel 插件系统基于以下核心概念：

1. **模块接口**：所有插件必须实现 `Module` 接口
2. **插件元数据**：描述插件的基本信息
3. **插件生命周期**：初始化、启动、运行、停止、卸载
4. **插件通信**：请求-响应模式和事件-订阅模式
5. **插件配置**：每个插件拥有独立的配置命名空间

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

使用 Kennel SDK 可以简化 Go 插件的开发。以下是一个基本的 Go 插件示例：

```go
package main

import (
	"context"
	"fmt"
	
	"github.com/lomehong/kennel/pkg/sdk/go"
)

// MyPlugin 实现了 Module 接口
type MyPlugin struct {
	*sdk.BaseModule
}

// NewMyPlugin 创建插件实例
func NewMyPlugin() *MyPlugin {
	base := sdk.NewBaseModule(
		"my-plugin",
		"我的插件",
		"1.0.0",
		"这是一个示例插件",
	)
	
	return &MyPlugin{
		BaseModule: base,
	}
}

// HandleRequest 处理请求
func (p *MyPlugin) HandleRequest(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	p.Logger.Info("收到请求", "action", req.Action)
	
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
	p.Logger.Info("收到事件", "type", event.Type)
	return nil
}

// 插件入口点
func main() {
	plugin := NewMyPlugin()
	sdk.RunModule(plugin)
}
```

### 构建 Go 插件

```bash
# 构建插件
go build -o my-plugin.exe src/main.go
```

## Python 插件开发

### 基本结构

使用 Kennel Python SDK 可以简化 Python 插件的开发。以下是一个基本的 Python 插件示例：

```python
#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from kennel.sdk.python import BaseModule, ModuleInfo, Request, Response, Event, run_module
from typing import Dict, Any, Optional

class MyPythonPlugin(BaseModule):
    """示例Python插件"""
    
    def __init__(self):
        """初始化插件"""
        super().__init__(
            id="my-python-plugin",
            name="我的Python插件",
            version="1.0.0",
            description="这是一个Python示例插件"
        )
        self.author = "开发者名称"
        self.license = "MIT"
        self.capabilities = ["example"]
    
    def init(self, config: Dict[str, Any]) -> None:
        """初始化模块"""
        super().init(config)
        self.logger.info("插件已初始化")
        
        # 使用配置辅助类获取配置
        debug_mode = self.config_helper.get_bool("debug_mode", False)
        if debug_mode:
            self.logger.debug("调试模式已启用")
    
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
    plugin = MyPythonPlugin()
    run_module(plugin)
```

### 运行 Python 插件

```bash
# 安装依赖
pip install -e path/to/kennel/pkg/sdk/python

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

### 访问配置

在 Go 插件中：

```go
// 在插件初始化时接收配置
func (p *MyPlugin) Init(ctx context.Context, config *plugin.ModuleConfig) error {
    // 调用基类初始化
    if err := p.BaseModule.Init(ctx, config); err != nil {
        return err
    }
    
    // 使用辅助函数获取配置
    option1 := sdk.GetConfigString(p.Config, "option1", "default")
    option2 := sdk.GetConfigInt(p.Config, "option2", 0)
    
    // 使用配置
    p.Logger.Info("配置已加载", "option1", option1, "option2", option2)
    
    return nil
}
```

在 Python 插件中：

```python
def init(self, config: Dict[str, Any]) -> None:
    # 调用基类初始化
    super().init(config)
    
    # 使用配置辅助类获取配置
    option1 = self.config_helper.get_string("option1", "default")
    option2 = self.config_helper.get_int("option2", 0)
    
    # 使用配置
    self.logger.info("配置已加载", option1=option1, option2=option2)
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

## 插件测试

### 单元测试

为插件编写单元测试：

```go
func TestMyPlugin_HandleRequest(t *testing.T) {
    p := NewMyPlugin()
    
    // 初始化插件
    ctx := context.Background()
    err := p.Init(ctx, &plugin.ModuleConfig{
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
    
    resp, err := p.HandleRequest(ctx, req)
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

## 插件部署

### 安装插件

将插件目录复制到 Kennel 的 `plugins` 目录下：

```
kennel/
├── plugins/
│   ├── my-plugin/
│   │   ├── plugin.json
│   │   ├── config.yaml
│   │   ├── my-plugin.exe
│   │   └── ...
```

### 启用插件

在 Kennel 的配置文件中启用插件：

```yaml
plugins:
  my-plugin:
    enabled: true
    # 插件特定配置
    option1: "custom-value"
    option2: 100
```

### 命令行管理

使用命令行工具管理插件：

```bash
# 列出所有插件
kennel plugin list

# 加载插件
kennel plugin load my-plugin

# 卸载插件
kennel plugin unload my-plugin
```

## 最佳实践

1. **模块化设计**：将插件功能拆分为多个小模块，每个模块负责特定功能
2. **错误处理**：妥善处理所有可能的错误，提供有用的错误信息
3. **资源管理**：在插件停止时释放所有资源
4. **配置验证**：验证插件配置，提供合理的默认值
5. **日志记录**：使用日志记录重要事件和错误
6. **文档**：为插件提供详细的文档，包括使用说明和API参考
7. **测试**：编写单元测试和集成测试，确保插件功能正常
8. **版本控制**：使用语义化版本控制插件版本
