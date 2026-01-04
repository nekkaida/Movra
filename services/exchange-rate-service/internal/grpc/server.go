package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/patteeraL/movra/services/exchange-rate-service/internal/model"
	"github.com/patteeraL/movra/services/exchange-rate-service/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ExchangeRateServer implements the gRPC ExchangeRateService
type ExchangeRateServer struct {
	UnimplementedExchangeRateServiceServer
	service *service.RateService
	logger  *zap.Logger
}

// NewExchangeRateServer creates a new gRPC server instance
func NewExchangeRateServer(svc *service.RateService, logger *zap.Logger) *ExchangeRateServer {
	return &ExchangeRateServer{
		service: svc,
		logger:  logger,
	}
}

// GetRate returns the current exchange rate for a currency pair
func (s *ExchangeRateServer) GetRate(ctx context.Context, req *GetRateRequest) (*GetRateResponse, error) {
	if req.SourceCurrency == "" || req.TargetCurrency == "" {
		return &GetRateResponse{
			Error: &Error{
				Code:    "INVALID_ARGUMENT",
				Message: "source_currency and target_currency are required",
			},
		}, nil
	}

	rate, err := s.service.GetRate(ctx, req.SourceCurrency, req.TargetCurrency)
	if err != nil {
		s.logger.Error("Failed to get rate",
			zap.String("source", req.SourceCurrency),
			zap.String("target", req.TargetCurrency),
			zap.Error(err),
		)
		return &GetRateResponse{
			Error: &Error{
				Code:    "RATE_NOT_AVAILABLE",
				Message: err.Error(),
			},
		}, nil
	}

	return &GetRateResponse{
		Rate: modelRateToProto(rate),
	}, nil
}

// LockRate locks a rate for a specified duration
func (s *ExchangeRateServer) LockRate(ctx context.Context, req *LockRateRequest) (*LockRateResponse, error) {
	if req.SourceCurrency == "" || req.TargetCurrency == "" {
		return &LockRateResponse{
			Error: &Error{
				Code:    "INVALID_ARGUMENT",
				Message: "source_currency and target_currency are required",
			},
		}, nil
	}

	durationSeconds := int(req.LockDurationSeconds)
	if durationSeconds <= 0 {
		durationSeconds = 30 // Default 30 seconds
	}

	locked, err := s.service.LockRate(ctx, req.SourceCurrency, req.TargetCurrency, durationSeconds)
	if err != nil {
		s.logger.Error("Failed to lock rate",
			zap.String("source", req.SourceCurrency),
			zap.String("target", req.TargetCurrency),
			zap.Error(err),
		)
		return &LockRateResponse{
			Error: &Error{
				Code:    "LOCK_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	return &LockRateResponse{
		LockedRate: modelLockedRateToProto(locked),
	}, nil
}

// GetLockedRate retrieves a previously locked rate
func (s *ExchangeRateServer) GetLockedRate(ctx context.Context, req *GetLockedRateRequest) (*GetLockedRateResponse, error) {
	if req.LockId == "" {
		return &GetLockedRateResponse{
			Error: &Error{
				Code:    "INVALID_ARGUMENT",
				Message: "lock_id is required",
			},
		}, nil
	}

	locked, err := s.service.GetLockedRate(ctx, req.LockId)
	if err != nil {
		s.logger.Error("Failed to get locked rate",
			zap.String("lockId", req.LockId),
			zap.Error(err),
		)
		return &GetLockedRateResponse{
			Error: &Error{
				Code:    "LOCK_NOT_FOUND",
				Message: err.Error(),
			},
		}, nil
	}

	return &GetLockedRateResponse{
		LockedRate: modelLockedRateToProto(locked),
	}, nil
}

// GetCorridors returns available currency corridors
func (s *ExchangeRateServer) GetCorridors(ctx context.Context, req *GetCorridorsRequest) (*GetCorridorsResponse, error) {
	corridors := s.service.GetCorridors(req.SourceCurrency)

	protoCorridors := make([]*Corridor, 0, len(corridors))
	for _, c := range corridors {
		protoCorridors = append(protoCorridors, modelCorridorToProto(&c))
	}

	return &GetCorridorsResponse{
		Corridors: protoCorridors,
	}, nil
}

// StreamRates streams real-time rate updates
func (s *ExchangeRateServer) StreamRates(req *StreamRatesRequest, stream ExchangeRateService_StreamRatesServer) error {
	if len(req.CurrencyPairs) == 0 {
		return status.Error(codes.InvalidArgument, "at least one currency pair is required")
	}

	// Parse currency pairs
	type pair struct {
		source, target string
	}
	pairs := make([]pair, 0, len(req.CurrencyPairs))
	for _, cp := range req.CurrencyPairs {
		var source, target string
		for i, c := range cp {
			if c == ':' {
				source = cp[:i]
				target = cp[i+1:]
				break
			}
		}
		if source == "" || target == "" {
			return status.Errorf(codes.InvalidArgument, "invalid currency pair format: %s (expected 'XXX:YYY')", cp)
		}
		pairs = append(pairs, pair{source, target})
	}

	// Stream rates every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ctx := stream.Context()

	// Send initial rates immediately
	for _, p := range pairs {
		rate, err := s.service.GetRate(ctx, p.source, p.target)
		if err != nil {
			s.logger.Warn("Failed to get rate for stream",
				zap.String("source", p.source),
				zap.String("target", p.target),
				zap.Error(err),
			)
			continue
		}

		if err := stream.Send(&RateUpdate{Rate: modelRateToProto(rate)}); err != nil {
			return err
		}
	}

	// Continue streaming
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for _, p := range pairs {
				rate, err := s.service.GetRate(ctx, p.source, p.target)
				if err != nil {
					continue
				}

				if err := stream.Send(&RateUpdate{Rate: modelRateToProto(rate)}); err != nil {
					return err
				}
			}
		}
	}
}

