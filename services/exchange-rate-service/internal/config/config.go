package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the exchange rate service
type Config struct {
	// Server ports
	HTTPPort int
	GRPCPort int

	// Redis connection
	RedisAddr string
	RedisPass string
	RedisDB   int

	// Observability
	JaegerURL       string
	MetricsEnabled  bool
	MetricsEndpoint string

	// Environment
	Environment string
	LogLevel    string

	// Rate caching
	RateCacheTTL int // seconds
	LockDuration int // seconds (default lock duration)
	MaxLockDuration int // seconds (maximum allowed lock duration)

	// Provider configuration
	ProviderType      string  // "simulated" or "openexchangerates"
	ProviderSpread    float64 // Base spread percentage (e.g., 0.005 for 0.5%)
	ProviderMaxDrift  float64 // Max drift percentage for simulated provider

	// OpenExchangeRates API (for future use)
	OXRAppID  string
	OXRAPIUrl string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		// Server ports
		HTTPPort: getEnvInt("HTTP_PORT", 8082),
		GRPCPort: getEnvInt("GRPC_PORT", 9092),

		// Redis connection
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass: getEnv("REDIS_PASSWORD", ""),
		RedisDB:   getEnvInt("REDIS_DB", 0),

		// Observability
		JaegerURL:       getEnv("JAEGER_URL", "http://localhost:14268/api/traces"),
		MetricsEnabled:  getEnvBool("METRICS_ENABLED", true),
		MetricsEndpoint: getEnv("METRICS_ENDPOINT", "/metrics"),

		// Environment
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		// Rate caching
		RateCacheTTL:    getEnvInt("RATE_CACHE_TTL", 60),
		LockDuration:    getEnvInt("LOCK_DURATION", 30),
		MaxLockDuration: getEnvInt("MAX_LOCK_DURATION", 120),

		// Provider configuration
		ProviderType:     getEnv("PROVIDER_TYPE", "simulated"),
		ProviderSpread:   getEnvFloat("PROVIDER_SPREAD", 0.005),
		ProviderMaxDrift: getEnvFloat("PROVIDER_MAX_DRIFT", 0.02),

		// OpenExchangeRates API
		OXRAppID:  getEnv("OXR_APP_ID", ""),
		OXRAPIUrl: getEnv("OXR_API_URL", "https://openexchangerates.org/api"),
	}
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development" || c.Environment == "dev"
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == "production" || c.Environment == "prod"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
