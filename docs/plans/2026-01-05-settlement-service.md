# Settlement Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a production-ready Settlement Service that processes payouts triggered by funded transfers, integrates with payout providers, and publishes status events.

**Architecture:** Event-driven service consuming `transfer.funded` events from Kafka, processing payouts through a provider interface, storing state in Redis, exposing gRPC for queries/admin, and publishing status updates back to Kafka.

**Tech Stack:** Go 1.21+, gRPC, Kafka (segmentio/kafka-go), Redis, Gin (HTTP health endpoints), Prometheus metrics, Zap logging

---

## Task 1: Provider Interface

**Files:**
- Create: `services/settlement-service/internal/provider/provider.go`

**Step 1: Create provider interface file**

```go
package provider

import (
	"context"
	"time"

	"github.com/patteeraL/movra/services/settlement-service/internal/model"
)

// ProviderResult represents the result of a payout operation
type ProviderResult struct {
	ProviderReference string
	Status            model.PayoutStatus
	FailureReason     string
	PickupCode        string
	PickupExpiresAt   *time.Time
}

// ProviderStatus represents the status from a provider check
type ProviderStatus struct {
	Status        model.PayoutStatus
	FailureReason string
	CompletedAt   *time.Time
}

// PayoutProvider defines the interface for payout providers
type PayoutProvider interface {
	// ProcessPayout initiates a payout with the provider
	ProcessPayout(ctx context.Context, payout *model.Payout) (*ProviderResult, error)

	// CheckStatus checks the current status of a payout
	CheckStatus(ctx context.Context, providerReference string) (*ProviderStatus, error)

	// CancelPayout cancels a pending/processing payout
	CancelPayout(ctx context.Context, providerReference string) error

	// Name returns the provider name
	Name() string
}
```

**Step 2: Verify file compiles**

