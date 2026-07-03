<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        <section class="card overflow-hidden">
          <div class="bg-gradient-to-br from-emerald-500 to-sky-600 px-6 py-7 text-white">
            <div class="flex flex-col gap-6 md:flex-row md:items-end md:justify-between">
              <div>
                <div class="mb-4 inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-white/20 backdrop-blur-sm">
                  <Icon name="calendar" size="xl" class="text-white" />
                </div>
                <p class="text-sm font-medium text-emerald-50">每日签到</p>
                <h1 class="mt-2 text-3xl font-bold">领取今日 API 余额</h1>
                <p class="mt-2 max-w-xl text-sm text-emerald-50">
                  每天签到一次，奖励会直接加入账户余额，可用于后续 API 消费。
                </p>
              </div>
              <div class="rounded-lg border border-white/25 bg-white/15 px-4 py-3 text-left backdrop-blur-sm md:min-w-44">
                <p class="text-xs text-emerald-50">当前余额</p>
                <p class="mt-1 text-2xl font-semibold">${{ formatMoney(displayBalance) }}</p>
              </div>
            </div>
          </div>

          <div class="p-6">
            <div class="grid gap-4 sm:grid-cols-3">
              <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
                <p class="text-sm text-gray-500 dark:text-dark-400">今日奖励</p>
                <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                  ${{ status?.reward_amount?.toFixed(2) || '0.02' }}
                </p>
              </div>
              <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
                <p class="text-sm text-gray-500 dark:text-dark-400">连续签到</p>
                <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
                  {{ status?.current_streak || 0 }} 天
                </p>
              </div>
              <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
                <p class="text-sm text-gray-500 dark:text-dark-400">今日状态</p>
                <p class="mt-2 text-2xl font-semibold" :class="status?.checked_in ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400'">
                  {{ status?.checked_in ? '已签到' : '待签到' }}
                </p>
              </div>
            </div>

            <button
              type="button"
              class="btn btn-primary mt-6 w-full py-3 text-base font-medium sm:w-auto sm:min-w-48"
              :disabled="loading || submitting || status?.checked_in"
              @click="submitCheckin"
            >
              <svg
                v-if="submitting"
                class="-ml-1 mr-2 h-5 w-5 animate-spin"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <Icon v-else name="checkCircle" size="md" class="mr-2" />
              {{ status?.checked_in ? '今日已领取' : '立即签到' }}
            </button>

            <p v-if="status?.checked_in" class="mt-3 text-sm text-gray-500 dark:text-dark-400">
              下次可签到：{{ formatDateTime(status.next_checkin_at) }}
            </p>

            <div
              v-if="balanceChangeSummary"
              class="mt-4 rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-500/30 dark:bg-emerald-500/10"
            >
              <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div class="flex items-center gap-3">
                  <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-white text-emerald-600 shadow-sm dark:bg-dark-900 dark:text-emerald-400">
                    <Icon name="gift" size="md" />
                  </div>
                  <div>
                    <p class="text-sm font-medium text-emerald-900 dark:text-emerald-100">今日已到账</p>
                    <p class="text-xs text-emerald-700 dark:text-emerald-200">
                      余额 ${{ formatMoney(balanceChangeSummary.balanceBefore) }} -> ${{ formatMoney(balanceChangeSummary.balanceAfter) }}
                    </p>
                  </div>
                </div>
                <p class="text-2xl font-semibold text-emerald-700 dark:text-emerald-300">
                  +${{ formatMoney(balanceChangeSummary.rewardAmount) }}
                </p>
              </div>
            </div>
          </div>
        </section>

        <aside class="card p-5">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">签到规则</h2>
          <div class="mt-4 space-y-3 text-sm text-gray-600 dark:text-dark-300">
            <p>每天按北京时间自然日计算一次。</p>
            <p>当前试验版固定奖励 $0.02，后续可接入连续奖励和后台配置。</p>
            <p>重复点击不会重复发放，服务端会按用户和日期做唯一校验。</p>
          </div>
        </aside>
      </div>

      <section class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-800">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">最近记录</h2>
        </div>
        <div class="divide-y divide-gray-100 dark:divide-dark-800">
          <div v-if="loading" class="p-6 text-sm text-gray-500 dark:text-dark-400">
            正在加载签到状态...
          </div>
          <div
            v-for="item in status?.recent_checkins || []"
            :key="item.id"
            class="flex items-center justify-between gap-4 px-6 py-4"
          >
            <div class="flex items-center gap-3">
              <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-50 dark:bg-emerald-900/20">
                <Icon name="gift" size="md" class="text-emerald-600 dark:text-emerald-400" />
              </div>
              <div>
                <p class="font-medium text-gray-900 dark:text-white">{{ item.checkin_date }}</p>
                <p class="text-sm text-gray-500 dark:text-dark-400">连续 {{ item.streak_days }} 天</p>
              </div>
            </div>
            <div class="text-right">
              <p class="font-semibold text-emerald-600 dark:text-emerald-400">+${{ item.reward_amount.toFixed(2) }}</p>
              <p v-if="item.balance_after !== undefined" class="text-xs text-gray-500 dark:text-dark-400">
                发放后余额 ${{ item.balance_after.toFixed(2) }}
              </p>
            </div>
          </div>
          <div v-if="!loading && (status?.recent_checkins?.length || 0) === 0" class="empty-state py-10">
            <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800">
              <Icon name="calendar" size="xl" class="text-gray-400 dark:text-dark-500" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">还没有签到记录</p>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { checkinAPI, type CheckinStatus } from '@/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'

