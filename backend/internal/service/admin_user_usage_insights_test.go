//go:build unit

package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type insightsRepoStub struct {
	userRepoStubForListUsers
	calls atomic.Int32
	read  func(context.Context, []int64) (map[int64]*UserUsageSummary, map[int64][]UserModelPreference, error)
}

func (r *insightsRepoStub) GetUserUsageInsightsByUserIDs(ctx context.Context, ids []int64) (map[int64]*UserUsageSummary, map[int64][]UserModelPreference, error) {
	r.calls.Add(1)
	if r.read != nil {
		return r.read(ctx, ids)
	}
	summaries := make(map[int64]*UserUsageSummary, len(ids))
	for _, id := range ids {
		summaries[id] = &UserUsageSummary{TotalRequests: 42}
	}
	return summaries, map[int64][]UserModelPreference{1: {{Model: "test-model"}}}, nil
}

func TestUserUsageInsights_LightweightListAndLegacyDefault(t *testing.T) {
	repo := &insightsRepoStub{userRepoStubForListUsers: userRepoStubForListUsers{users: []User{{ID: 1}}}}
	svc := &adminServiceImpl{userRepo: repo}
	include := false
	users, _, err := svc.ListUsers(context.Background(), 1, 20, UserListFilters{IncludeUsageInsights: &include}, "created_at", "desc")
	require.NoError(t, err)
	require.Nil(t, users[0].UsageSummary)
	require.Zero(t, repo.calls.Load())
	users, _, err = svc.ListUsers(context.Background(), 1, 20, UserListFilters{}, "created_at", "desc")
	require.NoError(t, err)
	require.Equal(t, int64(42), users[0].UsageSummary.TotalRequests)
	require.Equal(t, int32(1), repo.calls.Load())
}

func TestUserUsageInsights_CacheAcrossPagesExpiresAndOwnsData(t *testing.T) {
	repo := &insightsRepoStub{}
	svc := &adminServiceImpl{userRepo: repo}
	first, err := svc.GetUserUsageInsights(context.Background(), []int64{1, 2, 1})
	require.NoError(t, err)
	require.Len(t, first, 2)
	first[1].Summary.TotalRequests = 999
	first[1].ModelPreferences[0].Model = "mutated"
	next, err := svc.GetUserUsageInsights(context.Background(), []int64{2, 1})
	require.NoError(t, err)
	require.Equal(t, int64(42), next[1].Summary.TotalRequests)
	require.Equal(t, "test-model", next[1].ModelPreferences[0].Model)
	require.Equal(t, int32(1), repo.calls.Load())

	repo.read = func(_ context.Context, ids []int64) (map[int64]*UserUsageSummary, map[int64][]UserModelPreference, error) {
		require.Equal(t, []int64{3}, ids)
		return map[int64]*UserUsageSummary{3: {TotalRequests: 17}}, nil, nil
	}
	_, err = svc.GetUserUsageInsights(context.Background(), []int64{1, 3})
	require.NoError(t, err)
	require.Equal(t, int32(2), repo.calls.Load())
	svc.userUsageInsightsCache.mu.Lock()
	entry := svc.userUsageInsightsCache.entries[3]
	entry.expiresAt = time.Now().Add(-time.Second)
	svc.userUsageInsightsCache.entries[3] = entry
	svc.userUsageInsightsCache.mu.Unlock()
	_, err = svc.GetUserUsageInsights(context.Background(), []int64{3})
	require.NoError(t, err)
	require.Equal(t, int32(3), repo.calls.Load())
}

func TestUserUsageInsights_CoalescesAndWaiterCanCancel(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	repo := &insightsRepoStub{read: func(ctx context.Context, ids []int64) (map[int64]*UserUsageSummary, map[int64][]UserModelPreference, error) {
		close(started)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-release:
			return map[int64]*UserUsageSummary{1: {TotalRequests: 42}}, nil, nil
		}
	}}
	svc := &adminServiceImpl{userRepo: repo}
	first := make(chan error, 1)
	go func() { _, err := svc.GetUserUsageInsights(context.Background(), []int64{1}); first <- err }()
	<-started
	cancelCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := svc.GetUserUsageInsights(cancelCtx, []int64{1})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	second := make(chan error, 1)
	go func() { _, err := svc.GetUserUsageInsights(context.Background(), []int64{1}); second <- err }()
	close(release)
	require.NoError(t, <-first)
	require.NoError(t, <-second)
	require.Equal(t, int32(1), repo.calls.Load())
}

func TestUserUsageInsights_FailuresNotCachedAndBatchBounded(t *testing.T) {
	repo := &insightsRepoStub{read: func(context.Context, []int64) (map[int64]*UserUsageSummary, map[int64][]UserModelPreference, error) {
		return nil, nil, errors.New("database unavailable")
	}}
	svc := &adminServiceImpl{userRepo: repo}
	_, err := svc.GetUserUsageInsights(context.Background(), []int64{1})
	require.Error(t, err)
	repo.read = nil
	out, err := svc.GetUserUsageInsights(context.Background(), []int64{1})
	require.NoError(t, err)
	require.Equal(t, int64(42), out[1].Summary.TotalRequests)
	require.Equal(t, int32(2), repo.calls.Load())
	_, err = svc.GetUserUsageInsights(context.Background(), make([]int64, MaxUserUsageInsightsBatch+1))
	require.Error(t, err)
	_, err = svc.GetUserUsageInsights(context.Background(), []int64{-1})
	require.Error(t, err)
	require.Equal(t, int32(2), repo.calls.Load())
}
