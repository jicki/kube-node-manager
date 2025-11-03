package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// TaskStatus 任务状态枚举
type AnsibleTaskStatus string

const (
	AnsibleTaskStatusPending   AnsibleTaskStatus = "pending"
	AnsibleTaskStatusRunning   AnsibleTaskStatus = "running"
	AnsibleTaskStatusSuccess   AnsibleTaskStatus = "success"
	AnsibleTaskStatusFailed    AnsibleTaskStatus = "failed"
	AnsibleTaskStatusCancelled AnsibleTaskStatus = "cancelled"
)

// LogType 日志类型枚举
type AnsibleLogType string

const (
	AnsibleLogTypeStdout AnsibleLogType = "stdout"
	AnsibleLogTypeStderr AnsibleLogType = "stderr"
	AnsibleLogTypeEvent  AnsibleLogType = "event"
)

// SourceType 主机清单来源类型
type InventorySourceType string

const (
	InventorySourceK8s    InventorySourceType = "k8s"
	InventorySourceManual InventorySourceType = "manual"
)

// SSHKeyType SSH 密钥类型
type SSHKeyType string

const (
	SSHKeyTypePrivateKey SSHKeyType = "private_key"  // 私钥
	SSHKeyTypePassword   SSHKeyType = "password"      // 密码认证
)

// SSHAuthType SSH 认证类型（用于 Inventory）
type SSHAuthType string

const (
	SSHAuthTypeKey      SSHAuthType = "key"      // 使用密钥认证
	SSHAuthTypePassword SSHAuthType = "password" // 使用密码认证
)

// TaskPriority 任务优先级
type TaskPriority string

const (
	TaskPriorityHigh   TaskPriority = "high"   // 高优先级
	TaskPriorityMedium TaskPriority = "medium" // 中优先级（默认）
	TaskPriorityLow    TaskPriority = "low"    // 低优先级
)

// ExtraVars 额外变量类型（用于 JSON 序列化）
type ExtraVars map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (ev *ExtraVars) Scan(value interface{}) error {
	if value == nil {
		*ev = make(ExtraVars)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, ev)
}

// Value 实现 driver.Valuer 接口
func (ev ExtraVars) Value() (driver.Value, error) {
	if ev == nil {
		return nil, nil
	}
	return json.Marshal(ev)
}

// HostsData 主机数据类型
type HostsData map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (hd *HostsData) Scan(value interface{}) error {
	if value == nil {
		*hd = make(HostsData)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, hd)
}

// Value 实现 driver.Valuer 接口
func (hd HostsData) Value() (driver.Value, error) {
	if hd == nil {
		return nil, nil
	}
	return json.Marshal(hd)
}

// StringArray 字符串数组类型（用于 JSONB）
type StringArray []string

// Scan 实现 sql.Scanner 接口
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = make(StringArray, 0)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, sa)
}

// Value 实现 driver.Valuer 接口
func (sa StringArray) Value() (driver.Value, error) {
	if sa == nil {
		return json.Marshal([]string{})
	}
	return json.Marshal(sa)
}

