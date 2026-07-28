//go:build integration

package repository

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type APIKeyRepoSuite struct {
	suite.Suite
	ctx    context.Context
	client *dbent.Client
	repo   *apiKeyRepository
}

func (s *APIKeyRepoSuite) SetupTest() {
	s.ctx = context.Background()
	tx := testEntTx(s.T())
	s.client = tx.Client()
	s.repo = newAPIKeyRepositoryWithSQL(s.client, tx)
}

func TestAPIKeyRepoSuite(t *testing.T) {
	suite.Run(t, new(APIKeyRepoSuite))
}

// --- Create / GetByID / GetByKey ---

func (s *APIKeyRepoSuite) TestCreate() {
	user := s.mustCreateUser("create@test.com")

	key := &service.APIKey{
		UserID: user.ID,
		Key:    "sk-create-test",
		Name:   "Test Key",
		Status: service.StatusActive,
	}

	err := s.repo.Create(s.ctx, key)
	s.Require().NoError(err, "Create")
	s.Require().NotZero(key.ID, "expected ID to be set")

	got, err := s.repo.GetByID(s.ctx, key.ID)
	s.Require().NoError(err, "GetByID")
	s.Require().Equal("sk-create-test", got.Key)
}

func (s *APIKeyRepoSuite) TestGetByID_NotFound() {
	_, err := s.repo.GetByID(s.ctx, 999999)
	s.Require().Error(err, "expected error for non-existent ID")
}

func (s *APIKeyRepoSuite) TestGetByKey() {
	user := s.mustCreateUser("getbykey@test.com")
	group := s.mustCreateGroup("g-key")

	key := &service.APIKey{
		UserID:  user.ID,
		Key:     "sk-getbykey",
		Name:    "My Key",
		GroupID: &group.ID,
		Status:  service.StatusActive,
	}
	s.Require().NoError(s.repo.Create(s.ctx, key))

	got, err := s.repo.GetByKey(s.ctx, key.Key)
	s.Require().NoError(err, "GetByKey")
	s.Require().Equal(key.ID, got.ID)
	s.Require().NotNil(got.User, "expected User preload")
	s.Require().Equal(user.ID, got.User.ID)
	s.Require().NotNil(got.Group, "expected Group preload")
	s.Require().Equal(group.ID, got.Group.ID)
}

func (s *APIKeyRepoSuite) TestGetByKey_NotFound() {
	_, err := s.repo.GetByKey(s.ctx, "non-existent-key")
	s.Require().Error(err, "expected error for non-existent key")
}

