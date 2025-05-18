#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from abc import ABC, abstractmethod
from typing import Dict, List, Any, Optional
import time
import json
import sys
import os
import signal
import traceback

# 导入自定义模块
from .logger import create_logger, Logger, LogLevel
from .config import ConfigHelper

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

    def to_dict(self) -> Dict[str, Any]:
        """转换为字典"""
        return {
            "id": self.id,
            "name": self.name,
            "version": self.version,
            "description": self.description,
            "author": self.author,
            "license": self.license,
            "capabilities": self.capabilities,
            "supported_platforms": self.supported_platforms,
            "language": self.language
        }

class Request:
    """请求类"""
    def __init__(self, id: str, action: str, params: Dict[str, Any] = None,
                 metadata: Dict[str, str] = None, timeout: int = 30000):
        self.id = id
        self.action = action
        self.params = params or {}
        self.metadata = metadata or {}
        self.timeout = timeout

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'Request':
        """从字典创建请求"""
        return cls(
            id=data.get("id", ""),
            action=data.get("action", ""),
            params=data.get("params", {}),
            metadata=data.get("metadata", {}),
            timeout=data.get("timeout", 30000)
        )

class Response:
    """响应类"""
    def __init__(self, id: str, success: bool = True, data: Dict[str, Any] = None,
                 error: Dict[str, Any] = None, metadata: Dict[str, str] = None):
        self.id = id
        self.success = success
        self.data = data or {}
        self.error = error
        self.metadata = metadata or {}

    def to_dict(self) -> Dict[str, Any]:
        """转换为字典"""
        result = {
            "id": self.id,
            "success": self.success,
            "data": self.data,
            "metadata": self.metadata
        }
        if self.error:
            result["error"] = self.error
        return result

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

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'Event':
        """从字典创建事件"""
        return cls(
            id=data.get("id", ""),
            type=data.get("type", ""),
            source=data.get("source", ""),
            data=data.get("data", {}),
            metadata=data.get("metadata", {}),
            timestamp=data.get("timestamp", int(time.time() * 1000))
        )

    def to_dict(self) -> Dict[str, Any]:
        """转换为字典"""
        return {
            "id": self.id,
            "type": self.type,
            "source": self.source,
            "data": self.data,
            "metadata": self.metadata,
            "timestamp": self.timestamp
        }

class HealthStatus:
    """健康状态类"""
    def __init__(self, status: str = "healthy", details: Dict[str, Any] = None,
                 timestamp: int = None):
        self.status = status
        self.details = details or {}
        self.timestamp = timestamp or int(time.time() * 1000)

    def to_dict(self) -> Dict[str, Any]:
        """转换为字典"""
        return {
            "status": self.status,
            "details": self.details,
            "timestamp": self.timestamp
        }

class Module(ABC):
    """模块基类"""

    def __init__(self):
        """初始化模块"""
        self.logger = create_logger(self.__class__.__name__, "info")
        self.config = {}
        self.config_helper = ConfigHelper({})
        self.start_time = time.time()

    @abstractmethod
    def init(self, config: Dict[str, Any]) -> None:
        """初始化模块"""
        self.config = config
        self.config_helper = ConfigHelper(config)

    @abstractmethod
    def start(self) -> None:
        """启动模块"""
        self.start_time = time.time()

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

    def check_health(self) -> HealthStatus:
        """检查健康状态"""
        return HealthStatus(
            status="healthy",
            details={
                "uptime": time.time() - self.start_time
            },
            timestamp=int(time.time() * 1000)
        )

class BaseModule(Module):
    """基础模块实现"""

    def __init__(self, id: str, name: str, version: str, description: str = ""):
        """初始化基础模块"""
        super().__init__()
        self.id = id
        self.name = name
        self.version = version
        self.description = description
        self.author = ""
        self.license = ""
        self.capabilities = []
        self.supported_platforms = ["windows", "linux", "darwin"]

    def init(self, config: Dict[str, Any]) -> None:
        """初始化模块"""
        super().init(config)
        self.logger.info("初始化模块", id=self.id)

    def start(self) -> None:
        """启动模块"""
        super().start()
        self.logger.info("启动模块", id=self.id)

    def stop(self) -> None:
        """停止模块"""
        self.logger.info("停止模块", id=self.id)
        uptime = time.time() - self.start_time
        self.logger.info("运行时间", uptime=f"{uptime:.2f}秒")

    def get_info(self) -> ModuleInfo:
        """获取模块信息"""
        return ModuleInfo(
            id=self.id,
            name=self.name,
            version=self.version,
            description=self.description,
            author=self.author,
            license=self.license,
            capabilities=self.capabilities,
            supported_platforms=self.supported_platforms
        )

    def handle_request(self, request: Request) -> Response:
        """处理请求"""
        self.logger.info("处理请求", action=request.action)
        return Response(
            id=request.id,
            success=False,
            error={
                "code": "not_implemented",
                "message": f"未实现的操作: {request.action}"
            }
        )

    def handle_event(self, event: Event) -> bool:
        """处理事件"""
        self.logger.info("处理事件", type=event.type, source=event.source)
        return False

