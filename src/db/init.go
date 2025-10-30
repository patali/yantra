package db

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"gorm.io/gorm"
)

var (
	appDB            *Database
	createAppDbOnce  sync.Once
	dbProvider       DatabaseProvider = ProvidePostgresDB
)

// Database wraps the GORM DB instance
type Database struct {
	DB *gorm.DB
}

// InjectDBProvider allows injecting a custom database provider (useful for testing)
func InjectDBProvider(provider DatabaseProvider) {
	dbProvider = provider
}

// GetAppDB returns the singleton database instance
func GetAppDB(config *Config) (*Database, error) {
	var err error

	createAppDbOnce.Do(func() {
		log.Println("🔧 Initializing database connection...")

		var db *gorm.DB
		db, err = ConnectWithRetry(config)
		if err != nil {
			return
		}

		appDB = &Database{DB: db}

		// Run migrations
		if err = appDB.AutoMigrate(); err != nil {
			err = fmt.Errorf("failed to run GORM migrations: %w", err)
			return
		}

		log.Println("✅ Database initialization complete")
	})

	if err != nil {
		return nil, err
	}

	if appDB == nil {
		return nil, fmt.Errorf("database initialization failed")
	}

	return appDB, nil
}

// New creates a new database instance (deprecated: use GetAppDB for singleton pattern)
// Kept for backward compatibility
func New(databaseURL string, debug bool) (*Database, error) {
	config := &Config{
		DatabaseURL: databaseURL,
		Debug:       debug,
	}
	return GetAppDB(config)
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

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
