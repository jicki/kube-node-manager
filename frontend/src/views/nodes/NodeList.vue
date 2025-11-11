<template>
  <div class="node-list">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">节点管理</h1>
        <p class="page-description">管理Kubernetes集群节点</p>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="refreshData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <el-card class="search-card">
      <div class="search-section">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索节点名称或IP地址..."
          clearable
          @input="handleSearch"
          @clear="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
      
      <div class="filter-section">
        <el-row :gutter="12">
          <el-col :span="4">
            <el-select
              v-model="statusFilter"
              placeholder="状态筛选"
              clearable
              @change="handleFilterChange"
            >
              <el-option label="全部状态" value="" />
              <el-option label="Ready" value="Ready" />
              <el-option label="NotReady" value="NotReady" />
              <el-option label="Unknown" value="Unknown" />
            </el-select>
          </el-col>
          <el-col :span="4">
            <el-select
              v-model="roleFilter"
              placeholder="角色筛选"
              clearable
              @change="handleFilterChange"
            >
              <el-option label="全部角色" value="" />
              <el-option label="Master" value="master" />
              <el-option label="Worker" value="worker" />
            </el-select>
          </el-col>
          <el-col :span="5">
            <el-select
              v-model="schedulableFilter"
              placeholder="调度状态"
              clearable
              @change="handleFilterChange"
            >
              <el-option label="全部状态" value="" />
              <el-option label="可调度" value="schedulable" />
              <el-option label="有限调度" value="limited" />
              <el-option label="不可调度" value="unschedulable" />
            </el-select>
          </el-col>
          <el-col :span="5">
            <el-select
              v-model="nodeOwnershipFilter"
              placeholder="节点归属"
              clearable
              @change="handleFilterChange"
            >
              <el-option label="全部归属" value="" />
              <el-option 
                v-for="ownership in nodeOwnershipOptions" 
                :key="ownership" 
                :label="ownership" 
                :value="ownership" 
              />
            </el-select>
          </el-col>
          <el-col :span="6">
            <el-button type="primary" plain @click="showAdvancedSearch = !showAdvancedSearch">
              <el-icon><Filter /></el-icon>
              高级搜索
            </el-button>
          </el-col>
        </el-row>
        
        <!-- 高级搜索区域 -->
        <div v-show="showAdvancedSearch" class="advanced-search">
          <el-divider content-position="left">标签搜索</el-divider>
          <el-row :gutter="12">
            <el-col :span="12">
              <el-input
                v-model="labelKeyFilter"
                placeholder="输入标签键，如 node-role.kubernetes.io/master"
                clearable
                @input="handleFilterChange"
                @clear="handleFilterChange"
              >
                <template #prefix>
                  <el-icon><CollectionTag /></el-icon>
                </template>
              </el-input>
            </el-col>
            <el-col :span="12">
              <el-input
                v-model="labelValueFilter"
                placeholder="输入标签值（可选）"
                clearable
                @input="handleFilterChange"
                @clear="handleFilterChange"
              >
                <template #prefix>
                  <el-icon><Edit /></el-icon>
                </template>
              </el-input>
            </el-col>
          </el-row>
          
          <el-divider content-position="left">污点搜索</el-divider>
          <el-row :gutter="12">
            <el-col :span="8">
              <el-input
                v-model="taintKeyFilter"
                placeholder="输入污点键，如 node.kubernetes.io/unschedulable"
                clearable
                @input="handleFilterChange"
                @clear="handleFilterChange"
              >
                <template #prefix>
                  <el-icon><WarningFilled /></el-icon>
                </template>
              </el-input>
            </el-col>
            <el-col :span="8">
              <el-input
                v-model="taintValueFilter"
                placeholder="输入污点值（可选）"
                clearable
                @input="handleFilterChange"
                @clear="handleFilterChange"
              >
                <template #prefix>
                  <el-icon><Edit /></el-icon>
                </template>
              </el-input>
            </el-col>
            <el-col :span="8">
              <el-select
                v-model="taintEffectFilter"
                placeholder="污点效果（可选）"
                clearable
                @change="handleFilterChange"
              >
                <el-option label="NoSchedule" value="NoSchedule" />
                <el-option label="PreferNoSchedule" value="PreferNoSchedule" />
                <el-option label="NoExecute" value="NoExecute" />
              </el-select>
            </el-col>
          </el-row>
        </div>
      </div>
    </el-card>

    <!-- 批量操作栏 -->
    <div v-if="selectedNodes.length > 0" class="batch-actions">
      <div class="batch-info">
        <span>已选择 {{ selectedNodes.length }} 个节点</span>
        <el-button type="text" @click="clearSelection">清空选择</el-button>
      </div>
      <div class="batch-buttons">
        <el-button @click="batchCordon" :loading="batchLoading.cordon">
          <el-icon><Lock /></el-icon>
          禁止调度
        </el-button>
        <el-button @click="batchUncordon" :loading="batchLoading.uncordon">
          <el-icon><Unlock /></el-icon>
          解除调度
        </el-button>
        <el-button 
          v-if="authStore.role === 'admin'"
          type="danger"
          @click="batchDrain" 
          :loading="batchLoading.drain"
        >
          <el-icon><VideoPlay /></el-icon>
          驱逐节点
        </el-button>
        <el-divider direction="vertical" />
        <el-button type="warning" @click="showBatchDeleteLabelsDialog" :loading="batchLoading.deleteLabels">
          <el-icon><CollectionTag /></el-icon>
          批量删除标签
        </el-button>
        <el-button type="warning" @click="showBatchDeleteTaintsDialog" :loading="batchLoading.deleteTaints">
          <el-icon><WarningFilled /></el-icon>
          批量删除污点
        </el-button>
      </div>
    </div>

    <!-- 节点表格 -->
    <el-card class="table-card">
      <el-table
        v-loading="loading"
        :data="filteredNodes"
        style="width: 100%"
        @selection-change="handleSelectionChange"
        @sort-change="handleSortChange"
      >
        <!-- 空状态 -->
        <template #empty>
          <div class="empty-content">
            <el-empty
              v-if="!clusterStore.hasCluster"
              description="暂无集群配置"
              :image-size="100"
            >
              <template #description>
                <p>您还没有配置任何Kubernetes集群</p>
                <p>请先添加集群配置以开始管理节点</p>
              </template>
              <el-button type="primary" @click="$router.push('/clusters')">
                <el-icon><Plus /></el-icon>
                添加集群
              </el-button>
            </el-empty>
            
            <el-empty
              v-else
              description="当前集群暂无节点数据"
              :image-size="80"
            >
              <template #description>
                <p>当前集群中没有找到节点</p>
                <p>请检查集群连接状态或稍后重试</p>
              </template>
              <el-button @click="refreshData">
                <el-icon><Refresh /></el-icon>
                刷新数据
              </el-button>
            </el-empty>
          </div>
        </template>
        <el-table-column type="selection" width="55" />
        
        <el-table-column
          prop="name"
          label="节点名称"
          sortable="custom"
          min-width="200"
        >
          <template #default="{ row }">
            <div class="node-name-cell">
              <el-button
                type="text"
                class="node-name-link"
                @click="viewNodeDetail(row)"
              >
                {{ row.name }}
              </el-button>
              
              <!-- 标签行 -->
              <div class="node-info-row" v-if="hasLabelsToShow(row)">
                <span class="info-label">标签:</span>
                <div class="info-tags">
                  <!-- 显示主要角色标签 -->
                  <el-tag
                    v-for="role in getVisibleRoles(row.roles)"
                    :key="role"
                    :type="getNodeRoleType(role)"
                    size="small"
                    class="role-tag"
                  >
                    {{ formatNodeRoles([role]) }}
                  </el-tag>
                  
                  <!-- 显示重要标签不折叠 -->
                  <el-tag
                    v-for="label in getVisibleImportantLabels(row)"
                    :key="`label-${label.key}`"
                    size="small"
                    type="success"
                    class="important-label-tag"
                    :title="`${label.key}=${label.value}`"
                  >
                    {{ formatLabelDisplay(label.key, label.value) }}
                  </el-tag>
                  
                  <!-- 其他标签折叠按钮（如果有额外标签） -->
                  <el-dropdown 
                    v-if="hasOtherLabels(row)"
                    trigger="click" 
                    placement="bottom-start"
                    @command="(cmd) => handleLabelCommand(cmd, row)"
                  >
                    <el-tag
                      size="small"
                      class="more-labels-tag"
                      type="info"
                    >
                      <span>+{{ getOtherLabelsCount(row) }}</span>
                      <el-icon class="more-icon"><ArrowDown /></el-icon>
                    </el-tag>
                    <template #dropdown>
                      <el-dropdown-menu class="labels-dropdown">
                        <div class="dropdown-header">其他节点标签</div>
                        <div class="dropdown-content">
                          <!-- 其他标签 -->
                          <div v-if="getOtherLabels(row).length > 0" class="label-group">
                            <div class="group-title">系统标签</div>
                            <el-tooltip
                              v-for="label in getOtherLabels(row)"
                              :key="`other-${label.key}`"
                              :content="`${label.key}=${label.value}`"
                              placement="top"
                              :disabled="isDropdownLabelShort(label.key, label.value)"
                            >
                              <el-tag
                                size="small"
                                class="dropdown-tag"
                              >
                                {{ label.key }}: {{ label.value }}
                              </el-tag>
                            </el-tooltip>
                          </div>
                        </div>
                        <div class="dropdown-footer">
                          <el-button 
                            type="text" 
                            size="small"
                            @click="viewNodeDetail(row)"
                          >
                            查看详情
                          </el-button>
                        </div>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </div>
              
              <!-- 污点行 -->
              <div class="node-info-row" v-if="hasTaintsToShow(row)">
                <span class="info-label">污点:</span>
                <div class="info-tags">
                  <el-tag
                    v-for="taint in (row.taints || [])"
                    :key="`taint-${taint.key}`"
                    size="small"
                    type="warning"
                    class="taint-tag"
                    :title="formatTaintFullDisplay(taint)"
                  >
                    <el-icon style="margin-right: 2px;"><Warning /></el-icon>
                    {{ formatTaintDisplay(taint) }}
                  </el-tag>
                </div>
              </div>
              
              <!-- IP地址行 -->
              <div class="node-info-row" v-if="row.internal_ip || row.external_ip">
                <span class="info-label">IP:</span>
                <div class="info-tags">
                  <span class="ip-text" v-if="row.internal_ip">{{ row.internal_ip }}</span>
                  <span class="ip-text" v-if="row.external_ip">({{ row.external_ip }})</span>
                </div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column
          prop="status"
          label="状态"
          sortable="custom"
          width="100"
        >
          <template #default="{ row }">
            <el-tag
              :type="formatNodeStatus(row.status).type"
              size="small"
            >
              {{ formatNodeStatus(row.status).text }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column
          prop="schedulable"
          label="调度状态"
          width="100"
        >
          <template #default="{ row }">
            <el-tag
              :type="getSchedulingStatus(row).type"
              size="small"
            >
              <el-icon style="margin-right: 4px;">
                <component :is="getSchedulingStatus(row).icon" />
              </el-icon>
              {{ getSchedulingStatus(row).text }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="资源显示" min-width="320">
          <template #default="{ row }">
            <div class="resource-usage-grid">
              <!-- 顶部说明文字 -->
              <div class="resource-title">
                总量 / 可分配 / 使用量
              </div>
              
              <!-- 主要资源：CPU 和 内存 -->
              <div class="resource-main">
                <div class="resource-item-card">
                  <div class="resource-item-header">
                    <el-icon class="resource-icon cpu-icon"><Monitor /></el-icon>
                    <span class="resource-label">CPU</span>
                  </div>
                  <div class="resource-item-value">
                    <span class="resource-total">{{ formatCPU(row.capacity?.cpu) || 'N/A' }}</span>
                    <span class="resource-divider">/</span>
                    <span class="resource-value">{{ formatCPU(row.allocatable?.cpu) || 'N/A' }}</span>
                    <span class="resource-divider">/</span>
                    <span class="resource-usage">{{ formatCPU(row.usage?.cpu) || 'N/A' }}</span>
                  </div>
                </div>
                
                <div class="resource-item-card">
                  <div class="resource-item-header">
                    <el-icon class="resource-icon memory-icon"><Monitor /></el-icon>
                    <span class="resource-label">内存</span>
                  </div>
                  <div class="resource-item-value">
                    <span class="resource-total">{{ formatMemoryCorrect(row.capacity?.memory) }}</span>
                    <span class="resource-divider">/</span>
                    <span class="resource-value">{{ formatMemoryCorrect(row.allocatable?.memory) }}</span>
                    <span class="resource-divider">/</span>
                    <span class="resource-usage">{{ formatMemoryCorrect(row.usage?.memory) || 'N/A' }}</span>
                  </div>
                </div>
              </div>
              
              <!-- 次要资源：Pod 和 GPU -->
              <div class="resource-secondary">
                <div class="resource-item-card">
                  <div class="resource-item-header">
                    <el-icon class="resource-icon pods-icon"><Grid /></el-icon>
                    <span class="resource-label">Pod</span>
                  </div>
                  <div class="resource-item-value">
                    <span class="resource-total">{{ row.capacity?.pods || '0' }}</span>
                    <span class="resource-divider">/</span>
                    <span class="resource-value">{{ row.allocatable?.pods || '0' }}</span>
                    <span class="resource-divider">/</span>
                    <span class="resource-usage">{{ row.usage?.pods || 'N/A' }}</span>
                  </div>
                </div>
                
                <div class="resource-item-card" v-if="hasGPUResources(row)">
                  <div class="resource-item-header">
                    <el-icon class="resource-icon gpu-icon"><VideoPlay /></el-icon>
                    <span class="resource-label">GPU</span>
                  </div>
                  <div class="resource-item-value">
                    <span class="resource-total">{{ getGPUCount(row.capacity) || '0' }}</span>
                    <span class="resource-divider">/</span>
                    <span class="resource-value">{{ getGPUCount(row.allocatable) || '0' }}</span>
                    <span class="resource-divider">/</span>
                    <span class="resource-usage">{{ row.usage?.gpu || 'N/A' }}</span>
                  </div>
                </div>
                
                <!-- 如果没有GPU，显示占位空间保持对齐 -->
                <div class="resource-item-card resource-placeholder" v-else>
                  <div class="resource-item-header">
                    <span class="resource-label-placeholder">-</span>
                  </div>
                  <div class="resource-item-value">
                    <span class="resource-placeholder-text">- / - / -</span>
                  </div>
                </div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column
          prop="version"
          label="版本"
          width="120"
        >
          <template #default="{ row }">
            <span class="version-text">{{ row.version || 'N/A' }}</span>
          </template>
        </el-table-column>

        <el-table-column
          label="禁止调度信息"
          min-width="200"
        >
          <template #default="{ row }">
            <div class="cordon-info" v-if="getCordonInfo(row)">
              <div class="cordon-reason">
                <el-icon class="reason-icon"><Edit /></el-icon>
                <span class="reason-text">{{ getCordonInfo(row).reason || '无说明' }}</span>
              </div>
              <div class="cordon-operator">
                <el-icon class="operator-icon"><User /></el-icon>
                <span class="operator-text">{{ getCordonInfo(row).operator_name || getCordonInfo(row).operatorName || '未知用户' }}</span>
                <span class="timestamp">{{ formatTimeShort(getCordonInfo(row).timestamp) }}</span>
              </div>
            </div>
            <div v-else class="no-cordon-info">
              <span class="no-info-text">-</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column
          prop="created_at"
          label="创建时间"
          sortable="custom"
          width="180"
        >
          <template #default="{ row }">
            <span class="time-text">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button type="text" size="small" @click="viewNodeDetail(row)">
                <el-icon><View /></el-icon>
                详情
              </el-button>
              
              <el-button
                v-if="row.schedulable"
                type="text"
                size="small"
                @click="cordonNode(row)"
                :title="getSchedulingStatus(row).value === 'limited' ? '节点有污点但仍可调度，禁止调度后完全不可调度' : '禁止调度节点使其不可调度'"
              >
                <el-icon><Lock /></el-icon>
                禁止调度
              </el-button>
              
              <el-button
                v-else
                type="text"
                size="small"
                @click="uncordonNode(row)"
                title="解除调度限制使节点恢复调度能力"
              >
                <el-icon><Unlock /></el-icon>
                解除调度
              </el-button>
              
              <el-dropdown 
                @command="(cmd) => handleNodeAction(cmd, row)"
              >
                <el-button 
                  type="text" 
                  size="small" 
                  class="more-actions-btn"
                  title="更多操作"
                >
                  <el-icon><MoreFilled /></el-icon>
                  <span class="btn-text">更多</span>
                  <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item 
                      v-if="authStore.role === 'admin'"
                      command="drain"
                      divided
                    >
                      <el-icon style="color: #f56c6c;"><VideoPlay /></el-icon>
                      <span style="color: #f56c6c;">驱逐节点</span>
                    </el-dropdown-item>
                    <el-dropdown-item command="labels">
                      <el-icon><CollectionTag /></el-icon>
                      管理标签
                    </el-dropdown-item>
                    <el-dropdown-item command="taints">
                      <el-icon><WarningFilled /></el-icon>
                      管理污点
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.current"
          v-model:page-size="pagination.size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 节点详情对话框 -->
    <NodeDetailDialog
      v-model="detailDialogVisible"
      :node="selectedNode"
      @refresh="refreshData"
    />


    <!-- 禁止调度确认对话框 -->
    <el-dialog
      v-model="cordonConfirmVisible"
      :title="cordonReasonForm.isBatch ? '批量禁止调度' : '禁止调度'"
      width="600px"
      destroy-on-close
    >
      <div class="cordon-confirm-content">
        <div class="confirm-message">
          <el-icon class="confirm-icon"><WarningFilled /></el-icon>
          <span>{{ cordonConfirmMessage }}</span>
        </div>
        
        <div class="reason-section">
          <div class="section-title">
            <el-icon><Edit /></el-icon>
            <span>禁止调度原因（可选）</span>
          </div>
          <el-input
            v-model="cordonReasonForm.reason"
            type="textarea"
            :rows="3"
            placeholder="请输入禁止调度的原因，如：维护、升级、故障排查等（可选）"
            maxlength="200"
            show-word-limit
            clearable
          />
          <div class="help-text">
            <el-icon><QuestionFilled /></el-icon>
            <span>添加原因说明有助于团队协作和后续管理</span>
          </div>
        </div>
        
        <div v-if="cordonReasonForm.isBatch" class="selected-nodes-info">
          <div class="section-title">
            <el-icon><Grid /></el-icon>
            <span>将要禁止调度的节点：</span>
          </div>
          <div class="nodes-list">
            <el-tag 
              v-for="node in cordonReasonForm.nodes.slice(0, 5)" 
              :key="node.name" 
              type="warning" 
              size="small"
            >
              {{ node.name }}
            </el-tag>
            <span v-if="cordonReasonForm.nodes.length > 5">... 及其他 {{ cordonReasonForm.nodes.length - 5 }} 个节点</span>
          </div>
        </div>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="cordonConfirmVisible = false">取消</el-button>
          <el-button
            type="warning"
            @click="cordonReasonForm.isBatch ? confirmBatchCordon() : confirmCordon()"
            :loading="batchLoading.cordon"
          >
            <el-icon><Lock /></el-icon>
            {{ cordonReasonForm.isBatch ? '批量禁止调度' : '禁止调度' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 驱逐确认对话框 -->
    <el-dialog
      v-model="drainConfirmVisible"
      :title="drainReasonForm.isBatch ? '批量驱逐节点' : '驱逐节点'"
      width="600px"
      destroy-on-close
    >
      <div class="drain-confirm-content">
        <el-alert
          :title="drainReasonForm.isBatch ? '批量驱逐确认' : '驱逐确认'"
          :description="drainReasonForm.isBatch ? 
            `您即将驱逐以下 ${drainReasonForm.nodes.length} 个节点上的Pod，此操作将：\n1. 禁止节点调度（Cordon）\n2. 驱逐节点上的所有Pod（忽略DaemonSet）\n3. 删除EmptyDir数据\n请确认继续操作。` :
            `您即将驱逐节点 ${drainReasonForm.node?.name || ''} 上的Pod，此操作将：\n1. 禁止节点调度（Cordon）\n2. 驱逐节点上的所有Pod（忽略DaemonSet）\n3. 删除EmptyDir数据\n请确认继续操作。`
          "
          type="warning"
          show-icon
          :closable="false"
        />
        
        <el-form :model="drainReasonForm" label-width="100px" style="margin-top: 20px;">
          <el-form-item label="驱逐原因">
            <el-input
              v-model="drainReasonForm.reason"
              placeholder="请输入驱逐原因（可选）"
              type="textarea"
              :rows="3"
              maxlength="200"
              show-word-limit
            />
          </el-form-item>
          
          <el-form-item v-if="drainReasonForm.isBatch" label="目标节点">
            <div class="nodes-list">
              <el-tag
                v-for="(node, index) in drainReasonForm.nodes.slice(0, 5)"
                :key="node.name"
                type="danger"
                size="small"
              >
                {{ node.name }}
              </el-tag>
              <span v-if="drainReasonForm.nodes.length > 5">... 及其他 {{ drainReasonForm.nodes.length - 5 }} 个节点</span>
            </div>
          </el-form-item>
        </el-form>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="drainConfirmVisible = false">取消</el-button>
          <el-button
            type="danger"
            @click="drainReasonForm.isBatch ? confirmBatchDrain() : confirmDrain()"
            :loading="batchLoading.drain"
          >
            <el-icon><VideoPlay /></el-icon>
            {{ drainReasonForm.isBatch ? '批量驱逐' : '驱逐节点' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 批量删除标签对话框 -->
    <el-dialog
      v-model="batchDeleteLabelsVisible"
      title="批量删除标签"
      width="600px"
      destroy-on-close
    >
      <div class="batch-delete-content">
        <div class="selected-nodes-info">
          <p>将从以下 <strong>{{ selectedNodes.length }}</strong> 个节点删除指定标签：</p>
          <div class="nodes-list">
            <el-tag v-for="node in selectedNodes.slice(0, 5)" :key="node.name" type="info" size="small">
              {{ node.name }}
            </el-tag>
            <span v-if="selectedNodes.length > 5">... 及其他 {{ selectedNodes.length - 5 }} 个节点</span>
          </div>
        </div>
        
        <el-form :model="batchDeleteLabelsForm" ref="batchDeleteLabelsFormRef" label-width="120px">
          <el-form-item label="要删除的标签键" required>
            <el-select
              v-model="batchDeleteLabelsForm.keys"
              multiple
              placeholder="选择要删除的标签键"
              style="width: 100%"
              clearable
              @change="onLabelKeysChange"
            >
              <el-option
                v-for="key in availableLabelKeys"
                :key="key"
                :label="key"
                :value="key"
              />
            </el-select>
            <div class="form-help-text">
              可以输入自定义标签键，用回车确认添加
            </div>
          </el-form-item>
          
          <el-form-item label="自定义标签键">
            <el-input
              v-model="customLabelKey"
              placeholder="输入标签键，按回车添加"
              @keyup.enter="addCustomLabelKey"
              clearable
            >
              <template #append>
                <el-button @click="addCustomLabelKey" type="primary">添加</el-button>
              </template>
            </el-input>
          </el-form-item>
        </el-form>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="batchDeleteLabelsVisible = false">取消</el-button>
          <el-button
            type="danger"
            @click="confirmBatchDeleteLabels"
            :loading="batchLoading.deleteLabels"
            :disabled="batchDeleteLabelsForm.keys.length === 0"
          >
            确认删除
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 批量删除污点对话框 -->
    <el-dialog
      v-model="batchDeleteTaintsVisible"
      title="批量删除污点"
      width="600px"
      destroy-on-close
    >
      <div class="batch-delete-content">
        <div class="selected-nodes-info">
          <p>将从以下 <strong>{{ selectedNodes.length }}</strong> 个节点删除指定污点：</p>
          <div class="nodes-list">
            <el-tag v-for="node in selectedNodes.slice(0, 5)" :key="node.name" type="info" size="small">
              {{ node.name }}
            </el-tag>
            <span v-if="selectedNodes.length > 5">... 及其他 {{ selectedNodes.length - 5 }} 个节点</span>
          </div>
        </div>
        
        <el-form :model="batchDeleteTaintsForm" ref="batchDeleteTaintsFormRef" label-width="120px">
          <el-form-item label="要删除的污点键" required>
            <el-select
              v-model="batchDeleteTaintsForm.keys"
              multiple
              placeholder="选择要删除的污点键"
              style="width: 100%"
              clearable
              @change="onTaintKeysChange"
            >
              <el-option
                v-for="key in availableTaintKeys"
                :key="key"
                :label="key"
                :value="key"
              />
            </el-select>
            <div class="form-help-text">
              可以输入自定义污点键，用回车确认添加
            </div>
          </el-form-item>
          
          <el-form-item label="自定义污点键">
            <el-input
              v-model="customTaintKey"
              placeholder="输入污点键，按回车添加"
              @keyup.enter="addCustomTaintKey"
              clearable
            >
              <template #append>
                <el-button @click="addCustomTaintKey" type="primary">添加</el-button>
              </template>
            </el-input>
          </el-form-item>
        </el-form>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="batchDeleteTaintsVisible = false">取消</el-button>
          <el-button
            type="danger"
            @click="confirmBatchDeleteTaints"
            :loading="batchLoading.deleteTaints"
            :disabled="batchDeleteTaintsForm.keys.length === 0"
          >
            确认删除
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 进度对话框 -->
    <ProgressDialog 
      v-model="progressDialogVisible"
      :task-id="currentTaskId"
      @completed="handleProgressCompleted"
      @error="handleProgressError"
      @cancelled="handleProgressCancelled"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onActivated, reactive, nextTick, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useNodeStore } from '@/store/modules/node'
import { useClusterStore } from '@/store/modules/cluster'
import { useAuthStore } from '@/store/modules/auth'
import { formatTime, formatNodeStatus, formatNodeRoles, formatCPU, formatMemory } from '@/utils/format'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import NodeDetailDialog from './components/NodeDetailDialog.vue'
import ProgressDialog from '@/components/common/ProgressDialog.vue'
import labelApi from '@/api/label'
import taintApi from '@/api/taint'
import nodeApi from '@/api/node'
import {
  Refresh,
  Lock,
  Unlock,
  Download,
  View,
  MoreFilled,
  CollectionTag,
  WarningFilled,
  Warning,
  Plus,
  Check,
  Monitor,
  ArrowDown,
  Search,
  Grid,
  VideoPlay,
  QuestionFilled,
  Filter,
  Edit,
  User
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const nodeStore = useNodeStore()
const clusterStore = useClusterStore()
const authStore = useAuthStore()

// 响应式数据
const loading = ref(false)
const searchKeyword = ref('')
const statusFilter = ref('')
const roleFilter = ref('')
const schedulableFilter = ref('')
const selectedNode = ref(null)
const showAdvancedSearch = ref(false)

// 标签和污点过滤器
const labelKeyFilter = ref('')
const labelValueFilter = ref('')
const taintKeyFilter = ref('')
const taintValueFilter = ref('')
const taintEffectFilter = ref('')
const nodeOwnershipFilter = ref('')
const detailDialogVisible = ref(false)

// 禁止调度确认对话框相关
const cordonConfirmVisible = ref(false)
const cordonConfirmMessage = ref('')
const cordonReasonForm = ref({
  reason: '',
  node: null,
  isBatch: false,
  nodes: []
})

// 驱逐确认对话框相关
const drainConfirmVisible = ref(false)
const drainReasonForm = ref({
  reason: '',
  node: null,
  isBatch: false,
  nodes: []
})

// 批量操作加载状态
const batchLoading = reactive({
  cordon: false,
  uncordon: false,
  drain: false,
  deleteLabels: false,
  deleteTaints: false
})

// 进度对话框相关
const progressDialogVisible = ref(false)
const currentTaskId = ref('')
const progressFallbackTimer = ref(null)

// 批量删除标签对话框相关
const batchDeleteLabelsVisible = ref(false)
const batchDeleteLabelsForm = reactive({
  keys: []
})
const batchDeleteLabelsFormRef = ref(null)
const customLabelKey = ref('')
// 系统标签前缀列表
const systemLabelPrefixes = [
  'kubernetes.io/',
  'k8s.io/',
  'node.kubernetes.io/',
  'node-role.kubernetes.io/',
  'beta.kubernetes.io/',
  'failure-domain.beta.kubernetes.io/',
  'topology.kubernetes.io/'
]

// 检查是否为系统标签
const isSystemLabel = (key) => {
  return systemLabelPrefixes.some(prefix => key.startsWith(prefix))
}

const availableLabelKeys = computed(() => {
  // 从已选择的节点中提取所有标签键，并过滤系统标签
  const keys = new Set()
  selectedNodes.value.forEach(node => {
    if (node.labels) {
      Object.keys(node.labels).forEach(key => {
        // 只添加非系统标签
        if (!isSystemLabel(key)) {
          keys.add(key)
        }
      })
    }
  })
  return Array.from(keys).sort()
})

// 批量删除污点对话框相关
const batchDeleteTaintsVisible = ref(false)
const batchDeleteTaintsForm = reactive({
  keys: []
})
const batchDeleteTaintsFormRef = ref(null)
const customTaintKey = ref('')
// 系统污点前缀列表
const systemTaintPrefixes = [
  'node.kubernetes.io/',
  'node-role.kubernetes.io/',
  'topology.kubernetes.io/'
]

// 检查是否为系统污点
const isSystemTaint = (key) => {
  return systemTaintPrefixes.some(prefix => key.startsWith(prefix))
}

const availableTaintKeys = computed(() => {
  // 从已选择的节点中提取所有污点键，并过滤系统污点
  const keys = new Set()
  selectedNodes.value.forEach(node => {
    if (node.taints && Array.isArray(node.taints)) {
      node.taints.forEach(taint => {
        // 只添加非系统污点
        if (!isSystemTaint(taint.key)) {
          keys.add(taint.key)
        }
      })
    }
  })
  return Array.from(keys).sort()
})

// 移除本地的禁止调度历史管理，改用nodeStore

// 搜索和筛选处理函数

// 计算属性
const nodes = computed(() => nodeStore.nodes)
const selectedNodes = computed(() => nodeStore.selectedNodes)
const pagination = computed(() => nodeStore.pagination)
const nodeOwnershipOptions = computed(() => nodeStore.nodeOwnershipOptions)

const filteredNodes = computed(() => {
  return nodeStore.paginatedNodes || []
})

// 防抖搜索处理
let searchDebounceTimer = null
const handleSearch = () => {
  // 清除之前的定时器
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
  }
  
  // 设置防抖延迟
  searchDebounceTimer = setTimeout(() => {
    nodeStore.setFilters({
      name: searchKeyword.value,
      status: statusFilter.value,
      role: roleFilter.value,
      schedulable: schedulableFilter.value,
      labelKey: labelKeyFilter.value,
      labelValue: labelValueFilter.value,
      taintKey: taintKeyFilter.value,
      taintValue: taintValueFilter.value,
      taintEffect: taintEffectFilter.value,
      nodeOwnership: nodeOwnershipFilter.value
    })
  }, 300) // 300ms 防抖延迟
}

// 处理筛选变化
const handleFilterChange = () => {
  handleSearch() // 统一调用搜索处理
}

// 处理选择变化
const handleSelectionChange = (selection) => {
  nodeStore.setSelectedNodes(selection)
}

// 处理排序
const handleSortChange = ({ prop, order }) => {
  // 使用前端排序，不再调用后端API
  nodeStore.setSort({ prop, order })
}

// 分页处理
const handleSizeChange = (size) => {
  nodeStore.setPagination({ size, current: 1 })
  // 前端分页，不需要重新获取数据
}

const handleCurrentChange = (current) => {
  nodeStore.setPagination({ current })
  // 前端分页，不需要重新获取数据
}

// 获取节点数据
const fetchNodes = async (params = {}) => {
  try {
    loading.value = true
    await nodeStore.fetchNodes(params)
    // nodeStore.fetchNodes() 现在会自动获取禁止调度历史，不需要手动调用
  } catch (error) {
    ElMessage.error('获取节点数据失败')
  } finally {
    loading.value = false
  }
}

// 移除fetchCordonHistories函数，现在由nodeStore自动处理

// 获取节点的禁止调度信息
const getCordonInfo = (node) => {
  // 只有当节点处于不可调度状态时才显示历史信息
  if (node.schedulable === false) {
    return nodeStore.getCordonInfo(node.name)
  }
  return null
}

// 格式化时间（完整格式）
const formatTimeShort = (timestamp) => {
  if (!timestamp) return ''
  
  // 处理 ISO 8601 格式 (如: 2025-03-12T07:02:10Z) 或 Unix 时间戳
  const date = new Date(timestamp)
  
  // 检查日期是否有效
  if (isNaN(date.getTime())) {
    console.warn('Invalid timestamp:', timestamp)
    return '无效时间'
  }
  
  const now = new Date()
  const diff = now - date
  const diffHours = Math.floor(diff / (1000 * 60 * 60))
  const diffDays = Math.floor(diff / (1000 * 60 * 60 * 24))
  
  // 如果是今天，显示时间 + "今天"
  if (diffDays === 0) {
    const timeStr = date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    return `今天 ${timeStr}`
  } 
  // 如果是昨天
  else if (diffDays === 1) {
    const timeStr = date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    return `昨天 ${timeStr}`
  }
  // 如果是最近7天内
  else if (diffDays < 7) {
    const timeStr = date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    return `${diffDays}天前 ${timeStr}`
  }
  // 超过7天，显示完整日期时间
  else {
    return date.toLocaleString('zh-CN', { 
      year: 'numeric',
      month: '2-digit', 
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }
}

// 重置搜索和过滤条件
const resetSearchFilters = () => {
  searchKeyword.value = ''
  statusFilter.value = ''
  roleFilter.value = ''
  schedulableFilter.value = ''
  labelKeyFilter.value = ''
  labelValueFilter.value = ''
  taintKeyFilter.value = ''
  taintValueFilter.value = ''
  taintEffectFilter.value = ''
  nodeOwnershipFilter.value = ''
  
  // 重置 store 中的过滤状态
  nodeStore.resetFilters()
}

// 刷新数据
const refreshData = async () => {
  console.log('🔄 [refreshData] 开始刷新数据')
  
  try {
    // 重新加载集群信息
    console.log('🔄 [refreshData] 获取集群信息')
    await clusterStore.fetchClusters()
    clusterStore.loadCurrentCluster()
    console.log('✅ [refreshData] 集群信息已更新')
    
    // 如果没有当前集群，尝试设置第一个活跃集群
    if (!clusterStore.hasCurrentCluster && clusterStore.hasCluster) {
      const firstActiveCluster = clusterStore.activeClusters[0] || clusterStore.clusters[0]
      if (firstActiveCluster) {
        clusterStore.setCurrentCluster(firstActiveCluster)
        console.log('✅ [refreshData] 设置当前集群:', firstActiveCluster.name)
      }
    }
  } catch (error) {
    console.error('❌ [refreshData] 获取集群信息失败:', error)
  }
  
  // 刷新节点数据，fetchNodes现在会自动获取禁止调度历史
  console.log('🔄 [refreshData] 开始刷新节点列表...')
  try {
    await fetchNodes()
    console.log('✅ [refreshData] 节点数据刷新完成')
    console.log('📊 [refreshData] 当前节点数量:', nodeStore.nodes.length)
  } catch (error) {
    console.error('❌ [refreshData] 节点数据刷新失败:', error)
    throw error
  }
}

// 查看节点详情
const viewNodeDetail = (node) => {
  selectedNode.value = node
  detailDialogVisible.value = true
}

// 禁止调度节点
const cordonNode = (node) => {
  cordonConfirmMessage.value = `确认要禁止调度节点 "${node.name}" 吗？`
  cordonReasonForm.value = {
    reason: '',
    node: node,
    isBatch: false,
    nodes: []
  }
  cordonConfirmVisible.value = true
}

// 确认禁止调度
const confirmCordon = async () => {
  try {
    const { reason, node } = cordonReasonForm.value
    await nodeStore.cordonNode(node.name, reason)
    ElMessage.success(`节点 ${node.name} 已禁止调度`)
    cordonConfirmVisible.value = false
    // 刷新节点数据以显示最新的调度状态
    await refreshData()
  } catch (error) {
    ElMessage.error(`禁止调度节点失败: ${error.message}`)
  }
}

// 解除调度节点
const uncordonNode = async (node) => {
  try {
    await nodeStore.uncordonNode(node.name)
    ElMessage.success(`节点 ${node.name} 已解除调度限制`)
    // 刷新节点数据以显示最新的调度状态
    await refreshData()
  } catch (error) {
    ElMessage.error(`解除调度节点失败: ${error.message}`)
  }
}

// 驱逐节点


// 处理节点操作
const handleNodeAction = (command, node) => {
  switch (command) {
    case 'drain':
      drainNode(node)
      break
    case 'labels':
      // 跳转到标签管理页面，传递节点信息
      router.push({
        path: '/labels',
        query: {
          node_name: node.name,
          cluster_name: clusterStore.currentClusterName
        }
      })
      break
    case 'taints':
      // 跳转到污点管理页面，传递节点信息
      router.push({
        path: '/taints',
        query: {
          node_name: node.name,
          cluster_name: clusterStore.currentClusterName
        }
      })
      break
  }
}

// 批量操作
const batchCordon = () => {
  if (selectedNodes.value.length === 0) return
  
  cordonConfirmMessage.value = `确认要禁止调度选中的 ${selectedNodes.value.length} 个节点吗？`
  cordonReasonForm.value = {
    reason: '',
    node: null,
    isBatch: true,
    nodes: selectedNodes.value
  }
  cordonConfirmVisible.value = true
}

// 确认批量禁止调度
const confirmBatchCordon = async () => {
  try {
    batchLoading.cordon = true
    const { reason, nodes } = cordonReasonForm.value
    const nodeNames = nodes.map(node => node.name)
    
    // 检查节点数量，如果大于5个则使用带进度的API
    if (nodeNames.length > 5) {
      // 使用带进度推送的批量操作
      const progressResponse = await nodeApi.batchCordonWithProgress(
        nodeNames, 
        clusterStore.currentClusterName, 
        reason
      )
      
      // 获取任务ID
      currentTaskId.value = progressResponse.data.data.task_id
      progressDialogVisible.value = true
      cordonConfirmVisible.value = false
      
      // 启动降级方案：如果WebSocket断开，使用轮询检测完成
      startProgressFallback('cordon')
      
      // 不立即设置loading为false，等进度完成后再处理
    } else {
      // 对于少量节点，使用原有的同步方式
      await nodeStore.batchCordon(nodeNames, reason)
      ElMessage.success(`成功禁止调度 ${nodeNames.length} 个节点`)
      clearSelection()
      cordonConfirmVisible.value = false
      // 刷新节点数据以显示最新的调度状态
      await refreshData()
      batchLoading.cordon = false
    }
  } catch (error) {
    ElMessage.error(`批量禁止调度失败: ${error.message}`)
    if (cordonReasonForm.value.nodes.length <= 5) {
      batchLoading.cordon = false
    }
  }
}

const batchUncordon = async () => {
  if (selectedNodes.value.length === 0) return
  
  try {
    batchLoading.uncordon = true
    const nodeNames = selectedNodes.value.map(node => node.name)
    
    // 检查节点数量，如果大于5个则使用带进度的API
    if (nodeNames.length > 5) {
      // 使用带进度推送的批量操作
      const progressResponse = await nodeApi.batchUncordonWithProgress(
        nodeNames, 
        clusterStore.currentClusterName
      )
      
      // 获取任务ID
      currentTaskId.value = progressResponse.data.data.task_id
      progressDialogVisible.value = true
      
      // 启动降级方案：如果WebSocket断开，使用轮询检测完成
      startProgressFallback('uncordon')
      
      // 不立即设置loading为false，等进度完成后再处理
    } else {
      // 对于少量节点，使用原有的同步方式
      await nodeStore.batchUncordon(nodeNames)
      ElMessage.success(`成功解除调度限制 ${nodeNames.length} 个节点`)
      clearSelection()
      // 刷新节点数据以显示最新的调度状态
      await refreshData()
      batchLoading.uncordon = false
    }
  } catch (error) {
    ElMessage.error(`批量解除调度失败: ${error.message}`)
    if (selectedNodes.value.length <= 5) {
      batchLoading.uncordon = false
    }
  }
}

// 驱逐节点
const drainNode = (node) => {
  drainReasonForm.value = {
    reason: '',
    node: node,
    isBatch: false,
    nodes: []
  }
  drainConfirmVisible.value = true
}

// 确认驱逐
const confirmDrain = async () => {
  try {
    batchLoading.drain = true
    const { reason, node } = drainReasonForm.value
    
    await nodeApi.drainNode(node.name, clusterStore.currentClusterName, reason)
    ElMessage.success(`节点 ${node.name} 驱逐成功`)
    drainConfirmVisible.value = false
    await refreshData()
  } catch (error) {
    ElMessage.error(`驱逐节点失败: ${error.message}`)
  } finally {
    batchLoading.drain = false
  }
}

// 批量驱逐节点
const batchDrain = () => {
  if (selectedNodes.value.length === 0) {
    ElMessage.warning('请先选择节点')
    return
  }
  
  drainReasonForm.value = {
    reason: '',
    node: null,
    isBatch: true,
    nodes: [...selectedNodes.value]
  }
  drainConfirmVisible.value = true
}

// 确认批量驱逐
const confirmBatchDrain = async () => {
  try {
    batchLoading.drain = true
    const nodeNames = drainReasonForm.value.nodes.map(node => node.name)
    const reason = drainReasonForm.value.reason
    
    // 检查节点数量，如果大于5个则使用带进度的API
    if (nodeNames.length > 5) {
      // 使用带进度推送的批量操作
      const progressResponse = await nodeApi.batchDrainWithProgress(
        nodeNames, 
        clusterStore.currentClusterName, 
        reason
      )
      
      // 获取任务ID
      currentTaskId.value = progressResponse.data.data.task_id
      progressDialogVisible.value = true
      drainConfirmVisible.value = false
      
      // 启动降级方案：如果WebSocket断开，使用轮询检测完成
      startProgressFallback('drain')
      
      // 不立即设置loading为false，等进度完成后再处理
    } else {
      // 对于少量节点，使用原有的同步方式
      await nodeApi.batchDrain(nodeNames, clusterStore.currentClusterName, reason)
      ElMessage.success(`成功驱逐 ${nodeNames.length} 个节点`)
      clearSelection()
      drainConfirmVisible.value = false
      await refreshData()
      batchLoading.drain = false
    }
  } catch (error) {
    ElMessage.error(`批量驱逐节点失败: ${error.message}`)
    if (drainReasonForm.value.nodes.length <= 5) {
      batchLoading.drain = false
    }
  }
}

// 清空选择
const clearSelection = () => {
  nodeStore.clearSelectedNodes()
}

// 批量删除标签相关方法
const showBatchDeleteLabelsDialog = () => {
  if (selectedNodes.value.length === 0) {
    ElMessage.warning('请先选择节点')
    return
  }
  
  // 重置表单
  batchDeleteLabelsForm.keys = []
  batchDeleteLabelsVisible.value = true
}

const confirmBatchDeleteLabels = async () => {
  if (batchDeleteLabelsForm.keys.length === 0) {
    ElMessage.warning('请至少输入一个标签键')
    return
  }

  if (!clusterStore.currentClusterName) {
    ElMessage.error('请先选择集群')
    return
  }

  try {
    batchLoading.deleteLabels = true
    
    const requestData = {
      nodes: selectedNodes.value.map(node => node.name),
      keys: batchDeleteLabelsForm.keys,
      cluster_name: clusterStore.currentClusterName
    }

    // 检查节点数量，如果大于5个则使用带进度的API
    if (selectedNodes.value.length > 5) {
      // 使用带进度推送的批量删除
      const progressResponse = await labelApi.batchDeleteLabelsWithProgress(requestData)
      
      // 获取任务ID
      currentTaskId.value = progressResponse.data.data.task_id
      progressDialogVisible.value = true
      batchDeleteLabelsVisible.value = false
      
      // 启动降级方案：如果WebSocket断开，使用轮询检测完成
      startProgressFallback('deleteLabels')
      
      // 不立即设置loading为false，等进度完成后再处理
    } else{
      // 对于少量节点，使用原有的同步方式
      await labelApi.batchDeleteLabels(requestData, {
        params: { cluster_name: clusterStore.currentClusterName }
      })
      
      ElMessage.success(`成功删除 ${selectedNodes.value.length} 个节点的标签`)
      batchDeleteLabelsVisible.value = false
      clearSelection()
      // 刷新节点数据以显示最新的标签
      await refreshData()
    }
  } catch (error) {
    ElMessage.error(`批量删除标签失败: ${error.message || '未知错误'}`)
  } finally {
    if (selectedNodes.value.length <= 5) {
      batchLoading.deleteLabels = false
    }
  }
}

// 批量删除污点相关方法
const showBatchDeleteTaintsDialog = () => {
  if (selectedNodes.value.length === 0) {
    ElMessage.warning('请先选择节点')
    return
  }
  
  // 重置表单
  batchDeleteTaintsForm.keys = []
  batchDeleteTaintsVisible.value = true
}

const confirmBatchDeleteTaints = async () => {
  if (batchDeleteTaintsForm.keys.length === 0) {
    ElMessage.warning('请至少输入一个污点键')
    return
  }

  if (!clusterStore.currentClusterName) {
    ElMessage.error('请先选择集群')
    return
  }

  try {
    batchLoading.deleteTaints = true
    
    const requestData = {
      nodes: selectedNodes.value.map(node => node.name),
      keys: batchDeleteTaintsForm.keys,
      cluster_name: clusterStore.currentClusterName
    }

    // 检查节点数量，如果大于5个则使用带进度的API
    if (selectedNodes.value.length > 5) {
      // 使用带进度推送的批量删除
      const progressResponse = await taintApi.batchDeleteTaintsWithProgress(requestData)
      
      // 获取任务ID
      currentTaskId.value = progressResponse.data.data.task_id
      progressDialogVisible.value = true
      batchDeleteTaintsVisible.value = false
      
      // 启动降级方案：如果WebSocket断开，使用轮询检测完成
      startProgressFallback('deleteTaints')
      
      // 不立即设置loading为false，等进度完成后再处理
    } else {
      // 对于少量节点，使用原有的同步方式
      await taintApi.batchDeleteTaints(requestData, {
        params: { cluster_name: clusterStore.currentClusterName }
      })
      
      ElMessage.success(`成功删除 ${selectedNodes.value.length} 个节点的污点`)
      batchDeleteTaintsVisible.value = false
      clearSelection()
      // 刷新节点数据以显示最新的污点
      await refreshData()
    }
  } catch (error) {
    ElMessage.error(`批量删除污点失败: ${error.message || '未知错误'}`)
  } finally {
    if (selectedNodes.value.length <= 5) {
      batchLoading.deleteTaints = false
    }
  }
}

// 标签键管理方法
const onLabelKeysChange = (keys) => {
  // 当选择的标签键发生变化时的处理
  console.log('Selected label keys:', keys)
}

const addCustomLabelKey = () => {
  const key = customLabelKey.value.trim()
  if (key && !batchDeleteLabelsForm.keys.includes(key)) {
    batchDeleteLabelsForm.keys.push(key)
    customLabelKey.value = ''
    ElMessage.success(`已添加标签键: ${key}`)
  } else if (!key) {
    ElMessage.warning('请输入标签键')
  } else {
    ElMessage.warning('该标签键已存在')
  }
}

// 启动降级方案：定时检查任务完成并刷新
const startProgressFallback = (operationType) => {
  // 清除之前的定时器
  if (progressFallbackTimer.value) {
    clearTimeout(progressFallbackTimer.value)
  }
  
  console.log(`⏰ [降级方案] 启动定时器，操作类型: ${operationType}`)
  
  // 8秒后强制刷新（即使WebSocket没有推送完成消息）
  // 缩短超时时间，确保用户在WebSocket断开时也能快速看到更新
  progressFallbackTimer.value = setTimeout(async () => {
    console.log('⚠️ [降级方案] 触发：8秒超时，强制刷新节点数据（WebSocket可能断开）')
    
    // 重置loading状态
    if (operationType === 'deleteLabels') {
      batchLoading.deleteLabels = false
    } else if (operationType === 'deleteTaints') {
      batchLoading.deleteTaints = false
    } else if (operationType === 'cordon') {
      batchLoading.cordon = false
    } else if (operationType === 'uncordon') {
      batchLoading.uncordon = false
    } else if (operationType === 'drain') {
      batchLoading.drain = false
    }
    
    // 清除选择
    clearSelection()
    
    // 刷新数据（降级方案也进行双重刷新确保数据更新）
    console.log('🔄 [降级方案] 第一次刷新节点数据')
    try {
      await refreshData()
      console.log('✅ [降级方案] 第一次刷新完成')
    } catch (err) {
      console.error('❌ [降级方案] 第一次刷新失败:', err)
    }
    
    // 延迟再刷新一次，确保数据完全更新
    setTimeout(async () => {
      console.log('🔄 [降级方案] 第二次刷新节点数据')
      try {
        await refreshData()
        console.log('✅ [降级方案] 第二次刷新完成，数据已更新')
      } catch (err) {
        console.error('❌ [降级方案] 第二次刷新失败:', err)
      }
    }, 500)
    
    // 关闭进度对话框
    if (progressDialogVisible.value) {
      progressDialogVisible.value = false
      ElMessage.success('批量操作已完成，数据已刷新')
    }
  }, 8000) // 8秒超时（从30秒缩短，提升用户体验）
}

// 进度处理函数
const handleProgressCompleted = async (data) => {
  console.log('✅ [批量操作] 完成回调被触发', data)
  
  // 清除降级方案定时器
  if (progressFallbackTimer.value) {
    clearTimeout(progressFallbackTimer.value)
    progressFallbackTimer.value = null
    console.log('✅ [批量操作] 清除降级方案定时器（WebSocket成功推送完成消息）')
  }
  
  ElMessage.success('批量操作完成')
  
  // 先重置loading状态，避免影响刷新
  batchLoading.cordon = false
  batchLoading.uncordon = false
  batchLoading.drain = false
  batchLoading.deleteLabels = false
  batchLoading.deleteTaints = false
  
  // 清除选择
  clearSelection()
  
  // 立即刷新一次（不等待），然后延迟再刷新一次确保数据完全更新
  console.log('🔄 [批量操作] 立即刷新节点数据')
  refreshData().then(() => {
    console.log('✅ [批量操作] 第一次刷新完成')
  }).catch(err => {
    console.error('❌ [批量操作] 第一次刷新失败:', err)
  })
  
  // 延迟刷新以确保后端操作完全完成和缓存清除
  console.log('⏰ [批量操作] 设置800ms后二次刷新，确保缓存清除完成')
  setTimeout(async () => {
    console.log('🔄 [批量操作] 开始二次刷新节点数据')
    try {
      await refreshData()
      console.log('✅ [批量操作] 二次刷新完成，数据已更新')
    } catch (err) {
      console.error('❌ [批量操作] 二次刷新失败:', err)
    }
  }, 800)
}

const handleProgressError = (data) => {
  console.error('批量操作失败:', data)
  
  // 清除降级方案定时器
  if (progressFallbackTimer.value) {
    clearTimeout(progressFallbackTimer.value)
    progressFallbackTimer.value = null
  }
  
  ElMessage.error('批量操作失败')
  
  // 重置loading状态
  batchLoading.cordon = false
  batchLoading.uncordon = false
  batchLoading.drain = false
  batchLoading.deleteLabels = false
  batchLoading.deleteTaints = false
}

const handleProgressCancelled = () => {
  console.log('批量操作已取消')
  
  // 清除降级方案定时器
  if (progressFallbackTimer.value) {
    clearTimeout(progressFallbackTimer.value)
    progressFallbackTimer.value = null
  }
  
  // 重置loading状态
  batchLoading.cordon = false
  batchLoading.uncordon = false
  batchLoading.drain = false
  batchLoading.deleteLabels = false
  batchLoading.deleteTaints = false
}

// 污点键管理方法
const onTaintKeysChange = (keys) => {
  // 当选择的污点键发生变化时的处理
  console.log('Selected taint keys:', keys)
}

const addCustomTaintKey = () => {
  const key = customTaintKey.value.trim()
  if (key && !batchDeleteTaintsForm.keys.includes(key)) {
    batchDeleteTaintsForm.keys.push(key)
    customTaintKey.value = ''
    ElMessage.success(`已添加污点键: ${key}`)
  } else if (!key) {
    ElMessage.warning('请输入污点键')
  } else {
    ElMessage.warning('该污点键已存在')
  }
}

// 标签折叠相关方法
const getVisibleRoles = (roles) => {
  if (!roles || roles.length === 0) return []
  // 只显示第一个角色，其余通过折叠显示
  return roles.slice(0, 1)
}

const hasMoreLabels = (node) => {
  const roleCount = (node.roles && node.roles.length > 1) ? node.roles.length - 1 : 0
  const importantLabelsCount = getImportantLabels(node).length
  return roleCount + importantLabelsCount > 0
}

const getMoreLabelsCount = (node) => {
  const roleCount = (node.roles && node.roles.length > 1) ? node.roles.length - 1 : 0
  const importantLabelsCount = getImportantLabels(node).length
  return roleCount + importantLabelsCount
}

// 获取直接显示的重要标签（不折叠）
const getVisibleImportantLabels = (node) => {
  if (!node.labels) return []
  
  const visibleKeys = ['cluster', 'deeproute.cn/user-type']
  
  return Object.entries(node.labels)
    .filter(([key]) => visibleKeys.includes(key))
    .map(([key, value]) => ({ key, value }))
}

// 获取其他标签（折叠显示）
const getOtherLabels = (node) => {
  if (!node.labels) return []
  
  const visibleKeys = ['cluster', 'deeproute.cn/user-type']
  const systemKeys = [
    'node.kubernetes.io',
    'topology.kubernetes.io', 
    'kubernetes.io',
    'node-role.kubernetes.io',
    'beta.kubernetes.io'
  ]
  
  return Object.entries(node.labels)
    .filter(([key]) => {
      // 排除已直接显示的重要标签
      if (visibleKeys.includes(key)) return false
      // 包含系统标签
      return systemKeys.some(sysKey => key.startsWith(sysKey))
    })
    .map(([key, value]) => ({ key, value }))
    .slice(0, 15) // 限制显示数量
}

// 判断是否有其他标签
const hasOtherLabels = (node) => {
  return getOtherLabels(node).length > 0
}

// 获取其他标签数量
const getOtherLabelsCount = (node) => {
  return getOtherLabels(node).length
}

// 判断下拉标签是否较短，不需要tooltip
const isDropdownLabelShort = (key, value) => {
  const fullText = `${key}=${value}`
  return fullText.length <= 40 // 40个字符以内认为是短标签
}

// 判断是否有标签需要显示
const hasLabelsToShow = (node) => {
  // 检查是否有角色标签、重要标签或其他标签
  return (node.roles && node.roles.length > 0) || 
         getVisibleImportantLabels(node).length > 0 || 
         hasOtherLabels(node)
}

// 判断是否有污点需要显示
const hasTaintsToShow = (node) => {
  return node.taints && node.taints.length > 0
}

// 格式化标签显示
const formatLabelDisplay = (key, value) => {
  if (key === 'cluster') {
    return `集群: ${value}`
  }
  if (key === 'deeproute.cn/user-type') {
    return `类型: ${value}`
  }
  return `${key}: ${value}`
}

// 格式化污点显示（完整版本）
const formatTaintDisplay = (taint) => {
  if (!taint) return ''
  
  // 显示完整信息：key=value:effect，不截断
  let display = taint.key
  if (taint.value) {
    display += `=${taint.value}`
  }
  display += `:${taint.effect}`
  
  return display
}

// 格式化污点完整显示（用于tooltip）
const formatTaintFullDisplay = (taint) => {
  if (!taint) return ''
  
  let display = taint.key
  if (taint.value) {
    display += `=${taint.value}`
  }
  display += `:${taint.effect}`
  return display
}

// 获取智能调度状态
const getSchedulingStatus = (node) => {
  // 如果节点被cordon（不可调度）
  if (!node.schedulable) {
    return {
      text: '不可调度',
      type: 'danger',
      icon: 'Lock',
      value: 'unschedulable'
    }
  }
  
  // 检查是否有影响调度的污点
  const hasSchedulingTaints = node.taints && node.taints.length > 0 && 
    node.taints.some(taint => 
      taint.effect === 'NoSchedule' || 
      taint.effect === 'PreferNoSchedule'
    )
  
  if (hasSchedulingTaints) {
    return {
      text: '有限调度',
      type: 'warning', 
      icon: 'QuestionFilled',
      value: 'limited'
    }
  }
  
  // 没有污点且可调度
  return {
    text: '可调度',
    type: 'success',
    icon: 'Check',
    value: 'schedulable'
  }
}

// 获取节点角色类型
const getNodeRoleType = (role) => {
  if (role === 'master' || role === 'control-plane' || role.includes('control-plane') || role.includes('master')) {
    return 'danger'
  }
  return 'primary'
}

// 保留旧函数用于兼容
const getImportantLabels = (node) => {
  return getOtherLabels(node)
}

const handleLabelCommand = (command, node) => {
  // 处理下拉菜单命令
  switch (command) {
    case 'view-detail':
      viewNodeDetail(node)
      break
  }
}

// 正确的内存格式化函数，处理Kubernetes内存格式
const formatMemoryCorrect = (value) => {
  if (!value) return 'N/A'
  
  // 解析Kubernetes内存格式（例如：3906252Ki, 4Gi等）
  const memStr = String(value).trim()
  
  // 匹配数字和单位
  const match = memStr.match(/^(\d+(?:\.\d+)?)(.*)?$/)
  if (!match) return value
  
  const [, numStr, unit = ''] = match
  const num = parseFloat(numStr)
  
  // 单位转换表（字节）
  const unitMap = {
    'Ki': 1024,
    'Mi': 1024 * 1024,  
    'Gi': 1024 * 1024 * 1024,
    'Ti': 1024 * 1024 * 1024 * 1024,
    'K': 1000,
    'M': 1000 * 1000,
    'G': 1000 * 1000 * 1000,
    'T': 1000 * 1000 * 1000 * 1000,
    '': 1 // 如果没有单位，假设为字节
  }
  
  const multiplier = unitMap[unit] || 1
  const bytes = num * multiplier
  
  // 转换为适合显示的单位
  if (bytes >= 1024 * 1024 * 1024 * 1024) {
    return (bytes / (1024 * 1024 * 1024 * 1024)).toFixed(1) + ' Ti'
  } else if (bytes >= 1024 * 1024 * 1024) {
    return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' Gi'
  } else if (bytes >= 1024 * 1024) {
    return (bytes / (1024 * 1024)).toFixed(1) + ' Mi'
  } else if (bytes >= 1024) {
    return (bytes / 1024).toFixed(1) + ' Ki'
  } else {
    return bytes + ' B'
  }
}

// 检查节点是否有GPU资源
const hasGPUResources = (node) => {
  return getGPUCount(node.capacity) > 0 || getGPUCount(node.allocatable) > 0
}

// 获取GPU数量
const getGPUCount = (resources) => {
  if (!resources) return 0
  
  // 检查直接的GPU字段（新的API格式）
  if (resources.gpu && typeof resources.gpu === 'object') {
    let totalGPU = 0
    for (const [key, value] of Object.entries(resources.gpu)) {
      const count = parseInt(value)
      if (!isNaN(count)) {
        totalGPU += count
      }
    }
    if (totalGPU > 0) return totalGPU
  }
  
  // 支持多种GPU资源类型（旧格式兼容）
  const gpuKeys = [
    'nvidia.com/gpu',
    'amd.com/gpu', 
    'intel.com/gpu',
    'gpu'
  ]
  
  for (const key of gpuKeys) {
    if (resources[key]) {
      const count = parseInt(resources[key])
      return isNaN(count) ? 0 : count
    }
  }
  
  return 0
}

onMounted(async () => {
  // 页面进入时重置搜索状态，避免从其他页面切换回来时保留搜索条件
  resetSearchFilters()
  
  // 先加载集群信息
  try {
    await clusterStore.fetchClusters()
    clusterStore.loadCurrentCluster()
    
    // 如果没有当前集群，尝试设置第一个活跃集群
    if (!clusterStore.hasCurrentCluster && clusterStore.hasCluster) {
      const firstActiveCluster = clusterStore.activeClusters[0] || clusterStore.clusters[0]
      if (firstActiveCluster) {
        clusterStore.setCurrentCluster(firstActiveCluster)
      }
    }
  } catch (error) {
    console.warn('Failed to load cluster info:', error)
  }
  
  // 然后获取节点数据
  fetchNodes()
})

// 监听路由变化，从标签/污点管理切换回节点管理时自动刷新
let lastRoute = null
watch(() => route.name, async (newRouteName, oldRouteName) => {
  // 当从标签管理或污点管理切换到节点管理时刷新数据
  if (newRouteName === 'NodeList' && 
      (oldRouteName === 'LabelManage' || oldRouteName === 'TaintManage')) {
    console.log(`🔄 [路由切换] ${oldRouteName} -> ${newRouteName}, 强制刷新节点数据`)
    
    // 使用双重刷新机制确保获取到最新数据
    // 第一次：立即刷新
    refreshData().then(() => {
      console.log('✅ [路由切换] 第一次刷新完成')
    }).catch(err => {
      console.error('❌ [路由切换] 第一次刷新失败:', err)
    })
    
    // 第二次：延迟刷新（确保后端缓存清除完成）
    setTimeout(async () => {
      console.log('🔄 [路由切换] 开始二次刷新（确保缓存已清除）')
      try {
        await refreshData()
        console.log('✅ [路由切换] 二次刷新完成，数据已更新')
      } catch (err) {
        console.error('❌ [路由切换] 二次刷新失败:', err)
      }
    }, 800) // 延长到800ms，确保后端缓存清除完成
  }
  lastRoute = oldRouteName
})

// 使用 onActivated 处理 keep-alive 缓存场景
onActivated(async () => {
  // 如果是从其他页面切换回来，刷新数据
  if (lastRoute === 'LabelManage' || lastRoute === 'TaintManage') {
    console.log(`🔄 [页面激活] 从 ${lastRoute} 返回，强制刷新节点数据`)
    
    // 使用双重刷新机制确保获取到最新数据
    // 第一次：立即刷新
    refreshData().then(() => {
      console.log('✅ [页面激活] 第一次刷新完成')
    }).catch(err => {
      console.error('❌ [页面激活] 第一次刷新失败:', err)
    })
    
    // 第二次：延迟刷新
    setTimeout(async () => {
      console.log('🔄 [页面激活] 开始二次刷新（确保缓存已清除）')
      try {
        await refreshData()
        console.log('✅ [页面激活] 二次刷新完成，数据已更新')
      } catch (err) {
        console.error('❌ [页面激活] 二次刷新失败:', err)
      }
    }, 800) // 延长到800ms，确保后端缓存清除完成
  }
})
</script>

<style scoped>
.node-list {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-left {
  flex: 1;
}

.page-title {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  margin: 0 0 8px 0;
}

.page-description {
  color: #666;
  margin: 0;
  font-size: 14px;
}

.header-right {
  display: flex;
  gap: 12px;
}

.search-card {
  margin-bottom: 16px;
}

.search-section {
  margin-bottom: 12px;
}

.filter-section {
  margin-bottom: 12px;
}

.advanced-search {
  margin-top: 16px;
  padding: 16px;
  background: #fafafa;
  border-radius: 6px;
  border: 1px solid #e8e8e8;
}

.advanced-search .el-divider {
  margin: 16px 0 12px 0;
}

.advanced-search .el-divider:first-child {
  margin-top: 0;
}

.advanced-search .el-row {
  margin-bottom: 12px;
}

.advanced-search .el-row:last-child {
  margin-bottom: 0;
}

.batch-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #f0f9ff;
  border: 1px solid #bae7ff;
  border-radius: 6px;
  padding: 12px 16px;
  margin-bottom: 16px;
}

.batch-info {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 14px;
  color: #0958d9;
}

.batch-buttons {
  display: flex;
  gap: 8px;
}

.table-card :deep(.el-card__body) {
  padding: 0;
}

.table-card :deep(.el-table) {
  border-radius: 8px;
}

.table-card :deep(.el-table__header) {
  background: #fafafa;
}

.table-card :deep(.el-table__header-wrapper) th {
  background: #fafafa;
  font-weight: 700;
  color: #262626;
  font-size: 14px;
  border-bottom: 1px solid #f0f0f0;
  padding: 18px 0;
  letter-spacing: 0.3px;
  height: auto;
}

/* 确保表格行有足够高度 */
.table-card :deep(.el-table__body-wrapper) tr {
  transition: all 0.2s ease;
  height: auto;
  min-height: 80px;
}

.table-card :deep(.el-table__body-wrapper) .el-table__row {
  height: auto !important;
  min-height: 80px !important;
}

.table-card :deep(.el-table__body-wrapper) tr:hover {
  background: #f8f8f8;
}

.table-card :deep(.el-table td) {
  border-bottom: 1px solid #f5f5f5;
  padding: 24px 0;
  font-size: 14px;
  line-height: 1.6;
  vertical-align: top;
  height: auto;
  min-height: 80px;
}

.node-name-cell {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 0;
  min-height: 70px;
  justify-content: flex-start;
  width: 100%;
}

.node-name-link {
  font-weight: 600;
  padding: 0;
  height: auto;
  color: #1890ff;
  font-size: 15px;
  text-align: left;
  justify-content: flex-start;
  letter-spacing: 0.3px;
}

.node-name-link:hover {
  color: #40a9ff;
  text-decoration: underline;
}

.node-info-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-top: 6px;
  line-height: 1.5;
  min-height: 24px;
}

.info-label {
  color: #666;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  padding-top: 2px;
  min-width: 32px;
}

.info-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  align-items: flex-start;
  flex: 1;
}

.role-tag {
  font-size: 11px;
  height: 22px;
  line-height: 20px;
  font-weight: 500;
  border-radius: 11px;
  padding: 0 10px;
  letter-spacing: 0.2px;
  margin: 0;
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  min-width: fit-content;
  flex-shrink: 0;
}

.more-labels-tag {
  font-size: 11px;
  height: 22px;
  line-height: 20px;
  font-weight: 500;
  border-radius: 11px;
  padding: 0 8px;
  letter-spacing: 0.2px;
  margin: 0;
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  cursor: pointer;
  background: #f0f0f0 !important;
  border: 1px solid #d9d9d9 !important;
  color: #666 !important;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.more-labels-tag:hover {
  background: #e6f7ff !important;
  border-color: #91d5ff !important;
  color: #1890ff !important;
}

.more-icon {
  font-size: 10px;
  margin-left: 2px;
  transition: transform 0.2s ease;
}

.more-labels-tag:hover .more-icon {
  transform: translateY(1px);
}

/* 重要标签样式 */
.important-label-tag {
  font-size: 11px;
  height: 22px;
  line-height: 20px;
  font-weight: 600;
  border-radius: 11px;
  padding: 0 8px;
  letter-spacing: 0.2px;
  margin: 0;
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  min-width: fit-content;
  flex-shrink: 0;
  background: #f6ffed !important;
  border: 1px solid #b7eb8f !important;
  color: #389e0d !important;
  transition: all 0.2s ease;
}

.important-label-tag:hover {
  background: #e6fffb !important;
  border-color: #87e8de !important;
  color: #13c2c2 !important;
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(19, 194, 194, 0.1);
}

.taint-tag {
  font-size: 11px;
  height: auto;
  min-height: 22px;
  line-height: 20px;
  font-weight: 600;
  border-radius: 11px;
  padding: 2px 8px;
  letter-spacing: 0.2px;
  margin: 0;
  white-space: normal;
  word-break: break-all;
  display: inline-flex;
  align-items: center;
  background: #fff7e6 !important;
  border-color: #ffd591 !important;
  color: #d46b08 !important;
  max-width: none;
  flex-wrap: wrap;
}

.taint-tag:hover {
  background: #fff1b8 !important;
  border-color: #ffc53d !important;
  color: #ad4e00 !important;
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(212, 107, 8, 0.15);
}

/* IP 文本样式 */
.ip-text {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 12px;
  color: #1890ff;
  font-weight: 500;
  margin-right: 4px;
  line-height: 1.5;
}

/* 调度状态相关样式 */
.scheduling-status-tooltip {
  max-width: 300px;
  word-wrap: break-word;
}

.limited-scheduling-info {
  font-size: 12px;
  color: #666;
  margin-top: 4px;
  padding: 4px 8px;
  background: #fff7e6;
  border-radius: 4px;
  border-left: 3px solid #faad14;
}

.taint-tag .el-icon {
  font-size: 10px;
}

/* 下拉菜单样式 */
.labels-dropdown {
  min-width: 320px;
  max-width: 500px;  /* 增加最大宽度以容纳更长的标签 */
}

.dropdown-header {
  padding: 8px 12px;
  font-size: 12px;
  font-weight: 600;
  color: #666;
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.dropdown-content {
  padding: 12px;
  max-height: 300px;
  overflow-y: auto;
}

.label-group {
  margin-bottom: 12px;
}

.label-group:last-child {
  margin-bottom: 0;
}

.group-title {
  font-size: 11px;
  font-weight: 600;
  color: #999;
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.dropdown-tag {
  margin: 0 4px 4px 0;
  font-size: 11px;
  min-height: 22px;
  line-height: 20px;
  padding: 0 8px;
  border-radius: 11px;
  font-weight: 500;
  max-width: 280px;
  word-break: break-all;
  white-space: normal;
  display: inline-block;
  text-align: left;
}

.dropdown-footer {
  padding: 8px 12px;
  border-top: 1px solid #f0f0f0;
  background: #fafafa;
  text-align: center;
}

.dropdown-footer .el-button {
  font-size: 12px;
  padding: 4px 12px;
  height: 24px;
}

/* 新的卡片式布局样式 */
.resource-usage-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  background: #fafafa;
  border-radius: 8px;
  border-left: 3px solid #e8e8e8;
  transition: all 0.2s ease;
  min-width: 320px;
}

.resource-usage-grid:hover {
  background: #f0f9ff;
  border-left-color: #1890ff;
}

/* 顶部标题 */
.resource-title {
  text-align: center;
  font-size: 12px;
  color: #666;
  font-weight: 600;
  padding: 4px 0;
  margin-bottom: 4px;
  border-bottom: 1px solid #e8e8e8;
  letter-spacing: 0.5px;
}

/* 主要资源区域（CPU、内存） */
.resource-main {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 8px;
}

/* 次要资源区域（Pod、GPU） */
.resource-secondary {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

/* 资源项卡片 */
.resource-item-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
  background: #fff;
  border-radius: 6px;
  border: 1px solid #e8e8e8;
  transition: all 0.2s ease;
  min-height: 60px;
  justify-content: center;
}

.resource-item-card:hover {
  border-color: #91d5ff;
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.08);
}

/* 占位符卡片（无GPU时） */
.resource-placeholder {
  opacity: 0.3;
  border-style: dashed;
  background: transparent;
}

.resource-placeholder:hover {
  border-color: #e8e8e8;
  box-shadow: none;
}

/* 资源项标题 */
.resource-item-header {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  margin-bottom: 2px;
}

/* 资源项数值 */
.resource-item-value {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-weight: 600;
}

/* 占位符样式 */
.resource-label-placeholder {
  font-size: 12px;
  color: #ccc;
}

.resource-placeholder-text {
  font-size: 14px;
  color: #ccc;
}

.resource-icon {
  font-size: 14px;
  padding: 3px;
  border-radius: 3px;
}

.cpu-icon {
  color: #52c41a;
  background: rgba(82, 196, 26, 0.15);
}

.memory-icon {
  color: #1890ff;
  background: rgba(24, 144, 255, 0.15);
}

.pods-icon {
  color: #722ed1;
  background: rgba(114, 46, 209, 0.15);
}

.gpu-icon {
  color: #f5222d;
  background: rgba(245, 34, 45, 0.15);
}

.resource-label {
  color: #333;
  font-weight: 600;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

/* 保留原有样式用于兼容，但主要使用新的 resource-value-item */

.resource-value {
  color: #52c41a;
  font-family: 'SF Pro Display', -apple-system, BlinkMacSystemFont, 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.1px;
  text-shadow: 0 1px 2px rgba(82, 196, 26, 0.1);
  white-space: nowrap;
}

.resource-divider {
  color: #999;
  font-weight: 500;
  margin: 0 3px;
  font-size: 13px;
}

.resource-total {
  color: #1890ff;
  font-family: 'SF Pro Display', -apple-system, BlinkMacSystemFont, 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.1px;
  text-shadow: 0 1px 2px rgba(24, 144, 255, 0.1);
  white-space: nowrap;
}

.version-text {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 11px;
  color: #666;
  background: #f0f0f0;
  padding: 2px 6px;
  border-radius: 3px;
  display: inline-block;
  font-weight: 500;
}

.time-text {
  font-size: 12px;
  color: #999;
  font-family: 'SF Pro Text', -apple-system, BlinkMacSystemFont, sans-serif;
}

.action-buttons {
  display: flex;
  gap: 6px;
  align-items: center;
}

.action-buttons .el-button {
  padding: 6px 12px;
  font-size: 13px;
  border-radius: 6px;
  border: 1px solid transparent;
  font-weight: 500;
  letter-spacing: 0.2px;
}

.action-buttons .el-button--text {
  color: #666;
  background: #f5f5f5;
  border-color: #e8e8e8;
  transition: all 0.2s ease;
}

.action-buttons .el-button--text:hover {
  color: #1890ff;
  background: #e6f7ff;
  border-color: #91d5ff;
}

.more-actions-btn {
  min-width: 60px !important;
  position: relative;
  padding: 6px 8px !important;
  display: flex !important;
  align-items: center;
  justify-content: center;
  gap: 3px;
}

.more-actions-btn .btn-text {
  font-size: 12px;
  line-height: 1;
}

.more-actions-btn .el-icon--right {
  margin-left: 2px;
  font-size: 10px;
  opacity: 0.7;
}

.more-actions-btn:hover {
  background: #f0f0f0 !important;
  border-color: #d9d9d9 !important;
  color: #1890ff !important;
}

.more-actions-btn:hover .btn-text {
  color: #1890ff;
}


/* 下拉菜单项样式优化 */
.action-buttons .el-dropdown-menu,
.node-action-dropdown {
  min-width: 130px;
  padding: 6px 0;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  border: 1px solid #e8e8e8;
}

.action-buttons .el-dropdown-menu .el-dropdown-menu__item,
.node-action-dropdown .el-dropdown-menu__item {
  padding: 10px 14px;
  font-size: 13px;
  line-height: 1.4;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.2s ease;
  margin: 0 4px;
  border-radius: 4px;
}

.action-buttons .el-dropdown-menu .el-dropdown-menu__item:hover,
.node-action-dropdown .el-dropdown-menu__item:hover {
  background: #f0f9ff;
  color: #1890ff;
  transform: translateX(2px);
}

.action-buttons .el-dropdown-menu .el-dropdown-menu__item .el-icon,
.node-action-dropdown .el-dropdown-menu__item .el-icon {
  font-size: 14px;
  color: inherit;
}

.pagination-container {
  padding: 16px;
  display: flex;
  justify-content: flex-end;
  border-top: 1px solid #f0f0f0;
}

/* 禁止调度信息列样式 */
.cordon-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px;
  background: #fff7e6;
  border-radius: 6px;
  border-left: 3px solid #faad14;
  font-size: 12px;
  line-height: 1.4;
}

.cordon-reason {
  display: flex;
  align-items: flex-start;
  gap: 6px;
}

.reason-icon {
  color: #faad14;
  font-size: 14px;
  flex-shrink: 0;
  margin-top: 1px;
}

.reason-text {
  color: #d46b08;
  font-weight: 600;
  word-break: break-word;
  flex: 1;
  max-width: 160px;
}

.cordon-operator {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #666;
  font-size: 11px;
}

.operator-icon {
  color: #1890ff;
  font-size: 12px;
}

.operator-text {
  font-weight: 500;
  color: #333;
}

.timestamp {
  margin-left: auto;
  color: #999;
  font-style: italic;
}

.no-cordon-info {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px;
  color: #999;
  font-size: 14px;
}

.no-info-text {
  opacity: 0.6;
}

/* 禁止调度确认对话框样式 */
.cordon-confirm-content {
  padding: 8px 0;
}

.confirm-message {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: #fff7e6;
  border-radius: 8px;
  border-left: 4px solid #faad14;
  margin-bottom: 20px;
  font-size: 14px;
  color: #666;
}

.confirm-icon {
  color: #faad14;
  font-size: 18px;
  flex-shrink: 0;
}

.reason-section {
  margin-bottom: 20px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-weight: 600;
  color: #333;
  font-size: 14px;
}

.section-title .el-icon {
  color: #1890ff;
  font-size: 16px;
}

.help-text {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  font-size: 12px;
  color: #999;
}

.help-text .el-icon {
  font-size: 14px;
  color: #faad14;
}

.selected-nodes-info .nodes-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
  padding: 12px;
  background: #fafafa;
  border-radius: 6px;
  border: 1px solid #e8e8e8;
  align-items: center;
}

.selected-nodes-info .nodes-list .el-tag {
  font-size: 12px;
  height: 24px;
  line-height: 22px;
  padding: 0 10px;
  border-radius: 12px;
  font-weight: 500;
  white-space: nowrap;
  background: #fff7e6 !important;
  border: 1px solid #ffd591 !important;
  color: #d46b08 !important;
}

.selected-nodes-info .nodes-list span {
  font-size: 12px;
  color: #666;
  font-style: italic;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }
  
  .batch-actions {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
  
  .batch-info {
    justify-content: center;
  }
  
  .batch-buttons {
    justify-content: center;
  }
  
  .table-card :deep(.el-table) {
    font-size: 12px;
  }
  
  .table-card :deep(.el-table td) {
    padding: 20px 0;
    min-height: 70px;
  }
  
  .node-name-cell {
    min-height: 60px;
    gap: 8px;
  }
  
  .node-info-row {
    margin-top: 4px;
    min-height: 20px;
  }
  
  .info-tags {
    gap: 6px;
  }
  
  .role-tag {
    font-size: 10px;
    height: 20px;
    line-height: 18px;
    padding: 0 8px;
  }
  
  .more-labels-tag {
    font-size: 10px;
    height: 20px;
    line-height: 18px;
    padding: 0 6px;
  }
  
  .labels-dropdown {
    min-width: 280px;
    max-width: 450px;  /* 在小屏幕上也增加最大宽度 */
  }
  
  .dropdown-content {
    max-height: 200px;
  }
  
  .action-buttons {
    flex-direction: column;
    gap: 2px;
  }
  
  .more-actions-btn {
    min-width: 50px !important;
    padding: 4px 6px !important;
  }
  
  .more-actions-btn .btn-text {
    font-size: 11px;
  }
  
  .more-actions-btn .el-icon--right {
    font-size: 9px;
  }
  
  /* 移动端资源配置优化 */
  .resource-usage-grid {
    padding: 10px;
    gap: 8px;
    min-width: 300px;
  }
  
  .resource-title {
    font-size: 11px;
    padding: 3px 0;
  }
  
  .resource-main,
  .resource-secondary {
    gap: 8px;
    margin-bottom: 6px;
  }
  
  .resource-item-card {
    padding: 6px;
    min-height: 50px;
    gap: 4px;
  }
  
  .resource-icon {
    font-size: 12px;
    padding: 2px;
  }
  
  .resource-label {
    font-size: 10px;
    letter-spacing: 0.3px;
  }
  
  .resource-value,
  .resource-total {
    font-size: 12px;
    letter-spacing: 0;
  }
  
  .resource-divider {
    font-size: 12px;
    margin: 0 2px;
  }
  
  .resource-label-placeholder {
    font-size: 10px;
  }
  
  .resource-placeholder-text {
    font-size: 12px;
  }
  
  .resource-usage {
    color: #ff7875;
    font-weight: 600;
    font-family: 'Monaco', 'Consolas', monospace;
    font-size: 12px;
  }
}
</style>