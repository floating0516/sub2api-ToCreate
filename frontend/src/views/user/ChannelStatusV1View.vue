<template>
  <AppLayout>
    <div class="channel-status-page space-y-6 pb-12">
      <section class="cs-shell">
        <header class="cs-shell-header">
          <div class="min-w-0">
            <h1 class="cs-title">
              <span class="cs-title-icon" aria-hidden="true">
                <Icon name="chart" size="sm" />
              </span>
              {{ t('channelStatus.title') }}
            </h1>
            <div class="cs-subtitle">
              <span class="cs-live-dot" :class="loading ? 'is-empty' : overallDotClass"></span>
              <span v-if="loading" class="inline-flex items-center gap-1">
                <LoadingSpinner size="sm" />
                {{ t('channelStatus.updating') }}
              </span>
              <template v-else>
                <span>{{ overallLabel }}</span>
                <span v-if="latestUpdateLabel">{{ latestUpdateLabel }}</span>
              </template>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <AutoRefreshButton
              v-if="autoRefresh"
              :enabled="autoRefresh.enabled.value"
              :interval-seconds="autoRefresh.intervalSeconds.value"
              :countdown="autoRefresh.countdown.value"
              :intervals="autoRefresh.intervals"
              @update:enabled="autoRefresh.setEnabled"
              @update:interval="autoRefresh.setInterval"
            />
            <button
              class="cs-icon-btn"
              type="button"
              :title="t('common.refresh')"
              :disabled="loading"
              @click="manualReload"
            >
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </header>

        <div class="cs-toolbar">
          <div class="cs-pills" role="group" :aria-label="t('channelStatus.windowTab.7d')">
            <button
              v-for="option in windowOptions"
              :key="option.value"
              type="button"
              class="cs-pill"
              :class="currentWindow === option.value ? 'is-active' : ''"
              @click="handleWindowChange(option.value)"
            >
              {{ option.label }}
            </button>
          </div>

          <span class="cs-divider" aria-hidden="true"></span>

          <FilterMultiSelect
            v-model="selectedProviders"
            compact
            :label="t('channelStatus.filters.platform')"
            :all-label="t('channelStatus.filters.allPlatforms')"
            :options="providerOptions"
          />
          <FilterMultiSelect
            v-model="selectedGroups"
            compact
            :label="t('channelStatus.filters.group')"
            :all-label="t('channelStatus.filters.allGroups')"
            :options="groupOptions"
          />
          <FilterMultiSelect
            v-model="selectedModels"
            compact
            :label="t('channelStatus.filters.model')"
            :all-label="t('channelStatus.filters.allModels')"
            :options="modelOptions"
          />
          <button
            type="button"
            class="btn btn-ghost btn-sm shrink-0 !px-2 !py-1 text-xs"
            :disabled="!hasDimensionFilter"
            :class="!hasDimensionFilter ? 'opacity-40' : ''"
            @click="clearFilters"
          >
            {{ t('channelStatus.clearFilters') }}
          </button>

          <span class="cs-divider" aria-hidden="true"></span>

          <div class="cs-pills" role="group" :aria-label="t('channelStatus.trendView.label')">
            <button
              type="button"
              class="cs-pill"
              :class="trendView === 'pulse' ? 'is-active' : ''"
              @click="trendView = 'pulse'"
            >
              {{ t('channelStatus.trendView.pulse') }}
            </button>
            <button
              type="button"
              class="cs-pill"
              :class="trendView === 'cards' ? 'is-active' : ''"
              @click="trendView = 'cards'"
            >
              {{ t('channelStatus.trendView.cards') }}
            </button>
          </div>

          <div class="cs-pills" role="group" :aria-label="t('channelStatus.healthFilter.label')">
            <button
              type="button"
              class="cs-pill"
              :class="healthFilter === 'all' ? 'is-active' : ''"
              @click="healthFilter = 'all'"
            >
              {{ t('channelStatus.healthFilter.all') }}
            </button>
            <button
              type="button"
              class="cs-pill"
              :class="healthFilter === 'issues' ? 'is-active' : ''"
              @click="healthFilter = 'issues'"
            >
              {{ t('channelStatus.healthFilter.issues') }}
            </button>
          </div>
        </div>
      </section>

      <MonitorOverviewCards
        v-if="!loading || items.length > 0"
        :cards="overviewCards"
        :aria-label="t('channelStatus.summaryAria')"
      />
      <section v-else class="grid grid-cols-2 gap-3 xl:grid-cols-4" aria-hidden="true">
        <div v-for="i in 4" :key="i" class="h-[7.25rem] animate-pulse rounded-[1.25rem] bg-gray-100/80 dark:bg-dark-800" />
      </section>

      <MonitorStatusMatrix
        v-if="trendView === 'pulse'"
        :rows="matrixRows"
        @row-click="openDetailById"
      />

      <section v-if="trendView === 'cards' || visibleItems.length > 0">
        <h2 v-if="trendView === 'pulse' && visibleItems.length" class="cs-detail-heading">
          {{ t('channelStatus.detailSection') }}
        </h2>
        <MonitorCardGrid
          :items="visibleItems"
          :window="currentWindow"
          :countdown-seconds="countdown"
          :loading="loading"
          :detail-cache="detailCache"
          @card-click="openDetail"
        />
      </section>

      <MonitorDetailDialog
        :show="showDetail"
        :monitor-id="detailTarget?.id ?? null"
        :title="detailTitle"
        @close="closeDetail"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  list as listChannelMonitorViews,
  status as fetchChannelMonitorDetail,
  type UserMonitorView,
  type UserMonitorDetail,
} from '@/api/channelMonitor'
import AppLayout from '@/components/layout/AppLayout.vue'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import FilterMultiSelect from '@/features/channel-monitor-v2/FilterMultiSelect.vue'
import MonitorOverviewCards, {
  type OverviewCard,
  type OverviewTone,
} from '@/components/user/monitor/MonitorOverviewCards.vue'
import MonitorStatusMatrix, {
  type MatrixRow,
} from '@/components/user/monitor/MonitorStatusMatrix.vue'
import MonitorCardGrid from '@/components/user/monitor/MonitorCardGrid.vue'
import MonitorDetailDialog from '@/components/user/MonitorDetailDialog.vue'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import { DEFAULT_INTERVAL_SECONDS, STATUS_OPERATIONAL } from '@/constants/channelMonitor'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import type { MonitorWindow, OverallStatus } from '@/components/user/monitor/MonitorHero.vue'

