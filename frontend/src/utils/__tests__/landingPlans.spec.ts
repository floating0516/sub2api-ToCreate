import { describe, expect, it } from 'vitest'

import {
  formatPlanPrice,
  hasOriginalPrice,
  isRecommendedPlan,
  planPricePrefix,
  publicFeatureLines,
  purchasePathForPlan,
} from '@/utils/landingPlans'

describe('landingPlans', () => {
  it('keeps the first useful feature lines and hides settlement notes', () => {
    expect(publicFeatureLines([
      '7 天有效，每日 $50、每周 $350',
      '单位额度更优惠，推荐连续开发使用',
      '支持 GPT Pro 模型池与 OpenAI 兼容客户端',
      '周期额度不结转，到期后失效',
      '支付宝按人民币结算，余额支付按同数值扣除',
    ])).toEqual([
      '7 天有效，每日 $50、每周 $350',
      '单位额度更优惠，推荐连续开发使用',
      '支持 GPT Pro 模型池与 OpenAI 兼容客户端',
    ])
  })

  it('marks the recommended weekly plan', () => {
    expect(isRecommendedPlan({ name: 'GPT Pro 标准周卡', description: '推荐档' })).toBe(true)
    expect(isRecommendedPlan({ name: 'GPT Pro 日卡', description: '低门槛体验档' })).toBe(false)
  })

  it('formats the default RMB price and purchase path', () => {
    expect(planPricePrefix('')).toBe('¥')
    expect(formatPlanPrice(19.9)).toBe('19.90')
    expect(hasOriginalPrice({ price: 39.9, original_price: 48.16 })).toBe(true)
    expect(purchasePathForPlan(22)).toBe('/purchase?tab=subscription&plan_id=22')
  })
})
