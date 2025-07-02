package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aebalz/daily-vibe-tracker/internal/model"
	"github.com/aebalz/daily-vibe-tracker/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// VibeHandler encapsulates all handlers for the application.
type VibeHandler struct {
	Service       service.VibeServiceInterface
	HealthHandler *HealthHandler
}

// NewVibeHandler creates a new VibeHandler.
// func NewVibeHandler(svc service.VibeServiceInterface, healthHandler *HealthHandler) *VibeHandler {
// 	return &VibeHandler{
// 		Service:       svc,
// 		HealthHandler: healthHandler,
// 	}
// }

// --- Helper for error responses ---
func handleError(framework string, ctx interface{}, code int, message string, err error) error {
	fullMessage := message
	if err != nil {
		fullMessage = fmt.Sprintf("%s: %v", message, err)
	}
	if framework == "gin" {
		c := ctx.(*gin.Context)
		c.JSON(code, gin.H{"error": fullMessage})
		return nil // Gin handles the response
	}
	// framework == "fiber"
	c := ctx.(*fiber.Ctx)
	return c.Status(code).JSON(fiber.Map{"error": fullMessage})
}

// --- Request/Response Structs (examples, can be more specific) ---

// CreateVibeRequest defines the expected body for creating a vibe.
// The model.Vibe can often be used directly if validation tags are sufficient.
type CreateVibeRequest struct {
	Date        time.Time `json:"date" binding:"required"`
	Mood        string    `json:"mood" binding:"required"`
	EnergyLevel int       `json:"energy_level" binding:"required,min=1,max=10"`
	Notes       string    `json:"notes"`
	Activities  []string  `json:"activities"`
}

// UpdateVibeRequest defines the expected body for updating a vibe.
type UpdateVibeRequest struct {
	Date        time.Time `json:"date"` // Usually not updatable or handled carefully
	Mood        string    `json:"mood"`
	EnergyLevel int       `json:"energy_level" binding:"omitempty,min=1,max=10"`
	Notes       string    `json:"notes"`
	Activities  []string  `json:"activities"`
}

// PaginatedVibesResponse is a generic structure for paginated vibe lists.
type PaginatedVibesResponse struct {
	Data       []model.Vibe `json:"data"`
	Total      int64        `json:"total"`
	Limit      int          `json:"limit"`
	Offset     int          `json:"offset"`
	Page       int          `json:"page"`
	TotalPages int          `json:"total_pages"`
}

// --- Fiber Handlers ---

// CreateVibeFiber godoc
// @Summary Record a daily vibe
// @Description Adds a new daily vibe entry to the tracker.
// @Tags vibes
// @Accept json
// @Produce json
// @Param vibe body model.Vibe true "Vibe to add"
// @Success 201 {object} model.Vibe "Created vibe with ID"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes [post]
func (vh *VibeHandler) CreateVibeFiber(c *fiber.Ctx) error {
	var req model.Vibe // Using model.Vibe directly for simplicity
	if err := c.BodyParser(&req); err != nil {
		return handleError("fiber", c, http.StatusBadRequest, "Invalid request body", err)
	}

	// Basic validation (can be enhanced with a validator library)
	if req.Date.IsZero() || req.Mood == "" || req.EnergyLevel < 1 || req.EnergyLevel > 10 {
		return handleError("fiber", c, http.StatusBadRequest, "Missing required fields or invalid energy level", nil)
	}

	createdVibe, err := vh.Service.CreateVibe(&req)
	if err != nil {
		// Check for specific errors, e.g., duplicate date if unique constraint is violated
		// For now, a generic 500, but could be 409 Conflict etc.
		return handleError("fiber", c, http.StatusInternalServerError, "Failed to create vibe", err)
	}
	return c.Status(http.StatusCreated).JSON(createdVibe)
}

