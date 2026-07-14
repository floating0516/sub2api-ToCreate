<template>
  <AppLayout>
    <div class="space-y-4">
      <div class="inline-flex max-w-full rounded-md bg-gray-100 p-1 dark:bg-dark-800" role="tablist" :aria-label="t('payment.admin.productCatalog')">
        <button
          type="button"
          role="tab"
          data-testid="catalog-tab-plans"
          :aria-selected="activeCatalog === 'plans'"
          :class="catalogTabClass(activeCatalog === 'plans')"
          @click="selectCatalog('plans')"
        >
          <Icon name="creditCard" size="sm" />
          <span>{{ t('payment.admin.catalogPlans') }}</span>
        </button>
        <button
          type="button"
          role="tab"
          data-testid="catalog-tab-addons"
          :aria-selected="activeCatalog === 'addons'"
          :class="catalogTabClass(activeCatalog === 'addons')"
          @click="selectCatalog('addons')"
        >
          <Icon name="bolt" size="sm" />
          <span>{{ t('payment.admin.catalogAddons') }}</span>
        </button>
      </div>

      <section v-if="activeCatalog === 'plans'" class="space-y-4" data-testid="plans-catalog">
        <div class="flex flex-wrap items-center justify-end gap-2 border-b border-gray-200 pb-4 dark:border-dark-600">
          <button type="button" class="btn btn-secondary" :disabled="plansLoading" :title="t('common.refresh')" @click="loadPlans">
            <Icon name="refresh" size="md" :class="plansLoading ? 'animate-spin' : ''" />
          </button>
          <button type="button" class="btn btn-primary" @click="openPlanEdit(null)">
            <Icon name="plus" size="sm" />
            <span>{{ t('payment.admin.createPlan') }}</span>
          </button>
        </div>

        <DataTable :columns="planColumns" :data="plans" :loading="plansLoading">
          <template #cell-name="{ value, row }">
            <span class="text-sm font-medium" :class="getPlanNameClass(row.group_id)">{{ value }}</span>
          </template>
          <template #cell-group_id="{ value }">
            <span v-if="isGroupMissing(value)" class="text-sm">
              <span class="text-gray-400">#{{ value }}</span>
              <span class="ml-1 badge badge-danger">{{ t('payment.admin.groupMissing') }}</span>
            </span>
            <GroupBadge
              v-else-if="getGroup(value)"
              :name="getGroup(value)!.name"
              :platform="getGroup(value)!.platform"
              :rate-multiplier="getGroup(value)!.rate_multiplier"
            />
            <span v-else class="text-sm text-gray-400">-</span>
          </template>
          <template #cell-price="{ value, row }">
            <div class="text-sm">
              <span class="font-medium text-gray-900 dark:text-white">${{ (value ?? 0).toFixed(2) }}</span>
              <span v-if="row.original_price" class="ml-1 text-xs text-gray-400 line-through">${{ row.original_price.toFixed(2) }}</span>
            </div>
          </template>
          <template #cell-validity_days="{ value, row }">
            <span class="text-sm">{{ value }} {{ t('payment.admin.' + (row.validity_unit || 'days')) }}</span>
          </template>
          <template #cell-for_sale="{ value, row }">
            <button
              type="button"
              role="switch"
              :aria-checked="value"
              :class="saleSwitchClass(value, 'small')"
              @click="toggleForSale(row)"
            >
              <span :class="saleSwitchKnobClass(value, 'small')" />
            </button>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-2">
              <button type="button" class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400" @click="openPlanEdit(row)">
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>
              <button type="button" class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400" @click="confirmDeletePlan(row)">
                <Icon name="trash" size="sm" />
                <span class="text-xs">{{ t('common.delete') }}</span>
              </button>
            </div>
          </template>
        </DataTable>
      </section>

      <section v-else class="space-y-4" data-testid="addons-catalog">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 pb-4 dark:border-dark-600">
          <div class="flex min-w-0 items-center gap-3">
            <button
              type="button"
              role="switch"
              :aria-checked="paymentConfig?.addon_purchase_enabled === true"
              :disabled="configSaving || !paymentConfig"
              :class="saleSwitchClass(paymentConfig?.addon_purchase_enabled === true, 'normal')"
              @click="toggleAddonSales"
            >
              <span :class="saleSwitchKnobClass(paymentConfig?.addon_purchase_enabled === true, 'normal')" />
            </button>
            <div class="min-w-0">
              <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t('payment.admin.addonSales') }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('payment.admin.addonSalesHint') }}</div>
            </div>
          </div>
          <button type="button" class="btn btn-secondary" :disabled="addonProductsLoading" :title="t('common.refresh')" @click="loadAddonProducts">
            <Icon name="refresh" size="md" :class="addonProductsLoading ? 'animate-spin' : ''" />
          </button>
        </div>

        <DataTable :columns="addonColumns" :data="addonProducts" :loading="addonProductsLoading">
          <template #cell-sku="{ value }">
            <code class="text-xs text-gray-600 dark:text-gray-300">{{ value }}</code>
          </template>
          <template #cell-name="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>
          <template #cell-quota_usd="{ value }">
            <span class="tabular-nums">${{ formatNumber(value, 2) }}</span>
          </template>
          <template #cell-price="{ value }">
            <div class="flex flex-col">
              <span class="font-medium tabular-nums">{{ formatPaymentAmount(addonCnyAmount(value), 'CNY') }}</span>
              <span class="text-xs tabular-nums text-gray-500 dark:text-gray-400">{{ t('payment.admin.addonBalanceAmount') }} ${{ formatNumber(value, 2) }}</span>
            </div>
          </template>
          <template #cell-unit_price="{ row }">
            <div class="flex flex-col">
              <span class="tabular-nums text-gray-700 dark:text-gray-200">¥{{ formatNumber(addonCnyAmount(row.price) / row.quota_usd, 4) }}</span>
              <span class="text-xs tabular-nums text-gray-500 dark:text-gray-400">{{ t('payment.admin.addonBalanceAmount') }} ${{ formatNumber(row.price / row.quota_usd, 4) }}</span>
            </div>
          </template>
          <template #cell-original_price="{ value }">
            <span v-if="value != null" class="tabular-nums text-gray-500 line-through dark:text-gray-400">${{ formatNumber(value, 2) }}</span>
            <span v-else class="text-gray-400">-</span>
          </template>
          <template #cell-for_sale="{ value, row }">
            <div class="flex items-center gap-2">
              <button
                type="button"
                role="switch"
                :aria-label="t('payment.admin.forSale')"
                :aria-checked="value"
                :class="saleSwitchClass(value, 'small')"
                @click="toggleAddonForSale(row)"
              >
                <span :class="saleSwitchKnobClass(value, 'small')" />
              </button>
              <span class="text-xs" :class="value ? 'text-green-600 dark:text-green-400' : 'text-gray-500 dark:text-gray-400'">
                {{ value ? t('payment.admin.onSale') : t('payment.admin.offSale') }}
              </span>
            </div>
          </template>
          <template #cell-actions="{ row }">
            <button
              type="button"
              class="rounded-md p-2 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
              :title="t('common.edit')"
              :aria-label="t('common.edit')"
              @click="openAddonEdit(row)"
            >
              <Icon name="edit" size="sm" />
            </button>
          </template>
        </DataTable>
      </section>
    </div>

    <PlanEditDialog
      :show="showPlanDialog"
      :plan="editingPlan"
      :groups="groups"
      :payment-config="paymentConfig"
      @close="showPlanDialog = false"
      @saved="loadPlans"
    />
    <AddonProductEditDialog
      :show="showAddonDialog"
      :product="editingAddonProduct"
      :payment-config="paymentConfig"
      @close="showAddonDialog = false"
      @saved="loadAddonProducts"
    />
    <ConfirmDialog
      :show="showDeletePlanDialog"
      :title="t('payment.admin.deletePlan')"
      :message="t('payment.admin.deletePlanConfirm')"
      :confirm-text="t('common.delete')"
      danger
      @confirm="handleDeletePlan"
      @cancel="showDeletePlanDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI } from '@/api/admin/payment'
