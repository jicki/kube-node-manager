// 组件懒加载工具
import { defineAsyncComponent, h } from 'vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

// 默认加载组件
const DefaultLoadingComponent = {
  render() {
    return h(LoadingSpinner, {
      size: 'large',
      text: '组件加载中...'
    })
  }
}

// 默认错误组件
const DefaultErrorComponent = {
  render() {
    return h('div', {
      class: 'lazy-load-error'
    }, [
      h('div', { class: 'error-icon' }, '⚠️'),
      h('div', { class: 'error-text' }, '组件加载失败'),
      h('button', {
        class: 'retry-btn',
        onClick: () => window.location.reload()
      }, '重试')
    ])
  }
}

/**
 * 创建懒加载组件
 * @param {Function} loader - 组件加载器函数
 * @param {Object} options - 选项
 */
export function createLazyComponent(loader, options = {}) {
  const {
    loadingComponent = DefaultLoadingComponent,
    errorComponent = DefaultErrorComponent,
    delay = 200,
    timeout = 30000,
    retries = 3,
    onError = null
  } = options

  // 创建带重试机制的加载器
  const retryableLoader = async () => {
    let lastError = null
    
    for (let i = 0; i <= retries; i++) {
      try {
        const component = await loader()
        return component
      } catch (error) {
        lastError = error
        console.warn(`组件加载失败 (尝试 ${i + 1}/${retries + 1}):`, error)
        
        // 如果不是最后一次重试，等待一段时间后重试
        if (i < retries) {
          await new Promise(resolve => setTimeout(resolve, 1000 * Math.pow(2, i)))
        }
      }
    }
    
    // 所有重试都失败了
    if (onError) {
      onError(lastError)
    }
    throw lastError
  }

  return defineAsyncComponent({
    loader: retryableLoader,
    loadingComponent,
    errorComponent,
    delay,
    timeout
  })
}

/**
 * 预加载组件
 * @param {Function} loader - 组件加载器函数
 * @param {number} priority - 优先级 (0-100, 数字越大优先级越高)
 */
export function preloadComponent(loader, priority = 50) {
  return new Promise((resolve, reject) => {
    // 使用 requestIdleCallback 在浏览器空闲时预加载
    const preload = () => {
      loader()
        .then(resolve)
        .catch(reject)
    }

    if (typeof requestIdleCallback !== 'undefined') {
      requestIdleCallback(preload, { timeout: 5000 })
    } else {
      // 降级到 setTimeout
      setTimeout(preload, priority > 70 ? 0 : 1000)
    }
  })
}

/**
 * 批量预加载组件
 * @param {Array} components - 组件配置数组
 * @param {Object} options - 选项
 */
export function preloadComponents(components, options = {}) {
  const { 
    concurrent = 3, // 并发数量
    onProgress = null, // 进度回调
    onComplete = null, // 完成回调
    onError = null // 错误回调
  } = options

  const results = []
  let completed = 0
  let failed = 0

  // 按优先级排序
  const sortedComponents = [...components].sort((a, b) => 
    (b.priority || 50) - (a.priority || 50)
  )

  return new Promise((resolve, reject) => {
    const loadComponent = async (componentConfig, index) => {
      try {
        const component = await preloadComponent(
          componentConfig.loader, 
          componentConfig.priority
        )
        results[index] = { success: true, component, config: componentConfig }
        completed++
        
        if (onProgress) {
          onProgress({
            completed,
            failed,
            total: components.length,
            percentage: Math.round((completed + failed) / components.length * 100)
          })
        }
      } catch (error) {
        results[index] = { success: false, error, config: componentConfig }
        failed++
        
        if (onError) {
          onError(error, componentConfig)
        }
        
        if (onProgress) {
          onProgress({
            completed,
            failed,
            total: components.length,
            percentage: Math.round((completed + failed) / components.length * 100)
          })
        }
      }

      // 检查是否全部完成
      if (completed + failed === components.length) {
        if (onComplete) {
          onComplete({ completed, failed, results })
        }
        resolve(results)
      }
    }

    // 控制并发加载
    let currentIndex = 0
    const loadNext = () => {
      while (currentIndex < sortedComponents.length && 
             currentIndex < concurrent) {
        loadComponent(sortedComponents[currentIndex], currentIndex)
        currentIndex++
      }
    }

    loadNext()
  })
}

/**
 * 创建路由懒加载组件
 * @param {Function} loader - 路由组件加载器
 * @param {Object} options - 选项
 */
