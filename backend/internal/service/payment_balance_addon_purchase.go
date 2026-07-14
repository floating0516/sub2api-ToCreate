package service

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// PurchaseAddonWithBalance atomically deducts balance, records the completed
// order, and grants the selected subscription add-on.
func (s *PaymentService) PurchaseAddonWithBalance(ctx context.Context, req BalanceAddonPurchaseRequest) (*BalanceAddonPurchaseResponse, error) {
	if s == nil || s.entClient == nil || s.userRepo == nil || s.subscriptionSvc == nil ||
		s.subscriptionSvc.addonRepo == nil || s.configService == nil {
		return nil, infraerrors.New(500, "SERVICE_UNAVAILABLE", "payment service is not ready")
	}

	cfg, err := s.configService.GetPaymentConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("get payment config: %w", err)
	}
	request := CreateOrderRequest{
		UserID:         req.UserID,
		OrderType:      payment.OrderTypeAddon,
		AddonProductID: req.AddonProductID,
		SubscriptionID: req.SubscriptionID,
	}
	selection, err := s.validateAddonOrder(ctx, request, cfg)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.Status != payment.EntityStatusActive {
		return nil, infraerrors.Forbidden("USER_INACTIVE", "user account is disabled")
	}
	if user.Balance < selection.product.Price {
		return nil, ErrInsufficientBalance.WithMetadata(balancePurchaseErrorMetadata(user.Balance, selection.product.Price))
	}
	if s.notificationEmailService != nil {
		s.notificationEmailService.RememberRecipientLocale(ctx, req.UserID, user.Email, req.Locale)
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin add-on balance purchase transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	selection, err = s.validateAddonOrder(txCtx, request, cfg)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	outTradeNo, err := s.allocateOutTradeNo(txCtx, tx)
	if err != nil {
		return nil, err
	}
	newBalance, err := s.deductBalanceForPurchaseStrict(txCtx, tx.Client(), req.UserID, selection.product.Price)
	if err != nil {
		if infraerrors.Reason(err) == ErrInsufficientBalance.Reason {
			return nil, ErrInsufficientBalance.WithMetadata(balancePurchaseErrorMetadata(user.Balance, selection.product.Price))
		}
		return nil, fmt.Errorf("deduct balance: %w", err)
	}
	balanceBefore := newBalance + selection.product.Price
	providerSnapshot := attachAddonOrderSnapshot(nil, selection)

	order, err := tx.PaymentOrder.Create().
		SetUserID(req.UserID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetNillableUserNotes(psNilIfEmpty(user.Notes)).
		SetAmount(selection.product.Price).
		SetPayAmount(selection.product.Price).
		SetFeeRate(0).
		SetRechargeCode("").
		SetOutTradeNo(outTradeNo).
		SetPaymentType(PaymentTypeBalanceWallet).
		SetPaymentTradeNo(fmt.Sprintf("balance:addon:%d:%d", req.UserID, now.UnixNano())).
		SetOrderType(payment.OrderTypeAddon).
		SetSubscriptionGroupID(selection.subscription.GroupID).
		SetProviderSnapshot(providerSnapshot).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(now).
		SetPaidAt(now).
		SetCompletedAt(now).
		SetClientIP(req.ClientIP).
		SetSrcHost(req.SrcHost).
		SetNillableSrcURL(psNilIfEmpty(req.SrcURL)).
		Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("create balance add-on order: %w", err)
	}
	code := fmt.Sprintf("BAL-ADD-%d-%d", order.ID, now.UnixNano()%100000)
	order, err = tx.PaymentOrder.UpdateOneID(order.ID).SetRechargeCode(code).Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("set balance add-on order code: %w", err)
	}

	pack, err := s.subscriptionSvc.addonRepo.CreatePurchased(txCtx, CreatePurchasedSubscriptionAddonInput{
		OrderID:        order.ID,
		SubscriptionID: selection.subscription.ID,
		UserID:         req.UserID,
		GroupID:        selection.subscription.GroupID,
		QuotaUSD:       selection.product.QuotaUSD,
		ExpiresAt:      selection.subscription.ExpiresAt,
		Notes:          fmt.Sprintf("Purchased %s via balance order #%d", selection.product.SKU, order.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("grant balance add-on: %w", err)
	}
	snapshot, err := parseAddonOrderSnapshot(order)
	if err != nil {
		return nil, err
	}
	if err := validatePurchasedAddonPack(pack, order, snapshot, selection.subscription.ExpiresAt, true); err != nil {
		return nil, err
	}

	s.writeAuditLog(txCtx, order.ID, "ORDER_CREATED", fmt.Sprintf("user:%d", req.UserID), map[string]any{
		"orderType":      payment.OrderTypeAddon,
		"paymentType":    PaymentTypeBalanceWallet,
		"addonProductID": selection.product.ID,
		"subscriptionID": selection.subscription.ID,
		"amount":         selection.product.Price,
		"balanceBefore":  balanceBefore,
		"balanceAfter":   newBalance,
	})
	s.writeAuditLog(txCtx, order.ID, "BALANCE_DEDUCTED", "system", map[string]any{
		"amount":        selection.product.Price,
		"balanceBefore": balanceBefore,
		"balanceAfter":  newBalance,
	})
	s.writeAuditLog(txCtx, order.ID, "ADDON_GRANTED", "system", map[string]any{
		"addonID":        pack.ID,
		"addonProductID": selection.product.ID,
		"subscriptionID": selection.subscription.ID,
		"quotaUSD":       selection.product.QuotaUSD,
		"expiresAt":      pack.ExpiresAt,
	})
	s.writeAuditLog(txCtx, order.ID, "ADDON_SUCCESS", "system", map[string]any{
		"rechargeCode":  order.RechargeCode,
		"creditedQuota": selection.product.QuotaUSD,
		"payAmount":     order.PayAmount,
	})

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit add-on balance purchase transaction: %w", err)
	}

	s.invalidateBalancePurchaseCaches(req.UserID)
	return &BalanceAddonPurchaseResponse{
		OrderID:        order.ID,
		Amount:         selection.product.Price,
		Status:         order.Status,
		PaymentType:    order.PaymentType,
		AddonID:        pack.ID,
		AddonProductID: selection.product.ID,
		SubscriptionID: selection.subscription.ID,
		QuotaUSD:       selection.product.QuotaUSD,
		ExpiresAt:      pack.ExpiresAt,
		BalanceBefore:  balanceBefore,
		BalanceAfter:   newBalance,
	}, nil
}
