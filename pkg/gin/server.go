package gin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	swaggoFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/aebalz/daily-vibe-tracker/internal/config"
	"github.com/aebalz/daily-vibe-tracker/internal/handler" // Will be created later
	customMiddleware "github.com/aebalz/daily-vibe-tracker/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	// Import docs for swagger
	_ "github.com/aebalz/daily-vibe-tracker/docs"
)

const RequestIDKey = "requestID"

// NewGinServer creates and configures a new Gin application.
func NewGinServer(cfg *config.AppConfig, vibeHandler *handler.VibeHandler) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())        // Recovery middleware
	router.Use(requestIDMiddleware()) // Request ID middleware
	router.Use(loggingMiddleware())   // Custom logging middleware
	// Add Metrics and Rate Limiting middleware
	router.Use(customMiddleware.MetricsMiddlewareGin())
	router.Use(customMiddleware.RateLimiterGin(cfg.RateLimitPerSecond, cfg.RateLimitBurst))


	corsConfig := cors.DefaultConfig()
	if len(cfg.CorsAllowedOrigins) == 1 && cfg.CorsAllowedOrigins[0] == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = cfg.CorsAllowedOrigins
	}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	router.Use(cors.New(corsConfig))

	// Swagger UI
	// BasePath for swagger UI itself. If docs.SwaggerInfo.BasePath is /api/v1,
	// then swagger docs will be found relative to that for API calls, but the UI
	// itself is served from /swagger/*any
	// The /swagger/doc.json path is relative to the server root.
	url := ginSwagger.URL("/swagger/doc.json")
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggoFiles.Handler, url))


	// Prometheus Metrics Endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))


	// Routes
	// Health Check Route
	if vibeHandler != nil && vibeHandler.HealthHandler != nil {
		router.GET("/health", vibeHandler.HealthHandler.CheckHealthGin)
	} else {
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "initializing health handler"})
		})
	}

	// Vibe Routes
	apiV1 := router.Group("/api/v1") // All vibe routes will be under /api/v1
	{
		vibesGroup := apiV1.Group("/vibes")
		// Example of group specific middleware:
		// vibesGroup.Use(anotherMiddleware())

		vibesGroup.POST("/", vibeHandler.CreateVibeGin)
		vibesGroup.GET("/", vibeHandler.GetAllVibesGin)
		vibesGroup.GET("/stats", vibeHandler.GetVibeStatsGin)
		vibesGroup.GET("/today", vibeHandler.GetTodaysVibeRecommendationGin)
		vibesGroup.GET("/streak", vibeHandler.GetMoodStreakGin)
		vibesGroup.GET("/export", vibeHandler.ExportVibesGin)
		vibesGroup.POST("/bulk", vibeHandler.BulkImportVibesGin)
		vibesGroup.GET("/:id", vibeHandler.GetVibeByIDGin)
		vibesGroup.PUT("/:id", vibeHandler.UpdateVibeGin)
		vibesGroup.DELETE("/:id", vibeHandler.DeleteVibeGin)
	}

	return router
}

// requestIDMiddleware adds a request ID to each request.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set(RequestIDKey, requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// loggingMiddleware logs requests using a structured format.
func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next() // Process request

		// Log details after request has been processed
		end := time.Now()
		latency := end.Sub(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()
		requestID, _ := c.Get(RequestIDKey)

		if raw != "" {
			path = path + "?" + raw
		}

		// Using standard log package for simplicity, can be replaced with zerolog or other structured logger
		log.Printf("[GIN] %s | %3d | %13v | %15s | %s %s | %s | RequestID: %s",
			end.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
			errorMessage,
			requestID,
		)
	}
}

// StartGinServer starts the Gin server.
func StartGinServer(router *gin.Engine, cfg *config.AppConfig) (*http.Server, error) {
	addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
		IdleTimeout:  cfg.ServerIdleTimeout,
	}

	log.Printf("Starting GIN server on %s", addr)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	return srv, nil
}

// ShutdownGinServer gracefully shuts down the Gin server.
func ShutdownGinServer(srv *http.Server, timeout time.Duration) {
	log.Println("Shutting down GIN server...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("GIN server exiting")
}
