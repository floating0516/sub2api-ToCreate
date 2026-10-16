package service

import (
	"context"
	"database/sql"
	"errors"
)

var ErrSupportOrderForbidden = errors.New("order belongs to another user")

// ValidateSupportOrder reads ownership without syncing or fulfilling an order.
func (s *ManagedRechargeService) ValidateSupportOrder(ctx context.Context, userID, orderID int64) error {
	var ownerID int64
	err := s.db.QueryRowContext(ctx, "SELECT user_id FROM managed_recharge_orders WHERE id = $1", orderID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrManagedRechargeOrderMissing
	}
	if err != nil {
		return err
	}
	if ownerID != userID {
		return ErrSupportOrderForbidden
	}
	return nil
}
