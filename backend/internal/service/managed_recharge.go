package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	ManagedRechargeStatusAwaitingPayment = "awaiting_payment"
	ManagedRechargeStatusValidating     = "validating"
	ManagedRechargeStatusPaid           = "paid"
	ManagedRechargeStatusIssued         = "issued"
	ManagedRechargeStatusSubmitting     = "submitting"
	ManagedRechargeStatusQueued         = "queued"
	ManagedRechargeStatusProcessing     = "processing"
	ManagedRechargeStatusVerifying      = "verifying"
	ManagedRechargeStatusActionRequired = "action_required"
	ManagedRechargeStatusManualReview   = "manual_review"
	ManagedRechargeStatusCompleted      = "completed"
	ManagedRechargeStatusFailed         = "failed"
	ManagedRechargeStatusRefunded       = "refunded"

	ManagedRechargeFulfillmentProxy    = "proxy"
	ManagedRechargeFulfillmentExternal = "external"

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

type ManagedRechargeSecretProtector interface {
	SecretEncryptor
	BlindIndex(plaintext string) string
}

type ManagedRechargeService struct {
	db                *sql.DB
	encryptor         ManagedRechargeSecretProtector
	balanceCache      managedRechargeBalanceCache
	authCache         APIKeyAuthCacheInvalidator
	upstream          managedRechargeUpstream
	mockMode          bool
	mockStepSecs      int
	featureReady      bool
	fulfillmentReady  bool
	fulfillmentMode   string
	externalRedeemURL string
	paymentService    *PaymentService
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
	Enabled         bool                     `json:"enabled"`
	Balance         float64                  `json:"balance"`
	PaymentMethod   string                   `json:"payment_method"`
	Products        []ManagedRechargeProduct `json:"products"`
	MockMode        bool                     `json:"mock_mode"`
	MockStepSeconds int                      `json:"mock_step_seconds,omitempty"`
	FulfillmentMode string                   `json:"fulfillment_mode"`
}

type ManagedRechargeSessionValidation struct {
	Valid      bool   `json:"valid"`
	Email      string `json:"email,omitempty"`
	Membership string `json:"membership"`
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
	FulfillmentMode       string     `json:"fulfillment_mode"`
	CDKMasked             string     `json:"cdk_masked,omitempty"`
	RedemptionCode        string     `json:"redemption_code,omitempty"`
	RedemptionURL         string     `json:"redemption_url,omitempty"`
	Price                 float64    `json:"price"`
	Status                string     `json:"status"`
	PaymentOrderID        *int64     `json:"payment_order_id,omitempty"`
	PaymentStatus         string     `json:"payment_status,omitempty"`
	PaymentExpiresAt      *time.Time `json:"payment_expires_at,omitempty"`
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
	cdkStatus             string
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
	ClientIP       string
	IsMobile       bool
	SrcHost        string
	SrcURL         string
	ReturnURL      string
	Locale         string
}

type ManagedRechargeCheckout struct {
	Order   *ManagedRechargeOrder `json:"order"`
	Payment *CreateOrderResponse  `json:"payment,omitempty"`
}

type ManagedRechargePaymentSelection struct {
	OrderID     int64
	Price       float64
	ProductName string
}

type ManagedRechargeImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

type ManagedRechargeCDKVerification struct {
	ID                int64  `json:"id"`
	Valid             bool   `json:"valid"`
	ExpectedPlanType  string `json:"expected_plan_type"`
	ActualPlanType    string `json:"actual_plan_type,omitempty"`
	PlanName          string `json:"plan_name,omitempty"`
	ProcessingMode    string `json:"processing_mode,omitempty"`
	MatchesProduct    bool   `json:"matches_product"`
	VerificationScope string `json:"verification_scope"`
}

func NewManagedRechargeService(
	db *sql.DB,
	encryptor ManagedRechargeSecretProtector,
	balanceCache managedRechargeBalanceCache,
	authCache APIKeyAuthCacheInvalidator,
) *ManagedRechargeService {
	upstream, mockMode, mockStepSecs, upstreamErr := newManagedRechargeUpstreamFromEnvironment()
	fulfillmentMode, externalRedeemURL, fulfillmentConfigErr := managedRechargeFulfillmentConfigFromEnvironment()
	if upstreamErr == nil && fulfillmentConfigErr != nil {
		upstreamErr = fulfillmentConfigErr
	}
	fulfillmentEnabled, fulfillmentErr := managedRechargeFulfillmentEnabledFromEnvironment(mockMode)
	if upstreamErr == nil && fulfillmentErr != nil {
		upstreamErr = fulfillmentErr
	}
	if upstreamErr != nil {
		log.Printf("Warning: managed recharge provider is unavailable: %v", upstreamErr)
	}
	featureReady := db != nil && encryptor != nil && upstreamErr == nil
	return &ManagedRechargeService{
		db:                db,
		encryptor:         encryptor,
		balanceCache:      balanceCache,
		authCache:         authCache,
		upstream:          upstream,
		mockMode:          mockMode,
		mockStepSecs:      mockStepSecs,
		featureReady:      featureReady,
		fulfillmentReady:  featureReady && fulfillmentEnabled,
		fulfillmentMode:   fulfillmentMode,
		externalRedeemURL: externalRedeemURL,
	}
}

func (s *ManagedRechargeService) SetPaymentService(paymentService *PaymentService) {
	if s == nil {
		return
	}
	s.paymentService = paymentService
}

