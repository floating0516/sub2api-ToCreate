import { apiClient } from '../client'

export interface AccountContributionFeatureState {
  enabled: boolean
  submission_configured: boolean
  payout_configured: boolean
  submission_enabled: boolean
  payout_enabled: boolean
}

export interface AccountContributionAdminStats {
  contributors_total: number
  contributors_pending: number
  contributions_total: number
  contributions_active: number
  earning_entries_total: number
  total_earnings_cny_fen: number
  available_earnings_cny_fen: number
  payout_requests_total: number
  payout_requests_pending: number
  pending_payout_cny_fen: number
}

export interface AccountContributionAdminContributor {
  id: number
  user_id?: number | null
  email: string
  username: string
  status: string
  contributions: number
  created_at: string
}

export interface AccountContributionAdminAccount {
  id: number
  contributor_id: number
  contributor: string
  account_id?: number | null
  account_name: string
  platform: string
  status: string
  settlement_mode: string
  share_rate_bps: number
  created_at: string
}

export interface AccountContributionAdminEarning {
  id: number
  contributor_id: number
  contributor: string
  contribution_id: number
  account_name: string
  entry_type: string
  amount_cny_fen: number
  available_at: string
  created_at: string
}

export interface AccountContributionAdminPayout {
  id: number
  contributor_id: number
  contributor: string
  amount_cny_fen: number
  status: string
  method_type: string
  masked_destination: string
  requested_at: string
}

export interface AccountContributionAdminOverview {
  features: AccountContributionFeatureState
  stats: AccountContributionAdminStats
  contributors: AccountContributionAdminContributor[]
  contributions: AccountContributionAdminAccount[]
  earnings: AccountContributionAdminEarning[]
  payouts: AccountContributionAdminPayout[]
}

export async function getOverview(): Promise<AccountContributionAdminOverview> {
  const { data } = await apiClient.get<AccountContributionAdminOverview>(
    '/admin/account-contributions/overview',
  )
  return data
}

export const accountContributionsAPI = {
  getOverview,
}

export default accountContributionsAPI
