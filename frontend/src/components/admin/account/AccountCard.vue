<template>
  <article
    :class="[
      'account-card border bg-white shadow-sm dark:bg-dark-800',
      selected
        ? 'border-primary-400 ring-1 ring-primary-300 dark:border-primary-500 dark:ring-primary-700'
        : 'border-gray-200 dark:border-dark-700'
    ]"
  >
    <header data-swipe-select-content class="flex flex-col gap-3 p-4 sm:flex-row sm:items-start">
      <input
        type="checkbox"
        class="mt-1 h-4 w-4 flex-shrink-0 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        :checked="selected"
        :aria-label="t('admin.accounts.selectAccount', { name: account.name })"
        @click.stop
        @change="emit('toggle-select')"
      />

      <div class="min-w-0 flex-1">
        <div class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1.5">
          <h2 class="min-w-0 break-words text-sm font-semibold text-gray-900 dark:text-white">
            {{ account.name }}
          </h2>
          <span v-if="isFieldVisible('id')" class="font-mono text-xs text-gray-400 dark:text-dark-400">
            #{{ account.id }}
          </span>
          <template v-if="isFieldVisible('platform_type')">
            <PlatformTypeBadge
              :platform="account.platform"
              :type="account.type"
              :auth-mode="getOpenAIAuthMode(account)"
              :plan-type="getAccountPlanType(account)"
              :privacy-mode="getPrivacyMode(account)"
              :subscription-expires-at="getSubscriptionExpiresAt(account)"
            />
            <span
              v-if="getAntigravityTierLabel(account)"
              :class="['inline-block rounded px-1.5 py-0.5 text-[10px] font-medium', getAntigravityTierClass(account)]"
            >
              {{ getAntigravityTierLabel(account) }}
            </span>
          </template>
        </div>
        <div class="mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
          <span
            v-if="accountDisplayEmail(account)"
            class="min-w-0 break-all text-xs text-gray-500 dark:text-gray-400"
            :title="accountDisplayTitle(account)"
          >
            {{ accountDisplayEmail(account) }}
          </span>
          <span
            v-if="isFieldVisible('platform_type') && getOpenAICompactMeta(account)"
            :class="[
              'inline-flex items-center gap-1.5 text-[11px] font-medium leading-4',
              getOpenAICompactMeta(account)?.className
            ]"
            :title="getOpenAICompactTitle(account)"
          >
            <span :class="['h-1.5 w-1.5 rounded-full', getOpenAICompactMeta(account)?.dotClass]" />
            <span>{{ getOpenAICompactMeta(account)?.label }}</span>
          </span>
        </div>
      </div>

      <div class="flex flex-shrink-0 items-center self-end sm:self-start">
        <button
          type="button"
          class="account-card-icon-button hover:text-primary-600 dark:hover:text-primary-400"
          :title="t('common.edit')"
          :aria-label="t('common.edit')"
          @click.stop="emit('edit')"
        >
          <Icon name="edit" size="sm" />
        </button>
        <button
          type="button"
          class="account-card-icon-button hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
          :title="t('common.delete')"
          :aria-label="t('common.delete')"
          @click.stop="emit('delete')"
        >
          <Icon name="trash" size="sm" />
        </button>
        <button
          type="button"
          class="account-card-icon-button hover:text-gray-900 dark:hover:text-white"
          :title="t('common.more')"
          :aria-label="t('common.more')"
          @click.stop="emit('more', $event)"
        >
          <Icon name="more" size="sm" />
        </button>
      </div>
    </header>

    <div
      v-if="isFieldVisible('status') || isFieldVisible('schedulable')"
      data-swipe-select-content
      class="flex min-w-0 flex-wrap items-center justify-between gap-3 border-t border-gray-100 bg-gray-50/70 px-4 py-2.5 dark:border-dark-700 dark:bg-dark-900/30"
    >
      <div v-if="isFieldVisible('status')" class="min-w-0">
        <AccountStatusIndicator
          class="account-card-status max-w-full flex-wrap"
          :account="account"
          @show-temp-unsched="emit('show-temp-unsched')"
        />
      </div>
      <div v-if="isFieldVisible('schedulable')" class="ml-auto flex items-center gap-2">
        <span class="text-xs font-medium text-gray-500 dark:text-dark-400">
          {{ t('admin.accounts.columns.schedulable') }}
        </span>
        <button
          type="button"
          :disabled="togglingSchedulable"
          :class="[
            'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-dark-800',
            account.schedulable
              ? 'bg-primary-500 hover:bg-primary-600'
              : 'bg-gray-200 hover:bg-gray-300 dark:bg-dark-600 dark:hover:bg-dark-500'
          ]"
          :title="account.schedulable ? t('admin.accounts.schedulableEnabled') : t('admin.accounts.schedulableDisabled')"
          :aria-pressed="account.schedulable"
          @click.stop="emit('toggle-schedulable')"
        >
          <span
            :class="[
              'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
              account.schedulable ? 'translate-x-4' : 'translate-x-0'
            ]"
          />
        </button>
      </div>
    </div>

    <div
      v-if="hasVisibleCardSections"
      class="grid gap-px border-t border-gray-100 bg-gray-100 sm:grid-cols-2 xl:grid-cols-4 dark:border-dark-700 dark:bg-dark-700"
    >
      <section v-if="isFieldVisible('capacity') || isFieldVisible('today_stats')" data-swipe-select-content class="account-card-section">
        <h3 class="account-card-section-title">{{ t('admin.accounts.cardSections.activity') }}</h3>
        <div
          :class="[
            'grid gap-4',
            isFieldVisible('capacity') && isFieldVisible('today_stats') ? 'grid-cols-2' : 'grid-cols-1'
          ]"
        >
          <div v-if="isFieldVisible('capacity')" class="min-w-0">
            <div class="account-card-field-label">{{ t('admin.accounts.columns.capacity') }}</div>
            <AccountCapacityCell :account="account" />
          </div>
          <div v-if="isFieldVisible('today_stats')" class="min-w-0">
            <div class="account-card-field-label">{{ t('admin.accounts.columns.todayStats') }}</div>
            <AccountTodayStatsCell
              :stats="todayStats"
              :loading="todayStatsLoading"
              :error="todayStatsError"
            />
          </div>
        </div>
      </section>

      <section v-if="isFieldVisible('usage')" data-swipe-select-content class="account-card-section">
        <div data-test="usage-header" class="mb-3 flex items-center">
          <h3 class="account-card-section-title mb-0">{{ t('admin.accounts.columns.usageWindows') }}</h3>
          <HelpTooltip :content="t('admin.accounts.usageWindowsHint')" width-class="w-72" />
        </div>
        <AccountUsageCell
          :account="account"
          :today-stats="todayStats"
          :today-stats-loading="todayStatsLoading"
          :manual-refresh-token="usageManualRefreshToken"
        />
      </section>

      <section v-if="hasVisibleRoutingFields" data-swipe-select-content class="account-card-section">
        <h3 class="account-card-section-title">{{ t('admin.accounts.cardSections.routing') }}</h3>
        <dl class="space-y-3">
          <div v-if="!simpleMode && isFieldVisible('groups')" class="account-card-detail-row">
            <dt class="account-card-field-label">{{ t('admin.accounts.columns.groups') }}</dt>
            <dd class="min-w-0"><AccountGroupsCell :groups="account.groups" :max-display="4" /></dd>
          </div>
          <div v-if="isFieldVisible('proxy')" class="account-card-detail-row">
            <dt class="account-card-field-label">{{ t('admin.accounts.columns.proxy') }}</dt>
            <dd class="min-w-0 text-sm text-gray-700 dark:text-gray-300">
              <div v-if="account.proxy" class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                <span class="break-words">{{ account.proxy.name }}</span>
                <span v-if="account.proxy.country_code" class="text-xs text-gray-500 dark:text-gray-400">
                  ({{ account.proxy.country_code }})
                </span>
              </div>
              <span v-else class="text-gray-400 dark:text-dark-500">-</span>
              <div v-if="account.proxy?.expires_at" class="mt-1 flex flex-wrap items-center gap-2 text-xs">
                <span>{{ formatDateTime(account.proxy.expires_at) }}</span>
                <span :class="proxyExpiryBadge(account.proxy)">{{ proxyExpiryText(account.proxy) }}</span>
              </div>
              <div v-if="account.proxy_fallback_origin_id" class="mt-1 flex flex-wrap items-center gap-1">
                <span
                  class="inline-flex items-center rounded bg-yellow-100 px-1.5 py-0.5 text-xs font-medium text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
                  :title="t('admin.accounts.fallbackActiveTip', { origin: account.proxy_fallback_origin_name })"
                >
                  {{ t('admin.accounts.fallbackActive') }}
                </span>
                <button
                  type="button"
                  class="rounded border border-gray-300 px-1.5 py-0.5 text-xs text-gray-600 hover:bg-gray-100 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                  @click.stop="emit('revert-fallback')"
                >
                  {{ t('admin.accounts.revertProxy') }}
                </button>
              </div>
            </dd>
          </div>
          <div v-if="isFieldVisible('priority')" class="account-card-detail-row">
            <dt class="account-card-field-label">{{ t('admin.accounts.columns.priority') }}</dt>
            <dd class="text-sm text-gray-700 dark:text-gray-300">{{ account.priority }}</dd>
          </div>
          <div v-if="isFieldVisible('rate_multiplier')" class="account-card-detail-row">
            <dt class="account-card-field-label">{{ t('admin.accounts.columns.billingRateMultiplier') }}</dt>
            <dd class="font-mono text-sm text-gray-700 dark:text-gray-300">
              {{ (account.rate_multiplier ?? 1).toFixed(2) }}x
            </dd>
          </div>
          <div v-if="isFieldVisible('scheduler_score')" class="account-card-detail-row">
            <dt class="flex items-center">
              <span class="account-card-field-label">{{ t('admin.accounts.columns.schedulerScore') }}</span>
              <HelpTooltip :content="t('admin.accounts.schedulerScore.hint')" width-class="w-80" />
            </dt>
            <dd :data-test="`scheduler-score-${account.id}`" class="min-w-0">
              <div v-if="getSchedulerScoreRows(account).length" class="flex min-w-0 flex-col gap-0.5 font-mono text-[11px] leading-4">
                <div
                  v-for="score in getSchedulerScoreRows(account)"
                  :key="String(score.group_id)"
                  class="flex min-w-0 items-center gap-1 text-gray-700 dark:text-gray-300"
                  :title="`${formatSchedulerScoreGroup(score)} / ${formatSchedulerScore(score.base_score)} / ${formatStickySchedulerScore(score)}`"
                >
                  <span class="max-w-20 truncate text-gray-500 dark:text-dark-400">{{ formatSchedulerScoreGroup(score) }}</span>
                  <span class="text-gray-300 dark:text-gray-600">/</span>
                  <span>{{ formatSchedulerScore(score.base_score) }}</span>
                  <span class="text-gray-300 dark:text-gray-600">/</span>
                  <span class="text-primary-700 dark:text-primary-300">{{ formatStickySchedulerScore(score) }}</span>
                </div>
              </div>
              <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
            </dd>
          </div>
        </dl>
      </section>

      <section v-if="hasVisibleDetailFields" data-swipe-select-content class="account-card-section">
        <h3 class="account-card-section-title">{{ t('admin.accounts.cardSections.details') }}</h3>
        <dl class="space-y-3">
          <div v-if="isFieldVisible('last_used_at')" class="account-card-detail-row">
            <dt class="account-card-field-label">{{ t('admin.accounts.columns.lastUsed') }}</dt>
            <dd class="text-sm text-gray-600 dark:text-dark-300">{{ formatRelativeTime(account.last_used_at) }}</dd>
          </div>
          <div v-if="isFieldVisible('created_at')" class="account-card-detail-row">
            <dt class="account-card-field-label">{{ t('admin.accounts.columns.createdAt') }}</dt>
            <dd class="text-sm text-gray-600 dark:text-dark-300">{{ formatDateTime(account.created_at) }}</dd>
          </div>
          <div v-if="isFieldVisible('expires_at')" class="account-card-detail-row">
            <dt class="account-card-field-label">{{ t('admin.accounts.columns.expiresAt') }}</dt>
            <dd class="min-w-0 text-sm text-gray-600 dark:text-dark-300">
              <div>{{ formatExpiresAt(account.expires_at) }}</div>
              <div v-if="isExpired(account.expires_at) || (account.auto_pause_on_expired && account.expires_at)" class="mt-1 flex flex-wrap items-center gap-1">
                <span
                  v-if="isExpired(account.expires_at)"
                  class="inline-flex items-center rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
                >
                  {{ t('admin.accounts.expired') }}
                </span>
                <span
                  v-if="account.auto_pause_on_expired && account.expires_at"
                  class="inline-flex items-center rounded-md bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
                >
                  {{ t('admin.accounts.autoPauseOnExpired') }}
                </span>
              </div>
            </dd>
          </div>
          <div v-if="isFieldVisible('notes')" class="account-card-detail-row">
            <dt class="account-card-field-label">{{ t('admin.accounts.columns.notes') }}</dt>
            <dd v-if="account.notes" class="min-w-0 whitespace-pre-wrap break-words text-sm text-gray-600 dark:text-gray-300">
              {{ account.notes }}
            </dd>
            <dd v-else class="text-sm text-gray-400 dark:text-dark-500">-</dd>
          </div>
        </dl>
      </section>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AccountCapacityCell from '@/components/account/AccountCapacityCell.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import AccountTodayStatsCell from '@/components/account/AccountTodayStatsCell.vue'