func (s *ManagedRechargeService) GetCatalog(ctx context.Context, userID int64) (*ManagedRechargeCatalog, error) {
	if s == nil || s.db == nil {
		return &ManagedRechargeCatalog{Enabled: false, PaymentMethod: "alipay", Products: []ManagedRechargeProduct{}}, nil
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
		Enabled:         s.fulfillmentReady && len(products) > 0,
		Balance:         balance,
		PaymentMethod:   "alipay",
		Products:        products,
		MockMode:        s.mockMode,
		MockStepSeconds: s.mockStepSecs,
		FulfillmentMode: s.fulfillmentMode,
	}, nil
}

func (s *ManagedRechargeService) ListProducts(ctx context.Context) ([]ManagedRechargeProduct, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	return s.listProducts(ctx, false)
}

func (s *ManagedRechargeService) ValidateSession(ctx context.Context, raw string) (*ManagedRechargeSessionValidation, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	if s.fulfillmentMode == ManagedRechargeFulfillmentExternal {
		return nil, infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_SESSION_EXTERNAL", "Session is submitted only on the external redemption page")
	}
	session := strings.TrimSpace(raw)
	if session == "" || len(session) > managedRechargeSessionMaxBytes {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_SESSION_INVALID", "invalid Session payload")
	}
	localEmail, err := parseManagedRechargeSession(session)
	if err != nil {
		return nil, err
	}

	validated, err := s.upstream.validateSession(ctx, session)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "MANAGED_RECHARGE_SESSION_VALIDATION_UNAVAILABLE", "Session validation is temporarily unavailable")
	}
	if !validated.Valid {
		return &ManagedRechargeSessionValidation{Valid: false, Membership: "unknown"}, nil
	}

	email := strings.TrimSpace(validated.Email)
	if email == "" {
		email = localEmail
	}
	return &ManagedRechargeSessionValidation{
		Valid:      true,
		Email:      email,
		Membership: managedRechargeMembership(validated.Subscription),
	}, nil
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
		hashString := s.encryptor.BlindIndex(code)
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

func (s *ManagedRechargeService) VerifyCDK(ctx context.Context, id int64) (*ManagedRechargeCDKVerification, error) {
	if err := s.requireReady(); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_CDK_INVALID", "invalid CDK")
	}

	var ciphertext, expectedPlanType string
	err := s.db.QueryRowContext(ctx, `
		SELECT c.code_ciphertext, p.plan_type
		FROM managed_recharge_cdks c
		JOIN managed_recharge_products p ON p.id = c.product_id
		WHERE c.id = $1
	`, id).Scan(&ciphertext, &expectedPlanType)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, infraerrors.New(http.StatusNotFound, "MANAGED_RECHARGE_CDK_NOT_FOUND", "CDK not found")
	}
	if err != nil {
		return nil, fmt.Errorf("load managed recharge CDK for verification: %w", err)
	}

	code, err := s.encryptor.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt managed recharge CDK for verification: %w", err)
	}
	verified, err := s.upstream.verifyCDK(ctx, code)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "MANAGED_RECHARGE_UPSTREAM_UNAVAILABLE", "recharge provider is temporarily unavailable")
	}
	actualPlanType := normalizeManagedRechargePlanType(verified.PlanType)
	return &ManagedRechargeCDKVerification{
		ID:                id,
		Valid:             verified.Valid,
		ExpectedPlanType:  normalizeManagedRechargePlanType(expectedPlanType),
		ActualPlanType:    actualPlanType,
		PlanName:          strings.TrimSpace(verified.PlanName),
		ProcessingMode:    strings.TrimSpace(verified.ProcessingMode),
		MatchesProduct:    verified.Valid && actualPlanType != "" && actualPlanType == normalizeManagedRechargePlanType(expectedPlanType),
		VerificationScope: "verify_only",
	}, nil
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

