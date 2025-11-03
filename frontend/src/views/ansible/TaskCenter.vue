<template>
  <div class="ansible-task-center">
    <el-card class="header-card">
      <template #header>
        <div class="card-header">
          <span>Ansible 任务中心</span>
          <el-button type="primary" @click="showCreateDialog">
            <el-icon><Plus /></el-icon>
            启动任务
          </el-button>
        </div>
      </template>
      
      <!-- 统计卡片 -->
      <el-row :gutter="20" class="stats-row">
        <el-col :xs="24" :sm="12" :md="6" :lg="6">
          <div class="stat-card stat-card-primary">
            <div class="stat-content">
              <div class="stat-icon">
                <el-icon><DocumentCopy /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-value">{{ statistics.total_tasks || 0 }}</div>
                <div class="stat-label">总任务数</div>
              </div>
            </div>
          </div>
        </el-col>
        <el-col :xs="24" :sm="12" :md="6" :lg="6">
          <div class="stat-card stat-card-warning">
            <div class="stat-content">
              <div class="stat-icon">
                <el-icon><Loading /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-value">{{ statistics.running_tasks || 0 }}</div>
                <div class="stat-label">运行中</div>
              </div>
            </div>
          </div>
        </el-col>
        <el-col :xs="24" :sm="12" :md="6" :lg="6">
          <div class="stat-card stat-card-success">
            <div class="stat-content">
              <div class="stat-icon">
                <el-icon><CircleCheck /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-value">{{ statistics.status_counts?.success || 0 }}</div>
                <div class="stat-label">成功</div>
              </div>
            </div>
          </div>
        </el-col>
        <el-col :xs="24" :sm="12" :md="6" :lg="6">
          <div class="stat-card stat-card-danger">
            <div class="stat-content">
              <div class="stat-icon">
                <el-icon><CircleClose /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-value">{{ statistics.status_counts?.failed || 0 }}</div>
                <div class="stat-label">失败</div>
              </div>
            </div>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <!-- 最近使用任务 -->
    <el-card style="margin-top: 20px" v-if="recentTasks.length > 0">
      <template #header>
        <div class="card-header">
          <span>
            <el-icon style="vertical-align: middle; margin-right: 4px"><Clock /></el-icon>
            最近使用
          </span>
          <el-button text @click="loadRecentTasks">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </template>
      
      <el-row :gutter="12">
        <el-col 
          v-for="history in recentTasks" 
          :key="history.id" 
          :xs="24" :sm="12" :md="8" :lg="6"
          style="margin-bottom: 12px"
        >
          <el-card shadow="hover" class="recent-task-card">
            <div class="recent-task-header">
              <el-text truncated>{{ history.task_name }}</el-text>
              <el-dropdown trigger="click" @command="(cmd) => handleRecentTaskAction(cmd, history)">
                <el-icon class="recent-task-more"><MoreFilled /></el-icon>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="rerun">
                      <el-icon><RefreshRight /></el-icon>
                      重新执行
                    </el-dropdown-item>
                    <el-dropdown-item command="delete" divided>
                      <el-icon><Delete /></el-icon>
                      删除记录
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
            
            <div class="recent-task-info">
              <div class="info-item">
                <el-icon><Document /></el-icon>
                <span>{{ history.template?.name || '自定义 Playbook' }}</span>
              </div>
              <div class="info-item">
                <el-icon><List /></el-icon>
                <span>{{ history.inventory?.name || '-' }}</span>
              </div>
              <div class="info-item">
                <el-icon><Calendar /></el-icon>
                <span>{{ formatRecentTime(history.last_used_at) }}</span>
              </div>
              <div class="info-item">
                <el-icon><DataLine /></el-icon>
                <span>使用 {{ history.use_count }} 次</span>
              </div>
            </div>
            
            <div class="recent-task-tags">
              <el-tag v-if="history.dry_run" size="small" type="success" effect="dark">
                <el-icon style="margin-right: 4px"><View /></el-icon>
                检查模式
              </el-tag>
              <el-tag v-else size="small" type="primary" effect="plain">
                <el-icon style="margin-right: 4px"><Setting /></el-icon>
                正常模式
              </el-tag>
              <el-tag v-if="history.batch_config?.enabled" size="small" type="warning">
                分批执行
              </el-tag>
            </div>
            
            <el-button 
              type="primary" 
              size="small" 
              style="width: 100%; margin-top: 8px"
              @click="rerunTask(history)"
            >
              <el-icon><RefreshRight /></el-icon>
              快速执行
            </el-button>
          </el-card>
        </el-col>
      </el-row>
    </el-card>

    <!-- 筛选器 -->
    <el-card style="margin-top: 20px">
      <el-form :inline="true" :model="queryParams">
        <el-form-item label="状态">
          <el-select v-model="queryParams.status" placeholder="全部" clearable style="width: 120px">
            <el-option label="待执行" value="pending" />
            <el-option label="运行中" value="running" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键字">
          <el-input v-model="queryParams.keyword" placeholder="搜索任务名称" clearable style="width: 200px" @keyup.enter="handleQuery" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleQuery">查询</el-button>
          <el-button @click="handleReset">重置</el-button>
          <el-button @click="handleRefresh" :loading="loading">
            <el-icon><Refresh /></el-icon>
            刷新状态
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 任务列表 -->
    <el-card style="margin-top: 20px">
      <div style="margin-bottom: 16px">
        <el-button 
          type="danger" 
          :disabled="selectedTasks.length === 0"
          @click="handleBatchDelete"
        >
          批量删除 ({{ selectedTasks.length }})
        </el-button>
        <el-text type="info" size="small" style="margin-left: 16px">
          提示：只能删除已完成、失败或取消的任务
        </el-text>
      </div>
      <el-table 
        :data="tasks" 
        v-loading="loading" 
        style="width: 100%"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" :selectable="canSelectTask" />
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="任务名称" min-width="220">
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 8px">
              <el-icon v-if="row.dry_run" color="#67C23A" :size="18">
                <View />
              </el-icon>
              <el-icon v-else color="#409EFF" :size="18">
                <Setting />
              </el-icon>
              <span :style="{ color: row.dry_run ? '#67C23A' : '' }">{{ row.name }}</span>
              <el-tag v-if="row.dry_run" type="success" size="small" effect="dark">
                <el-icon style="margin-right: 4px"><View /></el-icon>
                检查
              </el-tag>
              <el-tag v-else type="primary" size="small" effect="plain">
                <el-icon style="margin-right: 4px"><Setting /></el-icon>
                正常
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="创建用户" width="120">
          <template #default="{ row }">
            {{ row.user?.username || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="150">
          <template #default="{ row }">
            <div>
              <el-tag :type="getStatusType(row.status)">
                {{ getStatusText(row.status) }}
              </el-tag>
              <el-tag v-if="row.is_timed_out" type="warning" size="small" style="margin-left: 4px">
                超时
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="优先级" width="100">
          <template #default="{ row }">
            <el-tag 
              :type="row.priority === 'high' ? 'danger' : row.priority === 'medium' ? 'warning' : 'info'"
              size="small"
            >
              <el-icon v-if="row.priority === 'high'"><Top /></el-icon>
              <el-icon v-else-if="row.priority === 'low'"><Bottom /></el-icon>
              <el-icon v-else><Minus /></el-icon>
              {{ row.priority === 'high' ? '高' : row.priority === 'medium' ? '中' : '低' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="200">
          <template #default="{ row }">
            <div v-if="row.status === 'running'">
              <el-progress 
                :percentage="calculateProgress(row)" 
                :status="row.hosts_failed > 0 ? 'exception' : 'success'" 
              />
              <!-- 显示批次信息 -->
              <div v-if="row.batch_config && row.batch_config.enabled" style="font-size: 12px; color: #909399; margin-top: 4px">
                批次: {{ row.current_batch }}/{{ row.total_batches }}
                <el-tag v-if="row.batch_status === 'paused'" type="warning" size="small" style="margin-left: 4px">
                  已暂停
                </el-tag>
              </div>
            </div>
            <div v-else-if="row.status === 'success' || row.status === 'failed'">
              {{ row.hosts_ok }}/{{ row.hosts_total }} 成功
              <div v-if="row.batch_config && row.batch_config.enabled" style="font-size: 12px; color: #909399">
                (分{{ row.total_batches }}批执行)
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="duration" label="耗时" width="150">
          <template #default="{ row }">
            <div>
              <span>{{ row.duration ? `${row.duration}秒` : '-' }}</span>
              <div v-if="row.timeout_seconds > 0" style="font-size: 12px; color: #909399">
                限制: {{ formatDuration(row.timeout_seconds) }}
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="480" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleViewLogs(row)">查看日志</el-button>
            
            <!-- 前置检查按钮（pending 或 failed 状态可执行） -->
            <el-button 
              size="small" 
              type="info" 
              @click="handlePreflightCheck(row)" 
              v-if="row.status === 'pending' || row.status === 'failed'"
            >
              <el-icon><Checked /></el-icon>
              执行检查
            </el-button>
            
            <!-- 批次控制按钮 -->
            <template v-if="row.status === 'running' && row.batch_config && row.batch_config.enabled">
              <el-button 
                size="small" 
                type="warning" 
                @click="handlePauseBatch(row)" 
                v-if="row.batch_status === 'running'"
              >
                暂停批次
              </el-button>
              <el-button 
                size="small" 
                type="success" 
                @click="handleContinueBatch(row)" 
                v-if="row.batch_status === 'paused'"
              >
                继续批次
              </el-button>
              <el-button 
                size="small" 
                type="danger" 
                @click="handleStopBatch(row)"
              >
                停止批次
              </el-button>
            </template>
            
            <!-- 普通取消按钮 -->
            <el-button 
              size="small" 
              type="warning" 
              @click="handleCancel(row)" 
              v-if="row.status === 'running' && (!row.batch_config || !row.batch_config.enabled)"
            >
              取消
            </el-button>
            
            <el-button 
              size="small" 
              type="primary" 
              @click="handleRetry(row)" 
              v-if="row.status === 'failed'"
            >
              重试
            </el-button>
            <el-button 
              size="small" 
              type="danger" 
              @click="handleDelete(row)" 
              v-if="row.status !== 'running' && row.status !== 'pending'"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="queryParams.page"
        v-model:page-size="queryParams.page_size"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleQuery"
        @current-change="handleQuery"
        style="margin-top: 20px"
      />
    </el-card>

    <!-- 创建任务对话框 -->
    <el-dialog v-model="createDialogVisible" title="启动 Ansible 任务" width="800px">
      <el-form :model="taskForm" label-width="120px">
        <el-form-item label="任务名称" required>
          <el-input v-model="taskForm.name" placeholder="请输入任务名称" />
        </el-form-item>
        <el-form-item label="选择模板">
          <el-select 
            v-model="taskForm.template_id" 
            placeholder="选择模板（可选）" 
            clearable 
            style="width: 100%"
            @change="loadEstimation"
          >
            <el-option 
              v-for="template in templates" 
              :key="template.id" 
              :label="template.name" 
              :value="template.id" 
            />
          </el-select>
        </el-form-item>
        <el-form-item label="主机清单" required>
          <el-select 
            v-model="taskForm.inventory_id" 
            placeholder="选择主机清单" 
            style="width: 100%"
            @change="loadEstimation"
          >
            <el-option 
              v-for="inventory in inventories" 
              :key="inventory.id" 
              :label="inventory.name" 
              :value="inventory.id" 
            />
          </el-select>
        </el-form-item>
        
        <!-- 任务执行预估 -->
        <el-form-item label="预估执行时间" v-if="estimation">
          <el-alert 
            :title="estimation.estimated_range"
            :type="estimation.confidence === 'high' ? 'success' : estimation.confidence === 'medium' ? 'warning' : 'info'"
            :closable="false"
          >
            <template #default>
              <div style="margin-top: 8px; font-size: 13px">
                <p><strong>平均时长:</strong> {{ formatEstimationDuration(estimation.avg_duration) }}</p>
                <p><strong>历史范围:</strong> {{ formatEstimationDuration(estimation.min_duration) }} - {{ formatEstimationDuration(estimation.max_duration) }}</p>
                <p><strong>成功率:</strong> {{ estimation.success_rate.toFixed(1) }}%</p>
                <p><strong>样本数量:</strong> {{ estimation.sample_size }} 次</p>
                <p><strong>置信度:</strong> {{ getConfidenceText(estimation.confidence) }}</p>
              </div>
            </template>
          </el-alert>
        </el-form-item>
        <el-form-item label="集群">
          <el-select v-model="taskForm.cluster_id" placeholder="选择集群（可选）" clearable style="width: 100%">
            <el-option 
              v-for="cluster in clusters" 
              :key="cluster.id" 
              :label="cluster.name" 
              :value="cluster.id" 
            />
          </el-select>
        </el-form-item>
        
        <el-form-item label="任务优先级">
          <el-select v-model="taskForm.priority" placeholder="选择任务优先级" style="width: 100%">
            <el-option label="高优先级" value="high">
              <span style="float: left">
                <el-icon color="#F56C6C"><Top /></el-icon>
                高优先级
              </span>
              <span style="float: right; color: #8492a6; font-size: 13px">优先执行</span>
            </el-option>
            <el-option label="中优先级" value="medium">
              <span style="float: left">
                <el-icon color="#E6A23C"><Minus /></el-icon>
                中优先级
              </span>
              <span style="float: right; color: #8492a6; font-size: 13px">默认</span>
            </el-option>
            <el-option label="低优先级" value="low">
              <span style="float: left">
                <el-icon color="#909399"><Bottom /></el-icon>
                低优先级
              </span>
              <span style="float: right; color: #8492a6; font-size: 13px">空闲时执行</span>
            </el-option>
          </el-select>
          <div style="margin-top: 8px; color: #909399; font-size: 12px">
            <el-icon><InfoFilled /></el-icon>
            高优先级任务会优先执行，低优先级任务会在系统空闲时执行
          </div>
        </el-form-item>
        
        <!-- 必需变量输入 -->
        <template v-if="selectedTemplate && selectedTemplate.required_vars && selectedTemplate.required_vars.length > 0">
          <el-divider content-position="left">
            <el-icon><Setting /></el-icon>
            模板变量配置
          </el-divider>
          <el-alert
            title="请提供以下必需变量"
            type="info"
            :closable="false"
            style="margin-bottom: 16px"
          >
            <template #default>
              该模板需要以下 {{ selectedTemplate.required_vars.length }} 个变量
            </template>
          </el-alert>
          
          <el-form-item 
            v-for="varName in selectedTemplate.required_vars" 
            :key="varName"
            :label="varName"
            :required="true"
          >
            <el-input 
              v-model="taskForm.extra_vars[varName]" 
              :placeholder="`请输入 ${varName} 的值`"
            >
              <template #prepend>
                <el-icon><Key /></el-icon>
              </template>
            </el-input>
            <div style="margin-top: 4px; color: #909399; font-size: 12px">
              变量名: {{ varName }}
            </div>
          </el-form-item>
          
          <el-divider />
        </template>
        
        <el-form-item label="执行模式">
          <div style="width: 100%">
            <el-radio-group v-model="taskForm.dry_run" style="width: 100%; display: flex; gap: 12px">
              <el-radio :label="false" border style="flex: 1; height: auto; padding: 12px">
                <div style="display: flex; align-items: flex-start; gap: 10px; width: 100%">
                  <el-icon color="#409EFF" :size="20"><Setting /></el-icon>
                  <div style="flex: 1">
                    <div style="font-weight: bold; margin-bottom: 4px">正常模式</div>
                    <div style="font-size: 12px; color: #909399; line-height: 1.4">实际执行并应用变更</div>
                  </div>
                </div>
              </el-radio>
              <el-radio :label="true" border style="flex: 1; height: auto; padding: 12px">
                <div style="display: flex; align-items: flex-start; gap: 10px; width: 100%">
                  <el-icon color="#67C23A" :size="20"><View /></el-icon>
                  <div style="flex: 1">
                    <div style="font-weight: bold; color: #67C23A; margin-bottom: 4px">检查模式 (Dry Run)</div>
                    <div style="font-size: 12px; color: #909399; line-height: 1.4">仅模拟执行，不实际变更</div>
                  </div>
                </div>
              </el-radio>
            </el-radio-group>
            <el-alert 
              v-if="taskForm.dry_run" 
              type="success" 
              :closable="false"
              style="margin-top: 12px"
            >
              <template #default>
                <div style="display: flex; align-items: center; gap: 8px">
                  <el-icon><InfoFilled /></el-icon>
                  <span>检查模式会模拟任务执行过程，显示将要进行的变更，但不会实际修改目标主机。适合用于验证 Playbook 和测试执行流程。</span>
                </div>
              </template>
            </el-alert>
          </div>
        </el-form-item>
        
        <!-- 分批执行配置 -->
        <el-form-item label="分批执行">
          <el-switch 
            v-model="batchEnabled" 
            active-text="启用（金丝雀/灰度发布）"
            inactive-text="禁用"
            @change="handleBatchToggle"
          />
        </el-form-item>
        
        <!-- 分批执行详细配置 -->
        <template v-if="batchEnabled">
          <el-form-item label="批次策略" style="margin-left: 20px">
            <el-radio-group v-model="batchStrategy">
              <el-radio label="size">固定数量</el-radio>
              <el-radio label="percent">百分比</el-radio>
            </el-radio-group>
          </el-form-item>
          
          <el-form-item 
            :label="batchStrategy === 'size' ? '每批主机数' : '每批百分比'" 
            style="margin-left: 20px"
          >
            <el-input-number 
              v-model="batchValue" 
              :min="1" 
              :max="batchStrategy === 'size' ? 100 : 100"
              :step="1"
              style="width: 180px"
            />
            <span style="margin-left: 8px; color: #909399">
              {{ batchStrategy === 'size' ? '台' : '%' }}
            </span>
          </el-form-item>
          
          <el-form-item label="执行控制" style="margin-left: 20px">
            <el-checkbox v-model="taskForm.batch_config.pause_after_batch">
              每批执行后暂停，等待手动确认
            </el-checkbox>
          </el-form-item>
          
          <el-form-item label="失败阈值" style="margin-left: 20px">
            <el-input-number 
              v-model="taskForm.batch_config.failure_threshold" 
              :min="0" 
              :max="100"
              style="width: 180px"
            />
            <span style="margin-left: 8px; color: #909399">
              失败主机数超过此值则停止执行
            </span>
          </el-form-item>
          
          <el-form-item label="单批失败率" style="margin-left: 20px">
            <el-input-number 
              v-model="taskForm.batch_config.max_batch_fail_rate" 
              :min="0" 
              :max="100"
              :step="5"
              style="width: 180px"
            />
            <span style="margin-left: 8px; color: #909399">
              % （单批失败率超过此值则停止）
            </span>
          </el-form-item>
          
          <div style="margin-left: 20px; padding: 12px; background: #f0f9ff; border-left: 3px solid #409eff; color: #606266; font-size: 13px">
            <el-icon><InfoFilled /></el-icon>
            分批执行适用于大规模变更，可以先在少量主机上验证，再逐步推广到所有主机，降低风险
          </div>
        </template>
        
        <!-- 超时配置 -->
        <el-divider />
        <el-form-item label="执行超时">
          <el-switch 
            v-model="timeoutEnabled" 
            active-text="启用超时控制"
            inactive-text="不限制"
            @change="handleTimeoutToggle"
          />
        </el-form-item>
        
        <el-form-item v-if="timeoutEnabled" label="超时时间" style="margin-left: 20px">
          <el-input-number 
            v-model="taskForm.timeout_seconds" 
            :min="60" 
            :max="86400"
            :step="60"
            style="width: 180px"
          />
          <span style="margin-left: 8px; color: #909399">
            秒（{{ formatDuration(taskForm.timeout_seconds) }}）
          </span>
          <div style="margin-top: 8px; color: #909399; font-size: 12px">
            <el-icon><InfoFilled /></el-icon>
            任务执行超过此时间将被自动取消，防止任务无限期运行
          </div>
        </el-form-item>
        
        <!-- 常用超时选项 -->
        <el-form-item v-if="timeoutEnabled" label="快速设置" style="margin-left: 20px">
          <el-button-group>
            <el-button size="small" @click="taskForm.timeout_seconds = 300">5分钟</el-button>
            <el-button size="small" @click="taskForm.timeout_seconds = 600">10分钟</el-button>
            <el-button size="small" @click="taskForm.timeout_seconds = 1800">30分钟</el-button>
            <el-button size="small" @click="taskForm.timeout_seconds = 3600">1小时</el-button>
            <el-button size="small" @click="taskForm.timeout_seconds = 7200">2小时</el-button>
          </el-button-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :loading="creating">
          {{ taskForm.dry_run ? '检查任务' : '启动任务' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 任务详情对话框（日志和可视化） -->
    <el-dialog 
      v-model="logDialogVisible" 
      title="任务详情" 
      width="85%"
      :close-on-click-modal="false"
    >
      <el-tabs v-model="detailActiveTab" type="border-card">
        <el-tab-pane label="执行日志" name="logs">
          <template #label>
            <span style="display: flex; align-items: center; gap: 6px">
              <el-icon><Document /></el-icon>
              执行日志
            </span>
          </template>
          <div style="height: 600px">
            <LogViewer :logs="logContent" :realtime="false" />
          </div>
        </el-tab-pane>
        <el-tab-pane label="执行可视化" name="visualization">
          <template #label>
            <span style="display: flex; align-items: center; gap: 6px">
              <el-icon><DataLine /></el-icon>
              执行可视化
            </span>
          </template>
          <div style="min-height: 600px; max-height: 80vh; overflow-y: auto">
            <TaskTimelineVisualization v-if="currentTaskId" :task-id="currentTaskId" />
          </div>
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="logDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 前置检查结果对话框 -->
    <el-dialog 
      v-model="preflightDialogVisible" 
      title="前置检查结果" 
      width="700px"
      :close-on-click-modal="false"
    >
      <div v-if="preflightResult" style="padding: 10px">
        <!-- 总体状态 -->
        <el-alert 
          :title="`总体状态: ${preflightResult.status === 'pass' ? '通过' : preflightResult.status === 'warning' ? '有警告' : '失败'}`"
          :type="getCheckStatusType(preflightResult.status)"
          :closable="false"
          style="margin-bottom: 20px"
        >
          <template #default>
            <div style="margin-top: 10px">
              <p>检查时间: {{ new Date(preflightResult.checked_at).toLocaleString() }}</p>
              <p>检查耗时: {{ preflightResult.duration }}ms</p>
            </div>
          </template>
        </el-alert>

        <!-- 检查摘要 -->
        <el-card shadow="never" style="margin-bottom: 20px">
          <template #header>
            <span style="font-weight: bold">检查摘要</span>
          </template>
          <el-row :gutter="20">
            <el-col :span="6">
              <el-statistic title="总检查项" :value="preflightResult.summary.total" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="通过" :value="preflightResult.summary.passed" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="警告" :value="preflightResult.summary.warnings" />
            </el-col>
            <el-col :span="6">
              <el-statistic title="失败" :value="preflightResult.summary.failed" />
            </el-col>
          </el-row>
        </el-card>

        <!-- 检查详情 -->
        <div style="margin-top: 20px">
          <h4 style="margin-bottom: 15px">检查详情</h4>
          <el-timeline>
            <el-timeline-item 
              v-for="(check, index) in preflightResult.checks" 
              :key="index"
              :type="getCheckStatusType(check.status)"
              :icon="getCheckStatusIcon(check.status)"
            >
              <el-card shadow="hover">
                <template #header>
                  <div style="display: flex; justify-content: space-between; align-items: center">
                    <span style="font-weight: bold">{{ check.name }}</span>
                    <el-tag :type="getCheckStatusType(check.status)" size="small">
                      {{ check.status === 'pass' ? '通过' : check.status === 'warning' ? '警告' : '失败' }}
                    </el-tag>
                  </div>
                </template>
                <div>
                  <p><strong>类别:</strong> {{ check.category }}</p>
                  <p><strong>消息:</strong> {{ check.message }}</p>
                  <p v-if="check.details"><strong>详情:</strong> {{ check.details }}</p>
                  <p style="color: #909399; font-size: 12px">
                    检查时间: {{ new Date(check.checked_at).toLocaleString() }} | 
                    耗时: {{ check.duration }}ms
                  </p>
                </div>
              </el-card>
            </el-timeline-item>
          </el-timeline>
        </div>

        <!-- 建议 -->
        <el-alert 
          v-if="preflightResult.status !== 'pass'"
          title="建议"
          type="info"
          :closable="false"
          style="margin-top: 20px"
        >
          <template #default>
            <ul style="margin: 0; padding-left: 20px">
              <li v-if="preflightResult.status === 'fail'">
                请先修复失败的检查项，然后重新执行检查
              </li>
              <li v-if="preflightResult.status === 'warning'">
                建议查看警告信息，评估风险后再决定是否执行任务
              </li>
              <li>您可以使用 Dry Run 模式进行更详细的测试</li>
            </ul>
          </template>
        </el-alert>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="preflightDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 环境和风险确认对话框 -->
    <ConfirmDialog
      v-if="confirmDialogVisible"
      v-model="confirmDialogVisible"
      ref="confirmDialogRef"
      :title="confirmDialogProps.title"
      :alert-title="confirmDialogProps.alertTitle"
      :alert-description="confirmDialogProps.alertDescription"
      :alert-type="'error'"
      :details="confirmDialogProps.details"
      @confirm="handleConfirmTask"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, DocumentCopy, Loading, CircleCheck, CircleClose, InfoFilled, Clock, MoreFilled, RefreshRight, Delete, Document, List, Calendar, DataLine, Setting, Key, Warning, QuestionFilled, Top, Bottom, Minus, View, Checked } from '@element-plus/icons-vue'
import * as ansibleAPI from '@/api/ansible'
import clusterAPI from '@/api/cluster'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import LogViewer from '@/components/LogViewer.vue'
import TaskTimelineVisualization from '@/components/ansible/TaskTimelineVisualization.vue'

// 数据
const tasks = ref([])
const total = ref(0)
const loading = ref(false)
const statistics = ref({})
const selectedTasks = ref([])
const recentTasks = ref([])

const queryParams = reactive({
  page: 1,
  page_size: 20,
  status: '',
  keyword: ''
})

const createDialogVisible = ref(false)
const logDialogVisible = ref(false)
const preflightDialogVisible = ref(false)
const creating = ref(false)
const preflightChecking = ref(false)
const preflightResult = ref(null)
const estimation = ref(null)
const detailActiveTab = ref('logs') // 任务详情对话框的活动 tab
const currentTaskId = ref(null) // 当前查看的任务 ID

const taskForm = reactive({
  name: '',
  template_id: null,
  cluster_id: null,
  inventory_id: null,
  extra_vars: {},
  dry_run: false,
  batch_config: {
    enabled: false,
    batch_size: 0,
    batch_percent: 20,
    pause_after_batch: false,
    failure_threshold: 0,
    max_batch_fail_rate: 50
  },
  timeout_seconds: 1800,  // 默认 30 分钟
  priority: 'medium'       // 任务优先级（high/medium/low）
})

// 分批执行相关状态
const batchEnabled = ref(false)
const batchStrategy = ref('percent') // 'size' 或 'percent'
const batchValue = ref(20) // 批次大小或百分比值

// 超时控制相关状态
const timeoutEnabled = ref(false)

const templates = ref([])
const inventories = ref([])
const clusters = ref([])
const logContent = ref('')

// 计算属性：选中的模板
const selectedTemplate = computed(() => {
  if (!taskForm.template_id) return null
  return templates.value.find(t => t.id === taskForm.template_id)
})

// 环境和风险确认对话框
const confirmDialogVisible = ref(false)
const confirmDialogRef = ref(null)
const pendingTaskData = ref(null)

// 方法
const loadTasks = async () => {
  loading.value = true
  try {
    const res = await ansibleAPI.listTasks(queryParams)
    console.log('任务列表响应:', res)
    // axios拦截器返回完整response，所以路径是: res.data.data 和 res.data.total
    tasks.value = res.data?.data || []
    total.value = res.data?.total || 0
  } catch (error) {
    console.error('加载任务列表失败:', error)
    ElMessage.error('加载任务列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 加载最近使用的任务
const loadRecentTasks = async () => {
  try {
    const res = await ansibleAPI.getRecentTasks(6) // 显示最近6个
    console.log('最近使用任务:', res)
    recentTasks.value = res.data?.data || []
  } catch (error) {
    console.error('加载最近使用任务失败:', error)
    // 不显示错误消息，静默失败
  }
}

// 格式化最近使用时间
const formatRecentTime = (dateStr) => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now - date
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  
  if (days > 0) return `${days} 天前`
  if (hours > 0) return `${hours} 小时前`
  if (minutes > 0) return `${minutes} 分钟前`
  return '刚刚'
}

// 重新执行任务
const rerunTask = (history) => {
  // 填充表单数据
  taskForm.name = history.task_name + ' (重新执行)'
  taskForm.template_id = history.template_id || null
  taskForm.inventory_id = history.inventory_id || null
  taskForm.cluster_id = history.cluster_id || null
  taskForm.playbook_content = history.playbook_content || ''
  taskForm.extra_vars = history.extra_vars || {}
  taskForm.dry_run = history.dry_run || false
  taskForm.batch_config = history.batch_config || {
    enabled: false,
    batch_size: 0,
    batch_percent: 20,
    pause_after_batch: false,
    failure_threshold: 5,
    max_batch_fail_rate: 30
  }
  
  // 同步分批执行UI状态
  if (taskForm.batch_config.enabled) {
    batchEnabled.value = true
    if (taskForm.batch_config.batch_size > 0) {
      batchStrategy.value = 'size'
      batchValue.value = taskForm.batch_config.batch_size
    } else if (taskForm.batch_config.batch_percent > 0) {
      batchStrategy.value = 'percent'
      batchValue.value = taskForm.batch_config.batch_percent
    }
  } else {
    batchEnabled.value = false
  }
  
  // 显示创建对话框
  createDialogVisible.value = true
  
  // 如果有模板ID，需要加载模板内容
  if (taskForm.template_id) {
    loadTemplateContent()
  }
}

// 处理最近使用任务的操作
const handleRecentTaskAction = async (command, history) => {
  if (command === 'rerun') {
    rerunTask(history)
  } else if (command === 'delete') {
    try {
      await ElMessageBox.confirm(
        '确定要删除这条历史记录吗？',
        '确认删除',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
      
      await ansibleAPI.deleteTaskHistory(history.id)
      ElMessage.success('删除成功')
      loadRecentTasks()
    } catch (error) {
      if (error !== 'cancel') {
        console.error('删除历史记录失败:', error)
        ElMessage.error('删除失败: ' + (error.message || '未知错误'))
      }
    }
  }
}

const loadStatistics = async () => {
  try {
    const res = await ansibleAPI.getStatistics()
    // axios拦截器返回完整response，所以路径是: res.data.data
    statistics.value = res.data?.data || {}
  } catch (error) {
    console.error('加载统计信息失败:', error)
  }
}

const loadTemplates = async () => {
  try {
    const res = await ansibleAPI.listTemplates({ page_size: 100 })
    // axios拦截器返回完整response，所以路径是: res.data.data
    templates.value = res.data?.data || []
  } catch (error) {
    console.error('加载模板失败:', error)
  }
}

const loadInventories = async () => {
  try {
    const res = await ansibleAPI.listInventories({ page_size: 100 })
    // axios拦截器返回完整response，所以路径是: res.data.data
    inventories.value = res.data?.data || []
  } catch (error) {
    console.error('加载主机清单失败:', error)
  }
}

const loadClusters = async () => {
  try {
    const res = await clusterAPI.getClusters()
    console.log('集群API完整响应:', res)
    console.log('响应数据:', res.data)
    // axios拦截器返回完整response，所以路径是: res.data.data.clusters
    clusters.value = res.data?.data?.clusters || []
    console.log('已加载集群:', clusters.value.length, '个', clusters.value)
  } catch (error) {
    console.error('加载集群失败:', error)
    ElMessage.error('加载集群失败: ' + error.message)
  }
}

const handleQuery = () => {
  queryParams.page = 1
  loadTasks()
}

const handleReset = () => {
  queryParams.status = ''
  queryParams.keyword = ''
  handleQuery()
}

const handleRefresh = () => {
  loadTasks()
  loadStatistics()
}

const showCreateDialog = () => {
  createDialogVisible.value = true
  loadTemplates()
  loadInventories()
  loadClusters()
  // 重置表单
  taskForm.extra_vars = {}
  // 重置分批执行状态
  batchEnabled.value = false
  batchStrategy.value = 'percent'
  batchValue.value = 20
}

// 处理分批执行开关
const handleBatchToggle = (enabled) => {
  taskForm.batch_config.enabled = enabled
  if (enabled) {
    // 根据策略设置批次大小
    if (batchStrategy.value === 'size') {
      taskForm.batch_config.batch_size = batchValue.value
      taskForm.batch_config.batch_percent = 0
    } else {
      taskForm.batch_config.batch_size = 0
      taskForm.batch_config.batch_percent = batchValue.value
    }
  }
}

// 超时控制切换
const handleTimeoutToggle = (enabled) => {
  if (!enabled) {
    taskForm.timeout_seconds = 0
  } else if (taskForm.timeout_seconds === 0) {
    taskForm.timeout_seconds = 1800 // 默认 30 分钟
  }
}

// 格式化时长显示
const formatDuration = (seconds) => {
  if (!seconds) return '不限制'
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return minutes > 0 ? `${hours}小时${minutes}分钟` : `${hours}小时`
}

// 格式化预估时长（支持小数秒）
const formatEstimationDuration = (seconds) => {
  if (!seconds) return '0秒'
  if (seconds < 60) return `${Math.round(seconds)}秒`
  if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60)
    const secs = Math.round(seconds % 60)
    return secs > 0 ? `${minutes}分钟${secs}秒` : `${minutes}分钟`
  }
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return minutes > 0 ? `${hours}小时${minutes}分钟` : `${hours}小时`
}

// 获取置信度文本
const getConfidenceText = (confidence) => {
  const textMap = {
    'high': '高（数据充足，预估可靠）',
    'medium': '中（数据适中，仅供参考）',
    'low': '低（数据较少，仅供参考）'
  }
  return textMap[confidence] || '未知'
}

// 加载任务执行预估
const loadEstimation = async () => {
  // 重置预估数据
  estimation.value = null
  
  // 需要模板ID或清单ID之一
  if (!taskForm.template_id && !taskForm.inventory_id) {
    return
  }
  
  try {
    let response
    
    if (taskForm.template_id && taskForm.inventory_id) {
      // 有模板和清单，使用组合预估
      response = await ansibleAPI.estimateByTemplateAndInventory(
        taskForm.template_id, 
        taskForm.inventory_id
      )
    } else if (taskForm.template_id) {
      // 只有模板
      response = await ansibleAPI.estimateByTemplate(taskForm.template_id)
    } else if (taskForm.inventory_id) {
      // 只有清单
      response = await ansibleAPI.estimateByInventory(taskForm.inventory_id)
    }
    
    if (response && response.data) {
      estimation.value = response.data
    }
  } catch (error) {
    // 预估失败不影响任务创建，静默处理
    console.log('预估加载失败:', error)
  }
}

// 监听策略和值的变化
const updateBatchConfig = () => {
  if (!batchEnabled.value) return
  
  if (batchStrategy.value === 'size') {
    taskForm.batch_config.batch_size = batchValue.value
    taskForm.batch_config.batch_percent = 0
  } else {
    taskForm.batch_config.batch_size = 0
    taskForm.batch_config.batch_percent = batchValue.value
  }
}

// 监听批次策略和值的变化
watch([batchStrategy, batchValue], updateBatchConfig)

const handleCreate = async () => {
  if (!taskForm.name) {
    ElMessage.warning('请输入任务名称')
    return
  }
  if (!taskForm.inventory_id) {
    ElMessage.warning('请选择主机清单')
    return
  }

  // 检查是否需要二次确认
  const selectedInventory = inventories.value.find(inv => inv.id === taskForm.inventory_id)
  const selectedTemplate = taskForm.template_id ? templates.value.find(tpl => tpl.id === taskForm.template_id) : null
  
  const isProduction = selectedInventory?.environment === 'production'
  const isHighRisk = selectedTemplate?.risk_level === 'high'
  
  if (isProduction || isHighRisk) {
    // 显示确认对话框
    pendingTaskData.value = { ...taskForm }
    confirmDialogVisible.value = true
    return
  }

  // 直接执行
  await executeTask(taskForm)
}

const executeTask = async (data) => {
  creating.value = true
  try {
    await ansibleAPI.createTask(data)
    ElMessage.success('任务已启动')
    createDialogVisible.value = false
    confirmDialogVisible.value = false
    loadTasks()
    loadStatistics()
    loadRecentTasks() // 刷新最近使用列表
  } catch (error) {
    ElMessage.error('启动任务失败: ' + error.message)
  } finally {
    creating.value = false
  }
}

const handleConfirmTask = async () => {
  if (!pendingTaskData.value) return
  
  if (confirmDialogRef.value) {
    confirmDialogRef.value.setConfirming(true)
  }
  
  await executeTask(pendingTaskData.value)
  
  if (confirmDialogRef.value) {
    confirmDialogRef.value.setConfirming(false)
  }
}

// 计算确认对话框属性
const confirmDialogProps = computed(() => {
  if (!pendingTaskData.value) return {}
  
  const selectedInventory = inventories.value.find(inv => inv.id === pendingTaskData.value.inventory_id)
  const selectedTemplate = pendingTaskData.value.template_id 
    ? templates.value.find(tpl => tpl.id === pendingTaskData.value.template_id) 
    : null
  
  const isProduction = selectedInventory?.environment === 'production'
  const isHighRisk = selectedTemplate?.risk_level === 'high'
  
  let title = '危险操作确认'
  let alertTitle = ''
  let alertDescription = ''
  
  if (isProduction && isHighRisk) {
    alertTitle = '生产环境 + 高风险操作'
    alertDescription = '您即将在生产环境执行高风险 Ansible 任务，此操作可能严重影响线上服务，请务必谨慎操作！'
  } else if (isProduction) {
    alertTitle = '生产环境操作'
    alertDescription = '您即将在生产环境执行 Ansible 任务，此操作可能影响线上服务，请谨慎操作。'
  } else if (isHighRisk) {
    alertTitle = '高风险操作'
    alertDescription = '此模板包含高风险操作（如删除、格式化等），执行前请仔细检查 Playbook 内容。'
  }
  
  const details = {
    '任务名称': pendingTaskData.value.name,
    '主机清单': selectedInventory?.name || '-',
    '环境': selectedInventory?.environment || '-',
    '模板': selectedTemplate?.name || '直接执行',
    '风险等级': selectedTemplate?.risk_level || '-'
  }
  
  return {
    title,
    alertTitle,
    alertDescription,
    details
  }
})

const handleViewLogs = async (row) => {
  currentTaskId.value = row.id // 设置当前任务 ID
  detailActiveTab.value = 'logs' // 默认显示日志 tab
  logDialogVisible.value = true
  try {
    const res = await ansibleAPI.getTaskLogs(row.id, { full: true })
    console.log('任务日志响应:', res)
    // axios拦截器返回完整response，数据是字符串格式
    logContent.value = res.data?.data || '暂无日志'
  } catch (error) {
    console.error('加载日志失败:', error)
    ElMessage.error('加载日志失败: ' + (error.message || '未知错误'))
  }
}

const handleCancel = async (row) => {
  try {
    await ElMessageBox.confirm('确定要取消此任务吗？', '提示', {
      type: 'warning'
    })
    await ansibleAPI.cancelTask(row.id)
    ElMessage.success('任务已取消')
    loadTasks()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('取消任务失败: ' + error.message)
    }
  }
}

const handleRetry = async (row) => {
  try {
    await ansibleAPI.retryTask(row.id)
    ElMessage.success('任务已重新启动')
    loadTasks()
  } catch (error) {
    ElMessage.error('重试任务失败: ' + error.message)
  }
}

// 批次控制方法
const handlePauseBatch = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要暂停任务 "${row.name}" 的批次执行吗？当前批次完成后将暂停。`,
      '暂停批次',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await ansibleAPI.pauseBatch(row.id)
    ElMessage.success('批次执行已暂停')
    loadTasks()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('暂停失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleContinueBatch = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要继续任务 "${row.name}" 的批次执行吗？将继续执行下一批次。`,
      '继续批次',
      {
        confirmButtonText: '继续',
        cancelButtonText: '取消',
        type: 'success'
      }
    )
    
    await ansibleAPI.continueBatch(row.id)
    ElMessage.success('批次执行已继续')
    loadTasks()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('继续失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleStopBatch = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要停止任务 "${row.name}" 的所有剩余批次吗？已完成的批次不会回滚，但剩余批次将不再执行。`,
      '停止批次',
      {
        confirmButtonText: '停止',
        cancelButtonText: '取消',
        type: 'error'
      }
    )
    
    await ansibleAPI.stopBatch(row.id)
    ElMessage.success('批次执行已停止')
    loadTasks()
    loadStatistics()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('停止失败: ' + (error.message || '未知错误'))
    }
  }
}

// 前置检查相关方法
const handlePreflightCheck = async (row) => {
  try {
    preflightChecking.value = true
    preflightResult.value = null
    
    const response = await ansibleAPI.runPreflightChecks(row.id)
    preflightResult.value = response.data
    preflightDialogVisible.value = true
    
    // 如果有失败或警告，提示用户
    if (preflightResult.value.status === 'fail') {
      ElMessage.warning('前置检查发现问题，请查看详情')
    } else if (preflightResult.value.status === 'warning') {
      ElMessage.warning('前置检查有警告，建议查看详情')
    } else {
      ElMessage.success('前置检查通过')
    }
  } catch (error) {
    console.error('前置检查失败:', error)
    ElMessage.error('前置检查失败: ' + (error.message || '未知错误'))
  } finally {
    preflightChecking.value = false
  }
}

// 获取检查状态类型
const getCheckStatusType = (status) => {
  const statusMap = {
    pass: 'success',
    warning: 'warning',
    fail: 'danger'
  }
  return statusMap[status] || ''
}

// 获取检查状态图标
const getCheckStatusIcon = (status) => {
  const iconMap = {
    pass: 'CircleCheck',
    warning: 'Warning',
    fail: 'CircleClose'
  }
  return iconMap[status] || 'QuestionFilled'
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除任务 "${row.name}" 吗？此操作不可恢复。`,
      '删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await ansibleAPI.deleteTask(row.id)
    ElMessage.success('删除成功')
    loadTasks()
    loadStatistics()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleBatchDelete = async () => {
  if (selectedTasks.value.length === 0) {
    ElMessage.warning('请先选择要删除的任务')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedTasks.value.length} 个任务吗？此操作不可恢复。`,
      '批量删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    const ids = selectedTasks.value.map(task => task.id)
    const res = await ansibleAPI.batchDeleteTasks(ids)
    
    const successCount = res.data?.success_count || 0
    const failedCount = res.data?.failed_count || 0
    
    if (failedCount > 0) {
      ElMessage.warning(`成功删除 ${successCount} 个任务，${failedCount} 个任务删除失败`)
      console.error('删除失败的任务:', res.data?.errors)
    } else {
      ElMessage.success(`成功删除 ${successCount} 个任务`)
    }
    
    selectedTasks.value = []
    loadTasks()
    loadStatistics()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败: ' + (error.message || '未知错误'))
    }
  }
}