// AnsibleTask Ansible 任务模型
type AnsibleTask struct {
	ID               uint              `json:"id" gorm:"primarykey"`
	Name             string            `json:"name" gorm:"not null;size:255;comment:任务名称"`
	TemplateID       *uint             `json:"template_id" gorm:"index;comment:关联模板ID"`
	ClusterID        *uint             `json:"cluster_id" gorm:"index;comment:关联集群ID"`
	InventoryID      *uint             `json:"inventory_id" gorm:"index;comment:关联主机清单ID"`
	Status           AnsibleTaskStatus `json:"status" gorm:"not null;index;size:50;comment:任务状态"`
	UserID           uint              `json:"user_id" gorm:"not null;index;comment:执行用户ID"`
	StartedAt        *time.Time        `json:"started_at" gorm:"comment:开始时间"`
	FinishedAt       *time.Time        `json:"finished_at" gorm:"comment:完成时间"`
	Duration         int               `json:"duration" gorm:"default:0;comment:执行时长(秒)"`
	HostsTotal       int               `json:"hosts_total" gorm:"default:0;comment:主机总数"`
	HostsOk          int               `json:"hosts_ok" gorm:"default:0;comment:成功主机数"`
	HostsFailed      int               `json:"hosts_failed" gorm:"default:0;comment:失败主机数"`
	HostsSkipped     int               `json:"hosts_skipped" gorm:"default:0;comment:跳过主机数"`
	ErrorMsg         string            `json:"error_msg" gorm:"type:text;comment:错误信息"`
	PlaybookContent  string            `json:"playbook_content" gorm:"type:text;not null;comment:Playbook内容"`
	FullLog          string            `json:"full_log" gorm:"type:text;comment:完整日志"`
	LogSize          int64             `json:"log_size" gorm:"default:0;comment:日志大小(bytes)"`
	ExtraVars        ExtraVars         `json:"extra_vars" gorm:"type:jsonb;comment:额外变量"`
	RetryPolicy      *RetryPolicy           `json:"retry_policy" gorm:"type:jsonb;comment:重试策略"`
	RetryCount       int                    `json:"retry_count" gorm:"default:0;comment:当前重试次数"`
	MaxRetries       int                    `json:"max_retries" gorm:"default:0;comment:最大重试次数"`
	DryRun           bool                   `json:"dry_run" gorm:"default:false;comment:是否为检查模式(Dry Run)"`
	BatchConfig      *BatchExecutionConfig  `json:"batch_config" gorm:"type:jsonb;comment:分批执行配置"`
	CurrentBatch     int                    `json:"current_batch" gorm:"default:0;comment:当前执行批次"`
	TotalBatches     int                    `json:"total_batches" gorm:"default:0;comment:总批次数"`
	BatchStatus      string                 `json:"batch_status" gorm:"size:50;comment:批次状态(running/paused/completed)"`
	PreflightChecks  *PreflightCheckResult  `json:"preflight_checks" gorm:"type:jsonb;comment:前置检查结果"`
	TimeoutSeconds   int                    `json:"timeout_seconds" gorm:"default:0;comment:超时时间(秒),0表示不限制"`
	IsTimedOut       bool                   `json:"is_timed_out" gorm:"default:false;comment:是否超时"`
	Priority         string                 `json:"priority" gorm:"size:20;default:'medium';index;comment:任务优先级(high/medium/low)"`
	QueuedAt         *time.Time             `json:"queued_at" gorm:"comment:入队时间"`
	WaitDuration     int                    `json:"wait_duration" gorm:"default:0;comment:等待时长(秒)"`
	ExecutionTimeline *TaskExecutionTimeline `json:"execution_timeline" gorm:"type:jsonb;comment:执行时间线"`
	
	// 工作流相关字段
	WorkflowExecutionID *uint       `json:"workflow_execution_id" gorm:"index;comment:工作流执行ID"`
	DependsOn           StringArray `json:"depends_on" gorm:"type:jsonb;comment:依赖的节点ID列表"`
	NodeID              string      `json:"node_id" gorm:"size:50;comment:工作流节点ID"`
	
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	DeletedAt        gorm.DeletedAt         `json:"-" gorm:"index"`

	// 关联 - 删除模板/清单时将任务的外键设置为 NULL
	Template  *AnsibleTemplate  `json:"template,omitempty" gorm:"foreignKey:TemplateID;constraint:OnDelete:SET NULL"`
	Cluster   *Cluster          `json:"cluster,omitempty" gorm:"foreignKey:ClusterID;constraint:OnDelete:SET NULL"`
	Inventory *AnsibleInventory `json:"inventory,omitempty" gorm:"foreignKey:InventoryID;constraint:OnDelete:SET NULL"`
	User      *User             `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Tags      []AnsibleTag      `json:"tags,omitempty" gorm:"many2many:ansible_task_tags;"`
	WorkflowExecution *AnsibleWorkflowExecution `json:"workflow_execution,omitempty" gorm:"foreignKey:WorkflowExecutionID;constraint:OnDelete:SET NULL"`
}

// TableName 指定表名
func (AnsibleTask) TableName() string {
	return "ansible_tasks"
}

// IsRunning 检查任务是否正在运行
func (t *AnsibleTask) IsRunning() bool {
	return t.Status == AnsibleTaskStatusRunning
}

// IsCompleted 检查任务是否已完成
func (t *AnsibleTask) IsCompleted() bool {
	return t.Status == AnsibleTaskStatusSuccess ||
		t.Status == AnsibleTaskStatusFailed ||
		t.Status == AnsibleTaskStatusCancelled
}

// MarkStarted 标记任务开始
func (t *AnsibleTask) MarkStarted() {
	now := time.Now()
	t.Status = AnsibleTaskStatusRunning
	t.StartedAt = &now
	t.UpdatedAt = now
}

// MarkCompleted 标记任务完成
func (t *AnsibleTask) MarkCompleted(success bool, errorMsg string) {
	now := time.Now()
	if success {
		t.Status = AnsibleTaskStatusSuccess
	} else {
		t.Status = AnsibleTaskStatusFailed
		t.ErrorMsg = errorMsg
	}
	t.FinishedAt = &now
	t.UpdatedAt = now

	// 计算执行时长
	if t.StartedAt != nil {
		t.Duration = int(now.Sub(*t.StartedAt).Seconds())
	}
}

// MarkCancelled 标记任务取消
func (t *AnsibleTask) MarkCancelled() {
	now := time.Now()
	t.Status = AnsibleTaskStatusCancelled
	t.FinishedAt = &now
	t.UpdatedAt = now

	// 计算执行时长
	if t.StartedAt != nil {
		t.Duration = int(now.Sub(*t.StartedAt).Seconds())
	}
}

// UpdateStats 更新统计信息
func (t *AnsibleTask) UpdateStats(total, ok, failed, skipped int) {
	t.HostsTotal = total
	t.HostsOk = ok
	t.HostsFailed = failed
	t.HostsSkipped = skipped
	t.UpdatedAt = time.Now()
}

// IsBatchEnabled 检查是否启用分批执行
func (t *AnsibleTask) IsBatchEnabled() bool {
	return t.BatchConfig != nil && t.BatchConfig.Enabled
}

// AddExecutionEvent 添加执行事件到时间线
func (t *AnsibleTask) AddExecutionEvent(phase ExecutionPhase, message string, details map[string]interface{}) {
	if t.ExecutionTimeline == nil {
		timeline := make(TaskExecutionTimeline, 0)
		t.ExecutionTimeline = &timeline
	}
	
	event := TaskExecutionEvent{
		Phase:     phase,
		Message:   message,
		Timestamp: time.Now(),
		Details:   details,
	}
	
	// 计算上一个事件的耗时
	if len(*t.ExecutionTimeline) > 0 {
		lastEvent := &(*t.ExecutionTimeline)[len(*t.ExecutionTimeline)-1]
		lastEvent.Duration = int(event.Timestamp.Sub(lastEvent.Timestamp).Milliseconds())
	}
	
	*t.ExecutionTimeline = append(*t.ExecutionTimeline, event)
}

// IsBatchPaused 检查批次是否暂停
func (t *AnsibleTask) IsBatchPaused() bool {
	return t.BatchStatus == "paused"
}

// MarkBatchCompleted 标记批次完成
func (t *AnsibleTask) MarkBatchCompleted() {
	t.BatchStatus = "completed"
	t.UpdatedAt = time.Now()
}

// AnsibleTemplate Ansible 任务模板模型
type AnsibleTemplate struct {
	ID              uint           `json:"id" gorm:"primarykey"`
	Name            string         `json:"name" gorm:"not null;size:255;comment:模板名称"` // 唯一索引由迁移文件创建
	Description     string         `json:"description" gorm:"type:text;comment:模板描述"`
	PlaybookContent string         `json:"playbook_content" gorm:"type:text;not null;comment:Playbook内容"`
	Variables       ExtraVars      `json:"variables" gorm:"type:jsonb;comment:变量定义"`
	RequiredVars    []string       `json:"required_vars" gorm:"type:jsonb;comment:必需变量列表"`
	Tags            string         `json:"tags" gorm:"size:255;comment:标签(逗号分隔)"`
	RiskLevel       string         `json:"risk_level" gorm:"size:20;default:'low';comment:风险等级(low/medium/high)"`
	UserID          uint           `json:"user_id" gorm:"not null;index;comment:创建用户ID"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (AnsibleTemplate) TableName() string {
	return "ansible_templates"
}

// AnsibleLog Ansible 任务执行日志模型
type AnsibleLog struct {
	ID         uint           `json:"id" gorm:"primarykey"`
	TaskID     uint           `json:"task_id" gorm:"not null;index;comment:关联任务ID"`
	LogType    AnsibleLogType `json:"log_type" gorm:"size:50;comment:日志类型"`
	Content    string         `json:"content" gorm:"type:text;not null;comment:日志内容"`
	LineNumber int            `json:"line_number" gorm:"comment:行号"`
	CreatedAt  time.Time      `json:"created_at" gorm:"index"`

	// 关联 - 删除任务时级联删除日志
	Task *AnsibleTask `json:"task,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE"`
}

// TableName 指定表名
func (AnsibleLog) TableName() string {
	return "ansible_logs"
}

// AnsibleInventory Ansible 主机清单模型
type AnsibleInventory struct {
	ID          uint                `json:"id" gorm:"primarykey"`
	Name        string              `json:"name" gorm:"not null;size:255;comment:清单名称"` // 唯一索引由迁移文件创建
	Description string              `json:"description" gorm:"type:text;comment:清单描述"`
	SourceType  InventorySourceType `json:"source_type" gorm:"size:50;comment:来源类型"`
	ClusterID   *uint               `json:"cluster_id" gorm:"index;comment:关联集群ID(可选)"`
	SSHKeyID    *uint               `json:"ssh_key_id" gorm:"index;comment:关联SSH密钥ID"`
	Content     string              `json:"content" gorm:"type:text;not null;comment:清单内容(INI或YAML)"`
	HostsData   HostsData           `json:"hosts_data" gorm:"type:jsonb;comment:结构化主机数据"`
	Environment string              `json:"environment" gorm:"size:20;default:'dev';comment:环境标签(dev/staging/production)"`
	UserID      uint                `json:"user_id" gorm:"not null;index;comment:创建用户ID"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	DeletedAt   gorm.DeletedAt      `json:"-" gorm:"index"`

	// 关联
	Cluster *Cluster       `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
	SSHKey  *AnsibleSSHKey `json:"ssh_key,omitempty" gorm:"foreignKey:SSHKeyID"`
	User    *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (AnsibleInventory) TableName() string {
	return "ansible_inventories"
}

// IsFromK8s 检查是否来自 K8s
func (i *AnsibleInventory) IsFromK8s() bool {
	return i.SourceType == InventorySourceK8s
}

// TaskListRequest 任务列表请求
type TaskListRequest struct {
	Page       int               `json:"page" form:"page"`
	PageSize   int               `json:"page_size" form:"page_size"`
	Status     AnsibleTaskStatus `json:"status" form:"status"`
	UserID     uint              `json:"user_id" form:"user_id"`
	ClusterID  uint              `json:"cluster_id" form:"cluster_id"`
	TemplateID uint              `json:"template_id" form:"template_id"`
	Keyword    string            `json:"keyword" form:"keyword"`
}

// TaskCreateRequest 任务创建请求
type TaskCreateRequest struct {
	Name            string                 `json:"name" binding:"required"`
	TemplateID      *uint                  `json:"template_id"`
	ClusterID       *uint                  `json:"cluster_id"`
	InventoryID     *uint                  `json:"inventory_id"`
	PlaybookContent string                 `json:"playbook_content"`
	ExtraVars       map[string]interface{} `json:"extra_vars"`
	DryRun          bool                   `json:"dry_run"`       // 是否为检查模式（不实际执行变更）
	BatchConfig     *BatchExecutionConfig  `json:"batch_config"`  // 分批执行配置
	TimeoutSeconds  int                    `json:"timeout_seconds"` // 超时时间（秒），0表示不限制
	Priority        string                 `json:"priority"`        // 任务优先级（high/medium/low），默认medium
}

// TemplateListRequest 模板列表请求
type TemplateListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Keyword  string `json:"keyword" form:"keyword"`
	UserID   uint   `json:"user_id" form:"user_id"`
}

// TemplateCreateRequest 模板创建请求
type TemplateCreateRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description"`
	PlaybookContent string                 `json:"playbook_content" binding:"required"`
	Variables       map[string]interface{} `json:"variables"`
	Tags            string                 `json:"tags"`
}

