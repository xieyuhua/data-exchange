import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
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
