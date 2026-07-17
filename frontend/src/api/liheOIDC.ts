import { apiClient } from './client'

export interface PreparedOIDCAuthorization {
  request_id: string
  expires_in: number
  unauthenticated_redirect_to?: string
}

export interface OIDCAuthorizationResult {
  redirect_to?: string
  expires_in?: number
  reauthenticate?: boolean
}

export async function prepareOIDCAuthorization(
  params: Record<string, string>
): Promise<PreparedOIDCAuthorization> {
  const { data } = await apiClient.post<PreparedOIDCAuthorization>('/oidc/prepare', { params })
  return data
}

export async function authorizeOIDC(requestId: string): Promise<OIDCAuthorizationResult> {
  const { data } = await apiClient.post<OIDCAuthorizationResult>('/oidc/authorize', {
    request_id: requestId,
  })
  return data
}
