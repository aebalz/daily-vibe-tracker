package fiber

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	swaggoFiber "github.com/swaggo/fiber-swagger"

	"github.com/aebalz/daily-vibe-tracker/internal/config"
	"github.com/aebalz/daily-vibe-tracker/internal/handler" // Will be created later
	customMiddleware "github.com/aebalz/daily-vibe-tracker/internal/middleware"

	customMiddleware "github.com/aebalz/daily-vibe-tracker/internal/middleware"

	// Import docs for swagger
	_ "github.com/aebalz/daily-vibe-tracker/docs"
	"github.com/gofiber/adaptor/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewFiberServer creates and configures a new Fiber application.
func NewFiberServer(cfg *config.AppConfig, vibeHandler *handler.VibeHandler) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
		IdleTimeout:  cfg.ServerIdleTimeout,
		ErrorHandler: customErrorHandler, // Optional: Custom error handler
	})

	// Middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${ip} ${status} - ${method} ${path} ${latency}\nREQUEST_ID: ${locals:requestid}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CorsAllowedOrigins[0], // Fiber's CORS AllowOrigins is a string. Adjust if multiple needed via other means.
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Add Custom Middleware (Metrics, Rate Limiting)
	// These should come after basic middleware like logger/requestid but before routes.
	app.Use(customMiddleware.MetricsMiddlewareFiber())
	// Apply rate limiter globally or to specific groups/routes as needed
	// Example: Global application (adjust rps and burst as needed)
	// For specific groups: api.Use(customMiddleware.RateLimiterFiber(10, 20))
	app.Use(customMiddleware.RateLimiterFiber(cfg.RateLimitPerSecond, cfg.RateLimitBurst))


	// Swagger UI
	// BasePath for swagger UI itself. If docs.SwaggerInfo.BasePath is /api/v1,
	// then swagger docs will be found relative to that for API calls, but the UI
	// itself is served from /swagger/*
	app.Get("/swagger/*", swaggoFiber.WrapHandler) // Serves Swagger UI

	// Prometheus Metrics Endpoint
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))


	// Routes
	// Health Check Route
	if vibeHandler != nil && vibeHandler.HealthHandler != nil {
		app.Get("/health", vibeHandler.HealthHandler.CheckHealthFiber)
	} else {
		app.Get("/health", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "initializing health handler"})
		})
	}

	// Vibe Routes
	apiV1 := app.Group("/api/v1") // All vibe routes will be under /api/v1
	{
		vibesGroup := apiV1.Group("/vibes")
		// Apply specific middleware to this group if needed
		// vibesGroup.Use(customMiddleware.AnotherSpecificMiddleware())

		vibesGroup.Post("/", vibeHandler.CreateVibeFiber)
		vibesGroup.Get("/", vibeHandler.GetAllVibesFiber)
		vibesGroup.Get("/stats", vibeHandler.GetVibeStatsFiber)
		vibesGroup.Get("/today", vibeHandler.GetTodaysVibeRecommendationFiber)
		vibesGroup.Get("/streak", vibeHandler.GetMoodStreakFiber)
		vibesGroup.Get("/export", vibeHandler.ExportVibesFiber)
		vibesGroup.Post("/bulk", vibeHandler.BulkImportVibesFiber)
		vibesGroup.Get("/:id", vibeHandler.GetVibeByIDFiber)
		vibesGroup.Put("/:id", vibeHandler.UpdateVibeFiber)
		vibesGroup.Delete("/:id", vibeHandler.DeleteVibeFiber)
	}


	return app
}

// customErrorHandler for Fiber
func customErrorHandler(ctx *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log the error internally
	log.Printf("Fiber Error: %v - Path: %s", err, ctx.Path())

	return ctx.Status(code).JSON(fiber.Map{
		"error":   true,
		"message": message,
	})
}

// StartFiberServer starts the Fiber server.
func StartFiberServer(app *fiber.App, cfg *config.AppConfig) error {
	addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	log.Printf("Starting Fiber server on %s", addr)
	return app.Listen(addr)
}
