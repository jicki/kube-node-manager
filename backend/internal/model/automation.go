package model

import (
	"time"

	"gorm.io/gorm"
)

// AnsiblePlaybook Ansible Playbook 记录（存储在数据库中）
type AnsiblePlaybook struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null;index" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Content     string         `gorm:"type:text;not null" json:"content"`      // Playbook YAML 内容
	Version     int            `gorm:"default:1" json:"version"`               // 版本号
	Category    string         `gorm:"type:varchar(50);index" json:"category"` // 分类：system, docker, kernel, security, custom
	Tags        string         `gorm:"type:varchar(255)" json:"tags"`          // 标签，逗号分隔
	Variables   string         `gorm:"type:json" json:"variables"`             // 参数定义（JSON 格式）
	IsBuiltin   bool           `gorm:"default:false;index" json:"is_builtin"`  // 是否内置（不可删除）
	IsActive    bool           `gorm:"default:true;index" json:"is_active"`    // 是否启用
	CreatedBy   uint           `json:"created_by"`
	UpdatedBy   uint           `json:"updated_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// AnsibleExecution Ansible Playbook 执行记录
type AnsibleExecution struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	TaskID       string         `gorm:"type:varchar(100);uniqueIndex" json:"task_id"`
	PlaybookID   uint           `gorm:"index" json:"playbook_id"`
	PlaybookName string         `gorm:"type:varchar(100)" json:"playbook_name"`
	ClusterName  string         `gorm:"type:varchar(100);index" json:"cluster_name"`
	ClusterID    *uint          `gorm:"index" json:"cluster_id"`
	TargetNodes  string         `gorm:"type:text" json:"target_nodes"`        // JSON 数组
	ExtraVars    string         `gorm:"type:json" json:"extra_vars"`          // 额外变量
	Tags         string         `gorm:"type:varchar(255)" json:"tags"`        // 执行的 tags
	CheckMode    bool           `gorm:"default:false" json:"check_mode"`      // 是否为检查模式
	Status       string         `gorm:"type:varchar(20);index" json:"status"` // pending, running, completed, failed, cancelled
	StartTime    *time.Time     `json:"start_time"`
	EndTime      *time.Time     `json:"end_time"`
	Duration     int            `json:"duration"`                // 执行时长（秒）
	Output       string         `gorm:"type:text" json:"output"` // 执行输出
	ErrorMessage string         `gorm:"type:text" json:"error_message"`
	SuccessCount int            `gorm:"default:0" json:"success_count"` // 成功节点数
	FailedCount  int            `gorm:"default:0" json:"failed_count"`  // 失败节点数
	UserID       uint           `gorm:"index" json:"user_id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 关联
	Playbook *AnsiblePlaybook `gorm:"foreignKey:PlaybookID" json:"playbook,omitempty"`
	User     *User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// SSHCredential SSH 凭据（加密存储）
type SSHCredential struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Username    string         `gorm:"type:varchar(50)" json:"username"`
	AuthType    string         `gorm:"type:varchar(20);not null" json:"auth_type"` // password, privatekey
	Password    string         `gorm:"type:text" json:"password"`                  // 加密存储
	PrivateKey  string         `gorm:"type:text" json:"private_key"`               // 加密存储
	Passphrase  string         `gorm:"type:varchar(255)" json:"passphrase"`        // 私钥密码（加密存储）
	Port        int            `gorm:"default:22" json:"port"`
	ClusterName string         `gorm:"type:varchar(100);index" json:"cluster_name"` // 关联集群
	NodePattern string         `gorm:"type:varchar(255)" json:"node_pattern"`       // 节点匹配模式（正则或通配符）
	IsDefault   bool           `gorm:"default:false;index" json:"is_default"`       // 是否为默认凭据
	CreatedBy   uint           `json:"created_by"`
	UpdatedBy   uint           `json:"updated_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Script 脚本记录
type Script struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null;index" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Content     string         `gorm:"type:text;not null" json:"content"`         // 脚本内容
	Language    string         `gorm:"type:varchar(20);not null" json:"language"` // shell, python, bash
	Version     int            `gorm:"default:1" json:"version"`                  // 版本号
	Category    string         `gorm:"type:varchar(50);index" json:"category"`    // 分类
	Tags        string         `gorm:"type:varchar(255)" json:"tags"`             // 标签
	Parameters  string         `gorm:"type:json" json:"parameters"`               // 参数定义
	IsBuiltin   bool           `gorm:"default:false;index" json:"is_builtin"`
	IsActive    bool           `gorm:"default:true;index" json:"is_active"`
	CreatedBy   uint           `json:"created_by"`
	UpdatedBy   uint           `json:"updated_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// ScriptExecution 脚本执行记录
