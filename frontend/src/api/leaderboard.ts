import { apiClient } from './client'

export type LeaderboardPeriod = 'week' | 'month'
export type LeaderboardRewardType = 'daily_card' | 'weekly_card'

export interface LeaderboardPreference {
  user_id: number
  anonymous: boolean
  updated_at?: string
}

export interface LeaderboardEntry {
  rank: number
  user_id: number
  display_name: string
  username?: string
  email?: string
  role?: 'admin' | 'user'
  anonymous: boolean
  token_count: number
}

export interface LeaderboardRewardSettings {
  enabled: boolean
  subscription_group_id?: number | null
  weekly_first_days: number
  monthly_first_days: number
}

export interface LeaderboardReward {
  id: number
  period: LeaderboardPeriod
  period_start: string
  period_end: string
  rank: number
  user_id: number
  token_count: number
  reward_type: LeaderboardRewardType
  redeem_code_id: number
  redeem_code?: string
  created_by?: number
  created_at: string
}

export interface LeaderboardSnapshot {
  period: LeaderboardPeriod
  period_start: string
  period_end: string
  today_tokens: number
  my_rank?: LeaderboardEntry | null
  entries: LeaderboardEntry[]
  preference: LeaderboardPreference
  reward_settings: LeaderboardRewardSettings
}

export interface AdminLeaderboardSnapshot {
  period: LeaderboardPeriod
  period_start: string
  period_end: string
  entries: LeaderboardEntry[]
  rewards: LeaderboardReward[]
  reward_settings: LeaderboardRewardSettings
}

export interface GenerateLeaderboardRewardResult {
  reward: LeaderboardReward
  code: string
}

export async function getLeaderboard(
  period: LeaderboardPeriod,
  limit = 20
): Promise<LeaderboardSnapshot> {
  const { data } = await apiClient.get<LeaderboardSnapshot>('/leaderboard', {
    params: { period, limit }
  })
  return data
}

export async function updateLeaderboardPrivacy(anonymous: boolean): Promise<LeaderboardPreference> {
  const { data } = await apiClient.put<LeaderboardPreference>('/leaderboard/privacy', {
    anonymous
  })
  return data
}

export const leaderboardAPI = {
  get: getLeaderboard,
  updatePrivacy: updateLeaderboardPrivacy
}

export default leaderboardAPI

