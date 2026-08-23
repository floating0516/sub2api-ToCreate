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
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
              {{ isExternalMode ? '支付宝付款后领取 CDK，并前往兑换页完成订阅' : '确认套餐与账号后，使用支付宝完成订阅' }}
            </p>
          </div>
        </div>
        <div class="border-l border-gray-200 pl-4 text-right dark:border-dark-700">
          <div class="text-xs text-gray-500 dark:text-dark-400">支付方式</div>
          <div class="mt-1 text-base font-semibold text-[#1677ff]">支付宝</div>
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
              <template v-if="isExternalMode">仅验证支付宝下单与模拟 CDK 发放；模拟 CDK 无法在真实兑换页使用。</template>
              <template v-else>不会连接真实供货商，仅接受页面提供的假 Session。状态约每 {{ catalog.mock_step_seconds || 10 }} 秒推进一次。</template>
            </div>
          </div>
        </div>
        <button v-if="!isExternalMode" type="button" class="btn btn-secondary btn-sm shrink-0" @click="fillMockSession">
          填入模拟 Session
        </button>
      </div>

      <div v-if="loading" class="flex min-h-64 items-center justify-center">
        <Icon name="refresh" size="lg" class="animate-spin text-primary-500" />
      </div>

      <section v-else-if="paymentState" class="mx-auto w-full max-w-md">
        <div class="mb-4 border-b border-gray-200 pb-4 text-center dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">支付宝付款</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">付款确认后才会发放 CDK 或提交充值任务</p>
        </div>
        <PaymentStatusPanel
          :order-id="paymentState.order_id"
          :amount="paymentState.amount"
          :pay-amount="paymentState.pay_amount"
          :qr-code="paymentState.qr_code || ''"
          :expires-at="paymentState.expires_at"
          payment-type="alipay"
          :pay-url="paymentState.pay_url"
          order-type="managed_recharge"
          :currency="paymentState.currency || 'CNY'"
          :out-trade-no="paymentState.out_trade_no"
          :mobile-alipay-deep-link="paymentState.alipay_mobile_precreate_deep_link === true"
          @success="handlePaymentSuccess"
          @done="handlePaymentDone"
        />
      </section>

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
                <span class="shrink-0 text-base font-semibold text-gray-900 dark:text-white">¥{{ formatAmount(product.price) }}</span>
              </button>
            </div>
          </section>

          <section v-if="isExternalMode" class="border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
            <div class="flex items-center gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
              <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gray-900 text-xs font-semibold text-white dark:bg-white dark:text-dark-900">2</span>
              <div>
                <h2 class="text-sm font-semibold text-gray-900 dark:text-white">领取专属 CDK</h2>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">付款成功后立即发放，不需要在本站提交 Session</p>
              </div>
            </div>
            <div class="grid gap-4 p-5 sm:grid-cols-3">
              <div class="border-l-2 border-gray-900 pl-3 dark:border-white">
                <div class="text-xs text-gray-500 dark:text-dark-400">付款</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">支付宝完成支付</div>
              </div>
              <div class="border-l-2 border-gray-300 pl-3 dark:border-dark-500">
                <div class="text-xs text-gray-500 dark:text-dark-400">领取</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">获得订单专属 CDK</div>
              </div>
              <div class="border-l-2 border-gray-300 pl-3 dark:border-dark-500">
                <div class="text-xs text-gray-500 dark:text-dark-400">兑换</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">前往兑换页提交账号</div>
              </div>
            </div>
          </section>

          <section v-else class="border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
            <div class="flex items-center gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
              <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gray-900 text-xs font-semibold text-white dark:bg-white dark:text-dark-900">2</span>
              <div>
                <h2 class="text-sm font-semibold text-gray-900 dark:text-white">提交 Session</h2>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">自动验证目标账号与当前订阅</p>
              </div>
            </div>

            <div class="space-y-5 p-5">
              <div
                data-testid="managed-recharge-session-guide"
                class="border-l-2 border-primary-500 pl-4 text-sm text-gray-600 dark:text-dark-300"
              >
                <p class="font-medium text-gray-900 dark:text-white">如何获取 Session？</p>
                <p class="mt-1 leading-6">
                  登录 ChatGPT 后，在浏览器打开
                  <a
                    href="https://chatgpt.com/api/auth/session"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="font-medium text-primary-600 underline decoration-primary-300 underline-offset-2 dark:text-primary-400"
                  >https://chatgpt.com/api/auth/session</a>，将页面返回的完整 JSON 全部复制并粘贴到下方输入框。
                </p>
              </div>

              <div>
                <label for="managed-recharge-session" class="input-label">Session JSON</label>
                <textarea
                  id="managed-recharge-session"
                  v-model="sessionJson"
                  data-testid="managed-recharge-session-input"
                  class="input min-h-36 resize-y font-mono text-xs leading-5"
                  autocomplete="off"
                  spellcheck="false"
                  :placeholder="catalog?.mock_mode ? '点击上方按钮填入模拟 Session' : '粘贴完整的 Session JSON'"
                  @input="scheduleSessionValidation"
                />
                <div v-if="sessionValidationState === 'checking'" class="mt-3 flex items-center gap-2 text-sm text-gray-500 dark:text-dark-400">
                  <Icon name="refresh" size="sm" class="animate-spin" />
                  <span>正在验证 Session</span>
                </div>
                <div
                  v-else-if="sessionValidationState === 'valid'"
                  data-testid="managed-recharge-session-result"
                  class="mt-3 border border-green-200 bg-green-50 px-4 py-3 dark:border-green-900/60 dark:bg-green-950/20"
                >
                  <div class="flex items-center gap-2 text-sm font-medium text-green-700 dark:text-green-300">
                    <Icon name="checkCircle" size="sm" />
                    <span>Session 格式有效</span>
                  </div>
                  <dl class="mt-3 grid gap-2 border-t border-green-200/80 pt-3 text-sm dark:border-green-900/60">
                    <div class="flex items-start justify-between gap-4">
                      <dt class="text-gray-500 dark:text-dark-400">账号邮箱</dt>
                      <dd class="min-w-0 break-all text-right font-medium text-gray-900 dark:text-white">{{ sessionEmail }}</dd>
                    </div>
                    <div class="flex items-center justify-between gap-4">
                      <dt class="text-gray-500 dark:text-dark-400">当前订阅</dt>
                      <dd class="font-medium text-gray-900 dark:text-white">{{ sessionMembershipLabel }}</dd>
                    </div>
                  </dl>
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
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">支付方式：支付宝</p>
              </div>
            </div>

            <div class="space-y-4 p-5">
              <dl class="space-y-3 text-sm">
                <div class="flex items-start justify-between gap-3">
                  <dt class="text-gray-500 dark:text-dark-400">订阅套餐</dt>
                  <dd class="max-w-[11rem] text-right font-medium text-gray-900 dark:text-white">{{ selectedProduct?.name || '尚未选择' }}</dd>
                </div>
                <div v-if="!isExternalMode" class="flex items-center justify-between gap-3">
                  <dt class="text-gray-500 dark:text-dark-400">目标账号</dt>
                  <dd class="max-w-[11rem] truncate text-right text-gray-900 dark:text-white" :title="sessionEmail">{{ sessionEmail || '等待识别' }}</dd>
                </div>
                <div class="flex items-center justify-between gap-3 border-t border-gray-100 pt-3 dark:border-dark-700">
                  <dt class="text-gray-500 dark:text-dark-400">应付金额</dt>
                  <dd class="text-xl font-semibold text-gray-900 dark:text-white">¥{{ formatAmount(selectedProduct?.price || 0) }}</dd>
                </div>
              </dl>

              <label class="flex items-start gap-3 text-xs leading-5 text-gray-600 dark:text-dark-300">
                <input v-model="agreed" data-testid="managed-recharge-agreement" type="checkbox" class="mt-0.5 h-4 w-4 shrink-0" />
                <span v-if="catalog?.mock_mode && !isExternalMode">我确认当前仅提交页面提供的模拟凭证，用于验证扣款、进度、补交和退款流程。</span>
                <span v-else-if="isExternalMode">我确认付款后将获得一枚订单专属 CDK，并前往第三方兑换服务完成账号充值。CDK 发放即视为交付，不支持自动退款；未完成时可使用同一 CDK 重试。</span>
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
                {{ submitting ? '正在创建支付宝订单' : `支付宝支付 ¥${formatAmount(selectedProduct?.price || 0)}` }}
              </button>

              <p class="text-center text-xs leading-5 text-gray-500 dark:text-dark-400">
                {{ isExternalMode ? 'CDK 发放后请妥善保存；兑换未完成时可使用同一 CDK 重新提交。' : '支付后可在下方查看处理进度；未完成的订单会按规则退款。' }}
              </p>
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
                <td class="max-w-56 truncate px-4 py-3 text-gray-700 dark:text-dark-200">
                  {{ order.account_email || (order.fulfillment_mode === 'external' ? '在兑换页填写' : '-') }}
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-dark-200">{{ order.product_name }}</td>
                <td class="px-4 py-3">
                  <span class="badge" :class="statusClass(order.status)">{{ statusLabel(order.status, order.fulfillment_mode) }}</span>
                  <div v-if="order.queue_position" class="mt-1 text-xs text-gray-500">队列 {{ order.queue_position }}/{{ order.queue_total || order.queue_position }}</div>
                  <div v-if="order.progress" class="mt-1 max-w-72 text-xs text-gray-500 dark:text-dark-400">{{ order.progress }}</div>
                  <div v-if="order.error_message" class="mt-1 max-w-72 text-xs text-red-600 dark:text-red-400">{{ order.error_message }}</div>
                  <div v-if="order.fulfillment_mode === 'external' && order.last_synced_at" class="mt-1 text-xs text-gray-400 dark:text-dark-500">
                    状态由兑换服务同步 · {{ formatSyncTime(order.last_synced_at) }}
                  </div>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-xs text-gray-500">{{ formatDate(order.created_at) }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right">
                  <a
                    v-if="order.fulfillment_mode === 'external' && order.redemption_url"
                    class="btn btn-primary btn-sm inline-flex"
                    :href="order.redemption_url"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <Icon name="externalLink" size="sm" class="mr-1.5" />
                    前往兑换
                  </a>
                  <button
                    v-else-if="order.fulfillment_mode === 'external' && order.status !== 'refunded'"
                    :data-testid="`managed-recharge-reveal-${order.id}`"
                    class="btn btn-secondary btn-sm"
                    @click="revealExternalOrder(order.id)"
                  >
                    查看 CDK
                  </button>
                  <button
                    v-else-if="order.status === 'action_required'"
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

    <BaseDialog :show="issuedOrder !== null" title="CDK 已发放" width="normal" @close="closeIssuedOrder">
      <div v-if="issuedOrder" class="space-y-5" data-testid="managed-recharge-issued-dialog">
        <div class="flex items-start gap-3 border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-800 dark:border-green-900/60 dark:bg-green-950/20 dark:text-green-200">
          <Icon name="checkCircle" size="md" class="mt-0.5 shrink-0" />
          <div>
            <div class="font-semibold">支付成功，CDK 已归属于当前订单</div>
            <div class="mt-1 text-xs leading-5">请保存 CDK，然后前往兑换页提交目标账号。本站会继续同步兑换进度。</div>
          </div>
        </div>
        <div>
          <div class="text-xs font-medium text-gray-500 dark:text-dark-400">订单号</div>
          <div class="mt-1 font-mono text-sm text-gray-700 dark:text-dark-200">{{ issuedOrder.order_no }}</div>
        </div>
        <div>
          <div class="text-xs font-medium text-gray-500 dark:text-dark-400">专属 CDK</div>
          <div class="mt-2 flex items-center gap-2 border border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-900">
            <code class="min-w-0 flex-1 break-all text-sm font-semibold text-gray-900 dark:text-white">{{ issuedOrder.redemption_code }}</code>
            <button type="button" class="btn btn-secondary h-9 w-9 shrink-0 p-0" :title="codeCopied ? '已复制' : '复制 CDK'" @click="copyIssuedCode">
              <Icon :name="codeCopied ? 'check' : 'copy'" size="sm" />
            </button>
          </div>
        </div>
        <p class="text-xs leading-5 text-gray-500 dark:text-dark-400">兑换页面由第三方提供。CDK 是敏感凭证，请勿转发；发放后不支持自动退款，兑换未完成时可使用同一 CDK 重试或联系客服核对。</p>
      </div>
      <template #footer>
        <button class="btn btn-secondary" type="button" @click="closeIssuedOrder">稍后处理</button>
        <a
          v-if="issuedOrder?.redemption_url"
          class="btn btn-primary inline-flex"
          :href="issuedOrder.redemption_url"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Icon name="externalLink" size="sm" class="mr-2" />
          前往兑换
        </a>
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
import PaymentStatusPanel from '@/components/payment/PaymentStatusPanel.vue'
import { getPaymentPopupFeatures } from '@/components/payment/providerConfig'
import {
  findManagedRechargeProduct,
  normalizeManagedRechargePlanKey,
} from '@/components/payment/managedRechargePlans'
import {
  createManagedRechargeOrder,
  getManagedRechargeCatalog,
  getManagedRechargeOrder,
  getManagedRechargeOrderStatus,
  listManagedRechargeOrders,
  submitManagedRechargeReplacementSession,
  validateManagedRechargeSession,
  type ManagedRechargeCatalog,
  type ManagedRechargeMembership,
  type ManagedRechargeOrder,
} from '@/api/managedRecharge'
import type { CreateOrderResult } from '@/types/payment'