// GetAllVibesFiber godoc
// @Summary Get vibes with filters
// @Description Retrieves a list of vibes, with optional filtering, pagination, and sorting.
// @Tags vibes
// @Accept json
// @Produce json
// @Param date query string false "Filter by date (YYYY-MM-DD)"
// @Param mood query string false "Filter by mood"
// @Param limit query int false "Pagination limit" default(10)
// @Param offset query int false "Pagination offset" default(0)
// @Param sort_by query string false "Field to sort by (e.g., date, mood, energy_level)" default(date)
// @Param sort_order query string false "Sort order (asc, desc)" default(desc)
// @Success 200 {object} PaginatedVibesResponse "List of vibes with pagination"
// @Failure 400 {object} map[string]string "Invalid query parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes [get]
func (vh *VibeHandler) GetAllVibesFiber(c *fiber.Ctx) error {
	filters := make(map[string]interface{})
	if dateStr := c.Query("date"); dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return handleError("fiber", c, http.StatusBadRequest, "Invalid date format for 'date' query parameter. Use YYYY-MM-DD.", err)
		}
		filters["date"] = parsedDate.Format("2006-01-02") // Service/Repo expects string YYYY-MM-DD for DATE() comparison
	}
	if mood := c.Query("mood"); mood != "" {
		filters["mood"] = mood
	}

	limit, _ := strconv.Atoi(c.Query("limit", strconv.Itoa(service.DefaultLimit)))
	offset, _ := strconv.Atoi(c.Query("offset", strconv.Itoa(service.DefaultOffset)))
	sortBy := c.Query("sort_by", service.DefaultSortBy)
	sortOrder := c.Query("sort_order", service.DefaultSortOrder)


	vibes, total, err := vh.Service.GetAllVibes(filters, limit, offset, sortBy, sortOrder)
	if err != nil {
		return handleError("fiber", c, http.StatusInternalServerError, "Failed to retrieve vibes", err)
	}

	page := 0
	if limit > 0 {
		page = (offset / limit) + 1
	}
	totalPages := 0
	if limit > 0 && total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit)) // Ceiling division
	}


	return c.JSON(PaginatedVibesResponse{
		Data:   vibes,
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Page: page,
		TotalPages: totalPages,
	})
}

// GetVibeByIDFiber godoc
// @Summary Get specific vibe
// @Description Retrieves details of a single vibe by its ID.
// @Tags vibes
// @Accept json
// @Produce json
// @Param id path int true "Vibe ID"
// @Success 200 {object} model.Vibe "Single vibe details"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Vibe not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/{id} [get]
func (vh *VibeHandler) GetVibeByIDFiber(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return handleError("fiber", c, http.StatusBadRequest, "Invalid vibe ID", err)
	}

	vibe, err := vh.Service.GetVibeByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return handleError("fiber", c, http.StatusNotFound, "Vibe not found", nil)
		}
		return handleError("fiber", c, http.StatusInternalServerError, "Failed to retrieve vibe", err)
	}
	return c.JSON(vibe)
}

// UpdateVibeFiber godoc
// @Summary Update vibe
// @Description Modifies an existing vibe entry.
// @Tags vibes
// @Accept json
// @Produce json
// @Param id path int true "Vibe ID"
// @Param vibe body UpdateVibeRequest true "Updated vibe data"
// @Success 200 {object} model.Vibe "Updated vibe"
// @Failure 400 {object} map[string]string "Invalid input or ID format"
// @Failure 404 {object} map[string]string "Vibe not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/{id} [put]
func (vh *VibeHandler) UpdateVibeFiber(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return handleError("fiber", c, http.StatusBadRequest, "Invalid vibe ID", err)
	}

	var req UpdateVibeRequest // Use specific update request struct
	if err := c.BodyParser(&req); err != nil {
		return handleError("fiber", c, http.StatusBadRequest, "Invalid request body", err)
	}

	// Map UpdateVibeRequest to model.Vibe for service layer
	// Note: This is a partial update. The service/repo layer needs to handle this correctly.
	// It's often better to fetch the existing record and then apply changes.
	// For simplicity, we pass what's given. The service layer's ValidateVibe will check.
	// Service's UpdateVibe should handle fetching existing and merging.
	vibeToUpdate := model.Vibe{
		// Date: req.Date, // Date update needs careful consideration
		Mood:        req.Mood,
		EnergyLevel: req.EnergyLevel,
		Notes:       req.Notes,
		Activities:  req.Activities,
	}
	// If a field is optional and not provided, it might be zero-valued.
	// GORM's `Updates` method handles non-zero fields, or use `Select` for explicit fields.
	// The service layer's `ValidateVibe` will run on this partial data.

	updatedVibe, err := vh.Service.UpdateVibe(uint(id), &vibeToUpdate)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return handleError("fiber", c, http.StatusNotFound, "Vibe not found to update", nil)
		}
		return handleError("fiber", c, http.StatusInternalServerError, "Failed to update vibe", err)
	}
	return c.JSON(updatedVibe)
}