func (s *ManagedRechargeService) CreateCheckout(ctx context.Context, userID int64, input ManagedRechargeCreateOrderInput) (*ManagedRechargeCheckout, error) {
	if err := s.requireFulfillmentReady(); err != nil {
		return nil, err
	}
	if s.paymentService == nil {
		return nil, infraerrors.ServiceUnavailable("MANAGED_RECHARGE_PAYMENT_UNAVAILABLE", "Alipay checkout is not configured")
	}
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.ProductID <= 0 || input.IdempotencyKey == "" || len(input.IdempotencyKey) > 128 {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_REQUEST_INVALID", "invalid recharge request")
	}

	session := strings.TrimSpace(input.Session)
	accountEmail := ""
	sessionCiphertext := ""
	if s.fulfillmentMode != ManagedRechargeFulfillmentExternal {
		if session == "" || len(session) > managedRechargeSessionMaxBytes {
			return nil, infraerrors.BadRequest("MANAGED_RECHARGE_SESSION_INVALID", "invalid Session payload")
		}
		var err error
		accountEmail, err = parseManagedRechargeSession(session)
		if err != nil {
			return nil, err
		}
		if s.mockMode && !isManagedRechargeMockSession(session) {
			return nil, infraerrors.BadRequest("MANAGED_RECHARGE_MOCK_SESSION_REQUIRED", "mock mode only accepts the provided test Session")
		}
		sessionCiphertext, err = s.encryptor.Encrypt(session)
		if err != nil {
			return nil, fmt.Errorf("encrypt managed recharge Session: %w", err)
		}
	}

	if existing, err := s.getOrderByIdempotency(ctx, userID, input.IdempotencyKey); err == nil {
		if existing.PaymentOrderID != nil {
			paymentResponse, paymentErr := s.paymentService.GetCreateOrderResponse(ctx, userID, *existing.PaymentOrderID)
			if paymentErr != nil {
				return nil, paymentErr
			}
			if err := s.attachExternalRedemption(existing); err != nil {
				return nil, err
			}
			return &ManagedRechargeCheckout{Order: existing, Payment: paymentResponse}, nil
		}
		if existing.Status != ManagedRechargeStatusAwaitingPayment {
			if err := s.attachExternalRedemption(existing); err != nil {
				return nil, err
			}
			return &ManagedRechargeCheckout{Order: existing}, nil
		}
		paymentResponse, paymentErr := s.createAlipayPayment(ctx, userID, existing, input)
		if paymentErr != nil {
			_ = s.failUnpaidOrder(context.Background(), existing.ID, "PAYMENT_PROVIDER_FAILED", "支付宝订单创建失败，请重新提交", false)
			return nil, paymentErr
		}
		updated, getErr := s.GetOrder(ctx, userID, existing.ID, false)
		if getErr != nil {
			return nil, getErr
		}
		return &ManagedRechargeCheckout{Order: updated, Payment: paymentResponse}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	orderCtx, cancelOrder := context.WithTimeout(context.Background(), managedRechargeOrderTimeout)
	defer cancelOrder()
	product, err := s.getActiveProduct(orderCtx, input.ProductID)
	if err != nil {
		return nil, err
	}
	orderNo, err := newManagedRechargeOrderNo()
	if err != nil {
		return nil, fmt.Errorf("generate managed recharge order number: %w", err)
	}
	var orderID int64
	err = s.db.QueryRowContext(orderCtx, `
		INSERT INTO managed_recharge_orders
		    (order_no, user_id, product_id, idempotency_key, price, status, account_email, session_ciphertext, fulfillment_mode)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, idempotency_key) DO NOTHING
		RETURNING id
	`, orderNo, userID, product.ID, input.IdempotencyKey, product.Price, ManagedRechargeStatusAwaitingPayment, accountEmail, sessionCiphertext, s.fulfillmentMode).Scan(&orderID)
	if errors.Is(err, sql.ErrNoRows) {
		existing, getErr := s.getOrderByIdempotency(orderCtx, userID, input.IdempotencyKey)
		if getErr != nil {
			return nil, getErr
		}
		if existing.PaymentOrderID == nil {
			return nil, infraerrors.Conflict("MANAGED_RECHARGE_ORDER_CONFLICT", "recharge order is being created")
		}
		paymentResponse, paymentErr := s.paymentService.GetCreateOrderResponse(orderCtx, userID, *existing.PaymentOrderID)
		if paymentErr != nil {
			return nil, paymentErr
		}
		return &ManagedRechargeCheckout{Order: existing, Payment: paymentResponse}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("create managed recharge order: %w", err)
	}

	cleanupUnpaid := true
	defer func() {
		if cleanupUnpaid {
			_ = s.runRecovery(func(recoveryCtx context.Context) error {
				return s.failUnpaidOrder(recoveryCtx, orderID, "ORDER_INTERRUPTED", "订单创建被中断，请重新提交", false)
			})
		}
	}()

	var reserved *ManagedRechargeCDK
	var plaintextCode string
	for attempt := 0; attempt < managedRechargeReserveAttempts; attempt++ {
		reserved, err = s.reserveNextCDK(orderCtx, orderID, product.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrManagedRechargeNoInventory
			}
			return nil, err
		}
		plaintextCode, err = s.encryptor.Decrypt(reserved.codeCiphertext)
		if err != nil {
			if recoveryErr := s.quarantineReservedCDK(orderCtx, orderID, "CDK_DECRYPT_FAILED"); recoveryErr != nil {
				return nil, recoveryErr
			}
			continue
		}
		verified, verifyErr := s.upstream.verifyCDK(orderCtx, plaintextCode)
		if verifyErr != nil {
			return nil, infraerrors.New(http.StatusBadGateway, "MANAGED_RECHARGE_UPSTREAM_UNAVAILABLE", "recharge provider is temporarily unavailable")
		}
		if verified.Valid {
			actualPlanType := normalizeManagedRechargePlanType(verified.PlanType)
			if actualPlanType == product.PlanType {
				break
			}
			if recoveryErr := s.reassignMismatchedCDK(orderCtx, orderID, product.PlanType, actualPlanType); recoveryErr != nil {
				return nil, recoveryErr
			}
			reserved = nil
			plaintextCode = ""
			continue
		}
		if recoveryErr := s.quarantineReservedCDK(orderCtx, orderID, "CDK_INVALID"); recoveryErr != nil {
			return nil, recoveryErr
		}
		reserved = nil
		plaintextCode = ""
	}
	if reserved == nil || plaintextCode == "" {
		return nil, ErrManagedRechargeNoInventory
	}

	prepared, err := s.getOrder(orderCtx, orderID, &userID)
	if err != nil {
		return nil, err
	}
	paymentResponse, err := s.createAlipayPayment(orderCtx, userID, prepared, input)
	if err != nil {
		return nil, err
	}
	cleanupUnpaid = false
	updated, err := s.GetOrder(orderCtx, userID, orderID, false)
	if err != nil {
		return nil, err
	}
	return &ManagedRechargeCheckout{Order: updated, Payment: paymentResponse}, nil
}

