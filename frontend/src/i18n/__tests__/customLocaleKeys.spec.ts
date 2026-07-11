import { describe, expect, it } from 'vitest'

import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

function getPath(obj: Record<string, any>, path: string): unknown {
  return path.split('.').reduce<unknown>((acc, key) => {
    if (acc && typeof acc === 'object' && key in acc) {
      return (acc as Record<string, unknown>)[key]
    }
    return undefined
  }, obj)
}

const requiredCustomKeys = [
  'userSubscriptions.usageTimeline',
  'userSubscriptions.timelineLoading',
  'userSubscriptions.timelineSummary',
  'userSubscriptions.noUsageInWindow',
  'userSubscriptions.usageAmount',
  'userSubscriptions.requests',
  'userSubscriptions.timelineBucketRelative',
  'admin.users.columns.usageOverview',
  'admin.users.columns.modelPreferences',
  'admin.users.last7d',
  'admin.users.last30d',
  'admin.users.usageOverview',
  'admin.dailyReport.title',
  'admin.dailyReport.description',
  'admin.users.modelPreferences',
  'admin.users.requestsToday',
  'admin.users.requests7d',
  'admin.users.requests30d',
  'admin.users.tokens30d',
  'admin.users.cost30d',
  'admin.users.totalCost',
  'admin.users.totalRequests',
  'admin.users.activeDays30d',
  'admin.users.activity',
  'admin.users.basicInfo',
  'admin.users.noModelPreference',
  'admin.users.failedToLoadDetails',
  'admin.accounts.stats.capacityTrend',
  'admin.accounts.stats.capacityPeak',
  'admin.accounts.stats.peakConcurrent',
  'admin.accounts.stats.avgConcurrent',
  'admin.accounts.stats.capacityLimit',
  'admin.accounts.stats.concurrentUsage',
  'admin.accounts.stats.samples',
  'admin.accounts.stats.waitingPeak',
  'payment.methods.balance_wallet',
  'payment.balancePurchase.currentBalance',
  'payment.balancePurchase.deductAmount',
  'payment.balancePurchase.insufficient',
  'payment.balancePurchase.useBalance',
  'payment.balancePurchase.success',
  'payment.subscriptionHint.title',
  'payment.subscriptionHint.samePlan',
  'payment.subscriptionHint.differentPlans',
  'payment.errors.INSUFFICIENT_BALANCE',
]

describe.each([
  ['zh', zh],
  ['en', en],
])('custom locale keys for %s', (_locale, messages) => {
  for (const key of requiredCustomKeys) {
    it(`has ${key}`, () => {
      expect(getPath(messages, key), key).toBeTruthy()
    })
  }
})
