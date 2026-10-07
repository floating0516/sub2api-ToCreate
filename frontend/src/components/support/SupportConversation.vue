<template>
  <section class="flex min-w-0 flex-col overflow-hidden rounded-3xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
    <header class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
      <div class="min-w-0">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('support.aiConsultation') }}</h2>
        <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('support.aiConversationHint') }}</p>
      </div>
      <div class="flex items-center gap-3">
        <span v-if="store.turns.length && !busy && !store.error" class="text-xs text-gray-500 dark:text-gray-400">{{ t('support.saved') }}</span>
        <button type="button" data-testid="new-conversation" class="btn btn-secondary btn-sm" :disabled="navigationBusy" @click="requestNavigation({ type: 'new' })">
          <Icon name="plus" size="sm" class="mr-2" />{{ t('support.newConversation') }}
        </button>
      </div>
    </header>

    <div class="grid min-w-0 lg:grid-cols-[16rem_minmax(0,1fr)]">
      <aside :aria-label="t('support.conversationHistory')" class="min-w-0 border-b border-gray-200 bg-gray-50/60 p-4 lg:border-b-0 lg:border-r dark:border-dark-700 dark:bg-dark-800/30">
        <div class="mb-3 flex items-center justify-between gap-2">
          <h3 class="text-sm font-semibold text-gray-700 dark:text-gray-200">{{ t('support.conversationHistory') }}</h3>
          <button type="button" data-testid="refresh-history" class="btn btn-ghost btn-sm" :title="t('common.refresh')" :aria-label="t('support.refreshHistory')" :disabled="store.historyLoading" @click="store.loadHistory(store.historyPage)">
            <Icon name="refresh" size="sm" />
          </button>
        </div>
        <p v-if="store.historyError" role="alert" class="mb-3 text-sm text-red-600 dark:text-red-300">{{ t(store.historyError) }}</p>
        <p v-else-if="store.historyLoading && !store.threads?.length" role="status" class="py-4 text-sm text-gray-500">{{ t('common.loading') }}</p>
        <p v-else-if="!store.threads?.length" class="py-4 text-sm text-gray-500 dark:text-gray-400">{{ t('support.noConversations') }}</p>
        <nav v-if="store.threads?.length" :aria-label="t('support.conversationHistory')" class="max-h-48 space-y-2 overflow-y-auto lg:max-h-[38rem]">
          <button v-for="thread in store.threads" :key="thread.thread_id" type="button" data-testid="history-thread"
            :aria-current="thread.thread_id === store.threadId ? 'page' : undefined" :disabled="navigationBusy"
            :class="['w-full rounded-xl border px-3 py-3 text-left transition disabled:cursor-wait', thread.thread_id === store.threadId ? 'border-primary-200 bg-primary-50 dark:border-primary-800 dark:bg-primary-950/40' : 'border-transparent hover:border-gray-200 hover:bg-white dark:hover:border-dark-600 dark:hover:bg-dark-800']"
            @click="requestNavigation({ type: 'select', id: thread.thread_id })">
            <span class="block break-words text-sm font-medium text-gray-800 dark:text-gray-100">{{ thread.title || t('support.untitledConversation') }}</span>
            <time :datetime="thread.updated_at" class="mt-1 block text-xs text-gray-500 dark:text-gray-400">{{ historyDate(thread.updated_at) }}</time>
            <span v-if="thread.has_draft" class="mt-2 inline-block rounded bg-amber-100 px-2 py-0.5 text-xs text-amber-800 dark:bg-amber-950 dark:text-amber-200">{{ t('support.pendingDraft') }}</span>
            <span v-else-if="thread.ticket_id" class="mt-2 inline-block text-xs text-emerald-700 dark:text-emerald-300">{{ t('support.created') }}</span>
          </button>
        </nav>
        <div v-if="store.historyPages > 1" class="mt-3 flex items-center justify-between gap-2 text-xs text-gray-500 dark:text-gray-400">
          <button type="button" class="btn btn-ghost btn-sm" :disabled="store.historyLoading || store.historyPage <= 1" @click="store.loadHistory(store.historyPage - 1)">{{ t('support.previousHistoryPage') }}</button>
          <span>{{ store.historyPage }} / {{ store.historyPages }}</span>
          <button type="button" class="btn btn-ghost btn-sm" :disabled="store.historyLoading || store.historyPage >= store.historyPages" @click="store.loadHistory(store.historyPage + 1)">{{ t('support.nextHistoryPage') }}</button>
        </div>
        <p v-if="store.switchError" role="alert" class="mt-3 text-sm text-red-600 dark:text-red-300">{{ t(store.switchError) }}</p>
      </aside>
      <div class="flex min-w-0 flex-col">
        <div ref="conversation" role="log" :aria-label="t('support.conversation')" aria-live="polite" :aria-busy="busy" class="max-h-[36rem] min-h-64 space-y-5 overflow-y-auto p-5 sm:p-6">
          <p v-if="restoring || store.switching" role="status" class="py-12 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</p>
          <div v-else-if="!store.turns.length" class="py-10 text-center">
            <Icon name="chatBubble" size="xl" class="mx-auto text-primary-500" />
            <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('support.emptyTitle') }}</h3>
            <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('support.emptyExample') }}</p>
          </div>
          <template v-else>
            <article
              v-for="(turn, index) in store.turns"
              :key="`${turn.role}-${turn.client_message_id || turn.reply_to || index}`"
              :class="turn.role === 'user' ? 'ml-4 sm:ml-16' : 'mr-4 sm:mr-16'"
            >
              <p class="mb-1 text-xs text-gray-500 dark:text-gray-400">{{ t(turn.role === 'user' ? 'support.you' : 'support.agent') }}</p>
              <div :class="['rounded-xl px-4 py-3 text-sm leading-6 text-gray-800 dark:text-gray-100', turn.role === 'user' ? 'bg-primary-50 dark:bg-primary-950/40' : 'bg-gray-50 dark:bg-dark-800']">
                <p class="whitespace-pre-wrap break-words">{{ turn.content }}</p>
                <p v-if="turn.generation" class="mt-2 text-xs text-gray-500 dark:text-gray-400" data-testid="answer-generation">{{ t(turn.generation.mode === 'model' ? 'support.modelAnswer' : 'support.localReference') }}</p>
                <details v-if="turn.tool_events?.length" class="mt-3 text-xs text-gray-600 dark:text-gray-300" data-testid="agent-steps">
                  <summary class="cursor-pointer font-medium">{{ t('support.agentSteps') }}</summary>
                  <ol class="mt-2 list-decimal space-y-1 pl-5">
                    <li v-for="(event, step) in turn.tool_events" :key="step">
                      {{ toolLabel(event.name) }} · {{ t(`support.toolStatus.${event.status}`) }}
                    </li>
                  </ol>
                </details>
                <div v-if="turn.evidence_action" class="mt-3 text-xs text-gray-600 dark:text-gray-300" data-testid="evidence-decision">
                  <p class="font-medium">{{ t(`support.decisions.${turn.evidence_action}`) }}</p>
                  <p v-if="turn.evidence_reason_code" class="mt-1">{{ reasonLabel(turn.evidence_reason_code) }}</p>
                  <template v-if="turn.missing_information?.length">
                    <p class="mt-2 font-medium">{{ t('support.missingInformation') }}</p>
                    <ul class="mt-1 list-disc space-y-1 pl-5">
                      <li v-for="item in turn.missing_information" :key="item" class="break-words">{{ item }}</li>
                    </ul>
                  </template>
                </div>
                <p v-else-if="turn.evidence_status" class="mt-3 text-xs text-gray-500 dark:text-gray-400">{{ t('support.evidenceStatus', { status: evidenceLabel(turn.evidence_status) }) }}</p>
                <div v-if="turn.citations?.length" class="mt-3 border-t border-gray-200 pt-3 text-xs dark:border-dark-600">
                  <p class="mb-2 flex items-center gap-2 font-medium">
                    <Icon name="book" size="sm" />{{ t('support.retrievedSources') }}
                  </p>
                  <div v-for="(citation, citationIndex) in turn.citations" :key="citation.chunk_id || citationIndex" class="mb-2 break-words text-gray-500 dark:text-gray-400">
                    {{ citation.source || citation.document_id || t('support.document') }}<span v-if="citation.section"> / {{ citation.section }}</span><span v-if="citation.version"> / {{ citation.version }}</span>
                    <span v-if="citation.chunk_id && turn.supported_source_ids?.includes(citation.chunk_id)" class="ml-2 text-emerald-700 dark:text-emerald-400">{{ t('support.supportedSource') }}</span>
                    <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1">
                      <a v-if="safeDocumentUrl(citation.url)" :href="safeDocumentUrl(citation.url)" target="_blank" rel="noopener noreferrer" class="text-primary-600 underline dark:text-primary-400" data-testid="source-link">{{ t('support.originalDocument') }}</a>
                      <span v-if="citation.fetched_at">{{ t('support.sourceSnapshot') }}: {{ citation.fetched_at.slice(0, 10) }}</span>
                      <span v-if="citation.scope === 'official'">{{ t('support.officialScope') }}</span>
                    </div>
                    <details v-if="citation.excerpt" class="mt-1">
                      <summary class="cursor-pointer text-primary-600 dark:text-primary-400">{{ t('support.viewExcerpt') }}</summary>
                      <blockquote class="mt-2 whitespace-pre-wrap border-l-2 border-gray-300 pl-3 text-gray-600 dark:border-dark-500 dark:text-gray-300">{{ citation.excerpt }}</blockquote>
                    </details>
                  </div>
                </div>
              </div>
            </article>
          </template>
          <p v-if="store.loading" role="status" class="text-sm text-gray-500 dark:text-gray-400">{{ t('support.processing') }}</p>
        </div>

        <div v-if="store.hasDraft" class="border-t border-amber-200 bg-amber-50 px-5 py-4 dark:border-amber-900 dark:bg-amber-950/30">
          <h3 class="flex items-center gap-2 text-sm font-semibold text-amber-900 dark:text-amber-200"><Icon name="inbox" size="sm" />{{ t('support.ticketDraftTitle') }}</h3>
          <p class="mt-3 break-words text-sm font-medium text-amber-950 dark:text-amber-100">{{ store.draft?.title }}</p>
          <p class="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-amber-800 dark:text-amber-300">{{ store.draft?.description }}</p>
          <dl class="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-xs text-amber-800 dark:text-amber-300">
            <div class="min-w-0 break-words"><dt class="inline font-medium">{{ t('support.product') }}: </dt><dd class="inline">{{ store.draft?.product }}</dd></div>
            <div><dt class="inline font-medium">{{ t('support.priority') }}: </dt><dd class="inline">{{ store.draft?.priority }}</dd></div>
          </dl>
          <div class="mt-4 flex flex-col gap-2 sm:flex-row">
            <button
              type="button"
              data-testid="confirm-draft"
              class="btn btn-primary justify-center"
              :disabled="busy"
              @click="confirmDraft(true)"
            >
              <Icon name="check" size="sm" class="mr-2" />{{ t('support.confirmTicket') }}
            </button>
            <button type="button" class="btn btn-secondary justify-center" :disabled="busy" @click="confirmDraft(false)">
              {{ t('support.cancelTicket') }}
            </button>
          </div>
        </div>
        <p v-if="store.ticketId" class="break-words border-t border-emerald-200 bg-emerald-50 px-5 py-3 text-sm text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/30 dark:text-emerald-300">{{ t('support.ticketSubmitted', { id: store.ticketId }) }}</p>
        <div v-if="store.error" role="alert" class="mx-5 mb-3 mt-3 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300">
          <Icon name="exclamationCircle" size="sm" class="mt-0.5 shrink-0" />
          <span>{{ t(store.error) }}</span>
          <button
            v-if="store.error === 'support.errors.loadThread'"
            type="button"
            class="ml-auto shrink-0"
            :title="t('common.refresh')"
            :aria-label="t('common.refresh')"
            :disabled="restoring || store.loading"
            @click="emit('refresh')"
          >
            <Icon name="refresh" size="sm" />
          </button>
        </div>
        <form class="border-t border-gray-200 p-5 dark:border-dark-700" @submit.prevent="submit">
          <label for="support-message" class="sr-only">{{ t('support.formDescription') }}</label>
          <textarea
            id="support-message"
            v-model="message"
            rows="3"
            maxlength="4000"
            :disabled="busy"
            class="input block min-h-28 w-full resize-y"
            :placeholder="t('support.placeholder')"
          />
          <div class="mt-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p class="flex min-w-0 items-start gap-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
              <Icon name="shield" size="sm" class="mt-0.5 shrink-0" />{{ t('support.securityDescription') }}
            </p>
            <button type="submit" class="btn btn-primary shrink-0 justify-center" :disabled="busy || !message.trim()">
              <Icon name="arrowRight" size="sm" class="mr-2" />{{ t(store.loading ? 'support.processing' : 'support.send') }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </section>
  <ConfirmDialog :show="!!pendingNavigation" :title="t('support.switchConversation')" :message="t('support.discardUnsentMessage')" :confirm-text="t('support.discardAndSwitch')" :cancel-text="t('support.keepEditing')" :confirm-disabled="navigationBusy" @cancel="pendingNavigation = null" @confirm="confirmNavigation" />
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { useSupportStore } from '@/stores/support'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{ restoring: boolean }>()
const emit = defineEmits<{ (event: 'refresh'): void }>()
const store = useSupportStore()
const appStore = useAppStore()
const authStore = useAuthStore()
const { t, te, locale } = useI18n()
const message = ref('')
const conversation = ref<HTMLElement | null>(null)
const navigationBusy = computed(() => props.restoring || store.loading || store.switching)
const busy = computed(() => navigationBusy.value || store.error === 'support.errors.loadThread')
type Navigation = { type: 'new' } | { type: 'select'; id: string }
const pendingNavigation = ref<Navigation | null>(null)

watch(() => authStore.user?.id, () => { message.value = ''; pendingNavigation.value = null }, { flush: 'sync' })
watch(() => store.conversationVersion, () => { message.value = '' })

function historyDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? t('support.notProvided') : date.toLocaleString(locale?.value || 'zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}
async function navigate(action: Navigation) {
  const succeeded = action.type === 'new' ? store.startNewThread() : await store.selectThread(action.id)
  if (succeeded) message.value = ''
}
function requestNavigation(action: Navigation) {
  if (navigationBusy.value || (action.type === 'select' && action.id === store.threadId && !store.error)) return
  if (message.value.trim()) pendingNavigation.value = action
  else void navigate(action)
}
async function confirmNavigation() {
  if (!pendingNavigation.value || navigationBusy.value) return
  const action = pendingNavigation.value
  pendingNavigation.value = null
  await navigate(action)
}

function evidenceLabel(status: string) {
  const key = `support.evidence.${status}`
  return te(key) ? t(key) : status
}
function reasonLabel(reason: string) {
  const key = `support.reasons.${reason}`
  return te(key) ? t(key) : t('support.reasons.unknown')
}
function toolLabel(name: string) {
  const allowed = ['search_knowledge', 'create_ticket_draft', 'validate_ticket', 'agent_limit']
  return t(`support.tools.${allowed.includes(name) ? name : 'unknown_tool'}`)
}
function safeDocumentUrl(value?: string): string | undefined {
  if (!value) return undefined
  try {
    const url = new URL(value)
    const hosts = ['developers.openai.com', 'platform.openai.com', 'learn.chatgpt.com', 'docs.anthropic.com', 'platform.claude.com', 'ai.google.dev', 'docs.x.ai']
    return url.protocol === 'https:' && hosts.includes(url.hostname) && !url.username && !url.password ? url.href : undefined
  } catch { return undefined }
}
async function submit() {
  if (busy.value) return
  const sent = await store.send(message.value)
  if (sent) message.value = ''
}
async function confirmDraft(confirm: boolean) {
  if (busy.value) return
  const succeeded = await store.confirmTicket(confirm)
  if (succeeded) appStore.showSuccess(t(confirm ? 'support.created' : 'support.draftCancelled'))
}
watch(() => store.turns.length, async () => {
  await nextTick()
  if (conversation.value) conversation.value.scrollTop = conversation.value.scrollHeight
})
</script>
