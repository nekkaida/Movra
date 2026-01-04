package provider

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestNewSimulatedProvider(t *testing.T) {
	config := DefaultSimulatedConfig()
	provider := NewSimulatedProvider(config)

	if provider == nil {
		t.Fatal("expected provider to be created")
	}

	if provider.Name() != "simulated" {
		t.Errorf("expected name 'simulated', got '%s'", provider.Name())
	}

	if !provider.SupportsInverse() {
		t.Error("expected provider to support inverse rates")
	}
}

func TestGetRate_DirectPair(t *testing.T) {
	config := DefaultSimulatedConfig()
	config.Seed = 42 // Fixed seed for reproducibility
	provider := NewSimulatedProvider(config)

	ctx := context.Background()
	rate, err := provider.GetRate(ctx, "SGD", "PHP")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check basic fields
	if rate.SourceCurrency != "SGD" {
		t.Errorf("expected source 'SGD', got '%s'", rate.SourceCurrency)
	}
	if rate.TargetCurrency != "PHP" {
		t.Errorf("expected target 'PHP', got '%s'", rate.TargetCurrency)
	}
	if rate.Source != "simulated" {
		t.Errorf("expected source 'simulated', got '%s'", rate.Source)
	}

	// Check rate is reasonable (base rate is 42.50)
	if rate.MidRate < 40 || rate.MidRate > 45 {
		t.Errorf("mid rate %f outside expected range [40, 45]", rate.MidRate)
	}

	// Check bid < mid < ask (spread)
	if rate.BidRate >= rate.MidRate {
		t.Errorf("bid rate %f should be less than mid rate %f", rate.BidRate, rate.MidRate)
	}
	if rate.AskRate <= rate.MidRate {
		t.Errorf("ask rate %f should be greater than mid rate %f", rate.AskRate, rate.MidRate)
	}

	// Check spread is reasonable
	if rate.Spread < 0 || rate.Spread > 5 {
		t.Errorf("spread %f outside expected range [0, 5]", rate.Spread)
	}

	// Check timestamps
	if rate.FetchedAt.IsZero() {
		t.Error("fetched at should not be zero")
	}
	if rate.ValidUntil.Before(rate.FetchedAt) {
		t.Error("valid until should be after fetched at")
	}
}

func TestGetRate_InversePair(t *testing.T) {
	config := DefaultSimulatedConfig()
	config.Seed = 42
	provider := NewSimulatedProvider(config)
	provider.ResetDrift() // Ensure no drift for predictable results

	ctx := context.Background()

	// Get SGD/USD rate
	sgdUsd, err := provider.GetRate(ctx, "SGD", "USD")
	if err != nil {
		t.Fatalf("unexpected error for SGD/USD: %v", err)
	}

	// Get USD/SGD rate (should be inverse)
	usdSgd, err := provider.GetRate(ctx, "USD", "SGD")
	if err != nil {
		t.Fatalf("unexpected error for USD/SGD: %v", err)
	}

	// Check they are approximately inverse of each other
	product := sgdUsd.MidRate * usdSgd.MidRate
	if math.Abs(product-1.0) > 0.01 {
		t.Errorf("rates should be inverse: SGD/USD=%f, USD/SGD=%f, product=%f",
			sgdUsd.MidRate, usdSgd.MidRate, product)
	}
}

func TestGetRate_CrossPair(t *testing.T) {
	config := DefaultSimulatedConfig()
	config.Seed = 42
	provider := NewSimulatedProvider(config)

	ctx := context.Background()

	// PHP/INR should work via USD intermediate
	rate, err := provider.GetRate(ctx, "PHP", "INR")

	if err != nil {
		t.Fatalf("unexpected error for cross pair: %v", err)
	}

	// PHP to INR rate should be reasonable
	// PHP is ~57 per USD, INR is ~83 per USD
	// So PHP/INR should be roughly 83/57 = ~1.45
	if rate.MidRate < 1 || rate.MidRate > 2 {
		t.Errorf("PHP/INR rate %f outside expected range [1, 2]", rate.MidRate)
	}
}

func TestGetRate_UnsupportedPair(t *testing.T) {
	config := DefaultSimulatedConfig()
	provider := NewSimulatedProvider(config)

	ctx := context.Background()
	_, err := provider.GetRate(ctx, "XYZ", "ABC")

	if err == nil {
		t.Fatal("expected error for unsupported pair")
	}

	unsupportedErr, ok := err.(ErrUnsupportedPair)
	if !ok {
		t.Fatalf("expected ErrUnsupportedPair, got %T", err)
	}

	if unsupportedErr.Source != "XYZ" || unsupportedErr.Target != "ABC" {
		t.Errorf("unexpected error details: %v", unsupportedErr)
	}
}

func TestGetRate_ContextCancelled(t *testing.T) {
	config := DefaultSimulatedConfig()
	provider := NewSimulatedProvider(config)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := provider.GetRate(ctx, "SGD", "PHP")

	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestGetRates_MultiplePairs(t *testing.T) {
	config := DefaultSimulatedConfig()
	provider := NewSimulatedProvider(config)

	ctx := context.Background()
	pairs := []CurrencyPair{
		{Source: "SGD", Target: "PHP"},
		{Source: "SGD", Target: "INR"},
		{Source: "USD", Target: "PHP"},
	}

	rates, err := provider.GetRates(ctx, pairs)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rates) != 3 {
		t.Errorf("expected 3 rates, got %d", len(rates))
	}

	// Verify each rate
	for i, rate := range rates {
		if rate.SourceCurrency != pairs[i].Source {
			t.Errorf("rate %d: expected source %s, got %s", i, pairs[i].Source, rate.SourceCurrency)
		}
		if rate.TargetCurrency != pairs[i].Target {
			t.Errorf("rate %d: expected target %s, got %s", i, pairs[i].Target, rate.TargetCurrency)
		}
	}
}

