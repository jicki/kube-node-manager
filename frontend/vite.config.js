import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import { fileURLToPath, URL } from 'node:url'
import { readFileSync } from 'fs'
import { resolve } from 'path'

// 读取VERSION文件
let version = 'dev'
try {
  const versionPath = resolve(__dirname, '../VERSION')
  version = readFileSync(versionPath, 'utf-8').trim()
} catch (err) {
  console.warn('Could not read VERSION file, using default version')
}

export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      resolvers: [ElementPlusResolver()],
      imports: ['vue', 'vue-router', 'pinia']
    }),
    Components({
      resolvers: [ElementPlusResolver()]
    })
  ],
  define: {
    // 注入版本信息到前端
    __APP_VERSION__: JSON.stringify(version),
    __BUILD_TIME__: JSON.stringify(new Date().toISOString())
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        // 从环境变量读取API目标地址，默认为本地开发地址
        target: process.env.VITE_API_TARGET || 'http://localhost:8080',
        changeOrigin: true,
        secure: process.env.VITE_API_TARGET?.startsWith('https') || false
      }
    }
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets'
  }
})