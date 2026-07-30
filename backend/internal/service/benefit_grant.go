package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	gocache "github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

const (
	BenefitGrantAudienceTodayActive      = "today_active"
	BenefitGrantAudienceRecentActive     = "recent_active"
	BenefitGrantAudienceRecentRegistered = "recent_registered"
	BenefitGrantAudienceAuthenticated    = "authenticated_activity"

	BenefitGrantDeliverySnapshot       = "snapshot"
	BenefitGrantDeliveryActivityWindow = "activity_window"

	BenefitGrantTypeSubscription = "subscription"
	BenefitGrantTypeBalance      = "balance"

	BenefitGrantConflictSkipActive   = "skip_active"
	BenefitGrantConflictExtendActive = "extend_active"
	BenefitGrantConflictNone         = "none"

	BenefitGrantEligibilityEligible       = "eligible"
	BenefitGrantEligibilityAlreadyGranted = "already_granted"
	BenefitGrantEligibilityConflict       = "conflict"

	BenefitGrantRecipientPending    = "pending"
	BenefitGrantRecipientProcessing = "processing"
	BenefitGrantRecipientGranted    = "granted"
	BenefitGrantRecipientSkipped    = "skipped"
	BenefitGrantRecipientFailed     = "failed"

	BenefitGrantActionCreate     = "created"
	BenefitGrantActionRenew      = "renewed"
	BenefitGrantActionExtend     = "extended"
	BenefitGrantActionBalanceAdd = "balance_added"
	BenefitGrantActionNone       = "none"

	BenefitGrantStatusScheduled = "scheduled"
	BenefitGrantStatusRunning   = "running"
	BenefitGrantStatusCompleted = "completed"
	BenefitGrantStatusPartial   = "partial"
	BenefitGrantStatusFailed    = "failed"

	benefitGrantNotesMaxRunes               = 1800
	benefitGrantAnnouncementContentMaxRunes = 20000
	benefitGrantMaxAudienceDays             = 365
	benefitGrantMaxAudienceUsers            = 100000
	benefitGrantMaxBalanceAmount            = 1000000
	benefitGrantBalanceScale                = 100000000
	benefitGrantRetryStaleAfter             = 10 * time.Minute
	benefitGrantActivityCampaignCacheTTL    = 5 * time.Second
	benefitGrantMaxActivityWindow           = 365 * 24 * time.Hour
)

var (
	ErrBenefitGrantAudienceChanged = infraerrors.Conflict(
		"BENEFIT_GRANT_AUDIENCE_CHANGED",
		"the target audience changed after preview; refresh the preview and try again",
	)
	ErrBenefitGrantUnavailable = infraerrors.ServiceUnavailable(
		"BENEFIT_GRANT_UNAVAILABLE",
		"benefit grant service is unavailable",
	)
	ErrBenefitGrantNotFound = infraerrors.NotFound(
		"BENEFIT_GRANT_NOT_FOUND",
		"benefit grant campaign not found",
	)
	ErrBenefitGrantOperationConflict = infraerrors.Conflict(
		"BENEFIT_GRANT_OPERATION_CONFLICT",
		"the operation key is already associated with a different benefit grant",
	)
)

// BenefitGrantInput keeps audience and benefit variants in one stable API shape.
// Fields that do not apply to the selected variant are normalized to zero values.
type BenefitGrantInput struct {
	OperationKey        string
	AudienceType        string
	AudienceDate        string
	AudienceDays        int
	Timezone            string
	BenefitType         string
	ConflictPolicy      string
	GroupID             int64
	ValidityDays        int
	BalanceAmount       float64
	Notes               string
	AnnouncementEnabled bool
	AnnouncementTitle   string
	AnnouncementContent string
	AnnouncementNotify  string
}

type ExecuteBenefitGrantInput struct {
	BenefitGrantInput
	ExpectedMatchedCount  int
	ExpectedEligibleCount int
	ExpectedSnapshot      string
	AssignedBy            int64
}

type CreateAutomaticBenefitGrantInput struct {
	OperationKey        string
	Timezone            string
	WindowStart         time.Time
	WindowEnd           time.Time
	BenefitType         string
	ConflictPolicy      string
	GroupID             int64
	ValidityDays        int
	BalanceAmount       float64
	Notes               string
	AnnouncementEnabled bool
	AnnouncementTitle   string
	AnnouncementContent string
	AnnouncementNotify  string
	AssignedBy          int64
}

type BenefitGrantAnnouncement struct {
	Title      string
	Content    string
	NotifyMode string
	StartsAt   *time.Time
}

type BenefitGrantPreview struct {
	OperationKey        string    `json:"operation_key"`
	AudienceType        string    `json:"audience_type"`
	AudienceDate        string    `json:"audience_date"`
	AudienceDays        int       `json:"audience_days"`
	Timezone            string    `json:"timezone"`
	WindowStart         time.Time `json:"window_start"`
	WindowEnd           time.Time `json:"window_end"`
	BenefitType         string    `json:"benefit_type"`
	ConflictPolicy      string    `json:"conflict_policy"`
	GroupID             int64     `json:"group_id"`
	ValidityDays        int       `json:"validity_days"`
	BalanceAmount       float64   `json:"balance_amount"`
	MatchedCount        int       `json:"matched_count"`
	EligibleCount       int       `json:"eligible_count"`
	AlreadyGrantedCount int       `json:"already_granted_count"`
	ConflictCount       int       `json:"conflict_count"`
	SnapshotToken       string    `json:"snapshot_token"`
}