Run: `cd d:/Codes/Movra/services/settlement-service && go build ./internal/provider/...`
Expected: No errors (or only import errors which we'll fix)

**Step 3: Commit**

```bash
git add services/settlement-service/internal/provider/provider.go
git commit -m "feat(settlement): add PayoutProvider interface"
```

---

## Task 2: Simulated Provider

**Files:**
- Create: `services/settlement-service/internal/provider/simulated.go`

**Step 1: Create simulated provider**

```go
package provider

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/patteeraL/movra/services/settlement-service/internal/model"
)

// SimulatedProvider simulates payout processing for development/testing
type SimulatedProvider struct {
	failureRate    int // percentage 0-100
	processingTime time.Duration
}

// NewSimulatedProvider creates a new simulated provider
func NewSimulatedProvider(failureRate int, processingTime time.Duration) *SimulatedProvider {
	return &SimulatedProvider{
		failureRate:    failureRate,
		processingTime: processingTime,
	}
}

func (p *SimulatedProvider) Name() string {
	return "simulated"
}

func (p *SimulatedProvider) ProcessPayout(ctx context.Context, payout *model.Payout) (*ProviderResult, error) {
	// Simulate processing time
	select {
	case <-time.After(p.processingTime):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Generate provider reference
	providerRef := fmt.Sprintf("SIM_%d", time.Now().UnixNano())

	// Simulate random failures
	if p.shouldFail() {
		return &ProviderResult{
			ProviderReference: providerRef,
			Status:            model.PayoutStatusFailed,
			FailureReason:     "Simulated failure: recipient account not found",
		}, nil
	}

	result := &ProviderResult{
		ProviderReference: providerRef,
		Status:            model.PayoutStatusCompleted,
	}

	// For cash pickup, generate pickup code
	if payout.Method == model.PayoutMethodCashPickup {
		result.Status = model.PayoutStatusReadyForPickup
		result.PickupCode = p.generatePickupCode()
		expiresAt := time.Now().Add(72 * time.Hour)
		result.PickupExpiresAt = &expiresAt
	}

	return result, nil
}

func (p *SimulatedProvider) CheckStatus(ctx context.Context, providerReference string) (*ProviderStatus, error) {
	// Simulated provider always returns completed for status checks
	now := time.Now()
	return &ProviderStatus{
		Status:      model.PayoutStatusCompleted,
		CompletedAt: &now,
	}, nil
}

func (p *SimulatedProvider) CancelPayout(ctx context.Context, providerReference string) error {
	// Simulated cancellation always succeeds
	return nil
}

func (p *SimulatedProvider) shouldFail() bool {
	if p.failureRate <= 0 {
		return false
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(100))
	return int(n.Int64()) < p.failureRate
}

func (p *SimulatedProvider) generatePickupCode() string {
	const digits = "0123456789"
	code := make([]byte, 8)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		code[i] = digits[n.Int64()]
	}
	return string(code)
}
```

**Step 2: Verify compilation**

Run: `cd d:/Codes/Movra/services/settlement-service && go build ./internal/provider/...`
Expected: No errors

**Step 3: Commit**

```bash
git add services/settlement-service/internal/provider/simulated.go
git commit -m "feat(settlement): add SimulatedProvider implementation"
```

---

## Task 3: Repository Interface and Redis Implementation

**Files:**
- Create: `services/settlement-service/internal/repository/repository.go`
- Create: `services/settlement-service/internal/repository/redis_repository.go`

**Step 1: Create repository interface**

```go
package repository

import (
	"context"

	"github.com/patteeraL/movra/services/settlement-service/internal/model"
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
	Status   model.PayoutStatus
	Method   model.PayoutMethod
	BatchID  string
	Limit    int
	Offset   int
}
```

**Step 2: Create Redis implementation**

```go
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/patteeraL/movra/services/settlement-service/internal/model"
	"github.com/redis/go-redis/v9"
)

const (
	payoutKeyPrefix   = "payout:"
	transferKeyPrefix = "payout:transfer:"
	payoutTTL         = 7 * 24 * time.Hour // 7 days
)

// RedisRepository implements PayoutRepository using Redis
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository creates a new Redis repository
func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

func (r *RedisRepository) SavePayout(ctx context.Context, payout *model.Payout) error {
	data, err := json.Marshal(payout)
	if err != nil {
		return fmt.Errorf("marshal payout: %w", err)
	}

	pipe := r.client.Pipeline()

	// Save payout by ID
	pipe.Set(ctx, payoutKeyPrefix+payout.ID, data, payoutTTL)

	// Save index by transfer ID
	pipe.Set(ctx, transferKeyPrefix+payout.TransferID, payout.ID, payoutTTL)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("save payout: %w", err)
	}

	return nil
}

func (r *RedisRepository) GetPayout(ctx context.Context, id string) (*model.Payout, error) {
	data, err := r.client.Get(ctx, payoutKeyPrefix+id).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("payout not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get payout: %w", err)
	}

	var payout model.Payout
	if err := json.Unmarshal(data, &payout); err != nil {
		return nil, fmt.Errorf("unmarshal payout: %w", err)
	}

	return &payout, nil
}

func (r *RedisRepository) GetPayoutByTransferID(ctx context.Context, transferID string) (*model.Payout, error) {
	payoutID, err := r.client.Get(ctx, transferKeyPrefix+transferID).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("payout not found for transfer: %s", transferID)
	}
	if err != nil {
		return nil, fmt.Errorf("get payout by transfer: %w", err)
	}

	return r.GetPayout(ctx, payoutID)
}

func (r *RedisRepository) ListPayouts(ctx context.Context, filter PayoutFilter) ([]*model.Payout, error) {
	// For Redis, we do a simple scan - in production, use a proper index or database
	var cursor uint64
	var payouts []*model.Payout

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, payoutKeyPrefix+"*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan payouts: %w", err)
		}

		for _, key := range keys {
			// Skip index keys
			if len(key) > len(transferKeyPrefix) && key[:len(transferKeyPrefix)] == transferKeyPrefix {
				continue
			}

			data, err := r.client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}

			var payout model.Payout
			if err := json.Unmarshal(data, &payout); err != nil {
				continue
			}

			// Apply filters
			if filter.Status != "" && payout.Status != filter.Status {
				continue
			}
			if filter.Method != "" && payout.Method != filter.Method {
				continue
			}
			if filter.BatchID != "" && payout.BatchID != filter.BatchID {
				continue
			}

			payouts = append(payouts, &payout)

			if filter.Limit > 0 && len(payouts) >= filter.Limit {
				return payouts, nil
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return payouts, nil
}

func (r *RedisRepository) UpdatePayoutStatus(ctx context.Context, id string, status model.PayoutStatus, failureReason string) error {
	payout, err := r.GetPayout(ctx, id)
	if err != nil {
		return err
	}

	payout.Status = status
	payout.FailureReason = failureReason
	payout.UpdatedAt = time.Now()

	if status == model.PayoutStatusCompleted || status == model.PayoutStatusPickedUp {
		now := time.Now()
		payout.CompletedAt = &now
	}

	return r.SavePayout(ctx, payout)
}
```

**Step 3: Verify compilation**

Run: `cd d:/Codes/Movra/services/settlement-service && go get github.com/redis/go-redis/v9 && go build ./internal/repository/...`
Expected: No errors

**Step 4: Commit**

```bash
git add services/settlement-service/internal/repository/
git commit -m "feat(settlement): add PayoutRepository with Redis implementation"
```

---

## Task 4: Configuration

**Files:**
- Create: `services/settlement-service/internal/config/config.go`

**Step 1: Create config file**

```go
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the settlement service
type Config struct {
	// Server
	HTTPPort string
	GRPCPort string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Kafka
	KafkaBrokers       string
	KafkaConsumerGroup string
	KafkaTopicFunded   string
	KafkaTopicStatus   string

	// Provider
	ProviderType           string // "simulated" or future real providers
	ProviderFailureRate    int
	ProviderProcessingTime time.Duration

	// Retry
	MaxRetries    int
	RetryInterval time.Duration
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		HTTPPort: getEnv("HTTP_PORT", "8083"),
		GRPCPort: getEnv("GRPC_PORT", "9083"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		KafkaBrokers:       getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "settlement-service"),
		KafkaTopicFunded:   getEnv("KAFKA_TOPIC_FUNDED", "transfer.funded"),
		KafkaTopicStatus:   getEnv("KAFKA_TOPIC_STATUS", "payout.status"),

		ProviderType:           getEnv("PROVIDER_TYPE", "simulated"),
		ProviderFailureRate:    getEnvInt("PROVIDER_FAILURE_RATE", 10),
		ProviderProcessingTime: getEnvDuration("PROVIDER_PROCESSING_TIME", 2*time.Second),

		MaxRetries:    getEnvInt("MAX_RETRIES", 3),
		RetryInterval: getEnvDuration("RETRY_INTERVAL", 5*time.Second),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
```

**Step 2: Verify compilation**

Run: `cd d:/Codes/Movra/services/settlement-service && go build ./internal/config/...`
Expected: No errors

**Step 3: Commit**

```bash
git add services/settlement-service/internal/config/
git commit -m "feat(settlement): add configuration management"
```

---

## Task 5: Service Layer

**Files:**
- Create: `services/settlement-service/internal/service/payout_service.go`

**Step 1: Create payout service**

```go
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/patteeraL/movra/services/settlement-service/internal/model"
	"github.com/patteeraL/movra/services/settlement-service/internal/provider"
	"github.com/patteeraL/movra/services/settlement-service/internal/repository"
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
```

**Step 2: Verify compilation**

Run: `cd d:/Codes/Movra/services/settlement-service && go build ./internal/service/...`
Expected: No errors

**Step 3: Commit**

```bash
git add services/settlement-service/internal/service/
git commit -m "feat(settlement): add PayoutService with business logic"
```

---

## Task 6: gRPC Server

**Files:**
- Create: `services/settlement-service/internal/grpc/server.go`

**Step 1: Create gRPC server with placeholder types**

```go
package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/patteeraL/movra/services/settlement-service/internal/model"
	"github.com/patteeraL/movra/services/settlement-service/internal/repository"
	"github.com/patteeraL/movra/services/settlement-service/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SettlementServer implements the gRPC SettlementService
type SettlementServer struct {
	UnimplementedSettlementServiceServer
	service *service.PayoutService
	logger  *zap.Logger
}

// NewSettlementServer creates a new gRPC server instance
func NewSettlementServer(svc *service.PayoutService, logger *zap.Logger) *SettlementServer {
	return &SettlementServer{
		service: svc,
		logger:  logger,
	}
}

// InitiatePayout initiates a new payout
func (s *SettlementServer) InitiatePayout(ctx context.Context, req *InitiatePayoutRequest) (*InitiatePayoutResponse, error) {
	if req.TransferId == "" {
		return &InitiatePayoutResponse{
			Error: &Error{Code: "INVALID_ARGUMENT", Message: "transfer_id is required"},
		}, nil
	}

	payout, err := s.service.InitiatePayout(ctx, &service.InitiatePayoutRequest{
		TransferID: req.TransferId,
		Method:     protoMethodToModel(req.Method),
		Amount:     req.Amount.Amount,
		Currency:   req.Amount.Currency,
		Recipient:  protoRecipientToModel(req.Recipient),
	})
	if err != nil {
		s.logger.Error("Failed to initiate payout", zap.Error(err))
		return &InitiatePayoutResponse{
			Error: &Error{Code: "INITIATE_FAILED", Message: err.Error()},
		}, nil
	}

	return &InitiatePayoutResponse{
		Payout: modelPayoutToProto(payout),
	}, nil
}

// GetPayout retrieves a payout by ID
func (s *SettlementServer) GetPayout(ctx context.Context, req *GetPayoutRequest) (*GetPayoutResponse, error) {
	if req.PayoutId == "" {
		return &GetPayoutResponse{
			Error: &Error{Code: "INVALID_ARGUMENT", Message: "payout_id is required"},
		}, nil
	}

	payout, err := s.service.GetPayout(ctx, req.PayoutId)
	if err != nil {
		return &GetPayoutResponse{
			Error: &Error{Code: "NOT_FOUND", Message: err.Error()},
		}, nil
	}

	return &GetPayoutResponse{
		Payout: modelPayoutToProto(payout),
	}, nil
}

// ListPayouts lists payouts with optional filters
func (s *SettlementServer) ListPayouts(ctx context.Context, req *ListPayoutsRequest) (*ListPayoutsResponse, error) {
	filter := repository.PayoutFilter{
		Status:  protoStatusToModel(req.StatusFilter),
		Method:  protoMethodToModel(req.MethodFilter),
		BatchID: req.BatchId,
		Limit:   int(req.Pagination.GetLimit()),
		Offset:  int(req.Pagination.GetOffset()),
	}

	if filter.Limit == 0 {
		filter.Limit = 20
	}

	payouts, err := s.service.ListPayouts(ctx, filter)
	if err != nil {
		return &ListPayoutsResponse{
			Error: &Error{Code: "LIST_FAILED", Message: err.Error()},
		}, nil
	}

	protoPayouts := make([]*Payout, len(payouts))
	for i, p := range payouts {
		protoPayouts[i] = modelPayoutToProto(p)
	}

	return &ListPayoutsResponse{
		Payouts: protoPayouts,
		Pagination: &PaginationResponse{
			Total:  int32(len(payouts)),
			Limit:  int32(filter.Limit),
			Offset: int32(filter.Offset),
		},
	}, nil
}

// RetryPayout retries a failed payout
func (s *SettlementServer) RetryPayout(ctx context.Context, req *RetryPayoutRequest) (*RetryPayoutResponse, error) {
	if req.PayoutId == "" {
		return &RetryPayoutResponse{
			Error: &Error{Code: "INVALID_ARGUMENT", Message: "payout_id is required"},
		}, nil
	}

	payout, err := s.service.RetryPayout(ctx, req.PayoutId)
	if err != nil {
		return &RetryPayoutResponse{
			Error: &Error{Code: "RETRY_FAILED", Message: err.Error()},
		}, nil
	}

	return &RetryPayoutResponse{
		Payout: modelPayoutToProto(payout),
	}, nil
}

// CancelPayout cancels a pending payout
func (s *SettlementServer) CancelPayout(ctx context.Context, req *CancelPayoutRequest) (*CancelPayoutResponse, error) {
	if req.PayoutId == "" {
		return &CancelPayoutResponse{
			Error: &Error{Code: "INVALID_ARGUMENT", Message: "payout_id is required"},
		}, nil
	}

	payout, err := s.service.CancelPayout(ctx, req.PayoutId, req.Reason)
	if err != nil {
		return &CancelPayoutResponse{
			Error: &Error{Code: "CANCEL_FAILED", Message: err.Error()},
		}, nil
	}

	return &CancelPayoutResponse{
		Payout: modelPayoutToProto(payout),
	}, nil
}

// GetPickupCode retrieves the pickup code for a cash pickup payout
func (s *SettlementServer) GetPickupCode(ctx context.Context, req *GetPickupCodeRequest) (*GetPickupCodeResponse, error) {
	if req.PayoutId == "" {
		return &GetPickupCodeResponse{
			Error: &Error{Code: "INVALID_ARGUMENT", Message: "payout_id is required"},
		}, nil
	}

	code, expiresAt, err := s.service.GetPickupCode(ctx, req.PayoutId)
	if err != nil {
		return &GetPickupCodeResponse{
			Error: &Error{Code: "PICKUP_CODE_UNAVAILABLE", Message: err.Error()},
		}, nil
	}

	resp := &GetPickupCodeResponse{
		PickupCode:         code,
		PickupLocationInfo: "Present this code at any authorized pickup location",
	}
	if expiresAt != nil {
		resp.ExpiresAt = timeToProtoTimestamp(*expiresAt)
	}

	return resp, nil
}

// Helper functions

func modelPayoutToProto(p *model.Payout) *Payout {
	payout := &Payout{
		Id:                p.ID,
		TransferId:        p.TransferID,
		Status:            modelStatusToProto(p.Status),
		Method:            modelMethodToProto(p.Method),
		Amount:            &Money{Currency: p.Currency, Amount: p.Amount},
		Currency:          p.Currency,
		Recipient:         modelRecipientToProto(p.Recipient),
		ProviderReference: p.ProviderReference,
		BatchId:           p.BatchID,
		PickupCode:        p.PickupCode,
		FailureReason:     p.FailureReason,
		RetryCount:        int32(p.RetryCount),
		CreatedAt:         timeToProtoTimestamp(p.CreatedAt),
		UpdatedAt:         timeToProtoTimestamp(p.UpdatedAt),
	}
	if p.PickupExpiresAt != nil {
		payout.PickupExpiresAt = timeToProtoTimestamp(*p.PickupExpiresAt)
	}
	if p.CompletedAt != nil {
		payout.CompletedAt = timeToProtoTimestamp(*p.CompletedAt)
	}
	return payout
}

func modelRecipientToProto(r model.Recipient) *RecipientDetails {
	return &RecipientDetails{
		Type:           modelMethodToProto(r.Type),
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

func protoRecipientToModel(r *RecipientDetails) model.Recipient {
	if r == nil {
		return model.Recipient{}
	}
	return model.Recipient{
		Type:           protoMethodToModel(r.Type),
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

func modelStatusToProto(s model.PayoutStatus) PayoutStatus {
	switch s {
	case model.PayoutStatusPending:
		return PayoutStatus_PAYOUT_STATUS_PENDING
	case model.PayoutStatusProcessing:
		return PayoutStatus_PAYOUT_STATUS_PROCESSING
	case model.PayoutStatusCompleted:
		return PayoutStatus_PAYOUT_STATUS_COMPLETED
	case model.PayoutStatusFailed:
		return PayoutStatus_PAYOUT_STATUS_FAILED
	case model.PayoutStatusCancelled:
		return PayoutStatus_PAYOUT_STATUS_CANCELLED
	case model.PayoutStatusReadyForPickup:
		return PayoutStatus_PAYOUT_STATUS_READY_FOR_PICKUP
	case model.PayoutStatusPickedUp:
		return PayoutStatus_PAYOUT_STATUS_PICKED_UP
	default:
		return PayoutStatus_PAYOUT_STATUS_UNSPECIFIED
	}
}

func protoStatusToModel(s PayoutStatus) model.PayoutStatus {
	switch s {
	case PayoutStatus_PAYOUT_STATUS_PENDING:
		return model.PayoutStatusPending
	case PayoutStatus_PAYOUT_STATUS_PROCESSING:
		return model.PayoutStatusProcessing
	case PayoutStatus_PAYOUT_STATUS_COMPLETED:
		return model.PayoutStatusCompleted
	case PayoutStatus_PAYOUT_STATUS_FAILED:
		return model.PayoutStatusFailed
	case PayoutStatus_PAYOUT_STATUS_CANCELLED:
		return model.PayoutStatusCancelled
	case PayoutStatus_PAYOUT_STATUS_READY_FOR_PICKUP:
		return model.PayoutStatusReadyForPickup
	case PayoutStatus_PAYOUT_STATUS_PICKED_UP:
		return model.PayoutStatusPickedUp
	default:
		return ""
	}
}

func modelMethodToProto(m model.PayoutMethod) PayoutMethod {
	switch m {
	case model.PayoutMethodBankAccount:
		return PayoutMethod_PAYOUT_METHOD_BANK_ACCOUNT
	case model.PayoutMethodMobileWallet:
		return PayoutMethod_PAYOUT_METHOD_MOBILE_WALLET
	case model.PayoutMethodCashPickup:
		return PayoutMethod_PAYOUT_METHOD_CASH_PICKUP
	default:
		return PayoutMethod_PAYOUT_METHOD_UNSPECIFIED
	}
}

func protoMethodToModel(m PayoutMethod) model.PayoutMethod {
	switch m {
	case PayoutMethod_PAYOUT_METHOD_BANK_ACCOUNT:
		return model.PayoutMethodBankAccount
	case PayoutMethod_PAYOUT_METHOD_MOBILE_WALLET:
		return model.PayoutMethodMobileWallet
	case PayoutMethod_PAYOUT_METHOD_CASH_PICKUP:
		return model.PayoutMethodCashPickup
	default:
		return ""
	}
}

func timeToProtoTimestamp(t time.Time) *Timestamp {
	return &Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

// Placeholder types for generated proto code

type UnimplementedSettlementServiceServer struct{}

func (UnimplementedSettlementServiceServer) InitiatePayout(context.Context, *InitiatePayoutRequest) (*InitiatePayoutResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitiatePayout not implemented")
}
func (UnimplementedSettlementServiceServer) GetPayout(context.Context, *GetPayoutRequest) (*GetPayoutResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPayout not implemented")
}
func (UnimplementedSettlementServiceServer) ListPayouts(context.Context, *ListPayoutsRequest) (*ListPayoutsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPayouts not implemented")
}
func (UnimplementedSettlementServiceServer) RetryPayout(context.Context, *RetryPayoutRequest) (*RetryPayoutResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RetryPayout not implemented")
}
func (UnimplementedSettlementServiceServer) CancelPayout(context.Context, *CancelPayoutRequest) (*CancelPayoutResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CancelPayout not implemented")
}
func (UnimplementedSettlementServiceServer) GetPickupCode(context.Context, *GetPickupCodeRequest) (*GetPickupCodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPickupCode not implemented")
}
func (UnimplementedSettlementServiceServer) mustEmbedUnimplementedSettlementServiceServer() {}

// Proto message types (placeholders)

type PayoutStatus int32

const (
	PayoutStatus_PAYOUT_STATUS_UNSPECIFIED     PayoutStatus = 0
	PayoutStatus_PAYOUT_STATUS_PENDING         PayoutStatus = 1
	PayoutStatus_PAYOUT_STATUS_PROCESSING      PayoutStatus = 2
	PayoutStatus_PAYOUT_STATUS_COMPLETED       PayoutStatus = 3
	PayoutStatus_PAYOUT_STATUS_FAILED          PayoutStatus = 4
	PayoutStatus_PAYOUT_STATUS_CANCELLED       PayoutStatus = 5
	PayoutStatus_PAYOUT_STATUS_READY_FOR_PICKUP PayoutStatus = 6
	PayoutStatus_PAYOUT_STATUS_PICKED_UP       PayoutStatus = 7
)

type PayoutMethod int32

const (
	PayoutMethod_PAYOUT_METHOD_UNSPECIFIED   PayoutMethod = 0
	PayoutMethod_PAYOUT_METHOD_BANK_ACCOUNT  PayoutMethod = 1
	PayoutMethod_PAYOUT_METHOD_MOBILE_WALLET PayoutMethod = 2
	PayoutMethod_PAYOUT_METHOD_CASH_PICKUP   PayoutMethod = 3
)

type Money struct {
	Currency string
	Amount   string
}

type Timestamp struct {
	Seconds int64
	Nanos   int32
}

type Error struct {
	Code    string
	Message string
	Details map[string]string
}

type Payout struct {
	Id                string
	TransferId        string
	Status            PayoutStatus
	Method            PayoutMethod
	Amount            *Money
	Currency          string
	Recipient         *RecipientDetails
	ProviderReference string
	BatchId           string
	PickupCode        string
	PickupExpiresAt   *Timestamp
	FailureReason     string
	RetryCount        int32
	CreatedAt         *Timestamp
	UpdatedAt         *Timestamp
	CompletedAt       *Timestamp
}

type RecipientDetails struct {
	Type           PayoutMethod
	BankName       string
	BankCode       string
	AccountNumber  string
	AccountName    string
	WalletProvider string
	MobileNumber   string
	FirstName      string
	LastName       string
	Country        string
}

type PaginationRequest struct {
	Limit  int32
	Offset int32
}

func (p *PaginationRequest) GetLimit() int32 {
	if p == nil {
		return 0
	}
	return p.Limit
}

func (p *PaginationRequest) GetOffset() int32 {
	if p == nil {
		return 0
	}
	return p.Offset
}

type PaginationResponse struct {
	Total  int32
	Limit  int32
	Offset int32
}

type InitiatePayoutRequest struct {
	TransferId string
	Amount     *Money
	Method     PayoutMethod
	Recipient  *RecipientDetails
}

type InitiatePayoutResponse struct {
	Payout *Payout
	Error  *Error
}

type GetPayoutRequest struct {
	PayoutId string
}

type GetPayoutResponse struct {
	Payout *Payout
	Error  *Error
}

type ListPayoutsRequest struct {
	Pagination   *PaginationRequest
	StatusFilter PayoutStatus
	MethodFilter PayoutMethod
	BatchId      string
}

type ListPayoutsResponse struct {
	Payouts    []*Payout
	Pagination *PaginationResponse
	Error      *Error
}

type RetryPayoutRequest struct {
	PayoutId string
}

type RetryPayoutResponse struct {
	Payout *Payout
	Error  *Error
}

type CancelPayoutRequest struct {
	PayoutId string
	Reason   string
}

type CancelPayoutResponse struct {
	Payout *Payout
	Error  *Error
}

type GetPickupCodeRequest struct {
	PayoutId string
}

type GetPickupCodeResponse struct {
	PickupCode         string
	ExpiresAt          *Timestamp
	PickupLocationInfo string
	Error              *Error
}

// RegisterSettlementServiceServer registers the server
func RegisterSettlementServiceServer(s interface{}, srv *SettlementServer) {
	fmt.Printf("Registered SettlementServiceServer: %v\n", srv)
}
```

**Step 2: Verify compilation**

Run: `cd d:/Codes/Movra/services/settlement-service && go build ./internal/grpc/...`
Expected: No errors

**Step 3: Commit**

```bash
git add services/settlement-service/internal/grpc/
git commit -m "feat(settlement): add gRPC server with 6 RPCs"
```

---

## Task 7: Kafka Consumer

**Files:**
- Create: `services/settlement-service/internal/kafka/consumer.go`

**Step 1: Create Kafka consumer**

```go
package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/patteeraL/movra/services/settlement-service/internal/model"
	"github.com/patteeraL/movra/services/settlement-service/internal/service"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// TransferFundedEvent represents the event when a transfer is funded
type TransferFundedEvent struct {
	TransferID    string          `json:"transferId"`
	Amount        string          `json:"amount"`
	Currency      string          `json:"currency"`
	PayoutMethod  string          `json:"payoutMethod"`
	Recipient     RecipientEvent  `json:"recipient"`
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
```

**Step 2: Verify compilation**

Run: `cd d:/Codes/Movra/services/settlement-service && go get github.com/segmentio/kafka-go && go build ./internal/kafka/...`
Expected: No errors

**Step 3: Commit**

```bash
git add services/settlement-service/internal/kafka/
git commit -m "feat(settlement): add Kafka consumer for transfer.funded events"
```

---

## Task 8: Wire Dependencies in main.go

**Files:**
- Modify: `services/settlement-service/cmd/server/main.go`

**Step 1: Update main.go with full wiring**

Replace the entire file content:

```go
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patteeraL/movra/services/settlement-service/internal/config"
	settlementgrpc "github.com/patteeraL/movra/services/settlement-service/internal/grpc"
	"github.com/patteeraL/movra/services/settlement-service/internal/kafka"
	"github.com/patteeraL/movra/services/settlement-service/internal/provider"
	"github.com/patteeraL/movra/services/settlement-service/internal/repository"
	"github.com/patteeraL/movra/services/settlement-service/internal/service"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Setup logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Load config
	cfg := config.Load()

	// Setup Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis not available, service will work without persistence", zap.Error(err))
	}

	// Create repository
	repo := repository.NewRedisRepository(redisClient)

	// Create provider
	var payoutProvider provider.PayoutProvider
	switch cfg.ProviderType {
	case "simulated":
		payoutProvider = provider.NewSimulatedProvider(cfg.ProviderFailureRate, cfg.ProviderProcessingTime)
	default:
		payoutProvider = provider.NewSimulatedProvider(10, 2*time.Second)
	}

	// Create service
	payoutService := service.NewPayoutService(repo, payoutProvider, logger, cfg.MaxRetries)

	// Setup Gin router for HTTP
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Health endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "settlement-service",
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": "settlement-service",
		})
	})

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	settlementServer := settlementgrpc.NewSettlementServer(payoutService, logger)
	settlementgrpc.RegisterSettlementServiceServer(grpcServer, settlementServer)

	// Register health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for grpcurl
	reflection.Register(grpcServer)

	// Create Kafka consumer
	kafkaConsumer := kafka.NewConsumer(
		cfg.KafkaBrokers,
		cfg.KafkaTopicFunded,
		cfg.KafkaConsumerGroup,
		payoutService,
		logger,
	)

	// Start HTTP server
	go func() {
		logger.Info("Starting HTTP server", zap.String("port", cfg.HTTPPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
		if err != nil {
			logger.Fatal("Failed to listen for gRPC", zap.Error(err))
		}
		logger.Info("Starting gRPC server", zap.String("port", cfg.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// Start Kafka consumer
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	go func() {
		if err := kafkaConsumer.Start(consumerCtx); err != nil {
			logger.Error("Kafka consumer error", zap.Error(err))
		}
	}()

	logger.Info("Settlement Service started",
		zap.String("httpPort", cfg.HTTPPort),
		zap.String("grpcPort", cfg.GRPCPort),
	)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down...")

	// Graceful shutdown
	cancelConsumer()
	kafkaConsumer.Close()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	grpcServer.GracefulStop()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown error", zap.Error(err))
	}

	logger.Info("Settlement Service stopped")
}
```

**Step 2: Update go.mod with dependencies**

Run: `cd d:/Codes/Movra/services/settlement-service && go get google.golang.org/grpc google.golang.org/grpc/health google.golang.org/grpc/health/grpc_health_v1 google.golang.org/grpc/reflection`

**Step 3: Verify build**

Run: `cd d:/Codes/Movra/services/settlement-service && go build ./cmd/server/...`
Expected: No errors

**Step 4: Commit**

```bash
git add services/settlement-service/cmd/server/main.go services/settlement-service/go.mod services/settlement-service/go.sum
git commit -m "feat(settlement): wire all dependencies in main.go"
```

---

## Task 9: Unit Tests for Provider

**Files:**
- Create: `services/settlement-service/internal/provider/simulated_test.go`

**Step 1: Create provider tests**

```go
package provider

import (
	"context"
	"testing"
	"time"

	"github.com/patteeraL/movra/services/settlement-service/internal/model"
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
```

**Step 2: Run tests**

Run: `cd d:/Codes/Movra/services/settlement-service && go test ./internal/provider/... -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add services/settlement-service/internal/provider/simulated_test.go
git commit -m "test(settlement): add unit tests for SimulatedProvider"
```

---

## Task 10: Unit Tests for Service

**Files:**
- Create: `services/settlement-service/internal/service/payout_service_test.go`

**Step 1: Create service tests with mock repository**

```go
package service

import (
	"context"
	"testing"
	"time"

	"github.com/patteeraL/movra/services/settlement-service/internal/model"
	"github.com/patteeraL/movra/services/settlement-service/internal/provider"
	"github.com/patteeraL/movra/services/settlement-service/internal/repository"
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
```

**Step 2: Run tests**

Run: `cd d:/Codes/Movra/services/settlement-service && go test ./internal/service/... -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add services/settlement-service/internal/service/payout_service_test.go
git commit -m "test(settlement): add unit tests for PayoutService"
```

---

## Success Criteria

- [ ] All packages compile without errors
- [ ] Provider tests pass: `go test ./internal/provider/... -v`
- [ ] Service tests pass: `go test ./internal/service/... -v`
- [ ] Service starts: `go run ./cmd/server/...`
- [ ] HTTP health endpoint works: `curl localhost:8083/health`
- [ ] gRPC reflection works: `grpcurl -plaintext localhost:9083 list`
- [ ] 10 atomic commits created
