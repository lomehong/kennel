# Kennel 跨语言支持设计

## 跨语言支持概述

Kennel 框架支持使用不同编程语言编写插件，初始阶段重点支持 Go 和 Python。跨语言支持基于以下核心原则：

1. **语言无关的通信协议**：使用 gRPC/Protobuf 定义接口
2. **进程隔离**：每个插件作为独立进程运行
3. **标准化接口**：所有语言实现相同的接口规范
4. **语言特定SDK**：为每种支持的语言提供SDK

## 通信协议

### gRPC 接口定义

使用 Protocol Buffers 定义语言无关的接口：

```protobuf
syntax = "proto3";

package kennel.plugin;

option go_package = "github.com/lomehong/kennel/pkg/plugin/proto";

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

// 初始化请求
message InitRequest {
  // 插件配置
  ModuleConfig config = 1;
}

// 初始化响应
message InitResponse {
  // 是否成功
  bool success = 1;
  // 错误信息
  ErrorInfo error = 2;
}

// 启动请求
message StartRequest {}

// 启动响应
message StartResponse {
  // 是否成功
  bool success = 1;
  // 错误信息
  ErrorInfo error = 2;
}

// 停止请求
message StopRequest {}

// 停止响应
message StopResponse {
  // 是否成功
  bool success = 1;
  // 错误信息
  ErrorInfo error = 2;
}

// 信息请求
message InfoRequest {}

// 信息响应
message InfoResponse {
  // 模块信息
  ModuleInfo info = 1;
}

// 请求
message Request {
  // 请求ID
  string id = 1;
  // 操作
  string action = 2;
  // 参数
  map<string, Value> params = 3;
  // 元数据
  map<string, string> metadata = 4;
  // 超时（毫秒）
  int64 timeout = 5;
}

// 响应
message Response {
  // 请求ID
  string id = 1;
  // 是否成功
  bool success = 2;
  // 数据
  map<string, Value> data = 3;
  // 错误信息
  ErrorInfo error = 4;
  // 元数据
  map<string, string> metadata = 5;
}

// 事件
message Event {
  // 事件ID
  string id = 1;
  // 事件类型
  string type = 2;
  // 事件源
  string source = 3;
  // 时间戳
  int64 timestamp = 4;
  // 数据
  map<string, Value> data = 5;
  // 元数据
  map<string, string> metadata = 6;
}

// 事件响应
message EventResponse {
  // 是否成功
  bool success = 1;
  // 错误信息
  ErrorInfo error = 2;
}

// 健康检查请求
message HealthCheckRequest {}

// 健康检查响应
message HealthCheckResponse {
  // 状态
  string status = 1;
  // 详情
  map<string, Value> details = 2;
  // 时间戳
  int64 timestamp = 3;
}

// 模块配置
message ModuleConfig {
  // ID
  string id = 1;
  // 名称
  string name = 2;
  // 版本
  string version = 3;
  // 设置
  map<string, Value> settings = 4;
  // 依赖
  repeated string dependencies = 5;
  // 资源限制
  ResourceLimits resources = 6;
}

// 模块信息
message ModuleInfo {
  // ID
  string id = 1;
  // 名称
  string name = 2;
  // 版本
  string version = 3;
  // 描述
  string description = 4;
  // 作者
  string author = 5;
  // 许可证
  string license = 6;
  // 能力
  repeated string capabilities = 7;
  // 支持的平台
  repeated string supported_platforms = 8;
  // 实现语言
  string language = 9;
}

// 错误信息
message ErrorInfo {
  // 错误代码
  string code = 1;
  // 错误消息
  string message = 2;
  // 错误详情
  map<string, Value> details = 3;
}

// 资源限制
message ResourceLimits {
  // 最大CPU使用率（百分比）
  double max_cpu = 1;
  // 最大内存使用量（字节）
  int64 max_memory = 2;
  // 最大磁盘使用量（字节）
  int64 max_disk = 3;
  // 最大网络使用量（字节/秒）
  int64 max_network = 4;
}

// 值类型
message Value {
  oneof kind {
    // 空值
    NullValue null_value = 1;
    // 布尔值
    bool bool_value = 2;
    // 整数值
    int64 int_value = 3;
    // 浮点值
    double double_value = 4;
    // 字符串值
    string string_value = 5;
    // 字节数组
    bytes bytes_value = 6;
    // 数组
    ListValue list_value = 7;
    // 对象
    MapValue map_value = 8;
  }
}

// 空值
enum NullValue {
  NULL_VALUE = 0;
}

// 列表值
message ListValue {
  // 值列表
  repeated Value values = 1;
}

// 映射值
message MapValue {
  // 键值对
  map<string, Value> fields = 1;
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

## Python 插件支持

### Python SDK 结构

Python SDK 提供以下组件：

1. **基础类**：实现插件接口的抽象基类
2. **gRPC 服务器**：处理来自主程序的请求
3. **辅助工具**：配置解析、日志记录等
4. **类型定义**：Python 类型提示

### Python 插件基类

```python
from abc import ABC, abstractmethod
from typing import Dict, List, Any, Optional
import time

