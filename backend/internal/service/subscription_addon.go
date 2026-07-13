package service

import (
	"context"
	"errors"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SubscriptionAddonStatusActive    = "active"
	SubscriptionAddonStatusExhausted = "exhausted"
	SubscriptionAddonStatusRevoked   = "revoked"
)

var (
	ErrSubscriptionAddonNotFound = infraerrors.NotFound("SUBSCRIPTION_ADDON_NOT_FOUND", "subscription add-on pack not found")
	ErrSubscriptionAddonInvalid  = infraerrors.BadRequest("SUBSCRIPTION_ADDON_INVALID", "subscription add-on pack is invalid")
)

type SubscriptionAddonPack struct {
	ID             int64
	SubscriptionID int64
	UserID         int64
	GroupID        int64
	QuotaUSD       float64
	UsedUSD        float64
	StartsAt       time.Time
	ExpiresAt      time.Time
	Status         string
	AssignedBy     *int64
	Notes          string
	RevokedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (p *SubscriptionAddonPack) RemainingUSD() float64 {
	if p == nil || p.QuotaUSD <= p.UsedUSD {
		return 0
	}
	return p.QuotaUSD - p.UsedUSD
}

func (p *SubscriptionAddonPack) IsUsableAt(now time.Time) bool {
	return p != nil &&
		p.Status == SubscriptionAddonStatusActive &&
		!now.Before(p.StartsAt) &&
		now.Before(p.ExpiresAt) &&
		p.RemainingUSD() > 0
}

type SubscriptionAddonSummary struct {
	TotalQuotaUSD    float64
	UsedUSD          float64
	RemainingUSD     float64
	ActivePackCount  int
	NearestExpiresAt *time.Time
}

type SubscriptionAddonQuotaTotal struct {
	EffectiveQuotaUSD float64
	UsedUSD           float64
}

type GrantSubscriptionAddonInput struct {
	SubscriptionID int64
	QuotaUSD       float64
	ExpiresAt      *time.Time
	AssignedBy     int64
	Notes          string
}

type SubscriptionAddonRepository interface {
	Create(ctx context.Context, pack *SubscriptionAddonPack) error
	GetByID(ctx context.Context, id int64) (*SubscriptionAddonPack, error)
	GetUsableForSubscription(ctx context.Context, subscriptionID int64, now time.Time) (*SubscriptionAddonPack, error)
	ListBySubscriptionID(ctx context.Context, subscriptionID int64) ([]SubscriptionAddonPack, error)
	GetActiveSummaries(ctx context.Context, subscriptionIDs []int64, now time.Time) (map[int64]SubscriptionAddonSummary, error)
	GetCurrentTermQuotaTotals(ctx context.Context, subscriptionIDs []int64, now time.Time) (map[int64]SubscriptionAddonQuotaTotal, error)
	GetGrantedQuotaForTerm(ctx context.Context, subscriptionID int64, startsAt, expiresAt time.Time) (float64, error)
	Revoke(ctx context.Context, id int64, revokedAt time.Time) error
}

func (s *SubscriptionService) GrantAddon(ctx context.Context, input *GrantSubscriptionAddonInput) (*SubscriptionAddonPack, error) {
	if s == nil || s.addonRepo == nil || input == nil || input.SubscriptionID <= 0 || input.QuotaUSD <= 0 {
		return nil, ErrSubscriptionAddonInvalid
	}

	sub, err := s.userSubRepo.GetByID(ctx, input.SubscriptionID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if sub.Status != SubscriptionStatusActive || !sub.ExpiresAt.After(now) {
		return nil, ErrSubscriptionInvalid
	}

	expiresAt := sub.ExpiresAt
	if input.ExpiresAt != nil {
		expiresAt = *input.ExpiresAt
	}
	if !expiresAt.After(now) || expiresAt.After(sub.ExpiresAt) {
		return nil, ErrSubscriptionAddonInvalid
	}

	pack := &SubscriptionAddonPack{
		SubscriptionID: sub.ID,
		UserID:         sub.UserID,
		GroupID:        sub.GroupID,
		QuotaUSD:       input.QuotaUSD,
		StartsAt:       now,
		ExpiresAt:      expiresAt,
		Status:         SubscriptionAddonStatusActive,
		Notes:          input.Notes,
	}
	if input.AssignedBy > 0 {
		pack.AssignedBy = &input.AssignedBy
	}
	if err := s.addonRepo.Create(ctx, pack); err != nil {
		return nil, err
	}
	return pack, nil
}

func (s *SubscriptionService) ListAddons(ctx context.Context, subscriptionID int64) ([]SubscriptionAddonPack, error) {
	if s == nil || s.addonRepo == nil {
		return nil, ErrSubscriptionAddonInvalid
	}
	if _, err := s.userSubRepo.GetByID(ctx, subscriptionID); err != nil {
		return nil, err
	}
	return s.addonRepo.ListBySubscriptionID(ctx, subscriptionID)
}

func (s *SubscriptionService) RevokeAddon(ctx context.Context, subscriptionID, addonID int64) error {
	if s == nil || s.addonRepo == nil {
		return ErrSubscriptionAddonInvalid
	}
	pack, err := s.addonRepo.GetByID(ctx, addonID)
	if err != nil {
		return err
	}
	if pack.SubscriptionID != subscriptionID || pack.Status != SubscriptionAddonStatusActive {
		return ErrSubscriptionAddonInvalid
	}
	return s.addonRepo.Revoke(ctx, addonID, time.Now())
}

func (s *SubscriptionService) ResolveUsageAccess(ctx context.Context, sub *UserSubscription, group *Group) (*UserSubscription, error) {
	needsMaintenance, err := s.ValidateAndCheckLimits(sub, group)
	if needsMaintenance {
		refreshed, maintenanceErr := s.EnsureWindowMaintenance(ctx, sub)
		if maintenanceErr != nil {
			return nil, ErrBillingServiceUnavailable.WithCause(maintenanceErr)
		}
		sub = refreshed
		_, err = s.ValidateAndCheckLimits(sub, group)
	}
	if err == nil || !isSubscriptionQuotaLimitError(err) {
		return sub, err
	}
	if s.addonRepo == nil {
		return sub, err
	}

	addon, addonErr := s.addonRepo.GetUsableForSubscription(ctx, sub.ID, time.Now())
	if addonErr == nil {
		sub.ActiveAddon = addon
		return sub, nil
	}
	if errors.Is(addonErr, ErrSubscriptionAddonNotFound) {
		return sub, err
	}
	return sub, ErrBillingServiceUnavailable.WithCause(addonErr)
}

func isSubscriptionQuotaLimitError(err error) bool {
	return errors.Is(err, ErrDailyLimitExceeded) ||
		errors.Is(err, ErrWeeklyLimitExceeded) ||
		errors.Is(err, ErrMonthlyLimitExceeded)
}

func (s *SubscriptionService) attachAddonSummaries(ctx context.Context, subs []UserSubscription) error {
	if s == nil || s.addonRepo == nil || len(subs) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(subs))
	for i := range subs {
		ids = append(ids, subs[i].ID)
	}
	summaries, err := s.addonRepo.GetActiveSummaries(ctx, ids, time.Now())
	if err != nil {
		return err
	}
	for i := range subs {
		if summary, ok := summaries[subs[i].ID]; ok {
			copy := summary
			subs[i].AddonSummary = &copy
		}
	}
	return nil
}
