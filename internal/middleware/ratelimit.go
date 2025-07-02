package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/time/rate"
)

// IPMeta stores the limiter and last seen time for an IP
type IPMeta struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	mu      sync.Mutex
	clients = make(map[string]*IPMeta)
)

// Cleanup visitors every minute
func init() {
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
}

func getVisitor(ip string, r rate.Limit, b int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	client, exists := clients[ip]
	if !exists {
		limiter := rate.NewLimiter(r, b)
		clients[ip] = &IPMeta{limiter, time.Now()}
		return limiter
	}

	client.lastSeen = time.Now()
	return client.limiter
}

// RateLimiterFiber creates a Fiber middleware for rate limiting.
// It uses a token bucket algorithm based on IP address.
func RateLimiterFiber(requestsPerSecond float64, burst int) fiber.Handler {
	r := rate.Limit(requestsPerSecond)
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		limiter := getVisitor(ip, r, burst)

		if !limiter.Allow() {
			// Adding a Retry-After header (optional, but good practice)
			// This is a simplified calculation; a more accurate one might be needed
			// depending on the rate.Limiter's state.
			c.Set("Retry-After", "60") // Suggest retrying after 60 seconds
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests. Please try again later.",
			})
		}
		return c.Next()
	}
}

// RateLimiterGin creates a Gin middleware for rate limiting.
func RateLimiterGin(requestsPerSecond float64, burst int) gin.HandlerFunc {
	r := rate.Limit(requestsPerSecond)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := getVisitor(ip, r, burst)

		if !limiter.Allow() {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please try again later.",
			})
			return
		}
		c.Next()
	}
}
