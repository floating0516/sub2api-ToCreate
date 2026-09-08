<template>
  <section
    class="card flex min-h-[360px] flex-col overflow-visible !rounded-3xl !border-0 !p-6 shadow-sm ring-1 ring-gray-900/5 dark:!bg-dark-800 dark:ring-dark-700"
  >
    <div class="card-header mb-4 flex shrink-0 flex-wrap items-start justify-between gap-3 !border-0 !p-0">
      <div class="min-w-0">
        <h2 class="flex items-center gap-2 text-sm font-bold text-gray-900 dark:text-white">
          <span class="inline-flex h-4 w-4 text-emerald-500" aria-hidden="true">
            <Icon name="grid" size="sm" />
          </span>
          {{ t('channelStatus.matrix.title') }}
        </h2>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
          {{ t('channelStatus.matrix.description') }}
        </p>
      </div>
      <span class="badge badge-gray shrink-0">{{ t('channelStatus.matrix.bucket') }}</span>
    </div>

    <div class="card-body min-h-0 flex-1 !p-0">
      <div
        v-if="rows.length"
        class="matrix-scroll max-h-[min(42vh,420px)] max-w-full overflow-auto rounded-2xl bg-gray-50/60 p-2 dark:bg-dark-900/30"
      >
        <div class="matrix-table w-full">
          <div
            class="matrix-header matrix-row sticky top-0 z-[3] bg-gray-50 text-[10px] font-semibold uppercase tracking-wide text-gray-500 dark:bg-dark-900 dark:text-gray-400"
          >
            <span>{{ t('channelStatus.matrix.dimension') }}</span>
            <span>{{ t('monitorCommon.availabilityPrefix') }}</span>
            <span>{{ t('channelStatus.matrix.latency') }}</span>
            <span>{{ t('channelStatus.matrix.ping') }}</span>
            <span class="pulse-axis flex justify-between gap-3">
              <i class="not-italic">{{ t('monitorCommon.past') }}</i>
              <i class="not-italic">{{ t('monitorCommon.now') }}</i>
            </span>
          </div>
          <button
            v-for="row in rows"
            :key="row.id"
            type="button"
            class="matrix-row w-full border-b border-gray-100/80 text-left dark:border-dark-700/60"
            @click="emit('rowClick', row.id)"
          >
            <div class="dimension-cell flex min-w-0 items-center gap-2 bg-white dark:bg-dark-800" :title="row.label">
              <span :class="['status-dot', cellClass(row.status)]"></span>
              <strong class="truncate text-xs font-semibold text-gray-800 dark:text-gray-100">{{ row.label }}</strong>
            </div>
            <strong class="summary-value bg-white text-xs font-medium tabular-nums text-gray-600 dark:bg-dark-800 dark:text-gray-300">
              {{ formatAvailability(row.availability) }}
            </strong>
            <strong class="summary-value bg-white text-xs font-medium tabular-nums text-gray-600 dark:bg-dark-800 dark:text-gray-300">
              {{ row.latency }}
            </strong>
            <strong class="summary-value bg-white text-xs font-medium tabular-nums text-gray-600 dark:bg-dark-800 dark:text-gray-300">
              {{ row.ping }}
            </strong>
            <div class="pulse-track grid items-stretch" :style="trackStyle(row.cells.length)">
              <span
                v-for="(cell, index) in row.cells"
                :key="`${row.id}:${index}`"
                class="pulse-cell relative rounded-sm border-0 p-0 outline-offset-1"
                :class="[cellClass(cell.status), cell.status === 'empty' ? 'is-empty' : 'has-data']"
                :title="cell.title"
                @mouseenter="showTooltip($event, cell)"
                @mousemove="moveTooltip($event)"
                @mouseleave="hideTooltip"
              />
            </div>
          </button>
        </div>
      </div>
      <div v-else class="flex min-h-[200px] items-center justify-center py-8">
        <EmptyState
          :title="t('channelStatus.matrix.emptyTitle')"
          :description="t('channelStatus.empty.filterDescription')"
        />
      </div>

      <div class="mt-4 flex flex-col gap-2" :aria-label="t('channelStatus.matrix.legendAria')">
        <div class="flex items-center gap-2 text-[11px] text-gray-500 dark:text-gray-400">
          <span class="shrink-0">{{ t('channelStatus.matrix.bad') }}</span>
          <div class="score-legend h-2.5 flex-1 overflow-hidden rounded-full"></div>
          <span class="shrink-0">{{ t('channelStatus.matrix.good') }}</span>
        </div>
        <div class="flex flex-wrap gap-4 text-[11px] text-gray-500 dark:text-gray-400">
          <span class="inline-flex items-center gap-1.5"><i class="status-dot health-score10"></i>{{ t('channelStatus.matrix.healthyLegend') }}</span>
          <span class="inline-flex items-center gap-1.5"><i class="status-dot health-score6"></i>{{ t('channelStatus.matrix.warningLegend') }}</span>
          <span class="inline-flex items-center gap-1.5"><i class="status-dot health-score2"></i>{{ t('channelStatus.matrix.criticalLegend') }}</span>
          <span class="inline-flex items-center gap-1.5"><i class="status-dot health-unknown"></i>{{ t('channelStatus.matrix.unknownLegend') }}</span>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="floatingTooltip.visible"
        class="matrix-floating-tooltip"
        :style="{ left: `${floatingTooltip.x}px`, top: `${floatingTooltip.y}px` }"
        role="tooltip"
      >
        <span
          v-for="(line, index) in floatingTooltip.lines"
          :key="`${index}:${line}`"
          class="matrix-floating-tooltip-line"
          :class="index === 0 ? 'matrix-floating-tooltip-title' : ''"
        >
          {{ line }}
        </span>
      </div>
    </Teleport>
  </section>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

