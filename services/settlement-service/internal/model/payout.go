package model

import (
	"time"
)

// PayoutStatus represents the status of a payout
type PayoutStatus string

const (
	PayoutStatusPending        PayoutStatus = "PENDING"
	PayoutStatusProcessing     PayoutStatus = "PROCESSING"
	PayoutStatusCompleted      PayoutStatus = "COMPLETED"
	PayoutStatusFailed         PayoutStatus = "FAILED"
	PayoutStatusCancelled      PayoutStatus = "CANCELLED"
	PayoutStatusReadyForPickup PayoutStatus = "READY_FOR_PICKUP"
	PayoutStatusPickedUp       PayoutStatus = "PICKED_UP"
)

// PayoutMethod represents the payout method
type PayoutMethod string

const (
	PayoutMethodBankAccount  PayoutMethod = "BANK_ACCOUNT"
	PayoutMethodMobileWallet PayoutMethod = "MOBILE_WALLET"
	PayoutMethodCashPickup   PayoutMethod = "CASH_PICKUP"
)

// Payout represents a payout record
type Payout struct {
	ID                 string       `json:"id"`
	TransferID         string       `json:"transferId"`
	Status             PayoutStatus `json:"status"`
	Method             PayoutMethod `json:"method"`
	Amount             string       `json:"amount"`
	Currency           string       `json:"currency"`
	Recipient          Recipient    `json:"recipient"`
	ProviderReference  string       `json:"providerReference,omitempty"`
	BatchID            string       `json:"batchId,omitempty"`
	PickupCode         string       `json:"pickupCode,omitempty"`
	PickupExpiresAt    *time.Time   `json:"pickupExpiresAt,omitempty"`
	FailureReason      string       `json:"failureReason,omitempty"`
	RetryCount         int          `json:"retryCount"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
	CompletedAt        *time.Time   `json:"completedAt,omitempty"`
}

// Recipient holds recipient details
type Recipient struct {
	Type           PayoutMethod `json:"type"`
	BankName       string       `json:"bankName,omitempty"`
	BankCode       string       `json:"bankCode,omitempty"`
	AccountNumber  string       `json:"accountNumber,omitempty"`
	AccountName    string       `json:"accountName,omitempty"`
	WalletProvider string       `json:"walletProvider,omitempty"`
	MobileNumber   string       `json:"mobileNumber,omitempty"`
	FirstName      string       `json:"firstName,omitempty"`
	LastName       string       `json:"lastName,omitempty"`
	Country        string       `json:"country,omitempty"`
}

// PayoutBatch represents a batch of payouts
type PayoutBatch struct {
	ID              string       `json:"id"`
	Method          PayoutMethod `json:"method"`
	Currency        string       `json:"currency"`
	TotalPayouts    int          `json:"totalPayouts"`
	CompletedPayouts int         `json:"completedPayouts"`
	FailedPayouts   int          `json:"failedPayouts"`
	TotalAmount     string       `json:"totalAmount"`
	CreatedAt       time.Time    `json:"createdAt"`
	CompletedAt     *time.Time   `json:"completedAt,omitempty"`
}
