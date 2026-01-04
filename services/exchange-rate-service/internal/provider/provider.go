package provider

import (
	"context"
	"time"
)

// CurrencyPair represents a pair of currencies for exchange rate lookup
type CurrencyPair struct {
	Source string
	Target string
}

// Rate represents a raw exchange rate from a provider
type Rate struct {
	SourceCurrency string
	TargetCurrency string
	MidRate        float64   // Mid-market rate
	BidRate        float64   // Rate to buy target currency (what we pay)
	AskRate        float64   // Rate to sell target currency (what customer pays)
	Spread         float64   // Spread percentage
	Source         string    // Provider name
	FetchedAt      time.Time // When the rate was fetched
	ValidUntil     time.Time // When the rate expires
}

// RateProvider defines the interface for exchange rate providers
// This follows the adapter pattern - implementations can be swapped
type RateProvider interface {
	// GetRate returns the exchange rate for a single currency pair
	GetRate(ctx context.Context, source, target string) (*Rate, error)

	// GetRates returns exchange rates for multiple currency pairs
	// More efficient for batch lookups
	GetRates(ctx context.Context, pairs []CurrencyPair) ([]*Rate, error)

	// Name returns the provider name (e.g., "simulated", "openexchangerates")
	Name() string

	// SupportsInverse returns true if the provider can calculate inverse rates
	SupportsInverse() bool
}

// ProviderConfig holds common configuration for providers
type ProviderConfig struct {
	// DefaultSpread is the default spread percentage to apply
	DefaultSpread float64

	// RateValidityDuration is how long a rate is valid
	RateValidityDuration time.Duration

	// SupportedPairs lists the currency pairs this provider supports
	// If empty, all pairs are considered supported
	SupportedPairs []CurrencyPair
}

// ErrUnsupportedPair is returned when a currency pair is not supported
type ErrUnsupportedPair struct {
	Source string
	Target string
}

func (e ErrUnsupportedPair) Error() string {
	return "unsupported currency pair: " + e.Source + "/" + e.Target
}

// ErrProviderUnavailable is returned when the provider cannot be reached
type ErrProviderUnavailable struct {
	Provider string
	Reason   string
}

func (e ErrProviderUnavailable) Error() string {
	return "provider " + e.Provider + " unavailable: " + e.Reason
}