import AccountUsageCell from '@/components/account/AccountUsageCell.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import { proxyExpiryBadgeClass, proxyExpiryLabelKey } from '@/utils/proxyExpiry'
import type { Account, AccountSchedulerGroupScore, Proxy as AccountProxy, WindowStats } from '@/types'

const props = withDefaults(defineProps<{
  account: Account
  selected?: boolean
  hiddenFields?: string[]
  simpleMode?: boolean
  togglingSchedulable?: boolean
  todayStats?: WindowStats | null
  todayStatsLoading?: boolean
  todayStatsError?: string | null
  usageManualRefreshToken?: number
}>(), {
  selected: false,
  hiddenFields: () => [],
  simpleMode: false,
  togglingSchedulable: false,
  todayStats: null,
  todayStatsLoading: false,
  todayStatsError: null,
  usageManualRefreshToken: 0
})

const emit = defineEmits<{
  (event: 'toggle-select'): void
  (event: 'edit'): void
  (event: 'delete'): void
  (event: 'more', mouseEvent: MouseEvent): void
  (event: 'toggle-schedulable'): void
  (event: 'show-temp-unsched'): void
  (event: 'revert-fallback'): void
}>()

const { t } = useI18n()
const hiddenFieldSet = computed(() => new Set(props.hiddenFields))
const isFieldVisible = (key: string) => !hiddenFieldSet.value.has(key)

