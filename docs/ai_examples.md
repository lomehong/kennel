# 终端管控AI助手示例查询

本文档提供了一系列示例查询，展示终端管控AI助手的功能和使用方法。

## 1. 进程管理

### 1.1 查询进程列表

**查询**：
```
列出当前运行的进程
```

**查询**：
```
显示系统中正在运行的前10个进程
```

**查询**：
```
哪些进程占用CPU最多？
```

### 1.2 查找特定进程

**查询**：
```
查找名称包含"chrome"的进程
```

**查询**：
```
有哪些浏览器进程正在运行？
```

**查询**：
```
查找占用内存超过100MB的进程
```

### 1.3 终止进程

**查询**：
```
终止PID为1234的进程
```

**查询**：
```
关闭所有Chrome浏览器进程
```

**查询**：
```
强制结束没有响应的程序
```

## 2. 命令执行

### 2.1 网络命令

**查询**：
```
显示当前网络配置
```

**查询**：
```
检查与百度的网络连接
```

**查询**：
```
显示当前网络连接状态
```

### 2.2 系统信息

**查询**：
```
显示系统信息
```

**查询**：
```
当前目录下有哪些文件？
```

**查询**：
```
显示当前用户名
```

### 2.3 组合查询

**查询**：
```
检查系统性能，包括CPU和内存使用情况
```

**查询**：
```
我的电脑运行很慢，可以帮我诊断一下吗？
```

**查询**：
```
帮我检查网络连接问题并提供解决方案
```

## 3. 预期响应示例

### 3.1 进程列表查询响应

```
当前系统中运行的进程如下：

1. System (PID: 4)
   - CPU: 0.1%
   - 内存: 0.1%
   - 用户: SYSTEM
   - 启动时间: 2023-05-18T08:00:00Z

2. explorer.exe (PID: 1234)
   - CPU: 1.5%
   - 内存: 2.3%
   - 用户: User
   - 启动时间: 2023-05-18T08:01:23Z

3. chrome.exe (PID: 5678)
   - CPU: 2.5%
   - 内存: 5.7%
   - 用户: User
   - 启动时间: 2023-05-18T09:12:34Z

...

共有25个进程正在运行。
```

### 3.2 查找特定进程响应

```
找到以下包含"chrome"的进程：

1. chrome.exe (PID: 5678)
   - CPU: 2.5%
   - 内存: 5.7%
   - 用户: User
   - 启动时间: 2023-05-18T09:12:34Z

2. chrome.exe (PID: 5679)
   - CPU: 1.2%
   - 内存: 4.3%
   - 用户: User
   - 启动时间: 2023-05-18T09:12:35Z

3. chrome.exe (PID: 5680)
   - CPU: 0.8%
   - 内存: 3.1%
   - 用户: User
   - 启动时间: 2023-05-18T09:12:36Z

共找到3个匹配的进程。
```

### 3.3 终止进程响应

```
已成功终止PID为5678的进程(chrome.exe)。

操作详情：
- 进程名称: chrome.exe
- 进程ID: 5678
- 用户: User
- 终止时间: 2023-05-18T10:15:23Z
- 终止方式: 正常终止

进程已成功关闭。
```

### 3.4 网络配置响应

```
当前网络配置信息如下：

Windows IP 配置

以太网适配器 以太网:
   连接特定的 DNS 后缀 . . . . . . . :
   IPv4 地址 . . . . . . . . . . . . : 192.168.1.100
   子网掩码  . . . . . . . . . . . . : 255.255.255.0
   默认网关. . . . . . . . . . . . . : 192.168.1.1

无线局域网适配器 WLAN:
   连接特定的 DNS 后缀 . . . . . . . :
   IPv4 地址 . . . . . . . . . . . . : 192.168.1.101
   子网掩码  . . . . . . . . . . . . : 255.255.255.0
   默认网关. . . . . . . . . . . . . : 192.168.1.1

网络配置摘要：
- 活动网络接口: 2个
- IP地址: 192.168.1.100, 192.168.1.101
- 默认网关: 192.168.1.1
- DNS服务器: 192.168.1.1, 8.8.8.8
```

### 3.5 网络连接检查响应

```
正在检查与百度的连接...

正在 Ping baidu.com [39.156.69.79] 具有 32 字节的数据:
来自 39.156.69.79 的回复: 字节=32 时间=36ms TTL=51
来自 39.156.69.79 的回复: 字节=32 时间=36ms TTL=51
来自 39.156.69.79 的回复: 字节=32 时间=36ms TTL=51
来自 39.156.69.79 的回复: 字节=32 时间=36ms TTL=51

39.156.69.79 的 Ping 统计信息:
    数据包: 已发送 = 4，已接收 = 4，丢失 = 0 (0% 丢失)，
往返行程的估计时间(以毫秒为单位):
    最短 = 36ms，最长 = 36ms，平均 = 36ms

连接状态分析：
- 连接状态: 正常
- 平均延迟: 36ms (良好)
- 丢包率: 0% (优秀)
- 网络质量: 良好

您的网络连接正常，可以正常访问互联网。
```

### 3.6 系统诊断响应

```
系统性能诊断结果：

CPU使用情况：
- 总体使用率: 45%
- 最高占用进程: chrome.exe (15%)
- CPU核心数: 8
- CPU状态: 正常

内存使用情况：
- 总物理内存: 16GB
- 已使用内存: 8GB (50%)
- 可用内存: 8GB (50%)
- 最高占用进程: chrome.exe (1.2GB)
- 内存状态: 正常

磁盘使用情况：
- C盘总容量: 500GB
- C盘已使用: 300GB (60%)
- C盘可用: 200GB (40%)
- 磁盘状态: 正常

网络状态：
- 当前连接: 以太网
- 下载速度: 95Mbps
- 上传速度: 35Mbps
- 网络状态: 良好

系统运行时间: 3天5小时

建议：
1. 关闭不必要的Chrome标签页，减少内存占用
2. 考虑定期重启系统，优化性能
3. 检查启动项，减少不必要的程序自启动
```

## 4. 使用技巧

1. **使用自然语言**：可以使用自然语言提问，不需要使用特定的命令格式。

2. **提供具体信息**：提供具体的信息可以获得更准确的响应，例如指定进程名称或PID。

3. **组合查询**：可以组合多个查询，例如"查找占用CPU最多的Chrome进程并终止它"。

4. **流式响应**：对于复杂的查询，可以使用流式响应，获得更好的用户体验。

5. **上下文理解**：AI助手能够理解上下文，可以进行多轮对话，例如：
   - 用户：列出所有Chrome进程
   - AI：[显示进程列表]
   - 用户：终止第一个
   - AI：[终止列表中的第一个进程]
