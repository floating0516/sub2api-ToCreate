package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	ManagedRechargeStatusValidating     = "validating"
	ManagedRechargeStatusPaid           = "paid"
	ManagedRechargeStatusSubmitting     = "submitting"
	ManagedRechargeStatusQueued         = "queued"
	ManagedRechargeStatusProcessing     = "processing"
	ManagedRechargeStatusVerifying      = "verifying"
	ManagedRechargeStatusActionRequired = "action_required"
	ManagedRechargeStatusManualReview   = "manual_review"
	ManagedRechargeStatusCompleted      = "completed"
	ManagedRechargeStatusFailed         = "failed"
	ManagedRechargeStatusRefunded       = "refunded"

	managedRechargeCDKAvailable = "available"
	managedRechargeCDKReserved  = "reserved"
	managedRechargeCDKUsed      = "used"
	managedRechargeCDKInvalid   = "invalid"
	managedRechargeCDKDisabled  = "disabled"

	managedRechargeSessionMaxBytes = 128 * 1024
	managedRechargeImportMaxCodes  = 500
	managedRechargeSyncInterval    = 8 * time.Second
	managedRechargeReserveAttempts = 3
	managedRechargeOrderTimeout    = 3 * time.Minute
	managedRechargeFulfillTimeout  = 90 * time.Second
	managedRechargeRecoveryTimeout = 10 * time.Second
	managedRechargeValidatingTTL   = 2 * time.Minute
	managedRechargePaidReviewTTL   = 10 * time.Minute
)

var (
	managedRechargeSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,63}$`)

	ErrManagedRechargeUnavailable  = infraerrors.New(http.StatusServiceUnavailable, "MANAGED_RECHARGE_UNAVAILABLE", "member recharge is not configured")
	ErrManagedRechargeNoInventory  = infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_OUT_OF_STOCK", "the selected product is out of stock")
	ErrManagedRechargeOrderMissing = infraerrors.New(http.StatusNotFound, "MANAGED_RECHARGE_ORDER_NOT_FOUND", "recharge order not found")
	ErrManagedRechargeProduct      = infraerrors.New(http.StatusBadRequest, "MANAGED_RECHARGE_PRODUCT_INVALID", "invalid recharge product")
)

type managedRechargeBalanceCache interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

type ManagedRechargeService struct {
	db           *sql.DB
	encryptor    SecretEncryptor
	balanceCache managedRechargeBalanceCache
	authCache    APIKeyAuthCacheInvalidator
	upstream     *managedRechargeUpstreamClient
	featureReady bool
}

type ManagedRechargeProduct struct {
	ID             int64     `json:"id"`
	Slug           string    `json:"slug"`
	PlanType       string    `json:"plan_type"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Price          float64   `json:"price"`
	Active         bool      `json:"active"`
	SortOrder      int       `json:"sort_order"`
	AvailableStock int       `json:"available_stock"`
	TotalStock     int       `json:"total_stock"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ManagedRechargeCatalog struct {
	Enabled  bool                     `json:"enabled"`
	Balance  float64                  `json:"balance"`
	Products []ManagedRechargeProduct `json:"products"`
}

type ManagedRechargeCDK struct {
	ID              int64      `json:"id"`
	ProductID       int64      `json:"product_id"`
	ProductName     string     `json:"product_name"`
	CodeMasked      string     `json:"code_masked"`
	Status          string     `json:"status"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	ReservedOrderID *int64     `json:"reserved_order_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	codeCiphertext  string
}

type ManagedRechargeOrder struct {
	ID                    int64      `json:"id"`
	OrderNo               string     `json:"order_no"`
	UserID                int64      `json:"user_id"`
	UserEmail             string     `json:"user_email,omitempty"`
	Username              string     `json:"username,omitempty"`
	ProductID             int64      `json:"product_id"`
	ProductSlug           string     `json:"product_slug"`
	ProductName           string     `json:"product_name"`
	CDKMasked             string     `json:"cdk_masked,omitempty"`
	Price                 float64    `json:"price"`
	Status                string     `json:"status"`
	AccountEmail          string     `json:"account_email"`
	UpstreamStatus        string     `json:"upstream_status,omitempty"`
	QueuePosition         int        `json:"queue_position,omitempty"`
	QueueTotal            int        `json:"queue_total,omitempty"`
	Progress              string     `json:"progress,omitempty"`
	ErrorCode             string     `json:"error_code,omitempty"`
	ErrorMessage          string     `json:"error_message,omitempty"`
	BalanceBefore         float64    `json:"balance_before,omitempty"`
	BalanceAfter          float64    `json:"balance_after,omitempty"`
	PaidAt                *time.Time `json:"paid_at,omitempty"`
	SubmittedAt           *time.Time `json:"submitted_at,omitempty"`
	CompletedAt           *time.Time `json:"completed_at,omitempty"`
	RefundedAt            *time.Time `json:"refunded_at,omitempty"`
	LastSyncedAt          *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	upstreamTaskID        string
	upstreamFailureReason string
	sessionCiphertext     string
	cdkCiphertext         string
}

type ManagedRechargeProductInput struct {
	Slug        string  `json:"slug"`
	PlanType    string  `json:"plan_type"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Active      bool    `json:"active"`
	SortOrder   int     `json:"sort_order"`
}

type ManagedRechargeCreateOrderInput struct {
	ProductID      int64
	Session        string
	IdempotencyKey string
}

type ManagedRechargeImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

func NewManagedRechargeService(
	db *sql.DB,
	encryptor SecretEncryptor,
	balanceCache managedRechargeBalanceCache,
	authCache APIKeyAuthCacheInvalidator,
) *ManagedRechargeService {
	return &ManagedRechargeService{
		db:           db,
		encryptor:    encryptor,
		balanceCache: balanceCache,
		authCache:    authCache,
		upstream:     newManagedRechargeUpstreamClient(),
		featureReady: db != nil && encryptor != nil,
	}
}

func (s *ManagedRechargeService) GetCatalog(ctx context.Context, userID int64) (*ManagedRechargeCatalog, error) {
	if s == nil || s.db == nil {
		return &ManagedRechargeCatalog{Enabled: false, Products: []ManagedRechargeProduct{}}, nil
	}
	var balance float64
	if err := s.db.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&balance); err != nil {
		return nil, fmt.Errorf("get managed recharge user balance: %w", err)
	}
	products, err := s.listProducts(ctx, true)
	if err != nil {
		return nil, err
	}
	return &ManagedRechargeCatalog{
		Enabled:  s.featureReady && len(products) > 0,
		Balance:  balance,
		Products: products,
	}, nil
}

