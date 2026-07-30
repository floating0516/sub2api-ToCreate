package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	dbusersubscription "github.com/Wei-Shaw/sub2api/ent/usersubscription"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const (
	BenefitGrantAudienceTodayActive = "today_active"
	BenefitGrantTypeSubscription    = "subscription"

	benefitGrantNotesMaxRunes = 1800
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
)

// BenefitGrantInput describes a target audience and the benefit to grant.
// The flat shape is intentional: new audience and benefit variants can be
// added without changing the preview/execute endpoint contract.
type BenefitGrantInput struct {
	AudienceType string
	AudienceDate string
	Timezone     string
	BenefitType  string
	GroupID      int64
	ValidityDays int
	Notes        string
}

type ExecuteBenefitGrantInput struct {
	BenefitGrantInput
	ExpectedMatchedCount  int
	ExpectedEligibleCount int
	AssignedBy            int64
}

type BenefitGrantPreview struct {
	AudienceType        string    `json:"audience_type"`
	AudienceDate        string    `json:"audience_date"`
	Timezone            string    `json:"timezone"`
	WindowStart         time.Time `json:"window_start"`
	WindowEnd           time.Time `json:"window_end"`
	BenefitType         string    `json:"benefit_type"`
	GroupID             int64     `json:"group_id"`
	ValidityDays        int       `json:"validity_days"`
	MatchedCount        int       `json:"matched_count"`
	EligibleCount       int       `json:"eligible_count"`
	AlreadyGrantedCount int       `json:"already_granted_count"`
	ConflictCount       int       `json:"conflict_count"`
}

type BenefitGrantResult struct {
	Preview      *BenefitGrantPreview `json:"preview"`
	GrantedCount int                  `json:"granted_count"`
	CreatedCount int                  `json:"created_count"`
	RenewedCount int                  `json:"renewed_count"`
	FailedCount  int                  `json:"failed_count"`
	SkippedCount int                  `json:"skipped_count"`
	Errors       []string             `json:"errors"`
}

type normalizedBenefitGrant struct {
	BenefitGrantInput
	windowStart time.Time
	windowEnd   time.Time
	marker      string
	grantNotes  string
}

type preparedBenefitGrant struct {
	preview         *BenefitGrantPreview
	eligibleUserIDs []int64
	grantNotes      string
}

type benefitSubscriptionState struct {
	status    string
	expiresAt time.Time
	notes     string
}

type benefitSubscriptionEligibility int

const (
	benefitSubscriptionEligible benefitSubscriptionEligibility = iota
	benefitSubscriptionAlreadyGranted
	benefitSubscriptionConflict
)

func (s *SubscriptionService) PreviewBenefitGrant(ctx context.Context, input *BenefitGrantInput) (*BenefitGrantPreview, error) {
	prepared, err := s.prepareBenefitGrant(ctx, input, time.Now())
	if err != nil {
		return nil, err
	}
	return prepared.preview, nil
}

func (s *SubscriptionService) ExecuteBenefitGrant(ctx context.Context, input *ExecuteBenefitGrantInput) (*BenefitGrantResult, error) {
	if input == nil {
		return nil, invalidBenefitGrantInput("request", "request body is required")
	}

	prepared, err := s.prepareBenefitGrant(ctx, &input.BenefitGrantInput, time.Now())
	if err != nil {
		return nil, err
	}

	preview := prepared.preview
	if preview.MatchedCount != input.ExpectedMatchedCount ||
		preview.EligibleCount != input.ExpectedEligibleCount {
		return nil, ErrBenefitGrantAudienceChanged.WithMetadata(map[string]string{
			"expected_matched_count":  strconv.Itoa(input.ExpectedMatchedCount),
			"actual_matched_count":    strconv.Itoa(preview.MatchedCount),
			"expected_eligible_count": strconv.Itoa(input.ExpectedEligibleCount),
			"actual_eligible_count":   strconv.Itoa(preview.EligibleCount),
		})
	}

	result := &BenefitGrantResult{
		Preview:      preview,
		SkippedCount: preview.AlreadyGrantedCount + preview.ConflictCount,
		Errors:       make([]string, 0),
	}
	if len(prepared.eligibleUserIDs) == 0 {
		return result, nil
	}

	bulk, err := s.BulkAssignSubscription(ctx, &BulkAssignSubscriptionInput{
		UserIDs:      prepared.eligibleUserIDs,
		GroupID:      preview.GroupID,
		ValidityDays: preview.ValidityDays,
		AssignedBy:   input.AssignedBy,
		Notes:        prepared.grantNotes,
	})
	if err != nil {
		return nil, err
	}

	result.GrantedCount = bulk.SuccessCount
	result.CreatedCount = bulk.CreatedCount
	result.RenewedCount = bulk.ReusedCount
	result.FailedCount = bulk.FailedCount
	result.Errors = bulk.Errors
	return result, nil
}

