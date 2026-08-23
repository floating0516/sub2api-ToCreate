package service

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type managedRechargeTestEncryptor struct{}

func (managedRechargeTestEncryptor) Encrypt(plaintext string) (string, error) {
	return plaintext, nil
}

func (managedRechargeTestEncryptor) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

func (managedRechargeTestEncryptor) BlindIndex(plaintext string) string {
	return plaintext
}

type managedRechargeVerifyOnlyUpstream struct {
	verifyCalls       int
	createCalls       int
	confirmCalls      int
	validationCalls   int
	sessionValidation *managedRechargeSessionValidationResponse
	lookupResponse    *managedRechargeLookupResponse
	lookupErr         error
}

func (u *managedRechargeVerifyOnlyUpstream) validateSession(_ context.Context, _ string) (*managedRechargeSessionValidationResponse, error) {
	u.validationCalls++
	if u.sessionValidation != nil {
		return u.sessionValidation, nil
	}
	return &managedRechargeSessionValidationResponse{Valid: true, Email: "verified@example.com"}, nil
}

func (u *managedRechargeVerifyOnlyUpstream) verifyCDK(_ context.Context, _ string) (*managedRechargeVerifyResponse, error) {
	u.verifyCalls++
	return &managedRechargeVerifyResponse{Valid: true, PlanType: "plus", PlanName: "ChatGPT Plus", ProcessingMode: "auto"}, nil
}

func (u *managedRechargeVerifyOnlyUpstream) createTask(_ context.Context, _, _ string) (*managedRechargeCreateResponse, error) {
	u.createCalls++
	return &managedRechargeCreateResponse{TaskID: "unexpected"}, nil
}

func (u *managedRechargeVerifyOnlyUpstream) confirmTask(_ context.Context, _ string) (*managedRechargeConfirmResponse, error) {
	u.confirmCalls++
	return &managedRechargeConfirmResponse{Status: "queued"}, nil
}

func (u *managedRechargeVerifyOnlyUpstream) lookupTask(_ context.Context, _ string) (*managedRechargeLookupResponse, error) {
	if u.lookupResponse != nil || u.lookupErr != nil {
		return u.lookupResponse, u.lookupErr
	}
	return &managedRechargeLookupResponse{}, nil
}

func (u *managedRechargeVerifyOnlyUpstream) submitReplacementSession(_ context.Context, _, _ string) (*managedRechargeReplacementSessionResponse, error) {
	return &managedRechargeReplacementSessionResponse{}, nil
}

func TestNormalizeManagedRechargePlanType(t *testing.T) {
	tests := map[string]string{
		"plus":         "plus",
		"ChatGPT_Plus": "plus",
		"pro":          "pro",
		"chatgpt-pro":  "pro",
		"team":         "",
		"":             "",
	}
	for input, expected := range tests {
		if actual := normalizeManagedRechargePlanType(input); actual != expected {
			t.Fatalf("normalizeManagedRechargePlanType(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestManagedRechargeMembership(t *testing.T) {
	tests := []struct {
		name         string
		subscription *managedRechargeSubscriptionValidation
		expected     string
	}{
		{name: "missing", expected: "free"},
		{name: "inactive", subscription: &managedRechargeSubscriptionValidation{PlanType: "plus"}, expected: "free"},
		{name: "plus", subscription: &managedRechargeSubscriptionValidation{PlanType: "chatgpt_plus", HasActiveSubscription: true}, expected: "plus"},
		{name: "pro 5x", subscription: &managedRechargeSubscriptionValidation{SubscriptionPlan: "ChatGPT Pro 5X", HasActiveSubscription: true}, expected: "pro"},
		{name: "unknown active", subscription: &managedRechargeSubscriptionValidation{PlanType: "team", HasActiveSubscription: true}, expected: "unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := managedRechargeMembership(test.subscription); actual != test.expected {
				t.Fatalf("managedRechargeMembership() = %q, want %q", actual, test.expected)
			}
		})
	}
}

func TestManagedRechargeValidateSessionReturnsVerifiedAccountAndMembership(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	upstream := &managedRechargeVerifyOnlyUpstream{
		sessionValidation: &managedRechargeSessionValidationResponse{
			Valid: true,
			Email: "verified@example.com",
			Subscription: &managedRechargeSubscriptionValidation{
				PlanType:              "chatgpt_plus",
				HasActiveSubscription: true,
			},
		},
	}
	service := &ManagedRechargeService{
		db:           db,
		encryptor:    managedRechargeTestEncryptor{},
		upstream:     upstream,
		featureReady: true,
	}
	result, err := service.ValidateSession(context.Background(), `{"user":{"email":"local@example.com"},"accessToken":"test-token"}`)
	if err != nil {
		t.Fatalf("validate Session: %v", err)
	}
	if !result.Valid || result.Email != "verified@example.com" || result.Membership != "plus" {
		t.Fatalf("unexpected validation result: %+v", result)
	}
}

func TestNormalizeManagedRechargeProductRequiresPlanType(t *testing.T) {
	input := ManagedRechargeProductInput{
		Slug:        "chatgpt_plus",
		Name:        "ChatGPT Plus",
		Description: "Managed recharge",
		Price:       10,
		Active:      true,
	}
	if _, err := normalizeManagedRechargeProduct(input); err == nil {
		t.Fatal("normalizeManagedRechargeProduct accepted a product without plan_type")
	}

	input.PlanType = "ChatGPT_Plus"
	normalized, err := normalizeManagedRechargeProduct(input)
	if err != nil {
		t.Fatalf("normalizeManagedRechargeProduct returned error: %v", err)
	}
	if normalized.PlanType != "plus" {
		t.Fatalf("normalized plan_type = %q, want plus", normalized.PlanType)
	}
}

func TestManagedRechargeRefundRejectsCompletedUpstream(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &ManagedRechargeService{db: db}
	mock.ExpectQuery("SELECT payment_order_id FROM managed_recharge_orders").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"payment_order_id"}).AddRow(nil))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT price, status, paid_at, user_id, upstream_status").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"price", "status", "paid_at", "user_id", "upstream_status"}).
			AddRow(20.0, ManagedRechargeStatusManualReview, time.Now(), int64(42), "completed"))
	mock.ExpectRollback()

	err = service.refundOrder(context.Background(), 9, 42, "ADMIN_REFUND", "refund")
	if infraerrors.Reason(err) != "MANAGED_RECHARGE_COMPLETED" {
		t.Fatalf("refund error reason = %q, want MANAGED_RECHARGE_COMPLETED", infraerrors.Reason(err))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("refund SQL expectations: %v", err)
	}
}

