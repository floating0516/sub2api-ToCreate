<template>
  <section data-testid="openai-quota-usage-chart">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
        {{ t('admin.accounts.openaiQuotaHistory.chartTitle') }}
      </h4>
      <div class="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
        <span class="inline-flex items-center gap-1.5">
          <span class="h-0.5 w-5 bg-blue-500"></span>
          {{ t('admin.accounts.openaiQuotaHistory.usageLegend') }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <span class="w-5 border-t border-dashed border-red-500"></span>
          {{ t('admin.accounts.openaiQuotaHistory.resetLegend') }}
        </span>
      </div>
    </div>

    <div v-if="chartData" class="h-60 min-h-60 w-full">
      <Line :data="chartData" :options="chartOptions" :plugins="chartPlugins" />
    </div>
    <div
      v-else
      class="flex h-40 items-center justify-center border-y border-gray-200 text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400"
    >
      {{ t('admin.accounts.openaiQuotaHistory.noSamples') }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  Filler,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Tooltip,
  type ChartData,
  type ChartOptions,
  type Plugin
} from 'chart.js'
import { Line } from 'vue-chartjs'
import type { OpenAIQuotaSample } from '@/api/admin/accounts'
import { formatDateTime } from '@/utils/format'

ChartJS.register(LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const props = defineProps<{
  samples: OpenAIQuotaSample[]
  resetTimes: string[]
}>()

const { t } = useI18n()

const isDarkMode = computed(() =>
  typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
)

const colors = computed(() => ({
  line: isDarkMode.value ? '#60a5fa' : '#2563eb',
  fill: isDarkMode.value ? 'rgba(96, 165, 250, 0.12)' : 'rgba(37, 99, 235, 0.10)',
  reset: isDarkMode.value ? '#f87171' : '#dc2626',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb',
  text: isDarkMode.value ? '#9ca3af' : '#6b7280'
}))

const normalizedSamples = computed(() =>
  props.samples
    .map((sample) => ({
      x: new Date(sample.observed_at).getTime(),
      y: sample.used_percent
    }))
    .filter((sample) => Number.isFinite(sample.x) && Number.isFinite(sample.y))
    .sort((left, right) => left.x - right.x)
)

const resetMarkers = computed(() =>
  props.resetTimes
    .map((value) => new Date(value).getTime())
    .filter((value) => Number.isFinite(value))
    .sort((left, right) => left - right)
)

const chartData = computed<ChartData<'line', { x: number; y: number }[]> | null>(() => {
  if (normalizedSamples.value.length === 0) return null

  return {
    datasets: [
      {
        label: t('admin.accounts.openaiQuotaHistory.usageLegend'),
        data: normalizedSamples.value,
        borderColor: colors.value.line,
        backgroundColor: colors.value.fill,
        borderWidth: 2,
        fill: true,
        tension: 0.18,
        pointRadius: normalizedSamples.value.length === 1 ? 3 : 0,
        pointHoverRadius: 4,
        pointHitRadius: 10
      }
    ]
  }
})

const formatAxisTime = (value: number): string => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const points = normalizedSamples.value
  const span = points.length > 1 ? points[points.length - 1].x - points[0].x : 0
  const options: Intl.DateTimeFormatOptions = span >= 48 * 60 * 60 * 1000
    ? { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }
    : { hour: '2-digit', minute: '2-digit' }
  return new Intl.DateTimeFormat(undefined, options).format(date)
}

const formatPercent = (value: number): string => {
  const rounded = Math.round(value * 10) / 10
  return `${Number.isInteger(rounded) ? rounded.toFixed(0) : rounded.toFixed(1)}%`
}

const chartOptions = computed<ChartOptions<'line'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  parsing: false,
  animation: false,
  interaction: { intersect: false, mode: 'nearest', axis: 'x' },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: isDarkMode.value ? '#1f2937' : '#ffffff',
      titleColor: isDarkMode.value ? '#f3f4f6' : '#111827',
      bodyColor: isDarkMode.value ? '#d1d5db' : '#4b5563',
      borderColor: colors.value.grid,
      borderWidth: 1,
      padding: 10,
      displayColors: false,
      callbacks: {
        title: (items) => {
          const timestamp = items[0]?.parsed.x
          return typeof timestamp === 'number' ? formatDateTime(new Date(timestamp).toISOString()) : ''
        },
        label: (context) => {
          const usedPercent = context.parsed.y
          return `${t('admin.accounts.openaiQuotaHistory.usageLegend')}: ${typeof usedPercent === 'number' ? formatPercent(usedPercent) : '-'}`
        }
      }
    }
  },
  scales: {
    x: {
      type: 'linear',
      grid: { display: false },
      ticks: {
        color: colors.value.text,
        maxTicksLimit: 7,
        autoSkip: true,
        font: { size: 10 },
        callback: (value) => formatAxisTime(Number(value))
      }
    },
    y: {
      type: 'linear',
      min: 0,
      max: 100,
      grid: { color: colors.value.grid },
      ticks: {
        color: colors.value.text,
        stepSize: 20,
        font: { size: 10 },
        callback: (value) => `${value}%`
      }
    }
  }
}))

const resetMarkerPlugin: Plugin<'line'> = {
  id: 'openAIQuotaResetMarkers',
  afterDatasetsDraw(chart) {
    const xScale = chart.scales.x
    if (!xScale || resetMarkers.value.length === 0) return

    const { ctx, chartArea } = chart
    ctx.save()
    ctx.strokeStyle = colors.value.reset
    ctx.lineWidth = 1.5
    ctx.setLineDash([5, 4])
    for (const resetAt of resetMarkers.value) {
      const x = xScale.getPixelForValue(resetAt)
      if (x < chartArea.left || x > chartArea.right) continue
      ctx.beginPath()
      ctx.moveTo(x, chartArea.top)
      ctx.lineTo(x, chartArea.bottom)
      ctx.stroke()
    }
    ctx.restore()
  }
}

const chartPlugins = [resetMarkerPlugin]
</script>
