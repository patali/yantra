package db

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	_numAttempts = 10
	_delay       = 1 * time.Second
	_openSync    sync.Mutex
)

// DatabaseProvider is a function type that provides a database connection
type DatabaseProvider func(config *Config) (*gorm.DB, error)

// Config holds database configuration
type Config struct {
	DatabaseURL string
	Debug       bool
}

// attemptToOpen tries to open a database connection with retry logic
func attemptToOpen(config *Config, provider DatabaseProvider) (*gorm.DB, error) {
	_openSync.Lock()
	defer _openSync.Unlock()

	var db *gorm.DB
	var err error

	for i := 1; i <= _numAttempts; i++ {
		log.Printf("📡 Attempting to connect to database (attempt %d/%d)...", i, _numAttempts)

		db, err = provider(config)
		if err == nil {
			log.Println("✅ Database connected successfully")
			return db, nil
		}

		if i < _numAttempts {
			log.Printf("⚠️  Connection failed, retrying in %v... (error: %v)", _delay, err)
			time.Sleep(_delay)
		}
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", _numAttempts, err)
}

// ProvidePostgresDB creates a PostgreSQL connection with GORM
func ProvidePostgresDB(config *Config) (*gorm.DB, error) {
	logLevel := logger.Silent
	if config.Debug {
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	db, err := gorm.Open(postgres.Open(config.DatabaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// ConnectWithRetry connects to the database with retry logic
func ConnectWithRetry(config *Config) (*gorm.DB, error) {
	return attemptToOpen(config, ProvidePostgresDB)
}
