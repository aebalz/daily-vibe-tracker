package service

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aebalz/daily-vibe-tracker/internal/model"
	"github.com/aebalz/daily-vibe-tracker/internal/repository"
	// "github.com/go-playground/validator/v10" // Example for more complex validation
)

// VibeServiceRequestLimitOffset defines default values for limit and offset.
const (
	DefaultLimit   = 10
	DefaultOffset  = 0
	MaxLimit       = 100
	DefaultSortBy  = "date"
	DefaultSortOrder = "desc"
)

// VibeServiceInterface defines the interface for vibe service operations.
type VibeServiceInterface interface {
	CreateVibe(vibe *model.Vibe) (*model.Vibe, error)
	GetVibeByID(id uint) (*model.Vibe, error)
	GetAllVibes(filters map[string]interface{}, limit, offset int, sortBy, sortOrder string) ([]model.Vibe, int64, error)
	UpdateVibe(id uint, updatedVibe *model.Vibe) (*model.Vibe, error)
	DeleteVibe(id uint) error

	GetVibeStatistics(period string) (map[string]interface{}, error)
	GetTodaysVibeRecommendation() (map[string]interface{}, error)
	GetMoodStreak(mood string) (map[string]interface{}, error)

	ExportVibes(filters map[string]interface{}, format string, sortBy, sortOrder string) ([]byte, string, error)
	BulkImportVibes(vibes []*model.Vibe) (int64, error)

	// ValidateVibe(vibe *model.Vibe) error // Example for a validation helper
}

	"context" // Required for cache operations

	"github.com/aebalz/daily-vibe-tracker/internal/config" // Required for AppConfig
	"github.com/aebalz/daily-vibe-tracker/pkg/cache"       // Required for RedisCache
)

// VibeService implements VibeServiceInterface.
type VibeService struct {
	VibeRepo repository.VibeRepositoryInterface
	Cache    *cache.RedisCache // Pointer to allow nil if cache connection fails
	Cfg      *config.AppConfig // To access CacheTTLExpiration etc.
	// validate *validator.Validate // For struct validation if needed
}

// NewVibeService creates a new VibeService.
func NewVibeService(vibeRepo repository.VibeRepositoryInterface, redisCache *cache.RedisCache, cfg *config.AppConfig) VibeServiceInterface {
	return &VibeService{
		VibeRepo: vibeRepo,
		Cache:    redisCache,
		Cfg:      cfg,
		// validate: validator.New(), // Initialize validator
	}
}

// --- Cache Key Generators ---
func getVibeCacheKey(id uint) string {
	return fmt.Sprintf("vibe:%d", id)
}

func getVibeStatsCacheKey(period string) string {
	// Normalize period for cache key consistency, e.g., daily for specific day, weekly for specific week number/year
	// For simplicity, using period string directly. Could add date context for more granular stats caching.
	// Example: "stats:week:2023-42", "stats:month:2023-10"
	// For now, just "stats:period_name" which means stats for "current" week/month/year based on when it's calculated.
	// This is okay if TTL is relatively short or invalidation is aggressive.
	return fmt.Sprintf("stats:%s", strings.ToLower(period))
}

// --- Helper for Cache Invalidation ---
func (s *VibeService) invalidateVibeCache(id uint) {
	if s.Cache != nil {
		key := getVibeCacheKey(id)
		err := s.Cache.Delete(context.Background(), key)
		if err != nil {
			fmt.Printf("Warning: failed to delete vibe %d from cache: %v\n", id, err)
		}
	}
}

func (s *VibeService) invalidateStatsCache(period string) {
	if s.Cache != nil {
		// This is a broad invalidation for the given period type.
		// More granular invalidation would require knowing the exact date ranges affected.
		key := getVibeStatsCacheKey(period)
		err := s.Cache.Delete(context.Background(), key)
		if err != nil {
			fmt.Printf("Warning: failed to delete stats cache for period %s: %v\n", period, err)
		}
		// Potentially invalidate all stats keys if a vibe change could affect multiple periods
		// e.g., s.Cache.DeletePattern(context.Background(), "stats:*")
		// For now, just the specific period type.
	}
}


