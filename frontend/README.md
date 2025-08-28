# Kubernetes Node Manager Frontend

基于 Vue 3 + Element Plus 的 Kubernetes 节点管理前端应用。

## 功能特性

- 🎯 **现代化 UI**: 基于 Vue 3 Composition API + Element Plus 设计
- 📱 **响应式设计**: 支持桌面端和移动端自适应
- 🔐 **权限管理**: 完整的用户认证和权限控制系统
- 🎨 **卡片式布局**: 直观的卡片式界面，颜色编码区分不同类型
- 🔍 **智能搜索**: 支持实时搜索、高级筛选和分页
- 📊 **数据可视化**: 实时统计和状态展示
- ⚡ **批量操作**: 支持节点、标签、污点的批量管理
- 🛡️ **错误处理**: 完善的错误处理和用户友好提示

## 主要页面

### 1. 登录页面 (Login.vue)
- 用户名/密码登录
- 验证码支持
- 记住用户名功能
- LDAP 登录支持

### 2. 概览页面 (Dashboard.vue)
- 集群和节点统计
- 节点状态分布图表
- 最近操作记录
- 快捷操作入口

### 3. 节点管理 (NodeList.vue)
- 节点列表展示
- 节点状态监控
- 资源使用情况
- 封锁/解封/驱逐操作
- 批量节点操作

### 4. 标签管理 (LabelManage.vue)
- 标签卡片式展示
- 标签与节点关联关系
- 批量标签应用/移除
- 标签搜索和分类

### 5. 污点管理 (TaintManage.vue)
- 污点效果可视化
- NoSchedule/PreferNoSchedule/NoExecute 支持
- 污点与节点关联管理
- 批量污点操作

### 6. 用户管理 (UserManage.vue)
- 用户角色权限管理
- 用户状态控制
- 密码重置功能
- 用户活动统计

## 技术栈

- **框架**: Vue 3 (Composition API)
- **状态管理**: Pinia
- **路由**: Vue Router 4
- **UI 组件**: Element Plus
- **图标**: Element Plus Icons
- **HTTP 客户端**: Axios
- **构建工具**: Vite
- **开发语言**: JavaScript

## 项目结构

```
src/
├── api/                    # API 接口定义
│   ├── auth.js            # 认证相关
│   ├── user.js            # 用户管理
│   ├── cluster.js         # 集群管理
│   ├── node.js            # 节点操作
│   ├── label.js           # 标签管理
│   ├── taint.js           # 污点管理
│   └── audit.js           # 审计日志
├── components/            # 组件目录
│   ├── common/           # 通用组件
│   │   ├── ConfirmDialog.vue     # 确认对话框
│   │   ├── LoadingSpinner.vue    # 加载动画
│   │   └── SearchBox.vue         # 搜索组件
│   └── layout/           # 布局组件
│       ├── Layout.vue            # 主布局
│       ├── Sidebar.vue           # 侧边栏
│       └── Header.vue            # 页头
├── router/               # 路由配置
│   └── index.js
├── store/                # 状态管理
│   ├── index.js          # Pinia 实例
│   └── modules/          # 状态模块
│       ├── auth.js       # 认证状态
│       ├── cluster.js    # 集群状态
│       └── node.js       # 节点状态
├── utils/                # 工具函数
│   ├── request.js        # HTTP 请求封装
│   ├── auth.js           # 认证工具
│   └── format.js         # 格式化工具
├── views/                # 页面组件
│   ├── login/           # 登录页面
│   ├── dashboard/       # 概览页面
│   ├── nodes/           # 节点管理
│   ├── labels/          # 标签管理
│   ├── taints/          # 污点管理
│   └── users/           # 用户管理
├── App.vue               # 根组件
└── main.js               # 入口文件
```

## 开发指南

### 环境要求

- Node.js 16+
- npm 或 yarn

### 安装依赖

```bash
npm install
```

### 启动开发服务器

```bash
npm run dev
```

应用将在 http://localhost:3000 启动

### 构建生产版本

```bash
npm run build
```

### 代码检查

```bash
npm run lint
```

## 配置说明

### 环境变量

创建 `.env.local` 文件配置环境变量：

```bash
# API 基础路径
VITE_API_BASE_URL=http://localhost:8080

# 应用版本
VITE_APP_VERSION=1.0.0

# 是否启用 LDAP 登录
VITE_ENABLE_LDAP=false
```

### API 代理配置

开发环境下，API 请求会被代理到后端服务。在 `vite.config.js` 中配置：

```javascript
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true
    }
  }
}
```

## 功能亮点

### 1. 智能搜索组件
- 实时搜索与防抖处理
- 高级筛选功能
- 搜索历史记录
- 搜索建议

### 2. 卡片式界面
- 标签和污点采用卡片布局
- 颜色编码区分不同类型
- 悬停效果和状态指示

### 3. 批量操作
- 选择多个节点进行批量操作
- 批量应用/移除标签和污点
- 操作进度提示

### 4. 权限控制
- 基于角色的权限管理
- 路由级别的权限验证
- 按钮级别的权限控制

### 5. 响应式设计
- 移动端优化
- 自适应布局
- 触摸友好的交互

## 开发规范

### 组件规范
- 使用 Vue 3 Composition API
- 组件名使用 PascalCase
- Props 定义类型和默认值
- 发出事件使用 kebab-case

### 样式规范
- 使用 scoped 样式
- 遵循 BEM 命名规范
- 使用 CSS 变量实现主题
- 响应式断点统一管理

### API 调用规范
- 统一错误处理
- 请求/响应拦截器
- 加载状态管理
- Token 自动刷新

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交变更
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License