// TemplateUpdateRequest 模板更新请求
type TemplateUpdateRequest struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	PlaybookContent string                 `json:"playbook_content"`
	Variables       map[string]interface{} `json:"variables"`
	Tags            string                 `json:"tags"`
}

// InventoryListRequest 主机清单列表请求
type InventoryListRequest struct {
	Page       int                 `json:"page" form:"page"`
	PageSize   int                 `json:"page_size" form:"page_size"`
	SourceType InventorySourceType `json:"source_type" form:"source_type"`
	ClusterID  uint                `json:"cluster_id" form:"cluster_id"`
	Keyword    string              `json:"keyword" form:"keyword"`
}

// InventoryCreateRequest 主机清单创建请求
type InventoryCreateRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	SourceType  InventorySourceType    `json:"source_type" binding:"required"`
	ClusterID   *uint                  `json:"cluster_id"`
	SSHKeyID    *uint                  `json:"ssh_key_id"` // 关联的 SSH 密钥 ID
	Content     string                 `json:"content" binding:"required"`
	HostsData   map[string]interface{} `json:"hosts_data"`
}

// InventoryUpdateRequest 主机清单更新请求
type InventoryUpdateRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	SSHKeyID    *uint                  `json:"ssh_key_id"` // 关联的 SSH 密钥 ID
	Content     string                 `json:"content"`
	HostsData   map[string]interface{} `json:"hosts_data"`
}