const route = useRoute()
const router = useRouter()
const catalog = ref<ManagedRechargeCatalog | null>(null)
const selectedProductId = ref<number | null>(null)
const sessionJson = ref('')
const sessionEmail = ref('')
const sessionMembership = ref<ManagedRechargeMembership>('unknown')
const sessionValidationState = ref<'idle' | 'checking' | 'valid' | 'invalid' | 'error'>('idle')
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
const issuedOrder = ref<ManagedRechargeOrder | null>(null)
const codeCopied = ref(false)
const paymentState = ref<CreateOrderResult | null>(null)
const paymentManagedOrderId = ref<number | null>(null)
let pollTimer: number | null = null
let sessionValidationTimer: number | null = null
let sessionValidationRequestID = 0
const mockSessionExample = JSON.stringify({
  user: { email: 'mock@example.com' },
  accessToken: 'mock-access-token',
}, null, 2)
const isExternalMode = computed(() => catalog.value?.fulfillment_mode === 'external')
const purchaseSteps = computed(() => isExternalMode.value
  ? [
      { title: '选择套餐', description: '确认档位与库存' },
      { title: '支付宝付款', description: '到账后立即发放' },
      { title: '前往兑换', description: '在兑换页提交账号' },
    ]
  : [
      { title: '选择套餐', description: '确认档位与库存' },
      { title: '提交账号', description: '粘贴 Session JSON' },
      { title: '支付宝付款', description: '到账后跟踪进度' },
    ])
