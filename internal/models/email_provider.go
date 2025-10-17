package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailProviderSettings struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AccountID       string    `gorm:"type:uuid;not null" json:"account_id"`
	Provider        string    `gorm:"not null" json:"provider"` // mailgun, ses, resend, smtp
	APIKey          *string   `json:"apiKey,omitempty"`
	Domain          *string   `json:"domain,omitempty"`
	FromEmail       *string   `json:"fromEmail,omitempty"`
	FromName        *string   `json:"fromName,omitempty"`
	Region          *string   `json:"region,omitempty"`
	AccessKeyID     *string   `json:"accessKeyId,omitempty"`
	SecretAccessKey *string   `json:"secretAccessKey,omitempty"`
	IsActive        bool      `gorm:"default:false" json:"isActive"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	SMTPHost        *string   `json:"smtpHost,omitempty"`
	SMTPPassword    *string   `json:"smtpPassword,omitempty"`
	SMTPPort        *int      `json:"smtpPort,omitempty"`
	SMTPSecure      *bool     `gorm:"default:true" json:"smtpSecure,omitempty"`
	SMTPUser        *string   `json:"smtpUser,omitempty"`

	// Relationships
	Account Account `gorm:"foreignKey:AccountID;constraint:OnDelete:CASCADE" json:"account,omitempty"`
}

func (EmailProviderSettings) TableName() string {
	return "email_provider_settings"
}

func (eps *EmailProviderSettings) BeforeCreate(tx *gorm.DB) error {
	if eps.ID == "" {
		eps.ID = uuid.New().String()
	}
	return nil
}