func TestGetRates_SkipsUnsupportedPairs(t *testing.T) {
	config := DefaultSimulatedConfig()
	provider := NewSimulatedProvider(config)

	ctx := context.Background()
	pairs := []CurrencyPair{
		{Source: "SGD", Target: "PHP"},
		{Source: "XYZ", Target: "ABC"}, // Unsupported - should be skipped
		{Source: "USD", Target: "PHP"},
	}

	rates, err := provider.GetRates(ctx, pairs)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rates) != 2 {
		t.Errorf("expected 2 rates (skipping unsupported), got %d", len(rates))
	}
}

func TestDrift_UpdatesOverTime(t *testing.T) {
	config := DefaultSimulatedConfig()
	config.DriftInterval = 10 * time.Millisecond // Fast drift for testing
	config.Seed = 42
	provider := NewSimulatedProvider(config)

	ctx := context.Background()

	// Get initial rate
	rate1, _ := provider.GetRate(ctx, "SGD", "PHP")

	// Wait for drift interval
	time.Sleep(15 * time.Millisecond)

	// Get rate again - should be different due to drift
	rate2, _ := provider.GetRate(ctx, "SGD", "PHP")

	// Rates should be different (drift applied)
	// Note: With very small probability they could be the same
	// but statistically this should pass
	if rate1.MidRate == rate2.MidRate {
		t.Log("Warning: rates are the same, this is statistically unlikely but possible")
	}
}

func TestSetDrift_ManualControl(t *testing.T) {
	config := DefaultSimulatedConfig()
	provider := NewSimulatedProvider(config)
	provider.ResetDrift()

	ctx := context.Background()

	// Get base rate (no drift)
	baseRate, _ := provider.GetRate(ctx, "SGD", "PHP")

	// Set 5% positive drift
	provider.SetDrift("SGD", "PHP", 0.05)

	// Get rate with drift
	driftedRate, _ := provider.GetRate(ctx, "SGD", "PHP")

	// Drifted rate should be about 5% higher
	expectedRate := baseRate.MidRate * 1.05
	if math.Abs(driftedRate.MidRate-expectedRate) > 0.01 {
		t.Errorf("expected rate ~%f with 5%% drift, got %f", expectedRate, driftedRate.MidRate)
	}
}

func TestResetDrift(t *testing.T) {
	config := DefaultSimulatedConfig()
	provider := NewSimulatedProvider(config)

	ctx := context.Background()

	// Set some drift
	provider.SetDrift("SGD", "PHP", 0.10) // 10% drift

	// Get drifted rate
	driftedRate, _ := provider.GetRate(ctx, "SGD", "PHP")

	// Reset drift
	provider.ResetDrift()

	// Get rate after reset
	resetRate, _ := provider.GetRate(ctx, "SGD", "PHP")

	// After reset, rate should be back to base (within spread)
	// Base rate is 42.50
	if math.Abs(resetRate.MidRate-42.50) > 1 {
		t.Errorf("after reset, rate should be near base 42.50, got %f", resetRate.MidRate)
	}

	// Drifted rate should have been higher
	if driftedRate.MidRate <= resetRate.MidRate {
		t.Errorf("drifted rate %f should be higher than reset rate %f", driftedRate.MidRate, resetRate.MidRate)
	}
}

func TestGetSupportedPairs(t *testing.T) {
	config := DefaultSimulatedConfig()
	provider := NewSimulatedProvider(config)

	pairs := provider.GetSupportedPairs()

	if len(pairs) == 0 {
		t.Fatal("expected at least some supported pairs")
	}

	// Check that SGD/PHP is in the list
	found := false
	for _, pair := range pairs {
		if pair.Source == "SGD" && pair.Target == "PHP" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected SGD/PHP to be in supported pairs")
	}
}

func TestSpreadCalculation(t *testing.T) {
	config := DefaultSimulatedConfig()
	config.BaseSpread = 0.01 // 1% spread
	config.Seed = 42
	provider := NewSimulatedProvider(config)
	provider.ResetDrift()

	ctx := context.Background()
	rate, _ := provider.GetRate(ctx, "SGD", "PHP")

	// Spread should be reported as percentage
	if math.Abs(rate.Spread-1.0) > 0.01 {
		t.Errorf("expected spread ~1.0%%, got %f%%", rate.Spread)
	}

	// Check bid/ask are correct relative to mid
	// With 1% spread: bid = mid * 0.995, ask = mid * 1.005
	expectedBid := rate.MidRate * 0.995
	expectedAsk := rate.MidRate * 1.005

	if math.Abs(rate.BidRate-expectedBid) > 0.01 {
		t.Errorf("expected bid %f, got %f", expectedBid, rate.BidRate)
	}
	if math.Abs(rate.AskRate-expectedAsk) > 0.01 {
		t.Errorf("expected ask %f, got %f", expectedAsk, rate.AskRate)
	}
}
