package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

type accountContributionQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type AccountContributionFeatureState struct {
	Enabled              bool `json:"enabled"`
	SubmissionConfigured bool `json:"submission_configured"`
	PayoutConfigured     bool `json:"payout_configured"`
	SubmissionEnabled    bool `json:"submission_enabled"`
	PayoutEnabled        bool `json:"payout_enabled"`
}

type AccountContributionAdminStats struct {
	ContributorsTotal       int64 `json:"contributors_total"`
	ContributorsPending     int64 `json:"contributors_pending"`
	ContributionsTotal      int64 `json:"contributions_total"`
	ContributionsActive     int64 `json:"contributions_active"`
	EarningEntriesTotal     int64 `json:"earning_entries_total"`
	TotalEarningsCNYFen     int64 `json:"total_earnings_cny_fen"`
	AvailableEarningsCNYFen int64 `json:"available_earnings_cny_fen"`
	PayoutRequestsTotal     int64 `json:"payout_requests_total"`
	PayoutRequestsPending   int64 `json:"payout_requests_pending"`
	PendingPayoutCNYFen     int64 `json:"pending_payout_cny_fen"`
}

type AccountContributionAdminContributor struct {
	ID            int64     `json:"id"`
	UserID        *int64    `json:"user_id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	Status        string    `json:"status"`
	Contributions int64     `json:"contributions"`
	CreatedAt     time.Time `json:"created_at"`
}

type AccountContributionAdminAccount struct {
	ID             int64     `json:"id"`
	ContributorID  int64     `json:"contributor_id"`
	Contributor    string    `json:"contributor"`
	AccountID      *int64    `json:"account_id"`
	AccountName    string    `json:"account_name"`
	Platform       string    `json:"platform"`
	Status         string    `json:"status"`
	SettlementMode string    `json:"settlement_mode"`
	ShareRateBPS   int       `json:"share_rate_bps"`
	CreatedAt      time.Time `json:"created_at"`
}

type AccountContributionAdminEarning struct {
	ID             int64     `json:"id"`
	ContributorID  int64     `json:"contributor_id"`
	Contributor    string    `json:"contributor"`
	ContributionID int64     `json:"contribution_id"`
	AccountName    string    `json:"account_name"`
	EntryType      string    `json:"entry_type"`
	AmountCNYFen   int64     `json:"amount_cny_fen"`
	AvailableAt    time.Time `json:"available_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type AccountContributionAdminPayout struct {
	ID                int64     `json:"id"`
	ContributorID     int64     `json:"contributor_id"`
	Contributor       string    `json:"contributor"`
	AmountCNYFen      int64     `json:"amount_cny_fen"`
	Status            string    `json:"status"`
	MethodType        string    `json:"method_type"`
	MaskedDestination string    `json:"masked_destination"`
	RequestedAt       time.Time `json:"requested_at"`
}

type AccountContributionAdminOverview struct {
	Features      AccountContributionFeatureState       `json:"features"`
	Stats         AccountContributionAdminStats         `json:"stats"`
	Contributors  []AccountContributionAdminContributor `json:"contributors"`
	Contributions []AccountContributionAdminAccount     `json:"contributions"`
	Earnings      []AccountContributionAdminEarning     `json:"earnings"`
	Payouts       []AccountContributionAdminPayout      `json:"payouts"`
}

func (s *adminServiceImpl) GetAccountContributionOverview(ctx context.Context) (*AccountContributionAdminOverview, error) {
	if s == nil || s.entClient == nil {
		return nil, fmt.Errorf("account contribution overview: database unavailable")
	}
	return loadAccountContributionAdminOverview(ctx, s.entClient)
}

func loadAccountContributionAdminOverview(ctx context.Context, db accountContributionQueryer) (*AccountContributionAdminOverview, error) {
	overview := &AccountContributionAdminOverview{
		Contributors:  make([]AccountContributionAdminContributor, 0),
		Contributions: make([]AccountContributionAdminAccount, 0),
		Earnings:      make([]AccountContributionAdminEarning, 0),
		Payouts:       make([]AccountContributionAdminPayout, 0),
	}

	if err := loadAccountContributionFeatureState(ctx, db, &overview.Features); err != nil {
		return nil, err
	}
	if err := loadAccountContributionStats(ctx, db, &overview.Stats); err != nil {
		return nil, err
	}
	if err := loadRecentContributors(ctx, db, &overview.Contributors); err != nil {
		return nil, err
	}
	if err := loadRecentContributions(ctx, db, &overview.Contributions); err != nil {
		return nil, err
	}
	if err := loadRecentContributionEarnings(ctx, db, &overview.Earnings); err != nil {
		return nil, err
	}
	if err := loadRecentContributionPayouts(ctx, db, &overview.Payouts); err != nil {
		return nil, err
	}

	return overview, nil
}

