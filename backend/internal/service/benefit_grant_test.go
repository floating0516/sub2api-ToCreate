package service

import (
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestNormalizeBenefitGrantInputBuildsTodayWindowAndStableMarker(t *testing.T) {
	now := time.Date(2026, time.July, 30, 4, 30, 0, 0, time.UTC)

	got, err := normalizeBenefitGrantInput(&BenefitGrantInput{
		AudienceType: BenefitGrantAudienceTodayActive,
		AudienceDate: "2026-07-30",
		Timezone:     "Asia/Shanghai",
		BenefitType:  BenefitGrantTypeSubscription,
		GroupID:      17,
		ValidityDays: 1,
		Notes:        "summer campaign",
	}, now)

	require.NoError(t, err)
	require.Equal(t, "2026-07-30", got.AudienceDate)
	require.Equal(t, "Asia/Shanghai", got.windowStart.Location().String())
	require.Equal(t, 0, got.windowStart.Hour())
	require.Equal(t, got.windowStart.AddDate(0, 0, 1), got.windowEnd)
	require.Contains(t, got.marker, "today_active:2026-07-30:Asia/Shanghai:subscription:17:1")
	require.Equal(t, got.marker+"\nsummer campaign", got.grantNotes)
}

func TestNormalizeBenefitGrantInputRejectsNonCurrentDate(t *testing.T) {
	now := time.Date(2026, time.July, 30, 4, 30, 0, 0, time.UTC)

	_, err := normalizeBenefitGrantInput(&BenefitGrantInput{
		AudienceType: BenefitGrantAudienceTodayActive,
		AudienceDate: "2026-07-29",
		Timezone:     "Asia/Shanghai",
		BenefitType:  BenefitGrantTypeSubscription,
		GroupID:      17,
		ValidityDays: 1,
	}, now)

	require.Error(t, err)
	require.Equal(t, "BENEFIT_GRANT_INVALID_INPUT", infraerrors.Reason(err))
}

func TestClassifyBenefitSubscription(t *testing.T) {
	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	marker := "[benefit-grant:today]"

	require.Equal(t, benefitSubscriptionAlreadyGranted, classifyBenefitSubscription(
		benefitSubscriptionState{
			status:    SubscriptionStatusExpired,
			expiresAt: now.Add(-time.Hour),
			notes:     "older note\n" + marker,
		},
		marker,
		now,
	))
	require.Equal(t, benefitSubscriptionEligible, classifyBenefitSubscription(
		benefitSubscriptionState{
			status:    SubscriptionStatusExpired,
			expiresAt: now.Add(-time.Hour),
		},
		marker,
		now,
	))
	require.Equal(t, benefitSubscriptionEligible, classifyBenefitSubscription(
		benefitSubscriptionState{
			status:    SubscriptionStatusActive,
			expiresAt: now.Add(-time.Minute),
		},
		marker,
		now,
	))
	require.Equal(t, benefitSubscriptionConflict, classifyBenefitSubscription(
		benefitSubscriptionState{
			status:    SubscriptionStatusActive,
			expiresAt: now.Add(time.Hour),
		},
		marker,
		now,
	))
	require.Equal(t, benefitSubscriptionConflict, classifyBenefitSubscription(
		benefitSubscriptionState{
			status:    SubscriptionStatusSuspended,
			expiresAt: now.Add(-time.Hour),
		},
		marker,
		now,
	))
}
