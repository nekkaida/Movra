package service

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/movra/exchange-rate-service/internal/config"
	"github.com/movra/exchange-rate-service/internal/model"
	"go.uber.org/zap"
)

// RateService handles exchange rate operations
type RateService struct {
	config *config.Config
	redis  *redis.Client
	logger *zap.Logger
}

// NewRateService creates a new RateService
func NewRateService(cfg *config.Config, redisClient *redis.Client, logger *zap.Logger) *RateService {
	return &RateService{
		config: cfg,
		redis:  redisClient,
		logger: logger,
	}
}

// GetRate retrieves the current exchange rate for a currency pair
func (s *RateService) GetRate(ctx context.Context, from, to string) (*model.ExchangeRate, error) {
	cacheKey := fmt.Sprintf("rate:%s:%s", from, to)

	// Try to get from cache
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		s.logger.Debug("Rate cache hit", zap.String("from", from), zap.String("to", to))
		// In real implementation, parse cached JSON
	}

	// Get rate (mock implementation)
	midRate := s.getMidMarketRate(from, to)
	if midRate == 0 {
		return nil, fmt.Errorf("unsupported currency pair: %s/%s", from, to)
	}

	// Add some realistic variance (-0.1% to +0.1%)
	variance := 1 + (rand.Float64()-0.5)*0.002
	midRate *= variance

	// Calculate margin
	margin := s.getMargin(from, to)
	buyRate := midRate * (1 - margin)

	rate := &model.ExchangeRate{
		SourceCurrency:   from,
		TargetCurrency:   to,
		Rate:             fmt.Sprintf("%.6f", midRate),
		BuyRate:          fmt.Sprintf("%.6f", buyRate),
		MarginPercentage: fmt.Sprintf("%.2f", margin*100),
		FetchedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(time.Duration(s.config.RateCacheTTL) * time.Second),
	}

	// Cache the rate
	s.redis.Set(ctx, cacheKey, rate.Rate, time.Duration(s.config.RateCacheTTL)*time.Second)

	s.logger.Info("Fetched rate",
		zap.String("from", from),
		zap.String("to", to),
		zap.String("rate", rate.Rate),
	)

	return rate, nil
}

// LockRate locks a rate for a specified duration
func (s *RateService) LockRate(ctx context.Context, from, to string, durationSeconds int) (*model.LockedRate, error) {
	if durationSeconds <= 0 {
		durationSeconds = s.config.LockDuration
	}
	if durationSeconds > 120 {
		durationSeconds = 120 // Max 2 minutes
	}

	rate, err := s.GetRate(ctx, from, to)
	if err != nil {
		return nil, err
	}

	lockID := uuid.New().String()
	lockedAt := time.Now()
	expiresAt := lockedAt.Add(time.Duration(durationSeconds) * time.Second)

	locked := &model.LockedRate{
		LockID:    lockID,
		Rate:      *rate,
		LockedAt:  lockedAt,
		ExpiresAt: expiresAt,
		Expired:   false,
	}

	// Store in Redis with expiry
	cacheKey := fmt.Sprintf("lock:%s", lockID)
	s.redis.Set(ctx, cacheKey, rate.BuyRate, time.Duration(durationSeconds)*time.Second)

	s.logger.Info("Rate locked",
		zap.String("lockId", lockID),
		zap.String("from", from),
		zap.String("to", to),
		zap.Int("duration", durationSeconds),
	)

	return locked, nil
}

// GetLockedRate retrieves a previously locked rate
func (s *RateService) GetLockedRate(ctx context.Context, lockID string) (*model.LockedRate, error) {
	cacheKey := fmt.Sprintf("lock:%s", lockID)

	buyRate, err := s.redis.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		return &model.LockedRate{
			LockID:  lockID,
			Expired: true,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	ttl, _ := s.redis.TTL(ctx, cacheKey).Result()

	rate, _ := strconv.ParseFloat(buyRate, 64)

	locked := &model.LockedRate{
		LockID: lockID,
		Rate: model.ExchangeRate{
			BuyRate: buyRate,
			Rate:    fmt.Sprintf("%.6f", rate/(1-0.003)), // Approximate mid-rate
		},
		LockedAt:  time.Now().Add(-time.Duration(s.config.LockDuration)*time.Second + ttl),
		ExpiresAt: time.Now().Add(ttl),
		Expired:   false,
	}

	return locked, nil
}

// GetCorridors returns all available corridors
func (s *RateService) GetCorridors(sourceCurrency string) []model.Corridor {
	if sourceCurrency == "" {
		return model.Corridors
	}

	var filtered []model.Corridor
	for _, c := range model.Corridors {
		if c.SourceCurrency == sourceCurrency {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// Mock mid-market rates
func (s *RateService) getMidMarketRate(from, to string) float64 {
	rates := map[string]map[string]float64{
		"SGD": {"PHP": 39.75, "INR": 62.50, "IDR": 11800, "USD": 0.74},
		"USD": {"PHP": 53.75, "SGD": 1.35},
		"PHP": {"SGD": 0.0252},
	}

	if fromRates, ok := rates[from]; ok {
		if rate, ok := fromRates[to]; ok {
			return rate
		}
	}
	return 0
}

// Get margin for a currency pair
func (s *RateService) getMargin(from, to string) float64 {
	for _, c := range model.Corridors {
		if c.SourceCurrency == from && c.TargetCurrency == to {
			margin, _ := strconv.ParseFloat(c.MarginPercentage, 64)
			return margin / 100
		}
	}
	return 0.003 // Default 0.3%
}
