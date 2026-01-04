package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/patteeraL/movra/services/exchange-rate-service/internal/config"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/model"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/provider"
	"go.uber.org/zap"
)

// MockProvider implements provider.RateProvider for testing
type MockProvider struct {
	GetRateFunc  func(ctx context.Context, source, target string) (*provider.Rate, error)
	GetRatesFunc func(ctx context.Context, pairs []provider.CurrencyPair) ([]*provider.Rate, error)
}

func (m *MockProvider) GetRate(ctx context.Context, source, target string) (*provider.Rate, error) {
	if m.GetRateFunc != nil {
		return m.GetRateFunc(ctx, source, target)
	}
	return &provider.Rate{
		SourceCurrency: source,
		TargetCurrency: target,
		MidRate:        42.50,
		BidRate:        42.29,
		AskRate:        42.71,
		Spread:         0.5,
		Source:         "mock",
		FetchedAt:      time.Now(),
		ValidUntil:     time.Now().Add(30 * time.Second),
	}, nil
}

func (m *MockProvider) GetRates(ctx context.Context, pairs []provider.CurrencyPair) ([]*provider.Rate, error) {
	if m.GetRatesFunc != nil {
		return m.GetRatesFunc(ctx, pairs)
	}
	rates := make([]*provider.Rate, 0, len(pairs))
	for _, pair := range pairs {
		rate, err := m.GetRate(ctx, pair.Source, pair.Target)
		if err != nil {
			continue
		}
		rates = append(rates, rate)
	}
	return rates, nil
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) SupportsInverse() bool {
	return true
}

// MockRepository implements repository.RateRepository for testing
type MockRepository struct {
	rates       map[string]*provider.Rate
	lockedRates map[string]*model.LockedRate
	SaveRateFunc      func(ctx context.Context, rate *provider.Rate, ttl time.Duration) error
	GetRateFunc       func(ctx context.Context, source, target string) (*provider.Rate, error)
	SaveLockedFunc    func(ctx context.Context, locked *model.LockedRate) error
	GetLockedFunc     func(ctx context.Context, lockID string) (*model.LockedRate, error)
	DeleteLockedFunc  func(ctx context.Context, lockID string) error
	ExtendLockedFunc  func(ctx context.Context, lockID string, newExpiry time.Time) error
	HealthFunc        func(ctx context.Context) error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		rates:       make(map[string]*provider.Rate),
		lockedRates: make(map[string]*model.LockedRate),
	}
}

func (m *MockRepository) SaveRate(ctx context.Context, rate *provider.Rate, ttl time.Duration) error {
	if m.SaveRateFunc != nil {
		return m.SaveRateFunc(ctx, rate, ttl)
	}
	key := rate.SourceCurrency + ":" + rate.TargetCurrency
	m.rates[key] = rate
	return nil
}

func (m *MockRepository) GetRate(ctx context.Context, source, target string) (*provider.Rate, error) {
	if m.GetRateFunc != nil {
		return m.GetRateFunc(ctx, source, target)
	}
	key := source + ":" + target
	rate, ok := m.rates[key]
	if !ok {
		return nil, nil // Cache miss
	}
	return rate, nil
}

func (m *MockRepository) SaveLockedRate(ctx context.Context, locked *model.LockedRate) error {
	if m.SaveLockedFunc != nil {
		return m.SaveLockedFunc(ctx, locked)
	}
	m.lockedRates[locked.LockID] = locked
	return nil
}

func (m *MockRepository) GetLockedRate(ctx context.Context, lockID string) (*model.LockedRate, error) {
	if m.GetLockedFunc != nil {
		return m.GetLockedFunc(ctx, lockID)
	}
	locked, ok := m.lockedRates[lockID]
	if !ok {
		return nil, nil
	}
	return locked, nil
}

func (m *MockRepository) DeleteLockedRate(ctx context.Context, lockID string) error {
	if m.DeleteLockedFunc != nil {
		return m.DeleteLockedFunc(ctx, lockID)
	}
	delete(m.lockedRates, lockID)
	return nil
}