// GenerateInventoryRequest 从集群生成清单请求
type GenerateInventoryRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	ClusterID   uint              `json:"cluster_id" binding:"required"`
	SSHKeyID    *uint             `json:"ssh_key_id"`  // 关联的 SSH 密钥 ID
	NodeLabels  map[string]string `json:"node_labels"` // 用于筛选节点的标签
}

// ======================== SSH 密钥管理 ========================

// AnsibleSSHKey SSH 密钥模型
type AnsibleSSHKey struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"not null"` // 唯一索引由迁移文件创建
	Description string         `json:"description"`
	Type        SSHKeyType     `json:"type" gorm:"not null"`                  // private_key 或 password
	Username    string         `json:"username" gorm:"not null"`              // SSH 用户名
	PrivateKey  string         `json:"-" gorm:"type:text"`                    // 私钥内容（加密存储）
	Passphrase  string         `json:"-" gorm:"type:text"`                    // 私钥密码（加密存储）
	Password    string         `json:"-" gorm:"type:text"`                    // SSH 密码（加密存储）
	Port        int            `json:"port" gorm:"default:22"`                // SSH 端口
	IsDefault   bool           `json:"is_default" gorm:"default:false"`       // 是否为默认密钥
	CreatedBy   uint           `json:"created_by" gorm:"not null"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// SSHKeyListRequest SSH 密钥列表请求
type SSHKeyListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Keyword  string `json:"keyword" form:"keyword"`
	Type     string `json:"type" form:"type"`
}

// SSHKeyCreateRequest SSH 密钥创建请求
type SSHKeyCreateRequest struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	Type        SSHKeyType `json:"type" binding:"required"`
	Username    string     `json:"username" binding:"required"`
	PrivateKey  string     `json:"private_key"`  // Type=private_key 时必需
	Passphrase  string     `json:"passphrase"`   // 私钥密码（可选）
	Password    string     `json:"password"`     // Type=password 时必需
	Port        int        `json:"port"`         // 默认 22
	IsDefault   bool       `json:"is_default"`
}

// SSHKeyUpdateRequest SSH 密钥更新请求
type SSHKeyUpdateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Username    string `json:"username"`
	PrivateKey  string `json:"private_key"`
	Passphrase  string `json:"passphrase"`
	Password    string `json:"password"`
	Port        int    `json:"port"`
	IsDefault   bool   `json:"is_default"`
}

// SSHKeyResponse SSH 密钥响应（不包含敏感信息）
type SSHKeyResponse struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        SSHKeyType `json:"type"`
	Username    string     `json:"username"`
	Port        int        `json:"port"`
	IsDefault   bool       `json:"is_default"`
	HasPrivateKey bool     `json:"has_private_key"` // 是否有私钥
	HasPassphrase bool     `json:"has_passphrase"` // 是否有密码短语
	HasPassword   bool     `json:"has_password"`   // 是否有密码
	CreatedBy   uint       `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ToResponse 转换为响应对象（不包含敏感信息）
func (k *AnsibleSSHKey) ToResponse() *SSHKeyResponse {
	return &SSHKeyResponse{
		ID:            k.ID,
		Name:          k.Name,
		Description:   k.Description,
		Type:          k.Type,
		Username:      k.Username,
		Port:          k.Port,
		IsDefault:     k.IsDefault,
		HasPrivateKey: k.PrivateKey != "",
		HasPassphrase: k.Passphrase != "",
		HasPassword:   k.Password != "",
		CreatedBy:     k.CreatedBy,
		CreatedAt:     k.CreatedAt,
		UpdatedAt:     k.UpdatedAt,
	}
}

// ======================== 定时任务调度 ========================

// AnsibleSchedule 定时任务调度模型
type AnsibleSchedule struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"not null;size:255;comment:调度任务名称"`
	Description string         `json:"description" gorm:"type:text;comment:调度任务描述"`
	TemplateID  uint           `json:"template_id" gorm:"not null;index;comment:关联模板ID"`
	InventoryID uint           `json:"inventory_id" gorm:"not null;index;comment:关联主机清单ID"`
	ClusterID   *uint          `json:"cluster_id" gorm:"index;comment:关联集群ID"`
	CronExpr    string         `json:"cron_expr" gorm:"not null;size:100;comment:Cron表达式"`
	ExtraVars   ExtraVars      `json:"extra_vars" gorm:"type:jsonb;comment:额外变量"`
	Enabled     bool           `json:"enabled" gorm:"default:true;index;comment:是否启用"`
	LastRunAt   *time.Time     `json:"last_run_at" gorm:"comment:上次执行时间"`
	NextRunAt   *time.Time     `json:"next_run_at" gorm:"index;comment:下次执行时间"`
	RunCount    int            `json:"run_count" gorm:"default:0;comment:执行次数"`
	UserID      uint           `json:"user_id" gorm:"not null;index;comment:创建用户ID"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联
	Template  *AnsibleTemplate  `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
	Inventory *AnsibleInventory `json:"inventory,omitempty" gorm:"foreignKey:InventoryID"`
	Cluster   *Cluster          `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
	User      *User             `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (AnsibleSchedule) TableName() string {
	return "ansible_schedules"
}

