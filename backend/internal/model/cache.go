package model

import (
	"time"
)

// CacheEntry 缓存条目
type CacheEntry struct {
	Key       string    `gorm:"primaryKey;size:255" json:"key"`
	Value     []byte    `gorm:"type:bytea;not null" json:"value"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (CacheEntry) TableName() string {
	return "cache_entries"
}