func TestManagedRechargeRefundIsNoopWhenAlreadyRefunded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &ManagedRechargeService{db: db}
	mock.ExpectQuery("SELECT payment_order_id FROM managed_recharge_orders").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"payment_order_id"}).AddRow(nil))
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT price, status, paid_at, user_id, upstream_status").
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"price", "status", "paid_at", "user_id", "upstream_status"}).
			AddRow(20.0, ManagedRechargeStatusRefunded, time.Now(), int64(42), "created"))
	mock.ExpectRollback()

	if err := service.refundOrder(context.Background(), 10, 42, "RETRY", "retry"); err != nil {
		t.Fatalf("already-refunded order returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("already-refunded SQL expectations: %v", err)
	}
}

func TestManagedRechargeMarkTaskCreatedRequiresPaidOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &ManagedRechargeService{db: db}
	mock.ExpectExec("UPDATE managed_recharge_orders").
		WithArgs(int64(11), "task-1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := service.markTaskCreated(context.Background(), 11, "task-1"); err == nil {
		t.Fatal("markTaskCreated accepted an order outside paid state")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("markTaskCreated SQL expectations: %v", err)
	}
}

func TestManagedRechargeMarkTaskCreatedClearsStoredSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &ManagedRechargeService{db: db}
	mock.ExpectExec("(?s)SET status = 'submitting'.*session_ciphertext = ''").
		WithArgs(int64(18), "task-18").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := service.markTaskCreated(context.Background(), 18, "task-18"); err != nil {
		t.Fatalf("mark task created: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mark task created SQL expectations: %v", err)
	}
}

func TestManagedRechargeManualReviewClearsStoredSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &ManagedRechargeService{db: db}
	mock.ExpectExec("(?s)SET status = 'manual_review'.*session_ciphertext = ''").
		WithArgs(int64(19), "UNCERTAIN", "manual review").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := service.markManualReview(context.Background(), 19, "UNCERTAIN", "manual review"); err != nil {
		t.Fatalf("mark manual review: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("manual review SQL expectations: %v", err)
	}
}

