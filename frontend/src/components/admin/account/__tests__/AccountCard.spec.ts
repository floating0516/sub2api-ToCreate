import { describe, expect, it, vi } from 'vitest'
import { shallowMount } from '@vue/test-utils'

import AccountCard from '../AccountCard.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const makeAccount = (overrides: Partial<Account> = {}): Account => ({
  id: 1,
  name: 'API account',
  platform: 'openai',
  type: 'apikey',
  status: 'active',
  schedulable: true,
  created_at: '2026-07-17T00:00:00Z',
  updated_at: '2026-07-17T00:00:00Z',
  credentials: { base_url: 'https://api.example.com/v1' },
  ...overrides
} as Account)

describe('AccountCard API key homepage link', () => {
  it('links API key account names to the upstream origin', () => {
    const wrapper = shallowMount(AccountCard, { props: { account: makeAccount() } })

    const link = wrapper.get('[data-test="account-homepage-link"]')
    expect(link.attributes('href')).toBe('https://api.example.com')
    expect(link.attributes('target')).toBe('_blank')
    expect(link.attributes('rel')).toBe('noopener noreferrer')
  })

  it('does not link OAuth accounts or unsafe URLs', () => {
    const oauth = shallowMount(AccountCard, {
      props: { account: makeAccount({ type: 'oauth' }) }
    })
    const unsafe = shallowMount(AccountCard, {
      props: { account: makeAccount({ credentials: { base_url: 'javascript:alert(1)' } }) }
    })

    expect(oauth.find('[data-test="account-homepage-link"]').exists()).toBe(false)
    expect(unsafe.find('[data-test="account-homepage-link"]').exists()).toBe(false)
  })
})
