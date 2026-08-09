<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <header class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 sm:flex-row sm:items-end sm:justify-between">
        <div class="min-w-0">
          <h1 class="text-2xl font-semibold text-gray-950 dark:text-white">
            {{ t('admin.accountContributions.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">
            {{ t('admin.accountContributions.description') }}
          </p>
        </div>
        <button
          type="button"
          class="btn btn-secondary inline-flex h-9 w-9 flex-shrink-0 items-center justify-center p-0"
          :disabled="loading"
          :title="t('admin.accountContributions.refresh')"
          :aria-label="t('admin.accountContributions.refresh')"
          @click="loadOverview"
        >
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
        </button>
      </header>

      <div
        v-if="overview"
        class="border-l-4 px-4 py-3"
        :class="overview.features.enabled
          ? 'border-emerald-500 bg-emerald-50 dark:bg-emerald-950/20'
          : 'border-amber-500 bg-amber-50 dark:bg-amber-950/20'"
      >
        <div class="flex flex-wrap items-center gap-x-5 gap-y-2 text-sm">
          <span class="font-medium text-gray-900 dark:text-white">
            {{ t('admin.accountContributions.readOnly') }}
          </span>
          <FeatureState
            :label="t('admin.accountContributions.featureStatus.main')"
            :enabled="overview.features.enabled"
          />
          <FeatureState
            :label="t('admin.accountContributions.featureStatus.submission')"
            :enabled="overview.features.submission_enabled"
            :configured="overview.features.submission_configured"
          />
          <FeatureState
            :label="t('admin.accountContributions.featureStatus.payout')"
            :enabled="overview.features.payout_enabled"
            :configured="overview.features.payout_configured"
          />
        </div>
      </div>

      <div
        v-if="loadError"
        class="border-l-4 border-red-500 bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-950/20 dark:text-red-200"
      >
        {{ loadError }}
      </div>

      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatBlock
          :label="t('admin.accountContributions.stats.contributors')"
          :value="formatInteger(stats.contributors_total)"
          :detail="t('admin.accountContributions.stats.pendingContributors', { count: formatInteger(stats.contributors_pending) })"
        />
        <StatBlock
          :label="t('admin.accountContributions.stats.accounts')"
          :value="formatInteger(stats.contributions_total)"
          :detail="t('admin.accountContributions.stats.activeAccounts', { count: formatInteger(stats.contributions_active) })"
        />
        <StatBlock
          :label="t('admin.accountContributions.stats.earnings')"
          :value="formatFen(stats.total_earnings_cny_fen)"
          :detail="t('admin.accountContributions.stats.availableEarnings', { amount: formatFen(stats.available_earnings_cny_fen) })"
        />
        <StatBlock
          :label="t('admin.accountContributions.stats.payouts')"
          :value="formatInteger(stats.payout_requests_total)"
          :detail="t('admin.accountContributions.stats.pendingPayouts', { count: formatInteger(stats.payout_requests_pending), amount: formatFen(stats.pending_payout_cny_fen) })"
        />
      </div>

      <section class="overflow-hidden border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
        <div class="overflow-x-auto border-b border-gray-200 px-3 dark:border-dark-700">
          <div class="flex min-w-max gap-1 py-2" role="tablist">
            <button
              v-for="tab in tabs"
              :key="tab.key"
              type="button"
              role="tab"
              :aria-selected="activeTab === tab.key"
              :class="tabClass(tab.key)"
              @click="activeTab = tab.key"
            >
              {{ tab.label }}
              <span class="text-xs tabular-nums text-gray-400 dark:text-dark-400">{{ tab.count }}</span>
            </button>
          </div>
        </div>

        <DataTable :columns="columns" :data="activeRows" :loading="loading && !overview">
          <template #cell-id="{ row }">
            <span class="font-mono text-xs text-gray-500 dark:text-dark-400">#{{ row.id }}</span>
          </template>
          <template #cell-owner="{ row }">
            <div class="min-w-0">
              <div class="max-w-64 truncate text-sm font-medium text-gray-900 dark:text-white">{{ row.owner || '-' }}</div>
              <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ row.ownerMeta || '-' }}</div>
            </div>
          </template>
          <template #cell-detail="{ row }">
            <div class="min-w-0">
              <div class="max-w-72 truncate text-sm text-gray-800 dark:text-gray-200">{{ row.detail || '-' }}</div>
              <div v-if="row.detailMeta" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ row.detailMeta }}</div>
            </div>
          </template>
          <template #cell-status="{ row }">
            <span class="inline-flex items-center gap-1.5 text-sm text-gray-700 dark:text-gray-300">
              <span class="h-2 w-2 rounded-full" :class="statusDotClass(row.status)"></span>
              {{ statusLabel(row.status) }}
            </span>
          </template>
          <template #cell-amount="{ row }">
            <span class="text-sm font-medium tabular-nums text-gray-900 dark:text-white">
              {{ row.amountFen == null ? '-' : formatFen(row.amountFen) }}
            </span>
          </template>
          <template #cell-time="{ row }">
            <span class="whitespace-nowrap text-sm text-gray-600 dark:text-dark-300">{{ formatDate(row.time) }}</span>
          </template>
          <template #empty>
            <div class="flex flex-col items-center py-10 text-center">
              <Icon name="inbox" size="xl" class="mb-3 text-gray-400 dark:text-dark-500" />
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ emptyLabel }}</p>
            </div>
          </template>
        </DataTable>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { accountContributionsAPI, type AccountContributionAdminOverview, type AccountContributionAdminStats } from '@/api/admin/accountContributions'
