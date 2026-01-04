package repository

import (
	"context"
	"time"

	"github.com/patteeraL/movra/services/exchange-rate-service/internal/model"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/provider"
)

// RateRepository defines the interface for rate storage operations
type RateRepository interface {
	// SaveRate stores an exchange rate with TTL
	SaveRate(ctx context.Context, rate *provider.Rate, ttl time.Duration) error

	// GetRate retrieves a cached exchange rate
	// Returns nil, nil if not found (cache miss)
	GetRate(ctx context.Context, source, target string) (*provider.Rate, error)

	// SaveLockedRate stores a locked rate for a transfer
	SaveLockedRate(ctx context.Context, locked *model.LockedRate) error

	// GetLockedRate retrieves a locked rate by ID
	// Returns nil, nil if not found or expired
	GetLockedRate(ctx context.Context, lockID string) (*model.LockedRate, error)

	// DeleteLockedRate removes a locked rate
	DeleteLockedRate(ctx context.Context, lockID string) error

	// ExtendLockedRate extends the expiration of a locked rate
	ExtendLockedRate(ctx context.Context, lockID string, newExpiry time.Time) error

	// Health checks if the repository is healthy
	Health(ctx context.Context) error
}

// CacheStats holds cache statistics
type CacheStats struct {
	Hits       int64
	Misses     int64
	Size       int64
	LastUpdate time.Time
}

// ErrNotFound is returned when a requested item is not in the repository
type ErrNotFound struct {
	Key string
}

func (e ErrNotFound) Error() string {
	return "not found: " + e.Key
}

// ErrExpired is returned when a locked rate has expired
type ErrExpired struct {
	LockID string
}

func (e ErrExpired) Error() string {
	return "rate lock expired: " + e.LockID
}
