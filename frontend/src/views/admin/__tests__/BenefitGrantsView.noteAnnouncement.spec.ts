import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import BenefitGrantsView from '../BenefitGrantsView.vue'

const { getAllGroups, listCampaigns } = vi.hoisted(() => ({
  getAllGroups: vi.fn(),
  listCampaigns: vi.fn()
}))

const translations: Record<string, string> = {
  'admin.benefitGrants.notes.templates.activeReward': 'Active reward note',
  'admin.benefitGrants.notes.templates.serviceCompensation': 'Service compensation note',
  'admin.benefitGrants.announcement.templates.activeReward.title': 'Active reward title',
  'admin.benefitGrants.announcement.templates.activeReward.content': 'Active reward content',
  'admin.benefitGrants.announcement.templates.serviceCompensation.title': 'Service compensation title',
  'admin.benefitGrants.announcement.templates.serviceCompensation.content': 'Service compensation content'
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => translations[key] ?? key
    })
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      getAll: getAllGroups
    },
    benefitGrants: {
      list: listCampaigns
    }
  }
}))

const SelectStub = {
  inheritAttrs: false,
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: `
    <select
      v-bind="$attrs"
      :value="modelValue"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option value=""></option>
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
  `
}

function mountView() {
  return mount(BenefitGrantsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        BaseDialog: { template: '<div><slot /></div>' },
        DataTable: true,
        Pagination: true,
        Select: SelectStub,
        Icon: true
      }
    }
  })
}

describe('BenefitGrantsView note announcement templates', () => {
  beforeEach(() => {
    getAllGroups.mockReset().mockResolvedValue([])
    listCampaigns.mockReset().mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0
    })
  })

  it('enables and generates an announcement when a campaign record is selected', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="note-template-select"]').setValue('active_reward')

    expect((wrapper.get('[data-test="activity-notes"]').element as HTMLTextAreaElement).value)
      .toBe('Active reward note')
    expect(wrapper.get('[data-test="announcement-toggle"]').attributes('aria-checked')).toBe('true')
    expect((wrapper.get('[data-test="announcement-title"]').element as HTMLInputElement).value)
      .toBe('Active reward title')
    expect((wrapper.get('[data-test="announcement-content"]').element as HTMLTextAreaElement).value)
      .toBe('Active reward content')
  })

  it('regenerates for another preset but preserves manual copy for custom records', async () => {
    const wrapper = mountView()
    await flushPromises()
    const templateSelect = wrapper.get('[data-test="note-template-select"]')

    await templateSelect.setValue('active_reward')
    await wrapper.get('[data-test="announcement-title"]').setValue('Manual title')
    await wrapper.get('[data-test="announcement-content"]').setValue('Manual content')
    await wrapper.get('[data-test="announcement-toggle"]').trigger('click')

    await templateSelect.setValue('service_compensation')

    expect(wrapper.get('[data-test="announcement-toggle"]').attributes('aria-checked')).toBe('true')
    expect((wrapper.get('[data-test="announcement-title"]').element as HTMLInputElement).value)
      .toBe('Service compensation title')
    expect((wrapper.get('[data-test="announcement-content"]').element as HTMLTextAreaElement).value)
      .toBe('Service compensation content')

    await wrapper.get('[data-test="announcement-title"]').setValue('Custom title')
    await wrapper.get('[data-test="announcement-content"]').setValue('Custom content')
    await templateSelect.setValue('custom')

    expect((wrapper.get('[data-test="announcement-title"]').element as HTMLInputElement).value)
      .toBe('Custom title')
    expect((wrapper.get('[data-test="announcement-content"]').element as HTMLTextAreaElement).value)
      .toBe('Custom content')
  })
})