const { t, locale } = useI18n()
const appStore = useAppStore()
const { providerLabel, formatMonitorModel, formatLatency, formatRelativeTime, statusLabel } =
  useChannelMonitorFormat()

const items = ref<UserMonitorView[]>([])
const loading = ref(false)
const currentWindow = ref<MonitorWindow>('7d')
const detailCache = reactive<Record<number, UserMonitorDetail>>({})
const showDetail = ref(false)
const detailTarget = ref<UserMonitorView | null>(null)
const selectedProviders = ref<string[]>([])
const selectedGroups = ref<string[]>([])
const selectedModels = ref<string[]>([])
const healthFilter = ref<'all' | 'issues'>('all')
const trendView = ref<'pulse' | 'cards'>('pulse')

let abortController: AbortController | null = null

const autoRefresh = useAutoRefresh({
  storageKey: 'channel-status-auto-refresh',
  intervals: [30, 60, 120] as const,
  defaultInterval: DEFAULT_INTERVAL_SECONDS,
  onRefresh: () => reload(true),
  shouldPause: () => document.hidden || loading.value,
})
const countdown = autoRefresh.countdown

const windowOptions = computed(() => [
  { value: '7d' as MonitorWindow, label: t('channelStatus.windowTab.7d') },
  { value: '15d' as MonitorWindow, label: t('channelStatus.windowTab.15d') },
  { value: '30d' as MonitorWindow, label: t('channelStatus.windowTab.30d') },
])

const providerOptions = computed(() => {
  const seen = new Set<string>()
  return items.value.flatMap((item) => {
    if (seen.has(item.provider)) return []
    seen.add(item.provider)
    return [{ value: item.provider, label: providerLabel(item.provider) }]
  })
})

const selectedProviderSet = computed(() => new Set(selectedProviders.value))

const groupOptions = computed(() => {
  const seen = new Set<string>()
  return items.value.flatMap((item) => {
    if (selectedProviderSet.value.size && !selectedProviderSet.value.has(item.provider)) return []
    const value = item.group_name || ''
    if (!value || seen.has(value)) return []
    seen.add(value)
    return [{ value, label: value }]
  })
})

const modelOptions = computed(() => {
  const seen = new Set<string>()
  return items.value.flatMap((item) => {
    if (selectedProviderSet.value.size && !selectedProviderSet.value.has(item.provider)) return []
    const value = item.primary_model || ''
    if (!value || seen.has(value)) return []
    seen.add(value)
    return [{ value, label: formatMonitorModel(value) }]
  })
})

const hasDimensionFilter = computed(
  () => selectedProviders.value.length + selectedGroups.value.length + selectedModels.value.length > 0
)

