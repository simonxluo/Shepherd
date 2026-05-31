import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import { readFileSync } from 'fs';

// 读取配置文件获取端口号
function getConfigPort(): number {
  try {
    const configPath = path.resolve(__dirname, 'public/config.yaml');
    const configContent = readFileSync(configPath, 'utf-8');
    const portMatch = configContent.match(/port:\s*(\d+)/);
    return portMatch ? parseInt(portMatch[1], 10) : 3000;
  } catch {
    return 3000; // 默认端口
  }
}

const configPort = getConfigPort();

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  // 公共目录，用于存放 config.yaml
  publicDir: 'public',
  server: {
    port: configPort,
    // 代理 API 请求到后端服务器
    proxy: {
      '/api': {
        target: 'http://localhost:9190',
        changeOrigin: true,
        secure: false,
      },
      '/v1': {
        target: 'http://localhost:9190',
        changeOrigin: true,
        secure: false,
      },
    },
    host: true, // 监听所有地址，方便局域网访问
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': [
            'react',
            'react-dom',
            'react-router-dom'
          ],
          'ui-vendor': [
            'zustand'
          ]
        }
      }
    }
  },
  optimizeDeps: {
    include: ['react', 'react-dom', 'react-router-dom'],
  },
});
