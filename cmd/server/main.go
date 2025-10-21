package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patali/yantra/internal/config"
	"github.com/patali/yantra/internal/controllers"
	"github.com/patali/yantra/internal/db"
	"github.com/patali/yantra/internal/executors"
	"github.com/patali/yantra/internal/middleware"
	riverinternal "github.com/patali/yantra/internal/river"
	"github.com/patali/yantra/internal/services"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Initialize database
	database, err := db.New(cfg.DatabaseURL, cfg.Environment == "development")
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("❌ Failed to run migrations: %v", err)
	}

	// Initialize workflow engine
	workflowEngine := services.NewWorkflowEngineService(database.DB)

	// Initialize River client with workflow engine
	riverClient, err := riverinternal.NewClient(ctx, cfg.DatabaseURL, workflowEngine)
	if err != nil {
		log.Fatalf("❌ Failed to create River client: %v", err)
	}

	// Start River workers
	if err := riverClient.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start River workers: %v", err)
	}
	defer riverClient.Stop(ctx)

	// Initialize services
	authService := services.NewAuthService(database.DB, cfg.JWTSecret)
	queueService := services.NewQueueService(riverClient.GetClient())
	workflowService := services.NewWorkflowService(database.DB, queueService)
	accountService := services.NewAccountService(database.DB)
	userService := services.NewUserService(database.DB)

	// Initialize and start scheduler service (using robfig/cron + River)
	schedulerService := services.NewSchedulerService(database.DB, queueService)
	workflowService.SetScheduler(schedulerService) // Link scheduler to workflow service
	if err := schedulerService.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start scheduler: %v", err)
	}
	defer schedulerService.Stop(ctx)

	// Initialize and start outbox worker
	outboxService := services.NewOutboxService(database.DB)
	executorFactory := executors.NewExecutorFactory(database.DB)

	// Initialize email service and inject into executor factory
	emailService := services.NewEmailService(database.DB)
	executorFactory.SetEmailService(emailService)

	outboxWorker := services.NewOutboxWorkerService(outboxService, executorFactory)
	outboxWorker.Start(ctx)
	defer outboxWorker.Stop()

	// Run cleanup routines on startup
	cleanupService := services.NewCleanupService(database.DB)
	if err := cleanupService.RunAllCleanups(ctx); err != nil {
		log.Printf("⚠️  Warning: Cleanup routines encountered errors: %v", err)
	}

	// Initialize Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.ErrorHandler())

	// Middleware to inject DB into context
	router.Use(func(c *gin.Context) {
		c.Set("db", database.DB)
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// API routes
	api := router.Group("/api")
	{
		// Auth routes
		authController := controllers.NewAuthController(authService)
		authController.RegisterRoutes(api)

		// Workflow routes
		workflowController := controllers.NewWorkflowController(workflowService)
		workflowController.RegisterRoutes(api, authService)

		// User routes
		userController := controllers.NewUserController(userService, authService)
		userController.RegisterRoutes(api, authService)

		// Account routes
		accountController := controllers.NewAccountController(accountService)
		accountController.RegisterRoutes(api, authService)

		// Settings routes
		settingsController := controllers.NewSettingsController(database.DB)
		settingsController.RegisterRoutes(api, authService)

		// Recovery routes
		recoveryController := controllers.NewRecoveryController(outboxService, workflowService, workflowEngine)
		recoveryController.RegisterRoutes(api, authService)

		// Migration routes (protected by API key)
		migrationController := controllers.NewMigrationController(database.DB)
		migrationController.RegisterRoutes(api)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 Starting Yantra backend on port %s...", cfg.Port)
		log.Printf("🔍 Health check: http://localhost:%s/health", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown with 10 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop River workers
	if err := riverClient.Stop(shutdownCtx); err != nil {
		log.Printf("⚠️  Error stopping River client: %v", err)
	}

	// Stop HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server shutdown complete")
}
