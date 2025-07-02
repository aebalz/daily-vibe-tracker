package middleware

// Logging middleware placeholder.
// Actual implementation is in pkg/fiber/server.go and pkg/gin/server.go for now.

// Example for Fiber (if refactored here):
/*
import "github.com/gofiber/fiber/v2"
import "github.com/gofiber/fiber/v2/middleware/logger"

func FiberLogger() fiber.Handler {
	return logger.New(logger.Config{
		Format: "[${time}] ${ip} ${status} - ${method} ${path} ${latency}\nREQUEST_ID: ${locals:requestid}\n",
	})
}
*/

// Example for Gin (if refactored here):
/*
import (
	"log"
	"time"
	"github.com/gin-gonic/gin"
)

const GinRequestIDKey = "requestID" // Assuming this constant would be shared or defined centrally

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next() // Process request

		end := time.Now()
		latency := end.Sub(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()
		requestID, _ := c.Get(GinRequestIDKey)

		if raw != "" {
			path = path + "?" + raw
		}

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
*/
