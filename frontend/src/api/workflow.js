import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

// 创建工作流
export const createWorkflow = async (data) => {
  const response = await axios.post(`${API_BASE_URL}/api/v1/ansible/workflows`, data)
  return response.data
}

// 获取工作流列表
export const listWorkflows = async (params) => {
  const response = await axios.get(`${API_BASE_URL}/api/v1/ansible/workflows`, { params })
  return response.data
}

// 获取工作流详情
export const getWorkflow = async (id) => {
  const response = await axios.get(`${API_BASE_URL}/api/v1/ansible/workflows/${id}`)
  return response.data
}

// 更新工作流
export const updateWorkflow = async (id, data) => {
  const response = await axios.put(`${API_BASE_URL}/api/v1/ansible/workflows/${id}`, data)
  return response.data
}

// 删除工作流
export const deleteWorkflow = async (id) => {
  const response = await axios.delete(`${API_BASE_URL}/api/v1/ansible/workflows/${id}`)
  return response.data
}

// 执行工作流
export const executeWorkflow = async (id) => {
  const response = await axios.post(`${API_BASE_URL}/api/v1/ansible/workflows/${id}/execute`)
  return response.data
}

// 获取工作流执行列表
export const listWorkflowExecutions = async (params) => {
  const response = await axios.get(`${API_BASE_URL}/api/v1/ansible/workflow-executions`, { params })
  return response.data
}

// 获取工作流执行详情
export const getWorkflowExecution = async (id) => {
  const response = await axios.get(`${API_BASE_URL}/api/v1/ansible/workflow-executions/${id}`)
  return response.data
}

// 取消工作流执行
export const cancelWorkflowExecution = async (id) => {
  const response = await axios.post(`${API_BASE_URL}/api/v1/ansible/workflow-executions/${id}/cancel`)
  return response.data
}

// 获取工作流执行状态（实时）
export const getWorkflowExecutionStatus = async (id) => {
  const response = await axios.get(`${API_BASE_URL}/api/v1/ansible/workflow-executions/${id}/status`)
  return response.data
}

