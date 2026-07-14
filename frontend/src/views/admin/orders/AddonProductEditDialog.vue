<template>
  <BaseDialog
    :show="show"
    :title="t('payment.admin.editAddonProduct')"
    width="wide"
    @close="emit('close')"
  >
    <form id="addon-product-form" class="space-y-4" @submit.prevent="handleSave">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('payment.admin.addonSku') }}</label>
          <input :value="product?.sku || ''" data-testid="addon-sku" type="text" class="input bg-gray-50 dark:bg-dark-800" readonly />
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.addonProductName') }} <span class="text-red-500">*</span></label>
          <input v-model="form.name" data-testid="addon-name" type="text" class="input" required />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('payment.admin.addonQuota') }} <span class="text-red-500">*</span></label>
          <input v-model.number="form.quota_usd" data-testid="addon-quota" type="number" min="0.0000000001" step="0.01" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.addonPrice') }} <span class="text-red-500">*</span></label>
          <input v-model.number="form.price" data-testid="addon-price" type="number" min="0.01" step="0.01" class="input" required />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('payment.admin.originalPrice') }}</label>
          <input v-model.number="form.original_price" data-testid="addon-original-price" type="number" min="0" step="0.01" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('payment.admin.sortOrder') }}</label>
          <input v-model.number="form.sort_order" data-testid="addon-sort-order" type="number" min="0" step="1" class="input" required />
        </div>
      </div>

      <div class="grid grid-cols-1 gap-2 rounded-md bg-gray-50 p-3 text-sm dark:bg-dark-800 sm:grid-cols-2">
        <div>
          <span class="text-gray-500 dark:text-gray-400">{{ t('payment.admin.addonCnyPayPreview') }}</span>
          <span class="ml-2 font-semibold text-gray-900 dark:text-white">{{ cnyPricePreview }}</span>
        </div>
        <div>
          <span class="text-gray-500 dark:text-gray-400">{{ t('payment.admin.addonBalancePayPreview') }}</span>
          <span class="ml-2 font-semibold text-gray-900 dark:text-white">{{ balancePricePreview }}</span>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <label class="text-sm text-gray-700 dark:text-gray-300">{{ t('payment.admin.forSale') }}</label>
        <button
          type="button"
          role="switch"
          :aria-checked="form.for_sale"
          :class="[
            'relative inline-flex h-6 w-11 shrink-0 rounded-full border-2 border-transparent transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            form.for_sale ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'
          ]"
          @click="form.for_sale = !form.for_sale"
        >
          <span :class="[
            'pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow transition-transform',
            form.for_sale ? 'translate-x-5' : 'translate-x-0'
          ]" />
        </button>
      </div>

      <p class="flex items-start gap-2 text-xs leading-relaxed text-gray-500 dark:text-gray-400">
        <Icon name="infoCircle" size="sm" class="mt-0.5 shrink-0" />
        <span>{{ t('payment.admin.addonPriceChangeNotice') }}</span>
      </p>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="submit" form="addon-product-form" class="btn btn-primary" :disabled="saving">
          {{ saving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminPaymentAPI } from '@/api/admin/payment'
import type { AdminPaymentConfig, UpdateSubscriptionAddonProductRequest } from '@/api/admin/payment'
import type { SubscriptionAddonProduct } from '@/types/payment'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatPaymentAmount } from '@/components/payment/currency'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  show: boolean
  product: SubscriptionAddonProduct | null
  paymentConfig?: AdminPaymentConfig | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)
const form = reactive({
  name: '',
  quota_usd: 0,
  price: 0,
  original_price: '' as number | '',
  for_sale: true,
  sort_order: 0,
})

const unitPrice = computed(() => {
  const quota = Number(form.quota_usd)
  const price = Number(form.price)
  if (!Number.isFinite(quota) || !Number.isFinite(price) || quota <= 0 || price <= 0) return 0
  return price / quota
})

const cnyPricePreview = computed(() => {
  const price = Number(form.price)
  const rate = Number(props.paymentConfig?.subscription_usd_to_cny_rate) || 0
  const quota = Number(form.quota_usd)
  if (!Number.isFinite(price) || price <= 0) return formatPaymentAmount(0, 'CNY')
  const amount = Math.round(price * (rate > 0 ? rate : 1) * 100) / 100
  const unit = Number.isFinite(quota) && quota > 0 ? amount / quota : 0
  return `${formatPaymentAmount(amount, 'CNY')} (¥${unit.toFixed(4)} / $1)`
})

const balancePricePreview = computed(() => {
  const price = Number(form.price)
  const amount = Number.isFinite(price) && price > 0 ? price : 0
  return `${formatPaymentAmount(amount, 'USD')} ($${unitPrice.value.toFixed(4)} / $1)`
})

watch([() => props.show, () => props.product], ([visible, product]) => {
  if (!visible || !product) return
  Object.assign(form, {
    name: product.name,
    quota_usd: product.quota_usd,
    price: product.price,
    original_price: product.original_price ?? '',
    for_sale: product.for_sale,
    sort_order: product.sort_order,
  })
}, { immediate: true })

function buildPayload(): UpdateSubscriptionAddonProductRequest | null {
  const name = form.name.trim()
  const quotaUSD = Number(form.quota_usd)
  const price = Number(form.price)
  const sortOrder = Number(form.sort_order)
  const originalPrice = form.original_price === '' ? null : Number(form.original_price)

  if (!name) {
    appStore.showError(t('payment.admin.addonNameRequired'))
    return null
  }
  if (!Number.isFinite(quotaUSD) || quotaUSD <= 0) {
    appStore.showError(t('payment.admin.addonQuotaRequired'))
    return null
  }
  if (!Number.isFinite(price) || price <= 0) {
    appStore.showError(t('payment.admin.priceRequired'))
    return null
  }
  if (!Number.isInteger(sortOrder) || sortOrder < 0) {
    appStore.showError(t('payment.admin.sortOrderInvalid'))
    return null
  }
  if (originalPrice !== null && (!Number.isFinite(originalPrice) || originalPrice < 0)) {
    appStore.showError(t('payment.admin.originalPriceInvalid'))
    return null
  }

  return {
    name,
    quota_usd: quotaUSD,
    price,
    original_price: originalPrice,
    for_sale: form.for_sale,
    sort_order: sortOrder,
  }
}

async function handleSave() {
  if (!props.product || saving.value) return
  const payload = buildPayload()
  if (!payload) return

  saving.value = true
  try {
    await adminPaymentAPI.updateAddonProduct(props.product.id, payload)
    appStore.showSuccess(t('common.saved'))
    emit('close')
    emit('saved')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    saving.value = false
  }
}
</script>
