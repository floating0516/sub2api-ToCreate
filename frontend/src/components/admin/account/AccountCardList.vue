<template>
  <div ref="scrollRef" class="account-card-list" role="list">
    <template v-if="loading">
      <div v-for="index in 4" :key="index" class="account-card-skeleton" aria-hidden="true">
        <div class="flex items-center gap-3 border-b border-gray-100 p-4 dark:border-dark-700">
          <div class="h-4 w-4 animate-pulse rounded bg-gray-200 dark:bg-dark-700" />
          <div class="min-w-0 flex-1 space-y-2">
            <div class="h-4 w-40 animate-pulse rounded bg-gray-200 dark:bg-dark-700" />
            <div class="h-3 w-56 max-w-full animate-pulse rounded bg-gray-100 dark:bg-dark-700" />
          </div>
          <div class="h-6 w-20 animate-pulse rounded bg-gray-200 dark:bg-dark-700" />
        </div>
        <div class="grid gap-px bg-gray-100 sm:grid-cols-2 xl:grid-cols-4 dark:bg-dark-700">
          <div v-for="section in 4" :key="section" class="space-y-3 bg-white p-4 dark:bg-dark-800">
            <div class="h-3 w-20 animate-pulse rounded bg-gray-200 dark:bg-dark-700" />
            <div class="h-4 w-full animate-pulse rounded bg-gray-100 dark:bg-dark-700" />
            <div class="h-4 w-3/4 animate-pulse rounded bg-gray-100 dark:bg-dark-700" />
          </div>
        </div>
      </div>
    </template>

    <div
      v-else-if="!data.length"
      class="flex min-h-64 flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 bg-white/70 p-10 text-center dark:border-dark-600 dark:bg-dark-800/60"
    >
      <Icon name="inbox" size="xl" class="mb-3 text-gray-400 dark:text-dark-500" />
      <p class="text-base font-medium text-gray-800 dark:text-gray-100">{{ t('admin.accounts.noAccountsYet') }}</p>
      <p class="mt-1 max-w-md text-sm text-gray-500 dark:text-dark-400">{{ t('admin.accounts.createFirstAccount') }}</p>
    </div>

    <template v-else>
      <div v-if="virtualPaddingTop" aria-hidden="true" :style="{ height: `${virtualPaddingTop}px` }" />
      <div
        v-for="item in renderRows"
        :key="item.row.id"
        :ref="item.measure ? measureElement : undefined"
        :data-index="item.index"
        :data-row-id="item.row.id"
        data-account-card-row
        role="listitem"
        class="account-card-row"
      >
        <slot :row="item.row" :index="item.index" />
      </div>
      <div v-if="virtualPaddingBottom" aria-hidden="true" :style="{ height: `${virtualPaddingBottom}px` }" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { observeElementRect as observeElementRectDefault, useVirtualizer } from '@tanstack/vue-virtual'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { Account } from '@/types'

const props = withDefaults(defineProps<{
  data: Account[]
  loading?: boolean
  virtualizeThreshold?: number
  estimateCardHeight?: number
  overscan?: number
}>(), {
  loading: false,
  virtualizeThreshold: 50,
  estimateCardHeight: 236,
  overscan: 4,
})

const { t } = useI18n()
const scrollRef = ref<HTMLElement | null>(null)
const isDesktopViewport = ref(
  typeof window === 'undefined' ? true : window.matchMedia('(min-width: 1024px)').matches,
)

let desktopMediaQuery: MediaQueryList | null = null
let desktopMediaListener: ((event: MediaQueryListEvent) => void) | null = null

onMounted(() => {
  desktopMediaQuery = window.matchMedia('(min-width: 1024px)')
  isDesktopViewport.value = desktopMediaQuery.matches
  desktopMediaListener = event => {
    isDesktopViewport.value = event.matches
  }
  desktopMediaQuery.addEventListener?.('change', desktopMediaListener)
})

onUnmounted(() => {
  if (desktopMediaQuery && desktopMediaListener) {
    desktopMediaQuery.removeEventListener?.('change', desktopMediaListener)
  }
  desktopMediaQuery = null
  desktopMediaListener = null
})

const sortedData = computed(() => props.data)
const shouldVirtualize = computed(
  () => isDesktopViewport.value && sortedData.value.length > props.virtualizeThreshold,
)

const estimatedViewportHeight = () => {
  if (typeof window === 'undefined') return 600
  return Math.max(window.innerHeight - 320, 400)
}

const observeElementRectNonZero = (
  instance: any,
  callback: (rect: { width: number; height: number }) => void,
) => observeElementRectDefault(instance, rect => {
  if (rect.height > 0) callback(rect)
})

const virtualizer = useVirtualizer(computed(() => ({
  count: shouldVirtualize.value ? sortedData.value.length : 0,
  getScrollElement: () => scrollRef.value,
  getItemKey: (index: number) => sortedData.value[index]?.id ?? index,
  estimateSize: () => props.estimateCardHeight,
  overscan: props.overscan,
  initialRect: { width: 0, height: estimatedViewportHeight() },
  observeElementRect: observeElementRectNonZero,
  useAnimationFrameWithResizeObserver: true,
})))

const virtualItems = computed(() => virtualizer.value.getVirtualItems())
const virtualPaddingTop = computed(() => {
  const items = virtualItems.value
  return items.length ? items[0].start : 0
})
const virtualPaddingBottom = computed(() => {
  const items = virtualItems.value
  if (!items.length) return 0
  return Math.max(0, virtualizer.value.getTotalSize() - items[items.length - 1].end)
})

const renderRows = computed<Array<{ index: number; row: Account; measure: boolean }>>(() => {
  if (shouldVirtualize.value) {
    return virtualItems.value.map(item => ({
      index: item.index,
      row: sortedData.value[item.index],
      measure: true,
    }))
  }
  return sortedData.value.map((row, index) => ({ index, row, measure: false }))
})

const measureElement = (element: any) => {
  if (element) virtualizer.value.measureElement(element as Element)
}

watch(
  () => sortedData.value.map(row => row.id),
  (current, previous) => {
    if (current.length === previous.length && current.every((id, index) => id === previous[index])) return
    virtualizer.value.measureElement(null)
    virtualizer.value.measure()
  },
  { flush: 'post' },
)

defineExpose({
  virtualizer,
  shouldVirtualize,
  sortedData,
  scrollEl: scrollRef,
})
</script>

<style scoped>
.account-card-list {
  min-height: 0;
  flex: 1;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 2px 4px 4px;
  scrollbar-gutter: stable;
}

.account-card-row,
.account-card-skeleton {
  padding-bottom: 12px;
}

.account-card-skeleton {
  overflow: hidden;
  border: 1px solid rgb(229 231 235);
  border-radius: 8px;
  background: white;
}

:global(.dark) .account-card-skeleton {
  border-color: rgb(55 65 81);
  background: rgb(31 41 55);
}

@media (max-width: 1023px) {
  .account-card-list {
    flex: none;
    overflow: visible;
    padding-inline: 0;
  }
}
</style>
