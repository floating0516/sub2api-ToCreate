package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	LeaderboardPeriodWeek  = "week"
	LeaderboardPeriodMonth = "month"

	LeaderboardRewardDailyCard  = "daily_card"
	LeaderboardRewardWeeklyCard = "weekly_card"

	SettingKeyLeaderboardRewardSettings = "leaderboard_reward_settings"
)

var (
	ErrLeaderboardPeriodInvalid   = infraerrors.BadRequest("LEADERBOARD_PERIOD_INVALID", "period must be week or month")
	ErrLeaderboardSettingsInvalid = infraerrors.BadRequest(
		"LEADERBOARD_SETTINGS_INVALID",
		"leaderboard reward settings are incomplete",
	)
	ErrLeaderboardRewardUnavailable = infraerrors.BadRequest(
		"LEADERBOARD_REWARD_UNAVAILABLE",
		"leaderboard reward is unavailable for this period",
	)
)

type LeaderboardRepository interface {
	GetPreference(ctx context.Context, userID int64) (*LeaderboardPreference, error)
	SetPreference(ctx context.Context, userID int64, anonymous bool) (*LeaderboardPreference, error)
	GetTodayTokens(ctx context.Context, userID int64, start, end time.Time) (int64, error)
	ListRankings(ctx context.Context, start, end time.Time, limit int) ([]LeaderboardEntry, error)
	GetUserRank(ctx context.Context, userID int64, start, end time.Time) (*LeaderboardEntry, error)
	CreateReward(ctx context.Context, reward *LeaderboardReward) error
	ListRewards(ctx context.Context, period string, start time.Time) ([]LeaderboardReward, error)
}

type LeaderboardPreference struct {
	UserID    int64     `json:"user_id"`
	Anonymous bool      `json:"anonymous"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LeaderboardEntry struct {
	Rank        int64  `json:"rank"`
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username,omitempty"`
	Email       string `json:"email,omitempty"`
	Role        string `json:"role,omitempty"`
	Anonymous   bool   `json:"anonymous"`
	TokenCount  int64  `json:"token_count"`
}

