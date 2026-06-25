package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server
	ServerPort    int
	AllowedOrigins []string

	// Redis
	RedisURL      string
	RedisPassword string
	RedisDB       int

	// Translation
	GeminiAPIKey  string
	GeminiModel   string
	DefaultEngine string

	// Cache
	CacheTTL time.Duration

	// Application
	Version   string
	LogLevel  string
	LogFormat string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	_ = godotenv.Load(".env") // ignore error, it's fine if it doesn't exist

	cfg := &Config{
		ServerPort:     getEnvInt("PORT", 8080),
		AllowedOrigins: getEnvSlice("CORS_ORIGINS", []string{"http://localhost:5173", "http://localhost:3000"}),

		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		GeminiAPIKey:  getEnv("GEMINI_API_KEY", ""),
		GeminiModel:   getEnv("GEMINI_MODEL", "gemini-2.5-flash"),
		DefaultEngine: getEnv("DEFAULT_ENGINE", "mymemory"),

		CacheTTL: getEnvDuration("CACHE_TTL", 1*time.Hour),

		Version:   getEnv("APP_VERSION", "1.0.0"),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		LogFormat: getEnv("LOG_FORMAT", "json"),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d", c.ServerPort)
	}
	if c.DefaultEngine != "gemini" && c.DefaultEngine != "mymemory" {
		return fmt.Errorf("invalid default engine: %s (must be 'gemini' or 'mymemory')", c.DefaultEngine)
	}
	return nil
}

// IsGeminiConfigured returns true if a Gemini API key or GCP credentials are set.
func (c *Config) IsGeminiConfigured() bool {
	return c.GeminiAPIKey != "" || os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != ""
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	if val, ok := os.LookupEnv(key); ok {
		return strings.Split(val, ",")
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