func (s *SubscriptionService) prepareBenefitGrant(
	ctx context.Context,
	input *BenefitGrantInput,
	now time.Time,
) (*preparedBenefitGrant, error) {
	normalized, err := normalizeBenefitGrantInput(input, now)
	if err != nil {
		return nil, err
	}
	if s == nil || s.entClient == nil || s.groupRepo == nil {
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

	userIDs, err := s.entClient.User.Query().
		Where(
			dbuser.StatusEQ(StatusActive),
			dbuser.RoleEQ(RoleUser),
			dbuser.LastActiveAtGTE(normalized.windowStart),
			dbuser.LastActiveAtLT(normalized.windowEnd),
		).
		Order(dbent.Asc(dbuser.FieldID)).
		IDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list benefit grant audience: %w", err)
	}

	existingByUserID := make(map[int64]benefitSubscriptionState, len(userIDs))
	if len(userIDs) > 0 {
		subscriptions, queryErr := s.entClient.UserSubscription.Query().
			Where(
				dbusersubscription.UserIDIn(userIDs...),
				dbusersubscription.GroupIDEQ(normalized.GroupID),
			).
			All(ctx)
		if queryErr != nil {
			return nil, fmt.Errorf("load audience subscriptions: %w", queryErr)
		}
		for _, subscription := range subscriptions {
			notes := ""
			if subscription.Notes != nil {
				notes = *subscription.Notes
			}
			existingByUserID[subscription.UserID] = benefitSubscriptionState{
				status:    subscription.Status,
				expiresAt: subscription.ExpiresAt,
				notes:     notes,
			}
		}
	}

	preview := &BenefitGrantPreview{
		AudienceType: normalized.AudienceType,
		AudienceDate: normalized.AudienceDate,
		Timezone:     normalized.Timezone,
		WindowStart:  normalized.windowStart,
		WindowEnd:    normalized.windowEnd,
		BenefitType:  normalized.BenefitType,
		GroupID:      normalized.GroupID,
		ValidityDays: normalized.ValidityDays,
		MatchedCount: len(userIDs),
	}
	eligibleUserIDs := make([]int64, 0, len(userIDs))
	for _, userID := range userIDs {
		state, exists := existingByUserID[userID]
		if !exists {
			preview.EligibleCount++
			eligibleUserIDs = append(eligibleUserIDs, userID)
			continue
		}

		switch classifyBenefitSubscription(state, normalized.marker, now) {
		case benefitSubscriptionEligible:
			preview.EligibleCount++
			eligibleUserIDs = append(eligibleUserIDs, userID)
		case benefitSubscriptionAlreadyGranted:
			preview.AlreadyGrantedCount++
		case benefitSubscriptionConflict:
			preview.ConflictCount++
		}
	}

	return &preparedBenefitGrant{
		preview:         preview,
		eligibleUserIDs: eligibleUserIDs,
		grantNotes:      normalized.grantNotes,
	}, nil
}

func normalizeBenefitGrantInput(input *BenefitGrantInput, now time.Time) (*normalizedBenefitGrant, error) {
	if input == nil {
		return nil, invalidBenefitGrantInput("request", "request body is required")
	}

	audienceType := strings.TrimSpace(input.AudienceType)
	if audienceType != BenefitGrantAudienceTodayActive {
		return nil, invalidBenefitGrantInput("audience_type", "unsupported audience type")
	}
	benefitType := strings.TrimSpace(input.BenefitType)
	if benefitType != BenefitGrantTypeSubscription {
		return nil, invalidBenefitGrantInput("benefit_type", "unsupported benefit type")
	}
	if input.GroupID <= 0 {
		return nil, invalidBenefitGrantInput("group_id", "group_id must be greater than zero")
	}
	if input.ValidityDays <= 0 || input.ValidityDays > MaxValidityDays {
		return nil, invalidBenefitGrantInput("validity_days", "validity_days is out of range")
	}

	notes := strings.TrimSpace(input.Notes)
	if len([]rune(notes)) > benefitGrantNotesMaxRunes {
		return nil, invalidBenefitGrantInput("notes", "notes is too long")
	}

	timezoneName := strings.TrimSpace(input.Timezone)
	if timezoneName == "" {
		timezoneName = timezone.Name()
	}
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return nil, invalidBenefitGrantInput("timezone", "timezone must be a valid IANA name")
	}

	now = now.In(location)
	audienceDate := strings.TrimSpace(input.AudienceDate)
	if audienceDate == "" {
		audienceDate = now.Format("2006-01-02")
	}
	day, err := time.ParseInLocation("2006-01-02", audienceDate, location)
	if err != nil {
		return nil, invalidBenefitGrantInput("audience_date", "audience_date must use YYYY-MM-DD")
	}
	if audienceDate != now.Format("2006-01-02") {
		return nil, invalidBenefitGrantInput("audience_date", "today_active only supports the current local date")
	}

	marker := fmt.Sprintf(
		"[benefit-grant:%s:%s:%s:%s:%d:%d]",
		audienceType,
		audienceDate,
		timezoneName,
		benefitType,
		input.GroupID,
		input.ValidityDays,
	)
	grantNotes := marker
	if notes != "" {
		grantNotes += "\n" + notes
	}

	return &normalizedBenefitGrant{
		BenefitGrantInput: BenefitGrantInput{
			AudienceType: audienceType,
			AudienceDate: audienceDate,
			Timezone:     timezoneName,
			BenefitType:  benefitType,
			GroupID:      input.GroupID,
			ValidityDays: input.ValidityDays,
			Notes:        notes,
		},
		windowStart: day,
		windowEnd:   day.AddDate(0, 0, 1),
		marker:      marker,
		grantNotes:  grantNotes,
	}, nil
}

func classifyBenefitSubscription(
	state benefitSubscriptionState,
	marker string,
	now time.Time,
) benefitSubscriptionEligibility {
	if strings.Contains(state.notes, marker) {
		return benefitSubscriptionAlreadyGranted
	}
	if state.status == SubscriptionStatusExpired ||
		(state.status != SubscriptionStatusSuspended && !state.expiresAt.After(now)) {
		return benefitSubscriptionEligible
	}
	return benefitSubscriptionConflict
}

func invalidBenefitGrantInput(field, message string) error {
	return infraerrors.BadRequest(
		"BENEFIT_GRANT_INVALID_INPUT",
		message,
	).WithMetadata(map[string]string{"field": field})
}
