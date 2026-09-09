import { describe, expect, it } from 'vitest'
import { resolveUserErrorHint } from '../userErrorHint'

describe('resolveUserErrorHint', () => {
  it('uses category when the message is generic English', () => {
    expect(resolveUserErrorHint({ category: 'quota', message: 'Upstream payment required' })).toBe('quota')
    expect(resolveUserErrorHint({ category: 'rate_limit', message: 'Rate limit exceeded' })).toBe('rate_limit')
    expect(resolveUserErrorHint({ category: 'auth', message: 'Invalid API key' })).toBe('auth')
  })

  it('refines invalid_request into model or context', () => {
    expect(
      resolveUserErrorHint({
        category: 'invalid_request',
        message: 'Model "gpt-missing" is not supported by any configured account',
      })
    ).toBe('invalid_request_model')
    expect(
      resolveUserErrorHint({
        category: 'invalid_request',
        message: 'This model\'s maximum context length was exceeded',
      })
    ).toBe('invalid_request_context')
  })

  it('refines upstream timeouts', () => {
    expect(resolveUserErrorHint({ category: 'upstream', message: 'Upstream response timed out' })).toBe(
      'upstream_timeout'
    )
  })

  it('recovers a category from common English when backend says other', () => {
    expect(resolveUserErrorHint({ category: 'other', message: 'No available accounts' })).toBe(
      'service_unavailable'
    )
    expect(resolveUserErrorHint({ category: 'other', message: 'model_not_found' })).toBe(
      'invalid_request_model'
    )
  })
})