func loadAccountContributionFeatureState(ctx context.Context, db accountContributionQueryer, state *AccountContributionFeatureState) error {
	rows, err := db.QueryContext(ctx, `
SELECT key, value
FROM settings
WHERE key IN (
    'account_contribution_enabled',
    'account_contribution_submission_enabled',
    'account_contribution_payout_enabled'
)
ORDER BY key`)
	if err != nil {
		return fmt.Errorf("account contribution overview settings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var key, raw string
		if err := rows.Scan(&key, &raw); err != nil {
			return fmt.Errorf("account contribution overview settings scan: %w", err)
		}
		value, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			value = false
		}
		switch key {
		case SettingKeyAccountContributionEnabled:
			state.Enabled = value
		case SettingKeyAccountContributionSubmissionEnabled:
			state.SubmissionConfigured = value
		case SettingKeyAccountContributionPayoutEnabled:
			state.PayoutConfigured = value
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("account contribution overview settings rows: %w", err)
	}

	state.SubmissionEnabled = state.Enabled && state.SubmissionConfigured
	state.PayoutEnabled = state.Enabled && state.PayoutConfigured
	return nil
}

func loadAccountContributionStats(ctx context.Context, db accountContributionQueryer, stats *AccountContributionAdminStats) error {
	rows, err := db.QueryContext(ctx, `
SELECT
    (SELECT COUNT(*) FROM contributor_profiles),
    (SELECT COUNT(*) FROM contributor_profiles WHERE status = 'pending'),
    (SELECT COUNT(*) FROM account_contributions),
    (SELECT COUNT(*) FROM account_contributions WHERE status = 'active'),
    (SELECT COUNT(*) FROM contributor_earning_ledger),
    COALESCE((SELECT SUM(amount_cny_fen) FROM contributor_earning_ledger), 0),
    COALESCE((
        SELECT SUM(l.amount_cny_fen)
        FROM contributor_earning_ledger l
        WHERE l.available_at <= NOW()
          AND NOT EXISTS (
              SELECT 1
              FROM contributor_payout_items pi
              WHERE pi.earning_entry_id = l.id
          )
    ), 0),
    (SELECT COUNT(*) FROM contributor_payout_requests),
    (SELECT COUNT(*) FROM contributor_payout_requests WHERE status IN ('requested', 'reviewing', 'approved', 'processing')),
    COALESCE((
        SELECT SUM(amount_cny_fen)
        FROM contributor_payout_requests
        WHERE status IN ('requested', 'reviewing', 'approved', 'processing')
    ), 0)
	`)
	if err != nil {
		return fmt.Errorf("account contribution overview stats: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return fmt.Errorf("account contribution overview stats rows: %w", err)
		}
		return fmt.Errorf("account contribution overview stats: no result")
	}
	if err := rows.Scan(
		&stats.ContributorsTotal,
		&stats.ContributorsPending,
		&stats.ContributionsTotal,
		&stats.ContributionsActive,
		&stats.EarningEntriesTotal,
		&stats.TotalEarningsCNYFen,
		&stats.AvailableEarningsCNYFen,
		&stats.PayoutRequestsTotal,
		&stats.PayoutRequestsPending,
		&stats.PendingPayoutCNYFen,
	); err != nil {
		return fmt.Errorf("account contribution overview stats: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("account contribution overview stats rows: %w", err)
	}
	return nil
}

func loadRecentContributors(ctx context.Context, db accountContributionQueryer, target *[]AccountContributionAdminContributor) error {
	rows, err := db.QueryContext(ctx, `
SELECT
    cp.id,
    cp.user_id,
    COALESCE(NULLIF(cp.user_email_snapshot, ''), u.email, ''),
    COALESCE(u.username, ''),
    cp.status,
    COUNT(ac.id),
    cp.created_at
FROM contributor_profiles cp
LEFT JOIN users u ON u.id = cp.user_id
LEFT JOIN account_contributions ac ON ac.contributor_id = cp.id
GROUP BY cp.id, cp.user_id, cp.user_email_snapshot, u.email, u.username, cp.status, cp.created_at
ORDER BY cp.created_at DESC, cp.id DESC
LIMIT 20`)
	if err != nil {
		return fmt.Errorf("account contribution overview contributors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var item AccountContributionAdminContributor
		var userID sql.NullInt64
		if err := rows.Scan(&item.ID, &userID, &item.Email, &item.Username, &item.Status, &item.Contributions, &item.CreatedAt); err != nil {
			return fmt.Errorf("account contribution overview contributor scan: %w", err)
		}
		item.UserID = nullableInt64Pointer(userID)
		*target = append(*target, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("account contribution overview contributor rows: %w", err)
	}
	return nil
}

func loadRecentContributions(ctx context.Context, db accountContributionQueryer, target *[]AccountContributionAdminAccount) error {
	rows, err := db.QueryContext(ctx, `
SELECT
    ac.id,
    ac.contributor_id,
    COALESCE(NULLIF(cp.user_email_snapshot, ''), u.email, ''),
    ac.account_id,
    COALESCE(NULLIF(ac.account_name_snapshot, ''), a.name, ''),
    COALESCE(NULLIF(ac.platform_snapshot, ''), a.platform, ''),
    ac.status,
    ac.settlement_mode,
    ac.share_rate_bps,
    ac.created_at
FROM account_contributions ac
JOIN contributor_profiles cp ON cp.id = ac.contributor_id
LEFT JOIN users u ON u.id = cp.user_id
LEFT JOIN accounts a ON a.id = ac.account_id
ORDER BY ac.created_at DESC, ac.id DESC
LIMIT 20`)
	if err != nil {
		return fmt.Errorf("account contribution overview accounts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var item AccountContributionAdminAccount
		var accountID sql.NullInt64
		if err := rows.Scan(
			&item.ID,
			&item.ContributorID,
			&item.Contributor,
			&accountID,
			&item.AccountName,
			&item.Platform,
			&item.Status,
			&item.SettlementMode,
			&item.ShareRateBPS,
			&item.CreatedAt,
		); err != nil {
			return fmt.Errorf("account contribution overview account scan: %w", err)
		}
		item.AccountID = nullableInt64Pointer(accountID)
		*target = append(*target, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("account contribution overview account rows: %w", err)
	}
	return nil
}

func loadRecentContributionEarnings(ctx context.Context, db accountContributionQueryer, target *[]AccountContributionAdminEarning) error {
	rows, err := db.QueryContext(ctx, `
SELECT
    l.id,
    l.contributor_id,
    COALESCE(NULLIF(cp.user_email_snapshot, ''), u.email, ''),
    l.contribution_id,
    COALESCE(NULLIF(ac.account_name_snapshot, ''), a.name, ''),
    l.entry_type,
    l.amount_cny_fen,
    l.available_at,
    l.created_at
FROM contributor_earning_ledger l
JOIN contributor_profiles cp ON cp.id = l.contributor_id
JOIN account_contributions ac ON ac.id = l.contribution_id
LEFT JOIN users u ON u.id = cp.user_id
LEFT JOIN accounts a ON a.id = ac.account_id
ORDER BY l.created_at DESC, l.id DESC
LIMIT 20`)
	if err != nil {
		return fmt.Errorf("account contribution overview earnings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var item AccountContributionAdminEarning
		if err := rows.Scan(
			&item.ID,
			&item.ContributorID,
			&item.Contributor,
			&item.ContributionID,
			&item.AccountName,
			&item.EntryType,
			&item.AmountCNYFen,
			&item.AvailableAt,
			&item.CreatedAt,
		); err != nil {
			return fmt.Errorf("account contribution overview earning scan: %w", err)
		}
		*target = append(*target, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("account contribution overview earning rows: %w", err)
	}
	return nil
}

func loadRecentContributionPayouts(ctx context.Context, db accountContributionQueryer, target *[]AccountContributionAdminPayout) error {
	rows, err := db.QueryContext(ctx, `
SELECT
    pr.id,
    pr.contributor_id,
    COALESCE(NULLIF(cp.user_email_snapshot, ''), u.email, ''),
    pr.amount_cny_fen,
    pr.status,
    COALESCE(pm.method_type, ''),
    COALESCE(pm.masked_destination, ''),
    pr.requested_at
FROM contributor_payout_requests pr
JOIN contributor_profiles cp ON cp.id = pr.contributor_id
LEFT JOIN users u ON u.id = cp.user_id
LEFT JOIN contributor_payout_methods pm ON pm.id = pr.payout_method_id
ORDER BY pr.requested_at DESC, pr.id DESC
LIMIT 20`)
	if err != nil {
		return fmt.Errorf("account contribution overview payouts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var item AccountContributionAdminPayout
		if err := rows.Scan(
			&item.ID,
			&item.ContributorID,
			&item.Contributor,
			&item.AmountCNYFen,
			&item.Status,
			&item.MethodType,
			&item.MaskedDestination,
			&item.RequestedAt,
		); err != nil {
			return fmt.Errorf("account contribution overview payout scan: %w", err)
		}
		*target = append(*target, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("account contribution overview payout rows: %w", err)
	}
	return nil
}

func nullableInt64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}
