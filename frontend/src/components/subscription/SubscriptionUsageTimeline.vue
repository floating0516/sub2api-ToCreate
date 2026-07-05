<template>
  <div class="space-y-1.5 pt-1">
    <div class="flex items-center justify-between gap-3 text-[11px] leading-4">
      <span class="font-medium text-gray-500 dark:text-dark-400">
        {{ t('userSubscriptions.usageTimeline') }}
      </span>
      <span class="truncate text-right text-gray-400 dark:text-gray-500">
        <template v-if="loading">{{ t('userSubscriptions.timelineLoading') }}</template>
        <template v-else-if="hasUsage">
          {{
            t('userSubscriptions.timelineSummary', {
              amount: formatCurrency(timeline?.total_actual_cost || 0),
              requests: formatNumber(timeline?.total_requests || 0)
            })
          }}
        </template>
        <template v-else>{{ t('userSubscriptions.noUsageInWindow') }}</template>
      </span>
    </div>

    <div
      class="relative grid h-5 gap-px"
      :style="gridStyle"
      :aria-label="t('userSubscriptions.usageTimeline')"
      @mouseleave="activeBucket = null"
    >
      <span
        v-for="bucket in displayBuckets"
        :key="bucket.index"
        class="block h-5 min-w-0 rounded-sm transition-colors"
        :class="bucketClass(bucket)"
        @mouseenter="activeBucket = bucket"
      ></span>
      <div
        v-if="activeBucket && bucketTooltip(activeBucket)"
        class="pointer-events-none absolute bottom-full left-1/2 z-20 mb-2 w-max max-w-[240px] -translate-x-1/2 rounded-md bg-gray-900 px-2.5 py-2 text-[11px] leading-4 text-white opacity-95 shadow-lg dark:bg-gray-100 dark:text-gray-900"
      >
        <div class="whitespace-nowrap font-medium">
          {{ bucketTooltip(activeBucket)?.range }}
        </div>
        <div class="mt-1 flex gap-3 whitespace-nowrap text-gray-200 dark:text-gray-600">
          <span>{{ t('userSubscriptions.usageAmount') }}: {{ formatCurrency(activeBucket.actual_cost) }}</span>
          <span>{{ t('userSubscriptions.requests') }}: {{ formatNumber(activeBucket.requests) }}</span>
        </div>
      </div>
    </div>

    <div v-if="timeline" class="flex justify-between text-[10px] leading-4 text-gray-400 dark:text-gray-500">
      <span>{{ formatEndpoint(timeline.start) }}</span>
      <span>{{ formatEndpoint(timeline.end) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatDateOnly, formatDateTime } from '@/utils/format'
import type {
  SubscriptionUsageTimeline,
  SubscriptionUsageTimelineBucket,
  SubscriptionUsageTimelineWindow
} from '@/types'

const props = withDefaults(defineProps<{
  timeline?: SubscriptionUsageTimeline | null
  window: SubscriptionUsageTimelineWindow
  loading?: boolean
}>(), {
  timeline: null,
  loading: false
})

const { t } = useI18n()
const activeBucket = ref<SubscriptionUsageTimelineBucket | null>(null)

const defaultBucketCount = computed(() => {
  switch (props.window) {
    case 'weekly':
      return 7
    case 'monthly':
      return 30
    case 'daily':
    default:
      return 24
  }
})

const displayBuckets = computed<SubscriptionUsageTimelineBucket[]>(() => {
  if (props.timeline?.buckets?.length) {
    return props.timeline.buckets
  }

  return Array.from({ length: defaultBucketCount.value }, (_, index) => ({
    index,
    start: '',
    end: '',
    requests: 0,
    actual_cost: 0
  }))
})

const gridStyle = computed(() => ({
  gridTemplateColumns: `repeat(${displayBuckets.value.length}, minmax(0, 1fr))`
}))

const maxBucketCost = computed(() => props.timeline?.max_bucket_cost || 0)

const hasUsage = computed(() => {
  return (props.timeline?.total_actual_cost || 0) > 0 || (props.timeline?.total_requests || 0) > 0
})

function bucketClass(bucket: SubscriptionUsageTimelineBucket): string {
  if (props.loading) {
    return 'animate-pulse bg-gray-200 dark:bg-dark-600'
  }

  if (!bucket.actual_cost || maxBucketCost.value <= 0) {
    return 'bg-gray-100 dark:bg-dark-700'
  }

  const ratio = bucket.actual_cost / maxBucketCost.value
  if (ratio >= 0.75) return 'bg-rose-500 dark:bg-rose-400'
  if (ratio >= 0.4) return 'bg-amber-500 dark:bg-amber-400'
  return 'bg-emerald-500 dark:bg-emerald-400'
}

function bucketTooltip(bucket: SubscriptionUsageTimelineBucket): { range: string } | null {
  if (!bucket.start || !bucket.end) return null

  return {
    range: `${formatDateTime(bucket.start)} - ${formatDateTime(bucket.end)}`
  }
}

function formatEndpoint(value: string): string {
  if (props.window === 'daily') {
    return formatDateTime(value, {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    })
  }

  return formatDateOnly(value)
}

function formatCurrency(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '$0.00'
  if (value < 0.01) return `$${value.toFixed(4)}`
  return `$${value.toFixed(2)}`
}

function formatNumber(value: number): string {
  return Number.isFinite(value) ? value.toLocaleString() : '0'
}
</script>