type ScriptExecution struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	TaskID       string         `gorm:"type:varchar(100);uniqueIndex" json:"task_id"`
	ScriptID     uint           `gorm:"index" json:"script_id"`
	ScriptName   string         `gorm:"type:varchar(100)" json:"script_name"`
	ClusterName  string         `gorm:"type:varchar(100);index" json:"cluster_name"`
	ClusterID    *uint          `gorm:"index" json:"cluster_id"`
	TargetNodes  string         `gorm:"type:text" json:"target_nodes"` // JSON 数组
	Parameters   string         `gorm:"type:json" json:"parameters"`   // 执行参数
	Status       string         `gorm:"type:varchar(20);index" json:"status"`
	StartTime    *time.Time     `json:"start_time"`
	EndTime      *time.Time     `json:"end_time"`
	Duration     int            `json:"duration"`
	Output       string         `gorm:"type:text" json:"output"`
	ErrorMessage string         `gorm:"type:text" json:"error_message"`
	SuccessCount int            `gorm:"default:0" json:"success_count"`
	FailedCount  int            `gorm:"default:0" json:"failed_count"`
	UserID       uint           `gorm:"index" json:"user_id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 关联
	Script *Script `gorm:"foreignKey:ScriptID" json:"script,omitempty"`
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Workflow 工作流定义
type Workflow struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null;index" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Definition  string         `gorm:"type:text;not null" json:"definition"` // 工作流定义（JSON 格式的 DAG）
	Version     int            `gorm:"default:1" json:"version"`
	Category    string         `gorm:"type:varchar(50);index" json:"category"`
	Tags        string         `gorm:"type:varchar(255)" json:"tags"`
	IsBuiltin   bool           `gorm:"default:false;index" json:"is_builtin"`
	IsActive    bool           `gorm:"default:true;index" json:"is_active"`
	CreatedBy   uint           `json:"created_by"`
	UpdatedBy   uint           `json:"updated_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// WorkflowExecution 工作流执行记录
type WorkflowExecution struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	TaskID       string         `gorm:"type:varchar(100);uniqueIndex" json:"task_id"`
	WorkflowID   uint           `gorm:"index" json:"workflow_id"`
	WorkflowName string         `gorm:"type:varchar(100)" json:"workflow_name"`
	ClusterName  string         `gorm:"type:varchar(100);index" json:"cluster_name"`
	ClusterID    *uint          `gorm:"index" json:"cluster_id"`
	TargetNodes  string         `gorm:"type:text" json:"target_nodes"`
	Parameters   string         `gorm:"type:json" json:"parameters"`
	Status       string         `gorm:"type:varchar(20);index" json:"status"`
	CurrentStep  string         `gorm:"type:varchar(100)" json:"current_step"` // 当前执行步骤
	StepResults  string         `gorm:"type:text" json:"step_results"`         // 步骤执行结果（JSON）
	StartTime    *time.Time     `json:"start_time"`
	EndTime      *time.Time     `json:"end_time"`
	Duration     int            `json:"duration"`
	Output       string         `gorm:"type:text" json:"output"`
	ErrorMessage string         `gorm:"type:text" json:"error_message"`
	UserID       uint           `gorm:"index" json:"user_id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// 关联
	Workflow *Workflow `gorm:"foreignKey:WorkflowID" json:"workflow,omitempty"`
	User     *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// AutomationConfig 自动化配置（存储在数据库中，而不只是配置文件）
type AutomationConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Key       string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"key"` // 配置键
	Value     string    `gorm:"type:text" json:"value"`                            // 配置值（JSON 格式）
	Category  string    `gorm:"type:varchar(50);index" json:"category"`            // 配置分类：automation, ansible, ssh, scripts, workflows
	IsSystem  bool      `gorm:"default:false" json:"is_system"`                    // 是否为系统配置
	UpdatedBy uint      `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (AnsiblePlaybook) TableName() string {
	return "ansible_playbooks"
}

func (AnsibleExecution) TableName() string {
	return "ansible_executions"
}

func (SSHCredential) TableName() string {
	return "ssh_credentials"
}

func (Script) TableName() string {
	return "scripts"
}

func (ScriptExecution) TableName() string {
	return "script_executions"
}

func (Workflow) TableName() string {
	return "workflows"
}

func (WorkflowExecution) TableName() string {
	return "workflow_executions"
}

func (AutomationConfig) TableName() string {
	return "automation_configs"
}