const hasVisibleRoutingFields = computed(() =>
  (!props.simpleMode && isFieldVisible('groups')) ||
  ['proxy', 'priority', 'scheduler_score', 'rate_multiplier'].some(isFieldVisible)
)
const hasVisibleDetailFields = computed(() =>
  ['last_used_at', 'created_at', 'expires_at', 'notes'].some(isFieldVisible)
)
const hasVisibleCardSections = computed(() =>
  isFieldVisible('capacity') ||
  isFieldVisible('today_stats') ||
  isFieldVisible('usage') ||
  hasVisibleRoutingFields.value ||
  hasVisibleDetailFields.value
)

function getAccountPlanType(row: any): string | undefined {
  if (row.platform === 'grok') {
    const extra = (row.extra || {}) as Record<string, any>
    const billing = extra.grok_billing_snapshot as Record<string, any> | undefined
    const quota = extra.grok_quota_snapshot as Record<string, any> | undefined
    return billing?.plan || quota?.subscription_tier || row.credentials?.subscription_tier ||
      extra.subscription_tier || row.credentials?.plan_type || row.parent_plan_type || undefined
  }
  return row.credentials?.plan_type || row.parent_plan_type || undefined
}

function getOpenAIAuthMode(row: any): string | undefined {
  if (row.platform !== 'openai' || row.type !== 'oauth') return undefined
  const authMode = row.credentials?.auth_mode
  return typeof authMode === 'string' && authMode.trim() ? authMode : undefined
}

