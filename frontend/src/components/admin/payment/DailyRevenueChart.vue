<template>
  <div class="card p-4">
    <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">
      {{ t('payment.admin.dailyRevenue') }}
    </h3>
    <div class="h-64">
      <div v-if="loading" class="flex h-full items-center justify-center">
        <LoadingSpinner size="md" />
      </div>
      <Line v-else-if="chartData" :data="chartData" :options="chartOptions" />
      <div
        v-else
        class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
      >
        {{ t('payment.admin.noData') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { DailyPaymentStats } from '@/types/payment'
import {
  getOracleChartSurface,
  ORACLE_CHART_COLORS,
  ORACLE_CHART_SERIES
} from '@/utils/oracleTheme'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)

const { t } = useI18n()

const props = defineProps<{
  data: DailyPaymentStats[]
  loading?: boolean
}>()

const colors = [
  [ORACLE_CHART_COLORS[0], `${ORACLE_CHART_COLORS[0]}1a`],
  [ORACLE_CHART_COLORS[1], `${ORACLE_CHART_COLORS[1]}1a`],
  [ORACLE_CHART_COLORS[2], `${ORACLE_CHART_COLORS[2]}1a`],
  [ORACLE_CHART_COLORS[4], `${ORACLE_CHART_COLORS[4]}1a`]
]

const chartData = computed(() => {
  if (!props.data || props.data.length === 0) return null
  const currencies = [...new Set(props.data.flatMap(day => Object.keys(day.amount)))].sort()
  return {
    labels: props.data.map(d => d.date),
    datasets: [
      ...currencies.map((currency, index) => {
        const [borderColor, backgroundColor] = colors[index % colors.length]
        return {
          label: `${currency} ${t('payment.admin.revenue')}`,
          data: props.data.map(day => day.amount[currency] || 0),
          borderColor,
          backgroundColor,
          fill: true,
          tension: 0.3,
          pointRadius: 3,
          pointHoverRadius: 5,
        }
      }),
      {
        label: t('payment.admin.orderCount'),
        data: props.data.map(d => d.count),
        borderColor: ORACLE_CHART_SERIES.green,
        backgroundColor: `${ORACLE_CHART_SERIES.green}1a`,
        fill: false,
        tension: 0.3,
        pointRadius: 3,
        pointHoverRadius: 5,
        yAxisID: 'y1',
      }
    ]
  }
})

const chartOptions = computed(() => {
  const surface = getOracleChartSurface(document.documentElement.classList.contains('dark'))
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { mode: 'index' as const, intersect: false },
    scales: {
      y: {
        type: 'linear' as const,
        display: true,
        position: 'left' as const,
        title: { display: true, text: t('payment.admin.revenue'), color: surface.text },
        grid: { color: surface.grid },
        ticks: { color: surface.text }
      },
      y1: {
        type: 'linear' as const,
        display: true,
        position: 'right' as const,
        title: { display: true, text: t('payment.admin.orderCount'), color: surface.text },
        grid: { drawOnChartArea: false },
        ticks: { color: surface.text }
      },
      x: {
        grid: { color: surface.grid },
        ticks: { color: surface.text }
      }
    },
    plugins: {
      legend: {
        position: 'top' as const,
        labels: { color: surface.text }
      },
      tooltip: {
        backgroundColor: surface.tooltipBackground,
        titleColor: surface.tooltipTitle,
        bodyColor: surface.tooltipBody,
        borderColor: surface.text,
        borderWidth: 2
      }
    }
  }
})
</script>