// ValidateVibe performs business logic validation on a vibe.
// GORM struct tags handle database-level validation. This is for service-level rules.
func (s *VibeService) ValidateVibe(vibe *model.Vibe) error {
	if vibe.EnergyLevel < 1 || vibe.EnergyLevel > 10 {
		return fmt.Errorf("energy level must be between 1 and 10")
	}
	if strings.TrimSpace(vibe.Mood) == "" {
		return fmt.Errorf("mood cannot be empty")
	}
	// Example: Check if date is not in the future (if that's a rule)
	// if vibe.Date.After(time.Now()) {
	// 	return fmt.Errorf("vibe date cannot be in the future")
	// }
	return nil
}

// CreateVibe handles the business logic for creating a new vibe.
func (s *VibeService) CreateVibe(vibe *model.Vibe) (*model.Vibe, error) {
	if err := s.ValidateVibe(vibe); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	// Additional business logic before saving, if any.
	// For example, normalizing mood strings to lowercase.
	vibe.Mood = strings.ToLower(strings.TrimSpace(vibe.Mood))

	createdVibe, err := s.VibeRepo.CreateVibe(vibe)
	if err != nil {
		return nil, err
	}
	// Invalidate stats cache as new data might change statistics
	s.invalidateStatsCache("week") // Invalidate all relevant periods or use a pattern
	s.invalidateStatsCache("month")
	s.invalidateStatsCache("year")
	// No need to invalidate GetVibeByID cache for a newly created vibe, as it won't be cached yet by its ID.
	return createdVibe, nil
}

// GetVibeByID retrieves a single vibe by its ID, using cache if available.
func (s *VibeService) GetVibeByID(id uint) (*model.Vibe, error) {
	if s.Cache != nil {
		var vibe model.Vibe
		cacheKey := getVibeCacheKey(id)
		if err := s.Cache.Get(context.Background(), cacheKey, &vibe); err == nil {
			// Cache hit
			return &vibe, nil
		}
		// Cache miss or error, proceed to fetch from DB
	}

	vibe, err := s.VibeRepo.GetVibeByID(id)
	if err != nil {
		return nil, err
	}

	if s.Cache != nil && vibe != nil { // vibe != nil to avoid caching non-existent records that returned error
		cacheKey := getVibeCacheKey(id)
		if err := s.Cache.Set(context.Background(), cacheKey, vibe); err != nil {
			fmt.Printf("Warning: failed to set vibe %d in cache: %v\n", id, err)
		}
	}
	return vibe, nil
}

// GetAllVibes retrieves vibes with filters, pagination, and sorting.
// Caching for GetAllVibes can be complex due to various filter combinations.
// Consider caching only for very common filter sets or use a very short TTL if implemented.
// For now, not caching GetAllVibes.
func (s *VibeService) GetAllVibes(filters map[string]interface{}, limit, offset int, sortBy, sortOrder string) ([]model.Vibe, int64, error) {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	if offset < 0 {
		offset = DefaultOffset
	}
	if sortBy == "" {
		sortBy = DefaultSortBy
	}
	if sortOrder == "" {
		sortOrder = DefaultSortOrder
	} else {
		sortOrder = strings.ToLower(sortOrder)
		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = DefaultSortOrder
		}
	}

	// Sanitize/validate filter values if necessary
	if mood, ok := filters["mood"].(string); ok {
		filters["mood"] = strings.ToLower(strings.TrimSpace(mood))
	}


	return s.VibeRepo.GetAllVibes(filters, limit, offset, sortBy, sortOrder)
}

// UpdateVibe handles the business logic for updating an existing vibe.
func (s *VibeService) UpdateVibe(id uint, updatedVibe *model.Vibe) (*model.Vibe, error) {
	if err := s.ValidateVibe(updatedVibe); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	// Ensure mood is consistent
	updatedVibe.Mood = strings.ToLower(strings.TrimSpace(updatedVibe.Mood))

	// The repository's UpdateVibe should fetch the existing record first.
	// Additional service-level checks can be done here if needed,
	// e.g., checking if the user is authorized to update this vibe (if users were implemented).
	resultVibe, err := s.VibeRepo.UpdateVibe(id, updatedVibe)
	if err != nil {
		return nil, err
	}
	// Invalidate caches
	s.invalidateVibeCache(id)
	s.invalidateStatsCache("week")
	s.invalidateStatsCache("month")
	s.invalidateStatsCache("year")
	return resultVibe, nil
}

