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
	verifyCalls  int
	createCalls  int
	confirmCalls int
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
	_, err = service.CreateOrder(context.Background(), 42, ManagedRechargeCreateOrderInput{
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
	_, err = service.CreateOrder(context.Background(), 42, ManagedRechargeCreateOrderInput{
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
