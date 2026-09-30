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
