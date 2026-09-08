<template>
  <section class="cs-kpi-grid" :aria-label="ariaLabelText">
    <article
      v-for="card in cards"
      :key="card.key"
      class="cs-kpi-card"
    >
      <span
        v-if="card.dot"
        class="cs-kpi-dot"
        :class="`is-${card.tone}`"
        aria-hidden="true"
      />
      <div class="min-w-0 flex-1">
        <p class="cs-kpi-label">{{ card.label }}</p>
        <p class="cs-kpi-value" :class="`is-${card.tone}`">{{ card.value }}</p>
        <div v-if="card.details.length > 1" class="cs-kpi-details">
          <span v-for="(part, index) in card.details" :key="`${card.key}:${index}`">{{ part }}</span>
        </div>
        <p v-else-if="card.details[0]" class="cs-kpi-detail">{{ card.details[0] }}</p>
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
export type OverviewTone = 'healthy' | 'warning' | 'critical' | 'neutral'

export interface OverviewCard {
  key: string
  label: string
  value: string
  details: string[]
  tone: OverviewTone
  dot?: boolean
}

defineProps<{
  cards: OverviewCard[]
  ariaLabelText: string
}>()
</script>

<style scoped>
.cs-kpi-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

@media (min-width: 1280px) {
  .cs-kpi-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 1rem;
  }
}

.cs-kpi-card {
  display: flex;
  min-height: 7.25rem;
  align-items: flex-start;
  gap: 0.75rem;
  border-radius: 1.25rem;
  background: color-mix(in srgb, var(--tc-surface, #fffefb) 92%, white);
  padding: 1.05rem 1.15rem 1rem;
  box-shadow:
    0 1px 2px rgb(17 24 39 / 4%),
    0 0 0 1px rgb(17 24 39 / 5%);
}

.cs-kpi-dot {
  margin-top: 0.45rem;
  height: 0.5rem;
  width: 0.5rem;
  flex: none;
  border-radius: 9999px;
}

.cs-kpi-dot.is-healthy,
.cs-kpi-value.is-healthy {
  color: #059669;
}

.cs-kpi-dot.is-healthy {
  background: #10b981;
}

.cs-kpi-dot.is-warning,
.cs-kpi-value.is-warning {
  color: #d97706;
}

.cs-kpi-dot.is-warning {
  background: #f59e0b;
}

.cs-kpi-dot.is-critical,
.cs-kpi-value.is-critical {
  color: #dc2626;
}

.cs-kpi-dot.is-critical {
  background: #ef4444;
}

.cs-kpi-dot.is-neutral {
  background: #9ca3af;
}

.cs-kpi-value.is-neutral {
  color: #374151;
}

.cs-kpi-label {
  color: #9ca3af;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.04em;
  line-height: 1.2;
}

.cs-kpi-value {
  margin-top: 0.35rem;
  font-size: 1.75rem;
  font-variant-numeric: tabular-nums;
  font-weight: 700;
  line-height: 1.15;
}

.cs-kpi-detail,
.cs-kpi-details {
  margin-top: 0.45rem;
  color: #9ca3af;
  font-size: 11px;
  line-height: 1.4;
}

.cs-kpi-details {
  display: flex;
  flex-wrap: wrap;
  gap: 0.15rem 0.65rem;
}

:global(.dark) .cs-kpi-card {
  background: rgb(35 38 32);
  box-shadow:
    0 1px 2px rgb(0 0 0 / 20%),
    0 0 0 1px rgb(255 255 255 / 6%);
}

:global(.dark) .cs-kpi-label,
:global(.dark) .cs-kpi-detail,
:global(.dark) .cs-kpi-details {
  color: #8b9086;
}

:global(.dark) .cs-kpi-value.is-healthy {
  color: #34d399;
}

:global(.dark) .cs-kpi-value.is-warning {
  color: #fbbf24;
}

:global(.dark) .cs-kpi-value.is-critical {
  color: #f87171;
}

:global(.dark) .cs-kpi-value.is-neutral {
  color: #f3f4f6;
}
</style>
