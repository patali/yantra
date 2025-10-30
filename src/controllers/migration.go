package controllers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/src/db"
	"gorm.io/gorm"
)

type MigrationController struct {
	db *gorm.DB
}

func NewMigrationController(db *gorm.DB) *MigrationController {
	return &MigrationController{
		db: db,
	}
}

// RegisterRoutes registers migration routes
func (ctrl *MigrationController) RegisterRoutes(rg *gin.RouterGroup) {
	migration := rg.Group("/migration")
	migration.Use(migrationAPIKeyMiddleware())
	{
		migration.POST("/run", ctrl.RunMigrations)
		migration.GET("/status", ctrl.GetMigrationStatus)
	}
}

// migrationAPIKeyMiddleware validates the MIGRATION_API_KEY
func migrationAPIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := os.Getenv("MIGRATION_API_KEY")

		// If no API key is set, disable the endpoint
		if apiKey == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Migration API is disabled. Set MIGRATION_API_KEY environment variable to enable.",
			})
			c.Abort()
			return
		}

		// Check Authorization header
		providedKey := c.GetHeader("X-Migration-Key")
		if providedKey == "" {
			providedKey = c.GetHeader("Authorization")
		}

		if providedKey != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing migration API key",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RunMigrations runs both River and GORM migrations programmatically
func (ctrl *MigrationController) RunMigrations(c *gin.Context) {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "DATABASE_URL not set",
		})
		return
	}

	results := make(map[string]any)

	// Get database instance
	database := &db.Database{DB: ctrl.db}

	// Run River migrations programmatically
	riverErr := database.RunRiverMigrations(ctx, databaseURL)
	results["river"] = map[string]any{
		"error": nil,
	}
	if riverErr != nil {
		results["river"].(map[string]any)["error"] = riverErr.Error()
	}

	// Run GORM migrations
	gormErr := database.AutoMigrate()
	results["gorm"] = map[string]any{
		"error": nil,
	}
	if gormErr != nil {
		results["gorm"].(map[string]any)["error"] = gormErr.Error()
	}

	// Determine overall status
	hasErrors := riverErr != nil || gormErr != nil
	status := "success"
	if hasErrors {
		status = "partial_failure"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"results": results,
	})
}

// GetMigrationStatus checks if migrations are needed
func (ctrl *MigrationController) GetMigrationStatus(c *gin.Context) {
	// Simple health check - try to query a known table
	var count int64
	err := ctrl.db.Raw("SELECT COUNT(*) FROM river_migration").Scan(&count).Error

	riverMigrated := err == nil

	c.JSON(http.StatusOK, gin.H{
		"river_migrated": riverMigrated,
		"database_url":   fmt.Sprintf("Connected to: %s", ctrl.db.Name()),
	})
}
