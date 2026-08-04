<template>
  <div ref="rootRef" class="dashboard-date-picker">
    <button
      type="button"
      class="dashboard-date-trigger"
      :class="isOpen && 'dashboard-date-trigger-open'"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      @click="toggle"
    >
      <Icon name="calendar" size="sm" class="dashboard-date-icon" />
      <span class="dashboard-date-value">{{ displayValue }}</span>
      <Icon
        name="chevronDown"
        size="sm"
        class="dashboard-date-chevron"
        :class="isOpen && 'rotate-180'"
      />
    </button>

    <Transition name="dashboard-popover">
      <div v-if="isOpen" class="dashboard-date-popover" role="dialog">
        <div class="dashboard-date-presets">
          <button
            v-for="preset in presets"
            :key="preset.value"
            type="button"
            class="dashboard-date-preset"
            :class="isPresetActive(preset) && 'dashboard-date-preset-active'"
            @click="selectPreset(preset)"
          >
            {{ t(preset.labelKey) }}
          </button>
        </div>

        <div class="dashboard-date-fields">
          <label class="dashboard-date-field">
            <span>{{ t('dates.startDate') }}</span>
            <input v-model="draftStart" type="date" :max="draftEnd || today" />
          </label>
          <Icon name="arrowRight" size="sm" class="dashboard-date-arrow" />
          <label class="dashboard-date-field">
            <span>{{ t('dates.endDate') }}</span>
            <input v-model="draftEnd" type="date" :min="draftStart" :max="today" />
          </label>
        </div>

        <div class="dashboard-date-actions">
          <button type="button" class="dashboard-date-cancel" @click="cancel">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="dashboard-date-apply" :disabled="!canApply" @click="apply">
            {{ t('dates.apply') }}
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

interface DatePreset {
  labelKey: string
  value: string
  days: number
}

const props = defineProps<{
  startDate: string
  endDate: string
}>()

const emit = defineEmits<{
  (event: 'update:startDate', value: string): void
  (event: 'update:endDate', value: string): void
  (event: 'change', value: { startDate: string; endDate: string }): void
}>()

const { t } = useI18n()
const rootRef = ref<HTMLElement | null>(null)
const isOpen = ref(false)
const draftStart = ref(props.startDate)
const draftEnd = ref(props.endDate)

const formatDateInput = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const today = computed(() => formatDateInput(new Date()))
const displayValue = computed(() => `${props.startDate} ～ ${props.endDate}`)
const canApply = computed(
  () => Boolean(draftStart.value && draftEnd.value && draftStart.value <= draftEnd.value)
)

const presets: DatePreset[] = [
  { labelKey: 'dates.today', value: 'today', days: 1 },
  { labelKey: 'dates.yesterday', value: 'yesterday', days: 0 },
  { labelKey: 'dates.last7Days', value: '7days', days: 7 },
  { labelKey: 'dates.last30Days', value: '30days', days: 30 }
]

const rangeForPreset = (preset: DatePreset) => {
  const end = new Date()
  if (preset.value === 'yesterday') end.setDate(end.getDate() - 1)
  const start = new Date(end)
  start.setDate(start.getDate() - Math.max(0, preset.days - 1))
  return { start: formatDateInput(start), end: formatDateInput(end) }
}

const isPresetActive = (preset: DatePreset): boolean => {
  const range = rangeForPreset(preset)
  return draftStart.value === range.start && draftEnd.value === range.end
}

const selectPreset = (preset: DatePreset) => {
  const range = rangeForPreset(preset)
  draftStart.value = range.start
  draftEnd.value = range.end
}

const toggle = () => {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    draftStart.value = props.startDate
    draftEnd.value = props.endDate
  }
}

const cancel = () => {
  draftStart.value = props.startDate
  draftEnd.value = props.endDate
  isOpen.value = false
}

const apply = () => {
  if (!canApply.value) return
  emit('update:startDate', draftStart.value)
  emit('update:endDate', draftEnd.value)
  emit('change', { startDate: draftStart.value, endDate: draftEnd.value })
  isOpen.value = false
}

const handlePointerDown = (event: MouseEvent) => {
  if (rootRef.value && !rootRef.value.contains(event.target as Node)) cancel()
}

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && isOpen.value) cancel()
}

