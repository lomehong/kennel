#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import sys
import os
import time
import platform
import json
import logging
from typing import Dict, Any, List

# 添加SDK路径
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "../../pkg/sdk/python")))

try:
    from module import Module, ModuleInfo, Request, Response, Event, run_module
except ImportError:
    print("无法导入SDK模块，请确保SDK已安装")
    sys.exit(1)

class ExamplePythonPlugin(Module):
    """示例Python插件"""
    
    def __init__(self):
        """初始化插件"""
        super().__init__()
        self.config = {}
        self.start_time = time.time()
        self.logger.setLevel(logging.DEBUG)
    
    def init(self, config: Dict[str, Any]) -> None:
        """初始化模块"""
        self.logger.info("初始化Python示例插件")
        self.config = config
        self.logger.info(f"配置: {json.dumps(config)}")
    
    def start(self) -> None:
        """启动模块"""
        self.logger.info("启动Python示例插件")
        self.start_time = time.time()
    
    def stop(self) -> None:
        """停止模块"""
        self.logger.info("停止Python示例插件")
        uptime = time.time() - self.start_time
        self.logger.info(f"运行时间: {uptime:.2f}秒")
    
    def get_info(self) -> ModuleInfo:
        """获取模块信息"""
        return ModuleInfo(
            id="example-python",
            name="Python示例插件",
            version="1.0.0",
            description="使用Python实现的示例插件",
            author="Kennel Team",
            license="MIT",
            capabilities=["example"],
            supported_platforms=["windows", "linux", "darwin"]
        )
    
    def handle_request(self, request: Request) -> Response:
        """处理请求"""
        self.logger.info(f"处理请求: {request.action}")
        
        if request.action == "hello":
            return Response(
                id=request.id,
                success=True,
                data={
                    "message": "Hello from Python!",
                    "timestamp": time.time()
                }
            )
        elif request.action == "get_system_info":
            return Response(
                id=request.id,
                success=True,
                data=self._get_system_info()
            )
        elif request.action == "echo":
            message = request.params.get("message", "")
            return Response(
                id=request.id,
                success=True,
                data={
                    "message": message,
                    "timestamp": time.time()
                }
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
        self.logger.info(f"处理事件: {event.type} 来自 {event.source}")
        
        if event.type == "system.startup":
            self.logger.info("系统启动事件")
            return True
        elif event.type == "system.shutdown":
            self.logger.info("系统关闭事件")
            return True
        
        self.logger.warning(f"未处理的事件类型: {event.type}")
        return False
    
    def _get_system_info(self) -> Dict[str, Any]:
        """获取系统信息"""
        return {
            "platform": platform.platform(),
            "system": platform.system(),
            "release": platform.release(),
            "version": platform.version(),
            "architecture": platform.architecture(),
            "machine": platform.machine(),
            "processor": platform.processor(),
            "python_version": platform.python_version(),
            "node": platform.node(),
            "uptime": time.time() - self.start_time
        }

if __name__ == "__main__":
    # 创建并运行插件
    plugin = ExamplePythonPlugin()
    run_module(plugin)