export function createLazyRoute(loader, options = {}) {
  const {
    chunkName = 'route-chunk',
    webpackPreload = false,
    webpackPrefetch = false
  } = options

  // 添加 webpack 魔法注释
  const enhancedLoader = () => {
    const comments = [`webpackChunkName: "${chunkName}"`]
    
    if (webpackPreload) {
      comments.push('webpackPreload: true')
    }
    
    if (webpackPrefetch) {
      comments.push('webpackPrefetch: true')
    }
    
    return loader()
  }

  return createLazyComponent(enhancedLoader, {
    ...options,
    delay: 100, // 路由组件加载延迟更短
    timeout: 15000 // 路由组件超时时间更短
  })
}

/**
 * 图片/资源预加载工具
 * @param {Array} resources - 资源URL数组
 * @param {Object} options - 选项
 */
export function preloadResources(resources, options = {}) {
  const {
    type = 'image', // image, script, style, font
    crossOrigin = null,
    onProgress = null,
    onComplete = null,
    onError = null
  } = options

  const results = []
  let loaded = 0
  let failed = 0

  return new Promise((resolve, reject) => {
    resources.forEach((resource, index) => {
      const link = document.createElement('link')
      link.rel = 'preload'
      
      // 设置资源类型
      if (type === 'image') {
        link.as = 'image'
        link.href = resource
      } else if (type === 'script') {
        link.as = 'script'
        link.href = resource
      } else if (type === 'style') {
        link.as = 'style'
        link.href = resource
      } else if (type === 'font') {
        link.as = 'font'
        link.href = resource
        link.type = 'font/woff2'
        link.crossOrigin = 'anonymous'
      }
      
      if (crossOrigin) {
        link.crossOrigin = crossOrigin
      }

      link.onload = () => {
        results[index] = { success: true, resource }
        loaded++
        
        if (onProgress) {
          onProgress({
            loaded,
            failed,
            total: resources.length,
            percentage: Math.round((loaded + failed) / resources.length * 100)
          })
        }
        
        if (loaded + failed === resources.length) {
          if (onComplete) {
            onComplete({ loaded, failed, results })
          }
          resolve(results)
        }
      }

      link.onerror = (error) => {
        results[index] = { success: false, error, resource }
        failed++
        
        if (onError) {
          onError(error, resource)
        }
        
        if (onProgress) {
          onProgress({
            loaded,
            failed,
            total: resources.length,
            percentage: Math.round((loaded + failed) / resources.length * 100)
          })
        }
        
        if (loaded + failed === resources.length) {
          if (onComplete) {
            onComplete({ loaded, failed, results })
          }
          resolve(results)
        }
      }

      document.head.appendChild(link)
    })
  })
}

/**
 * 智能预加载管理器
 */
export class LazyLoadManager {
  constructor() {
    this.preloadQueue = []
    this.isPreloading = false
    this.preloadedComponents = new Set()
  }

  // 添加到预加载队列
  addToQueue(loader, priority = 50, options = {}) {
    this.preloadQueue.push({ loader, priority, options })
    this.preloadQueue.sort((a, b) => b.priority - a.priority)
  }

  // 开始预加载
  async startPreload() {
    if (this.isPreloading) return
    
    this.isPreloading = true
    
    while (this.preloadQueue.length > 0) {
      const { loader, options } = this.preloadQueue.shift()
      
      try {
        const component = await preloadComponent(loader)
        this.preloadedComponents.add(loader)
        
        if (options.onSuccess) {
          options.onSuccess(component)
        }
      } catch (error) {
        if (options.onError) {
          options.onError(error)
        }
      }
      
      // 避免阻塞主线程
      await new Promise(resolve => setTimeout(resolve, 10))
    }
    
    this.isPreloading = false
  }

  // 检查组件是否已预加载
  isPreloaded(loader) {
    return this.preloadedComponents.has(loader)
  }

  // 清空队列
  clearQueue() {
    this.preloadQueue = []
  }
}

// 全局懒加载管理器实例
export const globalLazyLoadManager = new LazyLoadManager()

// 在应用启动时开始预加载
export function startGlobalPreload() {
  // 等待主要内容加载完成后开始预加载
  if (document.readyState === 'complete') {
    globalLazyLoadManager.startPreload()
  } else {
    window.addEventListener('load', () => {
      setTimeout(() => {
        globalLazyLoadManager.startPreload()
      }, 1000)
    })
  }
}
