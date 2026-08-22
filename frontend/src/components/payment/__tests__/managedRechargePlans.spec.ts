import { describe, expect, it } from 'vitest'
import type { ManagedRechargeProduct } from '@/api/managedRecharge'
import { findManagedRechargeProduct, normalizeManagedRechargePlanKey } from '../managedRechargePlans'

function product(overrides: Partial<ManagedRechargeProduct>): ManagedRechargeProduct {
  return {
    id: 1,
    slug: 'gpt-plus',
    plan_type: 'plus',
    name: 'Plus',
    description: '',
    price: 10,
    active: true,
    sort_order: 1,
    available_stock: 1,
    total_stock: 1,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('managed recharge plans', () => {
  it('accepts only supported route plan keys', () => {
    expect(normalizeManagedRechargePlanKey('plus')).toBe('plus')
    expect(normalizeManagedRechargePlanKey('pro-5x')).toBe('pro-5x')
    expect(normalizeManagedRechargePlanKey('pro-20x')).toBe('pro-20x')
    expect(normalizeManagedRechargePlanKey('pro')).toBeNull()
  })

  it('matches the two Pro variants by slug or display name', () => {
    const products = [
      product({ id: 2, slug: 'gpt-pro-5x', plan_type: 'pro', name: 'Pro 5x' }),
      product({ id: 3, slug: 'gpt-pro-20x', plan_type: 'pro', name: 'Pro（20 倍）' }),
    ]

    expect(findManagedRechargeProduct(products, 'pro-5x')?.id).toBe(2)
    expect(findManagedRechargeProduct(products, 'pro-20x')?.id).toBe(3)
  })

  it('does not fall back to a generic Pro product for the 20x route', () => {
    const products = [product({ id: 2, slug: 'mock-pro', plan_type: 'pro', name: 'Pro 模拟成功' })]

    expect(findManagedRechargeProduct(products, 'pro-5x')?.id).toBe(2)
    expect(findManagedRechargeProduct(products, 'pro-20x')).toBeNull()
  })
})
