import request from '@/utils/request'

const authApi = {
  // 用户登录
  login(credentials) {
    return request({
      url: '/api/auth/login',
      method: 'post',
      data: credentials
    })
  },

  // 获取用户信息
  getUserInfo() {
    return request({
      url: '/api/auth/user',
      method: 'get'
    })
  },

  // 刷新Token
  refreshToken() {
    return request({
      url: '/api/auth/refresh',
      method: 'post'
    })
  },

  // 用户登出
  logout() {
    return request({
      url: '/api/auth/logout',
      method: 'post'
    })
  },

  // 修改密码
  changePassword(data) {
    return request({
      url: '/api/auth/change-password',
      method: 'post',
      data
    })
  },

  // 验证Token有效性
  validateToken() {
    return request({
      url: '/api/auth/validate',
      method: 'get'
    })
  }
}

export default authApi