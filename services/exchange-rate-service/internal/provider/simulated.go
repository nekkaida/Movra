package provider

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// baseRates contains realistic mid-market rates as of 2024
// These are used as the foundation for simulated rates
var baseRates = map[string]float64{
	// SGD pairs (Singapore Dollar)
	"SGD/USD": 0.7450,
	"SGD/PHP": 42.50,
	"SGD/INR": 62.30,
	"SGD/IDR": 11650.00,
	"SGD/MYR": 3.48,
	"SGD/THB": 26.50,
	"SGD/VND": 18200.00,

	// USD pairs (US Dollar)
	"USD/SGD": 1.3423,
	"USD/PHP": 57.05,
	"USD/INR": 83.60,
	"USD/IDR": 15630.00,
	"USD/MYR": 4.67,
	"USD/THB": 35.55,
	"USD/VND": 24430.00,
	"USD/EUR": 0.9250,
	"USD/GBP": 0.7920,

	// EUR pairs (Euro)
	"EUR/USD": 1.0810,
	"EUR/SGD": 1.4510,
	"EUR/GBP": 0.8560,

	// GBP pairs (British Pound)
	"GBP/USD": 1.2630,
	"GBP/SGD": 1.6950,
	"GBP/EUR": 1.1680,
}

// SimulatedProviderConfig configures the simulated rate provider
type SimulatedProviderConfig struct {
	// BaseSpread is the base spread percentage (default 0.5%)
	BaseSpread float64

	// MaxDrift is the maximum random drift percentage (default 2%)
	MaxDrift float64

	// RateValidityDuration is how long rates are valid (default 30 seconds)
	RateValidityDuration time.Duration

	// DriftInterval is how often rates drift (default 5 seconds)
	DriftInterval time.Duration

	// Seed for random number generator (0 for current time)
	Seed int64
}

// DefaultSimulatedConfig returns default configuration
func DefaultSimulatedConfig() SimulatedProviderConfig {
	return SimulatedProviderConfig{
		BaseSpread:           0.005, // 0.5%
		MaxDrift:             0.02,  // 2%
		RateValidityDuration: 30 * time.Second,
		DriftInterval:        5 * time.Second,
		Seed:                 0,
	}
}

// SimulatedProvider provides simulated exchange rates
// It uses base rates with configurable drift and spread
type SimulatedProvider struct {
	config       SimulatedProviderConfig
	rng          *rand.Rand
	mu           sync.RWMutex
	currentDrift map[string]float64 // Current drift per pair
	lastDrift    time.Time          // When drift was last updated
}

// NewSimulatedProvider creates a new simulated rate provider
func NewSimulatedProvider(config SimulatedProviderConfig) *SimulatedProvider {
	seed := config.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return &SimulatedProvider{
		config:       config,
		rng:          rand.New(rand.NewSource(seed)),
		currentDrift: make(map[string]float64),
		lastDrift:    time.Time{},
	}
}

// Name returns the provider name
func (p *SimulatedProvider) Name() string {
	return "simulated"
}

// SupportsInverse returns true - simulated provider can calculate inverse rates
func (p *SimulatedProvider) SupportsInverse() bool {
	return true
}

// GetRate returns the exchange rate for a single currency pair
func (p *SimulatedProvider) GetRate(ctx context.Context, source, target string) (*Rate, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.updateDriftIfNeeded()

	midRate, err := p.getMidRate(source, target)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	spread := p.config.BaseSpread

	// Calculate bid/ask with spread
	// Bid = rate to buy target (lower)
	// Ask = rate to sell target (higher)
	bidRate := midRate * (1 - spread/2)
	askRate := midRate * (1 + spread/2)

	return &Rate{
		SourceCurrency: source,
		TargetCurrency: target,
		MidRate:        midRate,
		BidRate:        bidRate,
		AskRate:        askRate,
		Spread:         spread * 100, // Convert to percentage
		Source:         p.Name(),
		FetchedAt:      now,
		ValidUntil:     now.Add(p.config.RateValidityDuration),
	}, nil
}

// GetRates returns exchange rates for multiple currency pairs
func (p *SimulatedProvider) GetRates(ctx context.Context, pairs []CurrencyPair) ([]*Rate, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	rates := make([]*Rate, 0, len(pairs))
	for _, pair := range pairs {
		rate, err := p.GetRate(ctx, pair.Source, pair.Target)
		if err != nil {
			// Skip unsupported pairs, continue with others
			if _, ok := err.(ErrUnsupportedPair); ok {
				continue
			}
			return nil, err
		}
		rates = append(rates, rate)
	}

	return rates, nil
}

// getMidRate returns the mid-market rate with drift applied
func (p *SimulatedProvider) getMidRate(source, target string) (float64, error) {
	directKey := source + "/" + target
	inverseKey := target + "/" + source

	p.mu.RLock()
	drift := p.currentDrift[directKey]
	p.mu.RUnlock()

	// Check direct rate
	if baseRate, ok := baseRates[directKey]; ok {
		return baseRate * (1 + drift), nil
	}

	// Check inverse rate
	if baseRate, ok := baseRates[inverseKey]; ok {
		// For inverse, we need to invert both the rate and apply inverse drift
		p.mu.RLock()
		inverseDrift := p.currentDrift[inverseKey]
		p.mu.RUnlock()
		return 1.0 / (baseRate * (1 + inverseDrift)), nil
	}

	// Try to calculate via USD as intermediate
	if source != "USD" && target != "USD" {
		sourceToUSD, errSource := p.getMidRate(source, "USD")
		usdToTarget, errTarget := p.getMidRate("USD", target)
		if errSource == nil && errTarget == nil {
			return sourceToUSD * usdToTarget, nil
		}
	}

	return 0, ErrUnsupportedPair{Source: source, Target: target}
}

// updateDriftIfNeeded updates rate drift if enough time has passed
func (p *SimulatedProvider) updateDriftIfNeeded() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.lastDrift) < p.config.DriftInterval {
		return
	}

	// Update drift for all base pairs
	for pair := range baseRates {
		// Random drift between -MaxDrift and +MaxDrift
		drift := (p.rng.Float64()*2 - 1) * p.config.MaxDrift
		p.currentDrift[pair] = drift
	}

	p.lastDrift = time.Now()
}

// GetSupportedPairs returns all supported currency pairs
func (p *SimulatedProvider) GetSupportedPairs() []CurrencyPair {
	pairs := make([]CurrencyPair, 0, len(baseRates))
	for key := range baseRates {
		// Parse "SGD/PHP" format
		var source, target string
		for i, c := range key {
			if c == '/' {
				source = key[:i]
				target = key[i+1:]
				break
			}
		}
		if source != "" && target != "" {
			pairs = append(pairs, CurrencyPair{Source: source, Target: target})
		}
	}
	return pairs
}

// SetDrift manually sets drift for a currency pair (useful for testing)
func (p *SimulatedProvider) SetDrift(source, target string, drift float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentDrift[source+"/"+target] = drift
}

// ResetDrift resets all drift to zero
func (p *SimulatedProvider) ResetDrift() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentDrift = make(map[string]float64)
	p.lastDrift = time.Time{}
}
