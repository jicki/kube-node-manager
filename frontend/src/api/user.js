import request from '@/utils/request'

const userApi = {
  // 获取用户列表
  getUsers(params) {
    return request({
      url: '/api/v1/users',
      method: 'get',
      params
    })
  },

  // 获取用户详情
  getUserDetail(id) {
    return request({
      url: `/api/v1/users/${id}`,
      method: 'get'
    })
  },

  // 创建用户
  createUser(data) {
    return request({
      url: '/api/v1/users',
      method: 'post',
      data
    })
  },

  // 更新用户
  updateUser(id, data) {
    return request({
      url: `/api/v1/users/${id}`,
      method: 'put',
      data
    })
  },

  // 删除用户
  deleteUser(id) {
    return request({
      url: `/api/v1/users/${id}`,
      method: 'delete'
    })
  },

  // 批量删除用户
  batchDeleteUsers(ids) {
    return request({
      url: '/api/v1/users/batch-delete',
      method: 'post',
      data: { ids }
    })
  },

  // 重置用户密码
  resetPassword(id, data) {
    return request({
      url: `/api/v1/users/${id}/reset-password`,
      method: 'post',
      data
    })
  },

  // 启用/禁用用户
  toggleUserStatus(id, enabled) {
    return request({
      url: `/api/v1/users/${id}/toggle-status`,
      method: 'post',
      data: { enabled }
    })
  },

  // 获取用户角色列表
  getUserRoles() {
    return request({
      url: '/api/v1/users/roles',
      method: 'get'
    })
  }
}

export default userApi