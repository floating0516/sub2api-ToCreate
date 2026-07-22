package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	defaultOpenAIQuotaHistoryLimit = 20
	maxOpenAIQuotaHistoryLimit     = 100

	OpenAIQuotaResetSourceManual   = "manual"
	OpenAIQuotaResetSourceProvider = "provider"
	OpenAIQuotaResetSourceUnknown  = "unknown"
	OpenAIQuotaResetSourceAuto     = "auto"
)

var ErrOpenAIQuotaCycleNotFound = infraerrors.New(
	http.StatusNotFound,
	"OPENAI_QUOTA_CYCLE_NOT_FOUND",
	"quota cycle not found for this account",
)

// OpenAIQuotaCycle is one observed seven-day Codex quota cycle.
type OpenAIQuotaCycle struct {
	ID                   int64      `json:"id"`
	CycleStartedAt       time.Time  `json:"cycle_started_at"`
	LastObservedAt       time.Time  `json:"last_observed_at"`
	LastUsedPercent      float64    `json:"last_used_percent"`
	PeakUsedPercent      float64    `json:"peak_used_percent"`
	ProviderResetAt      *time.Time `json:"provider_reset_at,omitempty"`
	ResetObservedAt      *time.Time `json:"reset_observed_at,omitempty"`
	ResetToPercent       *float64   `json:"reset_to_percent,omitempty"`
	DetectionReason      string     `json:"detection_reason,omitempty"`
	AutomaticResetSource string     `json:"automatic_reset_source,omitempty"`
	ResetSource          string     `json:"reset_source,omitempty"`
	ResetSourceOverride  *string    `json:"reset_source_override,omitempty"`
	ResetSourceEvidence  string     `json:"reset_source_evidence,omitempty"`
}

// OpenAIQuotaSample is one downsampled point in the seven-day usage timeline.
type OpenAIQuotaSample struct {
	CycleID      int64     `json:"cycle_id"`
	ObservedAt   time.Time `json:"observed_at"`
	UsedPercent  float64   `json:"used_percent"`
	LocalCostUSD float64   `json:"local_cost_usd"` // Cumulative account-billed cost in this provider window.
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

type OpenAIQuotaResetSourceRepository interface {
	SetOpenAIQuotaResetSource(ctx context.Context, accountID, cycleID int64, resetSourceOverride *string) error
}

// OpenAIQuotaObservation combines the weekly quota point with a complete,
// sanitized reset-credit snapshot when both are available from one query.
type OpenAIQuotaObservation struct {
	ObservedAt          time.Time
	UsedPercent         *float64
	ProviderResetAt     *time.Time
	CreditSnapshotKnown bool
	CreditExpiresAt     []time.Time
}

// OpenAIQuotaObservationRecorder persists evidence used to classify a reset.
type OpenAIQuotaObservationRecorder interface {
	RecordOpenAIQuotaObservation(ctx context.Context, accountID int64, observation *OpenAIQuotaObservation) error
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

// SetQuotaHistoryResetSource applies or clears an administrator override for
// one closed quota cycle. "auto" clears the override and restores detection.
func (s *OpenAIQuotaService) SetQuotaHistoryResetSource(ctx context.Context, accountID, cycleID int64, resetSource string) error {
	if s == nil || s.accountRepo == nil {
		return infraerrors.New(http.StatusInternalServerError, "OPENAI_QUOTA_NOT_CONFIGURED", "openai quota service is not configured")
	}
	if cycleID <= 0 {
		return infraerrors.New(http.StatusBadRequest, "OPENAI_QUOTA_INVALID_CYCLE", "invalid quota cycle")
	}

	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil {
		return infraerrors.Newf(http.StatusNotFound, "OPENAI_QUOTA_ACCOUNT_NOT_FOUND", "account not found: %v", err)
	}
	if account.Platform != PlatformOpenAI || account.Type != AccountTypeOAuth || account.IsShadow() {
		return infraerrors.New(http.StatusBadRequest, "OPENAI_QUOTA_HISTORY_UNSUPPORTED", "quota history source overrides require a primary OpenAI OAuth account")
	}

	var override *string
	switch normalized := strings.ToLower(strings.TrimSpace(resetSource)); normalized {
	case OpenAIQuotaResetSourceAuto:
		override = nil
	case OpenAIQuotaResetSourceManual, OpenAIQuotaResetSourceProvider:
		override = &normalized
	default:
		return infraerrors.New(http.StatusBadRequest, "OPENAI_QUOTA_INVALID_RESET_SOURCE", "reset source must be auto, manual, or provider")
	}

	repo, ok := s.accountRepo.(OpenAIQuotaResetSourceRepository)
	if !ok {
		return infraerrors.New(http.StatusInternalServerError, "OPENAI_QUOTA_HISTORY_NOT_CONFIGURED", "openai quota history is not configured")
	}
	if err = repo.SetOpenAIQuotaResetSource(ctx, accountID, cycleID, override); err != nil {
		return err
	}
	return nil
}
