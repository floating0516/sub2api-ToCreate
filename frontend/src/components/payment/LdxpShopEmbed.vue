<template>
  <section
    data-testid="ldxp-shop-embed"
    class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800"
  >
    <header class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-dark-700 sm:px-5">
      <div class="min-w-0">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">
          {{ t('payment.memberRecharge.shopTitle') }}
        </h2>
        <p class="mt-0.5 text-sm text-gray-500 dark:text-dark-400">
          {{ t('payment.memberRecharge.shopSubtitle') }}
        </p>
      </div>

      <div class="flex shrink-0 items-center gap-2">
        <button
          type="button"
          class="btn btn-secondary h-9 w-9 p-0"
          :title="t('payment.memberRecharge.reloadShop')"
          :aria-label="t('payment.memberRecharge.reloadShop')"
          @click="reloadShop"
        >
          <Icon name="refresh" size="sm" />
        </button>
        <a
          :href="SHOP_URL"
          target="_blank"
          rel="noopener noreferrer"
          class="btn btn-secondary h-9 gap-2 px-3"
        >
          <Icon name="externalLink" size="sm" />
          <span>{{ t('payment.memberRecharge.openShop') }}</span>
        </a>
      </div>
    </header>

    <div class="relative h-[75vh] min-h-[620px] max-h-[900px] bg-white">
      <div
        v-if="loading"
        class="absolute inset-0 z-10 flex items-center justify-center bg-white dark:bg-dark-900"
        data-testid="ldxp-shop-loading"
      >
        <Icon name="refresh" size="lg" class="animate-spin text-primary-500" />
      </div>
      <iframe
        :key="frameKey"
        :src="SHOP_URL"
        :title="t('payment.memberRecharge.iframeTitle')"
        class="block h-full w-full border-0 bg-white"
        allow="payment; clipboard-read; clipboard-write"
        referrerpolicy="strict-origin-when-cross-origin"
        data-testid="ldxp-shop-frame"
        @load="handleFrameLoad"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const SHOP_URL = 'https://pay.ldxp.cn/shop/ToCreate'
const LOADING_FALLBACK_MS = 7000

const { t } = useI18n()
const loading = ref(true)
const frameKey = ref(0)
let loadingTimer: ReturnType<typeof setTimeout> | undefined

function clearLoadingTimer() {
  if (loadingTimer !== undefined) {
    clearTimeout(loadingTimer)
    loadingTimer = undefined
  }
}

function startLoadingTimer() {
  clearLoadingTimer()
  loadingTimer = setTimeout(() => {
    loading.value = false
    loadingTimer = undefined
  }, LOADING_FALLBACK_MS)
}

function handleFrameLoad() {
  clearLoadingTimer()
  loading.value = false
}

function reloadShop() {
  loading.value = true
  frameKey.value += 1
  startLoadingTimer()
}

onMounted(startLoadingTimer)
onBeforeUnmount(clearLoadingTimer)
</script>
