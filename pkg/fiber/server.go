package fiber

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	swaggoFiber "github.com/swaggo/fiber-swagger"

	"github.com/user/daily-vibe-tracker/internal/config"
	"github.com/user/daily-vibe-tracker/internal/handler" // Will be created later
	// Import docs for swagger
	_ "github.com/user/daily-vibe-tracker/docs"
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
		AllowOrigins: cfg.CorsAllowedOrigins[0], // Fiber's CORS AllowOrigins is a string, not a slice. Taking the first one.
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Swagger UI
	// Make sure SWAGGER_HOST and SWAGGER_BASE_PATH are set in config.env
	// e.g. SWAGGER_HOST=localhost:8080
	// e.g. SWAGGER_BASE_PATH=/api/v1
	// The URL will be http://localhost:8080/swagger/index.html if base path is /
	app.Get("/swagger/*", swaggoFiber.WrapHandler)


	// Routes
	// Example: app.Get("/", func(c *fiber.Ctx) error {
	// 	return c.SendString("Hello, Fiber World!")
	// })

	// Health Check Route (will be properly defined with a handler later)
	if vibeHandler != nil && vibeHandler.HealthHandler != nil { // Ensure HealthHandler is initialized
		app.Get("/health", vibeHandler.HealthHandler.CheckHealthFiber)
	} else {
		// Fallback if handler not ready (should not happen in final setup)
		app.Get("/health", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status": "initializing",
			})
		})
	}


	// TODO: Add vibe routes here once the handler is more complete
	// api := app.Group("/api/v1")
	// api.Get("/vibes", vibeHandler.GetAllVibesFiber)
	// api.Post("/vibes", vibeHandler.CreateVibeFiber)
	// api.Get("/vibes/:id", vibeHandler.GetVibeByIDFiber)
	// api.Put("/vibes/:id", vibeHandler.UpdateVibeFiber)
	// api.Delete("/vibes/:id", vibeHandler.DeleteVibeFiber)


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
