<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <section class="card overflow-hidden">
        <div class="border-b border-gray-100 bg-white px-6 py-5 dark:border-dark-800 dark:bg-dark-900">
          <div class="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
            <div class="flex items-start gap-4">
              <div class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-lg bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-300">
                <Icon name="calendar" size="lg" />
              </div>
              <div>
                <div class="flex flex-wrap items-center gap-2">
                  <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ campaignTitle }}</h1>
                  <span class="rounded-md border border-amber-200 bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
                    预览模式
                  </span>
                </div>
                <p class="mt-2 text-sm text-gray-600 dark:text-dark-300">
                  每日打卡领取限时签到额度，额度下月末到期；周末额外发当天整天日卡。
                </p>
              </div>
            </div>
            <div class="grid grid-cols-3 gap-3 sm:min-w-[420px]">
              <div class="rounded-lg border border-gray-200 px-4 py-3 dark:border-dark-700">
                <p class="text-xs text-gray-500 dark:text-dark-400">连续</p>
                <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ currentStreak }} 天</p>
              </div>
              <div class="rounded-lg border border-gray-200 px-4 py-3 dark:border-dark-700">
                <p class="text-xs text-gray-500 dark:text-dark-400">本月已签</p>
                <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ monthCheckedCount }} 天</p>
              </div>
              <div class="rounded-lg border border-gray-200 px-4 py-3 dark:border-dark-700">
                <p class="text-xs text-gray-500 dark:text-dark-400">月度额度上限</p>
                <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">$20</p>
              </div>
            </div>
          </div>
        </div>

        <div class="grid gap-0 lg:grid-cols-[minmax(0,1fr)_360px]">
          <div class="space-y-6 p-6">
            <div class="grid gap-4 md:grid-cols-3">
              <div class="rounded-lg border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-500/30 dark:bg-emerald-500/10">
                <div class="flex items-center gap-3">
                  <Icon name="gift" size="md" class="text-emerald-600 dark:text-emerald-300" />
                  <p class="text-sm font-medium text-emerald-900 dark:text-emerald-100">今日活动奖励</p>
                </div>
                <p class="mt-3 text-lg font-semibold text-emerald-900 dark:text-emerald-100">{{ todayPlan.title }}</p>
                <p class="mt-1 text-sm text-emerald-700 dark:text-emerald-200">{{ todayPlan.summary }}</p>
              </div>

              <div class="rounded-lg border border-sky-200 bg-sky-50 p-4 dark:border-sky-500/30 dark:bg-sky-500/10">
                <div class="flex items-center gap-3">
                  <Icon name="bolt" size="md" class="text-sky-600 dark:text-sky-300" />
                  <p class="text-sm font-medium text-sky-900 dark:text-sky-100">后端测试到账</p>
                </div>
                <p class="mt-3 text-lg font-semibold text-sky-900 dark:text-sky-100">${{ formatMoney(status?.reward_amount ?? 0.02) }}</p>
                <p class="mt-1 text-sm text-sky-700 dark:text-sky-200">当前仍写入余额，待替换为下月末到期额度</p>
              </div>

              <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/60">
                <div class="flex items-center gap-3">
                  <Icon name="badge" size="md" class="text-gray-600 dark:text-dark-300" />
                  <p class="text-sm font-medium text-gray-900 dark:text-white">签到额度进度</p>
                </div>
                <p class="mt-3 text-lg font-semibold text-gray-900 dark:text-white">${{ formatMoney(previewMonthlyCredit) }} / $20</p>
                <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">预估权益，统一 {{ monthlyCreditExpiryLabel }} 到期</p>
              </div>
            </div>

            <div v-if="balanceChangeSummary" class="rounded-lg border border-emerald-200 bg-white p-4 dark:border-emerald-500/30 dark:bg-dark-900">
              <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div class="flex items-center gap-3">
                  <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-50 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-300">
                    <Icon name="checkCircle" size="md" />
                  </div>
                  <div>
                    <p class="text-sm font-medium text-gray-900 dark:text-white">当前后端测试到账</p>
                    <p class="text-xs text-gray-500 dark:text-dark-400">
                      余额 ${{ formatMoney(balanceChangeSummary.balanceBefore) }} -> ${{ formatMoney(balanceChangeSummary.balanceAfter) }}
                    </p>
                  </div>
                </div>
                <p class="text-2xl font-semibold text-emerald-600 dark:text-emerald-300">+${{ formatMoney(balanceChangeSummary.rewardAmount) }}</p>
              </div>
            </div>

            <div class="rounded-lg border border-gray-200 dark:border-dark-700">
              <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-800">
                <div>
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">月度日历</h2>
                  <p class="text-sm text-gray-500 dark:text-dark-400">{{ campaignPeriodLabel }}</p>
                </div>
                <div class="text-right">
                  <p class="text-sm font-medium text-gray-900 dark:text-white">{{ monthProgressPercent }}%</p>
                  <p class="text-xs text-gray-500 dark:text-dark-400">完成度</p>
                </div>
              </div>

              <div class="grid grid-cols-7 gap-px bg-gray-100 p-px text-center text-xs dark:bg-dark-800">
                <div v-for="day in weekLabels" :key="day" class="bg-gray-50 px-2 py-2 font-medium text-gray-500 dark:bg-dark-900 dark:text-dark-400">
                  {{ day }}
                </div>
                <div
                  v-for="cell in calendarCells"
                  :key="cell.key"
                  class="min-h-[82px] bg-white p-2 text-left dark:bg-dark-950"
                  :class="cell.empty ? 'opacity-40' : ''"
                >
                  <template v-if="!cell.empty">
                    <div class="flex items-center justify-between gap-2">
                      <span
                        class="inline-flex h-7 w-7 items-center justify-center rounded-md text-sm font-semibold"
                        :class="dayNumberClass(cell)"
                      >
                        {{ cell.dayNumber }}
                      </span>
                      <Icon v-if="cell.checked" name="checkCircle" size="sm" class="text-emerald-500" />
                    </div>
                    <p class="mt-2 truncate text-xs font-medium" :class="cell.weekend ? 'text-amber-700 dark:text-amber-300' : 'text-gray-600 dark:text-dark-300'">
                      {{ cell.rewardShort }}
                    </p>
                    <p class="mt-1 text-[11px] text-gray-400 dark:text-dark-500">{{ cell.statusLabel }}</p>
                  </template>
                </div>
              </div>
            </div>
          </div>

          <aside class="space-y-5 border-t border-gray-100 p-6 dark:border-dark-800 lg:border-l lg:border-t-0">
            <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <div class="flex items-center justify-between">
                <div>
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">今日打卡</h2>
                  <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ status?.checked_in ? '今天已完成' : '预览：领取 $0.25 签到额度' }}</p>
                </div>
                <span class="rounded-md px-2 py-1 text-xs font-medium" :class="status?.checked_in ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300' : 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'">
                  {{ status?.checked_in ? '已签到' : '待签到' }}
                </span>
              </div>
              <button
                type="button"
                class="btn btn-primary mt-4 w-full py-3 text-base font-medium"
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
                {{ status?.checked_in ? '今日已领取' : '立即打卡' }}
              </button>
              <p v-if="status?.checked_in" class="mt-3 text-sm text-gray-500 dark:text-dark-400">
                下次可打卡：{{ formatDateTime(status.next_checkin_at) }}
              </p>
              <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">
                新方案额度有效期：{{ monthlyCreditExpiryLabel }} 到期。
              </p>
            </div>

            <div class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-500/30 dark:bg-amber-500/10">
              <div class="flex items-center gap-3">
                <Icon name="sun" size="md" class="text-amber-600 dark:text-amber-300" />
                <h2 class="text-base font-semibold text-amber-900 dark:text-amber-100">本周末奖励</h2>
              </div>
              <div class="mt-4 space-y-3">
                <div v-for="item in weekendRewards" :key="item.date" class="rounded-lg bg-white p-3 dark:bg-dark-900">
                  <div class="flex items-center justify-between gap-3">
                    <div>
                      <p class="font-medium text-gray-900 dark:text-white">{{ item.title }}</p>
                      <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ item.dateLabel }}</p>
                    </div>
                    <span class="rounded-md bg-amber-100 px-2 py-1 text-xs font-medium text-amber-700 dark:bg-amber-500/20 dark:text-amber-200">待接入</span>
                  </div>
                  <p class="mt-2 text-sm text-gray-600 dark:text-dark-300">{{ item.summary }}</p>
                </div>
              </div>
            </div>

            <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">里程碑</h2>
              <div class="mt-4 space-y-3">
                <div v-for="item in milestones" :key="item.days" class="space-y-2">
                  <div class="flex items-center justify-between gap-3 text-sm">
                    <span class="font-medium text-gray-700 dark:text-dark-200">连续 {{ item.days }} 天</span>
                    <span class="text-gray-500 dark:text-dark-400">{{ item.reward }}</span>
                  </div>
                  <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-800">
                    <div class="h-full rounded-full bg-emerald-500" :style="{ width: milestoneWidth(item.days) }"></div>
                  </div>
                </div>
              </div>
            </div>

            <div class="rounded-lg border border-gray-200 p-4 text-sm text-gray-600 dark:border-dark-700 dark:text-dark-300">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">活动规则</h2>
              <div class="mt-3 space-y-2 leading-6">
                <p>按北京时间自然日计算，每天只能打卡一次。</p>
                <p>签到额度每月最多 $20，不发永久额度，领取月份的下月末到期。</p>
                <p>周末日卡按当天 00:00 到次日 00:00 计算，不从点击时刻开始滚动。</p>
                <p>当前页面用于 18080 预览，后端仍发放测试余额，日卡和到期额度尚未真实接入。</p>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-800">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">最近记录</h2>
        </div>
        <div class="divide-y divide-gray-100 dark:divide-dark-800">
          <div v-if="loading" class="p-6 text-sm text-gray-500 dark:text-dark-400">
            正在加载签到状态...
          </div>
          <div
            v-for="item in checkinHistory.slice(0, 8)"
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
                当前测试余额 ${{ item.balance_after.toFixed(2) }}
              </p>
            </div>
          </div>
          <div v-if="!loading && checkinHistory.length === 0" class="empty-state py-10">
            <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-lg bg-gray-100 dark:bg-dark-800">
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
import { checkinAPI, type CheckinStatus, type UserCheckin } from '@/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'

