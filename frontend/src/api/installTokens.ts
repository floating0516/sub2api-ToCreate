import { apiClient } from './client'

export type InstallClient = 'claude-code' | 'codex' | 'gemini-cli'

export interface InstallTokenKeySummary {
  id: number
  name: string
  prefix: string
  group_id: number
  group_name: string
  platform: string
  rate_multiplier: number
}

export interface InstallTokenCommands {
  unix: string
  windows: string
}

export interface InstallTokenIssueResult {
  token: string
  client: InstallClient
  expires_at: string
  commands: InstallTokenCommands
  fallback_url: string
  key: InstallTokenKeySummary
}

export interface InstallTokenPeekResult {
  client: InstallClient
  expires_at: string
  provider_name: string
  endpoint: string
  key: InstallTokenKeySummary
}

export interface InstallTokenRedeemResult {
  client: InstallClient
  app: string
  provider_name: string
  homepage: string
  endpoint: string
  api_key: string
  model?: string
  usage_script: string
  deeplink: string
  confirm_url?: string
  key_name: string
}

export interface InstallCredentialRequest {
  token?: string
  receipt?: string
  os?: string
  arch?: string
}

export interface IssueInstallTokenRequest {
  client: InstallClient
  key_id: number
  previous_token?: string
}

export async function issueInstallToken(
  payload: IssueInstallTokenRequest
): Promise<InstallTokenIssueResult> {
  const { data } = await apiClient.post<InstallTokenIssueResult>('/install-token', payload)
  return data
}

export async function revokeInstallToken(token: string): Promise<{ status: string }> {
  const { data } = await apiClient.post<{ status: string }>('/install-token/revoke', { token })
  return data
}

export async function peekInstallCredential(
  payload: InstallCredentialRequest
): Promise<InstallTokenPeekResult> {
  const { data } = await apiClient.post<InstallTokenPeekResult>('/install-token/peek', payload)
  return data
}

export async function redeemInstallToken(
  payload: InstallCredentialRequest
): Promise<InstallTokenRedeemResult> {
  const { data } = await apiClient.post<InstallTokenRedeemResult>('/install-token/redeem', payload)
  return data
}

export async function confirmInstallReceipt(
  payload: InstallCredentialRequest
): Promise<InstallTokenRedeemResult> {
  const { data } = await apiClient.post<InstallTokenRedeemResult>('/install-token/confirm', payload)
  return data
}

export const installTokensAPI = {
  issue: issueInstallToken,
  revoke: revokeInstallToken,
  peek: peekInstallCredential,
  redeem: redeemInstallToken,
  confirm: confirmInstallReceipt
}

export default installTokensAPI