// DeleteVibe handles the business logic for deleting a vibe.
func (s *VibeService) DeleteVibe(id uint) error {
	// Add any business logic before deletion if needed.
	err := s.VibeRepo.DeleteVibe(id)
	if err != nil {
		return err
	}
	// Invalidate caches
	s.invalidateVibeCache(id)
	s.invalidateStatsCache("week")
	s.invalidateStatsCache("month")
	s.invalidateStatsCache("year")
	return nil
}

// GetVibeStatistics calculates and returns vibe statistics, using cache if available.
func (s *VibeService) GetVibeStatistics(period string) (map[string]interface{}, error) {
	cacheKey := getVibeStatsCacheKey(period)
	if s.Cache != nil {
		var stats map[string]interface{}
		if err := s.Cache.Get(context.Background(), cacheKey, &stats); err == nil {
			// Cache hit
			return stats, nil
		}
		// Cache miss or error, proceed to compute
	}

	var startDate, endDate time.Time
	now := time.Now()

	// Determine date range based on period
	switch strings.ToLower(period) {
	case "week":
		// Assuming week starts on Monday and ends on Sunday
		weekday := now.Weekday()
		if weekday == time.Sunday { // Adjust if week starts on Sunday
			startDate = now.AddDate(0, 0, -6)
		} else {
			startDate = now.AddDate(0, 0, -int(weekday)+1)
		}
		startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second) // End of Sunday
	case "month":
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second) // End of last day of month
	case "year":
		startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		endDate = time.Date(now.Year(), 12, 31, 23, 59, 59, 0, now.Location())
	default: // Default to current month if period is invalid or not specified
		// Or return an error: return nil, fmt.Errorf("invalid period: %s. valid options: week, month, year", period)
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	stats, err := s.VibeRepo.GetVibeStatistics(period, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Advanced Analytics: Mood patterns, correlations
	vibesForPeriod, err := s.VibeRepo.GetVibesForDateRange(startDate, endDate)
	if err != nil {
		// Log this error but don't fail the whole stats call, or decide if this data is critical
		// For now, we'll proceed without these advanced stats if data fetching fails
		fmt.Printf("Warning: could not fetch vibes for advanced analytics: %v\n", err)
	} else {
		if len(vibesForPeriod) > 0 {
			stats["mood_patterns"] = s.calculateMoodPatterns(vibesForPeriod)
			stats["mood_energy_correlation"] = s.calculateMoodEnergyCorrelation(vibesForPeriod)
			stats["activity_mood_correlation"] = s.calculateActivityMoodCorrelation(vibesForPeriod, 5) // Top 5 activities
		} else {
			stats["mood_patterns"] = "Not enough data for mood patterns."
			stats["mood_energy_correlation"] = "Not enough data for mood-energy correlation."
			stats["activity_mood_correlation"] = "Not enough data for activity-mood correlation."
		}
	}

	if s.Cache != nil && len(stats) > 0 { // len(stats) > 0 to avoid caching empty/error states if logic allows
		if err := s.Cache.Set(context.Background(), cacheKey, stats); err != nil {
			fmt.Printf("Warning: failed to set stats for period %s in cache: %v\n", period, err)
		}
	}

	return stats, nil
}


// calculateMoodPatterns identifies common transitions between moods.
// Assumes vibes are sorted by date.
func (s *VibeService) calculateMoodPatterns(vibes []model.Vibe) interface{} {
	if len(vibes) < 2 {
		return "Not enough data for mood patterns (need at least 2 entries)."
	}
	patterns := make(map[string]int)
	// Ensure vibes are sorted by date for accurate pattern detection
	// The GetVibesForDateRange repository method already sorts by date ASC

	for i := 0; i < len(vibes)-1; i++ {
		// Check if the next vibe is on the subsequent day for a true daily transition
		// This makes the pattern more meaningful as a "next day" transition.
		// For simplicity, we'll just count transitions between any two consecutive records in the period.
		// A more advanced version would filter for strictly consecutive days.
		pattern := fmt.Sprintf("%s -> %s", vibes[i].Mood, vibes[i+1].Mood)
		patterns[pattern]++
	}
	if len(patterns) == 0 {
		return "No mood transitions found in the period."
	}
	return patterns
}

