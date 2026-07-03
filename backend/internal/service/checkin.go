package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	checkinRewardAmount = 0.02
	checkinTimezone     = "Asia/Shanghai"
)

var ErrCheckinAlreadyClaimed = infraerrors.Conflict("CHECKIN_ALREADY_CLAIMED", "already checked in today")

type UserCheckin struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	CheckinDate  string     `json:"checkin_date"`
	RewardAmount float64    `json:"reward_amount"`
	StreakDays   int        `json:"streak_days"`
	BalanceAfter *float64   `json:"balance_after,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

type CheckinStatus struct {
	CheckedIn       bool          `json:"checked_in"`
	Today           string        `json:"today"`
	RewardAmount    float64       `json:"reward_amount"`
	CurrentStreak   int           `json:"current_streak"`
	LastCheckinDate string        `json:"last_checkin_date,omitempty"`
	NextCheckinAt   time.Time     `json:"next_checkin_at"`
	TodayCheckin    *UserCheckin  `json:"today_checkin,omitempty"`
	RecentCheckins  []UserCheckin `json:"recent_checkins"`
}

type CheckinRepository interface {
	GetByUserAndDate(ctx context.Context, userID int64, date string) (*UserCheckin, error)
	GetLatestBeforeDate(ctx context.Context, userID int64, date string) (*UserCheckin, error)
	GetLatest(ctx context.Context, userID int64) (*UserCheckin, error)
	Create(ctx context.Context, checkin *UserCheckin) error
	SetBalanceAfter(ctx context.Context, id int64, balance float64) error
	ListRecent(ctx context.Context, userID int64, limit int) ([]UserCheckin, error)
}

type CheckinService struct {
	repo                 CheckinRepository
	userRepo             UserRepository
	entClient            *dbent.Client
	billingCacheService  *BillingCacheService
	authCacheInvalidator APIKeyAuthCacheInvalidator
	location             *time.Location
}

func NewCheckinService(repo CheckinRepository, userRepo UserRepository, entClient *dbent.Client, billingCacheService *BillingCacheService, authCacheInvalidator APIKeyAuthCacheInvalidator) *CheckinService {
	loc, err := time.LoadLocation(checkinTimezone)
	if err != nil {
		loc = time.FixedZone(checkinTimezone, 8*60*60)
	}
	return &CheckinService{
		repo:                 repo,
		userRepo:             userRepo,
		entClient:            entClient,
		billingCacheService:  billingCacheService,
		authCacheInvalidator: authCacheInvalidator,
		location:             loc,
	}
}

func (s *CheckinService) GetStatus(ctx context.Context, userID int64) (*CheckinStatus, error) {
	today, next := s.todayAndNext(time.Now())
	todayCheckin, err := s.repo.GetByUserAndDate(ctx, userID, today)
	if err != nil {
		return nil, fmt.Errorf("get today checkin: %w", err)
	}
	latest, err := s.repo.GetLatest(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get latest checkin: %w", err)
	}
	recent, err := s.repo.ListRecent(ctx, userID, 14)
	if err != nil {
		return nil, fmt.Errorf("list recent checkins: %w", err)
	}

	status := &CheckinStatus{
		CheckedIn:      todayCheckin != nil,
		Today:          today,
		RewardAmount:   checkinRewardAmount,
		CurrentStreak:  currentStreakForLatest(latest, today, s.yesterdayDate(time.Now())),
		NextCheckinAt:  next,
		TodayCheckin:   todayCheckin,
		RecentCheckins: recent,
	}
	if latest != nil {
		status.LastCheckinDate = latest.CheckinDate
	}
	return status, nil
}

func (s *CheckinService) CheckIn(ctx context.Context, userID int64) (*CheckinStatus, error) {
	now := time.Now()
	today, _ := s.todayAndNext(now)
	yesterday := s.yesterdayDate(now)

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)

	latest, err := s.repo.GetLatestBeforeDate(txCtx, userID, today)
	if err != nil {
		return nil, fmt.Errorf("get previous checkin: %w", err)
	}
	streak := 1
	if latest != nil && latest.CheckinDate == yesterday {
		streak = latest.StreakDays + 1
	}

	checkin := &UserCheckin{
		UserID:       userID,
		CheckinDate:  today,
		RewardAmount: checkinRewardAmount,
		StreakDays:   streak,
	}
	if err := s.repo.Create(txCtx, checkin); err != nil {
		return nil, err
	}
	if err := s.userRepo.UpdateBalance(txCtx, userID, checkinRewardAmount); err != nil {
		return nil, fmt.Errorf("update user balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	if user, err := s.userRepo.GetByID(ctx, userID); err == nil {
		if err := s.repo.SetBalanceAfter(ctx, checkin.ID, user.Balance); err != nil {
			slog.Error("set checkin balance_after failed", "user_id", userID, "checkin_id", checkin.ID, "error", err)
		}
	} else {
		slog.Error("get user after checkin failed", "user_id", userID, "error", err)
	}
	s.invalidateBalanceCaches(ctx, userID)
	return s.GetStatus(ctx, userID)
}

func (s *CheckinService) ListHistory(ctx context.Context, userID int64, limit int) ([]UserCheckin, error) {
	if limit <= 0 || limit > 60 {
		limit = 30
	}
	return s.repo.ListRecent(ctx, userID, limit)
}

func (s *CheckinService) invalidateBalanceCaches(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in checkin balance cache invalidation", "user_id", userID, "recover", r)
			}
		}()
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.billingCacheService.InvalidateUserBalance(cacheCtx, userID); err != nil {
			slog.Error("invalidate checkin balance cache failed", "user_id", userID, "error", err)
		}
	}()
}

func (s *CheckinService) todayAndNext(now time.Time) (string, time.Time) {
	local := now.In(s.location)
	today := local.Format("2006-01-02")
	nextLocal := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, s.location)
	return today, nextLocal
}

func (s *CheckinService) yesterdayDate(now time.Time) string {
	return now.In(s.location).AddDate(0, 0, -1).Format("2006-01-02")
}

func currentStreakForLatest(latest *UserCheckin, today, yesterday string) int {
	if latest == nil {
		return 0
	}
	if latest.CheckinDate == today || latest.CheckinDate == yesterday {
		return latest.StreakDays
	}
	return 0
}
