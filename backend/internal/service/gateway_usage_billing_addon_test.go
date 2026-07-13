package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildUsageBillingCommandUsesSubscriptionAddon(t *testing.T) {
	addon := &SubscriptionAddonPack{
		ID:        77,
		QuotaUSD:  20,
		StartsAt:  time.Now().Add(-time.Hour),
		ExpiresAt: time.Now().Add(time.Hour),
		Status:    SubscriptionAddonStatusActive,
	}
	sub := &UserSubscription{ID: 33, ActiveAddon: addon}
	cmd := buildUsageBillingCommand("req-addon", nil, &postUsageBillingParams{
		Cost:               &CostBreakdown{TotalCost: 2, ActualCost: 1.5},
		User:               &User{ID: 1},
		APIKey:             &APIKey{ID: 2},
		Account:            &Account{ID: 3},
		Subscription:       sub,
		IsSubscriptionBill: true,
	})

	require.NotNil(t, cmd)
	require.Equal(t, int64(33), *cmd.SubscriptionID)
	require.Equal(t, int64(77), *cmd.AddonPackID)
	require.Equal(t, 1.5, cmd.AddonCost)
	require.Zero(t, cmd.SubscriptionCost)
}
