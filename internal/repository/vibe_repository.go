package repository

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aebalz/daily-vibe-tracker/internal/model"
	"gorm.io/gorm"
)

// VibeRepositoryInterface defines the interface for vibe repository operations.
type VibeRepositoryInterface interface {
	CreateVibe(vibe *model.Vibe) (*model.Vibe, error)
	GetVibeByID(id uint) (*model.Vibe, error)
	GetAllVibes(filters map[string]interface{}, limit, offset int, sortBy, sortOrder string) ([]model.Vibe, int64, error)
	UpdateVibe(id uint, updatedVibe *model.Vibe) (*model.Vibe, error)
	DeleteVibe(id uint) error

	// Analytics
	GetVibeStatistics(period string, startDate, endDate time.Time) (map[string]interface{}, error)
	GetVibesForDateRange(startDate, endDate time.Time) ([]model.Vibe, error)
	GetMoodStreak(mood string, checkCurrent bool) (int, error) // Simplified for now, not user-specific

	// Bulk and Export
	BulkInsertVibes(vibes []*model.Vibe) (int64, error)
	ExportVibes(filters map[string]interface{}, format string, sortBy, sortOrder string) ([]byte, string, error)
}

// VibeRepository implements VibeRepositoryInterface.
type VibeRepository struct {
	DB *gorm.DB
}

// NewVibeRepository creates a new VibeRepository.
func NewVibeRepository(db *gorm.DB) VibeRepositoryInterface {
	return &VibeRepository{DB: db}
}

// CreateVibe adds a new vibe to the database.
func (r *VibeRepository) CreateVibe(vibe *model.Vibe) (*model.Vibe, error) {
	result := r.DB.Create(vibe)
	if result.Error != nil {
		return nil, result.Error
	}
	return vibe, nil
}

// GetVibeByID retrieves a single vibe by its ID.
func (r *VibeRepository) GetVibeByID(id uint) (*model.Vibe, error) {
	var vibe model.Vibe
	result := r.DB.First(&vibe, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &vibe, nil
}

// GetAllVibes retrieves vibes with optional filters, pagination, and sorting.
func (r *VibeRepository) GetAllVibes(filters map[string]interface{}, limit, offset int, sortBy, sortOrder string) ([]model.Vibe, int64, error) {
	var vibes []model.Vibe
	var totalCount int64

	query := r.DB.Model(&model.Vibe{})

	// Apply filters
	if date, ok := filters["date"]; ok {
		query = query.Where("DATE(date) = ?", date)
	}
	if mood, ok := filters["mood"]; ok {
		query = query.Where("mood = ?", mood)
	}
	// Add more filters as needed, e.g., energy_level

	// Get total count before pagination
	err := query.Count(&totalCount).Error
	if err != nil {
		return nil, 0, err
	}

	// Apply sorting
	if sortBy != "" && sortOrder != "" {
		orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)
		query = query.Order(orderClause)
	} else {
		query = query.Order("date DESC") // Default sort
	}

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	result := query.Find(&vibes)
	if result.Error != nil {
		return nil, 0, result.Error
	}
	return vibes, totalCount, nil
}

// UpdateVibe modifies an existing vibe in the database.
func (r *VibeRepository) UpdateVibe(id uint, updatedVibe *model.Vibe) (*model.Vibe, error) {
	var existingVibe model.Vibe
	if err := r.DB.First(&existingVibe, id).Error; err != nil {
		return nil, err // Vibe not found
	}

	// GORM's Updates method only updates non-zero fields.
	// To update specific fields including clearing some, use Select or a map.
	// For simplicity here, we assume updatedVibe contains all fields to be set.
	// For more granular updates, consider using `r.DB.Model(&existingVibe).Select("field1", "field2").Updates(map[string]interface{}{...})`
	// or `r.DB.Model(&existingVibe).Updates(updatedVibe)` if all fields in updatedVibe are intended for update.
	// Let's ensure the ID is not changed and CreatedAt is preserved.
	updatedVibe.ID = id
	updatedVibe.CreatedAt = existingVibe.CreatedAt

	result := r.DB.Save(updatedVibe)
	if result.Error != nil {
		return nil, result.Error
	}
	return updatedVibe, nil
}

