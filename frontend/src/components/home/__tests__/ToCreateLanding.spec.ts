import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ToCreateLanding from '@/components/home/ToCreateLanding.vue'
import zhLanding from '@/i18n/locales/zh/landing'

const { getPublicPlans } = vi.hoisted(() => ({
  getPublicPlans: vi.fn(),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getPublicPlans,
  },
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: { n?: number }) => {
        if (key === 'home.pricing.validityDays') {
          return `${params?.n ?? 0} 天`
        }
        const parts = key.split('.')
        let current: unknown = zhLanding
        for (const part of parts) {
          if (current && typeof current === 'object' && part in current) {
            current = (current as Record<string, unknown>)[part]
          } else {
            return key
          }
        }
        return typeof current === 'string' ? current : key
      },
    }),
  }
})

function mountLanding(isAuthenticated = false) {
  return mount(ToCreateLanding, {
    props: {
      siteName: '测试站',
      siteLogo: '',
      siteSubtitle: '',
      docUrl: '',
      isAuthenticated,
      dashboardPath: '/dashboard',
      userInitial: '',
      isDark: false,
      currentYear: 2026,
      showModelPlazaEntry: false,
    },
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="typeof to === \'string\' ? to : \'\'"><slot /></a>',
        },
        LocaleSwitcher: { template: '<div />' },
        Icon: { template: '<span />' },
        EmailFirstAuthDialog: { template: '<div data-testid="email-auth-stub" />' },
      },
    },
  })
}

describe('ToCreateLanding pricing', () => {
  beforeEach(() => {
    getPublicPlans.mockReset().mockResolvedValue({
      data: [
        {
          id: 20,
          name: 'GPT Pro 日卡',
          description: '低门槛体验档，1 天内最多可用 $50 额度。',
          price: 6.9,
          validity_days: 1,
          validity_unit: 'days',
          features: ['1 天有效，最多可用 $50 额度', '支付宝按人民币结算'],
          sort_order: 10,
        },
        {
          id: 22,
          name: 'GPT Pro 标准周卡',
          description: '推荐档，适合一周连续开发，最高可用 $350 额度。',
          price: 39.9,
          validity_days: 7,
          validity_unit: 'days',
          features: ['7 天有效，每日 $50、每周 $350'],
          sort_order: 30,
        },
      ],
    })
  })

  it('renders live plan prices and the recommended weekly plan', async () => {
    const wrapper = mountLanding()
    await flushPromises()

    expect(wrapper.text()).toContain('当前在售价格')
    expect(wrapper.text()).toContain('GPT Pro 日卡')
    expect(wrapper.text()).toContain('¥6.90')
    expect(wrapper.text()).toContain('GPT Pro 标准周卡')
    expect(wrapper.text()).toContain('¥39.90')
    expect(wrapper.text()).toContain('推荐')
    expect(wrapper.text()).not.toContain('支付宝按人民币结算')
  })

  it('sends signed-in visitors to the matching purchase plan', async () => {
    const wrapper = mountLanding(true)
    await flushPromises()

    const buyLinks = wrapper.findAll('.tc-plan-action')
    expect(buyLinks.some((link) => link.attributes('href') === '/purchase?tab=subscription&plan_id=22')).toBe(true)
  })
})
