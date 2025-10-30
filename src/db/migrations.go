package db

import (
	"fmt"
	"log"

	"github.com/patali/yantra/src/db/models"
)

// AutoMigrate runs auto migrations for all models
// Note: For production, consider using a proper migration tool
func (d *Database) AutoMigrate() error {
	log.Println("🔄 Running GORM database migrations...")

	err := d.DB.AutoMigrate(
		&models.User{},
		&models.Account{},
		&models.AccountMember{},
		&models.Workflow{},
		&models.WorkflowVersion{},
		&models.WorkflowExecution{},
		&models.WorkflowNodeExecution{},
		&models.OutboxMessage{},
		&models.EmailProviderSettings{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✅ GORM database migrations completed")
	return nil
}
