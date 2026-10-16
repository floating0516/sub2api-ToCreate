import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import SupportTicketCreateDialog from '../SupportTicketCreateDialog.vue'
import SupportTicketDiscussion from '../SupportTicketDiscussion.vue'
import SupportTicketList from '../SupportTicketList.vue'
import SupportTicketRow from '../SupportTicketRow.vue'
import type { TicketMessage } from '../ticketPresentation'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key, locale: ref('zh-CN') }),
}))
enableAutoUnmount(afterEach)
const global = { stubs: {
  Icon: true,
  BaseDialog: { props: ['show'], emits: ['close'], template: '<div v-if="show" role="dialog"><slot /><slot name="footer" /></div>' },
  Select: { props: ['modelValue', 'options', 'disabled'], emits: ['update:modelValue'], template: '<select :value="modelValue" :disabled="disabled" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>' },
  Pagination: true,
} }
const record = { id: 'ticket-1', title: 'A long real title', description: 'Issue details', product: 'tocreate', priority: 'normal', status: 'pending_agent', created_at: '2026-09-30T00:00:00Z', updated_at: '2026-09-30T00:00:00Z' }
async function fillForm(wrapper: ReturnType<typeof mount>) {
  await wrapper.get('#support-create-title').setValue('  Test issue  ')
  await wrapper.get('#support-create-description').setValue('  These are detailed reproduction steps  ')
}
function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => { resolve = done })
  return { promise, resolve }
}

describe('Ticket creation', () => {
  it('allows a local draft but never submits when no implementation is wired', async () => {
    const wrapper = mount(SupportTicketCreateDialog, { props: { show: true }, global })
    await fillForm(wrapper)
    await wrapper.get('form').trigger('submit')
    expect(wrapper.get('[data-testid="submit-ticket"]').attributes('disabled')).toBeDefined()
    expect(wrapper.emitted('submitted')).toBeUndefined()
    expect(wrapper.text()).toContain('support.directCreateUnavailable')
  })

  it('validates trimmed title and description lengths before calling the API', async () => {
    const submitTicket = vi.fn()
    const wrapper = mount(SupportTicketCreateDialog, { props: { show: true, submitTicket }, global })
    await wrapper.get('#support-create-title').setValue(' abc ')
    await wrapper.get('#support-create-description').setValue('short')
    await wrapper.get('form').trigger('submit')
    expect(submitTicket).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('support.validation.titleLength')
    expect(wrapper.text()).toContain('support.validation.descriptionLength')
  })

  it('requires an order for order issues', async () => {
    const submitTicket = vi.fn()
    const wrapper = mount(SupportTicketCreateDialog, { props: { show: true, submitTicket }, global })
    await fillForm(wrapper)
    await wrapper.get('#support-create-type').setValue('order')
    await wrapper.get('form').trigger('submit')
    expect(submitTicket).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('support.validation.orderRequired')
  })

  it('prevents double submission and only succeeds on an explicit true result', async () => {
    const request = deferred<boolean>()
    const submitTicket = vi.fn(() => request.promise)
    const wrapper = mount(SupportTicketCreateDialog, { props: { show: true, submitTicket }, global })
    await fillForm(wrapper)
    await wrapper.get('form').trigger('submit')
    await wrapper.get('form').trigger('submit')
    expect(submitTicket).toHaveBeenCalledTimes(1)
    expect(submitTicket.mock.calls[0]).toEqual([expect.objectContaining({ title: 'Test issue', description: 'These are detailed reproduction steps' }), expect.any(String)])
    expect(wrapper.get('[data-testid="submit-ticket"]').attributes('disabled')).toBeDefined()
    request.resolve(true)
    await flushPromises()
    expect(wrapper.emitted('submitted')).toHaveLength(1)
    expect((wrapper.get('#support-create-title').element as HTMLInputElement).value).toBe('')
  })

  it('keeps the draft and idempotency key after failure; editing changes the key', async () => {
    const submitTicket = vi.fn().mockRejectedValueOnce(new Error('timeout')).mockResolvedValue(false)
    const wrapper = mount(SupportTicketCreateDialog, { props: { show: true, submitTicket }, global })
    await fillForm(wrapper)
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('support.errors.createTicket')
    expect((wrapper.get('#support-create-title').element as HTMLInputElement).value).toBe('  Test issue  ')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(submitTicket.mock.calls[1]![1]).toBe(submitTicket.mock.calls[0]![1])
    await wrapper.get('#support-create-title').setValue('A different issue')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(submitTicket.mock.calls[2]![1]).not.toBe(submitTicket.mock.calls[0]![1])
    expect(wrapper.emitted('submitted')).toBeUndefined()
  })

  it('confirms draft discard for the close button and Escape close event', async () => {
    const wrapper = mount(SupportTicketCreateDialog, { props: { show: true }, global })
    await fillForm(wrapper)
    wrapper.getComponent(BaseDialog).vm.$emit('close')
    await flushPromises()
    expect(wrapper.text()).toContain('support.unsavedDraft')
    expect(wrapper.emitted('close')).toBeUndefined()
    await wrapper.findAll('button').find((button) => button.text() === 'support.keepEditing')!.trigger('click')
    expect((wrapper.get('#support-create-title').element as HTMLInputElement).value).toContain('Test issue')
    wrapper.getComponent(BaseDialog).vm.$emit('close')
    await flushPromises()
    await wrapper.findAll('button').find((button) => button.text() === 'support.discardDraft')!.trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
    expect((wrapper.get('#support-create-title').element as HTMLInputElement).value).toBe('')
  })

  it('ignores a submission response after the account-scoped component unmounts', async () => {
    const request = deferred<boolean>()
    const wrapper = mount(SupportTicketCreateDialog, { props: { show: true, submitTicket: () => request.promise }, global })
    await fillForm(wrapper)
    await wrapper.get('form').trigger('submit')
    wrapper.unmount()
    request.resolve(true)
    await flushPromises()
    expect(wrapper.emitted('submitted')).toBeUndefined()
  })
})