const handleSelectionChange = (selection) => {
  selectedTasks.value = selection
}

const canSelectTask = (row) => {
  // 只能选择已完成、失败或取消的任务
  return row.status !== 'running' && row.status !== 'pending'
}

const copyLogs = async () => {
  try {
    await navigator.clipboard.writeText(logContent.value)
    ElMessage.success('日志已复制到剪贴板')
  } catch (error) {
    console.error('复制日志失败:', error)
    // 降级方案：使用传统方法
    const textArea = document.createElement('textarea')
    textArea.value = logContent.value
    document.body.appendChild(textArea)
    textArea.select()
    try {
      document.execCommand('copy')
      ElMessage.success('日志已复制到剪贴板')
    } catch (err) {
      ElMessage.error('复制失败，请手动复制')
    }
    document.body.removeChild(textArea)
  }
}

const getStatusType = (status) => {
  const types = {
    pending: '',
    running: 'warning',
    success: 'success',
    failed: 'danger',
    cancelled: 'info'
  }
  return types[status] || ''
}

const getStatusText = (status) => {
  const texts = {
    pending: '待执行',
    running: '运行中',
    success: '成功',
    failed: '失败',
    cancelled: '已取消'
  }
  return texts[status] || status
}

const calculateProgress = (task) => {
  if (task.hosts_total === 0) return 0
  return Math.round((task.hosts_ok + task.hosts_failed) / task.hosts_total * 100)
}

