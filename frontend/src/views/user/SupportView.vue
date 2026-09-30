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
          @click="openCreateDialog"
        >
          <Icon name="plus" size="sm" />
          {{ t('support.createTicket') }}
          <Icon name="arrowRight" size="sm" />
        </button>
      </header>

      <div v-if="listState === 'unavailable'" role="status" class="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm leading-6 text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
        <Icon name="infoCircle" size="sm" class="mt-1 shrink-0" />
        <p>{{ t('support.listUnavailable') }}</p>
      </div>

      <section :aria-label="t('support.statistics')" class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        <button
          v-for="stat in stats"
          :key="stat.key"
          type="button"
          :disabled="!portalAvailable"
          :title="totals ? stat.label : t('support.statisticsUnavailable')"
          :aria-pressed="statusFilter === stat.key"
          :class="['min-w-0 rounded-2xl border p-4 text-left shadow-sm disabled:cursor-not-allowed', statusFilter === stat.key ? 'border-primary-500 bg-primary-50 dark:bg-primary-950/30' : 'border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900']"
          @click="statusFilter = stat.key"
        >
          <span :class="['flex h-9 w-9 items-center justify-center rounded-lg', stat.color]"><Icon :name="stat.icon" size="sm" /></span>
          <span class="mt-3 block text-sm text-gray-500 dark:text-gray-400">{{ stat.label }}</span>
          <span class="mt-1 block text-2xl font-semibold text-gray-900 dark:text-white" :aria-label="totals ? undefined : t('support.statisticsUnavailable')">{{ totals?.[stat.key] ?? '--' }}</span>
        </button>
      </section>
      <div v-if="statsError" role="alert" class="flex items-center gap-3 text-sm text-red-600 dark:text-red-300"><span>{{ t('support.errors.loadStats') }}</span><button type="button" class="btn btn-ghost btn-sm" :title="t('common.refresh')" :aria-label="t('common.refresh')" @click="portal.loadStats"><Icon name="refresh" size="sm" /></button></div>

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
            <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t(portalAvailable ? 'support.ticketHistoryHint' : 'support.ticketRecordsHint') }}</p>
          </div>
          <fieldset :disabled="!portalAvailable" :class="['flex flex-col gap-2 sm:flex-row lg:max-w-md', !portalAvailable && 'opacity-60']">
            <label class="min-w-0 flex-1">
              <span class="sr-only">{{ t('support.searchPlaceholder') }}</span>
              <SearchInput v-model="search" :placeholder="t('support.searchPlaceholder')" />
            </label>
            <Select
              v-model="typeFilter"
              :options="typeOptions"
              :aria-label="t('support.type')"
              :aria-describedby="!portalAvailable ? 'support-type-unavailable' : undefined"
              :disabled="!portalAvailable"
              class="w-full sm:w-40"
            />
          </fieldset>
        </div>
        <p v-if="!portalAvailable" id="support-type-unavailable" class="px-5 pt-4 text-xs text-gray-500 sm:px-6 dark:text-gray-400">{{ t('support.typeFilterUnavailable') }}</p>
        <SupportTicketList :state="listState" :items="listItems" :total="total" :page="page" :page-size="portal.pageSize" :filtered="filtered" @update:page="page = $event" @open="openDetail" @retry="portal.refresh" @clear-filters="portal.clearFilters" @consultation="activeTab = 'chat'" />
      </section>

      <section v-if="!portalAvailable" v-show="activeTab === 'tickets'" :aria-label="t('support.currentTicket')" class="border-t border-gray-200 pt-5 dark:border-dark-700">
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
        <SupportTicketRow
          v-else-if="ticket"
          :ticket="displayTicket(ticket)"
          class="mt-4 rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900"
          @open="openDetail"
        />
        <div v-else class="px-6 py-12 text-center">
          <Icon name="inbox" size="xl" class="mx-auto text-gray-400" />
          <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('support.noLinkedTicket') }}</h3>
        </div>
      </section>

      <div v-show="activeTab === 'chat'" id="support-panel-chat" role="tabpanel" aria-labelledby="support-tab-chat">
        <SupportConversation :restoring="restoring" @refresh="refreshSupport" />
      </div>
    </div>

    <SupportTicketCreateDialog
      :key="auth.user?.id ?? 'guest'"
      :show="showCreateDialog"
      :submit-ticket="portalAvailable ? portal.create : undefined"
      :upload-file="portalAvailable ? supportAPI.uploadAttachment : undefined"
      :orders="orders"
      :orders-state="ordersState"
      @close="showCreateDialog = false"
      @consultation="openConsultation"
      @retry-orders="loadOrders"
      @submitted="onCreated"
    />

    <BaseDialog :show="showDetailDialog" :title="t('support.ticketDetails')" width="wide" :close-on-escape="!pendingAction && !actionBusy && !replyBusy" :show-close-button="!actionBusy && !replyBusy" @close="closeDetail">
      <div v-if="detailState === 'loading'" role="status" class="h-36 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800"><span class="sr-only">{{ t('common.loading') }}</span></div>
      <div v-else-if="detailState === 'error'" role="alert"><p class="text-sm text-red-600 dark:text-red-300">{{ t('support.errors.loadTicket') }}</p><button type="button" class="btn btn-secondary mt-3" @click="portal.open(selectedId)"><Icon name="refresh" size="sm" class="mr-2" />{{ t('common.refresh') }}</button></div>
      <template v-else-if="detail">
        <h2 class="mb-4 break-words text-lg font-semibold">{{ detail.title }}</h2>
        <div class="flex flex-wrap items-center gap-3 text-xs"><span class="break-all font-mono text-gray-500 dark:text-gray-400">{{ detail.id }}</span><span :class="['rounded-full px-2.5 py-1', ticketStatus(detail.status).color]">{{ t(ticketStatus(detail.status).key) }}</span></div>
        <dl class="mt-5 grid grid-cols-1 gap-4 border-y border-gray-200 py-4 text-sm sm:grid-cols-2 dark:border-dark-700">
          <div>
            <dt class="text-gray-500 dark:text-gray-400">{{ t('support.type') }}</dt>
            <dd class="mt-1">{{ t(ticketTypeKey(detail.type)) }}</dd>
          </div>
          <div>
            <dt class="text-gray-500 dark:text-gray-400">{{ t('support.priority') }}</dt>
            <dd class="mt-1">{{ t(priorityKey(detail.priority)) }}</dd>
          </div>
          <div>
            <dt class="text-gray-500 dark:text-gray-400">{{ t('support.product') }}</dt>
            <dd class="mt-1 break-words">{{ detail.product || t('support.notProvided') }}</dd>
          </div>
          <div>
            <dt class="text-gray-500 dark:text-gray-400">{{ t('support.updatedAt') }}</dt>
            <dd class="mt-1">{{ formatDate(detail.updated_at) }}</dd>
          </div>
          <div v-if="detail.order_id">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('support.relatedOrder') }}</dt>
            <dd class="mt-1 break-all">{{ detail.order_id }}</dd>
          </div>
        </dl>
        <h3 class="mt-5 text-sm font-semibold">{{ t('support.formDescription') }}</h3>
        <p class="mt-3 whitespace-pre-wrap break-words text-sm leading-6 text-gray-600 dark:text-gray-300">{{ detail.description }}</p>
        <SupportAttachments :attachments="originalAttachments" :download-file="supportAPI.downloadAttachment" />
        <SupportTicketDiscussion :key="`${auth.user?.id}:${detail.id}`" :messages="ticketMessages(detail)" :timeline-state="Array.isArray(detail.replies) ? 'ready' : 'unavailable'" :closed="detail.status === 'closed'" :resolved="detail.status === 'resolved'" :can-reply="!!detail.allowed_actions?.includes('reply') && !actionBusy && !replyBusy" :send-reply="Array.isArray(detail.allowed_actions) ? replyToTicket : undefined" :upload-file="uploadReplyAttachment" :download-file="supportAPI.downloadAttachment" :attachment-count="detail.attachments?.length ?? 0" />
        <p v-if="actionError" role="alert" class="mt-4 text-sm text-red-600 dark:text-red-300">{{ t('support.errors.ticketAction') }}</p>
      </template>
      <template #footer>
        <div class="flex w-full flex-wrap justify-end gap-3">
          <button v-for="action in stateActions" :key="action" type="button" :disabled="actionBusy || replyBusy" class="btn btn-secondary" @click="pendingAction = action"><Icon :name="action === 'resolve' ? 'checkCircle' : action === 'close' ? 'xCircle' : 'refresh'" size="sm" class="mr-2" />{{ t(`support.actions.${action}`) }}</button>
          <button type="button" class="btn btn-secondary" :disabled="actionBusy || replyBusy" @click="closeDetail">{{ t('common.close') }}</button>
        </div>
      </template>
    </BaseDialog>
    <ConfirmDialog :show="!!pendingAction" :title="pendingAction ? t(`support.actions.${pendingAction}`) : ''" :message="t('support.confirmStateChange')" :danger="pendingAction === 'close'" :confirm-disabled="actionBusy" @cancel="pendingAction = null" @confirm="confirmStateChange" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import SupportConversation from '@/components/support/SupportConversation.vue'
