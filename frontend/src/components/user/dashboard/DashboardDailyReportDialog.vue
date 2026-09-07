<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="wide"
    @close="emit('close')"
  >
    <div v-if="loading" class="daily-report-theme daily-report-loading">
      <LoadingSpinner size="lg" />
      <p>{{ t('dashboard.dailyReport.loading') }}</p>
    </div>

    <div v-else-if="errorMessage" class="daily-report-theme daily-report-error" role="alert">
      <Icon name="exclamationCircle" size="lg" />
      <p>{{ errorMessage }}</p>
      <button type="button" class="btn btn-secondary" @click="loadReport">
        <Icon name="refresh" size="sm" />
        {{ t('dashboard.dailyReport.retry') }}
      </button>
    </div>

    <div v-else-if="report" class="daily-report-theme daily-report-content">
      <section class="daily-report-narrative">
        <div class="daily-report-narrative-icon" aria-hidden="true">
          <Icon name="sparkles" size="lg" />
        </div>
        <div class="daily-report-narrative-copy">
          <p>{{ report.narrative }}</p>
          <div class="daily-report-meta">
            <span v-if="comparisonText" class="daily-report-comparison">
              {{ comparisonText }}
            </span>
            <span>{{ formatGeneratedAt(report.generated_at) }}</span>
          </div>
        </div>
      </section>

      <dl class="daily-report-metrics">
        <div>
          <dt>{{ t('dashboard.dailyReport.totalTokens') }}</dt>
          <dd>{{ formatTokens(report.summary.total_tokens) }}</dd>
          <small>{{ t('dashboard.dailyReport.tokenMix', {
            input: formatTokens(report.summary.input_tokens),
            output: formatTokens(report.summary.output_tokens)
          }) }}</small>
        </div>
        <div>
          <dt>{{ t('dashboard.dailyReport.requests') }}</dt>
          <dd>{{ formatNumber(report.summary.requests) }}</dd>
          <small>{{ t('dashboard.dailyReport.averageTokens', {
            value: formatTokens(report.summary.average_tokens_per_request)
          }) }}</small>
        </div>
        <div>
          <dt>{{ t('dashboard.dailyReport.models') }}</dt>
          <dd>{{ formatNumber(report.summary.model_count) }}</dd>
          <small>{{ topModelCaption }}</small>
        </div>
        <div>
          <dt>{{ t('dashboard.dailyReport.cacheHitRate') }}</dt>
          <dd>{{ formatPercent(report.summary.cache_hit_rate) }}</dd>
          <small>{{ t('dashboard.dailyReport.cacheRead', {
            value: formatTokens(report.summary.cache_read_tokens)
          }) }}</small>
        </div>
      </dl>

      <section class="daily-report-models">
        <header>
          <div>
            <h4>{{ t('dashboard.dailyReport.modelLineup') }}</h4>
            <p>{{ t('dashboard.dailyReport.modelLineupCaption') }}</p>
          </div>
          <span>{{ t('dashboard.dailyReport.modelCount', { count: report.models.length }) }}</span>
        </header>

        <div v-if="report.models.length" class="daily-report-model-list" role="list">
          <article
            v-for="(model, index) in report.models"
            :key="model.model"
            class="daily-report-model-row"
            data-test="daily-report-model"
            role="listitem"
          >
            <div class="daily-report-model-heading">
              <span class="daily-report-model-rank">{{ index + 1 }}</span>
              <strong :title="model.model">{{ model.model }}</strong>
              <span class="daily-report-model-share">{{ formatPercent(model.share) }}</span>
            </div>
            <div class="daily-report-model-track" aria-hidden="true">
              <i :style="{ width: `${Math.max(2, Math.min(100, model.share))}%` }" />
            </div>
            <dl class="daily-report-model-stats">
              <div data-test="model-requests">
                <dt>{{ t('dashboard.dailyReport.modelRequests') }}</dt>
                <dd>{{ formatNumber(model.requests) }}</dd>
              </div>
              <div data-test="model-tokens">
                <dt>{{ t('dashboard.dailyReport.modelTokens') }}</dt>
                <dd>{{ formatTokens(model.total_tokens) }}</dd>
              </div>
            </dl>
          </article>
        </div>

        <div v-else class="daily-report-empty">
          <Icon name="moon" size="lg" />
          <span>{{ t('dashboard.dailyReport.noModels') }}</span>
        </div>
      </section>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { usageAPI, type UserDailyReport } from '@/api/usage'

