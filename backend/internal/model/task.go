package model

import (
	"time"

	"gorm.io/gorm"
)

// TaskStatus 任务状态枚举
type TaskStatus string

const (
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// ProgressTask 进度任务模型 - 用于多副本环境下的状态共享
type ProgressTask struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	TaskID      string         `json:"task_id" gorm:"uniqueIndex;not null"` // 任务唯一标识
	UserID      uint           `json:"user_id" gorm:"not null;index"`       // 所属用户
	Action      string         `json:"action" gorm:"not null"`              // 操作类型 batch_label, batch_taint
	Status      TaskStatus     `json:"status" gorm:"not null;index"`        // 任务状态
	Current     int            `json:"current" gorm:"default:0"`            // 当前完成数量
	Total       int            `json:"total" gorm:"not null"`               // 总数量
	Progress    float64        `json:"progress" gorm:"default:0"`           // 进度百分比
	CurrentNode string         `json:"current_node"`                        // 当前处理的节点
	Message     string         `json:"message"`                             // 状态消息
	ErrorMsg    string         `json:"error_msg"`                           // 错误消息
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CompletedAt *time.Time     `json:"completed_at"` // 完成时间
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// ProgressMessage 待发送的进度消息模型
type ProgressMessage struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	TaskID    string         `json:"task_id" gorm:"not null;index"`
	Type      string         `json:"type" gorm:"not null"`      // progress, complete, error
	Action    string         `json:"action"`                    // batch_label, batch_taint
	Current   int            `json:"current"`                   // 当前完成数量
	Total     int            `json:"total"`                     // 总数量
	Progress  float64        `json:"progress"`                  // 进度百分比
	Message   string         `json:"message"`                   // 消息内容
	ErrorMsg  string         `json:"error_msg"`                 // 错误信息
	Processed bool           `json:"processed" gorm:"default:false;index"` // 是否已处理
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// ToProgressMessage 转换为进度消息
func (pt *ProgressTask) ToProgressMessage() *ProgressMessage {
	msgType := "progress"
	if pt.Status == TaskStatusCompleted {
		msgType = "complete"
	} else if pt.Status == TaskStatusFailed {
		msgType = "error"
	}

	return &ProgressMessage{
		UserID:   pt.UserID,
		TaskID:   pt.TaskID,
		Type:     msgType,
		Action:   pt.Action,
		Current:  pt.Current,
		Total:    pt.Total,
		Progress: pt.Progress,
		Message:  pt.Message,
		ErrorMsg: pt.ErrorMsg,
	}
}

// UpdateProgress 更新任务进度
func (pt *ProgressTask) UpdateProgress(current int, currentNode string) {
	pt.Current = current
	pt.CurrentNode = currentNode
	pt.Progress = float64(current) / float64(pt.Total) * 100
	pt.UpdatedAt = time.Now()
}

// MarkCompleted 标记任务完成
func (pt *ProgressTask) MarkCompleted() {
	pt.Status = TaskStatusCompleted
	pt.Current = pt.Total
	pt.Progress = 100
	now := time.Now()
	pt.CompletedAt = &now
	pt.UpdatedAt = now
}

// MarkFailed 标记任务失败
func (pt *ProgressTask) MarkFailed(errorMsg string) {
	pt.Status = TaskStatusFailed
	pt.ErrorMsg = errorMsg
	now := time.Now()
	pt.CompletedAt = &now
	pt.UpdatedAt = now
}

// IsCompleted 检查任务是否完成
func (pt *ProgressTask) IsCompleted() bool {
	return pt.Status == TaskStatusCompleted || pt.Status == TaskStatusFailed || pt.Status == TaskStatusCancelled
}

// TaskSearchRequest 任务搜索请求
type TaskSearchRequest struct {
	UserID   uint       `json:"user_id"`
	Status   TaskStatus `json:"status"`
	Action   string     `json:"action"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

// MessageSearchRequest 消息搜索请求
type MessageSearchRequest struct {
	UserID    uint `json:"user_id"`
	Processed bool `json:"processed"`
	Page      int  `json:"page"`
	PageSize  int  `json:"page_size"`
}