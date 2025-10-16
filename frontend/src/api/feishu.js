import request from '@/utils/request'

// Get Feishu settings
export const getFeishuSettings = () => {
  return request.get('/api/v1/feishu/settings')
}

// Update Feishu settings
export const updateFeishuSettings = (data) => {
  return request.put('/api/v1/feishu/settings', data)
}

// Test Feishu connection
export const testFeishuConnection = (data) => {
  return request.post('/api/v1/feishu/test', data)
}

// Query specific chat group
export const queryFeishuGroup = (chatId) => {
  return request.post('/api/v1/feishu/groups/query', { chat_id: chatId })
}

// List all chat groups
export const listFeishuGroups = () => {
  return request.get('/api/v1/feishu/groups')
}

// User binding
export const bindFeishuUser = (data) => {
  return request.post('/api/v1/feishu/bind', data)
}

export const unbindFeishuUser = () => {
  return request.delete('/api/v1/feishu/bind')
}

export const getFeishuBinding = () => {
  return request.get('/api/v1/feishu/bind')
}

