<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-5 sm:px-6">
      <header class="flex flex-wrap items-center justify-between gap-4 border-b border-gray-200 pb-5 dark:border-dark-700">
        <div class="flex min-w-0 items-center gap-3">
          <button
            data-testid="managed-recharge-back"
            type="button"
            class="btn btn-secondary h-9 w-9 shrink-0 p-0"
            title="返回充值订阅"
            aria-label="返回充值订阅"
            @click="goBack"
          >
            <Icon name="arrowLeft" size="sm" />
          </button>
          <div class="min-w-0">
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white sm:text-2xl">订阅 GPT Plus / Pro</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">确认套餐与账号后，使用站内余额完成订阅</p>
          </div>
        </div>
        <div class="border-l border-gray-200 pl-4 text-right dark:border-dark-700">
          <div class="text-xs text-gray-500 dark:text-dark-400">可用余额</div>
          <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">
            {{ formatAmount(catalog?.balance || 0) }}
          </div>
        </div>
      </header>

      <ol class="grid grid-cols-3 border-y border-gray-200 py-4 dark:border-dark-700" aria-label="订阅流程">
        <li
          v-for="(step, index) in purchaseSteps"
          :key="step.title"
          :data-testid="`managed-recharge-step-${index + 1}`"
          class="flex min-w-0 items-center justify-center gap-2 px-2 sm:gap-3"
        >
          <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gray-900 text-xs font-semibold text-white dark:bg-white dark:text-dark-900">
            {{ index + 1 }}
          </span>
          <span class="min-w-0">
            <span class="block text-xs font-medium text-gray-900 dark:text-white sm:text-sm">{{ step.title }}</span>
            <span class="mt-0.5 hidden text-xs text-gray-500 dark:text-dark-400 sm:block">{{ step.description }}</span>
          </span>
        </li>
      </ol>

      <div
        v-if="catalog?.mock_mode"
        class="flex flex-wrap items-center justify-between gap-3 border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700/70 dark:bg-amber-900/20 dark:text-amber-200"
      >
        <div class="flex min-w-0 items-start gap-3">
          <Icon name="infoCircle" size="md" class="mt-0.5 shrink-0" />
          <div>
            <div class="font-semibold">模拟测试环境</div>
            <div class="mt-0.5 text-xs leading-5">
              不会连接真实供货商，仅接受页面提供的假 Session。状态约每 {{ catalog.mock_step_seconds || 10 }} 秒推进一次。
            </div>
          </div>
        </div>
        <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="fillMockSession">
          填入模拟 Session
        </button>
      </div>

      <div v-if="loading" class="flex min-h-64 items-center justify-center">
        <Icon name="refresh" size="lg" class="animate-spin text-primary-500" />
      </div>

      <div v-else-if="!catalog?.enabled" class="border border-gray-200 bg-white p-8 text-center dark:border-dark-700 dark:bg-dark-800">
        <Icon name="cube" size="xl" class="mx-auto text-gray-400" />
        <h2 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">当前暂无可售套餐</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">库存补充后将在这里开放购买。</p>
      </div>

      <form
        v-else
        id="managed-recharge-form"
        class="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_320px]"
        @submit.prevent="submitOrder"
      >
        <div class="space-y-5">
          <section class="border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
            <div class="flex items-center gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
              <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gray-900 text-xs font-semibold text-white dark:bg-white dark:text-dark-900">1</span>
              <div>
                <h2 class="text-sm font-semibold text-gray-900 dark:text-white">选择订阅套餐</h2>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">请确认档位、库存和支付金额</p>
              </div>
            </div>

            <div v-if="requestedPlanUnavailable" class="flex gap-3 border-b border-amber-200 bg-amber-50 px-5 py-4 text-sm text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/25 dark:text-amber-200">
              <Icon name="infoCircle" size="md" class="mt-0.5 shrink-0" />
              <span>{{ requestedPlanLabel }} 暂未配置对应商品，请选择其他可用套餐。</span>
            </div>

            <div class="grid gap-3 p-5 sm:grid-cols-2">
              <button
                v-for="product in catalog.products"
                :key="product.id"
                :data-testid="`managed-recharge-product-${product.id}`"
                type="button"
                class="flex min-h-24 w-full items-center justify-between gap-4 rounded-lg border p-4 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-50"
                :class="selectedProductId === product.id
                  ? 'border-primary-500 bg-primary-50 ring-1 ring-primary-500 dark:bg-primary-900/15'
                  : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-600 dark:hover:bg-dark-700/60'"
                :aria-pressed="selectedProductId === product.id"
                :disabled="product.available_stock <= 0"
                @click="selectedProductId = product.id"
              >
                <span class="flex min-w-0 items-center gap-3">
                  <span
                    class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full border"
                    :class="selectedProductId === product.id
                      ? 'border-primary-500 bg-primary-500 text-white'
                      : 'border-gray-300 text-transparent dark:border-dark-500'"
                  >
                    <Icon name="check" size="sm" />
                  </span>
                  <span class="min-w-0">
                    <span class="block break-words text-sm font-semibold text-gray-900 dark:text-white">{{ product.name }}</span>
                    <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">
                      {{ product.available_stock > 0 ? `库存 ${product.available_stock}` : '暂时缺货' }}
                    </span>
                  </span>
                </span>
                <span class="shrink-0 text-base font-semibold text-gray-900 dark:text-white">{{ formatAmount(product.price) }}</span>
              </button>
            </div>
          </section>

          <section class="border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
            <div class="flex items-center gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
              <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gray-900 text-xs font-semibold text-white dark:bg-white dark:text-dark-900">2</span>
              <div>
                <h2 class="text-sm font-semibold text-gray-900 dark:text-white">提交目标账号 Session</h2>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">粘贴后会自动识别账号，确认无误再支付</p>
              </div>
            </div>

            <div class="space-y-4 p-5">
              <details
                v-if="!catalog?.mock_mode"
                data-testid="managed-recharge-session-guide"
                class="group rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-700/40"
              >
                <summary class="flex cursor-pointer list-none items-center gap-3 px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">
                  <Icon name="book" size="sm" class="shrink-0 text-primary-600 dark:text-primary-400" />
                  <span class="min-w-0 flex-1">第一次使用？查看详细获取教程</span>
                  <Icon name="chevronDown" size="sm" class="shrink-0 text-gray-400 transition-transform group-open:rotate-180" />
                </summary>
                <div class="border-t border-gray-200 px-4 py-4 dark:border-dark-600">
                  <ol class="space-y-4 text-sm text-gray-600 dark:text-dark-300">
                    <li v-for="(item, index) in sessionGuideSteps" :key="item.title" class="flex gap-3">
                      <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white text-xs font-semibold text-gray-700 shadow-sm dark:bg-dark-800 dark:text-dark-200">
                        {{ index + 1 }}
                      </span>
                      <span>
                        <span class="block font-medium text-gray-900 dark:text-white">{{ item.title }}</span>
                        <span class="mt-1 block text-xs leading-5">{{ item.description }}</span>
                      </span>
                    </li>
                  </ol>
                  <a
                    href="https://chatgpt.com/api/auth/session"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="btn btn-secondary btn-sm mt-4"
                  >
                    <Icon name="externalLink" size="sm" class="mr-2" />
                    打开 Session 页面
                  </a>
                  <p class="mt-3 text-xs leading-5 text-amber-700 dark:text-amber-300">
                    如果页面显示未登录或内容为空，请先登录需要订阅的 ChatGPT 账号，再刷新 Session 页面。
                  </p>
                </div>
              </details>

              <div v-else class="flex items-start gap-3 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:border-blue-900/60 dark:bg-blue-950/20 dark:text-blue-200">
                <Icon name="beaker" size="md" class="mt-0.5 shrink-0" />
                <span>点击页面上方“填入模拟 Session”，即可体验账号识别、余额扣款和订单进度。</span>
              </div>

              <div>
                <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
                  <label for="managed-recharge-session" class="input-label mb-0">Session JSON</label>
                  <span class="text-xs text-gray-500 dark:text-dark-400">粘贴时不会自动支付</span>
                </div>
                <textarea
                  id="managed-recharge-session"
                  v-model="sessionJson"
                  data-testid="managed-recharge-session-input"
                  class="input min-h-40 resize-y font-mono text-xs leading-5"
                  autocomplete="off"
                  spellcheck="false"
                  :placeholder="catalog?.mock_mode ? '点击上方按钮填入模拟 Session' : '粘贴从 { 开始到 } 结束的完整 Session JSON'"
                  @input="validateSession"
                />
                <div v-if="sessionEmail" class="mt-3 flex items-start gap-2 rounded-lg border border-green-200 bg-green-50 px-3 py-2.5 text-sm text-green-700 dark:border-green-900/60 dark:bg-green-950/20 dark:text-green-300">
                  <Icon name="checkCircle" size="sm" class="mt-0.5 shrink-0" />
                  <span class="min-w-0">
                    <span class="block text-xs">已识别目标账号</span>
                    <span class="mt-0.5 block break-all font-medium">{{ sessionEmail }}</span>
                  </span>
                </div>
                <p v-else-if="sessionError" class="mt-2 text-xs text-red-600 dark:text-red-400">{{ sessionError }}</p>
              </div>
            </div>
          </section>
        </div>

        <aside class="space-y-4 lg:sticky lg:top-5">
          <section class="border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
            <div class="flex items-center gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
              <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gray-900 text-xs font-semibold text-white dark:bg-white dark:text-dark-900">3</span>
              <div>
                <h2 class="text-sm font-semibold text-gray-900 dark:text-white">确认并支付</h2>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">支付方式：站内余额</p>
              </div>
            </div>

            <div class="space-y-4 p-5">
              <dl class="space-y-3 text-sm">
                <div class="flex items-start justify-between gap-3">
                  <dt class="text-gray-500 dark:text-dark-400">订阅套餐</dt>
                  <dd class="max-w-[11rem] text-right font-medium text-gray-900 dark:text-white">{{ selectedProduct?.name || '尚未选择' }}</dd>
                </div>
                <div class="flex items-center justify-between gap-3">
                  <dt class="text-gray-500 dark:text-dark-400">目标账号</dt>
                  <dd class="max-w-[11rem] truncate text-right text-gray-900 dark:text-white" :title="sessionEmail">{{ sessionEmail || '等待识别' }}</dd>
                </div>
                <div class="flex items-center justify-between gap-3 border-t border-gray-100 pt-3 dark:border-dark-700">
                  <dt class="text-gray-500 dark:text-dark-400">应付金额</dt>
                  <dd class="text-xl font-semibold text-gray-900 dark:text-white">{{ formatAmount(selectedProduct?.price || 0) }}</dd>
                </div>
                <div class="flex items-center justify-between gap-3 text-xs">
                  <dt class="text-gray-500 dark:text-dark-400">支付后余额</dt>
                  <dd :class="!selectedProduct || hasEnoughBalance ? 'text-gray-700 dark:text-dark-200' : 'text-red-600 dark:text-red-400'">
                    {{ formatAmount(balanceAfterPayment) }}
                  </dd>
                </div>
              </dl>

              <div v-if="selectedProduct && !hasEnoughBalance" class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-900/60 dark:bg-red-950/20 dark:text-red-300">
                余额不足，还差 {{ formatAmount(selectedProduct.price - (catalog?.balance || 0)) }}
              </div>

              <label class="flex items-start gap-3 text-xs leading-5 text-gray-600 dark:text-dark-300">
                <input v-model="agreed" type="checkbox" class="mt-0.5 h-4 w-4 shrink-0" />
                <span v-if="catalog?.mock_mode">我确认当前仅提交页面提供的模拟凭证，用于验证扣款、进度、补交和退款流程。</span>
                <span v-else>我确认账号归本人所有，并同意 Session 加密保存、临时传输给第三方履约服务；订单完成或退款后清除。</span>
              </label>

              <div v-if="submitError" class="border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/15 dark:text-red-300">
                {{ submitError }}
              </div>

              <button
                data-testid="managed-recharge-submit"
                type="submit"
                class="btn btn-primary w-full"
                :disabled="!canSubmit || submitting"
              >
                <Icon v-if="submitting" name="refresh" size="sm" class="mr-2 animate-spin" />
                <Icon v-else name="creditCard" size="sm" class="mr-2" />
                {{ submitting ? '正在提交' : `余额支付 ${formatAmount(selectedProduct?.price || 0)}` }}
              </button>

              <p class="text-center text-xs leading-5 text-gray-500 dark:text-dark-400">
                支付后可在下方查看处理进度；未完成的订单会按规则退款。
              </p>
            </div>

            <div class="flex gap-3 border-t border-gray-200 bg-gray-50 px-5 py-4 text-xs leading-5 text-gray-600 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-300">
              <Icon name="shield" size="md" class="mt-0.5 shrink-0 text-green-600 dark:text-green-400" />
              <span>请勿把 Session 发送到聊天群、工单截图或其他网站。提交后请保持目标账号登录，直到订单完成。</span>
            </div>
          </section>
        </aside>
      </form>

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
                  <div v-if="order.progress" class="mt-1 max-w-72 text-xs text-gray-500 dark:text-dark-400">{{ order.progress }}</div>
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
        <button
          v-if="catalog?.mock_mode"
          type="button"
          class="btn btn-secondary btn-sm"
          @click="replacementSession = mockSessionExample"
        >
          填入模拟 Session
        </button>
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
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  findManagedRechargeProduct,
  normalizeManagedRechargePlanKey,
} from '@/components/payment/managedRechargePlans'
import {
  createManagedRechargeOrder,
  getManagedRechargeCatalog,
  getManagedRechargeOrder,
  listManagedRechargeOrders,
  submitManagedRechargeReplacementSession,
  type ManagedRechargeCatalog,
  type ManagedRechargeOrder,
} from '@/api/managedRecharge'

