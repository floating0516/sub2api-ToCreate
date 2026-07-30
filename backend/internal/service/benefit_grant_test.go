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
		OperationKey:   "benefit-grant-operation-17",
		AudienceType:   BenefitGrantAudienceTodayActive,
		AudienceDate:   "2026-07-30",
		AudienceDays:   30,
		Timezone:       "Asia/Shanghai",
		BenefitType:    BenefitGrantTypeSubscription,
		ConflictPolicy: BenefitGrantConflictSkipActive,
		GroupID:        17,
		ValidityDays:   1,
		Notes:          "summer campaign",
	}, now)

	require.NoError(t, err)
	require.Equal(t, 1, got.AudienceDays)
	require.Equal(t, "2026-07-30", got.AudienceDate)
	require.Equal(t, "Asia/Shanghai", got.windowStart.Location().String())
	require.Equal(t, 0, got.windowStart.Hour())
	require.Equal(t, got.windowStart.AddDate(0, 0, 1), got.windowEnd)
	require.Equal(t, "[benefit-grant:benefit-grant-operation-17]", got.marker)
	require.Equal(t, got.marker+"\nsummer campaign", got.grantNotes)
	require.Len(t, got.requestHash, 64)
}

func TestNormalizeBenefitGrantInputBuildsRecentCalendarDayWindow(t *testing.T) {
	now := time.Date(2026, time.July, 30, 4, 30, 0, 0, time.UTC)

	got, err := normalizeBenefitGrantInput(&BenefitGrantInput{
		OperationKey:  "benefit-grant-operation-18",
		AudienceType:  BenefitGrantAudienceRecentActive,
		AudienceDate:  "2026-07-30",
		AudienceDays:  7,
		Timezone:      "Asia/Shanghai",
		BenefitType:   BenefitGrantTypeBalance,
		BalanceAmount: 2.5,
	}, now)

	require.NoError(t, err)
	require.Equal(t, time.Date(2026, time.July, 24, 0, 0, 0, 0, got.windowStart.Location()), got.windowStart)
	require.Equal(t, time.Date(2026, time.July, 31, 0, 0, 0, 0, got.windowEnd.Location()), got.windowEnd)
	require.Equal(t, BenefitGrantConflictNone, got.ConflictPolicy)
	require.Zero(t, got.GroupID)
	require.Zero(t, got.ValidityDays)
}

func TestNormalizeBenefitGrantInputRejectsNonCurrentDateForNewGrant(t *testing.T) {
	now := time.Date(2026, time.July, 30, 4, 30, 0, 0, time.UTC)

	_, err := normalizeBenefitGrantInput(&BenefitGrantInput{
		OperationKey: "benefit-grant-operation-19",
		AudienceType: BenefitGrantAudienceTodayActive,
		AudienceDate: "2026-07-29",
		AudienceDays: 1,
		Timezone:     "Asia/Shanghai",
		BenefitType:  BenefitGrantTypeSubscription,
		GroupID:      17,
		ValidityDays: 1,
	}, now)

	require.Error(t, err)
	require.Equal(t, "BENEFIT_GRANT_INVALID_INPUT", infraerrors.Reason(err))
}

func TestNormalizeBenefitGrantInputAllowsHistoricDateForIdempotentReplay(t *testing.T) {
	now := time.Date(2026, time.July, 31, 4, 30, 0, 0, time.UTC)

	got, err := normalizeBenefitGrantInputForReplay(&BenefitGrantInput{
		OperationKey:  "benefit-grant-operation-20",
		AudienceType:  BenefitGrantAudienceRecentRegistered,
		AudienceDate:  "2026-07-30",
		AudienceDays:  3,
		Timezone:      "Asia/Shanghai",
		BenefitType:   BenefitGrantTypeBalance,
		BalanceAmount: 1,
	}, now)

	require.NoError(t, err)
	require.Equal(t, "2026-07-30", got.AudienceDate)
	require.Equal(t, BenefitGrantAudienceRecentRegistered, got.AudienceType)
}

