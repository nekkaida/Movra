package model

import (
	"time"
)

// ExchangeRate represents a currency exchange rate
type ExchangeRate struct {
	SourceCurrency   string    `json:"sourceCurrency"`
	TargetCurrency   string    `json:"targetCurrency"`
	MidRate          float64   `json:"midRate"`          // Mid-market rate (raw)
	Rate             string    `json:"rate"`             // Mid-market rate (string for API)
	BuyRate          string    `json:"buyRate"`          // Rate we offer (includes margin)
	BidRate          float64   `json:"bidRate"`          // Rate to buy target currency
	AskRate          float64   `json:"askRate"`          // Rate to sell target currency
	Spread           float64   `json:"spread"`           // Spread percentage
	MarginPercentage string    `json:"marginPercentage"`
	Source           string    `json:"source"`           // Provider name
	FetchedAt        time.Time `json:"fetchedAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

// LockedRate represents a rate that has been locked for a transfer
type LockedRate struct {
	LockID    string       `json:"lockId"`
	Rate      ExchangeRate `json:"rate"`
	LockedAt  time.Time    `json:"lockedAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
	Expired   bool         `json:"expired"`
}

// Corridor represents a currency corridor configuration
type Corridor struct {
	SourceCurrency   string   `json:"sourceCurrency"`
	TargetCurrency   string   `json:"targetCurrency"`
	Enabled          bool     `json:"enabled"`
	FeePercentage    string   `json:"feePercentage"`
	FeeMinimum       Money    `json:"feeMinimum"`
	MarginPercentage string   `json:"marginPercentage"`
	PayoutMethods    []string `json:"payoutMethods"`
}

// Money represents a monetary amount
type Money struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

// RateLockRequest represents a request to lock a rate
type RateLockRequest struct {
	SourceCurrency  string `json:"sourceCurrency" binding:"required"`
	TargetCurrency  string `json:"targetCurrency" binding:"required"`
	DurationSeconds int    `json:"durationSeconds"`
}

// RateQuote represents a customer-facing rate quote with fees
type RateQuote struct {
	SourceCurrency   string    `json:"sourceCurrency"`
	TargetCurrency   string    `json:"targetCurrency"`
	SourceAmount     float64   `json:"sourceAmount"`     // Amount in source currency
	TargetAmount     float64   `json:"targetAmount"`     // Amount in target currency after conversion
	ExchangeRate     float64   `json:"exchangeRate"`     // Rate applied (includes margin)
	MidMarketRate    float64   `json:"midMarketRate"`    // Transparent mid-market rate
	Fee              float64   `json:"fee"`              // Fee in source currency
	TotalCost        float64   `json:"totalCost"`        // SourceAmount + Fee
	ValidUntil       time.Time `json:"validUntil"`       // When this quote expires
	QuoteID          string    `json:"quoteId"`          // Unique identifier for this quote
}

// Corridors is a list of all supported corridors
var Corridors = []Corridor{
	{
		SourceCurrency:   "SGD",
		TargetCurrency:   "PHP",
		Enabled:          true,
		FeePercentage:    "0.5",
		FeeMinimum:       Money{Currency: "SGD", Amount: "3.00"},
		MarginPercentage: "0.3",
		PayoutMethods:    []string{"BANK_ACCOUNT", "MOBILE_WALLET", "CASH_PICKUP"},
	},
	{
		SourceCurrency:   "SGD",
		TargetCurrency:   "INR",
		Enabled:          true,
		FeePercentage:    "0.5",
		FeeMinimum:       Money{Currency: "SGD", Amount: "3.00"},
		MarginPercentage: "0.35",
		PayoutMethods:    []string{"BANK_ACCOUNT", "MOBILE_WALLET"},
	},
	{
		SourceCurrency:   "SGD",
		TargetCurrency:   "IDR",
		Enabled:          true,
		FeePercentage:    "0.5",
		FeeMinimum:       Money{Currency: "SGD", Amount: "3.00"},
		MarginPercentage: "0.3",
		PayoutMethods:    []string{"BANK_ACCOUNT", "MOBILE_WALLET"},
	},
	{
		SourceCurrency:   "USD",
		TargetCurrency:   "PHP",
		Enabled:          true,
		FeePercentage:    "0.4",
		FeeMinimum:       Money{Currency: "USD", Amount: "2.00"},
		MarginPercentage: "0.25",
		PayoutMethods:    []string{"BANK_ACCOUNT", "MOBILE_WALLET", "CASH_PICKUP"},
	},
	{
		SourceCurrency:   "SGD",
		TargetCurrency:   "USD",
		Enabled:          true,
		FeePercentage:    "0.3",
		FeeMinimum:       Money{Currency: "SGD", Amount: "2.00"},
		MarginPercentage: "0.2",
		PayoutMethods:    []string{"BANK_ACCOUNT"},
	},
}
