<template>
  <AppLayout>
    <div class="mx-auto flex max-w-7xl flex-col gap-6 pb-8">
      <header class="flex flex-col gap-4 border-b border-gray-200 pb-5 dark:border-dark-700 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p class="text-sm font-medium text-primary-600 dark:text-primary-400">{{ t('support.admin.title') }}</p>
          <h1 class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ t('support.admin.title') }}</h1>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('support.ticketHistoryHint') }}</p>
        </div>
        <button type="button" class="btn btn-secondary" :disabled="refreshing" :title="t('common.refresh')" @click="refresh">
          <Icon name="refresh" size="sm" class="mr-2" />{{ t('common.refresh') }}
        </button>
      </header>

      <section :aria-label="t('support.statistics')" class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        <button
          v-for="stat in stats"
          :key="stat.key"
          type="button"
          :disabled="!portal.available.value"
          :aria-pressed="portal.status.value === stat.key"
          :class="['min-w-0 rounded-xl border p-4 text-left shadow-sm disabled:cursor-not-allowed', portal.status.value === stat.key ? 'border-primary-500 bg-primary-50 dark:bg-primary-950/30' : 'border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900']"
          @click="setStatus(stat.key)">
          <span :class="['flex h-9 w-9 items-center justify-center rounded-lg', stat.color]"><Icon :name="stat.icon" size="sm" /></span>
          <span class="mt-3 block text-sm text-gray-500 dark:text-gray-400">{{ stat.label }}</span>
          <span class="mt-1 block text-2xl font-semibold text-gray-900 dark:text-white">{{ portal.stats.value?.[stat.key] ?? '--' }}</span>
        </button>
      </section>

      <div v-if="portal.listState.value === 'unavailable'" role="status" class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">{{ t('support.admin.unavailable') }}</div>
      <div v-if="portal.statsError.value" role="alert" class="text-sm text-red-600 dark:text-red-300">{{ t('support.errors.loadStats') }}</div>

      <section class="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('support.ticketRecords') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ portal.total.value }} · {{ t('support.admin.requesterId') }}</p>
          </div>
          <div class="flex flex-col gap-2 sm:flex-row">
            <SearchInput v-model="portal.search.value" :placeholder="t('support.searchPlaceholder')" class="sm:w-64" />
            <Select v-model="portal.type.value" :options="typeOptions" :aria-label="t('support.type')" class="sm:w-40" />
            <Select v-model="portal.status.value" :options="statusOptions" :aria-label="t('support.status.pendingAgent')" class="sm:w-44" />
          </div>
        </div>
        <SupportTicketList
          :state="portal.listState.value"
          :items="listItems"
          :total="portal.total.value"
          :page="portal.page.value"
          :page-size="portal.pageSize"
          :filtered="portal.filtered.value"
          @update:page="portal.page.value = $event"
          @open="openDetail"
          @retry="refresh"
          @clear-filters="portal.clearFilters"
        />
      </section>
    </div>

    <BaseDialog :show="showDetail" :title="t('support.ticketDetails')" width="wide" :close-on-escape="!portal.actionBusy.value && !portal.replyBusy.value" :show-close-button="!portal.actionBusy.value && !portal.replyBusy.value" @close="closeDetail">
      <div v-if="portal.detailState.value === 'loading'" role="status" class="h-36 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800"><span class="sr-only">{{ t('common.loading') }}</span></div>
      <div v-else-if="portal.detailState.value === 'error'" role="alert" class="text-sm text-red-600 dark:text-red-300">
        <p>{{ t('support.errors.loadTicket') }}</p>
        <button type="button" class="btn btn-secondary mt-3" @click="portal.open(selectedId)"><Icon name="refresh" size="sm" class="mr-2" />{{ t('common.refresh') }}</button>
      </div>
      <template v-else-if="portal.detail.value">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0">
            <h2 class="break-words text-lg font-semibold text-gray-900 dark:text-white">{{ portal.detail.value.title }}</h2>
            <p class="mt-1 break-all font-mono text-xs text-gray-500 dark:text-gray-400">{{ portal.detail.value.id }}</p>
          </div>
          <span :class="['rounded-full px-2.5 py-1 text-xs font-medium', ticketStatus(portal.detail.value.status).color]">{{ t(ticketStatus(portal.detail.value.status).key) }}</span>
        </div>
        <dl class="mt-5 grid grid-cols-1 gap-4 border-y border-gray-200 py-4 text-sm sm:grid-cols-2 dark:border-dark-700">
          <div><dt class="text-gray-500 dark:text-gray-400">{{ t('support.admin.requesterId') }}</dt><dd class="mt-1 break-all font-mono">{{ portal.detail.value.requester_id || t('support.notProvided') }}</dd></div>
          <div><dt class="text-gray-500 dark:text-gray-400">{{ t('support.type') }}</dt><dd class="mt-1">{{ t(ticketTypeKey(portal.detail.value.type)) }}</dd></div>
          <div><dt class="text-gray-500 dark:text-gray-400">{{ t('support.priority') }}</dt><dd class="mt-1">{{ t(priorityKey(portal.detail.value.priority)) }}</dd></div>
          <div><dt class="text-gray-500 dark:text-gray-400">{{ t('support.updatedAt') }}</dt><dd class="mt-1">{{ formatDate(portal.detail.value.updated_at) }}</dd></div>
          <div v-if="portal.detail.value.order_id"><dt class="text-gray-500 dark:text-gray-400">{{ t('support.relatedOrder') }}</dt><dd class="mt-1 break-all">{{ portal.detail.value.order_id }}</dd></div>
        </dl>
        <h3 class="mt-5 text-sm font-semibold">{{ t('support.formDescription') }}</h3>
        <p class="mt-3 whitespace-pre-wrap break-words text-sm leading-6 text-gray-600 dark:text-gray-300">{{ portal.detail.value.description }}</p>
        <SupportAttachments :attachments="originalAttachments" :download-file="adminSupportAPI.downloadAttachment" />
        <SupportTicketDiscussion
          :key="portal.detail.value.id"
          :messages="ticketMessages(portal.detail.value)"
          timeline-state="ready"
          admin
          :closed="portal.detail.value.status === 'closed'"
          :resolved="portal.detail.value.status === 'resolved'"
          :can-reply="!!portal.detail.value.allowed_actions?.includes('reply') && !portal.actionBusy.value && !portal.replyBusy.value"
          :send-reply="replyToTicket"
          :upload-file="uploadReplyAttachment"
          :download-file="adminSupportAPI.downloadAttachment"
          :attachment-count="portal.detail.value.attachments?.length ?? 0"
        />
        <p v-if="portal.actionError.value" role="alert" class="mt-4 text-sm text-red-600 dark:text-red-300">{{ t('support.errors.ticketAction') }}</p>
      </template>
      <template #footer>
        <div class="flex w-full flex-wrap justify-end gap-3">
          <button v-for="action in stateActions" :key="action" type="button" :disabled="portal.actionBusy.value || portal.replyBusy.value" class="btn btn-secondary" @click="pendingAction = action"><Icon :name="action === 'resolve' ? 'checkCircle' : action === 'close' ? 'xCircle' : 'refresh'" size="sm" class="mr-2" />{{ t(`support.actions.${action}`) }}</button>
          <button type="button" class="btn btn-secondary" :disabled="portal.actionBusy.value || portal.replyBusy.value" @click="closeDetail">{{ t('common.close') }}</button>
        </div>
      </template>
    </BaseDialog>
    <ConfirmDialog :show="!!pendingAction" :title="pendingAction ? t(`support.actions.${pendingAction}`) : ''" :message="t('support.confirmStateChange')" :danger="pendingAction === 'close'" :confirm-disabled="portal.actionBusy.value" @cancel="pendingAction = null" @confirm="confirmStateChange" />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import SupportAttachments from '@/components/support/SupportAttachments.vue'
