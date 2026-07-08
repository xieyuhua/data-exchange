import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  // 生产构建的资源基路径用 /static/，与后端 r.StaticFS("/static", ...) 对应，
  // 避免资源路径与 /api 路由冲突；开发时用根路径，保持本地代理正常。
  base: process.env.NODE_ENV === 'production' ? '/static/' : '/',
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      '/api': 'http://0.0.0.0:7856'
    }
  },
  build: {
    outDir: '../static',
    emptyOutDir: true,
    assetsDir: 'assets'
  }
})