func (m *MockRepository) ExtendLockedRate(ctx context.Context, lockID string, newExpiry time.Time) error {
	if m.ExtendLockedFunc != nil {
		return m.ExtendLockedFunc(ctx, lockID, newExpiry)
	}
	if locked, ok := m.lockedRates[lockID]; ok {
		locked.ExpiresAt = newExpiry
		return nil
	}
	return errors.New("not found")
}

func (m *MockRepository) Health(ctx context.Context) error {
	if m.HealthFunc != nil {
		return m.HealthFunc(ctx)
	}
	return nil
}

func newTestService() (*RateService, *MockProvider, *MockRepository) {
	cfg := &config.Config{
		RateCacheTTL: 30,
		LockDuration: 60,
	}
	mockProvider := &MockProvider{}
	mockRepo := NewMockRepository()
	logger := zap.NewNop()

	svc := NewRateService(cfg, mockProvider, mockRepo, logger)
	return svc, mockProvider, mockRepo
}

func TestGetRate_CacheMiss_FetchesFromProvider(t *testing.T) {
	svc, mockProvider, _ := newTestService()

	providerCalled := false
	mockProvider.GetRateFunc = func(ctx context.Context, source, target string) (*provider.Rate, error) {
		providerCalled = true
		return &provider.Rate{
			SourceCurrency: source,
			TargetCurrency: target,
			MidRate:        42.50,
			BidRate:        42.29,
			AskRate:        42.71,
			Spread:         0.5,
			Source:         "mock",
			FetchedAt:      time.Now(),
			ValidUntil:     time.Now().Add(30 * time.Second),
		}, nil
	}

	ctx := context.Background()
	rate, err := svc.GetRate(ctx, "SGD", "PHP")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !providerCalled {
		t.Error("expected provider to be called on cache miss")
	}

	if rate.SourceCurrency != "SGD" || rate.TargetCurrency != "PHP" {
		t.Errorf("unexpected currencies: %s/%s", rate.SourceCurrency, rate.TargetCurrency)
	}

	if rate.MidRate != 42.50 {
		t.Errorf("expected mid rate 42.50, got %f", rate.MidRate)
	}
}

func TestGetRate_CacheHit_ReturnsFromCache(t *testing.T) {
	svc, mockProvider, mockRepo := newTestService()

	// Pre-populate cache
	cachedRate := &provider.Rate{
		SourceCurrency: "SGD",
		TargetCurrency: "PHP",
		MidRate:        42.00, // Different from provider
		BidRate:        41.79,
		AskRate:        42.21,
		Spread:         0.5,
		Source:         "cached",
		FetchedAt:      time.Now(),
		ValidUntil:     time.Now().Add(30 * time.Second),
	}
	mockRepo.rates["SGD:PHP"] = cachedRate

	providerCalled := false
	mockProvider.GetRateFunc = func(ctx context.Context, source, target string) (*provider.Rate, error) {
		providerCalled = true
		return &provider.Rate{MidRate: 50.00}, nil
	}

	ctx := context.Background()
	rate, err := svc.GetRate(ctx, "SGD", "PHP")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if providerCalled {
		t.Error("expected provider NOT to be called on cache hit")
	}

	if rate.MidRate != 42.00 {
		t.Errorf("expected cached mid rate 42.00, got %f", rate.MidRate)
	}

	if rate.Source != "cached" {
		t.Errorf("expected source 'cached', got %s", rate.Source)
	}
}

func TestGetRate_ProviderError_ReturnsError(t *testing.T) {
	svc, mockProvider, _ := newTestService()

	mockProvider.GetRateFunc = func(ctx context.Context, source, target string) (*provider.Rate, error) {
		return nil, errors.New("provider unavailable")
	}

	ctx := context.Background()
	_, err := svc.GetRate(ctx, "SGD", "PHP")

	if err == nil {
		t.Fatal("expected error when provider fails")
	}
}

func TestLockRate_CreatesLock(t *testing.T) {
	svc, _, mockRepo := newTestService()

	ctx := context.Background()
	locked, err := svc.LockRate(ctx, "SGD", "PHP", 60)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if locked.LockID == "" {
		t.Error("expected lock ID to be generated")
	}

	if locked.Expired {
		t.Error("expected lock to not be expired")
	}

	// Verify lock is stored in repository
	stored, _ := mockRepo.GetLockedRate(ctx, locked.LockID)
	if stored == nil {
		t.Error("expected lock to be stored in repository")
	}
}