const visibleItems = computed(() => {
  return items.value.filter((item) => {
    if (selectedProviders.value.length && !selectedProviders.value.includes(item.provider)) return false
    if (selectedGroups.value.length && !selectedGroups.value.includes(item.group_name || '')) return false
    if (selectedModels.value.length && !selectedModels.value.includes(item.primary_model || '')) return false
    if (healthFilter.value === 'issues' && item.primary_status === STATUS_OPERATIONAL) return false
    return true
  })
})

const overallStatus = computed<OverallStatus>(() => {
  if (visibleItems.value.length === 0) return 'operational'
  for (const it of visibleItems.value) {
    if (it.primary_status === 'failed' || it.primary_status === 'error') return 'degraded'
    if (it.primary_status !== STATUS_OPERATIONAL) return 'degraded'
  }
  return 'operational'
})

const overallLabel = computed(() => t(`channelStatus.overall.${overallStatus.value}`))
const overallDotClass = computed(() =>
  overallStatus.value === 'operational' ? 'is-operational' : 'is-degraded'
)
const latestUpdateLabel = computed(() => {
  let latest = 0
  for (const item of visibleItems.value) {
    const checked = item.timeline?.[0]?.checked_at
    if (checked) latest = Math.max(latest, Date.parse(checked))
  }
  if (!latest) return ''
  return t('channelStatus.updatedTo', { time: formatClock(latest) })
})

const detailTitle = computed(() => detailTarget.value?.name || t('channelStatus.detailTitle'))

const overviewCards = computed<OverviewCard[]>(() => {
  const rows = visibleItems.value
  const total = rows.length
  const failed = rows.filter((item) => item.primary_status === 'failed' || item.primary_status === 'error').length
  const degraded = rows.filter((item) => item.primary_status === 'degraded').length
  const healthy = rows.filter((item) => item.primary_status === STATUS_OPERATIONAL).length
  const availabilities = rows
    .map((item) => resolveAvailability(item))
    .filter((value): value is number => value != null && !Number.isNaN(value))
  const avgAvailability = availabilities.length
    ? availabilities.reduce((sum, value) => sum + value, 0) / availabilities.length
    : null
  const latencies = collectSamples(rows, 'latency_ms')
  const pings = collectSamples(rows, 'ping_latency_ms')
  const healthRate = total ? (healthy / total) * 100 : null

  return [
    {
      key: 'availability',
      label: t('channelStatus.metrics.availability'),
      value: formatPct(avgAvailability),
      details: [t('channelStatus.metrics.availabilityDetail', { failed, degraded })],
      tone: toneFromAvailability(avgAvailability),
      dot: true,
    },
    {
      key: 'latency',
      label: t('channelStatus.metrics.latency'),
      value: formatMs(percentile(latencies, 50)),
      details: splitDetail(
        t('channelStatus.metrics.latencyDetail', {
          avg: formatMs(average(latencies)),
          p90: formatMs(percentile(latencies, 90)),
        }),
      ),
      tone: toneFromLatency(percentile(latencies, 50)),
    },
    {
      key: 'ping',
      label: t('channelStatus.metrics.ping'),
      value: formatMs(percentile(pings, 50)),
      details: splitDetail(
        t('channelStatus.metrics.pingDetail', {
          avg: formatMs(average(pings)),
          p90: formatMs(percentile(pings, 90)),
        }),
      ),
      tone: toneFromLatency(percentile(pings, 50)),
    },
    {
      key: 'health',
      label: t('channelStatus.metrics.healthRate'),
      value: formatPct(healthRate),
      details: [t('channelStatus.metrics.healthDetail', { healthy, total })],
      tone: toneFromHealth(failed, degraded, total),
      dot: true,
    },
  ]
})

const matrixRows = computed<MatrixRow[]>(() =>
  visibleItems.value.map((item) => ({
    id: item.id,
    label: rowLabel(item),
    status: item.primary_status,
    availability: resolveAvailability(item),
    cells: timelineCells(item),
  }))
)

function rowLabel(item: UserMonitorView) {
  const parts = [providerLabel(item.provider)]
  if (item.group_name) parts.push(item.group_name)
  parts.push(item.name || formatMonitorModel(item.primary_model))
  return parts.join(' / ')
}

function timelineCells(item: UserMonitorView) {
  const length = 60
  const real = [...(item.timeline ?? [])].slice(0, length).reverse()
  const pad = Math.max(0, length - real.length)
  const cells = Array.from({ length: pad }, () => ({
    status: 'empty',
    title: t('channelStatus.matrix.noSample'),
  }))
  for (const point of real) {
    const latency = formatLatency(point.latency_ms)
    cells.push({
      status: point.status || 'empty',
      title: `${formatRelativeTime(point.checked_at)} · ${statusLabel(point.status)} · ${latency}ms`,
    })
  }
  return cells
}

