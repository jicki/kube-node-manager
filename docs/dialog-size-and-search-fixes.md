# 弹出框大小调整和搜索功能修复

## 修复概述

本次更新主要解决了两个重要问题：
1. 调整增大标签/污点应用到节点的弹出框大小
2. 修复标签管理中搜索没效果的问题

## 1. 弹出框大小调整

### 问题描述
标签管理和污点管理中的"应用到节点"弹出框太小，特别是在使用新的 NodeSelector 组件时，显示空间不足，影响用户体验。

### 解决方案
**弹出框宽度调整**:
- 原始宽度：`600px`
- 调整后宽度：`1200px`
- 添加响应式支持：移动端使用 `95vw`

**涉及文件**:
- `/frontend/src/views/taints/TaintManage.vue`
- `/frontend/src/views/labels/LabelManage.vue`

**具体修改**:
```vue
<!-- 修改前 -->
<el-dialog
  v-model="applyDialogVisible"
  :title="`应用模板: ${selectedTemplate?.name}`"
  width="600px"
>

<!-- 修改后 -->
<el-dialog
  v-model="applyDialogVisible"
  :title="`应用模板: ${selectedTemplate?.name}`"
  width="1200px"
  class="apply-dialog"
>
```

**CSS 样式增强**:
```css
/* 对话框样式 */
.apply-dialog {
  max-width: 90vw;
}

.apply-dialog .el-dialog__body {
  padding: 20px 24px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .apply-dialog {
    width: 95vw !important;
  }
}
```

### 用户体验改进
- **更大的显示空间**: 节点选择器有足够空间显示节点详情
- **更好的可读性**: 节点信息、状态、标签等一目了然
- **响应式设计**: 在不同屏幕尺寸下都有良好的显示效果
- **操作便捷性**: 批量选择节点更加方便

## 2. 标签管理搜索功能修复

### 问题描述
标签管理中的搜索功能存在逻辑错误，用户输入搜索关键词后，搜索结果没有正确显示，搜索功能失效。

### 问题分析
**根本原因**:
1. `filteredLabels` 计算属性的条件逻辑错误
2. 搜索状态管理不当
3. 搜索结果与原始数据的切换逻辑有问题

**原始错误代码**:
```javascript
const filteredLabels = computed(() => {
  const result = filteredAndSortedLabels.value.length > 0 ? 
    filteredAndSortedLabels.value : labels.value
  // 问题：当搜索结果为空时，length为0，会错误地使用原始数据
})
```

### 解决方案

**1. 添加搜索状态管理**:
```javascript
const isSearchActive = ref(false)

const applyFiltersAndSort = (keyword, filters) => {
  isSearchActive.value = !!(keyword || filters.type)
  // ... 搜索逻辑
}
```

**2. 修复计算属性逻辑**:
```javascript
const filteredLabels = computed(() => {
  // 如果有搜索条件，使用筛选后的结果，否则使用原始数据
  const result = isSearchActive.value ? filteredAndSortedLabels.value : labels.value
  
  return result.slice(
    (pagination.current - 1) * pagination.size,
    pagination.current * pagination.size
  )
})
```

**3. 添加搜索清空处理**:
```javascript
const handleSearchClear = () => {
  isSearchActive.value = false
  filteredAndSortedLabels.value = labels.value
}
```

**4. 在 SearchBox 组件中添加清空事件**:
```vue
<SearchBox
  v-model="searchKeyword"
  placeholder="搜索标签键值..."
  :advanced-search="true"
  :filters="searchFilters"
  :realtime="true"
  @search="handleSearch"
  @clear="handleSearchClear"
/>
```

**5. 数据加载时的状态处理**:
```javascript
const fetchLabels = async () => {
  // ... 数据加载逻辑
  if (response.data && response.data.code === 200) {
    labels.value = data.templates || []
    // 只在非搜索状态下重置筛选结果
    if (!isSearchActive.value) {
      filteredAndSortedLabels.value = labels.value
    }
  }
}
```

### 搜索功能特性

**搜索范围**:
- 模板名称模糊搜索
- 模板描述模糊搜索  
- 标签Key模糊搜索

**筛选条件**:
- 标签类型：系统标签、自定义标签
- 使用状态：使用中、未使用

**搜索逻辑**:
- 实时搜索（防抖处理）
- 大小写不敏感
- 支持组合搜索（关键词 + 筛选条件）

## 3. 一致性改进

为了保持功能一致性，同时为污点管理也添加了搜索清空处理功能：

```javascript
// 污点管理中也添加了相同的清空处理
const handleSearchClear = () => {
  filteredAndSortedTaints.value = taints.value
}
```

## 技术实现细节

### 状态管理优化
- 使用 `isSearchActive` 明确区分搜索状态和普通状态
- 避免依赖数组长度判断搜索状态的不可靠方法
- 在数据变化时正确维护搜索状态

### 组件通信改进
- SearchBox 组件通过事件与父组件通信
- 支持 `@search` 和 `@clear` 事件
- 父组件可以响应搜索清空操作

### 响应式设计考虑
- 大屏幕：1200px 宽度提供充足空间
- 中等屏幕：90vw 避免超出屏幕
- 小屏幕：95vw 最大化利用屏幕空间

## 测试验证

### 功能测试
- ✅ 前端构建成功
- ✅ 后端编译通过
- ✅ 所有单元测试通过
- ✅ 搜索功能正常工作
- ✅ 弹出框大小适中

### 用户体验测试
- ✅ 搜索响应及时
- ✅ 搜索结果准确
- ✅ 清空搜索功能正常
- ✅ 弹出框显示完整
- ✅ 移动端适配良好

## 影响范围

### 用户界面
- 标签管理页面搜索功能恢复正常
- 污点管理页面搜索功能保持正常
- 应用到节点的弹出框更大更友好

### 代码结构  
- 修复了搜索状态管理逻辑
- 添加了搜索清空事件处理
- 改进了CSS响应式设计

### 兼容性
- 向后兼容，不影响现有功能
- 新的弹出框大小提供更好的体验
- 搜索功能修复解决了用户痛点

## 后续改进建议

1. **性能优化**: 可考虑为大量数据添加虚拟滚动
2. **搜索增强**: 可添加搜索高亮显示
3. **用户偏好**: 可允许用户自定义弹出框大小
4. **搜索历史**: 可考虑添加最近搜索记录
5. **快捷操作**: 可添加常用筛选条件的快捷按钮