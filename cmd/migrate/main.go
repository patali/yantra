package main

import (
	"log"
	"os"

	"github.com/patali/yantra/internal/config"
	"github.com/patali/yantra/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database, err := db.New(cfg.DatabaseURL, false)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Running GORM migrations...")
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("✅ GORM migrations completed successfully")
	os.Exit(0)
}