// DeleteVibeFiber godoc
// @Summary Delete vibe
// @Description Removes a vibe entry from the tracker.
// @Tags vibes
// @Accept json
// @Produce json
// @Param id path int true "Vibe ID"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Vibe not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/{id} [delete]
func (vh *VibeHandler) DeleteVibeFiber(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil || id <= 0 {
		return handleError("fiber", c, http.StatusBadRequest, "Invalid vibe ID", err)
	}

	err = vh.Service.DeleteVibe(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return handleError("fiber", c, http.StatusNotFound, "Vibe not found to delete", nil)
		}
		return handleError("fiber", c, http.StatusInternalServerError, "Failed to delete vibe", err)
	}
	return c.Status(http.StatusOK).JSON(fiber.Map{"message": "Vibe deleted successfully"})
}

// GetVibeStatsFiber godoc
// @Summary Get vibe statistics
// @Description Retrieves statistics about vibes, such as mood distribution and average energy.
// @Tags vibes-analytics
// @Accept json
// @Produce json
// @Param period query string false "Time period for statistics (week, month, year)" default(month)
// @Success 200 {object} map[string]interface{} "Vibe statistics"
// @Failure 400 {object} map[string]string "Invalid period parameter"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/stats [get]
func (vh *VibeHandler) GetVibeStatsFiber(c *fiber.Ctx) error {
	period := c.Query("period", "month") // Default to month
	validPeriods := map[string]bool{"week": true, "month": true, "year": true}
	if !validPeriods[strings.ToLower(period)] {
		return handleError("fiber", c, http.StatusBadRequest, "Invalid period. Allowed values: week, month, year.", nil)
	}

	stats, err := vh.Service.GetVibeStatistics(period)
	if err != nil {
		return handleError("fiber", c, http.StatusInternalServerError, "Failed to retrieve vibe statistics", err)
	}
	return c.JSON(stats)
}

// GetTodaysVibeRecommendationFiber godoc
// @Summary Get today's vibe recommendation
// @Description Suggests activities based on historical vibe data.
// @Tags vibes-analytics
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Suggested activities and reason"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/today [get]
func (vh *VibeHandler) GetTodaysVibeRecommendationFiber(c *fiber.Ctx) error {
	recommendation, err := vh.Service.GetTodaysVibeRecommendation()
	if err != nil {
		return handleError("fiber", c, http.StatusInternalServerError, "Failed to generate recommendation", err)
	}
	return c.JSON(recommendation)
}

// GetMoodStreakFiber godoc
// @Summary Get current mood streak
// @Description Calculates the current and longest streak for a specific mood.
// @Tags vibes-analytics
// @Accept json
// @Produce json
// @Param mood query string true "Mood to calculate streak for"
// @Success 200 {object} map[string]interface{} "Streak information (current_streak, longest_streak)"
// @Failure 400 {object} map[string]string "Missing mood parameter"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/streak [get]
func (vh *VibeHandler) GetMoodStreakFiber(c *fiber.Ctx) error {
	mood := c.Query("mood")
	if mood == "" {
		return handleError("fiber", c, http.StatusBadRequest, "Missing 'mood' query parameter", nil)
	}

	streakInfo, err := vh.Service.GetMoodStreak(mood)
	if err != nil {
		return handleError("fiber", c, http.StatusInternalServerError, "Failed to calculate mood streak", err)
	}
	return c.JSON(streakInfo)
}

