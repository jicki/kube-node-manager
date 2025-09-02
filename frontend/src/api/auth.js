import request from '@/utils/request'

const authApi = {
  // 用户登录
  login(credentials) {
    return request({
      url: '/api/v1/auth/login',
      method: 'post',
      data: credentials
    })
  },

  // 获取用户信息
  getUserInfo() {
    return request({
      url: '/api/v1/auth/user',
      method: 'get'
    })
  },

  // 刷新Token
  refreshToken() {
    return request({
      url: '/api/v1/auth/refresh',
      method: 'post'
    })
  },

  // 用户登出
  logout() {
    return request({
      url: '/api/v1/auth/logout',
      method: 'post'
    })
  },

  // 修改密码
  changePassword(data) {
    return request({
      url: '/api/v1/auth/change-password',
      method: 'post',
      data
    })
  },

  // 更新个人信息
  updateProfile(data) {
    return request({
      url: '/api/v1/auth/profile',
      method: 'put',
      data
    })
  },

  // 验证Token有效性
  validateToken() {
    return request({
      url: '/api/v1/auth/validate',
      method: 'get'
    })
  },

  // 获取系统版本信息
  getVersion() {
    return request({
      url: '/api/v1/version',
      method: 'get'
    })
  }
}

export default authApi