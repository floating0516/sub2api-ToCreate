<template>
  <div class="dashboard-trend-stage">
    <div v-if="loading" class="dashboard-chart-loading">
      <LoadingSpinner size="md" />
    </div>
    <VChart
      v-if="hasData"
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
const updateOptions = { notMerge: false, lazyUpdate: false }
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
  const axisColor = dark ? '#9ca3af' : '#7c838d'
  const gridColor = dark ? '#283342' : '#e9ebee'
  const tooltipBackground = dark ? '#111827' : '#ffffff'
  const tooltipBorder = dark ? '#374151' : '#e2e5e9'
  const tooltipText = dark ? '#d1d5db' : '#4b5563'
  const tooltipTitle = dark ? '#f9fafb' : '#111318'
  const showSymbols = props.labels.length <= 31

  return {
    animation: true,
    animationThreshold: 2000,
    animationDuration: 480,
    animationDurationUpdate: 360,
    animationEasing: 'cubicOut',
    animationEasingUpdate: 'cubicOut',
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
          color: dark ? '#4b5563' : '#cfd4da',
          width: 1,
          type: 'dashed'
        }
      },
      extraCssText: `border-radius: 8px; box-shadow: 0 10px 28px rgba(17, 24, 39, ${dark ? '0.24' : '0.10'}); color: ${tooltipTitle};`
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
      name: 'Token',
      nameLocation: 'end',
      nameGap: 14,
      nameTextStyle: {
        color: axisColor,
        fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
        fontSize: 11,
        align: 'right'
      },
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
      id: `dashboard-series-${index}`,
      name: item.label,
      type: 'line',
      data: item.values,
      showSymbol: showSymbols,
      symbol: 'circle',
      symbolSize: 5,
      smooth: 0.12,
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
          borderColor: dark ? '#111827' : '#ffffff',
          borderWidth: 2,
          shadowBlur: 7,
          shadowColor: hexToRgba(item.color, 0.28)
        }
      },
      tooltip: {
        valueFormatter: (value: unknown) => `${formatTokens(Number(value))} Token`
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
  background: rgb(255 255 255 / 64%);
  backdrop-filter: blur(1.5px);
}

.dashboard-chart-empty {
  display: grid;
  height: 100%;
  place-items: center;
  color: #9ca3af;
  font-size: 13px;
}

:global(.dark) .dashboard-chart-loading {
  background: rgb(17 24 39 / 64%);
}

@media (max-width: 720px) {
  .dashboard-trend-stage {
    height: 300px;
    min-height: 300px;
  }
}
</style>
