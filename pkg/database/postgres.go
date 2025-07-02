package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aebalz/daily-vibe-tracker/internal/config"
	"github.com/aebalz/daily-vibe-tracker/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectDB initializes the database connection using GORM.
func ConnectDB(cfg *config.AppConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSslMode,
		cfg.DBTimezone,
	)

	logLevel := logger.Silent
	if cfg.AppEnv == "development" {
		logLevel = logger.Info
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logLevel,    // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Disable color
		},
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
		// NamingStrategy: schema.NamingStrategy{
		// TablePrefix: "dvt_", // Example: Add a table prefix
		// SingularTable: true, // Use singular table names
		// },
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connection established successfully.")
	return DB, nil
}

// MigrateDB runs GORM auto-migrations for the defined models.
// In a production environment, a more robust migration tool (like golang-migrate/migrate) is recommended.
func MigrateDB(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	err := db.AutoMigrate(&model.Vibe{})
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	log.Println("Database migration completed successfully.")
	return nil
}

// CloseDB closes the database connection.
func CloseDB() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			err = sqlDB.Close()
			if err != nil {
				log.Printf("Error closing database connection: %v\n", err)
			} else {
				log.Println("Database connection closed.")
			}
		}
	}
}

// PingDB checks the database connection.
func PingDB(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB for ping: %w", err)
	}
	return sqlDB.Ping()
}