func (s *APIKeyRepoSuite) TestLiheInternalKeysAreHiddenFromUserOperations() {
	user := s.mustCreateUser("lihe-hidden@test.com")
	group := s.mustCreateGroup("g-lihe-hidden")
	normal := s.mustCreateApiKey(user.ID, "sk-visible-lihe-test", "Visible", &group.ID)
	internal := s.mustCreateApiKey(
		user.ID,
		service.LiheInternalAPIKeyPrefix+"hidden-test-value",
		"lihe.chat / openai",
		&group.ID,
	)

	keys, page, err := s.repo.ListByUserID(
		s.ctx,
		user.ID,
		pagination.PaginationParams{Page: 1, PageSize: 20},
		service.APIKeyListFilters{},
	)
	s.Require().NoError(err)
	s.Require().Equal(int64(1), page.Total)
	s.Require().Len(keys, 1)
	s.Require().Equal(normal.ID, keys[0].ID)

	count, err := s.repo.CountByUserID(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().Equal(int64(1), count)

	_, err = s.repo.GetByID(s.ctx, internal.ID)
	s.Require().ErrorIs(err, service.ErrAPIKeyNotFound)
	_, err = s.repo.GetByKey(s.ctx, internal.Key)
	s.Require().ErrorIs(err, service.ErrAPIKeyNotFound)
	_, err = s.repo.GetByKeyForAuth(s.ctx, internal.Key)
	s.Require().ErrorIs(err, service.ErrAPIKeyNotFound)

	owned, err := s.repo.VerifyOwnership(s.ctx, user.ID, []int64{normal.ID, internal.ID})
	s.Require().NoError(err)
	s.Require().Equal([]int64{normal.ID}, owned)

	cacheKeys, err := s.repo.ListKeysByUserID(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().ElementsMatch([]string{normal.Key, internal.Key}, cacheKeys)
}

func (s *APIKeyRepoSuite) TestLiheOAuthRevokeLeavesSelectedAPIKeyActive() {
	user := s.mustCreateUser("lihe-revoke-source@test.com")
	group := s.mustCreateGroup("g-lihe-revoke-source")
	key := s.mustCreateApiKey(user.ID, "sk-lihe-revoke-source", "Source key", &group.ID)
	tokenID := s.mustCreateLiheTokenBinding(strings.Repeat("a", 64), user.ID, key.ID, group.ID, group.Platform)
	tokens, err := s.repo.ListLiheAccessTokens(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().Len(tokens, 1)
	s.Require().NotNil(tokens[0].APIKeyID)
	s.Require().Equal(key.ID, *tokens[0].APIKeyID)
	s.Require().Equal(key.Name, tokens[0].APIKeyName)
	s.Require().Equal([]string{group.Platform}, tokens[0].Providers)

	revoked, err := s.repo.RevokeLiheAccessTokenByID(s.ctx, tokenID, user.ID)
	s.Require().NoError(err)
	s.Require().True(revoked)

	var keyStatus string
	s.Require().NoError(scanSingleRow(
		s.ctx,
		s.repo.sql,
		`SELECT status FROM api_keys WHERE id = $1`,
		[]any{key.ID},
		&keyStatus,
	))
	s.Require().Equal(service.StatusAPIKeyActive, keyStatus)
}

func (s *APIKeyRepoSuite) TestLiheOAuthProviderTokensCoexistAndRevokeIndependently() {
	user := s.mustCreateUser("lihe-provider-tokens@test.com")
	openAIGroupEntity, err := s.client.Group.Create().
		SetName("g-lihe-openai-token").
		SetPlatform(service.PlatformOpenAI).
		SetStatus(service.StatusActive).
		Save(s.ctx)
	s.Require().NoError(err)
	openAIGroup := groupEntityToService(openAIGroupEntity)
	anthropicGroupEntity, err := s.client.Group.Create().
		SetName("g-lihe-anthropic-token").
		SetPlatform(service.PlatformAnthropic).
		SetStatus(service.StatusActive).
		Save(s.ctx)
	s.Require().NoError(err)
	anthropicGroup := groupEntityToService(anthropicGroupEntity)
	openAIKey := s.mustCreateApiKey(user.ID, "sk-lihe-openai-token", "OpenAI source", &openAIGroup.ID)
	anthropicKey := s.mustCreateApiKey(user.ID, "sk-lihe-anthropic-token", "Anthropic source", &anthropicGroup.ID)
	openAITokenHash := strings.Repeat("d", 64)
	anthropicTokenHash := strings.Repeat("e", 64)
	openAITokenID := s.mustCreateLiheTokenBinding(
		openAITokenHash,
		user.ID,
		openAIKey.ID,
		openAIGroup.ID,
		openAIGroup.Platform,
	)
	anthropicTokenID := s.mustCreateLiheTokenBinding(
		anthropicTokenHash,
		user.ID,
		anthropicKey.ID,
		anthropicGroup.ID,
		anthropicGroup.Platform,
	)

	tokens, err := s.repo.ListLiheAccessTokens(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().Len(tokens, 2)
	s.Require().ElementsMatch([]int64{openAITokenID, anthropicTokenID}, []int64{tokens[0].ID, tokens[1].ID})

	openAIResolved, err := s.repo.ResolveLiheAccessToken(s.ctx, openAITokenHash, "lihe-chat")
	s.Require().NoError(err)
	s.Require().NotNil(openAIResolved)
	s.Require().NotNil(openAIResolved.APIKey)
	s.Require().Equal(openAIKey.ID, openAIResolved.APIKey.ID)
	anthropicResolved, err := s.repo.ResolveLiheAccessToken(s.ctx, anthropicTokenHash, "lihe-chat")
	s.Require().NoError(err)
	s.Require().NotNil(anthropicResolved)
	s.Require().NotNil(anthropicResolved.APIKey)
	s.Require().Equal(anthropicKey.ID, anthropicResolved.APIKey.ID)

	revoked, err := s.repo.RevokeLiheAccessTokenByID(s.ctx, anthropicTokenID, user.ID)
	s.Require().NoError(err)
	s.Require().True(revoked)
	anthropicResolved, err = s.repo.ResolveLiheAccessToken(s.ctx, anthropicTokenHash, "lihe-chat")
	s.Require().NoError(err)
	s.Require().Nil(anthropicResolved)
	openAIResolved, err = s.repo.ResolveLiheAccessToken(s.ctx, openAITokenHash, "lihe-chat")
	s.Require().NoError(err)
	s.Require().NotNil(openAIResolved)
	s.Require().NotNil(openAIResolved.APIKey)
	s.Require().Equal(openAIKey.ID, openAIResolved.APIKey.ID)

	tokens, err = s.repo.ListLiheAccessTokens(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Require().Len(tokens, 1)
	s.Require().Equal(openAITokenID, tokens[0].ID)
}

func (s *APIKeyRepoSuite) TestLiheOAuthResolveRejectsChangedAPIKeyGroup() {
	user := s.mustCreateUser("lihe-changed-group@test.com")
	originalGroup := s.mustCreateGroup("g-lihe-original")
	newGroup := s.mustCreateGroup("g-lihe-new")
	key := s.mustCreateApiKey(user.ID, "sk-lihe-changed-group", "Source key", &originalGroup.ID)
	tokenHash := strings.Repeat("b", 64)
	s.mustCreateLiheTokenBinding(tokenHash, user.ID, key.ID, originalGroup.ID, originalGroup.Platform)

	resolved, err := s.repo.ResolveLiheAccessToken(s.ctx, tokenHash, "lihe-chat")
	s.Require().NoError(err)
	s.Require().True(resolved.BindingFound)
	s.Require().Equal(originalGroup.Platform, resolved.BindingProvider)
	s.Require().NotNil(resolved.APIKey)
	s.Require().Equal(key.ID, resolved.APIKey.ID)

	_, err = s.client.APIKey.UpdateOneID(key.ID).SetGroupID(newGroup.ID).Save(s.ctx)
	s.Require().NoError(err)

	resolved, err = s.repo.ResolveLiheAccessToken(s.ctx, tokenHash, "lihe-chat")
	s.Require().NoError(err)
	s.Require().True(resolved.BindingFound)
	s.Require().Nil(resolved.APIKey)
}

func (s *APIKeyRepoSuite) TestLiheOAuthLegacyInternalBindingRemainsCompatible() {
	user := s.mustCreateUser("lihe-legacy-binding@test.com")
	group := s.mustCreateGroup("g-lihe-legacy-binding")
	key := s.mustCreateApiKey(
		user.ID,
		service.LiheInternalAPIKeyPrefix+"legacy-binding-test",
		"Legacy Lihe key",
		&group.ID,
	)
	tokenHash := strings.Repeat("c", 64)
	tokenID := s.mustCreateLegacyLiheTokenBinding(tokenHash, user.ID, key.ID, group.ID, group.Platform)

	resolved, err := s.repo.ResolveLiheAccessToken(s.ctx, tokenHash, "lihe-chat")
	s.Require().NoError(err)
	s.Require().True(resolved.BindingFound)
	s.Require().Equal(group.Platform, resolved.BindingProvider)
	s.Require().NotNil(resolved.APIKey)
	s.Require().Equal(key.ID, resolved.APIKey.ID)

	revoked, err := s.repo.RevokeLiheAccessTokenByID(s.ctx, tokenID, user.ID)
	s.Require().NoError(err)
	s.Require().True(revoked)

	var keyStatus string
	s.Require().NoError(scanSingleRow(
		s.ctx,
		s.repo.sql,
		`SELECT status FROM api_keys WHERE id = $1`,
		[]any{key.ID},
		&keyStatus,
	))
	s.Require().Equal(service.StatusAPIKeyDisabled, keyStatus)
}

func (s *APIKeyRepoSuite) TestGetByKeyForAuth_PreservesMessagesDispatchModelConfig() {
	user := s.mustCreateUser("getbykey-auth-dispatch@test.com")
	group, err := s.client.Group.Create().
		SetName("g-auth-dispatch").
		SetPlatform(service.PlatformOpenAI).
		SetStatus(service.StatusActive).
		SetSubscriptionType(service.SubscriptionTypeStandard).
		SetRateMultiplier(1).
		SetAllowMessagesDispatch(true).
		SetDefaultMappedModel("gpt-5.4").
		SetMessagesDispatchModelConfig(service.OpenAIMessagesDispatchModelConfig{
			OpusMappedModel:   "gpt-5.4-nano",
			SonnetMappedModel: "gpt-5.3-codex",
			HaikuMappedModel:  "gpt-5.4-mini",
			ExactModelMappings: map[string]string{
				"claude-sonnet-4.5": "gpt-5.4-nano",
			},
		}).
		Save(s.ctx)
	s.Require().NoError(err)

	key := &service.APIKey{
		UserID:  user.ID,
		Key:     "sk-getbykey-auth-dispatch",
		Name:    "Dispatch Key",
		GroupID: &group.ID,
		Status:  service.StatusActive,
	}
	s.Require().NoError(s.repo.Create(s.ctx, key))

	got, err := s.repo.GetByKeyForAuth(s.ctx, key.Key)
	s.Require().NoError(err)
	s.Require().NotNil(got.Group)
	s.Require().True(got.Group.AllowMessagesDispatch)
	s.Require().Equal("gpt-5.4", got.Group.DefaultMappedModel)
	s.Require().Equal("gpt-5.4-nano", got.Group.MessagesDispatchModelConfig.OpusMappedModel)
	s.Require().Equal("gpt-5.4-nano", got.Group.MessagesDispatchModelConfig.ExactModelMappings["claude-sonnet-4.5"])
}

// --- Update ---

func (s *APIKeyRepoSuite) TestUpdate() {
	user := s.mustCreateUser("update@test.com")
	key := &service.APIKey{
		UserID: user.ID,
		Key:    "sk-update",
		Name:   "Original",
		Status: service.StatusActive,
	}
	s.Require().NoError(s.repo.Create(s.ctx, key))

	key.Name = "Renamed"
	key.Status = service.StatusDisabled
	err := s.repo.Update(s.ctx, key, service.APIKeyUpdateFields{Name: true, Status: true})
	s.Require().NoError(err, "Update")

	got, err := s.repo.GetByID(s.ctx, key.ID)
	s.Require().NoError(err, "GetByID after update")
	s.Require().Equal("sk-update", got.Key, "Update should not change key")
	s.Require().Equal(user.ID, got.UserID, "Update should not change user_id")
	s.Require().Equal("Renamed", got.Name)
	s.Require().Equal(service.StatusDisabled, got.Status)
}

func (s *APIKeyRepoSuite) TestUpdate_ClearGroupID() {
	user := s.mustCreateUser("cleargroup@test.com")
	group := s.mustCreateGroup("g-clear")
	key := &service.APIKey{
		UserID:  user.ID,
		Key:     "sk-clear-group",
		Name:    "Group Key",
		GroupID: &group.ID,
		Status:  service.StatusActive,
	}
	s.Require().NoError(s.repo.Create(s.ctx, key))

	key.GroupID = nil
	err := s.repo.Update(s.ctx, key, service.APIKeyUpdateFields{GroupID: true})
	s.Require().NoError(err, "Update")

	got, err := s.repo.GetByID(s.ctx, key.ID)
	s.Require().NoError(err)
	s.Require().Nil(got.GroupID, "expected GroupID to be cleared")
}

// --- Delete ---

func (s *APIKeyRepoSuite) TestDelete() {
	user := s.mustCreateUser("delete@test.com")
	key := &service.APIKey{
		UserID: user.ID,
		Key:    "sk-delete",
		Name:   "Delete Me",
		Status: service.StatusActive,
	}
	s.Require().NoError(s.repo.Create(s.ctx, key))

	err := s.repo.Delete(s.ctx, key.ID)
	s.Require().NoError(err, "Delete")

	_, err = s.repo.GetByID(s.ctx, key.ID)
	s.Require().Error(err, "expected error after delete")
}

func (s *APIKeyRepoSuite) TestCreate_AfterSoftDelete_AllowsSameKey() {
	user := s.mustCreateUser("recreate-after-soft-delete@test.com")
	const reusedKey = "sk-reuse-after-soft-delete"

	first := &service.APIKey{
		UserID: user.ID,
		Key:    reusedKey,
		Name:   "First Key",
		Status: service.StatusActive,
	}
	s.Require().NoError(s.repo.Create(s.ctx, first), "create first key")

	s.Require().NoError(s.repo.Delete(s.ctx, first.ID), "soft delete first key")

	second := &service.APIKey{
		UserID: user.ID,
		Key:    reusedKey,
		Name:   "Second Key",
		Status: service.StatusActive,
	}
	s.Require().NoError(s.repo.Create(s.ctx, second), "create second key with same key")
	s.Require().NotZero(second.ID)
	s.Require().NotEqual(first.ID, second.ID, "recreated key should be a new row")
}

// --- ListByUserID / CountByUserID ---

func (s *APIKeyRepoSuite) TestListByUserID() {
	user := s.mustCreateUser("listbyuser@test.com")
	s.mustCreateApiKey(user.ID, "sk-list-1", "Key 1", nil)
	s.mustCreateApiKey(user.ID, "sk-list-2", "Key 2", nil)

	keys, page, err := s.repo.ListByUserID(s.ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 10}, service.APIKeyListFilters{})
	s.Require().NoError(err, "ListByUserID")
	s.Require().Len(keys, 2)
	s.Require().Equal(int64(2), page.Total)
}

func (s *APIKeyRepoSuite) TestListByUserID_Pagination() {
	user := s.mustCreateUser("paging@test.com")
	for i := 0; i < 5; i++ {
		s.mustCreateApiKey(user.ID, "sk-page-"+string(rune('a'+i)), "Key", nil)
	}

	keys, page, err := s.repo.ListByUserID(s.ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 2}, service.APIKeyListFilters{})
	s.Require().NoError(err)
	s.Require().Len(keys, 2)
	s.Require().Equal(int64(5), page.Total)
	s.Require().Equal(3, page.Pages)
}

func (s *APIKeyRepoSuite) TestCountByUserID() {
	user := s.mustCreateUser("count@test.com")
	s.mustCreateApiKey(user.ID, "sk-count-1", "K1", nil)
	s.mustCreateApiKey(user.ID, "sk-count-2", "K2", nil)

	count, err := s.repo.CountByUserID(s.ctx, user.ID)
	s.Require().NoError(err, "CountByUserID")
	s.Require().Equal(int64(2), count)
}

// --- ListByGroupID / CountByGroupID ---

func (s *APIKeyRepoSuite) TestListByGroupID() {
	user := s.mustCreateUser("listbygroup@test.com")
	group := s.mustCreateGroup("g-list")

	s.mustCreateApiKey(user.ID, "sk-grp-1", "K1", &group.ID)
	s.mustCreateApiKey(user.ID, "sk-grp-2", "K2", &group.ID)
	s.mustCreateApiKey(user.ID, "sk-grp-3", "K3", nil) // no group

	keys, page, err := s.repo.ListByGroupID(s.ctx, group.ID, pagination.PaginationParams{Page: 1, PageSize: 10})
	s.Require().NoError(err, "ListByGroupID")
	s.Require().Len(keys, 2)
	s.Require().Equal(int64(2), page.Total)
	// User preloaded
	s.Require().NotNil(keys[0].User)
}

func (s *APIKeyRepoSuite) TestCountByGroupID() {
	user := s.mustCreateUser("countgroup@test.com")
	group := s.mustCreateGroup("g-count")
	s.mustCreateApiKey(user.ID, "sk-gc-1", "K1", &group.ID)

	count, err := s.repo.CountByGroupID(s.ctx, group.ID)
	s.Require().NoError(err, "CountByGroupID")
	s.Require().Equal(int64(1), count)
}

// --- ExistsByKey ---

func (s *APIKeyRepoSuite) TestExistsByKey() {
	user := s.mustCreateUser("exists@test.com")
	s.mustCreateApiKey(user.ID, "sk-exists", "K", nil)

	exists, err := s.repo.ExistsByKey(s.ctx, "sk-exists")
	s.Require().NoError(err, "ExistsByKey")
	s.Require().True(exists)

	notExists, err := s.repo.ExistsByKey(s.ctx, "sk-not-exists")
	s.Require().NoError(err)
	s.Require().False(notExists)
}

// --- SearchAPIKeys ---

func (s *APIKeyRepoSuite) TestSearchAPIKeys() {
	user := s.mustCreateUser("search@test.com")
	s.mustCreateApiKey(user.ID, "sk-search-1", "Production Key", nil)
	s.mustCreateApiKey(user.ID, "sk-search-2", "Development Key", nil)

	found, err := s.repo.SearchAPIKeys(s.ctx, user.ID, "prod", 10)
	s.Require().NoError(err, "SearchAPIKeys")
	s.Require().Len(found, 1)
	s.Require().Contains(found[0].Name, "Production")
}

func (s *APIKeyRepoSuite) TestSearchAPIKeys_NoKeyword() {
	user := s.mustCreateUser("searchnokw@test.com")
	s.mustCreateApiKey(user.ID, "sk-nk-1", "K1", nil)
	s.mustCreateApiKey(user.ID, "sk-nk-2", "K2", nil)

	found, err := s.repo.SearchAPIKeys(s.ctx, user.ID, "", 10)
	s.Require().NoError(err)
	s.Require().Len(found, 2)
}

func (s *APIKeyRepoSuite) TestSearchAPIKeys_NoUserID() {
	user := s.mustCreateUser("searchnouid@test.com")
	s.mustCreateApiKey(user.ID, "sk-nu-1", "TestKey", nil)

	found, err := s.repo.SearchAPIKeys(s.ctx, 0, "testkey", 10)
	s.Require().NoError(err)
	s.Require().Len(found, 1)
}

// --- ClearGroupIDByGroupID ---

func (s *APIKeyRepoSuite) TestClearGroupIDByGroupID() {
	user := s.mustCreateUser("cleargrp@test.com")
	group := s.mustCreateGroup("g-clear-bulk")

	k1 := s.mustCreateApiKey(user.ID, "sk-clr-1", "K1", &group.ID)
	k2 := s.mustCreateApiKey(user.ID, "sk-clr-2", "K2", &group.ID)
	s.mustCreateApiKey(user.ID, "sk-clr-3", "K3", nil) // no group

	affected, err := s.repo.ClearGroupIDByGroupID(s.ctx, group.ID)
	s.Require().NoError(err, "ClearGroupIDByGroupID")
	s.Require().Equal(int64(2), affected)

	got1, _ := s.repo.GetByID(s.ctx, k1.ID)
	got2, _ := s.repo.GetByID(s.ctx, k2.ID)
	s.Require().Nil(got1.GroupID)
	s.Require().Nil(got2.GroupID)

	count, _ := s.repo.CountByGroupID(s.ctx, group.ID)
	s.Require().Zero(count)
}

// --- Combined CRUD/Search/ClearGroupID (original test preserved as integration) ---

func (s *APIKeyRepoSuite) TestCRUD_Search_ClearGroupID() {
	user := s.mustCreateUser("k@example.com")
	group := s.mustCreateGroup("g-k")
	key := s.mustCreateApiKey(user.ID, "sk-test-1", "My Key", &group.ID)
	key.GroupID = &group.ID

	got, err := s.repo.GetByKey(s.ctx, key.Key)
	s.Require().NoError(err, "GetByKey")
	s.Require().Equal(key.ID, got.ID)
	s.Require().NotNil(got.User)
	s.Require().Equal(user.ID, got.User.ID)
	s.Require().NotNil(got.Group)
	s.Require().Equal(group.ID, got.Group.ID)

	key.Name = "Renamed"
	key.Status = service.StatusDisabled
	key.GroupID = nil
	s.Require().NoError(s.repo.Update(s.ctx, key, service.APIKeyUpdateFields{Name: true, Status: true, GroupID: true}), "Update")

	got2, err := s.repo.GetByID(s.ctx, key.ID)
	s.Require().NoError(err, "GetByID")
	s.Require().Equal("sk-test-1", got2.Key, "Update should not change key")
	s.Require().Equal(user.ID, got2.UserID, "Update should not change user_id")
	s.Require().Equal("Renamed", got2.Name)
	s.Require().Equal(service.StatusDisabled, got2.Status)
	s.Require().Nil(got2.GroupID)

	keys, page, err := s.repo.ListByUserID(s.ctx, user.ID, pagination.PaginationParams{Page: 1, PageSize: 10}, service.APIKeyListFilters{})
	s.Require().NoError(err, "ListByUserID")
	s.Require().Equal(int64(1), page.Total)
	s.Require().Len(keys, 1)

	exists, err := s.repo.ExistsByKey(s.ctx, "sk-test-1")
	s.Require().NoError(err, "ExistsByKey")
	s.Require().True(exists, "expected key to exist")

	found, err := s.repo.SearchAPIKeys(s.ctx, user.ID, "renam", 10)
	s.Require().NoError(err, "SearchAPIKeys")
	s.Require().Len(found, 1)
	s.Require().Equal(key.ID, found[0].ID)

	// ClearGroupIDByGroupID
	k2 := s.mustCreateApiKey(user.ID, "sk-test-2", "Group Key", &group.ID)
	k2.GroupID = &group.ID

	countBefore, err := s.repo.CountByGroupID(s.ctx, group.ID)
	s.Require().NoError(err, "CountByGroupID")
	s.Require().Equal(int64(1), countBefore, "expected 1 key in group before clear")

	affected, err := s.repo.ClearGroupIDByGroupID(s.ctx, group.ID)
	s.Require().NoError(err, "ClearGroupIDByGroupID")
	s.Require().Equal(int64(1), affected, "expected 1 affected row")

	got3, err := s.repo.GetByID(s.ctx, k2.ID)
	s.Require().NoError(err, "GetByID")
	s.Require().Nil(got3.GroupID, "expected GroupID cleared")

	countAfter, err := s.repo.CountByGroupID(s.ctx, group.ID)
	s.Require().NoError(err, "CountByGroupID after clear")
	s.Require().Equal(int64(0), countAfter, "expected 0 keys in group after clear")
}

func (s *APIKeyRepoSuite) mustCreateUser(email string) *service.User {
	s.T().Helper()

	u, err := s.client.User.Create().
		SetEmail(email).
		SetPasswordHash("test-password-hash").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(s.ctx)
	s.Require().NoError(err, "create user")
	return userEntityToService(u)
}

func (s *APIKeyRepoSuite) mustCreateGroup(name string) *service.Group {
	s.T().Helper()

	g, err := s.client.Group.Create().
		SetName(name).
		SetStatus(service.StatusActive).
		Save(s.ctx)
	s.Require().NoError(err, "create group")
	return groupEntityToService(g)
}

func (s *APIKeyRepoSuite) mustCreateApiKey(userID int64, key, name string, groupID *int64) *service.APIKey {
	s.T().Helper()

	k := &service.APIKey{
		UserID:  userID,
		Key:     key,
		Name:    name,
		GroupID: groupID,
		Status:  service.StatusActive,
	}
	s.Require().NoError(s.repo.Create(s.ctx, k), "create api key")
	return k
}

func (s *APIKeyRepoSuite) mustCreateLiheTokenBinding(
	tokenHash string,
	userID, apiKeyID, groupID int64,
	provider string,
) int64 {
	s.T().Helper()
	tokenID := s.mustCreateLiheAccessToken(tokenHash, userID)
	_, err := s.repo.sql.ExecContext(
		s.ctx,
		`INSERT INTO lihe_oauth_token_bindings (token_id, provider, group_id, source_api_key_id)
		 VALUES ($1, $2, $3, $4)`,
		tokenID,
		provider,
		groupID,
		apiKeyID,
	)
	s.Require().NoError(err)
	return tokenID
}

func (s *APIKeyRepoSuite) mustCreateLegacyLiheTokenBinding(
	tokenHash string,
	userID, apiKeyID, groupID int64,
	provider string,
) int64 {
	s.T().Helper()
	tokenID := s.mustCreateLiheAccessToken(tokenHash, userID)
	_, err := s.repo.sql.ExecContext(
		s.ctx,
		`INSERT INTO lihe_oauth_token_bindings (token_id, provider, group_id, api_key_id)
		 VALUES ($1, $2, $3, $4)`,
		tokenID,
		provider,
		groupID,
		apiKeyID,
	)
	s.Require().NoError(err)
	return tokenID
}

func (s *APIKeyRepoSuite) mustCreateLiheAccessToken(tokenHash string, userID int64) int64 {
	s.T().Helper()
	var tokenID int64
	err := scanSingleRow(
		s.ctx,
		s.repo.sql,
		`INSERT INTO lihe_oauth_access_tokens (user_id, token_hash, name, client_id, scopes)
		 VALUES ($1, $2, 'lihe.chat', 'lihe-chat', ARRAY['models:read', 'chat:write'])
		 RETURNING id`,
		[]any{userID, tokenHash},
		&tokenID,
	)
	s.Require().NoError(err)
	return tokenID
}

// --- IncrementQuotaUsed ---

func (s *APIKeyRepoSuite) TestIncrementQuotaUsed_Basic() {
	user := s.mustCreateUser("incr-basic@test.com")
	key := s.mustCreateApiKey(user.ID, "sk-incr-basic", "Incr", nil)

	newQuota, err := s.repo.IncrementQuotaUsed(s.ctx, key.ID, 1.5)
	s.Require().NoError(err, "IncrementQuotaUsed")
	s.Require().Equal(1.5, newQuota, "第一次递增后应为 1.5")

	newQuota, err = s.repo.IncrementQuotaUsed(s.ctx, key.ID, 2.5)
	s.Require().NoError(err, "IncrementQuotaUsed second")
	s.Require().Equal(4.0, newQuota, "第二次递增后应为 4.0")
}

func (s *APIKeyRepoSuite) TestIncrementQuotaUsed_NotFound() {
	_, err := s.repo.IncrementQuotaUsed(s.ctx, 999999, 1.0)
	s.Require().ErrorIs(err, service.ErrAPIKeyNotFound, "不存在的 key 应返回 ErrAPIKeyNotFound")
}

func (s *APIKeyRepoSuite) TestIncrementQuotaUsed_DeletedKey() {
	user := s.mustCreateUser("incr-deleted@test.com")
	key := s.mustCreateApiKey(user.ID, "sk-incr-del", "Deleted", nil)

	s.Require().NoError(s.repo.Delete(s.ctx, key.ID), "Delete")

	_, err := s.repo.IncrementQuotaUsed(s.ctx, key.ID, 1.0)
	s.Require().ErrorIs(err, service.ErrAPIKeyNotFound, "已删除的 key 应返回 ErrAPIKeyNotFound")
}

func (s *APIKeyRepoSuite) TestIncrementQuotaUsedAndGetState() {
	user := s.mustCreateUser("quota-state@test.com")
	key := s.mustCreateApiKey(user.ID, "sk-quota-state", "QuotaState", nil)
	key.Quota = 3
	key.QuotaUsed = 1
	s.Require().NoError(s.repo.Update(s.ctx, key, service.APIKeyUpdateFields{Quota: true, QuotaUsed: true}), "Update quota")

	state, err := s.repo.IncrementQuotaUsedAndGetState(s.ctx, key.ID, 2.5)
	s.Require().NoError(err, "IncrementQuotaUsedAndGetState")
	s.Require().NotNil(state)
	s.Require().Equal(3.5, state.QuotaUsed)
	s.Require().Equal(3.0, state.Quota)
	s.Require().Equal(service.StatusAPIKeyQuotaExhausted, state.Status)
	s.Require().Equal(key.Key, state.Key)

	got, err := s.repo.GetByID(s.ctx, key.ID)
	s.Require().NoError(err, "GetByID")
	s.Require().Equal(3.5, got.QuotaUsed)
	s.Require().Equal(service.StatusAPIKeyQuotaExhausted, got.Status)
}

// TestIncrementQuotaUsed_Concurrent 使用真实数据库验证并发原子性。
// 注意：此测试使用 testEntClient（非事务隔离），数据会真正写入数据库。
func TestIncrementQuotaUsed_Concurrent(t *testing.T) {
	client := testEntClient(t)
	repo := NewAPIKeyRepository(client, integrationDB).(*apiKeyRepository)
	ctx := context.Background()

	// 创建测试用户和 API Key
	u, err := client.User.Create().
		SetEmail("concurrent-incr-" + time.Now().Format(time.RFC3339Nano) + "@test.com").
		SetPasswordHash("hash").
		SetStatus(service.StatusActive).
		SetRole(service.RoleUser).
		Save(ctx)
	require.NoError(t, err, "create user")

	k := &service.APIKey{
		UserID: u.ID,
		Key:    "sk-concurrent-" + time.Now().Format(time.RFC3339Nano),
		Name:   "Concurrent",
		Status: service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, k), "create api key")
	t.Cleanup(func() {
		_ = client.APIKey.DeleteOneID(k.ID).Exec(ctx)
		_ = client.User.DeleteOneID(u.ID).Exec(ctx)
	})

	// 10 个 goroutine 各递增 1.0，总计应为 10.0
	const goroutines = 10
	const increment = 1.0
	var wg sync.WaitGroup
	errs := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = repo.IncrementQuotaUsed(ctx, k.ID, increment)
		}(i)
	}
	wg.Wait()

	for i, e := range errs {
		require.NoError(t, e, "goroutine %d failed", i)
	}

	// 验证最终结果
	got, err := repo.GetByID(ctx, k.ID)
	require.NoError(t, err, "GetByID")
	require.Equal(t, float64(goroutines)*increment, got.QuotaUsed,
		"并发递增后总和应为 %v，实际为 %v", float64(goroutines)*increment, got.QuotaUsed)
}

