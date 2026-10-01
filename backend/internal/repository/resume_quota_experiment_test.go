//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestResumeQuotaConcurrency(t *testing.T) {
	// Each goroutine receives its own real SQL transaction through the production services.
	// No outer test transaction, shared connection, or mutex-backed fake repository.
	ctx := context.Background()
	client := testEntClient(t)
	usageLogs := NewUsageLogRepository(client, integrationDB)
	for _, kind := range []string{"bill_same", "bill_distinct", "addon_bill_same", "addon_bill_distinct", "renew_same", "renew_distinct", "addon_order_same", "addon_order_distinct", "reset_renew_consume_daily", "reset_renew_consume_weekly", "reset_renew_consume_monthly"} {
		for _, c := range []int{1, 10, 50} {
			for round := 1; round <= 20; round++ {
				t.Run(fmt.Sprintf("%s/c%d/r%d", kind, c, round), func(t *testing.T) {
					suffix := fmt.Sprintf("resume-%s-%d-%d-%d", kind, c, round, time.Now().UnixNano())
					user := mustCreateUser(t, client, &service.User{Email: suffix + "@example.invalid", PasswordHash: "synthetic"})
					group := mustCreateGroup(t, client, &service.Group{Name: suffix, Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeSubscription})
					now := time.Now().UTC().Truncate(time.Microsecond)
					start := now.Add(-24 * time.Hour)
					expiry := now.Add(60 * 24 * time.Hour)
					entity, err := client.UserSubscription.Create().SetUserID(user.ID).SetGroupID(group.ID).SetStartsAt(start).SetExpiresAt(expiry).SetStatus(service.SubscriptionStatusActive).SetDailyWindowStart(start).SetWeeklyWindowStart(start).SetMonthlyWindowStart(start).Save(ctx)
					require.NoError(t, err)
					key := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, GroupID: &group.ID, Key: "sk-" + suffix, Name: suffix})
					subRepo := NewUserSubscriptionRepository(client)
					groupRepo := NewGroupRepository(client, integrationDB)
					addonRepo := NewSubscriptionAddonRepository(integrationDB)
					subSvc := service.ProvideSubscriptionService(groupRepo, subRepo, nil, client, nil, addonRepo)
					defer subSvc.Stop()
					paySvc := service.NewPaymentService(client, nil, nil, nil, subSvc, nil, nil, groupRepo, nil)
					billRepo := NewUsageBillingRepository(client, integrationDB)
					account := mustCreateAccount(t, client, &service.Account{Name: suffix, Type: service.AccountTypeAPIKey})
					logUsage := func(id string, cost float64, at time.Time) error {
						_, e := usageLogs.Create(ctx, &service.UsageLog{UserID: user.ID, APIKeyID: key.ID, AccountID: account.ID, RequestID: id, Model: "synthetic-fixed-cost", GroupID: &group.ID, SubscriptionID: &entity.ID, ActualCost: cost, TotalCost: cost, CreatedAt: at})
						return e
					}
					resetUsage := subRepo.ResetDailyUsage
					if strings.HasSuffix(kind, "_weekly") {
						resetUsage = subRepo.ResetWeeklyUsage
					}
					if strings.HasSuffix(kind, "_monthly") {
						resetUsage = subRepo.ResetMonthlyUsage
					}
					unique := c
					if strings.HasSuffix(kind, "_same") {
						unique = 1
					}
					orders := make([]*dbent.PaymentOrder, unique)
					if strings.HasPrefix(kind, "renew_") || strings.HasPrefix(kind, "addon_order_") {
						for i := range orders {
							b := client.PaymentOrder.Create().SetUserID(user.ID).SetUserEmail(user.Email).SetUserName("experiment").SetAmount(1.99).SetPayAmount(1.99).SetRechargeCode(fmt.Sprintf("%s-%d", suffix, i)).SetOutTradeNo(fmt.Sprintf("%s-%d", suffix, i)).SetPaymentType("alipay").SetPaymentTradeNo(fmt.Sprintf("synthetic-%s-%d", suffix, i)).SetStatus(service.OrderStatusPaid).SetExpiresAt(expiry).SetClientIP("127.0.0.1").SetSrcHost("experiment.invalid")
							if strings.HasPrefix(kind, "renew_") {
								b.SetOrderType("subscription").SetSubscriptionGroupID(group.ID).SetSubscriptionDays(1)
							} else {
								b.SetOrderType("addon").SetProviderSnapshot(map[string]interface{}{"addon_purchase": map[string]interface{}{"product_id": "1", "product_sku": "experiment-10", "product_name": "Synthetic", "subscription_id": strconv.FormatInt(entity.ID, 10), "group_id": strconv.FormatInt(group.ID, 10), "quota_usd": "10", "price": "1.99", "expires_at": expiry.Format(time.RFC3339Nano)}})
							}
							orders[i], err = b.Save(ctx)
							require.NoError(t, err)
						}
					}
					var pack *service.SubscriptionAddonPack
					if strings.HasPrefix(kind, "addon_bill_") {
						pack = &service.SubscriptionAddonPack{SubscriptionID: entity.ID, UserID: user.ID, GroupID: group.ID, QuotaUSD: 1000, StartsAt: start, ExpiresAt: expiry, Status: service.SubscriptionAddonStatusActive}
						require.NoError(t, addonRepo.Create(ctx, pack))
					}
					if strings.HasPrefix(kind, "reset_renew_consume") {
						_, err = billRepo.Apply(ctx, &service.UsageBillingCommand{RequestID: fmt.Sprintf("resume-%d-old", entity.ID), APIKeyID: key.ID, UserID: user.ID, SubscriptionID: &entity.ID, SubscriptionCost: 7})
						require.NoError(t, err)
						require.NoError(t, logUsage(fmt.Sprintf("resume-%d-old", entity.ID), 7, start))
						require.NoError(t, resetUsage(ctx, entity.ID, &start, now))
					}
					type outcome struct {
						applied   int
						conflicts int
						err       error
					}
					outs := make(chan outcome, c)
					ready := sync.WaitGroup{}
					ready.Add(c)
					gate := make(chan struct{})
					for i := 0; i < c; i++ {
						go func(i int) {
							ready.Done()
							<-gate
							idx := i
							if unique == 1 {
								idx = 0
							}
							o := outcome{}
							for repeat := 0; repeat < 2; repeat++ {
								var e error
								switch {
								case strings.HasPrefix(kind, "renew_"):
									e = paySvc.ExecuteSubscriptionFulfillment(ctx, orders[idx].ID)
								case strings.HasPrefix(kind, "addon_order_"):
									e = paySvc.ExecuteAddonFulfillment(ctx, orders[idx].ID)
								default:
									cmd := &service.UsageBillingCommand{RequestID: fmt.Sprintf("resume-%d-event-%d", entity.ID, idx), APIKeyID: key.ID, UserID: user.ID, SubscriptionID: &entity.ID, SubscriptionCost: 0.125}
									if pack != nil {
										cmd.SubscriptionCost = 0
										cmd.AddonPackID = &pack.ID
										cmd.AddonCost = 0.125
									}
									if strings.HasPrefix(kind, "reset_renew_consume") {
										e = resetUsage(ctx, entity.ID, &start, now)
										if e == nil && repeat == 0 {
											_, e = subSvc.ExtendSubscription(ctx, entity.ID, 1)
										}
									}
									if e == nil {
										var result *service.UsageBillingApplyResult
										result, e = billRepo.Apply(ctx, cmd)
										if result != nil && result.Applied {
											o.applied++
										}
										if e == nil {
											e = logUsage(cmd.RequestID, 0.125, now)
										}
									}
								}
								if e != nil {
									if (kind == "renew_same" || kind == "addon_order_same") && infraerrors.Code(e) == 409 && infraerrors.Message(e) == "order is being processed" {
										o.conflicts++
										break
									}
									o.err = e
									break
								}
							}
							outs <- o
						}(i)
					}
					ready.Wait()
					close(gate)
					applied := 0
					errorsCount := 0
					conflicts := 0
					for i := 0; i < c; i++ {
						o := <-outs
						applied += o.applied
						conflicts += o.conflicts
						if o.err != nil {
							errorsCount++
							t.Errorf("operation: %v", o.err)
						}
					}
					// A processing conflict is only acceptable if the owner already committed,
					// and a later retry succeeds without a second business effect.
					if strings.HasPrefix(kind, "renew_") || strings.HasPrefix(kind, "addon_order_") {
						for _, o := range orders {
							before, e := client.PaymentOrder.Get(ctx, o.ID)
							require.NoError(t, e)
							require.Equal(t, service.OrderStatusCompleted, before.Status)
							if strings.HasPrefix(kind, "renew_") {
								require.NoError(t, paySvc.ExecuteSubscriptionFulfillment(ctx, o.ID))
							} else {
								require.NoError(t, paySvc.ExecuteAddonFulfillment(ctx, o.ID))
							}
						}
					}
					actual, err := subRepo.GetByID(ctx, entity.ID)
					require.NoError(t, err)
					expectedUsage := float64(unique) * 0.125
					delta := 0.0
					duplicate := 0
					missing := 0
					expiryDelta := 0.0
					ledgerDelta := 0.0
					usageLogDelta := 0.0
					switch {
					case strings.HasPrefix(kind, "renew_"):
						expiryDelta = actual.ExpiresAt.Sub(expiry.Add(time.Duration(unique) * 24 * time.Hour)).Seconds()
						require.Equal(t, start, actual.StartsAt)
						require.Equal(t, start, *actual.MonthlyWindowStart)
						var n int
						require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT count(*) FROM payment_audit_logs WHERE action='SUBSCRIPTION_ASSIGNED' AND order_id IN (SELECT id::text FROM payment_orders WHERE user_id=$1)", user.ID).Scan(&n))
						duplicate = max(0, n-unique)
						missing = max(0, unique-n)
					case strings.HasPrefix(kind, "addon_order_"):
						var n int
						var total float64
						require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(quota_usd),0) FROM subscription_addon_packs WHERE subscription_id=$1", entity.ID).Scan(&n, &total))
						delta = total - float64(unique)*10
						duplicate = max(0, n-unique)
						missing = max(0, unique-n)
						_, err = subSvc.ExtendSubscription(ctx, entity.ID, 1)
						require.NoError(t, err)
						packs, e := addonRepo.ListBySubscriptionID(ctx, entity.ID)
						require.NoError(t, e)
						for _, p := range packs {
							require.True(t, p.ExpiresAt.Equal(expiry), "addon expiry must not grow with renewal")
						}
					default:
						duplicate = max(0, applied-unique)
						missing = max(0, unique-applied)
						if pack != nil {
							got, e := addonRepo.GetByID(ctx, pack.ID)
							require.NoError(t, e)
							delta = got.UsedUSD - expectedUsage
							var total float64
							var n int
							require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT coalesce(sum(cost_usd),0),count(*) FROM subscription_addon_usage WHERE addon_pack_id=$1", pack.ID).Scan(&total, &n))
							ledgerDelta = total - expectedUsage
							require.Equal(t, unique, n)
						} else {
							expectedDaily, expectedWeekly, expectedMonthly := expectedUsage, expectedUsage, expectedUsage
							if strings.HasPrefix(kind, "reset_renew_consume") {
								if !strings.HasSuffix(kind, "_daily") {
									expectedDaily += 7
								}
								if !strings.HasSuffix(kind, "_weekly") {
									expectedWeekly += 7
								}
								if !strings.HasSuffix(kind, "_monthly") {
									expectedMonthly += 7
								}
								expiryDelta = actual.ExpiresAt.Sub(expiry.Add(time.Duration(c) * 24 * time.Hour)).Seconds()
							}
							delta = math.Max(math.Abs(actual.DailyUsageUSD-expectedDaily), math.Max(math.Abs(actual.WeeklyUsageUSD-expectedWeekly), math.Abs(actual.MonthlyUsageUSD-expectedMonthly)))
						}
						var n int
						require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT count(*) FROM usage_billing_dedup WHERE api_key_id=$1", key.ID).Scan(&n))
						expectedN := unique
						if strings.HasPrefix(kind, "reset_renew_consume") {
							expectedN++
						}
						require.Equal(t, expectedN, n)
					}
					if !strings.HasPrefix(kind, "renew_") && !strings.HasPrefix(kind, "addon_order_") {
						var total, current float64
						var logCount int
						require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(actual_cost),0),coalesce(sum(actual_cost) FILTER (WHERE created_at >= $2),0) FROM usage_logs WHERE api_key_id=$1", key.ID, now).Scan(&logCount, &total, &current))
						expectedTotal := expectedUsage
						expectedCount := unique
						if strings.HasPrefix(kind, "reset_renew_consume") {
							expectedTotal += 7
							expectedCount++
						}
						usageLogDelta = total - expectedTotal
						require.Equal(t, expectedCount, logCount)
						require.InDelta(t, expectedUsage, current, 0.000001)
					}
					if len(orders) > 0 && (strings.HasPrefix(kind, "renew_") || strings.HasPrefix(kind, "addon_order_")) {
						for _, o := range orders {
							got, e := client.PaymentOrder.Get(ctx, o.ID)
							require.NoError(t, e)
							require.Equal(t, service.OrderStatusCompleted, got.Status)
						}
					}
					row := map[string]interface{}{"scenario": kind, "concurrency": c, "round": round, "duplicate_effects": duplicate, "missing_effects": missing, "quota_delta_usd": delta, "ledger_delta_usd": ledgerDelta, "usage_log_delta_usd": usageLogDelta, "expiry_delta_seconds": expiryDelta, "operation_errors": errorsCount, "inflight_conflict_responses": conflicts}
					raw, _ := json.Marshal(row)
					t.Log("EXPERIMENT_ROW " + string(raw))
					require.Zero(t, duplicate)
					require.Zero(t, missing)
					require.LessOrEqual(t, math.Abs(delta), 0.000001)
					require.LessOrEqual(t, math.Abs(ledgerDelta), 0.000001)
					require.LessOrEqual(t, math.Abs(usageLogDelta), 0.000001)
					require.Zero(t, expiryDelta)
				})
			}
		}
	}
}