function getPrivacyMode(row: any): string | undefined {
  const mode = row.extra?.privacy_mode || row.parent_privacy_mode
  return typeof mode === 'string' ? mode : undefined
}

function getSubscriptionExpiresAt(row: any): string | undefined {
  const expiresAt = row.credentials?.subscription_expires_at || row.parent_subscription_expires_at
  return typeof expiresAt === 'string' ? expiresAt : undefined
}

function getAntigravityTierFromRow(row: any): string | null {
  if (row.platform !== 'antigravity') return null
  const lca = row.extra?.load_code_assist as Record<string, unknown> | undefined
  const paid = lca?.paidTier as Record<string, unknown> | undefined
  if (paid && typeof paid.id === 'string') return paid.id
  const current = lca?.currentTier as Record<string, unknown> | undefined
  return current && typeof current.id === 'string' ? current.id : null
}

function getAntigravityTierLabel(row: any): string | null {
  switch (getAntigravityTierFromRow(row)) {
    case 'free-tier': return t('admin.accounts.tier.free')
    case 'g1-pro-tier': return t('admin.accounts.tier.pro')
    case 'g1-ultra-tier': return t('admin.accounts.tier.ultra')
    default: return null
  }
}

function getAntigravityTierClass(row: any): string {
  switch (getAntigravityTierFromRow(row)) {
    case 'free-tier': return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
    case 'g1-pro-tier': return 'bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300'
    case 'g1-ultra-tier': return 'bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300'
    default: return ''
  }
}