export interface MatrixCell {
  status: string
  title: string
  lines?: string[]
}

export interface MatrixRow {
  id: number
  label: string
  status: string
  availability: number | null
  latency: string
  ping: string
  cells: MatrixCell[]
}

defineProps<{
  rows: MatrixRow[]
}>()

const emit = defineEmits<{
  (e: 'rowClick', id: number): void
}>()

const { t } = useI18n()

const floatingTooltip = reactive({
  visible: false,
  x: 0,
  y: 0,
  lines: [] as string[],
})

function cellClass(status: string) {
  if (status === 'operational') return 'health-score10'
  if (status === 'degraded') return 'health-score6'
  if (status === 'failed' || status === 'error') return 'health-score1'
  return 'health-unknown'
}

function formatAvailability(value: number | null) {
  if (value == null || Number.isNaN(value)) return '—'
  return `${value.toFixed(2)}%`
}

function trackStyle(count: number) {
  return {
    gridTemplateColumns: `repeat(${Math.max(1, count)}, minmax(0, 1fr))`,
  }
}

function showTooltip(event: MouseEvent, cell: MatrixCell) {
  const lines = cell.lines?.length ? cell.lines : cell.title.split(' · ').filter(Boolean)
  if (!lines.length) return
  floatingTooltip.lines = lines
  floatingTooltip.visible = true
  moveTooltip(event)
}

function moveTooltip(event: MouseEvent) {
  floatingTooltip.x = event.clientX
  floatingTooltip.y = event.clientY - 10
}

function hideTooltip() {
  floatingTooltip.visible = false
}
</script>

<style scoped>
.matrix-row {
  display: grid;
  grid-template-columns:
    minmax(120px, 1.2fr)
    minmax(52px, 0.34fr)
    minmax(58px, 0.36fr)
    minmax(52px, 0.34fr)
    minmax(120px, 2.8fr);
  align-items: center;
  gap: 0.5rem clamp(0.25rem, 0.8vw, 0.625rem);
  min-height: 2.25rem;
}
.matrix-table,
.pulse-track {
  min-width: 0;
}
.pulse-track {
  height: 14px;
  gap: 2px;
}
.status-dot {
  display: inline-block;
  height: 0.5rem;
  width: 0.5rem;
  flex: none;
  border-radius: 9999px;
}
.health-score10 { background: #16a34a; }
.health-score9  { background: #22c55e; }
.health-score8  { background: #4ade80; }
.health-score7  { background: #a3e635; }
.health-score6  { background: #facc15; }
.health-score5  { background: #fbbf24; }
.health-score4  { background: #f59e0b; }
.health-score3  { background: #f97316; }
.health-score2  { background: #fb7185; }
.health-score1  { background: #f87171; }
.health-score0  { background: rgb(239, 67, 67); }
.health-healthy  { background: #22c55e; }
.health-warning  { background: #f59e0b; }
.health-critical { background: #ef4444; }
.health-unknown  { background: #9ca3af; }
.score-legend {
  background: linear-gradient(
    90deg,
    rgb(239, 67, 67) 0%,
    #f87171 15%,
    #f97316 30%,
    #f59e0b 45%,
    #facc15 55%,
    #a3e635 70%,
    #22c55e 85%,
    #16a34a 100%
  );
}
.pulse-cell {
  position: relative;
  min-width: 0;
  min-height: 14px;
}
.pulse-cell.has-data {
  cursor: help;
}
.pulse-cell.is-empty {
  opacity: 0.55;
  cursor: default;
}
.pulse-cell.has-data:hover,
.pulse-cell.has-data:focus-visible {
  outline: 2px solid rgb(16 185 129 / 0.55);
  outline-offset: 1px;
  z-index: 5;
}
.matrix-floating-tooltip {
  pointer-events: none;
  position: fixed;
  z-index: 9999;
  min-width: 11.5rem;
  max-width: min(18rem, calc(100vw - 1.5rem));
  transform: translate(-50%, -100%);
  border-radius: 0.75rem;
  border: 1px solid rgb(229 231 235);
  background: rgb(255 255 255);
  padding: 0.5rem 0.625rem;
  box-shadow: 0 18px 40px -12px rgb(0 0 0 / 0.28);
  white-space: nowrap;
}
:global(.dark) .matrix-floating-tooltip {
  border-color: rgb(55 65 81);
  background: rgb(17 24 39);
  color: rgb(229 231 235);
}
.matrix-floating-tooltip-line {
  display: block;
  font-size: 11px;
  line-height: 1.45;
  color: rgb(75 85 99);
}
:global(.dark) .matrix-floating-tooltip-line {
  color: rgb(209 213 219);
}
.matrix-floating-tooltip-title {
  margin-bottom: 0.2rem;
  font-weight: 600;
  color: rgb(17 24 39);
}
:global(.dark) .matrix-floating-tooltip-title {
  color: rgb(243 244 246);
}
@media (max-width: 640px) {
  .matrix-row {
    grid-template-columns: minmax(88px, 1fr) minmax(48px, 0.45fr) minmax(54px, 0.5fr) minmax(96px, 2.6fr);
    gap: 0.35rem;
  }
  .matrix-row > :nth-child(4) {
    display: none;
  }
}
</style>
