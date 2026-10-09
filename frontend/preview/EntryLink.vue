<template>
  <a class="entry-link" :class="`entry-link--${variant}`" href="https://api.lihe.chat/login">
    <span class="entry-label"><slot /></span>
    <span class="entry-arrow" aria-hidden="true">
      <Icon name="arrowRight" :stroke-width="1.6" />
    </span>
  </a>
</template>

<script setup lang="ts">
import Icon from '../src/components/icons/Icon.vue'
withDefaults(defineProps<{ variant?: 'primary' | 'quiet' }>(), { variant: 'primary' })
</script>