import type { AdminPaymentConfig, UpdateSubscriptionAddonProductRequest } from '@/api/admin/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import adminAPI from '@/api/admin'
import { formatPaymentAmount } from '@/components/payment/currency'
import type { SubscriptionAddonProduct, SubscriptionPlan } from '@/types/payment'
import type { AdminGroup } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import PlanEditDialog from './PlanEditDialog.vue'
import AddonProductEditDialog from './AddonProductEditDialog.vue'
import { platformTextClass } from '@/utils/platformColors'

type ProductCatalog = 'plans' | 'addons'
type SwitchSize = 'small' | 'normal'

const { t } = useI18n()
const appStore = useAppStore()
const activeCatalog = ref<ProductCatalog>('plans')

const groups = ref<AdminGroup[]>([])
const paymentConfig = ref<AdminPaymentConfig | null>(null)
const configSaving = ref(false)

async function loadGroups() {
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch { /* group labels are optional */ }
}

async function loadPaymentConfig() {
  try {
    const res = await adminPaymentAPI.getConfig()
    paymentConfig.value = res.data
  } catch { /* price previews and the global switch remain unavailable */ }
}

async function toggleAddonSales() {
  if (!paymentConfig.value || configSaving.value) return
  const next = !paymentConfig.value.addon_purchase_enabled
  configSaving.value = true
  try {
    await adminPaymentAPI.updateConfig({ addon_purchase_enabled: next })
    paymentConfig.value.addon_purchase_enabled = next
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    configSaving.value = false
  }
}