func TestNormalizeBenefitGrantInputRoundsBalanceToStoredPrecision(t *testing.T) {
	now := time.Date(2026, time.July, 30, 4, 30, 0, 0, time.UTC)

	got, err := normalizeBenefitGrantInput(&BenefitGrantInput{
		OperationKey:  "benefit-grant-operation-precision",
		AudienceType:  BenefitGrantAudienceTodayActive,
		AudienceDate:  "2026-07-30",
		AudienceDays:  1,
		Timezone:      "Asia/Shanghai",
		BenefitType:   BenefitGrantTypeBalance,
		BalanceAmount: 1.234567899,
	}, now)

	require.NoError(t, err)
	require.Equal(t, 1.2345679, got.BalanceAmount)
}

func TestNormalizeBenefitGrantInputRejectsBalanceBelowStoredPrecision(t *testing.T) {
	now := time.Date(2026, time.July, 30, 4, 30, 0, 0, time.UTC)

	_, err := normalizeBenefitGrantInput(&BenefitGrantInput{
		OperationKey:  "benefit-grant-operation-too-small",
		AudienceType:  BenefitGrantAudienceTodayActive,
		AudienceDate:  "2026-07-30",
		AudienceDays:  1,
		Timezone:      "Asia/Shanghai",
		BenefitType:   BenefitGrantTypeBalance,
		BalanceAmount: 0.000000001,
	}, now)

	require.Error(t, err)
	require.Equal(t, "BENEFIT_GRANT_INVALID_INPUT", infraerrors.Reason(err))
}

func TestClassifyBenefitSubscriptionHonorsMarkerAndConflictPolicy(t *testing.T) {
	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	marker := "[benefit-grant:today]"

	eligibility, action := classifyBenefitSubscription(
		BenefitGrantSubscriptionState{
			Status:    SubscriptionStatusExpired,
			ExpiresAt: now.Add(-time.Hour),
			Notes:     "older note\n" + marker,
		},
		true,
		marker,
		BenefitGrantConflictExtendActive,
		now,
	)
	require.Equal(t, BenefitGrantEligibilityAlreadyGranted, eligibility)
	require.Equal(t, BenefitGrantActionNone, action)

	eligibility, action = classifyBenefitSubscription(
		BenefitGrantSubscriptionState{
			Status:    SubscriptionStatusExpired,
			ExpiresAt: now.Add(-time.Hour),
		},
		true,
		marker,
		BenefitGrantConflictSkipActive,
		now,
	)
	require.Equal(t, BenefitGrantEligibilityEligible, eligibility)
	require.Equal(t, BenefitGrantActionRenew, action)

	eligibility, action = classifyBenefitSubscription(
		BenefitGrantSubscriptionState{
			Status:    SubscriptionStatusActive,
			ExpiresAt: now.Add(time.Hour),
		},
		true,
		marker,
		BenefitGrantConflictExtendActive,
		now,
	)
	require.Equal(t, BenefitGrantEligibilityEligible, eligibility)
	require.Equal(t, BenefitGrantActionExtend, action)

	eligibility, action = classifyBenefitSubscription(
		BenefitGrantSubscriptionState{
			Status:    SubscriptionStatusActive,
			ExpiresAt: now.Add(time.Hour),
		},
		true,
		marker,
		BenefitGrantConflictSkipActive,
		now,
	)
	require.Equal(t, BenefitGrantEligibilityConflict, eligibility)
	require.Equal(t, BenefitGrantActionNone, action)

	eligibility, action = classifyBenefitSubscription(
		BenefitGrantSubscriptionState{
			Status:    SubscriptionStatusSuspended,
			ExpiresAt: now.Add(-time.Hour),
		},
		true,
		marker,
		BenefitGrantConflictExtendActive,
		now,
	)
	require.Equal(t, BenefitGrantEligibilityConflict, eligibility)
	require.Equal(t, BenefitGrantActionNone, action)
}

func TestBenefitGrantSnapshotTokenIncludesRecipientIdentityAndDecision(t *testing.T) {
	recipients := []BenefitGrantRecipient{{
		UserID:        7,
		Eligibility:   BenefitGrantEligibilityEligible,
		PlannedAction: BenefitGrantActionCreate,
	}}
	first := benefitGrantSnapshotToken("request-hash", recipients)

	recipients[0].PlannedAction = BenefitGrantActionExtend
	second := benefitGrantSnapshotToken("request-hash", recipients)

	require.Len(t, first, 64)
	require.NotEqual(t, first, second)
}
