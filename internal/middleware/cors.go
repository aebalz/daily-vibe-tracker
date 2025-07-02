package middleware

// CORS middleware placeholder.
// Actual implementation is in pkg/fiber/server.go and pkg/gin/server.go for now.

// Example for Fiber (if refactored here):
/*
import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/aebalz/daily-vibe-tracker/internal/config" // Assuming config is accessible
)

func FiberCORS(cfg *config.AppConfig) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: cfg.CorsAllowedOrigins[0],
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	})
}
*/

// Example for Gin (if refactored here):
/*
import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/aebalz/daily-vibe-tracker/internal/config" // Assuming config is accessible
)

func GinCORS(cfg *config.AppConfig) gin.HandlerFunc {
	corsConfig := cors.DefaultConfig()
	if len(cfg.CorsAllowedOrigins) == 1 && cfg.CorsAllowedOrigins[0] == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = cfg.CorsAllowedOrigins
	}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	corsConfig.AllowMethods = []string{"GET", "POST, PUT, DELETE, OPTIONS"}
	return cors.New(corsConfig)
}
*/
