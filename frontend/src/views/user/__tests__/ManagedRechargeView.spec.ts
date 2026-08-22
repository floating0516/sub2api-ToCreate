import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ManagedRechargeView from '../ManagedRechargeView.vue'
import type { ManagedRechargeCatalog, ManagedRechargeProduct } from '@/api/managedRecharge'

const routeState = vi.hoisted(() => ({
  query: { plan: 'pro-5x' } as Record<string, unknown>,
}))
const routerPush = vi.hoisted(() => vi.fn())
const getCatalog = vi.hoisted(() => vi.fn())
const listOrders = vi.hoisted(() => vi.fn())
const createOrder = vi.hoisted(() => vi.fn())
const getOrder = vi.hoisted(() => vi.fn())
const submitReplacement = vi.hoisted(() => vi.fn())
const validateSession = vi.hoisted(() => vi.fn())

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({ push: routerPush }),
  }
})

vi.mock('@/api/managedRecharge', () => ({
  getManagedRechargeCatalog: getCatalog,
  listManagedRechargeOrders: listOrders,
  createManagedRechargeOrder: createOrder,
  getManagedRechargeOrder: getOrder,
  submitManagedRechargeReplacementSession: submitReplacement,
  validateManagedRechargeSession: validateSession,
}))

function product(overrides: Partial<ManagedRechargeProduct>): ManagedRechargeProduct {
  return {
    id: 1,
    slug: 'mock-plus',
    plan_type: 'plus',
    name: 'Plus',
    description: '',
    price: 5,
    active: true,
    sort_order: 1,
    available_stock: 2,
    total_stock: 2,
    created_at: '2026-08-22T00:00:00Z',
    updated_at: '2026-08-22T00:00:00Z',
    ...overrides,
  }
}

const catalog: ManagedRechargeCatalog = {
  enabled: true,
  balance: 50,
  mock_mode: false,
  products: [
    product({}),
    product({ id: 2, slug: 'gpt-pro-5x', plan_type: 'pro', name: 'Pro（5 倍）', price: 8 }),
    product({ id: 3, slug: 'gpt-pro-20x', plan_type: 'pro', name: 'Pro（20 倍）', price: 20 }),
  ],
}

describe('ManagedRechargeView purchase flow', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    routeState.query = { plan: 'pro-5x' }
    routerPush.mockReset()
    getCatalog.mockReset().mockResolvedValue(catalog)
    listOrders.mockReset().mockResolvedValue([])
    createOrder.mockReset()
    getOrder.mockReset()
    submitReplacement.mockReset()
    validateSession.mockReset().mockResolvedValue({
      valid: true,
      email: 'member@example.com',
      membership: 'plus',
    })
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
  })

  it('shows the compact Session guide and automatically validates the account', async () => {
    const wrapper = mount(ManagedRechargeView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: true,
          Icon: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.findAll('[data-testid^="managed-recharge-step-"]')).toHaveLength(3)
    expect(wrapper.get('[data-testid="managed-recharge-product-2"]').attributes('aria-pressed')).toBe('true')
    expect(wrapper.get('[data-testid="managed-recharge-session-guide"]').text()).toContain('如何获取 Session？')
    expect(wrapper.get('[data-testid="managed-recharge-session-guide"]').text()).toContain('将页面返回的完整 JSON 全部复制')
    expect(wrapper.get('[data-testid="managed-recharge-session-guide"]').text()).not.toContain('不要上传截图')
    expect(wrapper.text()).toContain('支付后余额')
    expect(wrapper.text()).toContain('42.00')

    await wrapper.get('[data-testid="managed-recharge-session-input"]').setValue(JSON.stringify({
      user: { email: 'member@example.com' },
      accessToken: 'test-token',
    }))

    expect(wrapper.text()).toContain('正在验证 Session')

    await vi.advanceTimersByTimeAsync(500)
    await flushPromises()

    expect(validateSession).toHaveBeenCalledWith(expect.stringContaining('member@example.com'))
    expect(wrapper.get('[data-testid="managed-recharge-session-result"]').text()).toContain('Session 格式有效')
    expect(wrapper.text()).toContain('member@example.com')
    expect(wrapper.text()).toContain('当前订阅')
    expect(wrapper.text()).toContain('Plus')

    await wrapper.get('[data-testid="managed-recharge-back"]').trigger('click')
    expect(routerPush).toHaveBeenCalledWith({ path: '/purchase', query: { tab: 'member' } })

    wrapper.unmount()
  })

  it('keeps the Session link visible in mock mode', async () => {
    getCatalog.mockResolvedValue({ ...catalog, mock_mode: true, mock_step_seconds: 10 })

    const wrapper = mount(ManagedRechargeView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: true,
          Icon: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="managed-recharge-session-guide"]').text()).toContain('chatgpt.com/api/auth/session')
    expect(wrapper.text()).toContain('填入模拟 Session')

    wrapper.unmount()
  })
})
