import { defineStore } from 'pinia'
import authApi from '@/api/auth'
import { getToken, setToken, removeToken } from '@/utils/auth'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: getToken(),
    userInfo: null,
    permissions: []
  }),

  getters: {
    isLoggedIn: (state) => !!state.token,
    username: (state) => state.userInfo?.username || '',
    role: (state) => state.userInfo?.role || '',
    hasPermission: (state) => {
      return (permission) => {
        if (state.userInfo?.role === 'admin') return true
        return state.permissions.includes(permission)
      }
    }
  },

  actions: {
    async login(credentials) {
      try {
        const response = await authApi.login(credentials)
        
        // 安全检查响应数据结构
        if (!response || !response.data) {
          throw new Error('登录响应数据为空')
        }
        
        const { token, user } = response.data
        
        // 验证必要的字段是否存在
        if (!token) {
          throw new Error('登录响应中缺少 token')
        }
        
        if (!user) {
          throw new Error('登录响应中缺少用户信息')
        }
        
        this.token = token
        this.userInfo = user
        this.permissions = user.permissions || []
        
        setToken(token)
        
        return response
      } catch (error) {
        // 提供更友好的错误信息
        if (error.response && error.response.status === 401) {
          throw new Error('用户名或密码错误')
        } else if (error.response && error.response.status >= 500) {
          throw new Error('服务器内部错误，请稍后重试')
        } else if (error.message === 'Network Error') {
          throw new Error('网络连接失败，请检查网络状态')
        }
        throw error
      }
    },

    async getUserInfo() {
      try {
        const response = await authApi.getUserInfo()
        this.userInfo = response.data
        this.permissions = response.data.permissions || []
        return response
      } catch (error) {
        this.logout()
        throw error
      }
    },

    logout() {
      this.token = null
      this.userInfo = null
      this.permissions = []
      removeToken()
    },

    async refreshToken() {
      try {
        const response = await authApi.refreshToken()
        
        // 安全检查响应数据结构
        if (!response || !response.data || !response.data.token) {
          throw new Error('刷新令牌响应无效')
        }
        
        const { token } = response.data
        this.token = token
        setToken(token)
        return response
      } catch (error) {
        this.logout()
        throw error
      }
    }
  }
})