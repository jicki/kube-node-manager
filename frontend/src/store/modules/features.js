import { defineStore } from 'pinia'
import request from '@/utils/request'

export const useFeaturesStore = defineStore('features', {
  state: () => ({
    features: {
      automation: {
        enabled: false,
        ansible: {
          enabled: true,
          binary_path: '/usr/bin/ansible-playbook',
          temp_dir: '/tmp/ansible-runs',
          timeout: 3600
        },
        ssh: {
          enabled: true,
          timeout: 30,
          max_concurrent: 50,
          connection_pool_size: 20
        },
        scripts: {
          enabled: true,
          timeout: 600
        },
        workflows: {
          enabled: true,
          max_steps: 50,
          step_timeout: 1800
        }
      }
    },
    loading: false,
    error: null
  }),

  getters: {
    // 是否启用自动化功能
    isAutomationEnabled: (state) => state.features.automation.enabled,
    
    // 是否启用 Ansible 功能
    isAnsibleEnabled: (state) => 
      state.features.automation.enabled && state.features.automation.ansible.enabled,
    
    // 是否启用 SSH 功能
    isSSHEnabled: (state) => 
      state.features.automation.enabled && state.features.automation.ssh.enabled,
    
    // 是否启用脚本功能
    isScriptsEnabled: (state) => 
      state.features.automation.enabled && state.features.automation.scripts.enabled,
    
    // 是否启用工作流功能
    isWorkflowsEnabled: (state) => 
      state.features.automation.enabled && state.features.automation.workflows.enabled,

    // 获取所有功能配置
    allFeatures: (state) => state.features
  },

  actions: {
    // 获取功能特性状态
    async fetchFeatures() {
      this.loading = true
      this.error = null
      try {
        const response = await request.get('/api/v1/features')
        if (response.data && response.data.data) {
          this.features = response.data.data
        }
      } catch (error) {
        console.error('Failed to fetch features:', error)
        this.error = error.response?.data?.message || '获取功能配置失败'
        // 如果获取失败，使用默认配置（所有功能关闭）
        this.features.automation.enabled = false
      } finally {
        this.loading = false
      }
    },

    // 更新自动化主开关
    async updateAutomationEnabled(enabled) {
      try {
        await request.put('/api/v1/features/automation/enabled', { enabled })
        this.features.automation.enabled = enabled
        return { success: true }
      } catch (error) {
        console.error('Failed to update automation enabled:', error)
        return { 
          success: false, 
          message: error.response?.data?.message || '更新失败' 
        }
      }
    },

    // 更新 Ansible 配置
    async updateAnsibleConfig(config) {
      try {
        await request.put('/api/v1/features/automation/ansible', config)
        this.features.automation.ansible = config
        return { success: true }
      } catch (error) {
        console.error('Failed to update ansible config:', error)
        return { 
          success: false, 
          message: error.response?.data?.message || '更新失败' 
        }
      }
    },

    // 更新 SSH 配置
    async updateSSHConfig(config) {
      try {
        await request.put('/api/v1/features/automation/ssh', config)
        this.features.automation.ssh = config
        return { success: true }
      } catch (error) {
        console.error('Failed to update ssh config:', error)
        return { 
          success: false, 
          message: error.response?.data?.message || '更新失败' 
        }
      }
    },

    // 重置为默认配置
    resetToDefault() {
      this.features = {
        automation: {
          enabled: false,
          ansible: {
            enabled: true,
            binary_path: '/usr/bin/ansible-playbook',
            temp_dir: '/tmp/ansible-runs',
            timeout: 3600
          },
          ssh: {
            enabled: true,
            timeout: 30,
            max_concurrent: 50,
            connection_pool_size: 20
          },
          scripts: {
            enabled: true,
            timeout: 600
          },
          workflows: {
            enabled: true,
            max_steps: 50,
            step_timeout: 1800
          }
        }
      }
    }
  }
})

