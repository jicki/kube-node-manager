import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'

import App from './App.vue'
import router from './router'
import pinia from './store'
import { startGlobalPreload, preloadResources } from '@/utils/lazy-load'

const app = createApp(App)

// 注册Element Plus图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

// 全局错误处理
app.config.errorHandler = (err, vm, info) => {
  console.error('Global error:', err, info)
  
  // 生产环境可以发送错误到监控服务
  if (process.env.NODE_ENV === 'production') {
    // TODO: 发送错误到监控服务 (如 Sentry)
  }
}

// 全局警告处理
app.config.warnHandler = (msg, vm, trace) => {
  if (process.env.NODE_ENV === 'development') {
    console.warn('Global warning:', msg, trace)
  }
}

// 性能监控
if (process.env.NODE_ENV === 'development') {
  app.config.performance = true
}

// 使用插件
app.use(pinia)
app.use(router)
app.use(ElementPlus, {
  locale: zhCn,
  size: 'default'
})

// 全局属性
app.config.globalProperties.$ELEMENT = {
  size: 'default'
}

// 全局混入 - 添加通用方法
app.mixin({
  methods: {
    // 格式化文件大小
    $formatFileSize(bytes) {
      if (bytes === 0) return '0 B'
      const k = 1024
      const sizes = ['B', 'KB', 'MB', 'GB']
      const i = Math.floor(Math.log(bytes) / Math.log(k))
      return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    },
    
    // 防抖函数
    $debounce(func, wait) {
      let timeout
      return function executedFunction(...args) {
        const later = () => {
          clearTimeout(timeout)
          func(...args)
        }
        clearTimeout(timeout)
        timeout = setTimeout(later, wait)
      }
    },
    
    // 节流函数
    $throttle(func, limit) {
      let inThrottle
      return function executedFunction(...args) {
        if (!inThrottle) {
          func.apply(this, args)
          inThrottle = true
          setTimeout(() => inThrottle = false, limit)
        }
      }
    }
  }
})

// 预加载关键资源
const preloadCriticalResources = async () => {
  try {
    // 预加载字体文件
    const fonts = [
      '/fonts/element-icons.woff2',  // Element Plus 图标字体
    ]
    
    // 预加载关键图片/图标
    const images = [
      // 可以添加应用中的关键图片
    ]
    
    if (fonts.length > 0) {
      await preloadResources(fonts, { 
        type: 'font',
        onProgress: ({ loaded, total, percentage }) => {
          console.log(`字体加载进度: ${percentage}% (${loaded}/${total})`)
        }
      })
    }
    
    if (images.length > 0) {
      await preloadResources(images, { 
        type: 'image',
        onProgress: ({ loaded, total, percentage }) => {
          console.log(`图片加载进度: ${percentage}% (${loaded}/${total})`)
        }
      })
    }
  } catch (error) {
    console.warn('资源预加载失败:', error)
  }
}

// 应用启动
app.mount('#app')

// 启动预加载
preloadCriticalResources()
startGlobalPreload()

console.log('🚀 应用已启动，版本:', process.env.NODE_ENV)