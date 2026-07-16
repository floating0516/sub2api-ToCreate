import { apiClient } from './client'

export interface LiheAccessTokenRecord {
  id: number
  name: string
  scopes: string[]
  providers: string[]
  last_used_at: string | null
  created_at: string
}

export interface LiheIntegration {
  enabled: boolean
  connect_url: string
  tokens: LiheAccessTokenRecord[]
}

export interface LiheAuthorizeResult {
  redirect_to: string
  expires_in: number
}

export async function getLiheIntegration(): Promise<LiheIntegration> {
  const { data } = await apiClient.get<LiheIntegration>('/oauth/lihe')
  return data
}

export async function authorizeLihe(query: Record<string, string>): Promise<LiheAuthorizeResult> {
  const { data } = await apiClient.get<LiheAuthorizeResult>('/oauth/lihe/authorize', {
    params: query,
  })
  return data
}

export async function revokeLiheToken(id: number): Promise<void> {
  await apiClient.delete(`/oauth/lihe/tokens/${id}`)
}

export default {
  getIntegration: getLiheIntegration,
  authorize: authorizeLihe,
  revokeToken: revokeLiheToken,
}
