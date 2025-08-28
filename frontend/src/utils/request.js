import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '@/store/modules/auth'
import router from '@/router'

// 创建axios实例
const service = axios.create({
  // 使用环境变量或默认值，开发模式使用空字符串依赖代理
  baseURL: import.meta.env.VITE_API_BASE_URL || (import.meta.env.DEV ? '' : ''),
  timeout: 60000, // 请求超时时间
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
service.interceptors.request.use(
  config => {
    const authStore = useAuthStore()
    
    // 在请求头中添加token
    if (authStore.token) {
      config.headers.Authorization = `Bearer ${authStore.token}`
    }
    
    // 处理FormData
    if (config.data instanceof FormData) {
      config.headers['Content-Type'] = 'multipart/form-data'
    }
    
    return config
  },
  error => {
    console.error('Request error:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
service.interceptors.response.use(
  response => {
    const res = response.data
    
    // 如果自定义状态码不是200，则判断为错误
    if (res.code && res.code !== 200) {
      ElMessage({
        message: res.message || '请求失败',
        type: 'error',
        duration: 5 * 1000
      })
      
      // 处理特定错误码
      if (res.code === 401) {
        const authStore = useAuthStore()
        authStore.logout()
        router.push('/login')
      }
      
      return Promise.reject(new Error(res.message || 'Error'))
    }
    
    return res
  },
  error => {
    const { response } = error
    let message = '网络错误，请稍后重试'
    
    if (response) {
      switch (response.status) {
        case 401:
          message = '登录已过期，请重新登录'
          const authStore = useAuthStore()
          authStore.logout()
          router.push('/login')
          break
        case 403:
          message = '没有权限访问此资源'
          break
        case 404:
          message = '请求的资源不存在'
          break
        case 422:
          message = response.data?.message || '请求参数验证失败'
          break
        case 429:
          message = '请求过于频繁，请稍后再试'
          break
        case 500:
          message = '服务器内部错误'
          break
        case 502:
          message = '网关错误'
          break
        case 503:
          message = '服务暂不可用'
          break
        case 504:
          message = '网关超时'
          break
        default:
          message = response.data?.message || `请求失败 (${response.status})`
      }
    } else if (error.code === 'ECONNABORTED') {
      message = '请求超时，请稍后重试'
    } else if (error.message === 'Network Error') {
      message = '网络连接失败，请检查网络'
    }
    
    ElMessage({
      message,
      type: 'error',
      duration: 5 * 1000
    })
    
    return Promise.reject(error)
  }
)

// 通用请求方法
export const request = service

// 文件下载方法
export const downloadFile = (url, filename) => {
  return service({
    url,
    method: 'get',
    responseType: 'blob'
  }).then(response => {
    const blob = new Blob([response.data])
    const link = document.createElement('a')
    link.href = window.URL.createObjectURL(blob)
    link.download = filename
    link.click()
    window.URL.revokeObjectURL(link.href)
  })
}

// 上传文件方法
export const uploadFile = (url, file, onUploadProgress) => {
  const formData = new FormData()
  formData.append('file', file)
  
  return service({
    url,
    method: 'post',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    },
    onUploadProgress
  })
}

export default service