func TestManagedRechargeVerifyCDKDoesNotCreateOrConfirmTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	upstream := &managedRechargeVerifyOnlyUpstream{}
	service := &ManagedRechargeService{
		db:           db,
		encryptor:    managedRechargeTestEncryptor{},
		upstream:     upstream,
		featureReady: true,
	}
	mock.ExpectQuery("SELECT c.code_ciphertext, p.plan_type").
		WithArgs(int64(17)).
		WillReturnRows(sqlmock.NewRows([]string{"code_ciphertext", "plan_type"}).AddRow("CDK-SECRET", "plus"))

	result, err := service.VerifyCDK(context.Background(), 17)
	if err != nil {
		t.Fatalf("verify CDK: %v", err)
	}
	if !result.Valid || !result.MatchesProduct || result.VerificationScope != "verify_only" {
		t.Fatalf("unexpected verification result: %+v", result)
	}
	if upstream.verifyCalls != 1 || upstream.createCalls != 0 || upstream.confirmCalls != 0 {
		t.Fatalf("unexpected upstream calls: verify=%d create=%d confirm=%d", upstream.verifyCalls, upstream.createCalls, upstream.confirmCalls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("verify CDK SQL expectations: %v", err)
	}
}

func TestManagedRechargeFailUnpaidOrderDoesNotReleasePaidInventory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &ManagedRechargeService{db: db}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT paid_at FROM managed_recharge_orders").
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"paid_at"}).AddRow(time.Now()))
	mock.ExpectRollback()

	if err := service.failUnpaidOrder(context.Background(), 12, "INTERRUPTED", "interrupted", false); err != nil {
		t.Fatalf("paid order cleanup returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("paid order cleanup SQL expectations: %v", err)
	}
}

func TestManagedRechargeMockModeRejectsNonTestSessionBeforeCreatingOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &ManagedRechargeService{
		db:               db,
		encryptor:        managedRechargeTestEncryptor{},
		upstream:         newManagedRechargeMockUpstream(10 * time.Second),
		mockMode:         true,
		featureReady:     true,
		fulfillmentReady: true,
	}
	_, err = service.createLegacyBalanceOrder(context.Background(), 42, ManagedRechargeCreateOrderInput{
		ProductID:      1,
		Session:        `{"user":{"email":"real@example.com"},"accessToken":"real-token"}`,
		IdempotencyKey: "mock-session-guard",
	})
	if infraerrors.Reason(err) != "MANAGED_RECHARGE_MOCK_SESSION_REQUIRED" {
		t.Fatalf("mock Session guard error reason = %q", infraerrors.Reason(err))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock Session guard touched the database: %v", err)
	}
}

func TestManagedRechargeRealVerificationOnlyModeRejectsOrdersBeforeDatabaseUse(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &ManagedRechargeService{
		db:           db,
		encryptor:    managedRechargeTestEncryptor{},
		upstream:     &managedRechargeVerifyOnlyUpstream{},
		featureReady: true,
	}
	_, err = service.createLegacyBalanceOrder(context.Background(), 42, ManagedRechargeCreateOrderInput{
		ProductID:      1,
		Session:        `{"user":{"email":"test@example.com"},"accessToken":"token"}`,
		IdempotencyKey: "verification-only-guard",
	})
	if infraerrors.Reason(err) != ErrManagedRechargeUnavailable.Reason {
		t.Fatalf("verification-only order error reason = %q, want %q", infraerrors.Reason(err), ErrManagedRechargeUnavailable.Reason)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("verification-only order touched the database: %v", err)
	}
}

