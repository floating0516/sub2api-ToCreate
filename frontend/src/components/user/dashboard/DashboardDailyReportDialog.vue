<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="wide"
    @close="emit('close')"
  >
    <div v-if="loading" class="daily-report-loading">
      <LoadingSpinner size="lg" />
      <p>{{ t('dashboard.dailyReport.loading') }}</p>
    </div>

    <div v-else-if="errorMessage" class="daily-report-error" role="alert">
      <Icon name="exclamationCircle" size="lg" />
      <p>{{ errorMessage }}</p>
      <button type="button" class="btn btn-secondary" @click="loadReport">
        <Icon name="refresh" size="sm" />
        {{ t('dashboard.dailyReport.retry') }}
      </button>
    </div>

    <div v-else-if="report" class="daily-report-content">
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
.daily-report-loading,
.daily-report-error {
  display: flex;
  min-height: 260px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  color: #6b7280;
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
  border: 1px solid #b7e4cc;
  border-radius: 8px;
  background: #effaf4;
  padding: 20px;
}

.daily-report-narrative-icon {
  display: grid;
  width: 42px;
  height: 42px;
  flex: 0 0 42px;
  place-items: center;
  border-radius: 8px;
  background: #168a58;
  color: #fff;
}

.daily-report-narrative-copy {
  min-width: 0;
}

.daily-report-narrative-copy > p {
  white-space: pre-line;
  color: #173d2c;
  font-size: 15px;
  line-height: 1.75;
}

.daily-report-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  margin-top: 12px;
  color: #527061;
  font-size: 12px;
}

.daily-report-comparison {
  border-radius: 999px;
  background: #d9f3e5;
  padding: 3px 8px;
  color: #126b45;
  font-weight: 600;
}

.daily-report-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border-block: 1px solid #e5e7eb;
}

.daily-report-metrics > div {
  min-width: 0;
  padding: 16px 18px;
}

.daily-report-metrics > div + div {
  border-left: 1px solid #e5e7eb;
}

.daily-report-metrics dt {
  color: #6b7280;
  font-size: 12px;
  font-weight: 600;
}

.daily-report-metrics dd {
  margin-top: 5px;
  overflow: hidden;
  color: #145c3d;
  font-size: 24px;
  font-weight: 800;
  line-height: 1.2;
  text-overflow: ellipsis;
}

.daily-report-metrics small {
  display: block;
  margin-top: 6px;
  overflow-wrap: anywhere;
  color: #7c838d;
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
  color: #111827;
  font-size: 15px;
  font-weight: 700;
}

.daily-report-models header p,
.daily-report-models header > span {
  color: #7c838d;
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
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.daily-report-model-rank {
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  border-radius: 5px;
  background: #eef2f6;
  color: #536273;
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
  color: #26313d;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.daily-report-model-share {
  color: #145c3d;
  font-size: 11px;
  font-weight: 700;
}

.daily-report-model-track {
  height: 4px;
  overflow: hidden;
  border-radius: 2px;
  background: #edf1f4;
}

.daily-report-model-track i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #28a96b;
}

.daily-report-model-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.daily-report-model-stats > div {
  min-width: 0;
}

.daily-report-model-stats > div + div {
  border-left: 1px solid #edf0f2;
  padding-left: 10px;
}

.daily-report-model-stats dt {
  color: #7c838d;
  font-size: 10px;
  line-height: 1.2;
}

.daily-report-model-stats dd {
  margin-top: 2px;
  overflow: hidden;
  color: #145c3d;
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
  border-block: 1px solid #e5e7eb;
  color: #7c838d;
  font-size: 13px;
}

:global(.dark) .daily-report-narrative {
  border-color: #275b43;
  background: #142d22;
}

:global(.dark) .daily-report-narrative-copy > p {
  color: #c5ead5;
}

:global(.dark) .daily-report-meta {
  color: #91b7a1;
}

:global(.dark) .daily-report-comparison {
  background: #204d37;
  color: #9ce0ba;
}

:global(.dark) .daily-report-metrics,
:global(.dark) .daily-report-metrics > div + div,
:global(.dark) .daily-report-model-row,
:global(.dark) .daily-report-model-stats > div + div,
:global(.dark) .daily-report-empty {
  border-color: #303b49;
}

:global(.dark) .daily-report-model-row {
  background: #19222d;
}

:global(.dark) .daily-report-metrics dd,
:global(.dark) .daily-report-models h4,
:global(.dark) .daily-report-model-heading strong {
  color: #e5e7eb;
}

:global(.dark) .daily-report-metrics dd,
:global(.dark) .daily-report-model-share,
:global(.dark) .daily-report-model-stats dd {
  color: #8bd7ad;
}

:global(.dark) .daily-report-model-rank,
:global(.dark) .daily-report-model-track {
  background: #273342;
  color: #b8c2cf;
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
    border-left: 1px solid #e5e7eb;
  }

  .daily-report-metrics > div:nth-child(n + 3) {
    border-top: 1px solid #e5e7eb;
  }

  :global(.dark) .daily-report-metrics > div:nth-child(even),
  :global(.dark) .daily-report-metrics > div:nth-child(n + 3) {
    border-color: #303b49;
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
