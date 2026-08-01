package main

import (
	"log"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type DrainStop func()

func provideDrainStop(
	dashboardAggregation *service.DashboardAggregationService,
	batchImageCleanup *service.BatchImageCleanupService,
	opsMetricsCollector *service.OpsMetricsCollector,
	accountCapacitySampler *service.AccountCapacitySamplerService,
	opsAggregation *service.OpsAggregationService,
	opsAlertEvaluator *service.OpsAlertEvaluatorService,
	opsCleanup *service.OpsCleanupService,
	opsScheduledReport *service.OpsScheduledReportService,
	accountExpiry *service.AccountExpiryService,
	proxyExpiry *service.ProxyExpiryService,
	subscriptionExpiry *service.SubscriptionExpiryService,
	usageCleanup *service.UsageCleanupService,
	idempotencyCleanup *service.IdempotencyCleanupService,
	scheduledTestRunner *service.ScheduledTestRunnerService,
	backupSvc *service.BackupService,
	paymentOrderExpiry *service.PaymentOrderExpiryService,
	channelMonitorRunner *service.ChannelMonitorRunner,
	quotaFlusher *service.UserPlatformQuotaUsageFlusher,
	upstreamBillingProbe *service.UpstreamBillingProbeService,
	ollamaCloudUsage *service.OllamaCloudUsageService,
) DrainStop {
	var once sync.Once
	return func() {
		once.Do(func() {
			service.CancelDelayedBackgroundStarts()

			steps := []struct {
				name string
				stop func()
			}{
				{"DashboardAggregationService", dashboardAggregation.Stop},
				{"BatchImageCleanupService", batchImageCleanup.Stop},
				{"OpsMetricsCollector", opsMetricsCollector.Stop},
				{"AccountCapacitySamplerService", accountCapacitySampler.Stop},
				{"OpsAggregationService", opsAggregation.Stop},
				{"OpsAlertEvaluatorService", opsAlertEvaluator.Stop},
				{"OpsCleanupService", opsCleanup.Stop},
				{"OpsScheduledReportService", opsScheduledReport.Stop},
				{"AccountExpiryService", accountExpiry.Stop},
				{"ProxyExpiryService", proxyExpiry.Stop},
				{"SubscriptionExpiryService", subscriptionExpiry.Stop},
				{"UsageCleanupService", usageCleanup.Stop},
				{"IdempotencyCleanupService", idempotencyCleanup.Stop},
				{"ScheduledTestRunnerService", scheduledTestRunner.Stop},
				{"BackupService", backupSvc.Stop},
				{"PaymentOrderExpiryService", paymentOrderExpiry.Stop},
				{"ChannelMonitorRunner", channelMonitorRunner.Stop},
				{"UserPlatformQuotaUsageFlusher", quotaFlusher.Stop},
				{"UpstreamBillingProbeService", upstreamBillingProbe.Stop},
				{"OllamaCloudUsageService", ollamaCloudUsage.Stop},
			}

			var wg sync.WaitGroup
			for i := range steps {
				step := steps[i]
				wg.Add(1)
				go func() {
					defer wg.Done()
					step.stop()
					log.Printf("[Drain] %s stopped", step.name)
				}()
			}
			wg.Wait()
		})
	}
}
