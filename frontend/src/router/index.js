import { createRouter, createWebHistory } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/store/modules/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/login/Login.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/',
      redirect: '/dashboard',
      component: () => import('@/components/layout/Layout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('@/views/dashboard/Dashboard.vue'),
          meta: { title: '概览', icon: 'Monitor', requiresAuth: true }
        },
        {
          path: 'clusters',
          name: 'ClusterManage',
          component: () => import('@/views/clusters/ClusterManage.vue'),
          meta: { title: '集群管理', icon: 'Connection', requiresAuth: true }
        },
        {
          path: 'nodes',
          name: 'NodeList',
          component: () => import('@/views/nodes/NodeList.vue'),
          meta: { title: '节点管理', icon: 'Monitor', requiresAuth: true }
        },
        {
          path: 'labels',
          name: 'LabelManage',
          component: () => import('@/views/labels/LabelManage.vue'),
          meta: { title: '标签管理', icon: 'CollectionTag', requiresAuth: true }
        },
        {
          path: 'taints',
          name: 'TaintManage',
          component: () => import('@/views/taints/TaintManage.vue'),
          meta: { title: '污点管理', icon: 'WarningFilled', requiresAuth: true }
        },
        {
          path: 'users',
          name: 'UserManage',
          component: () => import('@/views/users/UserManage.vue'),
          meta: { title: '用户管理', icon: 'User', requiresAuth: true, permission: 'admin' }
        },
        {
          path: 'audit',
          name: 'AuditLog',
          component: () => import('@/views/audit/AuditLog.vue'),
          meta: { title: '审计日志', icon: 'Document', requiresAuth: true }
        },
        {
          path: 'profile',
          name: 'UserProfile',
          component: () => import('@/views/profile/UserProfile.vue'),
          meta: { title: '个人信息', icon: 'User', requiresAuth: true }
        },
        {
          path: 'gitlab-settings',
          name: 'GitlabSettings',
          component: () => import('@/views/gitlab/GitlabSettings.vue'),
          meta: { title: 'GitLab 配置', icon: 'Setting', requiresAuth: true, permission: 'admin' }
        },
        {
          path: 'gitlab-runners',
          name: 'GitlabRunners',
          component: () => import('@/views/gitlab/GitlabRunners.vue'),
          meta: { title: 'GitLab Runners', icon: 'Connection', requiresAuth: true }
        },
        {
          path: 'gitlab-pipelines',
          name: 'GitlabPipelines',
          component: () => import('@/views/gitlab/GitlabPipelines.vue'),
          meta: { title: 'GitLab Pipelines', icon: 'List', requiresAuth: true }
        },
        {
          path: 'feishu-settings',
          name: 'FeishuSettings',
          component: () => import('@/views/feishu/FeishuSettings.vue'),
          meta: { title: '飞书配置', icon: 'Setting', requiresAuth: true, permission: 'admin' }
        },
        {
          path: 'feishu-groups',
          name: 'FeishuGroups',
          component: () => import('@/views/feishu/FeishuGroups.vue'),
          meta: { title: '飞书群组', icon: 'ChatDotSquare', requiresAuth: true }
        },
        {
          path: 'analytics',
          name: 'Analytics',
          component: () => import('@/views/analytics/Analytics.vue'),
          meta: { title: '统计分析', icon: 'DataAnalysis', requiresAuth: true }
        },
        {
          path: 'analytics/detail/:id',
          name: 'AnomalyDetail',
          component: () => import('@/views/analytics/AnomalyDetail.vue'),
          meta: { title: '异常详情', icon: 'Document', requiresAuth: true }
        },
        {
          path: 'analytics-report-settings',
          name: 'AnalyticsReportSettings',
          component: () => import('@/views/analytics/ReportSettings.vue'),
          meta: { title: '分析报告配置', icon: 'DataAnalysis', requiresAuth: true, permission: 'admin' }
        },
        {
          path: 'ansible-tasks',
          name: 'AnsibleTasks',
          component: () => import('@/views/ansible/TaskCenter.vue'),
          meta: { title: 'Ansible任务', icon: 'Operation', requiresAuth: true }
        },
        {
          path: 'ansible-templates',
          name: 'AnsibleTemplates',
          component: () => import('@/views/ansible/TaskTemplates.vue'),
          meta: { title: '任务模板', icon: 'Document', requiresAuth: true }
        },
        {
          path: 'ansible-inventories',
          name: 'AnsibleInventories',
          component: () => import('@/views/ansible/InventoryManage.vue'),
          meta: { title: '主机清单', icon: 'List', requiresAuth: true }
        },
        {
          path: 'ansible-ssh-keys',
          name: 'AnsibleSSHKeys',
          component: () => import('@/views/ansible/SSHKeyManage.vue'),
          meta: { title: 'SSH 密钥', icon: 'Key', requiresAuth: true, permission: 'admin' }
        },
        {
          path: 'ansible-schedules',
          name: 'AnsibleSchedules',
          component: () => import('@/views/ansible/ScheduleManage.vue'),
          meta: { title: '定时任务', icon: 'Timer', requiresAuth: true }
        }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/dashboard'
    }
  ]
})

// 路由守卫
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  
  // 检查是否需要认证
  if (to.meta.requiresAuth) {
    if (!authStore.token) {
      next('/login')
      return
    }
    
    // 如果有token但用户信息未加载，先获取用户信息
    if (!authStore.userInfo) {
      try {
        await authStore.getUserInfo()
      } catch (error) {
        console.error('Failed to get user info in router guard:', error)
        authStore.logout()
        next('/login')
        return
      }
    }
    
    // 检查权限
    if (to.meta.permission && !authStore.hasPermission(to.meta.permission)) {
      ElMessage.error('您没有权限访问此页面')
      next('/dashboard')
      return
    }
  }
  
  // 已登录用户访问登录页面直接跳转到首页
  if (to.path === '/login' && authStore.token) {
    next('/dashboard')
    return
  }
  
  next()
})

export default router