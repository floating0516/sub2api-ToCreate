<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div class="card p-5">
          <div class="flex items-center gap-3">
            <div class="rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
              <Icon name="cube" size="md" class="text-amber-600 dark:text-amber-400" />
            </div>
            <div>
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">今日 Token</p>
              <p class="text-2xl font-bold text-gray-900 dark:text-white">
                {{ formatTokens(snapshot?.today_tokens || 0) }}
              </p>
            </div>
          </div>
        </div>
        <div class="card p-5">
          <div class="flex items-center gap-3">
            <div class="rounded-lg bg-primary-100 p-2 dark:bg-primary-900/30">
              <Icon name="badge" size="md" class="text-primary-600 dark:text-primary-400" />
            </div>
            <div>
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">我的排名</p>
              <p class="text-2xl font-bold text-gray-900 dark:text-white">
                {{ snapshot?.my_rank ? `#${snapshot.my_rank.rank}` : '-' }}
              </p>
            </div>
          </div>
        </div>
        <div class="card p-5">
          <div class="flex items-center justify-between gap-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-slate-100 p-2 dark:bg-dark-700">
                <Icon name="eyeOff" size="md" class="text-slate-600 dark:text-slate-300" />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">匿名展示</p>
                <p class="text-sm text-gray-600 dark:text-gray-300">
                  {{ anonymous ? '当前匿名' : '当前实名' }}
                </p>
              </div>
            </div>
            <button
              type="button"
              class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
              :class="anonymous ? 'bg-primary-600' : 'bg-gray-300 dark:bg-dark-600'"
              :disabled="savingPrivacy"
              @click="togglePrivacy"
            >
              <span
                class="inline-block h-5 w-5 rounded-full bg-white transition-transform"
                :class="anonymous ? 'translate-x-5' : 'translate-x-1'"
              />
            </button>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">Token 排行</h2>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ periodLabel }} · {{ formatDate(snapshot?.period_start) }} 至 {{ formatDate(snapshot?.period_end) }}
            </p>
          </div>
          <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
            <button
              class="rounded-md px-3 py-1.5 text-sm font-medium"
              :class="period === 'week' ? activeTabClass : inactiveTabClass"
              @click="setPeriod('week')"
            >
              周榜
            </button>
            <button
              class="rounded-md px-3 py-1.5 text-sm font-medium"
              :class="period === 'month' ? activeTabClass : inactiveTabClass"
              @click="setPeriod('month')"
            >
              月榜
            </button>
          </div>
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
                <th class="px-5 py-3 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">Token</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="entry in snapshot.entries" :key="entry.user_id" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
                <td class="px-5 py-3">
                  <span class="badge" :class="entry.rank === 1 ? 'badge-warning' : 'badge-gray'">#{{ entry.rank }}</span>
                </td>
                <td class="px-5 py-3 text-sm font-medium text-gray-900 dark:text-white">
                  {{ entry.display_name }}
                </td>
                <td class="px-5 py-3 text-right text-sm font-semibold text-gray-900 dark:text-white">
                  {{ formatTokens(entry.token_count) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  getLeaderboard,
  updateLeaderboardPrivacy,
  type LeaderboardPeriod,
  type LeaderboardSnapshot
} from '@/api/leaderboard'
import { formatCompactNumber } from '@/utils/format'

const period = ref<LeaderboardPeriod>('week')
const snapshot = ref<LeaderboardSnapshot | null>(null)
const loading = ref(false)
const savingPrivacy = ref(false)

const anonymous = computed(() => snapshot.value?.preference.anonymous ?? false)
const periodLabel = computed(() => (period.value === 'week' ? '本周' : '本月'))
const activeTabClass = 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300'
const inactiveTabClass = 'text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white'

async function load() {
  loading.value = true
  try {
    snapshot.value = await getLeaderboard(period.value, 20)
  } finally {
    loading.value = false
  }
}

async function setPeriod(next: LeaderboardPeriod) {
  if (period.value === next) return
  period.value = next
  await load()
}

async function togglePrivacy() {
  savingPrivacy.value = true
  try {
    const pref = await updateLeaderboardPrivacy(!anonymous.value)
    if (snapshot.value) {
      snapshot.value.preference = pref
    }
    await load()
  } finally {
    savingPrivacy.value = false
  }
}

function formatTokens(value: number) {
  return formatCompactNumber(value)
}

function formatDate(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleDateString('zh-CN')
}

onMounted(load)
</script>

