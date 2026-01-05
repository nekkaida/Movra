package provider

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/movra/settlement-service/internal/model"
)

// SimulatedProvider simulates payout processing for development/testing
type SimulatedProvider struct {
	failureRate    int // percentage 0-100
	processingTime time.Duration
}

// NewSimulatedProvider creates a new simulated provider
func NewSimulatedProvider(failureRate int, processingTime time.Duration) *SimulatedProvider {
	return &SimulatedProvider{
		failureRate:    failureRate,
		processingTime: processingTime,
	}
}

func (p *SimulatedProvider) Name() string {
	return "simulated"
}

func (p *SimulatedProvider) ProcessPayout(ctx context.Context, payout *model.Payout) (*ProviderResult, error) {
	// Simulate processing time
	select {
	case <-time.After(p.processingTime):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Generate provider reference
	providerRef := fmt.Sprintf("SIM_%d", time.Now().UnixNano())

	// Simulate random failures
	if p.shouldFail() {
		return &ProviderResult{
			ProviderReference: providerRef,
			Status:            model.PayoutStatusFailed,
			FailureReason:     "Simulated failure: recipient account not found",
		}, nil
	}

	result := &ProviderResult{
		ProviderReference: providerRef,
		Status:            model.PayoutStatusCompleted,
	}

	// For cash pickup, generate pickup code
	if payout.Method == model.PayoutMethodCashPickup {
		result.Status = model.PayoutStatusReadyForPickup
		result.PickupCode = p.generatePickupCode()
		expiresAt := time.Now().Add(72 * time.Hour)
		result.PickupExpiresAt = &expiresAt
	}

	return result, nil
}

func (p *SimulatedProvider) CheckStatus(ctx context.Context, providerReference string) (*ProviderStatus, error) {
	// Simulated provider always returns completed for status checks
	now := time.Now()
	return &ProviderStatus{
		Status:      model.PayoutStatusCompleted,
		CompletedAt: &now,
	}, nil
}

func (p *SimulatedProvider) CancelPayout(ctx context.Context, providerReference string) error {
	// Simulated cancellation always succeeds
	return nil
}

func (p *SimulatedProvider) shouldFail() bool {
	if p.failureRate <= 0 {
		return false
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(100))
	return int(n.Int64()) < p.failureRate
}

func (p *SimulatedProvider) generatePickupCode() string {
	const digits = "0123456789"
	code := make([]byte, 8)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		code[i] = digits[n.Int64()]
	}
	return string(code)
}
