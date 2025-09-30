import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

// Get GitLab settings
export const getGitlabSettings = () => {
  return axios.get(`${API_BASE_URL}/gitlab/settings`)
}

// Update GitLab settings
export const updateGitlabSettings = (data) => {
  return axios.put(`${API_BASE_URL}/gitlab/settings`, data)
}

// Test GitLab connection
export const testGitlabConnection = (data) => {
  return axios.post(`${API_BASE_URL}/gitlab/test`, data)
}

// List GitLab runners
export const listGitlabRunners = (params) => {
  return axios.get(`${API_BASE_URL}/gitlab/runners`, { params })
}

// List GitLab pipelines
export const listGitlabPipelines = (params) => {
  return axios.get(`${API_BASE_URL}/gitlab/pipelines`, { params })
}