const appStore = useAppStore()
const authStore = useAuthStore()

const status = ref<CheckinStatus | null>(null)
const checkinHistory = ref<UserCheckin[]>([])
const loading = ref(false)
const submitting = ref(false)
const claimedBalanceChange = ref<BalanceChangeSummary | null>(null)

interface BalanceChangeSummary {
  rewardAmount: number
  balanceBefore: number
  balanceAfter: number
}

interface RewardPlan {
  title: string
  summary: string
  short: string
  tone: 'weekday' | 'saturday' | 'sunday'
}

interface CalendarCell {
  key: string
  empty: boolean
  date?: string
  dayNumber?: number
  checked?: boolean
  today?: boolean
  weekend?: boolean
  rewardShort?: string
  statusLabel?: string
}

const weekLabels = ['一', '二', '三', '四', '五', '六', '日']
const milestones = [
  { days: 3, reward: '$0.50 签到额度' },
  { days: 7, reward: '$1 + 保护卡' },
  { days: 14, reward: '$1.50 签到额度' },
  { days: 21, reward: '$2 签到额度' },
  { days: 30, reward: '补足到 $20' }
]
const dayMs = 24 * 60 * 60 * 1000
const monthlyCreditCap = 20
const dailyCheckinCredit = 0.25

const todayDate = computed(() => status.value?.today || formatShanghaiDate(new Date()))
const todayParts = computed(() => splitDate(todayDate.value))
const currentStreak = computed(() => status.value?.current_streak ?? 0)
const monthStart = computed(() => `${todayParts.value.year}-${pad2(todayParts.value.month)}-01`)
const monthDayCount = computed(() => new Date(Date.UTC(todayParts.value.year, todayParts.value.month, 0)).getUTCDate())
const campaignTitle = computed(() => `${todayParts.value.month} 月签到挑战`)
const campaignPeriodLabel = computed(() => {
  const start = monthStart.value
  const end = formatDateUTC(dateUTC(start) + monthDayCount.value * dayMs)
  return `${start} 至 ${end} 00:00`
})

