package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const (
	userUsageInsightsTTL        = time.Minute
	userUsageInsightsCacheLimit = 2048
	MaxUserUsageInsightsBatch   = 100
)

// AdminUserUsageInsightsReader is the read-only, admin-only batch capability.
type AdminUserUsageInsightsReader interface {
	GetUserUsageInsights(context.Context, []int64) (map[int64]UserUsageInsights, error)
}

type UserUsageInsights struct {
	Summary          *UserUsageSummary
	ModelPreferences []UserModelPreference
}

type userUsageInsightsEntry struct {
	value     UserUsageInsights
	expiresAt time.Time
}

// Entries are per user so changing page size or sort order can reuse them.
// Only one cache fill runs at a time, coalescing overlapping page requests and
// bounding expensive aggregate work. Waiting requests remain cancellable.
type userUsageInsightsCache struct {
	mu       sync.Mutex
	entries  map[int64]userUsageInsightsEntry
	inflight chan struct{}
}

func cloneUserUsageInsights(in UserUsageInsights) UserUsageInsights {
	out := UserUsageInsights{ModelPreferences: append([]UserModelPreference(nil), in.ModelPreferences...)}
	if in.Summary != nil {
		summary := *in.Summary
		if summary.LastUsageAt != nil {
			ts := *summary.LastUsageAt
			summary.LastUsageAt = &ts
		}
		out.Summary = &summary
	}
	return out
}

func (s *adminServiceImpl) GetUserUsageInsights(ctx context.Context, userIDs []int64) (map[int64]UserUsageInsights, error) {
	if len(userIDs) > MaxUserUsageInsightsBatch {
		return nil, errors.New("too many users in usage insights batch")
	}
	ids := make([]int64, 0, len(userIDs))
	seen := make(map[int64]bool, len(userIDs))
	for _, id := range userIDs {
		if id <= 0 {
			return nil, errors.New("user IDs must be positive")
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	out := make(map[int64]UserUsageInsights, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	reader, ok := s.userRepo.(userUsageInsightsBatchReader)
	if !ok {
		return nil, errors.New("user usage insights are unavailable")
	}
	cache := &s.userUsageInsightsCache
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		now := timezone.Now()
		cache.mu.Lock()
		missing := make([]int64, 0, len(ids))
		for _, id := range ids {
			entry, exists := cache.entries[id]
			if exists && now.Before(entry.expiresAt) {
				out[id] = cloneUserUsageInsights(entry.value)
			} else {
				missing = append(missing, id)
			}
		}
		if len(missing) == 0 {
			cache.mu.Unlock()
			return out, nil
		}
		if pending := cache.inflight; pending != nil {
			cache.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-pending:
				continue
			}
		}
		cache.inflight = make(chan struct{})
		cache.mu.Unlock()

		queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		summaries, preferences, err := reader.GetUserUsageInsightsByUserIDs(queryCtx, missing)
		cancel()

		cache.mu.Lock()
		if err == nil {
			if cache.entries == nil {
				cache.entries = make(map[int64]userUsageInsightsEntry)
			}
			for id, entry := range cache.entries {
				if !now.Before(entry.expiresAt) {
					delete(cache.entries, id)
				}
			}
			expiresAt := now.Add(userUsageInsightsTTL)
			// Today's totals must not survive the configured calendar-day boundary.
			if midnight := timezone.StartOfDay(now).AddDate(0, 0, 1); midnight.Before(expiresAt) {
				expiresAt = midnight
			}
			for _, id := range missing {
				if len(cache.entries) >= userUsageInsightsCacheLimit {
					// Bounded cache; evict the entry closest to expiry.
					var oldestID int64
					var oldestExpiry time.Time
					for cachedID, entry := range cache.entries {
						if oldestExpiry.IsZero() || entry.expiresAt.Before(oldestExpiry) {
							oldestID, oldestExpiry = cachedID, entry.expiresAt
						}
					}
					delete(cache.entries, oldestID)
				}
				summary := summaries[id]
				if summary == nil {
					summary = &UserUsageSummary{}
				}
				value := UserUsageInsights{Summary: summary, ModelPreferences: preferences[id]}
				cache.entries[id] = userUsageInsightsEntry{value: cloneUserUsageInsights(value), expiresAt: expiresAt}
				out[id] = cloneUserUsageInsights(value)
			}
		}
		close(cache.inflight)
		cache.inflight = nil
		cache.mu.Unlock()
		if err != nil {
			return nil, err // Failed/cancelled reads never become cached zero totals.
		}
		return out, nil
	}
}
