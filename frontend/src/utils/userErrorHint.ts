/**
 * 把用量「错误请求」的分类 + 原始英文 message 收成用户能看懂的说明 key。
 * 分类码与后端 MapUserErrorCategory 对齐；文案走 i18n `usage.errors.hints.*`。
 */

export const USER_ERROR_HINT_KINDS = [
  'auth',
  'rate_limit',
  'quota',
  'invalid_request',
  'invalid_request_model',
  'invalid_request_context',
  'service_unavailable',
  'upstream',
  'upstream_timeout',
  'internal',
  'cyber',
  'other',
] as const

export type UserErrorHintKind = (typeof USER_ERROR_HINT_KINDS)[number]

const CATEGORY_HINTS = new Set<UserErrorHintKind>([
  'auth',
  'rate_limit',
  'quota',
  'invalid_request',
  'service_unavailable',
  'upstream',
  'internal',
  'cyber',
  'other',
])

function hasAny(text: string, needles: string[]): boolean {
  return needles.some((needle) => text.includes(needle))
}

export function resolveUserErrorHint(input: {
  category?: string | null
  message?: string | null
}): UserErrorHintKind {
  const category = (input.category || '').toLowerCase()
  const message = (input.message || '').toLowerCase()

  const looksLikeUnknown = category === '' || category === 'other'

  if (category === 'invalid_request' || looksLikeUnknown) {
    if (
      hasAny(message, [
        'model_not_found',
        'model not found',
        'not supported by any',
        'supporting model',
        'is not supported',
        'unsupported configured model',
        'no such deployment',
        'does not exist',
      ])
    ) {
      return 'invalid_request_model'
    }
    if (
      hasAny(message, [
        'context length',
        'context_length',
        'too many tokens',
        'maximum context',
        'token limit',
        'max_tokens',
      ])
    ) {
      return 'invalid_request_context'
    }
  }

  if (category === 'upstream' || looksLikeUnknown) {
    if (hasAny(message, ['timeout', 'timed out', 'deadline exceeded'])) {
      return 'upstream_timeout'
    }
  }

  if (looksLikeUnknown) {
    if (hasAny(message, ['invalid api key', 'authentication_error'])) {
      return 'auth'
    }
    if (hasAny(message, ['rate limit', 'rate_limit', 'overloaded'])) {
      return 'rate_limit'
    }
    if (hasAny(message, ['insufficient', 'quota', 'payment required', 'billing', 'balance'])) {
      return 'quota'
    }
    if (hasAny(message, ['no available', 'no healthy', 'failover budget'])) {
      return 'service_unavailable'
    }
    if (hasAny(message, ['cyber_policy', 'content policy', 'safety system'])) {
      return 'cyber'
    }
  }

  if (CATEGORY_HINTS.has(category as UserErrorHintKind)) {
    return category as UserErrorHintKind
  }
  return 'other'
}

export function userErrorHintI18nKey(input: {
  category?: string | null
  message?: string | null
}): string {
  return `usage.errors.hints.${resolveUserErrorHint(input)}`
}
