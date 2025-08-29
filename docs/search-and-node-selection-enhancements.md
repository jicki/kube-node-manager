# 搜索功能统一与节点选择优化

## 概述

本次更新统一了标签管理与污点管理的搜索功能，并优化了节点应用功能，增加了节点搜索、筛选和全选功能。

## 主要功能改进

### 1. 统一搜索功能

#### SearchBox 组件
- **位置**: `/frontend/src/components/common/SearchBox.vue`
- **功能**: 统一的高级搜索组件
- **特性**:
  - 实时搜索支持
  - 高级筛选选项
  - 搜索历史记录
  - 搜索建议
  - 可配置的筛选字段

#### 标签管理搜索
- **搜索范围**: 模板名称、描述、标签Key
- **筛选选项**:
  - 标签类型：系统标签、自定义标签
  - 使用状态：使用中、未使用
- **实现位置**: `/frontend/src/views/labels/LabelManage.vue`

#### 污点管理搜索
- **搜索范围**: 模板名称、描述、污点Key
- **筛选选项**:
  - 污点效果：NoSchedule、PreferNoSchedule、NoExecute
  - 排序方式：创建时间、名称、污点数量
- **实现位置**: `/frontend/src/views/taints/TaintManage.vue`

### 2. 节点选择器组件

#### NodeSelector 组件
- **位置**: `/frontend/src/components/common/NodeSelector.vue`
- **功能**: 统一的节点选择组件
- **特性**:
  - 节点名称搜索
  - 状态筛选（Ready/NotReady）
  - 角色筛选（Master/Worker）
  - 标签筛选（支持 key=value 格式）
  - 全选/取消全选功能
  - 显示节点详细信息（IP、状态、标签）

#### 节点筛选功能
- **搜索**: 支持节点名称模糊搜索
- **状态筛选**: Ready、NotReady、全部状态
- **角色筛选**: Master、Worker、全部角色
- **标签筛选**: 支持 `key=value` 或 `key` 格式筛选

#### 批量操作支持
- **全选功能**: 一键选择所有筛选后的节点
- **选择状态显示**: 显示已选择/总数量
- **清空选择**: 一键清空所有选择

### 3. 后端搜索支持

#### 污点模板搜索 API
- **端点**: `GET /api/v1/taints/templates`
- **新增参数**:
  - `search`: 搜索关键词（名称、描述、污点Key）
  - `effect`: 按效果筛选（NoSchedule|PreferNoSchedule|NoExecute）
- **实现位置**: 
  - Handler: `/backend/internal/handler/taint/taint.go`
  - Service: `/backend/internal/service/taint/taint.go`

#### 搜索逻辑
- **数据库层搜索**: 名称和描述的模糊匹配
- **应用层搜索**: 污点Key的匹配和效果筛选
- **支持组合搜索**: 关键词搜索 + 效果筛选

## 技术实现细节

### 前端架构
1. **组件复用**: SearchBox 和 NodeSelector 可在多个页面复用
2. **响应式设计**: 支持移动端和桌面端适配
3. **性能优化**: 使用防抖和虚拟滚动提升性能
4. **状态管理**: 使用 Vue 3 Composition API 管理组件状态

### 后端架构
1. **分层搜索**: 数据库查询 + 应用层过滤
2. **扩展性**: 易于添加新的搜索和筛选条件
3. **性能考虑**: 避免在数据库层进行复杂的 JSON 查询

### 搜索流程
1. **输入关键词** → 实时搜索（防抖处理）
2. **选择筛选条件** → 组合筛选
3. **应用筛选** → 更新显示结果
4. **保存搜索历史** → 便于重复使用

## 用户体验改进

### 搜索体验
- **实时搜索**: 输入即时显示结果
- **智能提示**: 搜索建议和历史记录
- **多维度筛选**: 支持关键词+条件的组合搜索
- **清晰反馈**: 搜索无结果时的友好提示

### 节点选择体验
- **可视化信息**: 显示节点状态、角色、IP等关键信息
- **便捷筛选**: 多种筛选方式快速定位目标节点
- **批量操作**: 全选功能支持快速选择大量节点
- **选择反馈**: 实时显示已选择的节点数量

## 兼容性说明

### 向后兼容
- 保持原有API接口不变
- 新增的搜索参数为可选参数
- 原有功能正常工作，新功能为增强

### 浏览器支持
- 现代浏览器（Chrome 88+, Firefox 78+, Safari 14+）
- 移动端浏览器支持
- 响应式设计适配各种屏幕尺寸

## 测试覆盖

### 前端测试
- 组件单元测试（SearchBox, NodeSelector）
- 搜索功能集成测试
- 用户交互测试

### 后端测试
- API 接口测试
- 搜索逻辑单元测试
- 数据库查询测试
- 污点验证逻辑测试（已有测试保持通过）

## 使用示例

### 搜索污点模板
```javascript
// 通过关键词搜索
handleSearch({
  keyword: "master",
  filters: {
    effect: "NoSchedule",
    sort: "created_at"
  }
})
```

### 选择节点
```javascript
// 筛选Ready状态的Master节点
<NodeSelector
  v-model="selectedNodes"
  :nodes="availableNodes"
  :show-labels="true"
  :max-label-display="3"
/>
```

### 后端搜索API
```bash
# 搜索包含"master"关键词且效果为NoSchedule的模板
GET /api/v1/taints/templates?search=master&effect=NoSchedule&page=1&page_size=10
```

## 性能优化

1. **防抖搜索**: 避免频繁API调用
2. **虚拟滚动**: 处理大量节点数据
3. **懒加载**: 按需加载组件和数据
4. **缓存机制**: 搜索结果和节点信息缓存
5. **分页加载**: 避免一次性加载过多数据

## 未来扩展

1. **更多筛选条件**: 支持更细粒度的筛选
2. **保存筛选配置**: 用户可以保存常用的筛选条件
3. **导出功能**: 导出搜索和筛选结果
4. **高级搜索语法**: 支持更复杂的搜索表达式
5. **搜索分析**: 统计用户搜索行为和热门搜索词