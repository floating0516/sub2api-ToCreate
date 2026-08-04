<template>
  <div class="dashboard-calendar-stage">
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
    <div class="dashboard-calendar-legend" aria-hidden="true">
      <span>{{ t('dashboard.overview.lessUsage') }}</span>
      <i v-for="color in legendColors" :key="color" :style="{ backgroundColor: color }" />
      <span>{{ t('dashboard.overview.moreUsage') }}</span>
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
const updateOptions = { notMerge: false, lazyUpdate: false }
const lightColors = ['#e8f5ee', '#bfe7cf', '#80cfa4', '#43b67d', '#168a58']
const darkColors = ['#173329', '#1d4b38', '#236747', '#2d875a', '#42b875']
let themeObserver: MutationObserver | null = null

const hasData = computed(() => props.data.some((item) => item.value > 0))
const legendColors = computed(() => (isDark.value ? darkColors : lightColors))

const formatValue = (value: number): string => {
  const absolute = Math.abs(value)
  if (absolute >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(1)}B`
  if (absolute >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (absolute >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return Math.round(value).toLocaleString(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US')
}

const chartOption = computed<EChartsOption>(() => {
  const dark = isDark.value
  const colors = dark ? darkColors : lightColors
  const values = props.data.filter((item) => item.value > 0)
  const sortedValues = values.map((item) => item.value).sort((left, right) => left - right)
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
      right: 12,
      bottom: 24,
      left: 34,
      range: [props.startDate, props.endDate],
      cellSize: ['auto', 14],
      splitLine: { show: false },
      itemStyle: {
        color: empty,
        borderColor: surface,
        borderWidth: 3
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
      data: values.map((item) => [item.day, item.value]),
      itemStyle: {
        borderColor: surface,
        borderWidth: 2
      },
      emphasis: {
        itemStyle: {
          borderColor: dark ? '#d1d5db' : '#374151',
          borderWidth: 1,
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
})

onUnmounted(() => {
  themeObserver?.disconnect()
})
</script>

<style scoped>
.dashboard-calendar-stage {
  position: relative;
  height: 184px;
  min-height: 184px;
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
  position: absolute;
  right: 12px;
  bottom: 7px;
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--dashboard-subtle, #9ca3af);
  font-size: 10px;
}

.dashboard-calendar-legend i {
  width: 11px;
  height: 11px;
  border-radius: 2px;
}

:global(html.dark .dashboard-calendar-stage) {
  background: #111827;
}

:global(html.dark .dashboard-calendar-loading) {
  background: rgb(17 24 39 / 64%);
}

@media (min-width: 1181px) and (max-height: 1050px) {
  .dashboard-calendar-stage {
    height: 154px;
    min-height: 154px;
  }
}

@media (min-width: 1181px) and (max-height: 940px) {
  .dashboard-calendar-stage {
    height: 140px;
    min-height: 140px;
  }
}

@media (max-width: 720px) {
  .dashboard-calendar-stage {
    height: 170px;
    min-height: 170px;
  }
}
</style>