const selectedProduct = computed(() => catalog.value?.products.find((item) => item.id === selectedProductId.value) || null)
const sessionMembershipLabel = computed(() => ({
  free: 'Free',
  plus: 'Plus',
  pro: 'Pro',
  unknown: '暂未识别',
})[sessionMembership.value])
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
const canSubmit = computed(() => Boolean(
  selectedProduct.value &&
  selectedProduct.value.available_stock > 0 &&
  (isExternalMode.value || (sessionEmail.value && sessionValidationState.value === 'valid')) &&
  agreed.value,
))

function goBack(): void {
  router.push({ path: '/purchase', query: { tab: 'member' } })
}

function errorMessage(error: unknown): string {
  const structured = error as { reason?: string; code?: string; message?: string } | null
  const reason = structured?.reason || structured?.code || ''
  const knownMessages: Record<string, string> = {
    MANAGED_RECHARGE_OUT_OF_STOCK: '当前套餐已售罄，请选择其他套餐或稍后再试',
    MANAGED_RECHARGE_UPSTREAM_UNAVAILABLE: '兑换服务暂时不可用，本次不会扣款，请稍后重试',
    MANAGED_RECHARGE_UNAVAILABLE: '会员订阅服务暂未开放',
    MANAGED_RECHARGE_PAYMENT_UNAVAILABLE: '支付宝支付暂不可用，请稍后重试',
    MANAGED_RECHARGE_ALIPAY_REQUIRED: '会员订阅仅支持支付宝支付',
    MANAGED_RECHARGE_PRODUCT_INVALID: '所选套餐已下架，请刷新页面后重新选择',
  }
  if (knownMessages[reason]) return knownMessages[reason]
  if (structured?.message) return structured.message
  return '请求失败，请稍后重试'
}