type BenefitGrantCampaign struct {
	ID                  int64      `json:"id"`
	OperationKey        string     `json:"-"`
	RequestHash         string     `json:"-"`
	DeliveryMode        string     `json:"delivery_mode"`
	AudienceType        string     `json:"audience_type"`
	AudienceDate        string     `json:"audience_date"`
	AudienceDays        int        `json:"audience_days"`
	Timezone            string     `json:"timezone"`
	WindowStart         time.Time  `json:"window_start"`
	WindowEnd           time.Time  `json:"window_end"`
	BenefitType         string     `json:"benefit_type"`
	ConflictPolicy      string     `json:"conflict_policy"`
	GroupID             *int64     `json:"group_id,omitempty"`
	GroupName           string     `json:"group_name,omitempty"`
	ValidityDays        *int       `json:"validity_days,omitempty"`
	BalanceAmount       *float64   `json:"balance_amount,omitempty"`
	Notes               string     `json:"notes"`
	Marker              string     `json:"-"`
	AnnouncementID      *int64     `json:"announcement_id,omitempty"`
	AnnouncementTitle   string     `json:"announcement_title,omitempty"`
	AnnouncementContent string     `json:"announcement_content,omitempty"`
	AnnouncementNotify  string     `json:"announcement_notify_mode,omitempty"`
	Status              string     `json:"status"`
	MatchedCount        int        `json:"matched_count"`
	EligibleCount       int        `json:"eligible_count"`
	AlreadyGrantedCount int        `json:"already_granted_count"`
	ConflictCount       int        `json:"conflict_count"`
	GrantedCount        int        `json:"granted_count"`
	SkippedCount        int        `json:"skipped_count"`
	FailedCount         int        `json:"failed_count"`
	CreatedCount        int        `json:"created_count"`
	RenewedCount        int        `json:"renewed_count"`
	ExtendedCount       int        `json:"extended_count"`
	BalanceGrantedCount int        `json:"balance_granted_count"`
	CreatedBy           int64      `json:"created_by"`
	StartedAt           time.Time  `json:"started_at"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type BenefitGrantRecipient struct {
	ID               int64      `json:"id"`
	CampaignID       int64      `json:"campaign_id"`
	UserID           int64      `json:"user_id"`
	EmailSnapshot    string     `json:"email"`
	UsernameSnapshot string     `json:"username"`
	Eligibility      string     `json:"eligibility"`
	PlannedAction    string     `json:"planned_action"`
	Status           string     `json:"status"`
	ResultType       string     `json:"result_type,omitempty"`
	SubscriptionID   *int64     `json:"subscription_id,omitempty"`
	BalanceBefore    *float64   `json:"balance_before,omitempty"`
	BalanceAfter     *float64   `json:"balance_after,omitempty"`
	Error            string     `json:"error,omitempty"`
	AttemptCount     int        `json:"attempt_count"`
	LastAttemptAt    *time.Time `json:"last_attempt_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type BenefitGrantResult struct {
	Campaign      *BenefitGrantCampaign `json:"campaign"`
	Preview       *BenefitGrantPreview  `json:"preview,omitempty"`
	GrantedCount  int                   `json:"granted_count"`
	CreatedCount  int                   `json:"created_count"`
	RenewedCount  int                   `json:"renewed_count"`
	ExtendedCount int                   `json:"extended_count"`
	FailedCount   int                   `json:"failed_count"`
	SkippedCount  int                   `json:"skipped_count"`
	Errors        []string              `json:"errors"`
}

type BenefitGrantAudienceUser struct {
	ID       int64
	Email    string
	Username string
}

type BenefitGrantSubscriptionState struct {
	ID        int64
	UserID    int64
	Status    string
	ExpiresAt time.Time
	Notes     string
}

type BenefitGrantBalanceApplyResult struct {
	Claimed       bool
	Granted       bool
	BalanceBefore float64
	BalanceAfter  float64
	Error         string
}

type BenefitGrantRepository interface {
	ListAudience(ctx context.Context, audienceType string, windowStart, windowEnd time.Time) ([]BenefitGrantAudienceUser, error)
	ListSubscriptionStates(ctx context.Context, userIDs []int64, groupID int64) (map[int64]BenefitGrantSubscriptionState, error)
	GetCampaignByOperationKey(ctx context.Context, createdBy int64, operationKey string) (*BenefitGrantCampaign, error)
	CreateCampaign(ctx context.Context, campaign *BenefitGrantCampaign, recipients []BenefitGrantRecipient, announcement *BenefitGrantAnnouncement) (*BenefitGrantCampaign, bool, error)
	GetCampaign(ctx context.Context, id int64) (*BenefitGrantCampaign, error)
	ListCampaigns(ctx context.Context, page, pageSize int) ([]BenefitGrantCampaign, int64, error)
	ListAutomaticCampaignCandidates(ctx context.Context, now, startsBefore time.Time) ([]BenefitGrantCampaign, error)
	ListRecipients(ctx context.Context, campaignID int64, page, pageSize int, status string) ([]BenefitGrantRecipient, int64, error)
	GetRecipient(ctx context.Context, campaignID, userID int64) (*BenefitGrantRecipient, error)
	EnsureAutomaticRecipient(ctx context.Context, campaignID, userID int64, eligibility, plannedAction, status, resultType string) (*BenefitGrantRecipient, bool, error)
	ListActionableRecipients(ctx context.Context, campaignID int64, includeFailed bool, staleBefore time.Time, limit int) ([]BenefitGrantRecipient, error)
	ClaimRecipient(ctx context.Context, campaignID, userID int64, includeFailed bool, staleBefore time.Time) (bool, error)
	MarkRecipientGranted(ctx context.Context, campaignID, userID int64, resultType string, subscriptionID *int64) error
	MarkRecipientFailed(ctx context.Context, campaignID, userID int64, message string) error
	ApplyBalanceRecipient(ctx context.Context, campaignID, userID int64, amount float64, includeFailed bool, staleBefore time.Time) (*BenefitGrantBalanceApplyResult, error)
	RefreshCampaign(ctx context.Context, campaignID int64) (*BenefitGrantCampaign, error)
	ListAnnouncementGrantAccess(ctx context.Context, userID int64, announcementIDs []int64) (map[int64]bool, error)
	ListAnnouncementGrantedUsers(ctx context.Context, announcementID int64, userIDs []int64) (bool, map[int64]struct{}, error)
}

type BenefitGrantService struct {
	repo                BenefitGrantRepository
	groupRepo           GroupRepository
	subscriptionService *SubscriptionService
	billingCacheService *BillingCacheService
	activityCampaignMu  sync.RWMutex
	activityCampaigns   []BenefitGrantCampaign
	activityCacheUntil  time.Time
	activityCampaignSF  singleflight.Group
	activityRecipientSF singleflight.Group
	activityProcessed   *gocache.Cache
}

func NewBenefitGrantService(
	repo BenefitGrantRepository,
	groupRepo GroupRepository,
	subscriptionService *SubscriptionService,
	billingCacheService *BillingCacheService,
) *BenefitGrantService {
	return &BenefitGrantService{
		repo:                repo,
		groupRepo:           groupRepo,
		subscriptionService: subscriptionService,
		billingCacheService: billingCacheService,
		activityProcessed:   gocache.New(30*time.Minute, 10*time.Minute),
	}
}

type normalizedBenefitGrant struct {
	BenefitGrantInput
	deliveryMode string
	windowStart  time.Time
	windowEnd    time.Time
	marker       string
	groupName    string
	grantNotes   string
	requestHash  string
	announcement *BenefitGrantAnnouncement
}

type preparedBenefitGrant struct {
	normalized *normalizedBenefitGrant
	preview    *BenefitGrantPreview
	recipients []BenefitGrantRecipient
}

func (s *BenefitGrantService) Preview(ctx context.Context, input *BenefitGrantInput) (*BenefitGrantPreview, error) {
	now := time.Now()
	normalized, err := normalizeBenefitGrantInput(input, now)
	if err != nil {
		return nil, err
	}
	prepared, err := s.prepare(ctx, normalized, now)
	if err != nil {
		return nil, err
	}
	return prepared.preview, nil
}

