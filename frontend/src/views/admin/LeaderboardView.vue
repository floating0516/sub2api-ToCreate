<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card p-5">
        <div class="flex flex-wrap items-end gap-4">
          <div class="min-w-56">
            <label class="input-label">奖励订阅分组</label>
            <select v-model.number="settings.subscription_group_id" class="input">
              <option :value="null">请选择 GPT Pro 分组</option>
              <option v-for="group in subscriptionGroups" :key="group.id" :value="group.id">
                {{ group.name }}
              </option>
            </select>
          </div>
          <label class="flex items-center gap-2 pb-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="settings.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            启用奖励生成
          </label>
          <button class="btn btn-primary" :disabled="savingSettings" @click="saveSettings">
            保存配置
          </button>
          <div class="text-sm text-gray-500 dark:text-gray-400">
            周榜第一奖励日卡，月榜第一奖励周卡。
          </div>
        </div>
      </div>

      <div class="card">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">Token 排行管理</h2>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ period === 'week' ? '本周' : '本月' }} · {{ formatDate(snapshot?.period_start) }} 至 {{ formatDate(snapshot?.period_end) }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
              <button class="rounded-md px-3 py-1.5 text-sm font-medium" :class="period === 'week' ? activeTabClass : inactiveTabClass" @click="setPeriod('week')">周榜</button>
              <button class="rounded-md px-3 py-1.5 text-sm font-medium" :class="period === 'month' ? activeTabClass : inactiveTabClass" @click="setPeriod('month')">月榜</button>
            </div>
            <button class="btn btn-secondary" :disabled="loading" @click="load">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              刷新
            </button>
            <button class="btn btn-primary" :disabled="generating || !canGenerateReward" @click="generateReward">
              <Icon name="gift" size="sm" />
              生成第一名奖励
            </button>
          </div>
        </div>

        <div v-if="rewardCode" class="border-b border-emerald-100 bg-emerald-50 px-5 py-3 text-sm text-emerald-700 dark:border-emerald-900/40 dark:bg-emerald-900/20 dark:text-emerald-300">
          已生成兑换码：<code class="font-mono">{{ rewardCode }}</code>
        </div>

        <div v-if="loading" class="flex items-center justify-center py-12">
          <LoadingSpinner />
        </div>
        <div v-else-if="!snapshot?.entries.length" class="py-12 text-center text-sm text-gray-500 dark:text-gray-400">
          暂无排行数据
        </div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800/60">
              <tr>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">排名</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">用户</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">角色</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">匿名</th>
                <th class="px-5 py-3 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Token</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="entry in snapshot.entries" :key="entry.user_id" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
                <td class="px-5 py-3">
                  <span class="badge" :class="entry.rank === 1 ? 'badge-warning' : 'badge-gray'">#{{ entry.rank }}</span>
                </td>
                <td class="px-5 py-3">
                  <div class="text-sm font-medium text-gray-900 dark:text-white">{{ entry.display_name }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ entry.email || `用户 #${entry.user_id}` }}</div>
                </td>
                <td class="px-5 py-3">
                  <span class="badge" :class="entry.role === 'admin' ? 'badge-danger' : 'badge-primary'">
                    {{ entry.role === 'admin' ? '管理员' : '用户' }}
                  </span>
                </td>
                <td class="px-5 py-3 text-sm text-gray-600 dark:text-gray-300">
                  {{ entry.anonymous ? '匿名' : '实名' }}
                </td>
                <td class="px-5 py-3 text-right text-sm font-semibold text-gray-900 dark:text-white">
                  {{ formatTokens(entry.token_count) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-if="snapshot?.rewards.length" class="border-t border-gray-100 px-5 py-4 dark:border-dark-700">
          <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">本周期已生成奖励</h3>
          <div class="space-y-2">
            <div v-for="reward in snapshot.rewards" :key="reward.id" class="flex flex-wrap items-center justify-between gap-3 rounded-lg bg-gray-50 px-3 py-2 text-sm dark:bg-dark-800">
              <span class="text-gray-700 dark:text-gray-300">
                #{{ reward.rank }} · 用户 {{ reward.user_id }} · {{ reward.reward_type === 'daily_card' ? '日卡' : '周卡' }}
              </span>
              <code class="font-mono text-gray-900 dark:text-white">{{ reward.redeem_code }}</code>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  getAdminLeaderboard,
  getLeaderboardSettings,
  generateLeaderboardReward,
  updateLeaderboardSettings
} from '@/api/admin/leaderboard'
import { getAll as getAllGroups } from '@/api/admin/groups'
import type { AdminGroup } from '@/types'
import type {
  AdminLeaderboardSnapshot,
  LeaderboardPeriod,
  LeaderboardRewardSettings
} from '@/api/leaderboard'
import { formatCompactNumber } from '@/utils/format'

const period = ref<LeaderboardPeriod>('week')
const snapshot = ref<AdminLeaderboardSnapshot | null>(null)
const loading = ref(false)
const savingSettings = ref(false)
const generating = ref(false)
const rewardCode = ref('')
const subscriptionGroups = ref<AdminGroup[]>([])

const settings = reactive<LeaderboardRewardSettings>({
  enabled: false,
  subscription_group_id: null,
  weekly_first_days: 1,
  monthly_first_days: 7
})

const activeTabClass = 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300'
const inactiveTabClass = 'text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white'
const canGenerateReward = computed(() => settings.enabled && !!settings.subscription_group_id && !!snapshot.value?.entries.length)

async function load() {
  loading.value = true
  rewardCode.value = ''
  try {
    snapshot.value = await getAdminLeaderboard(period.value, 50)
    Object.assign(settings, snapshot.value.reward_settings)
  } finally {
    loading.value = false
  }
}

async function loadSettingsAndGroups() {
  const [remoteSettings, groups] = await Promise.all([getLeaderboardSettings(), getAllGroups()])
  Object.assign(settings, remoteSettings)
  subscriptionGroups.value = groups.filter((group) => group.subscription_type === 'subscription' && group.status === 'active')
}

async function setPeriod(next: LeaderboardPeriod) {
  if (period.value === next) return
  period.value = next
  await load()
}

async function saveSettings() {
  savingSettings.value = true
  try {
    const saved = await updateLeaderboardSettings({
      enabled: settings.enabled,
      subscription_group_id: settings.subscription_group_id || null,
      weekly_first_days: 1,
      monthly_first_days: 7
    })
    Object.assign(settings, saved)
    await load()
  } finally {
    savingSettings.value = false
  }
}

async function generateReward() {
  generating.value = true
  try {
    const result = await generateLeaderboardReward(period.value)
    rewardCode.value = result.code
    await load()
    rewardCode.value = result.code
  } finally {
    generating.value = false
  }
}

function formatTokens(value: number) {
  return formatCompactNumber(value)
}

function formatDate(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleDateString('zh-CN')
}

onMounted(async () => {
  await loadSettingsAndGroups()
  await load()
})
</script>
