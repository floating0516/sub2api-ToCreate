<template>
  <AppLayout>
    <div class="dashboard-page">
      <div v-if="errorMessage" class="dashboard-error" role="alert">
        <Icon name="exclamationCircle" size="sm" />
        <span>{{ errorMessage }}</span>
        <button type="button" @click="refreshDashboard">{{ t('dashboard.overview.retry') }}</button>
      </div>

      <section class="dashboard-metric-grid" :aria-label="t('dashboard.overview.coreMetrics')">
        <article v-for="metric in metrics" :key="metric.key" class="dashboard-card dashboard-metric-card">
          <p class="dashboard-metric-label">{{ metric.label }}</p>
          <div v-if="loadingOverview" class="dashboard-value-skeleton" />
          <p v-else class="dashboard-metric-value" :title="metric.value">{{ metric.value }}</p>
          <p v-if="!loadingOverview" class="dashboard-metric-scope">{{ metric.scope }}</p>
          <div v-else class="dashboard-trend-skeleton" />
          <dl
            v-if="!loadingOverview"
            class="dashboard-metric-details"
            :class="`dashboard-metric-details-${metric.details.length}`"
          >
            <div v-for="detail in metric.details" :key="detail.label">
              <dt>{{ detail.label }}</dt>
              <dd :title="detail.value">{{ detail.value }}</dd>
            </div>
          </dl>
        </article>
      </section>

      <section class="dashboard-card dashboard-calendar-card">
        <DashboardUsageCalendar
          :data="calendarData"
          :start-date="calendarStartDate"
          :end-date="calendarEndDate"
          :total-tokens="dashboardStats?.total_tokens || 0"
          :loading="loadingCalendar"
        />
      </section>

      <section class="dashboard-card dashboard-trend-card">
        <header class="dashboard-chart-header">
          <div class="dashboard-chart-heading">
            <div class="dashboard-title-row">
              <h2>{{ t('dashboard.tokenUsageTrend') }}</h2>
              <span class="dashboard-info" :title="t('dashboard.overview.trendInfo')">
                <Icon name="infoCircle" size="sm" />
              </span>
              <span class="dashboard-chart-scope">{{ chartScopeLabel }}</span>
            </div>
            <div class="dashboard-legend" aria-label="Legend">
              <span v-for="item in chartSeries" :key="item.label" :title="item.label">
                <i :style="{ backgroundColor: item.color }" />
                <b>{{ item.label }}</b>
              </span>
            </div>
          </div>

          <div class="dashboard-chart-controls">
            <DashboardDateRangePicker
              v-if="granularity === 'day'"
              v-model:start-date="startDate"
              v-model:end-date="endDate"
              class="dashboard-trend-date"
              compact
              @change="loadTrendSeries"
            />
            <DashboardDateRangePicker
              v-else
              :start-date="hourDate"
              :end-date="hourDate"
              class="dashboard-trend-date"
              mode="single"
              compact
              @change="onHourDateChange"
            />

            <div class="dashboard-segmented" role="group" :aria-label="t('dashboard.granularity')">
              <button
                type="button"
                :class="granularity === 'day' && 'active'"
                :aria-pressed="granularity === 'day'"
                @click="setGranularity('day')"
              >
                {{ t('dashboard.overview.daily') }}
              </button>
              <button
                type="button"
                :class="granularity === 'hour' && 'active'"
                :aria-pressed="granularity === 'hour'"
                @click="setGranularity('hour')"
              >
                {{ t('dashboard.overview.hourly') }}
              </button>
            </div>

            <label class="dashboard-select-shell dashboard-group-select">
              <select v-model="groupMode" :aria-label="t('dashboard.overview.groupBy')" @change="onGroupModeChange">
                <option value="model">{{ t('dashboard.overview.groupByModel') }}</option>
                <option value="api_key">{{ t('dashboard.overview.groupByApiKey') }}</option>
              </select>
              <Icon name="chevronDown" size="sm" />
            </label>

            <label v-if="groupMode === 'api_key'" class="dashboard-select-shell dashboard-key-select">
              <select v-model="selectedApiKeyID" :aria-label="t('dashboard.overview.selectApiKey')" @change="loadTrendSeries">
                <option :value="null">{{ t('dashboard.overview.allApiKeys') }}</option>
                <option v-for="key in apiKeys" :key="key.id" :value="key.id">
                  {{ apiKeyLabel(key) }}
                </option>
              </select>
              <Icon name="chevronDown" size="sm" />
            </label>
          </div>
        </header>

        <DashboardTrendChart :labels="chartLabels" :series="chartSeries" :loading="loadingChart" />
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import DashboardDateRangePicker from '@/components/user/dashboard/DashboardDateRangePicker.vue'
import DashboardUsageCalendar, {
  type DashboardCalendarPoint
} from '@/components/user/dashboard/DashboardUsageCalendar.vue'
import DashboardTrendChart, {
  type DashboardTrendSeries
} from '@/components/user/dashboard/DashboardTrendChart.vue'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import { paymentAPI } from '@/api/payment'
import { keysAPI } from '@/api/keys'
import { useAuthStore } from '@/stores/auth'
import { maskQuickStartKey } from '@/utils/quickstart'
import type {
  ApiKey,
  ModelTrendPoint,
  TrendDataPoint
} from '@/types'
import type { UserPaymentSummary } from '@/types/payment'
import { formatDateLocalInput } from '@/utils/format'

