package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/movra/settlement-service/internal/model"
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
			// Skip index keys (transfer: prefix)
			if strings.HasPrefix(key, transferKeyPrefix) {
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
