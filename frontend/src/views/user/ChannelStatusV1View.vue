<template>
  <AppLayout>
    <div class="monitor-ref-skin -mx-4 min-h-[calc(100vh-7rem)] space-y-6 bg-slate-50 px-4 pb-12 dark:bg-slate-950 md:-mx-6 md:px-6 lg:-mx-8 lg:px-8">
      <section
        class="card sticky top-0 z-20 !rounded-3xl !border-0 p-0 shadow-sm ring-1 ring-gray-900/5 backdrop-blur-sm dark:!bg-dark-800 dark:ring-dark-700 supports-[backdrop-filter]:bg-white/95 dark:supports-[backdrop-filter]:bg-dark-800/95"
      >
        <header class="page-header mb-0 flex flex-wrap items-start justify-between gap-4 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:px-6">
          <div class="min-w-0">
            <h1 class="page-title flex items-center gap-2 !font-sans text-xl !font-black text-gray-900 dark:text-white">
              <span class="inline-flex h-8 w-8 items-center justify-center rounded-xl bg-emerald-50 text-emerald-500 dark:bg-emerald-900/30 dark:text-emerald-400">
                <Icon name="chart" size="sm" />
              </span>
              {{ t('channelStatus.title') }}
            </h1>
            <div class="page-description mt-1.5 flex flex-wrap items-center gap-2 !text-xs text-gray-500 dark:text-gray-400">
              <span class="relative flex h-2 w-2 shrink-0">
                <span
                  class="relative inline-flex h-2 w-2 rounded-full"
                  :class="loading ? 'bg-gray-400' : overallDotClass"
                ></span>
              </span>
              <span v-if="loading" class="inline-flex items-center gap-1 text-primary-600 dark:text-primary-300">
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
              class="btn btn-secondary btn-icon flex h-8 w-8 items-center justify-center rounded-lg bg-gray-100 text-gray-500 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600"
              type="button"
              :title="t('common.refresh')"
              :disabled="loading"
              @click="manualReload"
            >
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </header>

        <div class="monitor-toolbar flex flex-nowrap items-center gap-1.5 overflow-x-auto px-4 py-3 sm:gap-2 sm:px-5">
          <div
            class="tabs inline-flex shrink-0"
            role="group"
            :aria-label="t('channelStatus.windowTab.7d')"
          >
            <button
              v-for="option in windowOptions"
              :key="option.value"
              type="button"
              class="tab !px-2.5 !py-1 text-xs sm:!px-3"
              :class="currentWindow === option.value ? 'tab-active' : ''"
              @click="handleWindowChange(option.value)"
            >
              {{ option.label }}
            </button>
          </div>

          <span class="mx-0.5 hidden h-5 w-px shrink-0 bg-gray-200 dark:bg-dark-700 sm:block" aria-hidden="true"></span>

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

          <span class="mx-0.5 hidden h-5 w-px shrink-0 bg-gray-200 dark:bg-dark-700 md:block" aria-hidden="true"></span>

          <div
            class="tabs inline-flex shrink-0"
            role="group"
            :aria-label="t('channelStatus.trendView.label')"
          >
            <button
              type="button"
              class="tab !px-2.5 !py-1 text-xs"
              :class="trendView === 'pulse' ? 'tab-active' : ''"
              @click="trendView = 'pulse'"
            >
              {{ t('channelStatus.trendView.pulse') }}
            </button>
            <button
              type="button"
              class="tab !px-2.5 !py-1 text-xs"
              :class="trendView === 'cards' ? 'tab-active' : ''"
              @click="trendView = 'cards'"
            >
              {{ t('channelStatus.trendView.cards') }}
            </button>
          </div>

          <div
            class="tabs inline-flex shrink-0"
            role="group"
            :aria-label="t('channelStatus.healthFilter.label')"
          >
            <button
              type="button"
              class="tab !px-2.5 !py-1 text-xs"
              :class="healthFilter === 'all' ? 'tab-active' : ''"
              @click="healthFilter = 'all'"
            >
              {{ t('channelStatus.healthFilter.all') }}
            </button>
            <button
              type="button"
              class="tab !px-2.5 !py-1 text-xs"
              :class="healthFilter === 'issues' ? 'tab-active' : ''"
              @click="healthFilter = 'issues'"
            >
              {{ t('channelStatus.healthFilter.issues') }}
            </button>
          </div>
        </div>
      </section>

      <section
        v-if="!loading || items.length > 0"
        class="grid grid-cols-2 gap-3 xl:grid-cols-4"
        :aria-label="t('channelStatus.summaryAria')"
      >
        <MetricCell
          :label="t('channelStatus.metrics.availability')"
          :value="kpis.availability.value"
          :detail="kpis.availability.detail"
          :state="kpis.availability.state"
        />
        <MetricCell
          :label="t('channelStatus.metrics.latency')"
          :value="kpis.latency.value"
          :detail="kpis.latency.detail"
          :state="kpis.latency.state"
        />
        <MetricCell
          :label="t('channelStatus.metrics.ping')"
          :value="kpis.ping.value"
          :detail="kpis.ping.detail"
          :state="kpis.ping.state"
        />
        <MetricCell
          :label="t('channelStatus.metrics.healthRate')"
          :value="kpis.health.value"
          :detail="kpis.health.detail"
          :state="kpis.health.state"
        />
      </section>
      <section v-else class="grid grid-cols-2 gap-3 xl:grid-cols-4" aria-hidden="true">
        <div v-for="i in 4" :key="i" class="h-24 animate-pulse rounded-2xl bg-gray-50 dark:bg-dark-900/30" />
      </section>

      <MonitorStatusMatrix
        v-if="trendView === 'pulse'"
        :rows="matrixRows"
        @row-click="openDetailById"
      />

      <section
        v-if="trendView === 'pulse'"
        class="card flex min-h-0 flex-col overflow-hidden !rounded-3xl !border-0 shadow-sm ring-1 ring-gray-900/5 dark:!bg-dark-800 dark:ring-dark-700"
      >
        <div class="border-b border-gray-100 px-5 pt-4 dark:border-dark-700 sm:px-6">
          <nav class="tabs w-full max-w-md sm:w-auto" role="tablist" :aria-label="t('channelStatus.tabs.aria')">
            <button type="button" role="tab" class="tab flex-1 tab-active sm:flex-none" aria-selected="true">
              {{ t('channelStatus.tabs.channels') }}
            </button>
          </nav>
        </div>
        <div class="min-h-0 max-h-[min(52vh,520px)] overflow-auto p-4 sm:p-5">
          <div v-if="visibleItems.length" class="table-container border-0">
            <table class="table monitor-table min-w-[720px]">
              <thead>
                <tr>
                  <th>{{ t('channelStatus.table.platformModel') }}</th>
                  <th>{{ t('channelStatus.table.availability') }}</th>
                  <th>{{ t('channelStatus.table.latency') }}</th>
                  <th>{{ t('channelStatus.table.ping') }}</th>
                  <th>{{ t('channelStatus.table.status') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="row in visibleItems"
                  :key="row.id"
                  class="cursor-pointer"
                  @click="openDetail(row)"
                >
                  <td>
                    <div class="flex items-center gap-2">
                      <span class="inline-block h-2 w-2 shrink-0 rounded-full" :class="rowDotClass(row.primary_status)"></span>
                      <div>
                        <span class="block text-xs text-gray-500 dark:text-dark-400">
                          {{ providerLabel(row.provider) }}<template v-if="row.group_name"> / {{ row.group_name }}</template>
                        </span>
                        <strong class="font-semibold text-gray-900 dark:text-white">
                          {{ row.name || formatMonitorModel(row.primary_model) }}
                        </strong>
                      </div>
                    </div>
                  </td>
                  <td>
                    <span class="block">{{ formatPct(resolveAvailability(row)) }}</span>
                    <small class="text-xs text-gray-400">{{ statusLabel(row.primary_status) }}</small>
                  </td>
                  <td>{{ formatMs(row.primary_latency_ms) }}</td>
                  <td>{{ formatMs(row.primary_ping_latency_ms) }}</td>
                  <td>{{ statusLabel(row.primary_status) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="empty-state py-10">
            <p class="empty-state-title text-base">{{ t('channelStatus.empty.title') }}</p>
            <p class="empty-state-description">
              {{ items.length ? t('channelStatus.empty.filterDescription') : t('channelStatus.empty.description') }}
            </p>
          </div>
        </div>
      </section>

      <MonitorCardGrid
        v-else
        :items="visibleItems"
        :window="currentWindow"
        :countdown-seconds="countdown"
        :loading="loading"
        :detail-cache="detailCache"
        @card-click="openDetail"
      />

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
import type { HealthState } from '@/api/channelMonitorV2'
import AppLayout from '@/components/layout/AppLayout.vue'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import FilterMultiSelect from '@/features/channel-monitor-v2/FilterMultiSelect.vue'
import MetricCell from '@/features/channel-monitor-v2/MetricCell.vue'
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
  overallStatus.value === 'operational' ? 'bg-green-500' : 'bg-amber-500'
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

const kpis = computed(() => {
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

  return {
    availability: {
      value: formatPct(avgAvailability),
      detail: t('channelStatus.metrics.availabilityDetail', { failed, degraded }),
      state: toneFromAvailability(avgAvailability),
    },
    latency: {
      value: formatMs(percentile(latencies, 50)),
      detail: t('channelStatus.metrics.latencyDetail', {
        avg: formatMs(average(latencies)),
        p90: formatMs(percentile(latencies, 90)),
      }),
      state: toneFromLatency(percentile(latencies, 50)),
    },
    ping: {
      value: formatMs(percentile(pings, 50)),
      detail: t('channelStatus.metrics.pingDetail', {
        avg: formatMs(average(pings)),
        p90: formatMs(percentile(pings, 90)),
      }),
      state: toneFromLatency(percentile(pings, 50)),
    },
    health: {
      value: formatPct(healthRate),
      detail: t('channelStatus.metrics.healthDetail', { healthy, total }),
      state: toneFromHealth(failed, degraded, total),
    },
  }
})

const matrixRows = computed<MatrixRow[]>(() =>
  visibleItems.value.map((item) => ({
    id: item.id,
    label: rowLabel(item),
    status: item.primary_status,
    availability: resolveAvailability(item),
    latency: formatMs(item.primary_latency_ms),
    ping: formatMs(item.primary_ping_latency_ms),
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
    lines: [t('channelStatus.matrix.noSample')],
  }))
  for (const point of real) {
    const latency = formatLatency(point.latency_ms)
    const ping = formatLatency(point.ping_latency_ms)
    cells.push({
      status: point.status || 'empty',
      title: `${formatRelativeTime(point.checked_at)} · ${statusLabel(point.status)} · ${latency}ms`,
      lines: [
        formatRelativeTime(point.checked_at),
        statusLabel(point.status),
        `${t('channelStatus.matrix.latency')} ${latency}ms`,
        `${t('channelStatus.matrix.ping')} ${ping}ms`,
      ],
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

function toneFromAvailability(value: number | null): HealthState {
  if (value == null) return 'unknown'
  if (value >= 99) return 'healthy'
  if (value >= 95) return 'warning'
  return 'critical'
}

function toneFromLatency(value: number | null): HealthState {
  if (value == null) return 'unknown'
  if (value <= 800) return 'healthy'
  if (value <= 2000) return 'warning'
  return 'critical'
}

function toneFromHealth(failed: number, degraded: number, total: number): HealthState {
  if (!total) return 'unknown'
  if (failed > 0) return 'critical'
  if (degraded > 0) return 'warning'
  return 'healthy'
}

function rowDotClass(status: string) {
  if (status === 'operational') return 'bg-emerald-500'
  if (status === 'degraded') return 'bg-amber-500'
  if (status === 'failed' || status === 'error') return 'bg-red-500'
  return 'bg-gray-300'
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
.monitor-ref-skin :deep(.card),
.monitor-ref-skin :deep(.stat-card) {
  background-color: #ffffff !important;
}
.monitor-ref-skin :deep(.tab-active) {
  background-color: #ffffff !important;
}
.monitor-ref-skin :deep(.select-trigger) {
  background-color: #ffffff !important;
}
:global(.dark) .monitor-ref-skin :deep(.card),
:global(.dark) .monitor-ref-skin :deep(.stat-card) {
  background-color: rgb(30 41 59) !important;
}
:global(.dark) .monitor-ref-skin :deep(.tab-active),
:global(.dark) .monitor-ref-skin :deep(.select-trigger) {
  background-color: rgb(51 65 85) !important;
}
</style>
