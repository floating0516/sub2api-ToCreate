package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const addonOrderSnapshotKey = "addon_purchase"

type addonOrderSelection struct {
	product      *SubscriptionAddonProduct
	subscription *UserSubscription
}

type addonOrderSnapshot struct {
	ProductID      int64
	ProductSKU     string
	ProductName    string
	SubscriptionID int64
	GroupID        int64
	QuotaUSD       float64
	Price          float64
	ExpiresAt      time.Time
}

func (s *PaymentService) ListAddonProductsForSale(ctx context.Context) ([]SubscriptionAddonProduct, error) {
	if s == nil || s.subscriptionSvc == nil || s.subscriptionSvc.addonRepo == nil {
		return []SubscriptionAddonProduct{}, nil
	}
	return s.subscriptionSvc.addonRepo.ListProducts(ctx, true)
}

func (s *PaymentService) validateAddonOrder(ctx context.Context, req CreateOrderRequest, cfg *PaymentConfig) (*addonOrderSelection, error) {
	if cfg == nil || !cfg.AddonPurchaseEnabled {
		return nil, infraerrors.Forbidden("ADDON_PURCHASE_DISABLED", "subscription add-on purchases are not available")
	}
	if s == nil || s.subscriptionSvc == nil || s.subscriptionSvc.addonRepo == nil {
		return nil, infraerrors.ServiceUnavailable("ADDON_PURCHASE_UNAVAILABLE", "subscription add-on purchase service is unavailable")
	}
	if req.AddonProductID <= 0 || req.SubscriptionID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_INPUT", "add-on product and subscription are required")
	}

	product, err := s.subscriptionSvc.addonRepo.GetProductByID(ctx, req.AddonProductID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionAddonProductNotFound) {
			return nil, infraerrors.NotFound("ADDON_PRODUCT_NOT_AVAILABLE", "subscription add-on product is not available")
		}
		return nil, fmt.Errorf("get subscription add-on product: %w", err)
	}
	if product == nil || !product.ForSale {
		return nil, infraerrors.NotFound("ADDON_PRODUCT_NOT_AVAILABLE", "subscription add-on product is not available")
	}
	if math.IsNaN(product.Price) || math.IsInf(product.Price, 0) || product.Price <= 0 ||
		math.IsNaN(product.QuotaUSD) || math.IsInf(product.QuotaUSD, 0) || product.QuotaUSD <= 0 {
		return nil, infraerrors.BadRequest("ADDON_PRODUCT_INVALID", "subscription add-on product is invalid")
	}
	if (cfg.MinAmount > 0 && product.Price < cfg.MinAmount) || (cfg.MaxAmount > 0 && product.Price > cfg.MaxAmount) {
		return nil, infraerrors.BadRequest("INVALID_AMOUNT", "add-on price is outside the configured payment range")
	}

	subscription, err := s.subscriptionSvc.GetByID(ctx, req.SubscriptionID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return nil, infraerrors.NotFound("SUBSCRIPTION_NOT_FOUND", "subscription not found")
		}
		return nil, fmt.Errorf("get target subscription: %w", err)
	}
	if subscription == nil {
		return nil, infraerrors.NotFound("SUBSCRIPTION_NOT_FOUND", "subscription not found")
	}
	if subscription.UserID != req.UserID {
		return nil, infraerrors.Forbidden("FORBIDDEN", "subscription does not belong to the current user")
	}
	now := time.Now()
	if subscription.Status != SubscriptionStatusActive || !subscription.ExpiresAt.After(now.Add(time.Minute)) {
		return nil, infraerrors.BadRequest("SUBSCRIPTION_NOT_ACTIVE", "subscription must remain active while the payment is completed")
	}

	return &addonOrderSelection{product: product, subscription: subscription}, nil
}

