package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"code", "method", "path"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests.",
			Buckets: prometheus.DefBuckets, // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
		},
		[]string{"code", "method", "path"},
	)
	// Add more metrics as needed, e.g. active requests, response size
)

// normalizePath attempts to reduce cardinality for path labels.
// Example: /api/v1/vibes/123 -> /api/v1/vibes/:id
// This needs to be adjusted based on actual routing patterns.
func normalizePath(path string, framework string, ctx interface{}) string {
	// Simple normalization for paths with IDs.
	// This is a basic example and might need to be more sophisticated
	// depending on the route structure.
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Example: /api/v1/vibes/{id}
	if len(parts) > 0 && strings.HasPrefix(path, "/api/v1/vibes/") && len(parts) == 4 {
		_, err := strconv.Atoi(parts[3])
		if err == nil {
			return "/" + strings.Join(parts[:3], "/") + "/:id"
		}
	}

	// For Fiber, try to get the matched route pattern if available
	if framework == "fiber" {
		fCtx := ctx.(*fiber.Ctx)
		routePath := fCtx.Route().Path
		if routePath != "" && routePath != "/" { // Avoid using generic "/" if specific route matched
			// Fiber paths might already be in a good format e.g. /api/v1/vibes/:id
			return routePath
		}
	}

	// For Gin, try to get the matched route pattern
	if framework == "gin" {
		gCtx := ctx.(*gin.Context)
		if gCtx.FullPath() != "" && gCtx.FullPath() != "/" {
			// Gin FullPath() usually gives something like /api/v1/vibes/:id
			return gCtx.FullPath()
		}
	}

	// Fallback to the provided path if no specific pattern matched or normalization applied
	return path
}

// MetricsMiddlewareFiber creates a Fiber middleware for collecting Prometheus metrics.
func MetricsMiddlewareFiber() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next() // Execute the next handler in the chain

		statusCode := c.Response().StatusCode()
		if err != nil { // If an error occurred in a subsequent handler that fiber handles
			var fiberError *fiber.Error
			if errors.As(err, &fiberError) {
				statusCode = fiberError.Code
			} else if statusCode == http.StatusOK {
				// If c.Next() returns an error but status code is still 200,
				// it might be an error that Fiber didn't map to a status code,
				// so we set it to 500.
				statusCode = http.StatusInternalServerError
			}
		}

		// Use c.Route().Path for potentially more accurate path templating if configured well.
		// path := c.Route().Path
		// If c.Route().Path is not specific enough (e.g. '/*' for a group), use c.Path()
		// and normalize it.
		path := normalizePath(c.Path(), "fiber", c)

		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(strconv.Itoa(statusCode), c.Method(), path).Inc()
		httpRequestDuration.WithLabelValues(strconv.Itoa(statusCode), c.Method(), path).Observe(duration)

		return err // Return the error so Fiber can handle it
	}
}

// MetricsMiddlewareGin creates a Gin middleware for collecting Prometheus metrics.
func MetricsMiddlewareGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next() // Process request

		statusCode := c.Writer.Status()

		// Use c.FullPath() for Gin, which usually gives the template path like /users/:id
		// path := c.FullPath()
		// If c.FullPath() is empty (e.g. for NoRoute), use c.Request.URL.Path and normalize.
		path := normalizePath(c.Request.URL.Path, "gin", c)
		if c.FullPath() != "" { // Prefer FullPath if available and not just root
			path = c.FullPath()
		}

		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(strconv.Itoa(statusCode), c.Request.Method, path).Inc()
		httpRequestDuration.WithLabelValues(strconv.Itoa(statusCode), c.Request.Method, path).Observe(duration)
	}
}
