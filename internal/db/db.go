package db

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/patali/yantra/internal/models"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
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

// RunRiverMigrations runs River queue migrations programmatically
func (d *Database) RunRiverMigrations(ctx context.Context, databaseURL string) error {
	log.Println("🌊 Running River migrations...")

	// Create a pgx pool for River migrations
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create pgx pool for migrations: %w", err)
	}
	defer pool.Close()

	// Create migrator and run migrations
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("failed to create River migrator: %w", err)
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("failed to run River migrations: %w", err)
	}

	log.Println("✅ River migrations completed")
	return nil
}

// AutoMigrate runs auto migrations for all models
// Note: For production, review generated SQL before applying
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

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
