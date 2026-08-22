<template>
  <div data-testid="managed-recharge-plans" class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
    <button
      v-for="plan in plans"
      :key="plan.key"
      :data-testid="`managed-recharge-plan-${plan.key}`"
      type="button"
      class="group flex min-h-[390px] w-full flex-col overflow-hidden rounded-lg border bg-white text-left shadow-sm transition-all hover:-translate-y-0.5 hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:bg-dark-800 dark:focus-visible:ring-offset-dark-900"
      :class="plan.cardClass"
      @click="emit('select', plan.key)"
    >
      <span class="relative block aspect-[16/9] w-full overflow-hidden bg-gray-950">
        <img
          :src="cardImage"
          :alt="plan.title"
          class="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.02]"
          :class="plan.imageClass"
        />
        <span class="absolute left-3 top-3 rounded-md px-2 py-1 text-xs font-semibold shadow-sm" :class="plan.badgeClass">
          {{ plan.badge }}
        </span>
      </span>

      <span class="flex flex-1 flex-col p-5">
        <span class="text-lg font-semibold text-gray-900 dark:text-white">{{ plan.title }}</span>
        <span class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">{{ plan.description }}</span>
        <span class="mt-auto flex items-center border-t border-gray-100 pt-4 text-sm font-semibold text-primary-600 dark:border-dark-700 dark:text-primary-400">
          <Icon name="sparkles" size="sm" class="mr-2" />
          {{ t('payment.memberRecharge.subscribeNow') }}
          <Icon name="arrowRight" size="sm" class="ml-auto transition-transform group-hover:translate-x-0.5" />
        </span>
      </span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import cardImage from '@/assets/managed-recharge/gpt-plus-pro-card.png'
import Icon from '@/components/icons/Icon.vue'
import type { ManagedRechargePlanKey } from './managedRechargePlans'

const emit = defineEmits<{
  select: [plan: ManagedRechargePlanKey]
}>()

const { t } = useI18n()

const plans = computed(() => [
  {
    key: 'plus' as const,
    title: t('payment.memberRecharge.plans.plus.title'),
    badge: t('payment.memberRecharge.plans.plus.badge'),
    description: t('payment.memberRecharge.plans.plus.description'),
    cardClass: 'border-emerald-200 hover:border-emerald-400 dark:border-emerald-900/70 dark:hover:border-emerald-700',
    badgeClass: 'bg-white/90 text-emerald-700 dark:bg-dark-900/90 dark:text-emerald-300',
    imageClass: 'object-right',
  },
  {
    key: 'pro-5x' as const,
    title: t('payment.memberRecharge.plans.pro5x.title'),
    badge: t('payment.memberRecharge.plans.pro5x.badge'),
    description: t('payment.memberRecharge.plans.pro5x.description'),
    cardClass: 'border-amber-200 hover:border-amber-400 dark:border-amber-900/70 dark:hover:border-amber-700',
    badgeClass: 'bg-gray-950/90 text-amber-300',
    imageClass: 'object-left',
  },
  {
    key: 'pro-20x' as const,
    title: t('payment.memberRecharge.plans.pro20x.title'),
    badge: t('payment.memberRecharge.plans.pro20x.badge'),
    description: t('payment.memberRecharge.plans.pro20x.description'),
    cardClass: 'border-rose-200 hover:border-rose-400 dark:border-rose-900/70 dark:hover:border-rose-700',
    badgeClass: 'bg-gray-950/90 text-rose-200',
    imageClass: 'object-center',
  },
])
</script>