function formatAmount(value: number): string {
  return Number(value || 0).toFixed(2)
}

function formatDate(value: string): string {
  return new Date(value).toLocaleString()
}

function formatSyncTime(value: string): string {
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function resetSessionValidation(): void {
  if (sessionValidationTimer !== null) {
    window.clearTimeout(sessionValidationTimer)
    sessionValidationTimer = null
  }
  sessionValidationRequestID += 1
  sessionEmail.value = ''
  sessionMembership.value = 'unknown'
  sessionValidationState.value = 'idle'
  sessionError.value = ''
}

function scheduleSessionValidation(): void {
  resetSessionValidation()
  const value = sessionJson.value.trim()
  if (!value) return

  let localEmail = ''
  try {
    const parsed = JSON.parse(value)
    localEmail = String(parsed?.user?.email || '').trim()
    if (!parsed?.accessToken || !localEmail) throw new Error('invalid')
  } catch {
    sessionError.value = 'Session JSON 格式不完整'
    sessionValidationState.value = 'invalid'
    return
  }

  sessionValidationState.value = 'checking'
  const requestID = sessionValidationRequestID
  sessionValidationTimer = window.setTimeout(() => {
    sessionValidationTimer = null
    void runSessionValidation(value, localEmail, requestID)
  }, 500)
}

async function runSessionValidation(value: string, localEmail: string, requestID: number): Promise<void> {
  try {
    const result = await validateManagedRechargeSession(value)
    if (requestID !== sessionValidationRequestID || value !== sessionJson.value.trim()) return
    if (!result.valid) {
      sessionValidationState.value = 'invalid'
      sessionError.value = 'Session 无效或已过期'
      return
    }
    sessionEmail.value = result.email?.trim() || localEmail
    sessionMembership.value = result.membership || 'unknown'
    sessionValidationState.value = 'valid'
  } catch {
    if (requestID !== sessionValidationRequestID || value !== sessionJson.value.trim()) return
    sessionValidationState.value = 'error'
    sessionError.value = '暂时无法验证 Session，请稍后重试'
  }
}

function fillMockSession(): void {
  sessionJson.value = mockSessionExample
  scheduleSessionValidation()
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
    const checkout = await createManagedRechargeOrder(
      selectedProduct.value.id,
      isExternalMode.value ? undefined : sessionJson.value,
      newIdempotencyKey(),
      `${window.location.origin}/member-recharge`,
      isMobileDevice(),
    )
    const order = checkout.order
    orders.value = [order, ...orders.value.filter((item) => item.id !== order.id)]
    agreed.value = false
    await loadCatalog()
    if (checkout.payment) {
      paymentManagedOrderId.value = order.id
      paymentState.value = checkout.payment
      if (!isExternalMode.value) {
        sessionJson.value = ''
        resetSessionValidation()
      }
      launchAlipayWindow(checkout.payment)
    } else if (isExternalMode.value && order.redemption_code) {
      issuedOrder.value = order
      codeCopied.value = false
    } else if (!isExternalMode.value) {
      sessionJson.value = ''
      resetSessionValidation()
    }
  } catch (error) {
    submitError.value = errorMessage(error)
  } finally {
    submitting.value = false
  }
}