const checkedDateSet = computed(() => new Set(checkinHistory.value.map((item) => item.checkin_date)))
const monthCheckedCount = computed(() => {
  const prefix = `${todayParts.value.year}-${pad2(todayParts.value.month)}-`
  return checkinHistory.value.filter((item) => item.checkin_date.startsWith(prefix)).length
})
const monthProgressPercent = computed(() => {
  if (monthDayCount.value <= 0) return 0
  return Math.round((monthCheckedCount.value / monthDayCount.value) * 100)
})

const todayPlan = computed(() => rewardPlanForDate(todayDate.value))
const monthlyCreditExpiryLabel = computed(() => {
  const expiryTs = Date.UTC(todayParts.value.year, todayParts.value.month + 1, 1) - dayMs
  return `${formatDateUTC(expiryTs)} 23:59`
})
const previewMonthlyCredit = computed(() => {
  const count = monthCheckedCount.value
  const streak = currentStreak.value
  let total = count * dailyCheckinCredit

  if (streak >= 3) total += 0.5
  if (streak >= 7) total += 1
  if (streak >= 14) total += 1.5
  if (streak >= 21) total += 2

  if (count >= 10) total += 1
  if (count >= 20) total += 2
  if (count >= 25) total += 2

  return Math.min(monthlyCreditCap, total)
})

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