func TestLockRate_CapsMaxDuration(t *testing.T) {
	svc, _, _ := newTestService()

	ctx := context.Background()
	locked, err := svc.LockRate(ctx, "SGD", "PHP", 300) // Request 5 minutes

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be capped at 120 seconds (2 minutes)
	duration := locked.ExpiresAt.Sub(locked.LockedAt)
	if duration > 121*time.Second {
		t.Errorf("expected duration to be capped at 120s, got %v", duration)
	}
}

func TestGetLockedRate_ReturnsLock(t *testing.T) {
	svc, _, mockRepo := newTestService()

	// Create a lock first
	ctx := context.Background()
	created, _ := svc.LockRate(ctx, "SGD", "PHP", 60)

	// Retrieve it
	retrieved, err := svc.GetLockedRate(ctx, created.LockID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.LockID != created.LockID {
		t.Errorf("expected lock ID %s, got %s", created.LockID, retrieved.LockID)
	}

	// Also verify via mock
	_ = mockRepo
}

func TestGetLockedRate_ExpiredLock_ReturnsExpired(t *testing.T) {
	svc, _, mockRepo := newTestService()

	// Don't create a lock - simulate not found
	ctx := context.Background()
	retrieved, err := svc.GetLockedRate(ctx, "nonexistent-lock-id")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !retrieved.Expired {
		t.Error("expected non-existent lock to be marked as expired")
	}

	_ = mockRepo
}

func TestDeleteLockedRate_RemovesLock(t *testing.T) {
	svc, _, mockRepo := newTestService()

	ctx := context.Background()

	// Create a lock
	created, _ := svc.LockRate(ctx, "SGD", "PHP", 60)

	// Verify it exists
	_, exists := mockRepo.lockedRates[created.LockID]
	if !exists {
		t.Fatal("expected lock to exist before deletion")
	}

	// Delete it
	err := svc.DeleteLockedRate(ctx, created.LockID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	_, exists = mockRepo.lockedRates[created.LockID]
	if exists {
		t.Error("expected lock to be deleted")
	}
}

func TestGetCorridors_All(t *testing.T) {
	svc, _, _ := newTestService()

	corridors := svc.GetCorridors("")

	if len(corridors) == 0 {
		t.Error("expected at least some corridors")
	}
}

func TestGetCorridors_FilteredBySource(t *testing.T) {
	svc, _, _ := newTestService()

	corridors := svc.GetCorridors("SGD")

	if len(corridors) == 0 {
		t.Error("expected SGD corridors")
	}

	for _, c := range corridors {
		if c.SourceCurrency != "SGD" {
			t.Errorf("expected all corridors to have SGD source, got %s", c.SourceCurrency)
		}
	}
}

func TestGetQuote_CalculatesFees(t *testing.T) {
	svc, _, _ := newTestService()

	ctx := context.Background()
	quote, err := svc.GetQuote(ctx, "SGD", "PHP", 100.0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if quote.SourceAmount != 100.0 {
		t.Errorf("expected source amount 100.0, got %f", quote.SourceAmount)
	}

	// Fee should be at least the minimum (SGD 3.00 for SGD/PHP)
	if quote.Fee < 3.0 {
		t.Errorf("expected fee >= 3.0, got %f", quote.Fee)
	}

	// Total cost should be source + fee
	expectedTotal := quote.SourceAmount + quote.Fee
	if quote.TotalCost != expectedTotal {
		t.Errorf("expected total cost %f, got %f", expectedTotal, quote.TotalCost)
	}

	// Target amount should be positive
	if quote.TargetAmount <= 0 {
		t.Errorf("expected positive target amount, got %f", quote.TargetAmount)
	}
}

func TestHealth_ChecksRepository(t *testing.T) {
	svc, _, mockRepo := newTestService()

	ctx := context.Background()

	// Healthy case
	err := svc.Health(ctx)
	if err != nil {
		t.Errorf("expected healthy service, got error: %v", err)
	}

	// Unhealthy case
	mockRepo.HealthFunc = func(ctx context.Context) error {
		return errors.New("redis connection failed")
	}

	err = svc.Health(ctx)
	if err == nil {
		t.Error("expected error when repository is unhealthy")
	}
}