function resolveAvailability(item: UserMonitorView): number | null {
  if (currentWindow.value === '7d') return item.availability_7d ?? null
  const detail = detailCache[item.id]
  if (!detail) return null
  const primary = detail.models.find((model) => model.model === item.primary_model)
  if (!primary) return null
  return currentWindow.value === '15d' ? primary.availability_15d ?? null : primary.availability_30d ?? null
}

function collectSamples(rows: UserMonitorView[], field: 'latency_ms' | 'ping_latency_ms') {
  const values: number[] = []
  for (const item of rows) {
    const latest = field === 'latency_ms' ? item.primary_latency_ms : item.primary_ping_latency_ms
    if (latest != null && !Number.isNaN(latest)) values.push(latest)
    for (const point of item.timeline || []) {
      const value = point[field]
      if (value != null && !Number.isNaN(value)) values.push(value)
    }
  }
  return values
}

function percentile(values: number[], p: number): number | null {
  if (!values.length) return null
  const sorted = [...values].sort((a, b) => a - b)
  const index = Math.min(sorted.length - 1, Math.max(0, Math.ceil((p / 100) * sorted.length) - 1))
  return sorted[index]
}

function average(values: number[]): number | null {
  if (!values.length) return null
  return values.reduce((sum, value) => sum + value, 0) / values.length
}

function formatPct(value: number | null) {
  if (value == null || Number.isNaN(value)) return '—'
  return `${value.toFixed(2)}%`
}

function formatMs(value: number | null) {
  if (value == null || Number.isNaN(value)) return '—'
  const formatted = new Intl.NumberFormat(locale.value || undefined, {
    maximumFractionDigits: 0,
  }).format(Math.round(value))
  return `${formatted}ms`
}

function splitDetail(value: string) {
  return value.split(/\s*[·|]\s*/).map((part) => part.trim()).filter(Boolean)
}

function toneFromAvailability(value: number | null): OverviewTone {
  if (value == null) return 'neutral'
  if (value >= 99) return 'healthy'
  if (value >= 95) return 'warning'
  return 'critical'
}

function toneFromLatency(value: number | null): OverviewTone {
  if (value == null) return 'neutral'
  if (value <= 800) return 'healthy'
  if (value <= 2000) return 'warning'
  return 'critical'
}

function toneFromHealth(failed: number, degraded: number, total: number): OverviewTone {
  if (!total) return 'neutral'
  if (failed > 0) return 'critical'
  if (degraded > 0) return 'warning'
  return 'healthy'
}

function formatClock(value: number) {
  return new Intl.DateTimeFormat(locale.value || undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function clearFilters() {
  selectedProviders.value = []
  selectedGroups.value = []
  selectedModels.value = []
}

async function reload(silent = false) {
  if (abortController) abortController.abort()
  const ctrl = new AbortController()
  abortController = ctrl
  if (!silent) loading.value = true
  try {
    const res = await listChannelMonitorViews({ signal: ctrl.signal })
    if (ctrl.signal.aborted || abortController !== ctrl) return
    items.value = res.items || []
  } catch (err: unknown) {
    const e = err as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('channelStatus.loadError')))
  } finally {
    if (abortController === ctrl) {
      if (!silent) loading.value = false
      countdown.value = DEFAULT_INTERVAL_SECONDS
      abortController = null
    }
  }
}

async function manualReload() {
  await reload(false)
  if (currentWindow.value !== '7d') {
    await Promise.all(items.value.map((it) => loadDetail(it.id, true)))
  }
}

async function loadDetail(id: number, force = false) {
  if (!force && detailCache[id]) return
  try {
    detailCache[id] = await fetchChannelMonitorDetail(id)
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('channelStatus.detailLoadError')))
  }
}

async function ensureDetailsForWindow() {
  if (currentWindow.value === '7d') return
  await Promise.all(items.value.map((it) => loadDetail(it.id)))
}

async function handleWindowChange(value: MonitorWindow) {
  currentWindow.value = value
  await ensureDetailsForWindow()
}

function openDetail(row: UserMonitorView) {
  detailTarget.value = row
  showDetail.value = true
}

function openDetailById(id: number) {
  const row = items.value.find((item) => item.id === id)
  if (row) openDetail(row)
}

function closeDetail() {
  showDetail.value = false
  detailTarget.value = null
}