// ExportVibesFiber godoc
// @Summary Export vibes data
// @Description Exports vibe data in CSV or JSON format.
// @Tags vibes-advanced
// @Produce plain text/csv application/json
// @Param format query string true "Export format (csv or json)"
// @Param date query string false "Filter by date (YYYY-MM-DD)"
// @Param mood query string false "Filter by mood"
// @Param sort_by query string false "Field to sort by (e.g., date, mood, energy_level)" default(date)
// @Param sort_order query string false "Sort order (asc, desc)" default(asc)
// @Success 200 {file} string "Vibe data in specified format"
// @Failure 400 {object} map[string]string "Invalid parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/export [get]
func (vh *VibeHandler) ExportVibesFiber(c *fiber.Ctx) error {
	format := c.Query("format")
	if format == "" {
		return handleError("fiber", c, http.StatusBadRequest, "Missing 'format' query parameter (csv or json)", nil)
	}
	format = strings.ToLower(format)
	if format != "csv" && format != "json" {
		return handleError("fiber", c, http.StatusBadRequest, "Invalid 'format'. Must be 'csv' or 'json'", nil)
	}


	filters := make(map[string]interface{})
	if dateStr := c.Query("date"); dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return handleError("fiber", c, http.StatusBadRequest, "Invalid date format for 'date' query parameter. Use YYYY-MM-DD.", err)
		}
		filters["date"] = parsedDate.Format("2006-01-02")
	}
	if mood := c.Query("mood"); mood != "" {
		filters["mood"] = mood
	}
	sortBy := c.Query("sort_by", service.DefaultSortBy) // Default sort for export might be different
	sortOrder := c.Query("sort_order", "asc") // Default to ascending for exports usually


	data, contentType, err := vh.Service.ExportVibes(filters, format, sortBy, sortOrder)
	if err != nil {
		return handleError("fiber", c, http.StatusInternalServerError, "Failed to export vibes", err)
	}

	c.Set(fiber.HeaderContentType, contentType)
	if format == "csv" {
		c.Set(fiber.HeaderContentDisposition, `attachment; filename="vibes_export.csv"`)
	} else if format == "json" {
		c.Set(fiber.HeaderContentDisposition, `attachment; filename="vibes_export.json"`)
	}
	return c.Send(data)
}

// BulkImportVibesFiber godoc
// @Summary Bulk import vibes
// @Description Imports multiple vibe entries from a JSON array.
// @Tags vibes-advanced
// @Accept json
// @Produce json
// @Param vibes body []model.Vibe true "Array of vibes to import"
// @Success 201 {object} map[string]interface{} "Number of vibes imported"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 500 {object} map[string]string "Internal server error during import"
// @Router /api/v1/vibes/bulk [post]
func (vh *VibeHandler) BulkImportVibesFiber(c *fiber.Ctx) error {
	var vibesToImport []*model.Vibe
	if err := c.BodyParser(&vibesToImport); err != nil {
		return handleError("fiber", c, http.StatusBadRequest, "Invalid request body for bulk import", err)
	}

	if len(vibesToImport) == 0 {
		return handleError("fiber", c, http.StatusBadRequest, "No vibes provided in the request body", nil)
	}

	count, err := vh.Service.BulkImportVibes(vibesToImport)
	if err != nil {
		// This could be a mix of validation errors or DB errors.
		// A more sophisticated error handling might return per-item status.
		return handleError("fiber", c, http.StatusInternalServerError, "Failed during bulk import", err)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message":        fmt.Sprintf("%d vibes imported successfully", count),
		"imported_count": count,
	})
}

// --- Gin Handlers ---

