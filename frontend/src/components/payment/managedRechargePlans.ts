import type { ManagedRechargeProduct } from '@/api/managedRecharge'

export const MANAGED_RECHARGE_PLAN_KEYS = ['plus', 'pro-5x', 'pro-20x'] as const

export type ManagedRechargePlanKey = typeof MANAGED_RECHARGE_PLAN_KEYS[number]

export function normalizeManagedRechargePlanKey(value: unknown): ManagedRechargePlanKey | null {
  if (typeof value !== 'string') return null
  return MANAGED_RECHARGE_PLAN_KEYS.includes(value as ManagedRechargePlanKey)
    ? value as ManagedRechargePlanKey
    : null
}

function productSearchText(product: ManagedRechargeProduct): string {
  return `${product.slug} ${product.name}`.toLowerCase()
}

export function findManagedRechargeProduct(
  products: ManagedRechargeProduct[],
  plan: ManagedRechargePlanKey,
): ManagedRechargeProduct | null {
  if (plan === 'plus') {
    return products.find(product => product.plan_type === 'plus') ?? null
  }

  const multiplier = plan === 'pro-5x' ? '5' : '20'
  const matched = products.find((product) => {
    if (product.plan_type !== 'pro') return false
    const searchText = productSearchText(product)
    return searchText.includes(`pro-${multiplier}x`)
      || searchText.includes(`pro_${multiplier}x`)
      || searchText.includes(`pro${multiplier}x`)
      || new RegExp(`(^|[^0-9])${multiplier}\\s*(x|倍)`, 'i').test(searchText)
  })

  if (matched) return matched
  if (plan === 'pro-5x') {
    return products.find(product => product.plan_type === 'pro' && product.slug === 'mock-pro') ?? null
  }
  return null
}
