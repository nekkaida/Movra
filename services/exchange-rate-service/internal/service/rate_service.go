package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/config"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/model"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/provider"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/repository"
	"go.uber.org/zap"
)

// RateService handles exchange rate operations
type RateService struct {
	config     *config.Config
	provider   provider.RateProvider
	repository repository.RateRepository
	logger     *zap.Logger
}

// NewRateService creates a new RateService with dependency injection
func NewRateService(
	cfg *config.Config,
	rateProvider provider.RateProvider,
	rateRepo repository.RateRepository,
	logger *zap.Logger,
) *RateService {
	return &RateService{
		config:     cfg,
		provider:   rateProvider,
		repository: rateRepo,
		logger:     logger,
	}
}

// GetRate retrieves the current exchange rate for a currency pair
func (s *RateService) GetRate(ctx context.Context, from, to string) (*model.ExchangeRate, error) {
	// Try to get from cache first
	cachedRate, err := s.repository.GetRate(ctx, from, to)
	if err != nil {
		s.logger.Warn("Cache lookup failed", zap.Error(err))
		// Continue to fetch from provider
	}

	if cachedRate != nil {
		s.logger.Debug("Rate cache hit",
			zap.String("from", from),
			zap.String("to", to),
			zap.String("source", cachedRate.Source),
		)
		return s.providerRateToModel(cachedRate, from, to), nil
	}

	// Fetch from provider
	rate, err := s.provider.GetRate(ctx, from, to)
	if err != nil {
		s.logger.Error("Failed to fetch rate from provider",
			zap.String("from", from),
			zap.String("to", to),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get rate for %s/%s: %w", from, to, err)
	}

	// Cache the rate
	cacheTTL := time.Duration(s.config.RateCacheTTL) * time.Second
	if err := s.repository.SaveRate(ctx, rate, cacheTTL); err != nil {
		s.logger.Warn("Failed to cache rate", zap.Error(err))
		// Don't fail the request, just log
	}

	s.logger.Info("Fetched rate from provider",
		zap.String("from", from),
		zap.String("to", to),
		zap.Float64("midRate", rate.MidRate),
		zap.String("source", rate.Source),
	)

	return s.providerRateToModel(rate, from, to), nil
}

// GetRates retrieves exchange rates for multiple currency pairs
func (s *RateService) GetRates(ctx context.Context, pairs []provider.CurrencyPair) ([]*model.ExchangeRate, error) {
	results := make([]*model.ExchangeRate, 0, len(pairs))
	uncachedPairs := make([]provider.CurrencyPair, 0)

	// Check cache for each pair
	for _, pair := range pairs {
		cachedRate, err := s.repository.GetRate(ctx, pair.Source, pair.Target)
		if err == nil && cachedRate != nil {
			results = append(results, s.providerRateToModel(cachedRate, pair.Source, pair.Target))
		} else {
			uncachedPairs = append(uncachedPairs, pair)
		}
	}

	// Fetch uncached rates from provider
	if len(uncachedPairs) > 0 {
		rates, err := s.provider.GetRates(ctx, uncachedPairs)
		if err != nil {
			return nil, fmt.Errorf("failed to get rates: %w", err)
		}

		cacheTTL := time.Duration(s.config.RateCacheTTL) * time.Second
		for _, rate := range rates {
			// Cache each rate
			if err := s.repository.SaveRate(ctx, rate, cacheTTL); err != nil {
				s.logger.Warn("Failed to cache rate", zap.Error(err))
			}
			results = append(results, s.providerRateToModel(rate, rate.SourceCurrency, rate.TargetCurrency))
		}
	}

	return results, nil
}

// LockRate locks a rate for a specified duration
func (s *RateService) LockRate(ctx context.Context, from, to string, durationSeconds int) (*model.LockedRate, error) {
	// Validate and cap duration
	if durationSeconds <= 0 {
		durationSeconds = s.config.LockDuration
	}
	if durationSeconds > 120 {
		durationSeconds = 120 // Max 2 minutes
	}

	// Get current rate
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

	// Store in repository
	if err := s.repository.SaveLockedRate(ctx, locked); err != nil {
		return nil, fmt.Errorf("failed to lock rate: %w", err)
	}

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
	locked, err := s.repository.GetLockedRate(ctx, lockID)
	if err != nil {
		if _, ok := err.(repository.ErrExpired); ok {
			return &model.LockedRate{
				LockID:  lockID,
				Expired: true,
			}, nil
		}
		return nil, err
	}

	if locked == nil {
		return &model.LockedRate{
			LockID:  lockID,
			Expired: true,
		}, nil
	}

	return locked, nil
}

// DeleteLockedRate removes a locked rate (e.g., after transfer is complete)
func (s *RateService) DeleteLockedRate(ctx context.Context, lockID string) error {
	if err := s.repository.DeleteLockedRate(ctx, lockID); err != nil {
		if _, ok := err.(repository.ErrNotFound); ok {
			return nil // Already deleted or expired
		}
		return err
	}
	return nil
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

// GetQuote generates a customer-facing rate quote
func (s *RateService) GetQuote(ctx context.Context, from, to string, sourceAmount float64) (*model.RateQuote, error) {
	rate, err := s.GetRate(ctx, from, to)
	if err != nil {
		return nil, err
	}

	// Get corridor for fee calculation
	corridor := s.getCorridor(from, to)
	if corridor == nil {
		return nil, fmt.Errorf("corridor not found: %s/%s", from, to)
	}

	// Calculate fee
	feePercent, _ := strconv.ParseFloat(corridor.FeePercentage, 64)
	feeMinAmount, _ := strconv.ParseFloat(corridor.FeeMinimum.Amount, 64)

	fee := sourceAmount * (feePercent / 100)
	if fee < feeMinAmount {
		fee = feeMinAmount
	}

	// Calculate conversion
	buyRate, _ := strconv.ParseFloat(rate.BuyRate, 64)
	targetAmount := sourceAmount * buyRate

	quote := &model.RateQuote{
		SourceCurrency: from,
		TargetCurrency: to,
		SourceAmount:   sourceAmount,
		TargetAmount:   targetAmount,
		ExchangeRate:   buyRate,
		MidMarketRate:  rate.MidRate,
		Fee:            fee,
		TotalCost:      sourceAmount + fee,
		ValidUntil:     rate.ExpiresAt,
		QuoteID:        uuid.New().String(),
	}

	return quote, nil
}

// getCorridor finds the corridor for a currency pair
func (s *RateService) getCorridor(from, to string) *model.Corridor {
	for _, c := range model.Corridors {
		if c.SourceCurrency == from && c.TargetCurrency == to {
			return &c
		}
	}
	return nil
}

// providerRateToModel converts a provider.Rate to model.ExchangeRate
func (s *RateService) providerRateToModel(rate *provider.Rate, from, to string) *model.ExchangeRate {
	// Get margin from corridor config
	margin := s.getMargin(from, to)

	// Calculate buy rate (rate offered to customer, includes margin)
	buyRate := rate.MidRate * (1 - margin)

	return &model.ExchangeRate{
		SourceCurrency:   from,
		TargetCurrency:   to,
		MidRate:          rate.MidRate,
		Rate:             fmt.Sprintf("%.6f", rate.MidRate),
		BuyRate:          fmt.Sprintf("%.6f", buyRate),
		BidRate:          rate.BidRate,
		AskRate:          rate.AskRate,
		Spread:           rate.Spread,
		MarginPercentage: fmt.Sprintf("%.2f", margin*100),
		Source:           rate.Source,
		FetchedAt:        rate.FetchedAt,
		ExpiresAt:        rate.ValidUntil,
	}
}

// getMargin returns the margin for a currency pair from corridor config
func (s *RateService) getMargin(from, to string) float64 {
	for _, c := range model.Corridors {
		if c.SourceCurrency == from && c.TargetCurrency == to {
			margin, _ := strconv.ParseFloat(c.MarginPercentage, 64)
			return margin / 100
		}
	}
	return 0.003 // Default 0.3%
}

// Health checks if the service and its dependencies are healthy
func (s *RateService) Health(ctx context.Context) error {
	return s.repository.Health(ctx)
}
