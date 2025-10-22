import { ElMessage, ElNotification, ElMessageBox } from 'element-plus'

/**
 * 错误级别定义
 */
export const ErrorLevel = {
  INFO: 'info',
  WARNING: 'warning',
  ERROR: 'error',
  CRITICAL: 'critical'
}

/**
 * 统一错误处理函数
 * @param {Error|string} error - 错误对象或错误消息
 * @param {string} level - 错误级别
 * @param {Object} options - 额外选项
 */
export function handleError(error, level = ErrorLevel.ERROR, options = {}) {
  // 提取错误消息
  let message = '操作失败'
  
  if (typeof error === 'string') {
    message = error
  } else if (error?.response?.data?.message) {
    message = error.response.data.message
  } else if (error?.message) {
    message = error.message
  }

  // 根据级别显示不同的提示
  switch (level) {
    case ErrorLevel.INFO:
      ElMessage.info({
        message,
        duration: options.duration || 3000,
        showClose: true
      })
      break

    case ErrorLevel.WARNING:
      ElMessage.warning({
        message,
        duration: options.duration || 5000,
        showClose: true
      })
      break

    case ErrorLevel.ERROR:
      ElNotification.error({
        title: options.title || '错误',
        message,
        duration: options.duration || 4500,
        position: options.position || 'top-right'
      })
      break

    case ErrorLevel.CRITICAL:
      ElMessageBox.alert(message, options.title || '严重错误', {
        confirmButtonText: options.confirmButtonText || '确定',
        type: 'error',
        center: true
      })
      break

    default:
      ElMessage.error({
        message,
        duration: 3000,
        showClose: true
      })
  }

  // 开发环境下打印详细错误
  if (process.env.NODE_ENV === 'development') {
    console.error('[ErrorHandler]', {
      level,
      message,
      error,
      options
    })
  }
}

/**
 * 快捷方法：显示成功消息
 */
export function showSuccess(message, duration = 3000) {
  ElMessage.success({
    message,
    duration,
    showClose: true
  })
}

/**
 * 快捷方法：显示信息消息
 */
export function showInfo(message, duration = 3000) {
  ElMessage.info({
    message,
    duration,
    showClose: true
  })
}

/**
 * 快捷方法：显示警告消息
 */
export function showWarning(message, duration = 5000) {
  ElMessage.warning({
    message,
    duration,
    showClose: true
  })
}

/**
 * 快捷方法：显示错误消息
 */
export function showError(message, duration = 4500) {
  ElNotification.error({
    title: '错误',
    message,
    duration,
    position: 'top-right'
  })
}

/**
 * HTTP 状态码错误处理
 */
export function handleHttpError(error) {
  const status = error?.response?.status
  
  switch (status) {
    case 400:
      handleError('请求参数错误，请检查输入', ErrorLevel.WARNING)
      break
    case 401:
      handleError('认证失败，请重新登录', ErrorLevel.ERROR, {
        title: '认证错误'
      })
      // 可以在这里触发登出逻辑
      break
    case 403:
      handleError('权限不足，无法执行此操作', ErrorLevel.ERROR, {
        title: '权限错误'
      })
      break
    case 404:
      handleError('请求的资源不存在', ErrorLevel.WARNING)
      break
    case 409:
      handleError('操作冲突，请刷新后重试', ErrorLevel.WARNING)
      break
    case 422:
      handleError('数据验证失败，请检查输入', ErrorLevel.WARNING)
      break
    case 429:
      handleError('请求过于频繁，请稍后再试', ErrorLevel.WARNING)
      break
    case 500:
      handleError('服务器内部错误，请联系管理员', ErrorLevel.ERROR, {
        title: '服务器错误'
      })
      break
    case 502:
    case 503:
    case 504:
      handleError('服务暂时不可用，请稍后重试', ErrorLevel.ERROR, {
        title: '服务不可用'
      })
      break
    default:
      handleError(error, ErrorLevel.ERROR)
  }
}

/**
 * 网络错误处理
 */
export function handleNetworkError(error) {
  if (!navigator.onLine) {
    handleError('网络连接已断开，请检查网络设置', ErrorLevel.ERROR, {
      title: '网络错误'
    })
  } else if (error.code === 'ECONNABORTED') {
    handleError('请求超时，请检查网络或稍后重试', ErrorLevel.WARNING)
  } else if (error.message === 'Network Error') {
    handleError('网络错误，无法连接到服务器', ErrorLevel.ERROR, {
      title: '网络错误'
    })
  } else {
    handleError(error, ErrorLevel.ERROR)
  }
}

/**
 * 业务逻辑错误处理
 */
export function handleBusinessError(code, message) {
  // 根据业务错误码自定义处理逻辑
  const errorMap = {
    'CLUSTER_NOT_FOUND': '集群不存在',
    'NODE_NOT_FOUND': '节点不存在',
    'PERMISSION_DENIED': '权限不足',
    'INVALID_PARAMETER': '参数错误',
    'OPERATION_FAILED': '操作失败'
  }

  const errorMessage = errorMap[code] || message || '操作失败'
  handleError(errorMessage, ErrorLevel.WARNING)
}

/**
 * 统一异常处理入口
 * 用于 axios 拦截器
 */
export function handleException(error) {
  if (error.response) {
    // 服务器返回错误响应
    handleHttpError(error)
  } else if (error.request) {
    // 请求已发送但没有收到响应
    handleNetworkError(error)
  } else {
    // 请求配置出错
    handleError(error.message || '请求配置错误', ErrorLevel.ERROR)
  }

  return Promise.reject(error)
}

export default {
  handleError,
  showSuccess,
  showInfo,
  showWarning,
  showError,
  handleHttpError,
  handleNetworkError,
  handleBusinessError,
  handleException,
  ErrorLevel
}

