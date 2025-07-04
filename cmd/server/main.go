package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aebalz/daily-vibe-tracker/docs"
	"github.com/aebalz/daily-vibe-tracker/internal/config"
	"github.com/aebalz/daily-vibe-tracker/internal/handler"
	"github.com/aebalz/daily-vibe-tracker/internal/repository"
	"github.com/aebalz/daily-vibe-tracker/internal/service"
	"github.com/aebalz/daily-vibe-tracker/pkg/database"

	fiberserver "github.com/aebalz/daily-vibe-tracker/pkg/fiber"
	ginserver "github.com/aebalz/daily-vibe-tracker/pkg/gin"
)

// @title Daily Vibe Tracker API
// @version 1.0
// @description This is a simple API for tracking daily vibes.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @schemes http https
func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.env")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logger (can be more sophisticated, e.g., using zerolog based on cfg.LogLevel)
	log.SetOutput(os.Stdout)
	log.Printf("Log level set to: %s", cfg.LogLevel) // Simple log, can be enhanced

	// Update Swagger info based on config
	docs.SwaggerInfo.Version = "1.0" // Prompt specified version 1.0
	docs.SwaggerInfo.Title = cfg.AppName + " - Daily Vibe Tracker API"
	docs.SwaggerInfo.Description = "Complete API for the Daily Vibe Tracker application, including vibe management, analytics, and advanced features."
	docs.SwaggerInfo.Host = cfg.SwaggerHost
	docs.SwaggerInfo.BasePath = cfg.SwaggerBasePath // Should be /api/v1 as per spec for vibe routes
	docs.SwaggerInfo.Schemes = cfg.SwaggerSchemes

	// Connect to database
	db, err := database.ConnectDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.CloseDB()

	// Run migrations
	if err := database.MigrateDB(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize dependencies (Repository, Service, Handler)
	// This is a simplified wire-up. In a larger app, consider dependency injection frameworks.

	// Health Handler (common for both frameworks)
	healthHandler := handler.NewHealthHandler(db)

	// Initialize Redis Cache
	// redisCache, err := cache.NewRedisCache(cfg)
	// if err != nil {
	// 	log.Printf("Warning: Failed to connect to Redis, caching will be disabled: %v", err)
	// 	// Depending on policy, you might choose to log.Fatalf here if cache is critical
	// 	// For now, we'll allow the app to run without cache if Redis connection fails.
	// 	redisCache = nil // Ensure it's nil if connection failed
	// } else {
	// 	log.Println("Successfully connected to Redis.")
	// 	defer redisCache.Close()
	// }

	// Vibe specific components
	vibeRepo := repository.NewVibeRepository(db)
	vibeSvc := service.NewVibeService(vibeRepo, cfg) // Pass cache and config

	// Main Vibe Handler (will contain all handlers)
	mainVibeHandler := &handler.VibeHandler{
		Service:       vibeSvc,
		HealthHandler: healthHandler,
	}

	// Graceful shutdown channel
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start the selected server
	switch cfg.ServerFramework {
	case "fiber":
		fiberApp := fiberserver.NewFiberServer(cfg, mainVibeHandler)
		go func() {
			if err := fiberserver.StartFiberServer(fiberApp, cfg); err != nil {
				log.Fatalf("Failed to start Fiber server: %v", err)
			}
		}()
		<-quit
		log.Println("Shutting down Fiber server...")
		if err := fiberApp.Shutdown(); err != nil {
			log.Printf("Error during Fiber server shutdown: %v", err)
		}
	case "gin":
		ginEngine := ginserver.NewGinServer(cfg, mainVibeHandler)
		httpServer, err := ginserver.StartGinServer(ginEngine, cfg)
		if err != nil {
			log.Fatalf("Failed to start GIN server: %v", err)
		}
		<-quit
		log.Println("Shutting down GIN server...")
		// Define a timeout for server shutdown, e.g., 5 seconds
		shutdownTimeout := 5 * time.Second
		ginserver.ShutdownGinServer(httpServer, shutdownTimeout)

	default:
		log.Fatalf("Unsupported server framework: %s. Supported: 'fiber', 'gin'", cfg.ServerFramework)
	}

	log.Println("Server gracefully stopped.")
}

// End of file
