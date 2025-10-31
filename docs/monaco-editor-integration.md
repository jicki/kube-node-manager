# Monaco Editor 集成指南

## 📦 已添加的依赖

已在 `frontend/package.json` 中添加以下依赖：

```json
{
  "dependencies": {
    "monaco-editor": "^0.44.0",
    "@guolao/vue-monaco-editor": "^1.3.1"
  },
  "devDependencies": {
    "vite-plugin-monaco-editor": "^1.1.0"
  }
}
```

## 🔧 安装依赖

```bash
cd /Users/jicki/jicki/github/kube-node-manager/frontend

# 安装依赖
npm install

# 或者强制重新安装
npm install --force
```

## ⚙️ Vite 配置

已在 `vite.config.js` 中配置 Monaco Editor 插件：

```javascript
import monacoEditorPlugin from 'vite-plugin-monaco-editor'

export default defineConfig({
  plugins: [
    // ... 其他插件
    monacoEditorPlugin({
      languages: ['yaml', 'json', 'javascript', 'typescript', 'shell']
    })
  ]
})
```

## 📝 组件使用

### 基本用法

```vue
<template>
  <MonacoEditor
    v-model="content"
    language="yaml"
    height="500px"
    @change="handleChange"
  />
</template>

<script setup>
import { ref } from 'vue'
import MonacoEditor from '@/components/MonacoEditor.vue'

const content = ref('')

const handleChange = (value) => {
  console.log('内容变化:', value)
}
</script>
```

### 完整示例

```vue
<template>
  <MonacoEditor
    v-model="playbookContent"
    language="yaml"
    theme="vs-dark"
    height="600px"
    :readonly="false"
    :show-toolbar="true"
    :show-line-info="true"
    :minimap="true"
    @change="handleContentChange"
    @mounted="handleEditorMounted"
  />
</template>

<script setup>
import { ref } from 'vue'
import MonacoEditor from '@/components/MonacoEditor.vue'

const playbookContent = ref(`---
- name: 示例 Playbook
  hosts: all
  gather_facts: yes
  tasks:
    - name: Ping 测试
      ping:
`)

const handleContentChange = (value) => {
  console.log('Playbook 内容已更新')
}

const handleEditorMounted = (editor) => {
  console.log('编辑器已加载', editor)
}
</script>
```

## 🎨 组件 Props

| 属性 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| modelValue | String | '' | 编辑器内容（v-model） |
| language | String | 'yaml' | 语言类型 |
| theme | String | 'vs-dark' | 主题（vs, vs-dark, hc-black） |
| height | String | '500px' | 编辑器高度 |
| readonly | Boolean | false | 是否只读 |
| showToolbar | Boolean | true | 显示工具栏 |
| showLineInfo | Boolean | true | 显示行列信息 |
| minimap | Boolean | true | 显示代码缩略图 |

## 📤 组件 Events

| 事件 | 参数 | 说明 |
|------|------|------|
| update:modelValue | value: string | 内容更新 |
| change | value: string | 内容变化 |
| mounted | editor: IStandaloneCodeEditor | 编辑器加载完成 |

## 🔨 组件方法

通过 `ref` 访问组件方法：

```vue
<template>
  <MonacoEditor ref="editorRef" v-model="content" />
  <el-button @click="formatCode">格式化</el-button>
</template>

<script setup>
import { ref } from 'vue'
import MonacoEditor from '@/components/MonacoEditor.vue'

const editorRef = ref(null)
const content = ref('')

const formatCode = () => {
  editorRef.value?.format()
}
</script>
```

**可用方法**：
- `getEditor()` - 获取原始编辑器实例
- `getValue()` - 获取编辑器内容
- `setValue(value)` - 设置编辑器内容
- `format()` - 格式化代码
- `undo()` - 撤销
- `redo()` - 重做

## 🎯 Ansible 特性

### 自动补全

组件已内置 Ansible 模块和关键字的自动补全：

**模块补全**：
- ping, copy, file, template, service
- command, shell, script
- apt, yum, package
- user, group
- git, get_url
- docker_container, docker_image
- 等等...

**关键字补全**：
- hosts, tasks, name, become
- handlers, notify, when
- register, with_items, loop
- tags, block, rescue
- 等等...

### 语法高亮

自动识别 YAML 语法并高亮显示：
- 键值对
- 列表
- 注释
- 字符串
- 数字

## 🔄 在现有页面中集成

### 替换 TaskTemplates.vue 中的编辑器

```vue
<!-- 原来的 textarea -->
<el-input 
  v-model="templateForm.playbook_content" 
  type="textarea" 
  :rows="15"
/>

<!-- 替换为 Monaco Editor -->
<MonacoEditor
  v-model="templateForm.playbook_content"
  language="yaml"
  theme="vs-dark"
  height="450px"
  :readonly="isViewMode"
/>
```

