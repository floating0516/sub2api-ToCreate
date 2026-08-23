<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <button
        data-testid="managed-recharge-back"
        type="button"
        class="btn btn-secondary h-9 w-9 p-0"
        :title="t('payment.memberRecharge.backToPurchase')"
        :aria-label="t('payment.memberRecharge.backToPurchase')"
        @click="goBack"
      >
        <Icon name="arrowLeft" size="sm" />
      </button>

      <LdxpShopEmbed />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LdxpShopEmbed from '@/components/payment/LdxpShopEmbed.vue'

const { t } = useI18n()
const router = useRouter()

function goBack() {
  router.push({ path: '/purchase', query: { tab: 'member' } })
}
</script>
