package model

import (
	"time"

	"gorm.io/gorm"
)

// Vibe represents the structure for a daily vibe entry.
type Vibe struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Date        time.Time      `json:"date" gorm:"uniqueIndex;not null"` // Ensure date is not null
	Mood        string         `json:"mood" gorm:"not null"`
	EnergyLevel int            `json:"energy_level" gorm:"check:energy_level >= 1 AND energy_level <= 10"`
	Notes       string         `json:"notes"`
	Activities  []string       `json:"activities" gorm:"type:text[]"` // For PostgreSQL text array
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"` // Add gorm.DeletedAt for soft deletes
}

// TableName specifies the table name for the Vibe model.
// This is optional if you want GORM to pluralize 'Vibe' to 'vibes'.
// func (Vibe) TableName() string {
//  return "vibes"
// }

// Helper type for Activities to work with GORM and text arrays.
// GORM's default handling for []string with text[] might need this,
// however, with recent GORM versions and pgx driver, it often works directly.
// If issues arise, we might need to implement Valuer and Scanner interfaces.
// For now, we'll rely on GORM's native handling.

/*
Example for custom scanner/valuer if needed later:
import (
    "database/sql/driver"
    "fmt"
    "strings"
    "github.com/lib/pq" // Or use native pgx array handling
)

// StringArray custom type for []string
type StringArray []string

// Scan implements the Scanner interface for StringArray
func (a *StringArray) Scan(value interface{}) error {
    switch v := value.(type) {
    case []byte:
        return pq.Array(a).Scan(v)
    case string:
        if v == "" {
            *a = []string{}
            return nil
        }
        // Assuming string format like "{val1,val2,val3}"
        str := strings.Trim(v, "{}")
        if str == "" {
            *a = []string{}
            return nil
        }
        *a = strings.Split(str, ",")
        return nil
    default:
        return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type *StringArray", value)
    }
}

// Value implements the Valuer interface for StringArray
func (a StringArray) Value() (driver.Value, error) {
    if len(a) == 0 {
        return "{}", nil // Or "NULL" if appropriate
    }
    return pq.Array(a).Value()
}

// Then in Vibe struct:
// Activities  StringArray    `json:"activities" gorm:"type:text[]"`
*/