// ScheduleListRequest 调度列表请求
type ScheduleListRequest struct {
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"page_size" form:"page_size"`
	Enabled   *bool  `json:"enabled" form:"enabled"`
	ClusterID uint   `json:"cluster_id" form:"cluster_id"`
	Keyword   string `json:"keyword" form:"keyword"`
}

// ScheduleCreateRequest 调度创建请求
type ScheduleCreateRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	TemplateID  uint                   `json:"template_id" binding:"required"`
	InventoryID uint                   `json:"inventory_id" binding:"required"`
	ClusterID   *uint                  `json:"cluster_id"`
	CronExpr    string                 `json:"cron_expr" binding:"required"`
	ExtraVars   map[string]interface{} `json:"extra_vars"`
	Enabled     bool                   `json:"enabled"`
}

// ScheduleUpdateRequest 调度更新请求
type ScheduleUpdateRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	TemplateID  uint                   `json:"template_id"`
	InventoryID uint                   `json:"inventory_id"`
	ClusterID   *uint                  `json:"cluster_id"`
	CronExpr    string                 `json:"cron_expr"`
	ExtraVars   map[string]interface{} `json:"extra_vars"`
	Enabled     *bool                  `json:"enabled"`
}

// ======================== 重试策略 ========================

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries    int  `json:"max_retries"`     // 最大重试次数
	RetryInterval int  `json:"retry_interval"`  // 重试间隔（秒）
	RetryOnError  bool `json:"retry_on_error"`  // 是否在错误时重试
}

// Scan 实现 sql.Scanner 接口
func (rp *RetryPolicy) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, rp)
}

// Value 实现 driver.Valuer 接口
func (rp RetryPolicy) Value() (driver.Value, error) {
	return json.Marshal(rp)
}

// ======================== 分批执行配置 ========================

// BatchExecutionConfig 分批执行配置
type BatchExecutionConfig struct {
	Enabled          bool   `json:"enabled"`            // 是否启用分批执行
	BatchSize        int    `json:"batch_size"`         // 每批主机数量（与 BatchPercent 二选一）
	BatchPercent     int    `json:"batch_percent"`      // 每批主机百分比（0-100）
	PauseAfterBatch  bool   `json:"pause_after_batch"`  // 每批执行后是否暂停等待确认
	FailureThreshold int    `json:"failure_threshold"`  // 失败阈值（失败主机数超过此值则停止）
	MaxBatchFailRate int    `json:"max_batch_fail_rate"` // 单批最大失败率（0-100，超过则停止）
}

// Scan 实现 sql.Scanner 接口
func (bec *BatchExecutionConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, bec)
}

// Value 实现 driver.Valuer 接口
func (bec BatchExecutionConfig) Value() (driver.Value, error) {
	return json.Marshal(bec)
}

// ======================== 前置检查 ========================

// PreflightCheckResult 前置检查结果
type PreflightCheckResult struct {
	Status      string              `json:"status"`       // overall/pass/warning/fail
	CheckedAt   time.Time           `json:"checked_at"`   // 检查时间
	Duration    int                 `json:"duration"`     // 检查耗时（毫秒）
	Checks      []PreflightCheck    `json:"checks"`       // 检查项列表
	Summary     PreflightSummary    `json:"summary"`      // 检查摘要
}

// PreflightCheck 单个检查项
type PreflightCheck struct {
	Name        string    `json:"name"`         // 检查项名称
	Category    string    `json:"category"`     // 类别（connectivity/resources/config）
	Status      string    `json:"status"`       // pass/warning/fail
	Message     string    `json:"message"`      // 检查结果消息
	Details     string    `json:"details"`      // 详细信息
	CheckedAt   time.Time `json:"checked_at"`   // 检查时间
	Duration    int       `json:"duration"`     // 耗时（毫秒）
}

// PreflightSummary 检查摘要
type PreflightSummary struct {
	Total       int `json:"total"`        // 总检查项数
	Passed      int `json:"passed"`       // 通过数
	Warnings    int `json:"warnings"`     // 警告数
	Failed      int `json:"failed"`       // 失败数
}

// Scan 实现 sql.Scanner 接口
func (pcr *PreflightCheckResult) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, pcr)
}

// Value 实现 driver.Valuer 接口
func (pcr PreflightCheckResult) Value() (driver.Value, error) {
	return json.Marshal(pcr)
}

// ======================== 收藏和快速操作 ========================

// AnsibleFavorite 收藏记录
type AnsibleFavorite struct {
	ID         uint           `json:"id" gorm:"primarykey"`
	UserID     uint           `json:"user_id" gorm:"not null;index;comment:用户ID"`
	TargetType string         `json:"target_type" gorm:"not null;size:50;comment:目标类型(task/template/inventory)"` // task/template/inventory
	TargetID   uint           `json:"target_id" gorm:"not null;comment:目标ID"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 注意：不定义外键约束，因为 TargetID 是动态引用，根据 TargetType 指向不同的表
	// 关联数据通过业务逻辑手动加载
}

// TableName 指定表名
func (AnsibleFavorite) TableName() string {
	return "ansible_favorites"
}

