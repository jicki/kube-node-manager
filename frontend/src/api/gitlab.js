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
