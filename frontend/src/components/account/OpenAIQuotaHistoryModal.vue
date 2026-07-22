<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.openaiQuotaHistory.title')"
    width="wide"
    @close="emit('close')"
  >
    <div class="space-y-5">
      <div class="flex min-w-0 items-center justify-between gap-3">
        <div class="min-w-0">
          <p class="truncate text-sm font-semibold text-gray-900 dark:text-gray-100">
            {{ account.name }}
          </p>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">OpenAI OAuth · 7d</p>
        </div>
        <button
          type="button"
          class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 disabled:cursor-not-allowed disabled:opacity-50 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
          :disabled="loading"
          :aria-label="t('admin.accounts.openaiQuotaHistory.refresh')"
          :title="t('admin.accounts.openaiQuotaHistory.refresh')"
          @click="loadHistory"
        >
          <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
        </button>
      </div>

      <div v-if="loading && !data" class="grid min-h-24 grid-cols-1 divide-y divide-gray-200 border-y border-gray-200 sm:grid-cols-3 sm:divide-x sm:divide-y-0 dark:divide-gray-700 dark:border-gray-700">
        <div v-for="index in 3" :key="index" class="space-y-2 px-4 py-4">
          <div class="h-3 w-20 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
          <div class="h-6 w-16 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
        </div>
      </div>

      <div
        v-else-if="error"
        class="flex min-h-24 items-center justify-center border-y border-red-200 px-4 text-center text-sm text-red-600 dark:border-red-900/60 dark:text-red-400"
      >
        {{ error }}
      </div>

      <template v-else>
        <div
          v-if="data?.current"
          class="grid grid-cols-1 divide-y divide-gray-200 border-y border-gray-200 sm:grid-cols-3 sm:divide-x sm:divide-y-0 dark:divide-gray-700 dark:border-gray-700"
        >
          <div class="px-4 py-3">
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.openaiQuotaHistory.currentUsage') }}
            </p>
            <p class="mt-1 text-xl font-semibold text-gray-900 tabular-nums dark:text-gray-100">
              {{ formatPercent(data.current.last_used_percent) }}
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
              {{ formatDateTime(data.current.last_observed_at) }}
            </p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.openaiQuotaHistory.cyclePeak') }}
            </p>
            <p class="mt-1 text-xl font-semibold text-emerald-600 tabular-nums dark:text-emerald-400">
              {{ formatPercent(data.current.peak_used_percent) }}
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
              {{ formatDateTime(data.current.cycle_started_at) }}
            </p>
          </div>
          <div class="px-4 py-3">
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.openaiQuotaHistory.expectedReset') }}
            </p>
            <p class="mt-1 text-sm font-semibold text-gray-900 tabular-nums dark:text-gray-100">
              {{ data.current.provider_reset_at ? formatDateTime(data.current.provider_reset_at) : '-' }}
            </p>
          </div>
        </div>

        <div
          v-else
          class="flex min-h-24 items-center justify-center border-y border-gray-200 text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400"
        >
          {{ t('admin.accounts.openaiQuotaHistory.noCurrent') }}
        </div>

        <OpenAIQuotaUsageChart
          :samples="data?.samples ?? []"
          :reset-markers="resetMarkers"
          @reset-marker-click="openResetSourceEditor"
        />

        <div
          v-if="selectedCycle"
          data-testid="reset-source-editor"
          class="border-y border-gray-200 py-3 dark:border-gray-700"
        >
          <div class="flex min-w-0 items-start justify-between gap-3">
            <div class="min-w-0">
              <p class="text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ t('admin.accounts.openaiQuotaHistory.editResetSource') }}
              </p>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                {{ selectedCycle.reset_observed_at ? formatDateTime(selectedCycle.reset_observed_at) : '-' }}
                · {{ t('admin.accounts.openaiQuotaHistory.automaticResult') }}:
                {{ resetSourceLabel(selectedCycle.automatic_reset_source ?? 'unknown') }}
              </p>
            </div>
            <button
              type="button"
              class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-100 hover:text-gray-800 disabled:opacity-50 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
              :disabled="savingSource"
              :aria-label="t('admin.accounts.openaiQuotaHistory.closeSourceEditor')"
              :title="t('admin.accounts.openaiQuotaHistory.closeSourceEditor')"
              @click="closeResetSourceEditor"
            >
              <Icon name="x" size="sm" />
            </button>
          </div>

          <div class="mt-3 flex flex-wrap items-center justify-between gap-3">
            <div
              class="inline-flex max-w-full overflow-hidden rounded border border-gray-200 dark:border-gray-700"
              role="radiogroup"
              :aria-label="t('admin.accounts.openaiQuotaHistory.resetSource')"
            >
              <button
                v-for="option in resetSourceOptions"
                :key="option.value"
                type="button"
                role="radio"
                class="min-h-8 border-r border-gray-200 px-3 py-1.5 text-xs font-medium transition-colors last:border-r-0 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700"
                :class="resetSourceOptionClass(option.value)"
                :aria-checked="resetSourceChoice === option.value"
                :disabled="savingSource"
                @click="resetSourceChoice = option.value"
              >
                {{ option.label }}
              </button>
            </div>

            <button
              type="button"
              data-testid="save-reset-source"
              class="inline-flex h-8 items-center gap-1.5 rounded bg-blue-600 px-3 text-xs font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-blue-500 dark:hover:bg-blue-600"
              :disabled="savingSource"
              @click="saveResetSource"
            >
              <Icon name="check" size="sm" :class="{ 'animate-pulse': savingSource }" />
              {{ t('common.save') }}
            </button>
          </div>
          <p v-if="sourceError" class="mt-2 text-xs text-red-600 dark:text-red-400">
            {{ sourceError }}
          </p>
        </div>

        <section>
          <h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
            {{ t('admin.accounts.openaiQuotaHistory.historyTitle') }}
          </h4>

          <div
            v-if="data?.history.length"
            class="max-h-[48vh] overflow-auto rounded border border-gray-200 dark:border-gray-700"
          >
            <table class="min-w-[820px] w-full table-fixed text-left text-sm">
              <thead class="sticky top-0 z-10 bg-gray-50 text-xs text-gray-500 dark:bg-gray-800 dark:text-gray-400">
                <tr>
                  <th class="w-[25%] px-4 py-2.5 font-medium">
                    {{ t('admin.accounts.openaiQuotaHistory.resetDetected') }}
                  </th>
                  <th class="w-[16%] px-4 py-2.5 font-medium">
                    {{ t('admin.accounts.openaiQuotaHistory.peakBeforeReset') }}
                  </th>
                  <th class="w-[14%] px-4 py-2.5 font-medium">
                    {{ t('admin.accounts.openaiQuotaHistory.afterReset') }}
                  </th>
                  <th class="w-[22%] px-4 py-2.5 font-medium">
                    {{ t('admin.accounts.openaiQuotaHistory.lastObserved') }}
                  </th>
                  <th class="w-[23%] px-4 py-2.5 font-medium">
                    {{ t('admin.accounts.openaiQuotaHistory.resetSource') }}
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
                <tr
                  v-for="cycle in data.history"
                  :key="cycle.id"
                  class="bg-white text-gray-700 dark:bg-gray-900 dark:text-gray-300"
                >
                  <td class="px-4 py-3 tabular-nums">
                    {{ cycle.reset_observed_at ? formatDateTime(cycle.reset_observed_at) : '-' }}
                  </td>
                  <td class="px-4 py-3 font-semibold text-emerald-600 tabular-nums dark:text-emerald-400">
                    {{ formatPercent(cycle.peak_used_percent) }}
                  </td>
                  <td class="px-4 py-3 tabular-nums">
                    {{ cycle.reset_to_percent == null ? '-' : formatPercent(cycle.reset_to_percent) }}
                  </td>
                  <td class="px-4 py-3 tabular-nums text-gray-500 dark:text-gray-400">
                    {{ formatDateTime(cycle.last_observed_at) }}
                  </td>
                  <td class="px-4 py-3">
                    <button
                      type="button"
                      class="inline-flex max-w-full items-center gap-1.5 rounded px-2 py-1 text-xs font-medium transition-colors hover:bg-gray-100 dark:hover:bg-gray-800"
                      :class="resetSourceBadgeClass(effectiveResetSource(cycle))"
                      :aria-label="t('admin.accounts.openaiQuotaHistory.editResetSource')"
                      :title="t('admin.accounts.openaiQuotaHistory.editResetSource')"
                      @click="openCycleResetSourceEditor(cycle)"
                    >
                      <span class="truncate">{{ resetSourceLabel(effectiveResetSource(cycle)) }}</span>
                      <Icon name="edit" size="xs" class="shrink-0" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div
            v-else
            class="flex min-h-28 items-center justify-center rounded border border-gray-200 text-sm text-gray-500 dark:border-gray-700 dark:text-gray-400"
          >
            {{ t('admin.accounts.openaiQuotaHistory.noHistory') }}
          </div>
        </section>
      </template>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import OpenAIQuotaUsageChart from './OpenAIQuotaUsageChart.vue'
