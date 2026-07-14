import { describe, expect, it } from 'vitest'
import {
  detectSubscriptionTier,
  subscriptionAccentBarClass,
  subscriptionButtonClass,
  subscriptionTextClass,
} from '@/utils/subscriptionColors'
import { platformTextClass } from '@/utils/platformColors'

describe('subscriptionColors', () => {
  it('maps Chinese and English light tier names to yellow', () => {
    expect(detectSubscriptionTier({ planName: 'GPT Pro 轻量月卡' })).toBe('light')
    expect(detectSubscriptionTier({ groupName: 'GPT Pro Light v2' })).toBe('light')
    expect(subscriptionAccentBarClass({ groupName: 'GPT Pro Light v2', platform: 'openai' })).toContain('yellow')
    expect(subscriptionButtonClass({ planName: '轻量周卡', platform: 'openai' })).toContain('yellow')
  })

  it('maps Chinese and English standard tier names to green', () => {
    expect(detectSubscriptionTier({ planName: 'GPT Pro 标准月卡' })).toBe('standard')
    expect(detectSubscriptionTier({ groupName: 'GPT Pro Standard v2' })).toBe('standard')
    expect(subscriptionAccentBarClass({ groupName: 'GPT Pro Standard v2', platform: 'openai' })).toContain('green')
    expect(subscriptionTextClass({ planName: '标准周卡', platform: 'anthropic' })).toContain('green')
  })

  it('keeps platform colors for tiers without a configured name', () => {
    const context = { planName: 'GPT Pro 月卡', groupName: 'GPT Pro', platform: 'anthropic' }
    expect(detectSubscriptionTier(context)).toBeNull()
    expect(subscriptionTextClass(context)).toBe(platformTextClass('anthropic'))
  })
})
