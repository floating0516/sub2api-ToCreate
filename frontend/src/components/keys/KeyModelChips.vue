<template>
  <div class="min-w-[160px] max-w-[280px]">
    <div v-if="models.length === 0" class="text-sm text-gray-400 dark:text-gray-500">
      {{ emptyLabel || t('keys.modelsEmpty') }}
    </div>
    <div v-else class="flex flex-wrap items-center gap-1">
      <button
        v-for="name in visible"
        :key="name"
        type="button"
        class="inline-flex max-w-full items-center rounded-md bg-slate-100 px-1.5 py-0.5 font-mono text-[11px] leading-5 text-slate-700 transition-colors hover:bg-slate-200 dark:bg-dark-700 dark:text-slate-200 dark:hover:bg-dark-600"
        :title="t('keys.modelsCopy')"
        @click="copy(name)"
      >
        <span class="truncate">{{ name }}</span>
      </button>
      <span
        v-if="hiddenCount > 0"
        class="text-[11px] text-gray-500 dark:text-gray-400"
      >{{ t('keys.modelsMore', { count: hiddenCount }) }}</span>
    </div>
    <RouterLink
      v-if="showViewAll"
      to="/available-channels"
      class="mt-1 inline-block text-[11px] text-primary-600 hover:underline dark:text-primary-400"
    >
      {{ t('keys.modelsViewAll') }}
    </RouterLink>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'

const props = withDefaults(defineProps<{
  models: string[]
  maxVisible?: number
  showViewAll?: boolean
  emptyLabel?: string
}>(), {
  maxVisible: 4,
  showViewAll: false,
})

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const visible = computed(() => {
  if (!props.maxVisible || props.maxVisible <= 0) return props.models
  return props.models.slice(0, props.maxVisible)
})

const hiddenCount = computed(() => {
  if (!props.maxVisible || props.maxVisible <= 0) return 0
  return Math.max(0, props.models.length - props.maxVisible)
})

async function copy(name: string) {
  await copyToClipboard(name, t('keys.copied'))
}
</script>
