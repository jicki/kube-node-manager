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
        const { token, user } = response.data
        
        this.token = token
        this.userInfo = user
        this.permissions = user.permissions || []
        
        setToken(token)
        
        return response
      } catch (error) {
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