class ModuleRunner:
    """模块运行器"""

    def __init__(self, module: Module):
        """初始化运行器"""
        self.module = module
        self.logger = create_logger("ModuleRunner", "info")

    def run(self):
        """运行模块"""
        self.logger.info("模块运行器启动")

        # 设置信号处理
        signal.signal(signal.SIGINT, self._handle_signal)
        signal.signal(signal.SIGTERM, self._handle_signal)

        # 读取环境变量
        plugin_id = os.environ.get("KENNEL_PLUGIN_ID", "")
        config_json = os.environ.get("KENNEL_PLUGIN_CONFIG", "{}")

        try:
            # 解析配置
            config = json.loads(config_json)

            # 初始化模块
            self.logger.info("初始化模块")
            self.module.init(config)

            # 启动模块
            self.logger.info("启动模块")
            self.module.start()

            # 向标准输出写入就绪信息
            ready_info = {
                "status": "ready",
                "plugin_id": plugin_id,
                "info": self.module.get_info().to_dict()
            }
            print(f"KENNEL_PLUGIN_READY:{json.dumps(ready_info)}")
            sys.stdout.flush()

            # 进入命令处理循环
            self.command_loop()

        except Exception as e:
            self.logger.error(f"模块运行错误: {e}")
            self.logger.error(traceback.format_exc())
            error_info = {
                "status": "error",
                "plugin_id": plugin_id,
                "error": str(e)
            }
            print(f"KENNEL_PLUGIN_ERROR:{json.dumps(error_info)}")
            sys.stdout.flush()
            sys.exit(1)

    def _handle_signal(self, signum, frame):
        """处理信号"""
        self.logger.info(f"收到信号: {signum}")
        try:
            self.module.stop()
        except Exception as e:
            self.logger.error(f"停止模块错误: {e}")
        sys.exit(0)

    def command_loop(self):
        """命令处理循环"""
        self.logger.info("进入命令处理循环")

        try:
            while True:
                # 读取命令
                line = sys.stdin.readline().strip()
                if not line:
                    continue

                # 解析命令
                if line.startswith("KENNEL_COMMAND:"):
                    command_json = line[len("KENNEL_COMMAND:"):]
                    self.handle_command(command_json)
                elif line == "KENNEL_STOP":
                    self.logger.info("收到停止命令")
                    break
        except KeyboardInterrupt:
            self.logger.info("收到中断信号")
        except Exception as e:
            self.logger.error(f"命令处理错误: {e}")
            self.logger.error(traceback.format_exc())
        finally:
            # 停止模块
            self.logger.info("停止模块")
            try:
                self.module.stop()
            except Exception as e:
                self.logger.error(f"停止模块错误: {e}")
                self.logger.error(traceback.format_exc())

    def handle_command(self, command_json: str):
        """处理命令"""
        try:
            command = json.loads(command_json)
            command_type = command.get("type", "")
            command_data = command.get("data", {})

            if command_type == "request":
                # 处理请求
                request = Request.from_dict(command_data)
                self.logger.info(f"处理请求: {request.action}")
                response = self.module.handle_request(request)
                self.send_response(response)
            elif command_type == "event":
                # 处理事件
                event = Event.from_dict(command_data)
                self.logger.info(f"处理事件: {event.type}")
                success = self.module.handle_event(event)
                self.send_event_response(event.id, success)
            elif command_type == "health_check":
                # 健康检查
                self.logger.info("健康检查")
                health = self.module.check_health()
                self.send_health_response(health)
            else:
                self.logger.warning(f"未知命令类型: {command_type}")
        except Exception as e:
            self.logger.error(f"处理命令错误: {e}")
            self.logger.error(traceback.format_exc())
            self.send_error_response(str(e))

    def send_response(self, response: Response):
        """发送响应"""
        response_json = json.dumps(response.to_dict())
        print(f"KENNEL_RESPONSE:{response_json}")
        sys.stdout.flush()

    def send_event_response(self, event_id: str, success: bool):
        """发送事件响应"""
        response = {
            "event_id": event_id,
            "success": success
        }
        response_json = json.dumps(response)
        print(f"KENNEL_EVENT_RESPONSE:{response_json}")
        sys.stdout.flush()

    def send_health_response(self, health: HealthStatus):
        """发送健康响应"""
        response = health.to_dict()
        response_json = json.dumps(response)
        print(f"KENNEL_HEALTH_RESPONSE:{response_json}")
        sys.stdout.flush()

    def send_error_response(self, error: str):
        """发送错误响应"""
        response = {
            "error": error
        }
        response_json = json.dumps(response)
        print(f"KENNEL_ERROR:{response_json}")
        sys.stdout.flush()

def run_module(module: Module):
    """运行模块"""
    runner = ModuleRunner(module)
    runner.run()
