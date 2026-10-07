import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import { supportAPI, type SupportChatResult, type SupportCitation, type SupportEvidenceDecision, type SupportThread, type SupportThreadSummary, type SupportTicketDraft } from '@/api/support'
import { useAuthStore } from './auth'

export interface SupportTurn extends SupportEvidenceDecision {
  role: 'user' | 'assistant'
  content: string
  citations?: SupportCitation[]
  evidence_status?: string
  created_at?: string
  client_message_id?: string
  reply_to?: string
}

interface PendingSupportMessage {
  message: string
  clientMessageId: string
  threadId: string
}

const threadKey = (userId: number | string) => `support.thread.${userId}`
const NEW_THREAD = 'new'

function evidenceDecision(result: SupportEvidenceDecision): SupportEvidenceDecision {
  return {
    generation: result.generation,
    tool_events: Array.isArray(result.tool_events) ? result.tool_events : [],
    evidence_action: result.evidence_action,
    evidence_reason_code: result.evidence_reason_code,
    supported_source_ids: Array.isArray(result.supported_source_ids) ? result.supported_source_ids : [],
    missing_information: Array.isArray(result.missing_information) ? result.missing_information : [],
  }
}

export const useSupportStore = defineStore('support', () => {
  const authStore = useAuthStore()
  const threadId = ref<string | null>(null)
  const turns = ref<SupportTurn[]>([])
  const draft = ref<SupportTicketDraft | null>(null)
  const ticketId = ref<string | null>(null)
  const status = ref<string>('')
  const loading = ref(false)
  const error = ref('')
  const threads = ref<SupportThreadSummary[]>([])
  const historyLoading = ref(false)
  const historyError = ref('')
  const historyPage = ref(1)
  const historyPages = ref(0)
  const historyTotal = ref(0)
  const switching = ref(false)
  const switchError = ref('')
  const conversationVersion = ref(0)
  const pendingMessage = ref<PendingSupportMessage | null>(null)
  const hasDraft = computed(() => !!draft.value && !ticketId.value && status.value === 'ticket_drafted')
  let accountVersion = 0
  let historyVersion = 0

  watch(() => authStore.user?.id, () => {
    accountVersion++
    threadId.value = null
    turns.value = []
    draft.value = null
    ticketId.value = null
    status.value = ''
    pendingMessage.value = null
    error.value = ''
    loading.value = false
    threads.value = []
    historyVersion++
    historyLoading.value = false
    historyError.value = ''
    historyPage.value = 1
    historyPages.value = 0
    historyTotal.value = 0
    switching.value = false
    switchError.value = ''
    conversationVersion.value++
  }, { flush: 'sync' })

  function restoreLocalThread() {
    const userId = authStore.user?.id
    const stored = userId ? localStorage.getItem(threadKey(userId)) : null
    threadId.value = stored === NEW_THREAD ? null : stored
  }

  function persistThread(id: string) {
    threadId.value = id
    const userId = authStore.user?.id
    if (userId) localStorage.setItem(threadKey(userId), id)
  }

  function clearStoredThread() {
    const userId = authStore.user?.id
    if (userId) localStorage.removeItem(threadKey(userId))
    threadId.value = null
    turns.value = []
    draft.value = null
    ticketId.value = null
    status.value = ''
  }

  function startNewThread() {
    if (loading.value || switching.value || !authStore.user?.id) return false
    clearStoredThread()
    localStorage.setItem(threadKey(authStore.user.id), NEW_THREAD)
    pendingMessage.value = null
    error.value = ''
    switchError.value = ''
    conversationVersion.value++
    return true
  }

  async function loadHistory(page = 1) {
    if (!authStore.user?.id) return false
    const account = accountVersion
    const version = ++historyVersion
    historyLoading.value = true
    historyError.value = ''
    try {
      const result = await supportAPI.listThreads({ page, page_size: 20 })
      if (account !== accountVersion || version !== historyVersion) return false
      threads.value = result.items
      historyPage.value = result.page
      historyPages.value = result.pages
      historyTotal.value = result.total
      return true
    } catch {
      if (account === accountVersion && version === historyVersion) historyError.value = 'support.errors.loadHistory'
      return false
    } finally {
      if (account === accountVersion && version === historyVersion) historyLoading.value = false
    }
  }

  async function selectThread(id: string) {
    if (loading.value || switching.value || !authStore.user?.id) return false
    if (id === threadId.value && !error.value) return true
    const version = accountVersion
    switching.value = true
    switchError.value = ''
    try {
      const result = await supportAPI.getThread(id)
      if (version !== accountVersion) return false
      applyThread(result)
      pendingMessage.value = null
      error.value = ''
      conversationVersion.value++
      return true
    } catch {
      if (version === accountVersion) switchError.value = 'support.errors.switchThread'
      return false
    } finally {
      if (version === accountVersion) switching.value = false
    }
  }

  function applyThread(result: SupportThread) {
    persistThread(result.thread_id)
    turns.value = (result.turns || []).map((turn) => ({
      role: turn.role === 'assistant' ? 'assistant' : 'user',
      content: String(turn.content || ''),
      citations: Array.isArray(turn.citations) ? turn.citations : [],
      evidence_status: typeof turn.evidence_status === 'string' ? turn.evidence_status : undefined,
      ...evidenceDecision(turn),
      created_at: typeof turn.created_at === 'string' ? turn.created_at : undefined,
      client_message_id: typeof turn.client_message_id === 'string' ? turn.client_message_id : undefined,
      reply_to: typeof turn.reply_to === 'string' ? turn.reply_to : undefined,
    }))
    draft.value = result.ticket_draft || null
    ticketId.value = result.ticket_id || null
    status.value = result.status || ''
  }

  function removeOptimisticTurn(clientMessageId: string) {
    turns.value = turns.value.filter((turn) => turn.client_message_id !== clientMessageId)
  }

  function isRecoverableRequestError(err: unknown) {
    const statusCode = (err as { status?: number }).status
    return statusCode === 0 || statusCode === 504
  }

  async function recoverMessage(id: string, clientMessageId: string, version: number) {
    try {
      const result = await supportAPI.getThread(id)
      if (version !== accountVersion) return false
      applyThread(result)
      return result.turns.some((turn) =>
        turn.client_message_id === clientMessageId || turn.reply_to === clientMessageId
      )
    } catch (err) {
      if (version === accountVersion && (err as { status?: number }).status === 404) clearStoredThread()
      return false
    }
  }

  async function refresh() {
    if (loading.value || switching.value || !authStore.user?.id) return
    const version = accountVersion
    error.value = ''
    switching.value = true
    try {
      await loadHistory()
      if (version !== accountVersion) return
      const stored = localStorage.getItem(threadKey(authStore.user!.id))
      restoreLocalThread()
      // On a new device restore the newest server session. An explicit new
      // conversation remains empty across refreshes until its first message.
      if (!stored && threads.value.length) threadId.value = threads.value[0]!.thread_id
      if (!threadId.value) return
      const result = await supportAPI.getThread(threadId.value)
      if (version !== accountVersion) return
      applyThread(result)
    } catch (err) {
      if (version !== accountVersion) return
      const statusCode = (err as { status?: number }).status
      if (statusCode === 404) {
        clearStoredThread()
      } else {
        error.value = 'support.errors.loadThread'
      }
    } finally {
      if (version === accountVersion) switching.value = false
    }
  }

  async function send(message: string) {
    const text = message.trim()
    if (!text || loading.value || switching.value || !authStore.user?.id) return false
    const version = accountVersion
    const retry = pendingMessage.value?.message === text ? pendingMessage.value : null
    const requestThreadId = retry?.threadId || threadId.value || crypto.randomUUID()
    const clientMessageId = retry?.clientMessageId || crypto.randomUUID()
    pendingMessage.value = { message: text, clientMessageId, threadId: requestThreadId }
    persistThread(requestThreadId)
    loading.value = true
    error.value = ''
    let completed = false
    turns.value.push({
      role: 'user',
      content: text,
      created_at: new Date().toISOString(),
      client_message_id: clientMessageId,
    })
    try {
      const result: SupportChatResult = await supportAPI.chat({
        message: text,
        client_message_id: clientMessageId,
        thread_id: requestThreadId,
      })
      if (version !== accountVersion) return false
      persistThread(result.thread_id)
      turns.value.push({
        role: 'assistant',
        content: result.answer || '',
        citations: result.citations || [],
        evidence_status: result.evidence_status,
        ...evidenceDecision(result),
        reply_to: clientMessageId,
      })
      draft.value = result.ticket_draft || null
      ticketId.value = result.ticket_id || null
      status.value = result.status || ''
      pendingMessage.value = null
      completed = true
    } catch (err) {
      if (version !== accountVersion) return false
      const recovered = isRecoverableRequestError(err)
        ? await recoverMessage(requestThreadId, clientMessageId, version)
        : false
      if (version !== accountVersion) return false
      if (recovered) {
        pendingMessage.value = null
        completed = true
      } else {
        removeOptimisticTurn(clientMessageId)
        error.value = 'support.errors.unavailable'
      }
    } finally {
      if (version === accountVersion) loading.value = false
    }
    if (completed && version === accountVersion) void loadHistory()
    return completed
  }

  async function confirmTicket(confirm: boolean) {
    if (!threadId.value || loading.value || switching.value || !authStore.user?.id) return false
    const version = accountVersion
    loading.value = true
    error.value = ''
    try {
      const result = await supportAPI.confirmTicket({ thread_id: threadId.value, confirm })
      if (version !== accountVersion) return false
      const expectedStatus = confirm ? 'ticket_submitted' : 'ticket_cancelled'
      if (result.status !== expectedStatus || (confirm && !result.ticket_id)) {
        error.value = 'support.errors.ticketAction'
        return false
      }
      status.value = result.status
      ticketId.value = result.ticket_id || null
      draft.value = null
      void loadHistory(historyPage.value)
      return true
    } catch {
      if (version === accountVersion) error.value = 'support.errors.ticketAction'
      return false
    } finally {
      if (version === accountVersion) loading.value = false
    }
  }

  return { threadId, turns, draft, ticketId, status, loading, error, pendingMessage, hasDraft,
    threads, historyLoading, historyError, historyPage, historyPages, historyTotal, switching, switchError, conversationVersion,
    restoreLocalThread, refresh, send, confirmTicket, startNewThread, loadHistory, selectThread }
})
