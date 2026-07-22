package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type openAIQuotaHistorySourceRepoStub struct {
	AccountRepository
	account      *Account
	accountID    int64
	cycleID      int64
	source       *string
	updateCalled int
}

func (r *openAIQuotaHistorySourceRepoStub) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

func (r *openAIQuotaHistorySourceRepoStub) GetOpenAIQuotaHistory(context.Context, int64, int) (*OpenAIQuotaHistoryResponse, error) {
	return &OpenAIQuotaHistoryResponse{}, nil
}

func (r *openAIQuotaHistorySourceRepoStub) SetOpenAIQuotaResetSource(
	_ context.Context,
	accountID, cycleID int64,
	source *string,
) error {
	r.accountID = accountID
	r.cycleID = cycleID
	r.source = source
	r.updateCalled++
	return nil
}

func TestSetQuotaHistoryResetSourceAppliesAndClearsOverride(t *testing.T) {
	repo := &openAIQuotaHistorySourceRepoStub{account: &Account{
		ID:       44,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}}
	quotaService := &OpenAIQuotaService{accountRepo: repo}

	require.NoError(t, quotaService.SetQuotaHistoryResetSource(context.Background(), 44, 5, " MANUAL "))
	require.Equal(t, int64(44), repo.accountID)
	require.Equal(t, int64(5), repo.cycleID)
	require.NotNil(t, repo.source)
	require.Equal(t, OpenAIQuotaResetSourceManual, *repo.source)

	require.NoError(t, quotaService.SetQuotaHistoryResetSource(context.Background(), 44, 5, "auto"))
	require.Nil(t, repo.source)
	require.Equal(t, 2, repo.updateCalled)

	require.Error(t, quotaService.SetQuotaHistoryResetSource(context.Background(), 44, 5, "unknown"))
	require.Equal(t, 2, repo.updateCalled)
}
