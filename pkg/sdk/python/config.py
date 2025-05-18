#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from typing import Dict, Any, List, Optional, Union, TypeVar, cast

T = TypeVar('T')

class ConfigHelper:
    """配置辅助类"""
    
    def __init__(self, config: Dict[str, Any]):
        """初始化配置辅助类"""
        self.config = config
    
    def get_string(self, key: str, default_value: str = "") -> str:
        """获取字符串配置"""
        if key in self.config:
            value = self.config[key]
            if isinstance(value, str):
                return value
        return default_value
    
    def get_int(self, key: str, default_value: int = 0) -> int:
        """获取整数配置"""
        if key in self.config:
            value = self.config[key]
            if isinstance(value, int):
                return value
            elif isinstance(value, float):
                return int(value)
            elif isinstance(value, str):
                try:
                    return int(value)
                except ValueError:
                    pass
        return default_value
    
    def get_float(self, key: str, default_value: float = 0.0) -> float:
        """获取浮点数配置"""
        if key in self.config:
            value = self.config[key]
            if isinstance(value, float):
                return value
            elif isinstance(value, int):
                return float(value)
            elif isinstance(value, str):
                try:
                    return float(value)
                except ValueError:
                    pass
        return default_value
    
    def get_bool(self, key: str, default_value: bool = False) -> bool:
        """获取布尔值配置"""
        if key in self.config:
            value = self.config[key]
            if isinstance(value, bool):
                return value
            elif isinstance(value, str):
                return value.lower() in ("true", "yes", "1", "on")
            elif isinstance(value, int):
                return value != 0
        return default_value
    
    def get_list(self, key: str, default_value: Optional[List[Any]] = None) -> List[Any]:
        """获取列表配置"""
        if default_value is None:
            default_value = []
        
        if key in self.config:
            value = self.config[key]
            if isinstance(value, list):
                return value
        return default_value
    
    def get_string_list(self, key: str, default_value: Optional[List[str]] = None) -> List[str]:
        """获取字符串列表配置"""
        if default_value is None:
            default_value = []
        
        value_list = self.get_list(key, [])
        result = []
        
        for item in value_list:
            if isinstance(item, str):
                result.append(item)
        
        if not result and default_value:
            return default_value
        
        return result
    
    def get_dict(self, key: str, default_value: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """获取字典配置"""
        if default_value is None:
            default_value = {}
        
        if key in self.config:
            value = self.config[key]
            if isinstance(value, dict):
                return value
        return default_value
    
    def get_nested(self, path: str, default_value: Any = None) -> Any:
        """获取嵌套配置"""
        keys = path.split(".")
        current = self.config
        
        for key in keys:
            if isinstance(current, dict) and key in current:
                current = current[key]
            else:
                return default_value
        
        return current
    
    def get_nested_string(self, path: str, default_value: str = "") -> str:
        """获取嵌套字符串配置"""
        value = self.get_nested(path)
        if isinstance(value, str):
            return value
        return default_value
    
    def get_nested_int(self, path: str, default_value: int = 0) -> int:
        """获取嵌套整数配置"""
        value = self.get_nested(path)
        if isinstance(value, int):
            return value
        elif isinstance(value, float):
            return int(value)
        elif isinstance(value, str):
            try:
                return int(value)
            except ValueError:
                pass
        return default_value
    
    def get_nested_bool(self, path: str, default_value: bool = False) -> bool:
        """获取嵌套布尔值配置"""
        value = self.get_nested(path)
        if isinstance(value, bool):
            return value
        elif isinstance(value, str):
            return value.lower() in ("true", "yes", "1", "on")
        elif isinstance(value, int):
            return value != 0
        return default_value
    
    def get_nested_dict(self, path: str, default_value: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """获取嵌套字典配置"""
        if default_value is None:
            default_value = {}
        
        value = self.get_nested(path)
        if isinstance(value, dict):
            return value
        return default_value
    
    def get_nested_list(self, path: str, default_value: Optional[List[Any]] = None) -> List[Any]:
        """获取嵌套列表配置"""
        if default_value is None:
            default_value = []
        
        value = self.get_nested(path)
        if isinstance(value, list):
            return value
        return default_value
