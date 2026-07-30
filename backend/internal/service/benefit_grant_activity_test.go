package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type automaticBenefitGrantReplayRepoStub struct {
	BenefitGrantRepository
	campaign *BenefitGrantCampaign
}

func (s *automaticBenefitGrantReplayRepoStub) GetCampaignByOperationKey(
	context.Context,
	int64,
	string,
) (*BenefitGrantCampaign, error) {
	return s.campaign, nil
}

func TestNormalizeAutomaticBenefitGrantUsesAuthenticatedActivityWindow(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	start := now.Add(time.Hour)
	end := start.Add(12 * time.Hour)

	normalized, err := normalizeAutomaticBenefitGrantInput(
		&CreateAutomaticBenefitGrantInput{
			OperationKey:        "benefit-grant-automatic-test-1",
			Timezone:            "Asia/Shanghai",
			WindowStart:         start,
			WindowEnd:           end,
			BenefitType:         BenefitGrantTypeBalance,
			BalanceAmount:       2.5,
			AnnouncementEnabled: true,
			AnnouncementTitle:   "Benefit received",
			AnnouncementContent: "Your campaign benefit is ready.",
			AnnouncementNotify:  AnnouncementNotifyModePopup,
		},
		now,
	)

	require.NoError(t, err)
	require.Equal(t, BenefitGrantDeliveryActivityWindow, normalized.deliveryMode)
	require.Equal(t, BenefitGrantAudienceAuthenticated, normalized.AudienceType)
	require.Equal(t, start, normalized.windowStart)
	require.Equal(t, end, normalized.windowEnd)
	require.NotNil(t, normalized.announcement)
	require.Equal(t, start, *normalized.announcement.StartsAt)
	require.Len(t, normalized.requestHash, 64)
}

func TestNormalizeAutomaticBenefitGrantRejectsExpiredWindow(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)

	_, err := normalizeAutomaticBenefitGrantInput(
		&CreateAutomaticBenefitGrantInput{
			OperationKey:  "benefit-grant-automatic-test-2",
			Timezone:      "Asia/Shanghai",
			WindowStart:   now.Add(-2 * time.Hour),
			WindowEnd:     now.Add(-time.Hour),
			BenefitType:   BenefitGrantTypeBalance,
			BalanceAmount: 1,
		},
		now,
	)

	require.Error(t, err)
}

func TestNormalizeAutomaticBenefitGrantReplayAllowsExpiredWindow(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	start := now.Add(-2 * time.Hour)
	end := now.Add(-time.Hour)

	normalized, err := normalizeAutomaticBenefitGrantInputForReplay(
		&CreateAutomaticBenefitGrantInput{
			OperationKey:  "benefit-grant-automatic-test-3",
			Timezone:      "Asia/Shanghai",
			WindowStart:   start,
			WindowEnd:     end,
			BenefitType:   BenefitGrantTypeBalance,
			BalanceAmount: 1,
		},
		now,
	)

	require.NoError(t, err)
	require.Equal(t, start, normalized.windowStart)
	require.Equal(t, end, normalized.windowEnd)
}

func TestCreateAutomaticReplaysExistingCampaignAfterWindowEnds(t *testing.T) {
	now := time.Now()
	input := &CreateAutomaticBenefitGrantInput{
		OperationKey:  "benefit-grant-automatic-test-4",
		Timezone:      "Asia/Shanghai",
		WindowStart:   now.Add(-2 * time.Hour),
		WindowEnd:     now.Add(-time.Hour),
		BenefitType:   BenefitGrantTypeBalance,
		BalanceAmount: 1,
		AssignedBy:    9,
	}
	normalized, err := normalizeAutomaticBenefitGrantInputForReplay(input, now)
	require.NoError(t, err)

	existing := &BenefitGrantCampaign{
		ID:          17,
		RequestHash: normalized.requestHash,
	}
	svc := NewBenefitGrantService(
		&automaticBenefitGrantReplayRepoStub{campaign: existing},
		nil,
		nil,
		nil,
	)

	campaign, err := svc.CreateAutomatic(context.Background(), input)

	require.NoError(t, err)
	require.Same(t, existing, campaign)
}

func TestFilterActiveAutomaticCampaignsIncludesOnlyCurrentWindow(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	campaigns := []BenefitGrantCampaign{
		{ID: 1, WindowStart: now.Add(-time.Minute), WindowEnd: now.Add(time.Minute)},
		{ID: 2, WindowStart: now.Add(time.Second), WindowEnd: now.Add(time.Minute)},
		{ID: 3, WindowStart: now.Add(-time.Minute), WindowEnd: now},
	}

	active := filterActiveAutomaticCampaigns(campaigns, now)

	require.Len(t, active, 1)
	require.Equal(t, int64(1), active[0].ID)
}