func (s *BenefitGrantService) Execute(ctx context.Context, input *ExecuteBenefitGrantInput) (*BenefitGrantResult, error) {
	if input == nil {
		return nil, invalidBenefitGrantInput("request", "request body is required")
	}
	if s == nil || s.repo == nil {
		return nil, ErrBenefitGrantUnavailable
	}
	if err := validateBenefitGrantOperationKey(input.OperationKey); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetCampaignByOperationKey(ctx, input.AssignedBy, strings.TrimSpace(input.OperationKey))
	if err != nil {
		return nil, err
	}
	if existing != nil {
		normalized, normalizeErr := normalizeBenefitGrantInputForReplay(&input.BenefitGrantInput, time.Now())
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		if !benefitGrantRequestMatches(existing, normalized) {
			return nil, ErrBenefitGrantOperationConflict
		}
		return s.processCampaign(ctx, existing, false, nil)
	}

	now := time.Now()
	normalized, err := normalizeBenefitGrantInput(&input.BenefitGrantInput, now)
	if err != nil {
		return nil, err
	}
	prepared, err := s.prepare(ctx, normalized, now)
	if err != nil {
		return nil, err
	}
	preview := prepared.preview
	if preview.MatchedCount != input.ExpectedMatchedCount ||
		preview.EligibleCount != input.ExpectedEligibleCount ||
		preview.SnapshotToken != strings.TrimSpace(input.ExpectedSnapshot) {
		return nil, ErrBenefitGrantAudienceChanged.WithMetadata(map[string]string{
			"expected_matched_count":  strconv.Itoa(input.ExpectedMatchedCount),
			"actual_matched_count":    strconv.Itoa(preview.MatchedCount),
			"expected_eligible_count": strconv.Itoa(input.ExpectedEligibleCount),
			"actual_eligible_count":   strconv.Itoa(preview.EligibleCount),
		})
	}

	campaign := campaignFromPrepared(prepared, input.AssignedBy)
	campaign, created, err := s.repo.CreateCampaign(
		ctx,
		campaign,
		prepared.recipients,
		normalized.announcement,
	)
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		return nil, ErrBenefitGrantUnavailable
	}
	if !created && !benefitGrantRequestMatches(campaign, normalized) {
		return nil, ErrBenefitGrantOperationConflict
	}
	return s.processCampaign(ctx, campaign, false, preview)
}

func (s *BenefitGrantService) CreateAutomatic(
	ctx context.Context,
	input *CreateAutomaticBenefitGrantInput,
) (*BenefitGrantCampaign, error) {
	if input == nil {
		return nil, invalidBenefitGrantInput("request", "request body is required")
	}
	if s == nil || s.repo == nil {
		return nil, ErrBenefitGrantUnavailable
	}
	operationKey := strings.TrimSpace(input.OperationKey)
	if err := validateBenefitGrantOperationKey(operationKey); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetCampaignByOperationKey(
		ctx,
		input.AssignedBy,
		operationKey,
	)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		normalized, normalizeErr := normalizeAutomaticBenefitGrantInputForReplay(
			input,
			time.Now(),
		)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		if existing.RequestHash != normalized.requestHash {
			return nil, ErrBenefitGrantOperationConflict
		}
		return existing, nil
	}

	now := time.Now()
	normalized, err := normalizeAutomaticBenefitGrantInput(input, now)
	if err != nil {
		return nil, err
	}
	if normalized.BenefitType == BenefitGrantTypeSubscription {
		if s.groupRepo == nil || s.subscriptionService == nil {
			return nil, ErrBenefitGrantUnavailable
		}
		group, err := s.groupRepo.GetByID(ctx, normalized.GroupID)
		if err != nil {
			return nil, fmt.Errorf("group not found: %w", err)
		}
		if !group.IsSubscriptionType() {
			return nil, ErrGroupNotSubscriptionType
		}
		if !group.IsActive() {
			return nil, invalidBenefitGrantInput("group_id", "subscription group must be active")
		}
		normalized.groupName = group.Name
	}

	campaign := campaignFromAutomatic(normalized, input.AssignedBy, now)
	campaign, created, err := s.repo.CreateCampaign(ctx, campaign, nil, normalized.announcement)
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		return nil, ErrBenefitGrantUnavailable
	}
	if !created && campaign.RequestHash != normalized.requestHash {
		return nil, ErrBenefitGrantOperationConflict
	}
	s.invalidateActivityCampaignCache()
	return campaign, nil
}

// ProcessAuthenticatedActivity grants every active activity-window campaign to
// an authenticated normal user. Repository constraints keep the operation
// once-per-user even when several requests arrive concurrently.
func (s *BenefitGrantService) ProcessAuthenticatedActivity(ctx context.Context, user *User) error {
	if s == nil || s.repo == nil || user == nil || user.ID <= 0 {
		return nil
	}
	if !user.IsActive() || user.Role != RoleUser {
		return nil
	}

	now := time.Now()
	campaigns, err := s.listActiveAutomaticCampaigns(ctx, now)
	if err != nil {
		return err
	}

	var processErrors []error
	for i := range campaigns {
		campaign := campaigns[i]
		cacheKey := automaticBenefitGrantCacheKey(campaign.ID, user.ID)
		if s.activityProcessed != nil {
			if _, found := s.activityProcessed.Get(cacheKey); found {
				continue
			}
		}

		_, processErr, _ := s.activityRecipientSF.Do(cacheKey, func() (any, error) {
			return nil, s.processAutomaticCampaignForUser(ctx, &campaign, user)
		})
		if processErr != nil {
			processErrors = append(
				processErrors,
				fmt.Errorf("campaign %d: %w", campaign.ID, processErr),
			)
		}
	}
	return errors.Join(processErrors...)
}

func (s *BenefitGrantService) processAutomaticCampaignForUser(
	ctx context.Context,
	campaign *BenefitGrantCampaign,
	user *User,
) error {
	if campaign == nil || user == nil {
		return nil
	}

	eligibility := BenefitGrantEligibilityEligible
	plannedAction := BenefitGrantActionBalanceAdd
	status := BenefitGrantRecipientPending
	resultType := ""
	if campaign.BenefitType == BenefitGrantTypeSubscription {
		if campaign.GroupID == nil {
			return invalidBenefitGrantInput("group_id", "subscription group is missing from campaign")
		}
		states, err := s.repo.ListSubscriptionStates(
			ctx,
			[]int64{user.ID},
			*campaign.GroupID,
		)
		if err != nil {
			return err
		}
		state, exists := states[user.ID]
		eligibility, plannedAction = classifyBenefitSubscription(
			state,
			exists,
			campaign.Marker,
			campaign.ConflictPolicy,
			time.Now(),
		)
		if eligibility != BenefitGrantEligibilityEligible {
			status = BenefitGrantRecipientSkipped
			resultType = eligibility
		}
	}

	recipient, _, err := s.repo.EnsureAutomaticRecipient(
		ctx,
		campaign.ID,
		user.ID,
		eligibility,
		plannedAction,
		status,
		resultType,
	)
	if err != nil || recipient == nil {
		return err
	}
	if recipient.Status == BenefitGrantRecipientPending ||
		recipient.Status == BenefitGrantRecipientProcessing {
		staleBefore := time.Now().Add(-benefitGrantRetryStaleAfter)
		if campaign.BenefitType == BenefitGrantTypeBalance {
			err = s.processBalanceRecipient(ctx, campaign, recipient, false, staleBefore)
		} else {
			err = s.processSubscriptionRecipient(ctx, campaign, recipient, false, staleBefore)
		}
		if err != nil {
			_, _ = s.repo.RefreshCampaign(context.Background(), campaign.ID)
			return err
		}
		recipient, err = s.repo.GetRecipient(ctx, campaign.ID, user.ID)
		if err != nil {
			return err
		}
	}

	if _, err := s.repo.RefreshCampaign(ctx, campaign.ID); err != nil {
		return err
	}
	if recipient != nil && isTerminalBenefitGrantRecipientStatus(recipient.Status) {
		s.cacheAutomaticRecipient(campaign, user.ID)
	}
	return nil
}