func (s *ManagedRechargeService) ListProducts(ctx context.Context) ([]ManagedRechargeProduct, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	return s.listProducts(ctx, false)
}

func (s *ManagedRechargeService) listProducts(ctx context.Context, activeOnly bool) ([]ManagedRechargeProduct, error) {
	query := `
		SELECT p.id, p.slug, p.plan_type, p.name, p.description, p.price, p.active, p.sort_order,
		       COUNT(c.id) FILTER (
		           WHERE c.status = 'available' AND (c.expires_at IS NULL OR c.expires_at > NOW())
		       ) AS available_stock,
		       COUNT(c.id) AS total_stock,
		       p.created_at, p.updated_at
		FROM managed_recharge_products p
		LEFT JOIN managed_recharge_cdks c ON c.product_id = p.id
	`
	if activeOnly {
		query += ` WHERE p.active = TRUE`
	}
	query += ` GROUP BY p.id ORDER BY p.sort_order ASC, p.id ASC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list managed recharge products: %w", err)
	}
	defer func() { _ = rows.Close() }()

	products := make([]ManagedRechargeProduct, 0)
	for rows.Next() {
		var product ManagedRechargeProduct
		if err := rows.Scan(
			&product.ID, &product.Slug, &product.PlanType, &product.Name, &product.Description, &product.Price,
			&product.Active, &product.SortOrder, &product.AvailableStock, &product.TotalStock,
			&product.CreatedAt, &product.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan managed recharge product: %w", err)
		}
		products = append(products, product)
	}
	return products, rows.Err()
}

func (s *ManagedRechargeService) CreateProduct(ctx context.Context, input ManagedRechargeProductInput) (*ManagedRechargeProduct, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	input, err := normalizeManagedRechargeProduct(input)
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO managed_recharge_products (slug, plan_type, name, description, price, active, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, input.Slug, input.PlanType, input.Name, input.Description, input.Price, input.Active, input.SortOrder).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create managed recharge product: %w", err)
	}
	return s.getProduct(ctx, id)
}

func (s *ManagedRechargeService) UpdateProduct(ctx context.Context, id int64, input ManagedRechargeProductInput) (*ManagedRechargeProduct, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	input, err := normalizeManagedRechargeProduct(input)
	if err != nil {
		return nil, err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE managed_recharge_products
		SET slug = $2, plan_type = $3, name = $4, description = $5, price = $6, active = $7,
		    sort_order = $8, updated_at = NOW()
		WHERE id = $1
	`, id, input.Slug, input.PlanType, input.Name, input.Description, input.Price, input.Active, input.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("update managed recharge product: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return nil, ErrManagedRechargeProduct
	}
	return s.getProduct(ctx, id)
}

func (s *ManagedRechargeService) getProduct(ctx context.Context, id int64) (*ManagedRechargeProduct, error) {
	products, err := s.listProducts(ctx, false)
	if err != nil {
		return nil, err
	}
	for i := range products {
		if products[i].ID == id {
			return &products[i], nil
		}
	}
	return nil, ErrManagedRechargeProduct
}

func normalizeManagedRechargeProduct(input ManagedRechargeProductInput) (ManagedRechargeProductInput, error) {
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	input.PlanType = normalizeManagedRechargePlanType(input.PlanType)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if !managedRechargeSlugPattern.MatchString(input.Slug) || input.PlanType == "" || input.Name == "" || len(input.Name) > 128 || input.Price <= 0 {
		return input, ErrManagedRechargeProduct
	}
	if len(input.Description) > 2000 {
		return input, ErrManagedRechargeProduct
	}
	return input, nil
}