// DeleteVibe removes a vibe from the database (soft delete if gorm.DeletedAt is used).
func (r *VibeRepository) DeleteVibe(id uint) error {
	result := r.DB.Delete(&model.Vibe{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound // Or a custom error
	}
	return nil
}

// GetVibeStatistics calculates statistics for a given period.
// For simplicity, 'period' is not fully implemented here but shows how date ranges would work.
// UserID is not used yet, assuming single-user context for now.
func (r *VibeRepository) GetVibeStatistics(period string, startDate, endDate time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Mood distribution
	var moodDistribution []struct {
		Mood  string
		Count int
	}
	err := r.DB.Model(&model.Vibe{}).
		Select("mood, count(*) as count").
		Where("date BETWEEN ? AND ?", startDate, endDate).
		Group("mood").
		Order("count DESC").
		Scan(&moodDistribution).Error
	if err != nil {
		return nil, fmt.Errorf("error getting mood distribution: %w", err)
	}
	stats["mood_distribution"] = moodDistribution

	// Average energy level
	var avgEnergyLevel float64
	err = r.DB.Model(&model.Vibe{}).
		Where("date BETWEEN ? AND ?", startDate, endDate).
		Select("COALESCE(AVG(energy_level), 0)"). // COALESCE to handle cases with no entries
		Row().Scan(&avgEnergyLevel)
	if err != nil {
		return nil, fmt.Errorf("error getting average energy level: %w", err)
	}
	stats["average_energy_level"] = avgEnergyLevel

	// Could add trends here, e.g., mood over time, requires more complex queries or processing

	return stats, nil
}

// GetVibesForDateRange retrieves all vibes within a specific date range.
func (r *VibeRepository) GetVibesForDateRange(startDate, endDate time.Time) ([]model.Vibe, error) {
	var vibes []model.Vibe
	result := r.DB.Where("date BETWEEN ? AND ?", startDate, endDate).Order("date ASC").Find(&vibes)
	if result.Error != nil {
		return nil, result.Error
	}
	return vibes, nil
}

// GetMoodStreak calculates the current or longest streak for a given mood.
// This is a simplified version. A robust implementation would need to handle gaps in dates carefully.
// `checkCurrent` true for current streak, false for longest.
func (r *VibeRepository) GetMoodStreak(mood string, checkCurrent bool) (int, error) {
	var vibes []model.Vibe
	// Fetch all vibes for the specific mood, ordered by date
	if err := r.DB.Model(&model.Vibe{}).Where("mood = ?", mood).Order("date DESC").Find(&vibes).Error; err != nil {
		return 0, err
	}

	if len(vibes) == 0 {
		return 0, nil
	}

	if checkCurrent {
		// Check if the latest vibe for this mood is today or yesterday to be part of "current" streak
		// This logic needs to be more precise based on how "current" is defined (e.g. consecutive days)
		// For now, let's assume "current" means consecutive days ending on the most recent entry for that mood.
		currentStreak := 0
		// Assuming vibes are sorted DESC by date
		lastDate := vibes[0].Date
		for i, vibe := range vibes {
			if i == 0 {
				currentStreak++
				lastDate = vibe.Date
				continue
			}
			// Check if the current vibe is one day before the lastVibe
			if lastDate.Sub(vibe.Date).Hours() == 24 { // Approximation, consider date part only
				currentStreak++
				lastDate = vibe.Date
			} else {
				break // Streak broken
			}
		}
		return currentStreak, nil
	} else { // Longest streak
		if len(vibes) == 0 {
			return 0, nil
		}
		longestStreak := 0
		currentStreak := 0
		// Iterating from oldest to newest would be easier for longest streak. Let's re-query or reverse.
		// For simplicity, re-querying ordered by ASC for longest streak calculation.
		if err := r.DB.Model(&model.Vibe{}).Where("mood = ?", mood).Order("date ASC").Find(&vibes).Error; err != nil {
			return 0, err
		}

		var prevDate time.Time
		for i, vibe := range vibes {
			if i == 0 {
				currentStreak = 1
				prevDate = vibe.Date
				continue
			}
			// Compare date part only for consecutive days
			y1, m1, d1 := prevDate.Date()
			y2, m2, d2 := vibe.Date.Date()
			expectedNextDate := time.Date(y1, m1, d1, 0, 0, 0, 0, prevDate.Location()).AddDate(0, 0, 1)

			if y2 == expectedNextDate.Year() && m2 == expectedNextDate.Month() && d2 == expectedNextDate.Day() {
				currentStreak++
			} else {
				// Streak broken, reset current streak
				if currentStreak > longestStreak {
					longestStreak = currentStreak
				}
				currentStreak = 1 // Start new streak
			}
			prevDate = vibe.Date
		}
		if currentStreak > longestStreak { // Check after loop for the last streak
			longestStreak = currentStreak
		}
		return longestStreak, nil
	}
}

// BulkInsertVibes inserts multiple vibes in a single transaction.
func (r *VibeRepository) BulkInsertVibes(vibes []*model.Vibe) (int64, error) {
	if len(vibes) == 0 {
		return 0, nil
	}
	// GORM's CreateBatchSize can be used, or just Create with a slice.
	// Using Create with a slice is generally efficient for PostgreSQL.
	result := r.DB.Create(&vibes)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// ExportVibes retrieves vibes based on filters and formats them as CSV or JSON.
func (r *VibeRepository) ExportVibes(filters map[string]interface{}, format string, sortBy, sortOrder string) ([]byte, string, error) {
	var vibes []model.Vibe
	query := r.DB.Model(&model.Vibe{})

	// Apply filters (similar to GetAllVibes)
	if date, ok := filters["date"]; ok {
		query = query.Where("DATE(date) = ?", date)
	}
	if mood, ok := filters["mood"]; ok {
		query = query.Where("mood = ?", mood)
	}
	// Add more filters as needed

	// Apply sorting
	if sortBy != "" && sortOrder != "" {
		orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)
		query = query.Order(orderClause)
	} else {
		query = query.Order("date ASC") // Default sort for export
	}

	if err := query.Find(&vibes).Error; err != nil {
		return nil, "", err
	}

	var data []byte
	var contentType string
	var err error

	switch strings.ToLower(format) {
	case "csv":
		contentType = "text/csv"
		var buffer bytes.Buffer
		writer := csv.NewWriter(&buffer)
		// Write header
		header := []string{"ID", "Date", "Mood", "EnergyLevel", "Notes", "Activities"}
		if err = writer.Write(header); err != nil {
			return nil, "", err
		}
		// Write rows
		for _, vibe := range vibes {
			row := []string{
				fmt.Sprintf("%d", vibe.ID),
				vibe.Date.Format(time.RFC3339),
				vibe.Mood,
				fmt.Sprintf("%d", vibe.EnergyLevel),
				vibe.Notes,
				strings.Join(vibe.Activities, ";"), // CSV friendly format for array
			}
			if err = writer.Write(row); err != nil {
				return nil, "", err
			}
		}
		writer.Flush()
		if err = writer.Error(); err != nil {
			return nil, "", err
		}
		data = buffer.Bytes()

	case "json":
		contentType = "application/json"
		data, err = json.Marshal(vibes)
		if err != nil {
			return nil, "", err
		}
	default:
		return nil, "", fmt.Errorf("unsupported export format: %s", format)
	}

	return data, contentType, nil
}

// Note: Database indexing optimization.
// GORM creates indexes defined in model tags (e.g., `uniqueIndex` on Date).
// For specific query patterns in GetAllVibes, GetVibeStatistics, etc.,
// additional indexes might be beneficial.
// Example:
// CREATE INDEX idx_vibes_mood_date ON vibes (mood, date);
// CREATE INDEX idx_vibes_energy_level ON vibes (energy_level);
// These would typically be managed by a separate migration tool in production.
// For now, we rely on GORM's auto-migration and model tags.
// If performance issues arise, analyze query plans (EXPLAIN) and add indexes.
// For instance, filtering by mood and sorting by date for `GetMoodStreak` could benefit from `(mood, date)`.
// Filtering by date range for statistics could benefit from an index on `date`. (already unique indexed)