import { extractApiErrorMessage } from '@/utils/apiError'

type TabKey = 'contributors' | 'contributions' | 'earnings' | 'payouts'

interface ContributionTableRow {
  id: number
  owner: string
  ownerMeta: string
  detail: string
  detailMeta: string
  status: string
  amountFen: number | null
  time: string
}

const emptyStats: AccountContributionAdminStats = {
  contributors_total: 0,
  contributors_pending: 0,
  contributions_total: 0,
  contributions_active: 0,
  earning_entries_total: 0,
  total_earnings_cny_fen: 0,
  available_earnings_cny_fen: 0,
  payout_requests_total: 0,
  payout_requests_pending: 0,
  pending_payout_cny_fen: 0,
}

const { t, locale } = useI18n()
const loading = ref(false)
const loadError = ref('')
const overview = ref<AccountContributionAdminOverview | null>(null)
const activeTab = ref<TabKey>('contributors')

const stats = computed(() => overview.value?.stats ?? emptyStats)
const columns = computed<Column[]>(() => [
  { key: 'id', label: t('admin.accountContributions.columns.id') },
  { key: 'owner', label: t('admin.accountContributions.columns.owner') },
  { key: 'detail', label: t('admin.accountContributions.columns.detail') },
  { key: 'status', label: t('admin.accountContributions.columns.status') },
  { key: 'amount', label: t('admin.accountContributions.columns.amount') },
  { key: 'time', label: t('admin.accountContributions.columns.time') },
])

const tabs = computed(() => [
  { key: 'contributors' as const, label: t('admin.accountContributions.tabs.contributors'), count: stats.value.contributors_total },
  { key: 'contributions' as const, label: t('admin.accountContributions.tabs.contributions'), count: stats.value.contributions_total },
  { key: 'earnings' as const, label: t('admin.accountContributions.tabs.earnings'), count: stats.value.earning_entries_total },
  { key: 'payouts' as const, label: t('admin.accountContributions.tabs.payouts'), count: stats.value.payout_requests_total },
])

