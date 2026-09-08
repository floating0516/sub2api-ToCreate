<template>
  <div class="dashboard-trend-stage">
    <div v-if="loading" class="dashboard-chart-loading">
      <LoadingSpinner size="md" />
    </div>
    <VChart
      v-if="hasData"
      :key="isDark ? 'dark' : 'light'"
      class="dashboard-echart"
      :option="chartOption"
      :update-options="updateOptions"
      :autoresize="{ throttle: 80 }"
    />
    <div v-else-if="!loading" class="dashboard-chart-empty">
      {{ t('dashboard.noDataAvailable') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import {
  AriaComponent,
  GridComponent,
  TooltipComponent
} from 'echarts/components'
import { UniversalTransition } from 'echarts/features'
import { CanvasRenderer } from 'echarts/renderers'
import type { EChartsOption } from 'echarts'
import VChart from 'vue-echarts'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { activityAxisColor } from '@/components/user/dashboard/dashboardActivityTheme'

use([
  CanvasRenderer,
  LineChart,
  GridComponent,
  TooltipComponent,
  AriaComponent,
  UniversalTransition
])

export interface DashboardTrendSeries {
  label: string
  color: string
  values: number[]
}

const props = defineProps<{
  labels: string[]
  series: DashboardTrendSeries[]
  loading?: boolean
}>()

const { t } = useI18n()
const isDark = ref(document.documentElement.classList.contains('dark'))
const updateOptions = {
  notMerge: false,
  lazyUpdate: false,
  replaceMerge: ['series', 'xAxis', 'yAxis']
}
let themeObserver: MutationObserver | null = null

const hasData = computed(() => props.series.some((item) => item.values.some((value) => value > 0)))

const formatTokens = (value: number): string => {
  const absolute = Math.abs(value)
  if (absolute >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(1)}B`
  if (absolute >= 1_000_000) return `${(value / 1_000_000).toFixed(0)}M`
  if (absolute >= 1_000) return `${(value / 1_000).toFixed(0)}K`
  return Math.round(value).toLocaleString()
}

const hexToRgba = (color: string, opacity: number): string => {
  const hex = color.replace('#', '')
  if (!/^[0-9a-fA-F]{6}$/.test(hex)) return color
  const red = Number.parseInt(hex.slice(0, 2), 16)
  const green = Number.parseInt(hex.slice(2, 4), 16)
  const blue = Number.parseInt(hex.slice(4, 6), 16)
  return `rgba(${red}, ${green}, ${blue}, ${opacity})`
}

const chartOption = computed<EChartsOption>(() => {
  const dark = isDark.value
  const axisColor = activityAxisColor(dark)
  const gridColor = dark ? 'rgba(240, 238, 230, 0.08)' : 'rgba(42, 47, 40, 0.08)'
  const tooltipBackground = dark ? '#232620' : '#fffefb'
  const tooltipBorder = dark ? 'rgba(240, 238, 230, 0.12)' : 'rgba(42, 47, 40, 0.12)'
  const tooltipText = axisColor
  const tooltipTitle = dark ? '#f1eee7' : '#3e413b'
  const chartSurface = dark ? '#232620' : '#fffefb'
  const showSymbols = props.labels.length <= 31

  return {
    backgroundColor: chartSurface,
    animation: true,
    animationThreshold: 2000,
    animationDuration: 300,
    animationDurationUpdate: 240,
    animationEasing: 'cubicOut',
    animationEasingUpdate: 'cubicOut',
    textStyle: {
      color: axisColor,
      fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif'
    },
    aria: {
      enabled: true,
      decal: { show: false }
    },
    grid: {
      top: 30,
      right: 12,
      bottom: 8,
      left: 8,
      containLabel: true
    },
    tooltip: {
      trigger: 'axis',
      confine: true,
      backgroundColor: tooltipBackground,
      borderColor: tooltipBorder,
      borderWidth: 1,
      padding: [10, 12],
      transitionDuration: 0.14,
      textStyle: {
        color: tooltipText,
        fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
        fontSize: 12
      },
      axisPointer: {
        type: 'line',
        lineStyle: {
          color: dark ? 'rgba(227, 180, 143, 0.45)' : 'rgba(170, 113, 73, 0.35)',
          width: 1,
          type: 'dashed'
        }
      },
      extraCssText: `max-width: calc(100% - 24px); white-space: normal; overflow-wrap: anywhere; border-radius: 8px; box-shadow: 0 10px 28px rgba(17, 24, 39, ${dark ? '0.24' : '0.10'}); color: ${tooltipTitle};`
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: props.labels,
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: {
        color: axisColor,
        fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
        fontSize: 11,
        hideOverlap: true,
        margin: 14
      }
    },
    yAxis: {
      type: 'value',
      min: 0,
      splitNumber: 6,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        color: axisColor,
        fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
        fontSize: 11,
        margin: 12,
        formatter: (value: number) => formatTokens(value)
      },
      splitLine: {
        show: true,
        lineStyle: {
          color: gridColor,
          width: 1,
          type: 'dashed'
        }
      }
    },
    series: props.series.map((item, index) => ({
      id: `dashboard-series-${item.label}`,
      name: item.label,
      type: 'line',
      data: item.values,
      showSymbol: showSymbols,
      symbol: 'circle',
      symbolSize: 5,
      smooth: 0.32,
      smoothMonotone: 'x',
      connectNulls: true,
      clip: true,
      universalTransition: true,
      lineStyle: {
        color: item.color,
        width: index === 0 ? 2.2 : 2,
        cap: 'round',
        join: 'round'
      },
      itemStyle: {
        color: item.color,
        borderWidth: 0
      },
      areaStyle: index === 0
        ? {
            color: hexToRgba(item.color, dark ? 0.12 : 0.09),
            origin: 'start'
          }
        : undefined,
      emphasis: {
        focus: 'series',
        scale: 1.45,
        lineStyle: { width: 2.8 },
        itemStyle: {
          color: item.color,
          borderColor: chartSurface,
          borderWidth: 2,
          shadowBlur: 7,
          shadowColor: hexToRgba(item.color, 0.28)
        }
      },
      tooltip: {
        valueFormatter: (value: unknown) => formatTokens(Number(value))
      }
    }))
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
.dashboard-trend-stage {
  position: relative;
  height: 350px;
  min-height: 350px;
  background: var(--dashboard-surface, #fffefb);
  transition: background-color 160ms ease;
}

.dashboard-echart {
  width: 100%;
  height: 100%;
}

.dashboard-chart-loading {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: grid;
  place-items: center;
  background: color-mix(in srgb, var(--dashboard-surface, #fffefb) 64%, transparent);
  backdrop-filter: blur(1.5px);
}

.dashboard-chart-empty {
  display: grid;
  height: 100%;
  place-items: center;
  color: var(--dashboard-subtle, #999b94);
  font-size: 13px;
}

@media (min-width: 1181px) and (max-height: 1050px) {
  .dashboard-trend-stage {
    height: 290px;
    min-height: 290px;
  }
}

@media (min-width: 1181px) and (max-height: 940px) {
  .dashboard-trend-stage {
    height: 270px;
    min-height: 270px;
  }
}

@media (max-width: 720px) {
  .dashboard-trend-stage {
    height: 300px;
    min-height: 300px;
  }
}
</style>
