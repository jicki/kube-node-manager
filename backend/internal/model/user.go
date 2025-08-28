package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;not null"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"`
	Role      UserRole  `json:"role" gorm:"default:user"`
	Status    UserStatus `json:"status" gorm:"default:active"`
	LastLogin *time.Time `json:"last_login"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
	RoleViewer UserRole = "viewer"
)

type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusInactive UserStatus = "inactive"
	StatusBlocked  UserStatus = "blocked"
)

func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

func (u *User) CanManageUsers() bool {
	return u.Role == RoleAdmin
}

func (u *User) CanManageNodes() bool {
	return u.Role == RoleAdmin || u.Role == RoleUser
}

func (u *User) CanViewNodes() bool {
	return u.Role == RoleAdmin || u.Role == RoleUser || u.Role == RoleViewer
}