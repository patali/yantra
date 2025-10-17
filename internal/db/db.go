package db

import (
	"fmt"
	"log"

	"github.com/patali/yantra/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	DB *gorm.DB
}

func New(databaseURL string, debug bool) (*Database, error) {
	logLevel := logger.Silent
	if debug {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	log.Println("✅ Database connected successfully")

	return &Database{DB: db}, nil
}

// AutoMigrate runs auto migrations for all models
// Note: For production, review generated SQL before applying
func (d *Database) AutoMigrate() error {
	log.Println("🔄 Running database migrations...")

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

	log.Println("✅ Database migrations completed")
	return nil
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