const activeRows = computed<ContributionTableRow[]>(() => {
  if (!overview.value) return []
  if (activeTab.value === 'contributors') {
    return overview.value.contributors.map(item => ({
      id: item.id,
      owner: item.email || item.username,
      ownerMeta: item.user_id ? t('admin.accountContributions.details.userId', { id: item.user_id }) : item.username,
      detail: t('admin.accountContributions.details.contributionCount', { count: item.contributions }),
      detailMeta: '',
      status: item.status,
      amountFen: null,
      time: item.created_at,
    }))
  }
  if (activeTab.value === 'contributions') {
    return overview.value.contributions.map(item => ({
      id: item.id,
      owner: item.contributor,
      ownerMeta: `#${item.contributor_id}`,
      detail: item.account_name || item.platform,
      detailMeta: [
        item.account_id ? t('admin.accountContributions.details.accountId', { id: item.account_id }) : '',
        t(`admin.accountContributions.settlementModes.${item.settlement_mode}`, item.settlement_mode),
        t('admin.accountContributions.details.shareRate', { rate: item.share_rate_bps / 100 }),
      ].filter(Boolean).join(' · '),
      status: item.status,
      amountFen: null,
      time: item.created_at,
    }))
  }
  if (activeTab.value === 'earnings') {
    return overview.value.earnings.map(item => ({
      id: item.id,
      owner: item.contributor,
      ownerMeta: `#${item.contributor_id}`,
      detail: item.account_name || `#${item.contribution_id}`,
      detailMeta: t('admin.accountContributions.details.availableAt', { time: formatDate(item.available_at) }),
      status: item.entry_type,
      amountFen: item.amount_cny_fen,
      time: item.created_at,
    }))
  }
  return overview.value.payouts.map(item => ({
    id: item.id,
    owner: item.contributor,
    ownerMeta: `#${item.contributor_id}`,
    detail: item.method_type
      ? t('admin.accountContributions.details.payoutMethod', { method: item.method_type, destination: item.masked_destination || '-' })
      : '-',
    detailMeta: '',
    status: item.status,
    amountFen: item.amount_cny_fen,
    time: item.requested_at,
  }))
})

const emptyLabel = computed(() => t(`admin.accountContributions.empty.${activeTab.value}`))

const FeatureState = defineComponent({
  props: {
    label: { type: String, required: true },
    enabled: { type: Boolean, required: true },
    configured: { type: Boolean, default: false },
  },
  setup(props) {
    return () => h('span', { class: 'inline-flex items-center gap-1.5 text-gray-700 dark:text-gray-300' }, [
      h('span', { class: ['h-2 w-2 rounded-full', props.enabled ? 'bg-emerald-500' : props.configured ? 'bg-amber-500' : 'bg-gray-400'] }),
      `${props.label}: ${props.enabled
        ? t('admin.accountContributions.featureStatus.enabled')
        : props.configured
          ? t('admin.accountContributions.featureStatus.configured')
          : t('admin.accountContributions.featureStatus.disabled')}`,
    ])
  },
})

const StatBlock = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
    detail: { type: String, required: true },
  },
  setup(props) {
    return () => h('div', { class: 'min-w-0 border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-900' }, [
      h('div', { class: 'text-xs font-medium text-gray-500 dark:text-dark-400' }, props.label),
      h('div', { class: 'mt-1 truncate text-xl font-semibold tabular-nums text-gray-950 dark:text-white' }, props.value),
      h('div', { class: 'mt-1 truncate text-xs text-gray-500 dark:text-dark-400', title: props.detail }, props.detail),
    ])
  },
})

function tabClass(key: TabKey): string[] {
  return [
    'inline-flex h-9 items-center gap-2 border-b-2 px-3 text-sm font-medium transition-colors',
    activeTab.value === key
      ? 'border-primary-500 text-primary-700 dark:text-primary-300'
      : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-dark-400 dark:hover:text-gray-200',
  ]
}

function statusLabel(status: string): string {
  return t(`admin.accountContributions.statuses.${status}`, status)
}

function statusDotClass(status: string): string {
  if (['active', 'accrual', 'paid', 'approved'].includes(status)) return 'bg-emerald-500'
  if (['pending', 'pending_review', 'testing', 'requested', 'reviewing', 'processing'].includes(status)) return 'bg-amber-500'
  if (['rejected', 'revoked', 'failed', 'quarantined'].includes(status)) return 'bg-red-500'
  return 'bg-gray-400'
}

function formatInteger(value: number): string {
  return new Intl.NumberFormat(locale.value).format(value || 0)
}

function formatFen(value: number): string {
  return new Intl.NumberFormat(locale.value, {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
  }).format((value || 0) / 100)
}

function formatDate(value: string): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString(locale.value)
}

async function loadOverview() {
  loading.value = true
  loadError.value = ''
  try {
    overview.value = await accountContributionsAPI.getOverview()
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, t('admin.accountContributions.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(loadOverview)
</script>