func (s *ManagedRechargeService) ImportCDKs(ctx context.Context, adminID, productID int64, codes []string, expiresAt *time.Time) (*ManagedRechargeImportResult, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	if productID <= 0 || len(codes) == 0 || len(codes) > managedRechargeImportMaxCodes {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_IMPORT_INVALID", "invalid CDK import")
	}
	if _, err := s.getProduct(ctx, productID); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin managed recharge CDK import: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result := &ManagedRechargeImportResult{}
	seen := make(map[string]struct{}, len(codes))
	for _, raw := range codes {
		code := strings.TrimSpace(raw)
		if code == "" || len(code) > 512 {
			result.Skipped++
			continue
		}
		hash := sha256.Sum256([]byte(code))
		hashString := hex.EncodeToString(hash[:])
		if _, ok := seen[hashString]; ok {
			result.Skipped++
			continue
		}
		seen[hashString] = struct{}{}

		ciphertext, err := s.encryptor.Encrypt(code)
		if err != nil {
			return nil, fmt.Errorf("encrypt managed recharge CDK: %w", err)
		}
		insertResult, err := tx.ExecContext(ctx, `
			INSERT INTO managed_recharge_cdks
			    (product_id, code_ciphertext, code_hash, code_masked, expires_at, created_by)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (code_hash) DO NOTHING
		`, productID, ciphertext, hashString, maskManagedRechargeCode(code), expiresAt, adminID)
		if err != nil {
			return nil, fmt.Errorf("insert managed recharge CDK: %w", err)
		}
		affected, _ := insertResult.RowsAffected()
		if affected == 1 {
			result.Imported++
		} else {
			result.Skipped++
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit managed recharge CDK import: %w", err)
	}
	return result, nil
}

func (s *ManagedRechargeService) ListCDKs(ctx context.Context, productID int64, status string, limit int) ([]ManagedRechargeCDK, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	query := `
		SELECT c.id, c.product_id, p.name, c.code_masked, c.status, c.expires_at,
		       c.reserved_order_id, c.created_at, c.updated_at
		FROM managed_recharge_cdks c
		JOIN managed_recharge_products p ON p.id = c.product_id
		WHERE ($1 = 0 OR c.product_id = $1)
		  AND ($2 = '' OR c.status = $2)
		ORDER BY c.id DESC
		LIMIT $3
	`
	rows, err := s.db.QueryContext(ctx, query, productID, strings.TrimSpace(status), limit)
	if err != nil {
		return nil, fmt.Errorf("list managed recharge CDKs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]ManagedRechargeCDK, 0)
	for rows.Next() {
		var item ManagedRechargeCDK
		var expires sql.NullTime
		var reserved sql.NullInt64
		if err := rows.Scan(
			&item.ID, &item.ProductID, &item.ProductName, &item.CodeMasked, &item.Status,
			&expires, &reserved, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan managed recharge CDK: %w", err)
		}
		if expires.Valid {
			item.ExpiresAt = &expires.Time
		}
		if reserved.Valid {
			item.ReservedOrderID = &reserved.Int64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *ManagedRechargeService) SetCDKStatus(ctx context.Context, id int64, status string) error {
	if err := s.requireReady(); err != nil {
		return err
	}
	status = strings.TrimSpace(status)
	if status != managedRechargeCDKAvailable && status != managedRechargeCDKDisabled && status != managedRechargeCDKInvalid {
		return infraerrors.BadRequest("MANAGED_RECHARGE_CDK_STATUS_INVALID", "invalid CDK status")
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE managed_recharge_cdks
		SET status = $2, updated_at = NOW()
		WHERE id = $1 AND reserved_order_id IS NULL AND status <> 'used'
	`, id, status)
	if err != nil {
		return fmt.Errorf("update managed recharge CDK status: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_CDK_LOCKED", "CDK is reserved or already used")
	}
	return nil
}

func (s *ManagedRechargeService) MoveCDK(ctx context.Context, id, productID int64) error {
	if err := s.requireReady(); err != nil {
		return err
	}
	if id <= 0 || productID <= 0 {
		return infraerrors.BadRequest("MANAGED_RECHARGE_CDK_MOVE_INVALID", "invalid CDK move")
	}
	if _, err := s.getProduct(ctx, productID); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE managed_recharge_cdks
		SET product_id = $2, updated_at = NOW()
		WHERE id = $1 AND status IN ('available', 'disabled', 'invalid')
		  AND reserved_order_id IS NULL
	`, id, productID)
	if err != nil {
		return fmt.Errorf("move managed recharge CDK: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_CDK_MOVE_BLOCKED", "reserved or used CDKs cannot be moved")
	}
	return nil
}

func (s *ManagedRechargeService) CreateOrder(ctx context.Context, userID int64, input ManagedRechargeCreateOrderInput) (*ManagedRechargeOrder, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.ProductID <= 0 || input.IdempotencyKey == "" || len(input.IdempotencyKey) > 128 {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_REQUEST_INVALID", "invalid recharge request")
	}
	session := strings.TrimSpace(input.Session)
	if session == "" || len(session) > managedRechargeSessionMaxBytes {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_SESSION_INVALID", "invalid Session payload")
	}
	accountEmail, err := parseManagedRechargeSession(session)
	if err != nil {
		return nil, err
	}
	if existing, err := s.getOrderByIdempotency(ctx, userID, input.IdempotencyKey); err == nil {
		return existing, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	orderCtx, cancelOrder := context.WithTimeout(context.Background(), managedRechargeOrderTimeout)
	defer cancelOrder()

	product, err := s.getActiveProduct(orderCtx, input.ProductID)
	if err != nil {
		return nil, err
	}
	sessionCiphertext, err := s.encryptor.Encrypt(session)
	if err != nil {
		return nil, fmt.Errorf("encrypt managed recharge Session: %w", err)
	}

	orderNo, err := newManagedRechargeOrderNo()
	if err != nil {
		return nil, fmt.Errorf("generate managed recharge order number: %w", err)
	}
	var orderID int64
	err = s.db.QueryRowContext(orderCtx, `
		INSERT INTO managed_recharge_orders
		    (order_no, user_id, product_id, idempotency_key, price, account_email, session_ciphertext)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, idempotency_key) DO NOTHING
		RETURNING id
	`, orderNo, userID, product.ID, input.IdempotencyKey, product.Price, accountEmail, sessionCiphertext).Scan(&orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return s.getOrderByIdempotency(orderCtx, userID, input.IdempotencyKey)
	}
	if err != nil {
		return nil, fmt.Errorf("create managed recharge order: %w", err)
	}
	cleanupUnpaid := true
	defer func() {
		if cleanupUnpaid {
			_ = s.runRecovery(func(recoveryCtx context.Context) error {
				return s.failUnpaidOrder(recoveryCtx, orderID, "ORDER_INTERRUPTED", "订单处理被中断，请重新提交", false)
			})
		}
	}()

	var reserved *ManagedRechargeCDK
	var plaintextCode string
	for attempt := 0; attempt < managedRechargeReserveAttempts; attempt++ {
		reserved, err = s.reserveNextCDK(orderCtx, orderID, product.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				if recoveryErr := s.runRecovery(func(recoveryCtx context.Context) error {
					return s.failUnpaidOrder(recoveryCtx, orderID, "OUT_OF_STOCK", "当前套餐暂时缺货", false)
				}); recoveryErr != nil {
					return nil, recoveryErr
				}
				return nil, ErrManagedRechargeNoInventory
			}
			return nil, err
		}
		plaintextCode, err = s.encryptor.Decrypt(reserved.codeCiphertext)
		if err != nil {
			if recoveryErr := s.runRecovery(func(recoveryCtx context.Context) error {
				return s.quarantineReservedCDK(recoveryCtx, orderID, "CDK_DECRYPT_FAILED")
			}); recoveryErr != nil {
				return nil, recoveryErr
			}
			continue
		}
		verified, verifyErr := s.upstream.verifyCDK(orderCtx, plaintextCode)
		if verifyErr != nil {
			if recoveryErr := s.runRecovery(func(recoveryCtx context.Context) error {
				return s.failUnpaidOrder(recoveryCtx, orderID, "UPSTREAM_UNAVAILABLE", "上游验证暂时不可用", false)
			}); recoveryErr != nil {
				return nil, recoveryErr
			}
			return nil, infraerrors.New(http.StatusBadGateway, "MANAGED_RECHARGE_UPSTREAM_UNAVAILABLE", "recharge provider is temporarily unavailable")
		}
		if verified.Valid {
			actualPlanType := normalizeManagedRechargePlanType(verified.PlanType)
			if actualPlanType == product.PlanType {
				break
			}
			if recoveryErr := s.runRecovery(func(recoveryCtx context.Context) error {
				return s.reassignMismatchedCDK(recoveryCtx, orderID, product.PlanType, actualPlanType)
			}); recoveryErr != nil {
				return nil, recoveryErr
			}
			reserved = nil
			plaintextCode = ""
			continue
		}
		if recoveryErr := s.runRecovery(func(recoveryCtx context.Context) error {
			return s.quarantineReservedCDK(recoveryCtx, orderID, "CDK_INVALID")
		}); recoveryErr != nil {
			return nil, recoveryErr
		}
		reserved = nil
		plaintextCode = ""
	}
	if reserved == nil || plaintextCode == "" {
		if recoveryErr := s.runRecovery(func(recoveryCtx context.Context) error {
			return s.failUnpaidOrder(recoveryCtx, orderID, "OUT_OF_STOCK", "当前套餐暂时缺货", false)
		}); recoveryErr != nil {
			return nil, recoveryErr
		}
		return nil, ErrManagedRechargeNoInventory
	}

	balanceBefore, balanceAfter, err := s.chargeOrder(orderCtx, orderID, userID, product.Price)
	if err != nil {
		if recoveryErr := s.runRecovery(func(recoveryCtx context.Context) error {
			return s.failUnpaidOrder(recoveryCtx, orderID, "PAYMENT_FAILED", "余额支付失败", false)
		}); recoveryErr != nil {
			return nil, recoveryErr
		}
		if infraerrors.Reason(err) == ErrInsufficientBalance.Reason {
			return nil, ErrInsufficientBalance.WithMetadata(balancePurchaseErrorMetadata(balanceBefore, product.Price))
		}
		return nil, err
	}
	cleanupUnpaid = false
	s.invalidateBalanceCaches(userID)

	fulfillCtx, cancelFulfill := context.WithTimeout(context.Background(), managedRechargeFulfillTimeout)
	defer cancelFulfill()
	created, createErr := s.upstream.createTask(fulfillCtx, plaintextCode, session)
	if createErr != nil {
		_ = s.markManualReview(fulfillCtx, orderID, "UPSTREAM_CREATE_UNCERTAIN", "上游提交结果不确定，已转人工核对")
		return s.GetOrder(fulfillCtx, userID, orderID, false)
	}
	if strings.TrimSpace(created.TaskID) == "" {
		_ = s.refundOrder(fulfillCtx, orderID, userID, "UPSTREAM_CREATE_REJECTED", "充值任务未被受理，余额已退回")
		return s.GetOrder(fulfillCtx, userID, orderID, false)
	}
	if err := s.markTaskCreated(fulfillCtx, orderID, created.TaskID); err != nil {
		return nil, err
	}

	confirmed, confirmErr := s.upstream.confirmTask(fulfillCtx, created.TaskID)
	if confirmErr != nil {
		_ = s.markManualReview(fulfillCtx, orderID, "UPSTREAM_CONFIRM_UNCERTAIN", "上游确认结果不确定，已转人工核对")
		return s.GetOrder(fulfillCtx, userID, orderID, false)
	}
	status := normalizeManagedRechargeAcceptedStatus(confirmed.Status)
	if status == "" {
		_ = s.markManualReview(fulfillCtx, orderID, "UPSTREAM_CONFIRM_UNCERTAIN", "上游确认结果不确定，已转人工核对")
		return s.GetOrder(fulfillCtx, userID, orderID, false)
	}
	updateResult, err := s.db.ExecContext(fulfillCtx, `
		UPDATE managed_recharge_orders
		SET status = $2, upstream_status = $3, submitted_at = NOW(), last_synced_at = NOW(),
		    error_code = '', error_message = '', updated_at = NOW()
		WHERE id = $1 AND status NOT IN ('refunded', 'completed')
	`, orderID, status, confirmed.Status)
	if err != nil {
		return nil, fmt.Errorf("mark managed recharge order submitted: %w", err)
	}
	if affected, _ := updateResult.RowsAffected(); affected == 0 {
		return s.GetOrder(fulfillCtx, userID, orderID, false)
	}
	_ = balanceAfter
	return s.GetOrder(fulfillCtx, userID, orderID, false)
}

func (s *ManagedRechargeService) GetOrder(ctx context.Context, userID, orderID int64, forceSync bool) (*ManagedRechargeOrder, error) {
	order, err := s.getOrder(ctx, orderID, &userID)
	if err != nil {
		return nil, err
	}
	return s.syncOrderIfNeeded(ctx, order, forceSync)
}

func (s *ManagedRechargeService) AdminGetOrder(ctx context.Context, orderID int64, forceSync bool) (*ManagedRechargeOrder, error) {
	order, err := s.getOrder(ctx, orderID, nil)
	if err != nil {
		return nil, err
	}
	return s.syncOrderIfNeeded(ctx, order, forceSync)
}

func (s *ManagedRechargeService) ListUserOrders(ctx context.Context, userID int64, limit int) ([]ManagedRechargeOrder, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.listOrders(ctx, `WHERE o.user_id = $1`, []any{userID, limit}, `$2`)
}

func (s *ManagedRechargeService) ListAdminOrders(ctx context.Context, status string, limit int) ([]ManagedRechargeOrder, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.listOrders(ctx, `WHERE ($1 = '' OR o.status = $1)`, []any{strings.TrimSpace(status), limit}, `$2`)
}

func (s *ManagedRechargeService) listOrders(ctx context.Context, where string, args []any, limitPlaceholder string) ([]ManagedRechargeOrder, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, managedRechargeOrderSelect+` `+where+` ORDER BY o.id DESC LIMIT `+limitPlaceholder, args...)
	if err != nil {
		return nil, fmt.Errorf("list managed recharge orders: %w", err)
	}
	defer func() { _ = rows.Close() }()
	orders := make([]ManagedRechargeOrder, 0)
	for rows.Next() {
		order, err := scanManagedRechargeOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, *order)
	}
	return orders, rows.Err()
}

func (s *ManagedRechargeService) SubmitReplacementSession(ctx context.Context, userID, orderID int64, session string) (*ManagedRechargeOrder, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	session = strings.TrimSpace(session)
	if session == "" || len(session) > managedRechargeSessionMaxBytes {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_SESSION_INVALID", "invalid Session payload")
	}
	email, err := parseManagedRechargeSession(session)
	if err != nil {
		return nil, err
	}
	order, err := s.getOrder(ctx, orderID, &userID)
	if err != nil {
		return nil, err
	}
	if order.Status != ManagedRechargeStatusActionRequired {
		return nil, infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_SESSION_NOT_REQUIRED", "this order does not require a new Session")
	}
	if order.AccountEmail != "" && !strings.EqualFold(order.AccountEmail, email) {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_ACCOUNT_MISMATCH", "Session account does not match this order")
	}
	code, err := s.encryptor.Decrypt(order.cdkCiphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt managed recharge CDK: %w", err)
	}
	ciphertext, err := s.encryptor.Encrypt(session)
	if err != nil {
		return nil, fmt.Errorf("encrypt replacement Session: %w", err)
	}
	updateResult, err := s.db.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET session_ciphertext = $2, account_email = $3, status = 'verifying',
		    error_code = '', error_message = '', updated_at = NOW()
		WHERE id = $1 AND status = 'action_required'
	`, orderID, ciphertext, email)
	if err != nil {
		return nil, fmt.Errorf("store replacement Session: %w", err)
	}
	if affected, _ := updateResult.RowsAffected(); affected == 0 {
		return nil, infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_SESSION_NOT_REQUIRED", "this order does not require a new Session")
	}
	fulfillCtx, cancelFulfill := context.WithTimeout(context.Background(), managedRechargeFulfillTimeout)
	defer cancelFulfill()
	result, err := s.upstream.submitReplacementSession(fulfillCtx, code, session)
	if err != nil {
		_ = s.markManualReview(fulfillCtx, orderID, "REPLACEMENT_SESSION_UNCERTAIN", "新 Session 提交结果不确定，已转人工核对")
		return s.GetOrder(fulfillCtx, userID, orderID, false)
	}
	status := ManagedRechargeStatusVerifying
	switch result.PostProcessStatus {
	case "action_required":
		status = ManagedRechargeStatusActionRequired
	case "manual_review":
		status = ManagedRechargeStatusManualReview
	}
	if _, err := s.db.ExecContext(fulfillCtx, `
		UPDATE managed_recharge_orders
		SET status = $2, session_ciphertext = '', last_synced_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, orderID, status); err != nil {
		return nil, fmt.Errorf("update replacement Session status: %w", err)
	}
	return s.GetOrder(fulfillCtx, userID, orderID, false)
}

func (s *ManagedRechargeService) AdminRefundOrder(ctx context.Context, adminID, orderID int64) (*ManagedRechargeOrder, error) {
	order, err := s.AdminGetOrder(ctx, orderID, true)
	if err != nil {
		return nil, err
	}
	if order.Status == ManagedRechargeStatusRefunded {
		return order, nil
	}
	if order.Status == ManagedRechargeStatusCompleted || strings.EqualFold(order.UpstreamStatus, "completed") {
		return nil, infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_COMPLETED", "completed recharges cannot be refunded")
	}
	if order.PaidAt == nil {
		return nil, infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_NOT_PAID", "only paid orders can be refunded")
	}
	if order.Status != ManagedRechargeStatusManualReview {
		return nil, infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_REFUND_REVIEW_REQUIRED", "only manual-review orders can be refunded")
	}
	if order.ErrorCode != "UPSTREAM_TASK_NOT_FOUND" || time.Since(*order.PaidAt) < managedRechargePaidReviewTTL {
		return nil, infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_REFUND_SYNC_REQUIRED", "provider must confirm that no task exists before refunding")
	}
	if err := s.refundOrder(ctx, orderID, order.UserID, "ADMIN_REFUND", "管理员已退款"); err != nil {
		return nil, err
	}
	_ = s.appendEvent(ctx, orderID, &adminID, "ADMIN_REFUND", map[string]any{})
	return s.getOrder(ctx, orderID, nil)
}

func (s *ManagedRechargeService) syncOrderIfNeeded(ctx context.Context, order *ManagedRechargeOrder, force bool) (*ManagedRechargeOrder, error) {
	if order == nil {
		return order, nil
	}
	if order.Status == ManagedRechargeStatusValidating {
		if time.Since(order.UpdatedAt) < managedRechargeValidatingTTL {
			return order, nil
		}
		if err := s.runRecovery(func(recoveryCtx context.Context) error {
			return s.failUnpaidOrder(recoveryCtx, order.ID, "ORDER_INTERRUPTED", "订单未完成扣款，请重新提交", false)
		}); err != nil {
			return nil, err
		}
		return s.getOrder(ctx, order.ID, &order.UserID)
	}
	if !managedRechargeStatusNeedsSync(order.Status) || order.cdkCiphertext == "" {
		return order, nil
	}
	if !force && order.LastSyncedAt != nil && time.Since(*order.LastSyncedAt) < managedRechargeSyncInterval {
		return order, nil
	}
	syncCtx, cancelSync := context.WithTimeout(context.Background(), managedRechargeFulfillTimeout)
	defer cancelSync()
	code, err := s.encryptor.Decrypt(order.cdkCiphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt managed recharge CDK for sync: %w", err)
	}
	result, err := s.upstream.lookupTask(syncCtx, code)
	if err != nil {
		errorCode := "SYNC_UNAVAILABLE"
		errorMessage := "状态同步暂时不可用"
		var upstreamHTTPError *managedRechargeUpstreamHTTPError
		if errors.As(err, &upstreamHTTPError) && upstreamHTTPError.StatusCode == http.StatusNotFound {
			errorCode = "UPSTREAM_TASK_NOT_FOUND"
			errorMessage = "暂未查询到履约任务，正在人工核对"
		}
		_, _ = s.db.ExecContext(syncCtx, `
			UPDATE managed_recharge_orders
			SET last_synced_at = NOW(), error_code = $2,
			    error_message = $3, updated_at = NOW()
			WHERE id = $1
		`, order.ID, errorCode, errorMessage)
		if order.Status == ManagedRechargeStatusPaid && time.Since(order.UpdatedAt) >= managedRechargePaidReviewTTL {
			_ = s.markManualReview(syncCtx, order.ID, "UPSTREAM_CREATE_UNCERTAIN", "上游提交结果不确定，已转人工核对")
		}
		return s.getOrder(syncCtx, order.ID, &order.UserID)
	}
	if strings.EqualFold(strings.TrimSpace(result.TaskStatus), "pending") {
		taskID := strings.TrimSpace(result.TaskID)
		if taskID == "" {
			taskID = strings.TrimSpace(order.upstreamTaskID)
		}
		confirmedAccepted := false
		if taskID != "" {
			confirmed, confirmErr := s.upstream.confirmTask(syncCtx, taskID)
			if confirmErr == nil {
				if accepted := normalizeManagedRechargeAcceptedStatus(confirmed.Status); accepted != "" {
					result.TaskID = taskID
					result.TaskStatus = confirmed.Status
					confirmedAccepted = true
				}
			}
		}
		if !confirmedAccepted {
			_ = s.markManualReview(syncCtx, order.ID, "UPSTREAM_CONFIRM_UNCERTAIN", "上游确认结果不确定，已转人工核对")
			return s.getOrder(syncCtx, order.ID, &order.UserID)
		}
	}

	nextStatus := normalizeManagedRechargeLookupStatus(result)
	if nextStatus == ManagedRechargeStatusFailed {
		if managedRechargeFailureNeedsManualReview(result.FailureReason) {
			_ = s.markManualReview(syncCtx, order.ID, "PAYMENT_RESULT_UNCERTAIN", "支付结果需要人工核对")
		} else {
			_ = s.refundOrder(syncCtx, order.ID, order.UserID, "UPSTREAM_FAILED", normalizeManagedRechargeFailure(result.FailureReason))
		}
		return s.getOrder(syncCtx, order.ID, &order.UserID)
	}

	fulfillmentCompleted := strings.EqualFold(strings.TrimSpace(result.TaskStatus), "completed")
	query := `
		UPDATE managed_recharge_orders
		SET status = $2, upstream_status = $3, upstream_failure_reason = $4,
		    queue_position = $5, queue_total = $6, progress = $7,
		    account_email = CASE WHEN $8 = '' THEN account_email ELSE $8 END,
		    error_code = '', error_message = '', last_synced_at = NOW(), updated_at = NOW(),
		    completed_at = CASE WHEN $9 THEN COALESCE(completed_at, NOW()) ELSE completed_at END,
		    session_ciphertext = CASE WHEN $9 THEN '' ELSE session_ciphertext END,
		    upstream_task_id = CASE WHEN $10 = '' THEN upstream_task_id ELSE $10 END
		WHERE id = $1 AND status NOT IN ('refunded', 'completed')
	`
	tx, err := s.db.BeginTx(syncCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin managed recharge sync: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	updateResult, err := tx.ExecContext(syncCtx, query, order.ID, nextStatus, result.TaskStatus, result.FailureReason,
		result.QueuePosition, result.QueueTotal, result.Progress, result.AccountEmail, fulfillmentCompleted, result.TaskID)
	if err != nil {
		return nil, fmt.Errorf("sync managed recharge order: %w", err)
	}
	affected, _ := updateResult.RowsAffected()
	if affected == 1 && fulfillmentCompleted {
		if _, err := tx.ExecContext(syncCtx, `
			UPDATE managed_recharge_cdks
			SET status = 'used', reserved_at = NULL, updated_at = NOW()
			WHERE id = (SELECT cdk_id FROM managed_recharge_orders WHERE id = $1) AND status = 'reserved'
		`, order.ID); err != nil {
			return nil, fmt.Errorf("mark managed recharge CDK used: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit managed recharge sync: %w", err)
	}
	return s.getOrder(syncCtx, order.ID, &order.UserID)
}

func (s *ManagedRechargeService) reserveNextCDK(ctx context.Context, orderID, productID int64) (*ManagedRechargeCDK, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin managed recharge inventory reservation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var item ManagedRechargeCDK
	var expiresAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT id, product_id, code_ciphertext, code_masked, status, expires_at, created_at, updated_at
		FROM managed_recharge_cdks
		WHERE product_id = $1 AND status = 'available'
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY expires_at ASC NULLS LAST, id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, productID).Scan(
		&item.ID, &item.ProductID, &item.codeCiphertext, &item.CodeMasked, &item.Status,
		&expiresAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_cdks
		SET status = 'reserved', reserved_order_id = $2, reserved_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, item.ID, orderID); err != nil {
		return nil, fmt.Errorf("reserve managed recharge CDK: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE managed_recharge_orders SET cdk_id = $2, updated_at = NOW() WHERE id = $1`, orderID, item.ID); err != nil {
		return nil, fmt.Errorf("attach managed recharge CDK to order: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit managed recharge inventory reservation: %w", err)
	}
	item.Status = managedRechargeCDKReserved
	return &item, nil
}

func (s *ManagedRechargeService) chargeOrder(ctx context.Context, orderID, userID int64, price float64) (float64, float64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("begin managed recharge payment: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var balanceAfter float64
	err = tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance - $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL AND balance >= $2
		RETURNING balance
	`, userID, price).Scan(&balanceAfter)
	if errors.Is(err, sql.ErrNoRows) {
		var current float64
		if getErr := tx.QueryRowContext(ctx, `SELECT balance FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&current); getErr != nil {
			return 0, 0, ErrUserNotFound
		}
		return current, current, ErrInsufficientBalance
	}
	if err != nil {
		return 0, 0, fmt.Errorf("deduct managed recharge balance: %w", err)
	}
	balanceBefore := balanceAfter + price
	orderUpdate, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET status = 'paid', balance_before = $2, balance_after = $3, paid_at = NOW(),
		    error_code = '', error_message = '', updated_at = NOW()
		WHERE id = $1 AND status = 'validating'
	`, orderID, balanceBefore, balanceAfter)
	if err != nil {
		return 0, 0, fmt.Errorf("mark managed recharge order paid: %w", err)
	}
	if affected, _ := orderUpdate.RowsAffected(); affected != 1 {
		return 0, 0, fmt.Errorf("managed recharge order is not payable")
	}
	if err := appendManagedRechargeEventTx(ctx, tx, orderID, nil, "BALANCE_PAID", map[string]any{
		"amount": price,
	}); err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("commit managed recharge payment: %w", err)
	}
	return balanceBefore, balanceAfter, nil
}

func (s *ManagedRechargeService) failUnpaidOrder(ctx context.Context, orderID int64, code, message string, quarantineCDK bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	cdkStatus := managedRechargeCDKAvailable
	if quarantineCDK {
		cdkStatus = managedRechargeCDKInvalid
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_cdks
		SET status = $2, reserved_order_id = NULL, reserved_at = NULL, updated_at = NOW()
		WHERE reserved_order_id = $1 AND status = 'reserved'
	`, orderID, cdkStatus); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET status = 'failed', cdk_id = NULL, session_ciphertext = '', error_code = $2,
		    error_message = $3, updated_at = NOW()
		WHERE id = $1 AND paid_at IS NULL
	`, orderID, code, message); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *ManagedRechargeService) quarantineReservedCDK(ctx context.Context, orderID int64, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_cdks
		SET status = 'invalid', reserved_order_id = NULL, reserved_at = NULL, updated_at = NOW()
		WHERE reserved_order_id = $1 AND status = 'reserved'
	`, orderID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET cdk_id = NULL, error_code = $2, error_message = '库存 CDK 已隔离，正在尝试下一枚',
		    updated_at = NOW()
		WHERE id = $1 AND paid_at IS NULL
	`, orderID, reason); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *ManagedRechargeService) reassignMismatchedCDK(ctx context.Context, orderID int64, expectedPlanType, actualPlanType string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var cdkID, currentProductID int64
	if err := tx.QueryRowContext(ctx, `
		SELECT id, product_id
		FROM managed_recharge_cdks
		WHERE reserved_order_id = $1 AND status = 'reserved'
		FOR UPDATE
	`, orderID).Scan(&cdkID, &currentProductID); err != nil {
		return err
	}

	targetProductID := currentProductID
	targetStatus := managedRechargeCDKDisabled
	if actualPlanType != "" {
		var detectedProductID int64
		err := tx.QueryRowContext(ctx, `
			SELECT id
			FROM managed_recharge_products
			WHERE plan_type = $1
			LIMIT 1
		`, actualPlanType).Scan(&detectedProductID)
		if err == nil {
			targetProductID = detectedProductID
			targetStatus = managedRechargeCDKAvailable
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_cdks
		SET product_id = $2, status = $3, reserved_order_id = NULL, reserved_at = NULL, updated_at = NOW()
		WHERE id = $1
	`, cdkID, targetProductID, targetStatus); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET cdk_id = NULL, error_code = 'CDK_PLAN_MISMATCH',
		    error_message = '库存套餐类型不匹配，正在尝试下一枚', updated_at = NOW()
		WHERE id = $1 AND paid_at IS NULL
	`, orderID); err != nil {
		return err
	}
	if err := appendManagedRechargeEventTx(ctx, tx, orderID, nil, "CDK_PLAN_REASSIGNED", map[string]any{
		"actual_plan_type":   actualPlanType,
		"expected_plan_type": expectedPlanType,
		"target_product_id":  targetProductID,
		"target_status":      targetStatus,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *ManagedRechargeService) refundOrder(ctx context.Context, orderID, userID int64, code, message string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin managed recharge refund: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var price float64
	var status string
	var paidAt sql.NullTime
	var orderUserID int64
	var upstreamStatus string
	err = tx.QueryRowContext(ctx, `SELECT price, status, paid_at, user_id, upstream_status FROM managed_recharge_orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&price, &status, &paidAt, &orderUserID, &upstreamStatus)
	if err != nil {
		return err
	}
	if status == ManagedRechargeStatusRefunded {
		return nil
	}
	if status == ManagedRechargeStatusCompleted {
		return infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_COMPLETED", "completed orders cannot be refunded automatically")
	}
	if strings.EqualFold(upstreamStatus, "completed") {
		return infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_COMPLETED", "completed recharges cannot be refunded")
	}
	if orderUserID != userID {
		return fmt.Errorf("managed recharge refund user mismatch")
	}
	if paidAt.Valid {
		if _, err := tx.ExecContext(ctx, `UPDATE users SET balance = balance + $2, updated_at = NOW() WHERE id = $1`, userID, price); err != nil {
			return fmt.Errorf("refund managed recharge balance: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_cdks
		SET status = 'invalid', reserved_order_id = NULL, reserved_at = NULL, updated_at = NOW()
		WHERE reserved_order_id = $1 AND status = 'reserved'
	`, orderID); err != nil {
		return fmt.Errorf("release managed recharge CDK: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET status = 'refunded', session_ciphertext = '', error_code = $2, error_message = $3,
		    refunded_at = COALESCE(refunded_at, NOW()), updated_at = NOW()
		WHERE id = $1
	`, orderID, code, message); err != nil {
		return fmt.Errorf("mark managed recharge order refunded: %w", err)
	}
	if err := appendManagedRechargeEventTx(ctx, tx, orderID, nil, "BALANCE_REFUNDED", map[string]any{"amount": price}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit managed recharge refund: %w", err)
	}
	s.invalidateBalanceCaches(userID)
	return nil
}

func (s *ManagedRechargeService) markTaskCreated(ctx context.Context, orderID int64, taskID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET status = 'submitting', upstream_task_id = $2, upstream_status = 'created', updated_at = NOW()
		WHERE id = $1 AND status = 'paid'
	`, orderID, taskID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("managed recharge order is no longer awaiting submission")
	}
	return nil
}

func (s *ManagedRechargeService) markManualReview(ctx context.Context, orderID int64, code, message string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET status = 'manual_review', error_code = $2, error_message = $3,
		    last_synced_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status NOT IN ('completed', 'refunded')
	`, orderID, code, message)
	return err
}

func (s *ManagedRechargeService) invalidateBalanceCaches(userID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if s.balanceCache != nil {
		_ = s.balanceCache.InvalidateUserBalance(ctx, userID)
	}
	if s.authCache != nil {
		s.authCache.InvalidateAuthCacheByUserID(ctx, userID)
	}
}

func (s *ManagedRechargeService) runRecovery(operation func(context.Context) error) error {
	recoveryCtx, cancel := context.WithTimeout(context.Background(), managedRechargeRecoveryTimeout)
	defer cancel()
	return operation(recoveryCtx)
}

func (s *ManagedRechargeService) getActiveProduct(ctx context.Context, id int64) (*ManagedRechargeProduct, error) {
	var product ManagedRechargeProduct
	err := s.db.QueryRowContext(ctx, `
		SELECT id, slug, plan_type, name, description, price, active, sort_order, created_at, updated_at
		FROM managed_recharge_products WHERE id = $1 AND active = TRUE
	`, id).Scan(&product.ID, &product.Slug, &product.PlanType, &product.Name, &product.Description, &product.Price,
		&product.Active, &product.SortOrder, &product.CreatedAt, &product.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrManagedRechargeProduct
	}
	if err != nil {
		return nil, fmt.Errorf("get active managed recharge product: %w", err)
	}
	return &product, nil
}

func (s *ManagedRechargeService) getOrderByIdempotency(ctx context.Context, userID int64, key string) (*ManagedRechargeOrder, error) {
	row := s.db.QueryRowContext(ctx, managedRechargeOrderSelect+` WHERE o.user_id = $1 AND o.idempotency_key = $2`, userID, key)
	return scanManagedRechargeOrder(row)
}

func (s *ManagedRechargeService) getOrder(ctx context.Context, orderID int64, userID *int64) (*ManagedRechargeOrder, error) {
	query := managedRechargeOrderSelect + ` WHERE o.id = $1`
	args := []any{orderID}
	if userID != nil {
		query += ` AND o.user_id = $2`
		args = append(args, *userID)
	}
	order, err := scanManagedRechargeOrder(s.db.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrManagedRechargeOrderMissing
	}
	return order, err
}

const managedRechargeOrderSelect = `
	SELECT o.id, o.order_no, o.user_id, COALESCE(u.email, ''), COALESCE(u.username, ''),
	       o.product_id, p.slug, p.name, COALESCE(c.code_masked, ''), o.price, o.status,
	       o.account_email, o.upstream_task_id, o.upstream_status, o.upstream_failure_reason,
	       o.queue_position, o.queue_total, o.progress, o.error_code, o.error_message,
	       COALESCE(o.balance_before, 0), COALESCE(o.balance_after, 0),
	       o.paid_at, o.submitted_at, o.completed_at, o.refunded_at, o.last_synced_at,
	       o.created_at, o.updated_at, o.session_ciphertext, COALESCE(c.code_ciphertext, '')
	FROM managed_recharge_orders o
	JOIN managed_recharge_products p ON p.id = o.product_id
	JOIN users u ON u.id = o.user_id
	LEFT JOIN managed_recharge_cdks c ON c.id = o.cdk_id
`

type managedRechargeRowScanner interface {
	Scan(dest ...any) error
}

func scanManagedRechargeOrder(scanner managedRechargeRowScanner) (*ManagedRechargeOrder, error) {
	var order ManagedRechargeOrder
	var paidAt, submittedAt, completedAt, refundedAt, lastSyncedAt sql.NullTime
	err := scanner.Scan(
		&order.ID, &order.OrderNo, &order.UserID, &order.UserEmail, &order.Username,
		&order.ProductID, &order.ProductSlug, &order.ProductName, &order.CDKMasked, &order.Price,
		&order.Status, &order.AccountEmail, &order.upstreamTaskID, &order.UpstreamStatus,
		&order.upstreamFailureReason, &order.QueuePosition, &order.QueueTotal, &order.Progress,
		&order.ErrorCode, &order.ErrorMessage, &order.BalanceBefore, &order.BalanceAfter,
		&paidAt, &submittedAt, &completedAt, &refundedAt, &lastSyncedAt,
		&order.CreatedAt, &order.UpdatedAt, &order.sessionCiphertext, &order.cdkCiphertext,
	)
	if err != nil {
		return nil, err
	}
	order.PaidAt = nullTimePointer(paidAt)
	order.SubmittedAt = nullTimePointer(submittedAt)
	order.CompletedAt = nullTimePointer(completedAt)
	order.RefundedAt = nullTimePointer(refundedAt)
	order.LastSyncedAt = nullTimePointer(lastSyncedAt)
	return &order, nil
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func (s *ManagedRechargeService) appendEvent(ctx context.Context, orderID int64, actorID *int64, eventType string, payload map[string]any) error {
	return appendManagedRechargeEventTx(ctx, s.db, orderID, actorID, eventType, payload)
}

type managedRechargeExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func appendManagedRechargeEventTx(ctx context.Context, execer managedRechargeExecer, orderID int64, actorID *int64, eventType string, payload map[string]any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode managed recharge event: %w", err)
	}
	_, err = execer.ExecContext(ctx, `
		INSERT INTO managed_recharge_events (order_id, actor_user_id, event_type, payload)
		VALUES ($1, $2, $3, $4::jsonb)
	`, orderID, actorID, eventType, string(encoded))
	return err
}

func (s *ManagedRechargeService) requireReady() error {
	if s == nil || !s.featureReady || s.db == nil || s.encryptor == nil || s.upstream == nil {
		return ErrManagedRechargeUnavailable
	}
	return nil
}

func parseManagedRechargeSession(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if match := regexp.MustCompile(`^(.+?)----(https?://.+)$`).FindStringSubmatch(trimmed); len(match) == 3 {
		email := strings.TrimSpace(match[1])
		if strings.Contains(email, "@") && len(email) <= 255 {
			return email, nil
		}
	}
	var payload struct {
		AccessToken string `json:"accessToken"`
		User        struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"user"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil || strings.TrimSpace(payload.AccessToken) == "" {
		return "", infraerrors.BadRequest("MANAGED_RECHARGE_SESSION_INVALID", "Session JSON is invalid or incomplete")
	}
	email := strings.TrimSpace(payload.User.Email)
	if email == "" {
		email = strings.TrimSpace(payload.User.Name)
	}
	if email == "" || len(email) > 255 {
		return "", infraerrors.BadRequest("MANAGED_RECHARGE_SESSION_INVALID", "Session does not contain an account email")
	}
	return email, nil
}

func normalizeManagedRechargePlanType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)
	switch normalized {
	case "plus", "chatgptplus":
		return "plus"
	case "pro", "chatgptpro":
		return "pro"
	default:
		return ""
	}
}

func normalizeManagedRechargeAcceptedStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "queued", "submitted":
		return ManagedRechargeStatusQueued
	case "processing":
		return ManagedRechargeStatusProcessing
	default:
		return ""
	}
}

func normalizeManagedRechargeLookupStatus(result *managedRechargeLookupResponse) string {
	if result == nil {
		return ManagedRechargeStatusProcessing
	}
	switch strings.ToLower(strings.TrimSpace(result.TaskStatus)) {
	case "pending", "queued", "submitted":
		return ManagedRechargeStatusQueued
	case "processing":
		return ManagedRechargeStatusProcessing
	case "completed":
		switch strings.ToLower(strings.TrimSpace(result.PostProcessStatus)) {
		case "action_required":
			return ManagedRechargeStatusActionRequired
		case "manual_review":
			return ManagedRechargeStatusManualReview
		case "pending", "processing", "retrying":
			return ManagedRechargeStatusVerifying
		default:
			return ManagedRechargeStatusCompleted
		}
	case "failed":
		return ManagedRechargeStatusFailed
	default:
		return ManagedRechargeStatusProcessing
	}
}

func managedRechargeStatusNeedsSync(status string) bool {
	switch status {
	case ManagedRechargeStatusPaid, ManagedRechargeStatusSubmitting, ManagedRechargeStatusQueued,
		ManagedRechargeStatusProcessing, ManagedRechargeStatusVerifying,
		ManagedRechargeStatusActionRequired, ManagedRechargeStatusManualReview:
		return true
	default:
		return false
	}
}

func managedRechargeFailureNeedsManualReview(reason string) bool {
	normalized := strings.ToLower(reason)
	return strings.Contains(normalized, "payment_submitted_uncertain") ||
		strings.Contains(normalized, "uncertain") ||
		strings.Contains(normalized, "paid")
}

func normalizeManagedRechargeFailure(reason string) string {
	normalized := strings.ToLower(reason)
	switch {
	case strings.Contains(normalized, "session"):
		return "Session 已失效，余额已退回"
	case strings.Contains(normalized, "existing_entitlement"), strings.Contains(normalized, "subscription"):
		return "账号当前订阅状态不符合充值条件，余额已退回"
	case strings.Contains(normalized, "account"):
		return "账号暂时不符合充值条件，余额已退回"
	default:
		return "充值未完成，余额已退回"
	}
}

func maskManagedRechargeCode(code string) string {
	code = strings.TrimSpace(code)
	if len(code) <= 8 {
		return "****"
	}
	return code[:4] + "..." + code[len(code)-4:]
}

func newManagedRechargeOrderNo() (string, error) {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return fmt.Sprintf("MR%s%s", time.Now().UTC().Format("20060102150405"), strings.ToUpper(hex.EncodeToString(random))), nil
}