const route = useRoute()
const router = useRouter()
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
const mockSessionExample = JSON.stringify({
  user: { email: 'mock@example.com' },
  accessToken: 'mock-access-token',
}, null, 2)
const purchaseSteps = [
  { title: '选择套餐', description: '确认档位与库存' },
  { title: '提交账号', description: '粘贴 Session JSON' },
  { title: '确认支付', description: '余额支付并跟踪进度' },
]
const sessionGuideSteps = [
  {
    title: '登录目标 ChatGPT 账号',
    description: '请在当前浏览器中登录真正需要开通 Plus 或 Pro 的账号，并确认右上角账号信息无误。',
  },
  {
    title: '打开 Session 页面',
    description: '点击下方按钮，新标签页会显示一段 JSON。如果显示未登录，请先完成 ChatGPT 登录后刷新。',
  },
  {
    title: '复制完整 JSON',
    description: '全选并复制从 { 开始到 } 结束的全部内容，不要只复制 accessToken，也不要上传截图。',
  },
  {
    title: '粘贴并核对账号',
    description: '返回本页粘贴 JSON，页面识别出的邮箱必须与需要订阅的账号一致。',
  },
  {
    title: '支付后保持登录',
    description: '订单处理期间不要退出该 ChatGPT 账号；如状态提示需要新 Session，请按相同步骤重新获取并补交。',
  },
]