const props = defineProps<{
  show: boolean
  date: string
  timezone: string
}>()

const emit = defineEmits<{
  (event: 'close'): void
}>()

const { t, locale } = useI18n()
const loading = ref(false)
const errorMessage = ref('')
const report = ref<UserDailyReport | null>(null)
let requestID = 0

const numberLocale = computed(() => (locale.value.startsWith('zh') ? 'zh-CN' : 'en-US'))

const dialogTitle = computed(() => t('dashboard.dailyReport.title', {
  date: formatReportDate(props.date)
}))

const comparisonText = computed(() => {
  const change = report.value?.comparison.token_change_pct
  if (change == null) return ''
  if (change >= 0) {
    return t('dashboard.dailyReport.moreThanYesterday', { value: Math.abs(change).toFixed(1) })
  }
  return t('dashboard.dailyReport.lessThanYesterday', { value: Math.abs(change).toFixed(1) })
})

const topModelCaption = computed(() => {
  const top = report.value?.models[0]
  if (!top) return t('dashboard.dailyReport.noTopModel')
  return t('dashboard.dailyReport.topModel', { model: top.model })
})

const formatReportDate = (dateValue: string): string => {
  const [year, month, day] = dateValue.split('-').map(Number)
  if (!year || !month || !day) return dateValue
  return new Intl.DateTimeFormat(numberLocale.value, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    timeZone: 'UTC'
  }).format(new Date(Date.UTC(year, month - 1, day)))
}

const formatNumber = (value: number): string =>
  new Intl.NumberFormat(numberLocale.value, { maximumFractionDigits: 0 }).format(value || 0)

