<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-5 sm:px-6">
      <header class="flex flex-wrap items-start justify-between gap-4 border-b border-gray-200 pb-5 dark:border-dark-700">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">会员代充</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">Plus / Pro 库存充值</p>
        </div>
        <div class="text-right">
          <div class="text-xs text-gray-500 dark:text-dark-400">可用余额</div>
          <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">
            {{ formatAmount(catalog?.balance || 0) }}
          </div>
        </div>
      </header>

      <div v-if="loading" class="flex min-h-64 items-center justify-center">
        <Icon name="refresh" size="lg" class="animate-spin text-primary-500" />
      </div>

      <div v-else-if="!catalog?.enabled" class="border border-gray-200 bg-white p-8 text-center dark:border-dark-700 dark:bg-dark-800">
        <Icon name="cube" size="xl" class="mx-auto text-gray-400" />
        <h2 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">当前暂无可售套餐</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">库存补充后将在这里开放购买。</p>
      </div>

      <div v-else class="grid gap-6 lg:grid-cols-[300px_minmax(0,1fr)]">
        <section class="border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-700">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">选择套餐</h2>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <button
              v-for="product in catalog.products"
              :key="product.id"
              type="button"
              class="flex w-full items-center justify-between gap-3 px-4 py-4 text-left transition-colors"
              :class="selectedProductId === product.id ? 'bg-primary-50 dark:bg-primary-900/15' : 'hover:bg-gray-50 dark:hover:bg-dark-700/60'"
              :disabled="product.available_stock <= 0"
              @click="selectedProductId = product.id"
            >
              <span class="min-w-0">
                <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">{{ product.name }}</span>
                <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">
                  {{ product.available_stock > 0 ? `库存 ${product.available_stock}` : '暂时缺货' }}
                </span>
              </span>
              <span class="shrink-0 text-base font-semibold text-gray-900 dark:text-white">{{ formatAmount(product.price) }}</span>
            </button>
          </div>
        </section>

        <section class="border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div class="border-b border-gray-200 px-5 py-3 dark:border-dark-700">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">提交账号</h2>
          </div>
          <form class="space-y-5 p-5" @submit.prevent="submitOrder">
            <div v-if="selectedProduct" class="grid gap-3 border-b border-gray-100 pb-4 text-sm dark:border-dark-700 sm:grid-cols-3">
              <div>
                <div class="text-xs text-gray-500 dark:text-dark-400">套餐</div>
                <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ selectedProduct.name }}</div>
              </div>
              <div>
                <div class="text-xs text-gray-500 dark:text-dark-400">支付金额</div>
                <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ formatAmount(selectedProduct.price) }}</div>
              </div>
              <div>
                <div class="text-xs text-gray-500 dark:text-dark-400">支付方式</div>
                <div class="mt-1 font-medium text-gray-900 dark:text-white">站内余额</div>
              </div>
            </div>

            <div>
              <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
                <label for="managed-recharge-session" class="input-label mb-0">Session JSON</label>
                <a
                  href="https://chatgpt.com/api/auth/session"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
                >
                  获取 Session
                </a>
              </div>
              <textarea
                id="managed-recharge-session"
                v-model="sessionJson"
                class="input min-h-36 resize-y font-mono text-xs"
                autocomplete="off"
                spellcheck="false"
                placeholder='粘贴包含 user 和 accessToken 的完整 JSON'
                @input="validateSession"
              />
              <div v-if="sessionEmail" class="mt-2 flex items-center gap-2 text-xs text-green-600 dark:text-green-400">
                <Icon name="checkCircle" size="sm" />
                <span class="break-all">{{ sessionEmail }}</span>
              </div>
              <p v-else-if="sessionError" class="mt-2 text-xs text-red-600 dark:text-red-400">{{ sessionError }}</p>
            </div>

            <label class="flex items-start gap-3 text-sm text-gray-600 dark:text-dark-300">
              <input v-model="agreed" type="checkbox" class="mt-0.5 h-4 w-4" />
              <span>我确认该账号归本人所有，并同意 Session 在本次订单中加密保存、临时传输给第三方履约服务；完成或退款后清除。</span>
            </label>

            <div v-if="submitError" class="border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/15 dark:text-red-300">
              {{ submitError }}
            </div>

            <button
              type="submit"
              class="btn btn-primary w-full sm:w-auto"
              :disabled="!canSubmit || submitting"
            >
              <Icon v-if="submitting" name="refresh" size="sm" class="mr-2 animate-spin" />
              <Icon v-else name="creditCard" size="sm" class="mr-2" />
              {{ submitting ? '正在提交' : `余额支付 ${formatAmount(selectedProduct?.price || 0)}` }}
            </button>
          </form>
        </section>
      </div>

      <section class="border-t border-gray-200 pt-5 dark:border-dark-700">
        <div class="mb-3 flex items-center justify-between gap-3">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">充值记录</h2>
          <button class="btn btn-secondary btn-sm" :disabled="ordersLoading" title="刷新" @click="loadOrders">
            <Icon name="refresh" size="sm" :class="ordersLoading ? 'animate-spin' : ''" />
          </button>
        </div>

        <div v-if="orders.length === 0" class="py-10 text-center text-sm text-gray-500 dark:text-dark-400">暂无充值记录</div>
        <div v-else class="overflow-x-auto border border-gray-200 dark:border-dark-700">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left font-medium text-gray-500">订单</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">账号</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">套餐</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">进度</th>
                <th class="px-4 py-3 text-left font-medium text-gray-500">时间</th>
                <th class="px-4 py-3 text-right font-medium text-gray-500">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-800">
              <tr v-for="order in orders" :key="order.id">
                <td class="whitespace-nowrap px-4 py-3 font-mono text-xs text-gray-700 dark:text-dark-200">{{ order.order_no }}</td>
                <td class="max-w-56 truncate px-4 py-3 text-gray-700 dark:text-dark-200">{{ order.account_email }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-dark-200">{{ order.product_name }}</td>
                <td class="px-4 py-3">
                  <span class="badge" :class="statusClass(order.status)">{{ statusLabel(order.status) }}</span>
                  <div v-if="order.queue_position" class="mt-1 text-xs text-gray-500">队列 {{ order.queue_position }}/{{ order.queue_total || order.queue_position }}</div>
                  <div v-if="order.error_message" class="mt-1 max-w-72 text-xs text-red-600 dark:text-red-400">{{ order.error_message }}</div>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-xs text-gray-500">{{ formatDate(order.created_at) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right">
                  <button
                    v-if="order.status === 'action_required'"
                    class="btn btn-secondary btn-sm"
                    @click="openReplacement(order)"
                  >
                    补交 Session
                  </button>
                  <button
                    v-else-if="isActiveStatus(order.status)"
                    class="btn btn-secondary btn-sm"
                    title="同步进度"
                    @click="refreshOrder(order.id)"
                  >
                    <Icon name="refresh" size="sm" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <BaseDialog :show="replacementOrder !== null" title="补交 Session" width="normal" @close="closeReplacement">
      <div class="space-y-4">
        <div class="text-sm text-gray-600 dark:text-dark-300">账号：{{ replacementOrder?.account_email }}</div>
        <textarea v-model="replacementSession" class="input min-h-40 font-mono text-xs" autocomplete="off" spellcheck="false" />
        <div v-if="replacementError" class="text-sm text-red-600 dark:text-red-400">{{ replacementError }}</div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="closeReplacement">取消</button>
        <button class="btn btn-primary" :disabled="!replacementSession.trim() || replacementSubmitting" @click="submitReplacement">
          {{ replacementSubmitting ? '正在提交' : '提交新 Session' }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  createManagedRechargeOrder,
  getManagedRechargeCatalog,
  getManagedRechargeOrder,
  listManagedRechargeOrders,
  submitManagedRechargeReplacementSession,
  type ManagedRechargeCatalog,
  type ManagedRechargeOrder,
} from '@/api/managedRecharge'

const catalog = ref<ManagedRechargeCatalog | null>(null)
const selectedProductId = ref<number | null>(null)
const sessionJson = ref('')
const sessionEmail = ref('')
const sessionError = ref('')
const agreed = ref(false)
const loading = ref(true)
const submitting = ref(false)
const submitError = ref('')
const orders = ref<ManagedRechargeOrder[]>([])
const ordersLoading = ref(false)
const replacementOrder = ref<ManagedRechargeOrder | null>(null)
const replacementSession = ref('')
const replacementError = ref('')
const replacementSubmitting = ref(false)
let pollTimer: number | null = null

const selectedProduct = computed(() => catalog.value?.products.find((item) => item.id === selectedProductId.value) || null)
const canSubmit = computed(() => Boolean(
  selectedProduct.value &&
  selectedProduct.value.available_stock > 0 &&
  sessionEmail.value &&
  agreed.value &&
  (catalog.value?.balance || 0) >= selectedProduct.value.price,
))

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

function validateSession(): void {
  sessionEmail.value = ''
  sessionError.value = ''
  const value = sessionJson.value.trim()
  if (!value) return
  const otpMatch = value.match(/^(.+?)----https?:\/\/.+$/)
  if (otpMatch?.[1]?.includes('@')) {
    sessionEmail.value = otpMatch[1].trim()
    return
  }
  try {
    const parsed = JSON.parse(value)
    const email = String(parsed?.user?.email || parsed?.user?.name || '').trim()
    if (!parsed?.accessToken || !email) throw new Error('invalid')
    sessionEmail.value = email
  } catch {
    sessionError.value = 'Session JSON 格式不完整'
  }
}

async function loadCatalog(): Promise<void> {
  catalog.value = await getManagedRechargeCatalog()
  if (!selectedProductId.value || !catalog.value.products.some((item) => item.id === selectedProductId.value)) {
    selectedProductId.value = catalog.value.products.find((item) => item.available_stock > 0)?.id || null
  }
}

async function loadOrders(): Promise<void> {
  ordersLoading.value = true
  try {
    orders.value = await listManagedRechargeOrders()
  } catch (error) {
    submitError.value = errorMessage(error)
  } finally {
    ordersLoading.value = false
  }
}

function newIdempotencyKey(): string {
  if (typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  return `mr-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

async function submitOrder(): Promise<void> {
  if (!canSubmit.value || !selectedProduct.value || submitting.value) return
  submitting.value = true
  submitError.value = ''
  try {
    const order = await createManagedRechargeOrder(selectedProduct.value.id, sessionJson.value, newIdempotencyKey())
    orders.value = [order, ...orders.value.filter((item) => item.id !== order.id)]
    sessionJson.value = ''
    sessionEmail.value = ''
    agreed.value = false
    await loadCatalog()
  } catch (error) {
    submitError.value = errorMessage(error)
  } finally {
    submitting.value = false
  }
}

function isActiveStatus(status: string): boolean {
  return ['validating', 'paid', 'submitting', 'queued', 'processing', 'verifying', 'manual_review'].includes(status)
}

async function refreshOrder(id: number): Promise<void> {
  try {
    const updated = await getManagedRechargeOrder(id)
    const index = orders.value.findIndex((item) => item.id === id)
    if (index >= 0) orders.value[index] = updated
    await loadCatalog()
  } catch {
    // Keep the last known order state when synchronization is temporarily unavailable.
  }
}

async function pollActiveOrders(): Promise<void> {
  const active = orders.value.filter((order) => isActiveStatus(order.status) || order.status === 'action_required')
  for (const order of active.slice(0, 5)) await refreshOrder(order.id)
}

function statusLabel(status: string): string {
  return ({
    validating: '验证库存', paid: '已支付', submitting: '正在提交', queued: '排队中', processing: '处理中',
    verifying: '确认订阅', action_required: '需要新 Session', manual_review: '人工核对', completed: '充值成功',
    failed: '未完成', refunded: '已退款',
  } as Record<string, string>)[status] || status
}

function statusClass(status: string): string {
  if (status === 'completed') return 'badge-success'
  if (status === 'refunded' || status === 'failed') return 'badge-danger'
  if (status === 'action_required' || status === 'manual_review') return 'badge-warning'
  return 'badge-info'
}

function openReplacement(order: ManagedRechargeOrder): void {
  replacementOrder.value = order
  replacementSession.value = ''
  replacementError.value = ''
}

function closeReplacement(): void {
  replacementOrder.value = null
  replacementSession.value = ''
  replacementError.value = ''
}

async function submitReplacement(): Promise<void> {
  if (!replacementOrder.value || !replacementSession.value.trim() || replacementSubmitting.value) return
  replacementSubmitting.value = true
  replacementError.value = ''
  try {
    const updated = await submitManagedRechargeReplacementSession(replacementOrder.value.id, replacementSession.value)
    const index = orders.value.findIndex((item) => item.id === updated.id)
    if (index >= 0) orders.value[index] = updated
    closeReplacement()
  } catch (error) {
    replacementError.value = errorMessage(error)
  } finally {
    replacementSubmitting.value = false
  }
}

onMounted(async () => {
  try {
    await Promise.all([loadCatalog(), loadOrders()])
  } catch (error) {
    submitError.value = errorMessage(error)
  } finally {
    loading.value = false
  }
  pollTimer = window.setInterval(pollActiveOrders, 10000)
})

onUnmounted(() => {
  if (pollTimer !== null) window.clearInterval(pollTimer)
})
</script>
