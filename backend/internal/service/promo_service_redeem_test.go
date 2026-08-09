//go:build unit

package service

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"
)

type sharedPromoRedeemRepoStub struct {
	PromoCodeRepository
	promo  *PromoCode
	usages map[int64]*PromoCodeUsage
	nextID int64
}

func (r *sharedPromoRedeemRepoStub) GetByCodeForUpdate(_ context.Context, code string) (*PromoCode, error) {
	if r.promo == nil || !strings.EqualFold(strings.TrimSpace(code), r.promo.Code) {
		return nil, ErrPromoCodeNotFound
	}
	clone := *r.promo
	return &clone, nil
}

func (r *sharedPromoRedeemRepoStub) GetUsageByPromoCodeAndUser(_ context.Context, promoCodeID, userID int64) (*PromoCodeUsage, error) {
	usage := r.usages[userID]
	if usage == nil || usage.PromoCodeID != promoCodeID {
		return nil, nil
	}
	clone := *usage
	return &clone, nil
}

func (r *sharedPromoRedeemRepoStub) CreateUsage(_ context.Context, usage *PromoCodeUsage) error {
	r.nextID++
	usage.ID = r.nextID
	clone := *usage
	r.usages[usage.UserID] = &clone
	return nil
}

func (r *sharedPromoRedeemRepoStub) IncrementUsedCount(_ context.Context, id int64) error {
	if r.promo != nil && r.promo.ID == id {
		r.promo.UsedCount++
	}
	return nil
}

func (r *sharedPromoRedeemRepoStub) ListUsagesByUser(_ context.Context, userID int64, limit int) ([]PromoCodeUsage, error) {
	usage := r.usages[userID]
	if usage == nil || limit == 0 {
		return []PromoCodeUsage{}, nil
	}
	clone := *usage
	if r.promo != nil {
		promoClone := *r.promo
		clone.PromoCode = &promoClone
	}
	return []PromoCodeUsage{clone}, nil
}

type sharedPromoUserRepoStub struct {
	UserRepository
	balances map[int64]float64
}

func (r *sharedPromoUserRepoStub) UpdateBalance(_ context.Context, userID int64, amount float64) error {
	r.balances[userID] += amount
	return nil
}

func newPromoRedeemTestClient(t *testing.T) *dbent.Client {
	t.Helper()

	db, err := sql.Open("sqlite", "file:promo_service_redeem?mode=memory&cache=shared&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	driver := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(driver)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestPromoServiceRedeemSharedCodeOncePerUser(t *testing.T) {
	future := time.Now().Add(time.Hour)
	promoRepo := &sharedPromoRedeemRepoStub{
		promo: &PromoCode{
			ID:          11,
			Code:        "SUMMER2026",
			BonusAmount: 5,
			MaxUses:     0,
			Status:      PromoCodeStatusActive,
			ExpiresAt:   &future,
		},
		usages: make(map[int64]*PromoCodeUsage),
	}
	userRepo := &sharedPromoUserRepoStub{balances: make(map[int64]float64)}
	service := NewPromoService(promoRepo, userRepo, nil, newPromoRedeemTestClient(t), nil)

	firstUsage, err := service.Redeem(context.Background(), 101, "summer2026")
	require.NoError(t, err)
	require.Equal(t, "SUMMER2026", firstUsage.PromoCode.Code)
	require.Equal(t, 5.0, userRepo.balances[101])

	_, err = service.Redeem(context.Background(), 101, "SUMMER2026")
	require.ErrorIs(t, err, ErrPromoCodeAlreadyUsed)
	require.Equal(t, 5.0, userRepo.balances[101])

	secondUsage, err := service.Redeem(context.Background(), 202, "SUMMER2026")
	require.NoError(t, err)
	require.Equal(t, int64(202), secondUsage.UserID)
	require.Equal(t, 5.0, userRepo.balances[202])
	require.Equal(t, 2, promoRepo.promo.UsedCount)

	history, err := service.ListUserUsages(context.Background(), 101, 25)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "SUMMER2026", history[0].PromoCode.Code)

	userIDs := make([]int, 0, len(promoRepo.usages))
	for userID := range promoRepo.usages {
		userIDs = append(userIDs, int(userID))
	}
	sort.Ints(userIDs)
	require.Equal(t, []int{101, 202}, userIDs)
}