import SupportTicketDiscussion from '@/components/support/SupportTicketDiscussion.vue'
import SupportTicketList from '@/components/support/SupportTicketList.vue'
import { priorityKey, ticketDate, ticketMessages, ticketStatus, ticketTypeKey } from '@/components/support/ticketPresentation'
import { useTicketPortal } from '@/composables/useTicketPortal'
import { adminSupportAPI } from '@/api/admin/support'
import { useAppStore } from '@/stores/app'

const { t, locale } = useI18n()
const app = useAppStore()
const portal = useTicketPortal(adminSupportAPI, true)
const showDetail = ref(false)
const selectedId = ref('')
const pendingAction = ref<'resolve' | 'close' | 'reopen' | null>(null)
const refreshing = ref(false)

const stats = computed(() => [
  { key: 'all' as const, label: t('support.stats.all'), icon: 'inbox' as const, color: 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-gray-300' },
  { key: 'pending_agent' as const, label: t('support.stats.pendingAgent'), icon: 'clock' as const, color: 'bg-orange-50 text-orange-600 dark:bg-orange-950/30 dark:text-orange-300' },
  { key: 'pending_user' as const, label: t('support.admin.pendingUser'), icon: 'chat' as const, color: 'bg-blue-50 text-blue-600 dark:bg-blue-950/30 dark:text-blue-300' },
  { key: 'resolved' as const, label: t('support.stats.resolved'), icon: 'checkCircle' as const, color: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/30 dark:text-emerald-300' },
  { key: 'closed' as const, label: t('support.stats.closed'), icon: 'xCircle' as const, color: 'bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-gray-400' },
])
const typeOptions = computed(() => [
  { value: 'all', label: t('support.allTypes') },
  { value: 'pre-sale', label: t('support.types.preSale') },
  { value: 'order', label: t('support.types.order') },
  { value: 'after-sale', label: t('support.types.afterSale') },
  { value: 'technical', label: t('support.types.technical') },
  { value: 'other', label: t('support.types.other') },
])
const statusOptions = computed(() => [
  { value: 'all', label: t('support.admin.allStatuses') },
  { value: 'pending_agent', label: t('support.stats.pendingAgent') },
  { value: 'pending_user', label: t('support.admin.pendingUser') },
  { value: 'resolved', label: t('support.stats.resolved') },
  { value: 'closed', label: t('support.stats.closed') },
])
const listItems = computed(() => portal.items.value.map((record) => ({ ...record, typeLabel: t(ticketTypeKey(record.type)), relatedOrderLabel: record.order_id || undefined })))
const stateActions = computed(() => (['resolve', 'close', 'reopen'] as const).filter((action) => portal.detail.value?.allowed_actions?.includes(action)))
const originalAttachments = computed(() => {
  const replyIds = new Set(portal.detail.value?.replies?.flatMap((reply) => reply.attachments.map((file) => file.id)))
  return portal.detail.value?.attachments?.filter((file) => !replyIds.has(file.id)) || []
})

async function refresh() {
  if (refreshing.value) return
  refreshing.value = true
  try { await portal.refresh() } finally { refreshing.value = false }
}
function setStatus(value: string) { portal.status.value = value }
function openDetail(id: string) { selectedId.value = id; showDetail.value = true; void portal.open(id) }
function closeDetail() { if (portal.actionBusy.value || portal.replyBusy.value) return; showDetail.value = false; pendingAction.value = null; portal.closeDetail() }
function uploadReplyAttachment(file: File) {
  if (!portal.detail.value?.allowed_actions?.includes('reply')) return Promise.reject(new Error('Reply is not allowed'))
  return adminSupportAPI.uploadAttachment(file, portal.detail.value.id)
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
function formatDate(value: string) { return ticketDate(value, locale.value) || t('support.notProvided') }
onMounted(refresh)
</script>
