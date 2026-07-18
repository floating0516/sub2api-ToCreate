package service

import (
	"context"
	"net/http"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	defaultOpenAIQuotaHistoryLimit = 20
	maxOpenAIQuotaHistoryLimit     = 100
)

// OpenAIQuotaCycle is one observed seven-day Codex quota cycle.
type OpenAIQuotaCycle struct {
	ID              int64      `json:"id"`
	CycleStartedAt  time.Time  `json:"cycle_started_at"`
	LastObservedAt  time.Time  `json:"last_observed_at"`
	LastUsedPercent float64    `json:"last_used_percent"`
	PeakUsedPercent float64    `json:"peak_used_percent"`
	ProviderResetAt *time.Time `json:"provider_reset_at,omitempty"`
	ResetObservedAt *time.Time `json:"reset_observed_at,omitempty"`
	ResetToPercent  *float64   `json:"reset_to_percent,omitempty"`
	DetectionReason string     `json:"detection_reason,omitempty"`
}

// OpenAIQuotaSample is one downsampled point in the seven-day usage timeline.
type OpenAIQuotaSample struct {
	CycleID     int64     `json:"cycle_id"`
	ObservedAt  time.Time `json:"observed_at"`
	UsedPercent float64   `json:"used_percent"`
}

type OpenAIQuotaHistoryResponse struct {
	Current *OpenAIQuotaCycle   `json:"current,omitempty"`
	History []OpenAIQuotaCycle  `json:"history"`
	Samples []OpenAIQuotaSample `json:"samples"`
	HasMore bool                `json:"has_more"`
}

// OpenAIQuotaHistoryRepository is deliberately narrower than AccountRepository
// so quota history does not expand every gateway test double.
type OpenAIQuotaHistoryRepository interface {
	GetOpenAIQuotaHistory(ctx context.Context, accountID int64, limit int) (*OpenAIQuotaHistoryResponse, error)
}

// OpenAIQuotaManualResetRecorder records a reset that was explicitly confirmed
// by the upstream reset-credit endpoint.
type OpenAIQuotaManualResetRecorder interface {
	RecordOpenAIQuotaManualReset(ctx context.Context, accountID int64, observedAt time.Time) error
}

func normalizeOpenAIQuotaHistoryLimit(limit int) int {
	if limit <= 0 {
		return defaultOpenAIQuotaHistoryLimit
	}
	if limit > maxOpenAIQuotaHistoryLimit {
		return maxOpenAIQuotaHistoryLimit
	}
	return limit
}

// GetQuotaHistory returns local observations only and never calls OpenAI.
func (s *OpenAIQuotaService) GetQuotaHistory(ctx context.Context, accountID int64, limit int) (*OpenAIQuotaHistoryResponse, error) {
	if s == nil || s.accountRepo == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "OPENAI_QUOTA_NOT_CONFIGURED", "openai quota service is not configured")
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil {
		return nil, infraerrors.Newf(http.StatusNotFound, "OPENAI_QUOTA_ACCOUNT_NOT_FOUND", "account not found: %v", err)
	}
	if account.Platform != PlatformOpenAI {
		return nil, infraerrors.New(http.StatusBadRequest, "OPENAI_QUOTA_INVALID_PLATFORM", "account is not an OpenAI account")
	}
	if account.Type != AccountTypeOAuth {
		return nil, infraerrors.New(http.StatusBadRequest, "OPENAI_QUOTA_INVALID_TYPE", "account is not an OAuth account")
	}
	if account.IsShadow() {
		return nil, ErrSparkShadowResetNotSupported
	}

	repo, ok := s.accountRepo.(OpenAIQuotaHistoryRepository)
	if !ok {
		return nil, infraerrors.New(http.StatusInternalServerError, "OPENAI_QUOTA_HISTORY_NOT_CONFIGURED", "openai quota history is not configured")
	}
	return repo.GetOpenAIQuotaHistory(ctx, accountID, normalizeOpenAIQuotaHistoryLimit(limit))
}
