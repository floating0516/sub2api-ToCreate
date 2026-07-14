import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AddonProductEditDialog from '../AddonProductEditDialog.vue'
import type { SubscriptionAddonProduct } from '@/types/payment'

const { updateAddonProduct, showError, showSuccess } = vi.hoisted(() => ({
  updateAddonProduct: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: { updateAddonProduct },
}))

const product: SubscriptionAddonProduct = {
  id: 2,
  sku: 'addon-usd-30',
  name: '30 美元加油包',
  quota_usd: 30,
  price: 5.49,
  original_price: 6.99,
  for_sale: true,
  sort_order: 20,
}

function mountDialog() {
  return mount(AddonProductEditDialog, {
    props: {
      show: true,
      product,
      paymentConfig: {
        enabled: true,
        min_amount: 0.01,
        max_amount: 1000,
        daily_limit: 1000,
        order_timeout_minutes: 30,
        max_pending_orders: 3,
        enabled_payment_types: ['alipay'],
        balance_disabled: false,
        balance_recharge_multiplier: 1,
        addon_purchase_enabled: false,
        subscription_usd_to_cny_rate: 0,
        recharge_fee_rate: 0,
        load_balance_strategy: 'round_robin',
        product_name_prefix: '',
        product_name_suffix: '',
        help_image_url: '',
        help_text: '',
      },
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        Icon: true,
      },
    },
  })
}

describe('AddonProductEditDialog', () => {
  beforeEach(() => {
    updateAddonProduct.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    updateAddonProduct.mockResolvedValue({ data: product })
  })

  it('keeps the SKU read-only and computes the unit price', async () => {
    const wrapper = mountDialog()

    expect(wrapper.get('[data-testid="addon-sku"]').attributes('readonly')).toBeDefined()
    expect((wrapper.get('[data-testid="addon-sku"]').element as HTMLInputElement).value).toBe('addon-usd-30')

    await wrapper.get('[data-testid="addon-quota"]').setValue('10')
    await wrapper.get('[data-testid="addon-price"]').setValue('2')

    expect(wrapper.text()).toContain('¥2.00')
    expect(wrapper.text()).toContain('¥0.2000 / $1')
    expect(wrapper.text()).toContain('$0.2000 / $1')
  })

  it('submits all mutable fields and omits the SKU', async () => {
    const wrapper = mountDialog()
    await wrapper.get('[data-testid="addon-name"]').setValue('  Updated add-on  ')
    await wrapper.get('[data-testid="addon-quota"]').setValue('40')
    await wrapper.get('[data-testid="addon-price"]').setValue('6.25')
    await wrapper.get('[data-testid="addon-original-price"]').setValue('7.5')
    await wrapper.get('[data-testid="addon-sort-order"]').setValue('25')

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(updateAddonProduct).toHaveBeenCalledWith(2, {
      name: 'Updated add-on',
      quota_usd: 40,
      price: 6.25,
      original_price: 7.5,
      for_sale: true,
      sort_order: 25,
    })
    expect(updateAddonProduct.mock.calls[0][1]).not.toHaveProperty('sku')
    expect(showSuccess).toHaveBeenCalled()
  })
})
