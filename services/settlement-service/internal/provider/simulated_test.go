package provider

import (
	"context"
	"testing"
	"time"

	"github.com/movra/settlement-service/internal/model"
)

func TestSimulatedProvider_ProcessPayout_Success(t *testing.T) {
	provider := NewSimulatedProvider(0, 10*time.Millisecond) // 0% failure rate

	payout := &model.Payout{
		ID:       "test_payout_1",
		Method:   model.PayoutMethodBankAccount,
		Amount:   "100.00",
		Currency: "SGD",
	}

	result, err := provider.ProcessPayout(context.Background(), payout)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Status != model.PayoutStatusCompleted {
		t.Errorf("expected status COMPLETED, got: %s", result.Status)
	}

	if result.ProviderReference == "" {
		t.Error("expected provider reference to be set")
	}
}

func TestSimulatedProvider_ProcessPayout_CashPickup(t *testing.T) {
	provider := NewSimulatedProvider(0, 10*time.Millisecond)

	payout := &model.Payout{
		ID:     "test_payout_2",
		Method: model.PayoutMethodCashPickup,
	}

	result, err := provider.ProcessPayout(context.Background(), payout)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Status != model.PayoutStatusReadyForPickup {
		t.Errorf("expected status READY_FOR_PICKUP, got: %s", result.Status)
	}

	if result.PickupCode == "" {
		t.Error("expected pickup code to be set")
	}

	if len(result.PickupCode) != 8 {
		t.Errorf("expected 8-digit pickup code, got: %d digits", len(result.PickupCode))
	}

	if result.PickupExpiresAt == nil {
		t.Error("expected pickup expiry to be set")
	}
}

func TestSimulatedProvider_ProcessPayout_Failure(t *testing.T) {
	provider := NewSimulatedProvider(100, 10*time.Millisecond) // 100% failure rate

	payout := &model.Payout{
		ID:     "test_payout_3",
		Method: model.PayoutMethodBankAccount,
	}

	result, err := provider.ProcessPayout(context.Background(), payout)
	if err != nil {
		t.Fatalf("expected no error (failure is returned in result), got: %v", err)
	}

	if result.Status != model.PayoutStatusFailed {
		t.Errorf("expected status FAILED, got: %s", result.Status)
	}

	if result.FailureReason == "" {
		t.Error("expected failure reason to be set")
	}
}

func TestSimulatedProvider_ProcessPayout_ContextCancellation(t *testing.T) {
	provider := NewSimulatedProvider(0, 5*time.Second) // Long processing time

	payout := &model.Payout{
		ID:     "test_payout_4",
		Method: model.PayoutMethodBankAccount,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := provider.ProcessPayout(ctx, payout)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestSimulatedProvider_CheckStatus(t *testing.T) {
	provider := NewSimulatedProvider(0, 10*time.Millisecond)

	status, err := provider.CheckStatus(context.Background(), "SIM_123")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if status.Status != model.PayoutStatusCompleted {
		t.Errorf("expected status COMPLETED, got: %s", status.Status)
	}
}

func TestSimulatedProvider_CancelPayout(t *testing.T) {
	provider := NewSimulatedProvider(0, 10*time.Millisecond)

	err := provider.CancelPayout(context.Background(), "SIM_123")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestSimulatedProvider_Name(t *testing.T) {
	provider := NewSimulatedProvider(0, 10*time.Millisecond)

	if provider.Name() != "simulated" {
		t.Errorf("expected name 'simulated', got: %s", provider.Name())
	}
}