// CreateVibeGin godoc
// @Summary Record a daily vibe
// @Description Adds a new daily vibe entry to the tracker.
// @Tags vibes
// @Accept json
// @Produce json
// @Param vibe body model.Vibe true "Vibe to add"
// @Success 201 {object} model.Vibe "Created vibe with ID"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes [post]
func (vh *VibeHandler) CreateVibeGin(c *gin.Context) {
	var req model.Vibe
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError("gin", c, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	if req.Date.IsZero() || req.Mood == "" || req.EnergyLevel < 1 || req.EnergyLevel > 10 {
		handleError("gin", c, http.StatusBadRequest, "Missing required fields or invalid energy level", nil)
		return
	}

	createdVibe, err := vh.Service.CreateVibe(&req)
	if err != nil {
		handleError("gin", c, http.StatusInternalServerError, "Failed to create vibe", err)
		return
	}
	c.JSON(http.StatusCreated, createdVibe)
}

// GetAllVibesGin godoc
// @Summary Get vibes with filters
// @Description Retrieves a list of vibes, with optional filtering, pagination, and sorting.
// @Tags vibes
// @Accept json
// @Produce json
// @Param date query string false "Filter by date (YYYY-MM-DD)"
// @Param mood query string false "Filter by mood"
// @Param limit query int false "Pagination limit" default(10)
// @Param offset query int false "Pagination offset" default(0)
// @Param sort_by query string false "Field to sort by (e.g., date, mood, energy_level)" default(date)
// @Param sort_order query string false "Sort order (asc, desc)" default(desc)
// @Success 200 {object} PaginatedVibesResponse "List of vibes with pagination"
// @Failure 400 {object} map[string]string "Invalid query parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes [get]
func (vh *VibeHandler) GetAllVibesGin(c *gin.Context) {
	filters := make(map[string]interface{})
	if dateStr := c.Query("date"); dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			handleError("gin", c, http.StatusBadRequest, "Invalid date format for 'date' query parameter. Use YYYY-MM-DD.", err)
			return
		}
		filters["date"] = parsedDate.Format("2006-01-02")
	}
	if mood := c.Query("mood"); mood != "" {
		filters["mood"] = mood
	}

	limitStr := c.DefaultQuery("limit", strconv.Itoa(service.DefaultLimit))
	limit, errL := strconv.Atoi(limitStr)
	if errL != nil {
		handleError("gin", c, http.StatusBadRequest, "Invalid limit parameter", errL)
		return
	}

	offsetStr := c.DefaultQuery("offset", strconv.Itoa(service.DefaultOffset))
	offset, errO := strconv.Atoi(offsetStr)
	if errO != nil {
		handleError("gin", c, http.StatusBadRequest, "Invalid offset parameter", errO)
		return
	}
	sortBy := c.DefaultQuery("sort_by", service.DefaultSortBy)
	sortOrder := c.DefaultQuery("sort_order", service.DefaultSortOrder)

	vibes, total, err := vh.Service.GetAllVibes(filters, limit, offset, sortBy, sortOrder)
	if err != nil {
		handleError("gin", c, http.StatusInternalServerError, "Failed to retrieve vibes", err)
		return
	}

	page := 0
	if limit > 0 {
		page = (offset / limit) + 1
	}
	totalPages := 0
	if limit > 0 && total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit)) // Ceiling division
	}

	c.JSON(http.StatusOK, PaginatedVibesResponse{
		Data:   vibes,
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Page: page,
		TotalPages: totalPages,
	})
}

// GetVibeByIDGin godoc
// @Summary Get specific vibe
// @Description Retrieves details of a single vibe by its ID.
// @Tags vibes
// @Accept json
// @Produce json
// @Param id path int true "Vibe ID"
// @Success 200 {object} model.Vibe "Single vibe details"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Vibe not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/{id} [get]
func (vh *VibeHandler) GetVibeByIDGin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		handleError("gin", c, http.StatusBadRequest, "Invalid vibe ID", err)
		return
	}

	vibe, err := vh.Service.GetVibeByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleError("gin", c, http.StatusNotFound, "Vibe not found", nil)
			return
		}
		handleError("gin", c, http.StatusInternalServerError, "Failed to retrieve vibe", err)
		return
	}
	c.JSON(http.StatusOK, vibe)
}

