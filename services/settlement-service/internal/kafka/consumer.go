package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/movra/settlement-service/internal/model"
	"github.com/movra/settlement-service/internal/service"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// TransferFundedEvent represents the event when a transfer is funded
type TransferFundedEvent struct {
	TransferID   string         `json:"transferId"`
	Amount       string         `json:"amount"`
	Currency     string         `json:"currency"`
	PayoutMethod string         `json:"payoutMethod"`
	Recipient    RecipientEvent `json:"recipient"`
}

// RecipientEvent represents recipient details in the event
type RecipientEvent struct {
	Type           string `json:"type"`
	BankName       string `json:"bankName,omitempty"`
	BankCode       string `json:"bankCode,omitempty"`
	AccountNumber  string `json:"accountNumber,omitempty"`
	AccountName    string `json:"accountName,omitempty"`
	WalletProvider string `json:"walletProvider,omitempty"`
	MobileNumber   string `json:"mobileNumber,omitempty"`
	FirstName      string `json:"firstName,omitempty"`
	LastName       string `json:"lastName,omitempty"`
	Country        string `json:"country,omitempty"`
}

// Consumer consumes transfer.funded events and initiates payouts
type Consumer struct {
	reader  *kafka.Reader
	service *service.PayoutService
	logger  *zap.Logger
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(brokers string, topic string, groupID string, svc *service.PayoutService, logger *zap.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokers},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{
		reader:  reader,
		service: svc,
		logger:  logger,
	}
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting Kafka consumer")

	for {
		select {
		case <-ctx.Done():
			return c.reader.Close()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				c.logger.Error("Failed to read message", zap.Error(err))
				continue
			}

			if err := c.handleMessage(ctx, msg); err != nil {
				c.logger.Error("Failed to handle message",
					zap.String("topic", msg.Topic),
					zap.Int64("offset", msg.Offset),
					zap.Error(err),
				)
			}
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	var event TransferFundedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	c.logger.Info("Received transfer.funded event",
		zap.String("transferId", event.TransferID),
		zap.String("amount", event.Amount),
		zap.String("currency", event.Currency),
	)

	_, err := c.service.InitiatePayout(ctx, &service.InitiatePayoutRequest{
		TransferID: event.TransferID,
		Method:     parsePayoutMethod(event.PayoutMethod),
		Amount:     event.Amount,
		Currency:   event.Currency,
		Recipient:  eventRecipientToModel(event.Recipient),
	})
	if err != nil {
		return fmt.Errorf("initiate payout: %w", err)
	}

	return nil
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}

func parsePayoutMethod(s string) model.PayoutMethod {
	switch s {
	case "BANK_ACCOUNT":
		return model.PayoutMethodBankAccount
	case "MOBILE_WALLET":
		return model.PayoutMethodMobileWallet
	case "CASH_PICKUP":
		return model.PayoutMethodCashPickup
	default:
		return model.PayoutMethodBankAccount
	}
}

func eventRecipientToModel(r RecipientEvent) model.Recipient {
	return model.Recipient{
		Type:           parsePayoutMethod(r.Type),
		BankName:       r.BankName,
		BankCode:       r.BankCode,
		AccountNumber:  r.AccountNumber,
		AccountName:    r.AccountName,
		WalletProvider: r.WalletProvider,
		MobileNumber:   r.MobileNumber,
		FirstName:      r.FirstName,
		LastName:       r.LastName,
		Country:        r.Country,
	}
}
