<template>
  <section :aria-label="t('support.timeline')" class="mt-6 border-t border-gray-200 pt-5 dark:border-dark-700">
    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('support.timeline') }}</h3>
    <p v-if="timelineState === 'unavailable'" role="status" class="mt-3 text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('support.replyUnavailable') }}</p>
    <div v-else-if="timelineState === 'loading'" role="status" class="mt-4 space-y-3">
      <span class="sr-only">{{ t('common.loading') }}</span>
      <div v-for="index in 2" :key="index" aria-hidden="true" class="h-20 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800" />
    </div>
    <div v-else-if="timelineState === 'error'" role="alert" class="mt-4">
      <p class="text-sm text-red-600 dark:text-red-300">{{ t('support.errors.loadTimeline') }}</p>
      <button type="button" class="btn btn-secondary mt-3" @click="emit('retry')"><Icon name="refresh" size="sm" class="mr-2" />{{ t('common.refresh') }}</button>
    </div>
    <template v-else>
      <p v-if="!messages.length" role="status" class="mt-3 text-sm text-gray-500 dark:text-gray-400">{{ t('support.noReplies') }}</p>
      <ol v-else class="mt-4 space-y-5">
        <li v-for="message in messages" :key="message.id" :class="['min-w-0 border-l-2 pl-4', messageBorder(message.role)]">
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
            <span class="font-medium">{{ t(roleKey(message.role)) }}</span>
            <time :datetime="messageDate(message.createdAt) ? message.createdAt : undefined">{{ messageDate(message.createdAt) || t('support.notProvided') }}</time>
          </div>
          <p class="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-gray-700 dark:text-gray-200">{{ message.content }}</p>
          <SupportAttachments v-if="message.attachments?.length" :attachments="message.attachments" :download-file="downloadFile" />
        </li>
      </ol>
    </template>
    <form class="mt-6 border-t border-gray-200 pt-5 dark:border-dark-700" @submit.prevent="reply">
      <label for="support-ticket-reply" class="label">{{ t('support.reply') }}</label>
      <p v-if="closed" id="support-reply-hint" role="status" class="mb-3 text-sm text-gray-500 dark:text-gray-400">{{ t('support.closedReply') }}</p>
      <p v-else-if="resolved" id="support-reply-hint" role="status" class="mb-3 text-sm text-gray-500 dark:text-gray-400">{{ t('support.resolvedReply') }}</p>
      <p v-else-if="!canReply || !sendReply" id="support-reply-hint" class="mb-3 text-sm text-gray-500 dark:text-gray-400">{{ t(sendReply ? 'support.replyNotAllowed' : 'support.replyNotConnected') }}</p>
      <textarea
        id="support-ticket-reply"
        ref="replyInput"
        v-model="text"
        class="input min-h-28"
        :disabled="!enabled || sending"
        :placeholder="t('support.replyPlaceholder')"
        :aria-describedby="!enabled ? 'support-reply-hint' : undefined"
        :aria-invalid="attempted && !validText"
      />
      <p v-if="attempted && !validText" role="alert" class="mt-2 text-sm text-red-600 dark:text-red-300">{{ t('support.validation.replyLength') }}</p>
      <p v-if="error" role="alert" class="mt-2 text-sm text-red-600 dark:text-red-300">{{ t('support.errors.replyTicket') }}</p>
      <p v-if="success" role="status" class="mt-2 text-sm text-emerald-700 dark:text-emerald-300">{{ t('support.replySent') }}</p>
      <div class="mt-3 flex flex-wrap justify-between gap-3">
        <SupportAttachmentPicker :upload-file="enabled ? uploadFile : undefined" :disabled="sending" :max-files="Math.max(0, 5 - attachmentCount)" :reset-key="attachmentReset" @change="attachmentIds = $event" @pending="attachmentsPending = $event" />
        <button type="submit" data-testid="send-reply" :disabled="!enabled || sending || attachmentsPending" class="btn btn-primary self-start"><Icon name="chat" size="sm" class="mr-2" />{{ t(sending ? 'support.processing' : 'support.sendReply') }}</button>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { ticketDate, type TicketMessage } from './ticketPresentation'
import SupportAttachments from './SupportAttachments.vue'
import SupportAttachmentPicker from './SupportAttachmentPicker.vue'
import type { SupportAttachment } from '@/api/support'

const props = withDefaults(defineProps<{
  messages?: TicketMessage[]
  timelineState?: 'unavailable' | 'loading' | 'error' | 'ready'
  closed?: boolean
  resolved?: boolean
  canReply?: boolean
  admin?: boolean
  attachmentCount?: number
  uploadFile?: (file: File) => Promise<SupportAttachment>
  downloadFile?: (id: string) => Promise<Blob>
  sendReply?: (text: string, attachmentIds: string[], messageId: string) => Promise<boolean>
}>(), { messages: () => [], timelineState: 'unavailable', closed: false, resolved: false, canReply: false, admin: false, attachmentCount: 0 })
const emit = defineEmits<{ retry: []; replied: [] }>()
const { t, locale } = useI18n()
const text = ref('')
const sending = ref(false)
const attempted = ref(false)
const error = ref(false)
const success = ref(false)
const replyInput = ref<HTMLTextAreaElement | null>(null)
const enabled = computed(() => !props.closed && !props.resolved && props.canReply && !!props.sendReply)
const validText = computed(() => { const length = Array.from(text.value.trim()).length; return length > 0 && length <= 8000 })
const attachmentIds = ref<string[]>([])
const attachmentsPending = ref(false)
const attachmentReset = ref(0)
let pendingRequest: { payload: string; id: string } | null = null
let disposed = false

function roleKey(role: string) {
  if (props.admin && role === 'user') return 'support.admin.requester'
  const roles: Record<string, string> = { user: 'user', staff: 'staff', system: 'system' }
  return `support.messageRoles.${Object.prototype.hasOwnProperty.call(roles, role) ? roles[role] : 'unknown'}`
}
function messageBorder(role: string) {
  if (role === 'user') return 'border-primary-300 dark:border-primary-700'
  if (role === 'staff') return 'border-blue-300 dark:border-blue-700'
  return 'border-gray-300 dark:border-dark-600'
}
function messageDate(value: string) {
  return ticketDate(value, locale.value)
}
async function reply() {
  if (!enabled.value || sending.value || !props.sendReply || attachmentsPending.value) return
  attempted.value = true
  success.value = false
  if (!validText.value) {
    replyInput.value?.focus()
    return
  }
  sending.value = true
  error.value = false
  try {
    const payload = JSON.stringify({ text: text.value.trim(), attachmentIds: attachmentIds.value })
    if (pendingRequest?.payload !== payload) pendingRequest = { payload, id: crypto.randomUUID() }
    const completed = await props.sendReply(text.value.trim(), [...attachmentIds.value], pendingRequest.id)
    if (disposed) return
    if (completed === true) {
      text.value = ''
      attachmentIds.value = []
      attachmentReset.value++
      pendingRequest = null
      attempted.value = false
      success.value = true
      emit('replied')
    } else error.value = true
  } catch {
    if (!disposed) error.value = true
  } finally {
    if (!disposed) sending.value = false
  }
}
onUnmounted(() => { disposed = true })
</script>
