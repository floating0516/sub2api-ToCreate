<template>
  <div class="dashboard-trend-stage">
    <div v-if="loading" class="dashboard-chart-loading">
      <LoadingSpinner size="md" />
    </div>
    <Line v-if="hasData" :data="chartData" :options="chartOptions" />
    <div v-else-if="!loading" class="dashboard-chart-empty">
      {{ t('dashboard.noDataAvailable') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Title,
  Tooltip,
  type ChartData,
  type ChartOptions
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

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
const isDark = computed(() => document.documentElement.classList.contains('dark'))
const hasData = computed(() => props.series.some((item) => item.values.some((value) => value > 0)))

const formatTokens = (value: number): string => {
  const absolute = Math.abs(value)
  if (absolute >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(1)}B`
  if (absolute >= 1_000_000) return `${(value / 1_000_000).toFixed(0)}M`
  if (absolute >= 1_000) return `${(value / 1_000).toFixed(0)}K`
  return value.toLocaleString()
}

const chartData = computed<ChartData<'line'>>(() => ({
  labels: props.labels,
  datasets: props.series.map((item, index) => ({
    label: item.label,
    data: item.values,
    borderColor: item.color,
    backgroundColor: index === 0 ? `${item.color}16` : `${item.color}08`,
    borderWidth: 2,
    pointRadius: 2.5,
    pointHoverRadius: 4.5,
    pointBorderWidth: 0,
    pointBackgroundColor: item.color,
    fill: index === 0,
    tension: 0.12,
    spanGaps: true
  }))
}))

const chartOptions = computed<ChartOptions<'line'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: { duration: 220 },
  interaction: { intersect: false, mode: 'index' },
  layout: { padding: { top: 8, right: 8, bottom: 0, left: 2 } },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: isDark.value ? '#111827' : '#ffffff',
      titleColor: isDark.value ? '#f9fafb' : '#111318',
      bodyColor: isDark.value ? '#d1d5db' : '#4b5563',
      borderColor: isDark.value ? '#374151' : '#e2e5e9',
      borderWidth: 1,
      padding: 12,
      displayColors: true,
      boxWidth: 7,
      boxHeight: 7,
      usePointStyle: true,
      callbacks: {
        label: (context) => ` ${context.dataset.label}: ${formatTokens(Number(context.raw))} Token`
      }
    }
  },
  scales: {
    x: {
      border: { display: false },
      grid: { display: false },
      ticks: {
        color: isDark.value ? '#9ca3af' : '#7c838d',
        maxRotation: 0,
        autoSkip: true,
        maxTicksLimit: 12,
        font: { size: 11, family: 'inherit' }
      }
    },
    y: {
      beginAtZero: true,
      border: { display: false },
      grid: {
        color: isDark.value ? '#283342' : '#e9ebee',
        borderDash: [4, 5],
        drawTicks: false
      },
      ticks: {
        color: isDark.value ? '#9ca3af' : '#7c838d',
        padding: 10,
        maxTicksLimit: 7,
        font: { size: 11, family: 'inherit' },
        callback: (value) => formatTokens(Number(value))
      },
      title: {
        display: true,
        text: 'Token',
        align: 'end',
        color: isDark.value ? '#9ca3af' : '#7c838d',
        font: { size: 11, family: 'inherit', weight: 500 }
      }
    }
  }
}))
</script>

<style scoped>
.dashboard-trend-stage {
  position: relative;
  height: 350px;
  min-height: 350px;
}

.dashboard-chart-loading {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: grid;
  place-items: center;
  background: rgb(255 255 255 / 70%);
}

.dashboard-chart-empty {
  display: grid;
  height: 100%;
  place-items: center;
  color: #9ca3af;
  font-size: 13px;
}

:global(.dark) .dashboard-chart-loading {
  background: rgb(17 24 39 / 70%);
}

@media (max-width: 720px) {
  .dashboard-trend-stage {
    height: 300px;
    min-height: 300px;
  }
}
</style>
