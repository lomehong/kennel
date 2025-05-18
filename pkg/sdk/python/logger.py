#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import logging
import sys
import os
import time
from enum import Enum
from typing import Dict, Any, Optional

class LogLevel(Enum):
    """日志级别"""
    TRACE = 5
    DEBUG = 10
    INFO = 20
    WARN = 30
    ERROR = 40
    OFF = 100

class Logger:
    """日志记录器"""
    
    def __init__(self, name: str, level: LogLevel = LogLevel.INFO):
        """初始化日志记录器"""
        self.name = name
        self.logger = logging.getLogger(name)
        self.logger.setLevel(level.value)
        
        # 添加控制台处理器
        handler = logging.StreamHandler()
        formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
        handler.setFormatter(formatter)
        self.logger.addHandler(handler)
        
        # 添加TRACE级别
        logging.addLevelName(LogLevel.TRACE.value, "TRACE")
    
    def trace(self, msg: str, **kwargs):
        """记录跟踪级别日志"""
        self._log(LogLevel.TRACE.value, msg, kwargs)
    
    def debug(self, msg: str, **kwargs):
        """记录调试级别日志"""
        self._log(LogLevel.DEBUG.value, msg, kwargs)
    
    def info(self, msg: str, **kwargs):
        """记录信息级别日志"""
        self._log(LogLevel.INFO.value, msg, kwargs)
    
    def warn(self, msg: str, **kwargs):
        """记录警告级别日志"""
        self._log(LogLevel.WARN.value, msg, kwargs)
    
    def error(self, msg: str, **kwargs):
        """记录错误级别日志"""
        self._log(LogLevel.ERROR.value, msg, kwargs)
    
    def _log(self, level: int, msg: str, kwargs: Dict[str, Any]):
        """记录日志"""
        if not self.logger.isEnabledFor(level):
            return
        
        # 格式化关键字参数
        if kwargs:
            args_str = " ".join([f"{k}={repr(v)}" for k, v in kwargs.items()])
            msg = f"{msg} {args_str}"
        
        self.logger.log(level, msg)
    
    def with_fields(self, **kwargs) -> 'Logger':
        """返回带有附加字段的新日志记录器"""
        new_logger = Logger(self.name, LogLevel(self.logger.level))
        new_logger.logger = self.logger
        
        # 使用过滤器添加字段
        class FieldFilter(logging.Filter):
            def filter(self, record):
                for k, v in kwargs.items():
                    setattr(record, k, v)
                return True
        
        new_logger.logger.addFilter(FieldFilter())
        return new_logger
    
    def named(self, name: str) -> 'Logger':
        """返回带有名称的新日志记录器"""
        return Logger(f"{self.name}.{name}", LogLevel(self.logger.level))
    
    def get_level(self) -> LogLevel:
        """获取日志级别"""
        return LogLevel(self.logger.level)
    
    def set_level(self, level: LogLevel):
        """设置日志级别"""
        self.logger.setLevel(level.value)
    
    def set_output(self, output_file: str):
        """设置日志输出文件"""
        # 移除所有处理器
        for handler in self.logger.handlers[:]:
            self.logger.removeHandler(handler)
        
        # 添加文件处理器
        handler = logging.FileHandler(output_file)
        formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
        handler.setFormatter(formatter)
        self.logger.addHandler(handler)

def create_logger(name: str, level: str = "info") -> Logger:
    """创建日志记录器"""
    log_level = LogLevel.INFO
    
    if level.lower() == "trace":
        log_level = LogLevel.TRACE
    elif level.lower() == "debug":
        log_level = LogLevel.DEBUG
    elif level.lower() == "info":
        log_level = LogLevel.INFO
    elif level.lower() == "warn":
        log_level = LogLevel.WARN
    elif level.lower() == "error":
        log_level = LogLevel.ERROR
    elif level.lower() == "off":
        log_level = LogLevel.OFF
    
    return Logger(name, log_level)
