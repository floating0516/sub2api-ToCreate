<template>
  <AppLayout>
    <div class="space-y-5">
      <header class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('admin.dailyReport.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.dailyReport.description') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button class="btn btn-secondary btn-sm" :title="t('admin.dailyReport.previousDay')" @click="moveDate(-1)"><Icon name="arrowLeft" size="sm" /></button>
          <input v-model="selectedDate" type="date" :max="today" class="input h-9 w-[150px]" @change="loadReport" />
          <button class="btn btn-secondary btn-sm" :disabled="selectedDate >= today" :title="t('admin.dailyReport.nextDay')" @click="moveDate(1)"><Icon name="arrowRight" size="sm" /></button>
          <button class="btn btn-secondary btn-sm gap-1.5" :disabled="loading" @click="loadReport"><Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />{{ t('admin.dailyReport.refresh') }}</button>
          <button class="btn btn-primary btn-sm gap-1.5" :disabled="!report" @click="exportCSV"><Icon name="download" size="sm" />{{ t('admin.dailyReport.export') }}</button>
        </div>
      </header>

      <div v-if="loading" class="flex justify-center py-20"><LoadingSpinner /></div>
      <template v-else-if="report">
        <div class="grid grid-cols-2 gap-3 xl:grid-cols-6">
          <div v-for="item in headlineMetrics" :key="item.key" class="card min-w-0 p-4">
            <p class="truncate text-xs font-medium text-gray-500 dark:text-gray-400">{{ item.label }}</p>
            <p class="mt-2 truncate text-xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
            <p class="mt-1 text-xs" :class="item.changeClass">{{ item.change }}</p>
          </div>
        </div>

        <section class="border-y border-gray-200 py-4 dark:border-dark-700">
          <div class="mb-3 flex items-center justify-between">
            <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.dailyReport.trend') }}</h2>
            <span class="text-xs text-gray-500">{{ report.timezone }}</span>
          </div>
          <div class="grid h-44 grid-cols-7 items-end gap-2">
            <div v-for="point in trendPoints" :key="point.date" class="flex h-full min-w-0 flex-col justify-end">
              <div class="mb-1 truncate text-center text-xs font-medium text-gray-700 dark:text-gray-200">${{ formatCost(point.actual_cost) }}</div>
              <div class="mx-auto w-full max-w-14 rounded-t bg-emerald-500/80" :style="{ height: trendHeight(point.actual_cost) }"></div>
              <div class="mt-2 truncate text-center text-xs text-gray-500">{{ point.date.slice(5) }}</div>
            </div>
          </div>
        </section>

        <div class="grid gap-5 xl:grid-cols-[minmax(0,2fr)_minmax(300px,1fr)]">
          <section class="min-w-0">
            <h2 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.dailyReport.groupBreakdown') }}</h2>
            <div class="overflow-x-auto border-y border-gray-200 dark:border-dark-700">
              <table class="min-w-full text-sm">
                <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400">
                  <tr><th class="w-10 px-3 py-2"></th><th class="px-3 py-2">{{ t('admin.dailyReport.group') }}</th><th class="px-3 py-2 text-right">{{ t('admin.dailyReport.users') }}</th><th class="px-3 py-2 text-right">{{ t('admin.dailyReport.requests') }}</th><th class="px-3 py-2 text-right">{{ t('admin.dailyReport.standardCost') }}</th><th class="px-3 py-2 text-right">{{ t('admin.dailyReport.actualCost') }}</th><th class="px-3 py-2 text-right">{{ t('admin.dailyReport.share') }}</th></tr>
                </thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <template v-for="group in report.groups" :key="group.group_id">
                    <tr class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
                      <td class="px-3 py-2"><button class="p-1" :title="t('admin.dailyReport.details')" @click="toggleGroup(group.group_id)"><Icon name="chevronRight" size="sm" class="transition-transform" :class="{ 'rotate-90': expandedGroups.has(group.group_id) }" /></button></td>
                      <td class="max-w-[240px] truncate px-3 py-2 font-medium text-gray-900 dark:text-white">{{ displayGroupName(group) }}</td>
                      <td class="px-3 py-2 text-right">{{ formatNumber(group.active_users) }}</td><td class="px-3 py-2 text-right">{{ formatNumber(group.requests) }}</td>
                      <td class="px-3 py-2 text-right">${{ formatCost(group.cost) }}</td><td class="px-3 py-2 text-right font-medium text-emerald-600">${{ formatCost(group.actual_cost) }}</td>
                      <td class="px-3 py-2 text-right">{{ percent(group.actual_cost, report.summary.total_actual_cost) }}</td>
                    </tr>
                    <tr v-if="expandedGroups.has(group.group_id)" class="bg-gray-50/70 dark:bg-dark-800/40">
                      <td></td><td colspan="6" class="px-3 py-3">
                        <div class="flex flex-wrap gap-2">
                          <span v-for="item in group.multipliers" :key="item.rate_multiplier" class="rounded border border-gray-200 bg-white px-2.5 py-1 text-xs dark:border-dark-600 dark:bg-dark-800"><b>{{ formatMultiplier(item.rate_multiplier) }}</b> · {{ formatNumber(item.requests) }} · <span class="text-emerald-600">${{ formatCost(item.actual_cost) }}</span></span>
                        </div>
                      </td>
                    </tr>
                  </template>
                  <tr v-if="report.groups.length === 0"><td colspan="7" class="px-3 py-10 text-center text-gray-500">{{ t('admin.dailyReport.noData') }}</td></tr>
                </tbody>
              </table>
            </div>
          </section>

          <section class="min-w-0">
            <h2 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.dailyReport.multiplierBreakdown') }}</h2>
            <div class="space-y-3">
              <div v-for="item in report.multipliers" :key="item.rate_multiplier" class="border-b border-gray-100 pb-2 dark:border-dark-700">
                <div class="flex items-center justify-between text-sm"><span class="font-semibold text-gray-900 dark:text-white">{{ formatMultiplier(item.rate_multiplier) }}</span><span class="font-medium text-emerald-600">${{ formatCost(item.actual_cost) }}</span></div>
                <div class="mt-1 h-1.5 overflow-hidden rounded bg-gray-100 dark:bg-dark-700"><div class="h-full bg-emerald-500" :style="{ width: percent(item.actual_cost, report.summary.total_actual_cost) }"></div></div>
                <div class="mt-1 flex justify-between text-xs text-gray-500"><span>{{ formatNumber(item.active_users) }} {{ t('admin.dailyReport.users') }}</span><span>{{ formatNumber(item.requests) }} {{ t('admin.dailyReport.requests') }}</span></div>
              </div>
              <p v-if="report.multipliers.length === 0" class="py-8 text-center text-sm text-gray-500">{{ t('admin.dailyReport.noData') }}</p>
            </div>
          </section>
        </div>

        <section>
          <h2 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.dailyReport.userRanking') }}</h2>
          <div class="overflow-x-auto border-y border-gray-200 dark:border-dark-700">
            <table class="min-w-full text-sm">
              <thead class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400"><tr><th class="px-3 py-2">#</th><th class="px-3 py-2">{{ t('admin.dailyReport.user') }}</th><th class="px-3 py-2 text-right">{{ t('admin.dailyReport.requests') }}</th><th class="px-3 py-2 text-right">{{ t('admin.dailyReport.tokens') }}</th><th class="px-3 py-2 text-right">{{ t('admin.dailyReport.standardCost') }}</th><th class="px-3 py-2 text-right">{{ t('admin.dailyReport.actualCost') }}</th></tr></thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="(user, index) in report.users" :key="user.user_id"><td class="px-3 py-2 text-gray-400">{{ index + 1 }}</td><td class="px-3 py-2"><div class="font-medium text-gray-900 dark:text-white">{{ user.email }}</div><div v-if="user.username" class="text-xs text-gray-500">{{ user.username }}</div></td><td class="px-3 py-2 text-right">{{ formatNumber(user.requests) }}</td><td class="px-3 py-2 text-right">{{ formatNumber(user.total_tokens) }}</td><td class="px-3 py-2 text-right">${{ formatCost(user.cost) }}</td><td class="px-3 py-2 text-right font-medium text-emerald-600">${{ formatCost(user.actual_cost) }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminUsageAPI } from '@/api/admin/usage'