import SupportTicketCreateDialog from '@/components/support/SupportTicketCreateDialog.vue'
import SupportTicketDiscussion from '@/components/support/SupportTicketDiscussion.vue'
import SupportTicketList from '@/components/support/SupportTicketList.vue'
import SupportTicketRow from '@/components/support/SupportTicketRow.vue'
import SupportAttachments from '@/components/support/SupportAttachments.vue'
import { priorityKey, ticketDate, ticketStatus, ticketTypeKey, ticketMessages } from '@/components/support/ticketPresentation'
import { useTicketPortal } from '@/composables/useTicketPortal'
import { listManagedRechargeOrders } from '@/api/managedRecharge'
import { supportAPI, type SupportTicketRecord } from '@/api/support'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useSupportStore } from '@/stores/support'

type Tab = 'tickets' | 'chat'
const { t, locale } = useI18n()
const store = useSupportStore()
const auth = useAuthStore()
const app = useAppStore()
const portal = useTicketPortal()
const { available: portalAvailable, listState, stats: totals, statsError, total, page, filtered, detail, detailState, actionBusy, replyBusy, actionError } = portal
const activeTab = ref<Tab>('tickets')
const search = portal.search
const typeFilter = portal.type
const statusFilter = portal.status
const selectedId = ref('')
const pendingAction = ref<'resolve' | 'close' | 'reopen' | null>(null)
const orders = ref<{ value: string; label: string }[]>([])
const ordersState = ref<'unavailable' | 'loading' | 'error' | 'ready'>('unavailable')
let orderVersion = 0
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
  { key: 'pending_agent', label: t('support.stats.pendingAgent'), icon: 'clock' as const, color: 'bg-orange-50 text-orange-600 dark:bg-orange-950/30 dark:text-orange-300' },
  { key: 'pending_user', label: t('support.stats.pendingUser'), icon: 'chat' as const, color: 'bg-blue-50 text-blue-600 dark:bg-blue-950/30 dark:text-blue-300' },
  { key: 'resolved', label: t('support.stats.resolved'), icon: 'checkCircle' as const, color: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/30 dark:text-emerald-300' },
  { key: 'closed', label: t('support.stats.closed'), icon: 'xCircle' as const, color: 'bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-gray-400' },
] as const)
const ticketTypes = computed(() => [
  { value: 'pre-sale', label: t('support.types.preSale') },
  { value: 'order', label: t('support.types.order') },
  { value: 'after-sale', label: t('support.types.afterSale') },
  { value: 'technical', label: t('support.types.technical') },
  { value: 'other', label: t('support.types.other') },
])
const typeOptions = computed(() => [{ value: 'all', label: t('support.allTypes') }, ...ticketTypes.value])
function displayTicket(record: SupportTicketRecord) {
  return { ...record, typeLabel: t(ticketTypeKey(record.type)), relatedOrderLabel: record.order_id || undefined }
}
const listItems = computed(() => portal.items.value.map(displayTicket))
const stateActions = computed(() => (['resolve', 'close', 'reopen'] as const).filter((action) => detail.value?.allowed_actions?.includes(action)))
const originalAttachments = computed(() => {
  const replyIds = new Set(detail.value?.replies?.flatMap((reply) => reply.attachments.map((file) => file.id)))
  return detail.value?.attachments?.filter((file) => !replyIds.has(file.id)) || []
})