function getGroup(id: number): AdminGroup | undefined {
  return groups.value.find(group => group.id === id)
}

function isGroupMissing(id: number): boolean {
  return id > 0 && !groups.value.some(group => group.id === id)
}

function getPlanNameClass(groupId: number): string {
  const group = getGroup(groupId)
  return group ? platformTextClass(group.platform) : 'text-gray-900 dark:text-white'
}

const plansLoading = ref(false)
const plans = ref<SubscriptionPlan[]>([])
const showPlanDialog = ref(false)
const showDeletePlanDialog = ref(false)
const editingPlan = ref<SubscriptionPlan | null>(null)
const deletingPlanId = ref<number | null>(null)

const planColumns = computed((): Column[] => [
  { key: 'id', label: 'ID' },
  { key: 'name', label: t('payment.admin.planName') },
  { key: 'group_id', label: t('payment.admin.group') },
  { key: 'price', label: t('payment.admin.price') },
  { key: 'validity_days', label: t('payment.admin.validityDays') },
  { key: 'for_sale', label: t('payment.admin.forSale') },
  { key: 'sort_order', label: t('payment.admin.sortOrder') },
  { key: 'actions', label: t('common.actions') },
])

async function loadPlans() {
  plansLoading.value = true
  try {
    const res = await adminPaymentAPI.getPlans()
    plans.value = (res.data || []).map((plan: Omit<SubscriptionPlan, 'features'> & { features: string | string[] }) => ({
      ...plan,
      features: typeof plan.features === 'string'
        ? plan.features.split('\n').map(feature => feature.trim()).filter(Boolean)
        : (plan.features || []),
    }))
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    plansLoading.value = false
  }
}

function openPlanEdit(plan: SubscriptionPlan | null) {
  editingPlan.value = plan
  showPlanDialog.value = true
}

async function toggleForSale(plan: SubscriptionPlan) {
  try {
    await adminPaymentAPI.updatePlan(plan.id, { for_sale: !plan.for_sale })
    plan.for_sale = !plan.for_sale
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  }
}

