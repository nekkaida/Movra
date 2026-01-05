package service

import (
	"context"
	"fmt"
	"time"

	"github.com/movra/settlement-service/internal/model"
	"github.com/movra/settlement-service/internal/provider"
	"github.com/movra/settlement-service/internal/repository"
	"go.uber.org/zap"
)

// PayoutService handles payout business logic
type PayoutService struct {
	repo       repository.PayoutRepository
	provider   provider.PayoutProvider
	logger     *zap.Logger
	maxRetries int
}

// NewPayoutService creates a new payout service
func NewPayoutService(
	repo repository.PayoutRepository,
	prov provider.PayoutProvider,
	logger *zap.Logger,
	maxRetries int,
) *PayoutService {
	return &PayoutService{
		repo:       repo,
		provider:   prov,
		logger:     logger,
		maxRetries: maxRetries,
	}
}

// InitiatePayout creates and processes a new payout
func (s *PayoutService) InitiatePayout(ctx context.Context, req *InitiatePayoutRequest) (*model.Payout, error) {
	now := time.Now()

	payout := &model.Payout{
		ID:         fmt.Sprintf("payout_%d", now.UnixNano()),
		TransferID: req.TransferID,
		Status:     model.PayoutStatusPending,
		Method:     req.Method,
		Amount:     req.Amount,
		Currency:   req.Currency,
		Recipient:  req.Recipient,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Save initial payout
	if err := s.repo.SavePayout(ctx, payout); err != nil {
		return nil, fmt.Errorf("save payout: %w", err)
	}

	// Process payout
	if err := s.processPayout(ctx, payout); err != nil {
		s.logger.Error("Failed to process payout",
			zap.String("payoutId", payout.ID),
			zap.Error(err),
		)
		// Don't return error - payout is saved with failed status
	}

	// Reload to get updated status
	return s.repo.GetPayout(ctx, payout.ID)
}

// GetPayout retrieves a payout by ID
func (s *PayoutService) GetPayout(ctx context.Context, id string) (*model.Payout, error) {
	return s.repo.GetPayout(ctx, id)
}

// ListPayouts retrieves payouts with filters
func (s *PayoutService) ListPayouts(ctx context.Context, filter repository.PayoutFilter) ([]*model.Payout, error) {
	return s.repo.ListPayouts(ctx, filter)
}

// RetryPayout retries a failed payout
func (s *PayoutService) RetryPayout(ctx context.Context, id string) (*model.Payout, error) {
	payout, err := s.repo.GetPayout(ctx, id)
	if err != nil {
		return nil, err
	}

	if payout.Status != model.PayoutStatusFailed {
		return nil, fmt.Errorf("can only retry failed payouts, current status: %s", payout.Status)
	}

	if payout.RetryCount >= s.maxRetries {
		return nil, fmt.Errorf("max retries (%d) exceeded", s.maxRetries)
	}

	// Reset for retry
	payout.Status = model.PayoutStatusPending
	payout.RetryCount++
	payout.FailureReason = ""
	payout.UpdatedAt = time.Now()

	if err := s.repo.SavePayout(ctx, payout); err != nil {
		return nil, fmt.Errorf("save payout for retry: %w", err)
	}

	// Process again
	if err := s.processPayout(ctx, payout); err != nil {
		s.logger.Error("Failed to process payout retry",
			zap.String("payoutId", payout.ID),
			zap.Int("retryCount", payout.RetryCount),
			zap.Error(err),
		)
	}

	return s.repo.GetPayout(ctx, payout.ID)
}

// CancelPayout cancels a pending payout
func (s *PayoutService) CancelPayout(ctx context.Context, id string, reason string) (*model.Payout, error) {
	payout, err := s.repo.GetPayout(ctx, id)
	if err != nil {
		return nil, err
	}

	if payout.Status != model.PayoutStatusPending && payout.Status != model.PayoutStatusFailed {
		return nil, fmt.Errorf("can only cancel pending or failed payouts, current status: %s", payout.Status)
	}

	// If has provider reference, cancel with provider
	if payout.ProviderReference != "" {
		if err := s.provider.CancelPayout(ctx, payout.ProviderReference); err != nil {
			s.logger.Warn("Failed to cancel with provider",
				zap.String("payoutId", id),
				zap.Error(err),
			)
		}
	}

	payout.Status = model.PayoutStatusCancelled
	payout.FailureReason = reason
	payout.UpdatedAt = time.Now()

	if err := s.repo.SavePayout(ctx, payout); err != nil {
		return nil, fmt.Errorf("save cancelled payout: %w", err)
	}

	return payout, nil
}

// GetPickupCode returns pickup code for cash pickup payouts
func (s *PayoutService) GetPickupCode(ctx context.Context, id string) (string, *time.Time, error) {
	payout, err := s.repo.GetPayout(ctx, id)
	if err != nil {
		return "", nil, err
	}

	if payout.Method != model.PayoutMethodCashPickup {
		return "", nil, fmt.Errorf("not a cash pickup payout")
	}

	if payout.PickupCode == "" {
		return "", nil, fmt.Errorf("pickup code not yet available")
	}

	return payout.PickupCode, payout.PickupExpiresAt, nil
}

func (s *PayoutService) processPayout(ctx context.Context, payout *model.Payout) error {
	// Update to processing
	payout.Status = model.PayoutStatusProcessing
	payout.UpdatedAt = time.Now()
	if err := s.repo.SavePayout(ctx, payout); err != nil {
		return fmt.Errorf("update to processing: %w", err)
	}

	// Call provider
	result, err := s.provider.ProcessPayout(ctx, payout)
	if err != nil {
		payout.Status = model.PayoutStatusFailed
		payout.FailureReason = err.Error()
		payout.UpdatedAt = time.Now()
		s.repo.SavePayout(ctx, payout)
		return fmt.Errorf("provider error: %w", err)
	}

	// Update with result
	payout.ProviderReference = result.ProviderReference
	payout.Status = result.Status
	payout.FailureReason = result.FailureReason
	payout.PickupCode = result.PickupCode
	payout.PickupExpiresAt = result.PickupExpiresAt
	payout.UpdatedAt = time.Now()

	if result.Status == model.PayoutStatusCompleted {
		now := time.Now()
		payout.CompletedAt = &now
	}

	if err := s.repo.SavePayout(ctx, payout); err != nil {
		return fmt.Errorf("save result: %w", err)
	}

	s.logger.Info("Payout processed",
		zap.String("payoutId", payout.ID),
		zap.String("status", string(payout.Status)),
		zap.String("providerRef", payout.ProviderReference),
	)

	return nil
}

// InitiatePayoutRequest represents a request to initiate a payout
type InitiatePayoutRequest struct {
	TransferID string
	Method     model.PayoutMethod
	Amount     string
	Currency   string
	Recipient  model.Recipient
}