const formatDate = (dateStr) => {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}

// 生命周期
let refreshTimer = null

onMounted(() => {
  loadTasks()
  loadStatistics()
  loadRecentTasks()
  
  // 每 5 秒自动刷新
  refreshTimer = setInterval(() => {
    if (tasks.value.some(t => t.status === 'running')) {
      loadTasks()
      loadStatistics()
    }
  }, 5000)
})

onBeforeUnmount(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
.ansible-task-center {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

/* 统计卡片样式 */
.stats-row {
  margin-bottom: 0;
}

.stat-card {
  padding: 20px;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  cursor: pointer;
  transition: all 0.3s ease;
  border-left: 4px solid;
  margin-bottom: 20px;
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.stat-card-primary {
  border-left-color: #409EFF;
}

.stat-card-success {
  border-left-color: #67C23A;
}

.stat-card-warning {
  border-left-color: #E6A23C;
}

.stat-card-danger {
  border-left-color: #F56C6C;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-size: 28px;
}

.stat-card-primary .stat-icon {
  background: rgba(64, 158, 255, 0.1);
  color: #409EFF;
}

.stat-card-success .stat-icon {
  background: rgba(103, 194, 58, 0.1);
  color: #67C23A;
}

.stat-card-warning .stat-icon {
  background: rgba(230, 162, 60, 0.1);
  color: #E6A23C;
}

.stat-card-danger .stat-icon {
  background: rgba(245, 108, 108, 0.1);
  color: #F56C6C;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 32px;
  font-weight: 600;
  color: #303133;
  line-height: 1;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  font-weight: 400;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.log-container {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 16px;
  border-radius: 4px;
  border: 1px solid #3e3e3e;
}

.log-content {
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
  color: #d4d4d4;
}

/* 优化日志中不同类型的文本颜色 */
.log-content :deep(.error) {
  color: #f48771;
}

.log-content :deep(.success) {
  color: #89d185;
}

.log-content :deep(.warning) {
  color: #e5c07b;
}

/* 最近使用任务卡片样式 */
.recent-task-card {
  height: 100%;
  cursor: pointer;
  transition: all 0.3s ease;
}

.recent-task-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.recent-task-card :deep(.el-card__body) {
  padding: 16px;
}

.recent-task-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid #EBEEF5;
}

.recent-task-header .el-text {
  font-weight: 600;
  font-size: 14px;
  flex: 1;
  margin-right: 8px;
}

.recent-task-more {
  cursor: pointer;
  font-size: 16px;
  color: #909399;
  transition: color 0.3s;
}

.recent-task-more:hover {
  color: #409EFF;
}

.recent-task-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #606266;
}

.info-item .el-icon {
  font-size: 14px;
  color: #909399;
}

.info-item span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.recent-task-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  min-height: 24px;
}
</style>