watch(
  [groupOptions, modelOptions],
  () => {
    if (groupOptions.value.length) {
      const allowed = new Set(groupOptions.value.map((item) => item.value))
      const next = selectedGroups.value.filter((value) => allowed.has(value))
      if (next.length !== selectedGroups.value.length) selectedGroups.value = next
    }
    if (modelOptions.value.length) {
      const allowed = new Set(modelOptions.value.map((item) => item.value))
      const next = selectedModels.value.filter((value) => allowed.has(value))
      if (next.length !== selectedModels.value.length) selectedModels.value = next
    }
  },
  { flush: 'post' },
)

watch(items, () => {
  void ensureDetailsForWindow()
})

watch(
  () => appStore.cachedPublicSettings?.channel_monitor_enabled,
  (enabled) => {
    if (enabled === false) autoRefresh.stop()
    else if (autoRefresh.enabled.value) autoRefresh.start()
  },
)

onMounted(() => {
  void reload(false)
  if (appStore.cachedPublicSettings?.channel_monitor_enabled !== false) {
    autoRefresh.setEnabled(true)
  }
})

onBeforeUnmount(() => {
  if (abortController) abortController.abort()
})
</script>

<style scoped>
.channel-status-page {
  --cs-surface: color-mix(in srgb, var(--tc-surface, #fffefb) 92%, white);
}

.cs-shell {
  border-radius: 1.5rem;
  background: var(--cs-surface);
  box-shadow:
    0 1px 2px rgb(17 24 39 / 4%),
    0 0 0 1px rgb(17 24 39 / 5%);
}

.cs-shell-header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid rgb(17 24 39 / 6%);
  padding: 1rem 1.25rem;
}

.cs-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #111827;
  font-size: 1.25rem;
  font-weight: 800;
}

.cs-title-icon {
  display: inline-flex;
  height: 2rem;
  width: 2rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.75rem;
  background: #ecfdf5;
  color: #10b981;
}

.cs-subtitle {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.4rem;
  color: #6b7280;
  font-size: 12px;
}

.cs-live-dot {
  display: inline-block;
  height: 0.5rem;
  width: 0.5rem;
  border-radius: 9999px;
}

.cs-live-dot.is-operational {
  background: #10b981;
}

.cs-live-dot.is-degraded {
  background: #f59e0b;
}

.cs-live-dot.is-empty {
  background: #9ca3af;
}

.cs-icon-btn {
  display: inline-flex;
  height: 2rem;
  width: 2rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.65rem;
  background: rgb(243 244 246);
  color: #6b7280;
}

.cs-icon-btn:hover {
  background: rgb(229 231 235);
}

.cs-toolbar {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.5rem;
  overflow-x: auto;
  padding: 0.8rem 1.1rem 1rem;
}

.cs-pills {
  display: inline-flex;
  flex: none;
  gap: 0.2rem;
  border-radius: 9999px;
  background: rgb(243 244 246);
  padding: 0.2rem;
}

.cs-pill {
  border-radius: 9999px;
  padding: 0.28rem 0.7rem;
  color: #6b7280;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.cs-pill.is-active {
  background: white;
  color: #111827;
  box-shadow: 0 1px 2px rgb(17 24 39 / 8%);
}

.cs-divider {
  display: none;
  height: 1.25rem;
  width: 1px;
  flex: none;
  background: rgb(17 24 39 / 8%);
}

.cs-detail-heading {
  margin-bottom: 0.85rem;
  color: #111827;
  font-size: 0.875rem;
  font-weight: 700;
}

@media (min-width: 640px) {
  .cs-divider {
    display: block;
  }
}

:global(.dark) .cs-shell {
  background: rgb(35 38 32);
  box-shadow:
    0 1px 2px rgb(0 0 0 / 20%),
    0 0 0 1px rgb(255 255 255 / 6%);
}

:global(.dark) .cs-shell-header {
  border-bottom-color: rgb(255 255 255 / 8%);
}

:global(.dark) .cs-title,
:global(.dark) .cs-detail-heading {
  color: white;
}

:global(.dark) .cs-title-icon {
  background: rgb(16 185 129 / 14%);
  color: #34d399;
}

:global(.dark) .cs-subtitle {
  color: #9ca3af;
}

:global(.dark) .cs-icon-btn {
  background: rgb(55 65 81 / 50%);
  color: #9ca3af;
}

:global(.dark) .cs-pills {
  background: rgb(23 25 22);
}

:global(.dark) .cs-pill {
  color: #9ca3af;
}

:global(.dark) .cs-pill.is-active {
  background: rgb(55 65 81);
  color: white;
}
</style>
