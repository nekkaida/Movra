package provider

import (
	"context"
	"time"

	"github.com/movra/settlement-service/internal/model"
)

// ProviderResult represents the result of a payout operation
type ProviderResult struct {
	ProviderReference string
	Status            model.PayoutStatus
	FailureReason     string
	PickupCode        string
	PickupExpiresAt   *time.Time
}

// ProviderStatus represents the status from a provider check
type ProviderStatus struct {
	Status        model.PayoutStatus
	FailureReason string
	CompletedAt   *time.Time
}

// PayoutProvider defines the interface for payout providers
type PayoutProvider interface {
	// ProcessPayout initiates a payout with the provider
	ProcessPayout(ctx context.Context, payout *model.Payout) (*ProviderResult, error)

	// CheckStatus checks the current status of a payout
	CheckStatus(ctx context.Context, providerReference string) (*ProviderStatus, error)

	// CancelPayout cancels a pending/processing payout
	CancelPayout(ctx context.Context, providerReference string) error

	// Name returns the provider name
	Name() string
}
