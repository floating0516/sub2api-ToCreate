import { describe, expect, it } from 'vitest'
import {
  detectSubscriptionTier,
  subscriptionAccentBarClass,
  subscriptionButtonClass,
  subscriptionTextClass,
} from '@/utils/subscriptionColors'
describe('subscriptionColors', () => {
  it('maps Chinese and English light tier names to the lighter bronze', () => {
    expect(detectSubscriptionTier({ planName: 'GPT Pro 轻量月卡' })).toBe('light')
    expect(detectSubscriptionTier({ groupName: 'GPT Pro Light v2' })).toBe('light')
    expect(subscriptionAccentBarClass({ groupName: 'GPT Pro Light v2', platform: 'openai' })).toContain('primary-200')
    expect(subscriptionButtonClass({ planName: '轻量周卡', platform: 'openai' })).toContain('primary-400')
  })

  it('maps Chinese and English standard tier names to the deeper bronze', () => {
    expect(detectSubscriptionTier({ planName: 'GPT Pro 标准月卡' })).toBe('standard')
    expect(detectSubscriptionTier({ groupName: 'GPT Pro Standard v2' })).toBe('standard')
    expect(subscriptionAccentBarClass({ groupName: 'GPT Pro Standard v2', platform: 'openai' })).toContain('primary-600')
    expect(subscriptionTextClass({ planName: '标准周卡', platform: 'anthropic' })).toContain('primary-700')
  })

  it('uses the lighter bronze for plans without a light or standard name', () => {
    const context = { planName: 'GPT Pro 月卡', groupName: 'GPT Pro', platform: 'anthropic' }
    expect(detectSubscriptionTier(context)).toBeNull()
    expect(subscriptionTextClass(context)).toContain('primary-600')
  })

  it('maps high-quota names to the deeper bronze', () => {
    expect(detectSubscriptionTier({ planName: 'GPT Pro 高额度周卡' })).toBe('standard')
    expect(subscriptionAccentBarClass({ planName: 'GPT Pro 高额度周卡' })).toContain('primary-600')
  })
})