class ModuleInfo:
    """模块信息类"""
    def __init__(self, id: str, name: str, version: str, description: str = "",
                 author: str = "", license: str = "", capabilities: List[str] = None,
                 supported_platforms: List[str] = None):
        self.id = id
        self.name = name
        self.version = version
        self.description = description
        self.author = author
        self.license = license
        self.capabilities = capabilities or []
        self.supported_platforms = supported_platforms or []
        self.language = "python"

class Request:
    """请求类"""
    def __init__(self, id: str, action: str, params: Dict[str, Any] = None,
                 metadata: Dict[str, str] = None, timeout: int = 30000):
        self.id = id
        self.action = action
        self.params = params or {}
        self.metadata = metadata or {}
        self.timeout = timeout

class Response:
    """响应类"""
    def __init__(self, id: str, success: bool = True, data: Dict[str, Any] = None,
                 error: Dict[str, Any] = None, metadata: Dict[str, str] = None):
        self.id = id
        self.success = success
        self.data = data or {}
        self.error = error
        self.metadata = metadata or {}

class Event:
    """事件类"""
    def __init__(self, id: str, type: str, source: str, data: Dict[str, Any] = None,
                 metadata: Dict[str, str] = None, timestamp: int = None):
        self.id = id
        self.type = type
        self.source = source
        self.data = data or {}
        self.metadata = metadata or {}
        self.timestamp = timestamp or int(time.time() * 1000)

class Module(ABC):
    """模块基类"""
    
    @abstractmethod
    def init(self, config: Dict[str, Any]) -> None:
        """初始化模块"""
        pass
    
    @abstractmethod
    def start(self) -> None:
        """启动模块"""
        pass
    
    @abstractmethod
    def stop(self) -> None:
        """停止模块"""
        pass
    
    @abstractmethod
    def get_info(self) -> ModuleInfo:
        """获取模块信息"""
        pass
    
    @abstractmethod
    def handle_request(self, request: Request) -> Response:
        """处理请求"""
        pass
    
    @abstractmethod
    def handle_event(self, event: Event) -> bool:
        """处理事件"""
        pass
    
    def check_health(self) -> Dict[str, Any]:
        """检查健康状态"""
        return {
            "status": "healthy",
            "details": {},
            "timestamp": int(time.time() * 1000)
        }
```

### Python 插件启动器

```python
import sys
import os
import json
import logging
import grpc
import signal
from concurrent import futures

# 导入生成的gRPC代码
from kennel.plugin.proto import plugin_pb2, plugin_pb2_grpc

# 导入模块基类
from kennel.plugin.base import Module

class PluginServicer(plugin_pb2_grpc.PluginServiceServicer):
    """gRPC服务实现"""
    
    def __init__(self, module: Module):
        self.module = module
    
    def Init(self, request, context):
        """初始化插件"""
        try:
            config = self._convert_config(request.config)
            self.module.init(config)
            return plugin_pb2.InitResponse(success=True)
        except Exception as e:
            return plugin_pb2.InitResponse(
                success=False,
                error=plugin_pb2.ErrorInfo(
                    code="init_error",
                    message=str(e)
                )
            )
    
    # 其他方法实现...

def serve(module: Module, port: int = 0):
    """启动gRPC服务器"""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    plugin_pb2_grpc.add_PluginServiceServicer_to_server(
        PluginServicer(module), server
    )
    
    # 使用动态端口
    port = server.add_insecure_port(f'127.0.0.1:{port}')
    server.start()
    
    # 向标准输出写入端口信息，主程序会读取这个信息
    handshake_info = {
        "protocol_version": 1,
        "port": port,
        "plugin_id": module.get_info().id
    }
    print(f"KENNEL_PLUGIN_HANDSHAKE:{json.dumps(handshake_info)}")
    sys.stdout.flush()
    
    # 设置信号处理
    def handle_signal(signum, frame):
        server.stop(0)
        sys.exit(0)
    
    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)
    
    # 保持运行
    server.wait_for_termination()
```