const appStore = useAppStore()
const authStore = useAuthStore()

const status = ref<CheckinStatus | null>(null)
const loading = ref(false)
const submitting = ref(false)
const claimedBalanceChange = ref<BalanceChangeSummary | null>(null)

interface BalanceChangeSummary {
  rewardAmount: number
  balanceBefore: number
  balanceAfter: number
}

const balanceChangeSummary = computed<BalanceChangeSummary | null>(() => {
  if (claimedBalanceChange.value) {
    return claimedBalanceChange.value
  }

  const todayCheckin = status.value?.today_checkin
  if (!status.value?.checked_in || todayCheckin?.balance_after === undefined) {
    return null
  }

  const rewardAmount = todayCheckin.reward_amount ?? status.value.reward_amount
  const balanceAfter = todayCheckin.balance_after
  return {
    rewardAmount,
    balanceBefore: balanceAfter - rewardAmount,
    balanceAfter
  }
})

const displayBalance = computed(() => claimedBalanceChange.value?.balanceAfter ?? authStore.user?.balance ?? 0)

function formatMoney(value: number) {
  return value.toFixed(2)
}

async function loadStatus() {
  loading.value = true
  try {
    status.value = await checkinAPI.getStatus()
  } catch (error) {
    console.error('Failed to load check-in status:', error)
    appStore.showError('加载签到状态失败')
  } finally {
    loading.value = false
  }
}

async function submitCheckin() {
  if (submitting.value || status.value?.checked_in) return
  submitting.value = true
  try {
    const balanceBefore = authStore.user?.balance ?? 0
    status.value = await checkinAPI.checkIn()
    const todayCheckin = status.value.today_checkin
    const rewardAmount = todayCheckin?.reward_amount ?? status.value.reward_amount
    const balanceAfter = todayCheckin?.balance_after ?? balanceBefore + rewardAmount
    claimedBalanceChange.value = {
      rewardAmount,
      balanceBefore,
      balanceAfter
    }
    await authStore.refreshUser()
    if (authStore.user?.balance !== undefined) {
      claimedBalanceChange.value = {
        ...claimedBalanceChange.value,
        balanceAfter: authStore.user.balance
      }
    }
    appStore.showSuccess('签到成功，奖励已加入余额')
  } catch (error: any) {
    const message = error?.message || '签到失败'
    appStore.showError(message)
    await loadStatus()
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  void loadStatus()
})
</script>