func (s *BenefitGrantService) listActiveAutomaticCampaigns(
	ctx context.Context,
	now time.Time,
) ([]BenefitGrantCampaign, error) {
	s.activityCampaignMu.RLock()
	if now.Before(s.activityCacheUntil) {
		cached := append([]BenefitGrantCampaign(nil), s.activityCampaigns...)
		s.activityCampaignMu.RUnlock()
		return filterActiveAutomaticCampaigns(cached, now), nil
	}
	s.activityCampaignMu.RUnlock()

	value, err, _ := s.activityCampaignSF.Do("active", func() (any, error) {
		loadNow := time.Now()
		cacheUntil := loadNow.Add(benefitGrantActivityCampaignCacheTTL)
		loaded, loadErr := s.repo.ListAutomaticCampaignCandidates(
			ctx,
			loadNow,
			cacheUntil,
		)
		if loadErr != nil {
			return nil, loadErr
		}
		s.activityCampaignMu.Lock()
		s.activityCampaigns = append([]BenefitGrantCampaign(nil), loaded...)
		s.activityCacheUntil = cacheUntil
		s.activityCampaignMu.Unlock()
		return loaded, nil
	})
	if err != nil {
		return nil, err
	}
	campaigns, _ := value.([]BenefitGrantCampaign)
	return filterActiveAutomaticCampaigns(campaigns, now), nil
}

func filterActiveAutomaticCampaigns(
	campaigns []BenefitGrantCampaign,
	now time.Time,
) []BenefitGrantCampaign {
	active := make([]BenefitGrantCampaign, 0, len(campaigns))
	for i := range campaigns {
		if now.Before(campaigns[i].WindowStart) || !now.Before(campaigns[i].WindowEnd) {
			continue
		}
		active = append(active, campaigns[i])
	}
	return active
}

func (s *BenefitGrantService) invalidateActivityCampaignCache() {
	if s == nil {
		return
	}
	s.activityCampaignSF.Forget("active")
	s.activityCampaignMu.Lock()
	s.activityCampaigns = nil
	s.activityCacheUntil = time.Time{}
	s.activityCampaignMu.Unlock()
}

func (s *BenefitGrantService) cacheAutomaticRecipient(
	campaign *BenefitGrantCampaign,
	userID int64,
) {
	if s == nil || s.activityProcessed == nil || campaign == nil {
		return
	}
	ttl := time.Until(campaign.WindowEnd) + time.Minute
	if ttl < time.Minute {
		ttl = time.Minute
	}
	s.activityProcessed.Set(
		automaticBenefitGrantCacheKey(campaign.ID, userID),
		true,
		ttl,
	)
}

func automaticBenefitGrantCacheKey(campaignID, userID int64) string {
	return strconv.FormatInt(campaignID, 10) + ":" + strconv.FormatInt(userID, 10)
}

func isTerminalBenefitGrantRecipientStatus(status string) bool {
	switch status {
	case BenefitGrantRecipientGranted,
		BenefitGrantRecipientSkipped,
		BenefitGrantRecipientFailed:
		return true
	default:
		return false
	}
}

func (s *BenefitGrantService) Retry(ctx context.Context, campaignID int64) (*BenefitGrantResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrBenefitGrantUnavailable
	}
	campaign, err := s.repo.GetCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	return s.processCampaign(ctx, campaign, true, nil)
}

func (s *BenefitGrantService) GetCampaign(ctx context.Context, id int64) (*BenefitGrantCampaign, error) {
	if s == nil || s.repo == nil {
		return nil, ErrBenefitGrantUnavailable
	}
	return s.repo.GetCampaign(ctx, id)
}

func (s *BenefitGrantService) ListCampaigns(ctx context.Context, page, pageSize int) ([]BenefitGrantCampaign, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, ErrBenefitGrantUnavailable
	}
	return s.repo.ListCampaigns(ctx, page, pageSize)
}

func (s *BenefitGrantService) ListRecipients(
	ctx context.Context,
	campaignID int64,
	page, pageSize int,
	status string,
) ([]BenefitGrantRecipient, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, ErrBenefitGrantUnavailable
	}
	if _, err := s.repo.GetCampaign(ctx, campaignID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListRecipients(ctx, campaignID, page, pageSize, strings.TrimSpace(status))
}