func (s *APIKeyRepoSuite) TestDeleteWithAudit_TombstonesWithoutRetainingCredential() {
	user := s.mustCreateUser("delwithaudit@test.com")
	key := &service.APIKey{
		UserID: user.ID,
		Key:    "sk-del-audit-1",
		Name:   "Audit Me",
		Status: service.StatusActive,
	}
	s.Require().NoError(s.repo.Create(s.ctx, key))

	s.Require().NoError(s.repo.DeleteWithAudit(s.ctx, key.ID))

	_, err := s.repo.GetByID(s.ctx, key.ID)
	s.Require().Error(err)

	var tombstone string
	var deletedAt time.Time
	rows, err := s.repo.sql.QueryContext(s.ctx, `SELECT key, deleted_at FROM api_keys WHERE id = $1`, key.ID)
	s.Require().NoError(err)
	s.Require().True(rows.Next())
	s.Require().NoError(rows.Scan(&tombstone, &deletedAt))
	s.Require().NoError(rows.Close())
	s.Require().NotEqual("sk-del-audit-1", tombstone)
	s.Require().Contains(tombstone, "__deleted__")

	var auditCount int
	auditRows, err := s.repo.sql.QueryContext(s.ctx,
		`SELECT COUNT(*) FROM deleted_api_key_audits WHERE api_key_id = $1`, key.ID)
	s.Require().NoError(err)
	s.Require().True(auditRows.Next())
	s.Require().NoError(auditRows.Scan(&auditCount))
	s.Require().NoError(auditRows.Close())
	s.Require().Zero(auditCount, "deleted credentials must not be retained")
}

func (s *APIKeyRepoSuite) TestDeleteWithAudit_RepeatIsIdempotent() {
	user := s.mustCreateUser("delwithaudit-idem@test.com")
	key := &service.APIKey{UserID: user.ID, Key: "sk-del-audit-2", Name: "K", Status: service.StatusActive}
	s.Require().NoError(s.repo.Create(s.ctx, key))

	s.Require().NoError(s.repo.DeleteWithAudit(s.ctx, key.ID))
	s.Require().NoError(s.repo.DeleteWithAudit(s.ctx, key.ID))
}

func (s *APIKeyRepoSuite) TestDeleteWithAudit_NotFound() {
	err := s.repo.DeleteWithAudit(s.ctx, 999999)
	s.Require().ErrorIs(err, service.ErrAPIKeyNotFound)
}
