import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import SupportConversation from '../SupportConversation.vue'

const support = vi.hoisted(() => ({
  turns: [] as Array<{ role: string; content: string; citations?: Array<{ source: string }> }>,
  draft: { title: 'Draft title', description: 'Draft description', product: 'Codex', priority: 'normal' },
  ticketId: null as string | null, error: '', loading: false, hasDraft: true,
  send: vi.fn(), confirmTicket: vi.fn(),
}))
const app = vi.hoisted(() => ({ showSuccess: vi.fn() }))
const auth = vi.hoisted(() => ({ user: { id: 42 } }))
vi.mock('@/stores/support', () => ({ useSupportStore: () => reactive(support) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => reactive(auth) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key, te: () => false }),
}))
enableAutoUnmount(afterEach)
const mountConversation = (restoring = false) => mount(SupportConversation, {
  props: { restoring }, global: { stubs: { Icon: true } },
})

beforeEach(() => {
  vi.resetAllMocks()
  support.turns = []
  support.error = ''
  support.loading = false
  support.hasDraft = true
  support.ticketId = null
  reactive(auth).user = { id: 42 }
})

describe('SupportConversation', () => {
  it('never reports success when confirmation fails', async () => {
    support.confirmTicket.mockResolvedValueOnce(false)
    const wrapper = mountConversation()
    await wrapper.get('[data-testid="confirm-draft"]').trigger('click')
    await flushPromises()
    expect(app.showSuccess).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="confirm-draft"]').exists()).toBe(true)
  })

  it('reports a confirmed successful submission', async () => {
    support.confirmTicket.mockResolvedValueOnce(true)
    const wrapper = mountConversation()
    await wrapper.get('[data-testid="confirm-draft"]').trigger('click')
    await flushPromises()
    expect(support.confirmTicket).toHaveBeenCalledWith(true)
    expect(app.showSuccess).toHaveBeenCalledWith('support.created')
  })

  it('preserves failed input for retry and clears it only after successful sending', async () => {
    support.send.mockResolvedValueOnce(false).mockResolvedValueOnce(true)
    const wrapper = mountConversation()
    await wrapper.get('textarea').setValue('A question for AI')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('A question for AI')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
  })

  it('keeps evidence citations and identifies the AI conversation', () => {
    support.turns = [{ role: 'assistant', content: 'An answer', citations: [{ source: 'Configuration guide' }] }]
    const wrapper = mountConversation()
    expect(wrapper.text()).toContain('Configuration guide')
    expect(wrapper.text()).toContain('support.aiConversationHint')
  })

  it('disables sending and confirmation during restoration', async () => {
    const wrapper = mountConversation(true)
    expect(wrapper.get('[data-testid="confirm-draft"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('textarea').attributes('disabled')).toBeDefined()
    await wrapper.get('form').trigger('submit')
    expect(support.send).not.toHaveBeenCalled()
  })

  it('clears unsent text when the account changes', async () => {
    const wrapper = mountConversation()
    await wrapper.get('textarea').setValue('Private unsent question')
    reactive(auth).user = { id: 43 }
    await flushPromises()
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
  })

  it('disables draft submission after a restore error and offers refresh', async () => {
    support.error = 'support.errors.loadThread'
    const wrapper = mountConversation()
    expect(wrapper.get('[data-testid="confirm-draft"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[aria-label="common.refresh"]').trigger('click')
    expect(wrapper.emitted('refresh')).toHaveLength(1)
    expect(support.confirmTicket).not.toHaveBeenCalled()
  })
})