func (s *BenefitGrantService) prepare(
	ctx context.Context,
	normalized *normalizedBenefitGrant,
	now time.Time,
) (*preparedBenefitGrant, error) {
	if normalized == nil || s == nil || s.repo == nil {
		return nil, ErrBenefitGrantUnavailable
	}

	if normalized.BenefitType == BenefitGrantTypeSubscription {
		if s.groupRepo == nil || s.subscriptionService == nil {
			return nil, ErrBenefitGrantUnavailable
		}
		group, err := s.groupRepo.GetByID(ctx, normalized.GroupID)
		if err != nil {
			return nil, fmt.Errorf("group not found: %w", err)
		}
		if !group.IsSubscriptionType() {
			return nil, ErrGroupNotSubscriptionType
		}
		if !group.IsActive() {
			return nil, invalidBenefitGrantInput("group_id", "subscription group must be active")
		}
		normalized.groupName = group.Name
	}

	users, err := s.repo.ListAudience(
		ctx,
		normalized.AudienceType,
		normalized.windowStart,
		normalized.windowEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("list benefit grant audience: %w", err)
	}
	if len(users) > benefitGrantMaxAudienceUsers {
		return nil, invalidBenefitGrantInput("audience", "target audience is too large")
	}

	states := make(map[int64]BenefitGrantSubscriptionState)
	if normalized.BenefitType == BenefitGrantTypeSubscription && len(users) > 0 {
		userIDs := make([]int64, 0, len(users))
		for i := range users {
			userIDs = append(userIDs, users[i].ID)
		}
		states, err = s.repo.ListSubscriptionStates(ctx, userIDs, normalized.GroupID)
		if err != nil {
			return nil, fmt.Errorf("load audience subscriptions: %w", err)
		}
	}

	preview := &BenefitGrantPreview{
		OperationKey:   normalized.OperationKey,
		AudienceType:   normalized.AudienceType,
		AudienceDate:   normalized.AudienceDate,
		AudienceDays:   normalized.AudienceDays,
		Timezone:       normalized.Timezone,
		WindowStart:    normalized.windowStart,
		WindowEnd:      normalized.windowEnd,
		BenefitType:    normalized.BenefitType,
		ConflictPolicy: normalized.ConflictPolicy,
		GroupID:        normalized.GroupID,
		ValidityDays:   normalized.ValidityDays,
		BalanceAmount:  normalized.BalanceAmount,
		MatchedCount:   len(users),
	}
	recipients := make([]BenefitGrantRecipient, 0, len(users))
	for i := range users {
		user := users[i]
		recipient := BenefitGrantRecipient{
			UserID:           user.ID,
			EmailSnapshot:    user.Email,
			UsernameSnapshot: user.Username,
			Eligibility:      BenefitGrantEligibilityEligible,
			Status:           BenefitGrantRecipientPending,
		}

		if normalized.BenefitType == BenefitGrantTypeBalance {
			recipient.PlannedAction = BenefitGrantActionBalanceAdd
			preview.EligibleCount++
			recipients = append(recipients, recipient)
			continue
		}

		state, exists := states[user.ID]
		eligibility, action := classifyBenefitSubscription(
			state,
			exists,
			normalized.marker,
			normalized.ConflictPolicy,
			now,
		)
		recipient.Eligibility = eligibility
		recipient.PlannedAction = action
		switch eligibility {
		case BenefitGrantEligibilityEligible:
			preview.EligibleCount++
		case BenefitGrantEligibilityAlreadyGranted:
			preview.AlreadyGrantedCount++
			recipient.Status = BenefitGrantRecipientSkipped
			recipient.ResultType = BenefitGrantEligibilityAlreadyGranted
		case BenefitGrantEligibilityConflict:
			preview.ConflictCount++
			recipient.Status = BenefitGrantRecipientSkipped
			recipient.ResultType = BenefitGrantEligibilityConflict
		}
		recipients = append(recipients, recipient)
	}
	preview.SnapshotToken = benefitGrantSnapshotToken(normalized.requestHash, recipients)

	return &preparedBenefitGrant{
		normalized: normalized,
		preview:    preview,
		recipients: recipients,
	}, nil
}

func (s *BenefitGrantService) processCampaign(
	ctx context.Context,
	campaign *BenefitGrantCampaign,
	includeFailed bool,
	preview *BenefitGrantPreview,
) (*BenefitGrantResult, error) {
	if campaign == nil {
		return nil, ErrBenefitGrantNotFound
	}
	staleBefore := time.Now().Add(-benefitGrantRetryStaleAfter)
	recipients, err := s.repo.ListActionableRecipients(
		ctx,
		campaign.ID,
		includeFailed,
		staleBefore,
		benefitGrantMaxAudienceUsers,
	)
	if err != nil {
		return nil, err
	}

	for i := range recipients {
		if err := ctx.Err(); err != nil {
			_, _ = s.repo.RefreshCampaign(context.Background(), campaign.ID)
			return nil, err
		}
		recipient := recipients[i]
		if campaign.BenefitType == BenefitGrantTypeBalance {
			if err := s.processBalanceRecipient(ctx, campaign, &recipient, includeFailed, staleBefore); err != nil {
				_, _ = s.repo.RefreshCampaign(context.Background(), campaign.ID)
				return nil, err
			}
			continue
		}
		if err := s.processSubscriptionRecipient(ctx, campaign, &recipient, includeFailed, staleBefore); err != nil {
			_, _ = s.repo.RefreshCampaign(context.Background(), campaign.ID)
			return nil, err
		}
	}

	refreshed, err := s.repo.RefreshCampaign(ctx, campaign.ID)
	if err != nil {
		return nil, err
	}
	return s.resultForCampaign(ctx, refreshed, preview), nil
}

func (s *BenefitGrantService) processBalanceRecipient(
	ctx context.Context,
	campaign *BenefitGrantCampaign,
	recipient *BenefitGrantRecipient,
	includeFailed bool,
	staleBefore time.Time,
) error {
	if campaign.BalanceAmount == nil {
		return invalidBenefitGrantInput("balance_amount", "balance amount is missing from campaign")
	}
	applied, err := s.repo.ApplyBalanceRecipient(
		ctx,
		campaign.ID,
		recipient.UserID,
		*campaign.BalanceAmount,
		includeFailed,
		staleBefore,
	)
	if err != nil {
		return err
	}
	if !applied.Granted || s.billingCacheService == nil {
		return nil
	}
	cacheCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.billingCacheService.InvalidateUserBalance(cacheCtx, recipient.UserID)
	return nil
}

func (s *BenefitGrantService) processSubscriptionRecipient(
	ctx context.Context,
	campaign *BenefitGrantCampaign,
	recipient *BenefitGrantRecipient,
	includeFailed bool,
	staleBefore time.Time,
) error {
	if campaign.GroupID == nil || campaign.ValidityDays == nil || s.subscriptionService == nil {
		return invalidBenefitGrantInput("subscription", "subscription campaign parameters are incomplete")
	}
	claimed, err := s.repo.ClaimRecipient(
		ctx,
		campaign.ID,
		recipient.UserID,
		includeFailed,
		staleBefore,
	)
	if err != nil || !claimed {
		return err
	}

	states, err := s.repo.ListSubscriptionStates(ctx, []int64{recipient.UserID}, *campaign.GroupID)
	if err != nil {
		return s.failSubscriptionRecipient(ctx, campaign.ID, recipient.UserID, err)
	}
	state, exists := states[recipient.UserID]
	if exists && strings.Contains(state.Notes, campaign.Marker) {
		subscriptionID := state.ID
		return s.repo.MarkRecipientGranted(
			ctx,
			campaign.ID,
			recipient.UserID,
			recipient.PlannedAction,
			&subscriptionID,
		)
	}

	eligibility, action := classifyBenefitSubscription(
		state,
		exists,
		campaign.Marker,
		campaign.ConflictPolicy,
		time.Now(),
	)
	if eligibility != BenefitGrantEligibilityEligible {
		return s.failSubscriptionRecipient(
			ctx,
			campaign.ID,
			recipient.UserID,
			fmt.Errorf("subscription changed after audience snapshot"),
		)
	}

	assignInput := &AssignSubscriptionInput{
		UserID:       recipient.UserID,
		GroupID:      *campaign.GroupID,
		ValidityDays: *campaign.ValidityDays,
		AssignedBy:   campaign.CreatedBy,
		Notes:        benefitGrantNotes(campaign.Marker, campaign.Notes),
	}
	var subscription *UserSubscription
	if action == BenefitGrantActionExtend {
		subscription, _, err = s.subscriptionService.AssignOrExtendSubscription(ctx, assignInput)
	} else {
		subscription, err = s.subscriptionService.AssignSubscription(ctx, assignInput)
	}
	if err != nil {
		return s.failSubscriptionRecipient(ctx, campaign.ID, recipient.UserID, err)
	}
	subscriptionID := subscription.ID
	return s.repo.MarkRecipientGranted(
		ctx,
		campaign.ID,
		recipient.UserID,
		action,
		&subscriptionID,
	)
}

