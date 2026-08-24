<template>
  <section
    data-testid="ldxp-shop-embed"
    class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800"
  >
    <header class="border-b border-gray-200 px-4 py-3 dark:border-dark-700 sm:px-5">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">
        {{ t('payment.memberRecharge.shopTitle') }}
      </h2>
    </header>

    <div class="flex min-h-[360px] flex-col items-center justify-center px-6 py-12 text-center">
      <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-950/40 dark:text-primary-300">
        <Icon name="externalLink" size="lg" />
      </div>
      <p class="mt-5 max-w-md text-sm leading-6 text-gray-500 dark:text-dark-400">
        {{ t('payment.memberRecharge.shopSubtitle') }}
      </p>
      <a
        :href="SHOP_URL"
        target="_blank"
        rel="noopener noreferrer"
        class="btn btn-primary mt-6 h-11 gap-2 px-5"
        data-testid="ldxp-shop-link"
      >
        <Icon name="externalLink" size="sm" />
        <span>{{ t('payment.memberRecharge.openShop') }}</span>
      </a>
      <span class="mt-3 text-xs text-gray-400 dark:text-dark-500">pay.ldxp.cn</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const SHOP_URL = 'https://pay.ldxp.cn/shop/ToCreate'

const { t } = useI18n()
</script>
