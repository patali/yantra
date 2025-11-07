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
	"github.com/patali/yantra/src/config"
	"github.com/patali/yantra/src/controllers"
	"github.com/patali/yantra/src/db"
	"github.com/patali/yantra/src/db/repositories"
	"github.com/patali/yantra/src/executors"
	"github.com/patali/yantra/src/middleware"
	riverinternal "github.com/patali/yantra/src/river"
	"github.com/patali/yantra/src/services"
)

func main() {
	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Run River migrations first
	if err := database.RunRiverMigrations(ctx, cfg.DatabaseURL); err != nil {
		log.Fatalf("❌ Failed to run River migrations: %v", err)
	}

	// Run GORM migrations
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("❌ Failed to run GORM migrations: %v", err)
	}

	// Initialize repository layer
	repo := repositories.NewRepository(database.DB)

	// Initialize email service (needed by workflow engine and outbox worker)
	emailService := services.NewEmailService(database.DB)

	// Initialize workflow engine with email service dependency
	workflowEngine := services.NewWorkflowEngineService(database.DB, emailService)

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

	// Initialize services with repository pattern
	systemEmailService := services.NewSystemEmailService(cfg)
	authService := services.NewAuthService(database.DB, cfg.JWTSecret, systemEmailService)
	queueService := services.NewQueueService(riverClient.GetClient())
	workflowService := services.NewWorkflowService(database.DB, queueService)
	accountService := services.NewAccountService(repo)
	userService := services.NewUserService(repo)

	// Initialize and start scheduler service (using robfig/cron + River)
	schedulerService := services.NewSchedulerService(database.DB, queueService)
	workflowService.SetScheduler(schedulerService)      // Link scheduler to workflow service
	workflowEngine.SetSchedulerService(schedulerService) // Link scheduler to workflow engine (for sleep nodes)
	if err := schedulerService.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start scheduler: %v", err)
	}
	defer schedulerService.Stop(ctx)

	// Initialize and start outbox worker
	outboxService := services.NewOutboxService(database.DB)
	executorFactory := executors.NewExecutorFactory(database.DB, emailService)

	outboxWorker := services.NewOutboxWorkerService(outboxService, executorFactory)
	outboxWorker.Start(ctx)
	defer outboxWorker.Stop()

	// Run cleanup routines on startup
	cleanupService := services.NewCleanupService(repo)
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
	router.Use(middleware.CORS(cfg.AllowedOrigins)) // SECURITY: Use environment-based allowed origins
	router.Use(middleware.ErrorHandler())
	router.Use(middleware.RateLimitByMinute(100, 20)) // SECURITY: Global rate limit - 100 req/min, burst 20

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
		// Auth routes with stricter rate limiting
		// SECURITY: 10 requests per minute to prevent brute force attacks
		authAPI := api.Group("")
		authAPI.Use(middleware.RateLimitByMinute(10, 3)) // Stricter: 10 req/min, burst 3
		authController := controllers.NewAuthController(authService)
		authController.RegisterRoutes(authAPI)

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

	// Cancel main context to stop all goroutines
	cancel()

	// Graceful shutdown with 10 second timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop scheduler service
	log.Println("📅 Stopping scheduler...")
	if err := schedulerService.Stop(shutdownCtx); err != nil {
		log.Printf("⚠️  Error stopping scheduler: %v", err)
	}

	// Stop outbox worker
	log.Println("📦 Stopping outbox worker...")
	outboxWorker.Stop()

	// Stop River workers
	log.Println("🌊 Stopping River workers...")
	if err := riverClient.Stop(shutdownCtx); err != nil {
		log.Printf("⚠️  Error stopping River client: %v", err)
	}

	// Stop HTTP server
	log.Println("🌐 Stopping HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server shutdown complete")
}