import type { DailyReportGroupStat, DailyReportResponse } from '@/api/admin/usage'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const report = ref<DailyReportResponse | null>(null)
const expandedGroups = ref(new Set<number>())
const toDateString = (date: Date) => date.toLocaleDateString('en-CA')
const today = toDateString(new Date())
const initialDate = new Date()
initialDate.setDate(initialDate.getDate() - 1)
const selectedDate = ref(toDateString(initialDate))
const formatNumber = (value: number) => new Intl.NumberFormat().format(value || 0)
const formatCost = (value: number) => Number(value || 0).toFixed(4)
const formatMultiplier = (value: number) => Number(value || 0).toFixed(2).replace(/\.00$/, '') + 'x'
const percent = (value: number, total: number) => total > 0 ? Math.min(100, value / total * 100).toFixed(1) + '%' : '0%'
const changeText = (current: number, previous: number) => previous > 0 ? (current >= previous ? '+' : '') + ((current - previous) / previous * 100).toFixed(1) + '%' : (current > 0 ? '+100.0%' : '0.0%')
const changeClass = (current: number, previous: number) => current > previous ? 'text-emerald-600' : current < previous ? 'text-red-500' : 'text-gray-400'
const metric = (key: string, label: string, value: string, current: number, previous: number) => ({ key, label, value, change: t('admin.dailyReport.comparedWithPrevious') + ' ' + changeText(current, previous), changeClass: changeClass(current, previous) })
const headlineMetrics = computed(() => report.value ? [
  metric('users', t('admin.dailyReport.activeUsers'), formatNumber(report.value.summary.active_users), report.value.summary.active_users, report.value.previous_summary.active_users),
  metric('requests', t('admin.dailyReport.requests'), formatNumber(report.value.summary.total_requests), report.value.summary.total_requests, report.value.previous_summary.total_requests),
  metric('tokens', t('admin.dailyReport.tokens'), formatNumber(report.value.summary.total_tokens), report.value.summary.total_tokens, report.value.previous_summary.total_tokens),
  metric('cost', t('admin.dailyReport.standardCost'), '$' + formatCost(report.value.summary.total_cost), report.value.summary.total_cost, report.value.previous_summary.total_cost),
  metric('actual', t('admin.dailyReport.actualCost'), '$' + formatCost(report.value.summary.total_actual_cost), report.value.summary.total_actual_cost, report.value.previous_summary.total_actual_cost),
  metric('account', t('admin.dailyReport.accountCost'), '$' + formatCost(report.value.summary.total_account_cost), report.value.summary.total_account_cost, report.value.previous_summary.total_account_cost)
] : [])
const trendPoints = computed(() => {
  if (!report.value) return []
  const byDate = new Map(report.value.trend.map(item => [item.date, item]))
  const end = new Date(report.value.date + 'T12:00:00')
  const points: DailyReportResponse['trend'] = []
  for (let offset = 6; offset >= 0; offset--) {
    const date = new Date(end)
    date.setDate(date.getDate() - offset)
    const key = toDateString(date)
    points.push(byDate.get(key) || { date: key, active_users: 0, requests: 0, actual_cost: 0 })
  }
  return points
})
const trendMax = computed(() => Math.max(0, ...trendPoints.value.map(item => item.actual_cost)))
const trendHeight = (value: number) => (trendMax.value > 0 ? Math.max(4, value / trendMax.value * 120) : 4) + 'px'

