package service

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *PaymentService) PurchaseSubscriptionWithBalance(ctx context.Context, req BalanceSubscriptionPurchaseRequest) (*BalanceSubscriptionPurchaseResponse, error) {
	if s == nil || s.entClient == nil || s.userRepo == nil || s.subscriptionSvc == nil || s.configService == nil {
		return nil, infraerrors.New(500, "SERVICE_UNAVAILABLE", "payment service is not ready")
	}
	plan, err := s.validateSubOrder(ctx, CreateOrderRequest{
		UserID:    req.UserID,
		OrderType: payment.OrderTypeSubscription,
		PlanID:    req.PlanID,
	})
	if err != nil {
		return nil, err
	}
	if plan.Price <= 0 {
		return nil, infraerrors.BadRequest("INVALID_AMOUNT", "subscription price must be positive")
	}

	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.Status != payment.EntityStatusActive {
		return nil, infraerrors.Forbidden("USER_INACTIVE", "user account is disabled")
	}
	if user.Balance < plan.Price {
		return nil, ErrInsufficientBalance.WithMetadata(balancePurchaseErrorMetadata(user.Balance, plan.Price))
	}
	if s.notificationEmailService != nil {
		s.notificationEmailService.RememberRecipientLocale(ctx, req.UserID, user.Email, req.Locale)
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin balance purchase transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	now := time.Now()
	outTradeNo, err := s.allocateOutTradeNo(txCtx, tx)
	if err != nil {
		return nil, err
	}
	newBalance, err := s.deductBalanceForPurchaseStrict(txCtx, tx.Client(), req.UserID, plan.Price)
	if err != nil {
		if infraerrors.Reason(err) == ErrInsufficientBalance.Reason {
			return nil, ErrInsufficientBalance.WithMetadata(balancePurchaseErrorMetadata(user.Balance, plan.Price))
		}
		return nil, fmt.Errorf("deduct balance: %w", err)
	}
	balanceBefore := newBalance + plan.Price

	order, err := tx.PaymentOrder.Create().
		SetUserID(req.UserID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetNillableUserNotes(psNilIfEmpty(user.Notes)).
		SetAmount(plan.Price).
		SetPayAmount(plan.Price).
		SetFeeRate(0).
		SetRechargeCode("").
		SetOutTradeNo(outTradeNo).
		SetPaymentType(PaymentTypeBalanceWallet).
		SetPaymentTradeNo(fmt.Sprintf("balance:%d:%d", req.UserID, now.UnixNano())).
		SetOrderType(payment.OrderTypeSubscription).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(now).
		SetPaidAt(now).
		SetCompletedAt(now).
		SetClientIP(req.ClientIP).
		SetSrcHost(req.SrcHost).
		SetPlanID(plan.ID).
		SetSubscriptionGroupID(plan.GroupID).
		SetSubscriptionDays(psComputeValidityDays(plan.ValidityDays, plan.ValidityUnit)).
		Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("create balance subscription order: %w", err)
	}
	code := fmt.Sprintf("BAL-%d-%d", order.ID, now.UnixNano()%100000)
	order, err = tx.PaymentOrder.UpdateOneID(order.ID).SetRechargeCode(code).Save(txCtx)
	if err != nil {
		return nil, fmt.Errorf("set balance subscription order code: %w", err)
	}

	sub, renewed, err := s.subscriptionSvc.AssignOrExtendSubscription(txCtx, &AssignSubscriptionInput{
		UserID:       req.UserID,
		GroupID:      plan.GroupID,
		ValidityDays: psComputeValidityDays(plan.ValidityDays, plan.ValidityUnit),
		AssignedBy:   0,
		Notes:        fmt.Sprintf("balance order %d", order.ID),
	})
	if err != nil {
		return nil, fmt.Errorf("assign subscription: %w", err)
	}

	s.writeAuditLog(txCtx, order.ID, "ORDER_CREATED", fmt.Sprintf("user:%d", req.UserID), map[string]any{
		"orderType":     payment.OrderTypeSubscription,
		"paymentType":   PaymentTypeBalanceWallet,
		"planID":        plan.ID,
		"amount":        plan.Price,
		"balanceBefore": balanceBefore,
		"balanceAfter":  newBalance,
	})
	s.writeAuditLog(txCtx, order.ID, "BALANCE_DEDUCTED", "system", map[string]any{
		"amount":        plan.Price,
		"balanceBefore": balanceBefore,
		"balanceAfter":  newBalance,
	})
	s.writeAuditLog(txCtx, order.ID, "SUBSCRIPTION_ASSIGNED", "system", map[string]any{
		"groupID":      plan.GroupID,
		"validityDays": psComputeValidityDays(plan.ValidityDays, plan.ValidityUnit),
		"renewed":      renewed,
	})
	s.writeAuditLog(txCtx, order.ID, "SUBSCRIPTION_SUCCESS", "system", map[string]any{
		"rechargeCode":   order.RechargeCode,
		"creditedAmount": order.Amount,
		"payAmount":      order.PayAmount,
	})

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit balance purchase transaction: %w", err)
	}

	s.invalidateBalancePurchaseCaches(req.UserID)
	s.dispatchPaymentFulfillmentNotification(order, "SUBSCRIPTION_SUCCESS")

	return &BalanceSubscriptionPurchaseResponse{
		OrderID:        order.ID,
		Amount:         plan.Price,
		Status:         order.Status,
		PaymentType:    order.PaymentType,
		PlanID:         plan.ID,
		BalanceBefore:  balanceBefore,
		BalanceAfter:   newBalance,
		SubscriptionID: sub.ID,
	}, nil
}

func balancePurchaseErrorMetadata(current, required float64) map[string]string {
	return map[string]string{
		"current":  fmt.Sprintf("%.2f", current),
		"required": fmt.Sprintf("%.2f", required),
	}
}

func (s *PaymentService) deductBalanceForPurchaseStrict(ctx context.Context, client *dbent.Client, userID int64, amount float64) (float64, error) {
	if amount <= 0 {
		user, err := s.userRepo.GetByID(ctx, userID)
		if err != nil {
			return 0, err
		}
		return user.Balance, nil
	}
	n, err := client.User.Update().
		Where(dbuser.IDEQ(userID), dbuser.DeletedAtIsNil(), dbuser.BalanceGTE(amount)).
		AddBalance(-amount).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		exists, err := client.User.Query().Where(dbuser.IDEQ(userID), dbuser.DeletedAtIsNil()).Exist(ctx)
		if err != nil {
			return 0, err
		}
		if !exists {
			return 0, ErrUserNotFound
		}
		return 0, ErrInsufficientBalance
	}
	updated, err := client.User.Get(ctx, userID)
	if err != nil {
		return 0, err
	}
	return updated.Balance, nil
}

func (s *PaymentService) invalidateBalancePurchaseCaches(userID int64) {
	if s == nil {
		return
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(context.Background(), userID)
	}
	if s.subscriptionSvc == nil || s.subscriptionSvc.billingCacheService == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.subscriptionSvc.billingCacheService.InvalidateUserBalance(ctx, userID)
	}()
}
