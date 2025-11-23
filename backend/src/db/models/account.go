package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Account struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (Account) TableName() string {
	return "accounts"
}

func (a *Account) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

type AccountMember struct {
	AccountID string    `gorm:"type:uuid;primaryKey" json:"accountId"`
	UserID    string    `gorm:"type:uuid;primaryKey" json:"userId"`
	Role      string    `gorm:"default:owner" json:"role"` // owner | admin | member
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (AccountMember) TableName() string {
	return "account_members"
}