function openCreateDialog() {
  showCreateDialog.value = true
  if (portalAvailable.value) void loadOrders()
}
async function loadOrders() {
  const version = ++orderVersion
  const userId = auth.user?.id
  ordersState.value = 'loading'
  try {
    const result = await listManagedRechargeOrders(100)
    if (disposed || version !== orderVersion || userId !== auth.user?.id) return
    orders.value = result.map((order) => ({ value: String(order.id), label: `${order.order_no} · ${order.product_name}` }))
    ordersState.value = 'ready'
  } catch {
    if (!disposed && version === orderVersion && userId === auth.user?.id) ordersState.value = 'error'
  }
}
async function onCreated() {
  showCreateDialog.value = false
  app.showSuccess(t('support.created'))
  await portal.refresh()
}
function openDetail(id: string) {
  selectedId.value = id
  showDetailDialog.value = true
  void portal.open(id)
}
function closeDetail() {
  if (actionBusy.value || replyBusy.value) return
  showDetailDialog.value = false
  pendingAction.value = null
  portal.closeDetail()
}
function uploadReplyAttachment(file: File) {
  if (!detail.value?.allowed_actions?.includes('reply')) return Promise.reject(new Error('Reply is not allowed'))
  return supportAPI.uploadAttachment(file, detail.value.id)
}
async function replyToTicket(text: string, ids: string[], messageId: string) {
  const completed = await portal.reply(text, ids, messageId)
  if (completed) app.showSuccess(t('support.replySent'))
  return completed
}
async function confirmStateChange() {
  const action = pendingAction.value
  if (!action) return
  const completed = await portal.transition(action)
  pendingAction.value = null
  if (completed) app.showSuccess(t('support.actionSuccess'))
}

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
  const userId = auth.user?.id
  restoring.value = true
  loadingTicket.value = true
  try {
    await store.refresh()
    if (!disposed && userId === auth.user?.id) {
      await loadTicket()
      if (!disposed && userId === auth.user?.id) await portal.refresh()
    }
  } finally {
    if (userId === auth.user?.id) restoring.value = false
  }
}
function formatDate(value: string) {
  return ticketDate(value, locale.value) || t('support.notProvided')
}

watch(() => store.ticketId, () => {
  if (!restoring.value) {
    void loadTicket()
    if (portalAvailable.value) void portal.refresh()
  }
})
watch(() => auth.user?.id, () => {
  requestVersion++
  ticket.value = null
  ticketError.value = ''
  loadingTicket.value = false
  restoring.value = false
  showCreateDialog.value = false
  showDetailDialog.value = false
  pendingAction.value = null
  selectedId.value = ''
  orderVersion++
  orders.value = []
  ordersState.value = 'unavailable'
  activeTab.value = 'tickets'
  const userId = auth.user?.id
  void nextTick(() => {
    if (!disposed && userId && userId === auth.user?.id) void refreshSupport()
  })
}, { flush: 'sync' })
onMounted(refreshSupport)
onUnmounted(() => {
  disposed = true
  requestVersion++
})
</script>