func (s *ManagedRechargeService) createAlipayPayment(ctx context.Context, userID int64, order *ManagedRechargeOrder, input ManagedRechargeCreateOrderInput) (*CreateOrderResponse, error) {
	if order == nil {
		return nil, ErrManagedRechargeOrderMissing
	}
	return s.paymentService.CreateOrder(ctx, CreateOrderRequest{
		UserID:                  userID,
		Amount:                  order.Price,
		PaymentType:             payment.TypeAlipay,
		ClientIP:                input.ClientIP,
		IsMobile:                input.IsMobile,
		SrcHost:                 input.SrcHost,
		SrcURL:                  input.SrcURL,
		ReturnURL:               input.ReturnURL,
		PaymentSource:            "managed_recharge_alipay",
		OrderType:                payment.OrderTypeManagedRecharge,
		Locale:                   input.Locale,
		ManagedRechargeOrderID: order.ID,
	})
}

func (s *ManagedRechargeService) ValidatePaymentSelection(ctx context.Context, userID, orderID int64) (*ManagedRechargePaymentSelection, error) {
	if s == nil || s.db == nil || orderID <= 0 {
		return nil, ErrManagedRechargeOrderMissing
	}
	var selection ManagedRechargePaymentSelection
	var status, cdkStatus string
	var paymentOrderID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT o.id, o.price, p.name, o.status, COALESCE(c.status, ''), o.payment_order_id
		FROM managed_recharge_orders o
		JOIN managed_recharge_products p ON p.id = o.product_id
		LEFT JOIN managed_recharge_cdks c ON c.id = o.cdk_id
		WHERE o.id = $1 AND o.user_id = $2
	`, orderID, userID).Scan(&selection.OrderID, &selection.Price, &selection.ProductName, &status, &cdkStatus, &paymentOrderID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrManagedRechargeOrderMissing
	}
	if err != nil {
		return nil, fmt.Errorf("load managed recharge payment selection: %w", err)
	}
	if status != ManagedRechargeStatusAwaitingPayment || cdkStatus != managedRechargeCDKReserved || paymentOrderID.Valid {
		return nil, infraerrors.Conflict("MANAGED_RECHARGE_NOT_PAYABLE", "managed recharge order is not awaiting payment")
	}
	return &selection, nil
}

func (s *ManagedRechargeService) ValidateRefundablePaymentOrder(ctx context.Context, paymentOrderID int64) error {
	var status, fulfillmentMode, errorCode string
	var paidAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT status, fulfillment_mode, error_code, paid_at
		FROM managed_recharge_orders
		WHERE payment_order_id = $1
	`, paymentOrderID).Scan(&status, &fulfillmentMode, &errorCode, &paidAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrManagedRechargeOrderMissing
	}
	if err != nil {
		return err
	}
	if fulfillmentMode == ManagedRechargeFulfillmentExternal {
		return infraerrors.Conflict("MANAGED_RECHARGE_EXTERNAL_CDK_ISSUED", "externally issued CDKs cannot be refunded automatically")
	}
	if status != ManagedRechargeStatusManualReview || !paidAt.Valid {
		return infraerrors.Conflict("MANAGED_RECHARGE_REFUND_REVIEW_REQUIRED", "managed recharge order is not eligible for refund")
	}
	if errorCode == "UPSTREAM_CREATE_REJECTED" || errorCode == "PAYMENT_AFTER_CANCELLATION" {
		return nil
	}
	if errorCode == "UPSTREAM_TASK_NOT_FOUND" && time.Since(paidAt.Time) >= managedRechargePaidReviewTTL {
		return nil
	}
	return infraerrors.Conflict("MANAGED_RECHARGE_REFUND_SYNC_REQUIRED", "provider must confirm that no task exists before refunding")
}

func (s *ManagedRechargeService) AttachPaymentOrder(ctx context.Context, orderID, paymentOrderID int64, expiresAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET payment_order_id = $2, updated_at = NOW()
		WHERE id = $1 AND status = 'awaiting_payment' AND payment_order_id IS NULL AND paid_at IS NULL
	`, orderID, paymentOrderID)
	if err != nil {
		return fmt.Errorf("attach managed recharge payment order: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return infraerrors.Conflict("MANAGED_RECHARGE_PAYMENT_CONFLICT", "managed recharge payment order is already attached")
	}
	_ = expiresAt
	return s.appendEvent(ctx, orderID, nil, "ALIPAY_ORDER_CREATED", map[string]any{"payment_order_id": paymentOrderID})
}

func (s *ManagedRechargeService) ReleaseUnpaidPaymentOrder(ctx context.Context, paymentOrderID int64, code, message string) error {
	var orderID int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM managed_recharge_orders
		WHERE payment_order_id = $1 AND paid_at IS NULL
	`, paymentOrderID).Scan(&orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.failUnpaidOrder(ctx, orderID, code, message, false)
}