func (s *BenefitGrantService) failSubscriptionRecipient(
	ctx context.Context,
	campaignID, userID int64,
	cause error,
) error {
	message := "subscription grant failed"
	if cause != nil {
		message = cause.Error()
	}
	if err := s.repo.MarkRecipientFailed(ctx, campaignID, userID, message); err != nil {
		return err
	}
	return nil
}

func (s *BenefitGrantService) resultForCampaign(
	ctx context.Context,
	campaign *BenefitGrantCampaign,
	preview *BenefitGrantPreview,
) *BenefitGrantResult {
	result := &BenefitGrantResult{
		Campaign:      campaign,
		Preview:       preview,
		GrantedCount:  campaign.GrantedCount,
		CreatedCount:  campaign.CreatedCount,
		RenewedCount:  campaign.RenewedCount,
		ExtendedCount: campaign.ExtendedCount,
		FailedCount:   campaign.FailedCount,
		SkippedCount:  campaign.SkippedCount,
		Errors:        make([]string, 0),
	}
	if campaign.FailedCount <= 0 {
		return result
	}
	failed, _, err := s.repo.ListRecipients(ctx, campaign.ID, 1, 20, BenefitGrantRecipientFailed)
	if err != nil {
		return result
	}
	for i := range failed {
		result.Errors = append(
			result.Errors,
			fmt.Sprintf("user %d: %s", failed[i].UserID, failed[i].Error),
		)
	}
	return result
}

func campaignFromPrepared(prepared *preparedBenefitGrant, assignedBy int64) *BenefitGrantCampaign {
	normalized := prepared.normalized
	preview := prepared.preview
	campaign := &BenefitGrantCampaign{
		OperationKey:        normalized.OperationKey,
		RequestHash:         normalized.requestHash,
		DeliveryMode:        BenefitGrantDeliverySnapshot,
		AudienceType:        normalized.AudienceType,
		AudienceDate:        normalized.AudienceDate,
		AudienceDays:        normalized.AudienceDays,
		Timezone:            normalized.Timezone,
		WindowStart:         normalized.windowStart,
		WindowEnd:           normalized.windowEnd,
		BenefitType:         normalized.BenefitType,
		ConflictPolicy:      normalized.ConflictPolicy,
		GroupName:           normalized.groupName,
		Notes:               normalized.Notes,
		Marker:              normalized.marker,
		Status:              BenefitGrantStatusRunning,
		MatchedCount:        preview.MatchedCount,
		EligibleCount:       preview.EligibleCount,
		AlreadyGrantedCount: preview.AlreadyGrantedCount,
		ConflictCount:       preview.ConflictCount,
		SkippedCount:        preview.AlreadyGrantedCount + preview.ConflictCount,
		CreatedBy:           assignedBy,
	}
	if normalized.BenefitType == BenefitGrantTypeSubscription {
		groupID := normalized.GroupID
		validityDays := normalized.ValidityDays
		campaign.GroupID = &groupID
		campaign.ValidityDays = &validityDays
	} else {
		amount := normalized.BalanceAmount
		campaign.BalanceAmount = &amount
	}
	return campaign
}

func campaignFromAutomatic(
	normalized *normalizedBenefitGrant,
	assignedBy int64,
	now time.Time,
) *BenefitGrantCampaign {
	status := BenefitGrantStatusRunning
	if now.Before(normalized.windowStart) {
		status = BenefitGrantStatusScheduled
	}
	campaign := &BenefitGrantCampaign{
		OperationKey:   normalized.OperationKey,
		RequestHash:    normalized.requestHash,
		DeliveryMode:   BenefitGrantDeliveryActivityWindow,
		AudienceType:   BenefitGrantAudienceAuthenticated,
		AudienceDate:   normalized.AudienceDate,
		AudienceDays:   normalized.AudienceDays,
		Timezone:       normalized.Timezone,
		WindowStart:    normalized.windowStart,
		WindowEnd:      normalized.windowEnd,
		BenefitType:    normalized.BenefitType,
		ConflictPolicy: normalized.ConflictPolicy,
		GroupName:      normalized.groupName,
		Notes:          normalized.Notes,
		Marker:         normalized.marker,
		Status:         status,
		CreatedBy:      assignedBy,
	}
	if normalized.BenefitType == BenefitGrantTypeSubscription {
		groupID := normalized.GroupID
		validityDays := normalized.ValidityDays
		campaign.GroupID = &groupID
		campaign.ValidityDays = &validityDays
	} else {
		amount := normalized.BalanceAmount
		campaign.BalanceAmount = &amount
	}
	return campaign
}

func normalizeBenefitGrantInput(input *BenefitGrantInput, now time.Time) (*normalizedBenefitGrant, error) {
	return normalizeBenefitGrantInputWithDatePolicy(input, now, true)
}

func normalizeBenefitGrantInputForReplay(input *BenefitGrantInput, now time.Time) (*normalizedBenefitGrant, error) {
	return normalizeBenefitGrantInputWithDatePolicy(input, now, false)
}

func normalizeBenefitGrantInputWithDatePolicy(
	input *BenefitGrantInput,
	now time.Time,
	requireCurrentDate bool,
) (*normalizedBenefitGrant, error) {
	if input == nil {
		return nil, invalidBenefitGrantInput("request", "request body is required")
	}

	operationKey := strings.TrimSpace(input.OperationKey)
	if err := validateBenefitGrantOperationKey(operationKey); err != nil {
		return nil, err
	}

	audienceType := strings.TrimSpace(input.AudienceType)
	switch audienceType {
	case BenefitGrantAudienceTodayActive,
		BenefitGrantAudienceRecentActive,
		BenefitGrantAudienceRecentRegistered:
	default:
		return nil, invalidBenefitGrantInput("audience_type", "unsupported audience type")
	}

	timezoneName := strings.TrimSpace(input.Timezone)
	if timezoneName == "" {
		timezoneName = timezone.Name()
	}
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return nil, invalidBenefitGrantInput("timezone", "timezone must be a valid IANA name")
	}

	localNow := now.In(location)
	audienceDate := strings.TrimSpace(input.AudienceDate)
	if audienceDate == "" {
		audienceDate = localNow.Format("2006-01-02")
	}
	day, err := time.ParseInLocation("2006-01-02", audienceDate, location)
	if err != nil {
		return nil, invalidBenefitGrantInput("audience_date", "audience_date must use YYYY-MM-DD")
	}
	if requireCurrentDate && audienceDate != localNow.Format("2006-01-02") {
		return nil, invalidBenefitGrantInput("audience_date", "audience only supports the current local date")
	}

	audienceDays := input.AudienceDays
	if audienceType == BenefitGrantAudienceTodayActive {
		audienceDays = 1
	}
	if audienceDays <= 0 || audienceDays > benefitGrantMaxAudienceDays {
		return nil, invalidBenefitGrantInput("audience_days", "audience_days is out of range")
	}
	windowStart := day.AddDate(0, 0, -(audienceDays - 1))
	windowEnd := day.AddDate(0, 0, 1)

	configuration, announcement, err := normalizeBenefitGrantConfiguration(input)
	if err != nil {
		return nil, err
	}
	configuration.AudienceType = audienceType
	configuration.AudienceDate = audienceDate
	configuration.AudienceDays = audienceDays
	configuration.Timezone = timezoneName

	normalized := &normalizedBenefitGrant{
		BenefitGrantInput: configuration,
		deliveryMode:      BenefitGrantDeliverySnapshot,
		windowStart:       windowStart,
		windowEnd:         windowEnd,
		marker:            "[benefit-grant:" + operationKey + "]",
		announcement:      announcement,
	}
	normalized.grantNotes = benefitGrantNotes(normalized.marker, normalized.Notes)
	normalized.requestHash = benefitGrantRequestHash(normalized)
	return normalized, nil
}