describe('Ticket discussion', () => {
  it('never invents an empty timeline when it is not connected', () => {
    const wrapper = mount(SupportTicketDiscussion, { global })
    expect(wrapper.text()).toContain('support.replyUnavailable')
    expect(wrapper.text()).not.toContain('support.noReplies')
    expect(wrapper.get('[data-testid="send-reply"]').attributes('disabled')).toBeDefined()
  })

  it('distinguishes user, support, system, and unknown senders and escapes content', () => {
    const messages: TicketMessage[] = ['user', 'staff', 'system', 'unknown'].map((role, index) => ({ id: String(index), role: role as TicketMessage['role'], content: '<img onerror="evil()">', createdAt: 'invalid-date' }))
    const wrapper = mount(SupportTicketDiscussion, { props: { messages, timelineState: 'ready' }, global })
    for (const role of ['user', 'staff', 'system', 'unknown']) expect(wrapper.text()).toContain(`support.messageRoles.${role}`)
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.text()).toContain('<img onerror="evil()">')
    expect(wrapper.text()).toContain('support.notProvided')
  })

  it.each(['closed', 'resolved'] as const)('blocks replies for %s tickets even if a callback is provided', async (state) => {
    const sendReply = vi.fn()
    const wrapper = mount(SupportTicketDiscussion, { props: { [state]: true, canReply: true, sendReply }, global })
    await wrapper.get('form').trigger('submit')
    expect(sendReply).not.toHaveBeenCalled()
    expect(wrapper.get('textarea').attributes('disabled')).toBeDefined()
  })

  it('retains failed replies and reuses the same idempotency key', async () => {
    const sendReply = vi.fn().mockRejectedValueOnce(new Error('timeout')).mockResolvedValueOnce(true)
    const wrapper = mount(SupportTicketDiscussion, { props: { canReply: true, sendReply }, global })
    await wrapper.get('textarea').setValue('  Still getting 401  ')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('support.errors.replyTicket')
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toContain('Still getting 401')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(sendReply.mock.calls[1]![2]).toBe(sendReply.mock.calls[0]![2])
    expect(wrapper.emitted('replied')).toHaveLength(1)
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('')
  })

  it('validates reply length and prevents duplicate in-flight replies', async () => {
    const request = deferred<boolean>()
    const sendReply = vi.fn(() => request.promise)
    const wrapper = mount(SupportTicketDiscussion, { props: { canReply: true, sendReply }, global })
    await wrapper.get('form').trigger('submit')
    expect(sendReply).not.toHaveBeenCalled()
    await wrapper.get('textarea').setValue('a'.repeat(8001))
    await wrapper.get('form').trigger('submit')
    expect(sendReply).not.toHaveBeenCalled()
    await wrapper.get('textarea').setValue('Another reply')
    await wrapper.get('form').trigger('submit')
    await wrapper.get('form').trigger('submit')
    expect(sendReply).toHaveBeenCalledTimes(1)
    request.resolve(false)
    await flushPromises()
    expect(wrapper.emitted('replied')).toBeUndefined()
  })
})

describe('Ticket list and row', () => {
  it('separates unavailable, loading, failed, and empty results', async () => {
    const wrapper = mount(SupportTicketList, { props: { state: 'unavailable' }, global })
    expect(wrapper.text()).toContain('support.listNotConnected')
    expect(wrapper.text()).not.toContain('support.emptyTickets')
    await wrapper.setProps({ state: 'loading' })
    expect(wrapper.attributes('aria-busy')).toBe('true')
    await wrapper.setProps({ state: 'error' })
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)
    await wrapper.setProps({ state: 'ready' })
    expect(wrapper.text()).toContain('support.emptyTickets')
    await wrapper.setProps({ filtered: true })
    expect(wrapper.text()).toContain('support.noMatchingTickets')
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('clearFilters')).toHaveLength(1)
  })

  it('does not call an empty out-of-range page an empty account', () => {
    const wrapper = mount(SupportTicketList, { props: { state: 'ready', total: 20, page: 3 }, global })
    expect(wrapper.text()).toContain('support.emptyTicketPage')
    expect(wrapper.text()).not.toContain('support.emptyTickets')
  })

  it('preserves long titles and renders unknown metadata safely', async () => {
    const title = 'very-long-ticket-title'.repeat(100)
    const wrapper = mount(SupportTicketRow, { props: { ticket: { ...record, title, status: '__proto__', priority: '__proto__', updated_at: 'invalid' } }, global })
    expect(wrapper.get('h3').text()).toBe(title)
    expect(wrapper.get('h3').classes()).toContain('break-words')
    expect(wrapper.text()).toContain('support.status.unknown')
    expect(wrapper.text()).toContain('support.priorityUnknown')
    expect(wrapper.text()).toContain('support.typeUnavailable')
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('open')).toEqual([[record.id]])
  })
})
