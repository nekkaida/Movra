package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPPort     int
	GRPCPort     int
	RedisAddr    string
	RedisPass    string
	JaegerURL    string
	Environment  string
	RateCacheTTL int // seconds
	LockDuration int // seconds
}

func Load() *Config {
	return &Config{
		HTTPPort:     getEnvInt("HTTP_PORT", 8082),
		GRPCPort:     getEnvInt("GRPC_PORT", 9092),
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:    getEnv("REDIS_PASSWORD", ""),
		JaegerURL:    getEnv("JAEGER_URL", "http://localhost:14268/api/traces"),
		Environment:  getEnv("ENVIRONMENT", "development"),
		RateCacheTTL: getEnvInt("RATE_CACHE_TTL", 60),
		LockDuration: getEnvInt("LOCK_DURATION", 30),
	}
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