const accountDisplayEmail = (row: any): string =>
  row.extra?.email_address || row.extra?.email || row.credentials?.email || row.parent_email || ''

const accountDisplayTitle = (row: any): string => {
  const email = accountDisplayEmail(row)
  return row.parent_chatgpt_account_id ? `${email} / ${row.parent_chatgpt_account_id}` : email
}

type OpenAICompactBadgeState = 'active' | 'blocked' | 'auto'

function getOpenAICompactState(row: any): OpenAICompactBadgeState | null {
  if (row.platform !== 'openai' || (row.type !== 'oauth' && row.type !== 'apikey')) return null
  const extra = row.extra as Record<string, unknown> | undefined
  const mode = typeof extra?.openai_compact_mode === 'string' ? extra.openai_compact_mode : 'auto'
  if (mode === 'force_on') return 'active'
  if (mode === 'force_off') return 'blocked'
  if (typeof extra?.openai_compact_supported === 'boolean') {
    return extra.openai_compact_supported ? 'active' : 'blocked'
  }
  return 'auto'
}

function getOpenAICompactMeta(row: any): { label: string; className: string; dotClass: string } | null {
  switch (getOpenAICompactState(row)) {
    case 'active':
      return {
        label: t('admin.accounts.openai.compactSupported'),
        className: 'text-emerald-600 dark:text-emerald-300',
        dotClass: 'bg-emerald-500 shadow-[0_0_0_2px_rgba(16,185,129,0.14)]'
      }
    case 'blocked':
      return {
        label: t('admin.accounts.openai.compactUnsupported'),
        className: 'text-rose-600 dark:text-rose-300',
        dotClass: 'bg-rose-500 shadow-[0_0_0_2px_rgba(244,63,94,0.14)]'
      }
    case 'auto':
      return {
        label: t('admin.accounts.openai.compactAuto'),
        className: 'text-slate-500 dark:text-slate-400',
        dotClass: 'bg-slate-300 dark:bg-slate-500'
      }
    default:
      return null
  }
}

