package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/aebalz/daily-vibe-tracker/pkg/database"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	DB *gorm.DB
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{DB: db}
}

// HealthCheckResponse defines the structure for the health check response.
type HealthCheckResponse struct {
	ServerStatus   string `json:"server_status"`
	DatabaseStatus string `json:"database_status"`
	Timestamp      string `json:"timestamp"`
}

// @Summary API Health Check
// @Description Check the health of the API and database connection.
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} HealthCheckResponse "Successfully checked health"
// @Failure 503 {object} HealthCheckResponse "Service unavailable if database ping fails"
// @Router /health [get]
// CheckHealthFiber is the health check endpoint handler for Fiber.
func (h *HealthHandler) CheckHealthFiber(c *fiber.Ctx) error {
	response := HealthCheckResponse{
		ServerStatus: "OK",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	err := database.PingDB(h.DB)
	if err != nil {
		response.DatabaseStatus = "Error: " + err.Error()
		return c.Status(fiber.StatusServiceUnavailable).JSON(response)
	}
	response.DatabaseStatus = "OK"
	return c.Status(fiber.StatusOK).JSON(response)
}

// CheckHealthGin is the health check endpoint handler for Gin.
// Swaggo annotations are typically placed above the general method or main handler registration,
// so the one above CheckHealthFiber will cover this too.
func (h *HealthHandler) CheckHealthGin(c *gin.Context) {
	response := HealthCheckResponse{
		ServerStatus: "OK",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	err := database.PingDB(h.DB)
	if err != nil {
		response.DatabaseStatus = "Error: " + err.Error()
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}
	response.DatabaseStatus = "OK"
	c.JSON(http.StatusOK, response)
}