type LeaderboardReward struct {
	ID           int64     `json:"id"`
	Period       string    `json:"period"`
	PeriodStart  time.Time `json:"period_start"`
	PeriodEnd    time.Time `json:"period_end"`
	Rank         int       `json:"rank"`
	UserID       int64     `json:"user_id"`
	TokenCount   int64     `json:"token_count"`
	RewardType   string    `json:"reward_type"`
	RedeemCodeID int64     `json:"redeem_code_id"`
	RedeemCode   string    `json:"redeem_code,omitempty"`
	CreatedBy    *int64    `json:"created_by,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type LeaderboardRewardSettings struct {
	Enabled             bool   `json:"enabled"`
	SubscriptionGroupID *int64 `json:"subscription_group_id,omitempty"`
	WeeklyFirstDays     int    `json:"weekly_first_days"`
	MonthlyFirstDays    int    `json:"monthly_first_days"`
}

type LeaderboardSnapshot struct {
	Period         string                    `json:"period"`
	PeriodStart    time.Time                 `json:"period_start"`
	PeriodEnd      time.Time                 `json:"period_end"`
	TodayTokens    int64                     `json:"today_tokens"`
	MyRank         *LeaderboardEntry         `json:"my_rank,omitempty"`
	Entries        []LeaderboardEntry        `json:"entries"`
	Preference     LeaderboardPreference     `json:"preference"`
	RewardSettings LeaderboardRewardSettings `json:"reward_settings"`
}

type AdminLeaderboardSnapshot struct {
	Period         string                    `json:"period"`
	PeriodStart    time.Time                 `json:"period_start"`
	PeriodEnd      time.Time                 `json:"period_end"`
	Entries        []LeaderboardEntry        `json:"entries"`
	Rewards        []LeaderboardReward       `json:"rewards"`
	RewardSettings LeaderboardRewardSettings `json:"reward_settings"`
}

type GenerateLeaderboardRewardInput struct {
	Period    string
	CreatedBy int64
}

type GenerateLeaderboardRewardResult struct {
	Reward LeaderboardReward `json:"reward"`
	Code   string            `json:"code"`
}

type LeaderboardService struct {
	repo        LeaderboardRepository
	settingRepo SettingRepository
	redeemSvc   *RedeemService
}

func NewLeaderboardService(repo LeaderboardRepository, settingRepo SettingRepository, redeemSvc *RedeemService) *LeaderboardService {
	return &LeaderboardService{repo: repo, settingRepo: settingRepo, redeemSvc: redeemSvc}
}

func NormalizeLeaderboardPeriod(period string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "", LeaderboardPeriodWeek:
		return LeaderboardPeriodWeek, nil
	case LeaderboardPeriodMonth:
		return LeaderboardPeriodMonth, nil
	default:
		return "", ErrLeaderboardPeriodInvalid
	}
}

func DefaultLeaderboardRewardSettings() LeaderboardRewardSettings {
	return LeaderboardRewardSettings{
		Enabled:          false,
		WeeklyFirstDays:  1,
		MonthlyFirstDays: 7,
	}
}

func (s *LeaderboardService) GetRewardSettings(ctx context.Context) (LeaderboardRewardSettings, error) {
	settings := DefaultLeaderboardRewardSettings()
	if s == nil || s.settingRepo == nil {
		return settings, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyLeaderboardRewardSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return settings, nil
		}
		return settings, err
	}
	if strings.TrimSpace(raw) == "" {
		return settings, nil
	}
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return settings, fmt.Errorf("parse leaderboard reward settings: %w", err)
	}
	normalizeLeaderboardRewardSettings(&settings)
	return settings, nil
}

func (s *LeaderboardService) UpdateRewardSettings(ctx context.Context, settings LeaderboardRewardSettings) (LeaderboardRewardSettings, error) {
	normalizeLeaderboardRewardSettings(&settings)
	if settings.SubscriptionGroupID != nil && *settings.SubscriptionGroupID <= 0 {
		return settings, infraerrors.BadRequest("LEADERBOARD_GROUP_INVALID", "subscription_group_id must be positive")
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		return settings, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyLeaderboardRewardSettings, string(raw)); err != nil {
		return settings, err
	}
	return settings, nil
}

func normalizeLeaderboardRewardSettings(settings *LeaderboardRewardSettings) {
	if settings.WeeklyFirstDays <= 0 {
		settings.WeeklyFirstDays = 1
	}
	if settings.MonthlyFirstDays <= 0 {
		settings.MonthlyFirstDays = 7
	}
}

func (s *LeaderboardService) GetUserSnapshot(ctx context.Context, userID int64, period string, limit int, now time.Time, loc *time.Location) (*LeaderboardSnapshot, error) {
	period, err := NormalizeLeaderboardPeriod(period)
	if err != nil {
		return nil, err
	}
	limit = normalizeLeaderboardLimit(limit)
	start, end := leaderboardWindow(period, now, loc)
	todayStart, todayEnd := todayWindow(now, loc)

	pref, err := s.repo.GetPreference(ctx, userID)
	if err != nil {
		return nil, err
	}
	todayTokens, err := s.repo.GetTodayTokens(ctx, userID, todayStart, todayEnd)
	if err != nil {
		return nil, err
	}
	entries, err := s.repo.ListRankings(ctx, start, end, limit)
	if err != nil {
		return nil, err
	}
	redactLeaderboardEntries(entries, false)
	myRank, err := s.repo.GetUserRank(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}
	if myRank != nil {
		redactLeaderboardEntry(myRank, false)
	}
	settings, err := s.GetRewardSettings(ctx)
	if err != nil {
		return nil, err
	}

	return &LeaderboardSnapshot{
		Period:         period,
		PeriodStart:    start,
		PeriodEnd:      end,
		TodayTokens:    todayTokens,
		MyRank:         myRank,
		Entries:        entries,
		Preference:     *pref,
		RewardSettings: settings,
	}, nil
}

func (s *LeaderboardService) GetAdminSnapshot(ctx context.Context, period string, limit int, now time.Time, loc *time.Location) (*AdminLeaderboardSnapshot, error) {
	period, err := NormalizeLeaderboardPeriod(period)
	if err != nil {
		return nil, err
	}
	limit = normalizeLeaderboardLimit(limit)
	start, end := leaderboardWindow(period, now, loc)
	entries, err := s.repo.ListRankings(ctx, start, end, limit)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		entries[i].DisplayName = resolveLeaderboardDisplayName(entries[i], true)
	}
	rewards, err := s.repo.ListRewards(ctx, period, start)
	if err != nil {
		return nil, err
	}
	settings, err := s.GetRewardSettings(ctx)
	if err != nil {
		return nil, err
	}
	return &AdminLeaderboardSnapshot{
		Period:         period,
		PeriodStart:    start,
		PeriodEnd:      end,
		Entries:        entries,
		Rewards:        rewards,
		RewardSettings: settings,
	}, nil
}

func (s *LeaderboardService) SetPreference(ctx context.Context, userID int64, anonymous bool) (*LeaderboardPreference, error) {
	return s.repo.SetPreference(ctx, userID, anonymous)
}

func (s *LeaderboardService) GenerateReward(ctx context.Context, input GenerateLeaderboardRewardInput, now time.Time, loc *time.Location) (*GenerateLeaderboardRewardResult, error) {
	period, err := NormalizeLeaderboardPeriod(input.Period)
	if err != nil {
		return nil, err
	}
	settings, err := s.GetRewardSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled || settings.SubscriptionGroupID == nil || *settings.SubscriptionGroupID <= 0 {
		return nil, ErrLeaderboardSettingsInvalid
	}
	start, end := leaderboardWindow(period, now, loc)
	top, err := s.repo.ListRankings(ctx, start, end, 1)
	if err != nil {
		return nil, err
	}
	if len(top) == 0 || top[0].TokenCount <= 0 {
		return nil, ErrLeaderboardRewardUnavailable
	}
	rewardType, validityDays := rewardForPeriod(period, settings)
	if rewardType == "" || validityDays <= 0 {
		return nil, ErrLeaderboardRewardUnavailable
	}
	existingRewards, err := s.repo.ListRewards(ctx, period, start)
	if err != nil {
		return nil, err
	}
	for _, existing := range existingRewards {
		if existing.Rank == 1 && existing.RewardType == rewardType {
			return &GenerateLeaderboardRewardResult{
				Reward: existing,
				Code:   existing.RedeemCode,
			}, nil
		}
	}
	code, err := GenerateRedeemCode()
	if err != nil {
		return nil, err
	}
	notes := fmt.Sprintf("Token排行榜奖励：%s榜第1名，周期 %s 至 %s",
		period, start.In(loc).Format("2006-01-02"), end.In(loc).Format("2006-01-02"))
	redeemCode := &RedeemCode{
		Code:         code,
		Type:         RedeemTypeSubscription,
		Value:        1,
		Status:       StatusUnused,
		Notes:        notes,
		GroupID:      settings.SubscriptionGroupID,
		ValidityDays: validityDays,
	}
	if err := s.redeemSvc.CreateCode(ctx, redeemCode); err != nil {
		return nil, err
	}
	createdBy := input.CreatedBy
	reward := &LeaderboardReward{
		Period:       period,
		PeriodStart:  start,
		PeriodEnd:    end,
		Rank:         1,
		UserID:       top[0].UserID,
		TokenCount:   top[0].TokenCount,
		RewardType:   rewardType,
		RedeemCodeID: redeemCode.ID,
		RedeemCode:   redeemCode.Code,
		CreatedBy:    &createdBy,
	}
	if err := s.repo.CreateReward(ctx, reward); err != nil {
		return nil, err
	}
	return &GenerateLeaderboardRewardResult{Reward: *reward, Code: redeemCode.Code}, nil
}

func rewardForPeriod(period string, settings LeaderboardRewardSettings) (string, int) {
	switch period {
	case LeaderboardPeriodWeek:
		return LeaderboardRewardDailyCard, settings.WeeklyFirstDays
	case LeaderboardPeriodMonth:
		return LeaderboardRewardWeeklyCard, settings.MonthlyFirstDays
	default:
		return "", 0
	}
}

func normalizeLeaderboardLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func leaderboardWindow(period string, now time.Time, loc *time.Location) (time.Time, time.Time) {
	if loc == nil {
		loc = time.Local
	}
	localNow := now.In(loc)
	switch period {
	case LeaderboardPeriodMonth:
		startLocal := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, loc)
		return startLocal.UTC(), startLocal.AddDate(0, 1, 0).UTC()
	default:
		weekday := int(localNow.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDay := localNow.AddDate(0, 0, -(weekday - 1))
		startLocal := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, loc)
		return startLocal.UTC(), startLocal.AddDate(0, 0, 7).UTC()
	}
}

func todayWindow(now time.Time, loc *time.Location) (time.Time, time.Time) {
	if loc == nil {
		loc = time.Local
	}
	localNow := now.In(loc)
	startLocal := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)
	return startLocal.UTC(), startLocal.AddDate(0, 0, 1).UTC()
}

func redactLeaderboardEntries(entries []LeaderboardEntry, admin bool) {
	for i := range entries {
		redactLeaderboardEntry(&entries[i], admin)
	}
}

func redactLeaderboardEntry(entry *LeaderboardEntry, admin bool) {
	if entry == nil {
		return
	}
	entry.DisplayName = resolveLeaderboardDisplayName(*entry, admin)
	if !admin {
		entry.Email = ""
		entry.Role = ""
	}
}

func resolveLeaderboardDisplayName(entry LeaderboardEntry, admin bool) string {
	if entry.Anonymous && !admin {
		return fmt.Sprintf("匿名用户 #%04d", entry.UserID%10000)
	}
	if name := strings.TrimSpace(entry.Username); name != "" {
		return name
	}
	if email := strings.TrimSpace(entry.Email); email != "" {
		if admin {
			return email
		}
		return maskLeaderboardEmail(email)
	}
	return fmt.Sprintf("用户 #%d", entry.UserID)
}

func maskLeaderboardEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		if len(email) <= 3 {
			return "***"
		}
		return email[:3] + "***"
	}
	local := parts[0]
	if local == "" {
		return "***@" + parts[1]
	}
	if len(local) <= 2 {
		return local[:1] + "***@" + parts[1]
	}
	return local[:2] + "***@" + parts[1]
}