func TestManagedRechargeExternalOrderAcceptsEmptySessionAndReturnsIssuedCDK(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now()
	order := ManagedRechargeOrder{
		ID:              21,
		OrderNo:         "MR-EXTERNAL-21",
		UserID:          42,
		ProductID:       1,
		ProductSlug:     "gpt-plus",
		ProductName:     "Plus",
		FulfillmentMode: ManagedRechargeFulfillmentExternal,
		Price:           5,
		Status:          ManagedRechargeStatusIssued,
		PaidAt:          &now,
		LastSyncedAt:    &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	mock.ExpectQuery("SELECT o.id.*WHERE o.user_id = \\$1 AND o.idempotency_key = \\$2").
		WithArgs(int64(42), "external-existing").
		WillReturnRows(managedRechargeOrderTestRows(order, "MOCK-PLUS-SUCCESS-021", managedRechargeCDKUsed))

	upstream := &managedRechargeVerifyOnlyUpstream{}
	service := &ManagedRechargeService{
		db:                db,
		encryptor:         managedRechargeTestEncryptor{},
		upstream:          upstream,
		featureReady:      true,
		fulfillmentReady:  true,
		fulfillmentMode:   ManagedRechargeFulfillmentExternal,
		externalRedeemURL: "https://redeem.example.test/recharge",
	}
	result, err := service.createLegacyBalanceOrder(context.Background(), 42, ManagedRechargeCreateOrderInput{
		ProductID:      1,
		IdempotencyKey: "external-existing",
	})
	if err != nil {
		t.Fatalf("return existing external order: %v", err)
	}
	if result.RedemptionCode != "MOCK-PLUS-SUCCESS-021" || result.RedemptionURL != "https://redeem.example.test/recharge?cdk=MOCK-PLUS-SUCCESS-021" {
		t.Fatalf("unexpected external redemption details: %+v", result)
	}
	if upstream.createCalls != 0 || upstream.confirmCalls != 0 {
		t.Fatalf("external order called proxy fulfillment: create=%d confirm=%d", upstream.createCalls, upstream.confirmCalls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("external existing order SQL expectations: %v", err)
	}
}

func TestManagedRechargeExternalModeRejectsSessionValidation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	upstream := &managedRechargeVerifyOnlyUpstream{}
	service := &ManagedRechargeService{
		db:              db,
		encryptor:       managedRechargeTestEncryptor{},
		upstream:        upstream,
		featureReady:    true,
		fulfillmentMode: ManagedRechargeFulfillmentExternal,
	}
	_, err = service.ValidateSession(context.Background(), `{"user":{"email":"member@example.com"},"accessToken":"secret"}`)
	if infraerrors.Reason(err) != "MANAGED_RECHARGE_SESSION_EXTERNAL" {
		t.Fatalf("external Session validation error reason = %q", infraerrors.Reason(err))
	}
	if upstream.validationCalls != 0 {
		t.Fatalf("external Session was sent upstream %d times", upstream.validationCalls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("external Session validation touched database: %v", err)
	}
}

func TestManagedRechargeExternalRedemptionRequiresUsedInventory(t *testing.T) {
	service := &ManagedRechargeService{
		encryptor:         managedRechargeTestEncryptor{},
		externalRedeemURL: "https://redeem.example.test/recharge",
	}
	now := time.Now()
	order := &ManagedRechargeOrder{
		FulfillmentMode: ManagedRechargeFulfillmentExternal,
		Status:          ManagedRechargeStatusManualReview,
		PaidAt:          &now,
		cdkCiphertext:   "SECRET-CDK",
		cdkStatus:       managedRechargeCDKReserved,
	}
	if err := service.attachExternalRedemption(order); err != nil {
		t.Fatalf("attach reserved external CDK: %v", err)
	}
	if order.RedemptionCode != "" || order.RedemptionURL != "" {
		t.Fatalf("reserved CDK was disclosed: %+v", order)
	}

	order.cdkStatus = managedRechargeCDKUsed
	if err := service.attachExternalRedemption(order); err != nil {
		t.Fatalf("attach used external CDK: %v", err)
	}
	if order.RedemptionCode != "SECRET-CDK" {
		t.Fatalf("used CDK was not disclosed to its owner: %+v", order)
	}
}

func TestManagedRechargeMarkExternalOrderIssuedConsumesInventoryAtomically(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &ManagedRechargeService{db: db}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE managed_recharge_cdks").
		WithArgs(int64(22)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE managed_recharge_orders").
		WithArgs(int64(22)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO managed_recharge_events").
		WithArgs(int64(22), nil, "CDK_ISSUED", `{"fulfillment_mode":"external"}`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := service.markExternalOrderIssued(context.Background(), 22); err != nil {
		t.Fatalf("mark external order issued: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("external issue SQL expectations: %v", err)
	}
}

func TestManagedRechargeExternalLookupNotFoundRemainsIssued(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now()
	order := ManagedRechargeOrder{
		ID:              23,
		OrderNo:         "MR-EXTERNAL-23",
		UserID:          42,
		ProductID:       1,
		ProductSlug:     "gpt-plus",
		ProductName:     "Plus",
		FulfillmentMode: ManagedRechargeFulfillmentExternal,
		Price:           5,
		Status:          ManagedRechargeStatusIssued,
		PaidAt:          &now,
		CreatedAt:       now,
		UpdatedAt:       now,
		cdkCiphertext:   "MOCK-PLUS-SUCCESS-023",
		cdkStatus:       managedRechargeCDKUsed,
	}
	mock.ExpectExec("SET last_synced_at = NOW\\(\\), error_code = '', error_message = ''").
		WithArgs(order.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	syncedAt := now.Add(time.Second)
	stored := order
	stored.LastSyncedAt = &syncedAt
	mock.ExpectQuery("SELECT o.id.*WHERE o.id = \\$1 AND o.user_id = \\$2").
		WithArgs(order.ID, order.UserID).
		WillReturnRows(managedRechargeOrderTestRows(stored, order.cdkCiphertext, managedRechargeCDKUsed))

	service := &ManagedRechargeService{
		db:        db,
		encryptor: managedRechargeTestEncryptor{},
		upstream: &managedRechargeVerifyOnlyUpstream{
			lookupErr: &managedRechargeUpstreamHTTPError{StatusCode: 404},
		},
	}
	result, err := service.syncOrderIfNeeded(context.Background(), &order, true)
	if err != nil {
		t.Fatalf("sync external order before user redemption: %v", err)
	}
	if result.Status != ManagedRechargeStatusIssued || result.ErrorMessage != "" {
		t.Fatalf("external 404 changed issued order: %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("external 404 SQL expectations: %v", err)
	}
}

func TestManagedRechargeExternalFailureStaysRetryableWithoutRefund(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now()
	order := ManagedRechargeOrder{
		ID:              24,
		OrderNo:         "MR-EXTERNAL-24",
		UserID:          42,
		ProductID:       1,
		ProductSlug:     "gpt-plus",
		ProductName:     "Plus",
		FulfillmentMode: ManagedRechargeFulfillmentExternal,
		Price:           5,
		Status:          ManagedRechargeStatusProcessing,
		PaidAt:          &now,
		CreatedAt:       now,
		UpdatedAt:       now,
		cdkCiphertext:   "MOCK-FAIL-REFUND-024",
		cdkStatus:       managedRechargeCDKUsed,
	}
	mock.ExpectExec("SET status = 'issued'").
		WithArgs(order.ID, "failed", "session_invalid", "上游返回失败").
		WillReturnResult(sqlmock.NewResult(0, 1))
	stored := order
	stored.Status = ManagedRechargeStatusIssued
	stored.ErrorCode = "EXTERNAL_REDEEM_RETRY"
	stored.ErrorMessage = "上次兑换未完成，CDK 仍归当前订单所有，请前往兑换页重新提交"
	mock.ExpectQuery("SELECT o.id.*WHERE o.id = \\$1 AND o.user_id = \\$2").
		WithArgs(order.ID, order.UserID).
		WillReturnRows(managedRechargeOrderTestRows(stored, order.cdkCiphertext, managedRechargeCDKUsed))

	service := &ManagedRechargeService{
		db:        db,
		encryptor: managedRechargeTestEncryptor{},
		upstream: &managedRechargeVerifyOnlyUpstream{
			lookupResponse: &managedRechargeLookupResponse{
				TaskStatus:    "failed",
				FailureReason: "session_invalid",
				Progress:      "上游返回失败",
			},
		},
	}
	result, err := service.syncOrderIfNeeded(context.Background(), &order, true)
	if err != nil {
		t.Fatalf("sync failed external redemption: %v", err)
	}
	if result.Status != ManagedRechargeStatusIssued || result.ErrorCode != "EXTERNAL_REDEEM_RETRY" || result.RefundedAt != nil {
		t.Fatalf("external failure was not kept retryable: %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("external failure SQL expectations: %v", err)
	}
}

func managedRechargeOrderTestRows(order ManagedRechargeOrder, cdkCiphertext, cdkStatus string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "order_no", "user_id", "user_email", "username", "product_id", "slug", "name",
		"fulfillment_mode", "code_masked", "price", "status", "payment_order_id", "payment_status", "payment_expires_at", "account_email", "upstream_task_id",
		"upstream_status", "upstream_failure_reason", "queue_position", "queue_total", "progress", "error_code",
		"error_message", "balance_before", "balance_after", "paid_at", "submitted_at", "completed_at",
		"refunded_at", "last_synced_at", "created_at", "updated_at", "session_ciphertext", "code_ciphertext", "cdk_status",
	}).AddRow(
		order.ID, order.OrderNo, order.UserID, order.UserEmail, order.Username, order.ProductID, order.ProductSlug, order.ProductName,
		order.FulfillmentMode, order.CDKMasked, order.Price, order.Status, managedRechargeNullableInt64(order.PaymentOrderID), order.PaymentStatus,
		managedRechargeNullableTime(order.PaymentExpiresAt), order.AccountEmail, order.upstreamTaskID,
		order.UpstreamStatus, order.upstreamFailureReason, order.QueuePosition, order.QueueTotal, order.Progress, order.ErrorCode,
		order.ErrorMessage, order.BalanceBefore, order.BalanceAfter, managedRechargeNullableTime(order.PaidAt),
		managedRechargeNullableTime(order.SubmittedAt), managedRechargeNullableTime(order.CompletedAt),
		managedRechargeNullableTime(order.RefundedAt), managedRechargeNullableTime(order.LastSyncedAt), order.CreatedAt, order.UpdatedAt,
		order.sessionCiphertext, cdkCiphertext, cdkStatus,
	)
}

func managedRechargeNullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func managedRechargeNullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}
