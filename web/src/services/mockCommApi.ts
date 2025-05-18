/**
 * 模拟通信API服务
 * 用于在后端服务不可用时提供模拟数据
 */

// 获取当前时间
const now = new Date();

// 模拟通信状态
export const mockCommStatus = {
  connected: true,
  status: "已连接",
  server: "ws://mock-server:9000/ws",
  uptime: "1h 30m 15s",
  timestamp: now.toISOString(),
};

// 模拟通信配置
export const mockCommConfig = {
  config: {
    server_url: "ws://mock-server:9000/ws",
    heartbeat_interval: "30s",
    reconnect_interval: "5s",
    max_reconnect_attempts: 10,
    security: {
      enable_tls: false,
      enable_encryption: true,
      enable_compression: true,
      compression_level: 6,
    },
  },
  timestamp: now.toISOString(),
};

// 模拟通信统计信息
export const mockCommStats = {
  messages_sent: 1024,
  messages_received: 896,
  bytes_sent: 102400,
  bytes_received: 89600,
  errors: 5,
  reconnects: 2,
  uptime: "1h 30m 15s",
  timestamp: now.toISOString(),
};

// 模拟通信日志
export const mockCommLogs = {
  logs: Array.from({ length: 10 }, (_, i) => ({
    timestamp: new Date(now.getTime() - i * 60000).toISOString(),
    level: "info",
    message: `模拟通讯日志 ${new Date(now.getTime() - i * 60000).toLocaleTimeString()}`,
    source: "comm-manager",
  })),
  limit: 10,
  offset: 0,
  level: "",
  total: 10,
};

// 模拟连接测试结果
export const mockCommConnectionTest = {
  success: true,
  message: "连接测试成功",
  duration: "0.567s",
  timestamp: now.toISOString(),
};

// 模拟发送接收测试结果
export const mockCommSendReceiveTest = {
  success: true,
  message: "发送消息成功",
  duration: "0.789s",
  response: {
    request_id: `req-${now.getTime()}`,
    success: true,
    data: "模拟响应数据",
    timestamp: now.toISOString(),
    mock: true,
  },
};

// 模拟加密测试结果
export const mockCommEncryptionTest = {
  success: true,
  message: "加密测试成功",
  duration: "0.345s",
  data_size: 1024,
  encrypted_size: 1040,
  decrypted_size: 1024,
  encryption_ratio: 1.015625,
  timestamp: now.toISOString(),
};

// 模拟压缩测试结果
export const mockCommCompressionTest = {
  success: true,
  message: "压缩测试成功",
  duration: "0.456s",
  data_size: 1024,
  compressed_size: 512,
  decompressed_size: 1024,
  compression_ratio: 0.5,
  timestamp: now.toISOString(),
};

// 模拟性能测试结果
export const mockCommPerformanceTest = {
  success: true,
  message: "性能测试成功",
  duration: "2.345s",
  result: {
    message_count: 1000,
    message_size: 1024,
    send_duration: "1.234s",
    send_throughput: 810.3728,
    send_size: 1024000,
    send_compressed_size: 512000,
    compression_ratio: 0.5,
  },
  timestamp: now.toISOString(),
};

// 模拟测试历史记录
export const mockCommTestHistory = [
  {
    type: "connection",
    timestamp: new Date(now.getTime() - 30 * 60000).toISOString(),
    server_url: "ws://mock-server:9000/ws",
    timeout: "10s",
    success: true,
    duration: "0.567s",
  },
  {
    type: "send-receive",
    timestamp: new Date(now.getTime() - 25 * 60000).toISOString(),
    message_type: "command",
    payload: {
      command: "ping",
      request_id: "req-20230517153112",
    },
    response: {
      request_id: "req-20230517153112",
      success: true,
      data: "pong",
    },
    success: true,
    duration: "0.789s",
  },
  {
    type: "encryption",
    timestamp: new Date(now.getTime() - 20 * 60000).toISOString(),
    data_size: 1024,
    encryption_key: "test-key",
    encrypted_size: 1040,
    decrypted_size: 1024,
    success: true,
    duration: "0.345s",
    encryption_ratio: 1.015625,
  },
  {
    type: "compression",
    timestamp: new Date(now.getTime() - 15 * 60000).toISOString(),
    data_size: 1024,
    compression_level: 6,
    compressed_size: 512,
    decompressed_size: 1024,
    success: true,
    duration: "0.456s",
    compression_ratio: 0.5,
  },
  {
    type: "performance",
    timestamp: new Date(now.getTime() - 10 * 60000).toISOString(),
    message_count: 1000,
    message_size: 1024,
    enable_encryption: true,
    enable_compression: true,
    compression_level: 6,
    success: true,
    duration: "2.345s",
    result: {
      send_duration: "1.234s",
      send_throughput: 810.3728,
      send_size: 1024000,
      send_compressed_size: 512000,
      compression_ratio: 0.5,
    },
  },
];