const formatTokens = (value: number): string => {
  const absolute = Math.abs(value || 0)
  if (absolute >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(1)}B`
  if (absolute >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (absolute >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return formatNumber(value)
}

const formatPercent = (value: number): string => `${Math.max(0, value || 0).toFixed(1)}%`

const formatGeneratedAt = (value: string): string => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(numberLocale.value, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

const loadReport = async () => {
  if (!props.show || !props.date) return
  const currentRequest = ++requestID
  loading.value = true
  errorMessage.value = ''
  try {
    const result = await usageAPI.getDailyReport({
      date: props.date,
      timezone: props.timezone,
      locale: locale.value
    })
    if (currentRequest !== requestID) return
    report.value = result
  } catch (error) {
    if (currentRequest !== requestID) return
    console.error('Failed to load daily report:', error)
    report.value = null
    errorMessage.value = t('dashboard.dailyReport.loadFailed')
  } finally {
    if (currentRequest === requestID) loading.value = false
  }
}

watch(
  () => [props.show, props.date, locale.value] as const,
  ([show]) => {
    if (show) void loadReport()
  },
  { immediate: true }
)
</script>

<style scoped>
.daily-report-theme {
  --report-ink: #262823;
  --report-copy: #3d2618;
  --report-muted: #6f726c;
  --report-subtle: #8a877e;
  --report-line: rgba(42, 47, 40, 0.12);
  --report-card: #f3efe7;
  --report-track: #e4ddd0;
  --report-peach: #f8f1ea;
  --report-peach-line: #e8c9a3;
  --report-chip: #f0e0d0;
  --report-accent: #aa7149;
  --report-accent-deep: #895634;
  --report-on-accent: #fffefb;
}

:global(html.dark) .daily-report-theme {
  --report-ink: #f1eee7;
  --report-copy: #f1eee7;
  --report-muted: #b6b7b0;
  --report-subtle: #858981;
  --report-line: rgba(240, 238, 230, 0.12);
  --report-card: #141511;
  --report-track: #2c2a24;
  --report-peach: #3a2a1c;
  --report-peach-line: #8a5a38;
  --report-chip: #5c4030;
  --report-accent: #d09a71;
  --report-accent-deep: #e3b48f;
  --report-on-accent: #fffefb;
}

.daily-report-loading,
.daily-report-error {
  display: flex;
  min-height: 260px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  color: var(--report-muted);
  text-align: center;
}

.daily-report-error {
  color: #b42318;
}

.daily-report-error button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.daily-report-content {
  display: grid;
  gap: 22px;
}

.daily-report-narrative {
  display: flex;
  gap: 16px;
  border: 1px solid var(--report-peach-line);
  border-radius: 8px;
  background: var(--report-peach);
  padding: 20px;
}

.daily-report-narrative-icon {
  display: grid;
  width: 42px;
  height: 42px;
  flex: 0 0 42px;
  place-items: center;
  border-radius: 8px;
  background: var(--report-accent);
  color: var(--report-on-accent);
}

.daily-report-narrative-copy {
  min-width: 0;
}

.daily-report-narrative-copy > p {
  white-space: pre-line;
  color: var(--report-copy);
  font-size: 15px;
  line-height: 1.75;
}

.daily-report-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  margin-top: 12px;
  color: var(--report-muted);
  font-size: 12px;
}

.daily-report-comparison {
  border-radius: 999px;
  background: var(--report-chip);
  padding: 3px 8px;
  color: var(--report-accent-deep);
  font-weight: 600;
}

.daily-report-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border-block: 1px solid var(--report-line);
}

.daily-report-metrics > div {
  min-width: 0;
  padding: 16px 18px;
}

.daily-report-metrics > div + div {
  border-left: 1px solid var(--report-line);
}

.daily-report-metrics dt {
  color: var(--report-muted);
  font-size: 12px;
  font-weight: 600;
}

.daily-report-metrics dd {
  margin-top: 5px;
  overflow: hidden;
  color: var(--report-accent-deep);
  font-size: 24px;
  font-weight: 800;
  line-height: 1.2;
  text-overflow: ellipsis;
}

.daily-report-metrics small {
  display: block;
  margin-top: 6px;
  overflow-wrap: anywhere;
  color: var(--report-subtle);
  font-size: 11px;
  line-height: 1.4;
}

.daily-report-models header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.daily-report-models h4 {
  color: var(--report-ink);
  font-size: 15px;
  font-weight: 700;
}

.daily-report-models header p,
.daily-report-models header > span {
  color: var(--report-subtle);
  font-size: 12px;
}

.daily-report-model-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 190px), 1fr));
  gap: 8px;
}

.daily-report-model-row {
  display: grid;
  min-width: 0;
  grid-template-rows: auto 4px auto;
  gap: 8px;
  padding: 11px 12px;
  border: 1px solid var(--report-line);
  border-radius: 8px;
  background: var(--report-card);
}

.daily-report-model-rank {
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  border-radius: 5px;
  background: var(--report-track);
  color: var(--report-muted);
  font-size: 11px;
  font-weight: 700;
}

.daily-report-model-heading {
  display: grid;
  min-width: 0;
  grid-template-columns: 22px minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
}

.daily-report-model-heading strong {
  min-width: 0;
  overflow: hidden;
  color: var(--report-ink);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.daily-report-model-share {
  color: var(--report-accent-deep);
  font-size: 11px;
  font-weight: 700;
}

.daily-report-model-track {
  height: 4px;
  overflow: hidden;
  border-radius: 2px;
  background: var(--report-track);
}

.daily-report-model-track i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--report-accent);
}

.daily-report-model-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.daily-report-model-stats > div {
  min-width: 0;
}

.daily-report-model-stats > div + div {
  border-left: 1px solid var(--report-line);
  padding-left: 10px;
}

.daily-report-model-stats dt {
  color: var(--report-subtle);
  font-size: 10px;
  line-height: 1.2;
}

.daily-report-model-stats dd {
  margin-top: 2px;
  overflow: hidden;
  color: var(--report-accent-deep);
  font-size: 13px;
  font-weight: 800;
  line-height: 1.2;
  text-overflow: ellipsis;
}

.daily-report-empty {
  display: flex;
  min-height: 120px;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border-block: 1px solid var(--report-line);
  color: var(--report-subtle);
  font-size: 13px;
}

@media (max-width: 720px) {
  .daily-report-narrative {
    padding: 16px;
  }

  .daily-report-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .daily-report-metrics > div + div {
    border-left: 0;
  }

  .daily-report-metrics > div:nth-child(even) {
    border-left: 1px solid var(--report-line);
  }

  .daily-report-metrics > div:nth-child(n + 3) {
    border-top: 1px solid var(--report-line);
  }
}

@media (max-width: 460px) {
  .daily-report-narrative-icon {
    width: 36px;
    height: 36px;
    flex-basis: 36px;
  }

  .daily-report-metrics dd {
    font-size: 20px;
  }
}
</style>
