<template>
  <svg class="dashboard-sparkline" viewBox="0 0 84 28" aria-hidden="true">
    <polyline :points="points" :stroke="color" />
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  values?: number[]
  color?: string
}>(), {
  values: () => [],
  color: '#22a06b'
})

const points = computed(() => {
  if (!props.values.length) return '2,14 82,14'
  const min = Math.min(...props.values)
  const max = Math.max(...props.values)
  const range = max - min || 1
  const step = props.values.length > 1 ? 80 / (props.values.length - 1) : 0
  return props.values
    .map((value, index) => {
      const x = 2 + index * step
      const y = 24 - ((value - min) / range) * 20
      return `${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(' ')
})
</script>

<style scoped>
.dashboard-sparkline {
  display: block;
  width: 84px;
  height: 28px;
}

.dashboard-sparkline polyline {
  fill: none;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.8;
}
</style>