// calculateMoodEnergyCorrelation calculates the average energy level for each mood.
func (s *VibeService) calculateMoodEnergyCorrelation(vibes []model.Vibe) interface{} {
	if len(vibes) == 0 {
		return "Not enough data for mood-energy correlation."
	}
	energyByMood := make(map[string][]int)
	for _, v := range vibes {
		energyByMood[v.Mood] = append(energyByMood[v.Mood], v.EnergyLevel)
	}

	avgEnergyByMood := make(map[string]float64)
	for mood, energies := range energyByMood {
		if len(energies) == 0 {
			continue
		}
		sum := 0
		for _, e := range energies {
			sum += e
		}
		avgEnergyByMood[mood] = float64(sum) / float64(len(energies))
	}
	if len(avgEnergyByMood) == 0 {
		return "No mood-energy correlations found."
	}
	return avgEnergyByMood
}

// calculateActivityMoodCorrelation identifies common moods for top N activities.
func (s *VibeService) calculateActivityMoodCorrelation(vibes []model.Vibe, topNActivities int) interface{} {
	if len(vibes) == 0 {
		return "Not enough data for activity-mood correlation."
	}

	activityFrequency := make(map[string]int)
	activityMoods := make(map[string]map[string]int) // activity -> mood -> count

	for _, vibe := range vibes {
		for _, activity := range vibe.Activities {
			if activity == "" {
				continue
			}
			activity = strings.ToLower(strings.TrimSpace(activity))
			activityFrequency[activity]++
			if _, ok := activityMoods[activity]; !ok {
				activityMoods[activity] = make(map[string]int)
			}
			activityMoods[activity][vibe.Mood]++
		}
	}

	if len(activityFrequency) == 0 {
		return "No activities logged in the period."
	}

	// Get top N activities
	type activityCount struct {
		Name  string
		Count int
	}
	var sortedActivities []activityCount
	for name, count := range activityFrequency {
		sortedActivities = append(sortedActivities, activityCount{Name: name, Count: count})
	}

	// Sort activities by frequency (descending)
	// Using a simple sort, for larger N consider a heap
	for i := 0; i < len(sortedActivities); i++ {
		for j := i + 1; j < len(sortedActivities); j++ {
			if sortedActivities[j].Count > sortedActivities[i].Count {
				sortedActivities[i], sortedActivities[j] = sortedActivities[j], sortedActivities[i]
			}
		}
	}

	limit := topNActivities
	if len(sortedActivities) < topNActivities {
		limit = len(sortedActivities)
	}

	result := make(map[string]interface{})
	for i := 0; i < limit; i++ {
		actName := sortedActivities[i].Name
		result[actName] = map[string]interface{}{
			"total_occurrences": sortedActivities[i].Count,
			"mood_distribution": activityMoods[actName],
		}
	}
	if len(result) == 0 {
		return "Could not determine top activities or their mood correlations."
	}
	return result
}


// GetTodaysVibeRecommendation provides a simple recommendation.
func (s *VibeService) GetTodaysVibeRecommendation() (map[string]interface{}, error) {
	// Simple recommendation: Suggest activities from past good days.
	// A "good day" could be defined as mood = "happy" or "great" and energy_level >= 7.
	// This is a placeholder for a more sophisticated algorithm.

	// Fetch recent positive vibes
	// For a more robust recommendation, consider a wider range or user-specific history.
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	vibes, err := s.VibeRepo.GetVibesForDateRange(threeMonthsAgo, time.Now())
	if err != nil {
		return nil, fmt.Errorf("could not fetch historical data for recommendation: %w", err)
	}

	var potentialActivities []string
	highEnergyMoods := map[string]bool{"happy": true, "great": true, "energetic": true, "excited": true, "motivated": true}

	for _, vibe := range vibes {
		if vibe.EnergyLevel >= 7 && highEnergyMoods[vibe.Mood] {
			for _, activity := range vibe.Activities {
				if activity != "" {
					potentialActivities = append(potentialActivities, activity)
				}
			}
		}
	}

	if len(potentialActivities) == 0 {
		return map[string]interface{}{
			"suggestion": "No specific activity suggestions based on recent high-energy, positive vibes. Maybe try something new today!",
			"reason":     "Could not find relevant past activities.",
		}, nil
	}

	// Pick a random activity from the list
	rand.Seed(time.Now().UnixNano())
	suggestedActivity := potentialActivities[rand.Intn(len(potentialActivities))]

	return map[string]interface{}{
		"suggestion": fmt.Sprintf("Based on past good days, you might enjoy: %s", suggestedActivity),
		"reason":     "This activity was associated with high energy and positive mood in the past.",
	}, nil
}

