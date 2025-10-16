import { defineStore } from 'pinia'
import * as feishuApi from '@/api/feishu'

export const useFeishuStore = defineStore('feishu', {
  state: () => ({
    settings: null,
    groups: [],
    loading: false,
    error: null
  }),

  getters: {
    isEnabled: (state) => state.settings?.enabled || false,
    hasAppSecret: (state) => state.settings?.has_app_secret || false
  },

  actions: {
    // Fetch Feishu settings
    async fetchSettings() {
      this.loading = true
      this.error = null
      try {
        const response = await feishuApi.getFeishuSettings()
        this.settings = response.data
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || 'Failed to fetch Feishu settings'
        throw error
      } finally {
        this.loading = false
      }
    },

    // Update Feishu settings
    async updateSettings(data) {
      this.loading = true
      this.error = null
      try {
        const response = await feishuApi.updateFeishuSettings(data)
        this.settings = response.data
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || 'Failed to update Feishu settings'
        throw error
      } finally {
        this.loading = false
      }
    },

    // Test Feishu connection
    async testConnection(data) {
      this.loading = true
      this.error = null
      try {
        const response = await feishuApi.testFeishuConnection(data)
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || 'Connection test failed'
        throw error
      } finally {
        this.loading = false
      }
    },

    // Query specific group
    async queryGroup(chatId) {
      this.loading = true
      this.error = null
      try {
        const response = await feishuApi.queryFeishuGroup(chatId)
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || 'Failed to query group'
        throw error
      } finally {
        this.loading = false
      }
    },

    // Fetch all groups
    async fetchGroups() {
      this.loading = true
      this.error = null
      try {
        const response = await feishuApi.listFeishuGroups()
        this.groups = response.data
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || 'Failed to fetch groups'
        throw error
      } finally {
        this.loading = false
      }
    },

    // Clear error
    clearError() {
      this.error = null
    }
  }
})

