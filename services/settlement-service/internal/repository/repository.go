package repository

import (
	"context"

	"github.com/movra/settlement-service/internal/model"
)

// PayoutRepository defines the interface for payout storage
type PayoutRepository interface {
	// SavePayout saves or updates a payout
	SavePayout(ctx context.Context, payout *model.Payout) error

	// GetPayout retrieves a payout by ID
	GetPayout(ctx context.Context, id string) (*model.Payout, error)

	// GetPayoutByTransferID retrieves a payout by transfer ID
	GetPayoutByTransferID(ctx context.Context, transferID string) (*model.Payout, error)

	// ListPayouts retrieves payouts with optional filters
	ListPayouts(ctx context.Context, filter PayoutFilter) ([]*model.Payout, error)

	// UpdatePayoutStatus updates only the status and related fields
	UpdatePayoutStatus(ctx context.Context, id string, status model.PayoutStatus, failureReason string) error
}

// PayoutFilter defines filters for listing payouts
type PayoutFilter struct {
	Status  model.PayoutStatus
	Method  model.PayoutMethod
	BatchID string
	Limit   int
	Offset  int
}
