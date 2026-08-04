<template>
  <AppLayout>
    <div class="dashboard-page">
      <div class="dashboard-toolbar">
        <DashboardDateRangePicker
          v-model:start-date="startDate"
          v-model:end-date="endDate"
          @change="refreshDashboard"
        />
      </div>

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
          <p v-if="!loadingOverview" class="dashboard-metric-trend">
            <span :class="metric.comparison.tone">{{ metric.comparison.value }}</span>
            {{ metric.comparison.suffix }}
          </p>
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

      <section class="dashboard-table-grid">
        <article class="dashboard-card dashboard-table-card">
          <header class="dashboard-table-header">
            <h2>{{ t('dashboard.overview.modelTokenUsage') }}</h2>
            <span>{{ selectedRangeLabel }}</span>
          </header>
          <div class="dashboard-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>{{ t('dashboard.model') }}</th>
                  <th>{{ t('dashboard.tokens') }}</th>
                  <th>{{ t('dashboard.overview.share') }}</th>
                  <th class="dashboard-spark-column">{{ t('dashboard.overview.trend') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in modelRows" :key="row.name">
                  <td>
                    <span class="dashboard-entity" :title="row.name">
                      <i :style="{ backgroundColor: row.color }" />
                      <strong>{{ row.name }}</strong>
                    </span>
                  </td>
                  <td>{{ formatTokens(row.tokens) }}</td>
                  <td>{{ formatPercent(row.share) }}</td>
                  <td class="dashboard-spark-column">
                    <DashboardSparkline :values="row.trend" :color="row.color" />
                  </td>
                </tr>
                <tr v-if="!loadingOverview && modelRows.length === 0">
                  <td colspan="4" class="dashboard-empty-row">{{ t('dashboard.noDataAvailable') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <RouterLink to="/usage" class="dashboard-table-action">
            {{ t('dashboard.overview.viewAllModels') }}
            <Icon name="arrowRight" size="sm" />
          </RouterLink>
        </article>

        <article class="dashboard-card dashboard-table-card">
          <header class="dashboard-table-header">
            <h2>{{ t('dashboard.overview.platformRanking') }}</h2>
            <span>{{ t('dashboard.overview.cumulative') }}</span>
          </header>
          <div class="dashboard-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>{{ t('dashboard.overview.platform') }}</th>
                  <th>{{ t('dashboard.overview.spend') }}</th>
                  <th>{{ t('dashboard.tokens') }}</th>
                  <th>{{ t('dashboard.overview.share') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in platformRows" :key="row.platform">
                  <td>
                    <span class="dashboard-entity" :title="platformLabel(row.platform)">
                      <span class="dashboard-platform-icon" :class="`dashboard-platform-${row.platform}`">
                        <Icon v-if="row.isOther" name="grid" size="sm" />
                        <PlatformIcon v-else :platform="asPlatform(row.platform)" size="md" />
                      </span>
                      <strong>{{ platformLabel(row.platform) }}</strong>
                    </span>
                  </td>
                  <td>{{ formatCurrency(row.cost) }}</td>
                  <td>{{ formatTokens(row.tokens) }}</td>
                  <td>{{ formatPercent(row.share) }}</td>
                </tr>
                <tr v-if="!loadingOverview && platformRows.length === 0">
                  <td colspan="4" class="dashboard-empty-row">{{ t('dashboard.noDataAvailable') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <RouterLink to="/usage" class="dashboard-table-action">
            {{ t('dashboard.overview.viewAllPlatforms') }}
            <Icon name="arrowRight" size="sm" />
          </RouterLink>
        </article>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import DashboardDateRangePicker from '@/components/user/dashboard/DashboardDateRangePicker.vue'
import DashboardSparkline from '@/components/user/dashboard/DashboardSparkline.vue'
import DashboardTrendChart, {
  type DashboardTrendSeries
} from '@/components/user/dashboard/DashboardTrendChart.vue'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import { keysAPI } from '@/api/keys'
import { maskQuickStartKey } from '@/utils/quickstart'
import type {
  ApiKey,
  GroupPlatform,
  ModelStat,
  TrendDataPoint,
  UsageStatsResponse
} from '@/types'
import { formatDateLocalInput } from '@/utils/format'

type Granularity = 'day' | 'hour'
type GroupMode = 'model' | 'api_key'

interface Comparison {
  value: string
  suffix: string
  tone: string
}

interface MetricItem {
  key: string
  label: string
  value: string
  comparison: Comparison
  details: MetricDetail[]
}

interface MetricDetail {
  label: string
  value: string
}

interface ModelRow {
  name: string
  tokens: number
  share: number
  color: string
  trend: number[]
}

interface PlatformRow {
  platform: string
  cost: number
  tokens: number
  share: number
  isOther?: boolean
}

const MODEL_COLOR_PALETTE = [
  '#22a06b',
  '#3b82f6',
  '#8b5cf6',
  '#d97738',
  '#0891b2',
  '#d14f7a',
  '#b8870b',
  '#4f6f8f'
]
const API_KEY_COLOR = '#22a06b'
const OTHER_COLOR = '#9ca3af'
const DAY_MS = 86_400_000

const { t, locale } = useI18n()
const endDate = ref(formatDateLocalInput(new Date()))
const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * DAY_MS)))
const granularity = ref<Granularity>('day')
const groupMode = ref<GroupMode>('model')
const selectedApiKeyID = ref<number | null>(null)

const dashboardStats = ref<UserDashboardStats | null>(null)
const rangeStats = ref<UsageStatsResponse | null>(null)
const previousStats = ref<UsageStatsResponse | null>(null)
const modelStats = ref<ModelStat[]>([])
const apiKeys = ref<ApiKey[]>([])
const chartLabels = ref<string[]>([])
const chartSeries = ref<DashboardTrendSeries[]>([])
const modelTrendCache = ref<Record<string, number[]>>({})
const loadingOverview = ref(true)
const loadingChart = ref(true)
const errorMessage = ref('')
let chartRequestID = 0

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

const previousRange = computed(() => {
  const days = inclusiveDayCount(startDate.value, endDate.value)
  const end = addDays(startDate.value, -1)
  return { start: addDays(end, -(days - 1)), end }
})

const selectedRangeLabel = computed(() => `${startDate.value} ～ ${endDate.value}`)
const chartScopeLabel = computed(() => {
  if (granularity.value === 'hour') {
    return t('dashboard.overview.hourlyScope', { date: endDate.value })
  }
  return t('dashboard.overview.dailyScope')
})

const formatCurrency = (value: number): string =>
  new Intl.NumberFormat(numberLocale.value, {
    style: 'currency',
    currency: 'USD',
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

const formatPercent = (value: number): string => `${(value || 0).toFixed(1)}%`

const apiKeyLabel = (key: ApiKey): string => `${key.name} · ${maskQuickStartKey(key.key)}`

const hashModelName = (name: string): number => {
  let hash = 2166136261
  for (let index = 0; index < name.length; index += 1) {
    hash ^= name.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }
  return hash >>> 0
}

const compare = (current: number, previous: number, lowerIsBetter = false): Comparison => {
  if (!previous) {
    return {
      value: t('dashboard.overview.noComparison'),
      suffix: '',
      tone: 'dashboard-trend-neutral'
    }
  }
  const delta = ((current - previous) / Math.abs(previous)) * 100
  const favorable = lowerIsBetter ? delta <= 0 : delta >= 0
  return {
    value: `${delta >= 0 ? '+' : ''}${delta.toFixed(1)}%`,
    suffix: t('dashboard.overview.vsPrevious'),
    tone: favorable ? 'dashboard-trend-positive' : 'dashboard-trend-negative'
  }
}

const metrics = computed<MetricItem[]>(() => {
  const current = rangeStats.value
  const previous = previousStats.value
  const days = inclusiveDayCount(startDate.value, endDate.value)
  const requests = current?.total_requests || 0
  return [
    {
      key: 'cost',
      label: t('dashboard.overview.totalSpend'),
      value: formatCurrency(current?.total_actual_cost || 0),
      comparison: compare(current?.total_actual_cost || 0, previous?.total_actual_cost || 0),
      details: [
        {
          label: t('dashboard.overview.standardSpend'),
          value: formatCurrency(current?.total_cost || 0)
        },
        {
          label: t('dashboard.overview.dailyAverageSpend'),
          value: formatCurrency((current?.total_actual_cost || 0) / days)
        }
      ]
    },
    {
      key: 'tokens',
      label: t('dashboard.totalTokens'),
      value: formatTokens(current?.total_tokens || 0),
      comparison: compare(current?.total_tokens || 0, previous?.total_tokens || 0),
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
          value: formatTokens(current?.total_cache_tokens || 0)
        }
      ]
    },
    {
      key: 'requests',
      label: t('dashboard.overview.totalRequests'),
      value: formatRequests(requests),
      comparison: compare(requests, previous?.total_requests || 0),
      details: [
        {
          label: t('dashboard.overview.dailyAverageRequests'),
          value: formatRequests(requests / days)
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
      comparison: compare(
        current?.average_duration_ms || 0,
        previous?.average_duration_ms || 0,
        true
      ),
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

const sortedModels = computed(() =>
  [...modelStats.value].sort((left, right) => right.total_tokens - left.total_tokens)
)

const visibleModelColors = computed<Record<string, string>>(() => {
  const names = [...new Set(sortedModels.value.slice(0, 3).map((item) => item.model))]
    .sort((left, right) => left.localeCompare(right))
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
})

const modelRows = computed<ModelRow[]>(() => {
  const totalTokens = sortedModels.value.reduce((sum, item) => sum + item.total_tokens, 0)
  const topModels = sortedModels.value.slice(0, 3)
  const rows = topModels.map((item) => ({
    name: item.model,
    tokens: item.total_tokens,
    share: totalTokens ? (item.total_tokens / totalTokens) * 100 : 0,
    color: visibleModelColors.value[item.model] || MODEL_COLOR_PALETTE[0],
    trend: modelTrendCache.value[item.model] || []
  }))
  const others = sortedModels.value.slice(3)
  if (others.length) {
    const otherTokens = others.reduce((sum, item) => sum + item.total_tokens, 0)
    rows.push({
      name: t('dashboard.overview.otherModels'),
      tokens: otherTokens,
      share: totalTokens ? (otherTokens / totalTokens) * 100 : 0,
      color: OTHER_COLOR,
      trend: modelTrendCache.value.__other__ || []
    })
  }
  return rows
})

const platformRows = computed<PlatformRow[]>(() => {
  const items = [...(dashboardStats.value?.by_platform || [])].sort(
    (left, right) => right.total_actual_cost - left.total_actual_cost
  )
  const totalCost = dashboardStats.value?.total_actual_cost || 0
  const rows: PlatformRow[] = items.slice(0, 3).map((item) => ({
    platform: item.platform,
    cost: item.total_actual_cost,
    tokens: item.total_tokens,
    share: totalCost ? (item.total_actual_cost / totalCost) * 100 : 0
  }))
  const others = items.slice(3)
  const listedCost = items.reduce((sum, item) => sum + item.total_actual_cost, 0)
  const unassignedCost = Math.max(0, totalCost - listedCost)
  if (others.length || unassignedCost > 0.0001) {
    const otherCost = others.reduce((sum, item) => sum + item.total_actual_cost, unassignedCost)
    const otherTokens = others.reduce((sum, item) => sum + item.total_tokens, 0)
    rows.push({
      platform: '__other__',
      cost: otherCost,
      tokens: otherTokens,
      share: totalCost ? (otherCost / totalCost) * 100 : 0,
      isOther: true
    })
  }
  return rows
})

const platformLabel = (platform: string): string => {
  const labels: Record<string, string> = {
    openai: 'OpenAI',
    anthropic: 'Claude',
    grok: 'Grok',
    gemini: 'Gemini',
    antigravity: 'Antigravity',
    composite: t('dashboard.overview.compositePlatform'),
    __other__: t('dashboard.overview.otherPlatforms')
  }
  return labels[platform] || platform
}

const asPlatform = (platform: string): GroupPlatform => platform as GroupPlatform

const buildBucketKeys = (): string[] => {
  if (granularity.value === 'hour') {
    return Array.from({ length: 24 }, (_, hour) => `${endDate.value} ${String(hour).padStart(2, '0')}:00`)
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
  start_date: granularity.value === 'hour' ? endDate.value : startDate.value,
  end_date: endDate.value,
  granularity: granularity.value,
  timezone: browserTimezone,
  ...extra
})

const loadModelTrendSeries = async (requestID: number, buckets: string[]) => {
  const models = sortedModels.value.slice(0, 3)
  if (!models.length) {
    const total = await usageAPI.getDashboardTrend(trendParams())
    if (requestID !== chartRequestID) return
    const values = normalizeTrend(total.trend || [], buckets)
    chartSeries.value = [{
      label: t('dashboard.overview.allModels'),
      color: MODEL_COLOR_PALETTE[0],
      values
    }]
    modelTrendCache.value = {}
    return
  }

  const responses = await Promise.all([
    ...models.map((model) => usageAPI.getDashboardTrend(trendParams({ model: model.model }))),
    usageAPI.getDashboardTrend(trendParams())
  ])
  if (requestID !== chartRequestID) return

  const modelValues = responses.slice(0, models.length).map((response) =>
    normalizeTrend(response.trend || [], buckets)
  )
  const totalValues = normalizeTrend(responses[responses.length - 1].trend || [], buckets)
  chartSeries.value = models.map((model, index) => ({
    label: model.model,
    color: visibleModelColors.value[model.model] || MODEL_COLOR_PALETTE[index],
    values: modelValues[index]
  }))

  const cache: Record<string, number[]> = {}
  models.forEach((model, index) => {
    cache[model.model] = modelValues[index]
  })
  cache.__other__ = totalValues.map((value, index) =>
    Math.max(0, value - modelValues.reduce((sum, points) => sum + (points[index] || 0), 0))
  )
  modelTrendCache.value = cache
}

const loadApiKeyTrendSeries = async (requestID: number, buckets: string[]) => {
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

const refreshDashboard = async () => {
  loadingOverview.value = true
  errorMessage.value = ''
  modelTrendCache.value = {}
  try {
    const [dashboard, current, previous, models, keys] = await Promise.all([
      usageAPI.getDashboardStats(),
      usageAPI.getStatsByDateRange(startDate.value, endDate.value),
      usageAPI.getStatsByDateRange(previousRange.value.start, previousRange.value.end),
      usageAPI.getDashboardModels({
        start_date: startDate.value,
        end_date: endDate.value,
        timezone: browserTimezone
      }),
      keysAPI.list(1, 100)
    ])
    dashboardStats.value = dashboard
    rangeStats.value = current
    previousStats.value = previous
    modelStats.value = models.models || []
    apiKeys.value = keys.items || []
  } catch (error) {
    console.error('Failed to load dashboard:', error)
    errorMessage.value = t('dashboard.overview.loadFailed')
  } finally {
    loadingOverview.value = false
  }
  await loadTrendSeries()
}

const setGranularity = async (value: Granularity) => {
  if (granularity.value === value) return
  granularity.value = value
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
  display: grid;
  gap: 20px;
  width: 100%;
  color: var(--dashboard-text);
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Microsoft YaHei", sans-serif;
  letter-spacing: 0;
}

.dashboard-toolbar {
  display: flex;
  min-height: 44px;
  justify-content: flex-end;
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
  background: #fff;
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

.dashboard-metric-trend {
  margin-top: 2px;
  color: var(--dashboard-subtle);
  font-size: 12px;
  line-height: 17px;
}

.dashboard-metric-trend span {
  margin-right: 3px;
  font-weight: 600;
}

.dashboard-trend-positive {
  color: #1b8a5a;
}

.dashboard-trend-negative {
  color: #dc5555;
}

.dashboard-trend-neutral {
  color: var(--dashboard-subtle);
}

.dashboard-metric-details {
  display: grid;
  min-width: 0;
  gap: 0;
  margin: auto 0 0;
  border-top: 1px solid #eff1f3;
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
  border-left: 1px solid #eff1f3;
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
  color: #4f5660;
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
  background: #f0f1f3;
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

.dashboard-trend-card {
  min-width: 0;
  padding: 22px 24px 24px;
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
}

.dashboard-title-row {
  display: flex;
  align-items: center;
  gap: 7px;
}

.dashboard-title-row h2,
.dashboard-table-header h2 {
  color: var(--dashboard-text);
  font-size: 15px;
  font-weight: 650;
  line-height: 22px;
}

.dashboard-info {
  display: inline-flex;
  color: #a1a7b0;
}

.dashboard-chart-scope,
.dashboard-table-header span {
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
  flex-wrap: wrap;
  gap: 8px 18px;
  margin-top: 12px;
}

.dashboard-legend span {
  display: flex;
  min-width: 0;
  max-width: 220px;
  align-items: center;
  gap: 7px;
  color: var(--dashboard-muted);
  font-size: 12px;
}

.dashboard-legend i,
.dashboard-entity > i {
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

.dashboard-segmented {
  display: flex;
  height: 38px;
  align-items: center;
  gap: 2px;
  border: 1px solid #e2e5e9;
  border-radius: 8px;
  background: #f7f8f9;
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
  background: #fff;
  color: #20242a;
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
  background: #fff;
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

.dashboard-table-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
}

.dashboard-table-card {
  display: flex;
  min-width: 0;
  min-height: 330px;
  flex-direction: column;
  overflow: hidden;
}

.dashboard-table-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 20px 22px 10px;
}

.dashboard-table-header span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-table-wrap {
  width: 100%;
  overflow-x: auto;
  padding: 0 22px;
}

.dashboard-table-wrap table {
  width: 100%;
  min-width: 500px;
  border-collapse: collapse;
}

.dashboard-table-wrap th {
  height: 42px;
  color: #858c96;
  font-size: 12px;
  font-weight: 500;
  text-align: right;
}

.dashboard-table-wrap th:first-child,
.dashboard-table-wrap td:first-child {
  text-align: left;
}

.dashboard-table-wrap td {
  height: 51px;
  border-top: 1px solid #eff1f3;
  color: #4f5660;
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  text-align: right;
  white-space: nowrap;
}

.dashboard-entity {
  display: inline-flex;
  max-width: 210px;
  align-items: center;
  gap: 9px;
  vertical-align: middle;
}

.dashboard-entity strong {
  overflow: hidden;
  color: #30343a;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-platform-icon {
  display: grid;
  flex: 0 0 auto;
  width: 26px;
  height: 26px;
  place-items: center;
  border-radius: 7px;
  background: #f3f4f6;
  color: #32363c;
}

.dashboard-platform-anthropic {
  background: #fff1e8;
  color: #d97738;
}

.dashboard-platform-gemini {
  background: #edf4ff;
  color: #3478ce;
}

.dashboard-platform-antigravity {
  background: #edf8f5;
  color: #237d69;
}

.dashboard-platform-__other__ {
  color: #7b828c;
}

.dashboard-spark-column {
  width: 98px;
}

.dashboard-spark-column svg {
  margin-left: auto;
}

.dashboard-empty-row {
  height: 112px !important;
  color: var(--dashboard-subtle) !important;
  text-align: center !important;
}

.dashboard-table-action {
  display: inline-flex;
  align-items: center;
  align-self: center;
  gap: 6px;
  margin: auto 0 16px;
  padding-top: 12px;
  color: #5d6570;
  font-size: 12px;
  font-weight: 600;
}

.dashboard-table-action:hover {
  color: #111318;
}

@keyframes dashboard-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}

:global(.dark) .dashboard-page {
  --dashboard-border: #2d3745;
  --dashboard-text: #f3f4f6;
  --dashboard-muted: #a6adb7;
  --dashboard-subtle: #7f8792;
}

:global(.dark) .dashboard-card,
:global(.dark) .dashboard-select-shell select {
  background: #111827;
}

:global(.dark) .dashboard-segmented {
  border-color: #374151;
  background: #172130;
}

:global(.dark) .dashboard-segmented button.active {
  background: #263142;
  color: #f9fafb;
}

:global(.dark) .dashboard-select-shell select {
  border-color: #374151;
  color: #d1d5db;
}

:global(.dark) .dashboard-table-wrap td {
  border-color: #283342;
  color: #b5bdc8;
}

:global(.dark) .dashboard-entity strong {
  color: #e5e7eb;
}

:global(.dark) .dashboard-metric-details,
:global(.dark) .dashboard-metric-details > div + div {
  border-color: #283342;
}

:global(.dark) .dashboard-metric-details dd {
  color: #c8ced7;
}

:global(.dark) .dashboard-platform-icon {
  background: #263142;
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

@media (max-width: 900px) {
  .dashboard-table-grid {
    grid-template-columns: 1fr;
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

  .dashboard-table-header,
  .dashboard-table-wrap {
    padding-left: 16px;
    padding-right: 16px;
  }
}
</style>