func attachAddonOrderSnapshot(snapshot map[string]any, addon *addonOrderSelection) map[string]any {
	if addon == nil || addon.product == nil || addon.subscription == nil {
		return snapshot
	}
	if snapshot == nil {
		snapshot = map[string]any{"schema_version": 2}
	}
	snapshot[addonOrderSnapshotKey] = map[string]any{
		"product_id":      strconv.FormatInt(addon.product.ID, 10),
		"product_sku":     addon.product.SKU,
		"product_name":    addon.product.Name,
		"subscription_id": strconv.FormatInt(addon.subscription.ID, 10),
		"group_id":        strconv.FormatInt(addon.subscription.GroupID, 10),
		"quota_usd":       strconv.FormatFloat(addon.product.QuotaUSD, 'f', -1, 64),
		"price":           strconv.FormatFloat(addon.product.Price, 'f', -1, 64),
		"expires_at":      addon.subscription.ExpiresAt.UTC().Format(time.RFC3339Nano),
	}
	return snapshot
}

func parseAddonOrderSnapshot(order *dbent.PaymentOrder) (*addonOrderSnapshot, error) {
	if order == nil || order.ProviderSnapshot == nil {
		return nil, errors.New("add-on order snapshot is missing")
	}
	raw, ok := order.ProviderSnapshot[addonOrderSnapshotKey]
	if !ok {
		return nil, errors.New("add-on order snapshot is missing")
	}
	values, ok := raw.(map[string]any)
	if !ok {
		return nil, errors.New("add-on order snapshot is malformed")
	}
	readString := func(key string) string {
		value, _ := values[key].(string)
		return strings.TrimSpace(value)
	}
	productID, err := strconv.ParseInt(readString("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		return nil, errors.New("add-on order product is invalid")
	}
	subscriptionID, err := strconv.ParseInt(readString("subscription_id"), 10, 64)
	if err != nil || subscriptionID <= 0 {
		return nil, errors.New("add-on order subscription is invalid")
	}
	groupID, err := strconv.ParseInt(readString("group_id"), 10, 64)
	if err != nil || groupID <= 0 {
		return nil, errors.New("add-on order group is invalid")
	}
	quotaUSD, err := strconv.ParseFloat(readString("quota_usd"), 64)
	if err != nil || quotaUSD <= 0 || math.IsNaN(quotaUSD) || math.IsInf(quotaUSD, 0) {
		return nil, errors.New("add-on order quota is invalid")
	}
	price, err := strconv.ParseFloat(readString("price"), 64)
	if err != nil || price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
		return nil, errors.New("add-on order price is invalid")
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, readString("expires_at"))
	if err != nil {
		return nil, errors.New("add-on order expiration is invalid")
	}
	productSKU := readString("product_sku")
	if productSKU == "" {
		return nil, errors.New("add-on order product SKU is invalid")
	}
	return &addonOrderSnapshot{
		ProductID:      productID,
		ProductSKU:     productSKU,
		ProductName:    readString("product_name"),
		SubscriptionID: subscriptionID,
		GroupID:        groupID,
		QuotaUSD:       quotaUSD,
		Price:          price,
		ExpiresAt:      expiresAt,
	}, nil
}