watch(
  () => [props.startDate, props.endDate],
  ([start, end]) => {
    if (!isOpen.value) {
      draftStart.value = start
      draftEnd.value = end
    }
  }
)

onMounted(() => {
  document.addEventListener('mousedown', handlePointerDown)
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('mousedown', handlePointerDown)
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.dashboard-date-picker {
  position: relative;
}

.dashboard-date-trigger {
  display: flex;
  width: 242px;
  height: 44px;
  align-items: center;
  gap: 9px;
  border: 1px solid #e2e5e9;
  border-radius: 9px;
  background: #fff;
  padding: 0 12px;
  color: #4b5563;
  font-size: 13px;
  font-weight: 500;
  transition: border-color 160ms ease, box-shadow 160ms ease;
}

.dashboard-date-trigger:hover,
.dashboard-date-trigger-open {
  border-color: #cfd4da;
}

.dashboard-date-trigger-open {
  box-shadow: 0 0 0 3px rgb(17 24 39 / 5%);
}

.dashboard-date-icon,
.dashboard-date-chevron,
.dashboard-date-arrow {
  flex: 0 0 auto;
  color: #8b929d;
}

.dashboard-date-chevron {
  margin-left: auto;
  transition: transform 160ms ease;
}

.dashboard-date-value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.dashboard-date-popover {
  position: absolute;
  z-index: 10;
  top: calc(100% + 8px);
  right: 0;
  width: min(430px, calc(100vw - 32px));
  overflow: hidden;
  border: 1px solid #e2e5e9;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 12px 32px rgb(17 24 39 / 10%);
}

.dashboard-date-presets {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
  padding: 12px;
  border-bottom: 1px solid #eef0f2;
}

.dashboard-date-preset {
  height: 34px;
  border-radius: 7px;
  color: #6b7280;
  font-size: 13px;
}

.dashboard-date-preset:hover {
  background: #f6f7f8;
  color: #111318;
}

.dashboard-date-preset-active {
  background: #f0fdf7;
  color: #18794e;
  font-weight: 600;
}

.dashboard-date-fields {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 18px minmax(0, 1fr);
  align-items: end;
  gap: 10px;
  padding: 15px 16px;
}

.dashboard-date-field {
  display: grid;
  gap: 6px;
  min-width: 0;
  color: #6b7280;
  font-size: 12px;
  font-weight: 500;
}

.dashboard-date-field input {
  width: 100%;
  height: 38px;
  min-width: 0;
  border: 1px solid #e2e5e9;
  border-radius: 7px;
  background: #fff;
  padding: 0 9px;
  color: #30343b;
  font-size: 13px;
  outline: none;
}

.dashboard-date-field input:focus {
  border-color: #aeb5be;
  box-shadow: 0 0 0 3px rgb(17 24 39 / 4%);
}

.dashboard-date-arrow {
  margin-bottom: 11px;
}

.dashboard-date-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  border-top: 1px solid #eef0f2;
  padding: 10px 12px;
}

.dashboard-date-cancel,
.dashboard-date-apply {
  height: 34px;
  border-radius: 7px;
  padding: 0 14px;
  font-size: 13px;
  font-weight: 600;
}

.dashboard-date-cancel {
  color: #606873;
}

.dashboard-date-cancel:hover {
  background: #f5f6f7;
}

.dashboard-date-apply {
  background: #17191d;
  color: #fff;
}

.dashboard-date-apply:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.dashboard-popover-enter-active,
.dashboard-popover-leave-active {
  transition: opacity 140ms ease, transform 140ms ease;
}

.dashboard-popover-enter-from,
.dashboard-popover-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

:global(.dark) .dashboard-date-trigger,
:global(.dark) .dashboard-date-popover,
:global(.dark) .dashboard-date-field input {
  border-color: #374151;
  background: #111827;
  color: #d1d5db;
}

:global(.dark) .dashboard-date-presets,
:global(.dark) .dashboard-date-actions {
  border-color: #283342;
}

:global(.dark) .dashboard-date-preset:hover,
:global(.dark) .dashboard-date-cancel:hover {
  background: #1f2937;
  color: #f9fafb;
}

@media (max-width: 560px) {
  .dashboard-date-trigger {
    width: min(242px, calc(100vw - 32px));
  }

  .dashboard-date-presets {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dashboard-date-fields {
    grid-template-columns: 1fr;
  }

  .dashboard-date-arrow {
    display: none;
  }
}
</style>