func normalizeAutomaticBenefitGrantInput(
	input *CreateAutomaticBenefitGrantInput,
	now time.Time,
) (*normalizedBenefitGrant, error) {
	return normalizeAutomaticBenefitGrantInputWithEndPolicy(input, now, true)
}

func normalizeAutomaticBenefitGrantInputForReplay(
	input *CreateAutomaticBenefitGrantInput,
	now time.Time,
) (*normalizedBenefitGrant, error) {
	return normalizeAutomaticBenefitGrantInputWithEndPolicy(input, now, false)
}

func normalizeAutomaticBenefitGrantInputWithEndPolicy(
	input *CreateAutomaticBenefitGrantInput,
	now time.Time,
	requireFutureEnd bool,
) (*normalizedBenefitGrant, error) {
	if input == nil {
		return nil, invalidBenefitGrantInput("request", "request body is required")
	}
	configuration, announcement, err := normalizeBenefitGrantConfiguration(&BenefitGrantInput{
		OperationKey:        input.OperationKey,
		BenefitType:         input.BenefitType,
		ConflictPolicy:      input.ConflictPolicy,
		GroupID:             input.GroupID,
		ValidityDays:        input.ValidityDays,
		BalanceAmount:       input.BalanceAmount,
		Notes:               input.Notes,
		AnnouncementEnabled: input.AnnouncementEnabled,
		AnnouncementTitle:   input.AnnouncementTitle,
		AnnouncementContent: input.AnnouncementContent,
		AnnouncementNotify:  input.AnnouncementNotify,
	})
	if err != nil {
		return nil, err
	}

	timezoneName := strings.TrimSpace(input.Timezone)
	if timezoneName == "" {
		timezoneName = timezone.Name()
	}
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return nil, invalidBenefitGrantInput("timezone", "timezone must be a valid IANA name")
	}

	windowStart := input.WindowStart
	windowEnd := input.WindowEnd
	if windowStart.IsZero() || windowEnd.IsZero() || !windowStart.Before(windowEnd) {
		return nil, invalidBenefitGrantInput("window", "window_start must be before window_end")
	}
	if requireFutureEnd && !windowEnd.After(now) {
		return nil, invalidBenefitGrantInput("window_end", "window_end must be in the future")
	}
	if windowEnd.Sub(windowStart) > benefitGrantMaxActivityWindow {
		return nil, invalidBenefitGrantInput("window", "activity window cannot exceed 365 days")
	}

	audienceDays := int(math.Ceil(windowEnd.Sub(windowStart).Hours() / 24))
	if audienceDays < 1 {
		audienceDays = 1
	}
	if audienceDays > benefitGrantMaxAudienceDays {
		audienceDays = benefitGrantMaxAudienceDays
	}
	configuration.AudienceType = BenefitGrantAudienceAuthenticated
	configuration.AudienceDate = windowStart.In(location).Format("2006-01-02")
	configuration.AudienceDays = audienceDays
	configuration.Timezone = timezoneName
	if announcement != nil {
		startsAt := windowStart
		announcement.StartsAt = &startsAt
	}

	normalized := &normalizedBenefitGrant{
		BenefitGrantInput: configuration,
		deliveryMode:      BenefitGrantDeliveryActivityWindow,
		windowStart:       windowStart,
		windowEnd:         windowEnd,
		marker:            "[benefit-grant:" + configuration.OperationKey + "]",
		announcement:      announcement,
	}
	normalized.grantNotes = benefitGrantNotes(normalized.marker, normalized.Notes)
	normalized.requestHash = benefitGrantRequestHash(normalized)
	return normalized, nil
}

func normalizeBenefitGrantConfiguration(
	input *BenefitGrantInput,
) (BenefitGrantInput, *BenefitGrantAnnouncement, error) {
	if input == nil {
		return BenefitGrantInput{}, nil, invalidBenefitGrantInput(
			"request",
			"request body is required",
		)
	}

	operationKey := strings.TrimSpace(input.OperationKey)
	if err := validateBenefitGrantOperationKey(operationKey); err != nil {
		return BenefitGrantInput{}, nil, err
	}
	notes := strings.TrimSpace(input.Notes)
	if len([]rune(notes)) > benefitGrantNotesMaxRunes {
		return BenefitGrantInput{}, nil, invalidBenefitGrantInput("notes", "notes is too long")
	}

	normalized := BenefitGrantInput{
		OperationKey: operationKey,
		Notes:        notes,
	}
	benefitType := strings.TrimSpace(input.BenefitType)
	conflictPolicy := strings.TrimSpace(input.ConflictPolicy)
	switch benefitType {
	case BenefitGrantTypeSubscription:
		if input.GroupID <= 0 {
			return BenefitGrantInput{}, nil, invalidBenefitGrantInput(
				"group_id",
				"group_id must be greater than zero",
			)
		}
		if input.ValidityDays <= 0 || input.ValidityDays > MaxValidityDays {
			return BenefitGrantInput{}, nil, invalidBenefitGrantInput(
				"validity_days",
				"validity_days is out of range",
			)
		}
		if conflictPolicy == "" {
			conflictPolicy = BenefitGrantConflictSkipActive
		}
		if conflictPolicy != BenefitGrantConflictSkipActive &&
			conflictPolicy != BenefitGrantConflictExtendActive {
			return BenefitGrantInput{}, nil, invalidBenefitGrantInput(
				"conflict_policy",
				"unsupported subscription conflict policy",
			)
		}
		normalized.BenefitType = BenefitGrantTypeSubscription
		normalized.ConflictPolicy = conflictPolicy
		normalized.GroupID = input.GroupID
		normalized.ValidityDays = input.ValidityDays
	case BenefitGrantTypeBalance:
		if math.IsNaN(input.BalanceAmount) ||
			math.IsInf(input.BalanceAmount, 0) ||
			input.BalanceAmount <= 0 ||
			input.BalanceAmount > benefitGrantMaxBalanceAmount {
			return BenefitGrantInput{}, nil, invalidBenefitGrantInput(
				"balance_amount",
				"balance_amount is out of range",
			)
		}
		balanceAmount := math.Round(input.BalanceAmount*benefitGrantBalanceScale) /
			benefitGrantBalanceScale
		if balanceAmount <= 0 {
			return BenefitGrantInput{}, nil, invalidBenefitGrantInput(
				"balance_amount",
				"balance_amount is below the supported precision",
			)
		}
		normalized.BenefitType = BenefitGrantTypeBalance
		normalized.ConflictPolicy = BenefitGrantConflictNone
		normalized.BalanceAmount = balanceAmount
	default:
		return BenefitGrantInput{}, nil, invalidBenefitGrantInput(
			"benefit_type",
			"unsupported benefit type",
		)
	}

	var announcement *BenefitGrantAnnouncement
	if input.AnnouncementEnabled {
		title := strings.TrimSpace(input.AnnouncementTitle)
		content := strings.TrimSpace(input.AnnouncementContent)
		notifyMode := strings.TrimSpace(input.AnnouncementNotify)
		if title == "" || len([]rune(title)) > 200 {
			return BenefitGrantInput{}, nil, invalidBenefitGrantInput(
				"announcement_title",
				"announcement title is invalid",
			)
		}
		if content == "" || len([]rune(content)) > benefitGrantAnnouncementContentMaxRunes {
			return BenefitGrantInput{}, nil, invalidBenefitGrantInput(
				"announcement_content",
				"announcement content is invalid",
			)
		}
		if notifyMode == "" {
			notifyMode = AnnouncementNotifyModeSilent
		}
		if notifyMode != AnnouncementNotifyModeSilent &&
			notifyMode != AnnouncementNotifyModePopup {
			return BenefitGrantInput{}, nil, invalidBenefitGrantInput(
				"announcement_notify_mode",
				"unsupported announcement notify mode",
			)
		}
		normalized.AnnouncementEnabled = true
		normalized.AnnouncementTitle = title
		normalized.AnnouncementContent = content
		normalized.AnnouncementNotify = notifyMode
		announcement = &BenefitGrantAnnouncement{
			Title:      title,
			Content:    content,
			NotifyMode: notifyMode,
		}
	}

	return normalized, announcement, nil
}

