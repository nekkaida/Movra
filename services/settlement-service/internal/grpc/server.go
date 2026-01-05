package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/movra/settlement-service/internal/model"
	"github.com/movra/settlement-service/internal/repository"
	"github.com/movra/settlement-service/internal/service"
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
	PayoutStatus_PAYOUT_STATUS_UNSPECIFIED      PayoutStatus = 0
	PayoutStatus_PAYOUT_STATUS_PENDING          PayoutStatus = 1
	PayoutStatus_PAYOUT_STATUS_PROCESSING       PayoutStatus = 2
	PayoutStatus_PAYOUT_STATUS_COMPLETED        PayoutStatus = 3
	PayoutStatus_PAYOUT_STATUS_FAILED           PayoutStatus = 4
	PayoutStatus_PAYOUT_STATUS_CANCELLED        PayoutStatus = 5
	PayoutStatus_PAYOUT_STATUS_READY_FOR_PICKUP PayoutStatus = 6
	PayoutStatus_PAYOUT_STATUS_PICKED_UP        PayoutStatus = 7
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
