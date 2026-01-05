package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the settlement service
type Config struct {
	// Server
	HTTPPort string
	GRPCPort string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Kafka
	KafkaBrokers       string
	KafkaConsumerGroup string
	KafkaTopicFunded   string
	KafkaTopicStatus   string

	// Provider
	ProviderType           string // "simulated" or future real providers
	ProviderFailureRate    int
	ProviderProcessingTime time.Duration

	// Retry
	MaxRetries    int
	RetryInterval time.Duration
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		HTTPPort: getEnv("HTTP_PORT", "8083"),
		GRPCPort: getEnv("GRPC_PORT", "9083"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		KafkaBrokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "settlement-service"),
		KafkaTopicFunded:   getEnv("KAFKA_TOPIC_FUNDED", "transfer.funded"),
		KafkaTopicStatus:   getEnv("KAFKA_TOPIC_STATUS", "payout.status"),

		ProviderType:           getEnv("PROVIDER_TYPE", "simulated"),
		ProviderFailureRate:    getEnvInt("PROVIDER_FAILURE_RATE", 10),
		ProviderProcessingTime: getEnvDuration("PROVIDER_PROCESSING_TIME", 2*time.Second),

		MaxRetries:    getEnvInt("MAX_RETRIES", 3),
		RetryInterval: getEnvDuration("RETRY_INTERVAL", 5*time.Second),
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
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