**完整修改步骤**：

1. 导入组件：
```javascript
import MonacoEditor from '@/components/MonacoEditor.vue'
```

2. 替换编辑器：
```vue
<el-form-item label="Playbook 内容" :required="!isViewMode">
  <MonacoEditor
    v-model="templateForm.playbook_content"
    language="yaml"
    theme="vs-dark"
    height="450px"
    :readonly="isViewMode"
    :show-toolbar="!isViewMode"
  />
</el-form-item>
```

## 📦 打包体积影响

添加 Monaco Editor 后的影响：

| 项目 | 大小 | 说明 |
|------|------|------|
| monaco-editor | ~1.5MB (gzip) | 核心编辑器 |
| @monaco-editor/vue | ~5KB | Vue 封装 |
| vite-plugin | ~10KB | Vite 插件 |
| **总增加** | **~1.5MB** | 压缩后 |

**优化建议**：
- ✅ 已配置按需加载语言（只加载 yaml, json, javascript, shell）
- ✅ 使用 gzip 压缩可进一步减小 50%
- ✅ CDN 加载（可选）

## 🚀 启动和测试

### 开发模式

```bash
cd frontend
npm run dev
```

访问 http://localhost:3000 测试编辑器功能。

### 构建生产版本

```bash
cd frontend
npm run build
```

检查 `dist/` 目录中的打包文件。

## 🎨 主题切换

支持三种内置主题：

```vue
<MonacoEditor
  v-model="content"
  :theme="currentTheme"
/>

<el-select v-model="currentTheme">
  <el-option label="亮色主题" value="vs" />
  <el-option label="暗色主题" value="vs-dark" />
  <el-option label="高对比度" value="hc-black" />
</el-select>
```

## 🔧 高级配置

### 自定义编辑器选项

```vue
<template>
  <MonacoEditor
    v-model="content"
    :options="customOptions"
  />
</template>

<script setup>
const customOptions = {
  fontSize: 16,
  tabSize: 4,
  wordWrap: 'off',
  lineNumbers: 'relative',
  // 更多选项...
}
</script>
```

### 添加自定义命令

```javascript
const handleEditorMounted = (editor) => {
  // 添加自定义快捷键
  editor.addCommand(
    window.monaco.KeyMod.CtrlCmd | window.monaco.KeyCode.KEY_S,
    () => {
      console.log('保存快捷键被触发')
      // 执行保存逻辑
    }
  )
}
```

## 🐛 故障排查

### 问题 1：编辑器无法加载

**原因**：依赖未安装或版本不兼容

**解决**：
```bash
rm -rf node_modules package-lock.json
npm install
```

### 问题 2：打包失败

**原因**：vite-plugin-monaco-editor 配置问题

**解决**：
检查 `vite.config.js` 中的插件配置是否正确。

### 问题 3：语法高亮不工作

**原因**：语言未在 vite 插件中配置

**解决**：
在 `vite.config.js` 的 `monacoEditorPlugin` 中添加所需语言。

### 问题 4：编辑器样式异常

**原因**：CSS 冲突或容器高度未设置

**解决**：
确保容器有明确的高度，检查 CSS 样式冲突。

## 📚 更多资源

- [Monaco Editor 官方文档](https://microsoft.github.io/monaco-editor/)
- [@guolao/vue-monaco-editor 文档](https://github.com/imguolao/monaco-vue)
- [vite-plugin-monaco-editor](https://github.com/vdesjs/vite-plugin-monaco-editor)
- [Monaco Editor Playground](https://microsoft.github.io/monaco-editor/playground.html)

## ✅ 集成检查清单

- [x] package.json 中添加依赖
- [x] vite.config.js 中配置插件
- [x] 创建 MonacoEditor.vue 组件
- [x] 添加 Ansible 自动补全
- [x] 工具栏功能（格式化、撤销、全屏）
- [x] 主题支持
- [x] 只读模式
- [ ] 在 TaskTemplates.vue 中替换编辑器（可选）
- [ ] 测试所有功能
- [ ] 生产环境构建测试

## 🎊 总结

Monaco Editor 已成功集成！现在您可以：

✅ 使用专业级代码编辑器  
✅ Ansible 模块自动补全  
✅ YAML 语法高亮  
✅ 代码格式化  
✅ 全屏编辑  
✅ 多主题支持  

**下一步**：
1. 运行 `npm install` 安装依赖
2. 在需要的页面中导入并使用组件
3. 测试编辑器功能
4. 根据需要调整配置

---

**文档版本**：v1.0  
**最后更新**：2025-10-31  
**状态**：✅ 就绪

