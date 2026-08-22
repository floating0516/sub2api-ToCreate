<template>
  <article
    data-testid="managed-recharge-entry-card"
    class="mx-auto w-full max-w-3xl overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800"
  >
    <div class="aspect-[2/1] overflow-hidden bg-gray-950">
      <img
        :src="cardImage"
        :alt="t('payment.memberRecharge.imageAlt')"
        class="h-full w-full object-cover"
      />
    </div>

    <div class="p-5 sm:p-6">
      <div class="flex flex-wrap items-center gap-2">
        <span class="rounded-md bg-emerald-50 px-2 py-1 text-xs font-semibold text-emerald-700 dark:bg-emerald-900/25 dark:text-emerald-300">
          {{ t('payment.memberRecharge.plus') }}
        </span>
        <span class="rounded-md bg-amber-50 px-2 py-1 text-xs font-semibold text-amber-700 dark:bg-amber-900/25 dark:text-amber-300">
          {{ t('payment.memberRecharge.pro') }}
        </span>
      </div>

      <h2 class="mt-4 text-xl font-semibold text-gray-900 dark:text-white">
        {{ t('payment.memberRecharge.title') }}
      </h2>
      <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
        {{ t('payment.memberRecharge.description') }}
      </p>

      <div class="mt-5 grid gap-3 sm:grid-cols-3">
        <div v-for="feature in features" :key="feature.label" class="flex items-start gap-2.5">
          <Icon :name="feature.icon" size="sm" class="mt-0.5 shrink-0 text-primary-600 dark:text-primary-400" />
          <span class="text-sm leading-5 text-gray-700 dark:text-gray-300">{{ feature.label }}</span>
        </div>
      </div>

      <button
        data-testid="managed-recharge-subscribe-button"
        type="button"
        class="btn btn-primary mt-6 min-h-11 w-full px-5 sm:w-auto"
        @click="emit('select')"
      >
        <Icon name="sparkles" size="sm" class="mr-2" />
        {{ t('payment.memberRecharge.subscribeNow') }}
        <Icon name="arrowRight" size="sm" class="ml-2" />
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import cardImage from '@/assets/managed-recharge/gpt-plus-pro-card.png'
import Icon from '@/components/icons/Icon.vue'

const emit = defineEmits<{
  select: []
}>()

const { t } = useI18n()

const features = computed(() => [
  { icon: 'sparkles' as const, label: t('payment.memberRecharge.featurePlan') },
  { icon: 'clock' as const, label: t('payment.memberRecharge.featureProgress') },
  { icon: 'shield' as const, label: t('payment.memberRecharge.featureRefund') },
])
</script>