func validateBenefitGrantOperationKey(operationKey string) error {
	if len(operationKey) < 8 || len(operationKey) > 128 {
		return invalidBenefitGrantInput("operation_key", "operation_key must be between 8 and 128 characters")
	}
	for _, char := range operationKey {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			strings.ContainsRune("-._:", char) {
			continue
		}
		return invalidBenefitGrantInput("operation_key", "operation_key contains unsupported characters")
	}
	return nil
}

func classifyBenefitSubscription(
	state BenefitGrantSubscriptionState,
	exists bool,
	marker string,
	conflictPolicy string,
	now time.Time,
) (string, string) {
	if exists && strings.Contains(state.Notes, marker) {
		return BenefitGrantEligibilityAlreadyGranted, BenefitGrantActionNone
	}
	if !exists {
		return BenefitGrantEligibilityEligible, BenefitGrantActionCreate
	}
	if state.Status == SubscriptionStatusSuspended {
		return BenefitGrantEligibilityConflict, BenefitGrantActionNone
	}
	if state.Status == SubscriptionStatusExpired || !state.ExpiresAt.After(now) {
		return BenefitGrantEligibilityEligible, BenefitGrantActionRenew
	}
	if state.Status == SubscriptionStatusActive &&
		conflictPolicy == BenefitGrantConflictExtendActive {
		return BenefitGrantEligibilityEligible, BenefitGrantActionExtend
	}
	return BenefitGrantEligibilityConflict, BenefitGrantActionNone
}

func benefitGrantNotes(marker, notes string) string {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return marker
	}
	return marker + "\n" + notes
}

func benefitGrantRequestHash(normalized *normalizedBenefitGrant) string {
	announcementEnabled := "false"
	announcementTitle := ""
	announcementContent := ""
	announcementNotify := ""
	if normalized.announcement != nil {
		announcementEnabled = "true"
		announcementTitle = normalized.announcement.Title
		announcementContent = normalized.announcement.Content
		announcementNotify = normalized.announcement.NotifyMode
	}
	return hashBenefitGrantParts(
		normalized.deliveryMode,
		normalized.AudienceType,
		normalized.AudienceDate,
		strconv.Itoa(normalized.AudienceDays),
		normalized.Timezone,
		normalized.windowStart.UTC().Format(time.RFC3339Nano),
		normalized.windowEnd.UTC().Format(time.RFC3339Nano),
		normalized.BenefitType,
		normalized.ConflictPolicy,
		strconv.FormatInt(normalized.GroupID, 10),
		strconv.Itoa(normalized.ValidityDays),
		strconv.FormatFloat(normalized.BalanceAmount, 'f', 8, 64),
		normalized.Notes,
		announcementEnabled,
		announcementTitle,
		announcementContent,
		announcementNotify,
	)
}

func benefitGrantRequestMatches(
	campaign *BenefitGrantCampaign,
	normalized *normalizedBenefitGrant,
) bool {
	if campaign == nil || normalized == nil {
		return false
	}
	if campaign.RequestHash == normalized.requestHash {
		return true
	}
	if normalized.deliveryMode != BenefitGrantDeliverySnapshot ||
		normalized.announcement != nil {
		return false
	}
	return campaign.RequestHash == benefitGrantLegacyRequestHash(normalized)
}

func benefitGrantLegacyRequestHash(normalized *normalizedBenefitGrant) string {
	return hashBenefitGrantParts(
		normalized.AudienceType,
		normalized.AudienceDate,
		strconv.Itoa(normalized.AudienceDays),
		normalized.Timezone,
		normalized.windowStart.UTC().Format(time.RFC3339Nano),
		normalized.windowEnd.UTC().Format(time.RFC3339Nano),
		normalized.BenefitType,
		normalized.ConflictPolicy,
		strconv.FormatInt(normalized.GroupID, 10),
		strconv.Itoa(normalized.ValidityDays),
		strconv.FormatFloat(normalized.BalanceAmount, 'f', 8, 64),
		normalized.Notes,
	)
}

func benefitGrantSnapshotToken(requestHash string, recipients []BenefitGrantRecipient) string {
	parts := make([]string, 0, 1+len(recipients)*3)
	parts = append(parts, requestHash)
	for i := range recipients {
		parts = append(
			parts,
			strconv.FormatInt(recipients[i].UserID, 10),
			recipients[i].Eligibility,
			recipients[i].PlannedAction,
		)
	}
	return hashBenefitGrantParts(parts...)
}

func hashBenefitGrantParts(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write([]byte(part))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func invalidBenefitGrantInput(field, message string) error {
	return infraerrors.BadRequest(
		"BENEFIT_GRANT_INVALID_INPUT",
		message,
	).WithMetadata(map[string]string{"field": field})
}
