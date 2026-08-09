<template>
  <div class="dashboard-activity-layout">
    <aside class="dashboard-activity-summary" :aria-label="t('dashboard.overview.usageSummary')">
      <header class="dashboard-activity-header">
        <div>
          <h2>{{ t('dashboard.overview.usageSummary') }}</h2>
        </div>
      </header>

      <dl class="dashboard-token-summary">
        <div v-for="item in summaryMetrics" :key="item.key">
          <dt>{{ item.label }}</dt>
          <dd v-if="!loading">
            <strong>{{ item.value }}</strong>
            <small>{{ item.caption }}</small>
          </dd>
          <dd v-else>
            <i class="dashboard-activity-skeleton dashboard-activity-skeleton-value" />
            <i class="dashboard-activity-skeleton dashboard-activity-skeleton-meta" />
          </dd>
        </div>
      </dl>

      <div class="dashboard-activity-insights">
        <span>
          {{ t('dashboard.overview.activeDays') }}
          <b v-if="!loading">{{ summary.activeDays }}</b>
          <i v-else class="dashboard-activity-skeleton dashboard-activity-skeleton-inline" />
        </span>
        <span>
          {{ t('dashboard.overview.peakDay') }}
          <b v-if="!loading && summary.peakDay">
            {{ formatDay(summary.peakDay) }} · {{ formatValue(summary.peakValue) }}
          </b>
          <b v-else-if="!loading">--</b>
          <i v-else class="dashboard-activity-skeleton dashboard-activity-skeleton-inline" />
        </span>
      </div>
    </aside>

    <div class="dashboard-calendar-panel">
      <div class="dashboard-calendar-panel-inner">
        <header class="dashboard-calendar-panel-header">
          <div class="dashboard-calendar-heading">
            <button
              type="button"
              class="dashboard-calendar-heading-button"
              :title="t('dashboard.dailyReport.todayReport')"
              @click="emit('select-day', endDate)"
            >
              <span>{{ t('dashboard.overview.tokenActivity') }}</span>
              <Icon name="sparkles" size="xs" />
            </button>
          </div>

          <div
            class="dashboard-calendar-modes"
            role="group"
            :aria-label="t('dashboard.overview.heatmapGranularity')"
          >
            <button
              v-for="item in modeOptions"
              :key="item.value"
              type="button"
              :class="activityMode === item.value && 'active'"
              :aria-pressed="activityMode === item.value"
              @click="activityMode = item.value"
            >
              {{ item.label }}
            </button>
          </div>
        </header>

        <div ref="calendarStageRef" class="dashboard-calendar-stage">
          <div v-if="loading" class="dashboard-calendar-loading">
            <LoadingSpinner size="md" />
          </div>
          <VChart
            class="dashboard-calendar-chart"
            :option="chartOption"
            :update-options="updateOptions"
            :autoresize="{ throttle: 80 }"
            @click="handleChartClick"
          />
          <div v-if="!loading && !hasData" class="dashboard-calendar-empty">
            {{ t('dashboard.noDataAvailable') }}
          </div>
          <div class="dashboard-calendar-legend" aria-hidden="true">
            <span>{{ t('dashboard.overview.lessUsage') }}</span>
            <i v-for="color in legendColors" :key="color" :style="{ backgroundColor: color }" />
            <span>{{ t('dashboard.overview.moreUsage') }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { use } from 'echarts/core'
import { HeatmapChart } from 'echarts/charts'
import {
  AriaComponent,
  CalendarComponent,
  TooltipComponent,
  VisualMapComponent
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsOption } from 'echarts'
import VChart from 'vue-echarts'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'

use([
  CanvasRenderer,
  HeatmapChart,
  CalendarComponent,
  TooltipComponent,
  VisualMapComponent,
  AriaComponent
])

export interface DashboardCalendarPoint {
  day: string
  value: number
}

type ActivityMode = 'daily' | 'weekly' | 'cumulative'

const props = defineProps<{
  data: DashboardCalendarPoint[]
  startDate: string
  endDate: string
  totalTokens?: number
  loading?: boolean
}>()

const emit = defineEmits<{
  (event: 'select-day', day: string): void
}>()

const { t, locale } = useI18n()
const isDark = ref(document.documentElement.classList.contains('dark'))
const activityMode = ref<ActivityMode>('daily')
const calendarStageRef = ref<HTMLElement | null>(null)
const calendarCellSize = ref(14)
const updateOptions = { notMerge: false, lazyUpdate: false }
const lightColors = ['#e8f5ee', '#bfe7cf', '#80cfa4', '#43b67d', '#168a58']
const darkColors = ['#173329', '#1d4b38', '#236747', '#2d875a', '#42b875']
const DAY_IN_MS = 24 * 60 * 60 * 1000
let themeObserver: MutationObserver | null = null
let calendarResizeObserver: ResizeObserver | null = null

const parseDay = (day: string): number => {
  const [year, month, date] = day.split('-').map(Number)
  if (!year || !month || !date) return Number.NaN
  return Date.UTC(year, month - 1, date)
}

const formatDayKey = (timestamp: number): string => {
  const date = new Date(timestamp)
  return [
    date.getUTCFullYear(),
    String(date.getUTCMonth() + 1).padStart(2, '0'),
    String(date.getUTCDate()).padStart(2, '0')
  ].join('-')
}

const startOfWeek = (timestamp: number): number => {
  const day = new Date(timestamp).getUTCDay()
  return timestamp - ((day + 6) % 7) * DAY_IN_MS
}

const usageByDay = computed(() => {
  const values = new Map<string, number>()
  props.data.forEach((item) => {
    const value = Number(item.value)
    if (!item.day || !Number.isFinite(value)) return
    values.set(item.day, (values.get(item.day) || 0) + Math.max(0, value))
  })
  return values
})

const dailyWindowEntries = computed<[string, number][]>(() => {
  const start = parseDay(props.startDate)
  const end = parseDay(props.endDate)
  if (!Number.isFinite(start) || !Number.isFinite(end)) return []

  const entries: [string, number][] = []
  for (let cursor = start; cursor <= end; cursor += DAY_IN_MS) {
    const day = formatDayKey(cursor)
    entries.push([day, usageByDay.value.get(day) || 0])
  }
  return entries
})

const hasData = computed(() => dailyWindowEntries.value.some(([, value]) => value > 0))
const legendColors = computed(() => (isDark.value ? darkColors : lightColors))

const formatValue = (value: number): string => {
  const absolute = Math.abs(value)
  if (absolute >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(1)}B`
  if (absolute >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (absolute >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return Math.round(value).toLocaleString(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US')
}

const formatDay = (day: string): string => {
  const timestamp = parseDay(day)
  if (!Number.isFinite(timestamp)) return day
  return new Intl.DateTimeFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC'
  }).format(new Date(timestamp))
}

const formatWeekRange = (day: string): string => {
  const timestamp = parseDay(day)
  if (!Number.isFinite(timestamp)) return day
  const start = startOfWeek(timestamp)
  return `${formatDay(formatDayKey(start))} - ${formatDay(formatDayKey(start + 6 * DAY_IN_MS))}`
}

const weeklyTotals = computed(() => {
  const values = new Map<string, number>()
  dailyWindowEntries.value.forEach(([day, value]) => {
    const week = formatDayKey(startOfWeek(parseDay(day)))
    values.set(week, (values.get(week) || 0) + value)
  })
  return values
})

const heatmapEntries = computed<[string, number][]>(() => {
  if (activityMode.value === 'weekly') {
    return dailyWindowEntries.value.map(([day]) => {
      const week = formatDayKey(startOfWeek(parseDay(day)))
      return [day, weeklyTotals.value.get(week) || 0] as [string, number]
    })
  }

  if (activityMode.value === 'cumulative') {
    let runningTotal = 0
    return dailyWindowEntries.value.map(([day, value]) => {
      runningTotal += value
      return [day, runningTotal] as [string, number]
    })
  }

  return dailyWindowEntries.value
})

const summary = computed(() => {
  let total = 0
  let activeDays = 0
  let peakDay = ''
  let peakValue = 0

  dailyWindowEntries.value.forEach(([day, value]) => {
    total += value
    if (value > 0) activeDays += 1
    if (value > peakValue) {
      peakDay = day
      peakValue = value
    }
  })

  const endTimestamp = parseDay(props.endDate)
  const weekStart = Number.isFinite(endTimestamp) ? startOfWeek(endTimestamp) : Number.NaN
  const weekKey = Number.isFinite(weekStart) ? formatDayKey(weekStart) : ''

  return {
    total,
    activeDays,
    endDay: usageByDay.value.get(props.endDate) || 0,
    currentWeek: weeklyTotals.value.get(weekKey) || 0,
    weekRange: Number.isFinite(weekStart) ? formatWeekRange(props.endDate) : '',
    peakDay,
    peakValue
  }
})

const summaryMetrics = computed(() => [
  {
    key: 'daily',
    label: t('dashboard.overview.dailyToken'),
    value: formatValue(summary.value.endDay),
    caption: formatDay(props.endDate)
  },
  {
    key: 'weekly',
    label: t('dashboard.overview.weeklyToken'),
    value: formatValue(summary.value.currentWeek),
    caption: summary.value.weekRange
  },
  {
    key: 'cumulative',
    label: t('dashboard.overview.cumulativeToken'),
    value: formatValue(props.totalTokens || 0),
    caption: t('dashboard.overview.accountLifetime')
  }
])

const modeOptions = computed<{ value: ActivityMode; label: string }[]>(() => [
  { value: 'daily', label: t('dashboard.overview.dailyHeatmap') },
  { value: 'weekly', label: t('dashboard.overview.weeklyHeatmap') },
  { value: 'cumulative', label: t('dashboard.overview.cumulativeHeatmap') }
])

const updateCalendarCellSize = () => {
  const stage = calendarStageRef.value
  if (!stage) return
  const widthSize = Math.floor((stage.clientWidth - 54) / 53)
  const heightSize = Math.floor((stage.clientHeight - 28) / 7)
  const nextSize = Math.max(14, Math.min(18, widthSize, heightSize))
  if (nextSize !== calendarCellSize.value) calendarCellSize.value = nextSize
}

const chartOption = computed<EChartsOption>(() => {
  const dark = isDark.value
  const colors = dark ? darkColors : lightColors
  const values = heatmapEntries.value.filter(([, value]) => value > 0)
  const sortedValues = values.map(([, value]) => value).sort((left, right) => left - right)
  const scaleCeiling = sortedValues[Math.floor((sortedValues.length - 1) * 0.95)] || 0
  const maxValue = Math.max(1, scaleCeiling)
  const surface = dark ? '#111827' : '#ffffff'
  const empty = dark ? '#1c2634' : '#f0f2f4'
  const text = dark ? '#9ca3af' : '#7c838d'

  return {
    backgroundColor: surface,
    animation: true,
    animationDuration: 260,
    animationDurationUpdate: 240,
    animationEasing: 'cubicOut',
    aria: {
      enabled: true,
      decal: { show: false }
    },
    tooltip: {
      confine: true,
      backgroundColor: dark ? '#111827' : '#ffffff',
      borderColor: dark ? '#374151' : '#e2e5e9',
      borderWidth: 1,
      padding: [9, 11],
      textStyle: {
        color: dark ? '#d1d5db' : '#4b5563',
        fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
        fontSize: 12
      },
      formatter: (params: unknown) => {
        const point = params as { value?: [string, number] }
        const [day = '', value = 0] = point.value || []
        const heading = activityMode.value === 'weekly' ? formatWeekRange(day) : day
        const label = activityMode.value === 'cumulative'
          ? t('dashboard.overview.windowCumulative')
          : t('dashboard.overview.tokenUsage')
        return `${heading}<br>${label} <strong>${formatValue(Number(value))}</strong><br><span style="color:${text}">${t('dashboard.dailyReport.open')}</span>`
      },
      extraCssText: `border-radius: 8px; box-shadow: 0 10px 28px rgba(17, 24, 39, ${dark ? '0.24' : '0.10'});`
    },
    visualMap: {
      show: false,
      min: 0,
      max: maxValue,
      calculable: false,
      inRange: { color: colors }
    },
    calendar: {
      top: 18,
      left: 'center',
      range: [props.startDate, props.endDate],
      cellSize: [calendarCellSize.value, calendarCellSize.value],
      splitLine: { show: false },
      itemStyle: {
        color: empty,
        borderColor: surface,
        borderWidth: 1.5,
        borderRadius: 3
      },
      dayLabel: {
        firstDay: 1,
        nameMap: locale.value.startsWith('zh') ? 'ZH' : 'EN',
        color: text,
        fontSize: 10,
        margin: 8
      },
      monthLabel: {
        nameMap: locale.value.startsWith('zh') ? 'ZH' : 'EN',
        color: text,
        fontSize: 10,
        margin: 8
      },
      yearLabel: { show: false }
    },
    series: [{
      id: 'dashboard-token-activity-calendar',
      type: 'heatmap',
      coordinateSystem: 'calendar',
      data: values.map(([day, value]) => [day, value]),
      cursor: 'pointer',
      itemStyle: {
        borderColor: surface,
        borderWidth: 1.5,
        borderRadius: 3
      },
      emphasis: {
        itemStyle: {
          borderColor: dark ? '#d1d5db' : '#374151',
          borderWidth: 1,
          borderRadius: 3,
          shadowBlur: 7,
          shadowColor: 'rgba(17, 24, 39, 0.18)'
        }
      }
    }]
  }
})

const handleChartClick = (params: unknown) => {
  const point = params as { value?: [string, number] }
  const day = point.value?.[0]
  if (day) emit('select-day', day)
}

onMounted(() => {
  themeObserver = new MutationObserver(() => {
    isDark.value = document.documentElement.classList.contains('dark')
  })
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })

  calendarResizeObserver = new ResizeObserver(updateCalendarCellSize)
  if (calendarStageRef.value) calendarResizeObserver.observe(calendarStageRef.value)
  requestAnimationFrame(updateCalendarCellSize)
})

onUnmounted(() => {
  themeObserver?.disconnect()
  calendarResizeObserver?.disconnect()
})
</script>

<style scoped>
.dashboard-activity-layout {
  display: grid;
  height: 208px;
  min-height: 208px;
  grid-template-columns: minmax(280px, 0.32fr) minmax(0, 1fr);
  gap: 24px;
}

.dashboard-activity-summary {
  display: flex;
  min-width: 0;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid var(--dashboard-divider, #eff1f3);
  padding-right: 24px;
}

.dashboard-activity-header,
.dashboard-calendar-panel-header {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}

.dashboard-activity-header {
  margin-bottom: 8px;
}

.dashboard-activity-header > div,
.dashboard-calendar-heading {
  display: grid;
  min-width: 0;
  gap: 1px;
}

.dashboard-activity-header h2,
.dashboard-calendar-heading-button > span {
  overflow: hidden;
  color: var(--dashboard-text, #111318);
  font-size: 15px;
  font-weight: 650;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-calendar-heading-button {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
  border: 0;
  padding: 0;
  background: transparent;
  color: var(--dashboard-text, #111318);
  cursor: pointer;
}

.dashboard-calendar-heading-button svg {
  flex: 0 0 auto;
  color: #168a58;
}

.dashboard-calendar-heading-button:hover > span {
  color: #168a58;
}

.dashboard-calendar-heading-button:focus-visible {
  outline: 2px solid #168a58;
  outline-offset: 4px;
}

.dashboard-activity-header span,
.dashboard-activity-header > small,
.dashboard-calendar-heading small {
  color: var(--dashboard-subtle, #9ca3af);
  font-size: 10px;
  font-weight: 500;
  line-height: 14px;
}

.dashboard-activity-header > small {
  flex: 0 0 auto;
}

.dashboard-token-summary {
  display: grid;
  min-height: 0;
  flex: 1;
  grid-template-rows: repeat(3, minmax(0, 1fr));
  margin: 0;
}

.dashboard-token-summary > div {
  display: grid;
  min-width: 0;
  grid-template-columns: minmax(94px, 0.8fr) minmax(0, 1.2fr);
  align-items: center;
  gap: 16px;
}

.dashboard-token-summary > div + div {
  border-top: 1px solid var(--dashboard-divider, #eff1f3);
}

.dashboard-token-summary dt {
  color: var(--dashboard-muted, #6b7280);
  font-size: 12px;
  font-weight: 550;
  line-height: 16px;
}

.dashboard-token-summary dd {
  display: grid;
  min-width: 0;
  justify-items: end;
  margin: 0;
}

.dashboard-token-summary strong {
  max-width: 100%;
  overflow: hidden;
  color: var(--dashboard-text, #111318);
  font-size: 19px;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  line-height: 23px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-token-summary dd small {
  max-width: 100%;
  overflow: hidden;
  color: var(--dashboard-subtle, #9ca3af);
  font-size: 9px;
  line-height: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-activity-insights {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  border-top: 1px solid var(--dashboard-divider, #eff1f3);
  padding-top: 6px;
  color: var(--dashboard-subtle, #9ca3af);
  font-size: 9px;
  line-height: 13px;
}

.dashboard-activity-insights span {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 5px;
  white-space: nowrap;
}

.dashboard-activity-insights b {
  overflow: hidden;
  color: var(--dashboard-muted, #6b7280);
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  text-overflow: ellipsis;
}

.dashboard-activity-skeleton {
  display: block;
  border-radius: 4px;
  background: var(--dashboard-skeleton, #f0f1f3);
  animation: dashboard-activity-pulse 1.4s ease-in-out infinite;
}

.dashboard-activity-skeleton-value {
  width: 74%;
  height: 17px;
}

.dashboard-activity-skeleton-meta {
  width: 52%;
  height: 8px;
  margin-top: 3px;
}

.dashboard-activity-skeleton-inline {
  width: 38px;
  height: 9px;
}

.dashboard-calendar-panel {
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-color: var(--dashboard-border, #e4e7eb) transparent;
  scrollbar-width: thin;
}

.dashboard-calendar-panel-inner {
  display: grid;
  width: 100%;
  min-width: 760px;
  height: 208px;
  grid-template-rows: 36px minmax(0, 1fr);
}

.dashboard-calendar-panel-header {
  padding: 0 8px 0 14px;
}

.dashboard-calendar-modes {
  display: inline-grid;
  flex: 0 0 auto;
  grid-template-columns: repeat(3, minmax(46px, 1fr));
  gap: 1px;
  border: 1px solid var(--dashboard-divider, #eff1f3);
  border-radius: 8px;
  padding: 1px;
  background: var(--dashboard-surface-subtle, #f7f8f9);
}

.dashboard-calendar-modes button {
  height: 23px;
  border: 0;
  border-radius: 5px;
  padding: 0 7px;
  background: transparent;
  color: var(--dashboard-muted, #6b7280);
  cursor: pointer;
  font-size: 10px;
  font-weight: 550;
  line-height: 23px;
  transition: background-color 160ms ease, color 160ms ease, box-shadow 160ms ease;
  white-space: nowrap;
}

.dashboard-calendar-modes button:hover {
  color: var(--dashboard-text, #111318);
}

.dashboard-calendar-modes button.active {
  background: var(--dashboard-surface-active, #fff);
  color: var(--dashboard-text, #111318);
  box-shadow: 0 1px 2px rgb(17 24 39 / 8%);
}

.dashboard-calendar-stage {
  position: relative;
  min-width: 0;
  min-height: 0;
  background: var(--dashboard-surface, #fff);
}

.dashboard-calendar-chart {
  width: 100%;
  height: 100%;
}

.dashboard-calendar-loading {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: grid;
  place-items: center;
  background: rgb(255 255 255 / 64%);
  backdrop-filter: blur(1.5px);
}

.dashboard-calendar-empty {
  position: absolute;
  inset: 20px 0 30px;
  display: grid;
  place-items: center;
  color: var(--dashboard-subtle, #9ca3af);
  font-size: 13px;
  pointer-events: none;
}

.dashboard-calendar-legend {
  position: absolute;
  right: 8px;
  bottom: 1px;
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--dashboard-subtle, #9ca3af);
  font-size: 9px;
  pointer-events: none;
}

.dashboard-calendar-legend i {
  width: 11px;
  height: 11px;
  border-radius: 3px;
}

:global(html.dark .dashboard-calendar-stage) {
  background: #111827;
}

:global(html.dark .dashboard-calendar-loading) {
  background: rgb(17 24 39 / 64%);
}

:global(html.dark .dashboard-calendar-modes) {
  border-color: #374151;
}

:global(html.dark .dashboard-calendar-modes button.active) {
  color: #f9fafb;
  box-shadow: none;
}

@keyframes dashboard-activity-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}

@media (min-width: 1181px) and (max-height: 1050px) {
  .dashboard-activity-layout,
  .dashboard-calendar-panel-inner {
    height: 178px;
    min-height: 178px;
  }

  .dashboard-activity-header {
    margin-bottom: 4px;
  }

  .dashboard-token-summary strong {
    font-size: 17px;
    line-height: 20px;
  }

  .dashboard-activity-insights {
    padding-top: 4px;
  }

  .dashboard-calendar-panel-inner {
    grid-template-rows: 32px minmax(0, 1fr);
  }

  .dashboard-calendar-modes button {
    height: 22px;
    line-height: 22px;
  }
}

@media (min-width: 1181px) and (max-height: 940px) {
  .dashboard-activity-layout,
  .dashboard-calendar-panel-inner {
    height: 164px;
    min-height: 164px;
  }

  .dashboard-activity-layout {
    grid-template-columns: minmax(270px, 0.3fr) minmax(0, 1fr);
    gap: 20px;
  }

  .dashboard-activity-summary {
    padding-right: 20px;
  }

  .dashboard-activity-header span {
    display: none;
  }

  .dashboard-token-summary > div {
    gap: 12px;
  }

  .dashboard-token-summary strong {
    font-size: 16px;
    line-height: 19px;
  }

  .dashboard-calendar-panel-inner {
    grid-template-rows: 30px minmax(0, 1fr);
  }
}

@media (max-width: 980px) {
  .dashboard-activity-layout {
    height: auto;
    min-height: 0;
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .dashboard-activity-summary {
    border-right: 0;
    border-bottom: 1px solid var(--dashboard-divider, #eff1f3);
    padding: 0 0 12px;
  }

  .dashboard-token-summary {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    grid-template-rows: auto;
    gap: 0;
  }

  .dashboard-token-summary > div {
    display: grid;
    grid-template-columns: 1fr;
    align-items: start;
    gap: 3px;
    padding: 8px 16px 8px 0;
  }

  .dashboard-token-summary > div + div {
    border-top: 0;
    border-left: 1px solid var(--dashboard-divider, #eff1f3);
    padding-left: 16px;
  }

  .dashboard-token-summary dd {
    justify-items: start;
  }

  .dashboard-activity-insights {
    margin-top: 4px;
  }

  .dashboard-calendar-panel-inner {
    height: 192px;
    min-height: 192px;
  }
}

@media (max-width: 640px) {
  .dashboard-token-summary {
    grid-template-columns: 1fr;
  }

  .dashboard-token-summary > div,
  .dashboard-token-summary > div + div {
    grid-template-columns: 110px minmax(0, 1fr);
    align-items: center;
    border-top: 1px solid var(--dashboard-divider, #eff1f3);
    border-left: 0;
    padding: 9px 0;
  }

  .dashboard-token-summary dd {
    justify-items: end;
  }

  .dashboard-activity-insights {
    align-items: flex-start;
    flex-direction: column;
    gap: 4px;
  }

  .dashboard-calendar-panel-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 6px;
    padding-bottom: 8px;
  }

  .dashboard-calendar-panel-inner {
    height: 224px;
    min-height: 224px;
    grid-template-rows: auto minmax(0, 1fr);
  }

  .dashboard-calendar-modes {
    width: 100%;
  }
}
</style>
