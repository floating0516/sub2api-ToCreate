import { describe, expect, it } from 'vitest'
import { modelsByGroupId } from '../keyAvailableModels'
import type { UserAvailableChannel } from '@/api/channels'

function channel(partial: UserAvailableChannel): UserAvailableChannel {
  return partial
}

describe('modelsByGroupId', () => {
  it('unions models across channels that share a group', () => {
    const channels = [
      channel({
        name: 'A',
        description: '',
        platforms: [
          {
            platform: 'openai',
            groups: [
              {
                id: 1,
                name: 'gpt',
                platform: 'openai',
                subscription_type: 'standard',
                rate_multiplier: 1,
                peak_rate_enabled: false,
                peak_start: '',
                peak_end: '',
                peak_rate_multiplier: 1,
                is_exclusive: false,
              },
            ],
            supported_models: [
              { name: 'gpt-5', platform: 'openai', pricing: null },
              { name: 'gpt-5.4', platform: 'openai', pricing: null },
            ],
          },
        ],
      }),
      channel({
        name: 'B',
        description: '',
        platforms: [
          {
            platform: 'openai',
            groups: [
              {
                id: 1,
                name: 'gpt',
                platform: 'openai',
                subscription_type: 'standard',
                rate_multiplier: 1,
                peak_rate_enabled: false,
                peak_start: '',
                peak_end: '',
                peak_rate_multiplier: 1,
                is_exclusive: false,
              },
            ],
            supported_models: [{ name: 'gpt-5.4', platform: 'openai', pricing: null }],
          },
        ],
      }),
    ]

    expect(modelsByGroupId(channels)[1]).toEqual(['gpt-5', 'gpt-5.4'])
  })

  it('keeps groups isolated', () => {
    const channels = [
      channel({
        name: 'A',
        description: '',
        platforms: [
          {
            platform: 'anthropic',
            groups: [
              {
                id: 2,
                name: 'claude',
                platform: 'anthropic',
                subscription_type: 'subscription',
                rate_multiplier: 1,
                peak_rate_enabled: false,
                peak_start: '',
                peak_end: '',
                peak_rate_multiplier: 1,
                is_exclusive: false,
              },
            ],
            supported_models: [{ name: 'claude-opus-4-6', platform: 'anthropic', pricing: null }],
          },
        ],
      }),
    ]

    expect(modelsByGroupId(channels)[1]).toBeUndefined()
    expect(modelsByGroupId(channels)[2]).toEqual(['claude-opus-4-6'])
  })
})
