<template>
  <div class="dashboard-activity-layout">
    <aside class="dashboard-activity-summary" :aria-label="t('dashboard.overview.usageSummary')">
      <header class="dashboard-activity-header">
        <h2>{{ t('dashboard.overview.dailyTokenUsage') }}</h2>
        <span>{{ t('dashboard.overview.recentYear') }}</span>
      </header>

      <div class="dashboard-activity-primary">
        <span>{{ t('dashboard.overview.yearlyTotal') }}</span>
        <strong v-if="!loading">{{ formatValue(summary.total) }}</strong>
        <i v-else class="dashboard-activity-skeleton dashboard-activity-skeleton-primary" />
        <small v-if="!loading">
          {{
            t('dashboard.overview.activeDaysValue', {
              active: summary.activeDays,
              total: summary.windowDays
            })
          }}
        </small>
        <i v-else class="dashboard-activity-skeleton dashboard-activity-skeleton-meta" />
      </div>

      <dl class="dashboard-activity-stats">
        <div v-for="item in summaryStats" :key="item.key">
          <dt>{{ item.label }}</dt>
          <dd v-if="!loading">{{ item.value }}</dd>
          <dd v-else><i class="dashboard-activity-skeleton dashboard-activity-skeleton-value" /></dd>
        </div>
      </dl>

      <div class="dashboard-activity-peak">
        <span>{{ t('dashboard.overview.peakDay') }}</span>
        <template v-if="!loading && summary.peakDay">
          <b>{{ formatDay(summary.peakDay) }}</b>
          <strong>{{ formatValue(summary.peakValue) }}</strong>
        </template>
        <span v-else-if="!loading" class="dashboard-activity-peak-empty">--</span>
        <i v-else class="dashboard-activity-skeleton dashboard-activity-skeleton-peak" />
      </div>
    </aside>

    <div class="dashboard-calendar-panel">
      <div class="dashboard-calendar-panel-inner">
        <header class="dashboard-calendar-panel-header">
          <span>{{ t('dashboard.overview.dailyDistribution') }}</span>
          <div class="dashboard-calendar-legend" aria-hidden="true">
            <span>{{ t('dashboard.overview.lessUsage') }}</span>
            <i v-for="color in legendColors" :key="color" :style="{ backgroundColor: color }" />
            <span>{{ t('dashboard.overview.moreUsage') }}</span>
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
          />
          <div v-if="!loading && !hasData" class="dashboard-calendar-empty">
            {{ t('dashboard.noDataAvailable') }}
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

const props = defineProps<{
  data: DashboardCalendarPoint[]
  startDate: string
  endDate: string
  loading?: boolean
}>()

const { t, locale } = useI18n()
const isDark = ref(document.documentElement.classList.contains('dark'))
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

const usageByDay = computed(() => {
  const values = new Map<string, number>()
  props.data.forEach((item) => {
    const value = Number(item.value)
    if (!item.day || !Number.isFinite(value)) return
    values.set(item.day, (values.get(item.day) || 0) + Math.max(0, value))
  })
  return values
})

const usageEntries = computed(() => Array.from(usageByDay.value.entries()))
const hasData = computed(() => usageEntries.value.some(([, value]) => value > 0))
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

const sumRecentDays = (days: number): number => {
  const end = parseDay(props.endDate)
  if (!Number.isFinite(end)) return 0
  let total = 0
  for (let offset = 0; offset < days; offset += 1) {
    total += usageByDay.value.get(formatDayKey(end - offset * DAY_IN_MS)) || 0
  }
  return total
}

const summary = computed(() => {
  const start = parseDay(props.startDate)
  const end = parseDay(props.endDate)
  const windowDays = Number.isFinite(start) && Number.isFinite(end)
    ? Math.max(1, Math.round((end - start) / DAY_IN_MS) + 1)
    : 365
  let total = 0
  let activeDays = 0
  let peakDay = ''
  let peakValue = 0

  usageEntries.value.forEach(([day, value]) => {
    total += value
    if (value > 0) activeDays += 1
    if (value > peakValue) {
      peakDay = day
      peakValue = value
    }
  })

  return {
    total,
    activeDays,
    windowDays,
    endDay: usageByDay.value.get(props.endDate) || 0,
    last7Days: sumRecentDays(7),
    last30Days: sumRecentDays(30),
    dailyAverage: total / windowDays,
    peakDay,
    peakValue
  }
})

