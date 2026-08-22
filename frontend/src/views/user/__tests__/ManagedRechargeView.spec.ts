import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ManagedRechargeView from '../ManagedRechargeView.vue'
import type { ManagedRechargeCatalog, ManagedRechargeOrder, ManagedRechargeProduct } from '@/api/managedRecharge'

const routeState = vi.hoisted(() => ({
  query: { plan: 'pro-5x' } as Record<string, unknown>,
}))
const routerPush = vi.hoisted(() => vi.fn())
const getCatalog = vi.hoisted(() => vi.fn())
const listOrders = vi.hoisted(() => vi.fn())
const createOrder = vi.hoisted(() => vi.fn())
const getOrder = vi.hoisted(() => vi.fn())
const getOrderStatus = vi.hoisted(() => vi.fn())
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
  getManagedRechargeOrderStatus: getOrderStatus,
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
  fulfillment_mode: 'proxy',
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
    getOrderStatus.mockReset()
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

  it('purchases an external CDK without Session and opens the redemption dialog', async () => {
    const externalCatalog: ManagedRechargeCatalog = {
      ...catalog,
      mock_mode: true,
      mock_step_seconds: 10,
      fulfillment_mode: 'external',
    }
    const issuedOrder: ManagedRechargeOrder = {
      id: 31,
      order_no: 'MR-EXTERNAL-31',
      user_id: 42,
      product_id: 2,
      product_slug: 'gpt-pro-5x',
      product_name: 'Pro（5 倍）',
      fulfillment_mode: 'external',
      redemption_code: 'MOCK-PRO-SUCCESS-031',
      redemption_url: 'https://redeem.desolate.codes/recharge?cdk=MOCK-PRO-SUCCESS-031',
      price: 8,
      status: 'issued',
      account_email: '',
      paid_at: '2026-08-22T00:00:00Z',
      created_at: '2026-08-22T00:00:00Z',
      updated_at: '2026-08-22T00:00:00Z',
    }
    getCatalog.mockResolvedValue(externalCatalog)
    createOrder.mockResolvedValue(issuedOrder)

    const wrapper = mount(ManagedRechargeView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          BaseDialog: {
            props: ['show', 'title'],
            template: '<div v-if="show"><h2>{{ title }}</h2><slot /><slot name="footer" /></div>',
          },
          Icon: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="managed-recharge-session-input"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('领取专属 CDK')
    expect(wrapper.text()).toContain('模拟 CDK 无法在真实兑换页使用')
    expect(wrapper.get('[data-testid="managed-recharge-step-2"]').text()).toContain('领取 CDK')

    await wrapper.get('[data-testid="managed-recharge-agreement"]').setValue(true)
    await wrapper.get('#managed-recharge-form').trigger('submit')
    await flushPromises()

    expect(createOrder).toHaveBeenCalledWith(2, undefined, expect.any(String))
    expect(wrapper.get('[data-testid="managed-recharge-issued-dialog"]').text()).toContain('MOCK-PRO-SUCCESS-031')
    const redeemLink = wrapper.get('a[href*="MOCK-PRO-SUCCESS-031"]')
    expect(redeemLink.attributes('target')).toBe('_blank')
    expect(redeemLink.text()).toContain('前往兑换')

    wrapper.unmount()
  })

  it('polls external order status without revealing the CDK', async () => {
    const externalCatalog: ManagedRechargeCatalog = {
      ...catalog,
      fulfillment_mode: 'external',
    }
    const issuedOrder: ManagedRechargeOrder = {
      id: 41,
      order_no: 'MR-EXTERNAL-41',
      user_id: 42,
      product_id: 2,
      product_slug: 'gpt-pro-5x',
      product_name: 'Pro（5 倍）',
      fulfillment_mode: 'external',
      price: 8,
      status: 'issued',
      account_email: '',
      progress: 'CDK 已发放，等待前往兑换页提交',
      last_synced_at: '2026-08-22T00:00:00Z',
      paid_at: '2026-08-22T00:00:00Z',
      created_at: '2026-08-22T00:00:00Z',
      updated_at: '2026-08-22T00:00:00Z',
    }
    const queuedOrder = {
      ...issuedOrder,
      status: 'queued',
      progress: '兑换任务已进入处理队列',
      queue_position: 2,
      queue_total: 4,
      last_synced_at: '2026-08-22T00:00:10Z',
    }
    getCatalog.mockResolvedValue(externalCatalog)
    listOrders.mockResolvedValue([issuedOrder])
    getOrderStatus.mockResolvedValue(queuedOrder)

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

    await vi.advanceTimersByTimeAsync(10000)
    await flushPromises()

    expect(getOrderStatus).toHaveBeenCalledWith(41)
    expect(getOrder).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('兑换任务已进入处理队列')
    expect(wrapper.text()).toContain('队列 2/4')
    expect(wrapper.text()).toContain('查看 CDK')

    wrapper.unmount()
  })
})
