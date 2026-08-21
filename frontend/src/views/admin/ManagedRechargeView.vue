<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-7xl flex-col gap-5 px-4 py-5 sm:px-6">
      <header class="flex flex-wrap items-start justify-between gap-4 border-b border-gray-200 pb-5 dark:border-dark-700">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">代充管理</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">管理套餐、CDK 库存和履约订单</p>
        </div>
        <button class="btn btn-secondary" :disabled="loading" title="刷新" @click="loadAll">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
        </button>
      </header>

      <div v-if="notice" class="border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-700 dark:border-green-900/50 dark:bg-green-900/15 dark:text-green-300">
        {{ notice }}
      </div>
      <div v-if="pageError" class="border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/15 dark:text-red-300">
        {{ pageError }}
      </div>

      <div class="inline-flex w-full max-w-md border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-800" role="tablist">
        <button
          v-for="item in tabs"
          :key="item.key"
          type="button"
          class="h-9 flex-1 px-3 text-sm font-medium transition-colors"
          :class="activeTab === item.key ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'"
          @click="activeTab = item.key"
        >
          {{ item.label }}
        </button>
      </div>

      <section v-if="activeTab === 'products'" class="space-y-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap gap-x-6 gap-y-2 text-sm text-gray-600 dark:text-dark-300">
            <span>套餐 {{ products.length }}</span>
            <span>在售 {{ activeProducts }}</span>
            <span>可用库存 {{ availableStock }}</span>
          </div>
          <button class="btn btn-primary" @click="openProductDialog()">
            <Icon name="plus" size="sm" class="mr-2" />
            新建套餐
          </button>
        </div>

        <div class="overflow-x-auto border border-gray-200 dark:border-dark-700">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left font-medium text-gray-500">套餐</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">标识</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">上游类型</th>
                <th class="px-4 py-3 text-right font-medium text-gray-500">售价</th>
                <th class="px-4 py-3 text-right font-medium text-gray-500">库存</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">状态</th>
                <th class="px-4 py-3 text-right font-medium text-gray-500">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="product in products" :key="product.id">
                <td class="px-4 py-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ product.name }}</div>
                  <div v-if="product.description" class="mt-1 max-w-md text-xs text-gray-500 dark:text-dark-400">{{ product.description }}</div>
                </td>
                <td class="whitespace-nowrap px-4 py-3 font-mono text-xs text-gray-600 dark:text-dark-300">{{ product.slug }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-dark-200">{{ planTypeLabel(product.plan_type) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right font-medium text-gray-900 dark:text-white">{{ formatAmount(product.price) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-dark-200">{{ product.available_stock }} / {{ product.total_stock }}</td>
                <td class="px-4 py-3">
                  <span class="badge" :class="product.active ? 'badge-success' : 'badge-gray'">{{ product.active ? '在售' : '停用' }}</span>
                </td>
                <td class="px-4 py-3 text-right">
                  <button class="btn btn-secondary btn-sm" title="编辑套餐" @click="openProductDialog(product)">
                    <Icon name="edit" size="sm" />
                  </button>
                </td>
              </tr>
              <tr v-if="!loading && products.length === 0">
                <td colspan="7" class="px-4 py-12 text-center text-gray-500 dark:text-dark-400">尚未创建套餐</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-else-if="activeTab === 'inventory'" class="space-y-4">
        <div class="flex flex-wrap items-center gap-3">
          <select v-model.number="cdkFilters.product_id" class="input w-full sm:w-52" @change="loadCDKs">
            <option :value="0">全部套餐</option>
            <option v-for="product in products" :key="product.id" :value="product.id">{{ product.name }}</option>
          </select>
          <select v-model="cdkFilters.status" class="input w-full sm:w-40" @change="loadCDKs">
            <option value="">全部状态</option>
            <option value="available">可用</option>
            <option value="reserved">已锁定</option>
            <option value="used">已使用</option>
            <option value="invalid">已隔离</option>
            <option value="disabled">已停用</option>
          </select>
          <div class="flex-1" />
          <button class="btn btn-primary" :disabled="products.length === 0" @click="openImportDialog">
            <Icon name="upload" size="sm" class="mr-2" />
            导入 CDK
          </button>
        </div>

        <div class="overflow-x-auto border border-gray-200 dark:border-dark-700">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left font-medium text-gray-500">CDK</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">套餐</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">状态</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">到期时间</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">关联订单</th>
                <th class="px-4 py-3 text-right font-medium text-gray-500">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="cdk in cdks" :key="cdk.id">
                <td class="whitespace-nowrap px-4 py-3 font-mono text-xs text-gray-700 dark:text-dark-200">{{ cdk.code_masked }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-dark-200">
                  <select
                    v-if="cdkCanMove(cdk.status)"
                    :value="cdk.product_id"
                    class="input h-8 w-40 py-1 text-xs"
                    :disabled="workingCDKs.has(cdk.id)"
                    @change="moveCDK(cdk, $event)"
                  >
                    <option v-for="product in products" :key="product.id" :value="product.id">{{ product.name }}</option>
                  </select>
                  <span v-else>{{ cdk.product_name }}</span>
                </td>
                <td class="px-4 py-3"><span class="badge" :class="cdkStatusClass(cdk.status)">{{ cdkStatusLabel(cdk.status) }}</span></td>
                <td class="whitespace-nowrap px-4 py-3 text-xs text-gray-500">{{ cdk.expires_at ? formatDate(cdk.expires_at) : '长期有效' }}</td>
                <td class="whitespace-nowrap px-4 py-3 font-mono text-xs text-gray-500">{{ cdk.reserved_order_id || '-' }}</td>
                <td class="px-4 py-3 text-right">
                  <button
                    v-if="cdk.status === 'available'"
                    class="btn btn-secondary btn-sm"
                    :disabled="workingCDKs.has(cdk.id)"
                    title="停用"
                    @click="setCDKStatus(cdk.id, 'disabled')"
                  >
                    <Icon name="ban" size="sm" />
                  </button>
                  <button
                    v-else-if="cdk.status === 'disabled' || cdk.status === 'invalid'"
                    class="btn btn-secondary btn-sm"
                    :disabled="workingCDKs.has(cdk.id)"
                    title="恢复为可用"
                    @click="setCDKStatus(cdk.id, 'available')"
                  >
                    <Icon name="check" size="sm" />
                  </button>
                  <span v-else class="text-gray-400">-</span>
                </td>
              </tr>
              <tr v-if="!loading && cdks.length === 0">
                <td colspan="6" class="px-4 py-12 text-center text-gray-500 dark:text-dark-400">没有符合条件的 CDK</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section v-else class="space-y-4">
        <div class="flex flex-wrap items-center gap-3">
          <select v-model="orderStatus" class="input w-full sm:w-48" @change="loadOrders">
            <option value="">全部订单状态</option>
            <option v-for="status in orderStatuses" :key="status" :value="status">{{ orderStatusLabel(status) }}</option>
          </select>
          <span class="text-sm text-gray-500 dark:text-dark-400">最近 {{ orders.length }} 条</span>
        </div>

        <div class="overflow-x-auto border border-gray-200 dark:border-dark-700">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left font-medium text-gray-500">订单</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">用户 / 账号</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">套餐</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">状态</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">上游状态</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">时间</th>
                <th class="px-4 py-3 text-right font-medium text-gray-500">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="order in orders" :key="order.id">
                <td class="px-4 py-3">
                  <div class="whitespace-nowrap font-mono text-xs text-gray-700 dark:text-dark-200">{{ order.order_no }}</div>
                  <div class="mt-1 text-xs text-gray-500">{{ formatAmount(order.price) }}</div>
                </td>
                <td class="px-4 py-3">
                  <div class="max-w-60 truncate text-gray-700 dark:text-dark-200">{{ order.user_email || order.username || `用户 ${order.user_id}` }}</div>
                  <div class="mt-1 max-w-60 truncate text-xs text-gray-500">{{ order.account_email }}</div>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-dark-200">{{ order.product_name }}</td>
                <td class="px-4 py-3">
                  <span class="badge" :class="orderStatusClass(order.status)">{{ orderStatusLabel(order.status) }}</span>
                  <div v-if="order.error_message" class="mt-1 max-w-64 text-xs text-red-600 dark:text-red-400">{{ order.error_message }}</div>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-xs text-gray-500">{{ order.upstream_status || '-' }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-xs text-gray-500">{{ formatDate(order.created_at) }}</td>
                <td class="px-4 py-3 text-right">
                  <div class="flex justify-end gap-2">
                    <button
                      v-if="orderNeedsSync(order.status)"
                      class="btn btn-secondary btn-sm"
                      :disabled="workingOrders.has(order.id)"
                      title="强制同步"
                      @click="syncOrder(order.id)"
                    >
                      <Icon name="refresh" size="sm" :class="workingOrders.has(order.id) ? 'animate-spin' : ''" />
                    </button>
                    <button
                      v-if="canRefund(order)"
                      class="btn btn-danger btn-sm"
                      :disabled="workingOrders.has(order.id)"
                      @click="openRefundDialog(order)"
                    >
                      退款
                    </button>
                  </div>
                </td>
              </tr>
              <tr v-if="!loading && orders.length === 0">
                <td colspan="7" class="px-4 py-12 text-center text-gray-500 dark:text-dark-400">没有符合条件的订单</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <BaseDialog :show="productDialogOpen" :title="editingProduct ? '编辑套餐' : '新建套餐'" width="normal" @close="closeProductDialog">
      <form id="managed-recharge-product-form" class="space-y-4" @submit.prevent="saveProduct">
        <div>
          <label class="input-label">套餐名称</label>
          <input v-model="productForm.name" class="input" maxlength="128" required />
        </div>
        <div>
          <label class="input-label">内部标识</label>
          <input v-model="productForm.slug" class="input font-mono" maxlength="64" pattern="[a-z0-9][a-z0-9_-]{1,63}" required placeholder="chatgpt_plus" />
        </div>
        <div>
          <label class="input-label">上游套餐类型</label>
          <select v-model="productForm.plan_type" class="input" required>
            <option value="plus">ChatGPT Plus</option>
            <option value="pro">ChatGPT Pro</option>
          </select>
        </div>
        <div>
          <label class="input-label">说明</label>
          <textarea v-model="productForm.description" class="input min-h-24 resize-y" maxlength="2000" />
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">售价</label>
            <input v-model.number="productForm.price" class="input" type="number" min="0.01" step="0.01" required />
          </div>
          <div>
            <label class="input-label">排序</label>
            <input v-model.number="productForm.sort_order" class="input" type="number" step="1" />
          </div>
        </div>
        <label class="flex items-center gap-3 text-sm text-gray-700 dark:text-dark-200">
          <input v-model="productForm.active" type="checkbox" class="h-4 w-4" />
          立即在用户端上架
        </label>
        <div v-if="dialogError" class="text-sm text-red-600 dark:text-red-400">{{ dialogError }}</div>
      </form>
      <template #footer>
        <button class="btn btn-secondary" @click="closeProductDialog">取消</button>
        <button form="managed-recharge-product-form" class="btn btn-primary" type="submit" :disabled="saving">{{ saving ? '保存中' : '保存' }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="importDialogOpen" title="批量导入 CDK" width="normal" @close="closeImportDialog">
      <form id="managed-recharge-import-form" class="space-y-4" @submit.prevent="importCDKs">
        <div>
          <label class="input-label">所属套餐</label>
          <select v-model.number="importForm.product_id" class="input" required>
            <option :value="0" disabled>请选择套餐</option>
            <option v-for="product in products" :key="product.id" :value="product.id">{{ product.name }}</option>
          </select>
        </div>
        <div>
          <label class="input-label">CDK 列表</label>
          <textarea v-model="importForm.codes" class="input min-h-56 resize-y font-mono text-xs" required placeholder="每行一枚 CDK，最多 500 枚" />
          <div class="mt-1 text-xs text-gray-500">已识别 {{ parsedImportCodes.length }} 枚，重复行会自动去除</div>
        </div>
        <div>
          <label class="input-label">统一到期时间（可选）</label>
          <input v-model="importForm.expires_at" class="input" type="datetime-local" />
        </div>
        <div v-if="dialogError" class="text-sm text-red-600 dark:text-red-400">{{ dialogError }}</div>
      </form>
      <template #footer>
        <button class="btn btn-secondary" @click="closeImportDialog">取消</button>
        <button form="managed-recharge-import-form" class="btn btn-primary" type="submit" :disabled="saving || parsedImportCodes.length === 0 || parsedImportCodes.length > 500">
          {{ saving ? '导入中' : '确认导入' }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog :show="refundOrder !== null" title="确认人工退款" width="normal" @close="refundOrder = null">
      <div class="space-y-3 text-sm text-gray-600 dark:text-dark-300">
        <p>订单 {{ refundOrder?.order_no }} 将退回 {{ formatAmount(refundOrder?.price || 0) }} 至用户余额。</p>
        <p class="border border-amber-200 bg-amber-50 px-3 py-2 text-amber-800 dark:border-amber-900/50 dark:bg-amber-900/15 dark:text-amber-300">退款后关联 CDK 会被隔离，不会自动回到可售库存。</p>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="refundOrder = null">取消</button>
        <button class="btn btn-danger" :disabled="saving" @click="confirmRefund">{{ saving ? '处理中' : '确认退款' }}</button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  adminCreateManagedRechargeProduct,
  adminImportManagedRechargeCDKs,
  adminListManagedRechargeCDKs,
  adminListManagedRechargeOrders,
  adminListManagedRechargeProducts,
  adminMoveManagedRechargeCDK,
  adminRefundManagedRechargeOrder,
  adminSetManagedRechargeCDKStatus,
  adminSyncManagedRechargeOrder,
  adminUpdateManagedRechargeProduct,
  type ManagedRechargeCDK,
  type ManagedRechargeOrder,
  type ManagedRechargeProduct,
  type ManagedRechargeProductInput,
} from '@/api/managedRecharge'

type TabKey = 'products' | 'inventory' | 'orders'

const tabs: Array<{ key: TabKey; label: string }> = [
  { key: 'products', label: '套餐' },
  { key: 'inventory', label: '库存' },
  { key: 'orders', label: '订单' },
]
const orderStatuses = ['validating', 'paid', 'submitting', 'queued', 'processing', 'verifying', 'action_required', 'manual_review', 'completed', 'failed', 'refunded']
const activeTab = ref<TabKey>('products')
const loading = ref(false)
const saving = ref(false)
const pageError = ref('')
const dialogError = ref('')
const notice = ref('')
const products = ref<ManagedRechargeProduct[]>([])
const cdks = ref<ManagedRechargeCDK[]>([])
const orders = ref<ManagedRechargeOrder[]>([])
const orderStatus = ref('')
const cdkFilters = reactive({ product_id: 0, status: '' })
const workingCDKs = ref(new Set<number>())
const workingOrders = ref(new Set<number>())
const productDialogOpen = ref(false)
const editingProduct = ref<ManagedRechargeProduct | null>(null)
const importDialogOpen = ref(false)
const refundOrder = ref<ManagedRechargeOrder | null>(null)
const productForm = reactive<ManagedRechargeProductInput>(emptyProductForm())
const importForm = reactive({ product_id: 0, codes: '', expires_at: '' })

const activeProducts = computed(() => products.value.filter((item) => item.active).length)
const availableStock = computed(() => products.value.reduce((total, item) => total + item.available_stock, 0))
const parsedImportCodes = computed(() => Array.from(new Set(importForm.codes.split(/\r?\n/).map((item) => item.trim()).filter(Boolean))))

function emptyProductForm(): ManagedRechargeProductInput {
  return { slug: '', plan_type: 'plus', name: '', description: '', price: 0, active: false, sort_order: 0 }
}

function errorMessage(error: unknown): string {
  if (error && typeof error === 'object' && 'message' in error) return String(error.message)
  return '请求失败，请稍后重试'
}

function formatAmount(value: number): string {
  return Number(value || 0).toFixed(2)
}

function formatDate(value: string): string {
  return new Date(value).toLocaleString()
}

async function loadProducts(): Promise<void> {
  products.value = await adminListManagedRechargeProducts()
}

async function loadCDKs(): Promise<void> {
  try {
    cdks.value = await adminListManagedRechargeCDKs({
      product_id: cdkFilters.product_id || undefined,
      status: cdkFilters.status || undefined,
      limit: 300,
    })
  } catch (error) {
    pageError.value = errorMessage(error)
  }
}

async function loadOrders(): Promise<void> {
  try {
    orders.value = await adminListManagedRechargeOrders(orderStatus.value, 200)
  } catch (error) {
    pageError.value = errorMessage(error)
  }
}

async function loadAll(): Promise<void> {
  loading.value = true
  pageError.value = ''
  try {
    await Promise.all([loadProducts(), loadCDKs(), loadOrders()])
  } catch (error) {
    pageError.value = errorMessage(error)
  } finally {
    loading.value = false
  }
}

function showNotice(message: string): void {
  notice.value = message
  window.setTimeout(() => {
    if (notice.value === message) notice.value = ''
  }, 4000)
}

function openProductDialog(product?: ManagedRechargeProduct): void {
  editingProduct.value = product || null
  Object.assign(productForm, product ? {
    slug: product.slug,
    plan_type: product.plan_type,
    name: product.name,
    description: product.description,
    price: product.price,
    active: product.active,
    sort_order: product.sort_order,
  } : emptyProductForm())
  dialogError.value = ''
  productDialogOpen.value = true
}

function closeProductDialog(): void {
  productDialogOpen.value = false
  editingProduct.value = null
  dialogError.value = ''
}

async function saveProduct(): Promise<void> {
  saving.value = true
  dialogError.value = ''
  try {
    if (editingProduct.value) {
      await adminUpdateManagedRechargeProduct(editingProduct.value.id, { ...productForm })
    } else {
      await adminCreateManagedRechargeProduct({ ...productForm })
    }
    closeProductDialog()
    await loadProducts()
    showNotice('套餐已保存')
  } catch (error) {
    dialogError.value = errorMessage(error)
  } finally {
    saving.value = false
  }
}

function openImportDialog(): void {
  importForm.product_id = cdkFilters.product_id || products.value[0]?.id || 0
  importForm.codes = ''
  importForm.expires_at = ''
  dialogError.value = ''
  importDialogOpen.value = true
}

function closeImportDialog(): void {
  importDialogOpen.value = false
  dialogError.value = ''
}

async function importCDKs(): Promise<void> {
  if (!importForm.product_id || parsedImportCodes.value.length === 0 || parsedImportCodes.value.length > 500) return
  saving.value = true
  dialogError.value = ''
  try {
    const result = await adminImportManagedRechargeCDKs({
      product_id: importForm.product_id,
      codes: parsedImportCodes.value,
      expires_at: importForm.expires_at ? new Date(importForm.expires_at).toISOString() : undefined,
    })
    closeImportDialog()
    await Promise.all([loadProducts(), loadCDKs()])
    showNotice(`已导入 ${result.imported} 枚，跳过 ${result.skipped} 枚`)
  } catch (error) {
    dialogError.value = errorMessage(error)
  } finally {
    saving.value = false
  }
}

async function setCDKStatus(id: number, status: string): Promise<void> {
  workingCDKs.value.add(id)
  workingCDKs.value = new Set(workingCDKs.value)
  pageError.value = ''
  try {
    await adminSetManagedRechargeCDKStatus(id, status)
    await Promise.all([loadProducts(), loadCDKs()])
  } catch (error) {
    pageError.value = errorMessage(error)
  } finally {
    workingCDKs.value.delete(id)
    workingCDKs.value = new Set(workingCDKs.value)
  }
}

async function moveCDK(cdk: ManagedRechargeCDK, event: Event): Promise<void> {
  const select = event.target as HTMLSelectElement
  const productID = Number(select.value)
  if (!productID || productID === cdk.product_id) return
  const previousProductID = cdk.product_id
  workingCDKs.value.add(cdk.id)
  workingCDKs.value = new Set(workingCDKs.value)
  pageError.value = ''
  try {
    await adminMoveManagedRechargeCDK(cdk.id, productID)
    await Promise.all([loadProducts(), loadCDKs()])
    showNotice('CDK 已移动到新套餐')
  } catch (error) {
    select.value = String(previousProductID)
    pageError.value = errorMessage(error)
  } finally {
    workingCDKs.value.delete(cdk.id)
    workingCDKs.value = new Set(workingCDKs.value)
  }
}

async function syncOrder(id: number): Promise<void> {
  workingOrders.value.add(id)
  workingOrders.value = new Set(workingOrders.value)
  pageError.value = ''
  try {
    const updated = await adminSyncManagedRechargeOrder(id)
    const index = orders.value.findIndex((item) => item.id === id)
    if (index >= 0) orders.value[index] = updated
  } catch (error) {
    pageError.value = errorMessage(error)
  } finally {
    workingOrders.value.delete(id)
    workingOrders.value = new Set(workingOrders.value)
  }
}

function openRefundDialog(order: ManagedRechargeOrder): void {
  refundOrder.value = order
}

async function confirmRefund(): Promise<void> {
  if (!refundOrder.value) return
  const id = refundOrder.value.id
  saving.value = true
  pageError.value = ''
  try {
    const updated = await adminRefundManagedRechargeOrder(id)
    const index = orders.value.findIndex((item) => item.id === id)
    if (index >= 0) orders.value[index] = updated
    refundOrder.value = null
    await loadProducts()
    showNotice('订单已退款，关联 CDK 已隔离')
  } catch (error) {
    pageError.value = errorMessage(error)
  } finally {
    saving.value = false
  }
}

function cdkStatusLabel(status: string): string {
  return ({ available: '可用', reserved: '已锁定', used: '已使用', invalid: '已隔离', disabled: '已停用' } as Record<string, string>)[status] || status
}

function cdkCanMove(status: string): boolean {
  return ['available', 'disabled', 'invalid'].includes(status)
}

function planTypeLabel(planType: string): string {
  return planType === 'pro' ? 'ChatGPT Pro' : 'ChatGPT Plus'
}

function cdkStatusClass(status: string): string {
  if (status === 'available') return 'badge-success'
  if (status === 'reserved') return 'badge-warning'
  if (status === 'used') return 'badge-info'
  return 'badge-gray'
}

function orderStatusLabel(status: string): string {
  return ({
    validating: '验证库存', paid: '已支付', submitting: '正在提交', queued: '排队中', processing: '处理中',
    verifying: '确认订阅', action_required: '待补 Session', manual_review: '人工核对', completed: '成功',
    failed: '失败', refunded: '已退款',
  } as Record<string, string>)[status] || status
}

function orderStatusClass(status: string): string {
  if (status === 'completed') return 'badge-success'
  if (status === 'failed' || status === 'refunded') return 'badge-danger'
  if (status === 'action_required' || status === 'manual_review') return 'badge-warning'
  return 'badge-info'
}

function orderNeedsSync(status: string): boolean {
  return ['validating', 'paid', 'submitting', 'queued', 'processing', 'verifying', 'action_required', 'manual_review'].includes(status)
}

function canRefund(order: ManagedRechargeOrder): boolean {
  if (order.status !== 'manual_review' || order.error_code !== 'UPSTREAM_TASK_NOT_FOUND' || !order.paid_at) return false
  return Date.now() - new Date(order.paid_at).getTime() >= 10 * 60 * 1000
}

onMounted(loadAll)
</script>