import {
  getOpenAIQuotaHistory,
  setOpenAIQuotaResetSource,
  type OpenAIQuotaCycle,
  type OpenAIQuotaHistoryResponse,
  type OpenAIQuotaResetSource,
  type OpenAIQuotaResetSourceSelection
} from '@/api/admin/accounts'
import type { Account } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  show: boolean
  account: Account
}>()

const emit = defineEmits<{
  (event: 'close'): void
}>()

const { t } = useI18n()
const loading = ref(false)
const error = ref('')
const data = ref<OpenAIQuotaHistoryResponse | null>(null)
const selectedCycleID = ref<number | null>(null)
const resetSourceChoice = ref<OpenAIQuotaResetSourceSelection>('auto')
const savingSource = ref(false)
const sourceError = ref('')
let requestID = 0

type QuotaResetMarker = {
  cycleId: number
  observedAt: string
  source: OpenAIQuotaResetSource
}

const effectiveResetSource = (cycle: OpenAIQuotaCycle): OpenAIQuotaResetSource => {
  if (cycle.reset_source === 'manual' || cycle.reset_source === 'provider' || cycle.reset_source === 'unknown') {
    return cycle.reset_source
  }
  return cycle.detection_reason === 'manual_reset' ? 'manual' : 'unknown'
}