const summaryStats = computed(() => [
  {
    key: 'end-day',
    label: t('dashboard.overview.endDayUsage'),
    value: formatValue(summary.value.endDay)
  },
  {
    key: 'last-7-days',
    label: t('dashboard.overview.last7Days'),
    value: formatValue(summary.value.last7Days)
  },
  {
    key: 'last-30-days',
    label: t('dashboard.overview.last30Days'),
    value: formatValue(summary.value.last30Days)
  },
  {
    key: 'daily-average',
    label: t('dashboard.overview.dailyAverage'),
    value: formatValue(summary.value.dailyAverage)
  }
])

const updateCalendarCellSize = () => {
  const stage = calendarStageRef.value
  if (!stage) return
  const widthSize = Math.floor((stage.clientWidth - 48) / 53)
  const heightSize = Math.floor((stage.clientHeight - 22) / 7)
  const nextSize = Math.max(14, Math.min(18, widthSize, heightSize))
  if (nextSize !== calendarCellSize.value) calendarCellSize.value = nextSize
}

const chartOption = computed<EChartsOption>(() => {
  const dark = isDark.value
  const colors = dark ? darkColors : lightColors
  const values = usageEntries.value.filter(([, value]) => value > 0)
  const sortedValues = values.map(([, value]) => value).sort((left, right) => left - right)
  const scaleCeiling = sortedValues[Math.floor((sortedValues.length - 1) * 0.95)] || 0
  const maxValue = Math.max(1, scaleCeiling)
  const surface = dark ? '#111827' : '#ffffff'
  const empty = dark ? '#1c2634' : '#f0f2f4'
  const text = dark ? '#9ca3af' : '#7c838d'

  return {
    backgroundColor: surface,
    animation: true,
    animationDuration: 420,
    animationDurationUpdate: 300,
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
        return `${day}<br><strong>${formatValue(Number(value))}</strong>`
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
      top: 20,
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
      id: 'dashboard-daily-token-calendar',
      type: 'heatmap',
      coordinateSystem: 'calendar',
      data: values.map(([day, value]) => [day, value]),
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
  grid-template-columns: 252px minmax(0, 1fr);
  gap: 22px;
}

.dashboard-activity-summary {
  display: flex;
  min-width: 0;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid var(--dashboard-divider, #eff1f3);
  padding-right: 22px;
}

.dashboard-activity-header,
.dashboard-calendar-panel-header {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.dashboard-activity-header {
  margin-bottom: 10px;
}

.dashboard-activity-header h2 {
  overflow: hidden;
  color: var(--dashboard-text, #111318);
  font-size: 15px;
  font-weight: 650;
  line-height: 22px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-activity-header span,
.dashboard-calendar-panel-header > span {
  flex: 0 0 auto;
  color: var(--dashboard-subtle, #9ca3af);
  font-size: 11px;
  line-height: 16px;
}

.dashboard-activity-primary {
  display: grid;
  min-width: 0;
  gap: 1px;
}

.dashboard-activity-primary > span,
.dashboard-activity-stats dt,
.dashboard-activity-peak > span:first-child {
  color: var(--dashboard-subtle, #9ca3af);
  font-size: 10px;
  font-weight: 500;
  line-height: 14px;
}

.dashboard-activity-primary strong {
  overflow: hidden;
  color: var(--dashboard-text, #111318);
  font-size: 24px;
  font-variant-numeric: tabular-nums;
  font-weight: 650;
  line-height: 29px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-activity-primary small {
  color: var(--dashboard-muted, #6b7280);
  font-size: 10px;
  line-height: 14px;
}

.dashboard-activity-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px 12px;
  margin: 8px 0 0;
}

.dashboard-activity-stats > div {
  min-width: 0;
}

.dashboard-activity-stats dd {
  min-height: 17px;
  margin: 0;
  overflow: hidden;
  color: var(--dashboard-text, #111318);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  line-height: 17px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-activity-peak {
  display: grid;
  min-width: 0;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  margin-top: auto;
  border-top: 1px solid var(--dashboard-divider, #eff1f3);
  padding-top: 6px;
  color: var(--dashboard-muted, #6b7280);
  font-size: 10px;
  line-height: 14px;
}

.dashboard-activity-peak b {
  overflow: hidden;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-activity-peak strong {
  color: var(--dashboard-text, #111318);
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.dashboard-activity-peak-empty {
  grid-column: 2 / -1;
  justify-self: end;
}

.dashboard-activity-skeleton {
  display: block;
  border-radius: 4px;
  background: var(--dashboard-skeleton, #f0f1f3);
  animation: dashboard-activity-pulse 1.4s ease-in-out infinite;
}

.dashboard-activity-skeleton-primary {
  width: 74%;
  height: 25px;
  margin: 2px 0;
}

.dashboard-activity-skeleton-meta {
  width: 52%;
  height: 9px;
  margin-top: 2px;
}

.dashboard-activity-skeleton-value {
  width: 66%;
  height: 12px;
  margin-top: 2px;
}

.dashboard-activity-skeleton-peak {
  width: 58%;
  height: 10px;
  grid-column: 2 / -1;
  justify-self: end;
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
  min-width: 800px;
  height: 208px;
  grid-template-rows: 22px minmax(0, 1fr);
  gap: 0;
}

.dashboard-calendar-panel-header {
  padding: 0 8px 0 14px;
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
  inset: 28px 0 38px;
  display: grid;
  place-items: center;
  color: var(--dashboard-subtle, #9ca3af);
  font-size: 13px;
  pointer-events: none;
}

.dashboard-calendar-legend {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 5px;
  color: var(--dashboard-subtle, #9ca3af);
  font-size: 10px;
}

.dashboard-calendar-legend i {
  width: 12px;
  height: 12px;
  border-radius: 3px;
}

:global(html.dark .dashboard-calendar-stage) {
  background: #111827;
}

:global(html.dark .dashboard-calendar-loading) {
  background: rgb(17 24 39 / 64%);
}

@keyframes dashboard-activity-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}

@media (min-width: 1181px) and (max-height: 1050px) {
  .dashboard-activity-layout {
    height: 178px;
    min-height: 178px;
  }

  .dashboard-activity-header {
    margin-bottom: 6px;
  }

  .dashboard-activity-primary strong {
    font-size: 21px;
    line-height: 25px;
  }

  .dashboard-activity-stats {
    gap: 4px 10px;
    margin-top: 5px;
  }

  .dashboard-activity-stats dd {
    min-height: 15px;
    font-size: 12px;
    line-height: 15px;
  }

  .dashboard-activity-peak {
    padding-top: 4px;
  }

  .dashboard-calendar-panel-inner {
    height: 178px;
  }
}

@media (min-width: 1181px) and (max-height: 940px) {
  .dashboard-activity-layout {
    height: 164px;
    min-height: 164px;
  }

  .dashboard-activity-header {
    margin-bottom: 4px;
  }

  .dashboard-activity-primary strong {
    font-size: 20px;
    line-height: 23px;
  }

  .dashboard-activity-primary small {
    line-height: 12px;
  }

  .dashboard-activity-stats {
    gap: 3px 10px;
    margin-top: 4px;
  }

  .dashboard-activity-peak {
    padding-top: 3px;
  }

  .dashboard-calendar-panel-inner {
    height: 164px;
  }
}

@media (max-width: 880px) {
  .dashboard-activity-layout {
    height: auto;
    min-height: 0;
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .dashboard-activity-summary {
    display: grid;
    grid-template-columns: minmax(150px, 0.8fr) minmax(260px, 1.4fr);
    gap: 12px 20px;
    border-right: 0;
    border-bottom: 1px solid var(--dashboard-divider, #eff1f3);
    padding: 0 0 12px;
  }

  .dashboard-activity-header {
    grid-column: 1 / -1;
    margin-bottom: 0;
  }

  .dashboard-activity-stats {
    margin-top: 0;
  }

  .dashboard-activity-peak {
    grid-column: 1 / -1;
    margin-top: 0;
  }
}

@media (max-width: 720px) {
  .dashboard-activity-summary {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .dashboard-activity-peak {
    grid-column: auto;
  }

  .dashboard-calendar-panel-inner {
    height: 192px;
  }
}
</style>
