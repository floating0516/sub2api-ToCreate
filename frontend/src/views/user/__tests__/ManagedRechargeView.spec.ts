import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ManagedRechargeView from '../ManagedRechargeView.vue'
import LdxpShopEmbed from '@/components/payment/LdxpShopEmbed.vue'

const routerPush = vi.hoisted(() => vi.fn())

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRouter: () => ({ push: routerPush }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

describe('ManagedRechargeView', () => {
  it('keeps the legacy route and renders the embedded shop', async () => {
    routerPush.mockReset()
    const wrapper = mount(ManagedRechargeView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
        },
      },
    })

    expect(wrapper.findComponent(LdxpShopEmbed).exists()).toBe(true)

    await wrapper.get('[data-testid="managed-recharge-back"]').trigger('click')
    expect(routerPush).toHaveBeenCalledWith({ path: '/purchase', query: { tab: 'member' } })
  })
})
