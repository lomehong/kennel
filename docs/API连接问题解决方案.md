# API连接问题解决方案

## 问题描述

在使用Web控制台时，插件管理页面无法显示插件数据，控制台报错：

```
GET http://localhost:8088/api/plugins net::ERR_CONNECTION_REFUSED
```

这表明前端无法连接到后端API服务。

## 原因分析

1. **后端服务未启动**：后端服务器可能未运行或未正确启动
2. **端口配置不正确**：后端服务器可能在不同的端口上运行
3. **防火墙阻止**：防火墙可能阻止了前端对后端的访问
4. **API路径配置错误**：API路径可能配置错误

## 解决方案

### 1. 确保后端服务正常运行

首先，确保后端服务已经启动并正常运行：

```powershell
# 在项目根目录执行
cd bin
.\agent.exe start
```

启动后，应该能看到类似以下的输出：

```
启动应用程序
启动Web控制台
Web控制台已启动，可以通过浏览器访问
```

### 2. 检查配置文件

检查`config.yaml`文件中的Web控制台配置：

```yaml
# Web控制台配置
web_console:
  enabled: true
  host: "0.0.0.0"
  port: 8088
  enable_https: false
  # ...其他配置...
```

确保：
- `enabled`设置为`true`
- `port`设置为`8088`（或者与前端配置匹配的端口）

### 3. 检查前端配置

检查`web/vite.config.ts`文件中的代理配置：

```typescript
server: {
  port: 3000,
  proxy: {
    '/api': {
      target: 'http://localhost:8088',
      changeOrigin: true,
    },
  },
},
```

确保`target`指向正确的后端地址和端口。

### 4. 使用模拟API（开发环境）

为了在后端服务不可用时仍能进行前端开发，我们添加了模拟API功能：

1. 在开发环境中，点击右下角的"模拟API"开关
2. 确认切换到模拟API
3. 页面将刷新并使用模拟数据

这样可以在后端服务不可用时继续进行前端开发和测试。

### 5. 检查网络连接

如果仍然无法连接，可以尝试：

1. 检查防火墙设置，确保允许端口`8088`的访问
2. 使用`curl`或`Postman`直接测试API：
   ```
   curl http://localhost:8088/api/plugins
   ```
3. 检查是否有其他程序占用了端口`8088`：
   ```powershell
   netstat -ano | findstr :8088
   ```

### 6. 修改端口配置

如果端口`8088`被占用，可以修改配置文件中的端口：

1. 修改`config.yaml`中的端口：
   ```yaml
   web_console:
     port: 8089  # 修改为其他可用端口
   ```

2. 修改`web/vite.config.ts`中的代理目标：
   ```typescript
   proxy: {
     '/api': {
       target: 'http://localhost:8089',  // 修改为与后端相同的端口
       changeOrigin: true,
     },
   },
   ```

3. 重新启动后端服务和前端开发服务器

## 环境变量配置

前端支持通过环境变量配置API基础URL：

1. 在`web`目录下创建`.env`文件：
   ```
   VITE_API_BASE_URL=/api
   ```

2. 或者创建`.env.development`文件用于开发环境：
   ```
   VITE_API_BASE_URL=http://localhost:8088/api
   ```

3. 或者创建`.env.production`文件用于生产环境：
   ```
   VITE_API_BASE_URL=/api
   ```

## 生产环境部署

在生产环境中，前端和后端通常部署在同一个服务器上，此时：

1. 确保`config.yaml`中的`static_dir`指向正确的前端构建目录：
   ```yaml
   web_console:
     static_dir: "web/dist"  # 或者其他前端构建输出目录
   ```

2. 确保前端构建时使用正确的API基础URL：
   ```
   VITE_API_BASE_URL=/api
   ```

3. 使用`build.ps1`脚本构建整个应用：
   ```powershell
   .\build.ps1
   ```

4. 启动应用：
   ```powershell
   cd bin
   .\agent.exe start
   ```

## 常见问题

### Q: 为什么切换到模拟API后仍然看不到数据？

A: 确保在切换后刷新页面，因为API基础URL的更改需要重新加载页面才能生效。

### Q: 如何知道后端API服务是否正常运行？

A: 可以通过以下方式检查：
- 查看后端服务的日志输出
- 使用浏览器访问`http://localhost:8088/api/ping`，应该返回`{"message":"pong"}`
- 使用`curl http://localhost:8088/api/ping`命令测试

### Q: 如何解决CORS（跨域）问题？

A: 确保`config.yaml`中的`allow_origins`配置正确：
```yaml
web_console:
  allow_origins: ["*", "http://localhost:3000"]
```

### Q: 如何在Docker环境中解决API连接问题？

A: 在Docker环境中，需要确保：
1. 容器间网络正确配置
2. 端口正确映射
3. 前端配置中使用正确的容器名或IP地址

## 结论

通过以上步骤，应该能够解决API连接问题。如果问题仍然存在，请检查应用日志以获取更详细的错误信息，或者联系技术支持团队。
