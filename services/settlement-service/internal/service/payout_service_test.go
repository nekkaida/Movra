package service

import (
	"context"
	"testing"
	"time"

	"github.com/movra/settlement-service/internal/model"
	"github.com/movra/settlement-service/internal/provider"
	"github.com/movra/settlement-service/internal/repository"
	"go.uber.org/zap"
)

// MockRepository is a simple in-memory repository for testing
type MockRepository struct {
	payouts map[string]*model.Payout
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		payouts: make(map[string]*model.Payout),
	}
}

func (r *MockRepository) SavePayout(ctx context.Context, payout *model.Payout) error {
	r.payouts[payout.ID] = payout
	return nil
}

func (r *MockRepository) GetPayout(ctx context.Context, id string) (*model.Payout, error) {
	if p, ok := r.payouts[id]; ok {
		return p, nil
	}
	return nil, nil
}

func (r *MockRepository) GetPayoutByTransferID(ctx context.Context, transferID string) (*model.Payout, error) {
	for _, p := range r.payouts {
		if p.TransferID == transferID {
			return p, nil
		}
	}
	return nil, nil
}

func (r *MockRepository) ListPayouts(ctx context.Context, filter repository.PayoutFilter) ([]*model.Payout, error) {
	var result []*model.Payout
	for _, p := range r.payouts {
		if filter.Status != "" && p.Status != filter.Status {
			continue
		}
		result = append(result, p)
	}
	return result, nil
}

func (r *MockRepository) UpdatePayoutStatus(ctx context.Context, id string, status model.PayoutStatus, failureReason string) error {
	if p, ok := r.payouts[id]; ok {
		p.Status = status
		p.FailureReason = failureReason
	}
	return nil
}

func TestPayoutService_InitiatePayout_Success(t *testing.T) {
	repo := NewMockRepository()
	prov := provider.NewSimulatedProvider(0, 10*time.Millisecond)
	logger, _ := zap.NewDevelopment()

	svc := NewPayoutService(repo, prov, logger, 3)

	payout, err := svc.InitiatePayout(context.Background(), &InitiatePayoutRequest{
		TransferID: "transfer_123",
		Method:     model.PayoutMethodBankAccount,
		Amount:     "100.00",
		Currency:   "SGD",
		Recipient: model.Recipient{
			Type:          model.PayoutMethodBankAccount,
			BankName:      "Test Bank",
			AccountNumber: "1234567890",
		},
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if payout == nil {
		t.Fatal("expected payout to be returned")
	}

	if payout.Status != model.PayoutStatusCompleted {
		t.Errorf("expected status COMPLETED, got: %s", payout.Status)
	}
}

func TestPayoutService_InitiatePayout_CashPickup(t *testing.T) {
	repo := NewMockRepository()
	prov := provider.NewSimulatedProvider(0, 10*time.Millisecond)
	logger, _ := zap.NewDevelopment()

	svc := NewPayoutService(repo, prov, logger, 3)

	payout, err := svc.InitiatePayout(context.Background(), &InitiatePayoutRequest{
		TransferID: "transfer_456",
		Method:     model.PayoutMethodCashPickup,
		Amount:     "50.00",
		Currency:   "PHP",
		Recipient: model.Recipient{
			Type:      model.PayoutMethodCashPickup,
			FirstName: "John",
			LastName:  "Doe",
			Country:   "PH",
		},
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if payout.Status != model.PayoutStatusReadyForPickup {
		t.Errorf("expected status READY_FOR_PICKUP, got: %s", payout.Status)
	}

	if payout.PickupCode == "" {
		t.Error("expected pickup code to be set")
	}
}

func TestPayoutService_GetPayout(t *testing.T) {
	repo := NewMockRepository()
	prov := provider.NewSimulatedProvider(0, 10*time.Millisecond)
	logger, _ := zap.NewDevelopment()

	svc := NewPayoutService(repo, prov, logger, 3)

	// Create a payout first
	created, _ := svc.InitiatePayout(context.Background(), &InitiatePayoutRequest{
		TransferID: "transfer_789",
		Method:     model.PayoutMethodBankAccount,
		Amount:     "200.00",
		Currency:   "SGD",
	})

	// Retrieve it
	payout, err := svc.GetPayout(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if payout.ID != created.ID {
		t.Errorf("expected ID %s, got: %s", created.ID, payout.ID)
	}
}

func TestPayoutService_CancelPayout(t *testing.T) {
	repo := NewMockRepository()
	prov := provider.NewSimulatedProvider(100, 10*time.Millisecond) // 100% failure to get a failed payout
	logger, _ := zap.NewDevelopment()

	svc := NewPayoutService(repo, prov, logger, 3)

	// Create a payout that will fail
	created, _ := svc.InitiatePayout(context.Background(), &InitiatePayoutRequest{
		TransferID: "transfer_cancel",
		Method:     model.PayoutMethodBankAccount,
		Amount:     "100.00",
		Currency:   "SGD",
	})

	// Cancel it
	cancelled, err := svc.CancelPayout(context.Background(), created.ID, "Customer requested")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cancelled.Status != model.PayoutStatusCancelled {
		t.Errorf("expected status CANCELLED, got: %s", cancelled.Status)
	}
}

func TestPayoutService_GetPickupCode(t *testing.T) {
	repo := NewMockRepository()
	prov := provider.NewSimulatedProvider(0, 10*time.Millisecond)
	logger, _ := zap.NewDevelopment()

	svc := NewPayoutService(repo, prov, logger, 3)

	// Create a cash pickup payout
	created, _ := svc.InitiatePayout(context.Background(), &InitiatePayoutRequest{
		TransferID: "transfer_pickup",
		Method:     model.PayoutMethodCashPickup,
		Amount:     "100.00",
		Currency:   "PHP",
	})

	// Get pickup code
	code, expiresAt, err := svc.GetPickupCode(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if code == "" {
		t.Error("expected pickup code")
	}

	if expiresAt == nil {
		t.Error("expected expiry time")
	}
}
