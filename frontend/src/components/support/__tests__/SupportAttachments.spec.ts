import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import SupportAttachmentPicker from '../SupportAttachmentPicker.vue'
import SupportAttachments from '../SupportAttachments.vue'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key }),
}))
enableAutoUnmount(afterEach)
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); vi.useRealTimers() })
const global = { stubs: { Icon: true } }
const attachment = { id: 'upload-1', filename: 'example.txt', size: 5, content_type: 'text/plain' }
async function selectFiles(wrapper: ReturnType<typeof mount>, files: File[]) {
  Object.defineProperty(wrapper.get('input').element, 'files', { configurable: true, value: files })
  await wrapper.get('input').trigger('change')
  await flushPromises()
}

describe('Ticket attachments', () => {
  it('does not accept files before upload is wired', () => {
    const wrapper = mount(SupportAttachmentPicker, { global })
    expect(wrapper.get('input').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('support.attachmentUnavailable')
  })

  it('rejects invalid types and oversized files before making a request', async () => {
    const uploadFile = vi.fn()
    const wrapper = mount(SupportAttachmentPicker, { props: { uploadFile }, global })
    await selectFiles(wrapper, [new File(['secret'], 'auth.json', { type: 'application/json' })])
    expect(uploadFile).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('support.validation.attachmentType')
    const large = new File(['a'], 'large.txt', { type: 'text/plain' })
    Object.defineProperty(large, 'size', { value: 10 * 1024 * 1024 + 1 })
    await selectFiles(wrapper, [large])
    expect(uploadFile).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('support.validation.attachmentSize')
  })

  it('keeps failed files retryable and only publishes successful attachment IDs', async () => {
    const uploadFile = vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce(attachment)
    const wrapper = mount(SupportAttachmentPicker, { props: { uploadFile }, global })
    await selectFiles(wrapper, [new File(['hello'], 'example.txt', { type: 'text/plain' })])
    expect(wrapper.text()).toContain('support.errors.uploadAttachment')
    expect(wrapper.emitted('change')!.at(-1)).toEqual([[]])
    expect(wrapper.emitted('pending')!.at(-1)).toEqual([true])
    await wrapper.get('[aria-label="common.refresh"]').trigger('click')
    await flushPromises()
    expect(wrapper.emitted('change')!.at(-1)).toEqual([[attachment.id]])
    expect(wrapper.emitted('pending')!.at(-1)).toEqual([false])
  })

  it('enforces the remaining ticket attachment count and supports removal', async () => {
    const uploadFile = vi.fn().mockResolvedValue(attachment)
    const wrapper = mount(SupportAttachmentPicker, { props: { uploadFile, maxFiles: 1 }, global })
    await selectFiles(wrapper, [new File(['hello'], 'first.txt', { type: 'text/plain' }), new File(['hello'], 'second.txt', { type: 'text/plain' })])
    expect(uploadFile).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('support.validation.attachmentCount')
    await wrapper.get('[aria-label="common.remove"]').trigger('click')
    expect(wrapper.emitted('change')!.at(-1)).toEqual([[]])
  })

  it('ignores uploads completed after a form reset', async () => {
    let resolve!: (value: typeof attachment) => void
    const uploadFile = vi.fn(() => new Promise<typeof attachment>((done) => { resolve = done }))
    const wrapper = mount(SupportAttachmentPicker, { props: { uploadFile }, global })
    await selectFiles(wrapper, [new File(['hello'], 'example.txt', { type: 'text/plain' })])
    await wrapper.setProps({ resetKey: 1 })
    resolve(attachment)
    await flushPromises()
    expect(wrapper.find('li').exists()).toBe(false)
    expect(wrapper.emitted('change')!.at(-1)).toEqual([[]])
  })

  it('downloads by ID through the authenticated callback and never follows response URLs', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const createObjectURL = vi.fn(() => 'blob:attachment')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', class extends URL { static createObjectURL = createObjectURL; static revokeObjectURL = revokeObjectURL })
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const downloadFile = vi.fn().mockResolvedValue(new Blob(['hello']))
    const wrapper = mount(SupportAttachments, { props: { attachments: [{ ...attachment, url: 'https://untrusted.invalid/file' }], downloadFile }, global })
    expect(wrapper.find('a').exists()).toBe(false)
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(downloadFile).toHaveBeenCalledWith(attachment.id)
    expect(click).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1000)
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:attachment')
  })

  it('shows download failure rather than reporting success', async () => {
    const wrapper = mount(SupportAttachments, { props: { attachments: [attachment], downloadFile: vi.fn().mockRejectedValue({ status: 403 }) }, global })
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('support.errors.downloadAttachment')
  })
})
