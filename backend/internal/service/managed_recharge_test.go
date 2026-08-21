package service

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

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