type Granularity = 'day' | 'hour'
type GroupMode = 'model' | 'api_key'

interface MetricItem {
  key: string
  label: string
  value: string
  scope: string
  details: MetricDetail[]
}

interface MetricDetail {
  label: string
  value: string
}

const MODEL_COLOR_PALETTE = [
  '#22a06b',
  '#3b82f6',
  '#8b5cf6',
  '#d97738',
  '#0891b2',
  '#d14f7a',
  '#b8870b',
  '#4f6f8f',
  '#dc5555',
  '#0f9f8f',
  '#6366f1',
  '#729b24'
]
const MAX_CHART_MODEL_SERIES = 8
const API_KEY_COLOR = '#22a06b'
const DAY_MS = 86_400_000
const CALENDAR_DAY_COUNT = 365

const { t, locale } = useI18n()
const authStore = useAuthStore()
const endDate = ref(formatDateLocalInput(new Date()))
const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * DAY_MS)))
const hourDate = ref(endDate.value)
const granularity = ref<Granularity>('day')
const groupMode = ref<GroupMode>('model')
const selectedApiKeyID = ref<number | null>(null)

const dashboardStats = ref<UserDashboardStats | null>(null)
const paymentSummary = ref<UserPaymentSummary | null>(null)
const apiKeys = ref<ApiKey[]>([])
const chartLabels = ref<string[]>([])
const chartSeries = ref<DashboardTrendSeries[]>([])
const calendarData = ref<DashboardCalendarPoint[]>([])
const loadingOverview = ref(true)
const loadingChart = ref(true)
const loadingCalendar = ref(true)
const errorMessage = ref('')
let chartRequestID = 0
let calendarRequestID = 0
let apiKeysPromise: Promise<void> | null = null

const numberLocale = computed(() => (locale.value.startsWith('zh') ? 'zh-CN' : 'en-US'))
const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone

const addDays = (value: string, days: number): string => {
  const date = new Date(`${value}T00:00:00`)
  date.setDate(date.getDate() + days)
  return formatDateLocalInput(date)
}

const inclusiveDayCount = (start: string, end: string): number => {
  const startTime = new Date(`${start}T00:00:00`).getTime()
  const endTime = new Date(`${end}T00:00:00`).getTime()
  return Math.max(1, Math.round((endTime - startTime) / DAY_MS) + 1)
}

const calendarEndDate = ref(formatDateLocalInput(new Date()))
const calendarStartDate = computed(() => addDays(calendarEndDate.value, -(CALENDAR_DAY_COUNT - 1)))
const chartScopeLabel = computed(() => {
  if (granularity.value === 'hour') {
    return t('dashboard.overview.hourlyScope', { date: hourDate.value })
  }
  return t('dashboard.overview.dailyScope')
})