// Helper functions to convert between model and proto types

func modelRateToProto(rate *model.ExchangeRate) *ExchangeRate {
	return &ExchangeRate{
		SourceCurrency:   rate.SourceCurrency,
		TargetCurrency:   rate.TargetCurrency,
		Rate:             rate.Rate,
		BuyRate:          rate.BuyRate,
		MarginPercentage: rate.MarginPercentage,
		FetchedAt:        timeToProtoTimestamp(rate.FetchedAt),
		ExpiresAt:        timeToProtoTimestamp(rate.ExpiresAt),
	}
}

func modelLockedRateToProto(locked *model.LockedRate) *LockedRate {
	return &LockedRate{
		LockId:    locked.LockID,
		Rate:      modelRateToProto(&locked.Rate),
		LockedAt:  timeToProtoTimestamp(locked.LockedAt),
		ExpiresAt: timeToProtoTimestamp(locked.ExpiresAt),
		Expired:   locked.Expired,
	}
}

func modelCorridorToProto(c *model.Corridor) *Corridor {
	return &Corridor{
		SourceCurrency:   c.SourceCurrency,
		TargetCurrency:   c.TargetCurrency,
		Enabled:          c.Enabled,
		FeePercentage:    c.FeePercentage,
		FeeMinimum:       &Money{Currency: c.FeeMinimum.Currency, Amount: c.FeeMinimum.Amount},
		MarginPercentage: c.MarginPercentage,
		PayoutMethods:    c.PayoutMethods,
	}
}

func timeToProtoTimestamp(t time.Time) *Timestamp {
	return &Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

func protoTimestampToTime(ts *Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return time.Unix(ts.Seconds, int64(ts.Nanos))
}

// Placeholder types for generated proto code
// These will be replaced by actual generated code from protoc

// UnimplementedExchangeRateServiceServer provides forward compatibility
type UnimplementedExchangeRateServiceServer struct{}

func (UnimplementedExchangeRateServiceServer) GetRate(context.Context, *GetRateRequest) (*GetRateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRate not implemented")
}
func (UnimplementedExchangeRateServiceServer) LockRate(context.Context, *LockRateRequest) (*LockRateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LockRate not implemented")
}
func (UnimplementedExchangeRateServiceServer) GetLockedRate(context.Context, *GetLockedRateRequest) (*GetLockedRateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLockedRate not implemented")
}
func (UnimplementedExchangeRateServiceServer) GetCorridors(context.Context, *GetCorridorsRequest) (*GetCorridorsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCorridors not implemented")
}
func (UnimplementedExchangeRateServiceServer) StreamRates(*StreamRatesRequest, ExchangeRateService_StreamRatesServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamRates not implemented")
}
func (UnimplementedExchangeRateServiceServer) mustEmbedUnimplementedExchangeRateServiceServer() {}

// ExchangeRateService_StreamRatesServer is the server stream interface
type ExchangeRateService_StreamRatesServer interface {
	Send(*RateUpdate) error
	Context() context.Context
}

// Proto message types (placeholders - will be replaced by generated code)

type GetRateRequest struct {
	SourceCurrency string
	TargetCurrency string
}

type GetRateResponse struct {
	Rate  *ExchangeRate
	Error *Error
}

type LockRateRequest struct {
	SourceCurrency      string
	TargetCurrency      string
	LockDurationSeconds int32
}

type LockRateResponse struct {
	LockedRate *LockedRate
	Error      *Error
}

type GetLockedRateRequest struct {
	LockId string
}

type GetLockedRateResponse struct {
	LockedRate *LockedRate
	Error      *Error
}

type GetCorridorsRequest struct {
	SourceCurrency string
}

type GetCorridorsResponse struct {
	Corridors []*Corridor
	Error     *Error
}

type StreamRatesRequest struct {
	CurrencyPairs []string
}

type RateUpdate struct {
	Rate *ExchangeRate
}

type ExchangeRate struct {
	SourceCurrency   string
	TargetCurrency   string
	Rate             string
	BuyRate          string
	MarginPercentage string
	FetchedAt        *Timestamp
	ExpiresAt        *Timestamp
}

type LockedRate struct {
	LockId    string
	Rate      *ExchangeRate
	LockedAt  *Timestamp
	ExpiresAt *Timestamp
	Expired   bool
}

type Corridor struct {
	SourceCurrency   string
	TargetCurrency   string
	Enabled          bool
	FeePercentage    string
	FeeMinimum       *Money
	MarginPercentage string
	PayoutMethods    []string
}

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

// RegisterExchangeRateServiceServer registers the server with a gRPC server
// This is a placeholder - actual registration uses generated code
func RegisterExchangeRateServiceServer(s interface{}, srv *ExchangeRateServer) {
	// This will be replaced by actual generated registration code
	fmt.Printf("Registered ExchangeRateServiceServer: %v\n", srv)
}
