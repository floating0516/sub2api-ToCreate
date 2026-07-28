import type { GroupPlatform } from '@/types'
import type {
  InstallClient,
  InstallCredentialRequest,
  IssueInstallTokenRequest
} from '@/api/installTokens'

const CLIENT_PLATFORMS: Record<InstallClient, readonly GroupPlatform[]> = {
  'claude-code': ['anthropic', 'antigravity', 'composite'],
  codex: ['openai'],
  'gemini-cli': ['gemini', 'antigravity']
}

export interface InstallConfirmationAction {
  kind: 'token' | 'receipt'
  request: InstallCredentialRequest
}

export function isQuickStartInstallerAccessible(
  isAdmin: boolean,
  featureEnabled: boolean,
): boolean {
  return isAdmin || featureEnabled
}

export function isQuickStartPlatformCompatible(
  client: InstallClient,
  platform: GroupPlatform | string | null | undefined
): boolean {
  if (!platform) return false
  return CLIENT_PLATFORMS[client].includes(platform as GroupPlatform)
}

export function buildInstallTokenIssueRequest(
  client: InstallClient,
  keyId: number,
  previousToken?: string
): IssueInstallTokenRequest {
  const payload: IssueInstallTokenRequest = {
    client,
    key_id: keyId
  }
  if (previousToken?.trim()) {
    payload.previous_token = previousToken.trim()
  }
  return payload
}

export function resolveInstallConfirmationAction(
  query: Record<string, unknown>
): InstallConfirmationAction | null {
  const receipt = typeof query.receipt === 'string' ? query.receipt.trim() : ''
  if (receipt) {
    return {
      kind: 'receipt',
      request: { receipt }
    }
  }

  const token = typeof query.token === 'string' ? query.token.trim() : ''
  if (token) {
    return {
      kind: 'token',
      request: { token }
    }
  }

  return null
}

export function maskQuickStartKey(value: string): string {
  const normalized = value.trim()
  if (normalized.length <= 10) return '******'
  return `${normalized.slice(0, 8)}****`
}

export function getBrowserPlatformMetadata(): Pick<InstallCredentialRequest, 'os' | 'arch'> {
  const userAgent = typeof navigator === 'undefined' ? '' : navigator.userAgent.toLowerCase()
  const platform = typeof navigator === 'undefined' ? '' : navigator.platform.toLowerCase()
  const source = `${userAgent} ${platform}`

  let os = 'browser'
  if (source.includes('windows')) os = 'windows'
  else if (source.includes('mac')) os = 'darwin'
  else if (source.includes('linux')) os = 'linux'

  let arch = 'unknown'
  if (source.includes('arm64') || source.includes('aarch64')) arch = 'arm64'
  else if (source.includes('x86_64') || source.includes('win64') || source.includes('x64')) arch = 'amd64'

  return { os, arch }
}