const selectedProduct = computed(() => catalog.value?.products.find((item) => item.id === selectedProductId.value) || null)
const requestedPlan = computed(() => normalizeManagedRechargePlanKey(route.query.plan))
const requestedPlanLabel = computed(() => ({
  plus: 'Plus',
  'pro-5x': 'Pro（5 倍）',
  'pro-20x': 'Pro（20 倍）',
})[requestedPlan.value || 'plus'])
const requestedPlanUnavailable = computed(() => Boolean(
  requestedPlan.value &&
  catalog.value &&
  !findManagedRechargeProduct(catalog.value.products, requestedPlan.value),
))
const hasEnoughBalance = computed(() => Boolean(
  selectedProduct.value &&
  (catalog.value?.balance || 0) >= selectedProduct.value.price,
))
const balanceAfterPayment = computed(() => {
  if (!selectedProduct.value) return catalog.value?.balance || 0
  return Math.max(0, (catalog.value?.balance || 0) - selectedProduct.value.price)
})
const canSubmit = computed(() => Boolean(
  selectedProduct.value &&
  selectedProduct.value.available_stock > 0 &&
  sessionEmail.value &&
  agreed.value &&
  hasEnoughBalance.value,
))

function goBack(): void {
  router.push({ path: '/purchase', query: { tab: 'member' } })
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

function fillMockSession(): void {
  sessionJson.value = mockSessionExample
  validateSession()
}

async function loadCatalog(): Promise<void> {
  catalog.value = await getManagedRechargeCatalog()
  if (selectedProductId.value && catalog.value.products.some((item) => item.id === selectedProductId.value)) return
  if (requestedPlan.value) {
    selectedProductId.value = findManagedRechargeProduct(catalog.value.products, requestedPlan.value)?.id ?? null
    return
  }
  selectedProductId.value = catalog.value.products.find((item) => item.available_stock > 0)?.id || null
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