func (s *PaymentService) ExecuteAddonFulfillment(ctx context.Context, orderID int64) error {
	order, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil {
		return infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if order.Status == OrderStatusCompleted {
		return nil
	}
	if psIsRefundStatus(order.Status) {
		return infraerrors.BadRequest("INVALID_STATUS", "refund-related order cannot fulfill")
	}
	if order.Status != OrderStatusPaid && order.Status != OrderStatusFailed && order.Status != OrderStatusRecharging {
		return infraerrors.BadRequest("INVALID_STATUS", "order cannot fulfill in status "+order.Status)
	}
	lease, err := s.acquirePaymentFulfillmentLease(ctx, order)
	if err != nil {
		return err
	}
	if lease == nil {
		return nil
	}
	if err := s.doAddon(ctx, order, lease); err != nil {
		s.markFailed(ctx, orderID, lease, err)
		return err
	}
	return nil
}

func (s *PaymentService) doAddon(ctx context.Context, order *dbent.PaymentOrder, lease *paymentFulfillmentLease) error {
	if s == nil || s.subscriptionSvc == nil || s.subscriptionSvc.addonRepo == nil {
		return errors.New("subscription add-on service is unavailable")
	}
	if order == nil || order.OrderType != payment.OrderTypeAddon {
		return errors.New("payment order is not an add-on order")
	}
	snapshot, err := parseAddonOrderSnapshot(order)
	if err != nil {
		return err
	}
	if math.Abs(snapshot.Price-order.Amount) > 0.0001 {
		return errors.New("add-on order price does not match its snapshot")
	}
	existing, err := s.subscriptionSvc.addonRepo.GetByPurchaseOrderID(ctx, order.ID)
	if err == nil {
		if err := validatePurchasedAddonPack(existing, order, snapshot, snapshot.ExpiresAt, false); err != nil {
			return err
		}
		if err := s.applyAffiliateRebateForOrder(ctx, order); err != nil {
			return err
		}
		return s.markCompleted(ctx, order, lease, "ADDON_SUCCESS")
	}
	if !errors.Is(err, ErrSubscriptionAddonNotFound) {
		return fmt.Errorf("look up purchased add-on: %w", err)
	}
	subscription, err := s.subscriptionSvc.GetByID(ctx, snapshot.SubscriptionID)
	if err != nil || subscription == nil {
		return errors.New("target subscription no longer exists")
	}
	now := time.Now()
	if subscription.UserID != order.UserID || subscription.Status != SubscriptionStatusActive || !subscription.ExpiresAt.After(now) {
		return errors.New("target subscription is no longer active")
	}
	expiresAt := snapshot.ExpiresAt
	if subscription.ExpiresAt.Before(expiresAt) {
		expiresAt = subscription.ExpiresAt
	}
	if !expiresAt.After(now) {
		return errors.New("add-on order has expired with its target subscription")
	}

	pack, err := s.subscriptionSvc.addonRepo.CreatePurchased(ctx, CreatePurchasedSubscriptionAddonInput{
		OrderID:        order.ID,
		SubscriptionID: subscription.ID,
		UserID:         order.UserID,
		GroupID:        subscription.GroupID,
		QuotaUSD:       snapshot.QuotaUSD,
		ExpiresAt:      expiresAt,
		Notes:          fmt.Sprintf("Purchased %s via payment order #%d", snapshot.ProductSKU, order.ID),
	})
	if err != nil {
		return fmt.Errorf("grant purchased add-on: %w", err)
	}
	if err := validatePurchasedAddonPack(pack, order, snapshot, expiresAt, true); err != nil {
		return err
	}
	if err := s.applyAffiliateRebateForOrder(ctx, order); err != nil {
		return err
	}
	return s.markCompleted(ctx, order, lease, "ADDON_SUCCESS")
}

func validatePurchasedAddonPack(pack *SubscriptionAddonPack, order *dbent.PaymentOrder, snapshot *addonOrderSnapshot, expiresAt time.Time, requireExactExpiry bool) error {
	if pack == nil || order == nil || snapshot == nil ||
		pack.SubscriptionID != snapshot.SubscriptionID || pack.UserID != order.UserID ||
		pack.GroupID != snapshot.GroupID || math.Abs(pack.QuotaUSD-snapshot.QuotaUSD) > 0.0001 {
		return errors.New("purchased add-on does not match its order")
	}
	if requireExactExpiry {
		if !pack.ExpiresAt.Equal(expiresAt) {
			return errors.New("purchased add-on expiration does not match its order")
		}
	} else if pack.ExpiresAt.After(expiresAt) {
		return errors.New("purchased add-on expiration exceeds its order")
	}
	return nil
}

func addonPaymentSubject(addon *addonOrderSelection) string {
	if addon == nil || addon.product == nil {
		return ""
	}
	return fmt.Sprintf("Sub2API Add-on $%s", strconv.FormatFloat(addon.product.QuotaUSD, 'f', -1, 64))
}
