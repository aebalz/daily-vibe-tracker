package service

import (
	// "github.com/user/daily-vibe-tracker/internal/model" // Will be needed for actual methods
	"github.com/user/daily-vibe-tracker/internal/repository"
)

// VibeServiceInterface defines the interface for vibe service operations.
// We'll define methods here in future prompts.
type VibeServiceInterface interface {
	// Example: CreateVibe(date time.Time, mood string, energyLevel int, notes string, activities []string) (*model.Vibe, error)
	// Example: GetVibe(id uint) (*model.Vibe, error)
}

// VibeService implements VibeServiceInterface.
type VibeService struct {
	VibeRepo repository.VibeRepositoryInterface
}

// NewVibeService creates a new VibeService.
func NewVibeService(vibeRepo repository.VibeRepositoryInterface) VibeServiceInterface {
	return &VibeService{VibeRepo: vibeRepo}
}

// Implement interface methods in subsequent prompts.
// For example:
// func (s *VibeService) CreateVibe(...) (*model.Vibe, error) {
//   // Business logic here, then call s.VibeRepo
//   return nil, nil
// }