function isMobileDevice(): boolean {
  return /Android|iPhone|iPad|iPod|Mobile/i.test(window.navigator.userAgent)
}

function launchAlipayWindow(payment: CreateOrderResult): void {
  if (!payment.pay_url || payment.qr_code) return
  if (isMobileDevice()) {
    window.location.href = payment.pay_url
    return
  }
  const popup = window.open(payment.pay_url, 'paymentPopup', getPaymentPopupFeatures())
  if (!popup || popup.closed) window.location.href = payment.pay_url
}

async function handlePaymentSuccess(): Promise<void> {
  await Promise.all([loadCatalog(), loadOrders()])
  const orderID = paymentManagedOrderId.value
  if (!orderID || !isExternalMode.value) return
  const order = orders.value.find((item) => item.id === orderID)
  if (order?.status === 'issued') await revealExternalOrder(orderID)
}

async function handlePaymentDone(): Promise<void> {
  paymentState.value = null
  paymentManagedOrderId.value = null
  await Promise.all([loadCatalog(), loadOrders()])
}

function isActiveStatus(status: string): boolean {
  return ['awaiting_payment', 'validating', 'paid', 'issued', 'submitting', 'queued', 'processing', 'verifying', 'manual_review'].includes(status)
}

async function refreshOrder(id: number): Promise<void> {
  try {
    const updated = await getManagedRechargeOrderStatus(id)
    const index = orders.value.findIndex((item) => item.id === id)
    if (index >= 0) {
      const current = orders.value[index]
      orders.value[index] = {
        ...updated,
        redemption_code: updated.redemption_code || current.redemption_code,
        redemption_url: updated.redemption_url || current.redemption_url,
      }
    }
    await loadCatalog()
  } catch {
    // Keep the last known order state when synchronization is temporarily unavailable.
  }
}

