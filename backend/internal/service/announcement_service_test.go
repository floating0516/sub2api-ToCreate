package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type announcementRepoStub struct {
	item *Announcement
}

func (s *announcementRepoStub) Create(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (s *announcementRepoStub) GetByID(_ context.Context, _ int64) (*Announcement, error) {
	if s.item == nil {
		return nil, ErrAnnouncementNotFound
	}
	return s.item, nil
}

func (s *announcementRepoStub) Update(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (*announcementRepoStub) Delete(context.Context, int64) error {
	return nil
}

func (*announcementRepoStub) List(context.Context, pagination.PaginationParams, AnnouncementListFilters) ([]Announcement, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *announcementRepoStub) ListActive(context.Context, time.Time) ([]Announcement, error) {
	if s.item == nil {
		return nil, nil
	}
	return []Announcement{*s.item}, nil
}

type announcementReadRepoStub struct {
	AnnouncementReadRepository
}

func (*announcementReadRepoStub) GetReadMapByUser(
	context.Context,
	int64,
	[]int64,
) (map[int64]time.Time, error) {
	return map[int64]time.Time{}, nil
}

type announcementUserRepoStub struct {
	UserRepository
	user *User
}

func (s *announcementUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	return s.user, nil
}

type announcementUserSubRepoStub struct {
	UserSubscriptionRepository
}

func (*announcementUserSubRepoStub) ListActiveByUserID(
	context.Context,
	int64,
) ([]UserSubscription, error) {
	return nil, nil
}

type announcementBenefitGrantRepoStub struct {
	BenefitGrantRepository
	access map[int64]bool
}

func (s *announcementBenefitGrantRepoStub) ListAnnouncementGrantAccess(
	context.Context,
	int64,
	[]int64,
) (map[int64]bool, error) {
	return s.access, nil
}

func TestAnnouncementServiceCreateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{}
	svc := NewAnnouncementService(repo, nil, nil, nil, nil)
	now := time.Unix(1776790020, 0)

	_, err := svc.Create(context.Background(), &CreateAnnouncementInput{
		Title:      "公告",
		Content:    "内容",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModePopup,
		StartsAt:   &now,
		EndsAt:     &now,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceUpdateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{
		item: &Announcement{
			ID:         1,
			Title:      "公告",
			Content:    "内容",
			Status:     AnnouncementStatusActive,
			NotifyMode: AnnouncementNotifyModePopup,
		},
	}
	svc := NewAnnouncementService(repo, nil, nil, nil, nil)
	now := time.Unix(1776790020, 0)
	startsAt := &now
	endsAt := &now

	_, err := svc.Update(context.Background(), 1, &UpdateAnnouncementInput{
		StartsAt: &startsAt,
		EndsAt:   &endsAt,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceShowsBoundAnnouncementOnlyAfterGrant(t *testing.T) {
	now := time.Now()
	repo := &announcementRepoStub{
		item: &Announcement{
			ID:         7,
			Title:      "Benefit received",
			Content:    "Your benefit is ready.",
			Status:     AnnouncementStatusActive,
			NotifyMode: AnnouncementNotifyModePopup,
			StartsAt:   &now,
			Targeting: AnnouncementTargeting{
				AnyOf: []AnnouncementConditionGroup{{
					AllOf: []AnnouncementCondition{{
						Type:     AnnouncementConditionTypeSubscription,
						Operator: AnnouncementOperatorIn,
						GroupIDs: []int64{9223372036854775807},
					}},
				}},
			},
		},
	}
	userRepo := &announcementUserRepoStub{
		user: &User{ID: 11, Role: RoleUser, Status: StatusActive},
	}
	benefitRepo := &announcementBenefitGrantRepoStub{
		access: map[int64]bool{7: true},
	}
	svc := NewAnnouncementService(
		repo,
		&announcementReadRepoStub{},
		userRepo,
		&announcementUserSubRepoStub{},
		benefitRepo,
	)

	items, err := svc.ListForUser(context.Background(), 11, false)

	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(7), items[0].Announcement.ID)
}

func TestAnnouncementServiceHidesBoundAnnouncementBeforeGrant(t *testing.T) {
	now := time.Now()
	repo := &announcementRepoStub{
		item: &Announcement{
			ID:         7,
			Title:      "Benefit received",
			Content:    "Your benefit is ready.",
			Status:     AnnouncementStatusActive,
			NotifyMode: AnnouncementNotifyModePopup,
			StartsAt:   &now,
		},
	}
	userRepo := &announcementUserRepoStub{
		user: &User{ID: 11, Role: RoleUser, Status: StatusActive},
	}
	benefitRepo := &announcementBenefitGrantRepoStub{
		access: map[int64]bool{7: false},
	}
	svc := NewAnnouncementService(
		repo,
		&announcementReadRepoStub{},
		userRepo,
		&announcementUserSubRepoStub{},
		benefitRepo,
	)

	items, err := svc.ListForUser(context.Background(), 11, false)

	require.NoError(t, err)
	require.Empty(t, items)
}