const resetMarkers = computed<QuotaResetMarker[]>(() =>
  data.value?.history.flatMap((cycle): QuotaResetMarker[] => (
    cycle.reset_observed_at
      ? [{
          cycleId: cycle.id,
          observedAt: cycle.reset_observed_at,
          source: effectiveResetSource(cycle)
        }]
      : []
  )) ?? []
)

const selectedCycle = computed(() =>
  data.value?.history.find((cycle) => cycle.id === selectedCycleID.value) ?? null
)

const resetSourceOptions = computed<Array<{
  value: OpenAIQuotaResetSourceSelection
  label: string
}>>(() => [
  { value: 'auto', label: t('admin.accounts.openaiQuotaHistory.autoDetection') },
  { value: 'manual', label: t('admin.accounts.openaiQuotaHistory.manualResetLegend') },
  { value: 'provider', label: t('admin.accounts.openaiQuotaHistory.providerResetLegend') }
])

const resetSourceLabel = (source: OpenAIQuotaResetSource): string => {
  switch (source) {
    case 'manual':
      return t('admin.accounts.openaiQuotaHistory.manualResetLegend')
    case 'provider':
      return t('admin.accounts.openaiQuotaHistory.providerResetLegend')
    default:
      return t('admin.accounts.openaiQuotaHistory.unknownResetLegend')
  }
}

const resetSourceBadgeClass = (source: OpenAIQuotaResetSource): string => {
  switch (source) {
    case 'manual':
      return 'text-green-700 dark:text-green-400'
    case 'provider':
      return 'text-red-700 dark:text-red-400'
    default:
      return 'text-gray-600 dark:text-gray-400'
  }
}

const resetSourceOptionClass = (source: OpenAIQuotaResetSourceSelection): string => {
  if (resetSourceChoice.value !== source) {
    return 'bg-white text-gray-600 hover:bg-gray-50 dark:bg-gray-900 dark:text-gray-300 dark:hover:bg-gray-800'
  }
  switch (source) {
    case 'manual':
      return 'bg-green-50 text-green-700 dark:bg-green-950/40 dark:text-green-300'
    case 'provider':
      return 'bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300'
    default:
      return 'bg-gray-100 text-gray-900 dark:bg-gray-700 dark:text-gray-100'
  }
}

const openResetSourceEditor = (marker: QuotaResetMarker) => {
  const cycle = data.value?.history.find((item) => item.id === marker.cycleId)
  if (!cycle) return
  selectedCycleID.value = cycle.id
  resetSourceChoice.value = cycle.reset_source_override ?? 'auto'
  sourceError.value = ''
}

const openCycleResetSourceEditor = (cycle: OpenAIQuotaCycle) => {
  openResetSourceEditor({
    cycleId: cycle.id,
    observedAt: cycle.reset_observed_at ?? cycle.last_observed_at,
    source: effectiveResetSource(cycle)
  })
}

const closeResetSourceEditor = () => {
  if (savingSource.value) return
  selectedCycleID.value = null
  sourceError.value = ''
}

const saveResetSource = async () => {
  const cycle = selectedCycle.value
  if (!cycle || savingSource.value) return
  savingSource.value = true
  sourceError.value = ''
  try {
    await setOpenAIQuotaResetSource(props.account.id, cycle.id, resetSourceChoice.value)
    await loadHistory()
    if (!error.value) {
      selectedCycleID.value = null
    }
  } catch (err) {
    sourceError.value = extractApiErrorMessage(
      err,
      t('admin.accounts.openaiQuotaHistory.saveSourceFailed')
    )
  } finally {
    savingSource.value = false
  }
}

const formatPercent = (value: number): string => {
  const rounded = Math.round(value * 10) / 10
  return `${Number.isInteger(rounded) ? rounded.toFixed(0) : rounded.toFixed(1)}%`
}

const loadHistory = async () => {
  const currentRequestID = ++requestID
  loading.value = true
  error.value = ''
  try {
    const response = await getOpenAIQuotaHistory(props.account.id)
    if (currentRequestID === requestID) {
      data.value = response
    }
  } catch (err) {
    if (currentRequestID === requestID) {
      error.value = extractApiErrorMessage(err, t('admin.accounts.openaiQuotaHistory.loadFailed'))
    }
  } finally {
    if (currentRequestID === requestID) {
      loading.value = false
    }
  }
}

watch(
  () => [props.show, props.account.id] as const,
  ([show]) => {
    requestID += 1
    data.value = null
    error.value = ''
    loading.value = false
    selectedCycleID.value = null
    resetSourceChoice.value = 'auto'
    savingSource.value = false
    sourceError.value = ''
    if (show) {
      void loadHistory()
    }
  },
  { immediate: true }
)
</script>
