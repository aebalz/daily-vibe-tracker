package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// AppConfig holds the application configuration.
type AppConfig struct {
	DBHost             string
	DBPort             int
	DBUser             string
	DBPassword         string
	DBName             string
	DBSslMode          string
	DBTimezone         string
	ServerPort         int
	ServerHost         string
	ServerFramework    string
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration
	ServerIdleTimeout  time.Duration
	AppEnv             string
	LogLevel           string
	AppName            string
	CorsAllowedOrigins []string
	RateLimitMax       int
	RateLimitWindow    time.Duration
	RateLimitPerSecond float64 // For middleware
	RateLimitBurst     int     // For middleware
	SwaggerHost        string
	SwaggerBasePath    string
	SwaggerSchemes     []string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	CacheTTLExpiration time.Duration
}

// LoadConfig loads configuration from .env file or environment variables.
func LoadConfig(envFile ...string) (*AppConfig, error) {
	if len(envFile) > 0 {
		if _, err := os.Stat(envFile[0]); err == nil {
			err := godotenv.Load(envFile[0])
			if err != nil {
				log.Printf("Warning: Could not load .env file: %v. Using environment variables or defaults.", err)
			}
		} else {
			log.Printf("Warning: Specified .env file %s not found. Using environment variables or defaults.", envFile[0])
		}
	} else {
		// Try loading default .env file if no specific file is provided
		if _, err := os.Stat("config.env"); err == nil {
			err := godotenv.Load("config.env")
			if err != nil {
				log.Printf("Warning: Could not load default config.env file: %v. Using environment variables or defaults.", err)
			}
		}
	}

	cfg := &AppConfig{
		DBHost:             getStringEnv("DB_HOST", "localhost"),
		DBPort:             getIntEnv("DB_PORT", 5432),
		DBUser:             getStringEnv("DB_USER", "postgres"),
		DBPassword:         getStringEnv("DB_PASSWORD", "password"),
		DBName:             getStringEnv("DB_NAME", "daily_vibe_tracker"),
		DBSslMode:          getStringEnv("DB_SSL_MODE", "disable"),
		DBTimezone:         getStringEnv("DB_TIMEZONE", "UTC"),
		ServerPort:         getIntEnv("SERVER_PORT", 8080),
		ServerHost:         getStringEnv("SERVER_HOST", "0.0.0.0"),
		ServerFramework:    strings.ToLower(getStringEnv("SERVER_FRAMEWORK", "fiber")),
		ServerReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", "15s"),
		ServerWriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", "15s"),
		ServerIdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", "60s"),
		AppEnv:             strings.ToLower(getStringEnv("APP_ENV", "development")),
		LogLevel:           strings.ToLower(getStringEnv("LOG_LEVEL", "info")),
		AppName:            getStringEnv("APP_NAME", "Daily Vibe Tracker"),
		CorsAllowedOrigins: getSliceEnv("CORS_ALLOWED_ORIGINS", "*"),
		RateLimitMax:       getIntEnv("RATE_LIMIT_MAX", 100),         // Example, might not be directly used if rps/burst used
		RateLimitWindow:    getDurationEnv("RATE_LIMIT_WINDOW", "1m"), // Example, might not be directly used
		RateLimitPerSecond: getFloatEnv("RATE_LIMIT_RPS", 10),         // Requests per second for limiter
		RateLimitBurst:     getIntEnv("RATE_LIMIT_BURST", 20),         // Burst for limiter
		SwaggerHost:        getStringEnv("SWAGGER_HOST", "localhost:8080"),
		SwaggerBasePath:    getStringEnv("SWAGGER_BASE_PATH", "/api/v1"), // Defaulting to /api/v1
		SwaggerSchemes:     getSliceEnv("SWAGGER_SCHEMES", "http,https"),
		RedisAddr:          getStringEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getStringEnv("REDIS_PASSWORD", ""), // No password by default
		RedisDB:            getIntEnv("REDIS_DB", 0),           // Default Redis DB
		CacheTTLExpiration: getDurationEnv("CACHE_TTL_EXPIRATION", "5m"),
	}

	// Validate framework choice
	if cfg.ServerFramework != "fiber" && cfg.ServerFramework != "gin" {
		log.Printf("Warning: Invalid SERVER_FRAMEWORK '%s'. Defaulting to 'fiber'.", cfg.ServerFramework)
		cfg.ServerFramework = "fiber"
	}

	// Validate APP_ENV
	validAppEnvs := map[string]bool{"development": true, "staging": true, "production": true}
	if !validAppEnvs[cfg.AppEnv] {
		log.Printf("Warning: Invalid APP_ENV '%s'. Defaulting to 'development'.", cfg.AppEnv)
		cfg.AppEnv = "development"
	}

	return cfg, nil
}

func getStringEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func getIntEnv(key string, defaultValue int) int {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid value for %s: %s. Using default %d.", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}

func getDurationEnv(key, defaultValue string) time.Duration {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		valueStr = defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid duration value for %s: %s. Using default %s.", key, valueStr, defaultValue)
		// Try parsing default value in case it's also bad (though it shouldn't be)
		defaultDur, _ := time.ParseDuration(defaultValue)
		return defaultDur
	}
	return value
}

func getSliceEnv(key, defaultValue string) []string {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		valueStr = defaultValue
	}
	if valueStr == "" {
		return []string{}
	}
	return strings.Split(valueStr, ",")
}

func getFloatEnv(key string, defaultValue float64) float64 {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		log.Printf("Warning: Invalid float value for %s: %s. Using default %f.", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}
