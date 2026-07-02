package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type accountCapacitySampleRepository struct {
	db *sql.DB
}

func NewAccountCapacitySampleRepository(db *sql.DB) service.AccountCapacitySampleRepository {
	return &accountCapacitySampleRepository{db: db}
}

func (r *accountCapacitySampleRepository) InsertSamples(ctx context.Context, samples []usagestats.AccountCapacitySample) error {
	if len(samples) == 0 {
		return nil
	}
	if r == nil || r.db == nil {
		return fmt.Errorf("nil account capacity sample repository")
	}

	accountIDs := make([]int64, 0, len(samples))
	sampledAts := make([]string, 0, len(samples))
	currentConcurrency := make([]int64, 0, len(samples))
	maxConcurrency := make([]int64, 0, len(samples))
	waitingCounts := make([]int64, 0, len(samples))

	for _, sample := range samples {
		if sample.AccountID <= 0 {
			continue
		}
		accountIDs = append(accountIDs, sample.AccountID)
		sampledAts = append(sampledAts, sample.SampledAt.UTC().Format(time.RFC3339Nano))
		currentConcurrency = append(currentConcurrency, nonNegativeInt64(sample.CurrentConcurrency))
		maxConcurrency = append(maxConcurrency, nonNegativeInt64(sample.MaxConcurrency))
		waitingCounts = append(waitingCounts, nonNegativeInt64(sample.WaitingCount))
	}
	if len(accountIDs) == 0 {
		return nil
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO account_capacity_samples (
			account_id,
			sampled_at,
			current_concurrency,
			max_concurrency,
			waiting_count
		)
		SELECT *
		FROM unnest(
			$1::bigint[],
			$2::timestamptz[],
			$3::bigint[],
			$4::bigint[],
			$5::bigint[]
		)
	`, pq.Array(accountIDs), pq.Array(sampledAts), pq.Array(currentConcurrency), pq.Array(maxConcurrency), pq.Array(waitingCounts))
	return err
}

func nonNegativeInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