function getOpenAICompactTitle(row: any): string {
  const checkedAt = typeof row.extra?.openai_compact_checked_at === 'string'
    ? row.extra.openai_compact_checked_at
    : ''
  const label = getOpenAICompactMeta(row)?.label || ''
  return checkedAt
    ? `${label} | ${t('admin.accounts.openai.compactLastChecked')}: ${formatDateTime(new Date(checkedAt))}`
    : label
}

const formatSchedulerScore = (value: unknown): string => {
  const number = Number(value)
  return Number.isFinite(number) ? number.toFixed(6).replace(/\.?0+$/, '') : '-'
}

const formatStickySchedulerScore = (score: AccountSchedulerGroupScore): string =>
  score.sticky_score_infinity ? '+\u221e' : formatSchedulerScore(score.sticky_score)

const getSchedulerScoreRows = (account: Account): AccountSchedulerGroupScore[] => {
  const groupRows = Array.isArray(account.scheduler_scores)
    ? account.scheduler_scores.filter(score => score.group_id != null)
    : []
  if (groupRows.length) return groupRows
  return account.scheduler_score ? [{ group_id: null, ...account.scheduler_score }] : []
}

const formatSchedulerScoreGroup = (score: AccountSchedulerGroupScore): string => {
  if (score.group_name) return score.group_name
  if (score.group_id != null) return `#${score.group_id}`
  return t('admin.accounts.schedulerScore.ungrouped')
}

const formatExpiresAt = (value: number | null) => {
  if (!value) return '-'
  return formatDateTime(
    new Date(value * 1000),
    { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false },
    'sv-SE'
  )
}

const isExpired = (value: number | null) => !!value && value * 1000 <= Date.now()
const proxyExpiryBadge = (proxy: AccountProxy): string => proxyExpiryBadgeClass(proxy.expires_at, proxy.status)
const proxyExpiryText = (proxy: AccountProxy): string => {
  const { key, params } = proxyExpiryLabelKey(proxy.expires_at, proxy.status)
  return params ? t(key, params) : t(key)
}
</script>

<style scoped>
.account-card {
  min-width: 0;
  border-radius: 8px;
}

.account-card-icon-button {
  @apply inline-flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-md text-gray-500 transition-colors hover:bg-gray-100 dark:text-dark-300 dark:hover:bg-dark-700;
}

.account-card-section {
  @apply min-w-0 bg-white p-4 dark:bg-dark-800;
}

.account-card-section-title {
  @apply mb-3 text-xs font-semibold uppercase text-gray-500 dark:text-dark-400;
  letter-spacing: 0;
}

.account-card-field-label {
  @apply mb-1 text-xs font-medium text-gray-500 dark:text-dark-400;
}

.account-card-detail-row {
  display: grid;
  grid-template-columns: minmax(4.5rem, 5.5rem) minmax(0, 1fr);
  column-gap: 0.5rem;
  align-items: start;
}

@media (max-width: 374px) {
  .account-card-detail-row {
    grid-template-columns: minmax(4rem, 4.5rem) minmax(0, 1fr);
  }
}

@media (max-width: 639px) {
  .account-card-status :deep(.columns-2),
  .account-card-status :deep(.columns-3) {
    columns: 1;
  }
}
</style>