// AnsibleTaskHistory 任务执行历史（用于快速重新执行）
type AnsibleTaskHistory struct {
	ID               uint           `json:"id" gorm:"primarykey"`
	UserID           uint           `json:"user_id" gorm:"not null;index;comment:用户ID"`
	TaskName         string         `json:"task_name" gorm:"size:255;comment:任务名称"`
	TemplateID       *uint          `json:"template_id" gorm:"comment:模板ID"`
	InventoryID      *uint          `json:"inventory_id" gorm:"comment:清单ID"`
	ClusterID        *uint          `json:"cluster_id" gorm:"comment:集群ID"`
	PlaybookContent  string         `json:"playbook_content" gorm:"type:text;comment:Playbook内容"`
	ExtraVars        ExtraVars      `json:"extra_vars" gorm:"type:jsonb;comment:额外变量"`
	DryRun           bool           `json:"dry_run" gorm:"default:false;comment:是否Dry Run"`
	BatchConfig      *BatchExecutionConfig `json:"batch_config" gorm:"type:jsonb;comment:分批配置"`
	LastUsedAt       time.Time      `json:"last_used_at" gorm:"index;comment:最后使用时间"`
	UseCount         int            `json:"use_count" gorm:"default:1;comment:使用次数"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	
	// 关联
	Template  *AnsibleTemplate  `json:"template,omitempty" gorm:"foreignKey:TemplateID"`
	Inventory *AnsibleInventory `json:"inventory,omitempty" gorm:"foreignKey:InventoryID"`
	Cluster   *Cluster          `json:"cluster,omitempty" gorm:"foreignKey:ClusterID"`
}

// TableName 指定表名
func (AnsibleTaskHistory) TableName() string {
	return "ansible_task_history"
}

// AnsibleTag 任务标签模型
type AnsibleTag struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"not null;size:50;uniqueIndex;comment:标签名称"`
	Color       string         `json:"color" gorm:"size:20;default:'#409EFF';comment:标签颜色"`
	Description string         `json:"description" gorm:"size:255;comment:标签描述"`
	UserID      uint           `json:"user_id" gorm:"not null;index;comment:创建用户ID"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 关联
	User  *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Tasks []AnsibleTask  `json:"tasks,omitempty" gorm:"many2many:ansible_task_tags;"`
}

// TableName 指定表名
func (AnsibleTag) TableName() string {
	return "ansible_tags"
}

// AnsibleTaskTag 任务标签关联表
type AnsibleTaskTag struct {
	TaskID    uint      `json:"task_id" gorm:"primarykey"`
	TagID     uint      `json:"tag_id" gorm:"primarykey"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (AnsibleTaskTag) TableName() string {
	return "ansible_task_tags"
}

// TagCreateRequest 标签创建请求
type TagCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// TagUpdateRequest 标签更新请求
type TagUpdateRequest struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// TagListRequest 标签列表请求
type TagListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Keyword  string `json:"keyword" form:"keyword"`
}

// BatchTagOperationRequest 批量标签操作请求
type BatchTagOperationRequest struct {
	TaskIDs []uint `json:"task_ids" binding:"required"`
	TagIDs  []uint `json:"tag_ids" binding:"required"`
	Action  string `json:"action" binding:"required,oneof=add remove"` // add: 添加标签, remove: 移除标签
}

// ExecutionPhase 任务执行阶段
type ExecutionPhase string

const (
	PhaseQueued         ExecutionPhase = "queued"          // 入队等待
	PhasePreflightCheck ExecutionPhase = "preflight_check" // 前置检查
	PhaseExecuting      ExecutionPhase = "executing"       // 执行中
	PhaseBatchPaused    ExecutionPhase = "batch_paused"    // 批次暂停
	PhaseCompleted      ExecutionPhase = "completed"       // 已完成
	PhaseFailed         ExecutionPhase = "failed"          // 失败
	PhaseCancelled      ExecutionPhase = "cancelled"       // 已取消
	PhaseTimeout        ExecutionPhase = "timeout"         // 超时
)

// TaskExecutionEvent 任务执行事件
type TaskExecutionEvent struct {
	Phase       ExecutionPhase `json:"phase"`        // 执行阶段
	Message     string         `json:"message"`      // 事件消息
	Timestamp   time.Time      `json:"timestamp"`    // 事件时间
	Duration    int            `json:"duration"`     // 阶段耗时（毫秒）
	BatchNumber int            `json:"batch_number"` // 批次号（如果适用）
	HostCount   int            `json:"host_count"`   // 主机数量
	SuccessCount int           `json:"success_count"` // 成功数量
	FailCount   int            `json:"fail_count"`   // 失败数量
	Details     map[string]interface{} `json:"details"` // 额外详情
}

// TaskExecutionTimeline 任务执行时间线
type TaskExecutionTimeline []TaskExecutionEvent

// Scan 实现 sql.Scanner 接口
func (t *TaskExecutionTimeline) Scan(value interface{}) error {
	if value == nil {
		*t = make(TaskExecutionTimeline, 0)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, t)
}

// Value 实现 driver.Valuer 接口
func (t TaskExecutionTimeline) Value() (driver.Value, error) {
	if t == nil {
		return json.Marshal([]TaskExecutionEvent{})
	}
	return json.Marshal(t)
}