// UpdateVibeGin godoc
// @Summary Update vibe
// @Description Modifies an existing vibe entry.
// @Tags vibes
// @Accept json
// @Produce json
// @Param id path int true "Vibe ID"
// @Param vibe body UpdateVibeRequest true "Updated vibe data"
// @Success 200 {object} model.Vibe "Updated vibe"
// @Failure 400 {object} map[string]string "Invalid input or ID format"
// @Failure 404 {object} map[string]string "Vibe not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/{id} [put]
func (vh *VibeHandler) UpdateVibeGin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		handleError("gin", c, http.StatusBadRequest, "Invalid vibe ID", err)
		return
	}

	var req UpdateVibeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleError("gin", c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	vibeToUpdate := model.Vibe{
		Mood:        req.Mood,
		EnergyLevel: req.EnergyLevel,
		Notes:       req.Notes,
		Activities:  req.Activities,
	}

	updatedVibe, err := vh.Service.UpdateVibe(uint(id), &vibeToUpdate)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleError("gin", c, http.StatusNotFound, "Vibe not found to update", nil)
			return
		}
		handleError("gin", c, http.StatusInternalServerError, "Failed to update vibe", err)
		return
	}
	c.JSON(http.StatusOK, updatedVibe)
}

// DeleteVibeGin godoc
// @Summary Delete vibe
// @Description Removes a vibe entry from the tracker.
// @Tags vibes
// @Accept json
// @Produce json
// @Param id path int true "Vibe ID"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid ID format"
// @Failure 404 {object} map[string]string "Vibe not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/{id} [delete]
func (vh *VibeHandler) DeleteVibeGin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		handleError("gin", c, http.StatusBadRequest, "Invalid vibe ID", err)
		return
	}

	err = vh.Service.DeleteVibe(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			handleError("gin", c, http.StatusNotFound, "Vibe not found to delete", nil)
			return
		}
		handleError("gin", c, http.StatusInternalServerError, "Failed to delete vibe", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Vibe deleted successfully"})
}

