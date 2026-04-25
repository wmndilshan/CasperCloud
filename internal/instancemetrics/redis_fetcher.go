package instancemetrics

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisFetcher reads JSON payloads written by the metrics poller.
type RedisFetcher struct {
	RDB *redis.Client
}

// FetchInstanceStatsPayload returns raw JSON written by the worker metrics poller.
func (f *RedisFetcher) FetchInstanceStatsPayload(ctx context.Context, instanceID uuid.UUID) ([]byte, error) {
	if f == nil || f.RDB == nil {
		return nil, ErrNoPayload
	}
	b, err := f.RDB.Get(ctx, StatsRedisKey(instanceID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrNoPayload
		}
		return nil, err
	}
	return b, nil
}
