package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/patteeraL/movra/services/exchange-rate-service/internal/model"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/provider"
	"github.com/redis/go-redis/v9"
)

const (
	// Key prefixes for Redis
	rateKeyPrefix   = "rate:"
	lockedKeyPrefix = "locked:"
)

// RedisRepository implements RateRepository using Redis
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis-backed repository
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{
		client: client,
	}
}

// rateKey generates the Redis key for an exchange rate
func rateKey(source, target string) string {
	return fmt.Sprintf("%s%s:%s", rateKeyPrefix, source, target)
}

// lockedKey generates the Redis key for a locked rate
func lockedKey(lockID string) string {
	return lockedKeyPrefix + lockID
}

// SaveRate stores an exchange rate with TTL
func (r *RedisRepository) SaveRate(ctx context.Context, rate *provider.Rate, ttl time.Duration) error {
	data, err := json.Marshal(rate)
	if err != nil {
		return fmt.Errorf("failed to marshal rate: %w", err)
	}

	key := rateKey(rate.SourceCurrency, rate.TargetCurrency)
	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save rate: %w", err)
	}

	return nil
}

// GetRate retrieves a cached exchange rate
func (r *RedisRepository) GetRate(ctx context.Context, source, target string) (*provider.Rate, error) {
	key := rateKey(source, target)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get rate: %w", err)
	}

	var rate provider.Rate
	if err := json.Unmarshal(data, &rate); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rate: %w", err)
	}

	// Check if rate is still valid
	if time.Now().After(rate.ValidUntil) {
		// Rate has expired, delete it and return cache miss
		_ = r.client.Del(ctx, key)
		return nil, nil
	}

	return &rate, nil
}

// SaveLockedRate stores a locked rate for a transfer
func (r *RedisRepository) SaveLockedRate(ctx context.Context, locked *model.LockedRate) error {
	data, err := json.Marshal(locked)
	if err != nil {
		return fmt.Errorf("failed to marshal locked rate: %w", err)
	}

	key := lockedKey(locked.LockID)
	ttl := time.Until(locked.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("locked rate has already expired")
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save locked rate: %w", err)
	}

	return nil
}

// GetLockedRate retrieves a locked rate by ID
func (r *RedisRepository) GetLockedRate(ctx context.Context, lockID string) (*model.LockedRate, error) {
	key := lockedKey(lockID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get locked rate: %w", err)
	}

	var locked model.LockedRate
	if err := json.Unmarshal(data, &locked); err != nil {
		return nil, fmt.Errorf("failed to unmarshal locked rate: %w", err)
	}

	// Check if rate has expired
	if time.Now().After(locked.ExpiresAt) {
		locked.Expired = true
		// Delete expired lock
		_ = r.client.Del(ctx, key)
		return nil, ErrExpired{LockID: lockID}
	}

	return &locked, nil
}

// DeleteLockedRate removes a locked rate
func (r *RedisRepository) DeleteLockedRate(ctx context.Context, lockID string) error {
	key := lockedKey(lockID)
	result, err := r.client.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to delete locked rate: %w", err)
	}

	if result == 0 {
		return ErrNotFound{Key: lockID}
	}

	return nil
}

// ExtendLockedRate extends the expiration of a locked rate
func (r *RedisRepository) ExtendLockedRate(ctx context.Context, lockID string, newExpiry time.Time) error {
	// Get existing locked rate
	locked, err := r.GetLockedRate(ctx, lockID)
	if err != nil {
		return err
	}
	if locked == nil {
		return ErrNotFound{Key: lockID}
	}

	// Update expiry
	locked.ExpiresAt = newExpiry

	// Save with new TTL
	return r.SaveLockedRate(ctx, locked)
}

// Health checks if Redis is healthy
func (r *RedisRepository) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// GetCacheStats returns cache statistics (optional method, not in interface)
func (r *RedisRepository) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	info, err := r.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis stats: %w", err)
	}

	// Parse basic stats from info string
	// This is a simplified version - full parsing would extract actual values
	_ = info

	dbSize, err := r.client.DBSize(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get DB size: %w", err)
	}

	return &CacheStats{
		Size:       dbSize,
		LastUpdate: time.Now(),
	}, nil
}