function confirmDeletePlan(plan: SubscriptionPlan) {
  deletingPlanId.value = plan.id
  showDeletePlanDialog.value = true
}

async function handleDeletePlan() {
  if (!deletingPlanId.value) return
  try {
    await adminPaymentAPI.deletePlan(deletingPlanId.value)
    appStore.showSuccess(t('common.deleted'))
    showDeletePlanDialog.value = false
    await loadPlans()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  }
}

const addonProductsLoading = ref(false)
const addonProductsLoaded = ref(false)
const addonProducts = ref<SubscriptionAddonProduct[]>([])
const editingAddonProduct = ref<SubscriptionAddonProduct | null>(null)
const showAddonDialog = ref(false)

const addonColumns = computed((): Column[] => [
  { key: 'sku', label: t('payment.admin.addonSku') },
  { key: 'name', label: t('payment.admin.addonProductName') },
  { key: 'quota_usd', label: t('payment.admin.addonQuota') },
  { key: 'price', label: t('payment.admin.addonPrice') },
  { key: 'unit_price', label: t('payment.admin.addonUnitPrice') },
  { key: 'original_price', label: t('payment.admin.originalPrice') },
  { key: 'for_sale', label: t('payment.admin.forSale') },
  { key: 'sort_order', label: t('payment.admin.sortOrder') },
  { key: 'actions', label: t('common.actions') },
])

async function loadAddonProducts() {
  addonProductsLoading.value = true
  try {
    const res = await adminPaymentAPI.getAddonProducts()
    addonProducts.value = res.data || []
    addonProductsLoaded.value = true
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    addonProductsLoading.value = false
  }
}

function addonProductPayload(product: SubscriptionAddonProduct, forSale = product.for_sale): UpdateSubscriptionAddonProductRequest {
  return {
    name: product.name,
    quota_usd: product.quota_usd,
    price: product.price,
    original_price: product.original_price ?? null,
    for_sale: forSale,
    sort_order: product.sort_order,
  }
}

async function toggleAddonForSale(product: SubscriptionAddonProduct) {
  try {
    const res = await adminPaymentAPI.updateAddonProduct(product.id, addonProductPayload(product, !product.for_sale))
    Object.assign(product, res.data)
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  }
}

function openAddonEdit(product: SubscriptionAddonProduct) {
  editingAddonProduct.value = product
  showAddonDialog.value = true
}

function selectCatalog(catalog: ProductCatalog) {
  activeCatalog.value = catalog
  if (catalog === 'addons' && !addonProductsLoaded.value) {
    void loadAddonProducts()
  }
}

function catalogTabClass(active: boolean): string[] {
  return [
    'flex min-h-9 items-center gap-2 rounded px-3 py-1.5 text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500',
    active
      ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
      : 'text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white',
  ]
}

function saleSwitchClass(active: boolean, size: SwitchSize): string[] {
  return [
    'relative inline-flex shrink-0 rounded-full border-2 border-transparent transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50',
    size === 'small' ? 'h-5 w-9' : 'h-6 w-11',
    active ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600',
  ]
}

function saleSwitchKnobClass(active: boolean, size: SwitchSize): string[] {
  return [
    'pointer-events-none inline-block rounded-full bg-white shadow transition-transform',
    size === 'small' ? 'h-4 w-4' : 'h-5 w-5',
    active
      ? (size === 'small' ? 'translate-x-4' : 'translate-x-5')
      : 'translate-x-0',
  ]
}

function formatNumber(value: number, digits: number): string {
  return Number.isFinite(Number(value)) ? Number(value).toFixed(digits) : '-'
}

function addonCnyAmount(price: number): number {
  const value = Number(price)
  if (!Number.isFinite(value) || value <= 0) return 0
  const rate = Number(paymentConfig.value?.subscription_usd_to_cny_rate) || 0
  return Math.round(value * (rate > 0 ? rate : 1) * 100) / 100
}

onMounted(() => {
  void loadGroups()
  void loadPaymentConfig()
  void loadPlans()
})
</script>
