# DLP v2.0 API参考文档

## 概述

DLP v2.0 提供了完整的RESTful API接口，支持系统管理、策略配置、监控查询等功能。

## 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **认证方式**: Bearer Token
- **数据格式**: JSON
- **字符编码**: UTF-8

## 认证

### 获取访问令牌

```http
POST /auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

**响应示例**:
```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "refresh_token": "refresh_token_here"
  }
}
```

### 使用令牌

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## 系统管理 API

### 1. 系统状态

#### 获取系统状态
```http
GET /system/status
```

**响应示例**:
```json
{
  "code": 200,
  "data": {
    "status": "running",
    "version": "2.0.0",
    "uptime": "72h30m15s",
    "components": {
      "interceptor": "running",
      "parser": "running",
      "analyzer": "running",
      "executor": "running"
    }
  }
}
```

#### 获取系统指标
```http
GET /system/metrics
```

**响应示例**:
```json
{
  "code": 200,
  "data": {
    "performance": {
      "packets_processed": 1234567,
      "packets_per_second": 1500,
      "average_latency_ms": 2.5,
      "memory_usage_mb": 512,
      "cpu_usage_percent": 15.6
    },
    "detection": {
      "total_analyzed": 98765,
      "sensitive_detected": 234,
      "false_positives": 12,
      "accuracy_rate": 0.95
    }
  }
}
```

### 2. 配置管理

#### 获取配置
```http
GET /config
```

#### 更新配置
```http
PUT /config
Content-Type: application/json