async function loadReport() {
  loading.value = true
  try { report.value = await adminUsageAPI.getDailyReport(selectedDate.value) }
  catch { appStore.showError(t('admin.dailyReport.loadFailed')) }
  finally { loading.value = false }
}
function moveDate(days: number) {
  const date = new Date(selectedDate.value + 'T12:00:00')
  date.setDate(date.getDate() + days)
  selectedDate.value = toDateString(date)
  loadReport()
}
function toggleGroup(id: number) {
  const next = new Set(expandedGroups.value)
  next.has(id) ? next.delete(id) : next.add(id)
  expandedGroups.value = next
}
function displayGroupName(group: DailyReportGroupStat) {
  if (group.group_id === 0) return t('admin.dailyReport.ungrouped')
  return group.group_name.startsWith('Deleted group #') ? t('admin.dailyReport.deletedGroup') + ' #' + group.group_id : group.group_name
}
function exportCSV() {
  if (!report.value) return
  const rows = [['group', 'multiplier', 'users', 'requests', 'tokens', 'standard_cost', 'actual_cost', 'account_cost']]
  report.value.groups.forEach(group => group.multipliers.forEach(item => rows.push([displayGroupName(group), String(item.rate_multiplier), String(item.active_users), String(item.requests), String(item.total_tokens), String(item.cost), String(item.actual_cost), String(item.account_cost)])))
  const csv = '\uFEFF' + rows.map(row => row.map(value => '"' + value.replace(/"/g, '""') + '"').join(',')).join('\n')
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = 'daily-report-' + report.value.date + '.csv'
  link.click()
  URL.revokeObjectURL(url)
  appStore.showSuccess(t('admin.dailyReport.exportSuccess'))
}
onMounted(loadReport)
</script>
