import axios from 'axios';

// 获取API基础URL，优先使用环境变量，然后使用默认值
const getBaseUrl = () => {
  // 如果定义了环境变量，使用环境变量
  if (import.meta.env.VITE_API_BASE_URL) {
    return import.meta.env.VITE_API_BASE_URL;
  }

  // 开发环境下，默认使用本地模拟API
  if (import.meta.env.DEV) {
    // 检查是否明确禁用了模拟API
    if (localStorage.getItem('use_mock_api') === 'false') {
      console.log('使用真实API');
      return '/api';
    }

    // 默认使用模拟API
    console.log('使用模拟API');
    return '/mock-api';
  }

  // 默认使用相对路径
  return '/api';
};

// 创建axios实例
const api = axios.create({
  baseURL: getBaseUrl(),
  timeout: 30000, // 增加超时时间到30秒，避免系统资源API超时
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    // 添加CSRF令牌
    const csrfToken = document.querySelector('meta[name="csrf-token"]')?.getAttribute('content');
    if (csrfToken) {
      config.headers['X-CSRF-Token'] = csrfToken;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    // 保存CSRF令牌
    const csrfToken = response.headers['x-csrf-token'];
    if (csrfToken) {
      let meta = document.querySelector('meta[name="csrf-token"]');
      if (!meta) {
        meta = document.createElement('meta');
        meta.setAttribute('name', 'csrf-token');
        document.head.appendChild(meta);
      }
      meta.setAttribute('content', csrfToken);
    }

    // 检查响应数据是否包含错误信息
    if (response.data && response.data.error) {
      console.error('API错误:', response.data.error);
      // 不拒绝Promise，让调用者处理错误
    }

    return response;
  },
  (error) => {
    // 处理网络错误
    if (!error.response) {
      console.error('网络错误:', error.message);
    }
    // 处理HTTP错误
    else {
      const { status, statusText, data } = error.response;
      console.error(`HTTP错误 ${status} (${statusText}):`, data);

      // 特殊处理某些HTTP状态码
      switch (status) {
        case 401:
          // 未授权，重新加载页面触发基本认证
          window.location.reload();
          break;
        case 403:
          console.error('权限不足，请联系管理员');
          break;
        case 404:
          console.error('请求的资源不存在');
          break;
        case 500:
          console.error('服务器内部错误');
          break;
      }
    }

    return Promise.reject(error);
  }
);

// 插件API
export const pluginsApi = {
  // 获取所有插件
  getPlugins: () => api.get('/plugins'),

  // 获取插件详情
  getPlugin: (id: string) => api.get(`/plugins/${id}`),

  // 更新插件状态
  updatePluginStatus: (id: string, enabled: boolean) =>
    api.put(`/plugins/${id}/status`, { enabled }),

  // 更新插件配置
  updatePluginConfig: (id: string, config: any) =>
    api.put(`/plugins/${id}/config`, { config }),

  // 获取插件日志
  getPluginLogs: (id: string, limit = 100, offset = 0, level = '') =>
    api.get(`/plugins/${id}/logs`, { params: { limit, offset, level } }),
};

// 指标API
export const metricsApi = {
  // 获取所有指标
  getMetrics: () => api.get('/metrics'),

  // 获取通讯指标
  getCommMetrics: () => api.get('/metrics/comm'),

  // 获取系统指标
  getSystemMetrics: () => api.get('/metrics/system'),
};

// 系统API
export const systemApi = {
  // 获取系统状态
  getSystemStatus: () => api.get('/system/status'),

  // 获取资源使用情况
  getSystemResources: () => api.get('/system/resources', {
    timeout: 60000, // 为系统资源API单独设置更长的超时时间（60秒）
    headers: {
      'Cache-Control': 'no-cache', // 禁用缓存
    },
  }),

  // 获取系统日志
  getSystemLogs: (limit = 100, offset = 0, level = '', source = '') =>
    api.get('/system/logs', { params: { limit, offset, level, source } }),

  // 获取系统事件
  getSystemEvents: (limit = 100, offset = 0, type = '', source = '') =>
    api.get('/system/events', { params: { limit, offset, type, source } }),
};

// 配置API
export const configApi = {
  // 获取配置
  getConfig: () => api.get('/config'),

  // 更新配置
  updateConfig: (config: any) => api.put('/config', { config }),

  // 重置配置
  resetConfig: () => api.post('/config/reset'),
};

// 通讯管理API
export const commApi = {
  // 获取通讯状态
  getCommStatus: () => api.get('/comm/status').then(response => response.data),

  // 连接到服务器
  connectComm: (params: any) => api.post('/comm/connect', params).then(response => response.data),

  // 断开连接
  disconnectComm: () => api.post('/comm/disconnect').then(response => response.data),

  // 获取通讯配置
  getCommConfig: () => api.get('/comm/config').then(response => response.data),

  // 获取通讯统计信息
  getCommStats: () => api.get('/comm/stats').then(response => response.data),

  // 获取通讯日志
  getCommLogs: (limit = 100, offset = 0, level = '') =>
    api.get('/comm/logs', { params: { limit, offset, level } }).then(response => response.data),

  // 测试通讯连接
  testCommConnection: (params: any) => api.post('/comm/test/connection', params).then(response => response.data),

  // 测试通讯发送和接收
  testCommSendReceive: (params: any) => api.post('/comm/test/send-receive', params).then(response => response.data),

  // 测试通讯加密
  testCommEncryption: (params: any) => api.post('/comm/test/encryption', params).then(response => response.data),

  // 测试通讯压缩
  testCommCompression: (params: any) => api.post('/comm/test/compression', params).then(response => response.data),

  // 测试通讯性能
  testCommPerformance: (params: any) => api.post('/comm/test/performance', params).then(response => response.data),

  // 获取通讯测试历史记录
  getCommTestHistory: () => api.get('/comm/test/history').then(response => response.data),
};

// 导出API函数
export const {
  getCommStatus,
  connectComm,
  disconnectComm,
  getCommConfig,
  getCommStats,
  getCommLogs,
  testCommConnection,
  testCommSendReceive,
  testCommEncryption,
  testCommCompression,
  testCommPerformance,
  getCommTestHistory,
} = commApi;

export default api;
