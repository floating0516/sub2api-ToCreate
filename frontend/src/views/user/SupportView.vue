<template>
  <AppLayout>
    <div class="mx-auto flex max-w-7xl flex-col gap-6 pb-8">
      <header class="flex flex-col gap-6 rounded-3xl border border-gray-200 bg-white px-6 py-7 shadow-sm sm:px-8 lg:flex-row lg:items-center lg:justify-between dark:border-dark-700 dark:bg-dark-900">
        <div class="min-w-0 max-w-2xl">
          <p class="text-sm font-medium text-primary-600 dark:text-primary-400">{{ t('support.centerEyebrow') }}</p>
          <h1 class="mt-2 text-3xl font-semibold text-gray-900 dark:text-white">{{ t('support.centerTitle') }}</h1>
          <p class="mt-3 text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('support.centerDescription') }}</p>
        </div>
        <button
          type="button"
          data-testid="create-ticket"
          class="inline-flex shrink-0 items-center justify-center gap-3 rounded-xl bg-gray-950 px-5 py-3 text-sm font-semibold text-white transition hover:bg-gray-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-500 dark:bg-white dark:text-gray-950 dark:hover:bg-gray-100"
          @click="showCreateDialog = true"
        >
          <Icon name="plus" size="sm" />
          {{ t('support.createTicket') }}
          <Icon name="arrowRight" size="sm" />
        </button>
      </header>

      <div role="status" class="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm leading-6 text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
        <Icon name="infoCircle" size="sm" class="mt-1 shrink-0" />
        <p>{{ t('support.listUnavailable') }}</p>
      </div>

      <section :aria-label="t('support.statistics')" class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        <button
          v-for="stat in stats"
          :key="stat.key"
          type="button"
          disabled
          :title="t('support.statisticsUnavailable')"
          class="min-w-0 cursor-not-allowed rounded-2xl border border-gray-200 bg-white p-4 text-left shadow-sm dark:border-dark-700 dark:bg-dark-900"
        >
          <span :class="['flex h-9 w-9 items-center justify-center rounded-lg', stat.color]"><Icon :name="stat.icon" size="sm" /></span>
          <span class="mt-3 block text-sm text-gray-500 dark:text-gray-400">{{ stat.label }}</span>
          <span class="mt-1 block text-2xl font-semibold text-gray-900 dark:text-white" :aria-label="t('support.statisticsUnavailable')">--</span>
        </button>
      </section>

      <div role="tablist" :aria-label="t('support.centerEyebrow')" class="flex gap-5 border-b border-gray-200 dark:border-dark-700">
        <button
          v-for="tab in tabs"
          :id="`support-tab-${tab.key}`"
          :key="tab.key"
          type="button"
          role="tab"
          :aria-selected="activeTab === tab.key"
          :aria-controls="`support-panel-${tab.key}`"
          :tabindex="activeTab === tab.key ? 0 : -1"
          :class="[
            'flex items-center gap-2 border-b-2 px-1 pb-3 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500',
            activeTab === tab.key ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-white',
          ]"
          @click="activeTab = tab.key"
          @keydown.right.prevent="focusTab(tab.key === 'tickets' ? 'chat' : 'tickets')"
          @keydown.left.prevent="focusTab(tab.key === 'tickets' ? 'chat' : 'tickets')"
          @keydown.home.prevent="focusTab('tickets')"
          @keydown.end.prevent="focusTab('chat')"
        >
          <Icon :name="tab.icon" size="sm" />
          {{ tab.label }}
        </button>
      </div>

      <section v-show="activeTab === 'tickets'" id="support-panel-tickets" role="tabpanel" aria-labelledby="support-tab-tickets" class="overflow-hidden rounded-3xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-4 border-b border-gray-200 px-5 py-5 sm:px-6 lg:flex-row lg:items-center lg:justify-between dark:border-dark-700">
          <div class="min-w-0">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('support.ticketRecords') }}</h2>
            <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('support.ticketRecordsHint') }}</p>
          </div>
          <fieldset disabled class="flex flex-col gap-2 opacity-60 sm:flex-row lg:max-w-md">
            <label class="min-w-0 flex-1">
              <span class="sr-only">{{ t('support.searchPlaceholder') }}</span>
              <SearchInput v-model="search" :placeholder="t('support.searchPlaceholder')" />
            </label>
            <Select
              v-model="typeFilter"
              :options="typeOptions"
              :aria-label="t('support.type')"
              aria-describedby="support-type-unavailable"
              disabled
              class="w-full sm:w-40"
            />
          </fieldset>
        </div>
        <p id="support-type-unavailable" class="px-5 pt-4 text-xs text-gray-500 sm:px-6 dark:text-gray-400">{{ t('support.typeFilterUnavailable') }}</p>
        <div class="px-6 py-12 text-center">
          <Icon name="inbox" size="xl" class="mx-auto text-gray-400" />
          <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('support.listNotConnected') }}</h3>
          <p class="mx-auto mt-2 max-w-lg text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('support.noTicketsHint') }}</p>
          <button type="button" class="btn btn-primary mt-5" @click="activeTab = 'chat'"><Icon name="chat" size="sm" class="mr-2" />{{ t('support.aiConsultation') }}</button>
        </div>
        <footer class="border-t border-gray-200 px-5 py-4 text-xs text-gray-500 sm:px-6 dark:border-dark-700 dark:text-gray-400">{{ t('support.paginationUnavailable') }}</footer>
      </section>

      <section v-show="activeTab === 'tickets'" :aria-label="t('support.currentTicket')" class="border-t border-gray-200 pt-5 dark:border-dark-700">
        <div class="px-1">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('support.currentTicket') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('support.currentThreadOnly') }}</p>
        </div>
        <div v-if="loadingTicket" role="status" class="space-y-3 p-5 sm:p-6">
          <span class="sr-only">{{ t('common.loading') }}</span>
          <div class="h-24 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-800" />
        </div>
        <div v-else-if="ticketError || store.error === 'support.errors.loadThread'" role="alert" class="px-6 py-12 text-center">
          <Icon name="exclamationCircle" size="lg" class="mx-auto text-red-500" />
          <p class="mt-3 text-sm text-gray-600 dark:text-gray-300">{{ t(ticketError || store.error) }}</p>
          <button type="button" class="btn btn-secondary mt-4" :disabled="store.loading" @click="refreshSupport"><Icon name="refresh" size="sm" class="mr-2" />{{ t('common.refresh') }}</button>
        </div>
        <button
          v-else-if="ticket"
          type="button"
          data-testid="ticket-row"
          class="group mt-4 flex w-full flex-col gap-4 rounded-xl border border-gray-200 bg-white px-5 py-6 text-left transition hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 sm:px-6 lg:flex-row lg:items-center dark:border-dark-700 dark:bg-dark-900 dark:hover:bg-dark-800"
          @click="showDetailDialog = true"
        >
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
              <span class="break-all font-mono">{{ ticket.id }}</span>
              <span class="rounded-full bg-gray-100 px-2 py-1 dark:bg-dark-800">{{ t('support.typeUnavailable') }}</span>
              <span :class="ticket.priority === 'high' ? 'text-red-600 dark:text-red-300' : 'text-gray-500 dark:text-gray-400'">{{ priorityLabel(ticket.priority) }}</span>
            </div>
            <h3 class="mt-2 break-words text-sm font-semibold text-gray-900 dark:text-white">{{ ticket.title }}</h3>
            <p class="mt-1 line-clamp-2 break-words text-sm leading-6 text-gray-500 dark:text-gray-400">{{ ticket.description }}</p>
            <p v-if="ticket.product" class="mt-2 break-words text-xs text-gray-500 dark:text-gray-400">{{ t('support.product') }}: {{ ticket.product }}</p>
          </div>
          <div class="flex min-w-0 flex-wrap items-center gap-3 lg:max-w-sm lg:justify-end">
            <span :class="['rounded-full px-2.5 py-1 text-xs font-medium', statusPresentation(ticket.status).color]">{{ statusPresentation(ticket.status).label }}</span>
            <time :datetime="ticket.updated_at" class="text-xs text-gray-500 dark:text-gray-400">{{ formatDate(ticket.updated_at) }}</time>
            <Icon name="chevronRight" size="sm" class="shrink-0 text-gray-400 group-hover:text-primary-600" />
          </div>
        </button>
        <div v-else class="px-6 py-12 text-center">
          <Icon name="inbox" size="xl" class="mx-auto text-gray-400" />
          <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('support.noLinkedTicket') }}</h3>
        </div>
      </section>

      <div v-show="activeTab === 'chat'" id="support-panel-chat" role="tabpanel" aria-labelledby="support-tab-chat">
        <SupportConversation :restoring="restoring" @refresh="refreshSupport" />
      </div>
    </div>

    <BaseDialog :show="showCreateDialog" :title="t('support.createTicket')" width="normal" @close="showCreateDialog = false">
      <p role="status" class="mb-5 rounded-lg bg-amber-50 p-3 text-sm leading-6 text-amber-800 dark:bg-amber-950/30 dark:text-amber-200">{{ t('support.directCreateUnavailable') }}</p>
      <fieldset disabled class="space-y-4 opacity-60">
        <div>
          <label for="support-create-type" class="label">{{ t('support.type') }}</label>
          <Select id="support-create-type" v-model="createType" :options="ticketTypes" disabled :aria-label="t('support.type')" />
        </div>
        <div>
          <label for="support-create-title" class="label">{{ t('support.formTitle') }}</label>
          <input id="support-create-title" class="input" :placeholder="t('support.formTitlePlaceholder')" />
        </div>
        <div>
          <label for="support-create-description" class="label">{{ t('support.formDescription') }}</label>
          <textarea id="support-create-description" class="input min-h-28" :placeholder="t('support.formDescriptionPlaceholder')" />
        </div>
        <div>
          <label for="support-create-order" class="label">{{ t('support.relatedOrder') }}</label>
          <input id="support-create-order" class="input" :placeholder="t('support.orderUnavailable')" />
        </div>
        <button type="button" disabled class="btn btn-secondary w-full justify-center">
          <Icon name="upload" size="sm" class="mr-2" />
          {{ t('support.chooseAttachment') }}
        </button>
      </fieldset>
      <p class="mt-3 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t('support.attachmentUnavailable') }}</p>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showCreateDialog = false">{{ t('common.close') }}</button>
        <button type="button" class="btn btn-primary" @click="openConsultation"><Icon name="chat" size="sm" class="mr-2" />{{ t('support.aiConsultation') }}</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="showDetailDialog && !!ticket" :title="t('support.ticketDetails')" width="wide" @close="showDetailDialog = false">
      <template v-if="ticket">
        <h2 class="mb-4 break-words text-lg font-semibold">{{ ticket.title }}</h2>
        <div class="flex flex-wrap items-center gap-3 text-xs"><span class="break-all font-mono text-gray-500 dark:text-gray-400">{{ ticket.id }}</span><span :class="['rounded-full px-2.5 py-1', statusPresentation(ticket.status).color]">{{ statusPresentation(ticket.status).label }}</span></div>
        <dl class="mt-5 grid grid-cols-1 gap-4 border-y border-gray-200 py-4 text-sm sm:grid-cols-2 dark:border-dark-700">
          <div>
            <dt class="text-gray-500 dark:text-gray-400">{{ t('support.type') }}</dt>
            <dd class="mt-1">{{ t('support.typeUnavailable') }}</dd>
          </div>
          <div>
            <dt class="text-gray-500 dark:text-gray-400">{{ t('support.priority') }}</dt>
            <dd class="mt-1">{{ priorityLabel(ticket.priority) }}</dd>
          </div>
          <div>
            <dt class="text-gray-500 dark:text-gray-400">{{ t('support.product') }}</dt>
            <dd class="mt-1 break-words">{{ ticket.product || t('support.notProvided') }}</dd>
          </div>
          <div>
            <dt class="text-gray-500 dark:text-gray-400">{{ t('support.updatedAt') }}</dt>
            <dd class="mt-1">{{ formatDate(ticket.updated_at) }}</dd>
          </div>
        </dl>
        <h3 class="mt-5 text-sm font-semibold">{{ t('support.formDescription') }}</h3>
        <p class="mt-3 whitespace-pre-wrap break-words text-sm leading-6 text-gray-600 dark:text-gray-300">{{ ticket.description }}</p>
        <p class="mt-5 rounded-lg bg-gray-50 p-3 text-sm leading-6 text-gray-500 dark:bg-dark-800 dark:text-gray-400">{{ t('support.replyUnavailable') }}</p>
      </template>
      <template #footer>
        <button type="button" class="btn btn-secondary" @click="showDetailDialog = false">{{ t('common.close') }}</button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import SupportConversation from '@/components/support/SupportConversation.vue'