func (s *ManagedRechargeService) FulfillPaidPaymentOrder(ctx context.Context, orderID, paymentOrderID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET status = 'paid', paid_at = COALESCE(paid_at, NOW()), error_code = '', error_message = '', updated_at = NOW()
		WHERE id = $1 AND payment_order_id = $2 AND status = 'awaiting_payment'
	`, orderID, paymentOrderID)
	if err != nil {
		return fmt.Errorf("mark managed recharge Alipay order paid: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 1 {
		if err := appendManagedRechargeEventTx(ctx, tx, orderID, nil, "ALIPAY_PAID", map[string]any{"payment_order_id": paymentOrderID}); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	order, err := s.getOrder(ctx, orderID, nil)
	if err != nil {
		return err
	}
	if order.PaymentOrderID == nil || *order.PaymentOrderID != paymentOrderID {
		return infraerrors.Conflict("MANAGED_RECHARGE_PAYMENT_MISMATCH", "managed recharge payment order mismatch")
	}
	if order.Status != ManagedRechargeStatusPaid {
		switch order.Status {
		case ManagedRechargeStatusIssued, ManagedRechargeStatusSubmitting, ManagedRechargeStatusQueued,
			ManagedRechargeStatusProcessing, ManagedRechargeStatusVerifying, ManagedRechargeStatusActionRequired,
			ManagedRechargeStatusManualReview, ManagedRechargeStatusCompleted, ManagedRechargeStatusRefunded:
			return nil
		case ManagedRechargeStatusFailed:
			if order.ErrorCode == "PAYMENT_CANCELLED" || order.ErrorCode == "PAYMENT_EXPIRED" {
				_ = s.markManualReview(ctx, orderID, "PAYMENT_AFTER_CANCELLATION", "支付宝已到账，但原订单已关闭，请管理员核对并原路退款")
				return nil
			}
			return infraerrors.Conflict("MANAGED_RECHARGE_NOT_PAYABLE", "managed recharge order is not payable")
		default:
			return infraerrors.Conflict("MANAGED_RECHARGE_NOT_PAYABLE", "managed recharge order is not payable")
		}
	}
	if order.cdkCiphertext == "" || order.cdkStatus != managedRechargeCDKReserved {
		_ = s.markManualReview(ctx, orderID, "CDK_RESERVATION_MISSING", "支付成功，但库存状态需要人工核对")
		return nil
	}
	if order.FulfillmentMode == ManagedRechargeFulfillmentExternal {
		if err := s.markExternalOrderIssued(ctx, orderID); err != nil {
			_ = s.markManualReview(ctx, orderID, "CDK_ISSUE_FAILED", "支付成功，但 CDK 发放状态需要人工核对")
		}
		return nil
	}

	code, err := s.encryptor.Decrypt(order.cdkCiphertext)
	if err != nil {
		_ = s.markManualReview(ctx, orderID, "CDK_DECRYPT_FAILED", "支付成功，但库存读取失败，需要人工核对")
		return nil
	}
	session, err := s.encryptor.Decrypt(order.sessionCiphertext)
	if err != nil {
		_ = s.markManualReview(ctx, orderID, "SESSION_DECRYPT_FAILED", "支付成功，但账号凭证读取失败，需要人工核对")
		return nil
	}

	fulfillCtx, cancel := context.WithTimeout(context.Background(), managedRechargeFulfillTimeout)
	defer cancel()
	created, createErr := s.upstream.createTask(fulfillCtx, code, session)
	if createErr != nil {
		_ = s.markManualReview(fulfillCtx, orderID, "UPSTREAM_CREATE_UNCERTAIN", "支付成功，上游提交结果需要人工核对")
		return nil
	}
	if strings.TrimSpace(created.TaskID) == "" {
		_ = s.markManualReview(fulfillCtx, orderID, "UPSTREAM_CREATE_REJECTED", "充值任务未被受理，请管理员核对并原路退款")
		return nil
	}
	if err := s.markTaskCreated(fulfillCtx, orderID, created.TaskID); err != nil {
		return err
	}
	confirmed, confirmErr := s.upstream.confirmTask(fulfillCtx, created.TaskID)
	if confirmErr != nil {
		_ = s.markManualReview(fulfillCtx, orderID, "UPSTREAM_CONFIRM_UNCERTAIN", "支付成功，上游确认结果需要人工核对")
		return nil
	}
	status := normalizeManagedRechargeAcceptedStatus(confirmed.Status)
	if status == "" {
		_ = s.markManualReview(fulfillCtx, orderID, "UPSTREAM_CONFIRM_UNCERTAIN", "支付成功，上游确认结果需要人工核对")
		return nil
	}
	_, err = s.db.ExecContext(fulfillCtx, `
		UPDATE managed_recharge_orders
		SET status = $2, upstream_status = $3, submitted_at = NOW(), last_synced_at = NOW(),
		    error_code = '', error_message = '', updated_at = NOW()
		WHERE id = $1 AND status NOT IN ('refunded', 'completed')
	`, orderID, status, confirmed.Status)
	return err
}

// createLegacyBalanceOrder is retained only for regression coverage of orders
// created before the Alipay-only checkout was introduced. It is not routed.
func (s *ManagedRechargeService) createLegacyBalanceOrder(ctx context.Context, userID int64, input ManagedRechargeCreateOrderInput) (*ManagedRechargeOrder, error) {
	if err := s.requireFulfillmentReady(); err != nil {
		return nil, err
	}
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.ProductID <= 0 || input.IdempotencyKey == "" || len(input.IdempotencyKey) > 128 {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_REQUEST_INVALID", "invalid recharge request")
	}
	session := strings.TrimSpace(input.Session)
	accountEmail := ""
	sessionCiphertext := ""
	if s.fulfillmentMode != ManagedRechargeFulfillmentExternal {
		if session == "" || len(session) > managedRechargeSessionMaxBytes {
			return nil, infraerrors.BadRequest("MANAGED_RECHARGE_SESSION_INVALID", "invalid Session payload")
		}
		var err error
		accountEmail, err = parseManagedRechargeSession(session)
		if err != nil {
			return nil, err
		}
		if s.mockMode && !isManagedRechargeMockSession(session) {
			return nil, infraerrors.BadRequest("MANAGED_RECHARGE_MOCK_SESSION_REQUIRED", "mock mode only accepts the provided test Session")
		}
		sessionCiphertext, err = s.encryptor.Encrypt(session)
		if err != nil {
			return nil, fmt.Errorf("encrypt managed recharge Session: %w", err)
		}
	}
	if existing, err := s.getOrderByIdempotency(ctx, userID, input.IdempotencyKey); err == nil {
		if err := s.attachExternalRedemption(existing); err != nil {
			return nil, err
		}
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
	orderNo, err := newManagedRechargeOrderNo()
	if err != nil {
		return nil, fmt.Errorf("generate managed recharge order number: %w", err)
	}
	var orderID int64
	err = s.db.QueryRowContext(orderCtx, `
		INSERT INTO managed_recharge_orders
		    (order_no, user_id, product_id, idempotency_key, price, account_email, session_ciphertext, fulfillment_mode)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, idempotency_key) DO NOTHING
		RETURNING id
	`, orderNo, userID, product.ID, input.IdempotencyKey, product.Price, accountEmail, sessionCiphertext, s.fulfillmentMode).Scan(&orderID)
	if errors.Is(err, sql.ErrNoRows) {
		existing, getErr := s.getOrderByIdempotency(orderCtx, userID, input.IdempotencyKey)
		if getErr != nil {
			return nil, getErr
		}
		if attachErr := s.attachExternalRedemption(existing); attachErr != nil {
			return nil, attachErr
		}
		return existing, nil
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
	if s.fulfillmentMode == ManagedRechargeFulfillmentExternal {
		if err := s.markExternalOrderIssued(orderCtx, orderID); err != nil {
			_ = s.markManualReview(orderCtx, orderID, "CDK_ISSUE_FAILED", "CDK 发放状态异常，已转人工核对")
			return s.GetOrder(orderCtx, userID, orderID, false)
		}
		return s.GetOrder(orderCtx, userID, orderID, false)
	}

	fulfillCtx, cancelFulfill := context.WithTimeout(context.Background(), managedRechargeFulfillTimeout)
	defer cancelFulfill()
	created, createErr := s.upstream.createTask(fulfillCtx, plaintextCode, session)
	if createErr != nil {
		_ = s.markManualReview(fulfillCtx, orderID, "UPSTREAM_CREATE_UNCERTAIN", "上游提交结果不确定，已转人工核对")
		return s.GetOrder(fulfillCtx, userID, orderID, false)
	}
	if strings.TrimSpace(created.TaskID) == "" {
		if refundErr := s.refundOrder(fulfillCtx, orderID, userID, "UPSTREAM_CREATE_REJECTED", "充值任务未被受理，余额已退回"); refundErr != nil {
			_ = s.markManualReview(fulfillCtx, orderID, "REFUND_FAILED", "充值任务未被受理，退款处理需要人工核对")
		}
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
	order, err = s.syncOrderIfNeeded(ctx, order, forceSync)
	if err != nil {
		return nil, err
	}
	if err := s.attachExternalRedemption(order); err != nil {
		return nil, err
	}
	return order, nil
}

// GetOrderStatus synchronizes a user order without decrypting or returning its CDK.
// The separate status path lets the frontend poll safely while keeping secret
// disclosure behind the explicit "view CDK" action.
func (s *ManagedRechargeService) GetOrderStatus(ctx context.Context, userID, orderID int64) (*ManagedRechargeOrder, error) {
	order, err := s.getOrder(ctx, orderID, &userID)
	if err != nil {
		return nil, err
	}
	return s.syncOrderIfNeeded(ctx, order, false)
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
	if err := s.requireFulfillmentReady(); err != nil {
		return nil, err
	}
	order, err := s.getOrder(ctx, orderID, &userID)
	if err != nil {
		return nil, err
	}
	if order.FulfillmentMode == ManagedRechargeFulfillmentExternal {
		return nil, infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_SESSION_EXTERNAL", "Session is submitted only on the external redemption page")
	}
	session = strings.TrimSpace(session)
	if session == "" || len(session) > managedRechargeSessionMaxBytes {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_SESSION_INVALID", "invalid Session payload")
	}
	email, err := parseManagedRechargeSession(session)
	if err != nil {
		return nil, err
	}
	if s.mockMode && !isManagedRechargeMockSession(session) {
		return nil, infraerrors.BadRequest("MANAGED_RECHARGE_MOCK_SESSION_REQUIRED", "mock mode only accepts the provided test Session")
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
	if strings.TrimSpace(result.Error) != "" {
		if _, updateErr := s.db.ExecContext(fulfillCtx, `
			UPDATE managed_recharge_orders
			SET status = 'action_required', session_ciphertext = '',
			    error_code = 'REPLACEMENT_SESSION_REJECTED',
			    error_message = '新 Session 无效或不匹配，请重新获取后提交',
			    last_synced_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`, orderID); updateErr != nil {
			return nil, fmt.Errorf("mark replacement Session rejected: %w", updateErr)
		}
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
	if order.FulfillmentMode == ManagedRechargeFulfillmentExternal && order.PaidAt != nil {
		return nil, infraerrors.New(http.StatusConflict, "MANAGED_RECHARGE_EXTERNAL_CDK_ISSUED", "externally issued CDKs cannot be refunded automatically")
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
	refundableCreateRejection := order.PaymentOrderID != nil && (order.ErrorCode == "UPSTREAM_CREATE_REJECTED" || order.ErrorCode == "PAYMENT_AFTER_CANCELLATION")
	refundableMissingTask := order.ErrorCode == "UPSTREAM_TASK_NOT_FOUND" && time.Since(*order.PaidAt) >= managedRechargePaidReviewTTL
	if !refundableCreateRejection && !refundableMissingTask {
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
	if order.Status == ManagedRechargeStatusManualReview &&
		(order.ErrorCode == "UPSTREAM_CREATE_REJECTED" || order.ErrorCode == "PAYMENT_AFTER_CANCELLATION") {
		return order, nil
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
			if order.FulfillmentMode == ManagedRechargeFulfillmentExternal && order.Status == ManagedRechargeStatusIssued {
				_, _ = s.db.ExecContext(syncCtx, `
					UPDATE managed_recharge_orders
					SET last_synced_at = NOW(), error_code = '', error_message = '', updated_at = NOW()
					WHERE id = $1
				`, order.ID)
				return s.getOrder(syncCtx, order.ID, &order.UserID)
			}
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
		if order.FulfillmentMode == ManagedRechargeFulfillmentExternal {
			_, updateErr := s.db.ExecContext(syncCtx, `
				UPDATE managed_recharge_orders
				SET status = 'issued', upstream_status = $2, upstream_failure_reason = $3,
				    queue_position = 0, queue_total = 0, progress = $4,
				    error_code = 'EXTERNAL_REDEEM_RETRY',
				    error_message = '上次兑换未完成，CDK 仍归当前订单所有，请前往兑换页重新提交',
				    last_synced_at = NOW(), updated_at = NOW()
				WHERE id = $1 AND status NOT IN ('refunded', 'completed')
			`, order.ID, result.TaskStatus, result.FailureReason, result.Progress)
			if updateErr != nil {
				return nil, fmt.Errorf("mark external redemption retryable: %w", updateErr)
			}
			return s.getOrder(syncCtx, order.ID, &order.UserID)
		}
		if managedRechargeFailureNeedsManualReview(result.FailureReason) {
			_ = s.markManualReview(syncCtx, order.ID, "PAYMENT_RESULT_UNCERTAIN", "支付结果需要人工核对")
		} else {
			if refundErr := s.refundOrder(syncCtx, order.ID, order.UserID, "UPSTREAM_FAILED", normalizeManagedRechargeFailure(result.FailureReason)); refundErr != nil {
				_ = s.markManualReview(syncCtx, order.ID, "REFUND_FAILED", "充值失败，但退款处理需要人工核对")
			}
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
	var paidAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT paid_at FROM managed_recharge_orders WHERE id = $1 FOR UPDATE`, orderID).Scan(&paidAt); err != nil {
		return err
	}
	if paidAt.Valid {
		return nil
	}
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
	var paymentOrderID sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT payment_order_id FROM managed_recharge_orders WHERE id = $1`, orderID).Scan(&paymentOrderID); err != nil {
		return err
	}
	if paymentOrderID.Valid {
		return s.refundAlipayOrder(ctx, orderID, userID, paymentOrderID.Int64, code, message)
	}

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

func (s *ManagedRechargeService) refundAlipayOrder(ctx context.Context, orderID, userID, paymentOrderID int64, code, message string) error {
	if s.paymentService == nil {
		return infraerrors.ServiceUnavailable("MANAGED_RECHARGE_REFUND_UNAVAILABLE", "Alipay refund service is unavailable")
	}
	paymentOrder, err := s.paymentService.GetOrder(ctx, paymentOrderID, userID)
	if err != nil {
		return err
	}
	plan, early, err := s.paymentService.PrepareRefund(ctx, paymentOrderID, paymentOrder.Amount, message, false, false)
	if err != nil {
		return err
	}
	if early != nil {
		return infraerrors.Conflict("MANAGED_RECHARGE_REFUND_REVIEW_REQUIRED", early.Warning)
	}
	result, err := s.paymentService.ExecuteRefund(ctx, plan)
	if err != nil {
		return err
	}
	if result == nil || !result.Success {
		warning := "支付宝退款结果需要人工核对"
		if result != nil && strings.TrimSpace(result.Warning) != "" {
			warning = result.Warning
		}
		_ = s.markManualReview(ctx, orderID, "ALIPAY_REFUND_PENDING", warning)
		return infraerrors.Conflict("MANAGED_RECHARGE_REFUND_PENDING", warning)
	}
	return s.markAlipayRefunded(ctx, orderID, code, message)
}

func (s *ManagedRechargeService) markAlipayRefunded(ctx context.Context, orderID int64, code, message string) error {
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
	result, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET status = 'refunded', session_ciphertext = '', error_code = $2, error_message = $3,
		    refunded_at = COALESCE(refunded_at, NOW()), updated_at = NOW()
		WHERE id = $1 AND status NOT IN ('completed', 'refunded')
	`, orderID, code, message)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return tx.Commit()
	}
	if err := appendManagedRechargeEventTx(ctx, tx, orderID, nil, "ALIPAY_REFUNDED", map[string]any{}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *ManagedRechargeService) markTaskCreated(ctx context.Context, orderID int64, taskID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET status = 'submitting', upstream_task_id = $2, upstream_status = 'created',
		    session_ciphertext = '', updated_at = NOW()
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
		    session_ciphertext = '', last_synced_at = NOW(), updated_at = NOW()
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
	       o.product_id, p.slug, p.name, o.fulfillment_mode, COALESCE(c.code_masked, ''), o.price, o.status,
	       o.payment_order_id, COALESCE(po.status, ''), po.expires_at,
	       o.account_email, o.upstream_task_id, o.upstream_status, o.upstream_failure_reason,
	       o.queue_position, o.queue_total, o.progress, o.error_code, o.error_message,
	       COALESCE(o.balance_before, 0), COALESCE(o.balance_after, 0),
	       o.paid_at, o.submitted_at, o.completed_at, o.refunded_at, o.last_synced_at,
	       o.created_at, o.updated_at, o.session_ciphertext, COALESCE(c.code_ciphertext, ''),
	       COALESCE(c.status, '')
	FROM managed_recharge_orders o
	JOIN managed_recharge_products p ON p.id = o.product_id
	JOIN users u ON u.id = o.user_id
	LEFT JOIN managed_recharge_cdks c ON c.id = o.cdk_id
	LEFT JOIN payment_orders po ON po.id = o.payment_order_id
`

type managedRechargeRowScanner interface {
	Scan(dest ...any) error
}

func scanManagedRechargeOrder(scanner managedRechargeRowScanner) (*ManagedRechargeOrder, error) {
	var order ManagedRechargeOrder
	var paymentOrderID sql.NullInt64
	var paymentExpiresAt, paidAt, submittedAt, completedAt, refundedAt, lastSyncedAt sql.NullTime
	err := scanner.Scan(
		&order.ID, &order.OrderNo, &order.UserID, &order.UserEmail, &order.Username,
		&order.ProductID, &order.ProductSlug, &order.ProductName, &order.FulfillmentMode, &order.CDKMasked, &order.Price,
		&order.Status, &paymentOrderID, &order.PaymentStatus, &paymentExpiresAt,
		&order.AccountEmail, &order.upstreamTaskID, &order.UpstreamStatus,
		&order.upstreamFailureReason, &order.QueuePosition, &order.QueueTotal, &order.Progress,
		&order.ErrorCode, &order.ErrorMessage, &order.BalanceBefore, &order.BalanceAfter,
		&paidAt, &submittedAt, &completedAt, &refundedAt, &lastSyncedAt,
		&order.CreatedAt, &order.UpdatedAt, &order.sessionCiphertext, &order.cdkCiphertext, &order.cdkStatus,
	)
	if err != nil {
		return nil, err
	}
	if paymentOrderID.Valid {
		order.PaymentOrderID = &paymentOrderID.Int64
	}
	order.PaymentExpiresAt = nullTimePointer(paymentExpiresAt)
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

func (s *ManagedRechargeService) markExternalOrderIssued(ctx context.Context, orderID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin external CDK issue: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `
		UPDATE managed_recharge_cdks
		SET status = 'used', reserved_at = NULL, updated_at = NOW()
		WHERE id = (SELECT cdk_id FROM managed_recharge_orders WHERE id = $1)
		  AND status = 'reserved'
	`, orderID)
	if err != nil {
		return fmt.Errorf("consume externally issued CDK: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("external CDK issue did not consume reserved inventory")
	}
	result, err = tx.ExecContext(ctx, `
		UPDATE managed_recharge_orders
		SET status = 'issued', upstream_status = 'awaiting_user', submitted_at = NOW(),
		    session_ciphertext = '', error_code = '', error_message = '',
		    last_synced_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'paid' AND fulfillment_mode = 'external'
	`, orderID)
	if err != nil {
		return fmt.Errorf("mark external order issued: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("external order was not in paid state")
	}
	if err := appendManagedRechargeEventTx(ctx, tx, orderID, nil, "CDK_ISSUED", map[string]any{"fulfillment_mode": ManagedRechargeFulfillmentExternal}); err != nil {
		return fmt.Errorf("record external CDK issue: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit external CDK issue: %w", err)
	}
	return nil
}

func (s *ManagedRechargeService) attachExternalRedemption(order *ManagedRechargeOrder) error {
	if order == nil || order.FulfillmentMode != ManagedRechargeFulfillmentExternal || order.PaidAt == nil ||
		order.Status == ManagedRechargeStatusRefunded || order.cdkStatus != managedRechargeCDKUsed || order.cdkCiphertext == "" {
		return nil
	}
	code, err := s.encryptor.Decrypt(order.cdkCiphertext)
	if err != nil {
		return fmt.Errorf("decrypt external redemption CDK: %w", err)
	}
	redeemURL, err := url.Parse(s.externalRedeemURL)
	if err != nil {
		return fmt.Errorf("parse external redemption URL: %w", err)
	}
	query := redeemURL.Query()
	query.Set("cdk", code)
	redeemURL.RawQuery = query.Encode()
	order.RedemptionCode = code
	order.RedemptionURL = redeemURL.String()
	return nil
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

func (s *ManagedRechargeService) requireFulfillmentReady() error {
	if err := s.requireReady(); err != nil {
		return err
	}
	if !s.fulfillmentReady {
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

func isManagedRechargeMockSession(raw string) bool {
	var payload struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(payload.User.Email), managedRechargeMockSessionEmail) &&
		strings.TrimSpace(payload.AccessToken) == managedRechargeMockAccessToken
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

func managedRechargeMembership(subscription *managedRechargeSubscriptionValidation) string {
	if subscription == nil || !subscription.HasActiveSubscription {
		return "free"
	}
	for _, value := range []string{subscription.PlanType, subscription.SubscriptionPlan} {
		normalized := strings.ToLower(strings.TrimSpace(value))
		normalized = strings.NewReplacer("_", "", "-", "", " ", "").Replace(normalized)
		if strings.Contains(normalized, "pro") {
			return "pro"
		}
		if strings.Contains(normalized, "plus") {
			return "plus"
		}
	}
	return "unknown"
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
	case ManagedRechargeStatusPaid, ManagedRechargeStatusIssued, ManagedRechargeStatusSubmitting, ManagedRechargeStatusQueued,
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
		return "Session 已失效，款项将原路退回"
	case strings.Contains(normalized, "existing_entitlement"), strings.Contains(normalized, "subscription"):
		return "账号当前订阅状态不符合充值条件，款项将原路退回"
	case strings.Contains(normalized, "account"):
		return "账号暂时不符合充值条件，款项将原路退回"
	default:
		return "充值未完成，款项将原路退回"
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
