import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AvailableChannelsTable from '../AvailableChannelsTable.vue'
import type { UserAvailableChannel, UserAvailableGroup } from '@/api/channels'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'availableChannels.summary') return `${params?.groups} groups / ${params?.models} models`
        if (key === 'availableChannels.showMoreGroups') return `${params?.count} more`
        return key
      },
    }),
  }
})

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
    copied: { value: false },
  }),
}))

function group(partial: Partial<UserAvailableGroup> & Pick<UserAvailableGroup, 'id' | 'name'>): UserAvailableGroup {
  return {
    platform: 'anthropic',
    subscription_type: 'standard',
    rate_multiplier: 1,
    peak_rate_enabled: false,
    peak_start: '',
    peak_end: '',
    peak_rate_multiplier: 1,
    is_exclusive: false,
    ...partial,
  }
}

const rows: UserAvailableChannel[] = [
  {
    name: 'Primary channel',
    description: 'Fast and reliable access',
    platforms: [
      {
        platform: 'anthropic',
        groups: [
          group({
            id: 1,
            name: 'Exclusive Pro',
            rate_multiplier: 1.2,
            peak_rate_enabled: true,
            peak_start: '08:00',
            peak_end: '10:00',
            peak_rate_multiplier: 1.5,
            is_exclusive: true,
          }),
          group({ id: 2, name: 'Public' }),
        ],
        supported_models: [{ name: 'claude-test', platform: 'anthropic', pricing: null }],
      },
    ],
  },
]

const baseProps = {
  columns: {
    name: 'Channel',
    description: 'Description',
    platform: 'Platform',
    groups: 'Groups and rates',
    supportedModels: 'Models and pricing',
  },
  rows,
  loading: false,
  pricingKeyPrefix: 'availableChannels.pricing',
  noPricingLabel: 'No pricing',
  noModelsLabel: 'No models',
  emptyLabel: 'No channels',
  userGroupRates: { 1: 0.8 },
}

function mountTable(props = {}) {
  return mount(AvailableChannelsTable, {
    props: { ...baseProps, ...props },
    global: {
      plugins: [createPinia()],
      stubs: {
        Icon: { props: ['name'], template: '<i :data-icon="name" />' },
        PlatformIcon: { template: '<i data-platform-icon />' },
        GroupBadge: {
          props: ['name', 'rateMultiplier', 'userRateMultiplier'],
          template:
            '<span data-group-badge>{{ name }}:{{ rateMultiplier }}:{{ userRateMultiplier }}</span>',
        },
        SupportedModelChip: {
          props: ['model', 'noPricingLabel'],
          template: '<span data-model-chip>{{ model.name }}:{{ noPricingLabel }}</span>',
        },
      },
    },
  })
}

describe('AvailableChannelsTable catalog cards', () => {
  it('renders one card per channel instead of a five-column table', () => {
    const wrapper = mountTable()

    expect(wrapper.find('table').exists()).toBe(false)
    expect(wrapper.findAll('[data-testid="channel-card"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('Primary channel')
    expect(wrapper.text()).toContain('Fast and reliable access')
    expect(wrapper.text()).toContain('2 groups / 1 models')
    expect(wrapper.text()).toContain('Groups and rates')
    expect(wrapper.text()).toContain('Models and pricing')
    expect(wrapper.findAll('[data-group-badge]')).toHaveLength(2)
    expect(wrapper.get('[data-group-badge]').text()).toBe('Exclusive Pro:1.2:0.8')
    expect(wrapper.get('[data-icon="clock"]')).toBeTruthy()
    expect(wrapper.text()).toContain('08:00')
    expect(wrapper.get('[data-model-chip]').text()).toBe('claude-test:No pricing')
  })

  it('keeps placeholders when a platform has no groups or models', () => {
    const wrapper = mountTable({
      rows: [
        {
          name: 'Fallback channel',
          description: '',
          platforms: [{ platform: 'openai', groups: [], supported_models: [] }],
        },
      ],
    })

    expect(wrapper.text()).toContain('Fallback channel')
    expect(wrapper.text()).toContain('openai')
    expect(wrapper.text()).toContain('No models')
    expect(wrapper.text()).toContain('-')
  })

  it('collapses long public group lists until expanded', async () => {
    const manyGroups = Array.from({ length: 10 }, (_, i) => group({ id: i + 1, name: `Group ${i + 1}` }))
    const wrapper = mountTable({
      rows: [
        {
          name: 'Busy channel',
          description: '',
          platforms: [{ platform: 'openai', groups: manyGroups, supported_models: [] }],
        },
      ],
    })

    expect(wrapper.findAll('[data-group-badge]')).toHaveLength(8)
    expect(wrapper.text()).toContain('2 more')

    await wrapper.get('button').trigger('click')
    expect(wrapper.findAll('[data-group-badge]')).toHaveLength(10)
    expect(wrapper.text()).toContain('availableChannels.showLessGroups')
  })

  it('provides loading and empty states', async () => {
    const wrapper = mountTable({ loading: true, rows: [] })

    expect(wrapper.get('[data-testid="channels-loading"] [data-icon="refresh"]')).toBeTruthy()

    await wrapper.setProps({ loading: false })

    expect(wrapper.get('[data-testid="channels-empty"]').text()).toContain('No channels')
  })
})