async function pollActiveOrders(): Promise<void> {
  const active = orders.value.filter((order) => {
    if (!isActiveStatus(order.status) && order.status !== 'action_required') return false
    return true
  })
  for (const order of active.slice(0, 5)) await refreshOrder(order.id)
}

function statusLabel(status: string, fulfillmentMode: ManagedRechargeOrder['fulfillment_mode'] = 'proxy'): string {
  if (status === 'issued') return '等待兑换'
  if (fulfillmentMode === 'external' && status === 'completed') return '订阅完成'
  return ({
    awaiting_payment: '等待付款',
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

async function revealExternalOrder(id: number): Promise<void> {
  submitError.value = ''
  try {
    const order = await getManagedRechargeOrder(id)
    const index = orders.value.findIndex((item) => item.id === id)
    if (index >= 0) orders.value[index] = order
    if (!order.redemption_code) throw new Error('CDK 暂时无法读取，请稍后重试')
    issuedOrder.value = order
    codeCopied.value = false
  } catch (error) {
    submitError.value = errorMessage(error)
  }
}

function closeIssuedOrder(): void {
  issuedOrder.value = null
  codeCopied.value = false
}

async function copyIssuedCode(): Promise<void> {
  const code = issuedOrder.value?.redemption_code
  if (!code) return
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(code)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = code
      textarea.setAttribute('readonly', 'true')
      textarea.style.cssText = 'position:fixed;left:0;top:0;width:1px;height:1px;opacity:0'
      document.body.appendChild(textarea)
      textarea.select()
      const copied = document.execCommand('copy')
      document.body.removeChild(textarea)
      if (!copied) throw new Error('copy failed')
    }
    codeCopied.value = true
  } catch {
    submitError.value = '复制失败，请手动选择 CDK 复制'
  }
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
  if (sessionValidationTimer !== null) window.clearTimeout(sessionValidationTimer)
  sessionValidationRequestID += 1
})
</script>
