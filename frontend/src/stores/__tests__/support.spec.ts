import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { reactive } from 'vue'
import { useSupportStore } from '@/stores/support'

const authStore = vi.hoisted(() => ({
  user: { id: 42 } as { id: number } | null,
}))

const supportMocks = vi.hoisted(() => ({
  chat: vi.fn(),
  confirmTicket: vi.fn(),
  getThread: vi.fn(),
  listThreads: vi.fn(),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => reactive(authStore),
}))

vi.mock('@/api/support', () => ({
  supportAPI: supportMocks,
}))

const draft = {
  title: 'Codex returns 401',
  description: 'The request fails with 401.',
  priority: 'normal',
  product: 'Codex',
}

describe('useSupportStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.resetAllMocks()
    supportMocks.listThreads.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    reactive(authStore).user = { id: 42 }
  })

  it('restores an unconfirmed ticket draft after refresh', async () => {
    localStorage.setItem('support.thread.42', '7d5584bd-22d4-45af-9971-d1019f36a45d')
    supportMocks.getThread.mockResolvedValue({
      thread_id: '7d5584bd-22d4-45af-9971-d1019f36a45d',
      turns: [],
      ticket_draft: draft,
      ticket_id: null,
      status: 'ticket_drafted',
    })

    const store = useSupportStore()
    await store.refresh()

    expect(store.threadId).toBe('7d5584bd-22d4-45af-9971-d1019f36a45d')
    expect(store.draft).toEqual(draft)
    expect(store.hasDraft).toBe(true)
  })

  it('starts an empty conversation without deleting the saved session or draft', async () => {
    const store = useSupportStore()
    store.threadId = 'old-thread'
    store.turns = [{ role: 'user', content: 'Old question' }]
    store.draft = draft
    store.status = 'ticket_drafted'
    expect(store.startNewThread()).toBe(true)
    expect(store.turns).toEqual([])
    expect(store.draft).toBeNull()
    expect(store.threadId).toBeNull()
    expect(localStorage.getItem('support.thread.42')).toBe('new')
    supportMocks.listThreads.mockResolvedValue({ items: [{ thread_id: 'old-thread' }], total: 1, page: 1, pages: 1 })
    await store.refresh()
    expect(supportMocks.getThread).not.toHaveBeenCalled()
    supportMocks.chat.mockResolvedValue({ thread_id: 'new-thread', answer: 'New answer', citations: [] })
    await store.send('New topic')
    expect(supportMocks.chat.mock.calls[0]![0].thread_id).not.toBe('old-thread')
    supportMocks.getThread.mockResolvedValue({ thread_id: 'old-thread', turns: [{ role: 'user', content: 'Old question' }], ticket_draft: draft, status: 'ticket_drafted' })
    expect(await store.selectThread('old-thread')).toBe(true)
    expect(store.draft).toEqual(draft)
    expect(store.turns).toEqual([expect.objectContaining({ content: 'Old question' })])
  })

  it('restores the latest server conversation on a device with no saved ID', async () => {
    supportMocks.listThreads.mockResolvedValue({ items: [{ thread_id: 'latest-thread' }], total: 1, page: 1, pages: 1 })
    supportMocks.getThread.mockResolvedValue({ thread_id: 'latest-thread', turns: [{ role: 'user', content: 'Server history' }] })
    const store = useSupportStore()
    await store.refresh()
    expect(store.threadId).toBe('latest-thread')
    expect(store.turns[0]!.content).toBe('Server history')
    expect(localStorage.getItem('support.thread.42')).toBe('latest-thread')
  })

  it('preserves the current conversation if switching fails and blocks navigation during sending', async () => {
    const store = useSupportStore()
    store.threadId = 'current-thread'
    store.draft = draft
    store.turns = [{ role: 'user', content: 'Current question' }]
    localStorage.setItem('support.thread.42', 'current-thread')
    supportMocks.getThread.mockRejectedValue({ status: 403 })
    expect(await store.selectThread('other-thread')).toBe(false)
    expect(store.threadId).toBe('current-thread')
    expect(store.draft).toEqual(draft)
    expect(store.switchError).toBe('support.errors.switchThread')
    expect(localStorage.getItem('support.thread.42')).toBe('current-thread')
    store.loading = true
    expect(store.startNewThread()).toBe(false)
    expect(await store.selectThread('other-thread')).toBe(false)
    expect(supportMocks.getThread).toHaveBeenCalledOnce()
  })

  it('ignores late session and history responses after an account change', async () => {
    let finishHistory!: (result: object) => void
    let finishThread!: (result: object) => void
    supportMocks.listThreads.mockImplementation(() => new Promise(resolve => { finishHistory = resolve }))
    supportMocks.getThread.mockImplementation(() => new Promise(resolve => { finishThread = resolve }))
    const store = useSupportStore()
    const history = store.loadHistory()
    const switching = store.selectThread('private-thread')
    reactive(authStore).user = { id: 43 }
    finishThread({ thread_id: 'private-thread', turns: [{ role: 'user', content: 'Private' }] })
    finishHistory({ items: [{ thread_id: 'private-thread' }], total: 1, page: 1, pages: 1 })
    expect(await history).toBe(false)
    expect(await switching).toBe(false)
    expect(store.threads).toEqual([])
    expect(store.turns).toEqual([])
    expect(localStorage.getItem('support.thread.43')).toBeNull()
  })

  it('does not let a history failure block the stored conversation', async () => {
    localStorage.setItem('support.thread.42', 'current-thread')
    supportMocks.listThreads.mockRejectedValue({ status: 503 })
    supportMocks.getThread.mockResolvedValue({ thread_id: 'current-thread', turns: [{ role: 'user', content: 'Existing question' }] })
    const store = useSupportStore()
    await store.refresh()
    expect(store.historyError).toBe('support.errors.loadHistory')
    expect(store.error).toBe('')
    expect(store.turns[0]!.content).toBe('Existing question')
  })

  it('restores the current conversation while a slow history request is pending', async () => {
    localStorage.setItem('support.thread.42', 'current-thread')
    let finishHistory!: (result: object) => void
    supportMocks.listThreads.mockImplementation(() => new Promise(resolve => { finishHistory = resolve }))
    supportMocks.getThread.mockResolvedValue({ thread_id: 'current-thread', turns: [{ role: 'user', content: 'Existing question' }] })
    const store = useSupportStore()
    await store.refresh()
    expect(store.turns[0]!.content).toBe('Existing question')
    expect(store.switching).toBe(false)
    expect(store.historyLoading).toBe(true)
    finishHistory({ items: [], total: 0, page: 1, pages: 0 })
  })

  it.each(['answer', 'clarify', 'refuse'] as const)('preserves %s decisions and excerpts on send and refresh', async (action) => {
    const evidence = {
      tool_events: [{ name: "search_knowledge", status: "completed", source_count: 2 }],
      evidence_action: action,
      evidence_reason_code: action === 'answer' ? 'supported' : 'specific_cause_not_established',
      supported_source_ids: action === 'answer' ? ['guide:1'] : [],
      missing_information: action === 'clarify' ? ['Client version', 'API endpoint'] : [],
      citations: [{ source: 'guide.md', chunk_id: 'guide:1', excerpt: 'Original document text' }],
    }
    supportMocks.chat.mockResolvedValueOnce({ thread_id: 'evidence-thread', answer: 'Response', ...evidence })
    const store = useSupportStore()
    expect(await store.send('Question')).toBe(true)
    expect(store.turns[1]).toMatchObject(evidence)
    supportMocks.getThread.mockResolvedValueOnce({
      thread_id: 'evidence-thread',
      turns: [{ role: 'assistant', content: 'Response', ...evidence }],
    })
    await store.refresh()
    expect(store.turns[0]).toMatchObject(evidence)
  })

  it('recovers diagnostic fields after a timeout', async () => {
    supportMocks.chat.mockImplementation(async (payload: { thread_id: string; client_message_id: string }) => {
      supportMocks.getThread.mockResolvedValueOnce({ thread_id: payload.thread_id, turns: [
        { role: 'assistant', content: 'Need details', reply_to: payload.client_message_id,
          evidence_action: 'clarify', evidence_reason_code: 'specific_cause_not_established',
          missing_information: ['Client version'], supported_source_ids: [] },
      ] })
      throw { status: 504 }
    })
    const store = useSupportStore()
    expect(await store.send('Question')).toBe(true)
    expect(store.turns[0]).toMatchObject({ evidence_action: 'clarify', missing_information: ['Client version'] })
    expect(store.pendingMessage).toBeNull()
  })

  it('removes a stale stored thread after a confirmed 404', async () => {
    localStorage.setItem('support.thread.42', '7d5584bd-22d4-45af-9971-d1019f36a45d')
    supportMocks.getThread.mockRejectedValue({ status: 404 })

    const store = useSupportStore()
    await store.refresh()

    expect(store.threadId).toBeNull()
    expect(localStorage.getItem('support.thread.42')).toBeNull()
  })

  it('reuses the message and thread IDs when retrying after a network error', async () => {
    supportMocks.chat
      .mockRejectedValueOnce({ status: 0 })
      .mockResolvedValueOnce({
        thread_id: 'server-thread',
        answer: 'Recovered answer',
        citations: [],
        status: 'answered',
      })
    supportMocks.getThread.mockRejectedValue({ status: 404 })

    const store = useSupportStore()
    await store.send('Codex returns 401')
    const firstPayload = supportMocks.chat.mock.calls[0][0]

    await store.send('Codex returns 401')
    const retryPayload = supportMocks.chat.mock.calls[1][0]

    expect(supportMocks.getThread).toHaveBeenCalledWith(firstPayload.thread_id)
    expect(retryPayload.client_message_id).toBe(firstPayload.client_message_id)
    expect(retryPayload.thread_id).toBe(firstPayload.thread_id)
    expect(store.pendingMessage).toBeNull()
  })

  it('reconciles a timed-out request from the server before offering a retry', async () => {
    supportMocks.chat.mockImplementation(async (payload: { client_message_id: string; thread_id: string }) => {
      supportMocks.getThread.mockResolvedValue({
        thread_id: payload.thread_id,
        turns: [
          {
            role: 'user',
            content: 'Codex returns 401',
            client_message_id: payload.client_message_id,
          },
          {
            role: 'assistant',
            content: 'Check the configured token.',
            reply_to: payload.client_message_id,
          },
        ],
        status: 'answered',
      })
      throw { status: 504 }
    })

    const store = useSupportStore()
    await store.send('Codex returns 401')

    expect(supportMocks.getThread).toHaveBeenCalledOnce()
    expect(store.turns).toHaveLength(2)
    expect(store.turns[1].content).toBe('Check the configured token.')
    expect(store.pendingMessage).toBeNull()
    expect(store.error).toBe('')
  })

  it.each([
    [true, 'ticket_submitted', 'ticket-123'] as const,
    [false, 'ticket_cancelled', null] as const,
  ])('clears the draft after confirm=%s', async (confirm, status, ticketId) => {
    supportMocks.confirmTicket.mockResolvedValue({
      status,
      ticket_id: ticketId,
    })
    const store = useSupportStore()
    store.threadId = '7d5584bd-22d4-45af-9971-d1019f36a45d'
    store.draft = draft
    store.status = 'ticket_drafted'

    expect(await store.confirmTicket(confirm)).toBe(true)

    expect(supportMocks.confirmTicket).toHaveBeenCalledWith({
      thread_id: '7d5584bd-22d4-45af-9971-d1019f36a45d',
      confirm,
    })
    expect(store.draft).toBeNull()
    expect(store.status).toBe(status)
    expect(store.ticketId).toBe(ticketId)
  })

  it('keeps the draft and returns false when confirmation fails', async () => {
    supportMocks.confirmTicket.mockRejectedValueOnce({ status: 503 })
    const store = useSupportStore()
    store.threadId = 'current-thread'
    store.draft = draft
    store.status = 'ticket_drafted'

    expect(await store.confirmTicket(true)).toBe(false)
    expect(store.draft).toEqual(draft)
    expect(store.ticketId).toBeNull()
    expect(store.error).toBe('support.errors.ticketAction')
    expect(store.loading).toBe(false)
  })

  it.each([
    { status: 'ticket_submitted', ticket_id: null },
    { status: 'ticket_drafted', ticket_id: 'ticket-123' },
  ])('rejects an incomplete confirmation response: %j', async (response) => {
    supportMocks.confirmTicket.mockResolvedValueOnce(response)
    const store = useSupportStore()
    store.threadId = 'current-thread'
    store.draft = draft
    store.status = 'ticket_drafted'

    expect(await store.confirmTicket(true)).toBe(false)
    expect(store.draft).toEqual(draft)
    expect(store.ticketId).toBeNull()
    expect(store.error).toBe('support.errors.ticketAction')
  })

  it('does not submit a draft without a thread or while busy', async () => {
    const store = useSupportStore()
    expect(await store.confirmTicket(true)).toBe(false)
    store.threadId = 'current-thread'
    store.loading = true
    expect(await store.confirmTicket(true)).toBe(false)
    expect(supportMocks.confirmTicket).not.toHaveBeenCalled()
  })

  it('clears a restoration error after a successful retry', async () => {
    localStorage.setItem('support.thread.42', 'current-thread')
    supportMocks.getThread.mockRejectedValueOnce({ status: 503 })
    const store = useSupportStore()
    await store.refresh()
    expect(store.error).toBe('support.errors.loadThread')

    supportMocks.getThread.mockResolvedValueOnce({ thread_id: 'current-thread', turns: [] })
    await store.refresh()
    expect(store.error).toBe('')
  })

  it('clears the old conversation and linked ticket when a thread no longer exists', async () => {
    localStorage.setItem('support.thread.42', 'current-thread')
    supportMocks.getThread.mockRejectedValueOnce({ status: 404 })
    const store = useSupportStore()
    store.turns = [{ role: 'user', content: 'An old question' }]
    store.draft = draft
    store.ticketId = 'stale-ticket'
    await store.refresh()

    expect(store.turns).toEqual([])
    expect(store.draft).toBeNull()
    expect(store.ticketId).toBeNull()
  })

  it('clears all visible state when the authenticated account changes', () => {
    const store = useSupportStore()
    store.turns = [{ role: 'user', content: 'Private question' }]
    store.draft = draft
    store.threadId = 'user-42-thread'
    store.ticketId = 'user-42-ticket'
    reactive(authStore).user = { id: 43 }

    expect(store.turns).toEqual([])
    expect(store.draft).toBeNull()
    expect(store.threadId).toBeNull()
    expect(store.ticketId).toBeNull()
  })

  it('ignores an earlier account chat response after switching accounts', async () => {
    let finishRequest!: (result: { thread_id: string; answer: string; citations: never[] }) => void
    supportMocks.chat.mockImplementationOnce(() => new Promise((resolve) => { finishRequest = resolve }))
    const store = useSupportStore()
    const request = store.send('Private question')
    reactive(authStore).user = { id: 43 }
    finishRequest({ thread_id: 'user-42-thread', answer: 'Private answer', citations: [] })

    expect(await request).toBe(false)
    expect(store.turns).toEqual([])
    expect(localStorage.getItem('support.thread.43')).toBeNull()
  })

  it('ignores confirmation from an earlier account and refuses unauthenticated sends', async () => {
    let finishRequest!: (result: { status: string; ticket_id: string }) => void
    supportMocks.confirmTicket.mockImplementationOnce(() => new Promise((resolve) => { finishRequest = resolve }))
    const store = useSupportStore()
    store.threadId = 'user-42-thread'
    store.draft = draft
    const request = store.confirmTicket(true)
    reactive(authStore).user = null
    finishRequest({ status: 'ticket_submitted', ticket_id: 'user-42-ticket' })

    expect(await request).toBe(false)
    expect(store.ticketId).toBeNull()
    expect(await store.send('Another question')).toBe(false)
  })
})
