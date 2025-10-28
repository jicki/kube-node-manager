/**
 * 防抖函数工具
 * 用于优化搜索、过滤等高频操作
 */

/**
 * 创建防抖函数
 * @param {Function} func - 要防抖的函数
 * @param {number} delay - 延迟时间（毫秒），默认300ms
 * @returns {Function} 防抖后的函数
 */
export function debounce(func, delay = 300) {
  let timeoutId = null
  
  const debounced = function(...args) {
    // 清除之前的定时器
    if (timeoutId) {
      clearTimeout(timeoutId)
    }
    
    // 设置新的定时器
    timeoutId = setTimeout(() => {
      func.apply(this, args)
      timeoutId = null
    }, delay)
  }
  
  // 添加取消方法
  debounced.cancel = function() {
    if (timeoutId) {
      clearTimeout(timeoutId)
      timeoutId = null
    }
  }
  
  // 添加立即执行方法
  debounced.flush = function(...args) {
    if (timeoutId) {
      clearTimeout(timeoutId)
      timeoutId = null
    }
    func.apply(this, args)
  }
  
  return debounced
}

/**
 * 创建节流函数
 * @param {Function} func - 要节流的函数
 * @param {number} limit - 时间限制（毫秒）
 * @returns {Function} 节流后的函数
 */
export function throttle(func, limit = 300) {
  let inThrottle = false
  let lastResult
  
  return function(...args) {
    if (!inThrottle) {
      lastResult = func.apply(this, args)
      inThrottle = true
      
      setTimeout(() => {
        inThrottle = false
      }, limit)
    }
    
    return lastResult
  }
}

/**
 * 用于Vue 3 Composition API的防抖hook
 * @param {Function} func - 要防抖的函数
 * @param {number} delay - 延迟时间（毫秒）
 * @returns {Function} 防抖后的函数
 */
export function useDebounceFn(func, delay = 300) {
  return debounce(func, delay)
}

/**
 * 用于Vue 3 Composition API的节流hook
 * @param {Function} func - 要节流的函数
 * @param {number} limit - 时间限制（毫秒）
 * @returns {Function} 节流后的函数
 */
export function useThrottleFn(func, limit = 300) {
  return throttle(func, limit)
}

export default {
  debounce,
  throttle,
  useDebounceFn,
  useThrottleFn
}

