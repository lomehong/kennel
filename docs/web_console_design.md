# Web控制台设计文档

## 概述

Web控制台是应用框架的一个重要组件，提供了一个基于Web的界面，用于管理和监控框架的运行状态。它允许用户通过浏览器查看框架的各种指标、管理插件、查看和修改配置等。

## 架构设计

Web控制台采用前后端分离的架构，后端使用Go语言实现，前端使用React和Ant Design实现。

### 后端架构

后端采用Gin框架实现，提供RESTful API供前端调用。后端主要包括以下组件：

1. **Web服务器**：负责处理HTTP请求，提供API接口
2. **API路由**：定义API路由，将请求转发到对应的处理函数
3. **处理函数**：实现API的具体功能，如获取插件列表、获取指标等
4. **中间件**：提供认证、日志、错误处理等功能
5. **服务集成**：与框架的其他组件（如插件管理器、通讯管理器等）集成

### 前端架构

前端采用React和Ant Design实现，提供用户友好的界面。前端主要包括以下组件：

1. **页面组件**：实现各个页面，如插件管理页面、指标监控页面等
2. **布局组件**：实现页面布局，如导航栏、侧边栏等
3. **数据组件**：实现数据展示，如表格、图表等
4. **表单组件**：实现表单，如配置表单、插件配置表单等
5. **API服务**：负责与后端API通信

## 功能设计

Web控制台提供以下核心功能：

### 1. 插件管理

- 显示所有已加载插件列表
- 查看插件详细信息
- 启用/禁用插件
- 配置插件参数
- 查看插件日志

### 2. 指标监控

- 显示通讯模块的连接状态
- 显示消息统计（发送/接收消息数、字节数等）
- 显示延迟指标（平均延迟、最大延迟等）
- 显示压缩指标（压缩率等）
- 显示加密指标
- 显示心跳指标
- 显示错误指标

### 3. 系统监控

- 显示框架运行状态
- 显示资源使用情况（CPU、内存等）
- 显示错误日志
- 显示系统事件

### 4. 配置管理

- 查看当前配置
- 修改配置
- 保存配置
- 重置配置

## API设计

Web控制台提供以下API接口：

### 插件管理API

- `GET /api/plugins`：获取所有插件列表
- `GET /api/plugins/:id`：获取指定插件的详细信息
- `PUT /api/plugins/:id/status`：启用/禁用插件
- `PUT /api/plugins/:id/config`：更新插件配置
- `GET /api/plugins/:id/logs`：获取插件日志

### 指标监控API

- `GET /api/metrics`：获取所有指标
- `GET /api/metrics/comm`：获取通讯模块指标
- `GET /api/metrics/system`：获取系统指标

### 系统监控API

- `GET /api/system/status`：获取系统状态
- `GET /api/system/resources`：获取资源使用情况
- `GET /api/system/logs`：获取系统日志
- `GET /api/system/events`：获取系统事件

### 配置管理API

- `GET /api/config`：获取当前配置
- `PUT /api/config`：更新配置
- `POST /api/config/reset`：重置配置

## 安全设计

Web控制台采用以下安全措施：

1. **认证**：使用基本认证或令牌认证，确保只有授权用户才能访问Web控制台
2. **授权**：根据用户角色限制访问权限
3. **HTTPS**：支持HTTPS，确保通信安全
4. **CSRF保护**：防止跨站请求伪造攻击
5. **XSS防护**：防止跨站脚本攻击
6. **请求限制**：限制请求频率，防止DoS攻击

## 配置选项

Web控制台支持以下配置选项：

- `enable_web_console`：是否启用Web控制台
- `web_console_port`：Web控制台端口
- `web_console_host`：Web控制台主机
- `web_console_username`：Web控制台用户名
- `web_console_password`：Web控制台密码
- `web_console_enable_https`：是否启用HTTPS
- `web_console_cert_file`：HTTPS证书文件
- `web_console_key_file`：HTTPS私钥文件
- `web_console_log_level`：Web控制台日志级别

## 集成方案

Web控制台将作为框架的一个可选组件，可以通过配置启用或禁用。它将与框架的其他组件（如插件管理器、通讯管理器等）集成，提供统一的管理界面。

集成步骤：

1. 在框架启动时，根据配置决定是否启动Web控制台
2. 如果启用Web控制台，创建Web控制台实例
3. 将框架的各个组件注入到Web控制台中
4. 启动Web控制台，监听指定端口
5. 在框架关闭时，优雅地关闭Web控制台

## 技术选型

- **后端框架**：Gin
- **前端框架**：React 18+
- **UI组件库**：Ant Design
- **状态管理**：React Context API
- **HTTP客户端**：Axios
- **图表库**：Ant Design Charts
- **构建工具**：Vite