// GetMoodStreak gets current and longest streak for a given mood.
func (s *VibeService) GetMoodStreak(mood string) (map[string]interface{}, error) {
	if strings.TrimSpace(mood) == "" {
		return nil, fmt.Errorf("mood parameter cannot be empty")
	}
	normalizedMood := strings.ToLower(strings.TrimSpace(mood))

	currentStreak, err := s.VibeRepo.GetMoodStreak(normalizedMood, true)
	if err != nil {
		return nil, fmt.Errorf("error calculating current streak for mood '%s': %w", normalizedMood, err)
	}

	longestStreak, err := s.VibeRepo.GetMoodStreak(normalizedMood, false)
	if err != nil {
		return nil, fmt.Errorf("error calculating longest streak for mood '%s': %w", normalizedMood, err)
	}

	return map[string]interface{}{
		"mood":           normalizedMood,
		"current_streak": currentStreak,
		"longest_streak": longestStreak,
	}, nil
}

// ExportVibes handles data export logic.
func (s *VibeService) ExportVibes(filters map[string]interface{}, format string, sortBy, sortOrder string) ([]byte, string, error) {
	if format == "" {
		return nil, "", fmt.Errorf("export format must be specified (e.g., csv, json)")
	}
	if sortBy == "" {
		sortBy = DefaultSortBy
	}
	if sortOrder == "" {
		sortOrder = DefaultSortOrder
	} else {
		sortOrder = strings.ToLower(sortOrder)
		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = DefaultSortOrder
		}
	}
	return s.VibeRepo.ExportVibes(filters, format, sortBy, sortOrder)
}

// BulkImportVibes handles bulk import of vibes.
func (s *VibeService) BulkImportVibes(vibes []*model.Vibe) (int64, error) {
	if len(vibes) == 0 {
		return 0, fmt.Errorf("no vibes provided for bulk import")
	}

	// Validate each vibe before attempting to insert
	for i, vibe := range vibes {
		if err := s.ValidateVibe(vibe); err != nil {
			return 0, fmt.Errorf("validation error for vibe at index %d: %w", i, err)
		}
		vibe.Mood = strings.ToLower(strings.TrimSpace(vibe.Mood)) // Normalize mood
	}

	// Additional business logic for bulk import can be added here.
	// For example, checking for duplicate dates if that's a constraint not handled by the DB upsert logic.
	// The current repository CreateVibe will fail on unique date constraint violations if not handled.
	// For true "import" functionality, one might consider an "upsert" strategy or error aggregation.
	// For now, we rely on the repository's BulkInsertVibes which uses GORM's batch create.

	return s.VibeRepo.BulkInsertVibes(vibes)
}

/*
// Example for advanced analytics functions (placeholders)
func (s *VibeService) calculateMoodPatterns(vibes []model.Vibe) interface{} {
	// Logic to find common sequences of moods
	// e.g., if "stressed" is often followed by "calm" after certain activities
	if len(vibes) < 2 {
		return "Not enough data for mood patterns."
	}
	patterns := make(map[string]int)
	for i := 0; i < len(vibes)-1; i++ {
		pattern := fmt.Sprintf("%s -> %s", vibes[i].Mood, vibes[i+1].Mood)
		patterns[pattern]++
	}
	return patterns
}

func (s *VibeService) calculateCorrelations(vibes []model.Vibe) interface{} {
	// Logic to find correlations, e.g., average energy level per mood
	if len(vibes) == 0 {
		return "Not enough data for correlations."
	}
	energyByMood := make(map[string][]int)
	for _, v := range vibes {
		energyByMood[v.Mood] = append(energyByMood[v.Mood], v.EnergyLevel)
	}

	avgEnergyByMood := make(map[string]float64)
	for mood, energies := range energyByMood {
		if len(energies) == 0 {
			continue
		}
		sum := 0
		for _, e := range energies {
			sum += e
		}
		avgEnergyByMood[mood] = float64(sum) / float64(len(energies))
	}
	return avgEnergyByMood
}
*/
