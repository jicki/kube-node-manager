import { defineStore } from 'pinia'
import * as gitlabApi from '@/api/gitlab'

export const useGitlabStore = defineStore('gitlab', {
  state: () => ({
    settings: null,
    runners: [],
    jobs: [],
    loading: false,
    error: null
  }),

  getters: {
    isEnabled: (state) => state.settings?.enabled || false,
    hasToken: (state) => state.settings?.has_token || false
  },

  actions: {
    // Fetch GitLab settings
    async fetchSettings() {
      this.loading = true
      this.error = null
      try {
        const response = await gitlabApi.getGitlabSettings()
        this.settings = response.data
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || 'Failed to fetch GitLab settings'
        throw error
      } finally {
        this.loading = false
      }
    },

    // Update GitLab settings
    async updateSettings(data) {
      this.loading = true
      this.error = null
      try {
        const response = await gitlabApi.updateGitlabSettings(data)
        this.settings = response.data
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || 'Failed to update GitLab settings'
        throw error
      } finally {
        this.loading = false
      }
    },

    // Test GitLab connection
    async testConnection(data) {
      this.loading = true
      this.error = null
      try {
        const response = await gitlabApi.testGitlabConnection(data)
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || 'Connection test failed'
        throw error
      } finally {
        this.loading = false
      }
    },

    // Fetch runners
    async fetchRunners(params = {}) {
      this.loading = true
      this.error = null
      try {
        const response = await gitlabApi.listGitlabRunners(params)
        this.runners = response.data
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || 'Failed to fetch runners'
        throw error
      } finally {
        this.loading = false
      }
    },

    // Fetch all jobs
    async fetchAllJobs(params = {}) {
      this.loading = true
      this.error = null
      try {
        const response = await gitlabApi.listAllJobs(params)
        this.jobs = response.data
        return response.data
      } catch (error) {
        this.error = error.response?.data?.error || 'Failed to fetch jobs'
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
