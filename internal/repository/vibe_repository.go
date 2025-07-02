package repository

import (
	"github.com/user/daily-vibe-tracker/internal/model"
	"gorm.io/gorm"
)

// VibeRepositoryInterface defines the interface for vibe repository operations.
// We'll define methods here in future prompts.
type VibeRepositoryInterface interface {
	// Example: CreateVibe(vibe *model.Vibe) error
	// Example: GetVibeByID(id uint) (*model.Vibe, error)
}

// VibeRepository implements VibeRepositoryInterface.
type VibeRepository struct {
	DB *gorm.DB
}

// NewVibeRepository creates a new VibeRepository.
func NewVibeRepository(db *gorm.DB) VibeRepositoryInterface {
	return &VibeRepository{DB: db}
}

// Implement interface methods in subsequent prompts.
// For example:
// func (r *VibeRepository) CreateVibe(vibe *model.Vibe) error {
// 	 return r.DB.Create(vibe).Error
// }
