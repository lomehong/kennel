# Kennel 跨语言支持指南

## 概述

Kennel 框架支持使用不同编程语言编写插件，目前支持 Go 和 Python 两种语言。本指南将介绍 Kennel 的跨语言支持机制和开发流程。

## 跨语言支持架构

Kennel 的跨语言支持基于以下核心原则：

1. **语言无关的通信协议**：使用 gRPC/Protobuf 定义接口
2. **进程隔离**：每个插件作为独立进程运行
3. **标准化接口**：所有语言实现相同的接口规范
4. **语言特定SDK**：为每种支持的语言提供SDK

## 通信协议

Kennel 使用 Protocol Buffers 定义语言无关的接口，通过 gRPC 进行通信。这使得不同语言编写的插件可以与 Kennel 框架无缝集成。

### 插件服务定义

```protobuf
syntax = "proto3";

package kennel.plugin;

// 插件服务定义
service PluginService {
  // 初始化插件
  rpc Init(InitRequest) returns (InitResponse);
  
  // 启动插件
  rpc Start(StartRequest) returns (StartResponse);
  
  // 停止插件
  rpc Stop(StopRequest) returns (StopResponse);
  
  // 获取插件信息
  rpc GetInfo(InfoRequest) returns (InfoResponse);
  
  // 处理请求
  rpc HandleRequest(Request) returns (Response);
  
  // 处理事件
  rpc HandleEvent(Event) returns (EventResponse);
  
  // 健康检查
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

## 插件启动协议

### 插件发现和启动

1. **插件清单**：每个插件目录包含一个 `plugin.json` 文件，描述插件信息和启动方式
2. **启动命令**：根据插件语言选择适当的启动命令
3. **环境变量**：通过环境变量传递初始化参数
4. **握手协议**：使用标准的握手协议确保通信正常

### 插件清单示例

```json
{
  "id": "python-example",
  "name": "Python示例插件",
  "version": "1.0.0",
  "description": "使用Python实现的示例插件",
  "entry_point": {
    "type": "python",
    "script": "main.py",
    "interpreter": "python3"
  },
  "dependencies": [],
  "capabilities": ["example"],
  "supported_platforms": ["windows", "linux", "darwin"],
  "language": "python",
  "author": "Kennel Team",
  "license": "MIT"
}
```

## Go 插件开发

### 使用 Go SDK

Go 插件可以直接使用 Kennel 的 Go SDK 进行开发，无需额外的通信层。

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
				"message": "Hello from Go!",
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

// 插件入口点
func main() {
	plugin := NewMyPlugin()
	sdk.RunModule(plugin)
}
```

## Python 插件开发

### 安装 Python SDK

首先，安装 Kennel 的 Python SDK：

```bash
# 从源码安装
pip install -e path/to/kennel/pkg/sdk/python

# 或者从 PyPI 安装（未来支持）
# pip install kennel-sdk
```

### 使用 Python SDK

使用 Python SDK 开发插件：

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

## 插件通信

### 请求-响应模式

插件可以处理来自框架的请求，无论使用何种语言，请求和响应的格式都是一致的：

```python
# Python 示例
def handle_request(self, request: Request) -> Response:
    if request.action == "get_data":
        # 获取参数
        id = request.params.get("id", "")
        
        # 处理请求
        data = self.get_data(id)
        
        # 返回响应
        return Response(
            id=request.id,
            success=True,
            data=data
        )
    
    return Response(
        id=request.id,
        success=False,
        error={
            "code": "unknown_action",
            "message": f"未知操作: {request.action}"
        }
    )
```

### 事件处理

插件可以处理框架发布的事件：

```python
# Python 示例
def handle_event(self, event: Event) -> bool:
    if event.type == "system.startup":
        # 处理系统启动事件
        self.on_system_startup()
        return True
    elif event.type == "user.login":
        # 处理用户登录事件
        username = event.data.get("username", "")
        self.on_user_login(username)
        return True
    
    return False
```

## 数据类型映射

不同语言之间的数据类型映射如下：

| Protocol Buffers | Go | Python |
|------------------|-------|--------|
| bool | bool | bool |
| int32, int64 | int, int64 | int |
| float, double | float32, float64 | float |
| string | string | str |
| bytes | []byte | bytes |
| repeated | slice | list |
| map | map | dict |
| message | struct | class |

## 插件隔离

Kennel 使用进程隔离确保插件之间的独立性和安全性：

1. **独立进程**：每个插件在独立的进程中运行
2. **资源限制**：可以为插件设置资源限制，如 CPU 和内存使用量
3. **错误隔离**：一个插件的崩溃不会影响其他插件或主程序
4. **安全边界**：插件只能通过定义的接口与主程序通信

## 插件部署

### 目录结构

插件应该按照以下目录结构部署：

```
plugins/
├── my-go-plugin/
│   ├── plugin.json
│   ├── config.yaml
│   └── my-go-plugin.exe
├── my-python-plugin/
│   ├── plugin.json
│   ├── config.yaml
│   └── main.py
```

### 配置

在 Kennel 的配置文件中启用插件：

```yaml
plugins:
  my-go-plugin:
    enabled: true
    # 插件特定配置
    option1: "value1"
  
  my-python-plugin:
    enabled: true
    # 插件特定配置
    debug_mode: true
```

## 跨语言开发最佳实践

1. **使用标准接口**：遵循 Kennel 定义的标准接口
2. **使用语言特定 SDK**：使用 Kennel 提供的语言特定 SDK
3. **处理序列化问题**：注意不同语言之间的数据类型映射
4. **错误处理**：妥善处理所有可能的错误
5. **资源管理**：在插件停止时释放所有资源
6. **日志记录**：使用日志记录重要事件和错误
7. **配置验证**：验证插件配置，提供合理的默认值
8. **文档**：为插件提供详细的文档，包括使用说明和API参考

## 故障排除

### 常见问题

1. **插件无法启动**：检查插件清单和入口点是否正确
2. **通信错误**：检查插件是否正确实现了接口
3. **序列化错误**：检查数据类型是否兼容
4. **资源限制**：检查插件是否超出了资源限制
5. **依赖问题**：检查插件依赖是否满足

### 调试技巧

1. **启用调试日志**：设置日志级别为 debug 或 trace
2. **检查插件日志**：查看插件的日志输出
3. **使用调试工具**：使用语言特定的调试工具
4. **检查进程状态**：使用系统工具检查插件进程状态
5. **检查通信**：使用网络工具检查插件和主程序之间的通信
