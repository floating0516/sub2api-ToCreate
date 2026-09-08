<template>
  <section class="cs-matrix-card">
    <header class="cs-matrix-header">
      <div class="min-w-0">
        <h2 class="cs-matrix-title">
          <span class="cs-matrix-title-icon" aria-hidden="true">
            <Icon name="grid" size="sm" />
          </span>
          {{ t('channelStatus.matrix.title') }}
        </h2>
        <p class="cs-matrix-desc">{{ t('channelStatus.matrix.description') }}</p>
      </div>
    </header>

    <div v-if="rows.length" class="cs-matrix-scroll">
      <div class="cs-matrix-table">
        <div class="cs-matrix-row cs-matrix-axis">
          <span>{{ t('channelStatus.matrix.dimension') }}</span>
          <span>{{ t('monitorCommon.availabilityPrefix') }}</span>
          <span class="cs-matrix-axis-range">
            <i>{{ t('monitorCommon.past') }}</i>
            <i>{{ t('monitorCommon.now') }}</i>
          </span>
        </div>
        <button
          v-for="row in rows"
          :key="row.id"
          type="button"
          class="cs-matrix-row cs-matrix-data"
          @click="emit('rowClick', row.id)"
        >
          <div class="cs-matrix-name" :title="row.label">
            <span class="cs-status-dot" :class="statusClass(row.status)"></span>
            <strong>{{ row.label }}</strong>
          </div>
          <strong class="cs-matrix-avail" :class="availabilityTone(row.availability)">
            {{ formatAvailability(row.availability) }}
          </strong>
          <div class="cs-matrix-track" :style="trackStyle(row.cells.length)">
            <span
              v-for="(cell, index) in row.cells"
              :key="`${row.id}:${index}`"
              class="cs-matrix-cell"
              :class="statusClass(cell.status)"
              :title="cell.title"
            />
          </div>
        </button>
      </div>
    </div>
    <EmptyState
      v-else
      :title="t('channelStatus.matrix.emptyTitle')"
      :description="t('channelStatus.empty.description')"
    />

    <div class="cs-matrix-legend">
      <span><i class="cs-status-dot is-operational"></i>{{ t('channelStatus.matrix.healthyLegend') }}</span>
      <span><i class="cs-status-dot is-degraded"></i>{{ t('channelStatus.matrix.warningLegend') }}</span>
      <span><i class="cs-status-dot is-failed"></i>{{ t('channelStatus.matrix.criticalLegend') }}</span>
      <span><i class="cs-status-dot is-empty"></i>{{ t('channelStatus.matrix.unknownLegend') }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

export interface MatrixCell {
  status: string
  title: string
}

export interface MatrixRow {
  id: number
  label: string
  status: string
  availability: number | null
  cells: MatrixCell[]
}

defineProps<{
  rows: MatrixRow[]
}>()

const emit = defineEmits<{
  (e: 'rowClick', id: number): void
}>()

const { t } = useI18n()

function statusClass(status: string) {
  if (status === 'operational') return 'is-operational'
  if (status === 'degraded') return 'is-degraded'
  if (status === 'failed' || status === 'error') return 'is-failed'
  return 'is-empty'
}

function availabilityTone(value: number | null) {
  if (value == null || Number.isNaN(value)) return 'is-empty'
  if (value >= 99) return 'is-operational'
  if (value >= 95) return 'is-degraded'
  return 'is-failed'
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
</script>

<style scoped>
.cs-matrix-card {
  display: flex;
  min-height: 22rem;
  flex-direction: column;
  border-radius: 1.5rem;
  background: color-mix(in srgb, var(--tc-surface, #fffefb) 92%, white);
  padding: 1.25rem 1.35rem 1.1rem;
  box-shadow:
    0 1px 2px rgb(17 24 39 / 4%),
    0 0 0 1px rgb(17 24 39 / 5%);
}

.cs-matrix-header {
  margin-bottom: 1rem;
}

.cs-matrix-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #1f2937;
  font-size: 0.875rem;
  font-weight: 700;
}

.cs-matrix-title-icon {
  display: inline-flex;
  color: #10b981;
}

.cs-matrix-desc {
  margin-top: 0.2rem;
  color: #6b7280;
  font-size: 12px;
}

.cs-matrix-scroll {
  max-height: min(42vh, 420px);
  overflow: auto;
  border-radius: 1rem;
  background: rgb(248 250 249);
  padding: 0.5rem;
}

.cs-matrix-table {
  min-width: 36rem;
}

.cs-matrix-row {
  display: grid;
  grid-template-columns: minmax(140px, 1.3fr) minmax(72px, 0.4fr) minmax(160px, 2.8fr);
  align-items: center;
  gap: 0.65rem;
  min-height: 2.15rem;
}

.cs-matrix-axis {
  position: sticky;
  top: 0;
  z-index: 1;
  color: #6b7280;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  background: rgb(248 250 249);
}

.cs-matrix-axis-range {
  display: flex;
  justify-content: space-between;
}

.cs-matrix-axis-range i {
  font-style: normal;
}

.cs-matrix-data {
  width: 100%;
  border-bottom: 1px solid rgb(17 24 39 / 4%);
  text-align: left;
}

.cs-matrix-data:hover {
  background: rgb(255 255 255 / 70%);
}

.cs-matrix-name {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.5rem;
}

.cs-matrix-name strong {
  overflow: hidden;
  color: #1f2937;
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cs-matrix-avail {
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.cs-matrix-track {
  display: grid;
  height: 14px;
  gap: 2px;
}

.cs-matrix-cell {
  min-width: 0;
  border-radius: 2px;
}

.cs-status-dot {
  display: inline-block;
  height: 0.5rem;
  width: 0.5rem;
  flex: none;
  border-radius: 9999px;
}

.is-operational {
  background: #10b981;
  color: #059669;
}

.is-degraded {
  background: #f59e0b;
  color: #d97706;
}

.is-failed {
  background: #ef4444;
  color: #dc2626;
}

.is-empty {
  background: #d1d5db;
  color: #9ca3af;
}

.cs-matrix-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  margin-top: 0.9rem;
  color: #6b7280;
  font-size: 11px;
}

.cs-matrix-legend span {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
}

:global(.dark) .cs-matrix-card {
  background: rgb(35 38 32);
  box-shadow:
    0 1px 2px rgb(0 0 0 / 20%),
    0 0 0 1px rgb(255 255 255 / 6%);
}

:global(.dark) .cs-matrix-title,
:global(.dark) .cs-matrix-name strong {
  color: #f3f4f6;
}

:global(.dark) .cs-matrix-desc,
:global(.dark) .cs-matrix-axis,
:global(.dark) .cs-matrix-legend {
  color: #9ca3af;
}

:global(.dark) .cs-matrix-scroll,
:global(.dark) .cs-matrix-axis {
  background: rgb(23 25 22);
}

:global(.dark) .cs-matrix-data {
  border-bottom-color: rgb(255 255 255 / 6%);
}

:global(.dark) .cs-matrix-data:hover {
  background: rgb(255 255 255 / 4%);
}

:global(.dark) .is-empty {
  background: #4b5563;
}
</style>