{
  "interceptor": {
    "enabled": true,
    "mode": "active",
    "interfaces": ["eth0"]
  },
  "analyzer": {
    "ocr_enabled": true,
    "ml_enabled": true
  }
}
```

## 策略管理 API

### 1. 策略CRUD

#### 获取策略列表
```http
GET /policies?page=1&size=20&category=pii
```

**响应示例**:
```json
{
  "code": 200,
  "data": {
    "total": 50,
    "page": 1,
    "size": 20,
    "items": [
      {
        "id": "policy_001",
        "name": "身份证号检测",
        "category": "pii",
        "enabled": true,
        "risk_level": "high",
        "actions": ["alert", "block"],
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

#### 创建策略
```http
POST /policies
Content-Type: application/json

{
  "name": "信用卡号检测",
  "description": "检测信用卡号码泄露",
  "category": "financial",
  "enabled": true,
  "risk_level": "high",
  "conditions": [
    {
      "type": "regex",
      "pattern": "\\b(?:\\d{4}[-\\s]?){3}\\d{4}\\b",
      "confidence": 0.9
    }
  ],
  "actions": [
    {
      "type": "alert",
      "config": {
        "channels": ["email", "webhook"]
      }
    },
    {
      "type": "block",
      "config": {
        "duration": "1h"
      }
    }
  ]
}
```

#### 更新策略
```http
PUT /policies/{policy_id}
```

#### 删除策略
```http
DELETE /policies/{policy_id}
```

### 2. 规则管理

#### 获取规则列表
```http
GET /rules?type=regex&enabled=true
```

#### 创建正则规则
```http
POST /rules/regex
Content-Type: application/json

{
  "name": "手机号检测",
  "pattern": "1[3-9]\\d{9}",
  "type": "phone",
  "category": "pii",
  "confidence": 0.9,
  "enabled": true
}
```

#### 创建关键词规则
```http
POST /rules/keyword
Content-Type: application/json

{
  "name": "机密文档",
  "keywords": ["机密", "内部", "confidential"],
  "type": "classification",
  "case_sensitive": false,
  "whole_word": true,
  "confidence": 0.8,
  "enabled": true
}
```

## 监控查询 API

### 1. 事件查询

#### 获取检测事件
```http
GET /events?start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&risk_level=high
```

**响应示例**:
```json
{
  "code": 200,
  "data": {
    "total": 100,
    "events": [
      {
        "id": "event_001",
        "timestamp": "2024-01-01T12:30:00Z",
        "type": "sensitive_data_detected",
        "risk_level": "high",
        "risk_score": 0.95,
        "source_ip": "192.168.1.100",
        "dest_ip": "203.0.113.1",
        "protocol": "HTTP",
        "matched_rules": ["身份证号检测"],
        "actions_taken": ["alert", "block"],
        "details": {
          "sensitive_type": "id_card",
          "masked_value": "11****19901201****",
          "context": "用户注册表单提交"
        }
      }
    ]
  }
}
```

#### 获取事件详情
```http
GET /events/{event_id}
```

### 2. 统计分析

#### 获取检测统计
```http
GET /statistics/detection?period=7d&group_by=risk_level
```

**响应示例**:
```json
{
  "code": 200,
  "data": {
    "period": "7d",
    "total_events": 1500,
    "groups": [
      {
        "key": "critical",
        "count": 50,
        "percentage": 3.33
      },
      {
        "key": "high", 
        "count": 200,
        "percentage": 13.33
      },
      {
        "key": "medium",
        "count": 500,
        "percentage": 33.33
      },
      {
        "key": "low",
        "count": 750,
        "percentage": 50.0
      }
    ]
  }
}
```

#### 获取趋势分析
```http
GET /statistics/trends?metric=detection_count&period=30d&interval=1d
```

## 执行器管理 API

### 1. 阻断管理

#### 获取阻断列表
```http
GET /executors/block/connections
```

**响应示例**:
```json
{
  "code": 200,
  "data": {
    "blocked_connections": [
      {
        "id": "block_001",
        "source_ip": "192.168.1.100",
        "dest_ip": "203.0.113.1",
        "port": 80,
        "protocol": "TCP",
        "reason": "敏感数据传输",
        "blocked_at": "2024-01-01T12:30:00Z",
        "expires_at": "2024-01-01T13:30:00Z"
      }
    ]
  }
}
```

#### 手动阻断IP
```http
POST /executors/block/ip
Content-Type: application/json

{
  "ip": "203.0.113.1",
  "duration": "1h",
  "reason": "手动阻断可疑IP"
}
```

#### 解除阻断
```http
DELETE /executors/block/ip/{ip}
```

### 2. 隔离管理

#### 获取隔离文件
```http
GET /executors/quarantine/files
```

#### 恢复隔离文件
```http
POST /executors/quarantine/files/{file_id}/restore
```

#### 删除隔离文件
```http
DELETE /executors/quarantine/files/{file_id}
```

## 分析器管理 API

### 1. OCR管理

#### 启用OCR功能
```http
POST /analyzers/ocr/enable
Content-Type: application/json

{
  "languages": ["eng", "chi_sim"],
  "confidence_threshold": 0.8
}
```

#### 测试OCR
```http
POST /analyzers/ocr/test
Content-Type: multipart/form-data

image: [binary data]
```

### 2. 机器学习

#### 获取模型信息
```http
GET /analyzers/ml/models
```

#### 训练模型
```http
POST /analyzers/ml/train
Content-Type: application/json

{
  "model_type": "text_classifier",
  "training_data": [
    {
      "text": "这是敏感信息",
      "label": "sensitive"
    }
  ]
}
```

## 错误处理

### 错误响应格式
```json
{
  "code": 400,
  "message": "请求参数错误",
  "error": "INVALID_PARAMETER",
  "details": {
    "field": "email",
    "reason": "格式不正确"
  },
  "timestamp": "2024-01-01T12:30:00Z",
  "request_id": "req_123456"
}
```

### 常见错误码
- `200`: 成功
- `400`: 请求参数错误
- `401`: 未授权
- `403`: 权限不足
- `404`: 资源不存在
- `429`: 请求频率限制
- `500`: 服务器内部错误

## 限流和配额

### 请求限制
- **认证接口**: 10次/分钟
- **查询接口**: 100次/分钟
- **管理接口**: 50次/分钟

### 响应头
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
```

## SDK示例

### Go SDK
```go
import "github.com/lomehong/kennel/sdk/go"

client := dlp.NewClient("http://localhost:8080", "your-token")
events, err := client.GetEvents(dlp.EventQuery{
    StartTime: time.Now().Add(-24 * time.Hour),
    EndTime:   time.Now(),
    RiskLevel: "high",
})
```

### Python SDK
```python
from dlp_sdk import DLPClient

client = DLPClient("http://localhost:8080", "your-token")
events = client.get_events(
    start_time="2024-01-01T00:00:00Z",
    end_time="2024-01-02T00:00:00Z",
    risk_level="high"
)
```

### JavaScript SDK
```javascript
import { DLPClient } from 'dlp-sdk';

const client = new DLPClient('http://localhost:8080', 'your-token');
const events = await client.getEvents({
  startTime: '2024-01-01T00:00:00Z',
  endTime: '2024-01-02T00:00:00Z',
  riskLevel: 'high'
});
```

## Webhook通知

### 配置Webhook
```http
POST /webhooks
Content-Type: application/json

{
  "url": "https://your-server.com/dlp-webhook",
  "events": ["sensitive_data_detected", "policy_violation"],
  "secret": "your-webhook-secret"
}
```

### Webhook负载示例
```json
{
  "event_type": "sensitive_data_detected",
  "timestamp": "2024-01-01T12:30:00Z",
  "data": {
    "event_id": "event_001",
    "risk_level": "high",
    "sensitive_type": "id_card",
    "source_ip": "192.168.1.100"
  },
  "signature": "sha256=..."
}
```

这个API文档提供了DLP v2.0系统的完整API接口说明，包括认证、系统管理、策略配置、监控查询等各个方面的功能。
