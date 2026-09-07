export interface PublicLandingPlan {
  id: number
  name: string
  description: string
  price: number
  original_price?: number | null
  currency?: string
  validity_days: number
  validity_unit: string
  features: string[]
  sort_order: number
}

const hiddenFeaturePattern = /支付宝|余额支付|alipay|balance payment/i

export function publicFeatureLines(features: string[] | undefined, limit = 3): string[] {
  return (features || [])
    .map((line) => line.trim())
    .filter((line) => line.length > 0 && !hiddenFeaturePattern.test(line))
    .slice(0, limit)
}

export function isRecommendedPlan(plan: Pick<PublicLandingPlan, 'name' | 'description'>): boolean {
  return /推荐/.test(plan.description) || /标准周卡/.test(plan.name)
}

export function planPricePrefix(currency?: string): string {
  const value = (currency || '').trim().toUpperCase()
  if (!value || value === 'CNY') return '¥'
  if (value === 'USD') return '$'
  return `${value} `
}

export function formatPlanPrice(price: number): string {
  return price.toFixed(2)
}

export function hasOriginalPrice(plan: Pick<PublicLandingPlan, 'price' | 'original_price'>): boolean {
  return typeof plan.original_price === 'number' && plan.original_price > plan.price
}

export function purchasePathForPlan(planId: number): string {
  return `/purchase?tab=subscription&plan_id=${planId}`
}
