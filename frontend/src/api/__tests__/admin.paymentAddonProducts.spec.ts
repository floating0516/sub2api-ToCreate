import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, put } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, put },
}))

import { adminPaymentAPI, type UpdateSubscriptionAddonProductRequest } from '@/api/admin/payment'

describe('admin add-on product API', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
  })

  it('lists all products from the admin catalog endpoint', async () => {
    get.mockResolvedValue({ data: [] })

    await adminPaymentAPI.getAddonProducts()

    expect(get).toHaveBeenCalledWith('/admin/payment/addon-products')
  })

  it('updates mutable fields without sending a SKU', async () => {
    const payload: UpdateSubscriptionAddonProductRequest = {
      name: '30 USD Add-on',
      quota_usd: 30,
      price: 5.49,
      original_price: null,
      for_sale: true,
      sort_order: 20,
    }
    put.mockResolvedValue({ data: { id: 2, sku: 'addon-usd-30', ...payload } })

    await adminPaymentAPI.updateAddonProduct(2, payload)

    expect(put).toHaveBeenCalledWith('/admin/payment/addon-products/2', payload)
    expect(put.mock.calls[0][1]).not.toHaveProperty('sku')
  })
})
