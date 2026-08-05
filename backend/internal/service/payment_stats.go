package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/redeemcode"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// --- Dashboard & Analytics ---

type UserPaymentSummary struct {
	Currency         string  `json:"currency"`
	GrossPaid        float64 `json:"gross_paid"`
	Refunded         float64 `json:"refunded"`
	NetPaid          float64 `json:"net_paid"`
	SubscriptionPaid float64 `json:"subscription_paid"`
	AddonPaid        float64 `json:"addon_paid"`
	BalancePaid      float64 `json:"balance_paid"`
	PlatformGranted  float64 `json:"platform_granted"`
}

// GetUserPaymentSummary returns the user's real external CNY cash flow for a
// half-open time range. Payments are attributed by paid_at and successful
// refunds by refund_at; balance-wallet purchases are internal transfers and
// therefore excluded.
func (s *PaymentService) GetUserPaymentSummary(ctx context.Context, userID int64, startTime, endTime time.Time) (*UserPaymentSummary, error) {
	result := &UserPaymentSummary{Currency: "CNY"}

	paidOrders, err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.UserIDEQ(userID),
			paymentorder.PaymentTypeNEQ(PaymentTypeBalanceWallet),
			paymentorder.PaidAtNotNil(),
			paymentorder.PaidAtGTE(startTime),
			paymentorder.PaidAtLT(endTime),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	for _, order := range paidOrders {
		if PaymentOrderCurrency(order) != result.Currency {
			continue
		}
		result.GrossPaid += order.PayAmount
		addUserPaymentAmount(result, order.OrderType, order.PayAmount)
	}

	refundedOrders, err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.UserIDEQ(userID),
			paymentorder.PaymentTypeNEQ(PaymentTypeBalanceWallet),
			paymentorder.StatusIn(OrderStatusPartiallyRefunded, OrderStatusRefunded),
			paymentorder.RefundAtNotNil(),
			paymentorder.RefundAtGTE(startTime),
			paymentorder.RefundAtLT(endTime),
			paymentorder.RefundAmountGT(0),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	for _, order := range refundedOrders {
		if PaymentOrderCurrency(order) != result.Currency {
			continue
		}
		refundAmount := calculateGatewayRefundAmount(
			order.Amount,
			order.PayAmount,
			order.RefundAmount,
			result.Currency,
		)
		result.Refunded += refundAmount
		addUserPaymentAmount(result, order.OrderType, -refundAmount)
	}

	grants, err := s.entClient.RedeemCode.Query().
		Where(
			redeemcode.UsedByEQ(userID),
			redeemcode.TypeEQ(AdjustmentTypeAdminBalance),
			redeemcode.ValueGT(0),
			redeemcode.UsedAtNotNil(),
			redeemcode.UsedAtGTE(startTime),
			redeemcode.UsedAtLT(endTime),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, grant := range grants {
		result.PlatformGranted += grant.Value
	}

	result.GrossPaid = roundAmount(result.GrossPaid)
	result.Refunded = roundAmount(result.Refunded)
	result.NetPaid = roundAmount(result.GrossPaid - result.Refunded)
	result.SubscriptionPaid = roundAmount(result.SubscriptionPaid)
	result.AddonPaid = roundAmount(result.AddonPaid)
	result.BalancePaid = roundAmount(result.BalancePaid)
	result.PlatformGranted = roundAmount(result.PlatformGranted)
	return result, nil
}

func addUserPaymentAmount(summary *UserPaymentSummary, orderType string, amount float64) {
	switch orderType {
	case payment.OrderTypeSubscription:
		summary.SubscriptionPaid += amount
	case payment.OrderTypeAddon:
		summary.AddonPaid += amount
	case payment.OrderTypeBalance:
		summary.BalancePaid += amount
	}
}

func (s *PaymentService) GetDashboardStats(ctx context.Context, days int) (*DashboardStats, error) {
	if days <= 0 {
		days = 30
	}
	now := time.Now()
	since := now.AddDate(0, 0, -days)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	paidStatuses := []string{OrderStatusCompleted, OrderStatusPaid, OrderStatusRecharging}

	orders, err := s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.StatusIn(paidStatuses...),
			paymentorder.PaymentTypeNEQ(PaymentTypeBalanceWallet),
			paymentorder.PaidAtGTE(since),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	st := &DashboardStats{}
	computeBasicStats(st, orders, todayStart)

	st.PendingOrders, err = s.entClient.PaymentOrder.Query().
		Where(paymentorder.StatusEQ(OrderStatusPending)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	st.DailySeries = buildDailySeries(orders, since, days)
	st.PaymentMethods = buildMethodDistribution(orders)
	st.TopUsers = buildTopUsers(orders)

	return st, nil
}

func computeBasicStats(st *DashboardStats, orders []*dbent.PaymentOrder, todayStart time.Time) {
	st.TotalAmount = make(CurrencyAmounts)
	st.TodayAmount = make(CurrencyAmounts)
	st.AvgAmount = make(CurrencyAmounts)
	currencyCounts := make(map[string]int)
	var todayCount int
	for _, o := range orders {
		currency := PaymentOrderCurrency(o)
		st.TotalAmount[currency] += o.PayAmount
		currencyCounts[currency]++
		if o.PaidAt != nil && !o.PaidAt.Before(todayStart) {
			st.TodayAmount[currency] += o.PayAmount
			todayCount++
		}
	}
	st.TotalCount = len(orders)
	st.TodayCount = todayCount
	for currency, totalAmount := range st.TotalAmount {
		st.AvgAmount[currency] = roundAmount(totalAmount / float64(currencyCounts[currency]))
	}
	roundCurrencyAmounts(st.TotalAmount)
	roundCurrencyAmounts(st.TodayAmount)
}

func buildDailySeries(orders []*dbent.PaymentOrder, since time.Time, days int) []DailyStats {
	dailyMap := make(map[string]*DailyStats)
	for _, o := range orders {
		if o.PaidAt == nil {
			continue
		}
		date := o.PaidAt.Format("2006-01-02")
		ds, ok := dailyMap[date]
		if !ok {
			ds = &DailyStats{Date: date, Amount: make(CurrencyAmounts)}
			dailyMap[date] = ds
		}
		ds.Amount[PaymentOrderCurrency(o)] += o.PayAmount
		ds.Count++
	}
	series := make([]DailyStats, 0, days)
	for i := 0; i < days; i++ {
		date := since.AddDate(0, 0, i+1).Format("2006-01-02")
		if ds, ok := dailyMap[date]; ok {
			roundCurrencyAmounts(ds.Amount)
			series = append(series, *ds)
		} else {
			series = append(series, DailyStats{Date: date, Amount: make(CurrencyAmounts)})
		}
	}
	return series
}

func buildMethodDistribution(orders []*dbent.PaymentOrder) []PaymentMethodStat {
	methodMap := make(map[string]*PaymentMethodStat)
	for _, o := range orders {
		ms, ok := methodMap[o.PaymentType]
		if !ok {
			ms = &PaymentMethodStat{Type: o.PaymentType, Amount: make(CurrencyAmounts)}
			methodMap[o.PaymentType] = ms
		}
		ms.Amount[PaymentOrderCurrency(o)] += o.PayAmount
		ms.Count++
	}
	methods := make([]PaymentMethodStat, 0, len(methodMap))
	for _, ms := range methodMap {
		roundCurrencyAmounts(ms.Amount)
		methods = append(methods, *ms)
	}
	sort.Slice(methods, func(i, j int) bool {
		return methods[i].Type < methods[j].Type
	})
	return methods
}

func buildTopUsers(orders []*dbent.PaymentOrder) TopUsersByCurrency {
	userMap := make(map[string]map[int64]*TopUserStat)
	for _, o := range orders {
		currency := PaymentOrderCurrency(o)
		users, ok := userMap[currency]
		if !ok {
			users = make(map[int64]*TopUserStat)
			userMap[currency] = users
		}
		us, ok := users[o.UserID]
		if !ok {
			us = &TopUserStat{UserID: o.UserID, Email: o.UserEmail}
			users[o.UserID] = us
		}
		us.Amount += o.PayAmount
	}
	result := make(TopUsersByCurrency, len(userMap))
	for currency, users := range userMap {
		userList := make([]*TopUserStat, 0, len(users))
		for _, us := range users {
			us.Amount = roundAmount(us.Amount)
			userList = append(userList, us)
		}
		sort.Slice(userList, func(i, j int) bool {
			return userList[i].Amount > userList[j].Amount
		})
		limit := topUsersLimit
		if len(userList) < limit {
			limit = len(userList)
		}
		result[currency] = make([]TopUserStat, 0, limit)
		for i := 0; i < limit; i++ {
			result[currency] = append(result[currency], *userList[i])
		}
	}
	return result
}

func roundCurrencyAmounts(amounts CurrencyAmounts) {
	for currency, amount := range amounts {
		amounts[currency] = roundAmount(amount)
	}
}

func roundAmount(amount float64) float64 {
	return math.Round(amount*100) / 100
}

// --- Audit Logs ---

func (s *PaymentService) writeAuditLog(ctx context.Context, oid int64, action, op string, detail map[string]any) {
	dj, _ := json.Marshal(detail)
	client := s.entClient
	if tx := dbent.TxFromContext(ctx); tx != nil {
		client = tx.Client()
	}
	_, err := client.PaymentAuditLog.Create().SetOrderID(strconv.FormatInt(oid, 10)).SetAction(action).SetDetail(string(dj)).SetOperator(op).Save(ctx)
	if err != nil {
		slog.Error("audit log failed", "orderID", oid, "action", action, "error", err)
	}
}

func (s *PaymentService) GetOrderAuditLogs(ctx context.Context, oid int64) ([]*dbent.PaymentAuditLog, error) {
	return s.entClient.PaymentAuditLog.Query().Where(paymentauditlog.OrderIDEQ(strconv.FormatInt(oid, 10))).Order(paymentauditlog.ByCreatedAt()).All(ctx)
}
