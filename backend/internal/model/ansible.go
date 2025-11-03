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
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	DeletedAt        gorm.DeletedAt         `json:"-" gorm:"index"`

	// 关联 - 删除模板/清单时将任务的外键设置为 NULL
	Template  *AnsibleTemplate  `json:"template,omitempty" gorm:"foreignKey:TemplateID;constraint:OnDelete:SET NULL"`
	Cluster   *Cluster          `json:"cluster,omitempty" gorm:"foreignKey:ClusterID;constraint:OnDelete:SET NULL"`
	Inventory *AnsibleInventory `json:"inventory,omitempty" gorm:"foreignKey:InventoryID;constraint:OnDelete:SET NULL"`
	User      *User             `json:"user,omitempty" gorm:"foreignKey:UserID"`
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

