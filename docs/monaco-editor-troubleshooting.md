# Monaco Editor 未显示问题排查指南

## 问题描述
在创建/编辑 Ansible 模板时，Playbook 内容编辑器显示的是普通文本框，而不是 Monaco Editor。

## 原因分析
代码已经正确集成了 Monaco Editor，但需要重新构建前端才能看到效果。

## 解决方案

### 方案 1：重新构建 Docker 镜像（推荐用于生产环境）

```bash
# 进入项目目录
cd /Users/jicki/jicki/github/kube-node-manager

# 重新构建 Docker 镜像
make docker-build

# 重启容器
docker-compose down
docker-compose up -d
```

构建时间：约 3-5 分钟（取决于网络速度）

### 方案 2：启动前端开发服务器（推荐用于开发测试）

```bash
# 进入前端目录
cd /Users/jicki/jicki/github/kube-node-manager/frontend

# 确保依赖已安装
npm install

# 启动开发服务器
npm run dev
```

然后访问：http://localhost:3000

开发服务器会自动热重载，修改代码后立即生效。

### 方案 3：仅重新构建前端静态文件

```bash
# 进入前端目录
cd /Users/jicki/jicki/github/kube-node-manager/frontend

# 构建前端
npm run build

# 重启后端服务以加载新的静态文件
# (如果使用 Docker)
docker-compose restart kube-node-manager
```

## 验证步骤

1. **打开浏览器**（建议使用无痕模式或清除缓存）
2. **访问应用**
3. **导航到** Ansible → 任务模板
4. **点击**"创建模板"按钮
5. **查看** Playbook 内容编辑器

### 预期效果

✅ **Monaco Editor 正确加载时应该看到：**

- 深色主题的代码编辑器
- 行号显示
- 语法高亮（YAML）
- 工具栏（格式化、撤销、重做、全屏按钮）
- 代码折叠功能
- 智能提示（输入时）
- 底部状态栏（行号、列号信息）

### 浏览器控制台检查

如果 Monaco Editor 仍未显示，按 F12 打开开发者工具，查看控制台是否有以下错误：

**常见错误 1：模块加载失败**
```
Failed to load module script: Expected a JavaScript module script but the server responded with a MIME type of "text/html"
```
**解决**：清除浏览器缓存，强制刷新（Ctrl+Shift+R 或 Cmd+Shift+R）

**常见错误 2：Monaco workers 加载失败**
```
Failed to load resource: net::ERR_FILE_NOT_FOUND
.../monaco-editor/esm/vs/base/worker/workerMain.js
```
**解决**：重新构建前端，确保 vite-plugin-monaco-editor 正常工作

**常见错误 3：Vue 组件未注册**
```
[Vue warn]: Failed to resolve component: MonacoEditor
```
**解决**：检查 TaskTemplates.vue 是否正确导入 MonacoEditor 组件

## 依赖版本确认

运行以下命令确认依赖已正确安装：

```bash
cd /Users/jicki/jicki/github/kube-node-manager/frontend
npm list @guolao/vue-monaco-editor monaco-editor vite-plugin-monaco-editor
```

**预期输出：**
```
kube-node-manager-frontend@1.0.14
├─┬ @guolao/vue-monaco-editor@1.6.0
│ └── monaco-editor@0.44.0 deduped
├── monaco-editor@0.44.0
└─┬ vite-plugin-monaco-editor@1.1.0
  └── monaco-editor@0.44.0 deduped
```

## 配置文件检查清单

### ✅ vite.config.js
```javascript
import monacoEditorPlugin from 'vite-plugin-monaco-editor'

export default defineConfig({
  plugins: [
    vue(),
    // ...
    monacoEditorPlugin({
      languages: ['yaml', 'json', 'javascript', 'typescript', 'shell']
    })
  ]
})
```

### ✅ TaskTemplates.vue
```vue
<script setup>
import MonacoEditor from '@/components/MonacoEditor.vue'
</script>

<template>
  <MonacoEditor
    v-model="templateForm.playbook_content"
    language="yaml"
    theme="vs-dark"
    height="500px"
    :readonly="isViewMode"
    :show-toolbar="!isViewMode"
  />
</template>
```

### ✅ MonacoEditor.vue
组件文件位置：`frontend/src/components/MonacoEditor.vue`

## 常见问题

### Q: 为什么开发环境能看到但生产环境看不到？
A: 因为 Monaco Editor 的 workers 文件需要被正确打包。确保：
1. `vite-plugin-monaco-editor` 在 `package.json` 的 `devDependencies` 中
2. Vite 构建时正确处理了 Monaco Editor 的静态资源
3. Docker 镜像包含了所有必要的静态文件

### Q: 编辑器加载很慢
A: Monaco Editor 体积较大（约 8MB），首次加载需要时间。可以：
1. 启用 CDN 加速
2. 配置 HTTP/2
3. 启用 gzip 压缩
4. 只加载需要的语言特性

### Q: 如何自定义编辑器主题？
A: 修改 `MonacoEditor.vue` 中的 `theme` prop：
- `vs` - 浅色主题
- `vs-dark` - 深色主题（当前使用）
- `hc-black` - 高对比度主题

## 联系支持

如果以上方案都无法解决问题，请：
1. 收集浏览器控制台错误信息
2. 收集 Network 标签页的请求详情
3. 提供 Docker logs 或前端构建日志
4. 提交 Issue 或联系技术支持

---

**文档更新时间**：2025-10-31  
**适用版本**：v1.0.14+

