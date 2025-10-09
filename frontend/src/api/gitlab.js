import request from '@/utils/request'

// Get GitLab settings
export const getGitlabSettings = () => {
  return request.get('/api/v1/gitlab/settings')
}

// Update GitLab settings
export const updateGitlabSettings = (data) => {
  return request.put('/api/v1/gitlab/settings', data)
}

// Test GitLab connection
export const testGitlabConnection = (data) => {
  return request.post('/api/v1/gitlab/test', data)
}

// List GitLab runners
export const listGitlabRunners = (params) => {
  return request.get('/api/v1/gitlab/runners', { params })
}

// List GitLab pipelines
export const listGitlabPipelines = (params) => {
  return request.get('/api/v1/gitlab/pipelines', { params })
}

// Get runner details
export const getGitlabRunner = (runnerId) => {
  return request.get(`/api/v1/gitlab/runners/${runnerId}`)
}

// Update runner
export const updateGitlabRunner = (runnerId, data) => {
  return request.put(`/api/v1/gitlab/runners/${runnerId}`, data)
}

// Delete runner
export const deleteGitlabRunner = (runnerId) => {
  return request.delete(`/api/v1/gitlab/runners/${runnerId}`)
}

// Create runner
export const createGitlabRunner = (data) => {
  return request.post('/api/v1/gitlab/runners', data)
}

// Get runner token
export const getGitlabRunnerToken = (runnerId) => {
  return request.get(`/api/v1/gitlab/runners/${runnerId}/token`)
}

// Reset runner token
export const resetGitlabRunnerToken = (runnerId) => {
  return request.post(`/api/v1/gitlab/runners/${runnerId}/reset-token`)
}