const accountStartDate = computed(() => {
  const createdAt = authStore.user?.created_at
  return createdAt ? formatDateLocalInput(new Date(createdAt)) : '1970-01-01'
})

const accountAgeDays = computed(() => inclusiveDayCount(accountStartDate.value, calendarEndDate.value))

const formatCNY = (value: number): string =>
  new Intl.NumberFormat(numberLocale.value, {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(value || 0)

const formatTokens = (value: number): string => {
  const absolute = Math.abs(value || 0)
  if (absolute >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (absolute >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return Math.round(value || 0).toLocaleString(numberLocale.value)
}

const formatRequests = (value: number): string => {
  const absolute = Math.abs(value || 0)
  if (absolute >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (absolute >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return Math.round(value || 0).toLocaleString(numberLocale.value)
}

const formatDuration = (value: number): string => {
  if (!value) return '0.00s'
  if (value < 1000) return `${Math.round(value)}ms`
  return `${(value / 1000).toFixed(2)}s`
}

const apiKeyLabel = (key: ApiKey): string => `${key.name} · ${maskQuickStartKey(key.key)}`

const hashModelName = (name: string): number => {
  let hash = 2166136261
  for (let index = 0; index < name.length; index += 1) {
    hash ^= name.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return hash >>> 0
}

const metrics = computed<MetricItem[]>(() => {
  const current = dashboardStats.value
  const requests = current?.total_requests || 0
  return [
    {
      key: 'cost',
      label: t('dashboard.overview.accountCredit'),
      value: formatCNY(authStore.user?.balance || 0),
      scope: t('dashboard.overview.currentAvailable'),
      details: [
        {
          label: t('dashboard.overview.lifetimeActualPayment'),
          value: formatCNY(paymentSummary.value?.net_paid || 0)
        },
        {
          label: t('dashboard.overview.balanceRecharges'),
          value: formatCNY(paymentSummary.value?.balance_paid || 0)
        },
        {
          label: t('dashboard.overview.platformGranted'),
          value: formatCNY(paymentSummary.value?.platform_granted || 0)
        }
      ]
    },
    {
      key: 'tokens',
      label: t('dashboard.totalTokens'),
      value: formatTokens(current?.total_tokens || 0),
      scope: t('dashboard.overview.accountLifetimeScope'),
      details: [
        {
          label: t('dashboard.overview.inputTokens'),
          value: formatTokens(current?.total_input_tokens || 0)
        },
        {
          label: t('dashboard.overview.outputTokens'),
          value: formatTokens(current?.total_output_tokens || 0)
        },
        {
          label: t('dashboard.overview.cacheTokens'),
          value: formatTokens(
            (current?.total_cache_creation_tokens || 0) +
            (current?.total_cache_read_tokens || 0)
          )
        }
      ]
    },
    {
      key: 'requests',
      label: t('dashboard.overview.totalRequests'),
      value: formatRequests(requests),
      scope: t('dashboard.overview.accountLifetimeScope'),
      details: [
        {
          label: t('dashboard.overview.dailyAverageRequests'),
          value: formatRequests(requests / accountAgeDays.value)
        },
        {
          label: t('dashboard.overview.tokensPerRequest'),
          value: formatTokens(requests ? (current?.total_tokens || 0) / requests : 0)
        }
      ]
    },
    {
      key: 'duration',
      label: t('dashboard.overview.averageResponseTime'),
      value: formatDuration(current?.average_duration_ms || 0),
      scope: t('dashboard.overview.accountLifetimeScope'),
      details: [
        {
          label: t('dashboard.overview.currentRPM'),
          value: formatRequests(dashboardStats.value?.rpm || 0)
        },
        {
          label: t('dashboard.overview.currentTPM'),
          value: formatTokens(dashboardStats.value?.tpm || 0)
        }
      ]
    }
  ]
})

const buildModelColors = (modelNames: string[]): Record<string, string> => {
  const names = [...new Set(modelNames)].sort((left, right) => left.localeCompare(right))
  const colors: Record<string, string> = {}
  const usedIndexes = new Set<number>()

  names.forEach((name) => {
    const startIndex = hashModelName(name) % MODEL_COLOR_PALETTE.length
    let colorIndex = startIndex
    for (let offset = 0; offset < MODEL_COLOR_PALETTE.length; offset += 1) {
      const candidate = (startIndex + offset) % MODEL_COLOR_PALETTE.length
      if (!usedIndexes.has(candidate)) {
        colorIndex = candidate
        break
      }
    }
    usedIndexes.add(colorIndex)
    colors[name] = MODEL_COLOR_PALETTE[colorIndex]
  })

  return colors
}

const buildBucketKeys = (): string[] => {
  if (granularity.value === 'hour') {
    return Array.from({ length: 24 }, (_, hour) => `${hourDate.value} ${String(hour).padStart(2, '0')}:00`)
  }
  const keys: string[] = []
  let cursor = startDate.value
  while (cursor <= endDate.value) {
    keys.push(cursor)
    cursor = addDays(cursor, 1)
  }
  return keys
}

const formatBucketLabel = (bucket: string): string => {
  if (granularity.value === 'hour') return bucket.slice(11)
  const [, month, day] = bucket.split('-')
  return locale.value.startsWith('zh') ? `${month}月${day}日` : `${month}/${day}`
}

const normalizeTrend = (trend: TrendDataPoint[], buckets: string[]): number[] => {
  const values = new Map(trend.map((point) => [point.date, point.total_tokens]))
  return buckets.map((bucket) => values.get(bucket) || 0)
}

const trendParams = (extra: { model?: string; api_key_id?: number } = {}) => ({
  start_date: granularity.value === 'hour' ? hourDate.value : startDate.value,
  end_date: granularity.value === 'hour' ? hourDate.value : endDate.value,
  granularity: granularity.value,
  timezone: browserTimezone,
  ...extra
})

const normalizeModelTrend = (
  trend: ModelTrendPoint[],
  model: string,
  buckets: string[]
): number[] => {
  const values = new Map(
    trend.filter((point) => point.model === model).map((point) => [point.date, point.total_tokens])
  )
  return buckets.map((bucket) => values.get(bucket) || 0)
}

const loadModelTrendSeries = async (requestID: number, buckets: string[]) => {
  const response = await usageAPI.getDashboardModelTrends({
    ...trendParams(),
    limit: MAX_CHART_MODEL_SERIES
  })
  if (requestID !== chartRequestID) return

  const trend = response.trend || []
  const rankedModels = [
    ...new Map(trend.map((point) => [point.model, point.rank] as const)).entries()
  ]
    .sort((left, right) => left[1] - right[1])
    .map(([model]) => model)
  const colors = buildModelColors(rankedModels)
  chartSeries.value = rankedModels.map((model, index) => ({
    label: model,
    color: colors[model] || MODEL_COLOR_PALETTE[index],
    values: normalizeModelTrend(trend, model, buckets)
  }))
}

const loadCalendarUsage = async () => {
  const requestID = ++calendarRequestID
  loadingCalendar.value = true
  try {
    const response = await usageAPI.getDashboardTrend({
      start_date: calendarStartDate.value,
      end_date: calendarEndDate.value,
      granularity: 'day',
      timezone: browserTimezone
    })
    if (requestID !== calendarRequestID) return
    calendarData.value = (response.trend || []).map((point) => ({
      day: point.date,
      value: point.total_tokens
    }))
  } catch (error) {
    if (requestID !== calendarRequestID) return
    console.error('Failed to load dashboard calendar:', error)
    calendarData.value = []
  } finally {
    if (requestID === calendarRequestID) loadingCalendar.value = false
  }
}

const ensureApiKeys = async (): Promise<void> => {
  if (apiKeys.value.length > 0) return
  if (apiKeysPromise) return apiKeysPromise
  apiKeysPromise = keysAPI.list(1, 100)
    .then((response) => {
      apiKeys.value = response.items || []
    })
    .finally(() => {
      apiKeysPromise = null
    })
  return apiKeysPromise
}

const loadApiKeyTrendSeries = async (requestID: number, buckets: string[]) => {
  await ensureApiKeys()
  if (requestID !== chartRequestID) return
  const selectedKey = apiKeys.value.find((key) => key.id === selectedApiKeyID.value)
  const response = await usageAPI.getDashboardTrend(
    trendParams(selectedKey ? { api_key_id: selectedKey.id } : {})
  )
  if (requestID !== chartRequestID) return
  chartSeries.value = [{
    label: selectedKey ? apiKeyLabel(selectedKey) : t('dashboard.overview.allApiKeys'),
    color: API_KEY_COLOR,
    values: normalizeTrend(response.trend || [], buckets)
  }]
}

const loadTrendSeries = async () => {
  const requestID = ++chartRequestID
  loadingChart.value = true
  const buckets = buildBucketKeys()
  chartLabels.value = buckets.map(formatBucketLabel)
  try {
    if (groupMode.value === 'api_key') {
      await loadApiKeyTrendSeries(requestID, buckets)
    } else {
      await loadModelTrendSeries(requestID, buckets)
    }
  } catch (error) {
    if (requestID !== chartRequestID) return
    console.error('Failed to load dashboard trend:', error)
    chartSeries.value = []
    errorMessage.value = t('dashboard.overview.loadFailed')
  } finally {
    if (requestID === chartRequestID) loadingChart.value = false
  }
}

const loadOverview = async () => {
  loadingOverview.value = true
  try {
    const [dashboard, currentPayment] =
      await Promise.all([
        usageAPI.getDashboardStats({ summary_only: true }),
        paymentAPI.getSummary({
          start_date: accountStartDate.value,
          end_date: calendarEndDate.value,
          timezone: browserTimezone
        }),
        authStore.refreshUser()
      ])
    dashboardStats.value = dashboard
    paymentSummary.value = currentPayment
  } catch (error) {
    console.error('Failed to load dashboard:', error)
    errorMessage.value = t('dashboard.overview.loadFailed')
  } finally {
    loadingOverview.value = false
  }
}

const refreshDashboard = async () => {
  errorMessage.value = ''
  await Promise.allSettled([
    loadOverview(),
    loadTrendSeries(),
    loadCalendarUsage()
  ])
}

const setGranularity = async (value: Granularity) => {
  if (granularity.value === value) return
  granularity.value = value
  await loadTrendSeries()
}

const onHourDateChange = async (value: { startDate: string; endDate: string }) => {
  hourDate.value = value.endDate
  await loadTrendSeries()
}

const onGroupModeChange = async () => {
  if (groupMode.value === 'model') selectedApiKeyID.value = null
  await loadTrendSeries()
}

onMounted(refreshDashboard)
</script>

<style scoped>
.dashboard-page {
  --dashboard-border: #e4e7eb;
  --dashboard-text: #111318;
  --dashboard-muted: #6b7280;
  --dashboard-subtle: #9ca3af;
  --dashboard-surface: #fff;
  --dashboard-surface-subtle: #f7f8f9;
  --dashboard-surface-active: #fff;
  --dashboard-divider: #eff1f3;
  --dashboard-skeleton: #f0f1f3;
  display: grid;
  gap: 20px;
  width: 100%;
  color: var(--dashboard-text);
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Microsoft YaHei", sans-serif;
  letter-spacing: 0;
}

.dashboard-error {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  border: 1px solid #fecaca;
  border-radius: 8px;
  background: #fff7f7;
  padding: 8px 12px;
  color: #b42318;
  font-size: 13px;
}

.dashboard-error button {
  margin-left: auto;
  font-weight: 600;
}

.dashboard-card {
  border: 1px solid var(--dashboard-border);
  border-radius: 12px;
  background: var(--dashboard-surface);
  box-shadow: 0 1px 2px rgb(17 24 39 / 2%);
}

.dashboard-metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 20px;
}

.dashboard-metric-card {
  display: flex;
  height: 160px;
  min-width: 0;
  flex-direction: column;
  overflow: hidden;
  padding: 17px 20px 14px;
}

.dashboard-metric-label {
  color: var(--dashboard-muted);
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
}

.dashboard-metric-value {
  margin-top: 5px;
  overflow: hidden;
  color: var(--dashboard-text);
  font-size: 28px;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  line-height: 34px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-metric-scope {
  margin-top: 2px;
  color: var(--dashboard-subtle);
  font-size: 12px;
  line-height: 17px;
}

.dashboard-metric-details {
  display: grid;
  min-width: 0;
  gap: 0;
  margin: auto 0 0;
  border-top: 1px solid var(--dashboard-divider);
  padding-top: 9px;
}

.dashboard-metric-details-2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.dashboard-metric-details-3 {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.dashboard-metric-details > div {
  min-width: 0;
  padding-right: 10px;
}

.dashboard-metric-details > div + div {
  border-left: 1px solid var(--dashboard-divider);
  padding-left: 10px;
}

.dashboard-metric-details dt {
  overflow: hidden;
  color: var(--dashboard-subtle);
  font-size: 10px;
  font-weight: 500;
  line-height: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-metric-details dd {
  margin: 1px 0 0;
  overflow: hidden;
  color: var(--dashboard-body-text);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-value-skeleton,
.dashboard-trend-skeleton {
  border-radius: 5px;
  background: var(--dashboard-skeleton);
  animation: dashboard-pulse 1.4s ease-in-out infinite;
}

.dashboard-value-skeleton {
  width: 68%;
  height: 30px;
  margin-top: 10px;
}

.dashboard-trend-skeleton {
  width: 48%;
  height: 12px;
  margin-top: 8px;
}

.dashboard-calendar-card,
.dashboard-trend-card {
  min-width: 0;
  padding: 22px 24px 24px;
}

.dashboard-calendar-card {
  overflow: hidden;
  padding-bottom: 12px;
}

.dashboard-chart-header {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.dashboard-chart-heading {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
}

.dashboard-title-row {
  display: flex;
  align-items: center;
  gap: 7px;
}

.dashboard-title-row h2 {
  color: var(--dashboard-text);
  font-size: 15px;
  font-weight: 650;
  line-height: 22px;
}

.dashboard-info {
  display: inline-flex;
  color: #a1a7b0;
}

.dashboard-chart-scope {
  color: var(--dashboard-subtle);
  font-size: 12px;
}

.dashboard-chart-scope {
  margin-left: 4px;
  border-left: 1px solid #e5e7eb;
  padding-left: 10px;
}

.dashboard-legend {
  display: flex;
  min-width: 0;
  flex-wrap: nowrap;
  gap: 8px 18px;
  margin-top: 12px;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
}

.dashboard-legend::-webkit-scrollbar {
  display: none;
}

.dashboard-legend span {
  display: flex;
  flex: 0 0 auto;
  min-width: 0;
  max-width: 220px;
  align-items: center;
  gap: 7px;
  color: var(--dashboard-muted);
  font-size: 12px;
}

.dashboard-legend i {
  flex: 0 0 auto;
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.dashboard-legend b {
  overflow: hidden;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-chart-controls {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  white-space: nowrap;
}

.dashboard-trend-date {
  flex: 0 0 auto;
}

.dashboard-segmented {
  display: flex;
  height: 38px;
  align-items: center;
  gap: 2px;
  border: 1px solid #e2e5e9;
  border-radius: 8px;
  background: var(--dashboard-surface-subtle);
  padding: 3px;
}

.dashboard-segmented button {
  height: 30px;
  min-width: 48px;
  border-radius: 6px;
  padding: 0 10px;
  color: #747b85;
  font-size: 12px;
  font-weight: 600;
}

.dashboard-segmented button.active {
  background: var(--dashboard-surface-active);
  color: var(--dashboard-text);
  box-shadow: 0 1px 2px rgb(17 24 39 / 8%);
}

.dashboard-select-shell {
  position: relative;
  display: block;
  height: 38px;
}

.dashboard-select-shell select {
  width: 100%;
  height: 100%;
  overflow: hidden;
  appearance: none;
  border: 1px solid #e2e5e9;
  border-radius: 8px;
  background: var(--dashboard-surface);
  padding: 0 32px 0 11px;
  color: #4b525c;
  font-size: 12px;
  font-weight: 500;
  outline: none;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-select-shell select:hover,
.dashboard-select-shell select:focus {
  border-color: #cbd0d7;
}

.dashboard-select-shell > svg {
  position: absolute;
  top: 11px;
  right: 10px;
  pointer-events: none;
  color: #9298a1;
}

.dashboard-group-select {
  width: 150px;
}

.dashboard-key-select {
  width: 230px;
}

@keyframes dashboard-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}

:global(html.dark .dashboard-page) {
  --dashboard-border: #2d3745;
  --dashboard-text: #f3f4f6;
  --dashboard-muted: #a6adb7;
  --dashboard-subtle: #7f8792;
  --dashboard-surface: #111827;
  --dashboard-surface-subtle: #172130;
  --dashboard-surface-active: #263142;
  --dashboard-divider: #283342;
  --dashboard-skeleton: #263142;
  color-scheme: dark;
}

:global(html.dark .dashboard-card) {
  border-color: var(--dashboard-border);
  background: var(--dashboard-surface);
  box-shadow: none;
}

:global(html.dark .dashboard-segmented) {
  border-color: #374151;
}

:global(html.dark .dashboard-segmented button.active) {
  color: #f9fafb;
}

:global(html.dark .dashboard-select-shell select) {
  border-color: #374151;
  background: var(--dashboard-surface);
  color: #d1d5db;
}

:global(html.dark .dashboard-select-shell select:hover),
:global(html.dark .dashboard-select-shell select:focus) {
  border-color: #4b5563;
}

:global(html.dark .dashboard-chart-scope) {
  border-color: var(--dashboard-divider);
}

:global(html.dark .dashboard-error) {
  border-color: #7f1d1d;
  background: #2b171b;
  color: #fca5a5;
}

@media (min-width: 1181px) and (max-height: 1050px) {
  .dashboard-page {
    gap: 14px;
    margin-block: -8px;
  }

  .dashboard-metric-grid {
    gap: 14px;
  }

  .dashboard-metric-card {
    height: 148px;
    padding: 14px 18px 12px;
  }

  .dashboard-metric-details {
    padding-top: 7px;
  }

  .dashboard-calendar-card {
    padding: 16px 20px 9px;
  }

  .dashboard-trend-card {
    padding: 16px 20px 18px;
  }

  .dashboard-chart-header {
    margin-bottom: 12px;
  }

  .dashboard-legend {
    margin-top: 8px;
  }

  .dashboard-segmented,
  .dashboard-select-shell {
    height: 36px;
  }

  .dashboard-segmented button {
    height: 28px;
  }

  .dashboard-select-shell > svg {
    top: 10px;
  }
}

@media (min-width: 1181px) and (max-height: 940px) {
  .dashboard-page {
    gap: 12px;
    margin-block: -12px;
  }

  .dashboard-metric-grid {
    gap: 12px;
  }

  .dashboard-metric-card {
    height: 144px;
    padding: 13px 16px 10px;
  }

  .dashboard-calendar-card {
    padding: 14px 18px 8px;
  }

  .dashboard-trend-card {
    padding: 14px 18px 16px;
  }

  .dashboard-chart-header {
    margin-bottom: 10px;
  }

  .dashboard-legend {
    margin-top: 6px;
  }
}

@media (max-width: 1180px) {
  .dashboard-metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dashboard-chart-header {
    flex-direction: column;
  }

  .dashboard-chart-controls {
    width: 100%;
    justify-content: flex-start;
  }
}

@media (max-width: 640px) {
  .dashboard-page {
    gap: 16px;
  }

  .dashboard-metric-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .dashboard-metric-card {
    height: 154px;
  }

  .dashboard-calendar-card,
  .dashboard-trend-card {
    padding: 18px 14px 18px;
  }

  .dashboard-chart-controls {
    display: grid;
    grid-template-columns: minmax(112px, 1fr) minmax(145px, 1fr);
    white-space: normal;
  }

  .dashboard-segmented,
  .dashboard-group-select,
  .dashboard-key-select {
    width: 100%;
  }

  .dashboard-key-select {
    grid-column: 1 / -1;
  }

  .dashboard-trend-date {
    grid-column: 1 / -1;
    width: 100%;
  }

  :deep(.dashboard-trend-date .dashboard-date-trigger) {
    width: 100%;
  }
}
</style>