const calendarCells = computed<CalendarCell[]>(() => {
  const start = dateUTC(monthStart.value)
  const firstWeekday = normalizedWeekday(start)
  const cells: CalendarCell[] = []

  for (let i = 0; i < firstWeekday; i += 1) {
    cells.push({ key: `empty-${i}`, empty: true })
  }

  const todayTs = dateUTC(todayDate.value)
  for (let day = 1; day <= monthDayCount.value; day += 1) {
    const date = `${todayParts.value.year}-${pad2(todayParts.value.month)}-${pad2(day)}`
    const ts = dateUTC(date)
    const weekday = new Date(ts).getUTCDay()
    const plan = rewardPlanForDate(date)
    const checked = checkedDateSet.value.has(date)
    cells.push({
      key: date,
      empty: false,
      date,
      dayNumber: day,
      checked,
      today: date === todayDate.value,
      weekend: weekday === 0 || weekday === 6,
      rewardShort: plan.short,
      statusLabel: checked ? '已签' : ts < todayTs ? '未签' : plan.tone === 'weekday' ? '基础' : '周末'
    })
  }

  return cells
})

const weekendRewards = computed(() => {
  const todayTs = dateUTC(todayDate.value)
  const mondayTs = todayTs - normalizedWeekday(todayTs) * dayMs
  const saturday = formatDateUTC(mondayTs + 5 * dayMs)
  const sunday = formatDateUTC(mondayTs + 6 * dayMs)
  return [
    {
      date: saturday,
      title: '周六打卡',
      dateLabel: `${saturday} 00:00 - ${sunday} 00:00`,
      summary: '周六整天日卡，不折算进钱包额度'
    },
    {
      date: sunday,
      title: '周日打卡',
      dateLabel: `${sunday} 00:00 - ${formatDateUTC(dateUTC(sunday) + dayMs)} 00:00`,
      summary: '周日整天日卡，不折算进钱包额度'
    }
  ]
})

function rewardPlanForDate(date: string): RewardPlan {
  const weekday = new Date(dateUTC(date)).getUTCDay()
  if (weekday === 6) {
    return {
      title: '$0.25 签到额度 + 周六整天日卡',
      summary: `签到额度下月末到期；日卡周六 00:00 到周日 00:00`,
      short: '$0.25 + 日卡',
      tone: 'saturday'
    }
  }
  if (weekday === 0) {
    return {
      title: '$0.25 签到额度 + 周日整天日卡',
      summary: `签到额度下月末到期；日卡周日 00:00 到周一 00:00`,
      short: '$0.25 + 日卡',
      tone: 'sunday'
    }
  }
  return {
    title: '$0.25 签到额度',
    summary: `正式规则接入后不发永久额度，统一 ${monthlyCreditExpiryLabel.value} 到期`,
    short: '$0.25',
    tone: 'weekday'
  }
}

function dayNumberClass(cell: CalendarCell) {
  if (cell.today) return 'bg-sky-600 text-white'
  if (cell.checked) return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-200'
  if (cell.weekend) return 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-200'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300'
}

function milestoneWidth(days: number) {
  return `${Math.min(100, Math.round((currentStreak.value / days) * 100))}%`
}

function formatMoney(value: number) {
  return value.toFixed(2)
}

function formatShanghaiDate(date: Date) {
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  }).formatToParts(date)
  const year = parts.find((part) => part.type === 'year')?.value || '1970'
  const month = parts.find((part) => part.type === 'month')?.value || '01'
  const day = parts.find((part) => part.type === 'day')?.value || '01'
  return `${year}-${month}-${day}`
}

function splitDate(value: string) {
  const [year, month, day] = value.split('-').map((part) => Number(part))
  return { year, month, day }
}

function pad2(value: number) {
  return String(value).padStart(2, '0')
}

function dateUTC(value: string) {
  const { year, month, day } = splitDate(value)
  return Date.UTC(year, month - 1, day)
}

function formatDateUTC(value: number) {
  const date = new Date(value)
  return `${date.getUTCFullYear()}-${pad2(date.getUTCMonth() + 1)}-${pad2(date.getUTCDate())}`
}

function normalizedWeekday(value: number) {
  const day = new Date(value).getUTCDay()
  return (day + 6) % 7
}

async function loadStatus() {
  loading.value = true
  try {
    const [statusData, historyData] = await Promise.all([
      checkinAPI.getStatus(),
      checkinAPI.getHistory(60)
    ])
    status.value = statusData
    checkinHistory.value = historyData
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
    const [userData, historyData] = await Promise.all([
      authStore.refreshUser(),
      checkinAPI.getHistory(60)
    ])
    checkinHistory.value = historyData
    claimedBalanceChange.value = {
      ...claimedBalanceChange.value,
      balanceAfter: userData.balance
    }
    appStore.showSuccess('打卡成功，当前后端测试奖励已加入余额')
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