import { supportAPI, type SupportTicketRecord } from '@/api/support'
import { useSupportStore } from '@/stores/support'

type Tab = 'tickets' | 'chat'
const { t, locale } = useI18n()
const store = useSupportStore()
const activeTab = ref<Tab>('tickets')
const search = ref('')
const typeFilter = ref('all')
const createType = ref('pre-sale')
const ticket = ref<SupportTicketRecord | null>(null)
const ticketError = ref('')
const loadingTicket = ref(true)
const restoring = ref(false)
const showCreateDialog = ref(false)
const showDetailDialog = ref(false)
let requestVersion = 0
let disposed = false

const tabs = computed(() => [
  { key: 'tickets' as const, label: t('support.ticketRecords'), icon: 'inbox' as const },
  { key: 'chat' as const, label: t('support.aiConsultation'), icon: 'chat' as const },
])
const stats = computed(() => [
  { key: 'all', label: t('support.stats.all'), icon: 'inbox' as const, color: 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-gray-300' },
  { key: 'pendingAgent', label: t('support.stats.pendingAgent'), icon: 'clock' as const, color: 'bg-orange-50 text-orange-600 dark:bg-orange-950/30 dark:text-orange-300' },
  { key: 'pendingUser', label: t('support.stats.pendingUser'), icon: 'chat' as const, color: 'bg-blue-50 text-blue-600 dark:bg-blue-950/30 dark:text-blue-300' },
  { key: 'resolved', label: t('support.stats.resolved'), icon: 'checkCircle' as const, color: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/30 dark:text-emerald-300' },
  { key: 'closed', label: t('support.stats.closed'), icon: 'xCircle' as const, color: 'bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-gray-400' },
])
const ticketTypes = computed(() => [
  { value: 'pre-sale', label: t('support.types.preSale') },
  { value: 'order', label: t('support.types.order') },
  { value: 'after-sale', label: t('support.types.afterSale') },
  { value: 'technical', label: t('support.types.technical') },
  { value: 'other', label: t('support.types.other') },
])
const typeOptions = computed(() => [{ value: 'all', label: t('support.allTypes') }, ...ticketTypes.value])

async function focusTab(tab: Tab) {
  activeTab.value = tab
  await nextTick()
  document.getElementById(`support-tab-${tab}`)?.focus()
}
function openConsultation() {
  showCreateDialog.value = false
  activeTab.value = 'chat'
}
async function loadTicket() {
  const version = ++requestVersion
  const id = store.ticketId
  ticket.value = null
  ticketError.value = ''
  showDetailDialog.value = false
  loadingTicket.value = true
  try {
    if (id) {
      const result = await supportAPI.getTicket(id)
      if (version === requestVersion) ticket.value = result
    }
  } catch {
    if (version === requestVersion) ticketError.value = 'support.errors.loadTicket'
  } finally {
    if (version === requestVersion) loadingTicket.value = false
  }
}
async function refreshSupport() {
  if (restoring.value || store.loading) return
  restoring.value = true
  loadingTicket.value = true
  try {
    await store.refresh()
    if (!disposed) await loadTicket()
  } finally {
    restoring.value = false
  }
}
function priorityLabel(priority: string) {
  const labels: Record<string, string> = { high: 'priorityHigh', normal: 'priorityNormal', low: 'priorityLow' }
  return t(`support.${labels[priority] || 'priorityUnknown'}`)
}
function statusPresentation(status: string) {
  const gray = 'bg-gray-100 text-gray-700 dark:bg-dark-800 dark:text-gray-300'
  const statuses: Record<string, { key: string; color: string }> = {
    submitted: { key: 'submitted', color: gray },
    pending_agent: { key: 'pendingAgent', color: 'bg-orange-50 text-orange-700 dark:bg-orange-950/30 dark:text-orange-300' },
    pending_user: { key: 'pendingUser', color: 'bg-blue-50 text-blue-700 dark:bg-blue-950/30 dark:text-blue-300' },
    resolved: { key: 'resolved', color: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300' },
    closed: { key: 'closed', color: gray },
  }
  const entry = statuses[status] || { key: 'unknown', color: gray }
  return { label: t(`support.status.${entry.key}`), color: entry.color }
}
function formatDate(value: string) {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return t('support.notProvided')
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}

watch(() => store.ticketId, () => { if (!restoring.value) void loadTicket() })
onMounted(refreshSupport)
onUnmounted(() => {
  disposed = true
  requestVersion++
})
</script>
