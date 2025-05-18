import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';
import {
  mockPlugins,
  mockPluginDetails,
  mockSystemStatus,
  mockSystemResources,
  mockMetrics,
  mockConfig
} from './src/services/mockApi';
import {
  mockCommStatus,
  mockCommConfig,
  mockCommStats,
  mockCommLogs,
  mockCommConnectionTest,
  mockCommSendReceiveTest,
  mockCommEncryptionTest,
  mockCommCompressionTest,
  mockCommPerformanceTest,
  mockCommTestHistory
} from './src/services/mockCommApi';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8088',
        changeOrigin: true,
        configure: (proxy, options) => {
          // 添加错误处理
          proxy.on('error', (err, req, res) => {
            console.log('代理错误:', err);
            if (!res.headersSent) {
              res.writeHead(500, {
                'Content-Type': 'application/json',
              });
            }
            res.end(JSON.stringify({ error: 'proxy_error', message: '无法连接到后端服务' }));
          });
        },
      },
      '/mock-api': {
        target: 'http://localhost:3000',
        changeOrigin: true,
        configure: (proxy, options) => {
          proxy.on('proxyReq', (proxyReq, req, res) => {
            // 拦截请求并返回模拟数据
            const url = req.url?.replace('/mock-api', '');

            // 根据请求路径返回不同的模拟数据
            if (url === '/plugins') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockPlugins));
              return true;
            }
            else if (url?.match(/^\/plugins\/[^\/]+$/)) {
              const id = url.split('/').pop();
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockPluginDetails[id] || { error: 'not_found' }));
              return true;
            }
            else if (url === '/system/status') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockSystemStatus));
              return true;
            }
            else if (url === '/system/resources') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockSystemResources));
              return true;
            }
            else if (url === '/metrics') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockMetrics));
              return true;
            }
            else if (url === '/config') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockConfig));
              return true;
            }
            // 通信测试相关的模拟API
            else if (url === '/comm/status') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockCommStatus));
              return true;
            }
            else if (url === '/comm/config') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockCommConfig));
              return true;
            }
            else if (url === '/comm/stats') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockCommStats));
              return true;
            }
            else if (url === '/comm/logs') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockCommLogs));
              return true;
            }
            else if (url === '/comm/test/history') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockCommTestHistory));
              return true;
            }
            else if (url === '/comm/test/connection' && req.method === 'POST') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockCommConnectionTest));
              return true;
            }
            else if (url === '/comm/test/send-receive' && req.method === 'POST') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockCommSendReceiveTest));
              return true;
            }
            else if (url === '/comm/test/encryption' && req.method === 'POST') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockCommEncryptionTest));
              return true;
            }
            else if (url === '/comm/test/compression' && req.method === 'POST') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockCommCompressionTest));
              return true;
            }
            else if (url === '/comm/test/performance' && req.method === 'POST') {
              res.writeHead(200, { 'Content-Type': 'application/json' });
              res.end(JSON.stringify(mockCommPerformanceTest));
              return true;
            }

            // 默认返回404
            res.writeHead(404, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ error: 'not_found', path: url }));
            return true;
          });
        },
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: false,
  },
});