// HostExecutionStatus 主机执行状态
type HostExecutionStatus struct {
	HostName     string    `json:"host_name"`      // 主机名
	Status       string    `json:"status"`         // 状态（ok/failed/skipped/unreachable）
	StartTime    time.Time `json:"start_time"`     // 开始时间
	EndTime      time.Time `json:"end_time"`       // 结束时间
	Duration     int       `json:"duration"`       // 执行时长（毫秒）
	TasksOk      int       `json:"tasks_ok"`       // 成功任务数
	TasksFailed  int       `json:"tasks_failed"`   // 失败任务数
	TasksSkipped int       `json:"tasks_skipped"`  // 跳过任务数
	Changed      bool      `json:"changed"`        // 是否有变更
	ErrorMessage string    `json:"error_message"`  // 错误信息
}

// TaskExecutionVisualization 任务执行可视化数据
type TaskExecutionVisualization struct {
	TaskID          uint                  `json:"task_id"`
	TaskName        string                `json:"task_name"`
	Status          string                `json:"status"`
	Timeline        TaskExecutionTimeline `json:"timeline"`         // 执行时间线
	HostStatuses    []HostExecutionStatus `json:"host_statuses"`    // 主机状态列表
	TotalDuration   int                   `json:"total_duration"`   // 总耗时（毫秒）
	PhaseDistribution map[string]int      `json:"phase_distribution"` // 各阶段耗时分布
}

// ============================================================================
// DAG 工作流相关模型
// ============================================================================

// AnsibleWorkflow 工作流定义
type AnsibleWorkflow struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"not null;size:255;comment:工作流名称"`
	Description string         `json:"description" gorm:"type:text;comment:工作流描述"`
	DAG         *WorkflowDAG   `json:"dag" gorm:"type:jsonb;comment:DAG定义"`
	UserID      uint           `json:"user_id" gorm:"not null;index;comment:创建用户ID"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	// 关联
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (AnsibleWorkflow) TableName() string {
	return "ansible_workflows"
}

// WorkflowDAG DAG 定义
type WorkflowDAG struct {
	Nodes []WorkflowNode `json:"nodes"`
	Edges []WorkflowEdge `json:"edges"`
}

// Scan 实现 sql.Scanner 接口
func (dag *WorkflowDAG) Scan(value interface{}) error {
	if value == nil {
		*dag = WorkflowDAG{Nodes: []WorkflowNode{}, Edges: []WorkflowEdge{}}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, dag)
}

// Value 实现 driver.Valuer 接口
func (dag WorkflowDAG) Value() (driver.Value, error) {
	return json.Marshal(dag)
}

// WorkflowNode 工作流节点
type WorkflowNode struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"` // task/start/end
	Label      string            `json:"label"`
	TaskConfig *TaskCreateRequest `json:"task_config,omitempty"`
	Position   Position          `json:"position"`
}

// Position 节点位置（用于UI展示）
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// WorkflowEdge 工作流边
type WorkflowEdge struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	Condition string `json:"condition,omitempty"` // 条件表达式
}

// AnsibleWorkflowExecution 工作流执行记录
type AnsibleWorkflowExecution struct {
	ID           uint           `json:"id" gorm:"primarykey"`
	WorkflowID   uint           `json:"workflow_id" gorm:"not null;index;comment:工作流ID"`
	Status       string         `json:"status" gorm:"size:50;default:'running';index;comment:执行状态"`
	StartedAt    time.Time      `json:"started_at" gorm:"comment:开始时间"`
	FinishedAt   *time.Time     `json:"finished_at" gorm:"comment:完成时间"`
	ErrorMessage string         `json:"error_message" gorm:"type:text;comment:错误信息"`
	UserID       uint           `json:"user_id" gorm:"not null;index;comment:执行用户ID"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	
	// 关联
	Workflow *AnsibleWorkflow `json:"workflow,omitempty" gorm:"foreignKey:WorkflowID"`
	User     *User            `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Tasks    []AnsibleTask    `json:"tasks,omitempty" gorm:"foreignKey:WorkflowExecutionID"`
}

// TableName 指定表名
func (AnsibleWorkflowExecution) TableName() string {
	return "ansible_workflow_executions"
}

// WorkflowCreateRequest 工作流创建请求
type WorkflowCreateRequest struct {
	Name        string       `json:"name" binding:"required"`
	Description string       `json:"description"`
	DAG         *WorkflowDAG `json:"dag" binding:"required"`
}

// WorkflowUpdateRequest 工作流更新请求
type WorkflowUpdateRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	DAG         *WorkflowDAG `json:"dag"`
}

// WorkflowListRequest 工作流列表请求
type WorkflowListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Keyword  string `json:"keyword" form:"keyword"`
}

// WorkflowExecutionListRequest 工作流执行记录列表请求
type WorkflowExecutionListRequest struct {
	Page       int    `json:"page" form:"page"`
	PageSize   int    `json:"page_size" form:"page_size"`
	WorkflowID uint   `json:"workflow_id" form:"workflow_id"`
	Status     string `json:"status" form:"status"`
}

