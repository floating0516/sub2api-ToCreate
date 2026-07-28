import { describe, expect, it } from 'vitest'
import {
  buildInstallTokenIssueRequest,
  isQuickStartInstallerAccessible,
  isQuickStartPlatformCompatible,
  resolveInstallConfirmationAction
} from '@/utils/quickstart'

describe('quickstart utils', () => {
  it.each([
    [false, false, false],
    [false, true, true],
    [true, false, true],
    [true, true, true],
  ])(
    'resolves installer access for admin=%s enabled=%s',
    (isAdmin, enabled, expected) => {
      expect(isQuickStartInstallerAccessible(isAdmin, enabled)).toBe(expected)
    },
  )

  it.each([
    ['claude-code', 'anthropic', true],
    ['claude-code', 'antigravity', true],
    ['claude-code', 'composite', true],
    ['claude-code', 'openai', false],
    ['codex', 'openai', true],
    ['codex', 'gemini', false],
    ['gemini-cli', 'gemini', true],
    ['gemini-cli', 'antigravity', true],
    ['gemini-cli', 'anthropic', false]
  ] as const)('matches %s with %s', (client, platform, expected) => {
    expect(isQuickStartPlatformCompatible(client, platform)).toBe(expected)
  })

  it('includes the previous token when refreshing an install command', () => {
    expect(buildInstallTokenIssueRequest('codex', 42, ' tcinst_previous ')).toEqual({
      client: 'codex',
      key_id: 42,
      previous_token: 'tcinst_previous'
    })
  })

  it('prefers a confirmation receipt over an original token', () => {
    expect(resolveInstallConfirmationAction({
      token: 'tcinst_original',
      receipt: 'tcrcp_fallback'
    })).toEqual({
      kind: 'receipt',
      request: { receipt: 'tcrcp_fallback' }
    })
  })

  it('uses the original token when no receipt is present', () => {
    expect(resolveInstallConfirmationAction({ token: 'tcinst_original' })).toEqual({
      kind: 'token',
      request: { token: 'tcinst_original' }
    })
  })
})