// GetVibeStatsGin godoc
// @Summary Get vibe statistics
// @Description Retrieves statistics about vibes, such as mood distribution and average energy.
// @Tags vibes-analytics
// @Accept json
// @Produce json
// @Param period query string false "Time period for statistics (week, month, year)" default(month)
// @Success 200 {object} map[string]interface{} "Vibe statistics"
// @Failure 400 {object} map[string]string "Invalid period parameter"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/stats [get]
func (vh *VibeHandler) GetVibeStatsGin(c *gin.Context) {
	period := c.DefaultQuery("period", "month")
	validPeriods := map[string]bool{"week": true, "month": true, "year": true}
	if !validPeriods[strings.ToLower(period)] {
		handleError("gin", c, http.StatusBadRequest, "Invalid period. Allowed values: week, month, year.", nil)
		return
	}

	stats, err := vh.Service.GetVibeStatistics(period)
	if err != nil {
		handleError("gin", c, http.StatusInternalServerError, "Failed to retrieve vibe statistics", err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetTodaysVibeRecommendationGin godoc
// @Summary Get today's vibe recommendation
// @Description Suggests activities based on historical vibe data.
// @Tags vibes-analytics
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Suggested activities and reason"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/today [get]
func (vh *VibeHandler) GetTodaysVibeRecommendationGin(c *gin.Context) {
	recommendation, err := vh.Service.GetTodaysVibeRecommendation()
	if err != nil {
		handleError("gin", c, http.StatusInternalServerError, "Failed to generate recommendation", err)
		return
	}
	c.JSON(http.StatusOK, recommendation)
}

// GetMoodStreakGin godoc
// @Summary Get current mood streak
// @Description Calculates the current and longest streak for a specific mood.
// @Tags vibes-analytics
// @Accept json
// @Produce json
// @Param mood query string true "Mood to calculate streak for"
// @Success 200 {object} map[string]interface{} "Streak information (current_streak, longest_streak)"
// @Failure 400 {object} map[string]string "Missing mood parameter"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/streak [get]
func (vh *VibeHandler) GetMoodStreakGin(c *gin.Context) {
	mood := c.Query("mood")
	if mood == "" {
		handleError("gin", c, http.StatusBadRequest, "Missing 'mood' query parameter", nil)
		return
	}

	streakInfo, err := vh.Service.GetMoodStreak(mood)
	if err != nil {
		handleError("gin", c, http.StatusInternalServerError, "Failed to calculate mood streak", err)
		return
	}
	c.JSON(http.StatusOK, streakInfo)
}

// ExportVibesGin godoc
// @Summary Export vibes data
// @Description Exports vibe data in CSV or JSON format.
// @Tags vibes-advanced
// @Produce plain text/csv application/json
// @Param format query string true "Export format (csv or json)"
// @Param date query string false "Filter by date (YYYY-MM-DD)"
// @Param mood query string false "Filter by mood"
// @Param sort_by query string false "Field to sort by (e.g., date, mood, energy_level)" default(date)
// @Param sort_order query string false "Sort order (asc, desc)" default(asc)
// @Success 200 {file} string "Vibe data in specified format"
// @Failure 400 {object} map[string]string "Invalid parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/vibes/export [get]
func (vh *VibeHandler) ExportVibesGin(c *gin.Context) {
	format := c.Query("format")
	if format == "" {
		handleError("gin", c, http.StatusBadRequest, "Missing 'format' query parameter (csv or json)", nil)
		return
	}
	format = strings.ToLower(format)
	if format != "csv" && format != "json" {
		handleError("gin", c, http.StatusBadRequest, "Invalid 'format'. Must be 'csv' or 'json'", nil)
		return
	}

	filters := make(map[string]interface{})
	if dateStr := c.Query("date"); dateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			handleError("gin", c, http.StatusBadRequest, "Invalid date format for 'date' query parameter. Use YYYY-MM-DD.", err)
			return
		}
		filters["date"] = parsedDate.Format("2006-01-02")
	}
	if mood := c.Query("mood"); mood != "" {
		filters["mood"] = mood
	}
	sortBy := c.DefaultQuery("sort_by", service.DefaultSortBy)
	sortOrder := c.DefaultQuery("sort_order", "asc")


	data, contentType, err := vh.Service.ExportVibes(filters, format, sortBy, sortOrder)
	if err != nil {
		handleError("gin", c, http.StatusInternalServerError, "Failed to export vibes", err)
		return
	}

	c.Header("Content-Type", contentType)
	if format == "csv" {
		c.Header("Content-Disposition", `attachment; filename="vibes_export.csv"`)
	} else if format == "json" {
		c.Header("Content-Disposition", `attachment; filename="vibes_export.json"`)
	}
	c.Data(http.StatusOK, contentType, data)
}

// BulkImportVibesGin godoc
// @Summary Bulk import vibes
// @Description Imports multiple vibe entries from a JSON array.
// @Tags vibes-advanced
// @Accept json
// @Produce json
// @Param vibes body []model.Vibe true "Array of vibes to import"
// @Success 201 {object} map[string]interface{} "Number of vibes imported"
// @Failure 400 {object} map[string]string "Invalid input"
// @Failure 500 {object} map[string]string "Internal server error during import"
// @Router /api/v1/vibes/bulk [post]
func (vh *VibeHandler) BulkImportVibesGin(c *gin.Context) {
	var vibesToImport []*model.Vibe
	if err := c.ShouldBindJSON(&vibesToImport); err != nil {
		handleError("gin", c, http.StatusBadRequest, "Invalid request body for bulk import", err)
		return
	}

	if len(vibesToImport) == 0 {
		handleError("gin", c, http.StatusBadRequest, "No vibes provided in the request body", nil)
		return
	}

	count, err := vh.Service.BulkImportVibes(vibesToImport)
	if err != nil {
		handleError("gin", c, http.StatusInternalServerError, "Failed during bulk import", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        fmt.Sprintf("%d vibes imported successfully", count),
		"imported_count": count,
	})
}
